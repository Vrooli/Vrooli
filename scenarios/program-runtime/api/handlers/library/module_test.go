package library

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library"

	programspb "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	"google.golang.org/protobuf/types/known/structpb"
	internalbindings "program-runtime/internal/bindings"
	"program-runtime/internal/contracts"
	"program-runtime/internal/library"
	"program-runtime/internal/programs"
	"program-runtime/internal/sessions"
)

type captureRunner struct {
	source       chan string
	materialized bool
}

type heldRunner struct{ release chan struct{} }

type countedDeclaredRunner struct {
	calls   atomic.Int32
	release chan struct{}
}

func (r *countedDeclaredRunner) Execute(context.Context, string, string, bool) (programs.Result, error) {
	r.calls.Add(1)
	if r.release != nil {
		<-r.release
	}
	return programs.Result{Stdout: `{"status":"ok"}`}, nil
}

func TestGetDeclaredExecutionIsReadOnlyAndPreservesOriginalReceipt(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	require.NoError(t, err)
	index := contracts.NewIndex()
	require.NoError(t, index.Load(root))
	contract, ok := index.Get("command-center", "vision-walk-prep")
	require.True(t, ok)
	manager := sessions.NewManager(sessions.Options{})
	runner := &countedDeclaredRunner{}
	service := programs.NewService(programs.Options{Runner: runner})
	h := &handler{contracts: index, sessions: manager, programs: service}
	query := &programsv1.GetDeclaredExecutionRequest{Name: contract.ID, IdempotencyKey: "observe-original"}
	missing, err := h.GetDeclaredExecution(t.Context(), connect.NewRequest(query))
	require.NoError(t, err)
	require.EqualValues(t, 2, missing.Msg.AdmissionContractVersion)
	require.False(t, missing.Msg.Found)
	require.Nil(t, missing.Msg.Program)
	require.Empty(t, manager.List(t.Context()))
	require.Zero(t, runner.calls.Load())
	admitted, err := h.RunDeclaredProgram(t.Context(), connect.NewRequest(&programsv1.RunDeclaredProgramRequest{Name: contract.ID, ExpectedDigest: contract.Digest, IdempotencyKey: query.IdempotencyKey, AdmissionDeadline: time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano), Provenance: programspb.Provenance_PROVENANCE_TEST}))
	require.NoError(t, err)
	for range 2 {
		// Recreating the handler must only observe the same durable identity.
		observed, err := (&handler{programs: service}).GetDeclaredExecution(t.Context(), connect.NewRequest(query))
		require.NoError(t, err)
		require.True(t, observed.Msg.Found)
		require.Equal(t, admitted.Msg.Program.Id, observed.Msg.Program.Id)
		require.Equal(t, admitted.Msg.Program.RequestDigest, observed.Msg.Program.RequestDigest)
		require.Equal(t, admitted.Msg.Program.Status, observed.Msg.Program.Status)
		require.Empty(t, observed.Msg.Program.Source)
	}
	require.EqualValues(t, 1, runner.calls.Load())
	for _, key := range []string{"", " padded", "line\nbreak", strings.Repeat("x", 129)} {
		_, err := h.GetDeclaredExecution(t.Context(), connect.NewRequest(&programsv1.GetDeclaredExecutionRequest{Name: contract.ID, IdempotencyKey: key}))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	}
	require.EqualValues(t, 1, runner.calls.Load())
}

