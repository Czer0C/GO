package jobs

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"project/utils"
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
	URL    string `json:"url"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
	// AverageFanSpeeds map[string]float64 `json:"average_fan_speeds,omitempty"`
	FanData map[string]float64 `json:"fan_data,omitempty"`
	POP     string             `json:"pop,omitempty"`
	PID     int                `json:"pid,omitempty"`
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

type Endpoint struct {
	piId     int
	endpoint string
	pop      string
}

var T_START = 1741478400
var T_END = 1741564800

var countProcessed = 0

func getEndpoints(timeStart int64, timeEnd int64, limit int) []Endpoint {

	urlGetPis := "https://smartpop.fpt.net/api/opms/pis?folderId=&isExtra="

	client := &http.Client{Timeout: 20 * time.Second}

	req, err := http.NewRequest("GET", urlGetPis, nil)

	if err != nil {
		fmt.Println("❌ Error creating request:", err)
		return nil
	}

	token := utils.GetTokenIOT() // Ensure this function works correctly

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
	// var endpoints []string

	var endpoints []Endpoint

	for _, pi := range piFolderResponse.Data {

		nextEndpoint := fmt.Sprintf("https://smartpop.fpt.net/api/opms/pis/%d/log/fan-pop?tsdatesta=%d&tsdateend=%d", pi.Id, timeStart, timeEnd)

		endpoints = append(endpoints, Endpoint{piId: pi.Id, endpoint: nextEndpoint, pop: pi.Name})
	}

	if limit == -1 {
		return endpoints
	}

	sliced := endpoints[:limit]

	return sliced

}

func getFanData(entries []map[string]any) map[string]float64 {
	fanRps := map[string]float64{"f1": 0, "f2": 0, "f3": 0, "f4": 0}
	countControlFan100 := map[string]int{"f1": 0, "f2": 0, "f3": 0, "f4": 0}

	// Iterate through all fan entries
	for _, fan := range entries {
		for i := 0; i < 4; i++ {
			rpsKey := fmt.Sprintf("rps_fan_pop_%d", i)
			controlKey := fmt.Sprintf("control_fan_pop_%d", i)
			fanKey := fmt.Sprintf("f%d", i+1)

			// Type assertion with safety check
			if rps, rpsOk := fan[rpsKey].(float64); rpsOk {
				if control, controlOk := fan[controlKey].(float64); controlOk && control == 100 {
					fanRps[fanKey] += rps
					countControlFan100[fanKey]++
				}
			}
		}
	}

	// Compute the average, avoiding NaN issues
	for key, count := range countControlFan100 {
		if count > 0 {
			fanRps[key] = math.Floor(fanRps[key] / float64(count))
		} else {
			fanRps[key] = 0 // Ensure default value is 0
		}
	}

	return fanRps
}

func writeCsvFile(results <-chan ApiResponse, outputFile string) {
	file, err := os.Create(outputFile)
	if err != nil {
		fmt.Println("Error creating CSV file:", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"PI ID", "POP", "Status", "Fan 1", "Fan 2", "Fan 3", "Fan 4"}
	writer.Write(header)

	for result := range results {
		pId := fmt.Sprintf("%d", result.PID)

		if result.Status == "success" {
			record := []string{
				pId, result.POP, result.Status,
				fmt.Sprintf("%.2f", result.FanData["f1"]),
				fmt.Sprintf("%.2f", result.FanData["f2"]),
				fmt.Sprintf("%.2f", result.FanData["f3"]),
				fmt.Sprintf("%.2f", result.FanData["f4"]),
			}
			writer.Write(record)
		} else {
			record := []string{pId, result.POP, result.Status, "", result.Error}
			writer.Write(record)
		}
	}

	fmt.Printf("Results have been written to %s\n", outputFile)
}

func showSpinner(done chan bool, pop string, total int) {
	spinners := []string{"-", "\\", "|", "/"}
	i := 0

	for {
		select {
		case <-done:
			fmt.Printf("\r %d/%d | %s ✅ \n", countProcessed, total, pop)
			return
		default:
			fmt.Printf("\r %s", spinners[i%len(spinners)])
			i++
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func fetchAPI(rawEndpoint Endpoint, wg *sync.WaitGroup, results chan<- ApiResponse, total int) {
	defer wg.Done()

	endpoint := rawEndpoint.endpoint

	client := &http.Client{Timeout: 60 * time.Second}

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		results <- ApiResponse{URL: endpoint, Status: "error", Error: err.Error()}
		return
	}

	token := utils.GetTokenIOT()

	// Add the authentication header
	req.Header.Add("x-access-token", token)

	loading := make(chan bool)

	go showSpinner(loading, rawEndpoint.pop, total) // Start spinner in a goroutine

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
		fmt.Printf("❌ | %s %d\n", rawEndpoint.pop, rawEndpoint.piId)
		return
	}

	fanData := getFanData(responseData.Data.Entries)

	countProcessed++

	loading <- true // Stop the spinner

	// Send results
	results <- ApiResponse{
		URL:     endpoint,
		Status:  "success",
		FanData: fanData,
		POP:     rawEndpoint.pop[:7],
		PID:     rawEndpoint.piId,
	}
}

func GetOpmsDataPipeline(limit int,
	startTime int64,
	endTime int64,
	rateLimit int,
	delaySeconds int,
	outputFile string) {

	endpoints := getEndpoints(startTime, endTime, limit)

	fmt.Printf("Found %d Endpoints\n", len(endpoints))

	var wg sync.WaitGroup
	results := make(chan ApiResponse, len(endpoints))

	fmt.Println("Starting API calls...")

	var countBatch int

	for i, endpoint := range endpoints {
		wg.Add(1)
		go fetchAPI(endpoint, &wg, results, len(endpoints))

		// Introduce a delay based on the rate limit
		if (i+1)%rateLimit == 0 {
			countBatch += rateLimit

			fmt.Printf("\nProcessing %d/%d ⚡ Rate limit reached, cooling down for %d seconds...\n", countBatch, len(endpoints), delaySeconds)

			time.Sleep(time.Duration(delaySeconds) * time.Second)
		}
	}

	wg.Wait()

	close(results)

	writeCsvFile(results, outputFile)
}
