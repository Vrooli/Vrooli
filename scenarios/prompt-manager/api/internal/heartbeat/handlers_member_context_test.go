package heartbeat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"prompt-manager/internal/paths"
	"prompt-manager/internal/store"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
)

type heartbeatManagementAuthority struct {
	capability string
	operation  string
	target     string
}

func (a heartbeatManagementAuthority) AuthorizeOperation(_ context.Context, _ identity.Principal, capability, operation, target string) error {
	if capability != a.capability || operation != a.operation || target != a.target {
		return authn.ErrOperation
	}
	return nil
}

func TestRequireBoundedHeartbeatManagementUsesExactTeamManage(t *testing.T) {
	requestAuthority := func(operation, target string) context.Context {
		ctx := identity.WithPrincipal(context.Background(), identity.Principal{
			Kind: identity.ActorAgent, Subject: "verified-run", Verified: true,
			Scopes: []string{"prompt-manager:write"},
		})
		return authn.WithRequestOperationAuthority(ctx, heartbeatManagementAuthority{
			capability: "prompt-manager:write", operation: operation, target: target,
		})
	}
	for _, tc := range []struct {
		name string
		ctx  context.Context
		want bool
	}{
		{name: "exact team.manage target", ctx: requestAuthority("team.manage", "prompt-manager:team/team-1"), want: true},
		{name: "wrong team target", ctx: requestAuthority("team.manage", "prompt-manager:team/other")},
		{name: "team.prepare is not team.manage", ctx: requestAuthority("team.prepare", "prompt-manager:team/team-1")},
		{name: "ordinary legacy caller", ctx: context.Background(), want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := requireBoundedHeartbeatManagement(tc.ctx, "team-1")
			if (err == nil) != tc.want {
				t.Fatalf("requireBoundedHeartbeatManagement error=%v, want allowed=%v", err, tc.want)
			}
		})
	}
}

func TestGetMemberContext_Success(t *testing.T) {
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()

	ctx := context.Background()
	team := newIndependentTestTeam("team-1", "Team")
	if err := teamStore.Create(ctx, team); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := agentStore.Create(ctx, &store.Agent{ID: "agent-1", DisplayName: "Agent"}); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := relationStore.SetTeamMember(ctx, &store.TeamMemberRelation{
		TeamID: "team-1", AgentID: "agent-1", Status: store.MemberStatusActive,
	}); err != nil {
		t.Fatalf("create membership: %v", err)
	}
	if err := teamStore.SetResponsibilities(ctx, "team-1", "agent-1", "Do important work"); err != nil {
		t.Fatalf("set responsibilities: %v", err)
	}

	executor := newTestExecutor(t, teamStore, agentStore, nil, "", nil, nil)
	handlers := NewHandlers(HandlersDeps{
		TeamStore:     teamStore,
		AgentStore:    agentStore,
		RelationStore: relationStore,
		Executor:      executor,
	})

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/members/agent-1/context", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w := httptest.NewRecorder()

	handlers.GetMemberContext(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp MemberContextResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.TeamID != "team-1" {
		t.Errorf("expected teamId 'team-1', got %q", resp.TeamID)
	}
	if resp.AgentID != "agent-1" {
		t.Errorf("expected agentId 'agent-1', got %q", resp.AgentID)
	}
	if resp.Prompt == "" {
		t.Error("expected non-empty prompt")
	}
}

func TestGetMemberContext_TeamNotFound(t *testing.T) {
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()

	executor := newTestExecutor(t, teamStore, agentStore, nil, "", nil, nil)
	handlers := NewHandlers(HandlersDeps{
		TeamStore:     teamStore,
		AgentStore:    agentStore,
		RelationStore: relationStore,
		Executor:      executor,
	})

	req := httptest.NewRequest(http.MethodGet, "/teams/nonexistent/members/agent-1/context", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent", "agentId": "agent-1"})
	w := httptest.NewRecorder()

	handlers.GetMemberContext(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetMemberContext_MemberNotFound(t *testing.T) {
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()

	ctx := context.Background()
	team := newIndependentTestTeam("team-1", "Team")
	if err := teamStore.Create(ctx, team); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := agentStore.Create(ctx, &store.Agent{ID: "agent-1", DisplayName: "Agent"}); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	// Note: not adding agent-1 as a member

	executor := newTestExecutor(t, teamStore, agentStore, nil, "", nil, nil)
	handlers := NewHandlers(HandlersDeps{
		TeamStore:     teamStore,
		AgentStore:    agentStore,
		RelationStore: relationStore,
		Executor:      executor,
	})

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/members/agent-1/context", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w := httptest.NewRecorder()

	handlers.GetMemberContext(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetTeamExecutionStatus_Success(t *testing.T) {
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()

	ctx := context.Background()
	team := newIndependentTestTeam("team-1", "Team")
	if err := teamStore.Create(ctx, team); err != nil {
		t.Fatalf("create team: %v", err)
	}

	exec := &captureExecutor{}
	teamExecStore := NewTeamExecutionStore(teamStore, exec, roots.Config, nil)
	handlers := NewHandlers(HandlersDeps{
		TeamStore:     teamStore,
		AgentStore:    agentStore,
		RelationStore: relationStore,
		TeamExecStore: teamExecStore,
	})

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/execution-status", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.GetTeamExecutionStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp TeamExecutionStatus
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.State != "idle" {
		t.Errorf("expected state 'idle', got %q", resp.State)
	}
	if resp.TeamID != "team-1" {
		t.Errorf("expected teamId 'team-1', got %q", resp.TeamID)
	}
}

func TestGetTeamExecutionStatus_TeamNotFound(t *testing.T) {
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()

	handlers := NewHandlers(HandlersDeps{
		TeamStore:     teamStore,
		AgentStore:    agentStore,
		RelationStore: relationStore,
	})

	req := httptest.NewRequest(http.MethodGet, "/teams/nonexistent/execution-status", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	w := httptest.NewRecorder()

	handlers.GetTeamExecutionStatus(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}
