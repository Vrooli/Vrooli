package backlog

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/testutil"
	"swarm-manager/internal/transitions"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

type fakeWorkshopWorkflow struct {
	start      agentmanager.WorkflowStart
	completion workshopCompletionFixture
	err        error
	starts     int
	collects   int
	invocation agentmanager.Invocation
	key        string
}

type workshopCompletionFixture struct {
	ExecutionID      string
	DefinitionDigest string
	RunID            string
	ProfileIdentity  string
	EntityKind       string
	EntityName       string
	EntityVersion    string
	Result           json.RawMessage
}

func (f *fakeWorkshopWorkflow) StartWorkflow(_ context.Context, invocation agentmanager.Invocation) (agentmanager.WorkflowStart, error) {
	f.starts++
	f.invocation, f.key = invocation, invocation.IdempotencyKey
	return f.start, f.err
}

func (f *fakeWorkshopWorkflow) CollectWorkflow(_ context.Context, _ string) (agentmanager.InvocationCompletion, error) {
	f.collects++
	input, _ := structpb.NewValue(map[string]any{"entity": map[string]any{"kind": f.completion.EntityKind, "name": f.completion.EntityName, "version": f.completion.EntityVersion}})
	var result any
	_ = json.Unmarshal(f.completion.Result, &result)
	output, _ := structpb.NewValue(map[string]any{"result": result})
	return agentmanager.InvocationCompletion{
		ExecutionID: f.completion.ExecutionID, DefinitionDigest: f.completion.DefinitionDigest,
		Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED, Input: input, Output: output,
		Attempts: []*domainpb.WorkflowNodeAttempt{{NodeId: "workshop", RunId: f.completion.RunID, ProfileIdentity: f.completion.ProfileIdentity}},
	}, f.err
}

