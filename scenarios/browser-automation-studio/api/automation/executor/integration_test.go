package executor

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/engine"
	"github.com/vrooli/browser-automation-studio/automation/events"
	"github.com/vrooli/browser-automation-studio/automation/recorder"
	"github.com/vrooli/browser-automation-studio/database"
)

var (
	pgContainer testcontainers.Container
	testDBURL   string
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Allow callers to supply their own database (e.g., CI) to avoid containers.
	if url := os.Getenv("TEST_DATABASE_URL"); url != "" {
		testDBURL = url
		os.Exit(m.Run())
	}

	container, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		postgres.WithDatabase("bas_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		fmt.Printf("skipping automation executor integration tests: %v\n", err)
		os.Exit(0)
	}

	pgContainer = container

	url, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Printf("failed to get postgres connection string: %v\n", err)
		_ = container.Terminate(ctx)
		os.Exit(1)
	}
	testDBURL = url

	code := m.Run()

	if pgContainer != nil {
		_ = pgContainer.Terminate(ctx)
	}

	os.Exit(code)
}

func TestSimpleExecutorPersistsArtifactsAndEvents(t *testing.T) {
	if testDBURL == "" {
		t.Skip("test database not initialized")
	}

	repo, cleanup := setupRepo(t)
	defer cleanup()

	executionID := uuid.New()
	workflowID := uuid.New()

	log := logrus.New()
	log.SetOutput(io.Discard)

	createWorkflowFixture(t, repo, workflowID, executionID)

	plan := contracts.ExecutionPlan{
		SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    executionID,
		WorkflowID:     workflowID,
		Instructions: []contracts.CompiledInstruction{
			{Index: 0, NodeID: "navigate-1", Type: "navigate", Params: map[string]any{"url": "https://example.test/1"}},
			{Index: 1, NodeID: "assert-1", Type: "assert", Params: map[string]any{"selector": "#ok"}},
		},
		CreatedAt: time.Now().UTC(),
	}

	memSink := events.NewMemorySink(contracts.DefaultEventBufferLimits)
	rec := recorder.NewDBRecorder(repo, nil, log)

	stub := &stubEngine{
		outcomes: map[int]contracts.StepOutcome{
			0: buildOutcome(plan.ExecutionID, 0, "navigate-1", "navigate"),
			1: buildOutcome(plan.ExecutionID, 1, "assert-1", "assert"),
		},
	}

	exec := NewSimpleExecutor(nil)
	req := Request{
		Plan:              plan,
		EngineName:        stub.Name(),
		EngineFactory:     engine.NewStaticFactory(stub),
		Recorder:          rec,
		EventSink:         memSink,
		HeartbeatInterval: 0, // keep deterministic
	}

	if err := exec.Execute(context.Background(), req); err != nil {
		t.Fatalf("executor returned error: %v", err)
	}

	assertDBArtifacts(t, repo, executionID)
	assertEvents(t, memSink.Events(), plan.ExecutionID)
}

func TestSimpleExecutorProducesLegacyCompatibleArtifacts(t *testing.T) {
	if testDBURL == "" {
		t.Skip("test database not initialized")
	}

	repo, cleanup := setupRepo(t)
	defer cleanup()

	executionID := uuid.New()
	workflowID := uuid.New()

	log := logrus.New()
	log.SetOutput(io.Discard)

	createWorkflowFixture(t, repo, workflowID, executionID)

	now := time.Now().UTC()
	plan := contracts.ExecutionPlan{
		SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    executionID,
		WorkflowID:     workflowID,
		Instructions: []contracts.CompiledInstruction{
			{Index: 0, NodeID: "navigate-1", Type: "navigate", Params: map[string]any{"url": "https://example.test/1"}},
			{Index: 1, NodeID: "assert-1", Type: "assert", Params: map[string]any{"selector": "#ok"}},
		},
		CreatedAt: now,
	}

	memSink := events.NewMemorySink(contracts.DefaultEventBufferLimits)
	rec := recorder.NewDBRecorder(repo, nil, log)

	stub := &stubEngine{
		outcomes: map[int]contracts.StepOutcome{
			0: buildOutcomeWith(now, executionID, 0, "navigate-1", "navigate", false),
			1: buildOutcomeWith(now, executionID, 1, "assert-1", "assert", true),
		},
	}

	exec := NewSimpleExecutor(nil)
	req := Request{
		Plan:              plan,
		EngineName:        stub.Name(),
		EngineFactory:     engine.NewStaticFactory(stub),
		Recorder:          rec,
		EventSink:         memSink,
		HeartbeatInterval: 0,
	}

	if err := exec.Execute(context.Background(), req); err != nil {
		t.Fatalf("executor returned error: %v", err)
	}

	ctx := context.Background()
	artifacts, err := repo.ListExecutionArtifacts(ctx, executionID)
	if err != nil {
		t.Fatalf("list execution artifacts: %v", err)
	}

	byStep := groupArtifactsByStep(artifacts)
	checkLegacyArtifactShape(t, byStep[0], false)
	checkLegacyArtifactShape(t, byStep[1], true)

	events := memSink.Events()
	if len(events) == 0 {
		t.Fatalf("expected events emitted")
	}
}

