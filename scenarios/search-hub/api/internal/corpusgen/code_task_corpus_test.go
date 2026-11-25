package corpusgen

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// These fixtures exercise metadata admission mechanics only. They are not
// evaluation cases and make no claim about coding-model quality.
type acceptingCodeTaskOwners struct{}

func (acceptingCodeTaskOwners) VerifyCodeTaskSource(context.Context, CodeTaskSourceRef) error {
	return nil
}
func (acceptingCodeTaskOwners) VerifyCodeTaskRoute(context.Context, GatewayRouteProof) error {
	return nil
}
func (acceptingCodeTaskOwners) VerifyCodeTaskRights(context.Context, string, string) error {
	return nil
}
func (acceptingCodeTaskOwners) VerifyCodeTaskPrivacy(context.Context, string, string) error {
	return nil
}
func (acceptingCodeTaskOwners) VerifyCodeTaskRepository(context.Context, CodeTaskRepositoryBase) error {
	return nil
}
func (acceptingCodeTaskOwners) VerifyCodeTaskObjectiveSuite(context.Context, CodeTaskRepositoryBase, CodeTaskObjectiveSuite) error {
	return nil
}
func (acceptingCodeTaskOwners) VerifyCodeTaskIndependentLabel(context.Context, CodeTaskIndependentLabelRef, string) error {
	return nil
}

func TestBuildCodeTaskCorpusRetainsRejectionsAndDoesNotSealBelowCoverage(t *testing.T) {
	candidate := codeTaskCandidateFixture(1, "single-file-change")
	candidate.Source.MessageSHA256 = "short"
	missingSuiteReceipt := codeTaskCandidateFixture(2, "single-file-change")
	missingSuiteReceipt.ObjectiveSuite.ValidationReceiptSHA256 = ""
	snapshot := BuildCodeTaskCorpus(context.Background(), []CodeTaskCandidate{candidate, missingSuiteReceipt}, "split-v1", acceptingCodeTaskOwners{})
	if snapshot.Sealed || len(snapshot.Cases) != 0 || len(snapshot.Rejections) != 2 || snapshot.Rejections[0].ReasonCode != "source_identity_incomplete" || snapshot.Rejections[1].ReasonCode != "objective_test_suite_identity_incomplete" {
		t.Fatalf("expected rejected source and unsealed corpus, got %+v", snapshot)
	}
	if len(snapshot.CoverageDeficits) == 0 {
		t.Fatal("unsealed corpus must report deficits")
	}
}

func TestBuildCodeTaskCorpusSealsOnlyCompleteMetadataDenominator(t *testing.T) {
	var candidates []CodeTaskCandidate
	index := 0
	for stratum, minimum := range codeTaskRequiredStrata {
		for n := 0; n < minimum; n++ {
			index++
			candidates = append(candidates, codeTaskCandidateFixture(index, stratum))
		}
	}
	if !strings.HasPrefix(codeTaskCaseID(candidates[0]), "lm22-") {
		t.Fatalf("Search Hub case identity %q is outside the Workspace Sandbox LM22 owner namespace", codeTaskCaseID(candidates[0]))
	}
	snapshot := BuildCodeTaskCorpus(context.Background(), candidates, "split-v1", acceptingCodeTaskOwners{})
	if !snapshot.Sealed || len(snapshot.Cases) < CodeTaskCorpusMinimumCases || len(snapshot.CaseIDs) != len(snapshot.Cases) {
		t.Fatalf("complete owner-verified metadata denominator did not seal: cases=%d deficits=%v", len(snapshot.Cases), snapshot.CoverageDeficits)
	}
	if snapshot.DenominatorSHA256 == "" || snapshot.CorpusSHA256 == "" {
		t.Fatal("sealed denominator is missing corpus or case-id digest")
	}
	projected, err := snapshot.EvaluationCases()
	if err != nil || len(projected) != len(snapshot.Cases) {
		t.Fatalf("sealed corpus did not project into the Gateway handoff: cases=%d err=%v", len(projected), err)
	}
	raw, err := json.Marshal(projected[0])
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"source_run_id", "source_message_role", "gateway_request_id", "selected_resource_role", "capacity_claim_id", "repository_revision", "objective_suite_sha256", "objective_suite_receipt_id", "objective_suite_receipt_sha256", "independent_label_receipt_sha256", "independent_reviewer_id", "privacy_decision_id"} {
		if fields[field] == nil {
			t.Fatalf("Gateway case projection omitted %q: %s", field, raw)
		}
	}
	snapshot.CaseIDs[0] = "tampered"
	if _, err := snapshot.EvaluationCases(); err == nil {
		t.Fatal("expected sealed denominator mutation to refuse evaluation projection")
	}
}

func codeTaskCandidateFixture(index int, stratum string) CodeTaskCandidate {
	uid := fmt.Sprintf("%08x-1111-4111-8111-%012x", index, index)
	sha := strings.Repeat("a", 64)
	return CodeTaskCandidate{
		Source:         CodeTaskSourceRef{RunID: uid, EventID: fmt.Sprintf("%08x-2222-4222-8222-%012x", index, index), Sequence: uint64(index), ConversationID: fmt.Sprintf("%08x-3333-4333-8333-%012x", index, index), RunStatus: "complete", MessageRole: "user", MessageSource: "operator", ActorID: "operator", MessageSHA256: sha},
		Route:          GatewayRouteProof{SourceRunID: uid, SourceEventID: fmt.Sprintf("%08x-2222-4222-8222-%012x", index, index), SourceSequence: uint64(index), GatewayEventID: uid, GatewayRequestID: uid, GatewayRoleEvidenceURI: "gateway-ledger://events/" + uid, TaskRole: "coding", Consumer: "opencode", Status: "succeeded", SelectedProvider: "local", SelectedModel: "fixture", SelectedResourceRole: CodeTaskCorpusRole, Locality: "local", CapacityClaimID: uid, CapacityRequestedBytes: 1, CapacityRequiredBytes: 1, CapacityGrantedBytes: 1, CapacityVerdict: "fit", CapacityEstimateEvidenceURI: "capacity-footprint://fixture"},
		Repository:     CodeTaskRepositoryBase{RepositoryID: "fixture", Revision: strings.Repeat("b", 40), TreeSHA256: sha},
		ObjectiveSuite: CodeTaskObjectiveSuite{SuiteID: "fixture-tests", Revision: strings.Repeat("b", 40), SHA256: sha, ValidationReceiptID: "fixture-test-receipt", ValidationReceiptSHA256: sha},
		TaskSpecSHA256: sha, ExpectedBehaviorSHA256: sha, RightsDecisionID: "rights-fixture", PrivacyDecisionID: "privacy-fixture",
		IndependentLabel: CodeTaskIndependentLabelRef{ReviewerID: "reviewer", EvidenceURI: "review://fixture", ReceiptSHA256: sha, ExpectedBehaviorSHA256: sha}, Split: "heldout", Strata: []string{stratum},
	}
}
