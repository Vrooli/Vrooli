package smoketest

import (
	"encoding/json"
	"fmt"
	"os"
)

type phasePair struct {
	name       string
	start, end LaunchEventName
}

var launchPhasePairs = []phasePair{
	{name: "process_to_splash_first_paint", start: EventDemoSpawn, end: EventSplashFirstPaint},
	{name: "splash_to_app_ready", start: EventSplashFirstPaint, end: EventAppReady},
	{name: "runtime_spawn_to_ready", start: EventRuntimeSpawned, end: EventRuntimeReady},
	{name: "server_to_app_ready", start: EventServerReady, end: EventAppReady},
}

func launchPhaseDurations(path string, kind LaunchRunKind) ([]PerformancePhase, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s launch trace: %w", kind, err)
	}
	var trace LaunchTrace
	if err := json.Unmarshal(data, &trace); err != nil {
		return nil, fmt.Errorf("decode %s launch trace: %w", kind, err)
	}
	if err := trace.Validate(); err != nil {
		return nil, fmt.Errorf("validate %s launch trace: %w", kind, err)
	}
	result := make([]PerformancePhase, 0, len(launchPhasePairs))
	for _, pair := range launchPhasePairs {
		duration, err := trace.Segment(pair.start, pair.end)
		if err != nil {
			result = append(result, PerformancePhase{Name: pair.name, Reason: err.Error()})
			continue
		}
		result = append(result, PerformancePhase{Name: pair.name, Available: true, DurationMs: duration.Milliseconds()})
	}
	return result, nil
}

func performanceStatus(protocol, demo []PerformancePhase, protocolErr, demoErr error) (string, string) {
	if protocolErr != nil || demoErr != nil {
		reason := ""
		if protocolErr != nil {
			reason = protocolErr.Error()
		}
		if demoErr != nil {
			if reason != "" {
				reason += "; "
			}
			reason += demoErr.Error()
		}
		return "degraded", reason
	}
	if len(protocol) == 0 && len(demo) == 0 {
		return "unmeasured", "launch traces were not persisted"
	}
	return "measured", ""
}

func refreshPerformanceStatus(status *Status) {
	protocol, protocolErr := launchPhaseDurations(launchTracePath(status.SmokeTestID, "protocol"), LaunchRunProtocol)
	demo, demoErr := launchPhaseDurations(launchTracePath(status.SmokeTestID, "demo"), LaunchRunDemo)
	status.ProtocolPhases, status.DemoPhases = protocol, demo
	status.PerformanceStatus, status.PerformanceReason = performanceStatus(protocol, demo, protocolErr, demoErr)
}
