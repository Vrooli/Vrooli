package runs

import (
	"vrooli-bridge/internal/runs"

	"google.golang.org/protobuf/types/known/timestamppb"

	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/shared"
)

// domainRunToProto translates a domain Run into its wire shape. The domain
// layer never imports proto; this is the single translation point (api-steer §7).
func domainRunToProto(r runs.Run) *runsv1.Run {
	out := &runsv1.Run{
		Id:             r.ID,
		NodeId:         r.NodeID,
		Scenario:       r.Scenario,
		Verb:           r.Verb,
		Args:           append([]string(nil), r.Args...),
		Status:         statusToProto(r.Status),
		ExitCode:       r.ExitCode,
		TimeoutSeconds: r.TimeoutSeconds,
		ArtifactRefs:   append([]string(nil), r.ArtifactRefs...),
		CreatedAt:      timestamppb.New(r.CreatedAt),
	}
	if !r.StartedAt.IsZero() {
		out.StartedAt = timestamppb.New(r.StartedAt)
	}
	if !r.FinishedAt.IsZero() {
		out.FinishedAt = timestamppb.New(r.FinishedAt)
	}
	return out
}

func statusToProto(s runs.RunStatus) runsv1.RunStatus {
	switch s {
	case runs.StatusQueued:
		return runsv1.RunStatus_RUN_STATUS_QUEUED
	case runs.StatusRunning:
		return runsv1.RunStatus_RUN_STATUS_RUNNING
	case runs.StatusPassed:
		return runsv1.RunStatus_RUN_STATUS_PASSED
	case runs.StatusFailed:
		return runsv1.RunStatus_RUN_STATUS_FAILED
	case runs.StatusAborted:
		return runsv1.RunStatus_RUN_STATUS_ABORTED
	default:
		return runsv1.RunStatus_RUN_STATUS_UNSPECIFIED
	}
}

// domainEventToProto translates a domain RunEvent into the shared channel
// RunEvent wire type (reused so the agent, SSE edge, and runs service speak one
// vocabulary).
func domainEventToProto(ev runs.RunEvent) *sharedv1.RunEvent {
	out := &sharedv1.RunEvent{
		RunId:       ev.RunID,
		Kind:        eventKindToProto(ev.Kind),
		Sequence:    ev.Sequence,
		LogChunk:    ev.LogChunk,
		Status:      ev.Status,
		ExitCode:    ev.ExitCode,
		ArtifactRef: ev.ArtifactRef,
	}
	if !ev.EmittedAt.IsZero() {
		out.EmittedAt = timestamppb.New(ev.EmittedAt)
	}
	return out
}

func eventKindToProto(k runs.EventKind) sharedv1.RunEventKind {
	switch k {
	case runs.EventLog:
		return sharedv1.RunEventKind_RUN_EVENT_KIND_LOG
	case runs.EventStatus:
		return sharedv1.RunEventKind_RUN_EVENT_KIND_STATUS
	case runs.EventExit:
		return sharedv1.RunEventKind_RUN_EVENT_KIND_EXIT
	case runs.EventArtifactRef:
		return sharedv1.RunEventKind_RUN_EVENT_KIND_ARTIFACT_REF
	default:
		return sharedv1.RunEventKind_RUN_EVENT_KIND_UNSPECIFIED
	}
}

// protoEventToDomain translates an inbound channel RunEvent (from the node-agent
// via ReportRunEvent) into the domain shape the runs service ingests.
func protoEventToDomain(ev *sharedv1.RunEvent) runs.RunEvent {
	out := runs.RunEvent{
		RunID:       ev.GetRunId(),
		Kind:        eventKindToDomain(ev.GetKind()),
		Sequence:    ev.GetSequence(),
		LogChunk:    ev.GetLogChunk(),
		Status:      ev.GetStatus(),
		ExitCode:    ev.GetExitCode(),
		ArtifactRef: ev.GetArtifactRef(),
	}
	if ev.GetEmittedAt() != nil {
		out.EmittedAt = ev.GetEmittedAt().AsTime()
	}
	return out
}

func eventKindToDomain(k sharedv1.RunEventKind) runs.EventKind {
	switch k {
	case sharedv1.RunEventKind_RUN_EVENT_KIND_LOG:
		return runs.EventLog
	case sharedv1.RunEventKind_RUN_EVENT_KIND_STATUS:
		return runs.EventStatus
	case sharedv1.RunEventKind_RUN_EVENT_KIND_EXIT:
		return runs.EventExit
	case sharedv1.RunEventKind_RUN_EVENT_KIND_ARTIFACT_REF:
		return runs.EventArtifactRef
	default:
		return runs.EventUnspecified
	}
}
