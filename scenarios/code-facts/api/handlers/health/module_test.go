package health_test

import (
	"net/http"
	"strings"
	"testing"

	"code-facts/handlers/health"
	"code-facts/internal/testutil/mocks"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

// TestModule_Shape pins the public contract.
func TestModule_Shape(t *testing.T) {
	m := health.Module(&mocks.FakePinger{}, "react-vite-test", "1.0.0")

	require.Equal(t, "health", m.Name)
	require.NotNil(t, m.Mount)
	require.Same(t, &health.Endpoints[0], &m.Endpoints[0],
		"Module.Endpoints must reference the package-level Endpoints slice")
	require.Len(t, health.Endpoints, 1, "health ships a single endpoint with /api/v1/health as a path alias")
}

// TestModule_BothAliasesReachable proves the /health and /api/v1/health
// paths both route to the same handler. A regression that drops one
// alias (forgot to register, typo) fails here, not in the e2e gate.
func TestModule_BothAliasesReachable(t *testing.T) {
	m := health.Module(&mocks.FakePinger{}, "react-vite-test", "1.0.0")
	r := mux.NewRouter()
	m.Mount(r)

	for _, path := range []string{"/health", "/api/v1/health"} {
		t.Run(path, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, path, nil)
			require.NoError(t, err)
			rw := newRecorder()
			r.ServeHTTP(rw, req)

			require.Equal(t, http.StatusOK, rw.status,
				"unexpected status; body=%s", rw.body.String())
			require.Contains(t, rw.body.String(), `"status"`,
				"response must include the status field")
		})
	}
}

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

func (r *recorder) Header() http.Header         { return r.headers }
func (r *recorder) Write(p []byte) (int, error) { return r.body.Write(p) }
func (r *recorder) WriteHeader(s int)           { r.status = s }
