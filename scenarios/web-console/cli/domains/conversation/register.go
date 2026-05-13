package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"connectrpc.com/connect"

	conversationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation"
	conversationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation/conversation_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	"web-console/cli/internal/support"
)

// Register builds the `conversation` subcommand group covering session
// history, the read/listened cursor, on-demand TTS summarization, and
// file-reference resolution / preview.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "conversation",
		Description: "Conversation history, cursors, summarization, and file references",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "get", Description: "Get a session's conversation history", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "cursor-set", Description: "Update a session's read/listened cursor (--body-file PATH)", Run: func(args []string) error { return runCursorSet(core, args) }},
			{Name: "summarize", Description: "Summarize one assistant event for TTS", Run: func(args []string) error { return runSummarize(core, args) }},
			{Name: "file-resolve", Description: "Resolve a file reference path for a session", Run: func(args []string) error { return runFileResolve(core, args) }},
			{Name: "file-content", Description: "Read a previewable file referenced by a session", Run: func(args []string) error { return runFileContent(core, args) }},
		},
	}
}

func newClient(core *cliapp.ScenarioApp) conversationconnect.ConversationServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return conversationconnect.NewConversationServiceClient(httpClient, baseURL)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("conversation get")
	session := fs.String("session", "", "Session ID (required)")
	since := fs.Int64("since", 0, "Only return events with sequence > since")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if *session == "" {
		return fmt.Errorf("--session is required")
	}

	resp, err := newClient(core).Get(context.Background(), connect.NewRequest(&conversationv1.GetRequest{
		SessionId:     *session,
		SinceSequence: *since,
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
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

// cursorPatchBody mirrors UpdateCursorRequest. Pointer fields toggle the
// matching has_* flag server-side.
type cursorPatchBody struct {
	LastSeenSequence     *int64 `json:"lastSeenSequence,omitempty"`
	LastListenedSequence *int64 `json:"lastListenedSequence,omitempty"`
}

func runCursorSet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("conversation cursor-set")
	session := fs.String("session", "", "Session ID (required)")
	bodyFile := fs.String("body-file", "", "Path to JSON body with cursor fields (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if *session == "" {
		return fmt.Errorf("--session is required")
	}

	raw, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	var body cursorPatchBody
	if err := json.Unmarshal(raw, &body); err != nil {
		return fmt.Errorf("decode --body-file: %w", err)
	}

	req := &conversationv1.UpdateCursorRequest{SessionId: *session}
	if body.LastSeenSequence != nil {
		req.LastSeenSequence = *body.LastSeenSequence
		req.HasLastSeenSequence = true
	}
	if body.LastListenedSequence != nil {
		req.LastListenedSequence = *body.LastListenedSequence
		req.HasLastListenedSequence = true
	}

	resp, err := newClient(core).UpdateCursor(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("conversation cursor-set", err, nil)
	}

	report := cliapp.MutationReport{
		Result: []string{fmt.Sprintf("cursor seen=%d listened=%d",
			resp.Msg.GetCursor().GetLastSeenSequence(),
			resp.Msg.GetCursor().GetLastListenedSequence())},
		NextCommand: []string{fmt.Sprintf("%s conversation get --session %s", support.CLIName, *session)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runSummarize(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("conversation summarize")
	session := fs.String("session", "", "Session ID (required)")
	event := fs.String("event", "", "Event ID (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if *session == "" || *event == "" {
		return fmt.Errorf("--session and --event are required")
	}

	resp, err := newClient(core).SummarizeEvent(context.Background(), connect.NewRequest(&conversationv1.SummarizeEventRequest{
		SessionId: *session,
		EventId:   *event,
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
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runFileResolve(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("conversation file-resolve")
	session := fs.String("session", "", "Session ID (required)")
	path := fs.String("path", "", "Path to resolve (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if *session == "" || *path == "" {
		return fmt.Errorf("--session and --path are required")
	}

	resp, err := newClient(core).ResolveFileReference(context.Background(), connect.NewRequest(&conversationv1.ResolveFileReferenceRequest{
		SessionId: *session,
		Path:      *path,
	}))
	if err != nil {
		return cliapp.WrapAPIError("conversation file-resolve", err, nil)
	}

	rows := []string{
		fmt.Sprintf("input:    %s", resp.Msg.GetInputPath()),
		fmt.Sprintf("resolved: %s", resp.Msg.GetResolvedPath()),
		fmt.Sprintf("exists:   %t", resp.Msg.GetExists()),
		fmt.Sprintf("basis:    %s", resp.Msg.GetResolutionBasis()),
		fmt.Sprintf("category: %s | previewable=%t", resp.Msg.GetCategory(), resp.Msg.GetCanPreview()),
	}
	if resp.Msg.GetHasLine() {
		rows = append(rows, fmt.Sprintf("line:     %d", resp.Msg.GetLine()))
	}
	report := cliapp.ListReport{
		Summary:        []string{"File reference"},
		ResultsHeading: "Resolution",
		Results:        rows,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runFileContent(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("conversation file-content")
	session := fs.String("session", "", "Session ID (required)")
	path := fs.String("path", "", "Path to read (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if *session == "" || *path == "" {
		return fmt.Errorf("--session and --path are required")
	}

	resp, err := newClient(core).GetFileReferenceContent(context.Background(), connect.NewRequest(&conversationv1.GetFileReferenceContentRequest{
		SessionId: *session,
		Path:      *path,
	}))
	if err != nil {
		return cliapp.WrapAPIError("conversation file-content", err, nil)
	}

	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, cliapp.ListReport{
			Summary:        []string{fmt.Sprintf("%s (%s)", resp.Msg.GetPath(), resp.Msg.GetCategory())},
			ResultsHeading: "Content",
			Results:        []string{resp.Msg.GetContent()},
		})
	}
	fmt.Fprintf(os.Stdout, "%s (%s, %s)\n", resp.Msg.GetPath(), resp.Msg.GetCategory(), resp.Msg.GetContentType())
	fmt.Fprintln(os.Stdout, resp.Msg.GetContent())
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
