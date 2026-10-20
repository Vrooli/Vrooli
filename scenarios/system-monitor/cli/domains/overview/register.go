package overview

import (
	"context"
	"fmt"
	"os"
	"time"

	"connectrpc.com/connect"
	alertspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/alerts"
	alertsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/alerts/alertsconnect"
	metricspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/metrics"
	metricsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/metrics/metricsconnect"
	settingspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/settings"
	settingsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/settings/settingsconnect"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	"system-monitor/cli/internal/support"
)

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	h := newHandlers(core)
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
					return runStatus(h, args)
				},
			},
			{
				Name:        "alerts",
				NeedsAPI:    true,
				Description: "Show server-evaluated alerts (active by default)",
				Run: func(args []string) error {
					return runAlerts(h, args)
				},
			},
			{
				Name:        "watch",
				NeedsAPI:    true,
				Description: "Stream the current metrics snapshot and alert state",
				Run: func(args []string) error {
					return runWatch(h, args)
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

type handlers struct {
	metrics  metricsconnect.MetricsServiceClient
	settings settingsconnect.SettingsServiceClient
	alerts   alertsconnect.AlertsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		metrics:  metricsconnect.NewMetricsServiceClient(httpClient, baseURL),
		settings: settingsconnect.NewSettingsServiceClient(httpClient, baseURL),
		alerts:   alertsconnect.NewAlertsServiceClient(httpClient, baseURL),
	}
}

type statusReport struct {
	cliapp.OperationalReport `json:",inline"`
	OverallStatus            string `json:"overall_status"`
	CPUThreshold             string `json:"cpu_threshold"`
	MemoryThreshold          string `json:"memory_threshold"`
	// MaintenanceState is the raw maintenance-orchestrator wire value
	// ("active" = monitoring running). MonitoringState is its plain reading.
	MaintenanceState string `json:"maintenance_state"`
	MonitoringState  string `json:"monitoring_state"`
	ActiveAlerts     int    `json:"active_alerts"`
}

// buildStatusReport assembles the status view from server state only; no
// alert is derived client-side.
func buildStatusReport(metrics *metricspb.MetricsResponse, settings *settingspb.SystemSettings, maintenanceState string, alerts *alertspb.ListAlertsResponse) statusReport {
	cpuThreshold, memoryThreshold, _ := support.MetricThresholds(settings)
	monitoring := support.MonitoringActive(settings, maintenanceState)
	overall := support.OverallStatus(monitoring, alerts)
	alertItems := support.AlertLines(alerts.GetAlerts())
	if !monitoring {
		alertItems = append([]string{"Monitoring is paused; alerts are not being evaluated."}, alertItems...)
	}
	alertItems = append(alertItems, support.RuleLines(alerts.GetRules())...)
	return statusReport{
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
						fmt.Sprintf("Monitoring: %s", support.MonitoringStateLabel(monitoring)),
						fmt.Sprintf("Metric interval: %ds", settings.GetMetricCollectionInterval()),
						fmt.Sprintf("Anomaly interval: %ds", settings.GetAnomalyDetectionInterval()),
						fmt.Sprintf("Cooldown: %ds", settings.GetCooldownPeriodSeconds()),
						fmt.Sprintf("Alert sustain: %ds", alerts.GetSustainSeconds()),
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
					Items:   alertItems,
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
		MaintenanceState: support.FormatMaybeString(maintenanceState, "inactive"),
		MonitoringState:  support.MonitoringStateLabel(monitoring),
		ActiveAlerts:     len(alerts.GetAlerts()),
	}
}

func runStatus(h *handlers, args []string) error {
	fs := support.NewFlagSet("status")
	fresh := fs.Bool("fresh", false, "Collect a fresh metrics snapshot")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	metrics, err := h.fetchCurrentMetrics(*fresh)
	if err != nil {
		return err
	}
	settings, err := h.fetchSettings()
	if err != nil {
		return err
	}
	maintenance, err := h.fetchMaintenanceState()
	if err != nil {
		return err
	}
	alerts, err := h.fetchAlerts(&alertspb.ListAlertsRequest{})
	if err != nil {
		return err
	}

	report := buildStatusReport(metrics, settings, maintenance.GetMaintenanceState(), alerts)
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report.OperationalReport)
}

type alertsReport struct {
	cliapp.ListReport `json:",inline"`
	Alerts            []*alertspb.Alert     `json:"alerts"`
	Rules             []*alertspb.RuleState `json:"rules"`
}

func runAlerts(h *handlers, args []string) error {
	fs := support.NewFlagSet("alerts")
	all := fs.Bool("all", false, "Include resolved alerts")
	events := fs.Bool("events", false, "Include point-in-time events (threshold violations, anomalies)")
	limit := fs.Int("limit", 50, "Maximum alerts to return")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	req := &alertspb.ListAlertsRequest{IncludeEvents: *events, Limit: int32(*limit)}
	if *all {
		req.Status = "all"
	}
	alerts, err := h.fetchAlerts(req)
	if err != nil {
		return err
	}
	if *jsonOutput {
		return cliapp.PrintProtoJSON(os.Stdout, alerts)
	}
	return cliapp.RenderListReport(os.Stdout, buildAlertsReport(alerts).ListReport)
}

