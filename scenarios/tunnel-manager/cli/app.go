package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const (
	appName        = "tunnel-manager"
	appVersion     = "0.1.0"
	defaultAPIBase = ""
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

type App struct {
	core  *cliapp.ScenarioApp
	Stdin io.Reader // defaults to os.Stdin; override in tests
}

func NewApp() (*App, error) {
	env := cliapp.StandardScenarioEnv(appName, cliapp.ScenarioEnvOptions{
		ExtraAPIEnvVars: []string{"API_BASE_URL", "VITE_API_BASE_URL"},
	})
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name:              appName,
		Version:           appVersion,
		Description:       "Tunnel Manager CLI",
		DefaultAPIBase:    defaultAPIBase,
		APIEnvVars:        env.APIEnvVars,
		APIPortEnvVars:    env.APIPortEnvVars,
		APIPortDetector:   cliutil.DetectPortFromVrooli(appName, "API_PORT"),
		ConfigDirEnvVars:  env.ConfigDirEnvVars,
		SourceRootEnvVars: env.SourceRootEnvVars,
		TokenEnvVars:      env.TokenEnvVars,
		BuildFingerprint:  buildFingerprint,
		BuildTimestamp:    buildTimestamp,
		BuildSourceRoot:   buildSourceRoot,
		AllowAnonymous:    true,
	})
	if err != nil {
		return nil, err
	}
	app := &App{core: core, Stdin: os.Stdin}
	app.core.SetCommands(app.registerCommands())
	return app, nil
}

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}

func (a *App) registerCommands() []cliapp.CommandGroup {
	health := cliapp.CommandGroup{
		Title: "Health",
		Commands: []cliapp.Command{
			{Name: "status", NeedsAPI: true, Description: "Show tunnel health, HA connections, error rate, management mode", Run: a.cmdStatus},
			{Name: "health detailed", NeedsAPI: true, Description: "Show detailed health with per-route status", Run: a.cmdHealthDetailed},
		},
	}

	routes := cliapp.CommandGroup{
		Title: "Routes",
		Commands: []cliapp.Command{
			{Name: "routes", NeedsAPI: true, Description: "Display route manifest with live per-route status", Run: a.cmdRoutes},
			{Name: "route get", NeedsAPI: true, Description: "Get a single route by ID", Run: a.cmdRouteGet},
			{Name: "route create", NeedsAPI: true, Description: "Create a new route", Run: a.cmdRouteCreate},
			{Name: "route update", NeedsAPI: true, Description: "Update an existing route by ID", Run: a.cmdRouteUpdate},
			{Name: "route delete", NeedsAPI: true, Description: "Delete a route by ID", Run: a.cmdRouteDelete},
		},
	}

	probes := cliapp.CommandGroup{
		Title: "Probes",
		Commands: []cliapp.Command{
			{Name: "probe", NeedsAPI: true, Description: "Run all internal + external probes and report results", Run: a.cmdProbe},
			{Name: "probes history", NeedsAPI: true, Description: "Show probe history", Run: a.cmdProbesHistory},
		},
	}

	audit := cliapp.CommandGroup{
		Title: "Audit",
		Commands: []cliapp.Command{
			{Name: "audit", NeedsAPI: true, Description: "Check port compliance and report violations", Run: a.cmdAudit},
		},
	}

	metrics := cliapp.CommandGroup{
		Title: "Metrics",
		Commands: []cliapp.Command{
			{Name: "metrics latest", NeedsAPI: true, Description: "Show latest tunnel metrics", Run: a.cmdMetricsLatest},
			{Name: "metrics history", NeedsAPI: true, Description: "Show metrics history", Run: a.cmdMetricsHistory},
		},
	}

	recovery := cliapp.CommandGroup{
		Title: "Recovery",
		Commands: []cliapp.Command{
			{Name: "recovery state", NeedsAPI: true, Description: "Show current recovery engine state", Run: a.cmdRecoveryState},
			{Name: "recovery trigger", NeedsAPI: true, Description: "Trigger a recovery action", Run: a.cmdRecoveryTrigger},
			{Name: "recovery events", NeedsAPI: true, Description: "List recent recovery events", Run: a.cmdRecoveryEvents},
			{Name: "recovery circuit-reset", NeedsAPI: true, Description: "Reset the recovery circuit breaker", Run: a.cmdRecoveryCircuitReset},
		},
	}

	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
		},
	}

	return []cliapp.CommandGroup{health, routes, probes, audit, metrics, recovery, config}
}

