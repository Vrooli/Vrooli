package chat

import (
	"time"

	chatv1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/chat"
	messagev1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/message"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/shared"
)

func ToProtoChat(c Chat) *chatv1.Chat {
	return &chatv1.Chat{
		Id:                  c.ID,
		Title:               c.Title,
		GroupId:             c.GroupID,
		SortOrder:           c.SortOrder,
		Model:               c.Model,
		WebSearchEnabled:    c.WebSearchEnabled,
		Mode:                c.Mode.Proto(),
		AgentHarness:        c.AgentHarness.Proto(),
		ActiveLeafMessageId: c.ActiveLeafMessageID,
		CreatedAt:           formatTime(c.CreatedAt),
		UpdatedAt:           formatTime(c.UpdatedAt),
	}
}

func ToProtoChats(chats []Chat) []*chatv1.Chat {
	out := make([]*chatv1.Chat, 0, len(chats))
	for _, c := range chats {
		out = append(out, ToProtoChat(c))
	}
	return out
}

func ToProtoGroup(g ChatGroup) *chatv1.ChatGroup {
	return &chatv1.ChatGroup{
		Id:        g.ID,
		Name:      g.Name,
		Color:     g.Color,
		Collapsed: g.Collapsed,
		SortOrder: g.SortOrder,
		CreatedAt: formatTime(g.CreatedAt),
		UpdatedAt: formatTime(g.UpdatedAt),
	}
}

func ToProtoGroups(groups []ChatGroup) []*chatv1.ChatGroup {
	out := make([]*chatv1.ChatGroup, 0, len(groups))
	for _, g := range groups {
		out = append(out, ToProtoGroup(g))
	}
	return out
}

func ToProtoMessage(m Message) *messagev1.Message {
	return &messagev1.Message{
		Id:                m.ID,
		ChatId:            m.ChatID,
		ParentMessageId:   m.ParentMessageID,
		SiblingIndex:      m.SiblingIndex,
		Role:              roleToProto(m.Role),
		Content:           m.Content,
		Model:             m.Model,
		CreatedAt:         formatTime(m.CreatedAt),
		UpdatedAt:         formatTime(m.UpdatedAt),
		SearchAttachments: ToProtoSearchAttachments(m.SearchAttachments),
	}
}

func ToProtoMessages(messages []Message) []*messagev1.Message {
	out := make([]*messagev1.Message, 0, len(messages))
	for _, m := range messages {
		out = append(out, ToProtoMessage(m))
	}
	return out
}

func ToProtoSearchAttachment(a SearchAttachment) *messagev1.SearchAttachment {
	return &messagev1.SearchAttachment{
		Id:        a.ID,
		Query:     a.Query,
		Hits:      ToProtoSearchHits(a.Hits),
		Degraded:  a.Degraded,
		Reason:    a.Reason,
		LatencyMs: a.LatencyMS,
		CreatedAt: formatTime(a.CreatedAt),
	}
}

func ToProtoSearchAttachments(attachments []SearchAttachment) []*messagev1.SearchAttachment {
	out := make([]*messagev1.SearchAttachment, 0, len(attachments))
	for _, attachment := range attachments {
		out = append(out, ToProtoSearchAttachment(attachment))
	}
	return out
}

func ToProtoSearchHit(hit SearchHit) *sharedv1.SearchHit {
	return &sharedv1.SearchHit{
		ProviderId:  hit.ProviderID,
		Type:        hit.Type,
		Title:       hit.Title,
		Snippet:     hit.Snippet,
		Path:        hit.Path,
		Score:       hit.Score,
		RerankScore: hit.RerankScore,
		Locations:   hit.Locations,
	}
}

func ToProtoSearchHits(hits []SearchHit) []*sharedv1.SearchHit {
	out := make([]*sharedv1.SearchHit, 0, len(hits))
	for _, hit := range hits {
		out = append(out, ToProtoSearchHit(hit))
	}
	return out
}

func roleToProto(role MessageRole) messagev1.MessageRole {
	switch role {
	case RoleSystem:
		return messagev1.MessageRole_MESSAGE_ROLE_SYSTEM
	case RoleAssistant:
		return messagev1.MessageRole_MESSAGE_ROLE_ASSISTANT
	case RoleAgent:
		return messagev1.MessageRole_MESSAGE_ROLE_AGENT
	default:
		return messagev1.MessageRole_MESSAGE_ROLE_USER
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}
