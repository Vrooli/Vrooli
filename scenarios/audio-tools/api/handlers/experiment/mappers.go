package experiment

import (
	intexp "audio-tools/internal/experiment"

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
	return &experimentv1.ExperimentRun{
		Id:            run.ID,
		ExperimentId:  run.ExperimentID,
		Strategy:      run.Strategy,
		ConditionJson: string(run.ConditionJSON),
		CreatedAt:     timestamppb.New(run.CreatedAt),
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
