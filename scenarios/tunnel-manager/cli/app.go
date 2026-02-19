package main

import (
	"encoding/json"
	"fmt"
	"os"
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
	core *cliapp.ScenarioApp
}

func NewApp() (*App, error) {
	env := cliapp.StandardScenarioEnv(appName, cliapp.ScenarioEnvOptions{
		ExtraAPIEnvVars:     []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		ExtraAPIPortEnvVars: []string{"API_PORT"},
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
	app := &App{core: core}
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
		},
	}

	routes := cliapp.CommandGroup{
		Title: "Routes",
		Commands: []cliapp.Command{
			{Name: "routes", NeedsAPI: true, Description: "Display route manifest with live per-route status", Run: a.cmdRoutes},
		},
	}

	probes := cliapp.CommandGroup{
		Title: "Probes",
		Commands: []cliapp.Command{
			{Name: "probe", NeedsAPI: true, Description: "Run all internal + external probes and report results", Run: a.cmdProbe},
		},
	}

	audit := cliapp.CommandGroup{
		Title: "Audit",
		Commands: []cliapp.Command{
			{Name: "audit", NeedsAPI: true, Description: "Check port compliance and report violations", Run: a.cmdAudit},
		},
	}

	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
		},
	}

	return []cliapp.CommandGroup{health, routes, probes, audit, config}
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
