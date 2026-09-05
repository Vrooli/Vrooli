package message

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"portal/internal/agentchat"
	internalchat "portal/internal/chat"
	"portal/internal/completion"
	"portal/internal/integrations/agentmanager"
	"portal/internal/integrations/openrouter"
	internalsearch "portal/internal/search"

	messagev1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/message"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/shared"
)

type Handler struct {
	service    *internalchat.Service
	completion *completion.Service
	agent      *agentchat.Service
	search     *internalsearch.Service
}

func NewHandler(service *internalchat.Service, completionService *completion.Service, agentService *agentchat.Service, searchService *internalsearch.Service) *Handler {
	return &Handler{service: service, completion: completionService, agent: agentService, search: searchService}
}

func (h *Handler) GetTree(ctx context.Context, req *connect.Request[messagev1.GetTreeRequest]) (*connect.Response[messagev1.GetTreeResponse], error) {
	messages, leaf, err := h.service.GetTree(ctx, req.Msg.GetChatId())
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&messagev1.GetTreeResponse{
		Messages:            internalchat.ToProtoMessages(messages),
		ActiveLeafMessageId: leaf,
	}), nil
}

func (h *Handler) SendMessage(ctx context.Context, req *connect.Request[messagev1.SendMessageRequest]) (*connect.Response[messagev1.SendMessageResponse], error) {
	var webSearch *bool
	if req.Msg.GetWebSearchEnabled() {
		v := true
		webSearch = &v
	}
	msg, err := h.service.SendUserMessage(ctx, internalchat.SendMessageInput{
		ChatID:          req.Msg.GetChatId(),
		ParentMessageID: req.Msg.GetParentMessageId(),
		Content:         req.Msg.GetContent(),
		Model:           req.Msg.GetModel(),
		WebSearch:       webSearch,
	})
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&messagev1.SendMessageResponse{UserMessage: internalchat.ToProtoMessage(msg)}), nil
}

func (h *Handler) EditMessage(ctx context.Context, req *connect.Request[messagev1.EditMessageRequest]) (*connect.Response[messagev1.EditMessageResponse], error) {
	msg, err := h.service.EditMessage(ctx, internalchat.BranchMessageInput{
		MessageID: req.Msg.GetMessageId(),
		Content:   req.Msg.GetContent(),
	})
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&messagev1.EditMessageResponse{Message: internalchat.ToProtoMessage(msg)}), nil
}

func (h *Handler) Regenerate(ctx context.Context, req *connect.Request[messagev1.RegenerateRequest]) (*connect.Response[messagev1.RegenerateResponse], error) {
	msg, err := h.service.Regenerate(ctx, internalchat.BranchMessageInput{
		MessageID: req.Msg.GetMessageId(),
		Content:   "",
		Model:     req.Msg.GetModel(),
	})
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&messagev1.RegenerateResponse{AssistantMessage: internalchat.ToProtoMessage(msg)}), nil
}