func TestSimpleExecutorEmitsLegacyEventPayloads(t *testing.T) {
	plan := contracts.ExecutionPlan{
		SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    uuid.New(),
		WorkflowID:     uuid.New(),
		Instructions: []contracts.CompiledInstruction{
			{Index: 0, NodeID: "nav-1", Type: "navigate", Params: map[string]any{"url": "https://example.test"}},
			{Index: 1, NodeID: "assert-1", Type: "assert", Params: map[string]any{"selector": "#ok"}},
		},
		CreatedAt: time.Now().UTC(),
	}

	memSink := events.NewMemorySink(contracts.DefaultEventBufferLimits)
	rec := &stubRecorder{}

	now := time.Now().UTC()
	stub := &stubEngine{
		outcomes: map[int]contracts.StepOutcome{
			0: buildOutcomeWith(now, plan.ExecutionID, 0, "nav-1", "navigate", false),
			1: buildOutcomeWith(now, plan.ExecutionID, 1, "assert-1", "assert", true),
		},
	}

	exec := NewSimpleExecutor(nil)
	req := Request{
		Plan:              plan,
		EngineName:        stub.Name(),
		EngineFactory:     engine.NewStaticFactory(stub),
		Recorder:          rec,
		EventSink:         memSink,
		HeartbeatInterval: 0,
	}

	if err := exec.Execute(context.Background(), req); err != nil {
		t.Fatalf("executor returned error: %v", err)
	}

	events := memSink.Events()
	if len(events) != 4 {
		t.Fatalf("expected 4 events (start/complete per step), got %d", len(events))
	}

	expectKind := func(idx int, kind contracts.EventKind) {
		if events[idx].Kind != kind {
			t.Fatalf("event %d expected kind %s, got %s", idx, kind, events[idx].Kind)
		}
	}
	expectKind(0, contracts.EventKindStepStarted)
	expectKind(1, contracts.EventKindStepCompleted)
	expectKind(2, contracts.EventKindStepStarted)
	expectKind(3, contracts.EventKindStepCompleted)

	checkStartPayload := func(ev contracts.EventEnvelope, expectedNode string) {
		payload, ok := ev.Payload.(map[string]any)
		if !ok {
			t.Fatalf("start payload type mismatch: %T", ev.Payload)
		}
		if payload["node_id"] != expectedNode {
			t.Fatalf("start payload node_id mismatch, got %v", payload["node_id"])
		}
		if payload["step_type"] == "" {
			t.Fatalf("start payload missing step_type")
		}
	}
	checkStartPayload(events[0], "nav-1")
	checkStartPayload(events[2], "assert-1")

	checkCompletePayload := func(ev contracts.EventEnvelope, expectedNode string, expectAssertion bool) {
		payload, ok := ev.Payload.(map[string]any)
		if !ok {
			t.Fatalf("complete payload type mismatch: %T", ev.Payload)
		}
		outcome, ok := payload["outcome"].(contracts.StepOutcome)
		if !ok {
			t.Fatalf("complete payload missing outcome: %+v", payload)
		}
		if outcome.NodeID != expectedNode {
			t.Fatalf("outcome node mismatch: got %s want %s", outcome.NodeID, expectedNode)
		}
		if outcome.StepIndex != *ev.StepIndex {
			t.Fatalf("outcome step index mismatch: got %d want %d", outcome.StepIndex, *ev.StepIndex)
		}
		if expectAssertion && outcome.Assertion == nil {
			t.Fatalf("expected assertion payload for node %s", expectedNode)
		}
		if !expectAssertion && outcome.Assertion != nil {
			t.Fatalf("did not expect assertion payload for node %s", expectedNode)
		}
		if _, ok := payload["artifacts"]; !ok {
			t.Fatalf("complete payload missing artifacts list")
		}
	}
	checkCompletePayload(events[1], "nav-1", false)
	checkCompletePayload(events[3], "assert-1", true)
}

