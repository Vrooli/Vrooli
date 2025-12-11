package executor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/engine"
	"github.com/vrooli/browser-automation-studio/automation/events"
	"github.com/vrooli/browser-automation-studio/automation/recorder"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/testutil/testdb"
)

var (
	testDBHandle *testdb.Handle
	// testDBMu protects concurrent access to the shared test database
	testDBMu sync.Mutex
)

func testBackend() database.Dialect {
	backend := strings.ToLower(strings.TrimSpace(os.Getenv("BAS_TEST_BACKEND")))
	if backend == "" {
		backend = strings.ToLower(strings.TrimSpace(os.Getenv("BAS_DB_BACKEND")))
	}
	if backend == "" {
		backend = string(database.DialectPostgres)
	}
	return database.Dialect(backend)
}

func TestMain(m *testing.M) {
	backend := testBackend()
	ctx := context.Background()

	if backend == database.DialectPostgres {
		handle, err := testdb.Start(ctx)
		if err != nil {
			fmt.Printf("skipping automation executor integration tests: %v\n", err)
			os.Exit(0)
		}
		testDBHandle = handle

		code := m.Run()

		handle.Terminate(ctx)

		os.Exit(code)
	}

	// SQLite or other dialects do not need the Postgres container.
	os.Exit(m.Run())
}

