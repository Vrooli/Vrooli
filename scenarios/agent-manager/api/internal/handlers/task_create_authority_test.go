package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"agent-manager/internal/orchestration"

	"github.com/google/uuid"
	coreidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
)

func taskCreateGrantFixture() *coreidentity.OperationGrant {
	return &coreidentity.OperationGrant{
		ID:         "approved-run-create",
		Policy:     "agent-management",
		Approver:   "owner-a",
		Operations: []string{"agent.investigate", "agent.manage", "run.create"},
		Targets:    []string{"agent-manager:*"},
		Rules: []coreidentity.OperationRule{
			{Policy: "agent-management", Operations: []string{"agent.investigate", "agent.manage"}, Targets: []string{"agent-manager:*"}},
			{Policy: "run-creation", Operations: []string{"run.create"}, Targets: []string{"agent-manager:*"}},
		},
		ExpiresAt: time.Now().Add(time.Hour),
	}
}

func taskCreateRequest(t *testing.T, principal coreidentity.Principal, token string) *http.Request {
	t.Helper()
	return taskCreateRequestWithTask(t, principal, token, &domainpb.Task{Title: "A133 task", ScopePath: "src/a133"})
}

func taskCreateRequestWithTask(t *testing.T, principal coreidentity.Principal, token string, task *domainpb.Task) *http.Request {
	t.Helper()
	body := encodeProtoJSON(t, &apipb.CreateTaskRequest{Task: task})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set(cliutil.HeaderAgentIdentityToken, token)
	}
	return req.WithContext(coreidentity.WithPrincipal(req.Context(), principal))
}

func TestTaskCreateUsesOneRunCreateAuthority(t *testing.T) {
	_, router := setupTestHandler(t)
	serve := AgentOperationMiddleware(router)
	base := coreidentity.Principal{Kind: coreidentity.ActorAgent, Subject: "prompt-manager", Verified: true, Scopes: []string{"agent-manager:write"}}

	grant := taskCreateGrantFixture()
	valid := base
	valid.Grant = grant
	forgedCreator := uuid.NewString()
	positive := httptest.NewRecorder()
	serve.ServeHTTP(positive, taskCreateRequestWithTask(t, valid, "", &domainpb.Task{Title: "A133 task", ScopePath: "src/a133", CreatedBy: forgedCreator}))
	if positive.Code != http.StatusCreated {
		t.Fatalf("approved multi-policy run.create task status=%d body=%s; want %d", positive.Code, positive.Body.String(), http.StatusCreated)
	}
	var created apipb.CreateTaskResponse
	if err := protojson.Unmarshal(positive.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode run.create task response: %v", err)
	}
	if created.GetTask().GetCreatedBy() != "" {
		t.Fatalf("grant-only agent task retained unverified creator %q (forged %q)", created.GetTask().GetCreatedBy(), forgedCreator)
	}

	wrongOperation := base
	wrongOperation.Grant = &coreidentity.OperationGrant{
		ID: "agent-management-only", Approver: "owner-a", Operations: []string{"agent.manage"}, Targets: []string{"agent-manager:*"},
		Rules:     []coreidentity.OperationRule{{Policy: "agent-management", Operations: []string{"agent.manage"}, Targets: []string{"agent-manager:*"}}},
		ExpiresAt: time.Now().Add(time.Hour),
	}
	wrongTarget := base
	wrongTarget.Grant = &coreidentity.OperationGrant{
		ID: "wrong-target", Approver: "owner-a", Operations: []string{"run.create"}, Targets: []string{"agent-manager:run/other"},
		Rules:     []coreidentity.OperationRule{{Policy: "run-creation", Operations: []string{"run.create"}, Targets: []string{"agent-manager:run/other"}}},
		ExpiresAt: time.Now().Add(time.Hour),
	}
	crossedRule := base
	crossedRule.Grant = &coreidentity.OperationGrant{
		ID: "crossed-rule", Approver: "owner-a", Operations: []string{"agent.manage", "run.create"}, Targets: []string{"agent-manager:*", "agent-manager:run/other"},
		Rules: []coreidentity.OperationRule{
			{Policy: "agent-management", Operations: []string{"agent.manage"}, Targets: []string{"agent-manager:*"}},
			{Policy: "run-creation", Operations: []string{"run.create"}, Targets: []string{"agent-manager:run/other"}},
		},
		ExpiresAt: time.Now().Add(time.Hour),
	}
	expired := base
	expired.Grant = taskCreateGrantFixture()
	expired.ExpiresAt = time.Now().Add(-time.Second)
	unverified := base
	unverified.Verified = false
	unverified.Grant = taskCreateGrantFixture()
	unbound := base
	unbound.Subject = ""
	unbound.Grant = taskCreateGrantFixture()
	noAuthority := base

	for _, tc := range []struct {
		name      string
		principal coreidentity.Principal
		token     string
		want      int
	}{
		{"wrong operation", wrongOperation, "", http.StatusForbidden},
		{"wrong target", wrongTarget, "", http.StatusForbidden},
		{"crossed operations and targets do not flatten", crossedRule, "", http.StatusForbidden},
		{"expired principal grant", expired, "", http.StatusForbidden},
		{"unverified credential", unverified, "unverified-run-token", http.StatusUnauthorized},
		{"unbound principal", unbound, "unbound-run-token", http.StatusUnauthorized},
		{"no ambient operation authority", noAuthority, "", http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			serve.ServeHTTP(rr, taskCreateRequest(t, tc.principal, tc.token))
			if rr.Code != tc.want {
				t.Fatalf("task create status=%d body=%s; want %d", rr.Code, rr.Body.String(), tc.want)
			}
		})
	}
}

