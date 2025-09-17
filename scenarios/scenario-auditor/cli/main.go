package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	version = "1.0.0"
	defaultAPIPort = "15001"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "scan":
		handleScan(os.Args[2:])
	case "rules":
		handleRules(os.Args[2:])
	case "fix":
		handleFix(os.Args[2:])
	case "health":
		handleHealth()
	case "version":
		fmt.Printf("scenario-auditor version %s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Scenario Auditor CLI - Standards enforcement for Vrooli scenarios")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  scenario-auditor <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  scan [scenario]     Scan scenario for standards violations")
	fmt.Println("  rules               List available rules")
	fmt.Println("  fix [scenario]      Generate fixes for violations") 
	fmt.Println("  health              Check API health")
	fmt.Println("  version             Show version")
	fmt.Println("  help                Show this help")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  scenario-auditor scan api-manager")
	fmt.Println("  scenario-auditor scan              # Scan current scenario")
	fmt.Println("  scenario-auditor rules --category config")
	fmt.Println("  scenario-auditor fix api-manager --auto")
	fmt.Println()
}

func handleScan(args []string) {
	scenario := "current"
	if len(args) > 0 {
		scenario = args[0]
	}

	fmt.Printf("Scanning scenario: %s\n", scenario)

	apiURL := getAPIURL()
	url := fmt.Sprintf("%s/api/v1/scan/%s", apiURL, scenario)

	resp, err := http.Post(url, "application/json", strings.NewReader("{}"))
	if err != nil {
		fmt.Printf("Error: Failed to scan scenario: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error: Failed to read response: %v\n", err)
		os.Exit(1)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: Scan failed with status %d: %s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("Error: Failed to parse response: %v\n", err)
		os.Exit(1)
	}

	printScanResults(result)
}

func handleRules(args []string) {
	var category string
	for i, arg := range args {
		if arg == "--category" && i+1 < len(args) {
			category = args[i+1]
			break
		}
	}

	fmt.Println("Available rules:")

	apiURL := getAPIURL()
	url := fmt.Sprintf("%s/api/v1/rules", apiURL)
	if category != "" {
		url += "?category=" + category
	}

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error: Failed to get rules: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error: Failed to read response: %v\n", err)
		os.Exit(1)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("Error: Failed to parse response: %v\n", err)
		os.Exit(1)
	}

	printRules(result)
}

func handleFix(args []string) {
	scenario := "current"
	autoApply := false

	for _, arg := range args {
		if arg == "--auto" {
			autoApply = true
		} else if !strings.HasPrefix(arg, "--") {
			scenario = arg
		}
	}

	fmt.Printf("Generating fixes for scenario: %s (auto-apply: %t)\n", scenario, autoApply)

	apiURL := getAPIURL()
	url := fmt.Sprintf("%s/api/v1/ai/fix/%s", apiURL, scenario)

	requestBody := fmt.Sprintf(`{"auto_apply": %t}`, autoApply)
	resp, err := http.Post(url, "application/json", strings.NewReader(requestBody))
	if err != nil {
		fmt.Printf("Error: Failed to generate fixes: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error: Failed to read response: %v\n", err)
		os.Exit(1)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: Fix generation failed with status %d: %s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("Error: Failed to parse response: %v\n", err)
		os.Exit(1)
	}

	printFixResults(result)
}

func handleHealth() {
	apiURL := getAPIURL()
	url := fmt.Sprintf("%s/api/v1/health", apiURL)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Printf("❌ API is not healthy: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Println("✅ API is healthy")
	} else {
		fmt.Printf("❌ API returned status: %d\n", resp.StatusCode)
		os.Exit(1)
	}
}

func getAPIURL() string {
	port := os.Getenv("API_PORT")
	if port == "" {
		port = defaultAPIPort
	}
	return fmt.Sprintf("http://localhost:%s", port)
}

func printScanResults(result map[string]interface{}) {
	summary, ok := result["summary"].(map[string]interface{})
	if !ok {
		fmt.Println("Error: Invalid scan result format")
		return
	}

	fmt.Printf("\n📊 Scan Results:\n")
	fmt.Printf("   Violations: %.0f\n", summary["total_violations"])
	fmt.Printf("   Score: %.1f/100\n", summary["score"])
	fmt.Printf("   Rules executed: %.0f\n", summary["rules_executed"])

	if violations, ok := result["violations"].([]interface{}); ok && len(violations) > 0 {
		fmt.Printf("\n🔍 Violations:\n")
		for _, v := range violations {
			if violation, ok := v.(map[string]interface{}); ok {
				severity := violation["severity"].(string)
				message := violation["message"].(string)
				filePath := violation["file_path"].(string)
				
				severityIcon := getSeverityIcon(severity)
				fmt.Printf("   %s %s: %s (%s)\n", severityIcon, strings.ToUpper(severity), message, filePath)
			}
		}
	} else {
		fmt.Println("\n✅ No violations found!")
	}
}

func printRules(result map[string]interface{}) {
	rules, ok := result["rules"].(map[string]interface{})
	if !ok {
		fmt.Println("Error: Invalid rules format")
		return
	}

	categories, ok := result["categories"].(map[string]interface{})
	if !ok {
		fmt.Println("Error: Invalid categories format")
		return
	}

	// Group rules by category
	rulesByCategory := make(map[string][]map[string]interface{})
	for _, rule := range rules {
		if r, ok := rule.(map[string]interface{}); ok {
			category := r["category"].(string)
			rulesByCategory[category] = append(rulesByCategory[category], r)
		}
	}

	// Print rules grouped by category
	for categoryID, categoryData := range categories {
		if cat, ok := categoryData.(map[string]interface{}); ok {
			name := cat["name"].(string)
			fmt.Printf("\n📂 %s (%s)\n", name, categoryID)
			
			if categoryRules, exists := rulesByCategory[categoryID]; exists {
				for _, rule := range categoryRules {
					name := rule["name"].(string)
					severity := rule["severity"].(string)
					enabled := rule["enabled"].(bool)
					
					status := "❌"
					if enabled {
						status = "✅"
					}
					
					severityIcon := getSeverityIcon(severity)
					fmt.Printf("   %s %s %s [%s]\n", status, severityIcon, name, severity)
				}
			}
		}
	}

	fmt.Printf("\nTotal: %.0f rules\n", result["total"])
}

func printFixResults(result map[string]interface{}) {
	success, ok := result["success"].(bool)
	if !ok || !success {
		fmt.Printf("❌ Fix generation failed: %s\n", result["error"])
		return
	}

	message := result["message"].(string)
	fmt.Printf("✅ %s\n", message)

	if fixes, ok := result["fixes"].([]interface{}); ok && len(fixes) > 0 {
		fmt.Printf("\n🔧 Generated Fixes:\n")
		for _, f := range fixes {
			if fix, ok := f.(map[string]interface{}); ok {
				filePath := fix["file_path"].(string)
				applied := fix["applied"].(bool)
				
				status := "📝"
				if applied {
					status = "✅"
				}
				
				fmt.Printf("   %s %s\n", status, filePath)
			}
		}
	}
}

func getSeverityIcon(severity string) string {
	switch severity {
	case "critical":
		return "🔴"
	case "high":
		return "🟠"
	case "medium":
		return "🟡"
	case "low":
		return "🔵"
	default:
		return "⚪"
	}
}