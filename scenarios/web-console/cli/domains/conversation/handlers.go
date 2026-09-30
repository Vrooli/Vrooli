package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"connectrpc.com/connect"

	conversationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation"
	conversationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation/conversation_v1connect"

	"github.com/vrooli/cli-core/cliapp"

	"web-console/cli/internal/support"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client conversationconnect.ConversationServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: conversationconnect.NewConversationServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	session := ctx.Flag("session")
	if session == "" {
		return fmt.Errorf("--session is required")
	}

	var sinceVal int64
	if raw := ctx.Flag("since"); raw != "" {
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return err
		}
		sinceVal = v
	}

	resp, err := h.client.Get(context.Background(), connect.NewRequest(&conversationv1.GetRequest{
		SessionId:     session,
		SinceSequence: sinceVal,
	}))
	if err != nil {
		return cliapp.WrapAPIError("conversation get", err, nil)
	}

	rows := []string{
		fmt.Sprintf("session: %s", resp.Msg.GetSessionId()),
		fmt.Sprintf("cursor seen=%d listened=%d", resp.Msg.GetCursor().GetLastSeenSequence(), resp.Msg.GetCursor().GetLastListenedSequence()),
	}
	for _, ev := range resp.Msg.GetEvents() {
		rows = append(rows, fmt.Sprintf("[%d] %s %s %q", ev.GetSequence(), ev.GetRole(), ev.GetCreatedAt(), truncate(ev.GetText(), 80)))
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d events", len(resp.Msg.GetEvents()))},
		ResultsHeading: "Conversation",
		Results:        rows,
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderListReport(ctx.Stdout(), report)
}

// cursorPatchBody mirrors UpdateCursorRequest. Pointer fields toggle the
// matching has_* flag server-side.
type cursorPatchBody struct {
	LastSeenSequence     *int64 `json:"lastSeenSequence,omitempty"`
	LastListenedSequence *int64 `json:"lastListenedSequence,omitempty"`
}

func (h *handlers) cursorSet(ctx cliapp.RunContext) error {
	session := ctx.Flag("session")
	if session == "" {
		return fmt.Errorf("--session is required")
	}

	raw, err := support.ReadJSONFile(ctx.Flag("body-file"), true)
	if err != nil {
		return err
	}
	var body cursorPatchBody
	if err := json.Unmarshal(raw, &body); err != nil {
		return fmt.Errorf("decode --body-file: %w", err)
	}

	req := &conversationv1.UpdateCursorRequest{SessionId: session}
	if body.LastSeenSequence != nil {
		req.LastSeenSequence = *body.LastSeenSequence
		req.HasLastSeenSequence = true
	}
	if body.LastListenedSequence != nil {
		req.LastListenedSequence = *body.LastListenedSequence
		req.HasLastListenedSequence = true
	}

	resp, err := h.client.UpdateCursor(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("conversation cursor-set", err, nil)
	}

	report := cliapp.MutationReport{
		Result: []string{fmt.Sprintf("cursor seen=%d listened=%d",
			resp.Msg.GetCursor().GetLastSeenSequence(),
			resp.Msg.GetCursor().GetLastListenedSequence())},
		NextCommand: []string{fmt.Sprintf("%s conversation get --session %s", support.CLIName, session)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderMutationReport(ctx.Stdout(), report)
}

func (h *handlers) summarize(ctx cliapp.RunContext) error {
	session := ctx.Flag("session")
	event := ctx.Flag("event")
	if session == "" || event == "" {
		return fmt.Errorf("--session and --event are required")
	}

	resp, err := h.client.SummarizeEvent(context.Background(), connect.NewRequest(&conversationv1.SummarizeEventRequest{
		SessionId: session,
		EventId:   event,
	}))
	if err != nil {
		return cliapp.WrapAPIError("conversation summarize", err, nil)
	}

	rows := []string{fmt.Sprintf("summarized: %t", resp.Msg.GetSummarized())}
	if resp.Msg.GetError() != "" {
		rows = append(rows, fmt.Sprintf("error: %s", resp.Msg.GetError()))
	}
	for i, p := range resp.Msg.GetSpeechParagraphs() {
		rows = append(rows, fmt.Sprintf("  [%d] %s", i+1, truncate(p, 120)))
	}
	report := cliapp.ListReport{
		Summary:        []string{"Summarize event"},
		ResultsHeading: "Result",
		Results:        rows,
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderListReport(ctx.Stdout(), report)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
