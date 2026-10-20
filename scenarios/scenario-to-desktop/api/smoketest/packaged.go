package smoketest

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

const PackagedLaunchSchemaVersion = "packaged-launch.v1"

type LaunchAvailability struct {
	State  string `json:"state"`
	Reason string `json:"reason"`
}

type PackagedLaunchMark struct {
	Name         string             `json:"name"`
	ElapsedNs    int64              `json:"elapsed_ns"`
	Availability LaunchAvailability `json:"availability"`
}

type PackagedLaunchMetric struct {
	Name         string             `json:"name"`
	Value        float64            `json:"value"`
	Unit         string             `json:"unit"`
	Availability LaunchAvailability `json:"availability"`
}

type PackagedProcessRole struct {
	Role         string             `json:"role"`
	GenericRole  string             `json:"generic_role"`
	ProcessCount int32              `json:"process_count"`
	RSSMB        float64            `json:"rss_mb"`
	PSSMB        float64            `json:"pss_mb"`
	USSMB        float64            `json:"uss_mb"`
	Availability LaunchAvailability `json:"availability"`
}

type LinuxDesktopDetail struct {
	ArtifactPageCacheResidencyPct float64 `json:"artifact_page_cache_residency_pct"`
}

type PackagedPlatformDetail struct {
	LinuxDesktop *LinuxDesktopDetail `json:"linux_desktop,omitempty"`
}

type PackagedOptimizationToggles struct {
	SharedRuntimeReadiness bool `json:"shared_runtime_readiness"`
	SecureCustomSchemeUI   bool `json:"secure_custom_scheme_ui"`
	V8SnapshotCodeCache    bool `json:"v8_snapshot_code_cache"`
	ElectronFuses          bool `json:"electron_fuses"`
}

type PackagedJourneyEvidence struct {
	JourneyID      string `json:"journey_id"`
	CaptureID      string `json:"capture_id"`
	Disposition    string `json:"disposition"`
	DegradedReason string `json:"degraded_reason,omitempty"`
}

type PackagedProcessLifetimeCheck struct {
	Stage     string `json:"stage"`
	SessionID string `json:"session_id"`
	Passed    bool   `json:"passed"`
	Remaining []int  `json:"remaining_descendant_pids,omitempty"`
	Assertion string `json:"assertion,omitempty"`
}

// PackagedLaunchSample is the on-disk JSON form of packaged-launch.v1. The
// proto remains the canonical cross-process contract; this struct is the
// recorder's JSON persistence adapter.
type PackagedLaunchSample struct {
	SchemaVersion        string                         `json:"schema_version"`
	SampleID             string                         `json:"sample_id"`
	Scenario             string                         `json:"scenario"`
	Ramp                 string                         `json:"ramp"`
	Strategy             string                         `json:"strategy"`
	OptimizationToggles  PackagedOptimizationToggles    `json:"toggle_attribution"`
	JourneyEvidence      *PackagedJourneyEvidence       `json:"journey_evidence,omitempty"`
	ProcessLifetime      []PackagedProcessLifetimeCheck `json:"process_lifetime,omitempty"`
	SmokeTestStatus      string                         `json:"smoke_test_status,omitempty"`
	SmokeTestError       string                         `json:"smoke_test_error,omitempty"`
	Platform             string                         `json:"platform"`
	CapturedAt           time.Time                      `json:"captured_at"`
	ExperimentID         string                         `json:"experiment_id"`
	HostFingerprint      string                         `json:"host_fingerprint"`
	ArtifactDigest       string                         `json:"artifact_digest"`
	Condition            string                         `json:"condition"`
	Display              string                         `json:"display"`
	ProfilerMode         string                         `json:"profiler_mode"`
	PowerState           string                         `json:"power_state"`
	Marks                []PackagedLaunchMark           `json:"marks"`
	Metrics              []PackagedLaunchMetric         `json:"metrics"`
	ProcessRoles         []PackagedProcessRole          `json:"process_roles"`
	AppInteractiveSource string                         `json:"app_interactive_source"`
	PlatformDetail       *PackagedPlatformDetail        `json:"platform_detail,omitempty"`
}