func (a *App) apiPath(v1Path string) string {
	v1Path = strings.TrimSpace(v1Path)
	if v1Path == "" {
		return ""
	}
	if !strings.HasPrefix(v1Path, "/") {
		v1Path = "/" + v1Path
	}
	base := strings.TrimRight(strings.TrimSpace(a.core.HTTPClient.BaseURL()), "/")
	if strings.HasSuffix(base, "/api/v1") {
		return v1Path
	}
	return "/api/v1" + v1Path
}

func (a *App) useJSON(args []string) bool {
	for _, arg := range args {
		if arg == "--json" || arg == "-j" {
			return true
		}
	}
	return false
}

// --- Status Command (OT-P0-007) ---

type tunnelStatusResponse struct {
	Status       string `json:"status"`
	Systemd      string `json:"systemd"`
	Ready        string `json:"ready"`
	ReadyLatency int    `json:"ready_latency_ms,omitempty"`
	Score        int    `json:"score"`
	Message      string `json:"message,omitempty"`
	CheckedAt    string `json:"checked_at"`
}

func (a *App) cmdStatus(args []string) error {
	body, err := a.core.APIClient.Get(a.apiPath("/tunnel/health"), nil)
	if err != nil {
		// Fall back to basic health check
		body, err = a.core.APIClient.Get(a.apiPath("/health"), nil)
		if err != nil {
			return err
		}
		if a.useJSON(args) {
			cliutil.PrintJSON(body)
			return nil
		}
		var parsed healthResponse
		if unmarshalErr := json.Unmarshal(body, &parsed); unmarshalErr == nil && parsed.Status != "" {
			fmt.Printf("Status: %s\n", parsed.Status)
			fmt.Printf("Ready: %v\n", parsed.Readiness)
			return nil
		}
		cliutil.PrintJSON(body)
		return nil
	}

	if a.useJSON(args) {
		cliutil.PrintJSON(body)
		return nil
	}

	var status tunnelStatusResponse
	if err := json.Unmarshal(body, &status); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	icon := "✅"
	if status.Status == "degraded" {
		icon = "⚠️"
	} else if status.Status == "unhealthy" {
		icon = "❌"
	}

	fmt.Printf("%s Tunnel Status: %s (score: %d/100)\n", icon, status.Status, status.Score)
	fmt.Printf("  Systemd: %s\n", status.Systemd)
	fmt.Printf("  Ready endpoint: %s", status.Ready)
	if status.ReadyLatency > 0 {
		fmt.Printf(" (%dms)", status.ReadyLatency)
	}
	fmt.Println()
	if status.Message != "" {
		fmt.Printf("  Message: %s\n", status.Message)
	}
	fmt.Printf("  Checked: %s\n", status.CheckedAt)
	return nil
}

// --- Routes Command (OT-P0-008) ---

type routeResponse struct {
	ID           int    `json:"id"`
	Subdomain    string `json:"subdomain"`
	ScenarioName string `json:"scenario_name"`
	LocalPort    int    `json:"local_port"`
	HealthPath   string `json:"health_path"`
	PublicURL    string `json:"public_url"`
	Enabled      bool   `json:"enabled"`
}

func (a *App) cmdRoutes(args []string) error {
	body, err := a.core.APIClient.Get(a.apiPath("/routes"), nil)
	if err != nil {
		return err
	}

	if a.useJSON(args) {
		cliutil.PrintJSON(body)
		return nil
	}

	var routes []routeResponse
	if err := json.Unmarshal(body, &routes); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if len(routes) == 0 {
		fmt.Println("No routes configured.")
		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "SUBDOMAIN\tSCENARIO\tPORT\tENABLED\tPUBLIC URL")
	for _, r := range routes {
		enabled := "yes"
		if !r.Enabled {
			enabled = "no"
		}
		fmt.Fprintf(tw, "%s\t%s\t%d\t%s\t%s\n", r.Subdomain, r.ScenarioName, r.LocalPort, enabled, r.PublicURL)
	}
	tw.Flush()
	return nil
}

