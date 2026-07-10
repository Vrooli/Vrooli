package modelpolicy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/domain"
)

// [REQ:REQ-P1-004] Named policy resolution produces explicit immutable candidates.
func TestResolveNamedPolicyProducesExplicitImmutableCandidates(t *testing.T) {
	state, err := NewState(ResolvePath(), Requirement{Required: true})
	if err != nil {
		t.Fatalf("new state: %v", err)
	}
	snapshot, err := state.ResolvePolicy("codex.smart")
	if err != nil {
		t.Fatalf("resolve named policy: %v", err)
	}
	if snapshot.PolicyRef != "codex.smart" || snapshot.CatalogDigest == "" {
		t.Fatalf("snapshot identity = %+v", snapshot)
	}
	if snapshot.Explanation.Source != ResolutionSourceNamedPolicy || snapshot.Explanation.RequestedPolicyRef != "codex.smart" {
		t.Fatalf("source = %q", snapshot.Explanation.Source)
	}
	if len(snapshot.Candidates) < 2 {
		t.Fatalf("candidates = %v, want fallback sequence", snapshot.Candidates)
	}
	for index, candidate := range snapshot.Candidates {
		if !candidate.RunnerType.IsValid() {
			t.Fatalf("candidate[%d] invalid runner: %+v", index, candidate)
		}
		switch candidate.SelectionType {
		case domain.ModelSelectionTypeModel:
			if strings.TrimSpace(candidate.Model) == "" {
				t.Fatalf("candidate[%d] uses empty model sentinel: %+v", index, candidate)
			}
		case domain.ModelSelectionTypeRunnerDefault:
			if candidate.Model != "" {
				t.Fatalf("candidate[%d] runner_default carries model: %+v", index, candidate)
			}
		default:
			t.Fatalf("candidate[%d] invalid selection: %+v", index, candidate)
		}
	}
	if snapshot.SelectedCandidate != snapshot.Candidates[0] {
		t.Fatalf("selected = %+v, first = %+v", snapshot.SelectedCandidate, snapshot.Candidates[0])
	}
}

func TestResolveDirectModelRequiresDeclaredCompatibility(t *testing.T) {
	state, err := NewState(ResolvePath(), Requirement{Required: true})
	if err != nil {
		t.Fatalf("new state: %v", err)
	}
	if _, err := state.ResolveDirectModel(domain.RunnerTypeCodex, "retired-model"); err == nil {
		t.Fatal("expected undeclared direct model rejection")
	}
	snapshot, err := state.ResolveDirectModel(domain.RunnerTypeCodex, "ollama/local-model")
	if err != nil {
		t.Fatalf("resolve dynamic direct model: %v", err)
	}
	if snapshot.SelectedCandidate.Model != "ollama/local-model" || snapshot.Explanation.Source != ResolutionSourceDirectModel {
		t.Fatalf("snapshot = %+v", snapshot)
	}
}

func TestResolveRunnerDefaultIsExplicit(t *testing.T) {
	state, err := NewState(ResolvePath(), Requirement{Required: true})
	if err != nil {
		t.Fatalf("new state: %v", err)
	}
	snapshot, err := state.ResolveRunnerDefault(domain.RunnerTypeClaudeCode)
	if err != nil {
		t.Fatalf("resolve runner default: %v", err)
	}
	if snapshot.SelectedCandidate.SelectionType != domain.ModelSelectionTypeRunnerDefault {
		t.Fatalf("selection = %+v", snapshot.SelectedCandidate)
	}
	if snapshot.SelectedCandidate.Model != "" {
		t.Fatalf("runner default manufactured model sentinel %q", snapshot.SelectedCandidate.Model)
	}
}

func TestResolvePolicyFailsWhenRequiredStateHasNoActiveRevision(t *testing.T) {
	state, loadErr := NewState(t.TempDir()+"/missing.json", Requirement{Required: true})
	if loadErr == nil {
		t.Fatal("expected initial load error")
	}
	_, err := state.ResolvePolicy("codex.fast")
	if err == nil || !strings.Contains(err.Error(), "failed to read catalog") {
		t.Fatalf("resolve error = %v, want retained load diagnostic", err)
	}
}

// [REQ:REQ-P1-004] Active catalog changes cannot rewrite an existing run snapshot.
func TestResolvedSnapshotDoesNotChangeAfterCatalogReload(t *testing.T) {
	data, err := os.ReadFile(ResolvePath())
	if err != nil {
		t.Fatalf("read checked-in catalog: %v", err)
	}
	path := filepath.Join(t.TempDir(), "model-policy-catalog.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write catalog fixture: %v", err)
	}
	state, err := NewState(path, Requirement{Required: true})
	if err != nil {
		t.Fatalf("new state: %v", err)
	}
	snapshot, err := state.ResolvePolicy("codex.fast")
	if err != nil {
		t.Fatalf("resolve snapshot: %v", err)
	}
	originalDigest := snapshot.CatalogDigest
	originalCandidates := append([]domain.ExecutionCandidate(nil), snapshot.Candidates...)

	updated := []byte(strings.Replace(string(data), "\"updatedAt\": \"2026-07-09\"", "\"updatedAt\": \"2026-07-10\"", 1))
	if err := os.WriteFile(path, updated, 0o644); err != nil {
		t.Fatalf("write updated catalog: %v", err)
	}
	if _, err := state.Reload(); err != nil {
		t.Fatalf("reload state: %v", err)
	}
	if state.Status().ActiveDigest == originalDigest {
		t.Fatal("fixture reload did not activate a new digest")
	}
	if snapshot.CatalogDigest != originalDigest {
		t.Fatalf("snapshot digest changed from %s to %s", originalDigest, snapshot.CatalogDigest)
	}
	if len(snapshot.Candidates) != len(originalCandidates) {
		t.Fatalf("snapshot candidate count changed: %v", snapshot.Candidates)
	}
	for index := range originalCandidates {
		if snapshot.Candidates[index] != originalCandidates[index] {
			t.Fatalf("snapshot candidate[%d] changed: before=%+v after=%+v", index, originalCandidates[index], snapshot.Candidates[index])
		}
	}
}
