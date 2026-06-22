package settings

import (
	"fmt"
	"os"
	"strings"
	"system-monitor/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/domain"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "settings",
		Description: "Inspect and update monitor settings and maintenance state",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "get", Description: "Get current settings", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "update", Description: "Update monitor settings", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "reset", Description: "Reset settings to defaults", Run: func(args []string) error { return runReset(core, args) }},
			{Name: "maintenance", Description: "Get or set maintenance state", Run: func(args []string) error { return runMaintenance(core, args) }},
		},
	}
}

type maintenanceStateResponse struct {
	Success          bool   `json:"success"`
	MaintenanceState string `json:"maintenanceState"`
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("settings get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/settings", nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		return support.PrettyPrintJSON(body)
	}

	var response apipb.GetSettingsResponse
	if err := support.DecodeProto(body, &response); err != nil {
		return err
	}
	maintenance, err := getMaintenance(core)
	if err != nil {
		return err
	}
	settings := response.GetSettings()
	if settings == nil {
		settings = &domainpb.SystemSettings{}
	}
	return cliapp.RenderOperationalReport(os.Stdout, cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Monitoring active: %s", support.BoolString(settings.GetActive(), "yes", "no")),
			fmt.Sprintf("Maintenance state: %s", support.FormatMaybeString(maintenance.MaintenanceState, "inactive")),
		},
		Triage: []cliapp.TriageGroup{
			{
				Heading: "Intervals",
				Items: []string{
					fmt.Sprintf("Metric collection: %ds", settings.GetMetricCollectionInterval()),
					fmt.Sprintf("Anomaly detection: %ds", settings.GetAnomalyDetectionInterval()),
					fmt.Sprintf("Threshold check: %ds", settings.GetThresholdCheckInterval()),
					fmt.Sprintf("Cooldown: %ds", settings.GetCooldownPeriodSeconds()),
				},
			},
			{
				Heading: "Thresholds",
				Items: []string{
					fmt.Sprintf("CPU threshold: %.1f%%", settings.GetCpuThreshold()),
					fmt.Sprintf("Memory threshold: %.1f%%", settings.GetMemoryThreshold()),
					fmt.Sprintf("Disk threshold: %.1f%%", settings.GetDiskThreshold()),
				},
			},
		},
		NextSteps: []string{
			"system-monitor settings update --active true",
			"system-monitor settings maintenance --state active",
			"system-monitor settings reset",
		},
	})
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("settings update")
	active := fs.String("active", "", "Set monitoring active to true or false")
	metricInterval := fs.Int("metric-interval", -1, "Metric collection interval in seconds")
	anomalyInterval := fs.Int("anomaly-interval", -1, "Anomaly detection interval in seconds")
	thresholdInterval := fs.Int("threshold-interval", -1, "Threshold check interval in seconds")
	cooldown := fs.Int("cooldown", -1, "Cooldown period in seconds")
	cpuThreshold := fs.Float64("cpu-threshold", -1, "CPU alert threshold")
	memoryThreshold := fs.Float64("memory-threshold", -1, "Memory alert threshold")
	diskThreshold := fs.Float64("disk-threshold", -1, "Disk alert threshold")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/settings", nil)
	if err != nil {
		return err
	}
	var current apipb.GetSettingsResponse
	if err := support.DecodeProto(body, &current); err != nil {
		return err
	}
	settings := current.GetSettings()
	if settings == nil {
		settings = &domainpb.SystemSettings{}
	}

	changed := false
	if parsed, err := support.ParseOptionalBool(*active); err != nil {
		return fmt.Errorf("--active %w", err)
	} else if parsed != nil {
		settings.Active = *parsed
		changed = true
	}
	if *metricInterval >= 0 {
		settings.MetricCollectionInterval = int32(*metricInterval)
		changed = true
	}
	if *anomalyInterval >= 0 {
		settings.AnomalyDetectionInterval = int32(*anomalyInterval)
		changed = true
	}
	if *thresholdInterval >= 0 {
		settings.ThresholdCheckInterval = int32(*thresholdInterval)
		changed = true
	}
	if *cooldown >= 0 {
		settings.CooldownPeriodSeconds = int32(*cooldown)
		changed = true
	}
	if *cpuThreshold >= 0 {
		settings.CpuThreshold = *cpuThreshold
		changed = true
	}
	if *memoryThreshold >= 0 {
		settings.MemoryThreshold = *memoryThreshold
		changed = true
	}
	if *diskThreshold >= 0 {
		settings.DiskThreshold = *diskThreshold
		changed = true
	}
	if !changed {
		return fmt.Errorf("no setting changes were provided")
	}

	updateBody, err := core.Request("PUT", "/settings", nil, &apipb.UpdateSettingsRequest{Settings: settings})
	if err != nil {
		return err
	}
	if *jsonOutput {
		return support.PrettyPrintJSON(updateBody)
	}
	var updated apipb.UpdateSettingsResponse
	if err := support.DecodeProto(updateBody, &updated); err != nil {
		return err
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result: []string{"System monitor settings updated."},
		Changes: []string{
			fmt.Sprintf("Active: %s", support.BoolString(updated.GetSettings().GetActive(), "yes", "no")),
			fmt.Sprintf("Metric interval: %ds", updated.GetSettings().GetMetricCollectionInterval()),
			fmt.Sprintf("Anomaly interval: %ds", updated.GetSettings().GetAnomalyDetectionInterval()),
			fmt.Sprintf("Threshold check: %ds", updated.GetSettings().GetThresholdCheckInterval()),
		},
		NextCommand: []string{"system-monitor settings get", "system-monitor status"},
	})
}