func buildAlertsReport(alerts *alertspb.ListAlertsResponse) alertsReport {
	evaluated := "never (evaluator has not run)"
	if alerts.GetEvaluatedAt() != nil {
		evaluated = support.FormatTimestamp(alerts.GetEvaluatedAt())
	}
	summary := []string{
		fmt.Sprintf("Evaluated: %s", evaluated),
		fmt.Sprintf("Monitoring: %s", support.MonitoringStateLabel(alerts.GetMonitoringActive())),
		fmt.Sprintf("Sustain: %ds before an alert opens or resolves", alerts.GetSustainSeconds()),
	}
	summary = append(summary, support.RuleLines(alerts.GetRules())...)
	return alertsReport{
		ListReport: cliapp.ListReport{
			Summary:        summary,
			ResultsHeading: "Alerts",
			Results:        support.AlertLines(alerts.GetAlerts()),
			RetrievalHints: []string{
				"system-monitor alerts --all --events",
				"system-monitor status",
				"system-monitor metrics detailed",
			},
		},
		Alerts: alerts.GetAlerts(),
		Rules:  alerts.GetRules(),
	}
}

func runWatch(h *handlers, args []string) error {
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
		metricsResp, err := h.metrics.GetCurrentMetrics(context.Background(), connect.NewRequest(&metricspb.GetCurrentMetricsRequest{Fresh: *fresh}))
		if err != nil {
			return cliapp.WrapAPIError("get current metrics", err, nil)
		}
		if metricsResp == nil || metricsResp.Msg == nil || metricsResp.Msg.GetMetrics() == nil {
			return fmt.Errorf("server returned no current metrics")
		}
		metrics := metricsResp.Msg.GetMetrics()
		if *jsonOutput {
			return cliapp.PrintProtoJSON(os.Stdout, metricsResp.Msg)
		}
		settings, err := h.fetchSettings()
		if err != nil {
			return err
		}
		maintenance, err := h.fetchMaintenanceState()
		if err != nil {
			return err
		}
		maintenanceState := maintenance.GetMaintenanceState()

		alerts, err := h.fetchAlerts(&alertspb.ListAlertsRequest{})
		if err != nil {
			return err
		}
		monitoring := support.MonitoringActive(settings, maintenanceState)

		fmt.Print("\033[H\033[2J")
		fmt.Println("System Monitor Watch")
		fmt.Printf("Time: %s\n", time.Now().Format("2006-01-02 15:04:05 MST"))
		fmt.Printf("Overall: %s\n", support.OverallStatus(monitoring, alerts))
		fmt.Printf("CPU: %s\n", support.FormatPercent(metrics.GetCpuUsage()))
		fmt.Printf("Memory: %s\n", support.FormatPercent(metrics.GetMemoryUsage()))
		fmt.Printf("TCP Connections: %d\n", metrics.GetTcpConnections())
		fmt.Printf("GPU: %s\n", support.FormatMaybePercent(metrics.GpuUsage))
		fmt.Printf("Monitoring: %s\n", support.MonitoringStateLabel(monitoring))
		fmt.Println()
		fmt.Println("Alerts:")
		for _, line := range support.AlertLines(alerts.GetAlerts()) {
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

func (h *handlers) fetchCurrentMetrics(fresh bool) (*metricspb.MetricsResponse, error) {
	resp, err := h.metrics.GetCurrentMetrics(context.Background(), connect.NewRequest(&metricspb.GetCurrentMetricsRequest{Fresh: fresh}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get current metrics", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetMetrics() == nil {
		return nil, fmt.Errorf("server returned no current metrics")
	}
	return resp.Msg.GetMetrics(), nil
}

func (h *handlers) fetchAlerts(req *alertspb.ListAlertsRequest) (*alertspb.ListAlertsResponse, error) {
	resp, err := h.alerts.ListAlerts(context.Background(), connect.NewRequest(req))
	if err != nil {
		return nil, cliapp.WrapAPIError("list alerts", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no alert state")
	}
	return resp.Msg, nil
}

func (h *handlers) fetchSettings() (*settingspb.SystemSettings, error) {
	resp, err := h.settings.GetSettings(context.Background(), connect.NewRequest(&settingspb.GetSettingsRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get settings", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetSettings() == nil {
		return &settingspb.SystemSettings{}, nil
	}
	return resp.Msg.GetSettings(), nil
}

func (h *handlers) fetchMaintenanceState() (*settingspb.GetMaintenanceStateResponse, error) {
	resp, err := h.settings.GetMaintenanceState(context.Background(), connect.NewRequest(&settingspb.GetMaintenanceStateRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get maintenance state", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return &settingspb.GetMaintenanceStateResponse{}, nil
	}
	return resp.Msg, nil
}
