package runs

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"test-genie/internal/executionevidence"
	sharedartifacts "test-genie/internal/shared/artifacts"
	sharedruns "test-genie/internal/shared/runs"
)

func seedCohortScenario(t *testing.T, scenariosRoot, artifactRoot, scenario, runID, phaseStatus, buildID string) {
	t.Helper()
	source := filepath.Join(scenariosRoot, scenario, ".vrooli")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "service.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(artifactRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	record := sharedruns.RunRecord{RunID: runID, Scenario: scenario, TargetKind: "scenario", TargetID: scenario, StartedAt: time.Now().UTC(), CompletedAt: time.Now().UTC(), Status: sharedruns.StatusFailed, GitSha: buildID, TreeDigest: "tree:" + scenario, Phases: []sharedruns.PhaseRecord{{Name: "experience", Status: phaseStatus}}}
	if phaseStatus == "passed" {
		record.Status = sharedruns.StatusPassed
	}
	index := sharedruns.NewIndex(artifactRoot)
	if err := index.Append(record); err != nil {
		t.Fatal(err)
	}
	runRoot := sharedartifacts.RunDir(artifactRoot, runID)
	if err := os.MkdirAll(runRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runRoot, "findings.json"), []byte(`{"phases":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	writer, err := executionevidence.NewWriter(runRoot)
	if err != nil {
		t.Fatal(err)
	}
	ref, err := writer.ReferenceExisting("findings", "findings.document", "findings.json", "application/json", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteManifest(executionevidence.Manifest{
		SchemaVersion: executionevidence.SchemaVersion, RunID: runID, Scenario: scenario, CreatedAt: time.Now().UTC(), Verdict: phaseStatus, Findings: ref,
		Phases: []executionevidence.PhaseSummary{{Name: "experience", Status: phaseStatus, Findings: &ref}},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestPersistExperienceCohortReceiptStoresCanonicalMixedEvidenceAndReplayIsImmutable(t *testing.T) {
	scenariosRoot := t.TempDir()
	artifactRoot := filepath.Join(t.TempDir(), "coverage")
	for _, scenario := range []struct{ name, run, status, build string }{
		{"brand-manager", "run-brand", "not_evaluated", "build-a"},
		{"nutrition-planner", "run-nutrition", "failed", ""},
		{"web-console", "run-web", "passed", "build-c"},
	} {
		seedCohortScenario(t, scenariosRoot, filepath.Join(artifactRoot, scenario.name), scenario.name, scenario.run, scenario.status, scenario.build)
	}
	svc := NewService(scenariosRoot, nil, nil, nil).SetArtifactRootResolver(func(scenario string) (string, error) { return filepath.Join(artifactRoot, scenario), nil })
	input := ExperienceCohortInput{Scenarios: []ExperienceCohortRunInput{
		{Scenario: "brand-manager", RunID: "run-brand", BuildIdentity: "build-a", InvokerStatus: "not_evaluated"},
		{Scenario: "nutrition-planner", RunID: "run-nutrition", InvokerStatus: "fail"},
		{Scenario: "web-console", RunID: "run-web", BuildIdentity: "build-c", InvokerStatus: "pass"},
	}}
	first, err := svc.PersistExperienceCohortReceipt(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if first.Receipt.Status != "failed" || first.Receipt.SchemaVersion != 1 || len(first.Receipt.Scenarios) != 3 {
		t.Fatalf("receipt = %+v", first.Receipt)
	}
	want := []string{"not_evaluated", "failed", "passed"}
	for i, status := range want {
		row := first.Receipt.Scenarios[i]
		if row.Verdict != status {
			t.Fatalf("row[%d] verdict = %q, want %q", i, row.Verdict, status)
		}
		if row.RunID == "" || row.Findings == nil || row.Findings.SHA256 == "" || len(row.Evidence) == 0 {
			t.Fatalf("row[%d] missing canonical evidence: %+v", i, row)
		}
	}
	if len(first.Receipt.Scenarios[1].Missing) == 0 {
		t.Fatal("missing build identity was not recorded")
	}
	storeRoot := filepath.Join(artifactRoot, "test-genie")
	path := filepath.Join(storeRoot, filepath.FromSlash(first.Artifact.RelativePath))
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var stored ExperienceCohortReceipt
	if err := json.Unmarshal(before, &stored); err != nil {
		t.Fatal(err)
	}
	second, err := svc.PersistExperienceCohortReceipt(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if second.Receipt.ReceiptID != first.Receipt.ReceiptID || second.Artifact.SHA256 != first.Artifact.SHA256 {
		t.Fatalf("replay changed immutable receipt: first=%+v second=%+v", first, second)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("replay overwrote the original receipt bytes")
	}
	loaded, err := svc.GetExperienceCohortReceipt(context.Background(), first.Receipt.ReceiptID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Artifact.SHA256 != first.Artifact.SHA256 || loaded.Receipt.Status != first.Receipt.Status {
		t.Fatalf("loaded durable receipt = %+v", loaded)
	}
}

func TestPersistExperienceCohortReceiptKeepsMissingEvidenceExplicitAndCancellationNonPassing(t *testing.T) {
	scenariosRoot := t.TempDir()
	artifactRoot := filepath.Join(t.TempDir(), "coverage")
	scenario := filepath.Join(scenariosRoot, "demo", ".vrooli")
	if err := os.MkdirAll(scenario, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scenario, "service.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(artifactRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := sharedruns.NewIndex(artifactRoot).Append(sharedruns.RunRecord{RunID: "run-missing", Scenario: "demo", TargetKind: "scenario", TargetID: "demo", StartedAt: time.Now().UTC(), CompletedAt: time.Now().UTC(), Status: sharedruns.StatusAborted}); err != nil {
		t.Fatal(err)
	}
	svc := NewService(scenariosRoot, nil, nil, nil).SetArtifactRootResolver(func(string) (string, error) { return artifactRoot, nil })
	input := ExperienceCohortInput{Cancelled: true, Scenarios: []ExperienceCohortRunInput{{Scenario: "demo", RunID: "run-missing", InvokerStatus: "cancelled", BuildIdentity: ""}}}
	stored, err := svc.PersistExperienceCohortReceipt(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	row := stored.Receipt.Scenarios[0]
	if stored.Receipt.Status != "cancelled" || row.Verdict != "" {
		t.Fatalf("cancelled/missing-evidence receipt = %+v", stored.Receipt)
	}
	for _, want := range []string{"returned_build_identity", "evidence_manifest", "canonical_verdict", "findings_artifact"} {
		found := false
		for _, missing := range row.Missing {
			if missing == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing provenance %q not represented: %+v", want, row.Missing)
		}
	}
}

func TestPersistExperienceCohortReceiptCanRecordCancellationBeforeAnyRunExists(t *testing.T) {
	scenariosRoot := t.TempDir()
	stateRoot := filepath.Join(t.TempDir(), "test-genie-state")
	svc := NewService(scenariosRoot, nil, nil, nil).SetArtifactRootResolver(func(scenario string) (string, error) {
		if scenario != "test-genie" {
			t.Fatalf("fallback artifact target = %q", scenario)
		}
		return stateRoot, nil
	})
	stored, err := svc.PersistExperienceCohortReceipt(context.Background(), ExperienceCohortInput{
		Cancelled: true,
		Scenarios: []ExperienceCohortRunInput{{Scenario: "brand-manager", InvokerStatus: "cancelled"}, {Scenario: "nutrition-planner", InvokerStatus: "cancelled"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if stored.Receipt.Status != "cancelled" || len(stored.Receipt.Scenarios) != 2 || stored.Artifact.SHA256 == "" {
		t.Fatalf("cancel-before-start receipt = %+v", stored)
	}
	for _, row := range stored.Receipt.Scenarios {
		if row.Verdict != "" || len(row.Missing) == 0 {
			t.Fatalf("cancellation invented a verdict or omitted provenance gap: %+v", row)
		}
	}
	if _, err := os.Stat(filepath.Join(stateRoot, filepath.FromSlash(stored.Artifact.RelativePath))); err != nil {
		t.Fatalf("durable cancellation artifact missing: %v", err)
	}
}

func TestPersistExperienceCohortReceiptMarksUnsafeBuildIdentityMissing(t *testing.T) {
	scenariosRoot := t.TempDir()
	artifactRoot := filepath.Join(t.TempDir(), "coverage")
	seedCohortScenario(t, scenariosRoot, filepath.Join(artifactRoot, "demo"), "demo", "run-unsafe", "passed", "source-hash")
	svc := NewService(scenariosRoot, nil, nil, nil).SetArtifactRootResolver(func(string) (string, error) { return filepath.Join(artifactRoot, "demo"), nil })
	stored, err := svc.PersistExperienceCohortReceipt(context.Background(), ExperienceCohortInput{Scenarios: []ExperienceCohortRunInput{{Scenario: "demo", RunID: "run-unsafe", BuildIdentity: "https://private.example/token=secret"}}})
	if err != nil {
		t.Fatal(err)
	}
	row := stored.Receipt.Scenarios[0]
	if row.BuildIdentity != "" {
		t.Fatalf("unsafe build identity leaked: %q", row.BuildIdentity)
	}
	for _, want := range []string{"invalid_returned_build_identity", "returned_build_identity"} {
		if !containsCohortValue(row.Missing, want) {
			t.Fatalf("missing %q in %+v", want, row.Missing)
		}
	}
	encoded, err := json.Marshal(stored.Receipt)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "private.example") || strings.Contains(string(encoded), "secret") {
		t.Fatalf("unsafe value leaked in receipt: %s", encoded)
	}
}

func containsCohortValue(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