func TestCloseDeclaredAdmissionPreservesAlreadyRunningExecution(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	require.NoError(t, err)
	index := contracts.NewIndex()
	require.NoError(t, index.Load(root))
	contract, ok := index.Get("command-center", "vision-walk-prep")
	require.True(t, ok)
	manager := sessions.NewManager(sessions.Options{})
	runner := &countedDeclaredRunner{release: make(chan struct{})}
	var once sync.Once
	t.Cleanup(func() { once.Do(func() { close(runner.release) }) })
	service := programs.NewService(programs.Options{Runner: runner})
	h := &handler{contracts: index, programs: service, sessions: manager}
	req := &programsv1.RunDeclaredProgramRequest{Name: contract.ID, ExpectedDigest: contract.Digest, IdempotencyKey: "already-running", AdmissionDeadline: time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano), Provenance: programspb.Provenance_PROVENANCE_TEST, Async: true}
	admitted, err := h.RunDeclaredProgram(t.Context(), connect.NewRequest(req))
	require.NoError(t, err)
	closed, err := h.CloseDeclaredAdmission(t.Context(), connect.NewRequest(req))
	require.NoError(t, err)
	require.Equal(t, admitted.Msg.Program.Id, closed.Msg.Program.Id)
	require.False(t, closed.Msg.Terminal, "closure falsely stopped already-admitted work")
	require.Len(t, manager.List(t.Context()), 1)
	once.Do(func() { close(runner.release) })
	terminal, done, err := service.Wait(t.Context(), admitted.Msg.Program.Id, 5*time.Second)
	require.NoError(t, err)
	require.True(t, done)
	require.Equal(t, programspb.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, terminal.Status)
	require.EqualValues(t, 1, runner.calls.Load())
	// An unused key is closed without a session or even a session manager.
	req.IdempotencyKey = "never-dispatched"
	closed, err = (&handler{contracts: index, programs: service}).CloseDeclaredAdmission(t.Context(), connect.NewRequest(req))
	require.NoError(t, err)
	require.True(t, closed.Msg.Terminal)
	require.Empty(t, closed.Msg.Program.SessionId)
	require.Empty(t, closed.Msg.Program.Source)
}

func TestDeclaredGrantsAreExplicitAndLimitedToPinnedDestructiveBindings(t *testing.T) {
	contract := contracts.Contract{Bindings: []contracts.BindingRef{{ID: "workspace-sandbox/change/promote", Effect: "destructive"}, {ID: "test-genie/validation/get", Effect: "read"}}}
	grants, err := declaredGrants(contract, []string{"binding:workspace-sandbox/change/promote", "binding:workspace-sandbox/change/promote"})
	require.NoError(t, err)
	require.Equal(t, []string{"binding:workspace-sandbox/change/promote"}, grants)
	for _, grant := range []string{"effect:destructive", "filesystem:write", "network:internal", "binding:workspace-sandbox/*", "binding:test-genie/validation/get", "binding:unrelated/ops/delete"} {
		_, err := declaredGrants(contract, []string{grant})
		require.Error(t, err, "overbroad or undeclared grant: %s", grant)
	}
	none, err := declaredGrants(contract, nil)
	require.NoError(t, err)
	require.Empty(t, none, "contract declaration must not mint authority")
}

func TestDeclaredKeyConcurrentHandlersReclaimOnlyUnusedSessions(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	require.NoError(t, err)
	index := contracts.NewIndex()
	require.NoError(t, index.Load(root))
	contract, ok := index.Get("command-center", "vision-walk-prep")
	require.True(t, ok)
	const count = 4
	ready, release := make(chan struct{}, count), make(chan struct{})
	runner := &countedDeclaredRunner{release: make(chan struct{})}
	var once, preflightOnce sync.Once
	t.Cleanup(func() { preflightOnce.Do(func() { close(release) }); once.Do(func() { close(runner.release) }) })
	manager := sessions.NewManager(sessions.Options{})
	service := programs.NewService(programs.Options{Runner: runner, Preflight: func(string) []*programspb.Diagnostic { ready <- struct{}{}; <-release; return nil }})
	h := &handler{contracts: index, sessions: manager, programs: service}
	request := &programsv1.RunDeclaredProgramRequest{Name: contract.ID, ExpectedDigest: contract.Digest, IdempotencyKey: "concurrent-handler", AdmissionDeadline: time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano), Provenance: programspb.Provenance_PROVENANCE_TEST, Async: true}
	type result struct {
		response *connect.Response[programsv1.RunDeclaredProgramResponse]
		err      error
	}
	results := make(chan result, count)
	for range count {
		go func() {
			r, e := h.RunDeclaredProgram(t.Context(), connect.NewRequest(request))
			results <- result{r, e}
		}()
	}
	for range count {
		select {
		case <-ready:
		case <-time.After(5 * time.Second):
			t.Fatal("handlers did not rendezvous")
		}
	}
	preflightOnce.Do(func() { close(release) })
	var id, session string
	for range count {
		r := <-results
		require.NoError(t, r.err)
		if id == "" {
			id, session = r.response.Msg.Program.Id, r.response.Msg.Program.SessionId
		}
		require.Equal(t, id, r.response.Msg.Program.Id)
		require.Equal(t, session, r.response.Msg.Program.SessionId)
	}
	require.Len(t, manager.List(t.Context()), 1, "losing admissions leaked sessions")
	_, err = manager.Get(t.Context(), session)
	require.NoError(t, err, "a replay reclaimed the winner's active session")
	once.Do(func() { close(runner.release) })
	request.Async = false
	r, err := h.RunDeclaredProgram(t.Context(), connect.NewRequest(request))
	require.NoError(t, err)
	require.True(t, r.Msg.Terminal)
	require.Equal(t, id, r.Msg.Program.Id)
	require.EqualValues(t, 1, runner.calls.Load())
}

