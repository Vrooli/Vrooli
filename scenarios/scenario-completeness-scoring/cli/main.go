package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	appVersion     = "0.1.0"
	defaultAPIBase = "http://localhost:17777"
)

type Config struct {
	APIBase string `json:"api_base"`
	Token   string `json:"token,omitempty"`
}

type App struct {
	configPath      string
	config          Config
	apiOverride     string
	httpClient      *http.Client
	configDirectory string
}

func main() {
	app, err := NewApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func NewApp() (*App, error) {
	dir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home directory: %w", err)
	}
	configDir := filepath.Join(dir, ".scenario-completeness-scoring")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return nil, fmt.Errorf("create config directory: %w", err)
	}
	configPath := filepath.Join(configDir, "config.json")
	cfg, err := loadConfig(configPath)
	if err != nil {
		return nil, err
	}
	return &App{
		configPath:      configPath,
		config:          cfg,
		apiOverride:     "",
		httpClient:      &http.Client{Timeout: 30 * time.Second},
		configDirectory: configDir,
	}, nil
}

func loadConfig(path string) (Config, error) {
	cfg := Config{}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("read config file: %w", err)
	}
	if len(data) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

func (a *App) saveConfig() error {
	payload, err := json.MarshalIndent(a.config, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := os.WriteFile(a.configPath, payload, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func (a *App) Run(args []string) error {
	if len(args) == 0 {
		a.printHelp()
		return nil
	}
	remaining := a.consumeGlobalFlags(args)
	if len(remaining) == 0 {
		a.printHelp()
		return nil
	}
	switch remaining[0] {
	case "help", "--help", "-h":
		a.printHelp()
		return nil
	case "version", "--version", "-v":
		fmt.Printf("scenario-completeness-scoring CLI version %s\n", appVersion)
		return nil
	case "configure":
		return a.cmdConfigure(remaining[1:])
	case "status":
		return a.cmdStatus()
	case "scores":
		return a.cmdScores(remaining[1:])
	case "score":
		return a.cmdScore(remaining[1:])
	case "calculate":
		return a.cmdCalculate(remaining[1:])
	case "history":
		return a.cmdHistory(remaining[1:])
	case "trends":
		return a.cmdTrends(remaining[1:])
	case "what-if":
		return a.cmdWhatIf(remaining[1:])
	case "config":
		return a.cmdConfig(remaining[1:])
	case "presets":
		return a.cmdPresets()
	case "preset":
		return a.cmdPreset(remaining[1:])
	case "collectors":
		return a.cmdCollectors()
	case "circuit-breaker":
		return a.cmdCircuitBreaker(remaining[1:])
	case "recommend":
		return a.cmdRecommend(remaining[1:])
	default:
		return fmt.Errorf("unknown command: %s", remaining[0])
	}
}

func (a *App) consumeGlobalFlags(args []string) []string {
	var rest []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--api-base" && i+1 < len(args) {
			a.apiOverride = args[i+1]
			i++
			continue
		}
		rest = args[i:]
		break
	}
	if rest == nil {
		return []string{}
	}
	return rest
}

func (a *App) cmdConfigure(args []string) error {
	if len(args) == 0 {
		payload, _ := json.MarshalIndent(a.config, "", "  ")
		fmt.Println(string(payload))
		return nil
	}
	if len(args) != 2 {
		return fmt.Errorf("usage: configure <key> <value>")
	}
	key := args[0]
	value := args[1]
	switch key {
	case "api_base":
		a.config.APIBase = value
	case "token":
		a.config.Token = value
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}
	if err := a.saveConfig(); err != nil {
		return err
	}
	fmt.Printf("Updated %s\n", key)
	return nil
}

func (a *App) cmdStatus() error {
	body, err := a.apiGet("/health", nil)
	if err != nil {
		return err
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	status := getString(parsed, "status")
	fmt.Printf("Status: %s\n", status)
	if readiness, ok := parsed["readiness"].(bool); ok {
		fmt.Printf("Ready: %v\n", readiness)
	}
	if ops, ok := parsed["operations"].(map[string]interface{}); ok && len(ops) > 0 {
		fmt.Println("Operations:")
		printJSONMap(ops, 2)
	} else {
		printJSON(body)
	}
	return nil
}

func (a *App) cmdScores(args []string) error {
	fs := flag.NewFlagSet("scores", flag.ContinueOnError)
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	body, err := a.apiGet("/api/v1/scores", nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		printJSON(body)
		return nil
	}
	var resp struct {
		Scenarios []struct {
			Scenario       string  `json:"scenario"`
			Category       string  `json:"category"`
			Score          float64 `json:"score"`
			Classification string  `json:"classification"`
			Partial        bool    `json:"partial"`
		} `json:"scenarios"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	if len(resp.Scenarios) == 0 {
		fmt.Println("No scenarios found.")
		return nil
	}
	for _, item := range resp.Scenarios {
		partial := ""
		if item.Partial {
			partial = " (partial)"
		}
		fmt.Printf("%-32s %-8s score=%5.2f classification=%s%s\n",
			item.Scenario, item.Category, item.Score, item.Classification, partial)
	}
	return nil
}

func (a *App) cmdScore(args []string) error {
	fs := flag.NewFlagSet("score", flag.ContinueOnError)
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	verbose := fs.Bool("verbose", false, "Show detailed breakdown")
	fs.BoolVar(verbose, "v", false, "Show detailed breakdown")
	metrics := fs.Bool("metrics", false, "Include raw metric counters")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *metrics {
		*verbose = true
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: score <scenario> [--json] [--verbose] [--metrics]")
	}
	scenarioName := fs.Arg(0)
	path := fmt.Sprintf("/api/v1/scores/%s", scenarioName)
	body, err := a.apiGet(path, nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		printJSON(body)
		return nil
	}
	var resp ScoreResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	formatValidationIssues(resp.ValidationAnalysis, *verbose)
	formatScoreSummary(resp)
	formatBaseMetrics(resp.Breakdown)
	formatActionPlan(resp)

	if *metrics {
		fmt.Println()
		fmt.Println("Metrics:")
		printJSONMap(resp.Metrics, 2)
	}

	formatComparisonContext(resp.ValidationAnalysis, resp.Score)
	return nil
}

func (a *App) cmdCalculate(args []string) error {
	fs := flag.NewFlagSet("calculate", flag.ContinueOnError)
	source := fs.String("source", "", "Source identifier for history tracking")
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	var tags multiValue
	fs.Var(&tags, "tag", "Tag to associate with snapshot (repeatable)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: calculate <scenario> [--source name] [--tag value] [--json]")
	}
	scenarioName := fs.Arg(0)
	payload := map[string]interface{}{}
	if *source != "" {
		payload["source"] = *source
	}
	if len(tags) > 0 {
		payload["tags"] = []string(tags)
	}
	path := fmt.Sprintf("/api/v1/scores/%s/calculate", scenarioName)
	body, err := a.apiRequest(http.MethodPost, path, nil, payload)
	if err != nil {
		return err
	}
	if *jsonOutput {
		printJSON(body)
		return nil
	}
	fmt.Printf("Recalculated %s\n", scenarioName)
	printJSON(body)
	return nil
}

func (a *App) cmdHistory(args []string) error {
	fs := flag.NewFlagSet("history", flag.ContinueOnError)
	limit := fs.Int("limit", 30, "Number of entries to show")
	source := fs.String("source", "", "Filter by source")
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	var tags multiValue
	fs.Var(&tags, "tag", "Filter by tag (repeatable)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: history <scenario> [--limit N] [--source name] [--tag value] [--json]")
	}
	scenarioName := fs.Arg(0)
	query := url.Values{}
	if *limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", *limit))
	}
	if *source != "" {
		query.Set("source", *source)
	}
	for _, tag := range tags {
		query.Add("tag", tag)
	}
	path := fmt.Sprintf("/api/v1/scores/%s/history", scenarioName)
	body, err := a.apiGet(path, query)
	if err != nil {
		return err
	}
	if *jsonOutput {
		printJSON(body)
		return nil
	}
	printJSON(body)
	return nil
}

func (a *App) cmdTrends(args []string) error {
	fs := flag.NewFlagSet("trends", flag.ContinueOnError)
	limit := fs.Int("limit", 30, "Number of entries to analyze")
	source := fs.String("source", "", "Filter by source")
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	var tags multiValue
	fs.Var(&tags, "tag", "Filter by tag")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: trends <scenario> [--limit N] [--source name] [--tag value] [--json]")
	}
	scenarioName := fs.Arg(0)
	query := url.Values{}
	if *limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", *limit))
	}
	if *source != "" {
		query.Set("source", *source)
	}
	for _, tag := range tags {
		query.Add("tag", tag)
	}
	path := fmt.Sprintf("/api/v1/scores/%s/trends", scenarioName)
	body, err := a.apiGet(path, query)
	if err != nil {
		return err
	}
	if *jsonOutput {
		printJSON(body)
		return nil
	}
	printJSON(body)
	return nil
}

func (a *App) cmdWhatIf(args []string) error {
	fs := flag.NewFlagSet("what-if", flag.ContinueOnError)
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	changesFile := fs.String("file", "", "Path to JSON file describing changes")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: what-if <scenario> [--file path] [--json]")
	}
	scenarioName := fs.Arg(0)
	payload := map[string]interface{}{
		"changes": []interface{}{},
	}
	if *changesFile != "" {
		data, err := os.ReadFile(*changesFile)
		if err != nil {
			return fmt.Errorf("read changes file: %w", err)
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			return fmt.Errorf("parse changes file: %w", err)
		}
	}
	path := fmt.Sprintf("/api/v1/scores/%s/what-if", scenarioName)
	body, err := a.apiRequest(http.MethodPost, path, nil, payload)
	if err != nil {
		return err
	}
	if *jsonOutput {
		printJSON(body)
		return nil
	}
	printJSON(body)
	return nil
}

func (a *App) cmdConfig(args []string) error {
	if len(args) > 0 && args[0] == "set" {
		return a.cmdConfigSet(args[1:])
	}
	body, err := a.apiGet("/api/v1/config", nil)
	if err != nil {
		return err
	}
	printJSON(body)
	return nil
}

func (a *App) cmdConfigSet(args []string) error {
	fs := flag.NewFlagSet("config set", flag.ContinueOnError)
	filePath := fs.String("file", "", "Path to JSON config file")
	inline := fs.String("json", "", "Inline JSON payload")
	if err := fs.Parse(args); err != nil {
		return err
	}
	var payload map[string]interface{}
	switch {
	case *filePath != "":
		data, err := os.ReadFile(*filePath)
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			return fmt.Errorf("parse json: %w", err)
		}
	case *inline != "":
		if err := json.Unmarshal([]byte(*inline), &payload); err != nil {
			return fmt.Errorf("parse json: %w", err)
		}
	default:
		return fmt.Errorf("config set requires --file or --json")
	}
	body, err := a.apiRequest(http.MethodPut, "/api/v1/config", nil, payload)
	if err != nil {
		return err
	}
	printJSON(body)
	return nil
}

func (a *App) cmdPresets() error {
	body, err := a.apiGet("/api/v1/config/presets", nil)
	if err != nil {
		return err
	}
	printJSON(body)
	return nil
}

func (a *App) cmdPreset(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: preset apply <name>")
	}
	if args[0] != "apply" {
		return fmt.Errorf("unknown preset subcommand: %s", args[0])
	}
	if len(args) < 2 {
		return fmt.Errorf("usage: preset apply <name>")
	}
	name := args[1]
	path := fmt.Sprintf("/api/v1/config/presets/%s/apply", name)
	body, err := a.apiRequest(http.MethodPost, path, nil, map[string]interface{}{})
	if err != nil {
		return err
	}
	printJSON(body)
	return nil
}

func (a *App) cmdCollectors() error {
	body, err := a.apiGet("/api/v1/health/collectors", nil)
	if err != nil {
		return err
	}
	printJSON(body)
	return nil
}

func (a *App) cmdCircuitBreaker(args []string) error {
	if len(args) > 0 && args[0] == "reset" {
		body, err := a.apiRequest(http.MethodPost, "/api/v1/health/circuit-breaker/reset", nil, map[string]interface{}{})
		if err != nil {
			return err
		}
		printJSON(body)
		return nil
	}
	body, err := a.apiGet("/api/v1/health/circuit-breaker", nil)
	if err != nil {
		return err
	}
	printJSON(body)
	return nil
}

func (a *App) cmdRecommend(args []string) error {
	fs := flag.NewFlagSet("recommend", flag.ContinueOnError)
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: recommend <scenario> [--json]")
	}
	scenarioName := fs.Arg(0)
	path := fmt.Sprintf("/api/v1/recommendations/%s", scenarioName)
	body, err := a.apiGet(path, nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		printJSON(body)
		return nil
	}
	printJSON(body)
	return nil
}

func (a *App) determineAPIBase() string {
	if a.apiOverride != "" {
		return strings.TrimRight(a.apiOverride, "/")
	}
	if env := os.Getenv("SCORING_API_BASE"); env != "" {
		return strings.TrimRight(env, "/")
	}
	if a.config.APIBase != "" {
		return strings.TrimRight(a.config.APIBase, "/")
	}
	if port := os.Getenv("API_PORT"); port != "" {
		return fmt.Sprintf("http://localhost:%s", port)
	}
	if port := os.Getenv("SCENARIO_COMPLETENESS_PORT"); port != "" {
		return fmt.Sprintf("http://localhost:%s", port)
	}
	if port := a.detectPortFromVrooli(); port != "" {
		return fmt.Sprintf("http://localhost:%s", port)
	}
	return defaultAPIBase
}

func (a *App) detectPortFromVrooli() string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "port", "scenario-completeness-scoring", "API_PORT")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func (a *App) apiGet(path string, query url.Values) ([]byte, error) {
	return a.apiRequest(http.MethodGet, path, query, nil)
}

func (a *App) apiRequest(method, path string, query url.Values, body interface{}) ([]byte, error) {
	base := a.determineAPIBase()
	endpoint := strings.TrimRight(base, "/") + path
	if query != nil && len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode payload: %w", err)
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, endpoint, reader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if a.config.Token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.config.Token))
	}
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		msg := extractErrorMessage(data)
		return nil, fmt.Errorf("api error (%d): %s", resp.StatusCode, msg)
	}
	return data, nil
}

func extractErrorMessage(data []byte) string {
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err == nil {
		if errObj, ok := parsed["error"].(map[string]interface{}); ok {
			if msg, ok := errObj["message"].(string); ok {
				return msg
			}
		}
	}
	return strings.TrimSpace(string(data))
}

func printJSON(data []byte) {
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, data, "", "  "); err != nil {
		fmt.Println(string(data))
		return
	}
	fmt.Println(pretty.String())
}

func printJSONMap(m map[string]interface{}, indent int) {
	if len(m) == 0 {
		fmt.Printf("%s(none)\n", strings.Repeat(" ", indent))
		return
	}
	prefix := strings.Repeat(" ", indent)
	for key, value := range m {
		switch v := value.(type) {
		case map[string]interface{}:
			fmt.Printf("%s%s:\n", prefix, key)
			printJSONMap(v, indent+2)
		default:
			jsonVal, err := json.Marshal(v)
			if err != nil {
				fmt.Printf("%s%s: %v\n", prefix, key, v)
				continue
			}
			fmt.Printf("%s%s: %s\n", prefix, key, string(jsonVal))
		}
	}
}

func (a *App) printHelp() {
	fmt.Print(`scenario-completeness-scoring CLI

Usage:
  scenario-completeness-scoring <command> [options]

Global Options:
  --api-base <url>   Override API base URL (default: auto-detected)

Commands:
  help                         Show this help message
  version                      Show CLI version
  configure [key value]        View or update local CLI settings (api_base, token)
  status                       Check API & collector health
  scores [--json]              List completeness scores for all scenarios
  score <scenario> [--json --verbose --metrics]
                               Show detailed score for a scenario
  calculate <scenario> [--source name] [--tag value] [--json]
                               Force score recalculation and save history
  history <scenario> [--limit N] [--source name] [--tag value] [--json]
                               View score history
  trends <scenario> [options]  View trend analysis for a scenario
  what-if <scenario> [--file path] [--json]
                               Run hypothetical improvement analysis
  config                       Show server scoring configuration
  config set --file path|--json '{...}'
                               Update server configuration
  presets                      List configuration presets
  preset apply <name>          Apply a configuration preset
  collectors                   Show collector health status
  circuit-breaker [reset]      View or reset the circuit breaker status
  recommend <scenario> [--json]
                               Get prioritized improvement recommendations
`)
}

type multiValue []string

func (m *multiValue) String() string {
	return strings.Join(*m, ",")
}

func (m *multiValue) Set(value string) error {
	*m = append(*m, value)
	return nil
}

type ScoreResponse struct {
	Scenario            string                    `json:"scenario"`
	Category            string                    `json:"category"`
	Score               float64                   `json:"score"`
	BaseScore           float64                   `json:"base_score"`
	ValidationPenalty   float64                   `json:"validation_penalty"`
	Classification      string                    `json:"classification"`
	Breakdown           ScoreBreakdown            `json:"breakdown"`
	Metrics             map[string]interface{}    `json:"metrics"`
	ValidationAnalysis  ValidationQualityAnalysis `json:"validation_analysis"`
	Recommendations     []Recommendation          `json:"recommendations"`
	PartialResult       map[string]interface{}    `json:"partial_result"`
	CalculatedTimestamp string                    `json:"calculated_at"`
}

type Recommendation struct {
	Message string  `json:"message"`
	Impact  float64 `json:"impact"`
}

func getString(m map[string]interface{}, key string) string {
	if value, ok := m[key].(string); ok {
		return value
	}
	return ""
}
