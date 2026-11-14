package validation

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	scenariov1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"workflow-health/internal/execution"
	"workflow-health/internal/workflows"
)

func evidenceFixture(t *testing.T) (*connectHandler, string, string) {
	t.Helper()
	root := t.TempDir()
	target := filepath.Join(root, "scenarios/demo")
	put := func(name, value string) {
		t.Helper()
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(value), 0644); err != nil {
			t.Fatal(err)
		}
	}
	put("scenarios/demo/.vrooli/testing.json", `{"version":"1.0.0","workflow":{"evidence":{"common_inputs":["scenarios/demo/shared.txt"],"scopes":[{"id":"settings","case_patterns":["bas/cases/settings.json"],"inputs":["scenarios/demo/settings.txt"]}]}}}`)
	put("scenarios/demo/shared.txt", "shared")
	put("scenarios/demo/settings.txt", "settings")
	put("scenarios/demo/unrelated.txt", "unrelated")
	put("scenarios/demo/bas/cases/settings.json", `{"metadata":{"name":"settings","execution_mode":"observer","labels":{"reset":"none"}},"nodes":[]}`)
	// An identity-owner double checks the requested bytes, not a hard-coded
	// identity. The real Test Genie owner is exercised by consumer qualification.
	resolve := func(ctx context.Context, req *validationv1.ResolveSourceIdentityRequest) (string, error) {
		var bytes []byte
		for _, input := range req.GetContentInputs() {
			var paths []string
			for _, selection := range input.GetSelections() {
				matches, err := filepath.Glob(filepath.Join(root, input.GetRoot(), selection.GetGlob()))
				if err != nil {
					return "", err
				}
				if len(matches) == 0 && selection.GetRequired() {
					return "", fmt.Errorf("missing required input")
				}
				paths = append(paths, matches...)
			}
			slices.Sort(paths)
			for _, p := range paths {
				b, err := os.ReadFile(p)
				if err != nil {
					return "", err
				}
				bytes = append(bytes, []byte(p)...)
				bytes = append(bytes, b...)
			}
		}
		return fmt.Sprintf("ci:v1:%x", sha256.Sum256(bytes)), nil
	}
	h := NewConnectHandler(Deps{RepoRoot: root, EvidenceIdentity: resolve, EvidenceRuntimeFresh: func(context.Context, string) error { return nil }})
	return h, root, target
}

func evidenceRun(t *testing.T, snapshots []evidenceSnapshot) *scenariov1.ValidationRun {
	t.Helper()
	detail, err := nativeDetail(providerReport{Evidence: snapshots, Runs: []execution.WorkflowRun{{Asset: workflows.WorkflowAsset{Path: "bas/cases/settings.json"}, Success: true, Status: "completed"}}})
	if err != nil {
		t.Fatal(err)
	}
	native, err := anypb.New(detail)
	if err != nil {
		t.Fatal(err)
	}
	return &scenariov1.ValidationRun{TerminalResult: &scenariov1.ValidateScenarioResponse{NativeDetail: native}}
}

func currentEvidenceState(t *testing.T, run *scenariov1.ValidationRun) string {
	t.Helper()
	var native structpb.Struct
	if err := run.GetTerminalResult().GetNativeDetail().UnmarshalTo(&native); err != nil {
		t.Fatal(err)
	}
	current := native.AsMap()["current_evidence"].(map[string]any)
	scopes := current["scopes"].([]any)
	if len(scopes) != 1 {
		t.Fatalf("scopes=%v", scopes)
	}
	return scopes[0].(map[string]any)["state"].(string)
}

func TestBrowserEvidenceSurvivesUnrelatedEditsButInvalidatesChangedDependencies(t *testing.T) {
	for _, tc := range []struct {
		name, path string
		want       string
	}{
		{"unrelated feature", "unrelated.txt", "current"},
		{"settings implementation", "settings.txt", "stale"},
		{"shared shell", "shared.txt", "stale"},
		{"test case", "bas/cases/settings.json", "stale"},
		{"scope declaration", ".vrooli/testing.json", "stale"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, _, target := evidenceFixture(t)
			snapshots := h.snapshotEvidence(context.Background(), "demo", target, []string{"bas/cases/settings.json"})
			snapshots = h.qualifyEvidence(context.Background(), snapshots)
			if len(snapshots) != 1 || snapshots[0].State != "qualified" {
				t.Fatalf("unqualified initial snapshot: %+v", snapshots)
			}
			retained := evidenceRun(t, snapshots)
			before, _ := json.Marshal(retained)
			p := filepath.Join(target, tc.path)
			b, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, append(b, []byte("\nchanged")...), 0644); err != nil {
				t.Fatal(err)
			}
			view := h.currentEvidence(context.Background(), retained)
			if got := currentEvidenceState(t, view); got != tc.want {
				t.Fatalf("state=%s want %s", got, tc.want)
			}
			after, _ := json.Marshal(retained)
			if string(before) != string(after) {
				t.Fatal("read projection rewrote retained evidence")
			}
		})
	}
}

