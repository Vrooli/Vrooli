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
}
