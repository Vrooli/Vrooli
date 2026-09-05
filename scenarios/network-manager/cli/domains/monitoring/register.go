package monitoring

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "monitoring"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"MonitoringService.ListMonitoringSchedules":  h.schedules,
		"MonitoringService.UpsertMonitoringSchedule": h.scheduleSet,
		"MonitoringService.RunMonitoringCheck":       h.run,
		"MonitoringService.ListMonitoringAlerts":     h.alerts,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("monitoring: load from manifest: %w", err)
	}
	return group, nil
}
