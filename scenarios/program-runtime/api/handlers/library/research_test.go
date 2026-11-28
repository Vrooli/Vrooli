package library

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library"
	programspb "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	"google.golang.org/protobuf/types/known/structpb"

	"program-runtime/internal/contracts"
	"program-runtime/internal/library"
	"program-runtime/internal/programs"
	"program-runtime/internal/sessions"
)

const researchPulseContract = `{
  "name": "research/ux/pulse", "version": "1", "helpers": {"program": "2"},
  "purpose": "Report the requested label from a research topic.",
  "inputs": {"label": {"type": "string", "required": true}},
  "assumptions": [], "invariants": ["No writes."], "bindings": [], "verbs": [],
  "outputs": {"status": {"enum": ["ok", "failed"]}, "phase": {"enum": ["validate", "report"]}, "signals": {"label": "Echoed label."}, "errors": {"classes": ["kernel_runtime"]}, "evidence": "None."},
  "budget": {"wall_ms": 30000, "inference_calls": 0, "delegated_runs": 0, "materialize_limit": 16, "output_bytes": 4096},
  "fixtures": [{"id": "label", "inputs": {"label": "x"}, "expect": {"status": ["ok"]}, "requires": []}], "rung": "S3", "owner_skill": "research-coordination"
}`

const researchPulseSource = `envelope = program.envelope("research/ux/pulse", "1")
envelope["signals"] = {"label": program.inputs()["label"]}
envelope["status"] = "ok"
envelope["phase"] = "report"
program.report()
`

func TestResearchProgramRunsThroughTheKernelByPrefixID(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	require.NoError(t, err)
	home := t.TempDir()
	dir := filepath.Join(home, "topics", "ux", "programs")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pulse.json"), []byte(researchPulseContract), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pulse.py"), []byte(researchPulseSource), 0o644))
	index := contracts.NewIndex()
	index.SetResearchHome(home)
	require.NoError(t, index.Load(root))

	runner := programs.NewSubprocessRunner(filepath.Join(root, "scenarios", "program-runtime", "kernel", "host", "engine.py"))
	defer runner.Close()
	manager := sessions.NewManager(sessions.Options{})
	service := programs.NewService(programs.Options{Runner: runner, AuthorizeSession: manager.AuthorizeSubmission, ValidateSession: func(id string) bool {
		_, err := manager.Get(context.Background(), id)
		return err == nil
	}})
	h := &handler{contracts: index, repoRoot: root, sessions: manager, programs: service}

	got, err := h.GetLibrary(t.Context(), connect.NewRequest(&programsv1.GetLibraryRequest{Name: "research/ux/pulse"}))
	require.NoError(t, err)
	require.Empty(t, got.Msg.GetProgram().GetValidationError())
	require.Equal(t, researchPulseSource, got.Msg.GetProgram().GetSource())

	inputs, err := structpb.NewStruct(map[string]any{"label": "fresh"})
	require.NoError(t, err)
	result, err := h.RunDeclaredProgram(t.Context(), connect.NewRequest(&programsv1.RunDeclaredProgramRequest{
		Name: "research/ux/pulse", Inputs: inputs, Provenance: programspb.Provenance_PROVENANCE_TEST,
	}))
	require.NoError(t, err)
	require.Equal(t, programspb.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, result.Msg.GetProgram().GetStatus(), result.Msg.GetProgram().GetFailureDetail())
	var envelope struct {
		Status  string            `json:"status"`
		Signals map[string]string `json:"signals"`
	}
	require.NoError(t, json.Unmarshal([]byte(result.Msg.GetProgram().GetStdout()), &envelope))
	require.Equal(t, "fresh", envelope.Signals["label"])

	_, err = h.RunDeclaredProgram(t.Context(), connect.NewRequest(&programsv1.RunDeclaredProgramRequest{
		Name: "research/ux/absent", Inputs: inputs, Provenance: programspb.Provenance_PROVENANCE_TEST,
	}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "an unknown research program is NotFound, which lets the CLI fall back to the files")

	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(library.Schema)))
	h.repo = library.NewRepository(db)
	listed, err := h.ListLibrary(t.Context(), connect.NewRequest(&programsv1.ListLibraryRequest{Limit: 500}))
	require.NoError(t, err)
	var found bool
	for _, program := range listed.Msg.GetPrograms() {
		found = found || program.GetName() == "research/ux/pulse"
	}
	require.True(t, found, "library list shows research programs")
}
