package heartbeat

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"prompt-manager/internal/paths"
	"prompt-manager/internal/sourceledger"
	"prompt-manager/internal/store"

	facetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/facets"
	scopesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/scopes"
	"google.golang.org/protobuf/proto"
)

func TestExecutorBuildSupervisionPromptUsesCompactLane(t *testing.T) {
	ctx := context.Background()
	fileStore := newFileStore(t, paths.RootsForTest(t))
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	if err := teamStore.Create(ctx, newIndependentTestTeam("supervisors", "Standing supervisors")); err != nil {
		t.Fatal(err)
	}
	executor := newTestExecutor(t, teamStore, fileStore.Agents().(*store.FileAgentStore), newMockAgentClient(), "", nil, nil)
	prompt, err := executor.BuildSupervisionPrompt(ctx, "supervisors", "leader")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(prompt, "Source Ledger") || strings.Contains(prompt, "Team Context Wake") || len(prompt) > 5000 {
		t.Fatalf("supervisor prompt was not compact: %d bytes\n%s", len(prompt), prompt)
	}
	if !strings.Contains(prompt, "compact Agent Manager owner cut") || !strings.Contains(prompt, "owner-qualified mandate") {
		t.Fatalf("supervisor guardrails missing: %s", prompt)
	}
}

func TestExecutorStandingSupervisorEnsuresSelectedTeamCorpus(t *testing.T) {
	for _, unavailable := range []bool{false, true} {
		name := "provisions-and-reuses"
		if unavailable {
			name = "owner-unavailable-blocks-launch"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			f := newSupervisionFixture(t)
			f.owner.rows = []EffortObservation{effort("new-effort")}
			f.tick(t)
			fileStore := newFileStore(t, paths.RootsForTest(t))
			teamStore := fileStore.Teams().(*store.FileTeamStore)
			if err := teamStore.Create(ctx, newIndependentTestTeam("supervisors", "New team")); err != nil {
				t.Fatal(err)
			}
			if err := teamStore.SetHeartbeatConfig(ctx, "supervisors", "leader", f.cfg); err != nil {
				t.Fatal(err)
			}

			var registered *scopesv1.Scope
			var ledgerMu sync.Mutex
			creates, checks := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ledgerMu.Lock()
				defer ledgerMu.Unlock()
				w.Header().Set("Content-Type", "application/proto")
				switch {
				case strings.HasSuffix(r.URL.Path, "/ListScopes"):
					checks++
					if unavailable {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusServiceUnavailable)
						_, _ = w.Write([]byte(`{"code":"unavailable","message":"qualification owner unavailable"}`))
						return
					}
					response := &scopesv1.ListScopesResponse{}
					if registered != nil {
						response.Scopes = []*scopesv1.Scope{registered}
					}
					data, _ := proto.Marshal(response)
					_, _ = w.Write(data)
				case strings.HasSuffix(r.URL.Path, "/CreateScope"):
					body, err := io.ReadAll(r.Body)
					if err != nil {
						t.Error(err)
					}
					var request scopesv1.CreateScopeRequest
					if err := proto.Unmarshal(body, &request); err != nil {
						t.Error(err)
					}
					registered = request.GetScope()
					creates++
					data, _ := proto.Marshal(&scopesv1.CreateScopeResponse{Scope: registered})
					_, _ = w.Write(data)
				case strings.HasSuffix(r.URL.Path, "/ListFacets"):
					body, err := io.ReadAll(r.Body)
					if err != nil {
						t.Error(err)
					}
					var request facetsv1.ListFacetsRequest
					if err := proto.Unmarshal(body, &request); err != nil || request.Scope != "team:supervisors" {
						t.Errorf("facet scope = %q, error = %v", request.Scope, err)
					}
					response := &facetsv1.ListFacetsResponse{}
					for _, facet := range sourceledger.TeamScopeFacets("supervisors") {
						response.Facets = append(response.Facets, &facetsv1.Facet{Id: facet.GetId(), Label: facet.GetLabel(), Guidance: facet.GetGuidance(), RetentionPolicy: facet.GetRetentionPolicy(), CompactionEligible: facet.GetCompactionEligible(), ResidentBudget: facet.GetResidentBudget()})
					}
					data, _ := proto.Marshal(response)
					_, _ = w.Write(data)
				default:
					t.Errorf("unexpected ledger operation: %s", r.URL.Path)
					w.WriteHeader(http.StatusBadRequest)
				}
			}))
			defer server.Close()
			teamStore.SetSourceLedger(sourceledger.NewAt(server.URL))
			executor := newTestExecutor(t, teamStore, fileStore.Agents().(*store.FileAgentStore), f.agent, "", nil, nil)
			executor.EffortSupervisor = f.s
			f.s.Prompt = func(context.Context, string, string) (string, error) {
				ledgerMu.Lock()
				defer ledgerMu.Unlock()
				f.prompts++
				if registered == nil {
					t.Error("standing prompt built before team corpus registration")
				}
				return "qualified team context", nil
			}

			_, err := executor.Execute(ctx, "supervisors", "leader", "")
			if unavailable {
				ledgerMu.Lock()
				defer ledgerMu.Unlock()
				var ownerErr *sourceledger.UnavailableError
				if !errors.As(err, &ownerErr) {
					t.Fatalf("expected typed source-ledger unavailability, got %v", err)
				}
				if checks != 1 || creates != 0 || f.prompts != 0 || len(f.agent.createTaskCalls) != 0 || len(f.agent.createRunCalls) != 0 {
					t.Fatalf("failed provisioning performed work: checks=%d creates=%d prompts=%d tasks=%d runs=%d", checks, creates, f.prompts, len(f.agent.createTaskCalls), len(f.agent.createRunCalls))
				}
				state, err := f.s.State.Load("supervisors", "leader")
				if err != nil || state.WakesInWindow != 0 || state.Pending == nil || state.Pending.DispatchStarted {
					t.Fatalf("failed provisioning charged or lost retryable wake: state=%+v err=%v", state, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := executor.Execute(ctx, "supervisors", "leader", ""); err != nil {
				t.Fatal(err)
			}
			ledgerMu.Lock()
			defer ledgerMu.Unlock()
			if registered == nil || registered.GetId() != "team:supervisors" || registered.GetFrontierTarget() != 16 || registered.GetWakeBudget() != 128 || len(registered.GetFacets()) != 6 {
				t.Fatalf("selected team scope not provisioned with owner defaults: %+v", registered)
			}
			if checks != 2 || creates != 1 || f.prompts != 1 || len(f.agent.createRunCalls) != 1 {
				t.Fatalf("repeated dispatch did not reuse scope/run: checks=%d creates=%d prompts=%d runs=%d", checks, creates, f.prompts, len(f.agent.createRunCalls))
			}
		})
	}
}

