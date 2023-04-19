package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	// "database/sql"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	// socketio "github.com/googollee/go-socket.io"
	// "github.com/gorilla/websocket"
	_ "github.com/go-sql-driver/mysql"
)

type ResultList struct {
	Result []Device `json:"result_list"`
}

type Device struct {
	Id string `json:"device_id"`
	Name string `json:"display_name"`
	ActiveState string `json:"active_state"`
	LatestPoint LatestDevicePoint `json:"latest_accurate_device_point"`
	Pinned bool `json:"pinned"`
	Hidden bool `json:"hidden"`
}

type LatestDevicePoint struct{
	Latitude float32 `json:"lat"`
	Longitude float32 `json:"lng"`
	DriveStatus DriveState `json:"device_state"`
}

type DriveState struct{
	Status string `json:"drive_status"`
}

type HideDevice struct{
	Id string `json:"Id"`
	Name string `json:"Name"`
}

type PinDevice struct{
	Id string `json:"Id"`
	Name string `json:"Name"`
}

var hiddenDevices []HideDevice
var pinnedDevices []PinDevice

var resList ResultList

var dbConn *sql.DB

func sendData(context *gin.Context){
	getDataFromOneStepGPS()
	getHiddenDevices()
	var temp []Device
	temp = nil
	arrMap := make(map[string]bool)
    for _, val := range hiddenDevices {
        arrMap[val.Id] = true
    }
	for i:=0;i<len(resList.Result);i++{
		if !arrMap[resList.Result[i].Id]{
			resList.Result[i].Hidden=false
			temp = append(temp, resList.Result[i])
		}else{
			resList.Result[i].Hidden=true
			temp = append(temp, resList.Result[i])
		}
	}
	resList.Result = temp
	resList.Result = MovePinnedDevicesUp(resList.Result)
	context.IndentedJSON(http.StatusOK,resList)
}

func MovePinnedDevicesUp(result []Device) []Device{
	getPinnedDevices()
	var temp []Device
	temp = nil
	arrMap := make(map[string]bool)
    for _, val := range pinnedDevices {
        arrMap[val.Id] = true
    }
	for i:=0;i<len(result);i++{
		if arrMap[result[i].Id]{
			result[i].Pinned = true
			temp = append(temp, result[i])
		}
	}
	for i:=0;i<len(result);i++{
		if !arrMap[result[i].Id]{
			result[i].Pinned = false
			temp = append(temp, result[i])
		}
	}
	return temp
}

func getDataFromOneStepGPS(){
	fmt.Println("getting data from OneStep")
	client := &http.Client{}
	request, err := http.NewRequest("GET", "https://track.onestepgps.com/v3/api/public/device?latest_point=true", nil)
	if err!= nil{
		return 
	}
	request.Header.Add("Authorization", "Bearer xMDOrYc8crMCDLIfd7CXyuS3D8w2BzcZdjuwgjPDkKI")
	response, err := client.Do(request)

	if err != nil {
		return
	}
	
	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return
	}

	json.Unmarshal(responseData, &resList)
}

func connectToMySQL(){
	db,err:= sql.Open("mysql","admin:MyAppPassword123!@tcp(3.101.22.26:3306)/trial")
	if err != nil {
		fmt.Println("error validating sql.Open arguments")
		panic(err.Error())
	}

	//defer db.Close()

	err = db.Ping()
	if err!=nil {
		fmt.Println("error verifying connection with db.ping")
		panic(err.Error())
	}

	fmt.Println("Connected To Database!")
	dbConn=db
}

func getPinnedDevices(){
	pinnedDevices = nil
	res,err:= dbConn.Query("SELECT * FROM `one_Step_gps_db`.`pinned_devices`")
	if err!=nil {
		fmt.Println("Error Querying From DB")
		fmt.Println(err.Error())
		panic(err.Error())
	}
	fmt.Println("Successfully Retrieved from DB")
	for res.Next(){
		var pDevices PinDevice

		err = res.Scan(&pDevices.Id,&pDevices.Name)
		if err != nil {
            panic(err.Error()) // proper error handling instead of panic in your app
        }
		fmt.Println(pDevices)
		pinnedDevices = append(pinnedDevices, pDevices)
	}
	
	defer res.Close()
}

func getHiddenDevices(){
	hiddenDevices=nil
	res,err:= dbConn.Query("SELECT * FROM `one_Step_gps_db`.`hidden_devices`")
	if err!=nil {
		fmt.Println("Error Querying From DB")
		fmt.Println(err.Error())
		panic(err.Error())
	}
	fmt.Println("Successfully Retrieved from DB")
	for res.Next(){
		var hDevices HideDevice

		err = res.Scan(&hDevices.Id,&hDevices.Name)
		if err != nil {
            panic(err.Error()) // proper error handling instead of panic in your app
        }
		fmt.Println(hDevices)
		hiddenDevices = append(hiddenDevices, hDevices)
	}
	
	defer res.Close()
}

func postHiddenDevices(c *gin.Context) {
    var hideDevice []HideDevice

    // Call BindJSON to bind the received JSON to
    // newAlbum.
    if err := c.BindJSON(&hideDevice); err != nil {
        return
    }

    // Add the new album to the slice.
    // albums = append(albums, newAlbum)
	//Delete existing records 
	_,err:= dbConn.Query("DELETE FROM `one_Step_gps_db`.`hidden_devices`")
	if err != nil {
		fmt.Println("Error deleting data from hidden devices table")
	}
    c.IndentedJSON(http.StatusCreated, hideDevice)
	for i:=0;i<len(hideDevice);i++{
		str := "INSERT INTO `one_Step_gps_db`.`hidden_devices`(device_id, display_name) VALUES ('"+ hideDevice[i].Id +"','" +  hideDevice[i].Name +"')"
		fmt.Println(str)
		_, err := dbConn.Query(str)

		// if there is an error inserting, handle it
		if err != nil {
			fmt.Println("Error adding data to db")
		}
	}
}

func postPinnedDevices(c *gin.Context) {
    var pinDevice []PinDevice

    // Call BindJSON to bind the received JSON to
    // newAlbum.
    if err := c.BindJSON(&pinDevice); err != nil {
        return
    }

    // Add the new album to the slice.
    // albums = append(albums, newAlbum)
	//Delete existing records 
	_,err:= dbConn.Query("DELETE FROM `one_Step_gps_db`.`pinned_devices`")
	if err != nil {
		fmt.Println("Error deleting data from pinned devices table")
	}

    c.IndentedJSON(http.StatusCreated, pinDevice)
	for i:=0;i<len(pinDevice);i++{
		str := "INSERT INTO `one_Step_gps_db`.`pinned_devices` (device_id, device_name) VALUES ('"+ pinDevice[i].Id +"','" +  pinDevice[i].Name +"')"
		fmt.Println(str)
		_, err := dbConn.Query(str)

		// if there is an error inserting, handle it
		if err != nil {
			fmt.Println("Error adding data to db")
		}
	}
}

func main(){
	connectToMySQL()
	router := gin.Default()
	// Use the Cors middleware with the Gin router
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST"}
	// router.Use(cors.New(cors.Config{
	// 	AllowOrigins:     []string{"*"},
	// }))
	router.Use(cors.New(config))
	router.GET("/devices",sendData)
	router.POST("/hideDevice", postHiddenDevices)
	router.POST("/pinDevice", postPinnedDevices)
	router.Run("0.0.0.0:8000")
}