func TestRunCreateGrantUsesVerifiedActiveCallerForCreatorWithoutOrchestratorScope(t *testing.T) {
	h, router := setupTestHandler(t)
	callerID, profileID := uuid.New(), uuid.New()
	h.svc.IdentityService = guardIdentityService{result: &orchestration.IdentityVerifyResult{
		Valid: true, Claims: &identity.Claims{RunID: callerID},
	}}
	h.svc.RunService = lineageRuns{runs: map[uuid.UUID]*domain.Run{
		callerID: {ID: callerID, Status: domain.RunStatusRunning, AgentProfileID: &profileID},
	}}
	h.svc.ProfileService = lineageProfiles{profiles: map[uuid.UUID]*domain.AgentProfile{
		profileID: {ID: profileID, DeclaredScopes: []string{"agent-manager:read"}},
	}}
	principal := coreidentity.Principal{
		Kind: coreidentity.ActorAgent, Subject: callerID.String(), Verified: true, Scopes: []string{"agent-manager:write"},
		Grant: taskCreateGrantFixture(),
	}
	request := taskCreateRequestWithTask(t, principal, "verified-run-token", &domainpb.Task{
		Title: "Granted child", ScopePath: "scenarios/example", CreatedBy: uuid.NewString(),
	})
	rr := httptest.NewRecorder()
	AgentOperationMiddleware(router).ServeHTTP(rr, request)
	if rr.Code != http.StatusCreated {
		t.Fatalf("run.create grant with active non-orchestrator caller status=%d body=%s; want %d", rr.Code, rr.Body.String(), http.StatusCreated)
	}
	var response apipb.CreateTaskResponse
	if err := protojson.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode task create response: %v", err)
	}
	if got := response.GetTask().GetCreatedBy(); got != callerID.String() {
		t.Fatalf("created_by=%q, want verified active caller run UUID %q", got, callerID)
	}
}

func TestHumanTaskCreatePreservesOperatorCreatorAttribution(t *testing.T) {
	_, router := setupTestHandler(t)
	principal := coreidentity.Principal{Kind: coreidentity.ActorHuman, Subject: "operator", Verified: true, Scopes: []string{"agent-manager:write"}}
	request := taskCreateRequestWithTask(t, principal, "", &domainpb.Task{
		Title: "Operator task", ScopePath: "scenarios/example", CreatedBy: "operator-label",
	})
	rr := httptest.NewRecorder()
	AgentOperationMiddleware(router).ServeHTTP(rr, request)
	if rr.Code != http.StatusCreated {
		t.Fatalf("operator task create status=%d body=%s; want %d", rr.Code, rr.Body.String(), http.StatusCreated)
	}
	var response apipb.CreateTaskResponse
	if err := protojson.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode task create response: %v", err)
	}
	if got := response.GetTask().GetCreatedBy(); got != "operator-label" {
		t.Fatalf("operator attribution changed: got %q", got)
	}
}

