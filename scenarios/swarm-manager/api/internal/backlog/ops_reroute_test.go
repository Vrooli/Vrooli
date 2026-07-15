package backlog

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/operatingmode"
	"swarm-manager/internal/opsbridge"
	"swarm-manager/internal/opscatalog"
	"swarm-manager/internal/opsrunner"
	"swarm-manager/internal/pathutil"
	"swarm-manager/internal/promptmanager"
	"swarm-manager/internal/testutil"
	"swarm-manager/internal/workshop"

	"github.com/gorilla/mux"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

// ---------------------------------------------------------------------------
// Test runner harness: the real production runner assembled exactly as
// registerBacklogOperationsRunner does — real catalog, real modes, real backlog
// ops handlers over the handler's own FileStore — with deterministic operating-
// mode seams so the live start returns a canned run id and completion is driven
// by feeding CommitResult, without spawning an agent.
// ---------------------------------------------------------------------------

// fakePhaseEngine is the live start seam: it returns a canned run association
// without spawning an agent.
type fakePhaseEngine struct {
	runID   string
	started bool
	err     error
	gotReq  operatingmode.StartTargetPhaseRequest
}

func (f *fakePhaseEngine) StartTargetPhase(_ context.Context, req operatingmode.StartTargetPhaseRequest) (operatingmode.RoundEnvelope, error) {
	f.started = true
	f.gotReq = req
	if f.err != nil {
		return operatingmode.RoundEnvelope{}, f.err
	}
	rid := f.runID
	if rid == "" {
		rid = "run-fixed"
	}
	return operatingmode.RoundEnvelope{RunID: rid, GeneratedAt: "2026-07-14T00:00:00Z"}, nil
}

type fakeSimEngine struct {
	byPreset map[string]operatingmode.SimulationResponse
}

func (f *fakeSimEngine) SimulateMode(_ context.Context, _ operatingmode.Mode, preset string) (operatingmode.SimulationResponse, error) {
	if f == nil || f.byPreset == nil {
		return operatingmode.SimulationResponse{}, nil
	}
	return f.byPreset[preset], nil
}

type noopRefresher struct{}

func (noopRefresher) RefreshRunByID(context.Context, string) (operatingmode.RoundEnvelope, bool, error) {
	return operatingmode.RoundEnvelope{}, false, nil
}

// setupTestHandlerWithRunner builds a handler over a temp store and injects the
// real production runner + scheduler wired to that store. The returned
// fakePhaseEngine lets a test assert the live start fired; the BacklogRunner
// lets a test drive completion via CommitResult / the observer.
func setupTestHandlerWithRunner(t *testing.T, runID string) (*Handler, string, *fakePhaseEngine, *opsbridge.BacklogRunner) {
	t.Helper()
	// Capture the REAL scenario source root for the catalog/modes before
	// disableAutoWorkshopSettings overrides SCENARIO_ROOT to the temp dir.
	scenarioRoot := pathutil.ResolveScenarioRoot("swarm-manager")
	rootDir := t.TempDir()
	for _, dir := range backlogKindDirs {
		testutil.MakeDir(t, filepath.Join(rootDir, dir))
	}
	disableAutoWorkshopSettings(t, rootDir)
	h := NewHandlerWithClients(rootDir, rootDir, &mockAgentService{}, &promptmanager.MockClient{Result: "test prompt"})
	scopeExecutionQueuerForTest(t, h, rootDir, &mockAgentService{})

	catalog, err := opscatalog.Load(scenarioRoot)
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	modeDefs, err := operatingmode.LoadModesFromDir(filepath.Join(scenarioRoot, "modes"))
	if err != nil {
		t.Fatalf("load modes: %v", err)
	}
	store := h.Store()
	locator := opsrunner.FSLocator{
		BacklogItemDir: func(kind, name string) (string, error) {
			return store.ItemDir(BacklogKind(kind), name), nil
		},
		InitiativeDir: func(name string) (string, error) { return filepath.Join(rootDir, "initiatives", name), nil },
		ScanRoots:     []string{rootDir},
	}
	registry := opsrunner.NewActionRegistry()
	RegisterOpsHandlers(registry, OpsHandlerDeps{Store: store})
	phase := &fakePhaseEngine{runID: runID}
	built, err := opsbridge.BuildBacklogRunner(opsbridge.BacklogRunnerConfig{
		Catalog:         catalog,
		ModeDefs:        modeDefs,
		PhaseEngine:     phase,
		SimEngine:       &fakeSimEngine{},
		Refresher:       noopRefresher{},
		Locator:         locator,
		Registry:        registry,
		AdvanceResolver: h,
		RequestedBy:     "reroute-test",
	})
	if err != nil {
		t.Fatalf("build runner: %v", err)
	}
	h.SetRunner(built.Runner, built.Scheduler)
	return h, rootDir, phase, built
}

