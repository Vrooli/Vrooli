package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"agent-manager/internal/orchestration/phases"
	"agent-manager/internal/orchestration/testutil/mocks"
	"agent-manager/internal/workflowruntime"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/authn"
	coreidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
)

type identityAckGate struct {
	workflowChildLauncher
	gate <-chan struct{}
}

func (l identityAckGate) StartFresh(ctx context.Context, req workflowruntime.ChildRequest) (workflowruntime.ChildState, error) {
	state, err := l.workflowChildLauncher.StartFresh(ctx, req)
	if err != nil {
		return state, err
	}
	select {
	case <-l.gate:
	case <-ctx.Done():
		return state, ctx.Err()
	}
	return state, nil
}

// Real workflow admission, launch and minting feed the actual owner verifier
// over HTTP, then cliutil and shared provenance. No verified-claims map is
// fabricated for the positive case. A deliberate lost acknowledgement holds
// attempt.RunID empty while the runner makes its first downstream request.
func TestWorkflowIssuedIdentityReachesVerifiedOwnerBeforeDispatchAcknowledgement(t *testing.T) {
	o, repos := newWorkflowChildOrchestrator(t)
	o.identitySecret = []byte("workflow-cross-owner-fixture")
	WithWorkflowRepository(repos.Workflows)(o)
	WithWorkflowExecutionRepository(repos.WorkflowExecutions)(o)
	attachRelayEngine(o, repos, newFakeRunLauncher())
	ack := make(chan struct{})
	runDone := make(chan struct{})
	captured := make(chan runner.ExecuteRequest, 1)
	o.workflowEngine.Children = identityAckGate{workflowChildLauncher: workflowChildLauncher{o: o}, gate: ack}
	registered, err := o.runners.Get(domain.RunnerTypeCodex)
	if err != nil {
		t.Fatal(err)
	}
	registered.(*mocks.TranscriptReplayRunner).ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		captured <- req
		select {
		case <-runDone:
			return &runner.ExecuteResult{Success: true}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	profile, err := o.CreateProfile(t.Context(), &domain.AgentProfile{Name: "narrow evidence child", ProfileKey: "fixture/evidence-child", RoleRef: "code.default", DeclaredScopes: []string{"git-control-tower:read"}, MaxTurns: 2, Timeout: time.Minute, SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeOff}})
	if err != nil {
		t.Fatal(err)
	}
	revision := relayDefinition()
	revision.Definition.Nodes = revision.Definition.Nodes[:1]
	revision.Definition.Nodes[0].Run.ProfileKey = profile.ProfileKey
	revision.Definition.Edges = nil
	if err = repos.Workflows.ActivateBatch(t.Context(), []*domain.WorkflowRevision{revision}); err != nil {
		t.Fatal(err)
	}
	ctx := coreidentity.WithPrincipal(t.Context(), coreidentity.Principal{Kind: coreidentity.ActorHuman, Verified: true, Subject: "fixture-owner", Scopes: []string{"agent-manager:write", "git-control-tower:read"}, ExpiresAt: time.Now().Add(time.Hour)})
	started := make(chan error, 1)
	var ackOnce sync.Once
	joined := false
	defer func() {
		ackOnce.Do(func() { close(ack) })
		close(runDone)
		if !joined {
			select {
			case <-started:
			case <-time.After(3 * time.Second):
				t.Error("workflow admission goroutine did not finish")
			}
		}
	}()
	go func() {
		_, err := o.StartWorkflowExecution(ctx, StartWorkflowExecutionRequest{Owner: "owner", WorkflowKey: revision.Key, Input: json.RawMessage(`{}`), IdempotencyKey: "actual-workflow-owner"})
		started <- err
	}()
	var request runner.ExecuteRequest
	select {
	case request = <-captured:
	case err := <-started:
		t.Fatalf("dispatch exited before mint: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("runner never received issued identity")
	}
	token := request.Environment[cliutil.EnvIdentityToken]
	if token == "" {
		t.Fatal("runner received no issued identity")
	}
	execution := request.Environment[workflowExecutionEnv]
	node := request.Environment[workflowNodeEnv]
	attempt := request.Environment[workflowAttemptEnv]
	attempts, err := repos.WorkflowExecutions.ListAttempts(t.Context(), uuid.MustParse(execution))
	if err != nil || len(attempts) != 1 || attempts[0].RunID != nil {
		t.Fatal("fixture did not exercise pre-acknowledgement window")
	}
	claims, err := identity.VerifyToken(token, o.identitySecret)
	if err != nil || claims.Meta["workflowExecutionId"] != execution || claims.Meta["workflowNodeId"] != node || claims.Meta["workflowAttemptId"] != attempt {
		t.Fatal("canonical mint lost its actual workflow coordinates", err)
	}
	verifyOwner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/v1/identity/verify" {
			http.NotFound(w, r)
			return
		}
		var body struct {
			Token string `json:"token"`
		}
		if json.NewDecoder(r.Body).Decode(&body) != nil {
			w.WriteHeader(400)
			return
		}
		result, err := o.VerifyIdentityToken(r.Context(), body.Token)
		if err != nil {
			http.Error(w, "verification unavailable", 503)
			return
		}
		_ = json.NewEncoder(w).Encode(result)
	}))
	defer verifyOwner.Close()
	t.Setenv("AGENT_MANAGER_API_BASE", verifyOwner.URL)
	downstream := httptest.NewServer(provenance.Middleware(provenance.CLIUtilVerifier{})(authn.Middleware(authn.Config{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := provenance.FromContext(r.Context())
		if _, err := authn.RequireCapability(r.Context(), "git-control-tower:read"); err != nil || !p.IsVerifiedAgent() || p.WorkflowExecutionID != execution || p.WorkflowNodeID != node {
			http.Error(w, "owning verified workflow child required", 403)
			return
		}
		if p.Attempt != 0 {
			http.Error(w, "attempt UUID became numeric ordinal", 500)
			return
		}
		w.WriteHeader(200)
	}))))
	defer downstream.Close()
	read := func(credential string) int {
		req, _ := http.NewRequest(http.MethodGet, downstream.URL, nil)
		req.Header.Set(cliutil.HeaderAgentIdentityToken, credential)
		req.Header.Set("X-Vrooli-Workflow-Execution-Id", execution)
		response, err := downstream.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		return response.StatusCode
	}
	if code := read(token); code != 200 {
		t.Fatalf("actual minted owning child denied before dispatch ACK: HTTP%d", code)
	}
	// Neither the private receipt alone nor caller labels can substitute for
	// exact retained attempt/task/idempotency membership before acknowledgement.
	decoyTask := &domain.Task{ID: uuid.New(), Title: "unrelated task", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(t.Context(), decoyTask); err != nil {
		t.Fatal(err)
	}
	original, err := repos.Runs.Get(t.Context(), request.RunID)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"missing receipt", "wrong task", "wrong key", "wrong private attempt", "wrong private execution"} {
		t.Run(name, func(t *testing.T) {
			raw, _ := json.Marshal(original)
			var altered domain.Run
			if err := json.Unmarshal(raw, &altered); err != nil {
				t.Fatal(err)
			}
			switch name {
			case "missing receipt":
				altered.ResolvedConfig.Admission.WorkflowIdentity = nil
			case "wrong task":
				altered.TaskID = decoyTask.ID
			case "wrong key":
				altered.IdempotencyKey = "foreign-key"
			case "wrong private attempt":
				altered.ResolvedConfig.Admission.WorkflowIdentity.AttemptID = uuid.New()
			case "wrong private execution":
				altered.ResolvedConfig.Admission.WorkflowIdentity.ExecutionID = uuid.New()
			}
			checked, err := identity.VerifyToken(token, o.identitySecret)
			if err != nil {
				t.Fatal(err)
			}
			if err = o.projectVerifiedWorkflowIdentity(t.Context(), &altered, checked); err != nil {
				t.Fatal(err)
			}
			if checked.Meta["workflowExecutionId"] != "" {
				t.Fatal("unproven pre-ACK membership retained workflow coordinates")
			}

		})
	}
	// A historical, active signed token may carry attacker-chosen environment
	// labels. Signature and read scope alone cannot establish workflow membership.
	foreign := &domain.Run{ID: uuid.New(), TaskID: decoyTask.ID, Status: domain.RunStatusRunning, OwnerSubject: "fixture-owner", OwnerScopes: []string{"git-control-tower:read"}, CustomEnv: map[string]string{workflowExecutionEnv: execution, workflowNodeEnv: node, workflowAttemptEnv: attempt}}
	forged, err := identity.GenerateToken(&identity.Claims{RunID: foreign.ID, TaskID: foreign.TaskID, Subject: foreign.OwnerSubject, Scopes: foreign.OwnerScopes, IssuedAt: time.Now().Unix(), ExpiresAt: time.Now().Add(time.Hour).Unix(), Meta: workflowIdentityMeta(foreign.CustomEnv)}, o.identitySecret)
	if err != nil {
		t.Fatal(err)
	}
	foreign.IdentityTokenHash = identity.HashToken(forged)
	if err = repos.Runs.Create(t.Context(), foreign); err != nil {
		t.Fatal(err)
	}
	baseline, err := o.VerifyIdentityToken(t.Context(), forged)
	if err != nil || !baseline.Valid || len(baseline.Claims.Scopes) != 1 || baseline.Claims.Scopes[0] != "git-control-tower:read" || baseline.Claims.Meta["workflowExecutionId"] != "" {
		t.Fatal("foreign labels changed baseline authority or retained workflow ownership", err)
	}
	if code := read(forged); code != 403 {
		t.Fatalf("historical foreign signed labels granted workflow ownership: HTTP%d", code)
	}
	if code := read(token + "tampered"); code != 403 {
		t.Fatal("invalid credential accepted")
	}
	ackOnce.Do(func() { close(ack) })
	select {
	case err := <-started:
		joined = true
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("workflow acknowledgement did not finish")
	}
	historical, err := repos.Runs.Get(t.Context(), request.RunID)
	if err != nil {
		t.Fatal(err)
	}
	historical.ResolvedConfig.Admission.WorkflowIdentity = nil
	profileForMint := profile
	historicalToken := phases.GenerateIdentityToken(t.Context(), phases.GenerateIdentityTokenInput{Run: historical, Profile: profileForMint, Task: request.Task, Secret: o.identitySecret, Meta: workflowIdentityMeta(historical.CustomEnv), Deps: phases.Deps{Runs: repos.Runs}})
	if code := read(historicalToken); code != 200 {
		t.Fatalf("historical exact retained RunID denied after ACK: HTTP%d", code)
	}
	// Continue/wake rotates identity through this actual continuation helper.
	// It retains the original run/task/CustomEnv, so ownership remains anchored
	// to the already acknowledged original attempt, without a fresh-task proof.
	continuationEnv, err := o.assembleContinuationEnv(t.Context(), historical, request.Task, profile, request.WorkingDir)
	if err != nil {
		t.Fatal(err)
	}
	if code := read(continuationEnv[cliutil.EnvIdentityToken]); code != 200 {
		t.Fatalf("real continuation issuance lost acknowledged workflow membership: HTTP%d", code)
	}
	t.Logf("actual minted child admitted before attempt acknowledgement; historical signed foreign labels denied; execution=%s", execution)
}

func TestPublicCreateRunRejectsWorkflowIdentityEnvironmentBeforeEffects(t *testing.T) {
	for _, key := range []string{workflowExecutionEnv, workflowNodeEnv, workflowAttemptEnv} {
		t.Run(key, func(t *testing.T) {
			o, _ := newWorkflowChildOrchestrator(t)
			_, err := o.CreateRun(t.Context(), CreateRunRequest{Force: true, Environment: map[string]string{key: "caller-chosen"}})
			if err == nil || !strings.Contains(fmt.Sprint(err), "environment") {
				t.Fatalf("public workflow environment was not rejected before admission: %v", err)
			}
		})
	}
}