func TestDeclaredKeyInvalidTransportIsRejectedWithoutSessions(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	require.NoError(t, err)
	index := contracts.NewIndex()
	require.NoError(t, index.Load(root))
	contract, ok := index.Get("command-center", "vision-walk-prep")
	require.True(t, ok)
	manager := sessions.NewManager(sessions.Options{})
	runner := &countedDeclaredRunner{}
	h := &handler{contracts: index, sessions: manager, programs: programs.NewService(programs.Options{Runner: runner})}
	for _, missing := range []string{"digest", "deadline", "key"} {
		r := &programsv1.RunDeclaredProgramRequest{Name: contract.ID, ExpectedDigest: contract.Digest, IdempotencyKey: "key", AdmissionDeadline: time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano), Provenance: programspb.Provenance_PROVENANCE_TEST}
		switch missing {
		case "digest":
			r.ExpectedDigest = ""
		case "deadline":
			r.AdmissionDeadline = ""
		case "key":
			r.IdempotencyKey = ""
		}
		_, err := h.RunDeclaredProgram(t.Context(), connect.NewRequest(r))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), missing)
	}
	require.Empty(t, manager.List(t.Context()))
	require.Zero(t, runner.calls.Load())
}

func TestDeclaredRequestKeyReattachesAfterOwnerRestartWithoutRepeatingEffects(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	require.NoError(t, err)
	index := contracts.NewIndex()
	require.NoError(t, index.Load(root))
	contract, ok := index.Get("command-center", "vision-walk-prep")
	require.True(t, ok)
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(t.Context(), db, apidb.SchemaProviderFunc(programs.Schema)))
	runner := &countedDeclaredRunner{}
	newHandler := func() *handler {
		return &handler{contracts: index, sessions: sessions.NewManager(sessions.Options{}), programs: programs.NewService(programs.Options{Store: db, Runner: runner})}
	}
	request := &programsv1.RunDeclaredProgramRequest{Name: contract.ID, ExpectedDigest: contract.Digest, IdempotencyKey: "qualification-attempt-1", AdmissionDeadline: time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano), Provenance: programspb.Provenance_PROVENANCE_TEST}
	first, err := newHandler().RunDeclaredProgram(t.Context(), connect.NewRequest(request))
	require.NoError(t, err)
	require.True(t, first.Msg.Terminal)
	recovered, err := newHandler().RunDeclaredProgram(t.Context(), connect.NewRequest(request))
	require.NoError(t, err)
	require.Equal(t, first.Msg.Program.Id, recovered.Msg.Program.Id, "lost response must recover the retained execution")
	require.EqualValues(t, 1, runner.calls.Load(), "reattachment repeated effects")
}

func (r *heldRunner) Execute(context.Context, string, string, bool) (programs.Result, error) {
	<-r.release
	return programs.Result{Stdout: `{"status":"ok"}`}, nil
}

