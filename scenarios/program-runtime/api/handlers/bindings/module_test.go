package bindings

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	internalbindings "program-runtime/internal/bindings"
	"program-runtime/internal/programs"
	"program-runtime/internal/sessions"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	repocontract "github.com/vrooli/repo-contract-go"
	bindingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

func TestEndpointsAreDeclared(t *testing.T) {
	if len(Endpoints) != 8 {
		t.Fatalf("endpoints=%d", len(Endpoints))
	}
}

func liveRegistry(t *testing.T) *internalbindings.Registry {
	t.Helper()
	root, err := repocontract.ResolveRepoRoot()
	require.NoError(t, err)
	registry, err := internalbindings.Load(root)
	require.NoError(t, err)
	return registry
}

func TestResolveActCellsLoadsOwnedDenominatorWhenRequestIsEmpty(t *testing.T) {
	service := &service{registry: liveRegistry(t)}
	response, err := service.ResolveActCells(context.Background(), connect.NewRequest(&bindingsv1.ResolveActCellsRequest{}))
	require.NoError(t, err)
	require.Len(t, response.Msg.GetCells(), 28)
	require.Equal(t, int32(28), response.Msg.GetAuditedCells())
	require.Equal(t, int32(28), response.Msg.GetTotalCells())
	require.Equal(t, "partial", response.Msg.GetDenominatorConfidence())
}

func TestResolveActCellsNamesUnavailableDenominator(t *testing.T) {
	service := &service{registry: liveRegistry(t), actSpacePath: filepath.Join(t.TempDir(), "missing-act-space.md")}
	_, err := service.ResolveActCells(context.Background(), connect.NewRequest(&bindingsv1.ResolveActCellsRequest{}))
	require.Error(t, err)
	require.Contains(t, err.Error(), "Act denominator")
	require.Contains(t, err.Error(), "missing-act-space.md")
}

func TestOrderedSearchHitsHonorsScoreBeforeIdentityTieBreak(t *testing.T) {
	hits := orderedSearchHits(&routingv1.QueryResponse{Ranked: []*routingv1.SearchHit{
		{Id: "lower", Score: 0.4},
		{Id: "best", Score: 1},
		{Id: "same-z", Score: 0.8},
		{Id: "same-a", Score: 0.8},
	}})
	require.Equal(t, []string{"best", "same-a", "same-z", "lower"}, []string{hits[0].GetId(), hits[1].GetId(), hits[2].GetId(), hits[3].GetId()})
}

func TestReviewedIntentVocabularyMakesDocumentedRunVerdictQueryDiscriminative(t *testing.T) {
	aliases := bindingIntentAliases(&bindingsv1.Binding{
		Id:       "test-genie/runs/status",
		Scenario: "test-genie",
		Group:    "runs",
		Command:  "status",
	})
	require.Contains(t, aliases, "read test run verdicts")
}

func TestExactReviewedIntentCandidateBeatsLexicalTie(t *testing.T) {
	preferred := &bindingsv1.Binding{Id: "search-hub/query/query"}
	competitor := &bindingsv1.Binding{Id: "architecture-cartographer/search/query"}
	got := exactReviewedIntentCandidate("search the project by intent", []*bindingsv1.Binding{competitor, preferred})
	require.Equal(t, preferred.GetId(), got.GetId())
}

type collectDelegatorFixture struct{}

func (collectDelegatorFixture) Delegate(context.Context, programs.DelegationRequest) (map[string]any, error) {
	return nil, nil
}
func (collectDelegatorFixture) Start(context.Context, programs.DelegationRequest) (map[string]any, error) {
	return nil, nil
}
func (collectDelegatorFixture) Collect(context.Context, string, string, int) (map[string]any, error) {
	return map[string]any{"status": "succeeded", "usage": map[string]any{"cost_micros": 5}}, nil
}

type fastTerminalDelegator struct{}

func (fastTerminalDelegator) Delegate(context.Context, programs.DelegationRequest) (map[string]any, error) {
	return nil, nil
}
func (fastTerminalDelegator) Start(context.Context, programs.DelegationRequest) (map[string]any, error) {
	return map[string]any{"execution_id": "fast-child", "status": "succeeded", "usage": map[string]any{"cost_micros": float64(11)}}, nil
}
func (fastTerminalDelegator) Collect(context.Context, string, string, int) (map[string]any, error) {
	return nil, nil
}