func ReadPackagedLaunch(path string) (PackagedLaunchSample, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return PackagedLaunchSample{}, fmt.Errorf("read packaged launch %q: %w", path, err)
	}
	var sample PackagedLaunchSample
	if err := json.Unmarshal(data, &sample); err != nil {
		return PackagedLaunchSample{}, fmt.Errorf("decode packaged launch %q: %w", path, err)
	}
	if err := sample.Validate(); err != nil {
		return PackagedLaunchSample{}, fmt.Errorf("validate packaged launch %q: %w", path, err)
	}
	return sample, nil
}

func (s PackagedLaunchSample) Validate() error {
	if s.SchemaVersion != PackagedLaunchSchemaVersion {
		return fmt.Errorf("unsupported packaged launch schema %q", s.SchemaVersion)
	}
	if strings.TrimSpace(s.SampleID) == "" || s.CapturedAt.IsZero() {
		return fmt.Errorf("sample_id and captured_at are required")
	}
	if s.Strategy == "electron-sidecar" && (s.OptimizationToggles.SharedRuntimeReadiness || s.OptimizationToggles.SecureCustomSchemeUI || s.OptimizationToggles.V8SnapshotCodeCache || s.OptimizationToggles.ElectronFuses) {
		return fmt.Errorf("electron-sidecar sample must attribute every E3 optimization toggle as disabled")
	}
	if s.Strategy == "electron-optimized" && !(s.OptimizationToggles.SharedRuntimeReadiness || s.OptimizationToggles.SecureCustomSchemeUI || s.OptimizationToggles.V8SnapshotCodeCache || s.OptimizationToggles.ElectronFuses) {
		return fmt.Errorf("electron-optimized sample must attribute at least one enabled E3 optimization toggle")
	}
	if s.JourneyEvidence != nil {
		if strings.TrimSpace(s.JourneyEvidence.JourneyID) == "" || strings.TrimSpace(s.JourneyEvidence.Disposition) == "" {
			return fmt.Errorf("journey evidence requires journey_id and disposition")
		}
		if s.JourneyEvidence.Disposition == "pass" && strings.TrimSpace(s.JourneyEvidence.CaptureID) == "" {
			return fmt.Errorf("a passing journey requires capture_id")
		}
	}
	for _, check := range s.ProcessLifetime {
		if strings.TrimSpace(check.Stage) == "" || strings.TrimSpace(check.SessionID) == "" {
			return fmt.Errorf("process lifetime check requires stage and session_id")
		}
		if !check.Passed || len(check.Remaining) > 0 {
			return fmt.Errorf("process lifetime assertion failed for %s: remaining descendants %v", check.Stage, check.Remaining)
		}
	}
	seen := make(map[string]struct{}, len(s.Marks))
	for i, mark := range s.Marks {
		if strings.TrimSpace(mark.Name) == "" || mark.ElapsedNs < 0 {
			return fmt.Errorf("mark %d has invalid name or elapsed_ns", i)
		}
		if _, ok := seen[mark.Name]; ok {
			return fmt.Errorf("mark %q is duplicated", mark.Name)
		}
		seen[mark.Name] = struct{}{}
		if mark.Availability.State != "available" && mark.Availability.State != "unavailable" {
			return fmt.Errorf("mark %q has invalid availability state %q", mark.Name, mark.Availability.State)
		}
		if strings.TrimSpace(mark.Availability.Reason) == "" {
			return fmt.Errorf("mark %q availability reason is required", mark.Name)
		}
	}
	for i, metric := range s.Metrics {
		if strings.TrimSpace(metric.Name) == "" || strings.TrimSpace(metric.Unit) == "" {
			return fmt.Errorf("metric %d requires name and unit", i)
		}
		if metric.Availability.State != "available" && metric.Availability.State != "unavailable" {
			return fmt.Errorf("metric %q has invalid availability state %q", metric.Name, metric.Availability.State)
		}
		if strings.TrimSpace(metric.Availability.Reason) == "" {
			return fmt.Errorf("metric %q availability reason is required", metric.Name)
		}
	}
	return nil
}

func (s PackagedLaunchSample) Mark(name string) (PackagedLaunchMark, bool) {
	for _, mark := range s.Marks {
		if mark.Name == name {
			return mark, true
		}
	}
	return PackagedLaunchMark{}, false
}