func TestTaskCreateKeepsIntrinsicOrchestratorAuthorityWithUnrelatedGrant(t *testing.T) {
	h, router := setupTestHandler(t)
	callerID, profileID := uuid.New(), uuid.New()
	h.svc.IdentityService = guardIdentityService{result: &orchestration.IdentityVerifyResult{
		Valid: true, Claims: &identity.Claims{RunID: callerID},
	}}
	h.svc.RunService = lineageRuns{runs: map[uuid.UUID]*domain.Run{
		callerID: {ID: callerID, Status: domain.RunStatusRunning, AgentProfileID: &profileID},
	}}
	h.svc.ProfileService = lineageProfiles{profiles: map[uuid.UUID]*domain.AgentProfile{
		profileID: {ID: profileID, DeclaredScopes: []string{OrchestrateScope}},
	}}

	principal := coreidentity.Principal{
		Kind: coreidentity.ActorAgent, Subject: callerID.String(), Verified: true,
		Scopes: []string{"agent-manager:write"},
		Grant: &coreidentity.OperationGrant{
			ID: "unrelated-agent-management", Approver: "owner-a", Operations: []string{"agent.manage"}, Targets: []string{"agent-manager:*"},
		},
	}
	rr := httptest.NewRecorder()
	AgentOperationMiddleware(router).ServeHTTP(rr, taskCreateRequest(t, principal, "live-run-token"))
	if rr.Code != http.StatusCreated {
		t.Fatalf("profile orchestrator with unrelated grant status=%d body=%s; want %d", rr.Code, rr.Body.String(), http.StatusCreated)
	}
}

func TestIntrinsicTaskCreateStoresVerifiedCreatorInsteadOfClientLabel(t *testing.T) {
	h, router := setupTestHandler(t)
	callerID, profileID := uuid.New(), uuid.New()
	h.svc.IdentityService = guardIdentityService{result: &orchestration.IdentityVerifyResult{
		Valid: true, Claims: &identity.Claims{RunID: callerID},
	}}
	h.svc.RunService = lineageRuns{runs: map[uuid.UUID]*domain.Run{
		callerID: {ID: callerID, Status: domain.RunStatusRunning, AgentProfileID: &profileID},
	}}
	h.svc.ProfileService = lineageProfiles{profiles: map[uuid.UUID]*domain.AgentProfile{
		profileID: {ID: profileID, DeclaredScopes: []string{OrchestrateScope}},
	}}
	principal := coreidentity.Principal{Kind: coreidentity.ActorAgent, Subject: callerID.String(), Verified: true, Scopes: []string{"agent-manager:write"}}
	for _, suppliedCreator := range []string{"", "spoofed-creator", "personal-planner-delivery-orchestrator"} {
		t.Run("supplied creator="+suppliedCreator, func(t *testing.T) {
			task := &domainpb.Task{Title: "child task", ScopePath: "scenarios/personal-planner", CreatedBy: suppliedCreator}
			rr := httptest.NewRecorder()
			AgentOperationMiddleware(router).ServeHTTP(rr, taskCreateRequestWithTask(t, principal, "verified-orchestrator-token", task))
			if rr.Code != http.StatusCreated {
				t.Fatalf("intrinsic task create status=%d body=%s; want %d", rr.Code, rr.Body.String(), http.StatusCreated)
			}
			var response apipb.CreateTaskResponse
			if err := protojson.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode task create response: %v", err)
			}
			if got := response.GetTask().GetCreatedBy(); got != callerID.String() {
				t.Fatalf("created_by=%q, want verified caller run UUID %q", got, callerID)
			}
		})
	}
}

