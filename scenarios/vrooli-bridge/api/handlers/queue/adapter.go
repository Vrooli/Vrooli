package queue

import (
	"context"
	"time"

	"vrooli-bridge/internal/channelsign"
	"vrooli-bridge/internal/presence"
	"vrooli-bridge/internal/queue"
	"vrooli-bridge/internal/relay"
	"vrooli-bridge/internal/runs"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	queuev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/queue"
)

// This file is the single translation point between the proto-free queue
// scheduler (its Job/Entry DTOs + Pusher/Aborter seams) and the concrete channel
// push, the runs service, and the proto wire types. The queue scheduler never
// imports proto or a sibling domain; these adapters do.

// channelPusher delivers a scheduled job to the node's held dial-out channel by
// translating queue.Job → channel.JobPush ServerFrame → signed envelope →
// presence-hub push. It satisfies the queue.Pusher seam. Every frame is signed
// with the control-plane identity key so the node verifies it against the pinned
// key before acting (SECURITY.md boundary 2).
type channelPusher struct {
	hub    *presence.Hub
	signer channelsign.Signer
}

// NewChannelPusher constructs the queue.Pusher that pushes signed JobPush frames.
func NewChannelPusher(hub *presence.Hub, signer channelsign.Signer) queue.Pusher {
	return channelPusher{hub: hub, signer: signer}
}

var _ queue.Pusher = channelPusher{}

func (p channelPusher) Push(_ context.Context, job queue.Job) (int, error) {
	frame := &channelv1.ServerFrame{
		FrameId: uuid.NewString(),
		Payload: &channelv1.ServerFrame_Job{
			Job: &channelv1.JobPush{
				RunId:          job.RunID,
				Scenario:       job.Scenario,
				Verb:           job.Verb,
				Args:           append([]string(nil), job.Args...),
				TimeoutSeconds: job.TimeoutSeconds,
				Outputs:        outputsToProto(job.Outputs),
			},
		},
	}
	payload, err := channelsign.Marshal(p.signer, frame)
	if err != nil {
		return 0, err
	}
	return p.hub.PushFrame(job.NodeID, frame.GetFrameId(), payload), nil
}

func (p channelPusher) IsAvailable(nodeID string) bool { return p.hub.IsOnline(nodeID) }

func outputsToProto(outputs []queue.Output) []*channelv1.ArtifactOutput {
	if len(outputs) == 0 {
		return nil
	}
	out := make([]*channelv1.ArtifactOutput, 0, len(outputs))
	for _, output := range outputs {
		out = append(out, &channelv1.ArtifactOutput{
			Name: output.Name, MediaType: output.MediaType, OutputFlag: output.OutputFlag, MaxBytes: output.MaxBytes,
		})
	}
	return out
}

// channelCanceller pushes a signed AbortJob frame to a node so it STOPS an
// in-flight run (OT-P1-004 node-side cancel). It satisfies the runs.Canceller
// seam.
type channelCanceller struct {
	hub    *presence.Hub
	signer channelsign.Signer
}

// NewChannelCanceller constructs the runs.Canceller that pushes signed AbortJob
// frames.
func NewChannelCanceller(hub *presence.Hub, signer channelsign.Signer) runs.Canceller {
	return channelCanceller{hub: hub, signer: signer}
}

var _ runs.Canceller = channelCanceller{}

func (c channelCanceller) CancelJob(_ context.Context, nodeID, runID, reason string) error {
	frame := &channelv1.ServerFrame{
		FrameId: uuid.NewString(),
		Payload: &channelv1.ServerFrame_Abort{
			Abort: &channelv1.AbortJob{RunId: runID, Reason: reason},
		},
	}
	payload, err := channelsign.Marshal(c.signer, frame)
	if err != nil {
		return err
	}
	c.hub.PushFrame(nodeID, frame.GetFrameId(), payload)
	return nil
}

// channelRelayPusher carries the short-lived relay request over the same
// signed, node-owned SSE channel as durable jobs. Keeping this adapter beside
// the job pusher makes the wire translation single-purpose and ensures relay
// frames are covered by the control-plane signature before they leave Bridge.
type channelRelayPusher struct {
	hub    *presence.Hub
	signer channelsign.Signer
}

func NewChannelRelayPusher(hub *presence.Hub, signer channelsign.Signer) relay.Pusher {
	return channelRelayPusher{hub: hub, signer: signer}
}

var _ relay.Pusher = channelRelayPusher{}

func (p channelRelayPusher) Push(_ context.Context, nodeID string, request relay.Request) (int, error) {
	frame := &channelv1.ServerFrame{
		FrameId: uuid.NewString(),
		Payload: &channelv1.ServerFrame_Relay{Relay: &channelv1.RelayRequest{
			CorrelationId:    request.CorrelationID,
			Scenario:         request.Scenario,
			Command:          request.Command,
			Args:             append([]string(nil), request.Args...),
			TimeoutSeconds:   request.TimeoutSeconds,
			MaxResponseBytes: request.MaxResponseBytes,
		}},
	}
	payload, err := channelsign.Marshal(p.signer, frame)
	if err != nil {
		return 0, err
	}
	return p.hub.PushFrame(nodeID, frame.GetFrameId(), payload), nil
}

