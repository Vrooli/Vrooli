package notes

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"google.golang.org/protobuf/encoding/protojson"

	errorsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/smoke-tier1/v1/errors"
	notesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/smoke-tier1/v1/notes"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// handlers bundles the closure over *cliapp.ScenarioApp so each
// Run-func has typed access to the API client without re-resolving it.
type handlers struct {
	core *cliapp.ScenarioApp
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	return &handlers{core: core}
}

// list issues GET /api/v1/notes and renders a ListReport on stdout.
func (h *handlers) list(args []string) error {
	body, err := h.core.Get("/notes", nil)
	if err != nil {
		return apiError("list notes", err, body)
	}

	var resp notesv1.ListNotesResponse
	if err := protojson.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("decode ListNotesResponse: %w", err)
	}

	results := make([]string, 0, len(resp.Notes))
	for _, n := range resp.Notes {
		results = append(results, formatNote(n))
	}

	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d note(s).", len(resp.Notes))},
		ResultsHeading: "Notes",
		Results:        results,
		RetrievalHints: []string{
			"`notes get <id>` — show a single note by id",
			"`notes create --title <title>` — create a new note",
		},
	})
}

// create POSTs a note and renders a MutationReport.
func (h *handlers) create(args []string) error {
	fs := flag.NewFlagSet("notes create", flag.ContinueOnError)
	title := fs.String("title", "", "Note title (required)")
	body := fs.String("body", "", "Note body")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *title == "" {
		return fmt.Errorf("--title is required")
	}

	reqBody := map[string]string{"title": *title, "body": *body}
	respBody, err := h.core.Request(http.MethodPost, "/notes", nil, reqBody)
	if err != nil {
		return apiError("create note", err, respBody)
	}

	var resp notesv1.CreateNoteResponse
	if err := protojson.Unmarshal(respBody, &resp); err != nil {
		return fmt.Errorf("decode CreateNoteResponse: %w", err)
	}
	if resp.Note == nil {
		return fmt.Errorf("server returned no note")
	}

	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Created note %s.", resp.Note.Id)},
		Changes: []string{formatNote(resp.Note)},
		NextCommand: []string{
			fmt.Sprintf("`notes get %s` — show this note", resp.Note.Id),
			"`notes list` — show all notes",
		},
	})
}

// get fetches a single note by id (positional argument).
func (h *handlers) get(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing note id; usage: notes get <id>")
	}
	id := args[0]

	body, err := h.core.Get("/notes/"+url.PathEscape(id), nil)
	if err != nil {
		return apiError(fmt.Sprintf("get note %q", id), err, body)
	}

	var resp notesv1.GetNoteResponse
	if err := protojson.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("decode GetNoteResponse: %w", err)
	}
	if resp.Note == nil {
		return fmt.Errorf("server returned no note")
	}

	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
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

// apiError surfaces a typed ErrorEnvelope when the response body
// decodes as one; otherwise wraps the underlying transport error.
//
// The body parameter is the response body returned alongside err on
// 2xx (typically empty when err != nil). On non-2xx, cli-core returns
// a *cliutil.APIError carrying the raw response — that's the load-
// bearing source for the typed envelope here.
func apiError(action string, err error, body []byte) error {
	if err == nil {
		return nil
	}

	// 2xx-with-body callers can pre-pass the bytes; non-2xx callers
	// rely on the APIError's RawResponse below.
	if env, ok := decodeEnvelope(body); ok {
		return fmt.Errorf("%s: %s: %s", action, env.Code, env.Message)
	}

	if apiErr, ok := err.(*cliutil.APIError); ok {
		if env, ok := decodeEnvelope(apiErr.RawResponse); ok {
			return fmt.Errorf("%s: %s: %s", action, env.Code, env.Message)
		}
		return fmt.Errorf("%s: %w", action, apiErr)
	}
	return fmt.Errorf("%s: %w", action, err)
}

func decodeEnvelope(body []byte) (*errorsv1.ErrorEnvelope, bool) {
	if len(body) == 0 {
		return nil, false
	}
	var env errorsv1.ErrorEnvelope
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(body, &env); err != nil {
		return nil, false
	}
	if env.Code == "" {
		return nil, false
	}
	return &env, true
}