func TestSimpleExecutorPersistsArtifactsAndEvents(t *testing.T) {
	repo, db, cleanup := setupRepo(t)
	defer cleanup()

	executionID := uuid.New()
	workflowID := uuid.New()

	log := logrus.New()
	log.SetOutput(io.Discard)

	createWorkflowFixture(t, repo, db, workflowID, executionID)

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
	repo, db, cleanup := setupRepo(t)
	defer cleanup()

	executionID := uuid.New()
	workflowID := uuid.New()

	log := logrus.New()
	log.SetOutput(io.Discard)

	createWorkflowFixture(t, repo, db, workflowID, executionID)

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

func setupRepo(t *testing.T) (database.Repository, *database.DB, func()) {
	t.Helper()

	backend := testBackend()
	log := logrus.New()
	log.SetOutput(io.Discard)

	// Save and clear environment variables
	oldURL := os.Getenv("DATABASE_URL")
	oldSkip := os.Getenv("BAS_SKIP_DEMO_SEED")
	oldHost := os.Getenv("POSTGRES_HOST")
	oldPort := os.Getenv("POSTGRES_PORT")
	oldUser := os.Getenv("POSTGRES_USER")
	oldPass := os.Getenv("POSTGRES_PASSWORD")
	oldDB := os.Getenv("POSTGRES_DB")
	oldBackend := os.Getenv("BAS_DB_BACKEND")
	oldSQLitePath := os.Getenv("BAS_SQLITE_PATH")

	restoreEnv := func() {
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
		if oldHost != "" {
			_ = os.Setenv("POSTGRES_HOST", oldHost)
		} else {
			_ = os.Unsetenv("POSTGRES_HOST")
		}
		if oldPort != "" {
			_ = os.Setenv("POSTGRES_PORT", oldPort)
		} else {
			_ = os.Unsetenv("POSTGRES_PORT")
		}
		if oldUser != "" {
			_ = os.Setenv("POSTGRES_USER", oldUser)
		} else {
			_ = os.Unsetenv("POSTGRES_USER")
		}
		if oldPass != "" {
			_ = os.Setenv("POSTGRES_PASSWORD", oldPass)
		} else {
			_ = os.Unsetenv("POSTGRES_PASSWORD")
		}
		if oldDB != "" {
			_ = os.Setenv("POSTGRES_DB", oldDB)
		} else {
			_ = os.Unsetenv("POSTGRES_DB")
		}
		if oldBackend != "" {
			_ = os.Setenv("BAS_DB_BACKEND", oldBackend)
		} else {
			_ = os.Unsetenv("BAS_DB_BACKEND")
		}
		if oldSQLitePath != "" {
			_ = os.Setenv("BAS_SQLITE_PATH", oldSQLitePath)
		} else {
			_ = os.Unsetenv("BAS_SQLITE_PATH")
		}
	}

	switch backend {
	case database.DialectPostgres:
		if testDBHandle == nil {
			t.Skip("postgres test database not initialized")
		}

		// Acquire exclusive access to the shared test database
		testDBMu.Lock()

		// Clear POSTGRES_* vars to ensure DATABASE_URL is used
		_ = os.Unsetenv("POSTGRES_HOST")
		_ = os.Unsetenv("POSTGRES_PORT")
		_ = os.Unsetenv("POSTGRES_USER")
		_ = os.Unsetenv("POSTGRES_PASSWORD")
		_ = os.Unsetenv("POSTGRES_DB")

		_ = os.Setenv("DATABASE_URL", testDBHandle.DSN)
		_ = os.Setenv("BAS_SKIP_DEMO_SEED", "true")
		_ = os.Setenv("BAS_DB_BACKEND", string(database.DialectPostgres))
		_ = os.Unsetenv("BAS_SQLITE_PATH")

		db, err := database.NewConnection(log)
		if err != nil {
			testDBMu.Unlock()
			restoreEnv()
			t.Fatalf("failed to connect to test db: %v", err)
		}

		if err := truncateAllWithRetry(db, 3); err != nil {
			_ = db.Close()
			testDBMu.Unlock()
			restoreEnv()
			t.Fatalf("truncate test tables: %v", err)
		}

		repo := database.NewRepository(db, log)

		cleanup := func() {
			_ = truncateAll(db)
			_ = db.Close()
			restoreEnv()
			testDBMu.Unlock()
		}

		return repo, db, cleanup
	case database.DialectSQLite:
		tmpDir := t.TempDir()
		sqlitePath := filepath.Join(tmpDir, "executor.db")

		_ = os.Setenv("BAS_DB_BACKEND", string(database.DialectSQLite))
		_ = os.Setenv("BAS_SQLITE_PATH", sqlitePath)
		_ = os.Setenv("BAS_SKIP_DEMO_SEED", "true")
		_ = os.Unsetenv("DATABASE_URL")
		_ = os.Unsetenv("POSTGRES_HOST")
		_ = os.Unsetenv("POSTGRES_PORT")
		_ = os.Unsetenv("POSTGRES_USER")
		_ = os.Unsetenv("POSTGRES_PASSWORD")
		_ = os.Unsetenv("POSTGRES_DB")

		db, err := database.NewConnection(log)
		if err != nil {
			restoreEnv()
			t.Fatalf("failed to connect to sqlite test db: %v", err)
		}

		if err := truncateAll(db); err != nil {
			_ = db.Close()
			restoreEnv()
			t.Fatalf("truncate sqlite tables: %v", err)
		}

		repo := database.NewRepository(db, log)
		cleanup := func() {
			_ = truncateAll(db)
			_ = db.Close()
			restoreEnv()
		}
		return repo, db, cleanup
	default:
		restoreEnv()
		t.Fatalf("unsupported BAS_TEST_BACKEND %q", backend)
		return nil, nil, nil
	}
}

func truncateAll(db *database.DB) error {
	if db == nil {
		return nil
	}
	ctx := context.Background()

	if db.Dialect().IsPostgres() {
		// Use a single TRUNCATE statement with all tables to avoid deadlock issues
		// and ensure atomicity of the cleanup operation.
		// Tables are listed in reverse dependency order (children before parents).
		query := `TRUNCATE
		exports,
		ai_generations,
		workflow_schedules,
		workflow_templates,
		execution_artifacts,
		execution_steps,
		execution_logs,
		screenshots,
		extracted_data,
		executions,
		workflow_versions,
		workflows,
		workflow_folders,
		projects
	CASCADE`

		if _, err := db.ExecContext(ctx, query); err != nil {
			return err
		}
		return nil
	}

	// SQLite: DELETE tables in dependency order and vacuum to reset autoincrement.
	tables := []string{
		"exports",
		"ai_generations",
		"workflow_schedules",
		"workflow_templates",
		"execution_artifacts",
		"execution_steps",
		"execution_logs",
		"screenshots",
		"extracted_data",
		"executions",
		"workflow_versions",
		"workflows",
		"workflow_folders",
		"projects",
	}
	for _, table := range tables {
		if _, err := db.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s", table)); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "no such table") {
				continue
			}
			return err
		}
	}
	_, _ = db.ExecContext(ctx, "VACUUM")
	return nil
}