func (p channelRelayPusher) Cancel(_ context.Context, nodeID, correlationID, reason string) (int, error) {
	frame := &channelv1.ServerFrame{
		FrameId: uuid.NewString(),
		Payload: &channelv1.ServerFrame_RelayCancel{RelayCancel: &channelv1.RelayCancel{
			CorrelationId: correlationID,
			Reason:        reason,
		}},
	}
	payload, err := channelsign.Marshal(p.signer, frame)
	if err != nil {
		return 0, err
	}
	return p.hub.PushFrame(nodeID, frame.GetFrameId(), payload), nil
}

// runsAborter wraps the runs service for the queue.Aborter seam (abort a run
// whose queued promotion could not be delivered).
type runsAborter struct {
	svc runs.Service
}

// NewAborter constructs the queue.Aborter that aborts undeliverable queued runs.
func NewAborter(svc runs.Service) queue.Aborter {
	return runsAborter{svc: svc}
}

var _ queue.Aborter = runsAborter{}

func (a runsAborter) Abort(ctx context.Context, runID, reason string) error {
	_, err := a.svc.Abort(ctx, runID, reason)
	return err
}

type durableRunStore struct{ svc runs.Service }

// NewDurableStore projects the runs domain into the queue's persistence seam.
// The scheduler remains proto-free and can be tested with an in-memory store,
// while production reconstructs its queue from the same SQLite-backed runs.
func NewDurableStore(svc runs.Service) queue.DurableStore { return durableRunStore{svc: svc} }

func (s durableRunStore) Load(ctx context.Context) ([]queue.DurableEntry, error) {
	all, err := s.svc.List(ctx, runs.ListFilter{})
	if err != nil {
		return nil, err
	}
	out := make([]queue.DurableEntry, 0, len(all))
	for _, run := range all {
		if run.Status.Terminal() {
			continue
		}
		state := queue.StateRunning
		if run.Status == runs.StatusQueued {
			state = queue.StateQueued
		}
		out = append(out, queue.DurableEntry{
			Job:   queue.Job{RunID: run.ID, NodeID: run.NodeID, Scenario: run.Scenario, Verb: run.Verb, Args: append([]string(nil), run.Args...), TimeoutSeconds: run.TimeoutSeconds},
			State: state, EnqueuedAt: run.QueuedSince, StartedAt: run.StartedAt,
			PushedAt: run.PushedAt, AckedAt: run.AckedAt, LeaseExpiresAt: run.DeliveryLeaseExpiresAt,
			DeliveryAttempts: run.DeliveryAttempts, Acked: run.Status == runs.StatusAcked,
		})
	}
	return out, nil
}

func (s durableRunStore) MarkQueued(ctx context.Context, runID string, at time.Time, detail ...string) error {
	reason := ""
	if len(detail) > 0 {
		reason = detail[0]
	}
	return s.svc.MarkDeliveryState(ctx, runID, runs.StatusQueued, reason, at)
}

func (s durableRunStore) MarkPushed(ctx context.Context, runID string, at, leaseExpiresAt time.Time) error {
	return s.svc.MarkDeliveryState(ctx, runID, runs.StatusPushed, "", at, leaseExpiresAt)
}

func (s durableRunStore) MarkFailedDelivery(ctx context.Context, runID, reason string, at time.Time) error {
	return s.svc.MarkDeliveryState(ctx, runID, runs.StatusFailedDelivery, reason, at)
}

// ---- snapshot -> proto translations (api-steer §7) ----

func nodeQueueToProto(nq queue.NodeQueue) *queuev1.NodeQueue {
	out := &queuev1.NodeQueue{
		NodeId:           nq.NodeID,
		ConcurrencyLimit: int32(nq.ConcurrencyLimit),
		Running:          int32(nq.Running),
		Queued:           int32(nq.Queued),
		Entries:          make([]*queuev1.QueueEntry, 0, len(nq.Entries)),
	}
	for _, e := range nq.Entries {
		out.Entries = append(out.Entries, entryToProto(e))
	}
	return out
}

func entryToProto(e queue.Entry) *queuev1.QueueEntry {
	out := &queuev1.QueueEntry{
		RunId:    e.Job.RunID,
		NodeId:   e.Job.NodeID,
		Scenario: e.Job.Scenario,
		Verb:     e.Job.Verb,
		Args:     append([]string(nil), e.Job.Args...),
		State:    stateToProto(e.State),
		Position: int32(e.Position),
	}
	if !e.EnqueuedAt.IsZero() {
		out.EnqueuedAt = timestamppb.New(e.EnqueuedAt)
	}
	if !e.StartedAt.IsZero() {
		out.StartedAt = timestamppb.New(e.StartedAt)
	}
	return out
}

func stateToProto(s queue.State) queuev1.QueueState {
	switch s {
	case queue.StateQueued:
		return queuev1.QueueState_QUEUE_STATE_QUEUED
	case queue.StateRunning:
		return queuev1.QueueState_QUEUE_STATE_RUNNING
	default:
		return queuev1.QueueState_QUEUE_STATE_UNSPECIFIED
	}
}
