package facts

import (
	"context"
	"strings"
	"testing"
)

func TestSearchInvariantCandidatesSeparatesCandidateFromEnforcement(t *testing.T) {
	response, err := NewService().SearchInvariantCandidates(context.Background(), "cache max bytes negative", "code-facts", "api", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Results) == 0 {
		t.Fatalf("expected source candidate; degradations=%+v", response.Degradations)
	}
	found := false
	for _, hit := range response.Results {
		if strings.HasSuffix(hit.Path, "api/main.go") && hit.EvidenceKind == "runtime-guard candidate" {
			found = true
			if hit.Attestation != "CANDIDATE_ONLY" || hit.SHA256 == "" || hit.Line <= 0 || hit.Enforcement.Type != "UNKNOWN" || hit.Enforcement.Database != "UNKNOWN" || hit.Enforcement.Test != "UNKNOWN" || hit.Enforcement.Runtime != "UNKNOWN" {
				t.Fatalf("candidate incorrectly upgraded to enforcement: %+v", hit)
			}
		}
	}
	if !found {
		t.Fatalf("expected cache guard candidate from main.go, got %+v", response.Results)
	}
}

func TestSearchInvariantCandidatesNegativeAndScopeLimits(t *testing.T) {
	response, err := NewService().SearchInvariantCandidates(context.Background(), "underwater penguin invoice telescope", "code-facts", "api", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Results) != 0 || len(response.Degradations) == 0 || !strings.Contains(response.Degradations[0].Code, "no_candidate") {
		t.Fatalf("negative query should abstain: %+v", response)
	}
	if _, err := NewService().SearchInvariantCandidates(context.Background(), "invariant", "code-facts", "../../cmd", 5); err == nil {
		t.Fatal("path escape accepted")
	}
}
