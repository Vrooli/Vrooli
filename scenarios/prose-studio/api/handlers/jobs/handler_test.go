package jobs

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
	internal "prose-studio/internal/jobs"
)

func TestRESTCreatesAndListsOpening(t *testing.T) {
	db, e := sql.Open("sqlite", "file:jobs-handler-test?mode=memory&cache=shared")
	require.NoError(t, e)
	defer db.Close()
	_, e = db.Exec(internal.Schema())
	require.NoError(t, e)
	h := New(internal.New(db, nil))
	create := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/openings", strings.NewReader(`{"company":"Acme","title":"Editor"}`))
	created := httptest.NewRecorder()
	h.ServeHTTP(created, create)
	require.Equal(t, http.StatusCreated, created.Code)
	require.Contains(t, created.Body.String(), "\"opening\"")
	list := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/openings", nil)
	listed := httptest.NewRecorder()
	h.ServeHTTP(listed, list)
	require.Equal(t, http.StatusOK, listed.Code)
	require.Contains(t, listed.Body.String(), "Acme")
}
