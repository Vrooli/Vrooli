// Package motionqualification validates managed viewer motion receipts.
package motionqualification

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	ContractRow   = "motion"
	EvidenceDir   = ".vrooli/runtime/rehabilitation-evidence"
	EvidenceGlob  = EvidenceDir + "/motion-receipt-*.json"
	MaxFrameBytes = 12*1024*1024 + 4*1024
)

var RequiredSources = []string{
	"docs/internal/REFRACTOR_CONTRACT.json",
	".vrooli/test-genie.json",
	".vrooli/program-runtime/setpoint-read.py",
	"api/automation/driver/types.go",
	"api/automation/driver/client_payload_test.go",
	"api/handlers/record_mode_types.go",
	"api/handlers/record_mode_test.go",
	"api/handlers/testutil_mock_services.go",
	"api/internal/motionqualification/receipt.go",
	"api/internal/motionqualification/receipt_test.go",
	"api/internal/testutil/testutil.go",
	"api/handlers/profilevalidation/provider.go",
	"api/handlers/profilevalidation/provider_test.go",
	"api/config/config.go",
	"api/websocket/hub.go",
	"api/websocket/hub_test.go",
	"api/cmd/motion-cohort/qualification.mjs",
	"api/cmd/motion-cohort/validation.mjs",
	"playwright-driver/src/frame-streaming/strategies/cdp-screencast.ts",
	"playwright-driver/tests/integration/motion-qualification.test.ts",
	"playwright-driver/tests/unit/frame-streaming/cdp-screencast-strategy.test.ts",
	"ui/src/domains/recording/capture/useFrameStream.ts",
	"ui/src/domains/recording/capture/useFrameStream.test.ts",
}

type Receipt struct {
	SchemaVersion int               `json:"schemaVersion"`
	ContractRow   string            `json:"contractRow"`
	Result        string            `json:"result"`
	BuildIdentity string            `json:"managedBuildIdentity"`
	ContractSHA   string            `json:"contractSha256"`
	SourceSHA256  map[string]string `json:"sourceSha256"`
	Baseline      baseline          `json:"baseline"`
	SlowReader    slowReader        `json:"slowReader"`
	Artifacts     []artifact        `json:"artifacts"`
}

type baseline struct {
	DurationMS          int64   `json:"durationMs"`
	RenderedFrames      int     `json:"renderedFrames"`
	UniqueFixtureFrames int     `json:"uniqueFixtureFrames"`
	RenderedFPS         float64 `json:"renderedFps"`
	P95FrameAgeMS       float64 `json:"p95FrameAgeMs"`
	MaxFrameAgeMS       float64 `json:"maxFrameAgeMs"`
	MaxFrameBytes       int     `json:"maxFrameBytes"`
	P95DecodeMS         float64 `json:"p95DecodeMs"`
	MaxDecodeMS         float64 `json:"maxDecodeMs"`
}

type slowReader struct {
	DurationMS           int64   `json:"durationMs"`
	StallCount           int     `json:"stallCount"`
	ReceivedFrames       int     `json:"receivedFrames"`
	DecodedFrames        int     `json:"decodedFrames"`
	RenderedFrames       int     `json:"renderedFrames"`
	MaxConcurrentDecodes int     `json:"maxConcurrentDecodes"`
	MaxFrameAgeMS        float64 `json:"maxFrameAgeMs"`
	MaxFrameBytes        int     `json:"maxFrameBytes"`
	MaxAPIQueueBytes     int     `json:"maxApiQueueBytes"`
	APIQueueBudgetBytes  int     `json:"apiQueueBudgetBytes"`
}

type artifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// Validate accepts only the newest receipt tied to this contract, source and
// managed build, with its raw owner artifacts intact.
func Validate(root, liveBuild string) error {
	if strings.TrimSpace(liveBuild) == "" {
		return fmt.Errorf("live managed build identity is empty")
	}
	paths, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(EvidenceGlob)))
	if err != nil {
		return err
	}
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))
	for _, path := range paths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			continue
		}
		var receipt Receipt
		if json.Unmarshal(data, &receipt) != nil || receipt.BuildIdentity != liveBuild {
			continue
		}
		if err := validate(root, receipt); err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(path), err)
		}
		return nil
	}
	return fmt.Errorf("no retained motion receipt matches live build %q", liveBuild)
}