func TestSimpleExecutorLinearGoldenEvents(t *testing.T) {
	executionID := uuid.New()
	workflowID := uuid.New()

	fixedStart := time.Date(2024, 10, 1, 12, 0, 0, 0, time.UTC)
	fixedEnd := fixedStart.Add(1500 * time.Millisecond)

	plan := contracts.ExecutionPlan{
		SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    executionID,
		WorkflowID:     workflowID,
		Instructions: []contracts.CompiledInstruction{
			{Index: 0, NodeID: "nav-1", Type: "navigate", Params: map[string]any{"url": "https://example.test"}},
		},
		CreatedAt: fixedStart,
	}

	memSink := events.NewMemorySink(contracts.DefaultEventBufferLimits)
	rec := &stubRecorder{}

	stub := &stubEngine{
		outcomes: map[int]contracts.StepOutcome{
			0: {
				Success:     true,
				StepIndex:   0,
				NodeID:      "nav-1",
				StepType:    "navigate",
				StartedAt:   fixedStart,
				CompletedAt: &fixedEnd,
				DurationMs:  1500,
				FinalURL:    "https://example.test/home",
			},
		},
	}

	exec := NewSimpleExecutor(nil)
	req := Request{
		Plan:              plan,
		EngineName:        stub.Name(),
		EngineFactory:     engine.NewStaticFactory(stub),
		Recorder:          rec,
		EventSink:         memSink,
		HeartbeatInterval: 0,
	}

	if err := exec.Execute(context.Background(), req); err != nil {
		t.Fatalf("executor returned error: %v", err)
	}

	evs := memSink.Events()
	if len(evs) != 2 {
		t.Fatalf("expected 2 events (start + complete), got %d", len(evs))
	}

	// Normalize dynamic fields so comparisons stay deterministic.
	for i := range evs {
		evs[i].Timestamp = time.Time{}
		evs[i].Sequence = 0
	}

	start := evs[0]
	if start.Kind != contracts.EventKindStepStarted {
		t.Fatalf("first event kind mismatch: want %s got %s", contracts.EventKindStepStarted, start.Kind)
	}
	if start.StepIndex == nil || *start.StepIndex != 0 {
		t.Fatalf("start event step index mismatch: %+v", start.StepIndex)
	}
	if payload, ok := start.Payload.(map[string]any); !ok || payload["node_id"] != "nav-1" {
		t.Fatalf("start payload node mismatch: %+v", start.Payload)
	}

	complete := evs[1]
	if complete.Kind != contracts.EventKindStepCompleted {
		t.Fatalf("second event kind mismatch: want %s got %s", contracts.EventKindStepCompleted, complete.Kind)
	}
	if complete.StepIndex == nil || *complete.StepIndex != 0 {
		t.Fatalf("complete event step index mismatch: %+v", complete.StepIndex)
	}
	payload, ok := complete.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", complete.Payload)
	}
	outcome, ok := payload["outcome"].(contracts.StepOutcome)
	if !ok {
		t.Fatalf("expected outcome payload in completion event, got %T", payload["outcome"])
	}
	if outcome.FinalURL != "https://example.test/home" {
		t.Fatalf("expected final_url to propagate, got %s", outcome.FinalURL)
	}
	if outcome.DurationMs != 1500 {
		t.Fatalf("expected duration to be preserved, got %d", outcome.DurationMs)
	}
}

