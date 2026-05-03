package notes

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	errorsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/smoke-tier1/v1/errors"
	notesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/smoke-tier1/v1/notes"

	"github.com/vrooli/cli-core/cliapp"

	"smoke-tier1/cli/internal/testutil"
)

// newCoreFor wires a *cliapp.ScenarioApp pointing at the supplied
// httptest server. The CLI handlers under test consume only the
// (Get, Request) surface; this minimal construction keeps the tests
// readable without standing up the full app shell.
func newCoreFor(t *testing.T, handler http.Handler) *cliapp.ScenarioApp {
	t.Helper()
	srv := testutil.NewAPIServer(t, handler)
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:           "smoke-tier1-test",
		Version:        "0.0.0-test",
		Description:    "Notes handler test",
		DefaultAPIBase: srv.URL,
		AllowAnonymous: true,
	})
	require.NoError(t, err)
	return core
}

// fakeAPI returns an http.Handler routing /api/v1/notes paths to the
// supplied response bytes and capturing the inbound request for
// assertions. Body is `[]byte` so callers feed proto-marshalled
// payloads (via testutil.MustMarshalProto) instead of hand-rolled JSON
// literals — drift between the test wire and the proto schema becomes
// a compile error at the test rather than a silent pass against stale
// JSON.
func fakeAPI(t *testing.T, status int, body []byte) (http.Handler, *http.Request, *string) {
	t.Helper()
	var captured *http.Request
	var capturedBody string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r
		if r.Body != nil {
			b, _ := io.ReadAll(r.Body)
			capturedBody = string(b)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	})
	return handler, captured, &capturedBody
}

func TestNotesList_RendersResults(t *testing.T) {
	// Proto-marshal the response so a future schema change (renamed
	// field, added required field) breaks at the test rather than
	// silently passing against stale JSON literals.
	body := testutil.MustMarshalProto(t, &notesv1.ListNotesResponse{
		Notes: []*notesv1.Note{
			{Id: "a", Title: "first", Body: "", CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z"},
			{Id: "b", Title: "second", Body: "x", CreatedAt: "2026-01-02T00:00:00Z", UpdatedAt: "2026-01-02T00:00:00Z"},
		},
	})
	handler, _, _ := fakeAPI(t, http.StatusOK, body)
	core := newCoreFor(t, handler)

	h := newHandlers(core)
	out := testutil.CaptureStdout(t, func() error { return h.list(nil) })

	require.Contains(t, out, "Found 2 note(s).")
	require.Contains(t, out, "first")
	require.Contains(t, out, "second")
}

func TestNotesList_SurfacesEnvelopeErrors(t *testing.T) {
	envelope := testutil.MustMarshalProto(t, &errorsv1.ErrorEnvelope{
		Code:    "internal",
		Message: "store down",
	})
	handler, _, _ := fakeAPI(t, http.StatusInternalServerError, envelope)
	core := newCoreFor(t, handler)

	h := newHandlers(core)
	err := h.list(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "internal")
	require.Contains(t, err.Error(), "store down")
}

func TestNotesCreate_RequiresTitle(t *testing.T) {
	handler, _, _ := fakeAPI(t, http.StatusOK, []byte(`{}`))
	core := newCoreFor(t, handler)

	h := newHandlers(core)
	err := h.create(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "--title")
}

func TestNotesCreate_PostsTitleAndBody(t *testing.T) {
	respBody := testutil.MustMarshalProto(t, &notesv1.CreateNoteResponse{
		Note: &notesv1.Note{
			Id: "new", Title: "hello", Body: "world",
			CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z",
		},
	})
	var lastBody string
	var lastMethod string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastMethod = r.Method
		b, _ := io.ReadAll(r.Body)
		lastBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(respBody)
	})
	core := newCoreFor(t, handler)

	h := newHandlers(core)
	out := testutil.CaptureStdout(t, func() error {
		return h.create([]string{"--title", "hello", "--body", "world"})
	})

	require.Equal(t, http.MethodPost, lastMethod)
	// The CLI's create handler currently posts a hand-rolled
	// map[string]string — the request shape ought to migrate to a
	// proto-marshalled CreateNoteRequest in lockstep with this test
	// learning to decode it. Until then, json.Unmarshal into a map is
	// the right shape for what's actually on the wire.
	var sent map[string]string
	require.NoError(t, json.Unmarshal([]byte(lastBody), &sent))
	require.Equal(t, "hello", sent["title"])
	require.Equal(t, "world", sent["body"])
	require.Contains(t, out, "Created note new.")
	require.Contains(t, out, "hello")
}

func TestNotesGet_ReportsNotFoundEnvelope(t *testing.T) {
	envelope := testutil.MustMarshalProto(t, &errorsv1.ErrorEnvelope{
		Code:    "not_found",
		Message: `note "ghost" not found`,
	})
	handler, _, _ := fakeAPI(t, http.StatusNotFound, envelope)
	core := newCoreFor(t, handler)

	h := newHandlers(core)
	err := h.get([]string{"ghost"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not_found")
	require.Contains(t, err.Error(), "ghost")
}

func TestNotesGet_RendersNote(t *testing.T) {
	respBody := testutil.MustMarshalProto(t, &notesv1.GetNoteResponse{
		Note: &notesv1.Note{
			Id: "abc", Title: "found", Body: "",
			CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z",
		},
	})
	var lastPath string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respBody)
	})
	core := newCoreFor(t, handler)

	h := newHandlers(core)
	out := testutil.CaptureStdout(t, func() error { return h.get([]string{"abc"}) })

	require.True(t, strings.HasSuffix(lastPath, "/notes/abc"),
		"GET path = %q, want suffix /notes/abc", lastPath)
	require.Contains(t, out, "Fetched note abc.")
	require.Contains(t, out, "found")
}

func TestNotesGet_RequiresID(t *testing.T) {
	handler, _, _ := fakeAPI(t, http.StatusOK, []byte(`{}`))
	core := newCoreFor(t, handler)
	h := newHandlers(core)
	err := h.get(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing note id")
}

// TestRegister_Wiring covers the SubcommandGroup the package exposes,
// proving the surface is registered with the cli-core shape downstream
// callers can dispatch through.
func TestRegister_Wiring(t *testing.T) {
	emptyList := testutil.MustMarshalProto(t, &notesv1.ListNotesResponse{})
	handler, _, _ := fakeAPI(t, http.StatusOK, emptyList)
	core := newCoreFor(t, handler)
	group := Register(core)

	require.Equal(t, "notes", group.Name)
	require.True(t, group.NeedsAPI)
	names := make([]string, 0, len(group.Subcommands))
	for _, sc := range group.Subcommands {
		names = append(names, sc.Name)
	}
	require.ElementsMatch(t, []string{"list", "create", "get"}, names)
}
