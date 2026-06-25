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

	notesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/notes/notes_v1connect"

	"plan-manager/handlers/notes"
	"plan-manager/internal/clock"
	localdb "plan-manager/internal/database"
	internalnotes "plan-manager/internal/notes"
	"plan-manager/internal/testutil/db"
)

func TestModule_Shape(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(internalnotes.Schema),
	))

	m := notes.ModuleWithBlobStore(apidb.NewFromPrimary(d), clock.System{}, blobstore.NewMemoryBlobStore(), log.New(io.Discard, "", 0))

	require.Equal(t, "notes", m.Name)
	require.NotNil(t, m.Mount, "Mount closure must be set")
	require.Same(t, &notes.Endpoints[0], &m.Endpoints[0],
		"Module.Endpoints must reference the package-level Endpoints slice")
	require.Len(t, notes.Endpoints, 5, "notes ships list, create, get, count, attach")
}

func TestModule_RoutesAreReachable(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(internalnotes.Schema),
	))

	m := notes.ModuleWithBlobStore(apidb.NewFromPrimary(d), clock.System{}, blobstore.NewMemoryBlobStore(), log.New(io.Discard, "", 0))
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

// Proto/Connect parity for the notes domain is now enforced globally by
// TestProtoConnectParity in api/internal/modules/registry_test.go,
// which walks every entry returned by modules.AllProtoFiles() — adding
// a new domain there gets parity coverage automatically, no per-domain
// test required.
