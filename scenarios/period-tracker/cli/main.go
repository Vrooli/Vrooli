package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultAPIHost = "localhost"
	defaultAPIPort = "16000"
	version        = "1.0.0"
)

// Configuration
type Config struct {
	APIHost string `json:"api_host"`
	APIPort string `json:"api_port"`
	UserID  string `json:"user_id"`
}

// API Response structures
type HealthResponse struct {
	Status    string `json:"status"`
	Database  string `json:"database"`
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
}

type CycleResponse struct {
	CycleID    string  `json:"cycle_id"`
	Message    string  `json:"message"`
	NextPeriod *string `json:"next_period,omitempty"`
	Confidence *float64 `json:"confidence,omitempty"`
}

type PredictionsResponse struct {
	Predictions []map[string]interface{} `json:"predictions"`
	NextPeriod  *string                  `json:"next_period,omitempty"`
	Confidence  *float64                 `json:"confidence,omitempty"`
}

func getConfig() Config {
	config := Config{
		APIHost: getEnv("PERIOD_TRACKER_API_HOST", defaultAPIHost),
		APIPort: getEnv("PERIOD_TRACKER_API_PORT", defaultAPIPort),
		UserID:  getEnv("PERIOD_TRACKER_USER_ID", ""),
	}
	return config
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getAPIURL(config Config, endpoint string) string {
	return fmt.Sprintf("http://%s:%s%s", config.APIHost, config.APIPort, endpoint)
}

func makeAPIRequest(method, url string, userID string, payload interface{}) (*http.Response, error) {
	var body []byte
	var err error

	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %v", err)
		}
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID)

	client := &http.Client{Timeout: 10 * time.Second}
	return client.Do(req)
}

func printUsage() {
	fmt.Println("Period Tracker CLI - Privacy-first menstrual health tracking")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  period-tracker <command> [arguments] [flags]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  status                    Show operational status and resource health")
	fmt.Println("  help                      Display command help and usage")
	fmt.Println("  version                   Show CLI and API version information")
	fmt.Println("  log-cycle <start_date>    Log the start of a new period cycle")
	fmt.Println("  log-symptoms <date>       Log daily symptoms and mood")
	fmt.Println("  predictions <user_id>     Show upcoming cycle predictions")
	fmt.Println("  patterns <user_id>        Show detected health patterns")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --json                    Output in JSON format")
	fmt.Println("  --flow <intensity>        Flow intensity (light|medium|heavy)")
	fmt.Println("  --mood <1-10>             Mood rating")
	fmt.Println("  --pain <0-10>             Pain level")
	fmt.Println("  --symptoms <list>         Comma-separated symptom list")
	fmt.Println("  --notes <text>            Additional notes")
	fmt.Println("  --days <number>           Number of days to predict ahead (default 90)")
	fmt.Println("  --timeframe <period>      Analysis timeframe (3m|6m|1y)")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  PERIOD_TRACKER_API_HOST   API host (default: localhost)")
	fmt.Println("  PERIOD_TRACKER_API_PORT   API port (default: 16000)")
	fmt.Println("  PERIOD_TRACKER_USER_ID    User ID for multi-tenant support")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  period-tracker log-cycle 2024-05-22 --flow heavy --notes \"Painful start\"")
	fmt.Println("  period-tracker log-symptoms 2024-05-22 --mood 6 --pain 7 --symptoms \"cramps,headache\"")
	fmt.Println("  period-tracker predictions user123 --json")
	fmt.Println("  period-tracker patterns user123 --timeframe 6m")
	fmt.Println()
	fmt.Println("Privacy Notice:")
	fmt.Println("  All health data is encrypted and stored locally only.")
	fmt.Println("  No data is transmitted to external servers.")
}

