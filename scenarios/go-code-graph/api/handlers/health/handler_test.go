package health_test

import (
	"io"
	"log"
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/healthtest"
	"github.com/vrooli/api-core/module"
	"github.com/vrooli/api-core/schedule"
	httpx "github.com/vrooli/api-core/servertest"
	"go-code-graph/handlers/health"
)

func TestHealthHandler(t *testing.T) {
	healthtest.AssertDatabaseContract(t, "react-vite-test", "1.0.0", func(pingErr error) (int, []byte, int64) {
		pinger := &databasetest.FakePinger{PingErr: pingErr}
		h := health.NewHandler(health.Deps{Pinger: pinger, Service: "react-vite-test", Version: "1.0.0"})
		mod := module.Module{Name: "health", Mount: func(r *mux.Router) {
			r.HandleFunc("/health", h).Methods(http.MethodGet)
		}}
		srv := apihttp.NewScenarioModuleHandler(schedule.System(), log.New(io.Discard, "", 0), mod)
		live := httpx.NewLiveServer(t, httpx.HandlerProviderFunc(func() http.Handler { return srv }))
		resp, body := live.Do(t, http.MethodGet, "/health", nil)
		return resp.StatusCode, body, pinger.Calls.Load()
	})
}
