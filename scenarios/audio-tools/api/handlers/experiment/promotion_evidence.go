package experiment

import (
	"context"
	"encoding/json"
	"fmt"

	intexp "audio-tools/internal/experiment"
	trustfloor "audio-tools/internal/qualification"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
)

// aggregatePromotionVerdicts overlays a report's per-run verdict with all
// compatible persisted real-time measurements from this machine. It never
// credits fault, recovery, browser, or device gates here: those require their
// own qualification artifacts and stay visible blockers until present.
func (h *connectHandler) aggregatePromotionVerdicts(ctx context.Context, anchor intexp.Experiment, report *evalv1.EvalReport) {
	if report == nil || h.deps.Manager == nil || h.deps.Service == nil {
		return
	}
	experiments, err := h.deps.Manager.List(ctx, intexp.ListFilter{Status: intexp.StatusSucceeded, Limit: 500})
	if err != nil {
		h.deps.Logger.Printf("experiment promotion evidence: list persisted reports: %v", err)
		return
	}

	measurements := make([]trustfloor.ReplayMeasurement, 0, len(report.GetPerStrategy()))
	for _, exp := range experiments {
		if !sameQualificationMachine(anchor.MachineJSON, exp.MachineJSON) {
			continue
		}
		candidate := report
		if exp.ID != anchor.ID {
			var loadErr error
			candidate, loadErr = h.decodedReport(ctx, exp)
			if loadErr != nil {
				h.deps.Logger.Printf("experiment promotion evidence: read report %s: %v", exp.ID, loadErr)
				continue
			}
		}
		measurements = append(measurements, replayMeasurements(candidate)...)
	}
	if len(measurements) == 0 {
		return
	}
	qualification, qualificationErr := h.deps.Service.ListQualificationEvidence(ctx, intexp.QualificationEvidenceFilter{})
	if qualificationErr != nil {
		h.deps.Logger.Printf("experiment promotion evidence: list qualification artifacts: %v", qualificationErr)
	}
	report.PromotionVerdicts = promotionVerdictsForMeasurements(measurements, qualificationMeasurements(anchor.MachineJSON, qualification))
}

func replayMeasurements(report *evalv1.EvalReport) []trustfloor.ReplayMeasurement {
	if report == nil {
		return nil
	}
	measurements := make([]trustfloor.ReplayMeasurement, 0, len(report.GetPerStrategy()))
	for _, row := range report.GetPerStrategy() {
		// Provider promotion is cell-scoped. Historical rows with no strategy
		// identity remain viewable in their report but cannot establish or
		// contaminate a provider-cell trust verdict.
		if row.GetEngineId() == "" || row.GetStrategy() == "" {
			continue
		}
		measurement := trustfloor.ReplayMeasurement{
			EngineID:       row.GetEngineId(),
			ModelID:        row.GetModelId(),
			Strategy:       row.GetStrategy(),
			PolicyProfile:  row.GetPolicyProfile(),
			WER:            row.GetWer(),
			ReplayLane:     row.GetReplayLane(),
			SafetyObserved: row.GetSafety() != nil,
			SafetyPassed:   row.GetSafety().GetPassed(),
		}
		for _, clip := range row.GetPerClip() {
			measurement.ClipDurationsMS = append(measurement.ClipDurationsMS, clip.GetAudioDurationMs())
		}
		measurements = append(measurements, measurement)
	}
	return measurements
}

func qualificationMeasurements(machineJSON []byte, items []intexp.QualificationEvidence) []trustfloor.QualificationMeasurement {
	out := make([]trustfloor.QualificationMeasurement, 0, len(items))
	for _, item := range items {
		if !sameQualificationMachine(machineJSON, item.MachineJSON) {
			continue
		}
		measurement := trustfloor.QualificationMeasurement{
			EngineID: item.EngineID, ModelID: item.ModelID, Strategy: item.Strategy, PolicyProfile: item.PolicyProfile,
			Kind: item.Kind, FaultProfile: item.FaultProfile, Passed: item.Passed,
		}
		if item.Kind == trustfloor.QualificationIntervalAccounting {
			accounting, err := parseIntervalAccountingArtifact(item.MachineJSON)
			if err == nil {
				measurement.AllIntervalsAccounted = accounting.AllIntervalsAccounted
				measurement.DuplicateCommittedSegments = accounting.DuplicateCommittedSegments
				measurement.SilentTerminalOutcomes = accounting.SilentTerminalOutcomes
			}
		}
		out = append(out, measurement)
	}
	return out
}

// intervalAccountingArtifact is deliberately part of the qualification
// machine provenance. A successful replay alone cannot prove that every input
// interval was accounted for, that no durable commit was duplicated, or that a
// terminal outcome was visible to the user.
type intervalAccountingArtifact struct {
	AllIntervalsAccounted      *bool `json:"all_intervals_accounted"`
	DuplicateCommittedSegments *int  `json:"duplicate_committed_segments"`
	SilentTerminalOutcomes     *int  `json:"silent_terminal_outcomes"`
}

type parsedIntervalAccountingArtifact struct {
	AllIntervalsAccounted      bool
	DuplicateCommittedSegments int
	SilentTerminalOutcomes     int
}

func parseIntervalAccountingArtifact(machineJSON []byte) (parsedIntervalAccountingArtifact, error) {
	var raw intervalAccountingArtifact
	if err := json.Unmarshal(machineJSON, &raw); err != nil {
		return parsedIntervalAccountingArtifact{}, err
	}
	if raw.AllIntervalsAccounted == nil || raw.DuplicateCommittedSegments == nil || raw.SilentTerminalOutcomes == nil {
		return parsedIntervalAccountingArtifact{}, fmt.Errorf("interval accounting evidence must include all_intervals_accounted, duplicate_committed_segments, and silent_terminal_outcomes")
	}
	if *raw.DuplicateCommittedSegments < 0 || *raw.SilentTerminalOutcomes < 0 {
		return parsedIntervalAccountingArtifact{}, fmt.Errorf("interval accounting evidence counts cannot be negative")
	}
	return parsedIntervalAccountingArtifact{
		AllIntervalsAccounted:      *raw.AllIntervalsAccounted,
		DuplicateCommittedSegments: *raw.DuplicateCommittedSegments,
		SilentTerminalOutcomes:     *raw.SilentTerminalOutcomes,
	}, nil
}

func promotionVerdictsForMeasurements(measurements []trustfloor.ReplayMeasurement, qualification []trustfloor.QualificationMeasurement) []*evalv1.PromotionVerdict {
	assessed := trustfloor.EvaluatePromotionMeasurements(measurements, qualification, trustfloor.DefaultThresholds)
	verdicts := make([]*evalv1.PromotionVerdict, 0, len(assessed))
	for _, item := range assessed {
		verdicts = append(verdicts, &evalv1.PromotionVerdict{
			EngineId:      item.EngineID,
			ModelId:       item.ModelID,
			Strategy:      item.Strategy,
			PolicyProfile: item.PolicyProfile,
			Stable:        item.Verdict.Stable,
			Reasons:       append([]string(nil), item.Verdict.Reasons...),
		})
	}
	return verdicts
}

type machineIdentity struct {
	Host   string `json:"host"`
	GOOS   string `json:"goos"`
	GOARCH string `json:"goarch"`
}

func sameQualificationMachine(left, right []byte) bool {
	var a, b machineIdentity
	if json.Unmarshal(left, &a) != nil || json.Unmarshal(right, &b) != nil {
		return false
	}
	return a.Host != "" && a.GOOS != "" && a.GOARCH != "" && a == b
}