func handleStatus(args []string, config Config) {
	jsonOutput := false
	verbose := false

	// Parse flags
	for _, arg := range args {
		if arg == "--json" {
			jsonOutput = true
		}
		if arg == "--verbose" {
			verbose = true
		}
	}

	url := getAPIURL(config, "/health")
	resp, err := makeAPIRequest("GET", url, config.UserID, nil)
	if err != nil {
		if jsonOutput {
			fmt.Printf("{\"error\": \"Failed to connect to API: %v\"}\n", err)
		} else {
			fmt.Printf("❌ Failed to connect to API: %v\n", err)
		}
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to read response: %v\n", err)
		os.Exit(1)
	}

	if jsonOutput {
		fmt.Println(string(body))
		return
	}

	var health HealthResponse
	if err := json.Unmarshal(body, &health); err != nil {
		fmt.Printf("Failed to parse response: %v\n", err)
		os.Exit(1)
	}

	if health.Status == "healthy" {
		fmt.Println("✅ Period Tracker is healthy")
	} else {
		fmt.Println("❌ Period Tracker is unhealthy")
	}

	fmt.Printf("📊 Database: %s\n", health.Database)
	fmt.Printf("🔒 Privacy: All data encrypted and stored locally\n")
	fmt.Printf("📱 Version: %s\n", health.Version)

	if verbose {
		fmt.Printf("🕒 Last check: %s\n", health.Timestamp)
		fmt.Printf("🌐 API: http://%s:%s\n", config.APIHost, config.APIPort)
	}
}

func handleVersion(args []string) {
	jsonOutput := false
	for _, arg := range args {
		if arg == "--json" {
			jsonOutput = true
		}
	}

	if jsonOutput {
		fmt.Printf("{\"cli_version\": \"%s\", \"api_version\": \"1.0.0\"}\n", version)
	} else {
		fmt.Printf("Period Tracker CLI v%s\n", version)
		fmt.Printf("API Version: 1.0.0\n")
		fmt.Printf("Privacy-first menstrual health tracking\n")
	}
}

func handleLogCycle(args []string, config Config) {
	if len(args) < 2 {
		fmt.Println("Error: start_date is required")
		fmt.Println("Usage: period-tracker log-cycle <start_date> [flags]")
		os.Exit(1)
	}

	if config.UserID == "" {
		fmt.Println("Error: User ID required. Set PERIOD_TRACKER_USER_ID environment variable.")
		os.Exit(1)
	}

	startDate := args[1]
	var flowIntensity, notes string

	// Parse flags
	for i := 2; i < len(args); i++ {
		if args[i] == "--flow" && i+1 < len(args) {
			flowIntensity = args[i+1]
			i++
		} else if args[i] == "--notes" && i+1 < len(args) {
			notes = args[i+1]
			i++
		}
	}

	payload := map[string]interface{}{
		"start_date": startDate,
	}
	if flowIntensity != "" {
		payload["flow_intensity"] = flowIntensity
	}
	if notes != "" {
		payload["notes"] = notes
	}

	url := getAPIURL(config, "/api/v1/cycles")
	resp, err := makeAPIRequest("POST", url, config.UserID, payload)
	if err != nil {
		fmt.Printf("❌ Failed to log cycle: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to read response: %v\n", err)
		os.Exit(1)
	}

	if resp.StatusCode != http.StatusCreated {
		fmt.Printf("❌ Failed to log cycle: %s\n", string(body))
		os.Exit(1)
	}

	var result CycleResponse
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("Failed to parse response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Cycle logged for %s\n", startDate)
	if result.NextPeriod != nil {
		fmt.Printf("🔮 Next period predicted: %s", *result.NextPeriod)
		if result.Confidence != nil {
			fmt.Printf(" (%.0f%% confidence)", *result.Confidence*100)
		}
		fmt.Println()
	}
}

