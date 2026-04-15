package health

import (
	"encoding/json"
	"fmt"
	"os"

	"tunnel-manager/cli/internal/flags"
	"tunnel-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

type tunnelStatusResponse struct {
	Status       string `json:"status"`
	Systemd      string `json:"systemd"`
	Ready        string `json:"ready"`
	ReadyLatency int    `json:"ready_latency_ms,omitempty"`
	Score        int    `json:"score"`
	Message      string `json:"message,omitempty"`
	CheckedAt    string `json:"checked_at"`
}

func CommandGroup(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Health",
		Commands: []cliapp.Command{
			{Name: "status", NeedsAPI: true, Description: "Show tunnel health, HA connections, error rate, and management mode", Run: func(args []string) error {
				return runStatus(deps, args)
			}},
		},
	}
}

func SubcommandGroup(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "health",
		Description: "Inspect detailed tunnel health state",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "detailed", NeedsAPI: true, Description: "Show detailed health with per-route status", Run: func(args []string) error {
				return runDetailed(deps, args)
			}},
		},
	}
}

func runStatus(deps support.Dependencies, args []string) error {
	body, err := deps.ScenarioApp().Get("/tunnel/health", nil)
	if err != nil {
		return err
	}
	if flags.HasJSONOutput(args) {
		cliutil.PrintJSON(body)
		return nil
	}

	var status tunnelStatusResponse
	if err := json.Unmarshal(body, &status); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Tunnel health: %s", status.Status),
			fmt.Sprintf("Score: %d/100", status.Score),
		},
		Triage: []cliapp.TriageGroup{
			{
				Heading: "Connectivity",
				Items: []string{
					fmt.Sprintf("Systemd: %s", status.Systemd),
					readyLine(status.Ready, status.ReadyLatency),
				},
			},
		},
		NextSteps: []string{
			"tunnel-manager health detailed",
			"tunnel-manager route list",
			"tunnel-manager probe run",
		},
	}
	if status.Message != "" || status.CheckedAt != "" {
		items := []string{}
		if status.Message != "" {
			items = append(items, "Message: "+status.Message)
		}
		if status.CheckedAt != "" {
			items = append(items, "Checked at: "+status.CheckedAt)
		}
		report.Triage = append(report.Triage, cliapp.TriageGroup{
			Heading: "Diagnostics",
			Items:   items,
		})
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func runDetailed(deps support.Dependencies, args []string) error {
	body, err := deps.ScenarioApp().Get("/health/detailed", nil)
	if err != nil {
		return err
	}
	if flags.HasJSONOutput(args) {
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

	report := cliapp.OperationalReport{
		Status: []string{
			"Overall: " + resp.Status,
			fmt.Sprintf("Tunnel connected: %t", resp.Tunnel.Connected),
		},
		NextSteps: []string{
			"tunnel-manager probe history",
			"tunnel-manager metrics latest",
		},
	}
	if resp.Tunnel.Uptime != "" {
		report.Status = append(report.Status, "Tunnel uptime: "+resp.Tunnel.Uptime)
	}
	if len(resp.Routes) > 0 {
		items := make([]string, 0, len(resp.Routes))
		for _, route := range resp.Routes {
			line := route.Subdomain + ": " + route.Status
			if route.LatencyMs > 0 {
				line += fmt.Sprintf(" (%dms)", route.LatencyMs)
			}
			items = append(items, line)
		}
		report.Triage = append(report.Triage, cliapp.TriageGroup{
			Heading: "Routes",
			Items:   items,
		})
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func readyLine(ready string, latency int) string {
	line := "Ready endpoint: " + ready
	if latency > 0 {
		line += fmt.Sprintf(" (%dms)", latency)
	}
	return line
}