// commitDelivery finalizes a running operation for a target with a delivered
// outcome + result, driving the policy transition exactly as the completion
// router would.
func commitDelivery(t *testing.T, built *opsbridge.BacklogRunner, kind BacklogKind, name, executionID, outcome string, result map[string]any) opsrunner.OperationResult {
	t.Helper()
	var raw json.RawMessage
	if result != nil {
		b, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("marshal delivered result: %v", err)
		}
		raw = b
	}
	res, err := built.Runner.CommitResult(context.Background(), opsrunner.CommitRequest{
		Target:          opsrunner.TargetRef{Kind: agentops.TargetBacklogItem, ID: string(kind) + "/" + name},
		ExecutionID:     executionID,
		Outcome:         outcome,
		DeliveredResult: raw,
		RequestedBy:     "reroute-test",
	})
	if err != nil {
		t.Fatalf("commit result: %v", err)
	}
	return res
}

func clarificationHandoff(answer string) map[string]any {
	return map[string]any{
		"handoff":  map[string]any{"summary": answer},
		"progress": "complete",
	}
}

// ---------------------------------------------------------------------------
// Research entrypoint: starts an operation, or fails closed when no runner.
// ---------------------------------------------------------------------------

func TestResearch_StartsOperationThroughRunner(t *testing.T) {
	h, rootDir, phase, _ := setupTestHandlerWithRunner(t, "run-research-1")
	createTestItem(t, rootDir, KindIdea, BacklogItem{Name: "idea-r", Title: "Idea", Status: StatusBacklog, Priority: 3})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/idea/idea-r/research", bytes.NewBufferString(`{"mode":"workshop"}`))
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "idea-r"})
	w := httptest.NewRecorder()
	h.Research(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	if !phase.started {
		t.Fatal("expected the live phase engine to start the operation")
	}
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["runId"] != "run-research-1" && body["run_id"] != "run-research-1" {
		t.Fatalf("expected run id in response, got %v", body)
	}
}

