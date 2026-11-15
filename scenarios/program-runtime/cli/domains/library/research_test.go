package library

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	libraryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library"
	libraryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library/library_v1connect"
)

const researchProgram = "research/ux/pulse"

// researchHome writes one binding-free research program and points the
// standalone runner at the repository copy beside the kernel's helper.
func researchHome(t *testing.T, source string, bindings string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("VROOLI_RESEARCH_HOME", home)
	dir := filepath.Join(home, "topics", "ux", "programs")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	contract := `{"name":"research/ux/pulse","version":"1","helpers":{"program":"2"},"purpose":"Pulse a topic.","inputs":{"label":{"type":"string","default":"x"},"limit":{"type":"integer","required":true}},"bindings":` + bindings + `,"outputs":{"status":{"enum":["ok","failed"]}}}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pulse.json"), []byte(contract), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pulse.py"), []byte(source), 0o644))
	runner, err := filepath.Abs("../../../kernel/host/standalone.py")
	require.NoError(t, err)
	previous := standaloneRunner
	standaloneRunner = func() string { return runner }
	t.Cleanup(func() { standaloneRunner = previous })
	return dir
}

const pulseSource = `import json
envelope = program.envelope("research/ux/pulse", "1")
envelope["signals"] = {"label": program.inputs()["label"], "limit": program.inputs()["limit"]}
envelope["status"] = "ok"
program.report()
`

func runContext(t *testing.T, args ...string) cliapp.OperationContext {
	t.Helper()
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "name", Required: true}}, Flags: []cliapp.Flag{{Name: "expected-digest"}, {Name: "idempotency-key"}, {Name: "admission-deadline"}, {Name: "input"}, {Name: "provenance"}, {Name: "caller-run-id"}, {Name: "caller-agent-profile"}, {Name: "caller-skill-id"}, {Name: "caller-harness"}, {Name: "grant"}, {Name: "version"}}}
	ctx, err := cliapp.NewTestRunContextFromArgs(schema, args, nil, nil, nil)
	require.NoError(t, err)
	return ctx
}

// deadServer returns a client whose API is not listening.
func deadClient(t *testing.T) libraryconnect.LibraryServiceClient {
	t.Helper()
	server := httptest.NewServer(nil)
	url := server.URL
	server.Close()
	return libraryconnect.NewLibraryServiceClient(server.Client(), url)
}

func TestLibraryRunRunsAResearchProgramFromFilesWhenTheAPIIsUnreachable(t *testing.T) {
	researchHome(t, pulseSource, "[]")
	var notes bytes.Buffer
	h := &handlers{client: deadClient(t), progress: &notes}
	response, err := h.run(runContext(t, researchProgram, "--input", "limit=3"))
	require.NoError(t, err)
	require.NoError(t, h.runOutcome(nil, response), "the outcome check is the same as for an API run")
	var envelope struct {
		Status  string         `json:"status"`
		Signals map[string]any `json:"signals"`
	}
	require.NoError(t, json.Unmarshal([]byte(response.GetProgram().GetStdout()), &envelope))
	require.Equal(t, "ok", envelope.Status)
	require.Equal(t, map[string]any{"label": "x", "limit": float64(3)}, envelope.Signals, "declared defaults apply standalone")
	require.Contains(t, notes.String(), "without Program Runtime")
	require.Contains(t, notes.String(), "pulse.json")

	_, err = h.run(runContext(t, researchProgram))
	require.ErrorContains(t, err, "missing required input")
}

func TestLibraryRunRefusesAStandaloneProgramThatNeedsBindings(t *testing.T) {
	researchHome(t, "rows = ai(\"x\")\n", `[{"id":"agent-manager/run/report","effect":"read"}]`)
	h := &handlers{client: deadClient(t)}
	response, err := h.run(runContext(t, researchProgram, "--input", "limit=1"))
	require.NoError(t, err)
	require.Contains(t, response.GetProgram().GetStdout(), "needs Program Runtime for agent-manager/run/report, ai")
	require.Contains(t, response.GetProgram().GetStdout(), "pulse.py and do its steps by hand")
	require.ErrorContains(t, h.runOutcome(nil, response), `"unavailable"`)
}

type refusingLibrary struct {
	libraryconnect.UnimplementedLibraryServiceHandler
	code connect.Code
}

func (s refusingLibrary) RunDeclaredProgram(context.Context, *connect.Request[libraryv1.RunDeclaredProgramRequest]) (*connect.Response[libraryv1.RunDeclaredProgramResponse], error) {
	return nil, connect.NewError(s.code, errors.New("server answer"))
}

func TestLibraryRunNeverRetriesALostResponseLocally(t *testing.T) {
	researchHome(t, pulseSource, "[]")
	_, handler := libraryconnect.NewLibraryServiceHandler(refusingLibrary{code: connect.CodeUnavailable})
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	h := &handlers{client: libraryconnect.NewLibraryServiceClient(server.Client(), server.URL)}
	_, err := h.run(runContext(t, researchProgram, "--input", "limit=1"))
	require.ErrorContains(t, err, "acceptance is unknown", "the API may have accepted it; never run it a second time")

	_, notFound := libraryconnect.NewLibraryServiceHandler(refusingLibrary{code: connect.CodeNotFound})
	unknownTo := httptest.NewServer(notFound)
	t.Cleanup(unknownTo.Close)
	h = &handlers{client: libraryconnect.NewLibraryServiceClient(unknownTo.Client(), unknownTo.URL)}
	response, err := h.run(runContext(t, researchProgram, "--input", "limit=1"))
	require.NoError(t, err, "an API that does not know the program never ran it")
	require.Equal(t, "ok", libraryRunStatus(response.GetProgram().GetStdout()))
}

func TestLibraryRunNamesAnUnreachableAPIForScenarioPrograms(t *testing.T) {
	h := &handlers{client: deadClient(t)}
	_, err := h.run(runContext(t, "program-runtime.setpoint-read"))
	require.ErrorContains(t, err, "nothing was submitted")
}

func TestLibraryGetReadsAResearchProgramFromFilesWhenTheAPIIsUnreachable(t *testing.T) {
	dir := researchHome(t, pulseSource, "[]")
	h := &handlers{client: deadClient(t)}
	got, err := h.get(runContext(t, researchProgram))
	require.NoError(t, err)
	program := got.GetProgram()
	require.Equal(t, researchProgram, program.GetName())
	require.Equal(t, pulseSource, program.GetSource())
	require.Equal(t, []string{"label", "limit"}, program.GetDeclaredInputs())
	real, _ := filepath.EvalSymlinks(filepath.Join(dir, "pulse.json"))
	require.Equal(t, real, program.GetPath())

	require.NoError(t, os.WriteFile(filepath.Join(dir, "pulse.py"), []byte("edited\n"), 0o644))
	again, err := h.get(runContext(t, researchProgram))
	require.NoError(t, err)
	require.Equal(t, "edited\n", again.GetProgram().GetSource(), "every read goes to the file")

	_, err = h.get(runContext(t, "research/ux/absent"))
	require.Error(t, err)
	_, err = h.run(runContext(t, "research/../etc"))
	require.ErrorContains(t, err, "invalid research id")
}
