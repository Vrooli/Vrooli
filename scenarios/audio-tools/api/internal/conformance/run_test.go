package conformance_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/conformance"
)

func validRun(lane conformance.Lane, assertions ...conformance.Assertion) conformance.Run {
	all := make([]conformance.Assertion, 0, len(assertions)+len(conformance.RequiredAssertions(lane)))
	reported := make(map[string]bool, len(assertions))
	for _, a := range assertions {
		reported[a.Name] = true
		all = append(all, a)
	}
	for _, name := range conformance.RequiredAssertions(lane) {
		if !reported[name] {
			all = append(all, conformance.Measured(name, true, "observed"))
		}
	}
	return conformance.Run{
		SchemaVersion: conformance.SchemaVersion,
		RunID:         "run-1",
		Lane:          lane,
		Profile:       "continuous",
		Cell:          conformance.Cell{EngineID: "whisper-local", ModelID: "base.en", Strategy: "vad", Policy: "default"},
		Code:          conformance.Code{CapturePackage: "sha256:aaa", Server: "sha256:bbb"},
		Assertions:    all,
	}
}

func reasonsContaining(t *testing.T, v conformance.Verdict, substr string) {
	t.Helper()
	for _, r := range v.Reasons {
		if strings.Contains(r, substr) {
			return
		}
	}
	t.Fatalf("no reason contained %q; got %v", substr, v.Reasons)
}

func TestFullyMeasuredRunQualifies(t *testing.T) {
	v := validRun(conformance.LaneAccelerated).Evaluate()
	require.True(t, v.Qualified, "reasons: %v", v.Reasons)
	require.Empty(t, v.Reasons)
}

// The central rule. The previous evidence surface reported
// duplicateCommittedSegments: 0 and silentTerminalOutcomes: 0 as literals that
// nothing computed, and the run read as green.
func TestNotMeasuredFailsTheRun(t *testing.T) {
	run := validRun(conformance.LaneAccelerated,
		conformance.NotMeasured("zero_duplicate_committed_segments", "harness has no segment ledger"))
	v := run.Evaluate()

	require.False(t, v.Qualified, "an unmeasured claim must not qualify")
	reasonsContaining(t, v, "was not measured")
	reasonsContaining(t, v, "harness has no segment ledger")
}

// Silence must be worth exactly as little as an explicit non-measurement,
// otherwise dropping a field becomes the cheapest way to go green.
func TestAbsentRequiredAssertionFailsTheRun(t *testing.T) {
	run := validRun(conformance.LaneAccelerated)
	filtered := run.Assertions[:0]
	for _, a := range run.Assertions {
		if a.Name != "interval_accounting_exactly_once" {
			filtered = append(filtered, a)
		}
	}
	run.Assertions = filtered

	v := run.Evaluate()
	require.False(t, v.Qualified)
	reasonsContaining(t, v, `required assertion "interval_accounting_exactly_once" is absent`)
}

func TestFailedAssertionFailsTheRun(t *testing.T) {
	run := validRun(conformance.LaneAccelerated,
		conformance.Measured("browser_retention_bounded", false, "retained 16 MiB at 524 s"))
	v := run.Evaluate()

	require.False(t, v.Qualified)
	reasonsContaining(t, v, "retained 16 MiB at 524 s")
}

func TestRealtimeLaneRequiresWallClockAssertions(t *testing.T) {
	accelerated := validRun(conformance.LaneAccelerated).Evaluate()
	require.True(t, accelerated.Qualified, "the accelerated lane must not require wall-clock claims")

	// The same assertion set is not enough for a realtime run.
	run := validRun(conformance.LaneAccelerated)
	run.Lane = conformance.LaneRealtime
	v := run.Evaluate()

	require.False(t, v.Qualified)
	for _, name := range conformance.RealtimeAssertions {
		reasonsContaining(t, v, name)
	}
}

// A virtual clock cannot fail to keep up, so a latency pass there is
// manufactured credit rather than evidence.
func TestAcceleratedLaneMayNotClaimWallClockProperties(t *testing.T) {
	run := validRun(conformance.LaneAccelerated,
		conformance.Measured("first_partial_latency_stable", true, "120 ms at minute 1 and minute 60"))
	v := run.Evaluate()

	require.False(t, v.Qualified)
	reasonsContaining(t, v, "only earnable in real time")
}

func TestRunIdentityAndCodeFingerprintAreRequired(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*conformance.Run)
		expect string
	}{
		{"missing run id", func(r *conformance.Run) { r.RunID = "" }, "runId is required"},
		{"missing capture fingerprint", func(r *conformance.Run) { r.Code.CapturePackage = "" }, "capturePackage fingerprint is required"},
		{"missing server fingerprint", func(r *conformance.Run) { r.Code.Server = "" }, "server fingerprint is required"},
		{"missing engine", func(r *conformance.Run) { r.Cell.EngineID = "" }, "cell.engineId is required"},
		{"missing strategy", func(r *conformance.Run) { r.Cell.Strategy = "" }, "cell.strategy is required"},
		{"unknown lane", func(r *conformance.Run) { r.Lane = "guess" }, "is not a recognised lane"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			run := validRun(conformance.LaneAccelerated)
			tc.mutate(&run)
			v := run.Evaluate()
			require.False(t, v.Qualified)
			reasonsContaining(t, v, tc.expect)
		})
	}
}

// Reporting the same claim twice is how a failed observation gets buried under
// a passing duplicate.
func TestDuplicateAssertionFailsTheRun(t *testing.T) {
	run := validRun(conformance.LaneAccelerated)
	run.Assertions = append(run.Assertions, conformance.Measured("interval_accounting_exactly_once", true, "again"))
	v := run.Evaluate()

	require.False(t, v.Qualified)
	reasonsContaining(t, v, "is reported more than once")
}

func TestMeasuredDerivesOutcomeFromObservation(t *testing.T) {
	require.Equal(t, conformance.OutcomePassed, conformance.Measured("x", true, "d").Outcome)
	require.Equal(t, conformance.OutcomeFailed, conformance.Measured("x", false, "d").Outcome)
	require.Equal(t, conformance.OutcomeNotMeasured, conformance.NotMeasured("x", "why").Outcome)
}
