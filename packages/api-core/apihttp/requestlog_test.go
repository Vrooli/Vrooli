package apihttp_test

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/scheduletest"
)

func TestRequestLoggingMiddlewareUsesInjectedClock(t *testing.T) {
	clock := scheduletest.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	buffer := &bytes.Buffer{}
	middleware := apihttp.RequestLoggingMiddleware(clock, log.New(buffer, "", 0))
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		clock.Advance(150 * time.Millisecond)
		w.WriteHeader(http.StatusAccepted)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/probe", nil))

	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusAccepted)
	}
	for _, want := range []string{"[GET]", "/probe", "150ms"} {
		if !bytes.Contains(buffer.Bytes(), []byte(want)) {
			t.Errorf("log %q does not contain %q", buffer.String(), want)
		}
	}
}

func TestRequestLoggingMiddlewareDefaultsNilLogger(t *testing.T) {
	middleware := apihttp.RequestLoggingMiddleware(scheduletest.New(time.Time{}), nil)
	handler := middleware(http.NotFoundHandler())
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				t.Errorf("handler panicked: %v", recovered)
			}
		}()
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/missing", nil))
	}()
}
