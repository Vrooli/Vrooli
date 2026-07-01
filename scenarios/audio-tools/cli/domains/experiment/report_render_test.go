package experiment

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"

	"audio-tools/cli/internal/testutil"
)

func TestPrintReportTableSurfacesDecisionSignals(t *testing.T) {
	app := testutil.NewTestApp(t, http.NewServeMux())
	ctx, buf := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})

	printReportTable(ctx, &evalv1.EvalReport{
		LatencyMeasured: false,
		Summary: &evalv1.EvalReportSummary{
			Recommendation:  "Prefer batch / clean for this corpus.",
			Confidence:      "low",
			Reasons:         []string{"Lowest WER after deterministic normalization."},
			ConfidenceNotes: []string{"Latency was not measured."},
		},
		Warnings: []*evalv1.ReportWarning{{
			Code:     "latency_not_measured",
			Severity: "info",
			Message:  "Latency columns were not measured because real-time repeats were disabled.",
		}},
		PerStrategy: []*evalv1.StrategyReport{{
			Label:               "batch / clean",
			Wer:                 0.02,
			WhisperCalls:        1,
			Rtf:                 0.4,
			WhisperAudioSeconds: 12,
			Verdict:             "winner",
			Safety:              &evalv1.SafetyGateReport{Passed: true},
			LengthCurves: []*evalv1.LengthBucketCurve{{
				Bucket:              "short",
				ClipCount:           2,
				Wer:                 0.02,
				MaxDroppedSpanWords: 0,
			}},
		}},
	})

	out := buf.String()
	require.Contains(t, out, "Recommendation: Prefer batch / clean for this corpus. (confidence: low)")
	require.Contains(t, out, "VERDICT")
	require.Contains(t, out, "winner")
	require.Contains(t, out, "Warnings:")
	require.Contains(t, out, "info/latency_not_measured")
	require.Contains(t, out, "Length curves:")
	require.Contains(t, out, "short")
	require.NotContains(t, out, "metrics=")
}

func TestPrintComparisonRendersWinnerRowsAndKeepsNilReportVisible(t *testing.T) {
	app := testutil.NewTestApp(t, http.NewServeMux())
	ctx, buf := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})

	mkReport := func(winnerWer, winnerRtf float64, passed bool) *evalv1.EvalReport {
		return &evalv1.EvalReport{
			Summary: &evalv1.EvalReportSummary{WinnerStrategy: "batch"},
			PerStrategy: []*evalv1.StrategyReport{
				{Strategy: "batch", Label: "batch / clean", Wer: winnerWer, Rtf: winnerRtf, WhisperCalls: 1, Safety: &evalv1.SafetyGateReport{Passed: passed}},
				{Strategy: "vad_segment", Label: "vad", Wer: winnerWer + 0.05, Rtf: winnerRtf + 1},
			},
		}
	}

	printComparison(ctx, []*experimentv1.ComparedExperiment{
		{
			Experiment: &experimentv1.Experiment{Id: "exp-aaa", Name: "alpha", Status: experimentv1.ExperimentStatus_EXPERIMENT_STATUS_SUCCEEDED},
			Report:     mkReport(0.08, 0.5, false),
		},
		{
			// Best: lowest winner WER.
			Experiment: &experimentv1.Experiment{Id: "exp-bbb", Name: "beta", Status: experimentv1.ExperimentStatus_EXPERIMENT_STATUS_SUCCEEDED},
			Report:     mkReport(0.02, 0.4, true),
		},
		{
			// Still running — nil report must remain visible.
			Experiment: &experimentv1.Experiment{Id: "exp-ccc", Name: "gamma", Status: experimentv1.ExperimentStatus_EXPERIMENT_STATUS_RUNNING},
			Report:     nil,
		},
	})

	out := buf.String()
	require.Contains(t, out, "exp-aaa")
	require.Contains(t, out, "exp-bbb")
	require.Contains(t, out, "exp-ccc", "nil-report experiment row must still render")
	require.Contains(t, out, "running", "nil-report experiment must show its status")
	require.Contains(t, out, "SAFE")
	require.Contains(t, out, "UNSAFE")
	require.Contains(t, out, "* best")
	// The best (beta) row carries the marker.
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "exp-bbb") {
			require.True(t, strings.HasPrefix(strings.TrimSpace(line), "*"), "best experiment row should be marked: %q", line)
		}
	}
}

func TestFormatRunStatusDoesNotDumpMetricsJSON(t *testing.T) {
	line := formatRunStatus(&experimentv1.ExperimentRun{
		Strategy:      "batch",
		MetricsJson:   `{"wer":0.12}`,
		ConditionJson: `{}`,
	})

	require.Equal(t, "batch - completed", strings.TrimSpace(line))
	require.NotContains(t, line, "metrics=")
}
