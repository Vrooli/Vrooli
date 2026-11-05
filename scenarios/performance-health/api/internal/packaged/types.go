package packaged

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

const SchemaVersion = "packaged-launch.v1"
const ExperimentSchemaVersion = "packaged-experiment.v1"

var ErrInvalidSample = errors.New("packaged: invalid packaged-launch sample")

type Availability struct {
	State  string `json:"state"`
	Reason string `json:"reason"`
}

type Mark struct {
	Name         string       `json:"name"`
	ElapsedNs    int64        `json:"elapsed_ns"`
	Availability Availability `json:"availability"`
}

type Metric struct {
	Name         string       `json:"name"`
	Value        float64      `json:"value"`
	Unit         string       `json:"unit"`
	Availability Availability `json:"availability"`
}

type ProcessRole struct {
	Role         string       `json:"role"`
	GenericRole  string       `json:"generic_role"`
	ProcessCount int32        `json:"process_count"`
	RSSMB        float64      `json:"rss_mb"`
	PSSMB        float64      `json:"pss_mb"`
	USSMB        float64      `json:"uss_mb"`
	Availability Availability `json:"availability"`
}

type LinuxDesktopDetail struct {
	ArtifactPageCacheResidencyPct float64 `json:"artifact_page_cache_residency_pct"`
}

type PlatformDetail struct {
	LinuxDesktop *LinuxDesktopDetail `json:"linux_desktop,omitempty"`
}

type JourneyEvidence struct {
	JourneyID      string `json:"journey_id"`
	CaptureID      string `json:"capture_id"`
	Disposition    string `json:"disposition"`
	DegradedReason string `json:"degraded_reason,omitempty"`
}

type Sample struct {
	SchemaVersion        string              `json:"schema_version"`
	SampleID             string              `json:"sample_id"`
	Scenario             string              `json:"scenario"`
	Ramp                 string              `json:"ramp"`
	Strategy             string              `json:"strategy"`
	ToggleAttribution    OptimizationToggles `json:"toggle_attribution"`
	JourneyEvidence      *JourneyEvidence    `json:"journey_evidence,omitempty"`
	Platform             string              `json:"platform"`
	CapturedAt           time.Time           `json:"captured_at"`
	ExperimentID         string              `json:"experiment_id"`
	HostFingerprint      string              `json:"host_fingerprint"`
	ArtifactDigest       string              `json:"artifact_digest"`
	Condition            string              `json:"condition"`
	Display              string              `json:"display"`
	ProfilerMode         string              `json:"profiler_mode"`
	PowerState           string              `json:"power_state"`
	Marks                []Mark              `json:"marks"`
	Metrics              []Metric            `json:"metrics"`
	ProcessRoles         []ProcessRole       `json:"process_roles"`
	AppInteractiveSource string              `json:"app_interactive_source"`
	PlatformDetail       *PlatformDetail     `json:"platform_detail,omitempty"`
}

func (s Sample) Validate() error {
	if s.SchemaVersion != SchemaVersion || s.SampleID == "" || s.CapturedAt.IsZero() || s.Scenario == "" || s.Ramp == "" || s.Strategy == "" || s.Platform == "" {
		return ErrInvalidSample
	}
	if s.Strategy == "electron-sidecar" && s.ToggleAttribution.Any() || s.Strategy == "electron-optimized" && !s.ToggleAttribution.Any() {
		return ErrInvalidSample
	}
	if s.Condition != "cold" && s.Condition != "warm" {
		return ErrInvalidSample
	}
	if s.JourneyEvidence != nil {
		if strings.TrimSpace(s.JourneyEvidence.JourneyID) == "" || strings.TrimSpace(s.JourneyEvidence.Disposition) == "" {
			return fmt.Errorf("%w: journey evidence requires journey_id and disposition", ErrInvalidSample)
		}
		if s.JourneyEvidence.Disposition == "pass" && strings.TrimSpace(s.JourneyEvidence.CaptureID) == "" {
			return fmt.Errorf("%w: a passing journey requires a capture_id", ErrInvalidSample)
		}
	}
	for _, mark := range s.Marks {
		if mark.Name == "" || mark.ElapsedNs < 0 || !validAvailability(mark.Availability) {
			return ErrInvalidSample
		}
	}
	for _, metric := range s.Metrics {
		if metric.Name == "" || metric.Unit == "" || !validAvailability(metric.Availability) {
			return ErrInvalidSample
		}
	}
	return nil
}