func TestExecutorExecuteFailsWhenConfigMissing(t *testing.T) {
	ctx := context.Background()
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	teamStore := fileStore.Teams().(*store.FileTeamStore)

	if err := agentStore.Create(ctx, &store.Agent{ID: "agent-1", DisplayName: "Agent"}); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := teamStore.Create(ctx, newIndependentTestTeam("team-1", "Team")); err != nil {
		t.Fatalf("create team: %v", err)
	}

	executor := newTestExecutor(t, teamStore, agentStore, nil, "", nil, nil)
	result, err := executor.Execute(ctx, "team-1", "agent-1", "profile-key")

	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if result == nil || result.Status != store.HeartbeatStatusFailed {
		t.Fatalf("expected failed status, got %+v", result)
	}
}

// TestCreateTaskRequestMatchesProtoSchema verifies the JSON produced by
// CreateTaskRequest uses field names compatible with the agent-manager proto
// schema. The agent-manager uses protojson with DiscardUnknown=false, so any
// unrecognised field (e.g. "prompt", "workingDir") causes an unmarshal error.
//
// Regression test for: "creating task: agent-manager error: validation error
// on body: invalid JSON request body"
func TestCreateTaskRequestMatchesProtoSchema(t *testing.T) {
	task := &Task{
		Title:       "Heartbeat: team-1/agent-1",
		Description: "Do the thing",
		ScopePath:   "/home/user/project",
		ProjectRoot: "/home/user/project",
	}
	req := CreateTaskRequest{Task: task}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Parse back to a generic map to inspect field names.
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	taskRaw, ok := envelope["task"]
	if !ok {
		t.Fatal("expected top-level 'task' field in request JSON")
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(taskRaw, &fields); err != nil {
		t.Fatalf("unmarshal task: %v", err)
	}

	// Allowed proto field names (snake_case as protojson expects).
	allowed := map[string]bool{
		"id": true, "title": true, "description": true,
		"scope_path": true, "project_root": true,
		"phase_prompt_ids": true, "context_attachments": true,
		"status": true, "created_by": true,
		"created_at": true, "updated_at": true,
	}

	for key := range fields {
		if !allowed[key] {
			t.Errorf("unexpected field %q in task JSON — agent-manager proto will reject this (DiscardUnknown=false)", key)
		}
	}

	// Required proto fields must be present and non-empty.
	for _, required := range []string{"title", "description", "scope_path"} {
		raw, ok := fields[required]
		if !ok {
			t.Errorf("required proto field %q missing from task JSON", required)
			continue
		}
		var val string
		if err := json.Unmarshal(raw, &val); err != nil {
			t.Errorf("field %q is not a string: %v", required, err)
		} else if val == "" {
			t.Errorf("required proto field %q is empty", required)
		}
	}
}