// --- Probe Command (OT-P0-009) ---

type probeResponse struct {
	Results []struct {
		Subdomain  string `json:"subdomain"`
		ProbeType  string `json:"probe_type"`
		Status     string `json:"status"`
		LatencyMs  int    `json:"latency_ms"`
		StatusCode int    `json:"status_code,omitempty"`
		ErrorMsg   string `json:"error_msg,omitempty"`
	} `json:"results"`
	Summary struct {
		Total int `json:"total"`
		Up    int `json:"up"`
		Down  int `json:"down"`
	} `json:"summary"`
}

func (a *App) cmdProbe(args []string) error {
	body, err := a.core.APIClient.Request("POST", a.apiPath("/probes"), nil, nil)
	if err != nil {
		return err
	}

	if a.useJSON(args) {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp probeResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if len(resp.Results) == 0 {
		fmt.Println("No routes to probe.")
		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "SUBDOMAIN\tTYPE\tSTATUS\tLATENCY\tHTTP\tERROR")
	for _, r := range resp.Results {
		icon := "✅"
		if r.Status != "up" {
			icon = "❌"
		}
		httpCode := ""
		if r.StatusCode > 0 {
			httpCode = fmt.Sprintf("%d", r.StatusCode)
		}
		latency := ""
		if r.LatencyMs > 0 {
			latency = fmt.Sprintf("%dms", r.LatencyMs)
		}
		errMsg := r.ErrorMsg
		if len(errMsg) > 50 {
			errMsg = errMsg[:47] + "..."
		}
		fmt.Fprintf(tw, "%s %s\t%s\t%s\t%s\t%s\t%s\n", icon, r.Subdomain, r.ProbeType, r.Status, latency, httpCode, errMsg)
	}
	tw.Flush()

	fmt.Printf("\nSummary: %d/%d up\n", resp.Summary.Up, resp.Summary.Total)
	return nil
}

// --- Audit Command (OT-P0-010) ---

type auditResponse struct {
	Results []struct {
		Subdomain    string `json:"subdomain"`
		ScenarioName string `json:"scenario_name"`
		ExpectedPort int    `json:"expected_port"`
		ActualPort   int    `json:"actual_port,omitempty"`
		Status       string `json:"status"`
		Detail       string `json:"detail,omitempty"`
	} `json:"results"`
	Total      int `json:"total"`
	Violations int `json:"violations"`
	Compliant  int `json:"compliant"`
}

func (a *App) cmdAudit(args []string) error {
	body, err := a.core.APIClient.Get(a.apiPath("/audit/ports"), nil)
	if err != nil {
		return err
	}

	if a.useJSON(args) {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp auditResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if len(resp.Results) == 0 {
		fmt.Println("No routes to audit.")
		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "SUBDOMAIN\tSCENARIO\tEXPECTED\tACTUAL\tSTATUS\tDETAIL")
	for _, r := range resp.Results {
		icon := "✅"
		if r.Status != "compliant" {
			icon = "❌"
		}
		actual := ""
		if r.ActualPort > 0 {
			actual = fmt.Sprintf("%d", r.ActualPort)
		}
		detail := r.Detail
		if len(detail) > 60 {
			detail = detail[:57] + "..."
		}
		fmt.Fprintf(tw, "%s %s\t%s\t%d\t%s\t%s\t%s\n", icon, r.Subdomain, r.ScenarioName, r.ExpectedPort, actual, r.Status, detail)
	}
	tw.Flush()

	if resp.Violations > 0 {
		fmt.Printf("\n❌ %d violation(s) found out of %d routes\n", resp.Violations, resp.Total)
	} else {
		fmt.Printf("\n✅ All %d routes compliant\n", resp.Total)
	}
	return nil
}

// --- Flag Parsing Helpers ---

func parseFlag(args []string, name string) (string, bool) {
	for i, arg := range args {
		if arg == "--"+name && i+1 < len(args) {
			return args[i+1], true
		}
	}
	return "", false
}

func parseBoolFlag(args []string, name string) bool {
	for _, arg := range args {
		if arg == "--"+name {
			return true
		}
	}
	return false
}

// --- Route CRUD Commands ---

func (a *App) cmdRouteGet(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: route get <id>")
	}
	id := args[0]

	body, err := a.core.APIClient.Get(a.apiPath("/routes/"+id), nil)
	if err != nil {
		return err
	}

	if a.useJSON(args) {
		cliutil.PrintJSON(body)
		return nil
	}

	var route routeResponse
	if err := json.Unmarshal(body, &route); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	enabled := "yes"
	if !route.Enabled {
		enabled = "no"
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "ID:\t%d\n", route.ID)
	fmt.Fprintf(tw, "Subdomain:\t%s\n", route.Subdomain)
	fmt.Fprintf(tw, "Scenario:\t%s\n", route.ScenarioName)
	fmt.Fprintf(tw, "Port:\t%d\n", route.LocalPort)
	fmt.Fprintf(tw, "Health Path:\t%s\n", route.HealthPath)
	fmt.Fprintf(tw, "Public URL:\t%s\n", route.PublicURL)
	fmt.Fprintf(tw, "Enabled:\t%s\n", enabled)
	tw.Flush()
	return nil
}

func (a *App) cmdRouteCreate(args []string) error {
	portStr, hasPort := parseFlag(args, "port")
	if !hasPort {
		return fmt.Errorf("--port is required")
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid --port value: %s", portStr)
	}

	payload := map[string]any{
		"local_port": port,
		"enabled":    true,
	}

	if v, ok := parseFlag(args, "subdomain"); ok {
		payload["subdomain"] = v
	}
	if v, ok := parseFlag(args, "scenario"); ok {
		payload["scenario_name"] = v
	}
	if v, ok := parseFlag(args, "health-path"); ok {
		payload["health_path"] = v
	}
	if v, ok := parseFlag(args, "public-url"); ok {
		payload["public_url"] = v
	}
	if v, ok := parseFlag(args, "enabled"); ok {
		payload["enabled"] = v == "true"
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/routes"), nil, payload)
	if err != nil {
		return err
	}

	if a.useJSON(args) {
		cliutil.PrintJSON(body)
		return nil
	}

	var route routeResponse
	if err := json.Unmarshal(body, &route); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Created route %d (%s -> :%d)\n", route.ID, route.Subdomain, route.LocalPort)
	return nil
}

func (a *App) cmdRouteUpdate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: route update <id> [--subdomain ...] [--scenario ...] [--port ...] [--health-path ...] [--public-url ...] [--enabled ...]")
	}
	id := args[0]

	payload := map[string]any{}

	if v, ok := parseFlag(args, "subdomain"); ok {
		payload["subdomain"] = v
	}
	if v, ok := parseFlag(args, "scenario"); ok {
		payload["scenario_name"] = v
	}
	if v, ok := parseFlag(args, "port"); ok {
		port, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid --port value: %s", v)
		}
		payload["local_port"] = port
	}
	if v, ok := parseFlag(args, "health-path"); ok {
		payload["health_path"] = v
	}
	if v, ok := parseFlag(args, "public-url"); ok {
		payload["public_url"] = v
	}
	if v, ok := parseFlag(args, "enabled"); ok {
		payload["enabled"] = v == "true"
	}

	body, err := a.core.APIClient.Request("PUT", a.apiPath("/routes/"+id), nil, payload)
	if err != nil {
		return err
	}

	if a.useJSON(args) {
		cliutil.PrintJSON(body)
		return nil
	}

	var route routeResponse
	if err := json.Unmarshal(body, &route); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Updated route %d (%s -> :%d)\n", route.ID, route.Subdomain, route.LocalPort)
	return nil
}

