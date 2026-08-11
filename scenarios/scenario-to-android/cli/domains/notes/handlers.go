package notes

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
	notesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-android/v1/notes"
	notesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-android/v1/notes/notes_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// defaultTimeWindow matches the manifest measure's `window` default so the
// CLI and the search-hub auto-answer path resolve the same range when the
// user omits --window.
const defaultTimeWindow = "this_week"

// handlers owns the notes API client used by the CLI command primitives.
type handlers struct {
	core   *cliapp.ScenarioApp
	client notesconnect.NotesServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: notesconnect.NewNotesServiceClient(httpClient, baseURL),
	}
}

// listCall runs Notes.ListNotes (the proto_list operation).
func (h *handlers) listCall(_ cliapp.OperationContext) (*notesv1.ListNotesResponse, error) {
	resp, err := h.client.ListNotes(context.Background(), connect.NewRequest(&notesv1.ListNotesRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list notes", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no notes response")
	}
	return resp.Msg, nil
}

// listReport renders the list response as the human ListReport.
func (h *handlers) listReport(_ cliapp.OperationContext, msg *notesv1.ListNotesResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.Notes))
	for _, n := range msg.Notes {
		results = append(results, formatNote(n))
	}
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d note(s).", len(msg.Notes))},
		ResultsHeading: "Notes",
		Results:        results,
		RetrievalHints: []string{
			"`notes get <id>` — show a single note by id",
			"`notes create --title <title>` — create a new note",
			"`notes attach <id> --file <path>` — attach a file to a note",
		},
	}
}

// createCall runs Notes.CreateNote (the proto_mutation operation).
func (h *handlers) createCall(ctx cliapp.OperationContext) (*notesv1.CreateNoteResponse, error) {
	resp, err := h.client.CreateNote(context.Background(), connect.NewRequest(&notesv1.CreateNoteRequest{
		Title: ctx.Flag("title"),
		Body:  ctx.Flag("body"),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("create note", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Note == nil {
		return nil, fmt.Errorf("server returned no note")
	}
	return resp.Msg, nil
}

// createReport renders the create response as the human MutationReport.
func (h *handlers) createReport(_ cliapp.OperationContext, msg *notesv1.CreateNoteResponse) cliapp.MutationReport {
	return cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Created note %s.", msg.Note.Id)},
		Changes: []string{formatNote(msg.Note)},
		NextCommand: []string{
			fmt.Sprintf("`notes get %s` — show this note", msg.Note.Id),
			"`notes list` — show all notes",
		},
	}
}

// getCall runs Notes.GetNote for a single id (the proto_list operation).
func (h *handlers) getCall(ctx cliapp.OperationContext) (*notesv1.GetNoteResponse, error) {
	id := ctx.Positional("id")
	resp, err := h.client.GetNote(context.Background(), connect.NewRequest(&notesv1.GetNoteRequest{Id: id}))
	if err != nil {
		return nil, cliapp.WrapAPIError(fmt.Sprintf("get note %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Note == nil {
		return nil, fmt.Errorf("server returned no note")
	}
	return resp.Msg, nil
}

// getReport renders the single-note response as the human ListReport.
func (h *handlers) getReport(_ cliapp.OperationContext, msg *notesv1.GetNoteResponse) cliapp.ListReport {
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched note %s.", msg.Note.Id)},
		ResultsHeading: "Note",
		Results:        []string{formatNote(msg.Note)},
	}
}

// countCall maps --window to the shared TimeWindow proto used by the manifest
// measure, then runs Notes.CountNotes (the proto_list operation).
func (h *handlers) countCall(ctx cliapp.OperationContext) (*notesv1.CountNotesResponse, error) {
	window := strings.TrimSpace(ctx.Flag("window"))
	if window == "" {
		window = defaultTimeWindow
	}
	token, err := timeWindowToken(window)
	if err != nil {
		return nil, err
	}
	resp, err := h.client.CountNotes(context.Background(), connect.NewRequest(&notesv1.CountNotesRequest{
		Window: &measuresv1.TimeWindow{Window: &measuresv1.TimeWindow_Token{Token: token}},
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("count notes", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no count response")
	}
	return resp.Msg, nil
}

// countReport renders the count response as the human ListReport. It re-derives
// the window token for display only (the call already validated it).
func (h *handlers) countReport(ctx cliapp.OperationContext, msg *notesv1.CountNotesResponse) cliapp.ListReport {
	window := strings.TrimSpace(ctx.Flag("window"))
	if window == "" {
		window = defaultTimeWindow
	}
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d note(s) created (%s).", msg.Count, window)},
		ResultsHeading: "Notes created",
		Results:        []string{fmt.Sprintf("%d (%s)", msg.Count, window)},
		RetrievalHints: []string{
			"`notes count --window last_30d` — widen the window",
			"`notes list` — show the notes themselves",
		},
	}
}

// timeWindowToken maps a lowercase canonical token (this_week, last_7d, …) to
// the generated proto enum. Unknown tokens are a usage error, never a silent
// fallback — a wrong window would silently answer the wrong question.
func timeWindowToken(token string) (measuresv1.TimeWindowToken, error) {
	name := "TIME_WINDOW_TOKEN_" + strings.ToUpper(token)
	v, ok := measuresv1.TimeWindowToken_value[name]
	if !ok || measuresv1.TimeWindowToken(v) == measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_UNSPECIFIED {
		return 0, fmt.Errorf("unknown time window %q (use one of: this_week, last_7d, last_30d, this_month, last_month, this_quarter)", token)
	}
	return measuresv1.TimeWindowToken(v), nil
}

// formatNote produces a one-line representation suitable for both
// ListReport and MutationReport result blocks.
func formatNote(n *notesv1.Note) string {
	if n == nil {
		return "(nil)"
	}
	created := ""
	if n.CreatedAt != nil {
		created = n.CreatedAt.AsTime().Format(time.RFC3339)
	}
	return fmt.Sprintf("%s — %s [created=%s attachments=%d]", n.Id, n.Title, created, len(n.AttachmentKeys))
}
