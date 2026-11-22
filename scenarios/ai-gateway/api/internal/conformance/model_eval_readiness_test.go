package conformance

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	aisearch "github.com/vrooli/ai-go/search"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	"github.com/vrooli/vrooli/packages/proto/searcheval"
	"google.golang.org/protobuf/proto"
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
	suites      []*evalv1.EvalSuite
	denominator map[string]*evalv1.GetFixedDenominatorResponse
	validation  map[string]*evalv1.GetLatestCorpusValidationResponse
	listErr     error
	// The source reads suites concurrently, so the call counters are atomic.
	getCalls        atomic.Int32
	validationCalls atomic.Int32
}

func (s *modelEvalClientStub) ListSuites(context.Context, *connect.Request[evalv1.ListSuitesRequest]) (*connect.Response[evalv1.ListSuitesResponse], error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return connect.NewResponse(&evalv1.ListSuitesResponse{Suites: s.suites}), nil
}

func (s *modelEvalClientStub) GetFixedDenominator(_ context.Context, req *connect.Request[evalv1.GetFixedDenominatorRequest]) (*connect.Response[evalv1.GetFixedDenominatorResponse], error) {
	s.getCalls.Add(1)
	response, ok := s.denominator[req.Msg.GetSuiteId()]
	if !ok {
		return connect.NewResponse(&evalv1.GetFixedDenominatorResponse{}), nil
	}
	return connect.NewResponse(response), nil
}

func (s *modelEvalClientStub) GetLatestCorpusValidation(_ context.Context, req *connect.Request[evalv1.GetLatestCorpusValidationRequest]) (*connect.Response[evalv1.GetLatestCorpusValidationResponse], error) {
	s.validationCalls.Add(1)
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
	if client.getCalls.Load() != 1 {
		t.Fatalf("fixed denominator API calls = %d, want 1", client.getCalls.Load())
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
	if client.getCalls.Load() != 0 {
		t.Fatalf("denominator reads after inventory failure = %d, want 0", client.getCalls.Load())
	}
	if client.validationCalls.Load() != 0 {
		t.Fatalf("corpus validation reads after inventory failure = %d, want 0", client.validationCalls.Load())
	}
}

func TestModelEvalReadinessReportsSuiteInventoryBoundaries(t *testing.T) {
	tests := []struct {
		name        string
		suites      []*evalv1.EvalSuite
		wantInvalid []string
		wantMissing []string
		wantDeficit int
	}{
		{name: "nil and empty suite identities", suites: []*evalv1.EvalSuite{nil, {}, {SuiteId: " "}}, wantInvalid: []string{"suite-inventory-has-empty-id", "suite-inventory-has-empty-id", "suite-inventory-has-empty-id"}, wantDeficit: modelEvalMinimumRetrievalSuites - 3},
		{name: "duplicate trimmed identity", suites: []*evalv1.EvalSuite{{SuiteId: "suite-a"}, {SuiteId: " suite-a "}}, wantInvalid: []string{"suite-a:duplicate-suite-id"}, wantMissing: []string{" suite-a ", "suite-a"}, wantDeficit: modelEvalMinimumRetrievalSuites - 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &modelEvalClientStub{suites: tt.suites, denominator: map[string]*evalv1.GetFixedDenominatorResponse{}}
			got, err := newSearchHubModelEvalSource(client).Readiness(context.Background(), ModelEvalRoleDenominators{})
			if err != nil {
				t.Fatal(err)
			}
			if got.Ready || got.SuiteInventoryDeficit != tt.wantDeficit {
				t.Fatalf("readiness=%+v, want blocked with deficit %d", got, tt.wantDeficit)
			}
			if !equalStrings(got.InvalidDenominatorIDs, tt.wantInvalid) || !equalStrings(got.MissingDenominatorIDs, tt.wantMissing) {
				t.Fatalf("invalid IDs=%v / missing IDs=%v, want %v / %v", got.InvalidDenominatorIDs, got.MissingDenominatorIDs, tt.wantInvalid, tt.wantMissing)
			}
		})
	}
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
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
	tests := []struct {
		name   string
		mutate func(*evalv1.GetLatestCorpusValidationResponse)
		want   string
	}{
		{name: "missing receipt", mutate: func(v *evalv1.GetLatestCorpusValidationResponse) { v.Found = false }, want: "receipt-missing"},
		{name: "suite identity", mutate: func(v *evalv1.GetLatestCorpusValidationResponse) { v.SuiteId = "other" }, want: "suite-identity-mismatch"},
		{name: "suite digest", mutate: func(v *evalv1.GetLatestCorpusValidationResponse) { v.SuiteSha256 = strings.Repeat("c", 64) }, want: "suite-digest-mismatch"},
		{name: "denominator cases", mutate: func(v *evalv1.GetLatestCorpusValidationResponse) { v.DenominatorCaseIds = []string{"other"} }, want: "denominator-case-set-mismatch"},
		{name: "receipt digest", mutate: func(v *evalv1.GetLatestCorpusValidationResponse) { v.ReceiptSha256 = "bad" }, want: "receipt-digest-invalid"},
		{name: "rollup denominator", mutate: func(v *evalv1.GetLatestCorpusValidationResponse) {
			v.Rollup = &evalv1.CorpusValidationRollup{Positives: 1}
		}, want: "rollup-denominator-mismatch"},
		{name: "invalid timestamp", mutate: func(v *evalv1.GetLatestCorpusValidationResponse) { v.CreatedAt = "invalid" }, want: "validation-timestamp-invalid"},
		{name: "stale timestamp", mutate: func(v *evalv1.GetLatestCorpusValidationResponse) {
			v.CreatedAt = now.Add(-modelEvalCorpusValidationMaxAge - time.Second).Format(time.RFC3339Nano)
		}, want: "validation-receipt-stale"},
		{name: "future timestamp", mutate: func(v *evalv1.GetLatestCorpusValidationResponse) {
			v.CreatedAt = now.Add(time.Second).Format(time.RFC3339Nano)
		}, want: "validation-receipt-stale"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidate := proto.Clone(validation).(*evalv1.GetLatestCorpusValidationResponse)
			tt.mutate(candidate)
			if got := validateModelEvalCorpusValidation(suite, snapshot, candidate, now); got != tt.want {
				t.Fatalf("reason = %q, want %q", got, tt.want)
			}
		})
	}
}
