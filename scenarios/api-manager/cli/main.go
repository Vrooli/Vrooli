package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	version = "1.0.0"
)

type APIClient struct {
	baseURL string
	client  *http.Client
}

func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *APIClient) get(path string) ([]byte, error) {
	resp, err := c.client.Get(c.baseURL + path)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func (c *APIClient) post(path string) ([]byte, error) {
	resp, err := c.client.Post(c.baseURL+path, "application/json", nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	baseURL := os.Getenv("API_MANAGER_URL")
	if baseURL == "" {
		fmt.Fprintf(os.Stderr, "❌ API_MANAGER_URL environment variable is required\n")
		fmt.Fprintf(os.Stderr, "\nSet the API Manager URL:\n")
		fmt.Fprintf(os.Stderr, "  export API_MANAGER_URL=http://localhost:<port>\n")
		fmt.Fprintf(os.Stderr, "\nThe port is dynamically allocated by the Vrooli lifecycle system.\n")
		os.Exit(1)
	}
	client := NewAPIClient(baseURL)

	command := os.Args[1]

	switch command {
	case "help", "--help", "-h":
		printHelp()
	case "version", "--version", "-v":
		fmt.Printf("api-manager CLI v%s\n", version)
	case "health":
		handleHealth(client)
	case "list", "ls":
		handleList(client, os.Args[2:])
	case "scan":
		handleScan(client, os.Args[2:])
	case "vulnerabilities", "vulns":
		handleVulnerabilities(client, os.Args[2:])
	case "status":
		handleStatus(client)
	case "discover":
		handleDiscover(client)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Printf(`API Manager CLI v%s

USAGE:
    api-manager <command> [options]

COMMANDS:
    help                    Show this help message
    version                 Show version information
    health                  Check API manager health status
    status                  Show system status and statistics
    
    list [scenarios|endpoints|vulns]  List resources
        scenarios           List all managed scenarios
        endpoints <scenario> List endpoints for a scenario
        
    scan <scenario>         Trigger security scan for a scenario
    discover               Discover and register new scenarios
    
    vulnerabilities [scenario]  Show vulnerabilities
        (no args)           Show all open vulnerabilities
        <scenario>          Show vulnerabilities for specific scenario

ENVIRONMENT VARIABLES:
    API_MANAGER_URL        Base URL for API manager (required)

EXAMPLES:
    api-manager list scenarios
    api-manager scan my-scenario
    api-manager vulnerabilities
    api-manager vulnerabilities my-scenario
    api-manager status

`, version)
}

func handleHealth(client *APIClient) {
	body, err := client.get("/api/v1/health")
	if err != nil {
		fmt.Printf("Error checking health: %v\n", err)
		os.Exit(1)
	}

	var health map[string]interface{}
	if err := json.Unmarshal(body, &health); err != nil {
		fmt.Printf("Error parsing response: %v\n", err)
		os.Exit(1)
	}

	status := health["status"].(string)
	if status == "ok" {
		fmt.Printf("✓ API Manager is healthy\n")
		fmt.Printf("  Service: %s v%s\n", health["service"], health["version"])
		fmt.Printf("  Database: %s\n", health["database"])
	} else {
		fmt.Printf("✗ API Manager is unhealthy\n")
		fmt.Printf("  Status: %s\n", status)
		if errorMsg, exists := health["error"]; exists {
			fmt.Printf("  Error: %s\n", errorMsg)
		}
		os.Exit(1)
	}
}

func handleList(client *APIClient, args []string) {
	if len(args) == 0 {
		args = []string{"scenarios"}
	}

	switch args[0] {
	case "scenarios":
		handleListScenarios(client)
	case "endpoints":
		if len(args) < 2 {
			fmt.Println("Error: scenario name required for endpoints listing")
			fmt.Println("Usage: api-manager list endpoints <scenario>")
			os.Exit(1)
		}
		handleListEndpoints(client, args[1])
	default:
		fmt.Printf("Unknown list type: %s\n", args[0])
		fmt.Println("Available: scenarios, endpoints")
		os.Exit(1)
	}
}

func handleListScenarios(client *APIClient) {
	body, err := client.get("/api/v1/scenarios")
	if err != nil {
		fmt.Printf("Error listing scenarios: %v\n", err)
		os.Exit(1)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Printf("Error parsing response: %v\n", err)
		os.Exit(1)
	}

	scenarios := response["scenarios"].([]interface{})
	count := int(response["count"].(float64))

	fmt.Printf("Scenarios (%d total):\n\n", count)
	
	if count == 0 {
		fmt.Println("No scenarios found. Run 'api-manager discover' to scan for scenarios.")
		return
	}

	for _, s := range scenarios {
		scenario := s.(map[string]interface{})
		name := scenario["name"].(string)
		description := scenario["description"].(string)
		status := scenario["status"].(string)
		
		fmt.Printf("• %s (%s)\n", name, status)
		if description != "" {
			fmt.Printf("  %s\n", description)
		}
		
		if apiPort, exists := scenario["api_port"]; exists && apiPort != nil {
			fmt.Printf("  Port: %.0f\n", apiPort.(float64))
		}
		
		if lastScanned, exists := scenario["last_scanned"]; exists && lastScanned != nil {
			fmt.Printf("  Last scanned: %s\n", lastScanned.(string))
		}
		
		fmt.Println()
	}
}

func handleListEndpoints(client *APIClient, scenarioName string) {
	path := fmt.Sprintf("/api/v1/scenarios/%s/endpoints", scenarioName)
	body, err := client.get(path)
	if err != nil {
		fmt.Printf("Error listing endpoints for %s: %v\n", scenarioName, err)
		os.Exit(1)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Printf("Error parsing response: %v\n", err)
		os.Exit(1)
	}

	endpoints := response["endpoints"].([]interface{})
	count := int(response["count"].(float64))

	fmt.Printf("API Endpoints for %s (%d total):\n\n", scenarioName, count)
	
	if count == 0 {
		fmt.Printf("No endpoints found for %s. Run 'api-manager scan %s' to analyze the scenario.\n", scenarioName, scenarioName)
		return
	}

	for _, e := range endpoints {
		endpoint := e.(map[string]interface{})
		method := endpoint["method"].(string)
		path := endpoint["path"].(string)
		
		fmt.Printf("• %s %s\n", method, path)
		
		if handler, exists := endpoint["handler_function"]; exists && handler != nil {
			fmt.Printf("  Handler: %s\n", handler.(string))
		}
		
		if filePath, exists := endpoint["file_path"]; exists && filePath != nil {
			fmt.Printf("  File: %s", filePath.(string))
			if lineNum, exists := endpoint["line_number"]; exists && lineNum != nil {
				fmt.Printf(":%v", lineNum)
			}
			fmt.Println()
		}
		
		if description, exists := endpoint["description"]; exists && description != nil && description.(string) != "" {
			fmt.Printf("  %s\n", description.(string))
		}
		
		fmt.Println()
	}
}

func handleScan(client *APIClient, args []string) {
	if len(args) == 0 {
		fmt.Println("Error: scenario name required")
		fmt.Println("Usage: api-manager scan <scenario>")
		os.Exit(1)
	}

	scenarioName := args[0]
	path := fmt.Sprintf("/api/v1/scenarios/%s/scan", scenarioName)
	
	fmt.Printf("Initiating scan for scenario: %s\n", scenarioName)
	
	body, err := client.post(path)
	if err != nil {
		fmt.Printf("Error scanning scenario: %v\n", err)
		os.Exit(1)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Printf("Error parsing response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ %s\n", response["message"].(string))
}

func handleVulnerabilities(client *APIClient, args []string) {
	var path string
	var title string
	
	if len(args) == 0 {
		path = "/api/v1/vulnerabilities"
		title = "All Open Vulnerabilities"
	} else {
		scenarioName := args[0]
		path = fmt.Sprintf("/api/v1/vulnerabilities/%s", scenarioName)
		title = fmt.Sprintf("Vulnerabilities for %s", scenarioName)
	}

	body, err := client.get(path)
	if err != nil {
		fmt.Printf("Error getting vulnerabilities: %v\n", err)
		os.Exit(1)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Printf("Error parsing response: %v\n", err)
		os.Exit(1)
	}

	vulnerabilities := response["vulnerabilities"].([]interface{})
	count := int(response["count"].(float64))

	fmt.Printf("%s (%d total):\n\n", title, count)
	
	if count == 0 {
		fmt.Println("No vulnerabilities found.")
		return
	}

	for _, v := range vulnerabilities {
		vuln := v.(map[string]interface{})
		severity := vuln["severity"].(string)
		category := vuln["category"].(string)
		title := vuln["title"].(string)
		
		// Color code severity
		severityDisplay := severity
		switch strings.ToLower(severity) {
		case "high":
			severityDisplay = "🔴 HIGH"
		case "medium":
			severityDisplay = "🟡 MEDIUM"
		case "low":
			severityDisplay = "🟢 LOW"
		}
		
		fmt.Printf("• [%s] %s - %s\n", severityDisplay, category, title)
		
		if scenarioName, exists := vuln["scenario_name"]; exists {
			fmt.Printf("  Scenario: %s\n", scenarioName.(string))
		}
		
		if description, exists := vuln["description"]; exists && description != nil && description.(string) != "" {
			fmt.Printf("  %s\n", description.(string))
		}
		
		if filePath, exists := vuln["file_path"]; exists && filePath != nil {
			fmt.Printf("  File: %s", filePath.(string))
			if lineNum, exists := vuln["line_number"]; exists && lineNum != nil {
				fmt.Printf(":%v", lineNum)
			}
			fmt.Println()
		}
		
		if recommendation, exists := vuln["recommendation"]; exists && recommendation != nil && recommendation.(string) != "" {
			fmt.Printf("  💡 %s\n", recommendation.(string))
		}
		
		fmt.Println()
	}
}

func handleStatus(client *APIClient) {
	body, err := client.get("/api/v1/system/status")
	if err != nil {
		fmt.Printf("Error getting system status: %v\n", err)
		os.Exit(1)
	}

	var status map[string]interface{}
	if err := json.Unmarshal(body, &status); err != nil {
		fmt.Printf("Error parsing response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("API Manager System Status\n")
	fmt.Printf("========================\n\n")
	
	systemStatus := status["status"].(string)
	if systemStatus == "operational" {
		fmt.Printf("Status: ✓ %s\n", systemStatus)
	} else {
		fmt.Printf("Status: ✗ %s\n", systemStatus)
	}
	
	fmt.Printf("Service: %s v%s\n", status["service"], status["version"])
	fmt.Printf("Database: %s\n", status["database_status"])
	fmt.Printf("\nStatistics:\n")
	fmt.Printf("  Active Scenarios: %.0f\n", status["active_scenarios"].(float64))
	fmt.Printf("  Open Vulnerabilities: %.0f\n", status["open_vulnerabilities"].(float64))
	fmt.Printf("  Tracked Endpoints: %.0f\n", status["tracked_endpoints"].(float64))
	fmt.Printf("\nLast Updated: %s\n", status["timestamp"].(string))
}

func handleDiscover(client *APIClient) {
	fmt.Println("Starting scenario discovery...")
	
	body, err := client.post("/api/v1/system/discover")
	if err != nil {
		fmt.Printf("Error starting discovery: %v\n", err)
		os.Exit(1)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Printf("Error parsing response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ %s\n", response["message"].(string))
}

