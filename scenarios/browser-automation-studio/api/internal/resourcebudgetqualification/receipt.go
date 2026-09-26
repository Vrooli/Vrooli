// Package resourcebudgetqualification validates current-candidate BAS resource
// measurements before the rehabilitation provider credits the resource row.
package resourcebudgetqualification

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	ContractRow       = "resource-budget"
	EvidenceDir       = ".vrooli/runtime/rehabilitation-evidence"
	MaxIdlePSSKiB     = 300 * 1024
	MaxFixturePSSKiB  = 1024 * 1024
	MaxIdleCPUPercent = 2.0
)

var RequiredSources = []string{
	"docs/internal/REFRACTOR_CONTRACT.json",
	"api/cmd/resource-budget-cohort/qualification.mjs",
	"api/automation/driver/client.go",
	"playwright-driver/src/routes/session-start.ts",
	"playwright-driver/src/session/manager.ts",
}

type receipt struct {
	SchemaVersion int    `json:"schemaVersion"`
	ContractRow   string `json:"contractRow"`
	Result        string `json:"result"`
	BuildBefore   string `json:"managedBuildIdentityBefore"`
	BuildAfter    string `json:"managedBuildIdentityAfter"`
	Platform      struct {
		OS                   string `json:"os"`
		WindowsPrivateMemory struct {
			Status string `json:"status"`
			Reason string `json:"reason"`
		} `json:"windowsPrivateMemory"`
	} `json:"platform"`
	Idle struct {
		StartedAt            string   `json:"startedAt"`
		EndedAt              string   `json:"endedAt"`
		DurationMS           int64    `json:"durationMs"`
		SampleIntervalMS     int      `json:"sampleIntervalMs"`
		SampleCount          int      `json:"sampleCount"`
		SessionsThroughout   int      `json:"sessionsThroughout"`
		RecordingsThroughout int      `json:"recordingsThroughout"`
		MaxCombinedPSSKiB    int      `json:"maxCombinedPssKiB"`
		AverageCPUPercent    float64  `json:"averageCombinedCPUPercent"`
		P95CPUPercent        float64  `json:"p95CombinedCPUPercent"`
		Samples              []sample `json:"samples"`
	} `json:"idleSampling"`
	Fixture struct {
		BrowserProcessCount int              `json:"browserProcessCount"`
		BrowserPIDs         []fixtureProcess `json:"browserPids"`
		BrowserPSSKiB       int              `json:"browserPssKiB"`
		ShellPSSKiB         int              `json:"shellPssKiB"`
		CombinedPSSKiB      int              `json:"combinedPssKiB"`
	} `json:"fixtureBrowserAndShell"`
	Cleanup struct {
		FixtureSessionClosed bool `json:"fixtureSessionClosed"`
		APIHealthy           bool `json:"apiHealthy"`
	} `json:"cleanup"`
	SourceSHA256  map[string]string `json:"source_sha256"`
	OwnerArtifact artifactRef       `json:"owner_artifact"`
}

type sample struct {
	PSSKiB struct {
		API    int `json:"api"`
		Driver int `json:"driver"`
	} `json:"pssKiB"`
	CombinedPSSKiB int `json:"combinedPssKiB"`
	CPUPercent     *struct {
		API    float64 `json:"api"`
		Driver float64 `json:"driver"`
	} `json:"cpuPercent"`
	Sessions   int `json:"sessions"`
	Recordings int `json:"activeRecordings"`
}

type fixtureProcess struct {
	PID    int `json:"pid"`
	PSSKiB int `json:"pssKiB"`
}

type artifactRef struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// Validate requires the owner-produced receipt to match the running build,
// current contract and measurement producer before granting the Linux cohort.
func Validate(scenarioRoot, liveBuild string) error {
	if strings.TrimSpace(liveBuild) == "" {
		return fmt.Errorf("live managed build identity is empty")
	}
	paths, err := filepath.Glob(filepath.Join(scenarioRoot, EvidenceDir, "resource-budget-w189-*.json"))
	if err != nil {
		return err
	}
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))
	for _, path := range paths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			continue
		}
		var candidate receipt
		if json.Unmarshal(data, &candidate) != nil || candidate.BuildAfter != liveBuild {
			continue
		}
		if err := validate(scenarioRoot, candidate); err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(path), err)
		}
		return nil
	}
	return fmt.Errorf("no retained resource-budget receipt matches live build %q", liveBuild)
}