func setupRepo(t *testing.T) (database.Repository, func()) {
	t.Helper()

	oldURL := os.Getenv("DATABASE_URL")
	oldSkip := os.Getenv("BAS_SKIP_DEMO_SEED")

	if err := os.Setenv("DATABASE_URL", testDBURL); err != nil {
		t.Fatalf("set DATABASE_URL: %v", err)
	}
	_ = os.Setenv("BAS_SKIP_DEMO_SEED", "true")

	log := logrus.New()
	log.SetOutput(io.Discard)

	db, err := database.NewConnection(log)
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}

	repo := database.NewRepository(db, log)

	cleanup := func() {
		ctx := context.Background()
		// Delete in reverse dependency order to avoid foreign key violations
		queries := []string{
			"DELETE FROM execution_artifacts WHERE execution_id IN (SELECT id FROM executions WHERE workflow_id IN (SELECT id FROM workflows WHERE folder_path LIKE '/test%'))",
			"DELETE FROM execution_steps WHERE execution_id IN (SELECT id FROM executions WHERE workflow_id IN (SELECT id FROM workflows WHERE folder_path LIKE '/test%'))",
			"DELETE FROM executions WHERE workflow_id IN (SELECT id FROM workflows WHERE folder_path LIKE '/test%')",
			"DELETE FROM workflows WHERE folder_path LIKE '/test%'",
			"DELETE FROM workflow_folders WHERE folder_path LIKE '/test%'",
			"DELETE FROM projects WHERE folder_path LIKE '/test%'",
		}
		for _, q := range queries {
			db.ExecContext(ctx, q)
		}
		_ = db.Close()

		if oldURL != "" {
			_ = os.Setenv("DATABASE_URL", oldURL)
		} else {
			_ = os.Unsetenv("DATABASE_URL")
		}
		if oldSkip != "" {
			_ = os.Setenv("BAS_SKIP_DEMO_SEED", oldSkip)
		} else {
			_ = os.Unsetenv("BAS_SKIP_DEMO_SEED")
		}
	}

	return repo, cleanup
}

func createWorkflowFixture(t *testing.T, repo database.Repository, workflowID, executionID uuid.UUID) {
	t.Helper()
	ctx := context.Background()

	project := &database.Project{
		ID:         uuid.New(),
		Name:       "automation-test-project",
		FolderPath: "/test/automation",
	}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	workflow := &database.Workflow{
		ID:         workflowID,
		ProjectID:  &project.ID,
		Name:       "automation-integration",
		FolderPath: "/test/automation",
		Version:    1,
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{"id": "navigate-1", "type": "navigate"},
				map[string]any{"id": "assert-1", "type": "assert"},
			},
			"edges": []any{},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := repo.CreateWorkflow(ctx, workflow); err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	execution := &database.Execution{
		ID:              executionID,
		WorkflowID:      workflowID,
		WorkflowVersion: workflow.Version,
		Status:          "running",
		TriggerType:     "manual",
		StartedAt:       time.Now(),
	}
	if err := repo.CreateExecution(ctx, execution); err != nil {
		t.Fatalf("create execution: %v", err)
	}
}

func assertDBArtifacts(t *testing.T, repo database.Repository, executionID uuid.UUID) {
	t.Helper()
	ctx := context.Background()

	steps, err := repo.ListExecutionSteps(ctx, executionID)
	if err != nil {
		t.Fatalf("list execution steps: %v", err)
	}
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}
	for _, step := range steps {
		if step.Status != "completed" {
			t.Fatalf("expected step %d status completed, got %s", step.StepIndex, step.Status)
		}
		if step.Metadata == nil {
			t.Fatalf("expected metadata for step %d", step.StepIndex)
		}
	}

	artifacts, err := repo.ListExecutionArtifacts(ctx, executionID)
	if err != nil {
		t.Fatalf("list execution artifacts: %v", err)
	}
	if len(artifacts) == 0 {
		t.Fatal("expected artifacts persisted")
	}

	found := map[string]bool{}
	for _, art := range artifacts {
		found[art.ArtifactType] = true
	}
	for _, required := range []string{"step_outcome", "console", "network", "assertion", "screenshot_inline"} {
		if !found[required] {
			t.Fatalf("expected artifact type %s to be present, got %+v", required, found)
		}
	}
}

