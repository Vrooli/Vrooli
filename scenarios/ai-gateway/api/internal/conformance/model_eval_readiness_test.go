package conformance

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	aisearch "github.com/vrooli/ai-go/search"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	"github.com/vrooli/vrooli/packages/proto/searcheval"
)

func TestAcceptedCallerCaseAllowsAgentManagerSequenceZero(t *testing.T) {
	requestID, err := searcheval.AgentManagerReplayRequestID("run-zero", "event-zero", 0, "chat.default", "vrooli-memory")
	if err != nil {
		t.Fatal(err)
	}
	note := "source=agent-manager://runs/run-zero/events/event-zero seq=0 gateway_event=route-zero gateway_request=" + requestID + " selected_resource_role=chat.default role_evidence=gateway-ledger://events/route-zero model=local-model consumer=vrooli-memory capacity_estimate=capacity-footprint://ollama/dynamic task_role=chat.default label_review=audit://labels/zero reviewer=independent"
	caseRecord := &evalv1.EvalCase{
		CaseId: evalCallerCaseID(note, "chat.default"), Query: "a reviewed query", Status: "reviewed",
		ExpectIds: []string{"src/file.go"}, Note: note,
		Tags: []string{"caller-audit", "agent-manager", "independently-reviewed", "resource-role:chat.default", "consumer:vrooli-memory", "task-role:chat.default"},
	}
	if !acceptedCallerCase(caseRecord) {
		t.Fatalf("sequence-zero reviewed caller case was rejected: %s", note)
	}
	caseRecord.Note = strings.Replace(note, "gateway_request="+requestID, "gateway_request=unbound", 1)
	if acceptedCallerCase(caseRecord) {
		t.Fatal("source-linked case with an unbound Gateway request ID was accepted")
	}
}

type modelEvalClientStub struct {
	suites          []*evalv1.EvalSuite
	denominator     map[string]*evalv1.GetFixedDenominatorResponse
	validation      map[string]*evalv1.GetLatestCorpusValidationResponse
	listErr         error
	getCalls        int
	validationCalls int
}

func (s *modelEvalClientStub) ListSuites(context.Context, *connect.Request[evalv1.ListSuitesRequest]) (*connect.Response[evalv1.ListSuitesResponse], error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return connect.NewResponse(&evalv1.ListSuitesResponse{Suites: s.suites}), nil
}

func (s *modelEvalClientStub) GetFixedDenominator(_ context.Context, req *connect.Request[evalv1.GetFixedDenominatorRequest]) (*connect.Response[evalv1.GetFixedDenominatorResponse], error) {
	s.getCalls++
	response, ok := s.denominator[req.Msg.GetSuiteId()]
	if !ok {
		return connect.NewResponse(&evalv1.GetFixedDenominatorResponse{}), nil
	}
	return connect.NewResponse(response), nil
}

func (s *modelEvalClientStub) GetLatestCorpusValidation(_ context.Context, req *connect.Request[evalv1.GetLatestCorpusValidationRequest]) (*connect.Response[evalv1.GetLatestCorpusValidationResponse], error) {
	s.validationCalls++
	response, ok := s.validation[req.Msg.GetSuiteId()]
	if !ok {
		return connect.NewResponse(&evalv1.GetLatestCorpusValidationResponse{}), nil
	}
	return connect.NewResponse(response), nil
}

