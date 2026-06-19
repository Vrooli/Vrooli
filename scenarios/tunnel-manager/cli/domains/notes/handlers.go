package notes

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
	notesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/notes"
	notesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/notes/notes_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// defaultTimeWindow matches the manifest measure's `window` default so the
// CLI and the search-hub auto-answer path resolve the same range when the
// user omits --window.
const defaultTimeWindow = "this_week"

// handlers bundles the closure over *cliapp.ScenarioApp so each
// RunCtx-func has typed access to the API client without re-resolving it.
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

// list calls the generated Connect-RPC Notes.List method. Output routing:
// human consumers see a ListReport; --json consumers see the proto-typed
// ListNotesResponse wire shape, identical to what `curl /Notes/List` returns.
func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListNotes(context.Background(), connect.NewRequest(&notesv1.ListNotesRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list notes", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no notes response")
	}

	results := make([]string, 0, len(resp.Msg.Notes))
	for _, n := range resp.Msg.Notes {
		results = append(results, formatNote(n))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d note(s).", len(resp.Msg.Notes))},
		ResultsHeading: "Notes",
		Results:        results,
		RetrievalHints: []string{
			"`notes get <id>` — show a single note by id",
			"`notes create --title <title>` — create a new note",
			"`notes attach <id> --file <path>` — attach a file to a note",
		},
	})
}

// create calls the generated Connect-RPC Notes.Create method. Required-flag
// enforcement (--title) is handled by cli-core's parser via the ArgSchema
// declared in register.go; this handler is reached only when the flag is
// present.
func (h *handlers) create(ctx cliapp.RunContext) error {
	resp, err := h.client.CreateNote(context.Background(), connect.NewRequest(&notesv1.CreateNoteRequest{
		Title: ctx.Flag("title"),
		Body:  ctx.Flag("body"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("create note", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Note == nil {
		return fmt.Errorf("server returned no note")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Created note %s.", resp.Msg.Note.Id)},
		Changes: []string{formatNote(resp.Msg.Note)},
		NextCommand: []string{
			fmt.Sprintf("`notes get %s` — show this note", resp.Msg.Note.Id),
			"`notes list` — show all notes",
		},
	})
}

// get calls the generated Connect-RPC Notes.Get method for a single note id.
// Required-positional enforcement is handled by cli-core's parser via the
// ArgSchema declared in register.go.
func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetNote(context.Background(), connect.NewRequest(&notesv1.GetNoteRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get note %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Note == nil {
		return fmt.Errorf("server returned no note")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched note %s.", resp.Msg.Note.Id)},
		ResultsHeading: "Note",
		Results:        []string{formatNote(resp.Msg.Note)},
	})
}

// count calls the generated Connect-RPC Notes.CountNotes method — the
// reference measure. It maps the --window token to the shared canonical
// TimeWindow proto (defaulting to this_week, matching the manifest measure
// default) so the same question answered through search-hub and through this
// command resolve the identical range.
func (h *handlers) count(ctx cliapp.RunContext) error {
	window := strings.TrimSpace(ctx.Flag("window"))
	if window == "" {
		window = defaultTimeWindow
	}
	token, err := timeWindowToken(window)
	if err != nil {
		return err
	}
	resp, err := h.client.CountNotes(context.Background(), connect.NewRequest(&notesv1.CountNotesRequest{
		Window: &measuresv1.TimeWindow{Window: &measuresv1.TimeWindow_Token{Token: token}},
	}))
	if err != nil {
		return cliapp.WrapAPIError("count notes", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no count response")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d note(s) created (%s).", resp.Msg.Count, window)},
		ResultsHeading: "Notes created",
		Results:        []string{fmt.Sprintf("%d (%s)", resp.Msg.Count, window)},
		RetrievalHints: []string{
			"`notes count --window last_30d` — widen the window",
			"`notes list` — show the notes themselves",
		},
	})
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
