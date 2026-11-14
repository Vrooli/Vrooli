package conformance

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
)

func TestAcceptedCallerCaseAllowsAgentManagerSequenceZero(t *testing.T) {
	note := "source=agent-manager://runs/run-zero/events/event-zero seq=0 gateway_event=route-zero gateway_request=request-zero role=chat.default role_evidence=gateway-ledger://events/route-zero model=local-model capacity_estimate=capacity-footprint://ollama/dynamic task_role=chat.default label_review=audit://labels/zero reviewer=independent"
	caseRecord := &evalv1.EvalCase{
		CaseId: evalCallerCaseID(note, "chat.default"), Query: "a reviewed query", Status: "reviewed",
		ExpectIds: []string{"src/file.go"}, Note: note,
		Tags: []string{"caller-audit", "agent-manager", "independently-reviewed", "role:chat.default", "task-role:chat.default"},
	}
	if !acceptedCallerCase(caseRecord) {
		t.Fatalf("sequence-zero reviewed caller case was rejected: %s", note)
	}
}

type modelEvalClientStub struct {
	suites      []*evalv1.EvalSuite
	denominator map[string]*evalv1.GetFixedDenominatorResponse
	listErr     error
	getCalls    int
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

func TestModelEvalReadinessListsExactMissingDenominatorAndRoleFloor(t *testing.T) {
	client := &modelEvalClientStub{
		suites:      []*evalv1.EvalSuite{{SuiteId: "knowledge-observatory.docs.starter", ProviderId: "knowledge-observatory.docs"}},
		denominator: map[string]*evalv1.GetFixedDenominatorResponse{},
	}
	got, err := newSearchHubModelEvalSource(client).Readiness(context.Background(), ModelEvalRoleDenominators{
		PolicyRoles: []string{"chat.default", "vision.default"}, TaskRoles: []string{"classify.fast", "summarize.faithful"},
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
}

func TestModelEvalFixedDenominatorSetDigestIsOrderIndependent(t *testing.T) {
	left := modelEvalFixedDenominatorSetDigest([]string{"b:corpus:denom", "a:corpus:denom"})
	right := modelEvalFixedDenominatorSetDigest([]string{"a:corpus:denom", "b:corpus:denom"})
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
}