func TestBrowserEvidenceRejectsMidRunDriftMissingInputsAndStaleRuntime(t *testing.T) {
	h, _, target := evidenceFixture(t)
	snapshots := h.snapshotEvidence(context.Background(), "demo", target, []string{"bas/cases/settings.json"})
	if err := os.WriteFile(filepath.Join(target, "settings.txt"), []byte("changed during run"), 0644); err != nil {
		t.Fatal(err)
	}
	snapshots = h.qualifyEvidence(context.Background(), snapshots)
	if snapshots[0].State != "stale" {
		t.Fatalf("mid-run drift: %+v", snapshots)
	}
	h.deps.EvidenceRuntimeFresh = func(context.Context, string) error { return fmt.Errorf("stale served build") }
	s := h.snapshotEvidence(context.Background(), "demo", target, []string{"bas/cases/settings.json"})
	s = h.qualifyEvidence(context.Background(), s)
	if s[0].State != "unknown" || !strings.Contains(s[0].Reason, "stale served build") {
		t.Fatalf("stale build qualified: %+v", s)
	}
}

func TestBrowserEvidenceHistoricalRunsRemainUnknown(t *testing.T) {
	h, _, _ := evidenceFixture(t)
	run := evidenceRun(t, nil)
	view := h.currentEvidence(context.Background(), run)
	var native structpb.Struct
	if err := view.GetTerminalResult().GetNativeDetail().UnmarshalTo(&native); err != nil {
		t.Fatal(err)
	}
	current := native.AsMap()["current_evidence"].(map[string]any)
	if current["historical_without_snapshot"] != true {
		t.Fatalf("historical evidence upgraded: %v", current)
	}
}

func TestBrowserEvidenceDoesNotInvalidateTheRunForAnUnrelatedEditDuringExecution(t *testing.T) {
	h, _, target := evidenceFixture(t)
	checks := 0
	h.deps.EvidenceRuntimeFresh = func(context.Context, string) error {
		checks++
		if checks > 1 {
			return fmt.Errorf("unrelated source edited since the running build")
		}
		return nil
	}
	s := h.snapshotEvidence(context.Background(), "demo", target, []string{"bas/cases/settings.json"})
	if err := os.WriteFile(filepath.Join(target, "unrelated.txt"), []byte("unrelated update"), 0644); err != nil {
		t.Fatal(err)
	}
	s = h.qualifyEvidence(context.Background(), s)
	if len(s) != 1 || s[0].State != "qualified" {
		t.Fatalf("unrelated source invalidated unchanged evidence: %+v", s)
	}
}

func TestBrowserEvidenceTracksTransitiveBASFragments(t *testing.T) {
	h, _, target := evidenceFixture(t)
	writeWorkflowJSON(t, filepath.Join(target, "bas/cases/settings.json"), `{"metadata":{"name":"settings"},"nodes":[{"id":"fragment","action":{"subflow":{"workflow_path":"actions/fragment.json"}}}]}`)
	writeWorkflowJSON(t, filepath.Join(target, "bas/actions/fragment.json"), `{"metadata":{"name":"fragment"},"nodes":[{"id":"nested","action":{"subflow":{"workflow_path":"actions/nested.json"}}}]}`)
	writeWorkflowJSON(t, filepath.Join(target, "bas/actions/nested.json"), `{"metadata":{"name":"nested"},"nodes":[]}`)
	s := h.snapshotEvidence(context.Background(), "demo", target, []string{"bas/cases/settings.json"})
	s = h.qualifyEvidence(context.Background(), s)
	if len(s) != 1 || s[0].State != "qualified" {
		t.Fatalf("fragment closure unavailable: %+v", s)
	}
	retained := evidenceRun(t, s)
	if err := os.WriteFile(filepath.Join(target, "bas/actions/nested.json"), []byte(`{"metadata":{"name":"changed"},"nodes":[]}`), 0644); err != nil {
		t.Fatal(err)
	}
	if state := currentEvidenceState(t, h.currentEvidence(context.Background(), retained)); state != "stale" {
		t.Fatalf("nested fragment edit retained evidence: %s", state)
	}
}

