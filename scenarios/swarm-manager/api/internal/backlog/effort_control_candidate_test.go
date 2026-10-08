package backlog

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/identity"
)

func withheldEffortCandidate() identity.CandidatePolicyBinding {
	return identity.CandidatePolicyBinding{
		EconomicalRunner: "opencode",
		EconomicalModel:  "opencode-go/model-one",
		EconomicalEffort: "runner-native",
		FallbackRunner:   "codex",
		FallbackModel:    "gpt-5.6-luna",
		Withheld:         []string{"openrouter/deepseek/deepseek-v4.1-flash"},
	}.Bind()
}

func newEffortControlCandidateService(t *testing.T, effortID string, candidate identity.CandidatePolicyBinding) *EffortControlService {
	t.Helper()
	source := &stubEffortPolicySource{
		bindings:   map[string]identity.PolicyBinding{effortID: {Source: "efforts/" + effortID + "/effort.json", Digest: "sha256:one"}},
		candidates: map[string]identity.CandidatePolicyBinding{effortID: candidate},
	}
	service := NewEffortControlService(NewFileEffortControlStore(t.TempDir()), source)
	if _, err := service.Admit(testEffortControlRequest(effortID, 1)); err != nil {
		t.Fatalf("admit: %v", err)
	}
	return service
}

func TestQualifyLastResortCandidateUsesAdmittedBinding(t *testing.T) {
	service := newEffortControlCandidateService(t, "aquila-one", testEffortCandidate("opencode-go/model-one"))
	qualification, err := service.QualifyLastResortCandidate("aquila-one", identity.Candidate{
		Runner:    "opencode",
		Model:     "opencode-go/model-one",
		Qualified: true,
	})
	if err != nil {
		t.Fatalf("bound qualified candidate refused: %v", err)
	}
	if !qualification.Qualified {
		t.Fatalf("expected a qualified disposition, got %+v", qualification)
	}

	if _, err := service.QualifyLastResortCandidate("aquila-one", identity.Candidate{
		Runner:    "unbound",
		Model:     "unbound-model",
		Qualified: true,
	}); !errors.Is(err, identity.ErrCandidateUnbound) {
		t.Fatalf("expected an unbound candidate to be refused, got %v", err)
	}
}

func TestQualifyLastResortCandidateRefusesWithheldBinding(t *testing.T) {
	service := newEffortControlCandidateService(t, "aquila-one", withheldEffortCandidate())
	_, err := service.QualifyLastResortCandidate("aquila-one", identity.Candidate{
		Runner:    "openrouter",
		Model:     "deepseek/deepseek-v4.1-flash",
		Metered:   true,
		Qualified: true,
	})
	if !errors.Is(err, identity.ErrCandidateWithheld) {
		t.Fatalf("expected the withheld route to be refused before billing, got %v", err)
	}
}

func TestQualifyLastResortCandidateRefusesUnknownEffort(t *testing.T) {
	service := newEffortControlCandidateService(t, "aquila-one", testEffortCandidate("opencode-go/model-one"))
	if _, err := service.QualifyLastResortCandidate("aquila-two", identity.Candidate{Runner: "opencode", Model: "opencode-go/model-one", Qualified: true}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected an unknown effort to be refused, got %v", err)
	}
}

func TestFileEffortPolicySourceProjectsWithheldFallback(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "aquila-one")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	document := map[string]any{
		"slug": "aquila-one",
		"policy": map[string]any{
			"model_selection": map[string]any{
				"worker_preference": map[string]any{
					"runner":          "opencode",
					"model":           "opencode-go/deepseek-v4.1-flash",
					"effort":          "runner-native",
					"fallback_runner": "codex",
					"fallback_model":  "gpt-5.6-luna",
				},
				"conditional_paid_fallback": map[string]any{
					"candidate_runner_model": "openrouter/deepseek/deepseek-v4.1-flash",
					"candidate_model":        "deepseek/deepseek-v4.1-flash",
					"eligibility_state":      "withheld-pending-funding-and-qualification",
				},
			},
		},
	}
	encoded, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "effort.json"), encoded, 0o600); err != nil {
		t.Fatal(err)
	}

	resolution, err := FileEffortPolicySource{Root: root}.ResolveEffortPolicy("aquila-one")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	candidate := resolution.Candidate
	if len(candidate.Withheld) == 0 {
		t.Fatalf("withheld conditional fallback was not projected: %+v", candidate)
	}
	err = candidate.QualifyLastResort(identity.Candidate{
		Runner:    "openrouter",
		Model:     "deepseek/deepseek-v4.1-flash",
		Metered:   true,
		Qualified: true,
	})
	if !errors.Is(err, identity.ErrCandidateWithheld) {
		t.Fatalf("expected the projected withheld fallback to be refused, got %v", err)
	}
}

func TestQualifyEffortCandidateOverHTTP(t *testing.T) {
	source := &stubEffortPolicySource{
		bindings:   map[string]identity.PolicyBinding{"aquila-one": {Source: "efforts/aquila-one/effort.json", Digest: "sha256:one"}},
		candidates: map[string]identity.CandidatePolicyBinding{"aquila-one": withheldEffortCandidate()},
	}
	h := NewHandler(t.TempDir(), t.TempDir())
	h.SetEffortControlService(NewEffortControlService(NewFileEffortControlStore(t.TempDir()), source))
	if recorder := doEffortJSON(t, h.AdmitEffort, http.MethodPost, "aquila-one", testEffortControlRequest("aquila-one", 1)); recorder.Code != http.StatusOK {
		t.Fatalf("admit: %d %s", recorder.Code, recorder.Body.String())
	}

	qualified := doEffortJSON(t, h.QualifyEffortCandidate, http.MethodPost, "aquila-one", effortCandidateQualifyRequest{
		Runner:    "opencode",
		Model:     "opencode-go/model-one",
		Qualified: true,
	})
	if qualified.Code != http.StatusOK {
		t.Fatalf("expected a bound candidate to qualify, got %d %s", qualified.Code, qualified.Body.String())
	}

	withheld := doEffortJSON(t, h.QualifyEffortCandidate, http.MethodPost, "aquila-one", effortCandidateQualifyRequest{
		Runner:    "openrouter",
		Model:     "deepseek/deepseek-v4.1-flash",
		Metered:   true,
		Qualified: true,
	})
	if withheld.Code != http.StatusForbidden {
		t.Fatalf("expected a withheld candidate to be forbidden, got %d %s", withheld.Code, withheld.Body.String())
	}
}
