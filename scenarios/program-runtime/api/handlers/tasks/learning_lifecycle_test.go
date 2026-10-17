package tasks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

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

// Real kernel processes share evidence; a later correction quarantines even the
// bundled fallback, and a separately verified replacement can restore service.
func TestDelayedFeedbackAcrossFreshKernelsQuarantinesAndRepairs(t *testing.T) {
	t.Setenv("PROGRAM_RUNTIME_LEARNING_STATE_DIR", t.TempDir())
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
	source := `import json
inputs = program.inputs()
learn.task(scope='fixture-usage', operation='fixture.feedback-lifecycle', key='same-task')
if inputs['mode'] == 'feedback':
    receipt = learn.feedback(inputs['feedback_ref'], 'contradicted', ['downstream:wrong-result'], dimension='verification')
    learn.outcome('verified_success', ['feedback:retained'])
    print(json.dumps({'status': 'ok', 'receipt': receipt}))
else:
    fragment = "def step(inputs, bindings):\n    return {'value': inputs['value']}"
    if inputs['mode'] == 'repair':
        fragment = "def step(inputs, bindings):\n    return dict(value=inputs['value'])"
    options = {'bindings': []}
    if inputs['mode'] == 'capability':
        options = {'capabilities': ['read a demo record']}
    try:
        result = learn.act('echo', 'echo value', {'value': inputs['value']}, {'type': 'object'}, **options,
            fallback_fragment=fragment, allow_ai=False, min_verified=1, min_contexts=1,
            verifier_revision='equals-v1', verify=lambda output: ('verified_success', ['postcondition:equal']) if output == {'value': inputs['value']} else ('failed', ['postcondition:mismatch']))
        learn.outcome('verified_success', ['task:equal'])
        print(json.dumps({'status': 'ok', 'result': result}))
    except Exception as exc:
        learn.outcome('failed', ['adaptation:unavailable'])
        print(json.dumps({'status': 'failed', 'reason': str(exc)}))
`
	declaration := []byte(`{"name":"fixture.feedback-lifecycle","version":"1","inputs":{"mode":{"type":"string","required":true},"value":{"type":"string","default":"value"},"feedback_ref":{"type":"string","default":""}},"budget":{"output_bytes":65536},"learning":{"capabilities":[{"scenario":"demo","effects":["read"]}]}}`)
	contract := archivedFixture(t, repo, "fixture.feedback-lifecycle", source, declaration)
	publicBase := filepath.Join("..", "..", "..", ".vrooli", "program-runtime", "learning-feedback")
	publicSource, err := os.ReadFile(publicBase + ".py")
	require.NoError(t, err)
	publicDeclaration, err := os.ReadFile(publicBase + ".json")
	require.NoError(t, err)
	publicContract := archivedFixture(t, repo, "program-runtime.learning-feedback", string(publicSource), publicDeclaration)

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()
	kernel, err := filepath.Abs("../../../kernel/host/engine.py")
	require.NoError(t, err)
	runner := programs.NewSubprocessRunnerWithBindings(kernel, []programs.BindingSpec{{ID: "demo/records/read", Scenario: "demo", Group: "records", Command: "read", Effect: "read"}}, server.URL+"/internal/program-runtime/bindings/execute", "")
	defer runner.Close()
	runner.SetDiscoveryURL(server.URL + "/discovery")
	mux.HandleFunc("/discovery", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"binding_id": "demo/records/read"}}))
	})
	runner.SetLibraryProvider(func() []programs.LibrarySpec {
		return []programs.LibrarySpec{
			{Name: contract.Name, Scenario: contract.Scenario, Contract: true, Current: true, Source: contract.Source, Declaration: contract.Declaration, Digest: contract.Digest},
			{Name: publicContract.Name, Scenario: publicContract.Scenario, Contract: true, Current: true, Source: publicContract.Source, Declaration: publicContract.Declaration, Digest: publicContract.Digest},
		}
	})
	manager := sessions.NewManager(sessions.Options{OnReclaimed: runner.KillSession})
	service := programs.NewService(programs.Options{Store: db, Runner: runner, OnTerminal: store.Recover})
	bridge := &Bridge{Store: store, Contracts: index, Library: repo, Sessions: manager, Programs: service}
	var resultDeliveryUnavailable atomic.Bool
	var fragmentReadsUnavailable atomic.Bool
	resultDeliveryUnavailable.Store(true)
	mux.HandleFunc("/internal/program-runtime/tasks/", func(w http.ResponseWriter, r *http.Request) {
		if resultDeliveryUnavailable.Load() && r.URL.Path == "/internal/program-runtime/tasks/learning_result_put" {
			http.Error(w, "transient result delivery outage", http.StatusServiceUnavailable)
			return
		}
		if fragmentReadsUnavailable.Load() && r.URL.Path == "/internal/program-runtime/tasks/fragment_get" {
			http.Error(w, "fragment store unavailable", http.StatusServiceUnavailable)
			return
		}
		bridge.ServeHTTP(w, r)
	})
	declared := libraryH.DeclaredRunner(nil, index, libraryH.RunDependencies{Repository: repo, Sessions: manager, Programs: service})
	run := func(mode, reference string) map[string]any {
		input, err := structpb.NewStruct(map[string]any{"mode": mode, "value": mode, "feedback_ref": reference})
		require.NoError(t, err)
		response, err := declared.RunDeclaredProgram(ctx, connect.NewRequest(&libraryv1.RunDeclaredProgramRequest{Name: contract.ID, ExpectedDigest: contract.Digest, Inputs: input, Provenance: programsv1.Provenance_PROVENANCE_TEST}))
		require.NoError(t, err)
		require.Equal(t, programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, response.Msg.Program.Status, response.Msg.Program.FailureDetail)
		var envelope map[string]any
		require.NoError(t, json.Unmarshal([]byte(response.Msg.Program.Stdout), &envelope))
		return envelope
	}
	initial := run("first", "")
	require.Equal(t, "ok", initial["status"], initial)
	first := initial["result"].(map[string]any)
	require.Equal(t, "fallback", first["fragment_source"])
	resultDeliveryUnavailable.Store(false)
	cached := run("second", "")
	require.Equal(t, "cache", cached["result"].(map[string]any)["fragment_source"])
	feedbackInputs, err := structpb.NewStruct(map[string]any{"feedback_ref": first["feedback_ref"], "disposition": "contradicted", "dimension": "verification", "evidence": []any{"user:later-correction"}})
	require.NoError(t, err)
	graded, err := declared.RunDeclaredProgram(ctx, connect.NewRequest(&libraryv1.RunDeclaredProgramRequest{Name: publicContract.ID, ExpectedDigest: publicContract.Digest, Inputs: feedbackInputs, Provenance: programsv1.Provenance_PROVENANCE_TEST}))
	require.NoError(t, err)
	require.Equal(t, programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, graded.Msg.Program.Status, graded.Msg.Program.FailureDetail)
	var feedbackEnvelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(graded.Msg.Program.Stdout), &feedbackEnvelope))
	require.Equal(t, "ok", feedbackEnvelope["status"], feedbackEnvelope)
	target, err := store.LearningResultStatus(ctx, first["feedback_ref"].(string), nil, "test")
	require.NoError(t, err)
	require.Equal(t, "fixture-usage", target["scope"])
	require.Equal(t, true, target["contradicted"])
	quarantined, err := store.BestFragment(ctx, first["step_key"].(string))
	require.Error(t, err, "contradicted persisted fragment must be excluded from best candidates")
	require.Nil(t, quarantined)
	var contradictions int
	require.NoError(t, db.QueryRow("SELECT contradicted_since_edit FROM learning_fragments WHERE step_key=?", first["step_key"]).Scan(&contradictions))
	require.Equal(t, 1, contradictions)
	fragmentReadsUnavailable.Store(true)
	rejected := run("third", "")
	require.Equal(t, "failed", rejected["status"], rejected)
	repaired := run("repair", "")
	require.Equal(t, "ok", repaired["status"], repaired)
	require.Equal(t, "fallback", repaired["result"].(map[string]any)["fragment_source"])
	require.EqualValues(t, 0, repaired["result"].(map[string]any)["model_calls"])
	fragmentReadsUnavailable.Store(false)
	capable := run("capability", "")
	require.Equal(t, "ok", capable["status"], capable)
	require.EqualValues(t, 0, capable["result"].(map[string]any)["model_calls"])
}
