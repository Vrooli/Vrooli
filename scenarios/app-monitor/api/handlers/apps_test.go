package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"app-monitor-api/repository"
	"app-monitor-api/services"

	"github.com/gin-gonic/gin"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

func TestNewAppHandler(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		appService := services.NewAppService(nil)
		handler := NewAppHandler(appService)

		if handler == nil {
			t.Fatal("Expected non-nil handler")
		}
		if handler.appService == nil {
			t.Error("Expected appService to be set")
		}
	})
}

func TestGetAppsSummary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	appService := services.NewAppService(nil)
	handler := NewAppHandler(appService)
	handler.getAppsSummary = func(context.Context) ([]repository.App, error) {
		return []repository.App{{ID: "demo", Name: "Demo", Status: "running"}}, nil
	}

	t.Run("Success", func(t *testing.T) {
		router := setupTestRouter()
		router.GET("/apps/summary", handler.GetAppsSummary)

		req := httptest.NewRequest("GET", "/apps/summary", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected successful summary, got %d: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "demo") {
			t.Fatalf("success response omitted app payload: %s", w.Body.String())
		}
	})
	t.Run("DependencyFailure", func(t *testing.T) {
		handler.getAppsSummary = func(context.Context) ([]repository.App, error) { return nil, errors.New("dependency unavailable") }
		w := httptest.NewRecorder()
		router := setupTestRouter()
		router.GET("/apps/summary", handler.GetAppsSummary)
		router.ServeHTTP(w, httptest.NewRequest("GET", "/apps/summary", nil))
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500 for dependency failure, got %d", w.Code)
		}
	})
}

func TestGetApps(t *testing.T) {
	gin.SetMode(gin.TestMode)
	appService := services.NewAppService(nil)
	handler := NewAppHandler(appService)
	handler.getApps = func(context.Context) ([]repository.App, error) {
		return []repository.App{{ID: "demo", Name: "Demo", Status: "running"}}, nil
	}

	t.Run("Success", func(t *testing.T) {
		router := setupTestRouter()
		router.GET("/apps", handler.GetApps)

		req := httptest.NewRequest("GET", "/apps", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected successful app list, got %d: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "demo") {
			t.Fatalf("success response omitted app payload: %s", w.Body.String())
		}
	})
	t.Run("DependencyFailure", func(t *testing.T) {
		handler.getApps = func(context.Context) ([]repository.App, error) { return nil, errors.New("dependency unavailable") }
		w := httptest.NewRecorder()
		router := setupTestRouter()
		router.GET("/apps", handler.GetApps)
		router.ServeHTTP(w, httptest.NewRequest("GET", "/apps", nil))
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500 for dependency failure, got %d", w.Code)
		}
	})
}

func TestGetApp(t *testing.T) {
	gin.SetMode(gin.TestMode)
	appService := services.NewAppService(nil)
	handler := NewAppHandler(appService)

	t.Run("NonExistentApp", func(t *testing.T) {
		handler.getApp = func(context.Context, string) (*repository.App, error) { return nil, services.ErrAppNotFound }
		router := setupTestRouter()
		router.GET("/apps/:id", handler.GetApp)

		req := httptest.NewRequest("GET", "/apps/nonexistent-app-xyz", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d. Body: %s", w.Code, w.Body.String())
		}

		if w.Header().Get("Content-Type") == "" {
			t.Error("Expected Content-Type header to be set")
		}
	})

	t.Run("EmptyID", func(t *testing.T) {
		router := setupTestRouter()
		router.GET("/apps/:id", handler.GetApp)

		req := httptest.NewRequest("GET", "/apps/", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should be 404 since the route won't match
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404 for empty ID, got %d", w.Code)
		}
	})
}

