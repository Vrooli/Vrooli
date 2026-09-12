package settings

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	settingspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/settings"
	settingsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/settings/settingsconnect"

	"github.com/vrooli/cli-core/cliapp"

	"system-monitor/cli/internal/support"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "settings",
		Description: "Inspect and update monitor settings and maintenance state",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "get", Description: "Get current settings", RunCtx: h.get},
			{Name: "update", Description: "Update monitor settings", Args: updateArgs(), RunCtx: h.update},
			{Name: "reset", Description: "Reset settings to defaults", RunCtx: h.reset},
			{Name: "maintenance", Description: "Get or set maintenance state", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "state", Description: "Set maintenance state to active or inactive"}}}, RunCtx: h.maintenance},
		},
	}
}

type handlers struct {
	client settingsconnect.SettingsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		client: settingsconnect.NewSettingsServiceClient(httpClient, baseURL),
	}
}

func updateArgs() cliapp.ArgSchema {
	return cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "active", Description: "Set monitoring active to true or false"},
		{Name: "metric-interval", Description: "Metric collection interval in seconds", Default: "-1"},
		{Name: "anomaly-interval", Description: "Anomaly detection interval in seconds", Default: "-1"},
		{Name: "threshold-interval", Description: "Threshold check interval in seconds", Default: "-1"},
		{Name: "cooldown", Description: "Cooldown period in seconds", Default: "-1"},
		{Name: "cpu-threshold", Description: "CPU alert threshold", Default: "-1"},
		{Name: "memory-threshold", Description: "Memory alert threshold", Default: "-1"},
		{Name: "disk-threshold", Description: "Disk alert threshold", Default: "-1"},
	}}
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	resp, err := h.client.GetSettings(context.Background(), connect.NewRequest(&settingspb.GetSettingsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get settings", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no settings response")
	}
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
	}

	maintenance, err := h.client.GetMaintenanceState(context.Background(), connect.NewRequest(&settingspb.GetMaintenanceStateRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get maintenance state", err, nil)
	}
	settings := resp.Msg.GetSettings()
	if settings == nil {
		settings = &settingspb.SystemSettings{}
	}
	return ctx.RenderOperational(cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Monitoring active: %s", support.BoolString(settings.GetActive(), "yes", "no")),
			fmt.Sprintf("Maintenance state: %s", support.FormatMaybeString(maintenance.Msg.GetMaintenanceState(), "inactive")),
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

func (h *handlers) update(ctx cliapp.RunContext) error {
	current, err := h.client.GetSettings(context.Background(), connect.NewRequest(&settingspb.GetSettingsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get current settings", err, nil)
	}
	if current == nil || current.Msg == nil {
		return fmt.Errorf("server returned no current settings")
	}
	settings := current.Msg.GetSettings()
	if settings == nil {
		settings = &settingspb.SystemSettings{}
	}

	changed, err := applySettingUpdates(ctx, settings)
	if err != nil {
		return err
	}
	if !changed {
		return fmt.Errorf("no setting changes were provided")
	}

	updated, err := h.client.UpdateSettings(context.Background(), connect.NewRequest(&settingspb.UpdateSettingsRequest{Settings: settings}))
	if err != nil {
		return cliapp.WrapAPIError("update settings", err, nil)
	}
	if updated == nil || updated.Msg == nil || updated.Msg.GetSettings() == nil {
		return fmt.Errorf("server returned no updated settings")
	}
	return cliapp.RenderProtoMutation(ctx, updated.Msg, cliapp.MutationReport{
		Result: []string{"System monitor settings updated."},
		Changes: []string{
			fmt.Sprintf("Active: %s", support.BoolString(updated.Msg.GetSettings().GetActive(), "yes", "no")),
			fmt.Sprintf("Metric interval: %ds", updated.Msg.GetSettings().GetMetricCollectionInterval()),
			fmt.Sprintf("Anomaly interval: %ds", updated.Msg.GetSettings().GetAnomalyDetectionInterval()),
			fmt.Sprintf("Threshold check: %ds", updated.Msg.GetSettings().GetThresholdCheckInterval()),
		},
		NextCommand: []string{"system-monitor settings get", "system-monitor status"},
	})
}

func (h *handlers) reset(ctx cliapp.RunContext) error {
	resp, err := h.client.ResetSettings(context.Background(), connect.NewRequest(&settingspb.ResetSettingsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("reset settings", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetSettings() == nil {
		return fmt.Errorf("server returned no reset settings")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{"System monitor settings reset to defaults."},
		Changes: []string{
			fmt.Sprintf("Active: %s", support.BoolString(resp.Msg.GetSettings().GetActive(), "yes", "no")),
			fmt.Sprintf("Metric interval: %ds", resp.Msg.GetSettings().GetMetricCollectionInterval()),
			fmt.Sprintf("CPU threshold: %.1f%%", resp.Msg.GetSettings().GetCpuThreshold()),
		},
		NextCommand: []string{"system-monitor settings get", "system-monitor status"},
	})
}

func (h *handlers) maintenance(ctx cliapp.RunContext) error {
	state := strings.TrimSpace(ctx.Flag("state"))
	if state == "" {
		response, err := h.client.GetMaintenanceState(context.Background(), connect.NewRequest(&settingspb.GetMaintenanceStateRequest{}))
		if err != nil {
			return cliapp.WrapAPIError("get maintenance state", err, nil)
		}
		if response == nil || response.Msg == nil {
			return fmt.Errorf("server returned no maintenance state")
		}
		return cliapp.RenderProtoList(ctx, response.Msg, cliapp.ListReport{
			Summary: []string{
				fmt.Sprintf("Maintenance state: %s", support.FormatMaybeString(response.Msg.GetMaintenanceState(), "inactive")),
			},
			ResultsHeading: "Maintenance",
			Results:        []string{"Maintenance mode suppresses normal health interpretation in the CLI status view."},
			RetrievalHints: []string{"system-monitor settings maintenance --state active", "system-monitor settings maintenance --state inactive"},
		})
	}

	next := strings.ToLower(state)
	if next != "active" && next != "inactive" {
		return fmt.Errorf("--state must be active or inactive")
	}
	response, err := h.client.SetMaintenanceState(context.Background(), connect.NewRequest(&settingspb.SetMaintenanceStateRequest{MaintenanceState: next}))
	if err != nil {
		return cliapp.WrapAPIError("set maintenance state", err, nil)
	}
	if response == nil || response.Msg == nil {
		return fmt.Errorf("server returned no maintenance state")
	}
	return cliapp.RenderProtoMutation(ctx, response.Msg, cliapp.MutationReport{
		Result:      []string{"Maintenance state updated."},
		Changes:     []string{fmt.Sprintf("New maintenance state: %s", response.Msg.GetMaintenanceState())},
		NextCommand: []string{"system-monitor status", "system-monitor settings get"},
	})
}

type int32Setting struct {
	flag  string
	apply func(int32)
}

type float64Setting struct {
	flag  string
	apply func(float64)
}

func applySettingUpdates(ctx cliapp.RunContext, settings *settingspb.SystemSettings) (bool, error) {
	activeChanged, err := applyActiveSetting(ctx, settings)
	if err != nil {
		return false, err
	}
	intChanged, err := applyInt32Settings(ctx, settings)
	if err != nil {
		return false, err
	}
	floatChanged, err := applyFloat64Settings(ctx, settings)
	if err != nil {
		return false, err
	}
	return activeChanged || intChanged || floatChanged, nil
}

func applyActiveSetting(ctx cliapp.RunContext, settings *settingspb.SystemSettings) (bool, error) {
	parsed, err := support.ParseOptionalBool(ctx.Flag("active"))
	if err != nil {
		return false, fmt.Errorf("--active %w", err)
	}
	if parsed == nil {
		return false, nil
	}
	settings.Active = *parsed
	return true, nil
}

func applyInt32Settings(ctx cliapp.RunContext, settings *settingspb.SystemSettings) (bool, error) {
	changed := false
	for _, item := range []int32Setting{
		{flag: "metric-interval", apply: func(value int32) { settings.MetricCollectionInterval = value }},
		{flag: "anomaly-interval", apply: func(value int32) { settings.AnomalyDetectionInterval = value }},
		{flag: "threshold-interval", apply: func(value int32) { settings.ThresholdCheckInterval = value }},
		{flag: "cooldown", apply: func(value int32) { settings.CooldownPeriodSeconds = value }},
	} {
		value, ok, err := optionalInt32(ctx, item.flag)
		if err != nil {
			return false, err
		}
		if ok {
			item.apply(value)
			changed = true
		}
	}
	return changed, nil
}

func applyFloat64Settings(ctx cliapp.RunContext, settings *settingspb.SystemSettings) (bool, error) {
	changed := false
	for _, item := range []float64Setting{
		{flag: "cpu-threshold", apply: func(value float64) { settings.CpuThreshold = value }},
		{flag: "memory-threshold", apply: func(value float64) { settings.MemoryThreshold = value }},
		{flag: "disk-threshold", apply: func(value float64) { settings.DiskThreshold = value }},
	} {
		value, ok, err := optionalFloat64(ctx, item.flag)
		if err != nil {
			return false, err
		}
		if ok {
			item.apply(value)
			changed = true
		}
	}
	return changed, nil
}

func optionalInt32(ctx cliapp.RunContext, name string) (int32, bool, error) {
	raw := strings.TrimSpace(ctx.Flag(name))
	if raw == "" || raw == "-1" {
		return 0, false, nil
	}
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || value < 0 {
		return 0, false, fmt.Errorf("--%s must be a non-negative integer", name)
	}
	return int32(value), true, nil
}

func optionalFloat64(ctx cliapp.RunContext, name string) (float64, bool, error) {
	raw := strings.TrimSpace(ctx.Flag(name))
	if raw == "" || raw == "-1" {
		return 0, false, nil
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || value < 0 {
		return 0, false, fmt.Errorf("--%s must be a non-negative number", name)
	}
	return value, true, nil
}
