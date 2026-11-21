package review

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBuildVerdictsSubmitsEveryDeclaredMode is the CLI-parity contract for the
// review run: the API refuses a run that omits any failure mode declared by the
// post type or the global review policy, so the CLI must forward every repeated
// --mode value in one request instead of a single verdict.
func TestBuildVerdictsSubmitsEveryDeclaredMode(t *testing.T) {
	modes := []string{
		"internal_vocabulary_leakage",
		"what_without_why",
		"credential_claim_by_persona",
		"fabricated_customer_testimonial",
		"missing_platform_disclosure",
		"real_person_impersonation",
	}
	verdicts := buildVerdicts(modes, nil, true, "evidence/run.log", "")
	require.Len(t, verdicts, len(modes), "every declared mode must be submitted")
	for i, mode := range modes {
		require.Equal(t, mode, verdicts[i].Mode)
		require.True(t, verdicts[i].Passed, "mode %q should pass", mode)
		require.Equal(t, "evidence/run.log", verdicts[i].Evidence)
	}
}

// TestBuildVerdictsMarksBlockedModes proves a mixed run keeps the passing modes
// passing and carries the finding only on the failing modes.
func TestBuildVerdictsMarksBlockedModes(t *testing.T) {
	verdicts := buildVerdicts(
		[]string{"what_without_why", "demo_theater"},
		[]string{"demo_theater"},
		true,
		"evidence/run.log",
		"demo shows an unbuilt pane",
	)
	require.Len(t, verdicts, 2)
	require.Equal(t, "what_without_why", verdicts[0].Mode)
	require.True(t, verdicts[0].Passed)
	require.Empty(t, verdicts[0].Finding)
	require.Equal(t, "demo_theater", verdicts[1].Mode)
	require.False(t, verdicts[1].Passed)
	require.Equal(t, "demo shows an unbuilt pane", verdicts[1].Finding)
}

// TestBuildVerdictsDefaultsBlockedWithoutPassFlag proves the legacy shape
// (no --passed) still records a blocking verdict rather than silently passing.
func TestBuildVerdictsDefaultsBlockedWithoutPassFlag(t *testing.T) {
	verdicts := buildVerdicts([]string{"capability_inflation"}, nil, false, "", "claim exceeds capture")
	require.Len(t, verdicts, 1)
	require.False(t, verdicts[0].Passed)
	require.Equal(t, "claim exceeds capture", verdicts[0].Finding)
}

// TestBuildVerdictsDeduplicatesAndIncludesBlockedOnly proves a mode named in
// both flags is submitted once, and a --blocked-only mode is not dropped.
func TestBuildVerdictsDeduplicatesAndIncludesBlockedOnly(t *testing.T) {
	verdicts := buildVerdicts(
		[]string{"what_without_why", "what_without_why"},
		[]string{"what_without_why", "demo_theater"},
		true,
		"",
		"",
	)
	require.Len(t, verdicts, 2)
	require.Equal(t, "what_without_why", verdicts[0].Mode)
	require.False(t, verdicts[0].Passed, "a mode named in --blocked must fail even when listed in --mode")
	require.Equal(t, "demo_theater", verdicts[1].Mode)
	require.False(t, verdicts[1].Passed)
}
