package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	Version = "1.0.0"
	DefaultAPIBaseURL = "http://localhost:8080/api/v1"
)

// Configuration
type Config struct {
	APIBaseURL string `json:"api_base_url"`
	APIKey     string `json:"api_key"`
}

// API Response types (matching the API)
type ScanResponse struct {
	TotalScenarios int           `json:"total_scenarios"`
	SaaSScenarios  int           `json:"saas_scenarios"`
	NewlyDetected  int           `json:"newly_detected"`
	Scenarios      []interface{} `json:"scenarios"`
}

type GenerateResponse struct {
	LandingPageID    string   `json:"landing_page_id"`
	PreviewURL       string   `json:"preview_url"`
	DeploymentStatus string   `json:"deployment_status"`
	ABTestVariants   []string `json:"ab_test_variants"`
}

type DeployResponse struct {
	DeploymentID        string `json:"deployment_id"`
	AgentSessionID      string `json:"agent_session_id,omitempty"`
	Status              string `json:"status"`
	EstimatedCompletion string `json:"estimated_completion"`
}

type Template struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Category     string                 `json:"category"`
	SaaSType     string                 `json:"saas_type"`
	Industry     string                 `json:"industry"`
	UsageCount   int                    `json:"usage_count"`
	Rating       float64                `json:"rating"`
	ConfigSchema map[string]interface{} `json:"config_schema"`
}

// CLI Application
type CLI struct {
	config *Config
}

func NewCLI() *CLI {
	config := &Config{
		APIBaseURL: getEnvOrDefault("SAAS_LANDING_API_URL", DefaultAPIBaseURL),
		APIKey:     os.Getenv("SAAS_LANDING_API_KEY"),
	}
	
	return &CLI{config: config}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (cli *CLI) makeRequest(method, endpoint string, body interface{}) ([]byte, error) {
	var reqBody []byte
	if body != nil {
		var err error
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}
	
	url := cli.config.APIBaseURL + endpoint
	req, err := http.NewRequest(method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	if cli.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cli.config.APIKey)
	}
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}
	
	return respBody, nil
}

func (cli *CLI) checkHealth() error {
	healthURL := strings.Replace(cli.config.APIBaseURL, "/api/v1", "/health", 1)
	
	resp, err := http.Get(healthURL)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return fmt.Errorf("API unhealthy: status %d", resp.StatusCode)
	}
	
	return nil
}

// Command implementations
func (cli *CLI) cmdStatus(args []string) error {
	var jsonOutput, verbose bool
	
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonOutput = true
		case "--verbose":
			verbose = true
		}
	}
	
	err := cli.checkHealth()
	status := map[string]interface{}{
		"status":  "healthy",
		"service": "saas-landing-manager",
		"version": Version,
		"api_url": cli.config.APIBaseURL,
	}
	
	if err != nil {
		status["status"] = "unhealthy"
		status["error"] = err.Error()
	}
	
	if verbose {
		status["timestamp"] = time.Now().Format(time.RFC3339)
		status["has_api_key"] = cli.config.APIKey != ""
	}
	
	if jsonOutput {
		output, _ := json.MarshalIndent(status, "", "  ")
		fmt.Println(string(output))
	} else {
		fmt.Printf("SaaS Landing Manager CLI v%s\n", Version)
		fmt.Printf("Status: %s\n", status["status"])
		fmt.Printf("API URL: %s\n", status["api_url"])
		if verbose {
			fmt.Printf("Has API Key: %v\n", status["has_api_key"])
			fmt.Printf("Timestamp: %s\n", status["timestamp"])
		}
	}
	
	return err
}

