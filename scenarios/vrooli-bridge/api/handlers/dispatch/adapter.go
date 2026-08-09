package dispatch

import (
	"context"
	"errors"

	"vrooli-bridge/internal/audit"
	"vrooli-bridge/internal/dispatch"
	"vrooli-bridge/internal/queue"
	"vrooli-bridge/internal/registry"
	"vrooli-bridge/internal/runs"
)

// This file is the single translation point between the proto-free dispatch
// domain seams and the concrete registry / runs / audit / presence services and
// the channel push. The dispatch domain never imports a sibling domain or proto;
// these adapters do.

// nodeReaderAdapter projects a registry node down to the dispatch TargetNode.
type nodeReaderAdapter struct {
	svc registry.Service
}

func (a nodeReaderAdapter) GetTarget(ctx context.Context, id string) (dispatch.TargetNode, error) {
	n, err := a.svc.Get(ctx, id)
	if err != nil {
		var notFound registry.ErrNodeNotFound
		if errors.As(err, &notFound) {
			return dispatch.TargetNode{}, dispatch.ErrNodeNotFound{ID: id}
		}
		return dispatch.TargetNode{}, err
	}
	return dispatch.TargetNode{
		ID:      n.ID,
		OS:      n.OS,
		Arch:    n.Arch,
		Scopes:  append([]string(nil), n.Scopes...),
		Revoked: n.Revoked(),
	}, nil
}

// runControllerAdapter wraps runs.Service for the dispatch RunController seam.
type runControllerAdapter struct {
	svc runs.Service
}

func (a runControllerAdapter) CreateRun(ctx context.Context, in dispatch.CreateRunInput) (string, error) {
	run, err := a.svc.Create(ctx, runs.CreateInput{
		NodeID:         in.NodeID,
		Scenario:       in.Scenario,
		Verb:           in.Verb,
		Args:           in.Args,
		TimeoutSeconds: in.TimeoutSeconds,
	})
	if err != nil {
		return "", err
	}
	return run.ID, nil
}

func (a runControllerAdapter) AbortRun(ctx context.Context, runID, reason string) error {
	_, err := a.svc.Abort(ctx, runID, reason)
	return err
}

// auditSinkAdapter wraps the audit Sink for the dispatch AuditSink seam,
// translating the dispatch Entry DTO into an immutable audit.Record.
type auditSinkAdapter struct {
	sink audit.Sink
}

func (a auditSinkAdapter) Record(ctx context.Context, e dispatch.Entry) error {
	outcome := audit.OutcomeRejected
	if e.Accepted {
		outcome = audit.OutcomeAccepted
	}
	_, err := a.sink.Append(ctx, audit.Record{
		Action:   audit.ActionDispatch,
		Actor:    e.Actor,
		NodeID:   e.NodeID,
		Scenario: e.Scenario,
		Verb:     e.Verb,
		Args:     e.Args,
		Outcome:  outcome,
		Detail:   e.Detail,
		RunID:    e.RunID,
	})
	return err
}

// jobPusherAdapter routes a dispatch PushedJob through the per-node job
// scheduler (queue domain) instead of pushing to the channel directly. The
// scheduler enforces bounded concurrency: a node with a free slot is pushed
// immediately; a busy node holds the job queued (its durable run stays QUEUED
// until a slot frees). It satisfies dispatch's existing JobPusher seam — the
// (delivered, err) contract is unchanged, so a queued job reports delivered=1
// (accepted) and a node that dropped reports delivered=0 (dispatch aborts the
// run), exactly as the old direct push did.
type jobPusherAdapter struct {
	scheduler *queue.Scheduler
}

func (a jobPusherAdapter) PushJob(ctx context.Context, nodeID string, job dispatch.PushedJob) (int, error) {
	_, delivered, err := a.scheduler.Submit(ctx, queue.Job{
		RunID:          job.RunID,
		NodeID:         nodeID,
		Scenario:       job.Scenario,
		Verb:           job.Verb,
		Args:           append([]string(nil), job.Args...),
		TimeoutSeconds: job.TimeoutSeconds,
		Outputs:        outputsToQueue(job.Outputs),
	})
	return delivered, err
}

func outputsToQueue(outputs []dispatch.ArtifactOutput) []queue.Output {
	if len(outputs) == 0 {
		return nil
	}
	out := make([]queue.Output, 0, len(outputs))
	for _, output := range outputs {
		out = append(out, queue.Output{Name: output.Name, MediaType: output.MediaType, OutputFlag: output.OutputFlag, MaxBytes: output.MaxBytes})
	}
	return out
}
