package experiment

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	intexp "audio-tools/internal/experiment"
	"audio-tools/internal/stt/trustfloor"

	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func domainToProto(exp intexp.Experiment) *experimentv1.Experiment {
	out := &experimentv1.Experiment{
		Id:          exp.ID,
		Name:        exp.Name,
		Status:      statusToProto(exp.Status),
		Recipe:      recipeFromJSON(exp.RecipeJSON),
		Error:       exp.Error,
		ResultRef:   exp.ResultRef,
		MachineJson: string(exp.MachineJSON),
		CreatedAt:   timestamppb.New(exp.CreatedAt),
	}
	if exp.StartedAt != nil {
		out.StartedAt = timestamppb.New(*exp.StartedAt)
	}
	if exp.FinishedAt != nil {
		out.FinishedAt = timestamppb.New(*exp.FinishedAt)
	}
	return out
}

func runToProto(run intexp.Run) *experimentv1.ExperimentRun {
	out := &experimentv1.ExperimentRun{
		Id:            run.ID,
		ExperimentId:  run.ExperimentID,
		Strategy:      run.Strategy,
		ConditionJson: string(run.ConditionJSON),
		CreatedAt:     timestamppb.New(run.CreatedAt),
	}
	var condition struct {
		EngineID      string `json:"engine_id"`
		PolicyProfile string `json:"policy_profile"`
		ReplayLane    string `json:"replay_lane"`
		FaultProfile  string `json:"fault_profile"`
	}
	if err := json.Unmarshal(run.ConditionJSON, &condition); err == nil {
		out.EngineId = condition.EngineID
		out.PolicyProfile = condition.PolicyProfile
		out.ReplayLane = replayLaneFromCondition(condition.ReplayLane)
		out.FaultProfile = condition.FaultProfile
	}
	return out
}

func replayLaneFromCondition(value string) experimentv1.ReplayLane {
	switch value {
	case "deterministic":
		return experimentv1.ReplayLane_REPLAY_LANE_DETERMINISTIC
	case "realtime":
		return experimentv1.ReplayLane_REPLAY_LANE_REALTIME
	case "product_path":
		return experimentv1.ReplayLane_REPLAY_LANE_PRODUCT_PATH
	default:
		return experimentv1.ReplayLane_REPLAY_LANE_UNSPECIFIED
	}
}

func eventToProto(ev intexp.ProgressEvent) *experimentv1.ExperimentEvent {
	return &experimentv1.ExperimentEvent{
		ExperimentId: ev.ExperimentID,
		Status:       statusToProto(ev.Status),
		Progress:     int32(ev.Progress),
		Message:      ev.Message,
		At:           timestamppb.New(ev.At),
	}
}

func statusToProto(s intexp.Status) experimentv1.ExperimentStatus {
	switch s {
	case intexp.StatusQueued:
		return experimentv1.ExperimentStatus_EXPERIMENT_STATUS_QUEUED
	case intexp.StatusRunning:
		return experimentv1.ExperimentStatus_EXPERIMENT_STATUS_RUNNING
	case intexp.StatusSucceeded:
		return experimentv1.ExperimentStatus_EXPERIMENT_STATUS_SUCCEEDED
	case intexp.StatusFailed:
		return experimentv1.ExperimentStatus_EXPERIMENT_STATUS_FAILED
	case intexp.StatusCanceled:
		return experimentv1.ExperimentStatus_EXPERIMENT_STATUS_CANCELED
	default:
		return experimentv1.ExperimentStatus_EXPERIMENT_STATUS_UNSPECIFIED
	}
}

func statusFromProto(s experimentv1.ExperimentStatus) intexp.Status {
	switch s {
	case experimentv1.ExperimentStatus_EXPERIMENT_STATUS_QUEUED:
		return intexp.StatusQueued
	case experimentv1.ExperimentStatus_EXPERIMENT_STATUS_RUNNING:
		return intexp.StatusRunning
	case experimentv1.ExperimentStatus_EXPERIMENT_STATUS_SUCCEEDED:
		return intexp.StatusSucceeded
	case experimentv1.ExperimentStatus_EXPERIMENT_STATUS_FAILED:
		return intexp.StatusFailed
	case experimentv1.ExperimentStatus_EXPERIMENT_STATUS_CANCELED:
		return intexp.StatusCanceled
	default:
		return ""
	}
}

func recipeFromJSON(data []byte) *experimentv1.ExperimentRecipe {
	recipe := &experimentv1.ExperimentRecipe{}
	if len(data) == 0 {
		return recipe
	}
	if err := protojson.Unmarshal(data, recipe); err != nil {
		return &experimentv1.ExperimentRecipe{}
	}
	return recipe
}

func runsToProto(runs []intexp.Run) []*experimentv1.ExperimentRun {
	out := make([]*experimentv1.ExperimentRun, 0, len(runs))
	for _, run := range runs {
		out = append(out, runToProto(run))
	}
	return out
}

func qualificationEvidenceToProto(evidence intexp.QualificationEvidence) *experimentv1.QualificationEvidence {
	return &experimentv1.QualificationEvidence{
		Id:            evidence.ID,
		EngineId:      evidence.EngineID,
		ModelId:       evidence.ModelID,
		Strategy:      evidence.Strategy,
		PolicyProfile: evidence.PolicyProfile,
		Kind:          qualificationKindToProto(evidence.Kind),
		FaultProfile:  evidence.FaultProfile,
		Passed:        evidence.Passed,
		ArtifactRef:   evidence.ArtifactRef,
		Notes:         evidence.Notes,
		MachineJson:   string(evidence.MachineJSON),
		ObservedAt:    timestamppb.New(evidence.ObservedAt),
	}
}

