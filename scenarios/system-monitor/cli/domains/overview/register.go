package overview

import (
	"fmt"
	"os"
	"time"

	metricspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/metrics"
	settingspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/settings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	"system-monitor/cli/internal/support"
)

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Overview",
		Commands: []cliapp.Command{
			core.StandardStatusCommand(cliapp.StatusCommandOptions{
				Name:        "health",
				Description: "Check API health and dependencies",
			}),
			{
				Name:        "status",
				NeedsAPI:    true,
				Description: "Show monitor health, thresholds, and current alerts",
				Run: func(args []string) error {
					return runStatus(core, args)
				},
			},
			{
				Name:        "alerts",
				NeedsAPI:    true,
				Description: "Show active alerts from the current metrics snapshot",
				Run: func(args []string) error {
					return runAlerts(core, args)
				},
			},
			{
				Name:        "watch",
				NeedsAPI:    true,
				Description: "Stream the current metrics snapshot and alert state",
				Run: func(args []string) error {
					return runWatch(core, args)
				},
			},
			{
				Name:        "dashboard",
				Description: "Open the system-monitor dashboard in a browser",
				Run: func(args []string) error {
					return runDashboard(args)
				},
			},
		},
	}
}

type maintenanceStateResponse struct {
	Success          bool   `json:"success"`
	MaintenanceState string `json:"maintenanceState"`
}

type statusReport struct {
	cliapp.OperationalReport `json:",inline"`
	OverallStatus            string `json:"overall_status"`
	CPUThreshold             string `json:"cpu_threshold"`
	MemoryThreshold          string `json:"memory_threshold"`
	MaintenanceState         string `json:"maintenance_state"`
}

func runStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("status")
	fresh := fs.Bool("fresh", false, "Collect a fresh metrics snapshot")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	metrics, _, err := fetchCurrentMetrics(core, *fresh)
	if err != nil {
		return err
	}
	settings, err := fetchSettings(core)
	if err != nil {
		return err
	}
	maintenance, err := fetchMaintenanceState(core)
	if err != nil {
		return err
	}

	cpuThreshold, memoryThreshold, _ := support.MetricThresholds(settings)
	alerts := support.DeriveAlerts(metrics, settings)
	overall := support.OverallStatus(metrics, settings, maintenance.MaintenanceState)
	report := statusReport{
		OperationalReport: cliapp.OperationalReport{
			Status: []string{
				fmt.Sprintf("Overall status: %s", overall),
				fmt.Sprintf("Snapshot time: %s", support.FormatTimestamp(metrics.GetTimestamp())),
				fmt.Sprintf("CPU: %s", support.FormatPercent(metrics.GetCpuUsage())),
				fmt.Sprintf("Memory: %s", support.FormatPercent(metrics.GetMemoryUsage())),
				fmt.Sprintf("TCP connections: %d", metrics.GetTcpConnections()),
				fmt.Sprintf("GPU: %s", support.FormatMaybePercent(metrics.GpuUsage)),
			},
			Triage: []cliapp.TriageGroup{
				{
					Heading: "Monitoring",
					Items: []string{
						fmt.Sprintf("Active: %s", support.BoolString(settings.GetActive(), "yes", "no")),
						fmt.Sprintf("Maintenance state: %s", support.FormatMaybeString(maintenance.MaintenanceState, "inactive")),
						fmt.Sprintf("Metric interval: %ds", settings.GetMetricCollectionInterval()),
						fmt.Sprintf("Anomaly interval: %ds", settings.GetAnomalyDetectionInterval()),
						fmt.Sprintf("Cooldown: %ds", settings.GetCooldownPeriodSeconds()),
					},
				},
				{
					Heading: "Thresholds",
					Items: []string{
						fmt.Sprintf("CPU threshold: %.1f%%", cpuThreshold),
						fmt.Sprintf("Memory threshold: %.1f%%", memoryThreshold),
					},
				},
				{
					Heading: "Alerts",
					Items:   support.AlertLines(alerts),
				},
			},
			NextSteps: []string{
				"system-monitor metrics detailed",
				"system-monitor alerts",
				"system-monitor dashboard",
				"system-monitor --auto-start investigations trigger --note \"describe the issue\"",
			},
		},
		OverallStatus:    overall,
		CPUThreshold:     fmt.Sprintf("%.1f%%", cpuThreshold),
		MemoryThreshold:  fmt.Sprintf("%.1f%%", memoryThreshold),
		MaintenanceState: support.FormatMaybeString(maintenance.MaintenanceState, "inactive"),
	}

	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report.OperationalReport)
}

type alertsReport struct {
	cliapp.ListReport `json:",inline"`
	Alerts            []support.Alert `json:"alerts"`
}