func (cli *CLI) cmdScan(args []string) error {
	var forceRescan, dryRun, jsonOutput bool
	var scenarioFilter string
	
	for i, arg := range args {
		switch arg {
		case "--force":
			forceRescan = true
		case "--dry-run":
			dryRun = true
		case "--json":
			jsonOutput = true
		case "--scenario":
			if i+1 < len(args) {
				scenarioFilter = args[i+1]
			}
		}
	}
	
	if dryRun && !jsonOutput {
		fmt.Println("Scanning scenarios (dry run mode)...")
	}
	
	requestBody := map[string]interface{}{
		"force_rescan":     forceRescan,
		"scenario_filter":  scenarioFilter,
	}
	
	respBody, err := cli.makeRequest("POST", "/scenarios/scan", requestBody)
	if err != nil {
		return err
	}
	
	var scanResp ScanResponse
	if err := json.Unmarshal(respBody, &scanResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	
	if jsonOutput {
		fmt.Println(string(respBody))
	} else {
		fmt.Printf("Scan completed:\n")
		fmt.Printf("  Total scenarios: %d\n", scanResp.TotalScenarios)
		fmt.Printf("  SaaS scenarios: %d\n", scanResp.SaaSScenarios)
		fmt.Printf("  Newly detected: %d\n", scanResp.NewlyDetected)
		
		if len(scanResp.Scenarios) > 0 {
			fmt.Printf("\nDetected SaaS scenarios:\n")
			for i, scenario := range scanResp.Scenarios {
				if scenarioMap, ok := scenario.(map[string]interface{}); ok {
					name := scenarioMap["scenario_name"]
					confidence := scenarioMap["confidence_score"]
					fmt.Printf("  %d. %s (confidence: %.2f)\n", i+1, name, confidence)
				}
			}
		}
	}
	
	return nil
}

func (cli *CLI) cmdGenerate(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("scenario name required")
	}
	
	var templateID string
	var enableABTesting, previewOnly bool
	scenarioName := args[0]
	
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--template":
			if i+1 < len(args) {
				templateID = args[i+1]
				i++
			}
		case "--ab-test":
			enableABTesting = true
		case "--preview-only":
			previewOnly = true
		}
	}
	
	requestBody := map[string]interface{}{
		"scenario_id":        scenarioName,
		"enable_ab_testing":  enableABTesting,
	}
	
	if templateID != "" {
		requestBody["template_id"] = templateID
	}
	
	fmt.Printf("Generating landing page for scenario: %s\n", scenarioName)
	if enableABTesting {
		fmt.Println("A/B testing enabled")
	}
	
	respBody, err := cli.makeRequest("POST", "/landing-pages/generate", requestBody)
	if err != nil {
		return err
	}
	
	var genResp GenerateResponse
	if err := json.Unmarshal(respBody, &genResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	
	fmt.Printf("✓ Landing page generated successfully!\n")
	fmt.Printf("  Landing Page ID: %s\n", genResp.LandingPageID)
	fmt.Printf("  Preview URL: %s\n", genResp.PreviewURL)
	fmt.Printf("  Status: %s\n", genResp.DeploymentStatus)
	
	if len(genResp.ABTestVariants) > 0 {
		fmt.Printf("  A/B Test Variants: %s\n", strings.Join(genResp.ABTestVariants, ", "))
	}
	
	if !previewOnly {
		fmt.Printf("\nTo deploy this landing page, run:\n")
		fmt.Printf("  saas-landing-manager deploy %s %s\n", genResp.LandingPageID, scenarioName)
	}
	
	return nil
}

func (cli *CLI) cmdDeploy(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: deploy <landing_page_id> <target_scenario> [--method direct|claude-agent] [--backup]")
	}
	
	landingPageID := args[0]
	targetScenario := args[1]
	
	var deploymentMethod = "claude_agent"
	var backupExisting = true
	
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--method":
			if i+1 < len(args) {
				method := args[i+1]
				if method == "direct" || method == "claude-agent" {
					if method == "claude-agent" {
						deploymentMethod = "claude_agent"
					} else {
						deploymentMethod = "direct"
					}
				}
				i++
			}
		case "--backup":
			backupExisting = true
		case "--no-backup":
			backupExisting = false
		}
	}
	
	requestBody := map[string]interface{}{
		"target_scenario":   targetScenario,
		"deployment_method": deploymentMethod,
		"backup_existing":   backupExisting,
	}
	
	fmt.Printf("Deploying landing page %s to scenario %s\n", landingPageID, targetScenario)
	fmt.Printf("Method: %s\n", deploymentMethod)
	
	respBody, err := cli.makeRequest("POST", fmt.Sprintf("/landing-pages/%s/deploy", landingPageID), requestBody)
	if err != nil {
		return err
	}
	
	var deployResp DeployResponse
	if err := json.Unmarshal(respBody, &deployResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	
	fmt.Printf("✓ Deployment initiated!\n")
	fmt.Printf("  Deployment ID: %s\n", deployResp.DeploymentID)
	fmt.Printf("  Status: %s\n", deployResp.Status)
	
	if deployResp.AgentSessionID != "" {
		fmt.Printf("  Claude Agent Session: %s\n", deployResp.AgentSessionID)
	}
	
	if deployResp.EstimatedCompletion != "" {
		fmt.Printf("  Estimated completion: %s\n", deployResp.EstimatedCompletion)
	}
	
	return nil
}

