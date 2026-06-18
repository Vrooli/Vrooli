package provision

import (
	"context"
	"errors"

	"vrooli-bridge/internal/audit"
	"vrooli-bridge/internal/presence"
	"vrooli-bridge/internal/provision"
	"vrooli-bridge/internal/registry"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	provisionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/provision"
)

// This file is the single translation point between the proto-free provision
// domain (its seams + DTOs) and the concrete registry / audit / presence
// services and the channel push. The provision domain never imports a sibling
// domain or proto; these adapters do.

// ---- proto <-> domain translations (api-steer §7) ----

func domainOpToProto(op provision.ProvisioningOp) *provisionv1.ProvisioningOp {
	out := &provisionv1.ProvisioningOp{
		Id:                op.ID,
		NodeId:            op.NodeID,
		TargetRevision:    op.TargetRevision,
		RollbackRevision:  op.RollbackRevision,
		Status:            statusToProto(op.Status),
		ResultingRevision: op.ResultingRevision,
		ExitCode:          op.ExitCode,
		CreatedAt:         timestamppb.New(op.CreatedAt),
	}
	if !op.StartedAt.IsZero() {
		out.StartedAt = timestamppb.New(op.StartedAt)
	}
	if !op.FinishedAt.IsZero() {
		out.FinishedAt = timestamppb.New(op.FinishedAt)
	}
	return out
}

func statusToProto(s provision.ProvisioningStatus) provisionv1.ProvisioningStatus {
	switch s {
	case provision.StatusQueued:
		return provisionv1.ProvisioningStatus_PROVISIONING_STATUS_QUEUED
	case provision.StatusRunning:
		return provisionv1.ProvisioningStatus_PROVISIONING_STATUS_RUNNING
	case provision.StatusCompleted:
		return provisionv1.ProvisioningStatus_PROVISIONING_STATUS_COMPLETED
	case provision.StatusFailed:
		return provisionv1.ProvisioningStatus_PROVISIONING_STATUS_FAILED
	case provision.StatusRolledBack:
		return provisionv1.ProvisioningStatus_PROVISIONING_STATUS_ROLLED_BACK
	default:
		return provisionv1.ProvisioningStatus_PROVISIONING_STATUS_UNSPECIFIED
	}
}

func domainEventToProto(ev provision.ProvisionEvent) *provisionv1.ProvisionEvent {
	out := &provisionv1.ProvisionEvent{
		OpId:     ev.OpID,
		Kind:     eventKindToProto(ev.Kind),
		Sequence: ev.Sequence,
		LogChunk: ev.LogChunk,
		Status:   ev.Status,
		Revision: ev.Revision,
		ExitCode: ev.ExitCode,
	}
	if !ev.EmittedAt.IsZero() {
		out.EmittedAt = timestamppb.New(ev.EmittedAt)
	}
	return out
}

func eventKindToProto(k provision.EventKind) provisionv1.ProvisionEventKind {
	switch k {
	case provision.EventLog:
		return provisionv1.ProvisionEventKind_PROVISION_EVENT_KIND_LOG
	case provision.EventStatus:
		return provisionv1.ProvisionEventKind_PROVISION_EVENT_KIND_STATUS
	case provision.EventVersion:
		return provisionv1.ProvisionEventKind_PROVISION_EVENT_KIND_VERSION
	case provision.EventExit:
		return provisionv1.ProvisionEventKind_PROVISION_EVENT_KIND_EXIT
	default:
		return provisionv1.ProvisionEventKind_PROVISION_EVENT_KIND_UNSPECIFIED
	}
}

func protoEventToDomain(ev *provisionv1.ProvisionEvent) provision.ProvisionEvent {
	out := provision.ProvisionEvent{
		OpID:     ev.GetOpId(),
		Kind:     eventKindToDomain(ev.GetKind()),
		Sequence: ev.GetSequence(),
		LogChunk: ev.GetLogChunk(),
		Status:   ev.GetStatus(),
		Revision: ev.GetRevision(),
		ExitCode: ev.GetExitCode(),
	}
	if ev.GetEmittedAt() != nil {
		out.EmittedAt = ev.GetEmittedAt().AsTime()
	}
	return out
}