func validAvailability(a Availability) bool {
	return (a.State == "available" || a.State == "unavailable") && a.Reason != ""
}

func (s Sample) Mark(name string) (Mark, bool) {
	for _, mark := range s.Marks {
		if mark.Name == name {
			return mark, true
		}
	}
	return Mark{}, false
}

func (s Sample) Metric(name string) (Metric, bool) {
	for _, metric := range s.Metrics {
		if metric.Name == name {
			return metric, true
		}
	}
	return Metric{}, false
}

type CaptureRequest struct {
	PlanOnly   bool                `json:"plan_only,omitempty"`
	Scenario   string              `json:"scenario"`
	Ramp       string              `json:"ramp"`
	Strategy   string              `json:"strategy"`
	Toggles    OptimizationToggles `json:"optimization_toggles"`
	Runs       int                 `json:"runs"`
	Condition  string              `json:"condition"`
	Artifact   string              `json:"artifact_path"`
	Experiment string              `json:"experiment_id,omitempty"`
	Journey    string              `json:"journey,omitempty"`
}

// CaptureWork describes planned launcher work, not observed process starts.
// Each measured-launch operation normally starts protocol and demo processes.
// Failed/cancelled work may start fewer; subprocesses are not enumerated here.
type CaptureWork struct {
	MeasurementSamples        int `json:"measurement_samples"`
	DiscardedWarmups          int `json:"discarded_warmups"`
	MeasuredLaunchOperations  int `json:"measured_launch_operations"`
	NominalProtocolDemoStarts int `json:"nominal_protocol_demo_starts"`
}

func (r CaptureRequest) PlannedWork() CaptureWork {
	return CaptureWork{MeasurementSamples: r.Runs, DiscardedWarmups: 1, MeasuredLaunchOperations: r.Runs + 1, NominalProtocolDemoStarts: 2 * (r.Runs + 1)}
}

type LaunchAttempt struct {
	Warmup     bool                `json:"warmup,omitempty"`
	Scenario   string              `json:"scenario"`
	Ramp       string              `json:"ramp"`
	Strategy   string              `json:"strategy"`
	Experiment string              `json:"experiment_id"`
	Condition  string              `json:"condition"`
	Toggles    OptimizationToggles `json:"toggle_attribution"`
	Attempted  time.Time           `json:"attempted_at"`
	Succeeded  bool                `json:"succeeded"`
	Error      string              `json:"error,omitempty"`
}

// ExperimentManifest declares a reproducible platform comparison. Missing
// artifacts and journeys remain explicit so reports cannot imply measurements.
type ExperimentManifest struct {
	SchemaVersion string              `json:"schema_version"`
	ExperimentID  string              `json:"experiment_id"`
	Platform      string              `json:"platform"`
	Subjects      []ExperimentSubject `json:"subjects"`
	SampleCount   int                 `json:"sample_count"`
	Modes         []string            `json:"modes"`
	Environment   EnvironmentMetadata `json:"environment"`
}

type ExperimentSubject struct {
	Scenario  string           `json:"scenario"`
	JourneyID string           `json:"journey_id"`
	Cells     []ExperimentCell `json:"cells"`
}

type ExperimentCell struct {
	Strategy       string              `json:"strategy"`
	Toggles        OptimizationToggles `json:"toggles"`
	ArtifactPath   string              `json:"artifact_path"`
	ArtifactDigest string              `json:"artifact_digest"`
}

type OptimizationToggles struct {
	SharedRuntimeReadiness bool `json:"shared_runtime_readiness"`
	SecureCustomSchemeUI   bool `json:"secure_custom_scheme_ui"`
	V8SnapshotCodeCache    bool `json:"v8_snapshot_code_cache"`
	ElectronFuses          bool `json:"electron_fuses"`
}

