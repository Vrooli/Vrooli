package operatorinputs

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/nodereach"
	"github.com/vrooli/vrooli/internal/operatorcapability"
	operatorinputsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/operatorinputs"
	operatorinputsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/operatorinputs/operatorinputsv1connect"
	domainoperatorinputs "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/operatorinputs"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/targetproxy"
	"google.golang.org/protobuf/proto"
)

type remoteOperatorInputCall struct {
	t      *testing.T
	client operatorinputsconnect.OperatorInputsServiceClient
}

func (r remoteOperatorInputCall) CallScenario(ctx context.Context, request nodereach.ScenarioRequest) ([]byte, error) {
	r.t.Helper()
	if request.NodeID != "node-7" || request.Scenario != "vrooli-onboarding" || request.Procedure != operatorinputsconnect.OperatorInputsServiceResolveOperatorInputsProcedure {
		r.t.Fatalf("remote call = node %q scenario %q procedure %q", request.NodeID, request.Scenario, request.Procedure)
	}
	var forwarded operatorinputsv1.ResolveOperatorInputsRequest
	if err := proto.Unmarshal(request.Body, &forwarded); err != nil {
		r.t.Fatalf("decode forwarded request: %v", err)
	}
	if forwarded.GetTarget() != "local" {
		r.t.Fatalf("forwarded target = %q, want node-local owner target", forwarded.GetTarget())
	}
	response, err := r.client.ResolveOperatorInputs(ctx, connect.NewRequest(&forwarded))
	if err != nil {
		return nil, err
	}
	return proto.Marshal(response.Msg)
}

type committedReplyLoss struct {
	t           *testing.T
	seen        []operatorcapability.ActionRequest
	committed   map[string]operatorcapability.Result
	effectCount int
}

func (p *committedReplyLoss) ApplyCapability(_ context.Context, request operatorcapability.ActionRequest) (operatorcapability.Result, error) {
	p.t.Helper()
	captured := request
	captured.Inputs = make(map[string]json.RawMessage, len(request.Inputs))
	for key, value := range request.Inputs {
		captured.Inputs[key] = append(json.RawMessage(nil), value...)
	}
	p.seen = append(p.seen, captured)
	if request.TargetID != "local" {
		p.t.Fatalf("capability target = %q, want node-local target", request.TargetID)
	}
	if result, ok := p.committed[request.IdempotencyKey]; ok {
		return result, nil
	}
	p.effectCount++ // The provider commits before its response is lost.
	result := operatorcapability.Result{CapabilityID: request.CapabilityID, State: operatorcapability.StateReady, Outcome: "complete"}
	p.committed[request.IdempotencyKey] = result
	return result, connect.NewError(connect.CodeUnavailable, errors.New("injected provider reply loss after commit"))
}

