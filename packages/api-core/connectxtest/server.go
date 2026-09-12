package connectxtest

import (
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
)

// StartTestServer mounts Connect services on a mux router and starts an
// httptest.Server. The server is closed through t.Cleanup.
func StartTestServer(t *testing.T, services ...connectx.ServiceMount) *httptest.Server {
	t.Helper()
	router := mux.NewRouter()
	connectx.RegisterServices(router, services...)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return server
}