func TestTaskUpdateAllowsCreatorOrchestratorToEditOnlyQueuedBrief(t *testing.T) {
	h, router, repos, _ := setupTestHandlerWithRunnerAndRepos(t, runner.NewMockRunner(domain.RunnerTypeClaudeCode))
	callerID, profileID, taskID := uuid.New(), uuid.New(), uuid.New()
	h.svc.IdentityService = guardIdentityService{result: &orchestration.IdentityVerifyResult{
		Valid: true, Claims: &identity.Claims{RunID: callerID},
	}}
	h.svc.RunService = lineageRuns{runs: map[uuid.UUID]*domain.Run{
		callerID: {ID: callerID, Status: domain.RunStatusRunning, AgentProfileID: &profileID},
	}}
	h.svc.ProfileService = lineageProfiles{profiles: map[uuid.UUID]*domain.AgentProfile{
		profileID: {ID: profileID, DeclaredScopes: []string{OrchestrateScope}},
	}}
	existing := &domain.Task{ID: taskID, Title: "Queued work", Description: "Draft", ScopePath: "scenarios/notification-hub", ProjectRoot: "/repo", Status: domain.TaskStatusQueued, CreatedBy: callerID.String()}
	if err := repos.Tasks.Create(context.Background(), existing); err != nil {
		t.Fatal(err)
	}

	principal := coreidentity.Principal{Kind: coreidentity.ActorAgent, Subject: callerID.String(), Verified: true, Scopes: []string{"agent-manager:write"}}
	for _, tc := range []struct {
		name        string
		title       string
		description string
	}{
		{name: "whitespace-only title", title: " \t\n", description: "Draft"},
		{name: "multibyte title exceeds byte limit", title: strings.Repeat("界", 86), description: "Draft"},
		{name: "description exceeds byte limit", title: "Queued work", description: strings.Repeat("x", 65537)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPut, "/api/v1/tasks/"+taskID.String(), bytes.NewReader(encodeProtoJSON(t, &apipb.UpdateTaskRequest{
				TaskId: taskID.String(),
				Task:   &domainpb.Task{Title: tc.title, Description: tc.description, ScopePath: existing.ScopePath},
			})))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set(cliutil.HeaderAgentIdentityToken, "owner-run-token")
			request = request.WithContext(coreidentity.WithPrincipal(request.Context(), principal))
			rr := httptest.NewRecorder()
			AgentOperationMiddleware(router).ServeHTTP(rr, request)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("invalid brief status=%d body=%s; want %d", rr.Code, rr.Body.String(), http.StatusBadRequest)
			}
			unchanged, err := repos.Tasks.Get(context.Background(), taskID)
			if err != nil {
				t.Fatal(err)
			}
			if unchanged.Title != existing.Title || unchanged.Description != existing.Description {
				t.Fatalf("rejected brief mutated task: title=%q description length=%d", unchanged.Title, len(unchanged.Description))
			}
		})
	}

	request := httptest.NewRequest(http.MethodPut, "/api/v1/tasks/"+taskID.String(), bytes.NewReader(encodeProtoJSON(t, &apipb.UpdateTaskRequest{
		TaskId: taskID.String(),
		Task:   &domainpb.Task{Title: "Queued work", Description: "Final reviewed brief", ScopePath: existing.ScopePath},
	})))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(cliutil.HeaderAgentIdentityToken, "owner-run-token")
	request = request.WithContext(coreidentity.WithPrincipal(request.Context(), principal))
	rr := httptest.NewRecorder()
	AgentOperationMiddleware(router).ServeHTTP(rr, request)
	if rr.Code != http.StatusOK {
		t.Fatalf("same creator orchestrator task brief update status=%d body=%s; want 200", rr.Code, rr.Body.String())
	}
	updated, err := repos.Tasks.Get(context.Background(), taskID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "Queued work" || updated.Description != "Final reviewed brief" || updated.ScopePath != existing.ScopePath || updated.ProjectRoot != existing.ProjectRoot || updated.Status != domain.TaskStatusQueued || updated.CurrentRunID != nil || updated.CreatedBy != callerID.String() {
		t.Fatalf("brief update changed non-brief or ownership state: %+v", updated)
	}
}

