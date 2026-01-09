package evidence

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"test-genie/internal/requirements/types"
)

// memReader implements Reader for testing.
type memReader struct {
	files map[string][]byte
	dirs  map[string][]fs.DirEntry
}

func newMemReader() *memReader {
	return &memReader{
		files: make(map[string][]byte),
		dirs:  make(map[string][]fs.DirEntry),
	}
}

func (r *memReader) ReadFile(path string) ([]byte, error) {
	if data, ok := r.files[path]; ok {
		return data, nil
	}
	return nil, os.ErrNotExist
}

func (r *memReader) ReadDir(path string) ([]fs.DirEntry, error) {
	if entries, ok := r.dirs[path]; ok {
		return entries, nil
	}
	return nil, os.ErrNotExist
}

func (r *memReader) Exists(path string) bool {
	_, hasFile := r.files[path]
	_, hasDir := r.dirs[path]
	return hasFile || hasDir
}

type memDirEntry struct {
	name  string
	isDir bool
}

func (e *memDirEntry) Name() string               { return e.name }
func (e *memDirEntry) IsDir() bool                { return e.isDir }
func (e *memDirEntry) Type() fs.FileMode          { return 0 }
func (e *memDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

func TestLoader_LoadAll_Empty(t *testing.T) {
	reader := newMemReader()
	loader := New(reader)

	bundle, err := loader.LoadAll(context.Background(), "/test/scenario")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if bundle == nil {
		t.Fatal("expected non-nil bundle")
	}
	if !bundle.IsEmpty() {
		t.Error("expected empty bundle")
	}
}

func TestLoader_LoadPhaseResults_NoDirectory(t *testing.T) {
	reader := newMemReader()
	loader := New(reader)

	results, err := loader.LoadPhaseResults(context.Background(), "/test/scenario")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected empty results, got: %d", len(results))
	}
}

func TestLoader_LoadPhaseResults_WithFiles(t *testing.T) {
	reader := newMemReader()
	reader.dirs["/test/scenario/coverage/phase-results"] = []fs.DirEntry{
		&memDirEntry{name: "unit.json", isDir: false},
	}
	reader.files["/test/scenario/coverage/phase-results/unit.json"] = []byte(`{
		"phase": "unit",
		"status": "passed",
		"duration_seconds": 5.5,
		"test_count": 10,
		"pass_count": 10,
		"requirement_ids": ["REQ-001", "REQ-002"]
	}`)

	loader := New(reader)
	results, err := loader.LoadPhaseResults(context.Background(), "/test/scenario")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected non-empty results")
	}
}

func TestLoader_LoadVitestEvidence_NoFile(t *testing.T) {
	reader := newMemReader()
	loader := New(reader)

	evidence, err := loader.LoadVitestEvidence(context.Background(), "/test/scenario")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(evidence) != 0 {
		t.Errorf("expected empty evidence, got: %d", len(evidence))
	}
}

func TestLoader_LoadVitestEvidence_WithFile(t *testing.T) {
	reader := newMemReader()
	// Vitest coverage file expects files array with requirements list
	reader.files["/test/scenario/ui/coverage/vitest-requirements.json"] = []byte(`{
		"files": [
			{
				"path": "src/component.ts",
				"requirements": ["REQ-001"],
				"coverage": 85.5,
				"coveredLines": 100,
				"totalLines": 117
			}
		],
		"testFiles": [
			{
				"path": "src/component.test.ts",
				"requirementId": "REQ-001",
				"status": "passed",
				"duration": 0.5
			}
		]
	}`)

	loader := New(reader)
	evidence, err := loader.LoadVitestEvidence(context.Background(), "/test/scenario")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(evidence) == 0 {
		t.Error("expected non-empty evidence")
	}
	if results, ok := evidence["REQ-001"]; !ok {
		t.Error("expected evidence for REQ-001")
	} else if len(results) == 0 {
		t.Error("expected at least 1 result for REQ-001")
	}
}

func TestLoader_LoadManualValidations_NoFile(t *testing.T) {
	reader := newMemReader()
	loader := New(reader)

	manifest, err := loader.LoadManualValidations(context.Background(), "/test/scenario")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if manifest != nil && manifest.Count() > 0 {
		t.Error("expected empty manifest")
	}
}

func TestLoader_LoadManualValidations_WithFile(t *testing.T) {
	reader := newMemReader()
	reader.files["/test/scenario/coverage/manual-validations/log.jsonl"] = []byte(`{"requirement_id": "REQ-001", "status": "passed", "validated_at": "2024-01-01T00:00:00Z"}
{"requirement_id": "REQ-002", "status": "failed", "validated_at": "2024-01-02T00:00:00Z"}`)

	loader := New(reader)
	manifest, err := loader.LoadManualValidations(context.Background(), "/test/scenario")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if manifest == nil {
		t.Fatal("expected non-nil manifest")
	}
	if manifest.Count() != 2 {
		t.Errorf("expected 2 entries, got: %d", manifest.Count())
	}
	if v, ok := manifest.Get("REQ-001"); !ok {
		t.Error("expected REQ-001 in manifest")
	} else if v.Status != "passed" {
		t.Errorf("unexpected status: %s", v.Status)
	}
}

