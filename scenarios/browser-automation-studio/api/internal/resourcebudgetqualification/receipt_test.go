package resourcebudgetqualification

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

func TestValidateAcceptsCurrentBoundedLinuxReceipt(t *testing.T) {
	root, build := writeValidFixture(t)
	if err := Validate(root, build); err != nil {
		t.Fatalf("valid current resource receipt rejected: %v", err)
	}
}

func TestValidateRejectsStaleBuildAndOverBudgetPSS(t *testing.T) {
	root, build := writeValidFixture(t)
	if err := Validate(root, "sha256:stale"); err == nil || !strings.Contains(err.Error(), "matches live build") {
		t.Fatalf("stale build error = %v", err)
	}
	path := filepath.Join(root, EvidenceDir, "resource-budget-w189-fixture.json")
	r := repocontracttest.ReadJSONFileInto[receipt](t, path)
	r.Idle.MaxCombinedPSSKiB = MaxIdlePSSKiB + 1
	testutil.WriteJSONFile(t, path, r)
	if err := Validate(root, build); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("over-budget PSS error = %v", err)
	}
}

func TestValidateRejectsSamplingWindowShortOfSixtySeconds(t *testing.T) {
	root, build := writeValidFixture(t)
	path := filepath.Join(root, EvidenceDir, "resource-budget-w189-fixture.json")
	r := repocontracttest.ReadJSONFileInto[receipt](t, path)
	r.Idle.DurationMS = 59999
	r.Idle.EndedAt = "2026-09-24T00:00:59.999Z"
	testutil.WriteJSONFile(t, path, r)
	if err := Validate(root, build); err == nil || !strings.Contains(err.Error(), "less than 60 seconds") {
		t.Fatalf("short sampling window error = %v", err)
	}
}

func TestValidateRejectsNonIdleSamplesAndUnreportedWindowsState(t *testing.T) {
	root, build := writeValidFixture(t)
	path := filepath.Join(root, EvidenceDir, "resource-budget-w189-fixture.json")
	r := repocontracttest.ReadJSONFileInto[receipt](t, path)
	r.Idle.Samples[22].Sessions = 1
	testutil.WriteJSONFile(t, path, r)
	if err := Validate(root, build); err == nil || !strings.Contains(err.Error(), "non-idle") {
		t.Fatalf("non-idle sample error = %v", err)
	}
	r = repocontracttest.ReadJSONFileInto[receipt](t, path)
	r.Idle.Samples[22].Sessions = 0
	r.Platform.WindowsPrivateMemory.Status = "not_measured"
	r.Platform.WindowsPrivateMemory.Reason = ""
	testutil.WriteJSONFile(t, path, r)
	if err := Validate(root, build); err == nil || !strings.Contains(err.Error(), "Windows private-memory") {
		t.Fatalf("missing Windows status error = %v", err)
	}
}

func TestValidateRejectsSourceAndArtifactTampering(t *testing.T) {
	root, build := writeValidFixture(t)
	source := filepath.Join(root, filepath.FromSlash(RequiredSources[1]))
	if err := os.WriteFile(source, []byte("changed source"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Validate(root, build); err == nil || !strings.Contains(err.Error(), "source digest mismatch") {
		t.Fatalf("source tampering error = %v", err)
	}
	root, build = writeValidFixture(t)
	artifact := filepath.Join(root, ".vrooli/runtime/rehabilitation-evidence/resource-budget-owner.json")
	if err := os.WriteFile(artifact, []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Validate(root, build); err == nil || !strings.Contains(err.Error(), "artifact digest mismatch") {
		t.Fatalf("artifact tampering error = %v", err)
	}
}

func writeValidFixture(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	const build = "sha256:resource-fixture"
	for _, relative := range RequiredSources {
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("source:"+relative), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	artifactPath := filepath.Join(root, ".vrooli/runtime/rehabilitation-evidence/resource-budget-owner.json")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
		t.Fatal(err)
	}
	artifact := []byte("owner measurement payload\n")
	if err := os.WriteFile(artifactPath, artifact, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(artifact)
	contractSum := sha256.Sum256([]byte("source:docs/internal/REFRACTOR_CONTRACT.json"))
	sources := make(map[string]string, len(RequiredSources))
	for _, relative := range RequiredSources {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(data)
		sources[relative] = hex.EncodeToString(digest[:])
	}
	if sources["docs/internal/REFRACTOR_CONTRACT.json"] != hex.EncodeToString(contractSum[:]) {
		t.Fatal("fixture contract digest mismatch")
	}
	samples := make([]map[string]any, 61)
	for index := range samples {
		var cpu any
		if index > 0 {
			cpu = map[string]float64{"api": 0.1, "driver": 0.1}
		}
		samples[index] = map[string]any{
			"pssKiB":           map[string]int{"api": 40000, "driver": 50000},
			"combinedPssKiB":   90000,
			"cpuPercent":       cpu,
			"sessions":         0,
			"activeRecordings": 0,
		}
	}
	r := map[string]any{
		"schemaVersion":              1,
		"contractRow":                ContractRow,
		"result":                     "passed",
		"managedBuildIdentityBefore": build,
		"managedBuildIdentityAfter":  build,
		"platform":                   map[string]any{"os": "linux", "windowsPrivateMemory": map[string]string{"status": "not_measured", "reason": "Linux runner"}},
		"idleSampling": map[string]any{
			"startedAt": "2026-09-24T00:00:00Z", "endedAt": "2026-09-24T00:01:01Z", "durationMs": 61000,
			"sampleIntervalMs": 1000, "sampleCount": 61, "sessionsThroughout": 0, "recordingsThroughout": 0,
			"maxCombinedPssKiB": 90000, "averageCombinedCPUPercent": 0.2, "p95CombinedCPUPercent": 0.2, "samples": samples,
		},
		"fixtureBrowserAndShell": map[string]any{
			"browserProcessCount": 1, "browserPids": []map[string]int{{"pid": 10, "pssKiB": 3000}},
			"browserPssKiB": 3000, "shellPssKiB": 1000, "combinedPssKiB": 4000,
		},
		"cleanup":       map[string]bool{"fixtureSessionClosed": true, "apiHealthy": true},
		"source_sha256": sources,
		"owner_artifact": map[string]string{
			"path":   ".vrooli/runtime/rehabilitation-evidence/resource-budget-owner.json",
			"sha256": hex.EncodeToString(sum[:]),
		},
	}
	testutil.WriteJSONFile(t, filepath.Join(root, EvidenceDir, "resource-budget-w189-fixture.json"), r)
	return root, build
}