func TestDeclaredAsyncAcceptanceSurvivesObserverCancellation(t *testing.T) { // [REQ:PRT-P0-010]
	root, err := filepath.Abs("../../../../..")
	require.NoError(t, err)
	index := contracts.NewIndex()
	require.NoError(t, index.Load(root))
	manager := sessions.NewManager(sessions.Options{})
	runner := &heldRunner{release: make(chan struct{})}
	service := programs.NewService(programs.Options{Runner: runner})
	h := &handler{contracts: index, sessions: manager, programs: service}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	response, err := h.RunDeclaredProgram(ctx, connect.NewRequest(&programsv1.RunDeclaredProgramRequest{
		Name: "command-center.vision-walk-prep", Provenance: programspb.Provenance_PROVENANCE_TEST, Async: true,
	}))
	require.NoError(t, err)
	require.NotEmpty(t, response.Msg.GetProgram().GetId())
	require.False(t, response.Msg.Terminal)
	cancel()
	_, err = manager.Get(context.Background(), response.Msg.Program.SessionId)
	require.NoError(t, err, "observation cancellation must not reclaim an active execution")
	close(runner.release)
	program, terminal, err := service.Wait(context.Background(), response.Msg.Program.Id, time.Second)
	require.NoError(t, err)
	require.True(t, terminal)
	require.Equal(t, programspb.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, program.Status)
}

func (r *captureRunner) Execute(_ context.Context, _ string, source string, materialized bool) (programs.Result, error) {
	r.materialized = materialized
	r.source <- source
	return programs.Result{Stdout: "{'status': 'ok'}\n"}, nil
}

func TestListLibraryWithoutQueryIncludesDeclaredContractRows(t *testing.T) {
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db,
		apidb.SchemaProviderFunc(programs.Schema),
		apidb.SchemaProviderFunc(internalbindings.Schema),
		apidb.SchemaProviderFunc(library.Schema)))

	repoRoot, err := filepath.Abs("../../../../..")
	require.NoError(t, err)
	index := contracts.NewIndex()
	require.NoError(t, index.Load(repoRoot))

	h := &handler{repo: library.NewRepository(db), contracts: index}
	response, err := h.ListLibrary(context.Background(), connect.NewRequest(&programsv1.ListLibraryRequest{Limit: 100}))
	require.NoError(t, err)

	var found bool
	for _, program := range response.Msg.GetPrograms() {
		if program.GetName() == "program-runtime.setpoint-read" {
			found = true
			require.Equal(t, "contract", program.GetKind())
			require.Equal(t, "program-runtime", program.GetScenario())
			break
		}
	}
	require.True(t, found, "bare library list must include declared contract rows")
	got, err := h.GetLibrary(context.Background(), connect.NewRequest(&programsv1.GetLibraryRequest{Name: "browser-automation-studio.smoke-flow"}))
	require.NoError(t, err)
	require.Contains(t, got.Msg.GetProgram().GetSource(), "smoke-flow")
	require.Equal(t, "contract", got.Msg.GetProgram().GetKind())
}

