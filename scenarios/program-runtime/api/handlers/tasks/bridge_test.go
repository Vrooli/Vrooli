package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	dbtest "github.com/vrooli/api-core/databasetest"
	libraryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	"google.golang.org/protobuf/types/known/structpb"
	libraryH "program-runtime/handlers/library"
	"program-runtime/internal/contracts"
	"program-runtime/internal/library"
	"program-runtime/internal/programs"
	"program-runtime/internal/sessions"
	taskstore "program-runtime/internal/tasks"
)

func archivedFixture(t *testing.T, repo *library.Repository, id, source string, declaration []byte) contracts.Contract {
	t.Helper()
	var raw struct {
		Inputs map[string]struct {
			Type     string
			Required bool
			Default  json.RawMessage
		}
		LearningTask *contracts.LearningTask `json:"learning_task"`
	}
	require.NoError(t, json.Unmarshal(declaration, &raw))
	parts := strings.SplitN(id, ".", 2)
	c := contracts.Contract{ID: id, Scenario: parts[0], Name: parts[1], Source: source, Declaration: declaration, Version: "1", WallMS: 10000, OutputBytes: 65536, Inputs: map[string]contracts.InputSpec{}, LearningTask: raw.LearningTask}
	for name, v := range raw.Inputs {
		c.Inputs[name] = contracts.InputSpec{Type: v.Type, Required: v.Required, Default: v.Default}
	}
	c.Digest = contracts.ContentDigest(declaration, []byte(source), nil)
	require.NoError(t, repo.RetainDeclared(context.Background(), c))
	return c
}

func TestTaskBridgeAllowsMemoryDegradedModeWithoutManifestEdge(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	db.SetMaxOpenConns(1)
	for _, schema := range []string{taskstore.Schema(), library.Schema(), programs.Schema()} {
		_, err := db.Exec(schema)
		require.NoError(t, err)
	}

	store := taskstore.NewStore(db)
	repo := library.NewRepository(db)
	index := contracts.NewIndex()
	declaration := []byte(`{"name":"fixture.learning","version":"1","inputs":{"value":{"type":"string","required":true}},"budget":{"output_bytes":4096},"learning_task":{"scope":"fixture-usage","operation":"fixture.learning","context_fields":["value"],"outcome":{"status_path":"signals.outcome","mapping":{"verified_success":"verified_success","failed":"failed"},"evidence_paths":["evidence"]}}}`)
	contract := archivedFixture(t, repo, "fixture.learning", `print({"signals":{"outcome":"verified_success"},"evidence":["evidence:fixture"]})`, declaration)

	sessionsManager := sessions.NewManager(sessions.Options{})
	session, err := sessionsManager.Create(ctx, "degraded-memory", "", nil)
	require.NoError(t, err)
	programService := programs.NewService(programs.Options{})
	program, err := programService.Submit(ctx, session.ID, "print(1)", programsv1.Provenance_PROVENANCE_AGENT, false)
	require.NoError(t, err)
	bridge := &Bridge{Store: store, Contracts: index, Library: repo, Sessions: sessionsManager, Programs: programService}

	token := "prt_resume_v1_" + strings.Repeat("a", 43)
	postStatus := func(action string, body map[string]any, wantStatus int) map[string]any {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/internal/program-runtime/tasks/"+action, bytes.NewReader(raw))
		res := httptest.NewRecorder()
		bridge.ServeHTTP(res, req)
		require.Equal(t, wantStatus, res.Code, res.Body.String())
		var out map[string]any
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &out))
		return out
	}
	post := func(action string, body map[string]any) map[string]any {
		return postStatus(action, body, http.StatusOK)
	}

	base := map[string]any{"resume_token": token, "session_id": session.ID, "program_id": program.Id, "operation": contract.ID, "digest": contract.Digest, "task_id": "task-degraded", "attempt_id": "attempt-degraded", "inputs": map[string]any{"value": "fixture"}}
	post("begin", base)
	record, err := store.Get(ctx, "attempt-degraded")
	require.NoError(t, err)
	require.Empty(t, record.FinishDigest, "a missing optional Memory program must not refuse checkpoint admission")

	start := map[string]any{"resume_token": token, "session_id": session.ID, "program_id": program.Id, "attempt_id": record.AttemptID, "prepare": map[string]any{"ok": true}}
	post("start", start)
	finishInputs := map[string]any{
		"scope": record.Scope,
		"attempt": map[string]any{
			"attempt_id": record.AttemptID, "task_id": record.TaskID, "context_key": record.ContextKey,
			"provenance": record.Provenance, "operation": contract.LearningTask.Operation,
			"started_at": record.StartedAt, "task_started_at": record.TaskStartedAt,
			"attempt_number": float64(record.AttemptNumber), "outcome": "verified_success",
			"evidence_refs": []any{"evidence:fixture"}, "recall_status": "unavailable", "advice": []any{},
		},
	}
	post("complete", map[string]any{"resume_token": token, "session_id": session.ID, "program_id": program.Id, "attempt_id": record.AttemptID, "result": map[string]any{"signals": map[string]any{"outcome": "verified_success"}, "evidence": []any{"evidence:fixture"}}, "finish_inputs": finishInputs, "outcome": "verified_success"})
	completed, err := store.Get(ctx, record.AttemptID)
	require.NoError(t, err)
	require.Equal(t, "blocked", completed.Delivery)
	require.Equal(t, "memory_not_installed", completed.LastError)

	// A declared must_start edge is the one admission policy that makes the
	// Memory finish program mandatory; the same missing artifact must refuse
	// before creating a second checkpoint.
	root := t.TempDir()
	manifestDir := filepath.Join(root, "scenarios", "fixture", ".vrooli")
	require.NoError(t, os.MkdirAll(manifestDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(manifestDir, "service.json"), []byte(`{"dependencies":{"scenarios":{"vrooli-memory":{"enabled":true,"required":true,"startup_policy":"must_start"}}}}`), 0o644))
	bridge.RepoRoot = root
	postStatus("begin", map[string]any{"resume_token": token, "session_id": session.ID, "program_id": program.Id, "operation": contract.ID, "digest": contract.Digest, "task_id": "task-must-start", "attempt_id": "attempt-must-start", "inputs": map[string]any{"value": "fixture"}}, http.StatusConflict)
}

