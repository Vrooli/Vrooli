package notes

import (
	"fmt"
	"net/http"
	"net/url"

	notesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/{{SCENARIO_ID}}/v1/notes"

	"github.com/vrooli/cli-core/cliapp"
)

// handlers bundles the closure over *cliapp.ScenarioApp so each
// RunCtx-func has typed access to the API client without re-resolving it.
type handlers struct {
	core *cliapp.ScenarioApp
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	return &handlers{core: core}
}

// list issues GET /api/v1/notes and renders a ListReport on stdout.
func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := cliapp.CallQuery[*notesv1.ListNotesResponse](h.core, "/notes", nil)
	if err != nil {
		return cliapp.WrapAPIError("list notes", err, nil)
	}

	results := make([]string, 0, len(resp.Notes))
	for _, n := range resp.Notes {
		results = append(results, formatNote(n))
	}

	return ctx.RenderList(cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d note(s).", len(resp.Notes))},
		ResultsHeading: "Notes",
		Results:        results,
		RetrievalHints: []string{
			"`notes get <id>` — show a single note by id",
			"`notes create --title <title>` — create a new note",
		},
	})
}

// create POSTs a note and renders a MutationReport. Required-flag
// enforcement (--title) is handled by cli-core's parser via the
// ArgSchema declared in register.go; this handler is reached only when
// the flag is present.
func (h *handlers) create(ctx cliapp.RunContext) error {
	resp, err := cliapp.Call[*notesv1.CreateNoteRequest, *notesv1.CreateNoteResponse](
		h.core, http.MethodPost, "/notes",
		&notesv1.CreateNoteRequest{Title: ctx.Flag("title"), Body: ctx.Flag("body")},
	)
	if err != nil {
		return cliapp.WrapAPIError("create note", err, nil)
	}
	if resp.Note == nil {
		return fmt.Errorf("server returned no note")
	}

	return ctx.RenderMutation(cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Created note %s.", resp.Note.Id)},
		Changes: []string{formatNote(resp.Note)},
		NextCommand: []string{
			fmt.Sprintf("`notes get %s` — show this note", resp.Note.Id),
			"`notes list` — show all notes",
		},
	})
}

// get fetches a single note by id (positional argument). Required-positional
// enforcement is handled by cli-core's parser via the ArgSchema declared in
// register.go.
func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := cliapp.CallQuery[*notesv1.GetNoteResponse](h.core, "/notes/"+url.PathEscape(id), nil)
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get note %q", id), err, nil)
	}
	if resp.Note == nil {
		return fmt.Errorf("server returned no note")
	}

	return ctx.RenderList(cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched note %s.", resp.Note.Id)},
		ResultsHeading: "Note",
		Results:        []string{formatNote(resp.Note)},
	})
}

// formatNote produces a one-line representation suitable for both
// ListReport and MutationReport result blocks.
func formatNote(n *notesv1.Note) string {
	if n == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s — %s [created=%s]", n.Id, n.Title, n.CreatedAt)
}
