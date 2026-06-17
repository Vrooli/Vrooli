package jobs

import (
	"time"

	internaljobs "image-tools/internal/jobs"

	"google.golang.org/protobuf/types/known/timestamppb"

	jobsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs"
)

func domainToProto(j internaljobs.Job) *jobsv1.Job {
	return &jobsv1.Job{
		Id:               j.ID,
		Operation:        j.Operation,
		Lane:             laneToProto(j.Lane),
		State:            stateToProto(j.State),
		Progress:         int32(j.Progress),
		Message:          j.Message,
		Error:            j.Error,
		ResultRef:        j.ResultRef,
		EstimatedSeconds: int32(j.EstimatedSeconds),
		CreatedAt:        timeToProto(&j.CreatedAt),
		StartedAt:        timeToProto(j.StartedAt),
		FinishedAt:       timeToProto(j.FinishedAt),
	}
}

func progressToProto(e internaljobs.ProgressEvent) *jobsv1.ProgressEvent {
	at := e.At
	return &jobsv1.ProgressEvent{
		JobId:    e.JobID,
		State:    stateToProto(e.State),
		Progress: int32(e.Progress),
		Message:  e.Message,
		At:       timeToProto(&at),
	}
}

func stateToProto(s internaljobs.State) jobsv1.JobState {
	switch s {
	case internaljobs.StateQueued:
		return jobsv1.JobState_JOB_STATE_QUEUED
	case internaljobs.StateRunning:
		return jobsv1.JobState_JOB_STATE_RUNNING
	case internaljobs.StateSucceeded:
		return jobsv1.JobState_JOB_STATE_SUCCEEDED
	case internaljobs.StateFailed:
		return jobsv1.JobState_JOB_STATE_FAILED
	case internaljobs.StateCanceled:
		return jobsv1.JobState_JOB_STATE_CANCELED
	default:
		return jobsv1.JobState_JOB_STATE_UNSPECIFIED
	}
}

func laneToProto(l internaljobs.Lane) jobsv1.JobLane {
	switch l {
	case internaljobs.LaneGPU:
		return jobsv1.JobLane_JOB_LANE_GPU
	case internaljobs.LaneCPU:
		return jobsv1.JobLane_JOB_LANE_CPU
	default:
		return jobsv1.JobLane_JOB_LANE_UNSPECIFIED
	}
}

// timeToProto returns nil for an unset/zero timestamp so the wire distinguishes
// "not yet started/finished" from the zero epoch.
func timeToProto(t *time.Time) *timestamppb.Timestamp {
	if t == nil || t.IsZero() {
		return nil
	}
	return timestamppb.New(*t)
}
