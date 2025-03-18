package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sync"
	"time"
)

type Response struct {
	Data    DataWrapper `json:"data"`
	Success bool        `json:"success"`
}

type DataWrapper struct {
	Data    []FanData `json:"data"`
	Success bool      `json:"success"`
}

type FanData struct {
	Timestamp      int `json:"timestamp"`
	ControlFanPop0 int `json:"control_fan_pop_0"`
	ControlFanPop1 int `json:"control_fan_pop_1"`
	ControlFanPop2 int `json:"control_fan_pop_2"`
	ControlFanPop3 int `json:"control_fan_pop_3"`
	RpsFanPop0     int `json:"rps_fan_pop_0"`
	RpsFanPop1     int `json:"rps_fan_pop_1"`
	RpsFanPop2     int `json:"rps_fan_pop_2"`
	RpsFanPop3     int `json:"rps_fan_pop_3"`
}

type ApiResponse struct {
	URL              string             `json:"url"`
	Status           string             `json:"status"`
	Error            string             `json:"error,omitempty"`
	AverageFanSpeeds map[string]float64 `json:"average_fan_speeds,omitempty"`
}

var MOCK_TOKEN = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOjEzNTIsInVzZXJOYW1lIjoidHJpZXVuaDdAZnB0LmNvbS52biIsImZ1bGxOYW1lIjoidHJpZXVuaDdAZnB0LmNvbS52biIsImVtYWlsIjoidHJpZXVuaDdAZnB0LmNvbSIsImFsbG93X2FjY2VzcyI6MSwicGVybWlzc2lvbnMiOlt7InJlc291cmNlTmFtZSI6InBhZ2VfcG9wIiwiYWN0aW9uTmFtZSI6ImNyZWF0ZSJ9LHsicmVzb3VyY2VOYW1lIjoicGFnZV9wb3AiLCJhY3Rpb25OYW1lIjoiZGVsZXRlIn0seyJyZXNvdXJjZU5hbWUiOiJwYWdlX3BvcCIsImFjdGlvbk5hbWUiOiJlZGl0In0seyJyZXNvdXJjZU5hbWUiOiJtZW51X3NldHRpbmciLCJhY3Rpb25OYW1lIjoidmlldyJ9LHsicmVzb3VyY2VOYW1lIjoicGFnZV9zZXR0aW5nX3N5c3RlbV91c2VycyIsImFjdGlvbk5hbWUiOiJ2aWV3In0seyJyZXNvdXJjZU5hbWUiOiJwYWdlX3NldHRpbmdfc3lzdGVtX3Jlc291cmNlcyIsImFjdGlvbk5hbWUiOiJ2aWV3In0seyJyZXNvdXJjZU5hbWUiOiJwYWdlX3NldHRpbmdfc3lzdGVtX3JvbGVzIiwiYWN0aW9uTmFtZSI6InZpZXcifSx7InJlc291cmNlTmFtZSI6InBhZ2Vfc2V0dGluZ19zeXN0ZW1fcm9sZXMiLCJhY3Rpb25OYW1lIjoiY3JlYXRlIn0seyJyZXNvdXJjZU5hbWUiOiJwYWdlX3NldHRpbmdfc3lzdGVtX3VzZXJzIiwiYWN0aW9uTmFtZSI6ImNyZWF0ZSJ9LHsicmVzb3VyY2VOYW1lIjoicGFnZV9zZXR0aW5nX3N5c3RlbV91c2VycyIsImFjdGlvbk5hbWUiOiJlZGl0In0seyJyZXNvdXJjZU5hbWUiOiJwYWdlX3NldHRpbmdfc3lzdGVtX3VzZXJzIiwiYWN0aW9uTmFtZSI6ImRlbGV0ZSJ9LHsicmVzb3VyY2VOYW1lIjoicGFnZV9zZXR0aW5nX3N5c3RlbV9yb2xlcyIsImFjdGlvbk5hbWUiOiJkZWxldGUifSx7InJlc291cmNlTmFtZSI6InBhZ2Vfc2V0dGluZ19zeXN0ZW1fcmVzb3VyY2VzIiwiYWN0aW9uTmFtZSI6ImRlbGV0ZSJ9LHsicmVzb3VyY2VOYW1lIjoicGFnZV9zZXR0aW5nX3N5c3RlbV9yZXNvdXJjZXMiLCJhY3Rpb25OYW1lIjoiY3JlYXRlIn0seyJyZXNvdXJjZU5hbWUiOiJwYWdlX3NldHRpbmdfc3lzdGVtX3Jlc291cmNlcyIsImFjdGlvbk5hbWUiOiJlZGl0In0seyJyZXNvdXJjZU5hbWUiOiJwYWdlX3NldHRpbmdfc3lzdGVtX3JvbGVzIiwiYWN0aW9uTmFtZSI6ImVkaXQifSx7InJlc291cmNlTmFtZSI6InBhZ2VfcG9wIiwiYWN0aW9uTmFtZSI6ImFkbWluIn0seyJyZXNvdXJjZU5hbWUiOiJwYWdlX3BvcCIsImFjdGlvbk5hbWUiOiJ2aWV3In0seyJyZXNvdXJjZU5hbWUiOiJwYWdlX3NldHRpbmdfc3lzdGVtX3ZlcnNpb25zIiwiYWN0aW9uTmFtZSI6ImFkbWluIn1dLCJpYXQiOjE3NDIyODk2MjksImV4cCI6MTc0MjMxODQyOX0.kjLQYwj11UyU6M5oUbw4YRSm1kEBLWSAU63CJ9WvD2U"