func TestResearchWorkshopUsesTypedWorkflowAdapter(t *testing.T) {
	root := t.TempDir()
	h := NewHandler(root, t.TempDir())
	installTransitionRegistry(t, h)
	executionID := uuid.NewString()
	fake := &fakeWorkshopWorkflow{start: agentmanager.WorkflowStart{ExecutionID: executionID, RunID: uuid.NewString(), DefinitionDigest: "sha256:def"}}
	h.SetWorkshopWorkflow(fake)
	item := BacklogItem{Name: "typed-workshop", Title: "Typed workshop", Description: "Decide the boundary", Status: StatusBacklog, Priority: 3, Tags: []string{"pilot"}, Created: "2026-07-16T00:00:00Z", Updated: "2026-07-16T00:00:00Z"}
	createTestItem(t, root, KindIdea, item)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/idea/typed-workshop/research", bytes.NewBufferString(`{"mode":"workshop","prompt":"focus on ownership"}`))
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": item.Name})
	w := httptest.NewRecorder()
	h.Research(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	entity := fake.invocation.Input.GetStructValue().GetFields()["entity"].GetStructValue().GetFields()
	if fake.starts != 1 || fake.invocation.WorkflowKey != "swarm-manager/backlog-workshop-round" || entity["kind"].GetStringValue() != "idea" || entity["name"].GetStringValue() != item.Name || fake.invocation.Input.GetStructValue().GetFields()["operatorNote"].GetStringValue() == "" || entity["version"].GetStringValue() == "" {
		t.Fatalf("workflow start = %#v", fake)
	}
	if fake.key == "" || !bytes.Contains(w.Body.Bytes(), []byte(executionID)) {
		t.Fatalf("missing stable correlation: key=%q body=%s", fake.key, w.Body.String())
	}
}

func TestWorkshopStartUsesDeclaredTransitionWorkflowLocator(t *testing.T) {
	root := t.TempDir()
	h := NewHandler(root, t.TempDir())
	registry, err := transitions.LoadFS(fstest.MapFS{"registry.json": &fstest.MapFile{Data: []byte(`{
  "schemaVersion":"swarm-transition/v1", "key":"backlog.refine", "subject":"backlog-item", "kind":"workflow",
  "workflow":{"owner":"declared-owner", "key":"declared/workshop"},
  "inputContract":"backlog-refinement-input/v1", "terminalOutcomes":["ready"], "applyAction":"apply_backlog_refinement"
}`)}}, ".")
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	h.SetTransitionRegistry(registry)
	fake := &fakeWorkshopWorkflow{start: agentmanager.WorkflowStart{ExecutionID: uuid.NewString(), DefinitionDigest: "sha256:def"}}
	h.SetWorkshopWorkflow(fake)
	item := BacklogItem{Name: "declared-workshop", Title: "Declared", Status: StatusBacklog, Priority: 3, Created: "2026-07-16T00:00:00Z", Updated: "2026-07-16T00:00:00Z"}
	createTestItem(t, root, KindIdea, item)

	if _, err := h.startWorkshopRoundWorkflow(context.Background(), itemWithKind(item, KindIdea), ""); err != nil {
		t.Fatalf("start workflow: %v", err)
	}
	if fake.invocation.Owner != "declared-owner" || fake.invocation.WorkflowKey != "declared/workshop" {
		t.Fatalf("invocation locator = %s/%s, want declared-owner/declared/workshop", fake.invocation.Owner, fake.invocation.WorkflowKey)
	}
}

func TestWorkshopWorkflowResultReconcilesAutomaticallyAcrossRestart(t *testing.T) {
	root := t.TempDir()
	item := BacklogItem{Name: "auto-apply", Title: "Auto apply", Description: "Reconcile me", Status: StatusBacklog, Priority: 3, Tags: []string{}, Created: "2026-07-16T00:00:00Z", Updated: "2026-07-16T00:00:00Z"}
	createTestItem(t, root, KindIdea, item)
	executionID := uuid.NewString()
	starter := NewHandler(root, t.TempDir())
	installTransitionRegistry(t, starter)
	starter.SetWorkshopWorkflow(&fakeWorkshopWorkflow{start: agentmanager.WorkflowStart{ExecutionID: executionID, DefinitionDigest: "sha256:def"}})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/idea/auto-apply/research", bytes.NewBufferString(`{"mode":"workshop"}`))
	request = mux.SetURLVars(request, map[string]string{"kind": "idea", "name": item.Name})
	response := httptest.NewRecorder()
	starter.Research(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("start status = %d: %s", response.Code, response.Body.String())
	}

	item = itemWithKind(item, KindIdea)
	restarted := NewHandler(root, t.TempDir())
	installTransitionRegistry(t, restarted)
	restarted.SetWorkshopWorkflow(&fakeWorkshopWorkflow{completion: workshopCompletionFixture{
		ExecutionID: executionID, DefinitionDigest: "sha256:def", EntityKind: "idea", EntityName: item.Name,
		EntityVersion: workshopSnapshotVersion(item, 0),
		Result:        json.RawMessage(`{"outcome":"no_questions","note":"automatically applied","readiness":{"problem_clarity":3,"scope_defined":3,"approach_solid":3,"testable":3,"risk_awareness":3}}`),
	}})
	if err := restarted.ProcessWorkshopWorkflows(context.Background()); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	workshopDir := filepath.Join(root, "ideas", item.Name, "workshop")
	testutil.AssertFileExists(t, filepath.Join(workshopDir, "round-001.json"))
	testutil.AssertFileNotExists(t, workshopWorkflowPendingPath(workshopDir, executionID))

	// A second boot has no pending correlation and cannot duplicate the round.
	if err := restarted.ProcessWorkshopWorkflows(context.Background()); err != nil {
		t.Fatalf("reconcile replay: %v", err)
	}
}

func TestApplyWorkshopWorkflowResultExactlyOnceAcrossRestart(t *testing.T) {
	root := t.TempDir()
	item := BacklogItem{Name: "apply-once", Title: "Apply once", Status: StatusBacklog, Priority: 3, Tags: []string{}, Created: "2026-07-16T00:00:00Z", Updated: "2026-07-16T00:00:00Z"}
	createTestItem(t, root, KindIdea, item)
	version := workshopSnapshotVersion(itemWithKind(item, KindIdea), 0)
	executionID := uuid.NewString()
	result := json.RawMessage(`{"outcome":"proposals","note":"Prefer the narrow seam","decisionQuestions":[{"id":"ownership","topic":"Who mutates?","context":"Keep the boundary explicit","options":[{"key":"A","label":"Swarm","rationale":"Domain owner","recommended":true},{"key":"B","label":"Agent Manager","rationale":"Rejected coupling","recommended":false}]}],"readiness":{"problem_clarity":3,"scope_defined":3,"approach_solid":2,"testable":2,"risk_awareness":2}}`)
	fake := &fakeWorkshopWorkflow{completion: workshopCompletionFixture{ExecutionID: executionID, DefinitionDigest: "sha256:def", RunID: uuid.NewString(), ProfileIdentity: "profile:swarm-manager/deep-work", EntityKind: "idea", EntityName: item.Name, EntityVersion: version, Result: result}}
	h := NewHandler(root, t.TempDir())
	h.SetWorkshopWorkflow(fake)
	first := applyWorkflowRequest(t, h, item.Name, executionID)
	if first.Code != http.StatusOK || !bytes.Contains(first.Body.Bytes(), []byte(`"applied":true`)) {
		t.Fatalf("first apply = %d: %s", first.Code, first.Body.String())
	}
	testutil.AssertFileExists(t, filepath.Join(root, "ideas", item.Name, "workshop", "round-001.json"))
	testutil.AssertFileExists(t, filepath.Join(root, "ideas", item.Name, "workshop", "workflow-provenance-"+executionID+".json"))

	// A new handler simulates an API restart. Durable provenance short-circuits
	// before the external result seam, so the round cannot be duplicated.
	restartedFake := &fakeWorkshopWorkflow{}
	restarted := NewHandler(root, t.TempDir())
	restarted.SetWorkshopWorkflow(restartedFake)
	second := applyWorkflowRequest(t, restarted, item.Name, executionID)
	if second.Code != http.StatusOK || !bytes.Contains(second.Body.Bytes(), []byte(`"idempotent":true`)) || restartedFake.collects != 0 {
		t.Fatalf("replay = %d collects=%d: %s", second.Code, restartedFake.collects, second.Body.String())
	}
	entries, _ := os.ReadDir(filepath.Join(root, "ideas", item.Name, "workshop"))
	rounds := 0
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".json" && len(entry.Name()) >= 6 && entry.Name()[:6] == "round-" {
			rounds++
		}
	}
	if rounds != 1 {
		t.Fatalf("round count = %d", rounds)
	}
}

