package notes

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	errorsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/{{SCENARIO_ID}}/v1/errors"
	notesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/{{SCENARIO_ID}}/v1/notes"

	clitest "{{SCENARIO_ID}}/cli/internal/testutil"
)

// captured holds the inbound request and body the fake API saw.
// Returned by reference from fakeAPI so tests assert on it after the
// handler under test runs — not at the moment fakeAPI is constructed.
//
// Pointer-with-mutex (rather than `*http.Request` directly) is the
// load-bearing shape: the handler closure mutates the struct's fields
// when the request arrives; tests read them after dispatch returns.
// The mutex covers the case where a future test fans out concurrent
// CLI calls against one fake.
type captured struct {
	mu     sync.Mutex
	method string
	path   string
	body   string
}

// recorded is the lock-free, value-safe view returned by snapshot.
// Distinct from captured so go vet's copylocks check stays quiet
// (returning a struct embedding sync.Mutex by value is the canonical
// vet violation; this carries the same fields without the lock).
type recorded struct {
	Method string
	Path   string
	Body   string
}

// snapshot returns a copy of the captured fields safe to assert on
// without holding the mutex across require.* calls.
func (c *captured) snapshot() recorded {
	c.mu.Lock()
	defer c.mu.Unlock()
	return recorded{Method: c.method, Path: c.path, Body: c.body}
}

// fakeAPI returns an http.Handler that serves (status, body) and
// records the inbound request method, path, and body into the
// returned *captured. The body argument should be proto-marshalled
// (via clitest.MustMarshalProto) so test wire bodies stay in lockstep
// with the proto schema — drift becomes a compile error rather than a
// silent pass against stale JSON.
func fakeAPI(t *testing.T, status int, body []byte) (http.Handler, *captured) {
	t.Helper()
	rec := &captured{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.mu.Lock()
		rec.method = r.Method
		rec.path = r.URL.Path
		if r.Body != nil {
			b, _ := io.ReadAll(r.Body)
			rec.body = string(b)
		}
		rec.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	})
	return handler, rec
}

func TestNotesList_RendersResults(t *testing.T) {
	// Proto-marshal the response so a future schema change (renamed
	// field, added required field) breaks at the test rather than
	// silently passing against stale JSON literals.
	body := clitest.MustMarshalProto(t, &notesv1.ListNotesResponse{
		Notes: []*notesv1.Note{
			{Id: "a", Title: "first", Body: "", CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z"},
			{Id: "b", Title: "second", Body: "x", CreatedAt: "2026-01-02T00:00:00Z", UpdatedAt: "2026-01-02T00:00:00Z"},
		},
	})
	handler, _ := fakeAPI(t, http.StatusOK, body)
	core := clitest.NewTestApp(t, handler)

	h := newHandlers(core)
	out := clitest.CaptureStdout(t, func() error { return h.list(nil) })

	require.Contains(t, out, "Found 2 note(s).")
	require.Contains(t, out, "first")
	require.Contains(t, out, "second")
}

func TestNotesList_SurfacesEnvelopeErrors(t *testing.T) {
	envelope := clitest.MustMarshalProto(t, &errorsv1.ErrorEnvelope{
		Code:    "internal",
		Message: "store down",
	})
	handler, _ := fakeAPI(t, http.StatusInternalServerError, envelope)
	core := clitest.NewTestApp(t, handler)

	h := newHandlers(core)
	err := h.list(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "internal")
	require.Contains(t, err.Error(), "store down")
}

func TestNotesCreate_RequiresTitle(t *testing.T) {
	handler, _ := fakeAPI(t, http.StatusOK, []byte(`{}`))
	core := clitest.NewTestApp(t, handler)

	h := newHandlers(core)
	err := h.create(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "--title")
}

func TestNotesCreate_PostsTitleAndBody(t *testing.T) {
	respBody := clitest.MustMarshalProto(t, &notesv1.CreateNoteResponse{
		Note: &notesv1.Note{
			Id: "new", Title: "hello", Body: "world",
			CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z",
		},
	})
	handler, rec := fakeAPI(t, http.StatusCreated, respBody)
	core := clitest.NewTestApp(t, handler)

	h := newHandlers(core)
	out := clitest.CaptureStdout(t, func() error {
		return h.create([]string{"--title", "hello", "--body", "world"})
	})

	got := rec.snapshot()
	require.Equal(t, http.MethodPost, got.Method)

	// Decode the wire body via protojson so a future CreateNoteRequest
	// schema change (renamed/added field) breaks at this assertion
	// rather than silently passing against a stale map[string]string
	// shape.
	var sent notesv1.CreateNoteRequest
	require.NoError(t, protojson.Unmarshal([]byte(got.Body), &sent))
	require.Equal(t, "hello", sent.Title)
	require.Equal(t, "world", sent.Body)

	require.Contains(t, out, "Created note new.")
	require.Contains(t, out, "hello")
}

func TestNotesGet_ReportsNotFoundEnvelope(t *testing.T) {
	envelope := clitest.MustMarshalProto(t, &errorsv1.ErrorEnvelope{
		Code:    "not_found",
		Message: `note "ghost" not found`,
	})
	handler, _ := fakeAPI(t, http.StatusNotFound, envelope)
	core := clitest.NewTestApp(t, handler)

	h := newHandlers(core)
	err := h.get([]string{"ghost"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not_found")
	require.Contains(t, err.Error(), "ghost")
}

func TestNotesGet_RendersNote(t *testing.T) {
	respBody := clitest.MustMarshalProto(t, &notesv1.GetNoteResponse{
		Note: &notesv1.Note{
			Id: "abc", Title: "found", Body: "",
			CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z",
		},
	})
	handler, rec := fakeAPI(t, http.StatusOK, respBody)
	core := clitest.NewTestApp(t, handler)

	h := newHandlers(core)
	out := clitest.CaptureStdout(t, func() error { return h.get([]string{"abc"}) })

	got := rec.snapshot()
	require.True(t, strings.HasSuffix(got.Path, "/notes/abc"),
		"GET path = %q, want suffix /notes/abc", got.Path)
	require.Contains(t, out, "Fetched note abc.")
	require.Contains(t, out, "found")
}

func TestNotesGet_RequiresID(t *testing.T) {
	handler, _ := fakeAPI(t, http.StatusOK, []byte(`{}`))
	core := clitest.NewTestApp(t, handler)
	h := newHandlers(core)
	err := h.get(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing note id")
}

// TestRegister_Wiring covers the SubcommandGroup the package exposes,
// proving the surface is registered with the cli-core shape downstream
// callers can dispatch through.
func TestRegister_Wiring(t *testing.T) {
	emptyList := clitest.MustMarshalProto(t, &notesv1.ListNotesResponse{})
	handler, _ := fakeAPI(t, http.StatusOK, emptyList)
	core := clitest.NewTestApp(t, handler)
	group := Register(core)

	require.Equal(t, "notes", group.Name)
	require.True(t, group.NeedsAPI)
	names := make([]string, 0, len(group.Subcommands))
	for _, sc := range group.Subcommands {
		names = append(names, sc.Name)
	}
	require.ElementsMatch(t, []string{"list", "create", "get"}, names)
}
