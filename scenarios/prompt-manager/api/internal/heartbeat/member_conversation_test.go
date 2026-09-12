package heartbeat

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"prompt-manager/internal/paths"
	"prompt-manager/internal/store"
)

func conversationFixture(t *testing.T) (*Handlers, *mockAgentClient, func(string) *httptest.ResponseRecorder) {
	t.Helper()
	fs := newFileStore(t, paths.RootsForTest(t))
	teams, agents := fs.Teams().(*store.FileTeamStore), fs.Agents().(*store.FileAgentStore)
	ctx := context.Background()
	if err := teams.Create(ctx, newIndependentTestTeam("team-1", "Team")); err != nil {
		t.Fatal(err)
	}
	if err := agents.Create(ctx, &store.Agent{ID: "agent-1", DisplayName: "Ada"}); err != nil {
		t.Fatal(err)
	}
	if err := fs.Relations().SetTeamMember(ctx, &store.TeamMemberRelation{TeamID: "team-1", AgentID: "agent-1", Status: store.MemberStatusActive}); err != nil {
		t.Fatal(err)
	}
	if err := teams.SetResponsibilities(ctx, "team-1", "agent-1", "CARE_FOR_REFERENCE_GARDEN"); err != nil {
		t.Fatal(err)
	}
	if err := teams.SetHeartbeatInstructions(ctx, "team-1", "agent-1", "AUTOMATIC_TASK_MUST_NOT_RUN"); err != nil {
		t.Fatal(err)
	}
	if err := teams.SetHeartbeatConfig(ctx, "team-1", "agent-1", &store.HeartbeatConfig{ProfileKey: "member.test.profile"}); err != nil {
		t.Fatal(err)
	}
	client := newMockAgentClient().WithCreateRunResponse(&Run{ID: "run-1", Status: "running"})
	h := NewHandlers(HandlersDeps{TeamStore: teams, AgentStore: agents, RelationStore: fs.Relations(), Executor: newTestExecutor(t, teams, agents, nil, "", nil, nil), AgentClient: client})
	return h, client, func(body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.CreateRun(w, httptest.NewRequest(http.MethodPost, "/runs", bytes.NewBufferString(body)))
		return w
	}
}

const conversationBody = `{"conversation":{"team_id":"team-1","agent_id":"agent-1","message":"How is the garden?","request_id":"7acf8d90-023b-4b1f-b39d-4c4b27756b7d"}}`

func TestMemberConversationContextAndIdentity(t *testing.T) {
	_, client, send := conversationFixture(t)
	w := send(conversationBody)
	if w.Code != 200 {
		t.Fatalf("%d: %s", w.Code, w.Body.String())
	}
	if len(client.createTaskCalls) != 1 || len(client.createRunCalls) != 1 {
		t.Fatal("expected one task and run")
	}
	task, run := client.createTaskCalls[0], client.createRunCalls[0]
	for _, want := range []string{"CARE_FOR_REFERENCE_GARDEN", "How is the garden?", "Ada"} {
		if !strings.Contains(task.Description, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(task.Description, "AUTOMATIC_TASK_MUST_NOT_RUN") {
		t.Fatal("conversation executes heartbeat")
	}
	if run.TaskID != task.ID || run.ProfileRef.ProfileKey != "member.test.profile" || run.IdempotencyKey == "" {
		t.Fatalf("wrong run binding: %+v", run)
	}
	encoded := run.Environment[attributionEnvVar]
	b, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatal(err)
	}
	var info store.AttributionInfo
	if err := json.Unmarshal(b, &info); err != nil {
		t.Fatal(err)
	}
	if info.SpawnOrigin != store.SpawnOriginConversation || info.TeamID == nil || *info.TeamID != "team-1" || info.MemberID == nil || *info.MemberID != "agent-1" {
		t.Fatalf("wrong attribution: %+v", info)
	}
	if err := validateAttribution(info, "team-1"); err != nil {
		t.Fatal(err)
	}
}

func TestMemberConversationRetryAndConflict(t *testing.T) {
	_, client, send := conversationFixture(t)
	if w := send(conversationBody); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	task, run := client.createTaskCalls[0], client.createRunCalls[0]
	client.getTasks = map[string]*Task{task.ID: task}
	client.listRunsResp = &ListRunsResponse{Runs: []*Run{{ID: "run-1", TaskID: task.ID, Tag: *run.Tag}}}
	if w := send(conversationBody); w.Code != 200 || !strings.Contains(w.Body.String(), "run-1") {
		t.Fatal(w.Body.String())
	}
	if len(client.createRunCalls) != 1 || len(client.createTaskCalls) != 1 {
		t.Fatal("retry duplicated work")
	}
	if w := send(strings.Replace(conversationBody, "How is the garden?", "Different message", 1)); w.Code != 409 {
		t.Fatalf("expected conflict: %d %s", w.Code, w.Body.String())
	}
}

func TestMemberConversationUnassignedPersona(t *testing.T) {
	_, client, send := conversationFixture(t)
	w := send(strings.Replace(conversationBody, `"team_id":"team-1",`, "", 1))
	if w.Code != http.StatusOK {
		t.Fatalf("%d: %s", w.Code, w.Body.String())
	}
	if len(client.createRunCalls) != 1 {
		t.Fatal("expected one conversation")
	}
	if len(client.createRunCalls[0].Environment) != 0 {
		t.Fatal("unassigned persona must not inherit team authority")
	}
	if strings.Contains(client.createTaskCalls[0].Description, "CARE_FOR_REFERENCE_GARDEN") {
		t.Fatal("unassigned conversation inherited team responsibilities")
	}
}

func TestMemberConversationValidationAndOutage(t *testing.T) {
	for name, body := range map[string]string{
		"missing member":         strings.Replace(conversationBody, "agent-1", "missing", 1),
		"wrong team":             strings.Replace(conversationBody, "team-1", "missing", 1),
		"empty message":          strings.Replace(conversationBody, "How is the garden?", " ", 1),
		"invalid retry identity": strings.Replace(conversationBody, "7acf8d90-023b-4b1f-b39d-4c4b27756b7d", "invalid", 1),
		"profile injection":      strings.Replace(conversationBody, `{"conversation":`, `{"profile_ref":{"profile_key":"other"},"conversation":`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			_, client, send := conversationFixture(t)
			w := send(body)
			if w.Code < 400 || len(client.createTaskCalls) > 0 || len(client.createRunCalls) > 0 {
				t.Fatalf("accepted invalid request: %d", w.Code)
			}
		})
	}
	_, client, send := conversationFixture(t)
	client.getTaskErr = fmt.Errorf("upstream unavailable")
	if w := send(conversationBody); w.Code != 502 || len(client.createTaskCalls) > 0 {
		t.Fatal("outage treated as missing task")
	}
}
