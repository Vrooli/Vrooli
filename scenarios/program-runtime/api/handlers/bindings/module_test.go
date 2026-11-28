package bindings

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
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
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	bindingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestEndpointsAreDeclared(t *testing.T) {
	if len(Endpoints) != 8 {
		t.Fatalf("endpoints=%d", len(Endpoints))
	}
}

type privateIdentityFixture struct{ name, digest, session string }

func (f privateIdentityFixture) ExecutionIdentity(context.Context, string) (string, string, string, error) {
	return f.name, f.digest, f.session, nil
}

func TestDirectSessionCannotDispatchPrivateOmission(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	require.NoError(t, err)
	digest := strings.Repeat("a", 64)
	manifest := `{"name":"workspace-sandbox","groups":[{"name":"sandbox","commands":[{"name":"create","binding":{"kind":"connect-rpc","service":"WorkspaceSandboxService","method":"CreateSandbox"},"governance":{"effect":"write","run_eligible":true,"permissions":["network:internal"]}}]}],"omitted":[{"service":"WorkspaceSandboxService","method":"CreateReplaySeal","reason":"private replay operation","program_only":{"binding_id":"workspace-sandbox/replay-seal/create","program":"program-runtime.bounty-replay","digest":"` + digest + `","effect":"write","permissions":["filesystem:read","filesystem:write","network:internal"]}}]}`
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	require.NoError(t, os.WriteFile(manifestPath, []byte(manifest), 0o600))
	registry, err := internalbindings.LoadFiles(filepath.Join(root, "packages/proto/gen/descriptor/image.binpb"), []string{manifestPath})
	require.NoError(t, err)
	manager := sessions.NewManager(sessions.Options{})
	session, err := manager.Create(context.Background(), "direct-user", "", nil)
	require.NoError(t, err)
	defer func() { _, _ = manager.Delete(context.Background(), session.ID, "test complete") }()

	for _, tc := range []struct {
		label, program, pinnedDigest, pinnedSession string
	}{
		{"unrelated program", "program-runtime.other", digest, session.ID},
		{"digest mismatch", "program-runtime.bounty-replay", strings.Repeat("b", 64), session.ID},
		{"missing digest", "program-runtime.bounty-replay", "", session.ID},
		{"raw session user-code has no declared program identity", "", "", session.ID},
		{"execution identity belongs to another session", "program-runtime.bounty-replay", digest, "different-session"},
	} {
		t.Run(tc.label, func(t *testing.T) {
			bridge := BridgeWithPrivateIdentity(registry, manager, privateIdentityFixture{tc.program, tc.pinnedDigest, tc.pinnedSession})
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"session_id":"`+session.ID+`","program_id":"raw-program","provenance":"PROVENANCE_OPERATOR","binding_id":"workspace-sandbox/replay-seal/create","args":{"sandboxId":"sbx","caseId":"case","role":"REPLAY_SEAL_ROLE_SOURCE"}}`))
			response := httptest.NewRecorder()
			bridge.ServeHTTP(response, request)
			require.NotEqual(t, http.StatusOK, response.Code)
			require.Contains(t, response.Body.String(), "binding does not exist")
		})
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

func TestJudgeIntentCarriesConfidentialLocalOnlyCallerContract(t *testing.T) {
	var received inferencev1.RunRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read Gateway request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(body, &received); err != nil {
			t.Errorf("decode Gateway request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"valueJson":"{\"binding_id\":\"search-hub/query/query\",\"confidence\":\"high\"}"}`))
	}))
	defer server.Close()

	registry := liveRegistry(t)
	registry.SetReachabilityResolver(func(context.Context, string) (string, error) { return server.URL, nil })
	owner := &service{registry: registry}
	selected, confidence, err := owner.judgeIntent(t.Context(), "search for a binding", []*bindingsv1.Binding{{Id: "search-hub/query/query"}})
	require.NoError(t, err)
	require.Equal(t, "search-hub/query/query", selected.GetId())
	require.Equal(t, "high", confidence)
	require.Equal(t, sharedv1.Profile_PROFILE_LOCAL_ONLY, received.GetProfile())
	require.Equal(t, sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL, received.GetPrivacyClass())
	require.Equal(t, "program_runtime.binding_discovery.intent_and_candidate_metadata", received.GetPrivacyProvenance())
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