func runAlerts(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("alerts")
	fresh := fs.Bool("fresh", false, "Collect a fresh metrics snapshot")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	metrics, _, err := fetchCurrentMetrics(core, *fresh)
	if err != nil {
		return err
	}
	settings, err := fetchSettings(core)
	if err != nil {
		return err
	}

	alerts := support.DeriveAlerts(metrics, settings)
	report := alertsReport{
		ListReport: cliapp.ListReport{
			Summary: []string{
				fmt.Sprintf("Snapshot time: %s", support.FormatTimestamp(metrics.GetTimestamp())),
				fmt.Sprintf("CPU: %s", support.FormatPercent(metrics.GetCpuUsage())),
				fmt.Sprintf("Memory: %s", support.FormatPercent(metrics.GetMemoryUsage())),
				fmt.Sprintf("TCP connections: %d", metrics.GetTcpConnections()),
			},
			ResultsHeading: "Alerts",
			Results:        support.AlertLines(alerts),
			RetrievalHints: []string{
				"system-monitor status",
				"system-monitor metrics current --fresh",
				"system-monitor investigations trigger --note \"investigate the current alert state\"",
			},
		},
		Alerts: alerts,
	}

	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report.ListReport)
}

func runWatch(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("watch")
	interval := fs.Int("interval", 2, "Refresh interval in seconds")
	iterations := fs.Int("iterations", 0, "Number of snapshots to render (0 means continuous)")
	fresh := fs.Bool("fresh", true, "Collect fresh metrics on each iteration")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if *interval <= 0 {
		return fmt.Errorf("--interval must be greater than 0")
	}
	if *jsonOutput && *iterations != 1 {
		return fmt.Errorf("--json requires --iterations 1")
	}

	count := 0
	for {
		metrics, raw, err := fetchCurrentMetrics(core, *fresh)
		if err != nil {
			return err
		}
		if *jsonOutput {
			return support.PrettyPrintJSON(raw)
		}
		settings, err := fetchSettings(core)
		if err != nil {
			return err
		}
		maintenance, err := fetchMaintenanceState(core)
		if err != nil {
			return err
		}

		fmt.Print("\033[H\033[2J")
		fmt.Println("System Monitor Watch")
		fmt.Printf("Time: %s\n", time.Now().Format("2006-01-02 15:04:05 MST"))
		fmt.Printf("Overall: %s\n", support.OverallStatus(metrics, settings, maintenance.MaintenanceState))
		fmt.Printf("CPU: %s\n", support.FormatPercent(metrics.GetCpuUsage()))
		fmt.Printf("Memory: %s\n", support.FormatPercent(metrics.GetMemoryUsage()))
		fmt.Printf("TCP Connections: %d\n", metrics.GetTcpConnections())
		fmt.Printf("GPU: %s\n", support.FormatMaybePercent(metrics.GpuUsage))
		fmt.Printf("Maintenance: %s\n", support.FormatMaybeString(maintenance.MaintenanceState, "inactive"))
		fmt.Println()
		fmt.Println("Alerts:")
		for _, line := range support.AlertLines(support.DeriveAlerts(metrics, settings)) {
			fmt.Printf("  %s\n", line)
		}

		count++
		if *iterations > 0 && count >= *iterations {
			return nil
		}
		time.Sleep(time.Duration(*interval) * time.Second)
	}
}

func runDashboard(args []string) error {
	fs := support.NewFlagSet("dashboard")
	printURL := fs.Bool("print-url", false, "Print the dashboard URL instead of opening it")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	result := support.DashboardResult{URL: support.DashboardURL()}
	if !*printURL {
		opened, err := support.OpenBrowser(result.URL)
		if err != nil {
			return err
		}
		result.Opened = opened
	}

	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, result)
	}
	if *printURL || !result.Opened {
		fmt.Println(result.URL)
		return nil
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result: []string{
			"Dashboard open command dispatched.",
		},
		Changes: []string{
			fmt.Sprintf("Dashboard URL: %s", result.URL),
		},
		NextCommand: []string{
			"system-monitor watch",
			"system-monitor metrics detailed",
		},
	})
}

func fetchCurrentMetrics(core *cliapp.ScenarioApp, fresh bool) (*metricspb.MetricsResponse, []byte, error) {
	var body []byte
	var err error
	if fresh {
		body, err = core.Get("/metrics/current", mapValues("fresh", "true"))
	} else {
		body, err = core.Get("/metrics/current", nil)
	}
	if err != nil {
		return nil, nil, err
	}
	var response metricspb.MetricsResponse
	if err := support.DecodeProto(body, &response); err != nil {
		return nil, nil, err
	}
	return &response, body, nil
}

func fetchSettings(core *cliapp.ScenarioApp) (*settingspb.SystemSettings, error) {
	body, err := core.Get("/settings", nil)
	if err != nil {
		return nil, err
	}
	var response settingspb.GetSettingsResponse
	if err := support.DecodeProto(body, &response); err != nil {
		return nil, err
	}
	if response.GetSettings() == nil {
		return &settingspb.SystemSettings{}, nil
	}
	return response.GetSettings(), nil
}

func fetchMaintenanceState(core *cliapp.ScenarioApp) (*maintenanceStateResponse, error) {
	body, err := core.Get("/maintenance/state", nil)
	if err != nil {
		return nil, err
	}
	var response maintenanceStateResponse
	if err := support.DecodeJSON(body, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func mapValues(key string, value string) map[string][]string {
	return map[string][]string{key: {value}}
}