func runReset(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("settings reset")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Request("POST", "/settings/reset", nil, nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		return support.PrettyPrintJSON(body)
	}

	var response apipb.ResetSettingsResponse
	if err := support.DecodeProto(body, &response); err != nil {
		return err
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result: []string{"System monitor settings reset to defaults."},
		Changes: []string{
			fmt.Sprintf("Active: %s", support.BoolString(response.GetSettings().GetActive(), "yes", "no")),
			fmt.Sprintf("Metric interval: %ds", response.GetSettings().GetMetricCollectionInterval()),
			fmt.Sprintf("CPU threshold: %.1f%%", response.GetSettings().GetCpuThreshold()),
		},
		NextCommand: []string{"system-monitor settings get", "system-monitor status"},
	})
}

func runMaintenance(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("settings maintenance")
	state := fs.String("state", "", "Set maintenance state to active or inactive")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	if strings.TrimSpace(*state) == "" {
		response, err := getMaintenance(core)
		if err != nil {
			return err
		}
		if *jsonOutput {
			return cliapp.PrintReportJSON(os.Stdout, response)
		}
		return cliapp.RenderOperationalReport(os.Stdout, cliapp.OperationalReport{
			Status: []string{
				fmt.Sprintf("Maintenance state: %s", support.FormatMaybeString(response.MaintenanceState, "inactive")),
			},
			Triage: []cliapp.TriageGroup{
				{Heading: "Effect", Items: []string{"Maintenance mode suppresses normal health interpretation in the CLI status view."}},
			},
			NextSteps: []string{"system-monitor settings maintenance --state active", "system-monitor settings maintenance --state inactive"},
		})
	}

	next := strings.ToLower(strings.TrimSpace(*state))
	if next != "active" && next != "inactive" {
		return fmt.Errorf("--state must be active or inactive")
	}
	body, err := core.Request("POST", "/maintenance/state", nil, &apipb.SetMaintenanceStateRequest{MaintenanceState: next})
	if err != nil {
		return err
	}
	if *jsonOutput {
		return support.PrettyPrintJSON(body)
	}
	var response apipb.SetMaintenanceStateResponse
	if err := support.DecodeProto(body, &response); err != nil {
		return err
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result:      []string{"Maintenance state updated."},
		Changes:     []string{fmt.Sprintf("New maintenance state: %s", response.GetMaintenanceState())},
		NextCommand: []string{"system-monitor status", "system-monitor settings get"},
	})
}

func getMaintenance(core *cliapp.ScenarioApp) (*maintenanceStateResponse, error) {
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
