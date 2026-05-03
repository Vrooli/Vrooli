package notes

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"

	"{{SCENARIO_ID}}/cli/internal/testutil"
)

// newCoreFor wires a *cliapp.ScenarioApp pointing at the supplied
// httptest server. The CLI handlers under test consume only the
// (Get, Request) surface; this minimal construction keeps the tests
// readable without standing up the full app shell.
func newCoreFor(t *testing.T, handler http.Handler) *cliapp.ScenarioApp {
	t.Helper()
	srv := testutil.NewAPIServer(t, handler)
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:           "{{SCENARIO_ID}}-test",
		Version:        "0.0.0-test",
		Description:    "Notes handler test",
		DefaultAPIBase: srv.URL,
		AllowAnonymous: true,
	})
	require.NoError(t, err)
	return core
}

// fakeAPI returns an http.Handler routing /api/v1/notes paths to the
// supplied response and capturing the inbound request for assertions.
func fakeAPI(t *testing.T, status int, body string) (http.Handler, *http.Request, *string) {
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
		_, _ = io.WriteString(w, body)
	})
	return handler, captured, &capturedBody
}

func TestNotesList_RendersResults(t *testing.T) {
	body := `{"notes":[
		{"id":"a","title":"first","body":"","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"},
		{"id":"b","title":"second","body":"x","created_at":"2026-01-02T00:00:00Z","updated_at":"2026-01-02T00:00:00Z"}
	]}`
	handler, _, _ := fakeAPI(t, http.StatusOK, body)
	core := newCoreFor(t, handler)

	h := newHandlers(core)
	out := testutil.CaptureStdout(t, func() error { return h.list(nil) })

	require.Contains(t, out, "Found 2 note(s).")
	require.Contains(t, out, "first")
	require.Contains(t, out, "second")
}

func TestNotesList_SurfacesEnvelopeErrors(t *testing.T) {
	envelope := `{"code":"internal","message":"store down"}`
	handler, _, _ := fakeAPI(t, http.StatusInternalServerError, envelope)
	core := newCoreFor(t, handler)

	h := newHandlers(core)
	err := h.list(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "internal")
	require.Contains(t, err.Error(), "store down")
}

func TestNotesCreate_RequiresTitle(t *testing.T) {
	handler, _, _ := fakeAPI(t, http.StatusOK, `{}`)
	core := newCoreFor(t, handler)

	h := newHandlers(core)
	err := h.create(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "--title")
}

func TestNotesCreate_PostsTitleAndBody(t *testing.T) {
	respBody := `{"note":{"id":"new","title":"hello","body":"world","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}}`
	var lastBody string
	var lastMethod string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastMethod = r.Method
		b, _ := io.ReadAll(r.Body)
		lastBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, respBody)
	})
	core := newCoreFor(t, handler)

	h := newHandlers(core)
	out := testutil.CaptureStdout(t, func() error {
		return h.create([]string{"--title", "hello", "--body", "world"})
	})

	require.Equal(t, http.MethodPost, lastMethod)
	var sent map[string]string
	require.NoError(t, json.Unmarshal([]byte(lastBody), &sent))
	require.Equal(t, "hello", sent["title"])
	require.Equal(t, "world", sent["body"])
	require.Contains(t, out, "Created note new.")
	require.Contains(t, out, "hello")
}

func TestNotesGet_ReportsNotFoundEnvelope(t *testing.T) {
	envelope := `{"code":"not_found","message":"note \"ghost\" not found"}`
	handler, _, _ := fakeAPI(t, http.StatusNotFound, envelope)
	core := newCoreFor(t, handler)

	h := newHandlers(core)
	err := h.get([]string{"ghost"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not_found")
	require.Contains(t, err.Error(), "ghost")
}

func TestNotesGet_RendersNote(t *testing.T) {
	respBody := `{"note":{"id":"abc","title":"found","body":"","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}}`
	var lastPath string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, respBody)
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
	handler, _, _ := fakeAPI(t, http.StatusOK, `{}`)
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
	handler, _, _ := fakeAPI(t, http.StatusOK, `{"notes":[]}`)
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
