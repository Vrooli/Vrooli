package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
)

func deterministicDockerServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/_ping"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		case strings.HasSuffix(r.URL.Path, "/version"):
			_ = json.NewEncoder(w).Encode(map[string]string{"ApiVersion": "1.41", "Version": "test"})
		case strings.HasSuffix(r.URL.Path, "/info"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"ID": "test-daemon", "Containers": 0})
		case strings.HasSuffix(r.URL.Path, "/containers/json"):
			_ = json.NewEncoder(w).Encode([]interface{}{})
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestNewDockerHandler(t *testing.T) {
	t.Run("WithDockerClient", func(t *testing.T) {
		// Try to create a Docker client
		dockerClient, _ := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

		handler := NewDockerHandler(dockerClient)

		if handler == nil {
			t.Fatal("Expected non-nil Docker handler")
		}

		if handler.docker != dockerClient {
			t.Error("Expected Docker client to be set")
		}
	})

	t.Run("WithNilClient", func(t *testing.T) {
		handler := NewDockerHandler(nil)

		if handler == nil {
			t.Fatal("Expected non-nil Docker handler even with nil client")
		}

		if handler.docker != nil {
			t.Error("Expected Docker client to be nil")
		}
	})
}

func TestGetDockerInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("DockerNotAvailable", func(t *testing.T) {
		handler := NewDockerHandler(nil)

		router := gin.New()
		router.GET("/docker/info", handler.GetDockerInfo)

		req := httptest.NewRequest("GET", "/docker/info", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}

		expectedBody := `{"error":"Docker not available"}`
		if w.Body.String() != expectedBody {
			t.Errorf("Expected body %s, got %s", expectedBody, w.Body.String())
		}
	})

	t.Run("WithDockerClient", func(t *testing.T) {
		server := deterministicDockerServer(t)
		defer server.Close()
		dockerClient, err := client.NewClientWithOpts(client.WithHost(server.URL), client.WithAPIVersionNegotiation())
		if err != nil {
			t.Fatal(err)
		}

		handler := NewDockerHandler(dockerClient)

		router := gin.New()
		router.GET("/docker/info", handler.GetDockerInfo)

		req := httptest.NewRequest("GET", "/docker/info", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected deterministic Docker info success, got %d: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "test-daemon") {
			t.Fatalf("missing daemon payload: %s", w.Body.String())
		}
	})
}

func TestGetContainers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("DockerNotAvailable", func(t *testing.T) {
		handler := NewDockerHandler(nil)

		router := gin.New()
		router.GET("/docker/containers", handler.GetContainers)

		req := httptest.NewRequest("GET", "/docker/containers", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}

		expectedBody := `{"error":"Docker not available"}`
		if w.Body.String() != expectedBody {
			t.Errorf("Expected body %s, got %s", expectedBody, w.Body.String())
		}
	})

	t.Run("WithDockerClient", func(t *testing.T) {
		server := deterministicDockerServer(t)
		defer server.Close()
		dockerClient, err := client.NewClientWithOpts(client.WithHost(server.URL), client.WithAPIVersionNegotiation())
		if err != nil {
			t.Fatal(err)
		}

		handler := NewDockerHandler(dockerClient)

		router := gin.New()
		router.GET("/docker/containers", handler.GetContainers)

		req := httptest.NewRequest("GET", "/docker/containers", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected deterministic container success, got %d: %s", w.Code, w.Body.String())
		}
		if body := w.Body.String(); body == "" || body[0] != '[' {
			t.Errorf("expected JSON array response, got: %s", body)
		}
	})
}

func TestDockerHandlerEdgeCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("InvalidHTTPMethod", func(t *testing.T) {
		handler := NewDockerHandler(nil)

		router := gin.New()
		router.GET("/docker/info", handler.GetDockerInfo)

		// Try POST instead of GET
		req := httptest.NewRequest("POST", "/docker/info", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound && w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 404 or 405 for invalid method, got %d", w.Code)
		}
	})

	t.Run("ConcurrentRequests", func(t *testing.T) {
		handler := NewDockerHandler(nil)

		router := gin.New()
		router.GET("/docker/info", handler.GetDockerInfo)
		router.GET("/docker/containers", handler.GetContainers)

		// Make concurrent requests
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				req := httptest.NewRequest("GET", "/docker/info", nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				done <- true
			}()
		}

		// Wait for all requests
		for i := 0; i < 10; i++ {
			<-done
		}

		// If we got here without panic, test passes
	})
}

func TestDockerHandlerIntegration(t *testing.T) {
	t.Run("InfoAndContainersCombined", func(t *testing.T) {
		gin.SetMode(gin.TestMode)

		server := deterministicDockerServer(t)
		defer server.Close()
		dockerClient, err := client.NewClientWithOpts(client.WithHost(server.URL), client.WithAPIVersionNegotiation())
		if err != nil {
			t.Fatal(err)
		}

		handler := NewDockerHandler(dockerClient)

		router := gin.New()
		router.GET("/docker/info", handler.GetDockerInfo)
		router.GET("/docker/containers", handler.GetContainers)

		// Test info endpoint
		infoReq := httptest.NewRequest("GET", "/docker/info", nil)
		infoW := httptest.NewRecorder()
		router.ServeHTTP(infoW, infoReq)

		// Test containers endpoint
		containersReq := httptest.NewRequest("GET", "/docker/containers", nil)
		containersW := httptest.NewRecorder()
		router.ServeHTTP(containersW, containersReq)

		if infoW.Code != http.StatusOK || containersW.Code != http.StatusOK {
			t.Fatalf("deterministic Docker endpoints failed: info=%d containers=%d", infoW.Code, containersW.Code)
		}
	})
}
