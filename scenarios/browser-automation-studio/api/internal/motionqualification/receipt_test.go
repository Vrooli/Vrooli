package motionqualification

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/browser-automation-studio/internal/testutil"
	"github.com/vrooli/repo-contract-go/repocontracttest"
)

func TestValidateAcceptsCurrentFiveMinuteMotionAndSlowReaderReceipt(t *testing.T) {
	root, build := writeValidReceipt(t)
	if err := Validate(root, build); err != nil {
		t.Fatalf("valid motion receipt rejected: %v", err)
	}
}

func TestValidateAllowsOneFrameOfFiniteWindowFPSUncertainty(t *testing.T) {
	root, build := writeValidReceipt(t)
	path := filepath.Join(root, EvidenceDir, "motion-receipt-test.json")
	receipt := repocontracttest.ReadJSONFileInto[Receipt](t, path)
	receipt.Baseline.DurationMS = 300_054
	receipt.Baseline.RenderedFrames = 9_001
	receipt.Baseline.UniqueFixtureFrames = 9_001
	receipt.Baseline.RenderedFPS = 9_001 / (300_054.0 / 1000)
	testutil.WriteJSONFile(t, path, receipt)
	if err := Validate(root, build); err != nil {
		t.Fatalf("one-frame boundary uncertainty rejected: %v", err)
	}
	receipt.Baseline.RenderedFrames = 8_999
	testutil.WriteJSONFile(t, path, receipt)
	if err := Validate(root, build); err == nil {
		t.Fatal("baseline below the required 9,000 rendered frames was accepted")
	}
	receipt.Baseline.RenderedFrames = 9_001
	receipt.Baseline.DurationMS = 301_000
	receipt.Baseline.RenderedFPS = 9_001 / 301
	testutil.WriteJSONFile(t, path, receipt)
	if err := Validate(root, build); err == nil {
		t.Fatal("baseline beyond two-frame boundary uncertainty was accepted")
	}
}

func TestValidateRejectsMotionOrSlowReaderThresholdViolations(t *testing.T) {
	root, build := writeValidReceipt(t)
	path := filepath.Join(root, EvidenceDir, "motion-receipt-test.json")
	r := repocontracttest.ReadJSONFileInto[Receipt](t, path)
	r.Baseline.P95FrameAgeMS = 100.1
	testutil.WriteJSONFile(t, path, r)
	if err := Validate(root, build); err == nil || !strings.Contains(err.Error(), "motion cohort") {
		t.Fatalf("over-age motion error = %v", err)
	}
	r = repocontracttest.ReadJSONFileInto[Receipt](t, path)
	r.Baseline.P95FrameAgeMS = 80
	r.SlowReader.ReceivedFrames = r.SlowReader.RenderedFrames
	testutil.WriteJSONFile(t, path, r)
	if err := Validate(root, build); err == nil || !strings.Contains(err.Error(), "slow-reader cohort") {
		t.Fatalf("unbounded slow-reader error = %v", err)
	}
}

func TestValidateRejectsChangedSourcesArtifactsAndStaleBuild(t *testing.T) {
	root, build := writeValidReceipt(t)
	if err := Validate(root, "sha256:stale"); err == nil || !strings.Contains(err.Error(), "matches live build") {
		t.Fatalf("stale build error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(RequiredSources[1])), []byte("changed source"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Validate(root, build); err == nil || !strings.Contains(err.Error(), "source digest mismatch") {
		t.Fatalf("source tampering error = %v", err)
	}
	root, build = writeValidReceipt(t)
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(EvidenceDir), "motion-owner-live.json"), []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Validate(root, build); err == nil || !strings.Contains(err.Error(), "artifact digest mismatch") {
		t.Fatalf("artifact tampering error = %v", err)
	}
}

func writeValidReceipt(t *testing.T) (string, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "scenarios", "browser-automation-studio")
	const build = "sha256:motion-fixture"
	sources := make(map[string][]byte, len(RequiredSources))
	for _, source := range RequiredSources {
		sources[source] = []byte("source:" + source)
	}
	testutil.WriteFiles(t, root, sources)
	artifactDir := filepath.Join(root, EvidenceDir)
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatal(err)
	}
	artifacts := []artifact{}
	for _, name := range []string{"motion-owner-live.json", "motion-owner-ui-tests.log"} {
		data := []byte("owner evidence:" + name)
		if err := os.WriteFile(filepath.Join(artifactDir, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		artifacts = append(artifacts, artifact{Path: filepath.ToSlash(filepath.Join(EvidenceDir, name)), SHA256: hex.EncodeToString(sum[:])})
	}
	sourceHashes := make(map[string]string, len(RequiredSources))
	for _, source := range RequiredSources {
		sum := sha256.Sum256([]byte("source:" + source))
		sourceHashes[source] = hex.EncodeToString(sum[:])
	}
	contractSum := sha256.Sum256([]byte("source:docs/internal/REFRACTOR_CONTRACT.json"))
	receipt := Receipt{
		SchemaVersion: 1, ContractRow: ContractRow, Result: "passed", BuildIdentity: build,
		ContractSHA: hex.EncodeToString(contractSum[:]), SourceSHA256: sourceHashes,
		Baseline:   baseline{DurationMS: 300_000, RenderedFrames: 9_100, UniqueFixtureFrames: 9_100, RenderedFPS: 30.33, P95FrameAgeMS: 80, MaxFrameAgeMS: 94, MaxFrameBytes: 120_000, P95DecodeMS: 4, MaxDecodeMS: 8},
		SlowReader: slowReader{DurationMS: 15_000, StallCount: 3, ReceivedFrames: 450, DecodedFrames: 220, RenderedFrames: 220, MaxConcurrentDecodes: 1, MaxFrameAgeMS: 530, MaxFrameBytes: 120_000, MaxAPIQueueBytes: 9_000_000, APIQueueBudgetBytes: MaxFrameBytes},
		Artifacts:  artifacts,
	}
	path := filepath.Join(artifactDir, "motion-receipt-test.json")
	testutil.WriteJSONFile(t, path, receipt)
	return root, build
}
