package componenttests

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseContractAcceptsOnlyDeclarativeComponentTrace(t *testing.T) {
	contract, err := ParseContract([]byte(`{"schemaVersion":"1","examples":[{"example":"idle","actions":[{"kind":"click","target":"voice-control"}],"assertions":[{"kind":"role","role":"button","name":"Start voice input"}],"claims":["action-present"]}]}`), "component")
	require.NoError(t, err)
	require.Equal(t, "idle", contract.Examples[0].Example)
}

func TestParseContractRejectsUnsafeAndWrongKindInput(t *testing.T) {
	_, err := ParseContract([]byte(`{"schemaVersion":"1","examples":[{"example":"idle","actions":[{"kind":"shell","target":"rm -rf /"}],"assertions":[{"kind":"role","role":"button","name":"go"}]}]}`), "component")
	var validation ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, "unsupported_action", validation.Code)

	_, err = ParseContract([]byte(`{"schemaVersion":"1","fixtures":[{"name":"start","assertions":[{"kind":"state","target":"status","value":"recording"}]}]}`), "component")
	require.ErrorAs(t, err, &validation)
	require.Equal(t, "hook_fixture_on_component", validation.Code)
	require.False(t, errors.Is(err, nil))
}

func TestParseContractRejectsExecutableLookingNamesAndUnboundedWait(t *testing.T) {
	_, err := ParseContract([]byte(`{"schemaVersion":"1","examples":[{"example":"../../escape","actions":[{"kind":"wait","durationMs":5001}],"assertions":[{"kind":"text","value":"safe"}]}]}`), "component")
	var validation ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, "invalid_trace_name", validation.Code)
}

func TestParseContractRejectsDuplicateClaimReferences(t *testing.T) {
	_, err := ParseContract([]byte(`{"schemaVersion":"1","examples":[{"example":"idle","assertions":[{"kind":"text","value":"Ready"}],"claims":["action","action"]}]}`), "component")
	var validation ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, "duplicate_claim_reference", validation.Code)
}
