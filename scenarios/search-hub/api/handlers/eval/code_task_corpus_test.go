package eval

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"search-hub/internal/corpusgen"
)

// ownerFakes are mechanics-only fakes. They have no source-body API and cannot
// produce a real-source qualification claim.
type ownerFakes struct {
	refuse                   string
	privacyCalls, labelCalls int
}

func (o *ownerFakes) VerifyCodeTaskSource(context.Context, corpusgen.CodeTaskSourceIdentityBinding) error {
	return nil
}
func (o *ownerFakes) VerifyCodeTaskActorIdentity(context.Context, corpusgen.CodeTaskSourceIdentityBinding) error {
	return nil
}
func (o *ownerFakes) VerifyCodeTaskRoute(context.Context, corpusgen.GatewayRouteProof) error {
	return nil
}
func (o *ownerFakes) VerifyCodeTaskRights(context.Context, string, string) error { return nil }
func (o *ownerFakes) VerifyCodeTaskPrivacy(context.Context, string, corpusgen.CodeTaskCorpusConsentBinding) error {
	o.privacyCalls++
	if o.refuse == "privacy" {
		return context.Canceled
	}
	return nil
}
func (o *ownerFakes) VerifyCodeTaskRepository(context.Context, corpusgen.CodeTaskRepositoryBase) error {
	return nil
}
func (o *ownerFakes) VerifyCodeTaskObjectiveSuite(context.Context, corpusgen.CodeTaskRepositoryBase, corpusgen.CodeTaskObjectiveSuite) error {
	return nil
}
func (o *ownerFakes) VerifyCodeTaskIndependentLabel(context.Context, corpusgen.CodeTaskIndependentLabelRef, corpusgen.CodeTaskIndependentLabelBinding) error {
	o.labelCalls++
	if o.refuse == "label" {
		return context.Canceled
	}
	return nil
}

func testOwnerClients(o *ownerFakes) corpusgen.CodeTaskEvidenceOwnerClients {
	return corpusgen.CodeTaskEvidenceOwnerClients{
		Source: o, SourceIdentity: o, Route: o, Rights: o, Privacy: o,
		Repository: o, Objective: o, IndependentLabel: o,
	}
}

func testCodeTaskCandidate(i int) corpusgen.CodeTaskCandidate {
	h := strings.Repeat("a", 64)
	rev := strings.Repeat("b", 40)
	return corpusgen.CodeTaskCandidate{
		Source:         corpusgen.CodeTaskSourceRef{RunID: "00000000-0000-4000-8000-000000000001", EventID: "00000000-0000-4000-8000-000000000002", Sequence: 1, ConversationID: "00000000-0000-4000-8000-000000000003", RunStatus: "complete", MessageRole: "user", MessageSource: "operator", ActorID: "operator", ActorSubjectSHA256: h, ActorSubjectDigestVersion: "agent-manager-source-author-hmac/v1", ActorIdentityReceiptID: "identity-receipt", ActorIdentityReceiptSHA256: h, MessageSHA256: h},
		Route:          corpusgen.GatewayRouteProof{SourceRunID: "00000000-0000-4000-8000-000000000001", SourceEventID: "00000000-0000-4000-8000-000000000002", SourceSequence: 1, GatewayEventID: "gateway-event", GatewayRequestID: "gateway-request", GatewayRoleEvidenceURI: "gateway-ledger://events/gateway-event", Status: "succeeded", SelectedProvider: "ollama", SelectedModel: "model", SelectedResourceRole: corpusgen.CodeTaskCorpusRole, Locality: "local", CapacityVerdict: "fit", CapacityClaimID: "claim", CapacityRequestedBytes: 10, CapacityRequiredBytes: 10, CapacityGrantedBytes: 20, CapacityEstimateEvidenceURI: "capacity-footprint://claim", TaskRole: "code", Consumer: "opencode"},
		Repository:     corpusgen.CodeTaskRepositoryBase{RepositoryID: "vrooli", Revision: rev, TreeSHA256: h},
		ObjectiveSuite: corpusgen.CodeTaskObjectiveSuite{SuiteID: "suite", Revision: rev, SHA256: h, ValidationReceiptID: "validation", ValidationReceiptSHA256: h},
		TaskSpecSHA256: h, ExpectedBehaviorSHA256: h, RightsDecisionID: "rights", PrivacyDecisionID: "privacy",
		IndependentLabel: corpusgen.CodeTaskIndependentLabelRef{ReviewerID: "reviewer", ReviewerSubjectSHA256: strings.Repeat("c", 64), EvidenceURI: "owner://label", ReceiptSHA256: h, ExpectedBehaviorSHA256: h}, Split: "heldout", Strata: []string{"repo-grounded-citation"},
	}
}

