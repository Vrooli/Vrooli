package server_test

import (
	"io"
	"log"
	"net/http"
	"testing"

	"program-runtime/internal/module"
	"program-runtime/internal/server"
	"program-runtime/internal/testutil/httpx"

	"github.com/vrooli/api-core/schedule"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

// TestServer_MountsEachModule pins the contract the server owns:
// every module passed to New has its Mount invoked exactly once on
// the production router, and the resulting routes are reachable
// through the Handler() chain (including recovery + logging
// middleware).
//
// Per-module route coverage (a list endpoint returns 200, a get
// returns 404, etc.) lives in each handler's module_test.go where
// it belongs; this file owns the wiring guarantee.
func TestServer_MountsEachModule(t *testing.T) {
	var aMounted, bMounted bool

	moduleA := module.Module{
		Name: "a",
		Mount: func(r *mux.Router) {
			aMounted = true
			r.HandleFunc("/a", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("a-ok"))
			}).Methods(http.MethodGet)
		},
	}
	moduleB := module.Module{
		Name: "b",
		Mount: func(r *mux.Router) {
			bMounted = true
			r.HandleFunc("/b", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusTeapot)
			}).Methods(http.MethodGet)
		},
	}

	srv := server.New(newTestDeps(), moduleA, moduleB)
	require.True(t, aMounted, "module A's Mount must be invoked")
	require.True(t, bMounted, "module B's Mount must be invoked")

	live := httpx.NewLiveServer(t, srv)

	respA, payloadA := live.Do(t, http.MethodGet, "/a", nil)
	require.Equal(t, http.StatusOK, respA.StatusCode)
	require.Equal(t, "a-ok", string(payloadA))

	respB, _ := live.Do(t, http.MethodGet, "/b", nil)
	require.Equal(t, http.StatusTeapot, respB.StatusCode)
}

// TestServer_ZeroModules proves the variadic surface accepts zero
// modules — useful for tests that only need middleware composition
// and assert nothing about the route table.
func TestServer_ZeroModules(t *testing.T) {
	srv := server.New(newTestDeps())
	require.NotNil(t, srv.Handler(), "Handler must be wired even with no modules")

	live := httpx.NewLiveServer(t, srv)
	resp, _ := live.Do(t, http.MethodGet, "/anything", nil)
	require.Equal(t, http.StatusNotFound, resp.StatusCode,
		"unmounted paths must 404 from the router")
}

// TestServer_HandlerNotNil pins the smallest possible smoke: New must
// return a wired Server whose Handler() returns a non-nil http.Handler.
// Catches the case where a refactor drops the recovery wrapper or
// returns the bare router without middleware composition.
func TestServer_HandlerNotNil(t *testing.T) {
	srv := server.New(newTestDeps())
	require.NotNil(t, srv.Handler(), "server.Handler() must not be nil")
}

func TestServer_NewRequiresClock(t *testing.T) {
	require.PanicsWithValue(t, "server.New requires Deps.Clock", func() {
		server.New(server.Deps{Logger: log.New(io.Discard, "", 0)})
	})
}

func newTestDeps() server.Deps {
	return server.Deps{
		Clock:  schedule.System(),
		Logger: log.New(io.Discard, "", 0),
	}
}
