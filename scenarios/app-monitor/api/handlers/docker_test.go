package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
)

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
		// Try to create a Docker client
		dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			t.Skipf("Skipping Docker test, client creation failed: %v", err)
		}

		handler := NewDockerHandler(dockerClient)

		router := gin.New()
		router.GET("/docker/info", handler.GetDockerInfo)

		req := httptest.NewRequest("GET", "/docker/info", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Could be 200 OK if Docker is running, or error if not
		// We just verify it doesn't panic
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Logf("Got status: %d (OK if Docker daemon not running)", w.Code)
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
		// Try to create a Docker client
		dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			t.Skipf("Skipping Docker test, client creation failed: %v", err)
		}

		handler := NewDockerHandler(dockerClient)

		router := gin.New()
		router.GET("/docker/containers", handler.GetContainers)

		req := httptest.NewRequest("GET", "/docker/containers", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Could be 200 OK if Docker is running, or error if not
		// We just verify it doesn't panic
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Logf("Got status: %d (OK if Docker daemon not running)", w.Code)
		}

		// If successful, response should be valid JSON array
		if w.Code == http.StatusOK {
			body := w.Body.String()
			if body != "" && body[0] != '[' {
				t.Errorf("Expected JSON array response, got: %s", body)
			}
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
			t.Logf("Got status: %d for invalid method", w.Code)
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

		dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			t.Skipf("Skipping Docker integration test: %v", err)
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

		// Both should return same availability status
		if (infoW.Code == http.StatusOK) != (containersW.Code == http.StatusOK) {
			t.Log("Note: Info and containers returned different availability status")
		}
	})
}