func getToken() string {
	return MOCK_TOKEN
}

type Pi struct {
	Id           int    `json:"id"`
	Name         string `json:"name"`
	Ip           string `json:"ip"`
	Address      string `json:"address"`
	BackendPort  int    `json:"backendPort"`
	Email        string `json:"email"`
	Username     string `json:"username"`
	BrokerUrl    string `json:"brokerUrl"`
	MqttUser     string `json:"mqttUser"`
	MqttPassword string `json:"mqttPassword"`
	Role         string `json:"role"`
}

type PiFolderResponse struct {
	Data []Pi `json:"data"`
}

var T_START = 1741478400
var T_END = 1741564800

func getEndpoints(timeStart int, timeEnd int) []string {
	urlGetPis := "https://smartpop.fpt.net/api/opms/pis?folderId=&isExtra="
	client := &http.Client{Timeout: 20 * time.Second}

	req, err := http.NewRequest("GET", urlGetPis, nil)
	if err != nil {
		fmt.Println("❌ Error creating request:", err)
		return nil
	}

	token := getToken() // Ensure this function works correctly
	req.Header.Add("x-access-token", token)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("❌ Error sending request:", err)
		return nil
	}
	defer resp.Body.Close() // Ensure response body is closed

	// Handle non-200 status codes
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("⚠️ API returned non-OK status: %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode))
		return nil
	}

	defer resp.Body.Close()

	// Define a structured response
	var piFolderResponse PiFolderResponse

	if err := json.NewDecoder(resp.Body).Decode(&piFolderResponse); err != nil {
		fmt.Println("Error decoding response:", err)
		return nil
	}

	// Extract IDs from response
	var endpoints []string

	for _, pi := range piFolderResponse.Data {

		nextEndpoint := fmt.Sprintf("https://smartpop.fpt.net/api/opms/pis/%d/log/fan-pop?tsdatesta=%d&tsdateend=%d", pi.Id, timeStart, timeEnd)

		endpoints = append(endpoints, nextEndpoint)

	}

	// return endpoints

	endpointsSlice10 := endpoints[:1]

	return endpointsSlice10

}

func getFanData(entries []map[string]any) map[string]float64 {

	//get max rps_fan_pop of each fan at control_fan = 100

	// Map to store count of fan speeds
	fanMaxRps := map[string]float64{
		"rps1": 0,
		"rps2": 0,
		"rps3": 0,
		"rps4": 0,
	}

	for _, entry := range entries {
		// Get the fan control
		controlFanPop0 := entry["control_fan_pop_0"].(int)
		controlFanPop1 := entry["control_fan_pop_1"].(int)
		controlFanPop2 := entry["control_fan_pop_2"].(int)
		controlFanPop3 := entry["control_fan_pop_3"].(int)

		if controlFanPop0 == 100 && fanMaxRps["rps1"] < float64(entry["rps_fan_pop_0"].(int)) {
			fanMaxRps["rps1"] = float64(entry["rps_fan_pop_0"].(int))
		}

		if controlFanPop1 == 100 && fanMaxRps["rps2"] < float64(entry["rps_fan_pop_1"].(int)) {
			fanMaxRps["rps2"] = float64(entry["rps_fan_pop_1"].(int))
		}

		if controlFanPop2 == 100 && fanMaxRps["rps3"] < float64(entry["rps_fan_pop_2"].(int)) {
			fanMaxRps["rps3"] = float64(entry["rps_fan_pop_2"].(int))
		}

		if controlFanPop3 == 100 && fanMaxRps["rps4"] < float64(entry["rps_fan_pop_3"].(int)) {
			fanMaxRps["rps4"] = float64(entry["rps_fan_pop_3"].(int))
		}
	}

	//print

	fmt.Println(fanMaxRps)

	return fanMaxRps
}

