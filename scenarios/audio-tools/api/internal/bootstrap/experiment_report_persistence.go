//go:build audio_lab

package bootstrap

import (
	"fmt"

	inteval "audio-tools/internal/eval"
	"audio-tools/internal/server"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func reportForPersistence(report inteval.EvalReport, mode experimentv1.ReportRetentionMode) (*evalv1.EvalReport, []byte, error) {
	var persisted *evalv1.EvalReport
	switch mode {
	case experimentv1.ReportRetentionMode_REPORT_RETENTION_MODE_UNSPECIFIED,
		experimentv1.ReportRetentionMode_REPORT_RETENTION_MODE_PRESERVE_TRANSCRIPTS:
		persisted = server.ReportToProto(report)
	case experimentv1.ReportRetentionMode_REPORT_RETENTION_MODE_METRICS_ONLY:
		persisted = server.MetricsOnlyReportToProto(report)
	default:
		return nil, nil, errInvalidReportRetentionMode(mode)
	}
	payload, err := protojson.Marshal(persisted)
	return persisted, payload, err
}

func recipeForCompletedExperiment(recipe *experimentv1.ExperimentRecipe, metricsOnly bool) *experimentv1.ExperimentRecipe {
	if !metricsOnly || recipe == nil {
		return recipe
	}
	persisted := proto.Clone(recipe).(*experimentv1.ExperimentRecipe)
	// Keep submitted inputs available to the in-flight worker, but don't retain
	// the generated long-form ground-truth transcript in the completed recipe.
	persisted.RealizedReference = ""
	return persisted
}

func errInvalidReportRetentionMode(mode experimentv1.ReportRetentionMode) error {
	return fmt.Errorf("unsupported report retention mode %d", mode)
}
