package message

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	messagev1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/message"
	messageconnect "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/message/message_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/shared"
	"google.golang.org/protobuf/encoding/protojson"
)

type handlers struct {
	client messageconnect.MessageServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: messageconnect.NewMessageServiceClient(httpClient, baseURL)}
}

func (h *handlers) tree(ctx cliapp.RunContext) error {
	chatID := ctx.Positional("chat-id")
	resp, err := h.client.GetTree(context.Background(), connect.NewRequest(&messagev1.GetTreeRequest{ChatId: chatID}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get message tree for chat %q", chatID), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no message tree")
	}
	results := make([]string, 0, len(resp.Msg.GetMessages()))
	for _, m := range resp.Msg.GetMessages() {
		results = append(results, formatMessage(m))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d message(s); active leaf %s.", len(resp.Msg.GetMessages()), resp.Msg.GetActiveLeafMessageId())},
		ResultsHeading: "Messages",
		Results:        results,
		RetrievalHints: []string{"`messages send <chat-id> <text>` - add a user message"},
	})
}

func (h *handlers) send(ctx cliapp.RunContext) error {
	req := &messagev1.SendMessageRequest{
		ChatId:           ctx.Positional("chat-id"),
		ParentMessageId:  ctx.Flag("parent"),
		Content:          ctx.Positional("text"),
		Model:            ctx.Flag("model"),
		WebSearchEnabled: ctx.BoolFlag("web-search"),
		SelectedSkillIds: splitCSV(ctx.Flag("skill-ids")),
	}
	resp, err := h.client.SendMessage(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("send message", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetUserMessage() == nil {
		return fmt.Errorf("server returned no user message")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Sent message %s.", resp.Msg.GetUserMessage().GetId())},
		Changes: []string{formatMessage(resp.Msg.GetUserMessage())},
		NextCommand: []string{
			fmt.Sprintf("`messages stream %s --from %s` - stream the response", req.ChatId, resp.Msg.GetUserMessage().GetId()),
			fmt.Sprintf("`messages tree %s` - inspect branches", req.ChatId),
		},
	})
}

func (h *handlers) edit(ctx cliapp.RunContext) error {
	id := ctx.Positional("message-id")
	resp, err := h.client.EditMessage(context.Background(), connect.NewRequest(&messagev1.EditMessageRequest{
		MessageId: id,
		Content:   ctx.Positional("text"),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("edit message %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetMessage() == nil {
		return fmt.Errorf("server returned no edited message")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Edited message %s.", resp.Msg.GetMessage().GetId())},
		Changes: []string{formatMessage(resp.Msg.GetMessage())},
	})
}

func (h *handlers) regenerate(ctx cliapp.RunContext) error {
	id := ctx.Positional("message-id")
	resp, err := h.client.Regenerate(context.Background(), connect.NewRequest(&messagev1.RegenerateRequest{
		MessageId: id,
		Model:     ctx.Flag("model"),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("regenerate from message %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetAssistantMessage() == nil {
		return fmt.Errorf("server returned no assistant message")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Regenerated assistant message %s.", resp.Msg.GetAssistantMessage().GetId())},
		Changes: []string{formatMessage(resp.Msg.GetAssistantMessage())},
	})
}

func (h *handlers) stream(ctx cliapp.RunContext) error {
	req := &messagev1.StreamCompletionRequest{
		ChatId:           ctx.Positional("chat-id"),
		FromMessageId:    ctx.Flag("from"),
		Model:            ctx.Flag("model"),
		WebSearchEnabled: ctx.BoolFlag("web-search"),
		SelectedSkillIds: splitCSV(ctx.Flag("skill-ids")),
		Mode:             parseChatMode(ctx.Flag("mode")),
	}
	stream, err := h.client.StreamCompletion(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("stream completion", err, nil)
	}
	if ctx.JSON() {
		for stream.Receive() {
			if err := cliapp.PrintProtoJSON(ctx.Stdout(), stream.Msg()); err != nil {
				return err
			}
		}
		return stream.Err()
	}
	fmt.Fprintf(ctx.Stdout(), "Streaming completion for chat %s\n", req.ChatId)
	for stream.Receive() {
		event := stream.Msg()
		switch event.GetKind() {
		case messagev1.CompletionEventKind_COMPLETION_EVENT_KIND_TOKEN:
			fmt.Fprint(ctx.Stdout(), event.GetText())
		case messagev1.CompletionEventKind_COMPLETION_EVENT_KIND_STATUS:
			fmt.Fprintf(ctx.Stdout(), "\nstatus: %s\n", event.GetText())
		case messagev1.CompletionEventKind_COMPLETION_EVENT_KIND_SEARCH_ATTACHMENT:
			fmt.Fprintf(ctx.Stdout(), "\nsearch attachment: %d hit(s) degraded=%t\n", len(event.GetSearchAttachment().GetHits()), event.GetSearchAttachment().GetDegraded())
		case messagev1.CompletionEventKind_COMPLETION_EVENT_KIND_AGENT_ACTIVITY:
			fmt.Fprintf(ctx.Stdout(), "\nagent: %s\n", event.GetText())
		case messagev1.CompletionEventKind_COMPLETION_EVENT_KIND_ERROR:
			fmt.Fprintf(ctx.Stdout(), "\nerror: %s %s\n", event.GetErrorCode(), event.GetErrorMessage())
		case messagev1.CompletionEventKind_COMPLETION_EVENT_KIND_DONE:
			fmt.Fprintln(ctx.Stdout(), "\ndone")
		default:
			body, _ := protojson.Marshal(event)
			fmt.Fprintf(ctx.Stdout(), "\nevent: %s\n", string(body))
		}
	}
	return stream.Err()
}

func parseChatMode(value string) sharedv1.ChatMode {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "agent":
		return sharedv1.ChatMode_CHAT_MODE_AGENT
	case "llm", "":
		return sharedv1.ChatMode_CHAT_MODE_LLM
	default:
		return sharedv1.ChatMode_CHAT_MODE_UNSPECIFIED
	}
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func formatMessage(m *messagev1.Message) string {
	if m == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s role=%s parent=%s sibling=%d model=%s attachments=%d content=%q",
		m.GetId(), m.GetRole().String(), m.GetParentMessageId(), m.GetSiblingIndex(), m.GetModel(), len(m.GetSearchAttachments()), trim(m.GetContent(), 96))
}

func trim(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "..."
}