func TestTaskUpdateRejectsUnownedOrDispatchedOrchestratorTargets(t *testing.T) {
	callerID, otherID := uuid.New(), uuid.New()
	for _, tc := range []struct {
		name              string
		createdBy         string
		status            domain.TaskStatus
		currentRun        bool
		orchestrate       bool
		verified          bool
		noPrincipal       bool
		changeScope       bool
		changeStatus      bool
		changeProjectRoot bool
		changeCreator     bool
		changeCurrentRun  bool
	}{
		{name: "other creator", createdBy: otherID.String(), status: domain.TaskStatusQueued, orchestrate: true, verified: true},
		{name: "unknown creator", status: domain.TaskStatusQueued, orchestrate: true, verified: true},
		{name: "profile lacks orchestrate scope", createdBy: callerID.String(), status: domain.TaskStatusQueued, verified: true},
		{name: "unverified run identity", createdBy: callerID.String(), status: domain.TaskStatusQueued, orchestrate: true},
		{name: "running task", createdBy: callerID.String(), status: domain.TaskStatusRunning, orchestrate: true, verified: true},
		{name: "queued task already has current run", createdBy: callerID.String(), status: domain.TaskStatusQueued, currentRun: true, orchestrate: true, verified: true},
		{name: "scope mutation is outside brief", createdBy: callerID.String(), status: domain.TaskStatusQueued, orchestrate: true, verified: true, changeScope: true},
		{name: "status mutation is outside brief", createdBy: callerID.String(), status: domain.TaskStatusQueued, orchestrate: true, verified: true, changeStatus: true},
		{name: "project root mutation is outside brief", createdBy: callerID.String(), status: domain.TaskStatusQueued, orchestrate: true, verified: true, changeProjectRoot: true},
		{name: "creator mutation is outside brief", createdBy: callerID.String(), status: domain.TaskStatusQueued, orchestrate: true, verified: true, changeCreator: true},
		{name: "run binding mutation is outside brief", createdBy: callerID.String(), status: domain.TaskStatusQueued, orchestrate: true, verified: true, changeCurrentRun: true},
		{name: "no verified principal", createdBy: callerID.String(), status: domain.TaskStatusQueued, orchestrate: true, noPrincipal: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, router, repos, _ := setupTestHandlerWithRunnerAndRepos(t, runner.NewMockRunner(domain.RunnerTypeClaudeCode))
			profileID, taskID := uuid.New(), uuid.New()
			profileScopes := []string(nil)
			if tc.orchestrate {
				profileScopes = []string{OrchestrateScope}
			}
			h.svc.IdentityService = guardIdentityService{result: &orchestration.IdentityVerifyResult{
				Valid: tc.verified, Claims: &identity.Claims{RunID: callerID},
			}}
			h.svc.RunService = lineageRuns{runs: map[uuid.UUID]*domain.Run{
				callerID: {ID: callerID, Status: domain.RunStatusRunning, AgentProfileID: &profileID},
			}}
			h.svc.ProfileService = lineageProfiles{profiles: map[uuid.UUID]*domain.AgentProfile{
				profileID: {ID: profileID, DeclaredScopes: profileScopes},
			}}
			task := &domain.Task{ID: taskID, Title: "Queued work", Description: "Draft", ScopePath: "scenarios/notification-hub", ProjectRoot: "/repo", Status: tc.status, CreatedBy: tc.createdBy}
			if tc.currentRun {
				runID := uuid.New()
				task.CurrentRunID = &runID
			}
			if err := repos.Tasks.Create(context.Background(), task); err != nil {
				t.Fatal(err)
			}

			requested := &domainpb.Task{Title: task.Title, Description: "Changed brief", ScopePath: task.ScopePath}
			if tc.changeScope {
				requested.ScopePath = "scenarios/other"
			}
			if tc.changeStatus {
				requested.Status = domainpb.TaskStatus_TASK_STATUS_RUNNING
			}
			if tc.changeProjectRoot {
				requested.ProjectRoot = "/another-repo"
			}
			if tc.changeCreator {
				requested.CreatedBy = otherID.String()
			}
			if tc.changeCurrentRun {
				requested.CurrentRunId = uuid.NewString()
			}
			request := httptest.NewRequest(http.MethodPut, "/api/v1/tasks/"+taskID.String(), bytes.NewReader(encodeProtoJSON(t, &apipb.UpdateTaskRequest{TaskId: taskID.String(), Task: requested})))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set(cliutil.HeaderAgentIdentityToken, "owner-run-token")
			if !tc.noPrincipal {
				principal := coreidentity.Principal{Kind: coreidentity.ActorAgent, Subject: callerID.String(), Verified: true, Scopes: []string{"agent-manager:write"}}
				request = request.WithContext(coreidentity.WithPrincipal(request.Context(), principal))
			}
			rr := httptest.NewRecorder()
			AgentOperationMiddleware(router).ServeHTTP(rr, request)
			wantStatus := http.StatusForbidden
			if tc.noPrincipal {
				wantStatus = http.StatusUnauthorized
			}
			if rr.Code != wantStatus {
				t.Fatalf("unauthorized task update status=%d body=%s; want %d", rr.Code, rr.Body.String(), wantStatus)
			}
			unchanged, err := repos.Tasks.Get(context.Background(), taskID)
			if err != nil {
				t.Fatal(err)
			}
			if unchanged.Title != task.Title || unchanged.Description != task.Description || unchanged.ScopePath != task.ScopePath || unchanged.ProjectRoot != task.ProjectRoot || unchanged.Status != task.Status || unchanged.CreatedBy != task.CreatedBy || (unchanged.CurrentRunID == nil) != (task.CurrentRunID == nil) {
				t.Fatalf("refused update mutated task: before=%+v after=%+v", task, unchanged)
			}
		})
	}
}