func (h *Handler) StreamCompletion(ctx context.Context, req *connect.Request[messagev1.StreamCompletionRequest], stream *connect.ServerStream[messagev1.CompletionEvent]) error {
	if req.Msg.GetMode() == sharedv1.ChatMode_CHAT_MODE_AGENT {
		return h.streamAgent(ctx, req.Msg, stream)
	}
	if h.completion == nil {
		return streamError(stream, "openrouter_unconfigured", "OpenRouter completion service is not configured")
	}
	if err := stream.Send(&messagev1.CompletionEvent{
		Kind: messagev1.CompletionEventKind_COMPLETION_EVENT_KIND_STATUS,
		Text: "starting OpenRouter completion",
	}); err != nil {
		return err
	}
	attachmentCh := h.startPassiveSearch(ctx, req.Msg)
	drainAttachment := func() error {
		if attachmentCh == nil {
			return nil
		}
		select {
		case result, ok := <-attachmentCh:
			if !ok {
				attachmentCh = nil
				return nil
			}
			if result.Err != nil || result.Attachment.ID == "" {
				return nil
			}
			return stream.Send(&messagev1.CompletionEvent{
				Kind:             messagev1.CompletionEventKind_COMPLETION_EVENT_KIND_SEARCH_ATTACHMENT,
				MessageId:        result.Attachment.MessageID,
				SearchAttachment: internalchat.ToProtoSearchAttachment(result.Attachment),
			})
		default:
			return nil
		}
	}

	result, err := h.completion.Stream(ctx, completion.StreamInput{
		ChatID:           req.Msg.GetChatId(),
		FromMessageID:    req.Msg.GetFromMessageId(),
		Model:            req.Msg.GetModel(),
		WebSearchEnabled: req.Msg.GetWebSearchEnabled(),
		SelectedSkillIDs: req.Msg.GetSelectedSkillIds(),
	}, func(ev openrouter.StreamEvent) error {
		if err := drainAttachment(); err != nil {
			return err
		}
		if ev.Token == "" {
			return nil
		}
		if err := stream.Send(&messagev1.CompletionEvent{
			Kind: messagev1.CompletionEventKind_COMPLETION_EVENT_KIND_TOKEN,
			Text: ev.Token,
		}); err != nil {
			return err
		}
		return drainAttachment()
	})
	if err != nil {
		if openrouter.IsMissingKey(err) {
			return streamError(stream, "openrouter_api_key_missing", err.Error())
		}
		var notFound internalchat.ErrNotFound
		if errors.As(err, &notFound) || errors.Is(err, internalchat.ErrInvalidInput) || errors.Is(err, completion.ErrNoCompletionMessages) {
			return streamError(stream, "completion_request_invalid", err.Error())
		}
		return streamError(stream, "openrouter_completion_failed", err.Error())
	}
	if err := drainAttachment(); err != nil {
		return err
	}
	return stream.Send(&messagev1.CompletionEvent{
		Kind:      messagev1.CompletionEventKind_COMPLETION_EVENT_KIND_DONE,
		MessageId: result.AssistantMessage.ID,
		Usage: &messagev1.UsageRecord{
			MessageId:        result.AssistantMessage.ID,
			Provider:         result.Usage.Provider,
			Model:            result.Usage.Model,
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
			CostUsd:          result.Usage.CostUSD,
		},
	})
}

func (h *Handler) startPassiveSearch(ctx context.Context, req *messagev1.StreamCompletionRequest) <-chan internalsearch.AttachmentResult {
	if h.search == nil || req.GetMode() == sharedv1.ChatMode_CHAT_MODE_AGENT {
		return nil
	}
	if req.GetChatId() == "" || req.GetFromMessageId() == "" {
		return nil
	}
	return h.search.StartAttachment(ctx, req.GetChatId(), req.GetFromMessageId())
}

func (h *Handler) streamAgent(ctx context.Context, req *messagev1.StreamCompletionRequest, stream *connect.ServerStream[messagev1.CompletionEvent]) error {
	if h.agent == nil {
		return streamError(stream, "agent_manager_unavailable", "agent-manager integration is not configured")
	}
	if err := stream.Send(&messagev1.CompletionEvent{
		Kind: messagev1.CompletionEventKind_COMPLETION_EVENT_KIND_STATUS,
		Text: "starting agent-manager run",
	}); err != nil {
		return err
	}
	result, err := h.agent.Stream(ctx, agentchat.StreamInput{
		ChatID:        req.GetChatId(),
		FromMessageID: req.GetFromMessageId(),
	}, func(ev agentmanager.ActivityEvent) error {
		return stream.Send(&messagev1.CompletionEvent{
			Kind:      messagev1.CompletionEventKind_COMPLETION_EVENT_KIND_AGENT_ACTIVITY,
			MessageId: ev.RunID,
			Text:      ev.Text,
		})
	})
	if err != nil {
		if errors.Is(err, agentmanager.ErrUnavailable) {
			return streamError(stream, "agent_manager_unavailable", err.Error())
		}
		var notFound internalchat.ErrNotFound
		if errors.As(err, &notFound) || errors.Is(err, internalchat.ErrInvalidInput) || errors.Is(err, agentchat.ErrNoAgentPrompt) {
			return streamError(stream, "agent_request_invalid", err.Error())
		}
		return streamError(stream, "agent_manager_run_failed", err.Error())
	}
	return stream.Send(&messagev1.CompletionEvent{
		Kind:      messagev1.CompletionEventKind_COMPLETION_EVENT_KIND_DONE,
		MessageId: result.Message.ID,
		Text:      "agent run completed",
	})
}

func streamError(stream *connect.ServerStream[messagev1.CompletionEvent], code, message string) error {
	return stream.Send(&messagev1.CompletionEvent{
		Kind:         messagev1.CompletionEventKind_COMPLETION_EVENT_KIND_ERROR,
		ErrorCode:    code,
		ErrorMessage: message,
		Text:         message,
	})
}

func connectError(err error) error {
	var notFound internalchat.ErrNotFound
	switch {
	case errors.As(err, &notFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, internalchat.ErrInvalidInput):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