// This is a full local kernel -> checkpoint HTTP -> SQLite -> pinned shared
// finish path. Only the governed Memory/domain owners are HTTP fixtures.
func TestTaskKernelBridgeDurableDeliveryAndFreshAuthorization(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	db.SetMaxOpenConns(1)
	for _, schema := range []string{taskstore.Schema(), library.Schema(), programs.Schema()} {
		_, err := db.Exec(schema)
		require.NoError(t, err)
	}
	store := taskstore.NewStore(db)
	repo := library.NewRepository(db)
	index := contracts.NewIndex()
	var specs []programs.LibrarySpec
	add := func(c contracts.Contract) {
		specs = append(specs, programs.LibrarySpec{Name: c.Name, Scenario: c.Scenario, Contract: true, Current: true, Source: c.Source, Declaration: c.Declaration, Digest: c.Digest})
	}
	for _, name := range []string{"prepare-attempt", "finish-attempt"} {
		base := filepath.Join("..", "..", "..", "..", "vrooli-memory", ".vrooli", "program-runtime", name)
		source, err := os.ReadFile(base + ".py")
		require.NoError(t, err)
		declaration, err := os.ReadFile(base + ".json")
		require.NoError(t, err)
		add(archivedFixture(t, repo, "vrooli-memory."+name, string(source), declaration))
	}
	declaration := []byte(`{"name":"fixture.child","version":"1","inputs":{},"budget":{"output_bytes":4096},"learning_task":{"scope":"fixture","operation":"execute","context_fields":[],"outcome":{"status_path":"signals.outcome","mapping":{"verified_success":"verified_success","failed":"failed"},"evidence_paths":["evidence"]}}}`)
	child := archivedFixture(t, repo, "fixture.child", `print(fixture.effects.run().head(1)[0])`, declaration)
	add(child)
	declaration = bytes.Replace(declaration, []byte("fixture.child"), []byte("fixture.parent"), 1)
	parent := archivedFixture(t, repo, "fixture.parent", `children = gather(lambda: lib.fixture.child(), lambda: lib.fixture.child())
print(children[0].head(1)[0])`, declaration)
	add(parent)
	genericDeclaration := []byte(`{"name":"fixture.generic","version":"1","inputs":{},"budget":{"output_bytes":4096}}`)
	generic := archivedFixture(t, repo, "fixture.generic", `learn.task(operation="fixture.generic")
print({"status":"ok","evidence":["generic-envelope"]})`, genericDeclaration)
	add(generic)
	var domainCalls, recordCalls atomic.Int32
	var failCapture atomic.Bool
	failCapture.Store(true)
	mux := http.NewServeMux()
	mux.HandleFunc("/internal/program-runtime/bindings/reachability", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"fixture":{"reachable":true},"vrooli-memory":{"reachable":true}}`)
	})
	mux.HandleFunc("/internal/program-runtime/bindings/execute", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		switch body["binding_id"] {
		case "fixture/effects/run":
			domainCalls.Add(1)
			fmt.Fprint(w, `{"rows":[{"status":"ok","signals":{"outcome":"verified_success"},"errors":[],"evidence":["artifact:verified"]}]}`)
		case "vrooli-memory/recall/recall":
			fmt.Fprint(w, `{"hits":[{"entryId":"advice-entry","text":"Use the existing evidence."}]}`)
		case "vrooli-memory/learning/record":
			recordCalls.Add(1)
			if failCapture.Load() {
				http.Error(w, "memory unavailable", 503)
				return
			}
			fmt.Fprint(w, `{"entryId":"capture-entry","existing":false}`)
		default:
			http.Error(w, "unexpected binding", 400)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	kernel, err := filepath.Abs("../../../kernel/host/engine.py")
	require.NoError(t, err)
	runner := programs.NewSubprocessRunnerWithBindings(kernel, []programs.BindingSpec{
		{ID: "fixture/effects/run", Scenario: "fixture", Group: "effects", Command: "run", Effect: "write", Reachable: true, RowsField: "rows"},
		{ID: "vrooli-memory/recall/recall", Scenario: "vrooli-memory", Group: "recall", Command: "recall", Effect: "read", Reachable: true, RowsField: "hits"},
		{ID: "vrooli-memory/learning/record", Scenario: "vrooli-memory", Group: "learning", Command: "record", Effect: "write", Reachable: true},
	}, server.URL+"/internal/program-runtime/bindings/execute", "")
	defer runner.Close()
	runner.SetLibraryProvider(func() []programs.LibrarySpec { return specs })
	manager := sessions.NewManager(sessions.Options{OnReclaimed: runner.KillSession})
	service := programs.NewService(programs.Options{Store: db, Runner: runner, OnTerminal: store.Recover})
	bridge := &Bridge{Store: store, Contracts: index, Library: repo, Sessions: manager, Programs: service}
	mux.Handle("/internal/program-runtime/tasks/", bridge)
	declared := libraryH.DeclaredRunner(nil, index, libraryH.RunDependencies{Repository: repo, Sessions: manager, Programs: service})

	// Direct API calls must enter tasks.run, and nested registered children must
	// inherit the parent's attempt even across gather threads.
	inputs, err := structpb.NewStruct(map[string]any{})
	require.NoError(t, err)
	response, err := declared.RunDeclaredProgram(ctx, connect.NewRequest(&libraryv1.RunDeclaredProgramRequest{Name: parent.ID, ExpectedDigest: parent.Digest, Inputs: inputs, Provenance: programsv1.Provenance_PROVENANCE_TEST}))
	require.NoError(t, err)
	require.Equal(t, programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, response.Msg.Program.Status, response.Msg.Program.FailureDetail)
	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(response.Msg.Program.Stdout), &result))
	learning := result["learning"].(map[string]any)
	require.Equal(t, "matched", learning["recall_status"])
	visible := learning["advice_candidates"].([]any)
	require.Len(t, visible, 1)
	require.Equal(t, "advice-entry", visible[0].(map[string]any)["entry_id"])
	require.Equal(t, "Use the existing evidence.", visible[0].(map[string]any)["text"])
	decisions := learning["advice_decisions"].([]any)
	require.Len(t, decisions, 1)
	require.Equal(t, "unassessed", decisions[0].(map[string]any)["decision"])
	require.Equal(t, "unknown", decisions[0].(map[string]any)["verdict"])
	attemptID := learning["attempt_id"].(string)
	token := learning["resume_token"].(string)
	require.True(t, validReceipt.MatchString(token))
	var persisted string
	require.NoError(t, db.QueryRow("SELECT stdout FROM programs WHERE id=?", response.Msg.Program.Id).Scan(&persisted))
	require.NotContains(t, persisted, token, "live library receipt must never be plaintext in the program database")
	require.Contains(t, persisted, "prt_sealed_v1_")
	require.EqualValues(t, 2, domainCalls.Load())
	pending, err := store.Pending(ctx, time.Now().Add(time.Minute))
	require.NoError(t, err)
	require.Len(t, pending, 1)
	attempt := pending[0].FinishInputs["attempt"].(map[string]any)
	require.Equal(t, "matched", attempt["recall_status"], "%+v", pending[0].Prepare)
	advice := attempt["advice"].([]any)
	require.Len(t, advice, 1)
	require.Equal(t, "unassessed", advice[0].(map[string]any)["decision"])
	require.Equal(t, "unknown", advice[0].(map[string]any)["verdict"])
	require.Equal(t, "verified_success", attempt["outcome"])
	frozen := pending[0].FinishInputs
	drainer := &taskstore.Drainer{Store: store, Deliver: Delivery(declared)}
	require.NoError(t, drainer.DrainOnce(ctx, time.Now().Add(time.Minute)))
	rec, err := store.Get(ctx, attemptID)
	require.NoError(t, err)
	require.Equal(t, "pending", rec.Delivery)
	require.EqualValues(t, 1, recordCalls.Load())
	// Simulate retry exhaustion, reclaim the admitting session, and resume from
	// a new kernel using only the receipt. The result and intent stay immutable.
	_, err = manager.Get(ctx, response.Msg.Program.SessionId)
	require.Error(t, err, "direct library runner reclaims the admitting session")
	_, err = store.Update(ctx, attemptID, func(r *taskstore.Record) error { r.Delivery = "blocked"; r.LastError = "temporary outage"; return nil })
	require.NoError(t, err)
	fresh, err := manager.Create(ctx, "fresh", "", nil)
	require.NoError(t, err)
	call := func(source string, provenance programsv1.Provenance) *programsv1.Program {
		p, err := service.Submit(ctx, fresh.ID, source, provenance, false)
		require.NoError(t, err)
		return p
	}
	p := call(fmt.Sprintf("tasks.get(attempt_id=%q)", attemptID), programsv1.Provenance_PROVENANCE_TEST)
	require.Equal(t, programsv1.ProgramStatus_PROGRAM_STATUS_FAILED, p.Status)
	p = call(fmt.Sprintf("tasks.get(attempt_id=%q, resume_token=%q)", attemptID, token), programsv1.Provenance_PROVENANCE_OPERATOR)
	require.Equal(t, programsv1.ProgramStatus_PROGRAM_STATUS_FAILED, p.Status)
	p = call(fmt.Sprintf("print(tasks.get(attempt_id=%q, resume_token=%q).head(1)[0]['state'])", attemptID, token), programsv1.Provenance_PROVENANCE_TEST)
	require.Equal(t, programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, p.Status, p.FailureDetail)
	require.Contains(t, p.Stdout, "completed")
	p = call(fmt.Sprintf("tasks.resume(attempt_id=%q, resume_token=%q)", attemptID, token), programsv1.Provenance_PROVENANCE_TEST)
	require.Equal(t, programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, p.Status, p.FailureDetail)
	failCapture.Store(false)
	require.NoError(t, drainer.DrainOnce(ctx, time.Now().Add(2*time.Minute)))
	rec, err = store.Get(ctx, attemptID)
	require.NoError(t, err)
	require.Equal(t, "delivered", rec.Delivery, rec.LastError)
	require.Equal(t, frozen, rec.FinishInputs)
	require.EqualValues(t, 2, domainCalls.Load())
	require.EqualValues(t, 2, recordCalls.Load())
	genericResponse, err := declared.RunDeclaredProgram(ctx, connect.NewRequest(&libraryv1.RunDeclaredProgramRequest{Name: generic.ID, ExpectedDigest: generic.Digest, Inputs: inputs, Provenance: programsv1.Provenance_PROVENANCE_AGENT}))
	require.NoError(t, err)
	require.Equal(t, programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, genericResponse.Msg.Program.Status, genericResponse.Msg.Program.FailureDetail)
}

func TestValidateCompletionAcceptsParentFirstLearningTree(t *testing.T) {
	record := &taskstore.Record{AttemptID: "root", TaskID: "task", AttemptNumber: 1, Operation: "fixture.generic", ContextKey: "root-context", Provenance: "agent", StartedAt: "2026-09-09T00:00:00Z", TaskStartedAt: "2026-09-09T00:00:00Z", Scope: "fixture-usage"}
	root := map[string]any{"attempt_id": "root", "task_id": "task", "operation": "fixture.generic", "context_key": "root-context", "provenance": "agent", "started_at": record.StartedAt, "task_started_at": record.TaskStartedAt, "attempt_number": float64(1), "outcome": "verified_success", "evidence_refs": []any{"run:ok"}, "recall_status": "unavailable", "advice": []any{}}
	child := map[string]any{"attempt_id": "child", "task_id": "task", "operation": "fixture.generic", "context_key": "step-context", "provenance": "agent", "started_at": record.StartedAt, "task_started_at": record.TaskStartedAt, "attempt_number": float64(1), "outcome": "unknown", "evidence_refs": []any{}, "recall_status": "no_match", "advice": []any{}, "parent_attempt_id": "root", "step_name": "collect"}
	req := request{Outcome: "verified_success", FinishInputs: map[string]any{"scope": record.Scope, "attempt": root, "attempts": []any{root, child}}}
	require.NoError(t, validateCompletion(record, contracts.Contract{ID: record.Operation}, req))
	child["parent_attempt_id"] = "missing"
	require.Error(t, validateCompletion(record, contracts.Contract{ID: record.Operation}, req))
}

// [REQ:LV-11] Fresh real kernels reuse a portable baseline and durable qualified code
// with no Memory, AI Gateway, Agent Manager, or domain-effect endpoint installed.
func TestPortableLearningAcrossFreshKernelsWithoutOptionalServices(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	db.SetMaxOpenConns(1)
	for _, schema := range []string{taskstore.Schema(), library.Schema(), programs.Schema()} {
		_, err := db.Exec(schema)
		require.NoError(t, err)
	}
	store := taskstore.NewStore(db)
	repo := library.NewRepository(db)
	index := contracts.NewIndex()
	base := filepath.Join("..", "..", "..", ".vrooli", "program-runtime", "learn-verbs-example")
	source, err := os.ReadFile(base + ".py")
	require.NoError(t, err)
	declaration, err := os.ReadFile(base + ".json")
	require.NoError(t, err)
	contract := archivedFixture(t, repo, "program-runtime.learn-verbs-example", string(source), declaration)
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()
	kernel, err := filepath.Abs("../../../kernel/host/engine.py")
	require.NoError(t, err)
	runner := programs.NewSubprocessRunnerWithBindings(kernel, nil, server.URL+"/internal/program-runtime/bindings/execute", "")
	defer runner.Close()
	runner.SetLibraryProvider(func() []programs.LibrarySpec {
		return []programs.LibrarySpec{{Name: contract.Name, Scenario: contract.Scenario, Contract: true, Current: true, Source: contract.Source, Declaration: contract.Declaration, Digest: contract.Digest}}
	})
	manager := sessions.NewManager(sessions.Options{OnReclaimed: runner.KillSession})
	service := programs.NewService(programs.Options{Store: db, Runner: runner, OnTerminal: store.Recover})
	bridge := &Bridge{Store: store, Contracts: index, Library: repo, Sessions: manager, Programs: service}
	mux.Handle("/internal/program-runtime/tasks/", bridge)
	declared := libraryH.DeclaredRunner(nil, index, libraryH.RunDependencies{Repository: repo, Sessions: manager, Programs: service})
	var stepKey string
	for i, value := range []string{"alpha", "beta", "gamma", "delta"} {
		inputs, err := structpb.NewStruct(map[string]any{"value": value, "mode": "baseline"})
		require.NoError(t, err)
		response, err := declared.RunDeclaredProgram(ctx, connect.NewRequest(&libraryv1.RunDeclaredProgramRequest{Name: contract.ID, ExpectedDigest: contract.Digest, Inputs: inputs, Provenance: programsv1.Provenance_PROVENANCE_TEST}))
		require.NoError(t, err)
		require.Equal(t, programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, response.Msg.Program.Status, response.Msg.Program.FailureDetail)
		var envelope map[string]any
		require.NoError(t, json.Unmarshal([]byte(response.Msg.Program.Stdout), &envelope))
		require.Equal(t, "ok", envelope["status"])
		acted := envelope["signals"].(map[string]any)["acted"].(map[string]any)
		require.Equal(t, value, acted["output"].(map[string]any)["value"])
		require.EqualValues(t, 0, acted["model_calls"])
		if i < 3 {
			require.Equal(t, "baseline", acted["fragment_source"])
		} else {
			require.Equal(t, "cache", acted["fragment_source"])
		}
		if stepKey != "" {
			require.Equal(t, stepKey, acted["step_key"])
		}
		stepKey = acted["step_key"].(string)
		var receipt map[string]any
		require.NoError(t, json.Unmarshal([]byte(response.Msg.Program.GetLearningJson()), &receipt))
		require.Equal(t, "blocked", receipt["delivery"])
		require.Equal(t, "memory_not_installed", receipt["last_error"])
	}
	fragment, err := store.BestFragment(ctx, stepKey)
	require.NoError(t, err)
	require.Equal(t, 4, fragment.Verified)
	require.Equal(t, 4, fragment.Contexts)
	require.Empty(t, fragment.TraceInputs)
	require.Equal(t, 1, fragment.CachedRuns)
	// The public transport uses enum names. Test evidence must never qualify live execution.
	inputs, err := structpb.NewStruct(map[string]any{"value": "live", "mode": "baseline"})
	require.NoError(t, err)
	response, err := declared.RunDeclaredProgram(ctx, connect.NewRequest(&libraryv1.RunDeclaredProgramRequest{Name: contract.ID, ExpectedDigest: contract.Digest, Inputs: inputs, Provenance: programsv1.Provenance_PROVENANCE_OPERATOR}))
	require.NoError(t, err)
	var live map[string]any
	require.NoError(t, json.Unmarshal([]byte(response.Msg.Program.Stdout), &live))
	acted := live["signals"].(map[string]any)["acted"].(map[string]any)
	require.Equal(t, "baseline", acted["fragment_source"])
	require.NotEqual(t, stepKey, acted["step_key"])
}

// [REQ:LV-11] Publication writes one reviewed declaration and rejects stale or unattended writes.
func TestBaselinePublicationPreservesSourceAndRequiresCurrentOperatorReview(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	db.SetMaxOpenConns(1)
	for _, schema := range []string{taskstore.Schema(), library.Schema()} {
		_, err := db.Exec(schema)
		require.NoError(t, err)
	}
	store := taskstore.NewStore(db)
	repo := library.NewRepository(db)
	base := filepath.Join("..", "..", "..", ".vrooli", "program-runtime", "learn-verbs-example")
	original, err := os.ReadFile(base + ".json")
	require.NoError(t, err)
	source, err := os.ReadFile(base + ".py")
	require.NoError(t, err)
	var document map[string]any
	require.NoError(t, json.Unmarshal(original, &document))
	baseline := document["learning"].(map[string]any)["baselines"].(map[string]any)["find-account"].(map[string]any)
	delete(document, "learning")
	declaration, err := json.Marshal(document)
	require.NoError(t, err)
	root := t.TempDir()
	path := filepath.Join(root, "example.json")
	require.NoError(t, os.WriteFile(path, declaration, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "example.py"), source, 0o644))
	c := contracts.Contract{ID: "program-runtime.learn-verbs-example", Scenario: "program-runtime", Name: "learn-verbs-example", Declaration: declaration, Source: string(source), SourcePath: path, Version: "1"}
	c.Digest = contracts.ContentDigest(declaration, source, nil)
	require.NoError(t, repo.RetainDeclared(ctx, c))
	f := taskstore.Fragment{StepKey: "fragment-v2:publication", Fragment: baseline["fragment"].(string), Compatibility: baseline["compatibility"].(map[string]any), StepName: "find-account", Source: string(source), Evidence: []string{"postcondition:equal"}}
	for i := 0; i < 5; i++ {
		f.AttemptID = fmt.Sprint(i)
		f.InputDigest = fmt.Sprint(i % 2)
		_, err = store.PutFragment(ctx, f, 1, 0)
		require.NoError(t, err)
	}
	manager := sessions.NewManager(sessions.Options{})
	session, err := manager.Create(ctx, "publication", "", nil)
	require.NoError(t, err)
	service := programs.NewService(programs.Options{})
	bridge := &Bridge{Store: store, Contracts: contracts.NewIndex(), Library: repo, Sessions: manager, Programs: service, RepoRoot: root}
	body := map[string]any{"session_id": session.ID, "operation": c.ID, "digest": c.Digest, "step_key": f.StepKey, "step_name": f.StepName, "min_verified": 5, "baseline": baseline}
	post := func(provenance programsv1.Provenance, want int) {
		program, err := service.Submit(ctx, session.ID, "print(1)", provenance, false)
		require.NoError(t, err)
		body["program_id"] = program.Id
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		response := httptest.NewRecorder()
		bridge.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/internal/program-runtime/tasks/fragment_publish", bytes.NewReader(raw)))
		require.Equal(t, want, response.Code, response.Body.String())
	}
	post(programsv1.Provenance_PROVENANCE_AGENT, http.StatusForbidden)
	unchanged, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, declaration, unchanged)
	post(programsv1.Provenance_PROVENANCE_OPERATOR, http.StatusOK)
	published, err := os.ReadFile(path)
	require.NoError(t, err)
	var result map[string]any
	require.NoError(t, json.Unmarshal(published, &result))
	require.Equal(t, baseline, result["learning"].(map[string]any)["baselines"].(map[string]any)["find-account"])
	unchanged, err = os.ReadFile(filepath.Join(root, "example.py"))
	require.NoError(t, err)
	require.Equal(t, source, unchanged)
	post(programsv1.Provenance_PROVENANCE_OPERATOR, http.StatusConflict)
}

// A promoted fragment must stop counting as a promotion candidate, or the setpoint row that
// counts candidates stays red after the right action and a healthy fleet can never reach band.
func TestMarkPublishedBaselinesFlagsOnlyTheDeclaredFragment(t *testing.T) {
	declared := []contracts.Contract{
		{ID: "demo.search", Declaration: []byte(`{"learning":{"baselines":{"row-normalization":{"fragment":"def step(inputs, bindings):\n    return {'ok': True}"}}}}`)},
		{ID: "demo.unparseable", Declaration: []byte(`not json`)},
		{ID: "demo.undeclared"},
	}
	fragments := []taskstore.Fragment{
		{StepKey: "a", StepName: "row-normalization", Fragment: "def step(inputs, bindings):\n    return {'ok': True}"},
		{StepKey: "b", StepName: "row-normalization", Fragment: "def step(inputs, bindings):\n    return {'ok': False}"},
		{StepKey: "c", StepName: "other-step", Fragment: "def step(inputs, bindings):\n    return {'ok': True}"},
	}
	markPublishedBaselines(declared, fragments)
	require.True(t, fragments[0].Published, "the exact declared fragment is published")
	require.False(t, fragments[1].Published, "a different candidate for the same step is still promotable")
	require.False(t, fragments[2].Published, "identical code under another step name is not this baseline")
}