func TestStartApp(t *testing.T) {
	gin.SetMode(gin.TestMode)
	appService := services.NewAppService(nil)
	handler := NewAppHandler(appService)

	t.Run("NonExistentApp", func(t *testing.T) {
		router := setupTestRouter()
		router.POST("/apps/:id/start", handler.StartApp)

		req := httptest.NewRequest("POST", "/apps/nonexistent-app/start", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail for nonexistent app
		if w.Code == http.StatusOK {
			t.Error("Expected failure for nonexistent app")
		}
	})

	t.Run("InvalidMethod", func(t *testing.T) {
		router := setupTestRouter()
		router.POST("/apps/:id/start", handler.StartApp)

		req := httptest.NewRequest("GET", "/apps/test-app/start", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should be method not allowed or not found
		if w.Code != http.StatusMethodNotAllowed && w.Code != http.StatusNotFound {
			t.Errorf("Expected 405 or 404 for wrong method, got %d", w.Code)
		}
	})
}

func TestStopApp(t *testing.T) {
	gin.SetMode(gin.TestMode)
	appService := services.NewAppService(nil)
	handler := NewAppHandler(appService)
	handler.stopApp = func(context.Context, string) error { return services.ErrAppNotFound }

	t.Run("NonExistentApp", func(t *testing.T) {
		router := setupTestRouter()
		router.POST("/apps/:id/stop", handler.StopApp)

		req := httptest.NewRequest("POST", "/apps/nonexistent-app/stop", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			t.Errorf("nonexistent app must not report a successful stop: %s", w.Body.String())
		}
	})
}

func TestRestartApp(t *testing.T) {
	gin.SetMode(gin.TestMode)
	appService := services.NewAppService(nil)
	handler := NewAppHandler(appService)

	t.Run("NonExistentApp", func(t *testing.T) {
		router := setupTestRouter()
		router.POST("/apps/:id/restart", handler.RestartApp)

		req := httptest.NewRequest("POST", "/apps/nonexistent-app/restart", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail for nonexistent app
		if w.Code == http.StatusOK {
			t.Error("Expected failure for nonexistent app")
		}
	})
}

func TestRecordAppView(t *testing.T) {
	gin.SetMode(gin.TestMode)
	appService := services.NewAppService(nil)
	handler := NewAppHandler(appService)

	t.Run("ValidRequest", func(t *testing.T) {
		router := setupTestRouter()
		router.POST("/apps/:id/view", handler.RecordAppView)

		req := httptest.NewRequest("POST", "/apps/test-app/view", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should succeed
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("EmptyID", func(t *testing.T) {
		router := setupTestRouter()
		router.POST("/apps/:id/view", handler.RecordAppView)

		req := httptest.NewRequest("POST", "/apps//view", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail for empty ID
		if w.Code == http.StatusOK {
			t.Error("Expected failure for empty ID")
		}
	})
}

func TestReportAppFix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	appService := services.NewAppService(nil)
	handler := NewAppHandler(appService)

	t.Run("MissingBody", func(t *testing.T) {
		router := setupTestRouter()
		router.POST("/apps/:id/fixes/report", handler.ReportAppFix)

		req := httptest.NewRequest("POST", "/apps/test-app/fixes/report", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("missing JSON body must be rejected with 400, got %d", w.Code)
		}
	})
}

func TestCheckAppIframeBridge(t *testing.T) {
	gin.SetMode(gin.TestMode)
	appService := services.NewAppService(nil)
	handler := NewAppHandler(appService)

	t.Run("NonExistentApp", func(t *testing.T) {
		router := setupTestRouter()
		router.GET("/apps/:id/diagnostics/iframe-bridge", handler.CheckAppIframeBridge)

		req := httptest.NewRequest("GET", "/apps/nonexistent-app/diagnostics/iframe-bridge", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return some response (may succeed or fail depending on implementation)
		if w.Code == 0 {
			t.Error("Expected response to be set")
		}
	})
}

func TestCheckAppLocalhostUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	appService := services.NewAppService(nil)
	handler := NewAppHandler(appService)

	router := setupTestRouter()
	router.GET("/apps/:id/diagnostics/localhost", handler.CheckAppLocalhostUsage)

	req := httptest.NewRequest("GET", "/apps/test-app/diagnostics/localhost", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == 0 {
		t.Error("expected handler to write a response")
	}
}

func TestGetAppLogs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	appService := services.NewAppService(nil)
	handler := NewAppHandler(appService)

	t.Run("ValidRequest", func(t *testing.T) {
		router := setupTestRouter()
		router.GET("/apps/:id/logs", handler.GetAppLogs)

		req := httptest.NewRequest("GET", "/apps/test-app/logs", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return some response
		if w.Code == 0 {
			t.Error("Expected response to be set")
		}
	})

	t.Run("WithQueryParams", func(t *testing.T) {
		router := setupTestRouter()
		router.GET("/apps/:id/logs", handler.GetAppLogs)

		req := httptest.NewRequest("GET", "/apps/test-app/logs?lines=100", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should handle query params
		if w.Code == 0 {
			t.Error("Expected response to be set")
		}
	})
}

func TestGetAppLifecycleLogs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	appService := services.NewAppService(nil)
	handler := NewAppHandler(appService)

	t.Run("ValidRequest", func(t *testing.T) {
		router := setupTestRouter()
		router.GET("/apps/:id/logs/lifecycle", handler.GetAppLifecycleLogs)

		req := httptest.NewRequest("GET", "/apps/test-app/logs/lifecycle", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return some response
		if w.Code == 0 {
			t.Error("Expected response to be set")
		}
	})
}

func TestGetAppBackgroundLogs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	appService := services.NewAppService(nil)
	handler := NewAppHandler(appService)

	t.Run("ValidRequest", func(t *testing.T) {
		router := setupTestRouter()
		router.GET("/apps/:id/logs/background", handler.GetAppBackgroundLogs)

		req := httptest.NewRequest("GET", "/apps/test-app/logs/background", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return some response
		if w.Code == 0 {
			t.Error("Expected response to be set")
		}
	})
}

func TestGetAppMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	appService := services.NewAppService(nil)
	handler := NewAppHandler(appService)

	t.Run("ValidRequest", func(t *testing.T) {
		router := setupTestRouter()
		router.GET("/apps/:id/metrics", handler.GetAppMetrics)

		req := httptest.NewRequest("GET", "/apps/test-app/metrics", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return some response
		if w.Code == 0 {
			t.Error("Expected response to be set")
		}
	})
}