func TestRemoteOperatorInputRetainsQuestionAndRetriesSameTargetActionAfterProviderLoss(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	requireOperatorInputQueue(t)
	provider := &committedReplyLoss{t: t, committed: make(map[string]operatorcapability.Result)}
	remoteService := domainoperatorinputs.Service{
		Applier:         provider,
		CurrentRevision: func(context.Context) (string, error) { return "node-revision-7", nil },
	}
	remotePath, remoteHandler := operatorinputsconnect.NewOperatorInputsServiceHandler(NewConnectHandler(remoteService))
	remoteMux := http.NewServeMux()
	remoteMux.Handle(remotePath, remoteHandler)
	remoteServer := httptest.NewServer(remoteMux)
	t.Cleanup(remoteServer.Close)
	remoteClient := operatorinputsconnect.NewOperatorInputsServiceClient(remoteServer.Client(), remoteServer.URL)

	reacher := remoteOperatorInputCall{t: t, client: remoteClient}
	localPath, localHandler := operatorinputsconnect.NewOperatorInputsServiceHandler(
		NewConnectHandler(domainoperatorinputs.Service{}),
		connect.WithInterceptors(targetproxy.Interceptor(reacher)),
	)
	localMux := http.NewServeMux()
	localMux.Handle(localPath, localHandler)
	localServer := httptest.NewServer(localMux)
	t.Cleanup(localServer.Close)
	client := operatorinputsconnect.NewOperatorInputsServiceClient(localServer.Client(), localServer.URL)

	queue := []operatorcapability.Request{{
		ID: "endpoint", Kind: operatorcapability.KindChoice, CapabilityID: "fixture/provider-endpoint",
		ActionID: "configure", InputID: "endpoint", Title: "Provider endpoint",
		Options: []string{"https://endpoint-a.example", "https://endpoint-b.example"}, Required: true,
	}}
	if err := operatorcapability.Replace(queue); err != nil {
		t.Fatalf("seed durable operator-input queue: %v", err)
	}
	request := &operatorinputsv1.ResolveOperatorInputsRequest{
		Target: "node-7", ExpectedRevision: "node-revision-7",
		Answers: []*operatorinputsv1.Answer{{RequestId: "endpoint", Value: "https://endpoint-b.example"}},
	}

	// An untargeted request is explicitly unsupported and cannot consume the
	// persisted question or dispatch a capability action.
	untargeted := proto.Clone(request).(*operatorinputsv1.ResolveOperatorInputsRequest)
	untargeted.Target = ""
	_, err := client.ResolveOperatorInputs(context.Background(), connect.NewRequest(untargeted))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("untargeted code = %v, want invalid argument; error=%v", connect.CodeOf(err), err)
	}
	pending, err := operatorcapability.Load()
	if err != nil || len(pending.Requests) != 1 || provider.effectCount != 0 {
		t.Fatalf("untargeted request changed queue/provider: pending=%+v effects=%d err=%v", pending, provider.effectCount, err)
	}

	_, err = client.ResolveOperatorInputs(context.Background(), connect.NewRequest(request))
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("first attempt code = %v, want unavailable; error=%v", connect.CodeOf(err), err)
	}
	if strings.Contains(err.Error(), "injected provider reply loss after commit") {
		t.Fatalf("provider detail leaked to operator: %v", err)
	}
	pending, err = operatorcapability.Load()
	if err != nil || len(pending.Requests) != 1 || pending.Requests[0].Decision != "" {
		t.Fatalf("provider loss did not retain the pending question: pending=%+v err=%v", pending, err)
	}
	if provider.effectCount != 1 || len(provider.seen) != 1 {
		t.Fatalf("first attempt provider effects=%d calls=%d, want one committed action", provider.effectCount, len(provider.seen))
	}
	first := provider.seen[0]
	wantKey := operatorcapability.StableIdempotencyKey("fixture/provider-endpoint", first.Inputs, "local")
	if first.TargetID != "local" || first.IdempotencyKey != wantKey {
		t.Fatalf("first provider action target/key = %q/%q, want local/%q", first.TargetID, first.IdempotencyKey, wantKey)
	}

	response, err := client.ResolveOperatorInputs(context.Background(), connect.NewRequest(proto.Clone(request).(*operatorinputsv1.ResolveOperatorInputsRequest)))
	if err != nil {
		t.Fatalf("retry after provider recovery: %v", err)
	}
	if len(response.Msg.GetOutcomes()) != 1 || response.Msg.GetOutcomes()[0].GetStatus() != operatorinputsv1.AnswerOutcomeStatus_ANSWER_OUTCOME_STATUS_APPLIED {
		t.Fatalf("recovery outcomes = %+v", response.Msg.GetOutcomes())
	}
	if provider.effectCount != 1 || len(provider.seen) != 2 {
		t.Fatalf("recovery replayed provider effect: effects=%d calls=%d", provider.effectCount, len(provider.seen))
	}
	if provider.seen[1].TargetID != first.TargetID || provider.seen[1].IdempotencyKey != first.IdempotencyKey {
		t.Fatalf("recovery changed target/key: first=%q/%q retry=%q/%q", first.TargetID, first.IdempotencyKey, provider.seen[1].TargetID, provider.seen[1].IdempotencyKey)
	}
	pending, err = operatorcapability.Load()
	if err != nil || len(pending.Requests) != 0 {
		t.Fatalf("successful recovery did not retire the answered question: pending=%+v err=%v", pending, err)
	}
}

func requireOperatorInputQueue(t *testing.T) {
	t.Helper()
	if err := operatorcapability.Replace(nil); err != nil {
		t.Fatalf("clear isolated operator-input queue: %v", err)
	}
}