func (a *App) cmdRouteDelete(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: route delete <id> [--yes]")
	}
	id := args[0]

	if !parseBoolFlag(args, "yes") {
		fmt.Printf("Delete route %s? [y/N] ", id)
		scanner := bufio.NewScanner(a.Stdin)
		answer := ""
		if scanner.Scan() {
			answer = strings.TrimSpace(scanner.Text())
		}
		if answer != "y" && answer != "Y" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	_, err := a.core.APIClient.Request("DELETE", a.apiPath("/routes/"+id), nil, nil)
	if err != nil {
		return err
	}

	fmt.Printf("Deleted route %s\n", id)
	return nil
}

// --- Metrics Commands ---

func (a *App) cmdMetricsLatest(args []string) error {
	body, err := a.core.APIClient.Get(a.apiPath("/metrics/latest"), nil)
	if err != nil {
		return err
	}

	if a.useJSON(args) {
		cliutil.PrintJSON(body)
		return nil
	}

	var metrics map[string]any
	if err := json.Unmarshal(body, &metrics); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for k, v := range metrics {
		fmt.Fprintf(tw, "%s:\t%v\n", k, v)
	}
	tw.Flush()
	return nil
}

func (a *App) cmdMetricsHistory(args []string) error {
	hours := "24"
	if v, ok := parseFlag(args, "hours"); ok {
		hours = v
	}

	body, err := a.core.APIClient.Get(a.apiPath("/metrics/history?hours="+hours), nil)
	if err != nil {
		return err
	}

	if a.useJSON(args) {
		cliutil.PrintJSON(body)
		return nil
	}

	var entries []struct {
		Timestamp     string  `json:"timestamp"`
		HAConnections int     `json:"ha_connections"`
		Errors        int     `json:"errors"`
		Streams       int     `json:"streams"`
		RTT           float64 `json:"rtt_ms"`
	}
	if err := json.Unmarshal(body, &entries); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if len(entries) == 0 {
		fmt.Println("No metrics history.")
		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "TIMESTAMP\tHA CONNS\tERRORS\tSTREAMS\tRTT")
	for _, e := range entries {
		fmt.Fprintf(tw, "%s\t%d\t%d\t%d\t%.1fms\n", e.Timestamp, e.HAConnections, e.Errors, e.Streams, e.RTT)
	}
	tw.Flush()
	return nil
}

