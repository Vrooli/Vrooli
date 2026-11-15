package authoring

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
)

func TestAuthoringEvalGateRequiresMeasuredStatusAndFloor(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		floorMet bool
		passes   bool
	}{
		{name: "measured above floor", status: "measured", floorMet: true, passes: true},
		{name: "measured below floor", status: "measured", floorMet: false, passes: false},
		{name: "partial", status: "partial", floorMet: true, passes: false},
		{name: "unavailable", status: "unavailable", floorMet: false, passes: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.passes, authoringEvalPasses(test.status, test.floorMet))
		})
	}
}

func TestWriteAuthoringReceiptIsCreateExclusiveAndHasRawDigest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "receipt.json")
	response := &programsv1.RunAuthoringEvalResponse{Suite: "evals/authoring.primary.json", Status: "measured", Met: 10}

	digest, err := writeAuthoringReceipt(path, response)
	require.NoError(t, err)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Equal(t, "evals/authoring.primary.json", decoded["suite"])
	require.Equal(t, "measured", decoded["status"])
	sum := sha256.Sum256(data)
	require.Equal(t, hex.EncodeToString(sum[:]), digest)

	mode, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), mode.Mode().Perm())

	_, err = writeAuthoringReceipt(path, response)
	require.Error(t, err, "an existing receipt must never be replaced")
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, data, after)
}

func TestResolveAuthoringSuitePathMakesRelativePathsAbsolute(t *testing.T) {
	resolved, err := resolveAuthoringSuitePath("scenarios/program-runtime/evals/authoring.pr6-comparison.json")
	require.NoError(t, err)
	require.True(t, filepath.IsAbs(resolved))
	require.True(t, strings.HasSuffix(filepath.ToSlash(resolved), "/scenarios/program-runtime/evals/authoring.pr6-comparison.json"))

	defaultPath, err := resolveAuthoringSuitePath("  ")
	require.NoError(t, err)
	require.Empty(t, defaultPath)
}

func TestAuthoringScoreSummaryReportsMeasuredDenominatorAndUnavailableCases(t *testing.T) {
	response := &programsv1.RunAuthoringEvalResponse{
		Cases: 4, Met: 2, Unavailable: 1, NotAttempted: 1, Floor: 3,
		Results: []*programsv1.AuthoringCaseResult{
			{CaseId: "pass", Authored: true, FirstAttemptOk: true, CaseStatus: "measured", AttemptCount: 1},
			{CaseId: "pass-too", Authored: true, FirstAttemptOk: true, CaseStatus: "measured", AttemptCount: 1},
			{CaseId: "route-unavailable", CaseStatus: "unavailable", AttemptCount: 1},
			{CaseId: "not-attempted", CaseStatus: "not_attempted"},
		},
	}
	summary := authoringScoreSummary(response, "partial")
	require.Contains(t, summary, "first_attempt=2/2 attempted (fixed tasks=4)")
	require.Contains(t, summary, "unavailable=1")
	require.Contains(t, summary, "not_attempted=1")
	require.Contains(t, summary, "first-attempt time-to-working (n=0): median/p90 unavailable")
}

func TestDurationPercentilesUseMedianAndNearestRankP90(t *testing.T) {
	median, p90 := durationPercentiles([]int64{50, 10, 40, 20, 30})
	require.EqualValues(t, 30, median)
	require.EqualValues(t, 50, p90)

	median, p90 = durationPercentiles([]int64{8, 4, 6, 2})
	require.EqualValues(t, 5, median)
	require.EqualValues(t, 8, p90)
}

func TestAuthoringScoreSummarySeparatesHeldoutPanelAndWorkingTimes(t *testing.T) {
	primaryWorking := int64(10)
	heldoutWorking := int64(20)
	response := &programsv1.RunAuthoringEvalResponse{
		Cases: 4, Met: 2, NotAttempted: 1, Floor: 0,
		Results: []*programsv1.AuthoringCaseResult{
			{CaseId: "single-binding-read", FirstAttemptOk: true, CaseStatus: "measured", AttemptCount: 1, ElapsedToFirstWorkingMs: &primaryWorking},
			{CaseId: "heldout-binding-shape", FirstAttemptOk: true, CaseStatus: "measured", AttemptCount: 1, ElapsedToFirstWorkingMs: &heldoutWorking},
			{CaseId: "heldout-owner-miss", CaseStatus: "measured", AttemptCount: 1},
			{CaseId: "heldout-not-run", CaseStatus: "not_attempted"},
		},
	}
	summary := authoringScoreSummary(response, "partial")
	require.Contains(t, summary, "first_attempt=2/3 attempted (fixed tasks=4)")
	require.Contains(t, summary, "primary=1/1 attempted (cases=1)")
	require.Contains(t, summary, "heldout=1/2 attempted (cases=3)")
	require.Contains(t, summary, "first-attempt time-to-working (n=2): median=15ms p90=20ms")
	require.Contains(t, summary, "primary time-to-working (n=1): median=10ms p90=10ms")
	require.Contains(t, summary, "heldout time-to-working (n=1): median=20ms p90=20ms")
}

func TestAuthoringCaseTimingRetainsRouteIndependentDigestsAndAttempts(t *testing.T) {
	working := int64(64)
	item := &programsv1.AuthoringCaseResult{
		CaseId: "heldout-binding-shape", FirstAttemptOk: true,
		SourceSha256: "source-digest", TaskSha256: "task-digest",
		GenerationElapsedMs: 11, ExecutionElapsedMs: 22, OracleElapsedMs: 3,
		ElapsedToFirstWorkingMs: &working, CaseStatus: "measured", AttemptCount: 1,
		RetryCount: 0,
	}
	detail := authoringCaseTiming(item)
	for _, expected := range []string{
		"generation_ms=11", "execution_ms=22", "oracle_ms=3", "time_to_working_ms=64",
		"case_status=measured", "attempts=1", "retries=0", "source_sha256=source-digest", "task_sha256=task-digest",
	} {
		require.Contains(t, detail, expected)
	}
}

func TestAuthoringCaseTimingDoesNotInventDurationsForUnattemptedCase(t *testing.T) {
	item := &programsv1.AuthoringCaseResult{CaseId: "heldout-not-run", CaseStatus: "not_attempted"}
	detail := authoringCaseTiming(item)
	require.Contains(t, detail, "case_status=not_attempted")
	require.Contains(t, detail, "attempts=0")
	require.Contains(t, detail, "retries=0")
	require.NotContains(t, detail, "generation_ms=")
	require.NotContains(t, detail, "execution_ms=")
	require.NotContains(t, detail, "oracle_ms=")
}
