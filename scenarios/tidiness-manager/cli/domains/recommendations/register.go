package recommendations

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"tidiness-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "tidiness-manager"

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Recommendations",
		Commands: []cliapp.Command{
			{
				Name:        "recommend-refactors",
				NeedsAPI:    true,
				Description: "Rank reductions by measured expected net lines when calibrated",
				Run: func(args []string) error {
					return run(core, args)
				},
			},
		},
	}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("recommend-refactors")
	scenario := fs.String("scenario", "", "Scenario name")
	limit := fs.Int("limit", 5, "Maximum number of reduction opportunities to return")
	minLines := fs.Int("min-lines", 0, "Minimum unadjusted net lines after replacement")
	format := fs.String("format", "detailed", "Output format: detailed, paths, json")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if *scenario == "" && fs.NArg() > 0 {
		*scenario = fs.Arg(0)
	}
	if *scenario == "" {
		return fmt.Errorf("scenario is required")
	}

	query := map[string]string{
		"scenario": *scenario,
		"limit":    strconv.Itoa(*limit),
	}
	if *minLines > 0 {
		query["min_lines"] = strconv.Itoa(*minLines)
	}

	body, err := core.Get("/agent/refactor-recommendations", support.BuildQuery(query))
	if err != nil {
		return err
	}

	var response support.RecommendationsResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}

	switch *format {
	case "paths":
		for _, rec := range response.Recommendations {
			for _, location := range rec.Locations {
				fmt.Fprintln(os.Stdout, location.Path)
			}
		}
		return nil
	case "json":
		return cliapp.PrintReportJSON(os.Stdout, response)
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Scenario: %s", response.Scenario),
			fmt.Sprintf("Recommendations: %d", response.Count),
			"Ranking: expected net lines when calibrated; otherwise unadjusted net lines",
		},
		ResultsHeading: "Candidates",
		Results:        recommendationRows(response.Recommendations),
		RetrievalHints: []string{fmt.Sprintf("%s recommend-refactors %s --json", cliName, response.Scenario)},
	}
	report.Summary = append(report.Summary, detectorCalibrationSummary(response.Detectors)...)
	report.RetrievalHints = append(report.RetrievalHints, response.Warnings...)
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, response)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

type detectorCalibrationPayload struct {
	ID          string `json:"id"`
	Calibration struct {
		Basis           string   `json:"basis"`
		HistoricalCases int      `json:"historical_cases"`
		Precision       *float64 `json:"precision"`
		Recall          *float64 `json:"recall"`
		HistoricalAudit *struct {
			SourceLabelCount            int `json:"source_label_count"`
			ArchivedPredictionsJoinable int `json:"archived_predictions_joinable"`
		} `json:"historical_audit"`
		EvidenceFrames []struct {
			FrameID              string   `json:"frame_id"`
			Cases                int      `json:"cases"`
			TruePositive         int      `json:"true_positive"`
			FalsePositive        int      `json:"false_positive"`
			FalseNegative        int      `json:"false_negative"`
			Precision            *float64 `json:"precision"`
			Recall               *float64 `json:"recall"`
			CompleteCensus       bool     `json:"complete_census"`
			AdjudicationComplete bool     `json:"adjudication_complete"`
			ResponseComplete     bool     `json:"response_complete"`
			MeasurementMethod    string   `json:"measurement_method"`
			Targets              []struct {
				Scenario   string `json:"scenario"`
				SourceHash string `json:"source_hash"`
				ConfigHash string `json:"config_hash"`
			} `json:"targets"`
		} `json:"evidence_frames"`
	} `json:"calibration"`
}

func detectorCalibrationSummary(rawDetectors []json.RawMessage) []string {
	var rows []string
	for _, raw := range rawDetectors {
		var detector detectorCalibrationPayload
		if err := json.Unmarshal(raw, &detector); err != nil || detector.ID == "" {
			continue
		}
		rows = append(rows, detectorCalibrationRow(detector))
	}
	if len(rows) == 0 {
		return nil
	}
	return append([]string{"Detector calibration (scoped evidence; fleet rates stay unknown unless stated):"}, rows...)
}