func TestApplyWorkshopWorkflowResultRejectsStaleSnapshot(t *testing.T) {
	root := t.TempDir()
	item := BacklogItem{Name: "stale", Title: "Stale", Status: StatusBacklog, Priority: 3, Tags: []string{}, Created: "2026-07-16T00:00:00Z", Updated: "2026-07-16T00:00:00Z"}
	createTestItem(t, root, KindIdea, item)
	executionID := uuid.NewString()
	fake := &fakeWorkshopWorkflow{completion: workshopCompletionFixture{ExecutionID: executionID, EntityKind: "idea", EntityName: item.Name, EntityVersion: "sha256:stale", Result: json.RawMessage(`{"outcome":"no_questions","note":"ready","readiness":{"problem_clarity":3,"scope_defined":3,"approach_solid":3,"testable":3,"risk_awareness":3}}`)}}
	h := NewHandler(root, t.TempDir())
	h.SetWorkshopWorkflow(fake)
	response := applyWorkflowRequest(t, h, item.Name, executionID)
	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d: %s", response.Code, response.Body.String())
	}
}

func TestApplyWorkshopWorkflowNoQuestionsCreatesInformationalRound(t *testing.T) {
	root := t.TempDir()
	item := itemWithKind(BacklogItem{Name: "no-questions", Title: "No questions", Status: StatusBacklog, Priority: 3, Tags: []string{}, Created: "2026-07-16T00:00:00Z", Updated: "2026-07-16T00:00:00Z"}, KindIdea)
	createTestItem(t, root, KindIdea, item)
	executionID := uuid.NewString()
	fake := &fakeWorkshopWorkflow{completion: workshopCompletionFixture{ExecutionID: executionID, EntityKind: "idea", EntityName: item.Name, EntityVersion: workshopSnapshotVersion(item, 0), Result: json.RawMessage(`{"outcome":"no_questions","note":"The boundary is already explicit","readiness":{"problem_clarity":3,"scope_defined":3,"approach_solid":3,"testable":3,"risk_awareness":3}}`)}}
	h := NewHandler(root, t.TempDir())
	h.SetWorkshopWorkflow(fake)
	response := applyWorkflowRequest(t, h, item.Name, executionID)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"applied":true`)) {
		t.Fatalf("status = %d: %s", response.Code, response.Body.String())
	}
	data, err := os.ReadFile(filepath.Join(root, "ideas", item.Name, "workshop", "round-001.json"))
	if err != nil || !bytes.Contains(data, []byte("The boundary is already explicit")) {
		t.Fatalf("informational round = %q, err=%v", data, err)
	}
}

func TestApplyWorkshopWorkflowAbstentionRecordsProvenanceWithoutRound(t *testing.T) {
	root := t.TempDir()
	item := itemWithKind(BacklogItem{Name: "abstain", Title: "Abstain", Status: StatusBacklog, Priority: 3, Tags: []string{}, Created: "2026-07-16T00:00:00Z", Updated: "2026-07-16T00:00:00Z"}, KindIdea)
	createTestItem(t, root, KindIdea, item)
	executionID := uuid.NewString()
	fake := &fakeWorkshopWorkflow{completion: workshopCompletionFixture{ExecutionID: executionID, EntityKind: "idea", EntityName: item.Name, EntityVersion: workshopSnapshotVersion(item, 0), Result: json.RawMessage(`{"outcome":"abstained","reason":"insufficient evidence"}`)}}
	h := NewHandler(root, t.TempDir())
	h.SetWorkshopWorkflow(fake)
	response := applyWorkflowRequest(t, h, item.Name, executionID)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"applied":true`)) {
		t.Fatalf("status = %d: %s", response.Code, response.Body.String())
	}
	testutil.AssertFileNotExists(t, filepath.Join(root, "ideas", item.Name, "workshop", "round-001.json"))
	testutil.AssertFileExists(t, filepath.Join(root, "ideas", item.Name, "workshop", "workflow-provenance-"+executionID+".json"))
}

func applyWorkflowRequest(t *testing.T, h *Handler, name, executionID string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/idea/"+name+"/workshop/workflow/"+executionID+"/apply", nil)
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": name, "executionID": executionID})
	w := httptest.NewRecorder()
	h.ApplyWorkshopWorkflowResult(w, req)
	return w
}

func itemWithKind(item BacklogItem, kind BacklogKind) BacklogItem {
	item.Kind = kind
	return item
}
