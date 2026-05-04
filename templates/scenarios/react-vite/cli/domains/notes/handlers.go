package notes

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	notesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/{{SCENARIO_ID}}/v1/notes"
	notesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/{{SCENARIO_ID}}/v1/notes/notes_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// handlers bundles the closure over *cliapp.ScenarioApp so each
// RunCtx-func has typed access to the API client without re-resolving it.
type handlers struct {
	core   *cliapp.ScenarioApp
	client notesconnect.NotesClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: notesconnect.NewNotesClient(httpClient, baseURL),
	}
}

// list calls the generated Connect-RPC Notes.List method and renders a
// ListReport on stdout.
func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.List(context.Background(), connect.NewRequest(&notesv1.ListNotesRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list notes", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no notes response")
	}

	results := make([]string, 0, len(resp.Msg.Notes))
	notes := make([]noteJSON, 0, len(resp.Msg.Notes))
	for _, n := range resp.Msg.Notes {
		results = append(results, formatNote(n))
		notes = append(notes, toNoteJSON(n))
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d note(s).", len(resp.Msg.Notes))},
		ResultsHeading: "Notes",
		Results:        results,
		RetrievalHints: []string{
			"`notes get <id>` — show a single note by id",
			"`notes create --title <title>` — create a new note",
			"`notes attach <id> --file <path>` — attach a file to a note",
		},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), notesListJSON{
			Summary:        report.Summary,
			Notes:          notes,
			RetrievalHints: report.RetrievalHints,
		})
	}
	return ctx.RenderList(report)
}

// create calls the generated Connect-RPC Notes.Create method and renders a
// MutationReport. Required-flag enforcement (--title) is handled by cli-core's
// parser via the ArgSchema declared in register.go; this handler is reached
// only when the flag is present.
func (h *handlers) create(ctx cliapp.RunContext) error {
	resp, err := h.client.Create(context.Background(), connect.NewRequest(&notesv1.CreateNoteRequest{
		Title: ctx.Flag("title"),
		Body:  ctx.Flag("body"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("create note", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Note == nil {
		return fmt.Errorf("server returned no note")
	}

	report := cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Created note %s.", resp.Msg.Note.Id)},
		Changes: []string{formatNote(resp.Msg.Note)},
		NextCommand: []string{
			fmt.Sprintf("`notes get %s` — show this note", resp.Msg.Note.Id),
			"`notes list` — show all notes",
		},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), noteMutationJSON{
			Result:      report.Result,
			Note:        toNoteJSON(resp.Msg.Note),
			NextCommand: report.NextCommand,
		})
	}
	return ctx.RenderMutation(report)
}

// get calls the generated Connect-RPC Notes.Get method for a single note id.
// Required-positional enforcement is handled by cli-core's parser via the
// ArgSchema declared in register.go.
func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.Get(context.Background(), connect.NewRequest(&notesv1.GetNoteRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get note %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Note == nil {
		return fmt.Errorf("server returned no note")
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched note %s.", resp.Msg.Note.Id)},
		ResultsHeading: "Note",
		Results:        []string{formatNote(resp.Msg.Note)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), noteGetJSON{
			Summary: report.Summary,
			Note:    toNoteJSON(resp.Msg.Note),
		})
	}
	return ctx.RenderList(report)
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

type noteJSON struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Body           string   `json:"body"`
	CreatedAt      string   `json:"created_at,omitempty"`
	UpdatedAt      string   `json:"updated_at,omitempty"`
	AttachmentKeys []string `json:"attachment_keys,omitempty"`
}

type notesListJSON struct {
	Summary        []string   `json:"summary,omitempty"`
	Notes          []noteJSON `json:"notes"`
	RetrievalHints []string   `json:"retrieval_hints,omitempty"`
}

type noteMutationJSON struct {
	Result      []string `json:"result,omitempty"`
	Note        noteJSON `json:"note"`
	NextCommand []string `json:"next_command,omitempty"`
}

type noteGetJSON struct {
	Summary []string `json:"summary,omitempty"`
	Note    noteJSON `json:"note"`
}

func toNoteJSON(n *notesv1.Note) noteJSON {
	if n == nil {
		return noteJSON{}
	}
	out := noteJSON{
		ID:             n.Id,
		Title:          n.Title,
		Body:           n.Body,
		AttachmentKeys: append([]string(nil), n.AttachmentKeys...),
	}
	if n.CreatedAt != nil {
		out.CreatedAt = n.CreatedAt.AsTime().Format(time.RFC3339)
	}
	if n.UpdatedAt != nil {
		out.UpdatedAt = n.UpdatedAt.AsTime().Format(time.RFC3339)
	}
	return out
}