func validate(root string, r Receipt) error {
	if r.SchemaVersion != 1 || r.ContractRow != ContractRow || r.Result != "passed" || r.BuildIdentity == "" {
		return fmt.Errorf("motion receipt schema, outcome, row, or build identity is invalid")
	}
	contractSHA, err := fileSHA(filepath.Join(root, "docs/internal/REFRACTOR_CONTRACT.json"))
	if err != nil {
		return err
	}
	if r.ContractSHA != contractSHA {
		return fmt.Errorf("contract digest mismatch")
	}
	if len(r.SourceSHA256) != len(RequiredSources) {
		return fmt.Errorf("receipt has %d source digests; want %d", len(r.SourceSHA256), len(RequiredSources))
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
		if got != want {
			return fmt.Errorf("source digest mismatch: %s", source)
		}
	}
	if err := validateBaseline(r.Baseline); err != nil {
		return err
	}
	if err := validateSlowReader(r.SlowReader); err != nil {
		return err
	}
	if len(r.Artifacts) != 2 {
		return fmt.Errorf("receipt has %d raw artifacts; want 2", len(r.Artifacts))
	}
	for _, ref := range r.Artifacts {
		if err := validateArtifact(root, ref); err != nil {
			return err
		}
	}
	return nil
}

func validateBaseline(m baseline) error {
	measuredFPS := float64(m.RenderedFrames) / (float64(m.DurationMS) / 1000)
	// Allow at most one frame of finite-window count uncertainty; the
	// independent 9,000-frame floor remains exact.
	boundaryFPS := 1000 / float64(m.DurationMS)
	if m.DurationMS < 300_000 || m.RenderedFrames < 9_000 || m.RenderedFPS+boundaryFPS < 30 || measuredFPS+boundaryFPS < 30 ||
		m.UniqueFixtureFrames < 9_000 || m.P95FrameAgeMS < 0 || m.P95FrameAgeMS > 100 ||
		m.MaxFrameAgeMS < m.P95FrameAgeMS || m.MaxFrameBytes <= 0 || m.MaxFrameBytes > MaxFrameBytes ||
		m.P95DecodeMS < 0 || m.P95DecodeMS > 100 || m.MaxDecodeMS < m.P95DecodeMS || m.MaxDecodeMS > 250 {
		return fmt.Errorf("five-minute motion cohort is incomplete or outside FPS, frame-age, byte, or decode bounds")
	}
	return nil
}

func validateSlowReader(m slowReader) error {
	if m.DurationMS < 15_000 || m.StallCount < 2 || m.ReceivedFrames <= m.RenderedFrames || m.DecodedFrames < m.RenderedFrames ||
		m.MaxConcurrentDecodes != 1 || m.MaxFrameAgeMS <= 0 || m.MaxFrameAgeMS > 1_000 ||
		m.MaxFrameBytes <= 0 || m.MaxFrameBytes > MaxFrameBytes || m.MaxAPIQueueBytes <= 0 ||
		m.APIQueueBudgetBytes != MaxFrameBytes || m.MaxAPIQueueBytes > m.APIQueueBudgetBytes {
		return fmt.Errorf("slow-reader cohort is incomplete or exceeded its bounded decoder/frame-age/frame-byte limits")
	}
	return nil
}

func validateArtifact(root string, ref artifact) error {
	path := filepath.Clean(filepath.FromSlash(ref.Path))
	if path == "." || filepath.IsAbs(path) || path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator)) ||
		!strings.HasPrefix(path, filepath.FromSlash(EvidenceDir)+string(filepath.Separator)) || len(ref.SHA256) != sha256.Size*2 {
		return fmt.Errorf("owner artifact path or digest is invalid")
	}
	got, err := fileSHA(filepath.Join(root, path))
	if err != nil {
		return err
	}
	if got != ref.SHA256 {
		return fmt.Errorf("owner artifact digest mismatch: %s", ref.Path)
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