// TestResearch_ForwardsTypedCallerContext proves the operator's research prompt
// and attached context reach the engine as typed structured operator inputs (the
// research-refine mode's caller-context providers read them), without any mode
// caller-input — the finding-93b0286c fix. The repeated context fields render
// one-per-line.
func TestResearch_ForwardsTypedCallerContext(t *testing.T) {
	h, rootDir, phase, _ := setupTestHandlerWithRunner(t, "run-research-ctx")
	createTestItem(t, rootDir, KindIdea, BacklogItem{Name: "idea-ctx", Title: "Idea", Status: StatusBacklog, Priority: 3})

	body := `{"mode":"workshop","prompt":"focus on the auth boundary","context_paths":["api/a.go","api/b.go"],"context_target_ids":["scenario-x"],"context_requirement_ids":["REQ-1"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/idea/idea-ctx/research", bytes.NewBufferString(body))
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "idea-ctx"})
	w := httptest.NewRecorder()
	h.Research(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	oi := phase.gotReq.OperatorInputs
	if oi["USER_PROMPT"] != "focus on the auth boundary" {
		t.Fatalf("USER_PROMPT = %q, want the operator prompt; inputs=%+v", oi["USER_PROMPT"], oi)
	}
	if oi["CONTEXT_PATHS"] != "api/a.go\napi/b.go" {
		t.Fatalf("CONTEXT_PATHS = %q, want one path per line", oi["CONTEXT_PATHS"])
	}
	if oi["CONTEXT_TARGETS"] != "scenario-x" {
		t.Fatalf("CONTEXT_TARGETS = %q", oi["CONTEXT_TARGETS"])
	}
	if oi["CONTEXT_REQUIREMENTS"] != "REQ-1" {
		t.Fatalf("CONTEXT_REQUIREMENTS = %q", oi["CONTEXT_REQUIREMENTS"])
	}
	// The operator note channel stays empty: research context is NOT collapsed into it.
	if phase.gotReq.Note != "" {
		t.Fatalf("operator note = %q, want empty for research context", phase.gotReq.Note)
	}
}

// TestResearch_NoContextForwardsNoInputs proves a research request with no operator
// context still starts (an empty caller-input set the mode accepts) — the regression
// the empty-set engine invariant guards.
func TestResearch_NoContextForwardsNoInputs(t *testing.T) {
	h, rootDir, phase, _ := setupTestHandlerWithRunner(t, "run-research-empty")
	createTestItem(t, rootDir, KindIdea, BacklogItem{Name: "idea-empty", Title: "Idea", Status: StatusBacklog, Priority: 3})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/idea/idea-empty/research", bytes.NewBufferString(`{"mode":"workshop"}`))
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "idea-empty"})
	w := httptest.NewRecorder()
	h.Research(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	if len(phase.gotReq.OperatorInputs) != 0 {
		t.Fatalf("expected no structured operator inputs, got %+v", phase.gotReq.OperatorInputs)
	}
}

// TestResearchRefineCallerInputs_BuildsAndOmits proves the builder includes only
// non-empty operator context and omits absent fields (a nil map for an empty
// request), so the pinned snapshot carries exactly what the operator supplied.
func TestResearchRefineCallerInputs_BuildsAndOmits(t *testing.T) {
	if got := researchRefineCallerInputs(&apipb.BacklogResearchRequest{}); got != nil {
		t.Fatalf("empty request should yield nil caller inputs, got %+v", got)
	}
	prompt := "do the thing"
	got := researchRefineCallerInputs(&apipb.BacklogResearchRequest{
		Prompt:       &prompt,
		ContextPaths: []string{"x.go", "y.go"},
	})
	if got["USER_PROMPT"] != "do the thing" {
		t.Fatalf("USER_PROMPT = %v", got["USER_PROMPT"])
	}
	if got["CONTEXT_PATHS"] != "x.go\ny.go" {
		t.Fatalf("CONTEXT_PATHS = %v", got["CONTEXT_PATHS"])
	}
	if _, ok := got["CONTEXT_TARGETS"]; ok {
		t.Fatalf("absent context targets must be omitted, got %+v", got)
	}
	if _, ok := got["CONTEXT_REQUIREMENTS"]; ok {
		t.Fatalf("absent context requirements must be omitted, got %+v", got)
	}
}

func TestResearch_UnavailableWithoutRunner(t *testing.T) {
	h, rootDir := setupTestHandlerWithAgent(t, &mockAgentService{})
	createTestItem(t, rootDir, KindIdea, BacklogItem{Name: "idea-u", Title: "Idea", Status: StatusBacklog, Priority: 3})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/idea/idea-u/research", bytes.NewBufferString(`{"mode":"workshop"}`))
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "idea-u"})
	w := httptest.NewRecorder()
	h.Research(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 when no runner is wired; body=%s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Deferred auto-advance: scheduler intent replaces the ticker.
// ---------------------------------------------------------------------------

func TestDeferredAdvance_SchedulesCancelsAndDedups(t *testing.T) {
	h, rootDir, _, built := setupTestHandlerWithRunner(t, "run-x")
	createTestItem(t, rootDir, KindExecute, BacklogItem{Name: "adv", Title: "Adv", Status: StatusBacklog, Priority: 3})

	// Schedule a deferred advance intent.
	if err := h.scheduleDeferredAdvanceIntent(KindExecute, "adv", agentops.OpWorkshopRound, "2030-01-01T00:00:00Z"); err != nil {
		t.Fatalf("schedule: %v", err)
	}
	if !h.hasScheduledAdvance(KindExecute, "adv") {
		t.Fatal("expected a scheduled advance intent after scheduling")
	}

	// Scheduling again REPLACES (cancel-then-schedule): still exactly one intent.
	if err := h.scheduleDeferredAdvanceIntent(KindExecute, "adv", agentops.OpWorkshopRound, "2031-01-01T00:00:00Z"); err != nil {
		t.Fatalf("reschedule: %v", err)
	}
	w, found, err := built.Repo.Load(agentops.TargetBacklogItem, "execute/adv")
	if err != nil || !found {
		t.Fatalf("load workflow: err=%v found=%v", err, found)
	}
	count := 0
	for _, tm := range w.Timers {
		if tm.Intent == advanceIntentName {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one advance intent after replace, got %d", count)
	}

	// Cancel removes it.
	if err := h.cancelDeferredAdvanceIntent(KindExecute, "adv"); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if h.hasScheduledAdvance(KindExecute, "adv") {
		t.Fatal("expected no scheduled advance intent after cancel")
	}
}

func TestDeferredAdvance_SurvivesRestart(t *testing.T) {
	h, rootDir, _, built := setupTestHandlerWithRunner(t, "run-x")
	createTestItem(t, rootDir, KindExecute, BacklogItem{Name: "restart", Title: "R", Status: StatusBacklog, Priority: 3})
	if err := h.scheduleDeferredAdvanceIntent(KindExecute, "restart", agentops.OpWorkshopRound, "2030-01-01T00:00:00Z"); err != nil {
		t.Fatalf("schedule: %v", err)
	}
	// A fresh repo over the same root (simulating a restart) reloads the intent
	// from durable workflow state.
	freshRepo := opsrunner.NewWorkflowRepo(opsrunner.FSLocator{
		BacklogItemDir: func(kind, name string) (string, error) { return h.Store().ItemDir(BacklogKind(kind), name), nil },
		InitiativeDir:  func(name string) (string, error) { return filepath.Join(rootDir, "initiatives", name), nil },
		ScanRoots:      []string{rootDir},
	})
	_ = built
	w, found, err := freshRepo.Load(agentops.TargetBacklogItem, "execute/restart")
	if err != nil || !found {
		t.Fatalf("reload workflow: err=%v found=%v", err, found)
	}
	got := false
	for _, tm := range w.Timers {
		if tm.Intent == advanceIntentName && tm.Operation == agentops.OpWorkshopRound {
			got = true
		}
	}
	if !got {
		t.Fatal("expected the advance intent to survive a restart in durable workflow state")
	}
}

// ---------------------------------------------------------------------------
// AdvanceResolver: readiness decides workshop-round vs workshop-finalize.
// ---------------------------------------------------------------------------

func TestResolveAdvance_PicksRoundThenFinalize(t *testing.T) {
	h, rootDir, _, _ := setupTestHandlerWithRunner(t, "run-x")
	createTestItem(t, rootDir, KindExecute, BacklogItem{Name: "res", Title: "Res", Status: StatusBacklog, Priority: 3})
	w := agentops.WorkflowInstance{Domain: agentops.WorkflowDomain{Kind: "backlog-item", ID: "execute/res"}}

	// No rounds yet -> another workshop-round.
	req, ok, err := h.ResolveAdvance(context.Background(), w, agentops.ScheduledIntent{Intent: advanceIntentName, Operation: agentops.OpWorkshopRound})
	if err != nil || !ok {
		t.Fatalf("resolve advance: err=%v ok=%v", err, ok)
	}
	if req.Operation != agentops.OpWorkshopRound {
		t.Fatalf("operation = %q, want workshop-round with no rounds", req.Operation)
	}
	if req.OperationVersion != pinnedOperationVersion {
		t.Fatalf("operation version = %q, want %q", req.OperationVersion, pinnedOperationVersion)
	}
	if req.IdempotencyKey == "" {
		t.Fatal("expected a round-count-derived idempotency key")
	}
}

// ---------------------------------------------------------------------------
// Clarification: end-to-end through the real runner + policy.
// ---------------------------------------------------------------------------

// seedDecisionRound writes a workshop round with one decision item so a
// clarification can target it.
func seedDecisionRound(t *testing.T, rootDir string, kind BacklogKind, name string) {
	t.Helper()
	itemDir := filepath.Join(rootDir, backlogKindDirs[kind], name)
	round := workshop.Round{
		RoundNum: 1,
		Mode:     "workshop",
		Items: []workshop.Item{
			{ID: "d1", Type: "decision", Topic: "storage", Text: "pick a store"},
		},
	}
	testutil.WriteJSONFile(t, filepath.Join(itemDir, "workshop", "round-001.json"), round)
}

func TestClarification_CreateCommitsThreadThenStartsOperation(t *testing.T) {
	h, rootDir, phase, built := setupTestHandlerWithRunner(t, "run-clar-1")
	createTestItem(t, rootDir, KindExecute, BacklogItem{Name: "clar", Title: "Clar", Status: StatusBacklog, Priority: 3})
	seedDecisionRound(t, rootDir, KindExecute, "clar")

	body := `{"round_number":1,"item_id":"d1","message":"why sqlite?"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/execute/clar/workshop/clarification", bytes.NewBufferString(body))
	req = mux.SetURLVars(req, map[string]string{"kind": "execute", "name": "clar"})
	w := httptest.NewRecorder()
	h.CreateClarification(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	if !phase.started {
		t.Fatal("expected the clarification operation to start")
	}

	// The thread was committed BEFORE the async op, carrying the user message and
	// correlated to the live run.
	itemDir := h.Store().ItemDir(KindExecute, "clar")
	threads, err := workshop.LoadAllClarifications(itemDir)
	if err != nil || len(threads) != 1 {
		t.Fatalf("threads: err=%v n=%d", err, len(threads))
	}
	thread := threads[0]
	if thread.RunID != "run-clar-1" {
		t.Fatalf("thread run id = %q, want run-clar-1", thread.RunID)
	}
	if len(thread.Messages) != 1 || thread.Messages[0].Role != "user" {
		t.Fatalf("expected one user message, got %+v", thread.Messages)
	}

	// Drive the operation's completion: the start-clarification handler appends the
	// assistant turn (correlated by run id) and resolves the thread.
	wf, found, err := built.Repo.Load(agentops.TargetBacklogItem, "execute/clar")
	if err != nil || !found {
		t.Fatalf("load workflow: err=%v found=%v", err, found)
	}
	execID := wf.Operations[len(wf.Operations)-1].ExecutionID
	commitDelivery(t, built, KindExecute, "clar", execID, "completed", clarificationHandoff("use sqlite because it is simple"))

	updated, err := workshop.LoadClarificationByID(itemDir, thread.ID)
	if err != nil || updated == nil {
		t.Fatalf("reload thread: err=%v", err)
	}
	if len(updated.Messages) != 2 || updated.Messages[1].Role != "assistant" {
		t.Fatalf("expected an appended assistant turn, got %+v", updated.Messages)
	}
	if updated.Messages[1].Content != "use sqlite because it is simple" {
		t.Fatalf("assistant content = %q", updated.Messages[1].Content)
	}
	if updated.Status != "resolved" {
		t.Fatalf("thread status = %q, want resolved on completed outcome", updated.Status)
	}
}

// TestClarification_CreateForwardsQuestionAndTopic proves the operator's question
// and the targeted decision topic reach the engine as typed structured operator
// inputs the clarify mode's providers read.
func TestClarification_CreateForwardsQuestionAndTopic(t *testing.T) {
	h, rootDir, phase, _ := setupTestHandlerWithRunner(t, "run-clar-fwd")
	createTestItem(t, rootDir, KindExecute, BacklogItem{Name: "clarf", Title: "Clar", Status: StatusBacklog, Priority: 3})
	seedDecisionRound(t, rootDir, KindExecute, "clarf") // decision item Topic: "storage"

	body := `{"round_number":1,"item_id":"d1","message":"why sqlite?"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/execute/clarf/workshop/clarification", bytes.NewBufferString(body))
	req = mux.SetURLVars(req, map[string]string{"kind": "execute", "name": "clarf"})
	w := httptest.NewRecorder()
	h.CreateClarification(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	oi := phase.gotReq.OperatorInputs
	if oi["USER_QUESTION"] != "why sqlite?" {
		t.Fatalf("USER_QUESTION = %q, want the operator question; inputs=%+v", oi["USER_QUESTION"], oi)
	}
	if oi["DECISION_TOPIC"] != "storage" {
		t.Fatalf("DECISION_TOPIC = %q, want the decision topic", oi["DECISION_TOPIC"])
	}
}

// TestClarification_ContinueForwardsRequiredUserMessage proves the follow-up turn
// forwards USER_MESSAGE — the continue contract declares it REQUIRED, so without
// forwarding the runner would fail closed. It also exercises the continue path end
// to end through the real runner (no prior coverage did).
func TestClarification_ContinueForwardsRequiredUserMessage(t *testing.T) {
	h, rootDir, phase, _ := setupTestHandlerWithRunner(t, "run-clar-cont")
	createTestItem(t, rootDir, KindExecute, BacklogItem{Name: "clarc", Title: "Clar", Status: StatusBacklog, Priority: 3})
	seedDecisionRound(t, rootDir, KindExecute, "clarc")

	// Open the thread first.
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/execute/clarc/workshop/clarification", bytes.NewBufferString(`{"round_number":1,"item_id":"d1","message":"why sqlite?"}`))
	createReq = mux.SetURLVars(createReq, map[string]string{"kind": "execute", "name": "clarc"})
	createW := httptest.NewRecorder()
	h.CreateClarification(createW, createReq)
	if createW.Code != http.StatusCreated {
		t.Fatalf("create status = %d; body=%s", createW.Code, createW.Body.String())
	}
	itemDir := h.Store().ItemDir(KindExecute, "clarc")
	threads, err := workshop.LoadAllClarifications(itemDir)
	if err != nil || len(threads) != 1 {
		t.Fatalf("threads: err=%v n=%d", err, len(threads))
	}
	threadID := threads[0].ID

	// Continue the thread and assert the follow-up message flows as USER_MESSAGE.
	contReq := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/execute/clarc/workshop/clarification/"+threadID+"/continue", bytes.NewBufferString(`{"message":"and what about backups?"}`))
	contReq = mux.SetURLVars(contReq, map[string]string{"kind": "execute", "name": "clarc", "threadId": threadID})
	contW := httptest.NewRecorder()
	h.ContinueClarification(contW, contReq)
	if contW.Code != http.StatusOK {
		t.Fatalf("continue status = %d, want 200; body=%s", contW.Code, contW.Body.String())
	}
	if phase.gotReq.OperatorInputs["USER_MESSAGE"] != "and what about backups?" {
		t.Fatalf("USER_MESSAGE = %q, want the follow-up message; inputs=%+v", phase.gotReq.OperatorInputs["USER_MESSAGE"], phase.gotReq.OperatorInputs)
	}
}

func TestClarificationTurn_AbstainWritesFallback(t *testing.T) {
	h, rootDir, _, built := setupTestHandlerWithRunner(t, "run-clar-2")
	createTestItem(t, rootDir, KindExecute, BacklogItem{Name: "clarab", Title: "C", Status: StatusBacklog, Priority: 3})
	seedDecisionRound(t, rootDir, KindExecute, "clarab")

	body := `{"round_number":1,"item_id":"d1","message":"explain"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/execute/clarab/workshop/clarification", bytes.NewBufferString(body))
	req = mux.SetURLVars(req, map[string]string{"kind": "execute", "name": "clarab"})
	w := httptest.NewRecorder()
	h.CreateClarification(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create status = %d; body=%s", w.Code, w.Body.String())
	}

	itemDir := h.Store().ItemDir(KindExecute, "clarab")
	threads, _ := workshop.LoadAllClarifications(itemDir)
	thread := threads[0]
	wf, _, _ := built.Repo.Load(agentops.TargetBacklogItem, "execute/clarab")
	execID := wf.Operations[len(wf.Operations)-1].ExecutionID
	// An abstaining (needs-attention) outcome carries no result; the handler still
	// writes a fallback turn so the thread is never silently stalled.
	commitDelivery(t, built, KindExecute, "clarab", execID, "needs-attention", nil)

	updated, _ := workshop.LoadClarificationByID(itemDir, thread.ID)
	if len(updated.Messages) != 2 || updated.Messages[1].Role != "assistant" {
		t.Fatalf("expected a fallback assistant turn, got %+v", updated.Messages)
	}
	if updated.Messages[1].Content != clarificationFallbackAnswer {
		t.Fatalf("expected fallback content, got %q", updated.Messages[1].Content)
	}
	if updated.Status == "resolved" {
		t.Fatal("an abstaining clarification should not resolve the thread")
	}
}