func TestAgentStartBridgeSettlesFastTerminalChild(t *testing.T) {
	manager := sessions.NewManager(sessions.Options{})
	session, err := manager.Create(t.Context(), "declared-program:fast", t.TempDir(), nil)
	require.NoError(t, err)
	body := `{"session_id":"` + session.ID + `","owner":"owner","workflow_key":"owner/workflow","idempotency_key":"key"}`
	response := httptest.NewRecorder()
	AgentStartBridge(manager, fastTerminalDelegator{}).ServeHTTP(response, httptest.NewRequest("POST", "/", strings.NewReader(body)))
	require.Equal(t, 200, response.Code)
	child, err := manager.GetDelegation(t.Context(), session.ID, "fast-child")
	require.NoError(t, err)
	require.Equal(t, "succeeded", child.LastStatus)
	require.True(t, child.UsageSettled)
	got, err := manager.Get(t.Context(), session.ID)
	require.NoError(t, err)
	require.EqualValues(t, 11, got.DelegationCostMicros)
	require.True(t, got.DelegationSpendMeasured)
}

type sharedExecutionDelegator struct{}

func (sharedExecutionDelegator) Delegate(context.Context, programs.DelegationRequest) (map[string]any, error) {
	return sharedExecutionResult(), nil
}
func (sharedExecutionDelegator) Start(context.Context, programs.DelegationRequest) (map[string]any, error) {
	return sharedExecutionResult(), nil
}
func (sharedExecutionDelegator) Collect(context.Context, string, string, int) (map[string]any, error) {
	return sharedExecutionResult(), nil
}

func sharedExecutionResult() map[string]any {
	return map[string]any{"execution_id": "shared-execution", "status": "succeeded", "owner": "owner", "workflow_key": "owner/workflow", "idempotency_key": "shared-key", "cost_micros": float64(7)}
}

func TestSyncAndAsyncDelegationShareExecutionSettlementOnce(t *testing.T) {
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(t.Context(), db, apidb.SchemaProviderFunc(sessions.Schema)))
	manager := sessions.NewManager(sessions.Options{Store: db})
	session, err := manager.Create(t.Context(), "declared-program:shared-execution", t.TempDir(), nil)
	require.NoError(t, err)
	body := `{"session_id":"` + session.ID + `","owner":"owner","workflow_key":"owner/workflow","idempotency_key":"shared-key"}`
	delegator := sharedExecutionDelegator{}
	for range 2 {
		response := httptest.NewRecorder()
		AgentBridge(manager, delegator).ServeHTTP(response, httptest.NewRequest("POST", "/", strings.NewReader(body)))
		require.Equal(t, http.StatusOK, response.Code)
	}
	response := httptest.NewRecorder()
	AgentStartBridge(manager, delegator).ServeHTTP(response, httptest.NewRequest("POST", "/", strings.NewReader(body)))
	require.Equal(t, http.StatusOK, response.Code)
	restarted := sessions.NewManager(sessions.Options{Store: db})
	child, err := restarted.GetDelegation(t.Context(), session.ID, "shared-execution")
	require.NoError(t, err)
	require.Equal(t, "shared-key", child.IdempotencyKey)
	require.True(t, child.UsageSettled)
	got, err := restarted.Get(t.Context(), session.ID)
	require.NoError(t, err)
	require.EqualValues(t, 7, got.DelegationCostMicros)
}

func TestAgentCollectBridgeRetainsTerminalDelegationStatus(t *testing.T) {
	manager := sessions.NewManager(sessions.Options{})
	session, err := manager.Create(t.Context(), "bridge-test", t.TempDir(), nil)
	require.NoError(t, err)
	require.NoError(t, manager.SaveDelegation(t.Context(), &sessions.Delegation{SessionID: session.ID, ExecutionID: "child-1", Owner: "owner", WorkflowKey: "owner/workflow", IdempotencyKey: "key", CreatedAt: time.Now().UTC(), LastStatus: "running"}))
	body := `{"session_id":"` + session.ID + `","execution_id":"child-1","wait_seconds":1}`
	bridge := AgentCollectBridge(manager, collectDelegatorFixture{})
	for range 2 {
		response := httptest.NewRecorder()
		bridge.ServeHTTP(response, httptest.NewRequest("POST", "/", strings.NewReader(body)))
		require.Equal(t, 200, response.Code)
	}
	delegation, err := manager.GetDelegation(t.Context(), session.ID, "child-1")
	require.NoError(t, err)
	require.Equal(t, "succeeded", delegation.LastStatus)
	settled, err := manager.Get(t.Context(), session.ID)
	require.NoError(t, err)
	require.EqualValues(t, 5, settled.DelegationCostMicros, "repeated collection must not settle child usage twice")
	require.True(t, settled.DelegationUsageObserved)
	require.True(t, settled.DelegationSpendMeasured)
}