func TestOwnerBoundBuilderRequiresIdentityConsentAndLabelClientsBeforeRequestProjection(t *testing.T) {
	for _, missing := range []string{"identity", "consent", "label"} {
		t.Run(missing, func(t *testing.T) {
			owners := &ownerFakes{}
			clients := testOwnerClients(owners)
			switch missing {
			case "identity":
				clients.SourceIdentity = nil
			case "consent":
				clients.Privacy = nil
			case "label":
				clients.IndependentLabel = nil
			}
			_, err := NewOwnerBoundCodeTaskCorpusBuilder(clients)
			require.Error(t, err)
			// Production handler has no builder until this constructor succeeds.
			h := NewConnectHandler(Deps{})
			r := httptest.NewRequest("POST", "/eval/code-task-corpus/build", strings.NewReader("not-read"))
			w := httptest.NewRecorder()
			h.BuildCodeTaskCorpus(w, r)
			require.Equal(t, 503, w.Code)
			require.Zero(t, owners.privacyCalls)
			require.Zero(t, owners.labelCalls)
		})
	}
}

func TestCodeTaskCallerRefusesConsentOrLabelBeforeProjection(t *testing.T) {
	for _, refusal := range []string{"privacy", "label"} {
		t.Run(refusal, func(t *testing.T) {
			owners := &ownerFakes{refuse: refusal}
			builder, err := NewOwnerBoundCodeTaskCorpusBuilder(testOwnerClients(owners))
			require.NoError(t, err)
			h := NewConnectHandler(Deps{CodeTaskCorpus: builder})
			body, err := json.Marshal(codeTaskCorpusBuildRequest{SplitVersion: "split-v1", Candidates: []corpusgen.CodeTaskCandidate{testCodeTaskCandidate(1)}})
			require.NoError(t, err)
			r := httptest.NewRequest("POST", "/eval/code-task-corpus/build", bytes.NewReader(body))
			w := httptest.NewRecorder()
			h.BuildCodeTaskCorpus(w, r)
			require.Equal(t, 422, w.Code)
			var response codeTaskCorpusBuildResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
			require.False(t, response.Snapshot.Sealed)
			require.Empty(t, response.Cases, "a refused case must never be projected")
			require.Empty(t, response.Snapshot.Cases)
			require.Len(t, response.Snapshot.Rejections, 1)
			if refusal == "privacy" {
				require.Equal(t, "privacy_owner_refused", response.Snapshot.Rejections[0].ReasonCode)
				require.Zero(t, owners.labelCalls)
			} else {
				require.Equal(t, "independent_label_owner_refused", response.Snapshot.Rejections[0].ReasonCode)
				require.Equal(t, 1, owners.labelCalls)
			}
		})
	}
}

func TestCodeTaskCallerRejectsUnknownBodyFields(t *testing.T) {
	owners := &ownerFakes{}
	builder, err := NewOwnerBoundCodeTaskCorpusBuilder(testOwnerClients(owners))
	require.NoError(t, err)
	h := NewConnectHandler(Deps{CodeTaskCorpus: builder})
	r := httptest.NewRequest("POST", "/eval/code-task-corpus/build", strings.NewReader(`{"split_version":"v1","task_body":"must not enter corpus"}`))
	w := httptest.NewRecorder()
	h.BuildCodeTaskCorpus(w, r)
	require.Equal(t, 400, w.Code)
	require.Zero(t, owners.privacyCalls)
	require.Zero(t, owners.labelCalls)
}

func TestCodeTaskCallerDoesNotDisclosePartialAcceptedLineage(t *testing.T) {
	owners := &ownerFakes{}
	builder, err := NewOwnerBoundCodeTaskCorpusBuilder(testOwnerClients(owners))
	require.NoError(t, err)
	h := NewConnectHandler(Deps{CodeTaskCorpus: builder})
	body, err := json.Marshal(codeTaskCorpusBuildRequest{SplitVersion: "split-v1", Candidates: []corpusgen.CodeTaskCandidate{testCodeTaskCandidate(1)}})
	require.NoError(t, err)
	r := httptest.NewRequest("POST", "/eval/code-task-corpus/build", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.BuildCodeTaskCorpus(w, r)
	require.Equal(t, 422, w.Code, "one case is below the frozen denominator minimum")
	var response codeTaskCorpusBuildResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.False(t, response.Snapshot.Sealed)
	require.Empty(t, response.Snapshot.Cases)
	require.Empty(t, response.Snapshot.CaseIDs)
	require.Empty(t, response.Cases)
	require.Empty(t, response.Snapshot.CorpusSHA256)
}