func (cli *CLI) cmdTemplateList(args []string) error {
	var category, saasType string
	
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--category":
			if i+1 < len(args) {
				category = args[i+1]
				i++
			}
		case "--saas-type":
			if i+1 < len(args) {
				saasType = args[i+1]
				i++
			}
		}
	}
	
	endpoint := "/templates"
	params := make([]string, 0)
	if category != "" {
		params = append(params, "category="+category)
	}
	if saasType != "" {
		params = append(params, "saas_type="+saasType)
	}
	
	if len(params) > 0 {
		endpoint += "?" + strings.Join(params, "&")
	}
	
	respBody, err := cli.makeRequest("GET", endpoint, nil)
	if err != nil {
		return err
	}
	
	var response struct {
		Templates []Template `json:"templates"`
	}
	
	if err := json.Unmarshal(respBody, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	
	fmt.Printf("Available templates:\n\n")
	for i, template := range response.Templates {
		fmt.Printf("%d. %s (%s)\n", i+1, template.Name, template.ID)
		fmt.Printf("   Category: %s | SaaS Type: %s | Industry: %s\n", 
			template.Category, template.SaaSType, template.Industry)
		fmt.Printf("   Usage: %d | Rating: %.1f/5.0\n", template.UsageCount, template.Rating)
		fmt.Println()
	}
	
	return nil
}

func (cli *CLI) cmdAnalytics(args []string) error {
	var scenario, timeframe, format string
	
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--scenario":
			if i+1 < len(args) {
				scenario = args[i+1]
				i++
			}
		case "--timeframe":
			if i+1 < len(args) {
				timeframe = args[i+1]
				i++
			}
		case "--format":
			if i+1 < len(args) {
				format = args[i+1]
				i++
			}
		}
	}
	
	respBody, err := cli.makeRequest("GET", "/analytics/dashboard", nil)
	if err != nil {
		return err
	}
	
	var dashboard map[string]interface{}
	if err := json.Unmarshal(respBody, &dashboard); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	
	if format == "json" {
		fmt.Println(string(respBody))
		return nil
	}
	
	fmt.Printf("Landing Page Analytics\n")
	fmt.Printf("======================\n")
	
	if totalPages, ok := dashboard["total_pages"].(float64); ok {
		fmt.Printf("Total landing pages: %.0f\n", totalPages)
	}
	
	if activeTests, ok := dashboard["active_ab_tests"].(float64); ok {
		fmt.Printf("Active A/B tests: %.0f\n", activeTests)
	}
	
	if conversionRate, ok := dashboard["average_conversion_rate"].(float64); ok {
		fmt.Printf("Average conversion rate: %.2f%%\n", conversionRate*100)
	}
	
	return nil
}

func (cli *CLI) cmdHelp(args []string) error {
	var showAll bool
	var command string
	
	for _, arg := range args {
		switch arg {
		case "--all":
			showAll = true
		default:
			if !strings.HasPrefix(arg, "--") {
				command = arg
			}
		}
	}
	
	if command != "" {
		// Show specific command help
		return cli.showCommandHelp(command)
	}
	
	fmt.Printf("SaaS Landing Manager CLI v%s\n\n", Version)
	fmt.Printf("USAGE:\n")
	fmt.Printf("  saas-landing-manager <command> [options]\n\n")
	
	fmt.Printf("COMMANDS:\n")
	fmt.Printf("  status          Show service health and status\n")
	fmt.Printf("  scan           Scan scenarios for SaaS opportunities\n")
	fmt.Printf("  generate       Generate landing page for scenario\n")
	fmt.Printf("  deploy         Deploy landing page to scenario\n")
	fmt.Printf("  template       Manage templates (list, create, preview)\n")
	fmt.Printf("  analytics      View performance analytics\n")
	fmt.Printf("  help           Show this help message\n")
	fmt.Printf("  version        Show version information\n\n")
	
	if showAll {
		fmt.Printf("GLOBAL OPTIONS:\n")
		fmt.Printf("  --json         Output in JSON format\n")
		fmt.Printf("  --verbose      Show verbose output\n\n")
		
		fmt.Printf("ENVIRONMENT VARIABLES:\n")
		fmt.Printf("  SAAS_LANDING_API_URL    API base URL (default: %s)\n", DefaultAPIBaseURL)
		fmt.Printf("  SAAS_LANDING_API_KEY    API authentication key\n\n")
	}
	
	fmt.Printf("Examples:\n")
	fmt.Printf("  saas-landing-manager scan --force\n")
	fmt.Printf("  saas-landing-manager generate funnel-builder --ab-test\n")
	fmt.Printf("  saas-landing-manager deploy abc123 funnel-builder --method claude-agent\n")
	fmt.Printf("  saas-landing-manager template list --category base\n\n")
	
	fmt.Printf("For more details: saas-landing-manager help <command>\n")
	
	return nil
}

