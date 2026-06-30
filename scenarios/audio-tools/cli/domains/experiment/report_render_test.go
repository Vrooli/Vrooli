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

func TestFormatRunStatusDoesNotDumpMetricsJSON(t *testing.T) {
	line := formatRunStatus(&experimentv1.ExperimentRun{
		Strategy:      "batch",
		MetricsJson:   `{"wer":0.12}`,
		ConditionJson: `{}`,
	})

	require.Equal(t, "batch - completed", strings.TrimSpace(line))
	require.NotContains(t, line, "metrics=")
}
