package soak

import (
	"testing"

	"audio-tools/internal/conformance"
	"github.com/stretchr/testify/require"
)

func TestQualificationRunPassedRequiresEveryAssertion(t *testing.T) {
	run := conformance.Run{
		SchemaVersion: conformance.SchemaVersion,
		RunID:         "browser-soak-test",
		Lane:          conformance.LaneRealtime,
		Cell:          conformance.Cell{EngineID: "kyutai", ModelID: "test", Strategy: "passthrough", Policy: "default"},
		Code:          conformance.Code{CapturePackage: "sha256:capture", Server: "sha256:server"},
	}
	for _, name := range conformance.RequiredAssertions(conformance.LaneRealtime) {
		run.Assertions = append(run.Assertions, conformance.Measured(name, true, "focused qualification test"))
	}
	require.True(t, qualificationRunPassed(run, nil))

	run.Assertions[0].Outcome = conformance.OutcomeFailed
	require.False(t, qualificationRunPassed(run, nil))
	require.False(t, qualificationRunPassed(run, requireError{}))
}

type requireError struct{}

func (requireError) Error() string { return "run failed" }