func assertEvents(t *testing.T, events []contracts.EventEnvelope, executionID uuid.UUID) {
	t.Helper()
	if len(events) < 4 {
		t.Fatalf("expected at least 4 events (start/complete per step), got %d", len(events))
	}
	var lastSeq uint64
	for _, ev := range events {
		if ev.ExecutionID != executionID {
			t.Fatalf("unexpected execution id %s", ev.ExecutionID)
		}
		if ev.Sequence <= lastSeq {
			t.Fatalf("events not strictly ordered: prev=%d curr=%d", lastSeq, ev.Sequence)
		}
		lastSeq = ev.Sequence
	}
	// Ensure each step emitted start and completion.
	seen := map[int]map[contracts.EventKind]bool{}
	for _, ev := range events {
		if ev.StepIndex == nil {
			continue
		}
		idx := *ev.StepIndex
		if seen[idx] == nil {
			seen[idx] = map[contracts.EventKind]bool{}
		}
		seen[idx][ev.Kind] = true
	}
	for _, idx := range []int{0, 1} {
		kinds := seen[idx]
		if kinds == nil || !kinds[contracts.EventKindStepStarted] || !kinds[contracts.EventKindStepCompleted] {
			t.Fatalf("missing start/completed events for step %d: %+v", idx, kinds)
		}
	}
}

func groupArtifactsByStep(artifacts []*database.ExecutionArtifact) map[int][]*database.ExecutionArtifact {
	out := make(map[int][]*database.ExecutionArtifact)
	for _, art := range artifacts {
		if art.StepIndex == nil {
			continue
		}
		out[*art.StepIndex] = append(out[*art.StepIndex], art)
	}
	return out
}

func checkLegacyArtifactShape(t *testing.T, artifacts []*database.ExecutionArtifact, expectAssertion bool) {
	t.Helper()

	required := map[string]bool{
		"step_outcome":      false,
		"console":           false,
		"network":           false,
		"extracted_data":    false,
		"screenshot_inline": false,
		"dom_snapshot":      false,
		"timeline_frame":    false,
	}
	if expectAssertion {
		required["assertion"] = false
	}

	for _, art := range artifacts {
		if _, ok := required[art.ArtifactType]; ok {
			required[art.ArtifactType] = true
		}
		switch art.ArtifactType {
		case "step_outcome":
			payload, _ := art.Payload["outcome"].(map[string]any)
			if payload == nil {
				t.Fatalf("step_outcome missing payload: %+v", art.Payload)
			}
			if _, ok := payload["node_id"]; !ok {
				t.Fatalf("step_outcome missing node_id: %+v", payload)
			}
			if _, ok := payload["step_index"]; !ok {
				t.Fatalf("step_outcome missing step_index: %+v", payload)
			}
		case "console":
			entries, _ := art.Payload["entries"].([]any)
			if len(entries) == 0 {
				t.Fatalf("console artifact missing entries: %+v", art.Payload)
			}
		case "network":
			events, _ := art.Payload["events"].([]any)
			if len(events) == 0 {
				t.Fatalf("network artifact missing events: %+v", art.Payload)
			}
		case "assertion":
			if !expectAssertion {
				t.Fatalf("unexpected assertion artifact for non-assert step")
			}
			if _, ok := art.Payload["assertion"]; !ok {
				t.Fatalf("assertion artifact missing payload")
			}
		case "extracted_data":
			if _, ok := art.Payload["value"]; !ok {
				t.Fatalf("extracted_data artifact missing value: %+v", art.Payload)
			}
		case "screenshot_inline":
			if _, ok := art.Payload["base64"]; !ok {
				t.Fatalf("inline screenshot missing base64 payload: %+v", art.Payload)
			}
		case "dom_snapshot":
			if _, ok := art.Payload["html"]; !ok {
				t.Fatalf("dom_snapshot missing html: %+v", art.Payload)
			}
		case "timeline_frame":
			expectTimelineKeys(t, art.Payload)
		}
	}

	for typ, seen := range required {
		if !seen {
			t.Fatalf("missing artifact type %s in %+v", typ, required)
		}
	}
}

func expectTimelineKeys(t *testing.T, payload map[string]any) {
	t.Helper()
	for _, key := range []string{"stepIndex", "nodeId", "stepType", "screenshotUrl", "artifactIds"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("timeline payload missing %s: %+v", key, payload)
		}
	}
	url, _ := payload["screenshotUrl"].(string)
	if url == "" || !strings.HasPrefix(url, "inline:") {
		t.Fatalf("expected inline screenshot url, got %v", payload["screenshotUrl"])
	}
}