func TestModelEvalReadinessListsExactMissingDenominatorAndRoleFloor(t *testing.T) {
	client := &modelEvalClientStub{
		suites:      []*evalv1.EvalSuite{{SuiteId: "knowledge-observatory.docs.starter", ProviderId: "knowledge-observatory.docs"}},
		denominator: map[string]*evalv1.GetFixedDenominatorResponse{},
	}
	got, err := newSearchHubModelEvalSource(client).Readiness(context.Background(), ModelEvalRoleDenominators{
		PolicyRoles: []string{"chat.default", "vision.default"}, TaskRoles: []string{"classify.fast", "summarize.faithful"}, Consumers: []string{"client-rehearsal", "vrooli-memory"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Ready || got.RegisteredSuites != 1 || got.RequiredRetrievalSuites != modelEvalMinimumRetrievalSuites {
		t.Fatalf("unexpected readiness: %+v", got)
	}
	if client.getCalls != 1 {
		t.Fatalf("fixed denominator API calls = %d, want 1", client.getCalls)
	}
	if len(got.MissingDenominatorIDs) != 1 || got.MissingDenominatorIDs[0] != "knowledge-observatory.docs.starter" || got.SuiteInventoryDeficit != 34 {
		t.Fatalf("missing denominator evidence = %v", got.MissingDenominatorIDs)
	}
	if got.FixedDenominatorSetSHA256 != "" {
		t.Fatalf("incomplete denominator set received digest %q", got.FixedDenominatorSetSHA256)
	}
	if len(got.RoleCoverage) != 2 || !got.RoleCoverage[0].Deficient || !got.RoleCoverage[1].Deficient {
		t.Fatalf("role coverage did not block both policy roles: %+v", got.RoleCoverage)
	}
	if len(got.TaskRoleCoverage) != 2 || !got.TaskRoleCoverage[0].Deficient || !got.TaskRoleCoverage[1].Deficient {
		t.Fatalf("task-role coverage did not block both AI Gateway task roles: %+v", got.TaskRoleCoverage)
	}
	if len(got.ConsumerCoverage) != 2 || !got.ConsumerCoverage[0].Deficient || !got.ConsumerCoverage[1].Deficient {
		t.Fatalf("consumer coverage did not block both observed consumers: %+v", got.ConsumerCoverage)
	}
}

func TestModelEvalFixedDenominatorSetDigestIsOrderIndependent(t *testing.T) {
	left := searcheval.CaseIDsDigest([]string{"b:corpus:denom", "a:corpus:denom"})
	right := searcheval.CaseIDsDigest([]string{"a:corpus:denom", "b:corpus:denom"})
	if left == "" || left != right {
		t.Fatalf("fixed denominator set digest is not deterministic: %q != %q", left, right)
	}
}

func TestModelEvalReadinessPropagatesSearchHubInventoryFailure(t *testing.T) {
	want := errors.New("search hub unavailable")
	client := &modelEvalClientStub{listErr: want}
	_, err := newSearchHubModelEvalSource(client).Readiness(context.Background(), ModelEvalRoleDenominators{PolicyRoles: []string{"chat.default"}})
	if err == nil || !errors.Is(err, want) {
		t.Fatalf("Readiness error = %v, want %v", err, want)
	}
	if client.getCalls != 0 {
		t.Fatalf("denominator reads after inventory failure = %d, want 0", client.getCalls)
	}
	if client.validationCalls != 0 {
		t.Fatalf("corpus validation reads after inventory failure = %d, want 0", client.validationCalls)
	}
}

func TestModelEvalCorpusValidationKeepsStaleExpectedDocumentsInDenominator(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	caseRecord := &evalv1.EvalCase{CaseId: "case-1", Query: "query", Status: "reviewed", ExpectIds: []string{"deleted-doc"}}
	suite := &evalv1.EvalSuite{SuiteId: "suite", ProviderId: "provider", Cases: []*evalv1.EvalCase{caseRecord}}
	sealed, err := aisearch.NewFixedDenominator("suite", []aisearch.TestCase{{ID: "case-1", Query: "query", Status: "reviewed", ExpectIDs: []string{"deleted-doc"}}}, "independent-reviewer", "audit://suite")
	if err != nil {
		t.Fatal(err)
	}
	snapshot := &evalv1.FixedDenominatorSnapshot{
		SuiteId: "suite", ProviderId: "provider", CorpusSha256: searcheval.SuiteCorpusDigest(suite),
		ReviewerId: "independent-reviewer", ReviewEvidenceUri: "audit://suite", ReviewedAt: now.Format(time.RFC3339Nano),
		CaseIds: []string{"case-1"}, Cases: []*evalv1.EvalCase{caseRecord}, DenominatorSha256: sealed.SHA256,
	}
	validation := &evalv1.GetLatestCorpusValidationResponse{
		Found: true, SuiteId: "suite", ProviderId: "provider", SuiteSha256: snapshot.GetCorpusSha256(),
		DenominatorSha256: strings.Repeat("a", 64), DenominatorCaseIds: []string{"case-1"},
		ReceiptSha256: strings.Repeat("b", 64), CreatedAt: now.Format(time.RFC3339Nano),
		Rollup: &evalv1.CorpusValidationRollup{Positives: 1, Stale: 1},
	}
	if reason := validateModelEvalSnapshot(suite, snapshot); reason != "" {
		t.Fatalf("valid sealed snapshot rejected: %s", reason)
	}
	if reason := validateModelEvalCorpusValidation(suite, snapshot, validation, now); reason != "" {
		t.Fatalf("stale expected document removed or rejected from fixed denominator: %s", reason)
	}
}
