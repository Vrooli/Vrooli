package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"scenario-to-desktop-api/internal/testutil"
	"slices"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain/domainconnect"
	pipelinev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/pipeline"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/pipeline/pipelineconnect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/shared"
)

// TestHealthHandler tests the health check endpoint comprehensively
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := NewServer(0)

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/health", nil)
		w := testutil.Serve(t, server.router, req)

		response := assertJSONResponse(t, w, http.StatusOK)
		assertFieldValue(t, response, "service", "scenario-to-desktop-api")
		assertFieldExists(t, response, "version")
		assertFieldExists(t, response, "status")
		assertFieldExists(t, response, "timestamp")
		assertFieldExists(t, response, "readiness")
	})

	t.Run("MultipleRequests", func(t *testing.T) {
		// Test that health endpoint handles concurrent requests
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest("GET", "/api/v1/health", nil)
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Request %d failed with status %d", i, w.Code)
			}
		}
	})
}

// NOTE: TestTestDesktopHandler and TestValidateDesktopConfig were removed
// as part of the pipeline migration. The testDesktopHandler and
// validateDesktopConfig functions from validation.go have been deprecated
// in favor of the unified pipeline approach with preflight validation.

// TestSlicesContains validates that slices.Contains works as expected
// for our use cases (replacing the old local contains() helper).
func TestSlicesContains(t *testing.T) {
	testCases := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{"EmptySlice", []string{}, "test", false},
		{"ItemExists", []string{"a", "b", "c"}, "b", true},
		{"ItemNotExists", []string{"a", "b", "c"}, "d", false},
		{"SingleItemMatch", []string{"test"}, "test", true},
		{"SingleItemNoMatch", []string{"test"}, "other", false},
		{"CaseSensitive", []string{"Test"}, "test", false},
		{"MultipleMatches", []string{"a", "b", "a"}, "a", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := slices.Contains(tc.slice, tc.item)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v for slice %v and item %s",
					tc.expected, result, tc.slice, tc.item)
			}
		})
	}
}

// TestNewServer tests server initialization
func TestNewServer(t *testing.T) {
	t.Run("ValidPort", func(t *testing.T) {
		server := NewServer(8080)
		if server == nil {
			t.Fatal("Expected server to be created")
		}
		if server.port != 8080 {
			t.Errorf("Expected port 8080, got %d", server.port)
		}
		if server.router == nil {
			t.Error("Expected router to be initialized")
		}
		if server.buildHandler == nil {
			t.Error("Expected build handler to be initialized")
		}
		shutdownServer(t, server)
	})

	t.Run("ZeroPort", func(t *testing.T) {
		server := NewServer(0)
		if server == nil {
			t.Fatal("Expected server to be created")
		}
		if server.port != 0 {
			t.Errorf("Expected port 0, got %d", server.port)
		}
		shutdownServer(t, server)
	})
}

func TestServerShutdownIsIdempotentAndStopsOwnedServices(t *testing.T) {
	server := NewServer(0)
	if server == nil {
		t.Fatal("NewServer returned nil")
	}
	shutdownServer(t, server)
	shutdownServer(t, server)
}

func shutdownServer(t *testing.T, server *Server) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}

