package smoketest

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
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

type packagedPhasePair struct{ name, start, end string }

var packagedPhasePairs = []packagedPhasePair{
	{name: "process_to_splash_first_paint", start: "process_spawn", end: "first_window"},
	{name: "splash_to_app_ready", start: "first_contentful_paint", end: "app_interactive"},
	{name: "runtime_spawn_to_ready", start: "runtime_spawn", end: "runtime_ready"},
	{name: "server_to_app_ready", start: "backend_ready", end: "app_interactive"},
}

func launchPhaseDurations(path string, kind LaunchRunKind) ([]PerformancePhase, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s launch trace: %w", kind, err)
	}
	var header struct {
		SchemaVersion string `json:"schema_version"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return nil, fmt.Errorf("decode %s launch evidence: %w", kind, err)
	}
	if header.SchemaVersion == PackagedLaunchSchemaVersion {
		var sample PackagedLaunchSample
		if err := json.Unmarshal(data, &sample); err != nil {
			return nil, fmt.Errorf("decode %s packaged launch: %w", kind, err)
		}
		if err := sample.Validate(); err != nil {
			return nil, fmt.Errorf("validate %s packaged launch: %w", kind, err)
		}
		return PackagedLaunchPhases(sample), nil
	}
	var trace LaunchTrace
	if err := json.Unmarshal(data, &trace); err != nil {
		return nil, fmt.Errorf("decode %s legacy launch trace: %w", kind, err)
	}
	if err := trace.Validate(); err != nil {
		return nil, fmt.Errorf("validate %s legacy launch trace: %w", kind, err)
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

func PackagedLaunchPhases(sample PackagedLaunchSample) []PerformancePhase {
	result := make([]PerformancePhase, 0, len(packagedPhasePairs))
	for _, pair := range packagedPhasePairs {
		start, startOK := sample.Mark(pair.start)
		end, endOK := sample.Mark(pair.end)
		if !startOK || !endOK {
			result = append(result, PerformancePhase{Name: pair.name, Reason: "required packaged-launch marks are missing"})
			continue
		}
		if start.Availability.State != "available" || end.Availability.State != "available" {
			reason := start.Availability.Reason
			if end.Availability.State != "available" {
				reason = end.Availability.Reason
			}
			result = append(result, PerformancePhase{Name: pair.name, Reason: reason})
			continue
		}
		if end.ElapsedNs < start.ElapsedNs {
			result = append(result, PerformancePhase{Name: pair.name, Reason: "packaged-launch mark order is reversed"})
			continue
		}
		result = append(result, PerformancePhase{Name: pair.name, Available: true, DurationMs: time.Duration(end.ElapsedNs - start.ElapsedNs).Milliseconds()})
	}
	return result
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
	protocol, protocolErr := launchPhaseDurations(packagedLaunchPath(status.SmokeTestID, "protocol"), LaunchRunProtocol)
	demo, demoErr := launchPhaseDurations(packagedLaunchPath(status.SmokeTestID, "demo"), LaunchRunDemo)
	status.ProtocolPhases, status.DemoPhases = protocol, demo
	status.PerformanceStatus, status.PerformanceReason = performanceStatus(protocol, demo, protocolErr, demoErr)
}
