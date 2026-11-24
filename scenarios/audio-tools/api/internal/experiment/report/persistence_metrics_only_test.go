//go:build audio_lab

package report

import (
	"encoding/json"
	"testing"

	inteval "audio-tools/internal/eval"

	"github.com/stretchr/testify/require"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
)

func TestExperimentRunsForMetricsOnlyReportOmitsLabelsAndRealizedDetails(t *testing.T) {
	const secret = "PRIVATE-CONDITION-TRANSCRIPT"
	report := inteval.EvalReport{PerStrategy: []inteval.StrategyReport{{
		Strategy: "batch", Label: secret, EngineID: "engine-a", ModelID: "model-b", PolicyProfile: "policy-c", ReplayLane: "realtime", FaultProfile: "clean",
	}}}
	protoReport := &evalv1.EvalReport{PerStrategy: []*evalv1.StrategyReport{{Strategy: "batch"}}}
	runs, err := ExperimentRunsForMetricsOnlyReport(report, protoReport)
	require.NoError(t, err)
	require.Len(t, runs, 1)
	require.Equal(t, "batch", runs[0].Strategy)
	require.NotContains(t, string(runs[0].ConditionJSON), secret)
	require.NotContains(t, string(runs[0].ConditionJSON), "realized")
	var condition map[string]any
	require.NoError(t, json.Unmarshal(runs[0].ConditionJSON, &condition))
	require.Equal(t, "engine-a", condition["engine_id"])
	require.Equal(t, "model-b", condition["model_id"])
	fullRuns, err := ExperimentRunsForReport(report, protoReport, map[string]any{"diagnostic": secret})
	require.NoError(t, err)
	require.Contains(t, string(fullRuns[0].ConditionJSON), secret, "existing persistence behavior remains available")
	_, err = ExperimentRunsForMetricsOnlyReport(report, &evalv1.EvalReport{})
	require.Error(t, err)
}
