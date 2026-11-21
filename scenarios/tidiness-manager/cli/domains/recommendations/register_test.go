package recommendations

import (
	"encoding/json"
	"strings"
	"testing"

	"tidiness-manager/cli/internal/support"
)

func TestRecommendationRowsLabelUncalibratedNetAsUnadjusted(t *testing.T) {
	recommendation := support.Recommendation{Title: "Consolidate duplicated bodies", DetectorID: "duplication.identical-function-body", NetLines: 4}
	recommendation.Calibration.Basis = "uncalibrated"
	rows := recommendationRows([]support.Recommendation{recommendation})
	if len(rows) != 1 || !strings.Contains(rows[0], "unadjusted net 4 lines") || !strings.Contains(rows[0], "unknown precision") || strings.Contains(rows[0], "estimated net") {
		t.Fatalf("uncalibrated recommendation must not claim an estimate: %v", rows)
	}
}

func TestDetectorCalibrationSummaryShowsMeasuredScopesAndUnmeasuredDetectors(t *testing.T) {
	rows := detectorCalibrationSummary([]json.RawMessage{
		json.RawMessage(`{"id":"duplication.identical-function-body","calibration":{"evidence_frames":[{"frame_id":"dr1-probability-go-target-a","cases":5,"precision":1,"recall":1,"response_complete":true},{"frame_id":"dr1-probability-go-target-b","cases":0,"response_complete":false}]}}`),
		json.RawMessage(`{"id":"duplication.identical-function-body.typescript","calibration":{"evidence_frames":[{"frame_id":"typescript-census","cases":19,"precision":1,"recall":0.8947368421052632,"complete_census":true,"adjudication_complete":true,"response_complete":true,"targets":[{"scenario":"browser-automation-studio","source_hash":"source","config_hash":"config"}] }]}}`),
		json.RawMessage(`{"id":"duplication.identical-function-body.javascript","calibration":{"evidence_frames":[{"frame_id":"javascript-census","cases":3,"precision":1,"recall":1,"complete_census":true,"adjudication_complete":true,"response_complete":true,"measurement_method":"local owner-path replay","targets":[{"scenario":"prompt-manager","source_hash":"source","config_hash":"config"}] }]}}`),
		json.RawMessage(`{"id":"migration.unbound-cli-handler","calibration":{"historical_audit":{"source_label_count":8}}}`),
		json.RawMessage(`{"id":"standards.scenario-auditor","calibration":{"basis":"not-applicable: policy findings are not reduction predictions"}}`),
	})
	joined := strings.Join(rows, "\n")
	for _, want := range []string{
		"duplication.identical-function-body: precision 100.0% and recall 100.0% on 1 exact target frames with defined rates; 1/2 target responses complete (5 labelled groups); fleet precision/recall unknown",
		"duplication.identical-function-body.typescript: precision 100.0%, recall 89.5% for browser-automation-studio exact target [source source, config config]; fleet precision/recall unknown",
		"duplication.identical-function-body.javascript: precision 100.0%, recall 100.0% for prompt-manager exact target [source source, config config]; local owner-path replay; fleet precision/recall unknown",
		"migration.unbound-cli-handler: precision/recall unmeasured; 8 source-identified selected labels, no representative target census",
		"standards.scenario-auditor: precision/recall not applicable",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("detector calibration summary missing %q in:\n%s", want, joined)
		}
	}
}

func TestRecommendationRowsExplainUnknownPrecisionFromHistoricalAudit(t *testing.T) {
	recommendation := support.Recommendation{Title: "Consolidate duplicated bodies", DetectorID: "duplication.identical-function-body", NetLines: 4}
	recommendation.Calibration.Basis = "uncalibrated"
	recommendation.Calibration.HistoricalAudit = &support.DetectorHistoricalAudit{
		RunsReviewed: 48, HuntEntriesReviewed: 110, ArchivedPredictionsJoinable: 0,
	}
	rows := recommendationRows([]support.Recommendation{recommendation})
	if len(rows) != 1 || !strings.Contains(rows[0], "audited 48 runs and 110 hunt entries") || !strings.Contains(rows[0], "no archived output joined") {
		t.Fatalf("unknown precision must explain the retained evidence gap: %v", rows)
	}
}

func TestRecommendationRowsKeepPolicyFindingsOutOfReductionEstimates(t *testing.T) {
	recommendation := support.Recommendation{Title: "Policy finding", DetectorID: "standards.scenario-auditor"}
	recommendation.Calibration.Basis = "not-applicable: policy findings are not reduction predictions"
	rows := recommendationRows([]support.Recommendation{recommendation})
	if len(rows) != 1 || !strings.Contains(rows[0], "not a reduction estimate") || !strings.Contains(rows[0], "not applicable") || strings.Contains(rows[0], "net 0 lines") {
		t.Fatalf("policy finding must preserve non-reduction semantics: %v", rows)
	}
}
