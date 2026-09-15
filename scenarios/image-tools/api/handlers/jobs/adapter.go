package jobs

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	internaljobs "image-tools/internal/jobs"

	"google.golang.org/protobuf/types/known/timestamppb"

	jobsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs"
)

// payloadEcho is the subset of the stored ai.Payload needed to reconstruct the
// request. It is declared locally rather than importing the ai package, which
// would invert the dependency edge.
type payloadEcho struct {
	Operation  string            `json:"operation"`
	ModelID    string            `json:"model_id"`
	Backend    string            `json:"backend"`
	Tier       string            `json:"tier"`
	Variations int               `json:"variations"`
	Params     map[string]string `json:"params"`
	Adapters   []struct {
		ID string `json:"id"`
	} `json:"adapters"`
}

func domainToProto(j internaljobs.Job) *jobsv1.Job {
	out := &jobsv1.Job{
		Id:               j.ID,
		Operation:        j.Operation,
		Lane:             laneToProto(j.Lane),
		State:            stateToProto(j.State),
		Progress:         int32(j.Progress),
		Message:          j.Message,
		Error:            j.Error,
		ResultRef:        j.ResultRef,
		ResultMeta:       j.Meta,
		EstimatedSeconds: int32(j.EstimatedSeconds),
		CreatedAt:        timeToProto(&j.CreatedAt),
		StartedAt:        timeToProto(j.StartedAt),
		FinishedAt:       timeToProto(j.FinishedAt),
		Request:          payloadToRequest(j.Payload),
	}
	if refs := j.Meta["result_refs"]; refs != "" {
		out.ResultRefs = strings.Split(refs, ",")
	} else if j.ResultRef != "" {
		out.ResultRefs = []string{j.ResultRef}
	}
	return out
}

// payloadToRequest rebuilds the typed request echo from the stored payload.
// A missing or unparsable payload yields nil rather than an error: the echo is
// provenance, and a legacy job with no payload must still be readable.
func payloadToRequest(raw []byte) *jobsv1.JobRequest {
	if len(raw) == 0 {
		return nil
	}
	var p payloadEcho
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil
	}
	req := &jobsv1.JobRequest{
		Operation:  p.Operation,
		ModelId:    p.ModelID,
		Backend:    p.Backend,
		Tier:       p.Tier,
		Variations: int32(p.Variations),
		Role:       p.Params["openrouter_role"],
		Prompt:     p.Params["prompt"],
	}
	req.NegativePrompt = p.Params["negative_prompt"]
	if v, err := strconv.ParseInt(p.Params["seed"], 10, 64); err == nil {
		req.Seed = v
	}
	if v, err := strconv.Atoi(p.Params["width"]); err == nil {
		req.Width = int32(v)
	}
	if v, err := strconv.Atoi(p.Params["height"]); err == nil {
		req.Height = int32(v)
	}
	for _, a := range p.Adapters {
		if a.ID != "" {
			req.Adapters = append(req.Adapters, a.ID)
		}
	}
	return req
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
	case internaljobs.LaneNetwork:
		return jobsv1.JobLane_JOB_LANE_NETWORK
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