type EnvironmentMetadata struct {
	HostFingerprint string `json:"host_fingerprint"`
	Display         string `json:"display"`
	ProfilerMode    string `json:"profiler_mode"`
	PowerState      string `json:"power_state"`
}

func (m ExperimentManifest) Validate() error {
	if m.SchemaVersion != ExperimentSchemaVersion {
		return errors.New("schema_version must be packaged-experiment.v1")
	}
	if strings.TrimSpace(m.ExperimentID) == "" {
		return errors.New("experiment_id is required")
	}
	if m.Platform != "linux-x64" && m.Platform != "macos-x64" {
		return errors.New("platform must be linux-x64 or macos-x64")
	}
	if m.SampleCount != 10 {
		return errors.New("sample_count must be 10 for the declared comparison matrix")
	}
	if len(m.Subjects) != 4 {
		return errors.New("subjects must contain all four reference scenarios exactly once")
	}
	if len(m.Modes) != 2 || !contains(m.Modes, "cold") || !contains(m.Modes, "warm") {
		return errors.New("modes must contain cold and warm exactly once")
	}
	for _, value := range []string{m.Environment.HostFingerprint, m.Environment.Display, m.Environment.ProfilerMode, m.Environment.PowerState} {
		if strings.TrimSpace(value) == "" {
			return errors.New("environment host_fingerprint, display, profiler_mode, and power_state are required")
		}
	}
	seenSubjects := map[string]bool{}
	for _, subject := range m.Subjects {
		if !allowedSubject(subject.Scenario) {
			return fmt.Errorf("unsupported subject %q; expected one of the current reference scenarios: %s", subject.Scenario, strings.Join(referenceSubjects, ", "))
		}
		if seenSubjects[subject.Scenario] {
			return fmt.Errorf("subject %q is duplicated", subject.Scenario)
		}
		seenSubjects[subject.Scenario] = true
		if strings.TrimSpace(subject.JourneyID) == "" {
			return fmt.Errorf("subject %q requires journey_id", subject.Scenario)
		}
		if len(subject.Cells) != 6 {
			return fmt.Errorf("subject %q must declare six cells: baseline, all toggles, and four single-toggle cells", subject.Scenario)
		}
		seen := map[string]bool{}
		for _, cell := range subject.Cells {
			key := cell.Identity()
			if seen[key] {
				return fmt.Errorf("subject %q duplicates cell %q", subject.Scenario, key)
			}
			seen[key] = true
			if cell.Strategy != "electron-sidecar" && cell.Strategy != "electron-optimized" {
				return fmt.Errorf("cell strategy %q is unsupported", cell.Strategy)
			}
			if cell.Strategy == "electron-sidecar" && cell.Toggles.Any() {
				return errors.New("electron-sidecar baseline must have every optimization toggle disabled")
			}
			if cell.Strategy == "electron-optimized" && !validOptimizedCell(cell.Toggles) {
				return fmt.Errorf("optimized cell %q must enable all toggles or exactly one toggle", key)
			}
			if cell.ArtifactPath != "" || cell.ArtifactDigest != "" {
				if strings.TrimSpace(cell.ArtifactPath) == "" || !validDigest(cell.ArtifactDigest) {
					return fmt.Errorf("cell %q requires an artifact_path and sha256 digest together when supplied", key)
				}
			}
		}
		for _, required := range []ExperimentCell{{Strategy: "electron-sidecar"}, {Strategy: "electron-optimized", Toggles: OptimizationToggles{true, true, true, true}}, {Strategy: "electron-optimized", Toggles: OptimizationToggles{SharedRuntimeReadiness: true}}, {Strategy: "electron-optimized", Toggles: OptimizationToggles{SecureCustomSchemeUI: true}}, {Strategy: "electron-optimized", Toggles: OptimizationToggles{V8SnapshotCodeCache: true}}, {Strategy: "electron-optimized", Toggles: OptimizationToggles{ElectronFuses: true}}} {
			if !seen[required.Identity()] {
				return fmt.Errorf("subject %q is missing required cell %q", subject.Scenario, required.Identity())
			}
		}
	}
	for _, required := range referenceSubjects {
		if !seenSubjects[required] {
			return fmt.Errorf("subjects is missing required reference scenario %q", required)
		}
	}
	return nil
}