func detectorCalibrationRow(detector detectorCalibrationPayload) string {
	calibration := detector.Calibration
	if strings.HasPrefix(calibration.Basis, "not-applicable:") {
		return fmt.Sprintf("%s: precision/recall not applicable", detector.ID)
	}
	if calibration.Precision != nil && calibration.Recall != nil {
		return fmt.Sprintf("%s: fleet precision %.1f%%, recall %.1f%%", detector.ID, 100**calibration.Precision, 100**calibration.Recall)
	}
	var probabilityTargets, respondedTargets, rateTargets, groups int
	var minPrecision, maxPrecision, minRecall, maxRecall float64
	var exactFrames []detectorCalibrationPayloadFrame
	for _, frame := range calibration.EvidenceFrames {
		if strings.HasPrefix(frame.FrameID, "dr1-probability-go-target-") {
			probabilityTargets++
			if frame.ResponseComplete {
				respondedTargets++
				groups += frame.Cases
			}
			if frame.ResponseComplete && frame.Precision != nil && frame.Recall != nil {
				if rateTargets == 0 {
					minPrecision, maxPrecision = *frame.Precision, *frame.Precision
					minRecall, maxRecall = *frame.Recall, *frame.Recall
				} else {
					minPrecision = min(minPrecision, *frame.Precision)
					maxPrecision = max(maxPrecision, *frame.Precision)
					minRecall = min(minRecall, *frame.Recall)
					maxRecall = max(maxRecall, *frame.Recall)
				}
				rateTargets++
			}
			continue
		}
		if frame.CompleteCensus && frame.AdjudicationComplete && frame.ResponseComplete && frame.Precision != nil && frame.Recall != nil {
			exactFrames = append(exactFrames, detectorCalibrationPayloadFrame{
				FrameID: frame.FrameID, Precision: frame.Precision, Recall: frame.Recall,
				MeasurementMethod: frame.MeasurementMethod, Targets: frame.Targets,
			})
		}
	}
	if probabilityTargets > 0 {
		rate := fmt.Sprintf("%d exact target frames with defined rates", rateTargets)
		if rateTargets > 0 && minPrecision == maxPrecision && minRecall == maxRecall {
			rate = fmt.Sprintf("precision %.1f%% and recall %.1f%% on %d exact target frames with defined rates", 100*minPrecision, 100*minRecall, rateTargets)
		} else if rateTargets > 0 {
			rate = fmt.Sprintf("precision %.1f–%.1f%% and recall %.1f–%.1f%% across %d exact target frames", 100*minPrecision, 100*maxPrecision, 100*minRecall, 100*maxRecall, rateTargets)
		}
		return fmt.Sprintf("%s: %s; %d/%d target responses complete (%d labelled groups); fleet precision/recall unknown", detector.ID, rate, respondedTargets, probabilityTargets, groups)
	}
	if len(exactFrames) > 0 {
		frame := exactFrames[len(exactFrames)-1]
		scope := "exact source/config target"
		if len(frame.Targets) > 0 && frame.Targets[0].Scenario != "" {
			target := frame.Targets[0]
			scope = target.Scenario + " exact target"
			if target.SourceHash != "" && target.ConfigHash != "" {
				scope += fmt.Sprintf(" [source %s, config %s]", hashPrefix(target.SourceHash), hashPrefix(target.ConfigHash))
			}
		}
		method := ""
		if strings.Contains(frame.MeasurementMethod, "owner-path replay") {
			method = "; local owner-path replay"
		}
		return fmt.Sprintf("%s: precision %.1f%%, recall %.1f%% for %s%s; fleet precision/recall unknown", detector.ID, 100**frame.Precision, 100**frame.Recall, scope, method)
	}
	sourceLabels := 0
	if calibration.HistoricalAudit != nil {
		sourceLabels = calibration.HistoricalAudit.SourceLabelCount
	}
	if sourceLabels > 0 {
		return fmt.Sprintf("%s: precision/recall unmeasured; %d source-identified selected labels, no representative target census", detector.ID, sourceLabels)
	}
	return fmt.Sprintf("%s: precision/recall unmeasured", detector.ID)
}

func hashPrefix(value string) string {
	if len(value) > 8 {
		return value[:8]
	}
	return value
}

type detectorCalibrationPayloadFrame struct {
	FrameID           string
	Precision         *float64
	Recall            *float64
	MeasurementMethod string
	Targets           []struct {
		Scenario   string `json:"scenario"`
		SourceHash string `json:"source_hash"`
		ConfigHash string `json:"config_hash"`
	}
}

func recommendationRows(recommendations []support.Recommendation) []string {
	if len(recommendations) == 0 {
		return []string{"No refactor candidates found"}
	}
	rows := make([]string, 0, len(recommendations))
	for _, rec := range recommendations {
		reduction := reductionEstimate(rec)
		precision := recommendationPrecision(rec)
		line := fmt.Sprintf("%s | %s | %s | %s", rec.Title, reduction, precision, rec.DetectorID)
		rows = append(rows, line)
	}

	return rows
}

func reductionEstimate(rec support.Recommendation) string {
	if rec.ExpectedNetLines != nil {
		return fmt.Sprintf("%.1f expected net lines", *rec.ExpectedNetLines)
	}
	if strings.HasPrefix(rec.Calibration.Basis, "not-applicable:") {
		return "not a reduction estimate"
	}
	return fmt.Sprintf("unadjusted net %d lines", rec.NetLines)
}

func recommendationPrecision(rec support.Recommendation) string {
	if strings.HasPrefix(rec.Calibration.Basis, "not-applicable:") {
		return "not applicable"
	}
	if rec.Calibration.Precision == nil {
		if audit := rec.Calibration.HistoricalAudit; audit != nil && audit.ArchivedPredictionsJoinable == 0 {
			return fmt.Sprintf("unknown precision; audited %d runs and %d hunt entries, no archived output joined", audit.RunsReviewed, audit.HuntEntriesReviewed)
		}
		return "unknown precision"
	}
	if rec.Calibration.ScopeFrameID != "" {
		return fmt.Sprintf("measured target precision %.0f%% (%d groups)", 100**rec.Calibration.Precision, rec.Calibration.ScopeCases)
	}
	return fmt.Sprintf("historical precision %.0f%% (%d labelled cases)", 100**rec.Calibration.Precision, rec.Calibration.HistoricalCases)
}
