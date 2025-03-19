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
	URL           string             `json:"url"`
	Status        string             `json:"status"`
	Error         string             `json:"error,omitempty"`
	ProcessedData map[string]float64 `json:"processedData,omitempty"`
	POP           string             `json:"pop,omitempty"`
	PID           int                `json:"pid,omitempty"`
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

var countProcessed = 0

const LOG_FAN_PATTERN = "https://smartpop.fpt.net/api/opms/pis/%d/log/fan-pop?tsdatesta=%d&tsdateend=%d"
const LOG_CURRENT_PATTERN = "https://smartpop.fpt.net/api/opms/pis/%d/log/device/7?lineid=7&regIds=0&tsdatesta=%d&tsdateend=%d"
const LOG_TEMP_PATTERN = "https://smartpop.fpt.net/api/opms/pis/%d/log/temperature?tsdatesta=%d&tsdateend=%d"
const LOG_AC_PATTERN = "https://smartpop.fpt.net/api/opms/pis/%d/log/air-cond?tsdatesta=%d&tsdateend=%d"

const DELTA_TIME = int64(8 * 3600) // 8 hours in seconds

func getEndpoints(timeStart int64, timeEnd int64, limit int, mode string) []Endpoint {

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

	var endpoints []Endpoint

	for _, pi := range piFolderResponse.Data {

		var pattern string

		switch mode {
		case "FAN":
			pattern = LOG_FAN_PATTERN
		case "CURRENT":
			pattern = LOG_CURRENT_PATTERN
		case "TEMP":
			pattern = LOG_TEMP_PATTERN
		case "AC":
			pattern = LOG_AC_PATTERN
		default:
			fmt.Println("Invalid mode")
			return nil
		}

		nextEndpoint := fmt.Sprintf(pattern, pi.Id, timeStart, timeEnd)

		endpoints = append(endpoints, Endpoint{piId: pi.Id, endpoint: nextEndpoint, pop: pi.Name})
	}

	if limit == -1 {
		return endpoints
	}

	sliced := endpoints[:limit]

	return sliced

}