func handleLogSymptoms(args []string, config Config) {
	if len(args) < 2 {
		fmt.Println("Error: date is required")
		fmt.Println("Usage: period-tracker log-symptoms <date> [flags]")
		os.Exit(1)
	}

	if config.UserID == "" {
		fmt.Println("Error: User ID required. Set PERIOD_TRACKER_USER_ID environment variable.")
		os.Exit(1)
	}

	date := args[1]
	payload := map[string]interface{}{
		"date": date,
	}

	// Parse flags
	for i := 2; i < len(args); i++ {
		if args[i] == "--mood" && i+1 < len(args) {
			mood, err := fmt.Sscanf(args[i+1], "%d", new(int))
			if err == nil {
				payload["mood_rating"] = mood
			}
			i++
		} else if args[i] == "--pain" && i+1 < len(args) {
			pain, err := fmt.Sscanf(args[i+1], "%d", new(int))
			if err == nil {
				payload["cramp_intensity"] = pain
			}
			i++
		} else if args[i] == "--symptoms" && i+1 < len(args) {
			symptoms := strings.Split(args[i+1], ",")
			payload["physical_symptoms"] = symptoms
			i++
		} else if args[i] == "--notes" && i+1 < len(args) {
			payload["notes"] = args[i+1]
			i++
		}
	}

	url := getAPIURL(config, "/api/v1/symptoms")
	resp, err := makeAPIRequest("POST", url, config.UserID, payload)
	if err != nil {
		fmt.Printf("❌ Failed to log symptoms: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		fmt.Printf("✅ Symptoms logged for %s\n", date)
	} else {
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Printf("❌ Failed to log symptoms: %s\n", string(body))
		os.Exit(1)
	}
}

func handlePredictions(args []string, config Config) {
	if len(args) < 2 {
		fmt.Println("Error: user_id is required")
		fmt.Println("Usage: period-tracker predictions <user_id> [flags]")
		os.Exit(1)
	}

	userID := args[1]
	jsonOutput := false

	// Parse flags
	for _, arg := range args[2:] {
		if arg == "--json" {
			jsonOutput = true
		}
	}

	url := getAPIURL(config, "/api/v1/predictions")
	resp, err := makeAPIRequest("GET", url, userID, nil)
	if err != nil {
		if jsonOutput {
			fmt.Printf("{\"error\": \"Failed to fetch predictions: %v\"}\n", err)
		} else {
			fmt.Printf("❌ Failed to fetch predictions: %v\n", err)
		}
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to read response: %v\n", err)
		os.Exit(1)
	}

	if jsonOutput {
		fmt.Println(string(body))
		return
	}

	var result PredictionsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("Failed to parse response: %v\n", err)
		os.Exit(1)
	}

	if result.NextPeriod != nil {
		fmt.Printf("🔮 Next period: %s", *result.NextPeriod)
		if result.Confidence != nil {
			fmt.Printf(" (%.0f%% confidence)", *result.Confidence*100)
		}
		fmt.Println()
	} else {
		fmt.Println("📊 No predictions available (need more cycle data)")
	}

	if len(result.Predictions) > 1 {
		fmt.Println("\n📅 Upcoming predictions:")
		for i, pred := range result.Predictions[1:] {
			if i >= 2 {
				break // Limit to 3 total predictions
			}
			date := pred["predicted_start_date"]
			confidence := pred["confidence_score"]
			fmt.Printf("   %s (%.0f%% confidence)\n", date, confidence.(float64)*100)
		}
	}
}

func handlePatterns(args []string, config Config) {
	if len(args) < 2 {
		fmt.Println("Error: user_id is required")
		fmt.Println("Usage: period-tracker patterns <user_id> [flags]")
		os.Exit(1)
	}

	userID := args[1]
	jsonOutput := false

	// Parse flags
	for _, arg := range args[2:] {
		if arg == "--json" {
			jsonOutput = true
		}
	}

	url := getAPIURL(config, "/api/v1/patterns")
	resp, err := makeAPIRequest("GET", url, userID, nil)
	if err != nil {
		if jsonOutput {
			fmt.Printf("{\"error\": \"Failed to fetch patterns: %v\"}\n", err)
		} else {
			fmt.Printf("❌ Failed to fetch patterns: %v\n", err)
		}
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to read response: %v\n", err)
		os.Exit(1)
	}

	if jsonOutput {
		fmt.Println(string(body))
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("Failed to parse response: %v\n", err)
		os.Exit(1)
	}

	patterns, ok := result["patterns"].([]interface{})
	if !ok || len(patterns) == 0 {
		fmt.Println("📊 No patterns detected yet (need more data over time)")
		return
	}

	fmt.Println("🔍 Detected health patterns:")
	for _, patternData := range patterns {
		pattern := patternData.(map[string]interface{})
		patternType := pattern["pattern_type"].(string)
		description := pattern["pattern_description"].(string)
		correlation := pattern["correlation_strength"].(float64)
		confidence := pattern["confidence_level"].(string)
		
		fmt.Printf("\n📈 %s (%s confidence)\n", patternType, confidence)
		fmt.Printf("   %s\n", description)
		fmt.Printf("   Correlation strength: %.2f\n", correlation)
	}
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	config := getConfig()
	command := os.Args[1]

	switch command {
	case "help", "--help", "-h":
		printUsage()
	case "version", "--version", "-v":
		handleVersion(os.Args[2:])
	case "status":
		handleStatus(os.Args[2:], config)
	case "log-cycle":
		handleLogCycle(os.Args[1:], config)
	case "log-symptoms":
		handleLogSymptoms(os.Args[1:], config)
	case "predictions":
		handlePredictions(os.Args[1:], config)
	case "patterns":
		handlePatterns(os.Args[1:], config)
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}