package apihttp_test

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/module"
	"github.com/vrooli/api-core/schedule"
)

func TestNewModuleRouterMountsRoutesAndWrapsMiddleware(t *testing.T) {
	mounted := 0
	handler := apihttp.NewModuleRouter([]module.Module{{
		Name: "example",
		Mount: func(router *mux.Router) {
			mounted++
			router.HandleFunc("/example", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusAccepted)
			}).Methods(http.MethodGet)
		},
	}}, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			w.Header().Set("X-Test-Middleware", "applied")
			next.ServeHTTP(w, request)
		})
	})

	if mounted != 1 {
		t.Fatalf("module mounted %d times, want 1", mounted)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/example", nil))
	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusAccepted)
	}
	if got := response.Header().Get("X-Test-Middleware"); got != "applied" {
		t.Fatalf("middleware header = %q, want applied", got)
	}
}

func TestNewScenarioModuleHandlerRequiresClockAndMountsModule(t *testing.T) {
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("expected missing clock to panic")
			}
		}()
		apihttp.NewScenarioModuleHandler(nil, nil)
	}()

	handler := apihttp.NewScenarioModuleHandler(schedule.System(), log.New(io.Discard, "", 0), module.Module{
		Name: "example",
		Mount: func(router *mux.Router) {
			router.HandleFunc("/example", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusAccepted) }).Methods(http.MethodGet)
		},
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/example", nil))
	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusAccepted)
	}
}