func TestBrowserEvidenceMissingDependenciesAndAmbiguousScopesRemainUnknown(t *testing.T) {
	for _, kind := range []string{"missing input", "duplicate scope", "owner unavailable"} {
		t.Run(kind, func(t *testing.T) {
			h, _, target := evidenceFixture(t)
			switch kind {
			case "missing input":
				if err := os.Remove(filepath.Join(target, "shared.txt")); err != nil {
					t.Fatal(err)
				}
			case "duplicate scope":
				p := filepath.Join(target, ".vrooli/testing.json")
				b, _ := os.ReadFile(p)
				s := strings.Replace(string(b), `"scopes":[`, `"scopes":[{"id":"settings","case_patterns":["bas/cases/settings.json"],"inputs":["scenarios/demo/settings.txt"]},`, 1)
				if err := os.WriteFile(p, []byte(s), 0644); err != nil {
					t.Fatal(err)
				}
			case "owner unavailable":
				h.deps.EvidenceIdentity = func(context.Context, *validationv1.ResolveSourceIdentityRequest) (string, error) {
					return "", fmt.Errorf("identity owner unavailable")
				}
			}
			s := h.snapshotEvidence(context.Background(), "demo", target, []string{"bas/cases/settings.json"})
			if len(s) != 1 || s[0].State != "unknown" || s[0].Reason == "" {
				t.Fatalf("unproven dependency closure qualified: %+v", s)
			}
		})
	}
}

func TestBrowserEvidenceScopesInvalidateIndependentlyAndShareCommonChanges(t *testing.T) {
	for _, tc := range []struct {
		name, file string
		want       map[string]string
	}{
		{"local feature edit", "settings.txt", map[string]string{"settings": "stale", "other": "current"}},
		{"shared shell edit", "shared.txt", map[string]string{"settings": "stale", "other": "stale"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, _, target := evidenceFixture(t)
			writeWorkflowJSON(t, filepath.Join(target, "bas/cases/other.json"), `{"metadata":{"name":"other"},"nodes":[]}`)
			if err := os.WriteFile(filepath.Join(target, "other.txt"), []byte("other feature"), 0644); err != nil {
				t.Fatal(err)
			}
			config := `{"version":"1.0.0","workflow":{"evidence":{"common_inputs":["scenarios/demo/shared.txt"],"scopes":[{"id":"settings","case_patterns":["bas/cases/settings.json"],"inputs":["scenarios/demo/settings.txt"]},{"id":"other","case_patterns":["bas/cases/other.json"],"inputs":["scenarios/demo/other.txt"]}]}}}`
			if err := os.WriteFile(filepath.Join(target, ".vrooli/testing.json"), []byte(config), 0644); err != nil {
				t.Fatal(err)
			}
			snapshots := h.qualifyEvidence(context.Background(), h.snapshotEvidence(context.Background(), "demo", target, []string{"bas/cases/settings.json", "bas/cases/other.json"}))
			if len(snapshots) != 2 {
				t.Fatalf("expected two evidence scopes: %+v", snapshots)
			}
			for _, snapshot := range snapshots {
				if snapshot.State != "qualified" {
					t.Fatalf("initial evidence unqualified: %+v", snapshots)
				}
			}
			retained := evidenceRun(t, snapshots)
			if err := os.WriteFile(filepath.Join(target, tc.file), []byte("changed"), 0644); err != nil {
				t.Fatal(err)
			}
			view := h.currentEvidence(context.Background(), retained)
			var native structpb.Struct
			if err := view.GetTerminalResult().GetNativeDetail().UnmarshalTo(&native); err != nil {
				t.Fatal(err)
			}
			current := native.AsMap()["current_evidence"].(map[string]any)
			for _, raw := range current["scopes"].([]any) {
				scope := raw.(map[string]any)
				name := scope["scope"].(string)
				if scope["state"] != tc.want[name] {
					t.Fatalf("%s state=%v want=%s", name, scope["state"], tc.want[name])
				}
			}
		})
	}
}
