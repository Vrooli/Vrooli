package corpusgen

import (
	"strings"
	"testing"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
)

func TestAuditedCallerCaseKeepsUnreviewedCallerOutOfAcceptance(t *testing.T) {
	event := callerEventFixture()
	got, err := AuditedCallerCase(event, gatewayRouteFixture(), CallerLabelReview{})
	if err != nil {
		t.Fatal(err)
	}
	if got.GetStatus() != "candidate" || len(got.GetExpectIds()) != 0 || got.GetQuery() != event.Query {
		t.Fatalf("unreviewed caller case entered acceptance: %+v", got)
	}
}

func TestAuditedCallerCaseBindsValidatedLabelAndRoleEvidence(t *testing.T) {
	event := callerEventFixture()
	review := CallerLabelReview{Reviewer: "reviewer", EvidenceURI: "audit://repo/retry-handler", ExpectedIDs: []string{"scenarios/retry/handler.go"}, IndependentlyValidated: true}
	got, err := AuditedCallerCase(event, gatewayRouteFixture(), review)
	if err != nil {
		t.Fatal(err)
	}
	if got.GetStatus() != "reviewed" || len(got.GetExpectIds()) != 1 || got.GetTags()[2] != "role:chat.default" {
		t.Fatalf("reviewed case lost provenance: %+v", got)
	}
	if got.GetTags()[3] != "task-role:classify.fast" || !strings.Contains(got.GetNote(), "task_role=classify.fast") {
		t.Fatalf("reviewed case lost distinct task-role provenance: %+v", got)
	}
}

func TestAuditedCallerCaseRequiresAuthenticatedTypedEventAndRoleSource(t *testing.T) {
	valid := callerEventFixture()
	route := gatewayRouteFixture()
	route.Role = "unknown.role"
	if _, err := AuditedCallerCase(valid, route, CallerLabelReview{}); err == nil {
		t.Fatal("expected unsupported model role to fail")
	}
	invalid := valid
	invalid.ContentClass = "tool-call"
	if _, err := AuditedCallerCase(invalid, gatewayRouteFixture(), CallerLabelReview{}); err == nil {
		t.Fatal("expected wrong caller event type to fail")
	}
	invalid = valid
	invalid.EvidenceRef = "agent-manager://runs/other/events/event-1"
	if _, err := AuditedCallerCase(invalid, gatewayRouteFixture(), CallerLabelReview{}); err == nil {
		t.Fatal("expected mismatched event evidence ref to fail")
	}
	invalid = valid
	invalid.SourceVerified = false
	if _, err := AuditedCallerCase(invalid, gatewayRouteFixture(), CallerLabelReview{}); err == nil {
		t.Fatal("expected unverified event source to fail")
	}
	invalid = valid
	invalid.Sequence++
	if _, err := AuditedCallerCase(invalid, gatewayRouteFixture(), CallerLabelReview{}); err == nil {
		t.Fatal("expected mismatched Gateway source sequence to fail")
	}
	for name, mutate := range map[string]func(*GatewayRouteProof){
		"failed route":              func(r *GatewayRouteProof) { r.Status = "failed" },
		"remote route":              func(r *GatewayRouteProof) { r.Locality = "remote" },
		"missing gateway task role": func(r *GatewayRouteProof) { r.TaskRole = "" },
		"no model identity":         func(r *GatewayRouteProof) { r.SelectedModel = "" },
		"no gateway receipt":        func(r *GatewayRouteProof) { r.GatewayRequestID = "" },
		"no capacity claim":         func(r *GatewayRouteProof) { r.CapacityClaimID = "" },
		"capacity undergrant":       func(r *GatewayRouteProof) { r.CapacityGrantedBytes = 1 },
		"no capacity estimate":      func(r *GatewayRouteProof) { r.CapacityEstimateEvidenceURI = "" },
	} {
		t.Run(name, func(t *testing.T) {
			route := gatewayRouteFixture()
			mutate(&route)
			if _, err := AuditedCallerCase(valid, route, CallerLabelReview{}); err == nil {
				t.Fatal("invalid Gateway route proof entered corpus")
			}
		})
	}
}

func TestGatewayTaskRoleCoverageIsSeparateFromResourceRoleCoverage(t *testing.T) {
	review := CallerLabelReview{Reviewer: "reviewer", EvidenceURI: "audit://expected-docs", ExpectedIDs: []string{"src/retry.go"}, IndependentlyValidated: true}
	caseRecord, err := AuditedCallerCase(callerEventFixture(), gatewayRouteFixture(), review)
	if err != nil {
		t.Fatal(err)
	}
	counts, err := ValidateGatewayTaskRoleCoverage([]*evalv1.EvalCase{caseRecord}, []string{"classify.fast", "summarize.faithful"}, 1)
	if err == nil || counts["classify.fast"] != 1 || counts["summarize.faithful"] != 0 {
		t.Fatalf("task role counts = %v, error=%v; want one classify.fast case and an exact missing summary role", counts, err)
	}
	caseRecord.Note = strings.Replace(caseRecord.GetNote(), " task_role=classify.fast", "", 1)
	counts, err = ValidateGatewayTaskRoleCoverage([]*evalv1.EvalCase{caseRecord}, []string{"classify.fast"}, 1)
	if err == nil || counts["classify.fast"] != 0 {
		t.Fatalf("task-role case without matching route provenance counted: %v, %v", counts, err)
	}
}

func callerEventFixture() CallerEvent {
	return CallerEvent{RunID: "run-1", EventID: "event-1", Sequence: 42, Harness: "claude-code", ContentClass: "user-message", EvidenceRef: "agent-manager://runs/run-1/events/event-1", Query: "where is retry handled", SourceVerified: true}
}

func gatewayRouteFixture() GatewayRouteProof {
	return GatewayRouteProof{SourceRunID: "run-1", SourceEventID: "event-1", SourceSequence: 42, GatewayEventID: "route-1", GatewayRequestID: "request-1", GatewayRoleEvidenceURI: "gateway-ledger://events/route-1", TaskRole: "classify.fast", Role: "chat.default", Status: "succeeded", SelectedProvider: "ollama", SelectedModel: "qwen3.5:4b", Locality: "local", CapacityClaimID: "claim-1", CapacityRequestedBytes: 1024, CapacityRequiredBytes: 1024, CapacityGrantedBytes: 1024, CapacityVerdict: "fit", CapacityEstimateEvidenceURI: "capacity-footprint://ollama/dynamic"}
}
