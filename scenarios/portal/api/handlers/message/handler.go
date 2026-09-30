package message

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"connectrpc.com/connect"

	contextcapturehandler "portal/handlers/contextcapture"
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
	context    contextcapturehandler.Service
}

func NewHandler(service *internalchat.Service, completionService *completion.Service, agentService *agentchat.Service, searchService *internalsearch.Service, contextServices ...contextcapturehandler.Service) *Handler {
	var contextService contextcapturehandler.Service
	if len(contextServices) > 0 {
		contextService = contextServices[0]
	}
	return &Handler{service: service, completion: completionService, agent: agentService, search: searchService, context: contextService}
}

func (h *Handler) GetAgentRun(ctx context.Context, req *connect.Request[messagev1.AgentRunRequest]) (*connect.Response[messagev1.AgentRunResponse], error) {
	return h.agentRun(ctx, req, false)
}

func (h *Handler) StopAgentRun(ctx context.Context, req *connect.Request[messagev1.AgentRunRequest]) (*connect.Response[messagev1.AgentRunResponse], error) {
	return h.agentRun(ctx, req, true)
}

func (h *Handler) agentRun(ctx context.Context, req *connect.Request[messagev1.AgentRunRequest], stop bool) (*connect.Response[messagev1.AgentRunResponse], error) {
	if strings.TrimSpace(req.Msg.GetChatId()) == "" || strings.TrimSpace(req.Msg.GetMessageId()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("chat and message identity required"))
	}
	if h.agent == nil {
		return nil, connect.NewError(connect.CodeUnavailable, agentmanager.ErrUnavailable)
	}
	input := agentchat.StreamInput{ChatID: req.Msg.GetChatId(), FromMessageID: req.Msg.GetMessageId()}
	var state agentmanager.RunState
	var err error
	if stop {
		state, err = h.agent.Stop(ctx, input)
	} else {
		state, err = h.agent.Run(ctx, input)
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("no agent admission for this message"))
		}
		if errors.Is(err, agentmanager.ErrStopUnconfirmed) || errors.Is(err, agentmanager.ErrUnavailable) {
			return nil, connect.NewError(connect.CodeUnavailable, err)
		}
		return nil, connectError(err)
	}
	return connect.NewResponse(&messagev1.AgentRunResponse{RunId: state.RunID, Status: state.Status, Terminal: state.Terminal}), nil
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
	contextIDs := append([]string(nil), req.Msg.GetContextDocumentIds()...)
	if len(contextIDs) > 0 {
		owner := internalchat.RequestOwner(ctx)
		if owner == "" || h.context == nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("context attachments require an authenticated account"))
		}
		for _, id := range contextIDs {
			if err := h.context.ValidateReference(ctx, owner, id); err != nil {
				return nil, connectError(err)
			}
		}
	}
	var webSearch *bool
	if req.Msg.GetWebSearchEnabled() {
		v := true
		webSearch = &v
	}
	msg, err := h.service.SendUserMessage(ctx, internalchat.SendMessageInput{
		ChatID:             req.Msg.GetChatId(),
		ParentMessageID:    req.Msg.GetParentMessageId(),
		Content:            req.Msg.GetContent(),
		Model:              req.Msg.GetModel(),
		WebSearch:          webSearch,
		ContextDocumentIDs: contextIDs,
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
	if _, err := h.service.GetChat(ctx, req.Msg.GetChatId()); err != nil {
		return connectError(err)
	}
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
	result, err := h.completion.Stream(ctx, completion.StreamInput{
		ChatID:           req.Msg.GetChatId(),
		FromMessageID:    req.Msg.GetFromMessageId(),
		Model:            req.Msg.GetModel(),
		WebSearchEnabled: req.Msg.GetWebSearchEnabled(),
		SelectedSkillIDs: req.Msg.GetSelectedSkillIds(),
	}, func(ev openrouter.StreamEvent) error {
		if ev.Token == "" {
			return nil
		}
		if err := stream.Send(&messagev1.CompletionEvent{
			Kind: messagev1.CompletionEventKind_COMPLETION_EVENT_KIND_TOKEN,
			Text: ev.Token,
		}); err != nil {
			return err
		}
		return nil
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
		if errors.Is(err, agentchat.ErrAgentImagesUnsupported) {
			return streamError(stream, "agent_images_unsupported", "the selected agent does not support image context")
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

func (h *Handler) ListAgentAdmissions(ctx context.Context, req *connect.Request[messagev1.ListAgentAdmissionsRequest]) (*connect.Response[messagev1.ListAgentAdmissionsResponse], error) {
	if h.agent == nil {
		return nil, connect.NewError(connect.CodeUnavailable, agentmanager.ErrUnavailable)
	}
	page, err := h.agent.ListAdmissions(ctx, req.Msg.GetPageToken(), int(req.Msg.GetPageSize()))
	if err != nil {
		if errors.Is(err, agentchat.ErrInvalidPage) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connectError(err)
	}
	response := &messagev1.ListAgentAdmissionsResponse{NextPageToken: page.NextPageToken}
	for _, binding := range page.Bindings {
		response.Admissions = append(response.Admissions, &messagev1.AgentAdmission{ChatId: binding.ChatID, MessageId: binding.MessageID})
	}
	return connect.NewResponse(response), nil
}
