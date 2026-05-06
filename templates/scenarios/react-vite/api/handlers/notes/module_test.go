package notes_test

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/blobstore"
	apidb "github.com/vrooli/api-core/database"

	notesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/{{SCENARIO_ID}}/v1/notes"
	notesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/{{SCENARIO_ID}}/v1/notes/notes_v1connect"

	"{{SCENARIO_ID}}/handlers/notes"
	"{{SCENARIO_ID}}/internal/clock"
	localdb "{{SCENARIO_ID}}/internal/database"
	internalnotes "{{SCENARIO_ID}}/internal/notes"
	"{{SCENARIO_ID}}/internal/testutil/db"
)

func TestModule_Shape(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(internalnotes.Schema),
	))

	m := notes.ModuleWithBlobStore(d, clock.System{}, blobstore.NewMemoryBlobStore(), log.New(io.Discard, "", 0))

	require.Equal(t, "notes", m.Name)
	require.NotNil(t, m.Mount, "Mount closure must be set")
	require.Same(t, &notes.Endpoints[0], &m.Endpoints[0],
		"Module.Endpoints must reference the package-level Endpoints slice")
	require.Len(t, notes.Endpoints, 4, "notes ships list, create, get, attach")
}

func TestModule_RoutesAreReachable(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(internalnotes.Schema),
	))

	m := notes.ModuleWithBlobStore(d, clock.System{}, blobstore.NewMemoryBlobStore(), log.New(io.Discard, "", 0))
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
			method:       http.MethodPost,
			path:         notesconnect.NotesServiceListNotesProcedure,
			body:         `{}`,
			wantStatus:   http.StatusOK,
			wantContains: `{`,
		},
		{
			name:         "create_happy",
			method:       http.MethodPost,
			path:         notesconnect.NotesServiceCreateNoteProcedure,
			body:         `{"title":"first","body":"hello"}`,
			wantStatus:   http.StatusOK,
			wantContains: `"note"`,
		},
		{
			name:         "create_rejects_empty_title",
			method:       http.MethodPost,
			path:         notesconnect.NotesServiceCreateNoteProcedure,
			body:         `{"title":""}`,
			wantStatus:   http.StatusBadRequest,
			wantContains: `"invalid_argument"`,
		},
		{
			name:         "get_not_found",
			method:       http.MethodPost,
			path:         notesconnect.NotesServiceGetNoteProcedure,
			body:         `{"id":"missing"}`,
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
			req.Header.Set("Content-Type", "application/json")
			rw := httptest.NewRecorder()
			r.ServeHTTP(rw, req)

			require.Equal(t, tc.wantStatus, rw.Code,
				"unexpected status; body=%s", rw.Body.String())
			require.Contains(t, rw.Body.String(), tc.wantContains)
		})
	}
}

// TestEndpoints_ParityWithProtoService is the drift-killer for the
// hand-authored Endpoints metadata. Path strings already link to the
// generated *Procedure constants so renames break compilation; this test
// closes the remaining hole — adding `rpc Foo(...)` to notes.proto without
// adding a notes_foo Endpoints entry would otherwise silently ship a
// .vrooli/endpoints.json that disagrees with the running server.
//
// The proto's notes.NotesService is the source of truth: every method it
// declares must have exactly one matching Endpoints entry, identified by
// the Connect procedure path.
func TestEndpoints_ParityWithProtoService(t *testing.T) {
	svc := notesv1.File_{{SCENARIO_ID_SNAKE}}_v1_notes_notes_proto.
		Services().ByName("NotesService")
	require.NotNil(t, svc, "notes proto must declare a NotesService service")

	byPath := make(map[string]int, len(notes.Endpoints))
	for _, ep := range notes.Endpoints {
		byPath[ep.Path]++
	}

	methods := svc.Methods()
	for i := 0; i < methods.Len(); i++ {
		m := methods.Get(i)
		wantPath := "/" + string(svc.FullName()) + "/" + string(m.Name())
		count := byPath[wantPath]
		require.Equal(t, 1, count,
			"proto method %q (path %q) must have exactly one Endpoints entry; found %d",
			m.Name(), wantPath, count)
	}
}
