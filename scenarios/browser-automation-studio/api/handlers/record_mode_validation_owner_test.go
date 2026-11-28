package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/vrooli/browser-automation-studio/internal/httpjson"
)

func TestRecordModeAuthoringRoutesShareMissingSessionOwner(t *testing.T) {
	handler := &Handler{}
	routes := []struct {
		name string
		run  func(http.ResponseWriter, *http.Request)
	}{
		{name: "selector validation", run: handler.ValidateSelector},
		{name: "replay preview", run: handler.ReplayRecordingPreview},
	}
	var canonicalBody string
	for _, route := range routes {
		t.Run(route.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live//authoring-check", strings.NewReader(`{}`))
			routeContext := chi.NewRouteContext()
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeContext))
			response := httptest.NewRecorder()

			route.run(response, req)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusBadRequest, response.Body.String())
			}
			if !strings.Contains(response.Body.String(), `"sessionId"`) && !strings.Contains(response.Body.String(), `"session_id"`) {
				t.Fatalf("missing-session response does not identify its field: %s", response.Body.String())
			}
			if canonicalBody == "" {
				canonicalBody = response.Body.String()
			} else if response.Body.String() != canonicalBody {
				t.Fatalf("callers returned different canonical validation responses:\nfirst: %s\nnext:  %s", canonicalBody, response.Body.String())
			}
		})
	}
}

func TestRecordModeAuthoringRoutesShareBoundedJSONBodyOwner(t *testing.T) {
	handler := &Handler{}
	routes := []struct {
		name string
		run  func(http.ResponseWriter, *http.Request)
	}{
		{name: "selector validation", run: handler.ValidateSelector},
		{name: "replay preview", run: handler.ReplayRecordingPreview},
	}
	oversized := bytes.Repeat([]byte(" "), int(httpjson.MaxBodyBytes())+1)
	var canonicalBody string
	for _, route := range routes {
		t.Run(route.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live/session/authoring-check", bytes.NewReader(oversized))
			routeContext := chi.NewRouteContext()
			routeContext.URLParams.Add("sessionId", "session")
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeContext))
			response := httptest.NewRecorder()

			route.run(response, req)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusBadRequest, response.Body.String())
			}
			if canonicalBody == "" {
				canonicalBody = response.Body.String()
			} else if response.Body.String() != canonicalBody {
				t.Fatalf("callers returned different bounded-body errors:\nfirst: %s\nnext:  %s", canonicalBody, response.Body.String())
			}
		})
	}
}
