package notes_test

import (
	"context"
	"io"
	"log"
	"net/http"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	"{{SCENARIO_ID}}/handlers/notes"
	"{{SCENARIO_ID}}/internal/clock"
	localdb "{{SCENARIO_ID}}/internal/database"
	internalnotes "{{SCENARIO_ID}}/internal/notes"
	"{{SCENARIO_ID}}/internal/testutil/db"
)

// TestModule_Shape pins the public contract: Module returns a fully
// populated module.Module with the canonical name, a non-nil Mount
// closure, and the Endpoints slice referenced from endpoints.go.
func TestModule_Shape(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(internalnotes.Schema),
	))

	m := notes.Module(d, clock.System{}, log.New(io.Discard, "", 0))

	require.Equal(t, "notes", m.Name)
	require.NotNil(t, m.Mount, "Mount closure must be set")
	require.Same(t, &notes.Endpoints[0], &m.Endpoints[0],
		"Module.Endpoints must reference the package-level Endpoints slice (codegen reads it)")
	require.Len(t, notes.Endpoints, 3, "notes ships list, create, get")
}

// TestModule_RoutesAreReachable proves every endpoint declared in
// Endpoints[] is actually mounted by Mount and routed to the handler
// chain. Each case exercises the wire shape end-to-end against a real
// in-memory sqlite, so a regression that breaks routing or the
// service↔repository wiring fails here, not in the e2e gate.
func TestModule_RoutesAreReachable(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(internalnotes.Schema),
	))

	m := notes.Module(d, clock.System{}, log.New(io.Discard, "", 0))
	r := mux.NewRouter()
	m.Mount(r)

	cases := []struct {
		name         string
		method       string
		path         string
		body         string
		wantStatus   int
		wantContains string
	}{
		{
			name:         "list_empty",
			method:       http.MethodGet,
			path:         "/api/v1/notes",
			wantStatus:   http.StatusOK,
			wantContains: `{`,
		},
		{
			name:         "create_happy",
			method:       http.MethodPost,
			path:         "/api/v1/notes",
			body:         `{"title":"first","body":"hello"}`,
			wantStatus:   http.StatusCreated,
			wantContains: `"note"`,
		},
		{
			name:         "create_rejects_empty_title",
			method:       http.MethodPost,
			path:         "/api/v1/notes",
			body:         `{"title":""}`,
			wantStatus:   http.StatusBadRequest,
			wantContains: `"invalid_request"`,
		},
		{
			name:         "get_not_found",
			method:       http.MethodGet,
			path:         "/api/v1/notes/missing",
			wantStatus:   http.StatusNotFound,
			wantContains: `"not_found"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var body io.Reader
			if tc.body != "" {
				body = strings.NewReader(tc.body)
			}
			req, err := http.NewRequest(tc.method, tc.path, body)
			require.NoError(t, err)
			rw := newRecorder()
			r.ServeHTTP(rw, req)

			require.Equal(t, tc.wantStatus, rw.status,
				"unexpected status; body=%s", rw.body.String())
			require.Contains(t, rw.body.String(), tc.wantContains)
		})
	}
}

// recorder is a minimal http.ResponseWriter capture: avoids pulling in
// the live-server harness for what is fundamentally a routing/handler
// smoke. The live-server harness is reserved for cross-cutting tests
// (server_test) that need real socket semantics.
type recorder struct {
	headers http.Header
	body    *strings.Builder
	status  int
}

func newRecorder() *recorder {
	return &recorder{
		headers: http.Header{},
		body:    &strings.Builder{},
		status:  http.StatusOK,
	}
}

func (r *recorder) Header() http.Header        { return r.headers }
func (r *recorder) Write(p []byte) (int, error) { return r.body.Write(p) }
func (r *recorder) WriteHeader(s int)           { r.status = s }