// truncateAllWithRetry attempts to truncate all tables with retries for transient errors.
func truncateAllWithRetry(db *database.DB, maxRetries int) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if err := truncateAll(db); err != nil {
			lastErr = err
			// Check if it's a deadlock error and retry
			if strings.Contains(err.Error(), "deadlock") {
				time.Sleep(time.Duration(50*(i+1)) * time.Millisecond)
				continue
			}
			return err
		}
		return nil
	}
	return lastErr
}

func createWorkflowFixture(t *testing.T, repo database.Repository, db *database.DB, workflowID, executionID uuid.UUID) {
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

// TestContextCancellationPersistsFailure verifies that when a context is cancelled
// mid-execution, the executor:
// 1. Returns context.Canceled error
// 2. Records a step outcome with FailureKindCancelled
// 3. Emits a StepFailed event with proper failure taxonomy
// 4. Uses context.WithoutCancel for cleanup persistence (A15)
func TestContextCancellationPersistsFailure(t *testing.T) {
	repo, db, cleanup := setupRepo(t)
	defer cleanup()

	executionID := uuid.New()
	workflowID := uuid.New()

	log := logrus.New()
	log.SetOutput(io.Discard)

	createWorkflowFixture(t, repo, db, workflowID, executionID)

	plan := contracts.ExecutionPlan{
		SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    executionID,
		WorkflowID:     workflowID,
		Instructions: []contracts.CompiledInstruction{
			{Index: 0, NodeID: "slow-nav", Type: "navigate", Params: map[string]any{"url": "https://slow.example.test"}},
		},
		CreatedAt: time.Now().UTC(),
	}

	memSink := events.NewMemorySink(contracts.DefaultEventBufferLimits)
	rec := recorder.NewDBRecorder(repo, nil, log)

	// Signal channel to know when the slow step has started
	stepStarted := make(chan struct{})

	// Use a slow session that will be cancelled mid-execution
	slowStub := &slowCancellableEngineWithSignal{
		delay:     500 * time.Millisecond,
		onRunStep: func() { close(stepStarted) },
	}

	exec := NewSimpleExecutor(nil)
	req := Request{
		Plan:              plan,
		EngineName:        slowStub.Name(),
		EngineFactory:     engine.NewStaticFactory(slowStub),
		Recorder:          rec,
		EventSink:         memSink,
		HeartbeatInterval: 0,
	}

	// Create a context that will be cancelled during execution
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after the step has started but before it completes
	go func() {
		<-stepStarted // Wait for step to actually start
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := exec.Execute(ctx, req)

	// 1. Verify we get a cancellation error
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled error, got: %v", err)
	}

	// 2. Verify events were emitted (check events first since they're in-memory)
	evs := memSink.Events()
	var foundFailedEvent bool

	for _, ev := range evs {
		if ev.Kind == contracts.EventKindStepFailed {
			foundFailedEvent = true
			payload, ok := ev.Payload.(map[string]any)
			if !ok {
				t.Fatalf("step_failed event has wrong payload type: %T", ev.Payload)
			}
			outcome, ok := payload["outcome"].(contracts.StepOutcome)
			if !ok {
				t.Fatalf("step_failed event payload outcome is not StepOutcome: %T", payload["outcome"])
			}
			if outcome.Failure == nil {
				t.Error("step_failed event outcome should have failure")
			} else {
				if outcome.Failure.Kind != contracts.FailureKindCancelled {
					t.Errorf("expected cancelled failure kind, got %q", outcome.Failure.Kind)
				}
				if outcome.Failure.Retryable {
					t.Error("cancelled failure should not be retryable")
				}
				if outcome.Failure.Source != contracts.FailureSourceExecutor {
					t.Errorf("expected executor failure source, got %q", outcome.Failure.Source)
				}
			}
			break
		}
	}

	if !foundFailedEvent {
		t.Error("expected step_failed event to be emitted on cancellation")
	}

	// 3. Verify the failure was persisted to the database
	ctx2 := context.Background()
	artifacts, dbErr := repo.ListExecutionArtifacts(ctx2, executionID)
	if dbErr != nil {
		t.Fatalf("failed to list execution artifacts: %v", dbErr)
	}

	// Should have recorded at least one step outcome artifact
	var foundCancelledOutcome bool
	for _, art := range artifacts {
		if art.ArtifactType == "step_outcome" {
			payload, ok := art.Payload["outcome"].(map[string]any)
			if !ok {
				continue
			}
			failure, ok := payload["failure"].(map[string]any)
			if !ok {
				continue
			}
			if kind, ok := failure["kind"].(string); ok && kind == "cancelled" {
				foundCancelledOutcome = true
				// Verify failure taxonomy
				if retryable, ok := failure["retryable"].(bool); ok && retryable {
					t.Error("cancelled failure should not be retryable")
				}
				if source, ok := failure["source"].(string); ok && source != "executor" {
					t.Errorf("cancelled failure source should be 'executor', got %q", source)
				}
				break
			}
		}
	}

	if !foundCancelledOutcome {
		t.Error("expected step_outcome artifact with cancelled failure kind")
	}
}

// slowCancellableEngineWithSignal is a test engine that signals when a step starts
// and introduces delay to simulate slow operations, allowing for controlled cancellation testing.
type slowCancellableEngineWithSignal struct {
	delay     time.Duration
	onRunStep func() // Called when a non-probe step starts
}

func (e *slowCancellableEngineWithSignal) Name() string { return "slow-cancellable-engine" }

func (e *slowCancellableEngineWithSignal) Capabilities(context.Context) (contracts.EngineCapabilities, error) {
	return contracts.EngineCapabilities{
		SchemaVersion:         contracts.CapabilitiesSchemaVersion,
		Engine:                e.Name(),
		MaxConcurrentSessions: 1,
	}, nil
}

func (e *slowCancellableEngineWithSignal) StartSession(context.Context, engine.SessionSpec) (engine.EngineSession, error) {
	return &slowCancellableSessionWithSignal{delay: e.delay, onRunStep: e.onRunStep}, nil
}

type slowCancellableSessionWithSignal struct {
	delay       time.Duration
	onRunStep   func()
	signalFired bool
}

func (s *slowCancellableSessionWithSignal) Run(ctx context.Context, instruction contracts.CompiledInstruction) (contracts.StepOutcome, error) {
	// Handle entry probe immediately
	if instruction.Index == -1 || instruction.NodeID == entryProbeNodeID {
		now := time.Now().UTC()
		return contracts.StepOutcome{
			SchemaVersion:  contracts.StepOutcomeSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			StepIndex:      instruction.Index,
			NodeID:         instruction.NodeID,
			StepType:       instruction.Type,
			Success:        true,
			StartedAt:      now,
			CompletedAt:    &now,
		}, nil
	}

	// Signal that the step has started (only once)
	if s.onRunStep != nil && !s.signalFired {
		s.signalFired = true
		s.onRunStep()
	}

	// Wait for the delay, checking for context cancellation
	select {
	case <-ctx.Done():
		return contracts.StepOutcome{}, ctx.Err()
	case <-time.After(s.delay):
		now := time.Now().UTC()
		return contracts.StepOutcome{
			SchemaVersion:  contracts.StepOutcomeSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			StepIndex:      instruction.Index,
			NodeID:         instruction.NodeID,
			StepType:       instruction.Type,
			Success:        true,
			StartedAt:      now,
			CompletedAt:    &now,
		}, nil
	}
}

func (s *slowCancellableSessionWithSignal) Reset(context.Context) error { return nil }
func (s *slowCancellableSessionWithSignal) Close(context.Context) error { return nil }

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
	if instruction.Index == -1 || instruction.NodeID == entryProbeNodeID {
		now := time.Now().UTC()
		end := now
		return contracts.StepOutcome{
			SchemaVersion:  contracts.StepOutcomeSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			StepIndex:      instruction.Index,
			NodeID:         instruction.NodeID,
			StepType:       instruction.Type,
			Success:        true,
			StartedAt:      now,
			CompletedAt:    &end,
		}, nil
	}
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

func (s *stubRecorder) UpdateCheckpoint(ctx context.Context, executionID uuid.UUID, stepIndex int, totalSteps int) error {
	return nil
}

func ptr[T any](v T) *T {
	return &v
}
