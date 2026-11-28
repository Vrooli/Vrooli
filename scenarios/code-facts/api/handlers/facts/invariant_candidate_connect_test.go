package facts

import (
	"context"
	"testing"

	internalfacts "code-facts/internal/facts"
	"connectrpc.com/connect"
	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

func TestInvariantCandidateConnectPositiveAndNegative(t *testing.T) {
	handler := NewConnectHandler(Deps{Service: internalfacts.NewService()})
	positive, err := handler.SearchInvariantCandidates(context.Background(), connect.NewRequest(&factsv1.SearchInvariantCandidatesRequest{Query: "cache max bytes less than zero", Scenario: "code-facts", Path: "api", Limit: 5}))
	if err != nil {
		t.Fatal(err)
	}
	if len(positive.Msg.GetResults()) == 0 {
		t.Fatalf("expected candidate; response=%+v", positive.Msg)
	}
	found := false
	for _, hit := range positive.Msg.GetResults() {
		if hit.GetPath() == "api/main.go" && hit.GetEvidenceKind() == "runtime-guard candidate" {
			found = true
			if hit.GetAttestation() != "CANDIDATE_ONLY" || hit.GetEnforcement().GetRuntime() != "UNKNOWN" || hit.GetEnforcement().GetTest() != "UNKNOWN" {
				t.Fatalf("candidate was overclaimed: %+v", hit)
			}
		}
	}
	if !found {
		t.Fatalf("expected runtime candidate from api/main.go: %+v", positive.Msg.GetResults())
	}
	negative, err := handler.SearchInvariantCandidates(context.Background(), connect.NewRequest(&factsv1.SearchInvariantCandidatesRequest{Query: "underwater penguin invoice telescope", Scenario: "code-facts", Path: "api", Limit: 5}))
	if err != nil {
		t.Fatal(err)
	}
	if len(negative.Msg.GetResults()) != 0 || len(negative.Msg.GetDegradations()) == 0 {
		t.Fatalf("negative query should abstain: %+v", negative.Msg)
	}
}