func eventKindToDomain(k provisionv1.ProvisionEventKind) provision.EventKind {
	switch k {
	case provisionv1.ProvisionEventKind_PROVISION_EVENT_KIND_LOG:
		return provision.EventLog
	case provisionv1.ProvisionEventKind_PROVISION_EVENT_KIND_STATUS:
		return provision.EventStatus
	case provisionv1.ProvisionEventKind_PROVISION_EVENT_KIND_VERSION:
		return provision.EventVersion
	case provisionv1.ProvisionEventKind_PROVISION_EVENT_KIND_EXIT:
		return provision.EventExit
	default:
		return provision.EventUnspecified
	}
}

func domainVersionToProto(v provision.NodeVersion) *provisionv1.NodeVersion {
	out := &provisionv1.NodeVersion{
		NodeId:   v.NodeID,
		Revision: v.Revision,
		OpId:     v.OpID,
	}
	if !v.ReportedAt.IsZero() {
		out.ReportedAt = timestamppb.New(v.ReportedAt)
	}
	return out
}

// ---- seam adapters (proto-free domain <-> concrete services) ----

// nodeReaderAdapter projects a registry node down to the provision TargetNode.
type nodeReaderAdapter struct {
	svc registry.Service
}

var _ provision.NodeReader = nodeReaderAdapter{}

func (a nodeReaderAdapter) GetTarget(ctx context.Context, id string) (provision.TargetNode, error) {
	n, err := a.svc.Get(ctx, id)
	if err != nil {
		var notFound registry.ErrNodeNotFound
		if errors.As(err, &notFound) {
			return provision.TargetNode{}, provision.ErrNodeNotFound{ID: id}
		}
		return provision.TargetNode{}, err
	}
	return provision.TargetNode{ID: n.ID, Revoked: n.Revoked()}, nil
}

// auditSinkAdapter wraps the audit Sink for the provision AuditSink seam,
// translating the provision Entry DTO into an immutable audit.Record stamped
// with ActionProvision. The target/rollback revisions ride Args, and the op id
// is recorded as the record's RunID so the audit trail joins op→record.
type auditSinkAdapter struct {
	sink audit.Sink
}

var _ provision.AuditSink = auditSinkAdapter{}

func (a auditSinkAdapter) Record(ctx context.Context, e provision.Entry) error {
	outcome := audit.OutcomeRejected
	if e.Accepted {
		outcome = audit.OutcomeAccepted
	}
	args := []string{e.TargetRevision}
	if e.RollbackRevision != "" {
		args = append(args, "rollback="+e.RollbackRevision)
	}
	_, err := a.sink.Append(ctx, audit.Record{
		Action:  audit.ActionProvision,
		Actor:   e.Actor,
		NodeID:  e.NodeID,
		Verb:    "provision",
		Args:    args,
		Outcome: outcome,
		Detail:  e.Detail,
		RunID:   e.OpID,
	})
	return err
}

// commandPusherAdapter translates a provision PushedCommand into a
// channel.ProvisionCommand wrapped in a ServerFrame, serialises it with
// protojson (compact single-line JSON the agent decodes with DiscardUnknown),
// and pushes it to every live channel the node holds via the presence hub.
type commandPusherAdapter struct {
	hub *presence.Hub
}

var _ provision.CommandPusher = commandPusherAdapter{}

func (a commandPusherAdapter) PushProvision(_ context.Context, nodeID string, cmd provision.PushedCommand) (int, error) {
	frame := &channelv1.ServerFrame{
		Payload: &channelv1.ServerFrame_Provision{
			Provision: &channelv1.ProvisionCommand{
				OpId:             cmd.OpID,
				TargetRevision:   cmd.TargetRevision,
				RollbackRevision: cmd.RollbackRevision,
			},
		},
	}
	payload, err := protojson.Marshal(frame)
	if err != nil {
		return 0, err
	}
	return a.hub.Push(nodeID, payload), nil
}