func TestLoader_LoadAll_WithAllSources(t *testing.T) {
	reader := newMemReader()

	// Phase results
	reader.dirs["/test/scenario/coverage/phase-results"] = []fs.DirEntry{
		&memDirEntry{name: "unit.json", isDir: false},
	}
	reader.files["/test/scenario/coverage/phase-results/unit.json"] = []byte(`{
		"phase": "unit",
		"status": "passed"
	}`)

	// Vitest evidence (correct format with files/testFiles arrays)
	reader.files["/test/scenario/ui/coverage/vitest-requirements.json"] = []byte(`{
		"files": [{"path": "src/test.ts", "requirements": ["REQ-001"]}]
	}`)

	// Manual validations
	reader.files["/test/scenario/coverage/manual-validations/log.jsonl"] = []byte(`{"requirement_id": "REQ-002", "status": "passed"}`)

	loader := New(reader)
	bundle, err := loader.LoadAll(context.Background(), "/test/scenario")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if bundle == nil {
		t.Fatal("expected non-nil bundle")
	}
	if bundle.IsEmpty() {
		t.Error("expected non-empty bundle")
	}
	if len(bundle.VitestEvidence) == 0 {
		t.Error("expected vitest evidence")
	}
	if bundle.ManualValidations == nil || bundle.ManualValidations.Count() == 0 {
		t.Error("expected manual validations")
	}
}

func TestEvidenceBundle_IsEmpty(t *testing.T) {
	bundle := types.NewEvidenceBundle()

	if !bundle.IsEmpty() {
		t.Error("new bundle should be empty")
	}

	bundle.PhaseResults["test"] = []types.EvidenceRecord{{}}

	if bundle.IsEmpty() {
		t.Error("bundle with phase results should not be empty")
	}
}

func TestManualValidation_IsExpired(t *testing.T) {
	pastDate := time.Now().Add(-24 * time.Hour)
	futureDate := time.Now().Add(24 * time.Hour)

	expired := types.ManualValidation{
		RequirementID: "REQ-001",
		ExpiresAt:     pastDate,
	}
	if !expired.IsExpired() {
		t.Error("past expiration should be expired")
	}

	notExpired := types.ManualValidation{
		RequirementID: "REQ-002",
		ExpiresAt:     futureDate,
	}
	if notExpired.IsExpired() {
		t.Error("future expiration should not be expired")
	}

	noExpiration := types.ManualValidation{
		RequirementID: "REQ-003",
	}
	if noExpiration.IsExpired() {
		t.Error("no expiration should not be expired")
	}
}

func TestManualValidation_ToLiveStatus(t *testing.T) {
	passed := types.ManualValidation{
		RequirementID: "REQ-001",
		Status:        "passed",
	}
	if passed.ToLiveStatus() != types.LivePassed {
		t.Errorf("expected passed, got: %s", passed.ToLiveStatus())
	}

	failed := types.ManualValidation{
		RequirementID: "REQ-002",
		Status:        "failed",
	}
	if failed.ToLiveStatus() != types.LiveFailed {
		t.Errorf("expected failed, got: %s", failed.ToLiveStatus())
	}

	expired := types.ManualValidation{
		RequirementID: "REQ-003",
		Status:        "passed",
		ExpiresAt:     time.Now().Add(-24 * time.Hour),
	}
	if expired.ToLiveStatus() != types.LiveNotRun {
		t.Errorf("expired should return not_run, got: %s", expired.ToLiveStatus())
	}
}

func TestEvidenceMap_Merge(t *testing.T) {
	map1 := types.EvidenceMap{
		"REQ-001": []types.EvidenceRecord{{Phase: "unit"}},
	}
	map2 := types.EvidenceMap{
		"REQ-001": []types.EvidenceRecord{{Phase: "integration"}},
		"REQ-002": []types.EvidenceRecord{{Phase: "unit"}},
	}

	map1.Merge(map2)

	if len(map1["REQ-001"]) != 2 {
		t.Errorf("expected 2 records for REQ-001, got: %d", len(map1["REQ-001"]))
	}
	if len(map1["REQ-002"]) != 1 {
		t.Errorf("expected 1 record for REQ-002, got: %d", len(map1["REQ-002"]))
	}
}

// Integration test using real filesystem
func TestLoader_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	scenarioDir := filepath.Join(tmpDir, "scenario")
	phaseResultsDir := filepath.Join(scenarioDir, "coverage", "phase-results")
	manualDir := filepath.Join(scenarioDir, "coverage", "manual-validations")
	vitestDir := filepath.Join(scenarioDir, "ui", "coverage")

	// Create directories
	for _, dir := range []string{phaseResultsDir, manualDir, vitestDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("create dir: %v", err)
		}
	}

	// Write phase result
	phaseData := []byte(`{"phase": "unit", "status": "passed", "test_count": 5}`)
	if err := os.WriteFile(filepath.Join(phaseResultsDir, "unit.json"), phaseData, 0644); err != nil {
		t.Fatalf("write phase result: %v", err)
	}

	// Write vitest evidence
	vitestData := []byte(`[{"requirement_id": "REQ-001", "status": "passed", "file_path": "test.ts"}]`)
	if err := os.WriteFile(filepath.Join(vitestDir, "vitest-requirements.json"), vitestData, 0644); err != nil {
		t.Fatalf("write vitest: %v", err)
	}

	// Write manual validations
	manualData := []byte(`{"requirement_id": "REQ-002", "status": "passed", "validated_at": "2024-01-01T00:00:00Z"}`)
	if err := os.WriteFile(filepath.Join(manualDir, "log.jsonl"), manualData, 0644); err != nil {
		t.Fatalf("write manual: %v", err)
	}

	loader := NewDefault()
	bundle, err := loader.LoadAll(context.Background(), scenarioDir)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if bundle.IsEmpty() {
		t.Error("expected non-empty bundle")
	}

	t.Logf("Loaded evidence bundle:")
	t.Logf("  - Phase results: %d requirement mappings", len(bundle.PhaseResults))
	t.Logf("  - Vitest evidence: %d requirements", len(bundle.VitestEvidence))
	if bundle.ManualValidations != nil {
		t.Logf("  - Manual validations: %d entries", bundle.ManualValidations.Count())
	}
}
