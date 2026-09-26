package library

import (
	"bytes"
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	"github.com/stretchr/testify/require"
	libraryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	programsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs/programs_v1connect"
)

func TestRepeatedInputsPreserveChannelAndSelectors(t *testing.T) {
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "input"}}}
	ctx, err := cliapp.NewTestRunContextFromArgs(schema, []string{"--input", "channel=test", "--input", "limit=3", "--input", `request={"a":[1,2]}`}, nil, nil, nil)
	require.NoError(t, err)
	inputs, err := declaredInputs(ctx)
	require.NoError(t, err)
	require.Equal(t, "test", inputs["channel"])
	require.Equal(t, float64(3), inputs["limit"])
	require.Equal(t, map[string]any{"a": []any{float64(1), float64(2)}}, inputs["request"])
	duplicate, err := cliapp.NewTestRunContextFromArgs(schema, []string{"--input", "channel=test", "--input", "channel=operator"}, nil, nil, nil)
	require.NoError(t, err)
	_, err = declaredInputs(duplicate)
	require.ErrorContains(t, err, "duplicate input")
	_, err = parseInputPairs("channel=test,channel=operator")
	require.ErrorContains(t, err, "duplicate input")
}

type interruptedWait struct {
	programsconnect.UnimplementedProgramServiceHandler
}

func (interruptedWait) WaitForProgram(context.Context, *connect.Request[programsv1.WaitForProgramRequest]) (*connect.Response[programsv1.WaitForProgramResponse], error) {
	return nil, connect.NewError(connect.CodeUnavailable, errors.New("unexpected EOF"))
}

func TestInterruptedLibraryWaitRetainsIdentityAndSafeRecovery(t *testing.T) {
	_, handler := programsconnect.NewProgramServiceHandler(interruptedWait{})
	server := httptest.NewServer(handler)
	defer server.Close()
	var progress bytes.Buffer
	h := &handlers{programs: programsconnect.NewProgramServiceClient(server.Client(), server.URL), progress: &progress}
	_, err := h.awaitAccepted(&libraryv1.RunDeclaredProgramResponse{Program: &programsv1.Program{Id: "prog_durable", Status: programsv1.ProgramStatus_PROGRAM_STATUS_ACCEPTED}})
	require.ErrorContains(t, err, "unexpected EOF")
	require.ErrorContains(t, err, "program-runtime programs get prog_durable --json")
	require.ErrorContains(t, err, "do not resubmit automatically")
	require.Contains(t, progress.String(), "program accepted: prog_durable")
}

func TestSplitInputPairsPreservesStructuredJSON(t *testing.T) {
	parts := splitInputPairs(`request={"owner":"fixture","input":{"a":1,"b":["x","y"]}},wait_seconds=2`)
	require.Equal(t, []string{`request={"owner":"fixture","input":{"a":1,"b":["x","y"]}}`, "wait_seconds=2"}, parts)
	inputs, err := parseInputPairs(strings.Join(parts, ","))
	require.NoError(t, err)
	require.Equal(t, float64(2), inputs["wait_seconds"])
}

func TestLibraryRunStatusAndOutcomeGate(t *testing.T) {
	require.Equal(t, "ok", libraryRunStatus("{'status': 'ok', 'phase': 'report'}"))
	require.Equal(t, "ok", libraryRunStatus(`{"phase":"report","status":"ok"}`))
	require.Equal(t, "partial", libraryRunStatus("{'status': 'partial'}"))
	require.Empty(t, libraryRunStatus("no envelope"))

	h := &handlers{}
	require.NoError(t, h.runOutcome(nil, &libraryv1.RunDeclaredProgramResponse{Terminal: true, Program: &programsv1.Program{
		Status: programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, Stdout: "{'status': 'partial'}",
	}}))
	require.EqualError(t, h.runOutcome(nil, &libraryv1.RunDeclaredProgramResponse{Terminal: true, Program: &programsv1.Program{
		Status: programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, Stdout: "{'status': 'failed'}",
	}}), `library run envelope status "failed"`)
}