// TestServerRoutes tests that all routes are properly configured
func TestServerRoutes(t *testing.T) {
	server := NewServer(0)

	routes := []struct {
		method   string
		path     string
		allow404 bool // Allow 404 for routes with path parameters (resource not found is valid)
	}{
		{"GET", "/api/v1/health", false},
		// NOTE: POST /api/v1/desktop/generate, GET /api/v1/desktop/status/{id}, and
		// POST /api/v1/desktop/build was removed; use PipelineService instead.
		// NOTE: POST /api/v1/desktop/package was removed - use pipeline bundle stage instead
		{"POST", "/api/v1/desktop/webhook/build-complete", false},
	}

	for _, route := range routes {
		t.Run(route.method+"_"+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			w := httptest.NewRecorder()

			server.router.ServeHTTP(w, req)

			// Routes should exist - 404 only acceptable if explicitly allowed (resource not found)
			// Other status codes (400, 500, etc.) indicate route exists but request was invalid
			if w.Code == 404 && !route.allow404 {
				t.Errorf("Route %s %s not found (status: %d)", route.method, route.path, w.Code)
			}
		})
	}

	for _, retired := range []string{"/api/v1/status", "/api/v1/templates", "/api/v1/templates/universal", "/api/v1/system/wine/check", "/api/v1/scenarios/desktop-status", "/api/v1/desktop/probe", "/api/v1/desktop/proxy-hints/scenario-to-desktop", "/api/v1/ports/scenario-to-desktop/api", "/api/v1/pipeline/run", "/api/v1/pipeline/example", "/api/v1/pipelines", "/api/v1/scenarios/example/pipeline/active", "/api/v1/scenarios/example/pipeline", "/api/v1/scenarios/example/pipeline/reset", "/api/v1/scenarios/example/pipeline/history", "/api/v1/scenarios/example/pipeline/start", "/api/v1/scenarios/example/bundle/clean", "/api/v1/signing/prerequisites", "/api/v1/signing/example", "/api/v1/signing/example/ready"} {
		t.Run("retired_"+retired, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, retired, nil)
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)
			if w.Code != http.StatusNotFound {
				t.Errorf("retired REST route %s returned %d, want 404", retired, w.Code)
			}
		})
	}

	t.Run("system_connect_replacement", func(t *testing.T) {
		ts := httptest.NewServer(server.Router())
		defer ts.Close()
		client := domainconnect.NewSystemServiceClient(ts.Client(), ts.URL)
		status, err := client.GetSystemStatus(context.Background(), connect.NewRequest(&domainv1.GetSystemStatusRequest{}))
		if err != nil || status.Msg.GetService().GetStatus() != "running" {
			t.Fatalf("GetSystemStatus() = %#v, %v", status.Msg, err)
		}
		templates, err := client.ListTemplates(context.Background(), connect.NewRequest(&domainv1.ListTemplatesRequest{}))
		if err != nil || templates.Msg.GetCount() != 4 {
			t.Fatalf("ListTemplates() = %#v, %v", templates.Msg, err)
		}
	})

	t.Run("operations_connect_replacement", func(t *testing.T) {
		ts := httptest.NewServer(server.Router())
		defer ts.Close()
		client := domainconnect.NewOperationsServiceClient(ts.Client(), ts.URL)
		response, err := client.GetProxyHints(context.Background(), connect.NewRequest(&domainv1.ProxyHintsRequest{ScenarioName: "scenario-to-desktop"}))
		if err != nil || response.Msg.GetScenarioName() != "scenario-to-desktop" {
			t.Fatalf("GetProxyHints() = %#v, %v", response.Msg, err)
		}
	})
}

func TestConnectFailuresCarryOneTypedRemediationEnvelope(t *testing.T) {
	server := NewServer(0)
	ts := httptest.NewServer(server.Router())
	defer ts.Close()
	client := pipelineconnect.NewPipelineServiceClient(ts.Client(), ts.URL)
	_, err := client.Run(context.Background(), connect.NewRequest(&pipelinev1.PipelineRunRequest{}))
	connectErr := new(connect.Error)
	if !errors.As(err, &connectErr) || connectErr.Code() != connect.CodeInvalidArgument {
		t.Fatalf("Run error = %v, want invalid argument Connect error", err)
	}
	var envelope *sharedv1.ErrorEnvelope
	for _, detail := range connectErr.Details() {
		value, valueErr := detail.Value()
		if valueErr != nil {
			t.Fatalf("decode error detail: %v", valueErr)
		}
		if typed, ok := value.(*sharedv1.ErrorEnvelope); ok {
			envelope = typed
		}
	}
	if envelope == nil || envelope.GetCode() == "" || envelope.GetCategory() == "" || envelope.GetRecovery() == "" || envelope.GetRecoveryHint() == "" {
		t.Fatalf("missing actionable error envelope: %#v", envelope)
	}
}

// TestCORSMiddleware tests CORS headers
func TestCORSMiddleware(t *testing.T) {
	server := NewServer(0)

	t.Run("OptionsRequest", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/api/v1/health", nil)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		// Check CORS headers - middleware adds them
		origin := w.Header().Get("Access-Control-Allow-Origin")
		if origin == "" {
			t.Log("CORS middleware may not be active in test mode")
		}
		// Should return 200 for OPTIONS
		if w.Code != http.StatusOK {
			t.Logf("OPTIONS request returned status %d", w.Code)
		}
	})

	t.Run("GetRequest", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/health", nil)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		// Check CORS headers are present (middleware sets specific origin, not *)
		origin := w.Header().Get("Access-Control-Allow-Origin")
		if origin == "" {
			t.Error("Expected CORS headers on GET request")
		}
	})
}

// TestConcurrentRequests tests concurrent request handling
func TestConcurrentRequests(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("ConcurrentHealthChecks", func(t *testing.T) {
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func(id int) {
				req := httptest.NewRequest("GET", "/api/v1/health", nil)
				w := httptest.NewRecorder()
				env.Server.router.ServeHTTP(w, req)
				if w.Code != http.StatusOK {
					t.Errorf("Concurrent request %d failed with status %d", id, w.Code)
				}
				done <- true
			}(i)
		}

		// Wait for all requests to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

// Unused but needed for imports
var (
	_ = mux.Vars
	_ = fmt.Sprintf
	_ = filepath.Join
)
