package library

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	libraryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
)

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
