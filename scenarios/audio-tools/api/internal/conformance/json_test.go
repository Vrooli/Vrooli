package conformance_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/conformance"
)

func TestWriteJSONEmitsOneCompleteRunDocument(t *testing.T) {
	run := conformance.Run{
		SchemaVersion: conformance.SchemaVersion,
		RunID:         "run-json-1",
		Lane:          conformance.LaneAccelerated,
		Profile:       "realistic",
		Cell:          conformance.Cell{EngineID: "local", ModelID: "base", Strategy: "passthrough", Policy: "default"},
		Code:          conformance.Code{CapturePackage: "sha256:capture", Server: "sha256:server"},
		Assertions:    []conformance.Assertion{conformance.Measured("interval_accounting_exactly_once", true, "all ranges observed once")},
	}
	var output bytes.Buffer
	require.NoError(t, run.WriteJSON(&output))
	var decoded conformance.Run
	require.NoError(t, json.Unmarshal(output.Bytes(), &decoded))
	require.Equal(t, run.RunID, decoded.RunID)
	require.Contains(t, output.String(), "\"assertions\"")
}

func TestWriteJSONRejectsNilWriter(t *testing.T) {
	var run conformance.Run
	require.Error(t, run.WriteJSON(nil))
}
