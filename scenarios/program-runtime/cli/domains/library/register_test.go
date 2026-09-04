package library

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
)

func testContract() contractDefinition {
	return contractDefinition{Inputs: map[string]inputSpec{
		"required": {Type: "string", Required: true},
		"window":   {Type: "string", Default: json.RawMessage(`"last_7d"`), Enum: []string{"last_7d", "this_week"}},
	}}
}

func TestParseAndValidateInputsRejectsUnknownKey(t *testing.T) {
	_, err := parseAndValidateInputs("required=ok,unexpected=value", testContract())
	require.EqualError(t, err, `unknown input "unexpected"`)
}

func TestParseAndValidateInputsRejectsMissingRequiredKey(t *testing.T) {
	_, err := parseAndValidateInputs("", testContract())
	require.EqualError(t, err, `missing required input "required"`)
}

func TestParseAndValidateInputsAppliesDefaultsAndEnums(t *testing.T) {
	inputs, err := parseAndValidateInputs("required=ok", testContract())
	require.NoError(t, err)
	require.Equal(t, "ok", inputs["required"])
	require.Equal(t, "last_7d", inputs["window"])

	_, err = parseAndValidateInputs("required=ok,window=invalid", testContract())
	require.EqualError(t, err, "input \"window\" must be one of last_7d, this_week")
}

func TestSplitInputPairsPreservesStructuredJSON(t *testing.T) {
	parts := splitInputPairs(`request={"owner":"fixture","input":{"a":1,"b":["x","y"]}},wait_seconds=2`)
	require.Equal(t, []string{`request={"owner":"fixture","input":{"a":1,"b":["x","y"]}}`, "wait_seconds=2"}, parts)
}

func TestValidateBudgetRequiresAsyncForLongContracts(t *testing.T) {
	_, err := validateBudget(contractDefinition{Budget: struct {
		Async  bool  `json:"async"`
		WallMS int64 `json:"wall_ms"`
	}{WallMS: 120001}})
	require.EqualError(t, err, "contract budget.wall_ms=120001 exceeds the 120000 ms synchronous bound; declare budget.async=true")

	wait, err := validateBudget(contractDefinition{Budget: struct {
		Async  bool  `json:"async"`
		WallMS int64 `json:"wall_ms"`
	}{Async: true, WallMS: 300000}})
	require.NoError(t, err)
	require.EqualValues(t, 300000, wait)
}

func TestLibraryRunStatusAndOutcomeGate(t *testing.T) {
	require.Equal(t, "ok", libraryRunStatus("{'status': 'ok', 'phase': 'report'}"))
	require.Equal(t, "partial", libraryRunStatus("{'status': 'partial'}"))
	require.Empty(t, libraryRunStatus("no envelope"))

	h := &handlers{}
	require.NoError(t, h.runOutcome(nil, &programsv1.SubmitProgramResponse{Program: &programsv1.Program{
		Status: programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, Stdout: "{'status': 'partial'}",
	}}))
	require.EqualError(t, h.runOutcome(nil, &programsv1.SubmitProgramResponse{Program: &programsv1.Program{
		Status: programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, Stdout: "{'status': 'failed'}",
	}}), `library run envelope status "failed"`)
}