func (cli *CLI) showCommandHelp(command string) error {
	switch command {
	case "scan":
		fmt.Printf("USAGE: saas-landing-manager scan [options]\n\n")
		fmt.Printf("Scan scenarios directory to detect SaaS opportunities\n\n")
		fmt.Printf("OPTIONS:\n")
		fmt.Printf("  --force          Force rescan even if recently scanned\n")
		fmt.Printf("  --dry-run        Show what would be detected without saving\n")
		fmt.Printf("  --scenario NAME  Filter to specific scenario\n")
		fmt.Printf("  --json           Output results in JSON format\n")
		
	case "generate":
		fmt.Printf("USAGE: saas-landing-manager generate <scenario> [options]\n\n")
		fmt.Printf("Generate landing page for a SaaS scenario\n\n")
		fmt.Printf("OPTIONS:\n")
		fmt.Printf("  --template ID    Use specific template ID\n")
		fmt.Printf("  --ab-test        Enable A/B testing with variants\n")
		fmt.Printf("  --preview-only   Generate preview without deploying\n")
		
	case "deploy":
		fmt.Printf("USAGE: saas-landing-manager deploy <landing_page_id> <target_scenario> [options]\n\n")
		fmt.Printf("Deploy landing page to target scenario\n\n")
		fmt.Printf("OPTIONS:\n")
		fmt.Printf("  --method TYPE    Deployment method (direct, claude-agent)\n")
		fmt.Printf("  --backup         Backup existing landing page (default)\n")
		fmt.Printf("  --no-backup      Skip backing up existing files\n")
		
	default:
		return fmt.Errorf("no help available for command: %s", command)
	}
	
	return nil
}

func (cli *CLI) cmdVersion(args []string) error {
	var jsonOutput bool
	for _, arg := range args {
		if arg == "--json" {
			jsonOutput = true
		}
	}
	
	if jsonOutput {
		version := map[string]string{
			"version": Version,
			"service": "saas-landing-manager-cli",
		}
		output, _ := json.MarshalIndent(version, "", "  ")
		fmt.Println(string(output))
	} else {
		fmt.Printf("SaaS Landing Manager CLI v%s\n", Version)
	}
	
	return nil
}

// Main CLI router
func (cli *CLI) Run(args []string) error {
	if len(args) < 2 {
		return cli.cmdHelp(nil)
	}
	
	command := args[1]
	commandArgs := args[2:]
	
	switch command {
	case "status":
		return cli.cmdStatus(commandArgs)
	case "scan":
		return cli.cmdScan(commandArgs)
	case "generate":
		return cli.cmdGenerate(commandArgs)
	case "deploy":
		return cli.cmdDeploy(commandArgs)
	case "template":
		if len(commandArgs) == 0 {
			return fmt.Errorf("template subcommand required (list, create, preview)")
		}
		subcommand := commandArgs[0]
		switch subcommand {
		case "list":
			return cli.cmdTemplateList(commandArgs[1:])
		default:
			return fmt.Errorf("unknown template subcommand: %s", subcommand)
		}
	case "analytics":
		return cli.cmdAnalytics(commandArgs)
	case "help":
		return cli.cmdHelp(commandArgs)
	case "version":
		return cli.cmdVersion(commandArgs)
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

func main() {
	cli := NewCLI()
	
	if err := cli.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}