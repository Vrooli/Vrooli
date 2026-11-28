package scenario

import (
	"context"
	"strings"
	"testing"

	"vrooli-bridge/internal/registry"
	registrymocks "vrooli-bridge/internal/registry/mocks"
	internal "vrooli-bridge/internal/scenario"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/scopecatalog"
)

type scenarioTestPresence struct{}

func (scenarioTestPresence) IsOnline(string) bool     { return true }
func (scenarioTestPresence) Dispatchable(string) bool { return true }

type scenarioTestPusher struct {
	broker   *internal.Broker
	requests []internal.Request
}

func (p *scenarioTestPusher) Push(_ context.Context, nodeID string, request internal.Request) (int, error) {
	p.requests = append(p.requests, request)
	return 1, p.broker.Deliver(nodeID, internal.Response{
		CorrelationID: request.CorrelationID,
		Body:          []byte("forwarded"),
	})
}

func newScenarioTestService(t *testing.T, scopes []string) (internal.Service, *scenarioTestPusher) {
	t.Helper()
	broker := internal.NewBroker()
	pusher := &scenarioTestPusher{broker: broker}
	nodes := &registrymocks.FakeService{GetOut: registry.Node{ID: "node-1", Scopes: scopes}}
	catalog := scopecatalog.Catalog{Scopes: []scopecatalog.Scope{
		{Scenario: "vrooli-onboarding", Value: "vrooli-onboarding:read", Effect: scopecatalog.EffectRead, Service: "OperatorInputsService", Method: "ListOperatorInputs"},
		{Scenario: "vrooli-onboarding", Value: "vrooli-onboarding:write", Effect: scopecatalog.EffectWrite, Service: "OperatorInputsService", Method: "ResolveOperatorInputs", RunEligible: false},
		{Scenario: "vrooli-onboarding", Value: "vrooli-onboarding:read", Effect: scopecatalog.EffectRead, Service: "ApplyService", Method: "GetApplyRun"},
	}}
	return newServiceWithCatalog(nodes, scenarioTestPresence{}, pusher, broker, catalog, nil), pusher
}

func TestScenarioProxyAdmitsDeclaredConnectProcedure(t *testing.T) {
	svc, pusher := newScenarioTestService(t, []string{"vrooli-onboarding:read", "vrooli-bridge:read"})

	_, err := svc.Call(context.Background(), internal.Request{
		NodeID: "node-1", Scenario: "vrooli-onboarding",
		Service: "vrooli.vrooli_onboarding.v1.operatorinputs.OperatorInputsService",
		Method:  "ListOperatorInputs",
	})

	require.NoError(t, err)
	require.Len(t, pusher.requests, 1)
}

func TestScenarioProxyRefusesUndeclaredProcedureByProcedureName(t *testing.T) {
	svc, _ := newScenarioTestService(t, []string{"vrooli-onboarding:read", "vrooli-bridge:read"})

	_, err := svc.Call(context.Background(), internal.Request{
		NodeID: "node-1", Scenario: "vrooli-onboarding",
		Service: "vrooli.vrooli_onboarding.v1.operatorinputs.OperatorInputsService",
		Method:  "NotDeclared",
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "OperatorInputsService/NotDeclared")
	require.NotContains(t, err.Error(), "governed catalog")
}

func TestScenarioProxyRefusesMissingNamespaceGrantByScopeName(t *testing.T) {
	svc, _ := newScenarioTestService(t, []string{"vrooli-bridge:read"})

	_, err := svc.Call(context.Background(), internal.Request{
		NodeID: "node-1", Scenario: "vrooli-onboarding",
		Service: "vrooli.vrooli_onboarding.v1.operatorinputs.OperatorInputsService",
		Method:  "ListOperatorInputs",
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "vrooli-onboarding:read")
}

// Finding P3: owner identity and the node grant authorize a proxy write;
// run_eligible only controls prompt-manager action invocation.
func TestScenarioProxyAdmitsOwnerWriteWhenRunIsNotEligible(t *testing.T) {
	svc, pusher := newScenarioTestService(t, []string{"vrooli-onboarding:write", "vrooli-bridge:write"})

	_, err := svc.Call(context.Background(), internal.Request{
		Actor: "owner", NodeID: "node-1", Scenario: "vrooli-onboarding",
		Service: "vrooli.vrooli_onboarding.v1.operatorinputs.OperatorInputsService",
		Method:  "ResolveOperatorInputs",
	})

	require.NoError(t, err)
	require.Len(t, pusher.requests, 1)
}

func TestScenarioProxyParameterisedRunIsAdmittedByProcedure(t *testing.T) {
	svc, pusher := newScenarioTestService(t, []string{"vrooli-onboarding:read", "vrooli-bridge:read"})

	_, err := svc.Call(context.Background(), internal.Request{
		NodeID: "node-1", Scenario: "vrooli-onboarding",
		Service: "vrooli.vrooli_onboarding.v1.apply.ApplyService",
		Method:  "GetApplyRun", HTTPPath: "api/v2/apply/run-123",
	})

	require.NoError(t, err)
	require.Len(t, pusher.requests, 1)
	require.Equal(t, "GetApplyRun", pusher.requests[0].Method)
}

func TestSplitProcedureRejectsRESTPath(t *testing.T) {
	_, _, err := splitProcedure("/api/v2/operator-inputs")

	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "scenario proxy accepts Connect procedures"))
	require.Contains(t, err.Error(), "/api/v2/operator-inputs")
}