func (c ExperimentCell) Identity() string {
	return fmt.Sprintf("%s:%t:%t:%t:%t", c.Strategy, c.Toggles.SharedRuntimeReadiness, c.Toggles.SecureCustomSchemeUI, c.Toggles.V8SnapshotCodeCache, c.Toggles.ElectronFuses)
}
func (t OptimizationToggles) Any() bool {
	return t.SharedRuntimeReadiness || t.SecureCustomSchemeUI || t.V8SnapshotCodeCache || t.ElectronFuses
}
func (t OptimizationToggles) Identity() string {
	return fmt.Sprintf("%t:%t:%t:%t", t.SharedRuntimeReadiness, t.SecureCustomSchemeUI, t.V8SnapshotCodeCache, t.ElectronFuses)
}
func validOptimizedCell(t OptimizationToggles) bool {
	n := 0
	for _, v := range []bool{t.SharedRuntimeReadiness, t.SecureCustomSchemeUI, t.V8SnapshotCodeCache, t.ElectronFuses} {
		if v {
			n++
		}
	}
	return n == 1 || n == 4
}
func contains(values []string, target string) bool {
	n := 0
	for _, v := range values {
		if v == target {
			n++
		}
	}
	return n == 1
}
func allowedSubject(value string) bool {
	for _, required := range referenceSubjects {
		if value == required {
			return true
		}
	}
	return false
}

// Keep this closed set synchronized with scenario-to-desktop TARGETS.md. The
// selected fourth reference is the approved LPBS replacement for GCT.
var referenceSubjects = []string{"hello-desktop", "system-monitor", "web-console", "landing-page-business-suite"}

func validDigest(value string) bool {
	if len(value) != 71 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

type MetricSummary struct {
	Available   int      `json:"available_samples"`
	Unavailable int      `json:"unavailable_samples"`
	P50         *float64 `json:"p50,omitempty"`
	P95         *float64 `json:"p95,omitempty"`
	Min         *float64 `json:"min,omitempty"`
	Max         *float64 `json:"max,omitempty"`
	Spread      *float64 `json:"spread,omitempty"`
}

type ReportCell struct {
	Scenario          string                   `json:"scenario"`
	Ramp              string                   `json:"ramp"`
	Strategy          string                   `json:"strategy"`
	Platform          string                   `json:"platform"`
	Condition         string                   `json:"condition"`
	ExperimentID      string                   `json:"experiment_id"`
	Comparable        bool                     `json:"comparable"`
	Samples           int                      `json:"samples"`
	AdvisoryBudgets   map[string]float64       `json:"advisory_budgets,omitempty"`
	ToggleAttribution OptimizationToggles      `json:"toggle_attribution"`
	AttributionStatus string                   `json:"attribution_status"`
	JourneyParity     string                   `json:"journey_parity"`
	JourneyReceipts   []JourneyEvidence        `json:"journey_receipts,omitempty"`
	WarmupReliability ReliabilityEvidence      `json:"warmup_reliability"`
	Reliability       ReliabilityEvidence      `json:"reliability"`
	ArtifactDigest    string                   `json:"artifact_digest"`
	Environment       EnvironmentMetadata      `json:"environment"`
	Metrics           map[string]MetricSummary `json:"metrics"`
}

type ReliabilityEvidence struct {
	Attempts   int      `json:"attempts"`
	Successful int      `json:"successful"`
	Failed     int      `json:"failed"`
	Status     string   `json:"status"`
	Failures   []string `json:"failures,omitempty"`
}

type Report struct {
	SchemaVersion string       `json:"schema_version"`
	Cells         []ReportCell `json:"cells"`
	MissingData   []string     `json:"missing_data,omitempty"`
}
