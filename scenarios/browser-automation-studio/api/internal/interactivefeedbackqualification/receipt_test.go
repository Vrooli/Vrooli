package interactivefeedbackqualification

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateRequiresBothCurrentCohorts(t *testing.T) {
	root := t.TempDir()
	writeFixtureFiles(t, root)
	writeReceipt(t, root, "local", "sha256:current")
	if err := Validate(root, "sha256:current"); err == nil {
		t.Fatal("Validate accepted a single cohort")
	}
	writeReceipt(t, root, "remote", "sha256:current")
	if err := Validate(root, "sha256:current"); err != nil {
		t.Fatalf("Validate rejected complete cohorts: %v", err)
	}
}

func TestValidateRejectsStaleOrOutOfBandCohort(t *testing.T) {
	root := t.TempDir()
	writeFixtureFiles(t, root)
	writeReceipt(t, root, "local", "sha256:stale")
	writeReceipt(t, root, "remote", "sha256:current")
	if err := Validate(root, "sha256:current"); err == nil {
		t.Fatal("Validate accepted a stale cohort")
	}
}

func writeFixtureFiles(t *testing.T, root string) {
	t.Helper()
	for _, relative := range []string{contractPath, RequiredSourceFile} {
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(relative+" fixture\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func writeReceipt(t *testing.T, root, cohort, build string) {
	t.Helper()
	contractSHA := shaFor(t, filepath.Join(root, filepath.FromSlash(contractPath)))
	testSHA := shaFor(t, filepath.Join(root, filepath.FromSlash(RequiredSourceFile)))
	path := filepath.Join(root, filepath.FromSlash(".vrooli/runtime/rehabilitation-evidence/interactive-feedback-"+cohort+"-test.json"))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	receipt := map[string]any{
		"evidence_kind": "interactive_feedback_cohort", "outcome_id": "interactive-feedback", "status": "passed",
		"managed_build_identity": build, "cohort": cohort,
		"source_sha256": map[string]string{contractPath: contractSHA, RequiredSourceFile: testSHA},
		"measurement": map[string]any{
			"sample_count": 1000, "correlated_receipt_count": 1000, "correlated_canvas_paint_count": 1000,
			"correlation_complete": true, "receipt_sequences_monotonic": true,
			"p50_ms": 40, "p95_ms": 80, "p99_ms": 150,
		},
	}
	data, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func shaFor(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