func validate(root string, r receipt) error {
	if r.SchemaVersion != 1 || r.ContractRow != ContractRow || r.Result != "passed" ||
		r.BuildBefore == "" || r.BuildBefore != r.BuildAfter || r.Platform.OS != "linux" {
		return fmt.Errorf("receipt schema, outcome, platform, or build identity is invalid")
	}
	contractSHA, err := fileSHA(filepath.Join(root, "docs/internal/REFRACTOR_CONTRACT.json"))
	if err != nil {
		return err
	}
	if r.SourceSHA256["docs/internal/REFRACTOR_CONTRACT.json"] != contractSHA {
		return fmt.Errorf("contract digest mismatch")
	}
	for _, source := range RequiredSources {
		want, ok := r.SourceSHA256[source]
		if !ok || len(want) != sha256.Size*2 {
			return fmt.Errorf("receipt lacks source digest: %s", source)
		}
		got, sourceErr := fileSHA(filepath.Join(root, filepath.FromSlash(source)))
		if sourceErr != nil {
			return sourceErr
		}
		if want != got {
			return fmt.Errorf("source digest mismatch: %s", source)
		}
	}
	if err := validateArtifact(root, r.OwnerArtifact); err != nil {
		return err
	}
	if r.Idle.SampleIntervalMS < 900 || r.Idle.SampleCount < 61 || len(r.Idle.Samples) != r.Idle.SampleCount ||
		r.Idle.SessionsThroughout != 0 || r.Idle.RecordingsThroughout != 0 {
		return fmt.Errorf("idle measurement has insufficient samples or was not idle")
	}
	started, err := time.Parse(time.RFC3339Nano, r.Idle.StartedAt)
	if err != nil {
		return fmt.Errorf("parse sampling start: %w", err)
	}
	ended, err := time.Parse(time.RFC3339Nano, r.Idle.EndedAt)
	if err != nil {
		return fmt.Errorf("parse sampling end: %w", err)
	}
	if ended.Sub(started) < 60*time.Second || r.Idle.DurationMS < 60000 {
		return fmt.Errorf("idle sampling lasted less than 60 seconds")
	}
	observedMaxPSS := 0
	cpu := make([]float64, 0, len(r.Idle.Samples)-1)
	for index, observation := range r.Idle.Samples {
		if observation.Sessions != 0 || observation.Recordings != 0 ||
			observation.PSSKiB.API <= 0 || observation.PSSKiB.Driver <= 0 ||
			observation.CombinedPSSKiB != observation.PSSKiB.API+observation.PSSKiB.Driver {
			return fmt.Errorf("sample %d is incomplete, non-idle, or has inconsistent PSS", index)
		}
		if observation.CombinedPSSKiB > observedMaxPSS {
			observedMaxPSS = observation.CombinedPSSKiB
		}
		if index > 0 {
			if observation.CPUPercent == nil || observation.CPUPercent.API < 0 || observation.CPUPercent.Driver < 0 {
				return fmt.Errorf("sample %d has invalid CPU counters", index)
			}
			cpu = append(cpu, observation.CPUPercent.API+observation.CPUPercent.Driver)
		}
	}
	if observedMaxPSS != r.Idle.MaxCombinedPSSKiB || observedMaxPSS > MaxIdlePSSKiB {
		return fmt.Errorf("combined idle PSS is inconsistent or exceeds %d KiB", MaxIdlePSSKiB)
	}
	if len(cpu) == 0 {
		return fmt.Errorf("idle CPU sample is empty")
	}
	sort.Float64s(cpu)
	average := 0.0
	for _, value := range cpu {
		average += value
	}
	average /= float64(len(cpu))
	p95 := cpu[min(len(cpu)-1, int(math.Ceil(float64(len(cpu))*0.95))-1)]
	if average >= MaxIdleCPUPercent || p95 >= MaxIdleCPUPercent ||
		abs(r.Idle.AverageCPUPercent-average) > 0.05 || abs(r.Idle.P95CPUPercent-p95) > 0.05 {
		return fmt.Errorf("idle CPU is inconsistent or exceeds %.2f%% of one core", MaxIdleCPUPercent)
	}
	fixturePSS := r.Fixture.BrowserPSSKiB + r.Fixture.ShellPSSKiB
	observedBrowserPSS := 0
	for _, process := range r.Fixture.BrowserPIDs {
		if process.PID <= 0 || process.PSSKiB <= 0 {
			return fmt.Errorf("fixture browser process list is invalid")
		}
		observedBrowserPSS += process.PSSKiB
	}
	if r.Fixture.BrowserProcessCount < 1 || r.Fixture.BrowserProcessCount != len(r.Fixture.BrowserPIDs) ||
		observedBrowserPSS != r.Fixture.BrowserPSSKiB || fixturePSS != r.Fixture.CombinedPSSKiB ||
		fixturePSS > MaxFixturePSSKiB || r.Fixture.ShellPSSKiB <= 0 {
		return fmt.Errorf("fixture browser/shell PSS is incomplete, inconsistent, or exceeds %d KiB", MaxFixturePSSKiB)
	}
	if !r.Cleanup.FixtureSessionClosed || !r.Cleanup.APIHealthy {
		return fmt.Errorf("managed fixture cleanup or API health check failed")
	}
	if r.Platform.WindowsPrivateMemory.Status == "" || (r.Platform.WindowsPrivateMemory.Status != "measured" && strings.TrimSpace(r.Platform.WindowsPrivateMemory.Reason) == "") {
		return fmt.Errorf("Windows private-memory comparison must be reported, including why it is unavailable")
	}
	return nil
}

func validateArtifact(root string, ref artifactRef) error {
	if strings.TrimSpace(ref.Path) == "" || len(ref.SHA256) != sha256.Size*2 || filepath.IsAbs(ref.Path) {
		return fmt.Errorf("owner artifact path or digest is invalid")
	}
	clean := filepath.Clean(filepath.FromSlash(ref.Path))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("owner artifact escapes scenario root")
	}
	got, err := fileSHA(filepath.Join(root, clean))
	if err != nil {
		return err
	}
	if got != ref.SHA256 {
		return fmt.Errorf("owner artifact digest mismatch")
	}
	return nil
}

func fileSHA(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func abs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