func qualificationEvidenceFromProto(evidence *experimentv1.QualificationEvidence) (intexp.QualificationEvidence, error) {
	if evidence == nil {
		return intexp.QualificationEvidence{}, errors.New("qualification evidence is required")
	}
	kind, err := qualificationKindFromProto(evidence.GetKind())
	if err != nil {
		return intexp.QualificationEvidence{}, err
	}
	if evidence.GetEngineId() == "" || evidence.GetModelId() == "" || evidence.GetStrategy() == "" {
		return intexp.QualificationEvidence{}, errors.New("qualification evidence engine_id, model_id, and strategy are required")
	}
	if evidence.GetArtifactRef() == "" {
		return intexp.QualificationEvidence{}, errors.New("qualification evidence artifact_ref is required")
	}
	if kind == trustfloor.QualificationFault {
		if !isRequiredFault(evidence.GetFaultProfile()) {
			return intexp.QualificationEvidence{}, fmt.Errorf("qualification fault_profile %q is not required", evidence.GetFaultProfile())
		}
	} else if evidence.GetFaultProfile() != "" {
		return intexp.QualificationEvidence{}, errors.New("fault_profile is valid only for fault qualification evidence")
	}
	machineJSON := []byte(evidence.GetMachineJson())
	if len(machineJSON) > 0 && !json.Valid(machineJSON) {
		return intexp.QualificationEvidence{}, errors.New("qualification evidence machine_json must be valid JSON")
	}
	if kind == trustfloor.QualificationIntervalAccounting {
		accounting, accountingErr := parseIntervalAccountingArtifact(machineJSON)
		if accountingErr != nil {
			return intexp.QualificationEvidence{}, accountingErr
		}
		if evidence.GetPassed() && (!accounting.AllIntervalsAccounted || accounting.DuplicateCommittedSegments != 0 || accounting.SilentTerminalOutcomes != 0) {
			return intexp.QualificationEvidence{}, errors.New("passing interval accounting evidence requires complete coverage with zero duplicate commits and zero silent terminals")
		}
	}
	observed := time.Time{}
	if evidence.GetObservedAt() != nil {
		if err := evidence.GetObservedAt().CheckValid(); err != nil {
			return intexp.QualificationEvidence{}, fmt.Errorf("qualification evidence observed_at: %w", err)
		}
		observed = evidence.GetObservedAt().AsTime()
	}
	return intexp.QualificationEvidence{
		ID: evidence.GetId(), EngineID: evidence.GetEngineId(), ModelID: evidence.GetModelId(), Strategy: evidence.GetStrategy(),
		PolicyProfile: evidence.GetPolicyProfile(), Kind: kind, FaultProfile: evidence.GetFaultProfile(),
		Passed: evidence.GetPassed(), ArtifactRef: evidence.GetArtifactRef(), Notes: evidence.GetNotes(),
		MachineJSON: machineJSON, ObservedAt: observed,
	}, nil
}

func qualificationKindToProto(kind string) experimentv1.QualificationEvidenceKind {
	switch kind {
	case trustfloor.QualificationIntervalAccounting:
		return experimentv1.QualificationEvidenceKind_QUALIFICATION_EVIDENCE_KIND_INTERVAL_ACCOUNTING
	case trustfloor.QualificationBoundedRecovery:
		return experimentv1.QualificationEvidenceKind_QUALIFICATION_EVIDENCE_KIND_BOUNDED_RECOVERY
	case trustfloor.QualificationFault:
		return experimentv1.QualificationEvidenceKind_QUALIFICATION_EVIDENCE_KIND_FAULT
	case trustfloor.QualificationBrowserProductPath:
		return experimentv1.QualificationEvidenceKind_QUALIFICATION_EVIDENCE_KIND_BROWSER_PRODUCT_PATH
	case trustfloor.QualificationDevice:
		return experimentv1.QualificationEvidenceKind_QUALIFICATION_EVIDENCE_KIND_DEVICE
	default:
		return experimentv1.QualificationEvidenceKind_QUALIFICATION_EVIDENCE_KIND_UNSPECIFIED
	}
}

func qualificationKindFromProto(kind experimentv1.QualificationEvidenceKind) (string, error) {
	switch kind {
	case experimentv1.QualificationEvidenceKind_QUALIFICATION_EVIDENCE_KIND_INTERVAL_ACCOUNTING:
		return trustfloor.QualificationIntervalAccounting, nil
	case experimentv1.QualificationEvidenceKind_QUALIFICATION_EVIDENCE_KIND_BOUNDED_RECOVERY:
		return trustfloor.QualificationBoundedRecovery, nil
	case experimentv1.QualificationEvidenceKind_QUALIFICATION_EVIDENCE_KIND_FAULT:
		return trustfloor.QualificationFault, nil
	case experimentv1.QualificationEvidenceKind_QUALIFICATION_EVIDENCE_KIND_BROWSER_PRODUCT_PATH:
		return trustfloor.QualificationBrowserProductPath, nil
	case experimentv1.QualificationEvidenceKind_QUALIFICATION_EVIDENCE_KIND_DEVICE:
		return trustfloor.QualificationDevice, nil
	default:
		return "", errors.New("qualification evidence kind is required")
	}
}

func isRequiredFault(fault string) bool {
	for _, required := range trustfloor.RequiredFaults {
		if fault == required {
			return true
		}
	}
	return false
}