func writeCsvFile(results <-chan ApiResponse) {
	// Create a CSV file
	file, err := os.Create("fan_speeds.csv")

	if err != nil {

		fmt.Println("Error creating CSV file:", err)

		return

	}

	defer file.Close()

	writer := csv.NewWriter(file)

	defer writer.Flush()

	// Write CSV header
	header := []string{"PI ID", "Status", "Fan", "Average Speed"}

	writer.Write(header)

	// define regexp

	re := regexp.MustCompile(`pis/(\d+)`)

	// Write CSV rows

	for result := range results {

		// get number after pis/

		// example: /pis/1363 -> 1363

		matches := re.FindStringSubmatch(result.URL)

		piId := matches[1]

		if result.Status == "success" {

			for key, value := range result.AverageFanSpeeds {

				record := []string{piId, result.Status, key, fmt.Sprintf("%.2f", value)}

				writer.Write(record)

			}

		} else {

			record := []string{piId, result.Status, "", result.Error}

			writer.Write(record)

		}

	}

	fmt.Println("Results have been written to fan_speeds.csv")
}

func fetchAPI(endpoint string, wg *sync.WaitGroup, results chan<- ApiResponse) {
	defer wg.Done()

	client := &http.Client{Timeout: 60 * time.Second}

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		results <- ApiResponse{URL: endpoint, Status: "error", Error: err.Error()}
		return
	}

	token := getToken()

	// Add the authentication header
	req.Header.Add("x-access-token", token)

	resp, err := client.Do(req)
	if err != nil {
		results <- ApiResponse{URL: endpoint, Status: "error", Error: err.Error()}
		return
	}
	defer resp.Body.Close()
	// Handle non-200 status codes

	//get id from endpoint
	re := regexp.MustCompile(`pis/(\d+)`)
	matches := re.FindStringSubmatch(endpoint)
	id := matches[1]

	if resp.StatusCode != http.StatusOK {

		fmt.Printf("⚠️ API returned non-OK status for: %d %s for id: %s\n", resp.StatusCode, http.StatusText(resp.StatusCode), id)
		return
	}

	defer resp.Body.Close()

	// Define a structured response
	var responseData struct {
		Data struct {
			Success bool             `json:"success"`
			Entries []map[string]any `json:"data"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		results <- ApiResponse{URL: endpoint, Status: "error", Error: err.Error()}
		return
	}

	if !responseData.Data.Success {
		results <- ApiResponse{URL: endpoint, Status: "error", Error: "API call failed"}
		fmt.Printf("❌ API failed: %s\n", id)
		return
	}

	averageFanSpeeds := getFanData(responseData.Data.Entries)

	// Send results
	results <- ApiResponse{
		URL:              endpoint,
		Status:           "success",
		AverageFanSpeeds: averageFanSpeeds,
	}
}

func getOpmsDataPipeline() {
	endpoints := getEndpoints(T_START, T_END)

	fmt.Printf("Found %d Endpoints\n", len(endpoints))

	var wg sync.WaitGroup
	results := make(chan ApiResponse, len(endpoints))

	fmt.Println("Starting API calls...")

	var countProcessed int

	for i, endpoint := range endpoints {
		wg.Add(1)
		go fetchAPI(endpoint, &wg, results)

		// Introduce a 10s delay for every 50 requests
		if (i+1)%50 == 0 {
			countProcessed += 50
			fmt.Printf("Processed %d/%d ⚡ Rate limit reached, cooling down for 20s...\n", countProcessed, len(endpoints))
			time.Sleep(20 * time.Second)
		}
	}

	wg.Wait()
	close(results)

	writeCsvFile(results)
}

func main() {

	getOpmsDataPipeline()

}

// package main

// import (
// 	"fmt"
// 	"log"
// 	"net/http"
// 	"project/database"
// 	"project/handlers"

// 	"github.com/gorilla/mux"
// )

// func main() {
// 	database.Connect()

// 	router := mux.NewRouter()
// 	router.HandleFunc("/users", handlers.GetUsers).Methods("GET")
// 	router.HandleFunc("/users", handlers.CreateUser).Methods("POST")
// 	router.HandleFunc("/users/{id}", handlers.UpdateUser).Methods("PUT")
// 	router.HandleFunc("/users/{id}", handlers.DeleteUser).Methods("DELETE")

// 	fmt.Println("Server running on port 8080")
// 	log.Fatal(http.ListenAndServe(":8000", router))
// }

// package main

// import (
// 	"fmt"
// 	"log"
// 	"net/http"
// 	"project/database"
// 	"project/handlers"

// 	"github.com/gorilla/mux"
// )

// func main() {
// 	database.Connect()

// 	router := mux.NewRouter()
// 	router.HandleFunc("/users", handlers.GetUsers).Methods("GET")
// 	router.HandleFunc("/users/{id}", handlers.GetUser).Methods("GET")
// 	router.HandleFunc("/users", handlers.CreateUser).Methods("POST")
// 	router.HandleFunc("/users/{id}", handlers.UpdateUser).Methods("PUT")
// 	router.HandleFunc("/users/{id}", handlers.DeleteUser).Methods("DELETE")

// 	fmt.Println("Server running on port 8080!")
// 	log.Fatal(http.ListenAndServe(":8080", router))
// }