func processData(entries []map[string]any, mode string) map[string]float64 {

	switch mode {
	case "FAN":
		{
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
	case "CURRENT":
		{
			fmt.Println("To be implemented")

			return nil
		}
	case "TEMP":
		{
			fanTemps := map[string]float64{"t1Max": -1, "t2Max": -1, "t3Max": -1, "t4Max": -1, "t1Min": 999, "t2Min": 999, "t3Min": 999, "t4Min": 999}

			for _, fan := range entries {
				for i := 0; i < 4; i++ {
					tempKey := fmt.Sprintf("temperature_%d", i)
					fanKey := fmt.Sprintf("t%d", i+1)

					// Type assertion with safety check
					if temp, tempOk := fan[tempKey].(float64); tempOk {
						if temp > fanTemps[fmt.Sprintf("%sMax", fanKey)] {
							fanTemps[fmt.Sprintf("%sMax", fanKey)] = temp
						}

						if temp < fanTemps[fmt.Sprintf("%sMin", fanKey)] {
							fanTemps[fmt.Sprintf("%sMin", fanKey)] = temp
						}
					}
				}
			}

			return fanTemps
		}
	case "AC":
		{
			fanAcs := map[string]float64{"acDurationOnByControl": 0, "acDurationOffByControl": 0, "acDurationOnByCurrent": 0, "acDurationOffByCurrent": 0}

			var avgCurrent float64

			for _, fan := range entries {
				if currentAc, currentAcOk := fan["current_ac"].(float64); currentAcOk {
					avgCurrent += currentAc
				}
			}

			avgCurrent = avgCurrent / float64(len(entries))

			for i := 1; i < len(entries); i++ {
				prev := entries[i-1]
				next := entries[i]

				if prevControlAc, prevControlAcOk := prev["control_ac"].(float64); prevControlAcOk {
					if _, nextControlAcOk := next["control_ac"].(float64); nextControlAcOk {
						if prevControlAc == 1 {
							fanAcs["acDurationOnByControl"] += next["timestamp"].(float64) - prev["timestamp"].(float64)
						} else {
							fanAcs["acDurationOffByControl"] += next["timestamp"].(float64) - prev["timestamp"].(float64)
						}
					}

					if prevCurrentAc, prevCurrentAcOk := prev["current_ac"].(float64); prevCurrentAcOk {
						if prevCurrentAc < avgCurrent {
							fanAcs["acDurationOnByCurrent"] += next["timestamp"].(float64) - prev["timestamp"].(float64)
						} else {
							fanAcs["acDurationOffByCurrent"] += next["timestamp"].(float64) - prev["timestamp"].(float64)
						}
					}
				}
			}

			fanAcs["acDurationOnByControl"] = math.Floor(fanAcs["acDurationOnByControl"] / 60)
			fanAcs["acDurationOffByControl"] = math.Floor(fanAcs["acDurationOffByControl"] / 60)
			fanAcs["acDurationOnByCurrent"] = math.Floor(fanAcs["acDurationOnByCurrent"] / 60)
			fanAcs["acDurationOffByCurrent"] = math.Floor(fanAcs["acDurationOffByCurrent"] / 60)

			return fanAcs

		}
	default:
		{
			fmt.Println("Invalid mode")
			return nil
		}
	}

}

func writeCsvFile(results []ApiResponse, outputFile string, mode string) {
	file, err := os.Create(outputFile)
	if err != nil {
		fmt.Println("Error creating CSV file:", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"PI ID", "POP", "Status"}

	switch mode {
	case "FAN":
		{
			header = append(header, "F1", "F2", "F3", "F4")
		}
	case "CURRENT":
		{
			fmt.Println("To be implemented")
		}
	case "TEMP":
		{
			header = append(header, "T1 Max", "T2 Max", "T3 Max", "T4 Max", "T1 Min", "T2 Min", "T3 Min", "T4 Min")
		}
	case "AC":
		{
			header = append(header, "AC Duration On By Control", "AC Duration Off By Control", "AC Duration On By Current", "AC Duration Off By Current")
		}
	default:
		{
			fmt.Println("Invalid mode")
		}
	}

	writer.Write(header)

	for _, result := range results {
		pId := fmt.Sprintf("%d", result.PID)

		if result.Status == "success" {
			record := []string{
				pId, result.POP, result.Status,
			}

			switch mode {
			case "FAN":
				{
					record = append(record,
						fmt.Sprintf("%.2f", result.ProcessedData["f1"]),
						fmt.Sprintf("%.2f", result.ProcessedData["f2"]),
						fmt.Sprintf("%.2f", result.ProcessedData["f3"]),
						fmt.Sprintf("%.2f", result.ProcessedData["f4"]),
					)
				}
			case "CURRENT":
				{
					fmt.Println("To be implemented")
				}
			case "TEMP":
				{
					record = append(record,
						fmt.Sprintf("%.2f", result.ProcessedData["t1Max"]),
						fmt.Sprintf("%.2f", result.ProcessedData["t2Max"]),
						fmt.Sprintf("%.2f", result.ProcessedData["t3Max"]),
						fmt.Sprintf("%.2f", result.ProcessedData["t4Max"]),
						fmt.Sprintf("%.2f", result.ProcessedData["t1Min"]),
						fmt.Sprintf("%.2f", result.ProcessedData["t2Min"]),
						fmt.Sprintf("%.2f", result.ProcessedData["t3Min"]),
						fmt.Sprintf("%.2f", result.ProcessedData["t4Min"]),
					)
				}
			case "AC":
				{
					record = append(record,
						fmt.Sprintf("%.2f", result.ProcessedData["acDurationOnByControl"]),
						fmt.Sprintf("%.2f", result.ProcessedData["acDurationOffByControl"]),
						fmt.Sprintf("%.2f", result.ProcessedData["acDurationOnByCurrent"]),
						fmt.Sprintf("%.2f", result.ProcessedData["acDurationOffByCurrent"]),
					)
				}
			default:
				{
					fmt.Println("Invalid mode")
				}
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

func fetchAPI(rawEndpoint Endpoint, wg *sync.WaitGroup, results chan<- ApiResponse, total int, mode string) {
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

	processedData := processData(responseData.Data.Entries, mode)

	countProcessed++

	loading <- true // Stop the spinner

	// Send results
	results <- ApiResponse{
		URL:           endpoint,
		Status:        "success",
		ProcessedData: processedData,
		POP:           rawEndpoint.pop[:7],
		PID:           rawEndpoint.piId,
	}
}

func GetOpmsDataPipeline(limit int, startTime int64, endTime int64, rateLimit int, delaySeconds int, outputFile string, mode string) {

	endpoints := getEndpoints(startTime, endTime, limit, mode)

	// fmt.Println(endpoints[0])

	fmt.Printf("Found %d Endpoints\n", len(endpoints))

	var wg sync.WaitGroup
	results := make(chan ApiResponse, len(endpoints))

	fmt.Println("Starting API calls...")

	var countBatch int

	for i, endpoint := range endpoints {
		wg.Add(1)
		go fetchAPI(endpoint, &wg, results, len(endpoints), mode)

		// Introduce a delay based on the rate limit
		if (i+1)%rateLimit == 0 {
			countBatch += rateLimit

			fmt.Printf("\nProcessing %d/%d ⚡ Rate limit reached, cooling down for %d seconds...\n", countBatch, len(endpoints), delaySeconds)

			time.Sleep(time.Duration(delaySeconds) * time.Second)
		}
	}

	wg.Wait()

	close(results)

	fResults := []ApiResponse{}

	for result := range results {
		fResults = append(fResults, result)
	}

	writeCsvFile(fResults, outputFile, mode)
}

func GetSingleOpmsFromLongRangee(
	startTime int64,
	endTime int64,
	piId int,
	mode string,
	rateLimit int,
	delaySeconds int,
) {
	pattern := ""

	switch mode {
	case "FAN":
		pattern = LOG_FAN_PATTERN
	case "CURRENT":
		pattern = LOG_CURRENT_PATTERN
	case "TEMP":
		pattern = LOG_TEMP_PATTERN
	case "AC":
		pattern = LOG_AC_PATTERN
	default:
		fmt.Println("Invalid mode")
		return
	}

	intervals := splitTimeRange(startTime, endTime, DELTA_TIME)

	endpoints := []Endpoint{}

	for _, interval := range intervals {
		url := fmt.Sprintf(pattern, piId, interval[0], interval[1])

		endpoints = append(endpoints, Endpoint{piId: piId, endpoint: url, pop: "SINGLE_POP"})
	}

	dateStart := time.Unix(startTime, 0).Format("2006-01-02 15:04:05")
	dateEnd := time.Unix(endTime, 0).Format("2006-01-02 15:04:05")

	fmt.Printf("📆 %s to %s\n⚡ Fetching %d APIs for %d ⌛", dateStart, dateEnd, len(endpoints), piId)

	// fetch api

	var wg sync.WaitGroup
	results := make(chan ApiResponse, len(endpoints))

	fmt.Println("Starting API calls...")

	var countBatch int

	for i, endpoint := range endpoints {
		wg.Add(1)
		go fetchAPI(endpoint, &wg, results, len(endpoints), mode)

		// Introduce a delay based on the rate limit
		if (i+1)%rateLimit == 0 {
			countBatch += rateLimit

			fmt.Printf("\nProcessing %d/%d ⚡ Rate limit reached, cooling down for %d seconds...\n", countBatch, len(endpoints), delaySeconds)

			time.Sleep(time.Duration(delaySeconds) * time.Second)
		}
	}

	wg.Wait()

	close(results)

	fileName := fmt.Sprintf("opms_%d_%s.csv", piId, mode)

	fmt.Printf("Results have been written to %s\n", fileName)

	mergedData := map[string]float64{}

	fmt.Println(len(results))

	// TODO: handle get min/max for mode = TEMP
	for result := range results {
		for key, value := range result.ProcessedData {
			if _, ok := mergedData[key]; !ok {
				mergedData[key] = 0
			}
			mergedData[key] += value
		}
	}

	fmt.Println(len(results))

	switch mode {
	case "FAN":
		{
			for key, value := range mergedData {
				mergedData[key] = math.Floor(value / float64(len(intervals)))
			}
		}
	case "TEMP":
		{
			// t1Max := mergedData["t1Max"]
			// t2Max := mergedData["t2Max"]
			// t3Max := mergedData["t3Max"]
			// t4Max := mergedData["t4Max"]

		}
	case "AC":
		{

		}
	default:
		{
			fmt.Println("Invalid mode")
		}
	}

	resultSingle := []ApiResponse{
		{
			URL:           "Single",
			Status:        "success",
			ProcessedData: mergedData,
			POP:           "SINGLE_POP",
			PID:           piId,
		},
	}

	writeCsvFile(resultSingle, fileName, mode)

}

func splitTimeRange(startTime, endTime int64, delta int64) [][2]int64 {
	var intervals [][2]int64

	for start := startTime; start < endTime; start += delta {
		subEnd := start + delta

		if subEnd > endTime {
			subEnd = endTime
		}

		intervals = append(intervals, [2]int64{start, subEnd})
	}

	return intervals
}
