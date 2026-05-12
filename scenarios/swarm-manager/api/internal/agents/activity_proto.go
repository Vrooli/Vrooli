package agentactivity

import (
	domain "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
	"google.golang.org/protobuf/proto"
)

func recordToProto(record Record) *domain.AgentActivity {
	msg := &domain.AgentActivity{
		ActivityId:      record.ActivityID,
		OwnerType:       string(record.OwnerType),
		OwnerName:       record.OwnerName,
		Purpose:         string(record.Purpose),
		InteractionType: string(record.InteractionType),
		Status:          string(record.Status),
		RequestedAt:     record.RequestedAt,
		Metadata:        record.Metadata,
		UpdatedAt:       record.UpdatedAt,
	}
	if record.OwnerKind != "" {
		msg.OwnerKind = proto.String(record.OwnerKind)
	}
	if record.OwnerTitle != "" {
		msg.OwnerTitle = proto.String(record.OwnerTitle)
	}
	if record.ExecutionID != "" {
		msg.ExecutionId = proto.String(record.ExecutionID)
	}
	if record.TaskID != "" {
		msg.TaskId = proto.String(record.TaskID)
	}
	if record.RunID != "" {
		msg.RunId = proto.String(record.RunID)
	}
	if record.StartedAt != "" {
		msg.StartedAt = proto.String(record.StartedAt)
	}
	if record.FinishedAt != "" {
		msg.FinishedAt = proto.String(record.FinishedAt)
	}
	if record.FailureReason != "" {
		msg.FailureReason = proto.String(record.FailureReason)
	}
	if record.RequestedBy != "" {
		msg.RequestedBy = proto.String(record.RequestedBy)
	}
	return msg
}
