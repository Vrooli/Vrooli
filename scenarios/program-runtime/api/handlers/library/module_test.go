package library

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

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

	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../.."))
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
	service := programs.NewService(programs.Options{
		Runner: runner,
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
