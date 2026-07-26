package agentsessions

import (
	"swarm-manager/internal/identity"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/shared"
	"google.golang.org/protobuf/proto"
)

func AttributionFromProvenance(prov identity.Provenance) Attribution {
	attr := Attribution{
		Type:        AttributionType(prov.Type),
		RunID:       prov.RunID,
		TaskID:      prov.TaskID,
		ProfileKey:  prov.ProfileKey,
		SessionID:   prov.SessionID,
		SessionKind: Kind(prov.SessionKind),
		Source:      prov.Source,
	}
	if attr.Type == "" {
		attr.Type = AttributionOperator
	}
	return attr
}

func AttributionToProto(attr Attribution) *sharedpb.AgentSessionAttribution {
	msg := &sharedpb.AgentSessionAttribution{Type: string(attr.Type)}
	if attr.RunID != "" {
		msg.RunId = proto.String(attr.RunID)
	}
	if attr.TaskID != "" {
		msg.TaskId = proto.String(attr.TaskID)
	}
	if attr.ProfileKey != "" {
		msg.ProfileKey = proto.String(attr.ProfileKey)
	}
	if attr.SessionID != "" {
		msg.SessionId = proto.String(attr.SessionID)
	}
	if attr.SessionKind != "" {
		msg.SessionKind = proto.String(string(attr.SessionKind))
	}
	if attr.Source != "" {
		msg.Source = proto.String(attr.Source)
	}
	return msg
}

func messageToProto(message Message) *domainpb.AgentSessionMessage {
	msg := &domainpb.AgentSessionMessage{
		Id:            message.ID,
		Role:          string(message.Role),
		Content:       message.Content,
		CreatedAt:     message.CreatedAt,
		AttachmentIds: append([]string(nil), message.AttachmentIDs...),
	}
	for _, item := range message.Context {
		msg.Context = append(msg.Context, contextItemToProto(item))
	}
	return msg
}

func contextItemToProto(item ContextItem) *sharedpb.AgentSessionContextItem {
	msg := &sharedpb.AgentSessionContextItem{
		Type:       string(item.Type),
		Ref:        item.Ref,
		Title:      item.Title,
		Summary:    item.Summary,
		SelectedAt: item.SelectedAt,
	}
	if item.NodeID != "" {
		msg.NodeId = proto.String(item.NodeID)
	}
	if item.MetadataJSON != "" {
		msg.MetadataJson = proto.String(item.MetadataJSON)
	}
	return msg
}

func attachmentToProto(attachment Attachment) *sharedpb.AgentSessionAttachment {
	msg := &sharedpb.AgentSessionAttachment{
		Id:        attachment.ID,
		Filename:  attachment.Filename,
		CreatedAt: attachment.CreatedAt,
	}
	if attachment.ContentType != "" {
		msg.ContentType = proto.String(attachment.ContentType)
	}
	if attachment.SizeBytes >= 0 {
		msg.SizeBytes = proto.Int64(attachment.SizeBytes)
	}
	return msg
}

func proposalToProto(proposal Proposal) *domainpb.AgentSessionProposal {
	msg := &domainpb.AgentSessionProposal{
		Id:          proposal.ID,
		Kind:        string(proposal.Kind),
		Status:      string(proposal.Status),
		Summary:     proposal.Summary,
		PayloadJson: proposal.PayloadJSON,
		CreatedAt:   proposal.CreatedAt,
		UpdatedAt:   proposal.UpdatedAt,
	}
	if proposal.Attribution != nil {
		msg.Attribution = AttributionToProto(*proposal.Attribution)
	}
	return msg
}

func proposalTargetToProto(target *ProposalTarget) *domainpb.AgentSessionProposalTarget {
	if target == nil {
		return nil
	}
	return &domainpb.AgentSessionProposalTarget{
		Type: string(target.Type),
		Ref:  target.Ref,
		Name: target.Name,
	}
}

func artifactToProto(artifact Artifact) *sharedpb.AgentSessionArtifact {
	msg := &sharedpb.AgentSessionArtifact{
		Id:           artifact.ID,
		SessionId:    artifact.SessionID,
		ArtifactType: string(artifact.ArtifactType),
		Action:       string(artifact.Action),
		EntityRef:    artifact.EntityRef,
		CreatedAt:    artifact.CreatedAt,
	}
	if artifact.Title != "" {
		msg.Title = proto.String(artifact.Title)
	}
	if artifact.ProposalID != "" {
		msg.ProposalId = proto.String(artifact.ProposalID)
	}
	if artifact.ActivityID != "" {
		msg.ActivityId = proto.String(artifact.ActivityID)
	}
	if artifact.RunID != "" {
		msg.RunId = proto.String(artifact.RunID)
	}
	if artifact.MutationSource != "" {
		msg.MutationSource = proto.String(artifact.MutationSource)
	}
	if artifact.Attribution != nil {
		msg.Attribution = AttributionToProto(*artifact.Attribution)
	}
	return msg
}

func runEventToProto(event RunEvent) *apipb.AgentSessionRunEvent {
	return &apipb.AgentSessionRunEvent{
		Id:              event.ID,
		RunId:           event.RunID,
		Sequence:        event.Sequence,
		CreatedAt:       event.CreatedAt,
		EventType:       event.EventType,
		Role:            event.Role,
		Content:         event.Content,
		ToolName:        event.ToolName,
		ToolCallId:      event.ToolCallID,
		Input:           event.Input,
		Output:          event.Output,
		Error:           event.Error,
		Status:          event.Status,
		PreviousStatus:  event.PreviousStatus,
		ProgressPhase:   event.ProgressPhase,
		ProgressPercent: event.ProgressPercent,
		ProgressMessage: event.ProgressMessage,
		Summary:         event.Summary,
		RawJson:         event.RawJSON,
	}
}

func SessionToProto(session Session) *domainpb.AgentSession {
	msg := &domainpb.AgentSession{
		Id:        session.ID,
		Title:     session.Title,
		Kind:      string(session.Kind),
		Status:    string(session.Status),
		SkillId:   session.SkillID,
		CreatedAt: session.CreatedAt,
		UpdatedAt: session.UpdatedAt,
	}
	if session.TaskID != "" {
		msg.TaskId = proto.String(session.TaskID)
	}
	if session.RunID != "" {
		msg.RunId = proto.String(session.RunID)
	}
	if session.ProfileKey != "" {
		msg.ProfileKey = proto.String(session.ProfileKey)
	}
	if session.FailureReason != "" {
		msg.FailureReason = proto.String(session.FailureReason)
	}
	if session.CreatedBy != nil {
		msg.CreatedBy = AttributionToProto(*session.CreatedBy)
	}
	msg.ProposalTarget = proposalTargetToProto(session.ProposalTarget)
	for _, message := range session.Messages {
		msg.Messages = append(msg.Messages, messageToProto(message))
	}
	for _, proposal := range session.Proposals {
		msg.Proposals = append(msg.Proposals, proposalToProto(proposal))
	}
	for _, artifact := range session.Artifacts {
		msg.Artifacts = append(msg.Artifacts, artifactToProto(artifact))
	}
	for _, attachment := range session.Attachments {
		msg.Attachments = append(msg.Attachments, attachmentToProto(attachment))
	}
	return msg
}