func TestTaskUpdateStillAllowsAuthorizedOperator(t *testing.T) {
	_, router, repos, _ := setupTestHandlerWithRunnerAndRepos(t, runner.NewMockRunner(domain.RunnerTypeClaudeCode))
	taskID := uuid.New()
	task := &domain.Task{ID: taskID, Title: "Operator task", Description: "Old", ScopePath: "scenarios/old", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPut, "/api/v1/tasks/"+taskID.String(), bytes.NewReader(encodeProtoJSON(t, &apipb.UpdateTaskRequest{
		TaskId: taskID.String(), Task: &domainpb.Task{Title: "Operator task", Description: "Updated", ScopePath: "scenarios/new"},
	})))
	request.Header.Set("Content-Type", "application/json")
	principal := coreidentity.Principal{Kind: coreidentity.ActorHuman, Subject: "fixture-owner", Verified: true, Scopes: []string{"agent-manager:write"}}
	request = request.WithContext(coreidentity.WithPrincipal(request.Context(), principal))
	rr := httptest.NewRecorder()
	AgentOperationMiddleware(router).ServeHTTP(rr, request)
	if rr.Code != http.StatusOK {
		t.Fatalf("authorized operator update status=%d body=%s; want 200", rr.Code, rr.Body.String())
	}
	updated, err := repos.Tasks.Get(context.Background(), taskID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ScopePath != "scenarios/new" || updated.Description != "Updated" {
		t.Fatalf("operator task update did not retain existing capability: %+v", updated)
	}
}

func TestTaskUpdatePreservesExactAgentManageGrantPath(t *testing.T) {
	_, router, repos, _ := setupTestHandlerWithRunnerAndRepos(t, runner.NewMockRunner(domain.RunnerTypeClaudeCode))
	taskID := uuid.New()
	task := &domain.Task{ID: taskID, Title: "Managed task", Description: "Old", ScopePath: "scenarios/old", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPut, "/api/v1/tasks/"+taskID.String(), bytes.NewReader(encodeProtoJSON(t, &apipb.UpdateTaskRequest{
		TaskId: taskID.String(), Task: &domainpb.Task{Title: "Managed task", Description: "Updated", ScopePath: "scenarios/new"},
	})))
	request.Header.Set("Content-Type", "application/json")
	principal := coreidentity.Principal{Kind: coreidentity.ActorAgent, Subject: "global-task-manager", Verified: true, Scopes: []string{"agent-manager:write"}, Grant: taskCreateGrantFixture()}
	request = request.WithContext(coreidentity.WithPrincipal(request.Context(), principal))
	rr := httptest.NewRecorder()
	AgentOperationMiddleware(router).ServeHTTP(rr, request)
	if rr.Code != http.StatusOK {
		t.Fatalf("exact agent.manage task update status=%d body=%s; want 200", rr.Code, rr.Body.String())
	}
	updated, err := repos.Tasks.Get(context.Background(), taskID)
	if err != nil || updated == nil {
		t.Fatalf("read globally managed task: task=%+v err=%v", updated, err)
	}
	if updated.ScopePath != "scenarios/new" || updated.Description != "Updated" {
		t.Fatalf("exact agent.manage update lost its existing global path: %+v", updated)
	}
}