// --- Probes History Command ---

func (a *App) cmdProbesHistory(args []string) error {
	limit := "100"
	if v, ok := parseFlag(args, "limit"); ok {
		limit = v
	}

	body, err := a.core.APIClient.Get(a.apiPath("/probes/history?limit="+limit), nil)
	if err != nil {
		return err
	}

	if a.useJSON(args) {
		cliutil.PrintJSON(body)
		return nil
	}

	var entries []struct {
		Timestamp string `json:"timestamp"`
		Route     string `json:"route"`
		ProbeType string `json:"probe_type"`
		Status    string `json:"status"`
		LatencyMs int    `json:"latency_ms"`
		ErrorMsg  string `json:"error_msg,omitempty"`
	}
	if err := json.Unmarshal(body, &entries); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if len(entries) == 0 {
		fmt.Println("No probe history.")
		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "TIMESTAMP\tROUTE\tTYPE\tSTATUS\tLATENCY\tERROR")
	for _, e := range entries {
		latency := ""
		if e.LatencyMs > 0 {
			latency = fmt.Sprintf("%dms", e.LatencyMs)
		}
		errMsg := e.ErrorMsg
		if len(errMsg) > 50 {
			errMsg = errMsg[:47] + "..."
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n", e.Timestamp, e.Route, e.ProbeType, e.Status, latency, errMsg)
	}
	tw.Flush()
	return nil
}

// --- Recovery Commands ---

func (a *App) cmdRecoveryState(args []string) error {
	body, err := a.core.APIClient.Get(a.apiPath("/recovery/state"), nil)
	if err != nil {
		return err
	}

	if a.useJSON(args) {
		cliutil.PrintJSON(body)
		return nil
	}

	var state struct {
		Status         string `json:"status"`
		Failures       int    `json:"failures"`
		BackoffSeconds int    `json:"backoff_seconds"`
		CircuitState   string `json:"circuit_state"`
	}
	if err := json.Unmarshal(body, &state); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "Status:\t%s\n", state.Status)
	fmt.Fprintf(tw, "Failures:\t%d\n", state.Failures)
	fmt.Fprintf(tw, "Backoff:\t%ds\n", state.BackoffSeconds)
	fmt.Fprintf(tw, "Circuit:\t%s\n", state.CircuitState)
	tw.Flush()
	return nil
}