type stubEngine struct {
	outcomes map[int]contracts.StepOutcome
}

func (s *stubEngine) Name() string { return "stub-engine" }

func (s *stubEngine) Capabilities(context.Context) (contracts.EngineCapabilities, error) {
	return contracts.EngineCapabilities{
		SchemaVersion:         contracts.CapabilitiesSchemaVersion,
		Engine:                s.Name(),
		MaxConcurrentSessions: 1,
	}, nil
}

func (s *stubEngine) StartSession(context.Context, engine.SessionSpec) (engine.EngineSession, error) {
	return &stubSession{outcomes: s.outcomes}, nil
}

type stubSession struct {
	outcomes map[int]contracts.StepOutcome
}

func (s *stubSession) Run(ctx context.Context, instruction contracts.CompiledInstruction) (contracts.StepOutcome, error) {
	out, ok := s.outcomes[instruction.Index]
	if !ok {
		return contracts.StepOutcome{}, fmt.Errorf("no stub outcome for step %d", instruction.Index)
	}
	// Simulate a tiny bit of runtime to make timestamps realistic.
	select {
	case <-ctx.Done():
		return contracts.StepOutcome{}, ctx.Err()
	case <-time.After(5 * time.Millisecond):
	}
	return out, nil
}

func (s *stubSession) Reset(context.Context) error { return nil }
func (s *stubSession) Close(context.Context) error { return nil }

func buildOutcome(executionID uuid.UUID, idx int, nodeID, stepType string) contracts.StepOutcome {
	now := time.Now().UTC()
	return contracts.StepOutcome{
		SchemaVersion:  contracts.StepOutcomeSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    executionID,
		StepIndex:      idx,
		NodeID:         nodeID,
		StepType:       stepType,
		Success:        true,
		StartedAt:      now,
		CompletedAt:    ptr(time.Now().UTC()),
		DurationMs:     12,
		FinalURL:       "https://example.test/" + nodeID,
		Screenshot: &contracts.Screenshot{
			Data:        []byte{0x01, 0x02, 0x03},
			MediaType:   "image/png",
			CaptureTime: now,
			Width:       800,
			Height:      600,
		},
		DOMSnapshot: &contracts.DOMSnapshot{
			HTML:        "<html><body>" + nodeID + "</body></html>",
			CollectedAt: now,
		},
		ConsoleLogs: []contracts.ConsoleLogEntry{
			{Type: "log", Text: "hello", Timestamp: now},
		},
		Network: []contracts.NetworkEvent{
			{Type: "request", URL: "https://example.test", Method: "GET", Timestamp: now},
		},
		ExtractedData: map[string]any{"foo": "bar"},
		Assertion: &contracts.AssertionOutcome{
			Mode:    "equals",
			Success: true,
			Message: "ok",
		},
	}
}

func buildOutcomeWith(now time.Time, executionID uuid.UUID, idx int, nodeID, stepType string, withAssertion bool) contracts.StepOutcome {
	out := buildOutcome(executionID, idx, nodeID, stepType)
	out.StartedAt = now
	out.CompletedAt = ptr(time.Now().UTC())
	out.Assertion = nil
	if withAssertion {
		out.Assertion = &contracts.AssertionOutcome{
			Mode:    "equals",
			Success: true,
			Message: "ok",
		}
	}
	return out
}

type stubRecorder struct {
	outcomes  []contracts.StepOutcome
	telemetry []contracts.StepTelemetry
}

func (s *stubRecorder) RecordStepOutcome(ctx context.Context, plan contracts.ExecutionPlan, outcome contracts.StepOutcome) (recorder.RecordResult, error) {
	s.outcomes = append(s.outcomes, outcome)
	return recorder.RecordResult{
		StepID:      uuid.New(),
		ArtifactIDs: []uuid.UUID{uuid.New()},
	}, nil
}

func (s *stubRecorder) RecordTelemetry(ctx context.Context, plan contracts.ExecutionPlan, telemetry contracts.StepTelemetry) error {
	s.telemetry = append(s.telemetry, telemetry)
	return nil
}

func (s *stubRecorder) MarkCrash(ctx context.Context, executionID uuid.UUID, failure contracts.StepFailure) error {
	return nil
}

func ptr[T any](v T) *T {
	return &v
}
