package dispatch

import (
	"context"
	"errors"

	"vrooli-bridge/internal/audit"
	"vrooli-bridge/internal/dispatch"
	"vrooli-bridge/internal/presence"
	"vrooli-bridge/internal/registry"
	"vrooli-bridge/internal/runs"

	"google.golang.org/protobuf/encoding/protojson"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
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

// jobPusherAdapter translates a dispatch PushedJob into a channel.JobPush
// wrapped in a ServerFrame, serialises it with protojson (compact single-line
// JSON, matching what the agent decodes with DiscardUnknown), and pushes it to
// every live channel the node holds via the presence hub.
type jobPusherAdapter struct {
	hub *presence.Hub
}

func (a jobPusherAdapter) PushJob(_ context.Context, nodeID string, job dispatch.PushedJob) (int, error) {
	frame := &channelv1.ServerFrame{
		Payload: &channelv1.ServerFrame_Job{
			Job: &channelv1.JobPush{
				RunId:          job.RunID,
				Scenario:       job.Scenario,
				Verb:           job.Verb,
				Args:           append([]string(nil), job.Args...),
				TimeoutSeconds: job.TimeoutSeconds,
			},
		},
	}
	payload, err := protojson.Marshal(frame)
	if err != nil {
		return 0, err
	}
	return a.hub.Push(nodeID, payload), nil
}