func (a *App) cmdRecoveryTrigger(args []string) error {
	payload := map[string]any{
		"force": parseBoolFlag(args, "force"),
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/recovery/trigger"), nil, payload)
	if err != nil {
		return err
	}

	if a.useJSON(args) {
		cliutil.PrintJSON(body)
		return nil
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for k, v := range result {
		fmt.Fprintf(tw, "%s:\t%v\n", k, v)
	}
	tw.Flush()
	return nil
}

func (a *App) cmdRecoveryEvents(args []string) error {
	limit := "50"
	if v, ok := parseFlag(args, "limit"); ok {
		limit = v
	}

	body, err := a.core.APIClient.Get(a.apiPath("/recovery/events?limit="+limit), nil)
	if err != nil {
		return err
	}

	if a.useJSON(args) {
		cliutil.PrintJSON(body)
		return nil
	}

	var events []struct {
		Timestamp string `json:"timestamp"`
		Trigger   string `json:"trigger"`
		Action    string `json:"action"`
		Outcome   string `json:"outcome"`
		Details   string `json:"details,omitempty"`
	}
	if err := json.Unmarshal(body, &events); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if len(events) == 0 {
		fmt.Println("No recovery events.")
		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "TIMESTAMP\tTRIGGER\tACTION\tOUTCOME\tDETAILS")
	for _, e := range events {
		details := e.Details
		if len(details) > 60 {
			details = details[:57] + "..."
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", e.Timestamp, e.Trigger, e.Action, e.Outcome, details)
	}
	tw.Flush()
	return nil
}

func (a *App) cmdRecoveryCircuitReset(args []string) error {
	body, err := a.core.APIClient.Request("POST", a.apiPath("/recovery/circuit/reset"), nil, nil)
	if err != nil {
		return err
	}

	if a.useJSON(args) {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Println("Circuit breaker reset.")
	return nil
}

// --- Health Detailed Command ---

func (a *App) cmdHealthDetailed(args []string) error {
	body, err := a.core.APIClient.Get(a.apiPath("/health/detailed"), nil)
	if err != nil {
		return err
	}

	if a.useJSON(args) {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp struct {
		Status string `json:"status"`
		Tunnel struct {
			Connected bool   `json:"connected"`
			Uptime    string `json:"uptime"`
		} `json:"tunnel"`
		Routes []struct {
			Subdomain string `json:"subdomain"`
			Status    string `json:"status"`
			LatencyMs int    `json:"latency_ms"`
		} `json:"routes"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Overall: %s\n", resp.Status)
	fmt.Printf("Tunnel connected: %v\n", resp.Tunnel.Connected)
	if resp.Tunnel.Uptime != "" {
		fmt.Printf("Tunnel uptime: %s\n", resp.Tunnel.Uptime)
	}

	if len(resp.Routes) > 0 {
		fmt.Println()
		tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "SUBDOMAIN\tSTATUS\tLATENCY")
		for _, r := range resp.Routes {
			latency := ""
			if r.LatencyMs > 0 {
				latency = fmt.Sprintf("%dms", r.LatencyMs)
			}
			fmt.Fprintf(tw, "%s\t%s\t%s\n", r.Subdomain, r.Status, latency)
		}
		tw.Flush()
	}
	return nil
}

// --- legacy types for fallback ---

type healthResponse struct {
	Status     string            `json:"status"`
	Service    string            `json:"service"`
	Version    string            `json:"version"`
	Readiness  bool              `json:"readiness"`
	Timestamp  string            `json:"timestamp"`
	Deps       map[string]string `json:"dependencies"`
	Error      string            `json:"error,omitempty"`
	Message    string            `json:"message,omitempty"`
	Operations map[string]any    `json:"operations,omitempty"`
}