func TestRunDeclaredProgramResolvesDefaultsWaitsAndReclaimsSession(t *testing.T) { // [REQ:REQ-P2-010]
	root, err := filepath.Abs("../../../../..")
	require.NoError(t, err)
	index := contracts.NewIndex()
	require.NoError(t, index.Load(root))

	manager := sessions.NewManager(sessions.Options{})
	runner := &captureRunner{source: make(chan string, 1)}
	store := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), store, apidb.SchemaProviderFunc(programs.Schema)))
	service := programs.NewService(programs.Options{
		Store:  store,
		Runner: runner,
		UsageReceipt: func(ctx context.Context, id string) (programs.UsageReceipt, error) {
			session, err := manager.Get(ctx, id)
			if err != nil {
				return programs.UsageReceipt{}, err
			}
			if !strings.HasPrefix(session.Name, "declared-program:") {
				return programs.UsageReceipt{}, nil
			}
			return programs.UsageReceipt{AccountingComplete: true, ChargeMeasured: true, Basis: "dedicated_declared_session"}, nil
		},
		ValidateSession: func(id string) bool {
			_, getErr := manager.Get(context.Background(), id)
			return getErr == nil
		},
	})
	h := &handler{contracts: index, repoRoot: root, sessions: manager, programs: service}
	input, err := structpb.NewStruct(map[string]any{
		"policy": map[string]any{
			"version": "fixture-v1", "event_count_threshold": 3.0,
			"friction_threshold": 0.8, "quiet_seconds": 30.0,
			"event_count_enabled": false, "friction_enabled": false, "terminal_enabled": false,
			"deadline_reached": false, "quiet_reached": false,
			"allowed_actions": []any{"observe", "park"},
		},
		"current_cursor":       "cursor-1",
		"proposed_next_cursor": "cursor-1",
		"allow_inference":      false,
	})
	require.NoError(t, err)
	contract, _ := index.Get("agent-manager", "supervision-evaluate")
	require.Len(t, contract.Digest, 64)
	response, err := h.RunDeclaredProgram(context.Background(), connect.NewRequest(&programsv1.RunDeclaredProgramRequest{
		Name: "agent-manager.supervision-evaluate", ExpectedDigest: contract.Digest, Inputs: input, Provenance: programspb.Provenance_PROVENANCE_TEST,
	}))
	require.NoError(t, err)
	require.True(t, response.Msg.GetTerminal())
	require.Equal(t, programspb.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, response.Msg.GetProgram().GetStatus())
	rows := service.List(context.Background(), response.Msg.GetProgram().GetSessionId(), true)
	require.NotEmpty(t, rows)
	require.Equal(t, "agent-manager.supervision-evaluate", rows[0].GetProgramName()) // [REQ:PRT-P1-006]
	require.Equal(t, contract.Digest, rows[0].GetProgramDigest())
	require.Empty(t, response.Msg.GetProgram().GetSource(), "execution response must not echo executable source")
	require.Less(t, response.Msg.GetWaitedMillis(), int64(1000), "waited time must report elapsed time, not the configured ceiling")
	_, driftErr := h.RunDeclaredProgram(context.Background(), connect.NewRequest(&programsv1.RunDeclaredProgramRequest{Name: "agent-manager.supervision-evaluate", ExpectedDigest: strings.Repeat("0", 64), Inputs: input, Provenance: programspb.Provenance_PROVENANCE_TEST}))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(driftErr))
	source := <-runner.source
	require.Contains(t, source, `\"allow_inference\":false`)
	require.Contains(t, source, `\"events\":[]`)
	require.True(t, strings.Contains(source, "json.loads(") && strings.Contains(source, "supervision-evaluate"))
	_, getErr := manager.Get(context.Background(), response.Msg.GetProgram().GetSessionId())
	require.True(t, errors.Is(getErr, sessions.ErrNotFound))
	restarted := programs.NewService(programs.Options{Store: store})
	retained, err := restarted.Get(context.Background(), response.Msg.GetProgram().GetId())
	require.NoError(t, err)
	require.True(t, retained.GetUsageAccountingComplete())
	require.True(t, retained.GetUsageChargeMeasured())
	require.Equal(t, "dedicated_declared_session", retained.GetUsageBasis())
}

func TestDeclaredBriefingSelectsExpandedOutputTier(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	require.NoError(t, err)
	index := contracts.NewIndex()
	require.NoError(t, index.Load(root))
	manager := sessions.NewManager(sessions.Options{})
	runner := &captureRunner{source: make(chan string, 1)}
	service := programs.NewService(programs.Options{Runner: runner})
	h := &handler{contracts: index, repoRoot: root, sessions: manager, programs: service}
	result, err := h.RunDeclaredProgram(context.Background(), connect.NewRequest(&programsv1.RunDeclaredProgramRequest{Name: "command-center.vision-walk-prep", Provenance: programspb.Provenance_PROVENANCE_TEST}))
	require.NoError(t, err)
	require.True(t, result.Msg.Terminal)
	require.True(t, runner.materialized, "complete briefing must use the declared 64 KB output tier")
}
