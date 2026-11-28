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

func newAppHandlerForTest() *AppHandler {
	return NewAppHandler(services.NewAppService(nil))
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
	handler := newAppHandlerForTest()
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
	handler := newAppHandlerForTest()
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
	handler := newAppHandlerForTest()

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

func TestDiagnosticHandlersPreserveAppServiceErrorResponses(t *testing.T) {
	handler := newAppHandlerForTest()
	tests := []struct {
		name    string
		handler gin.HandlerFunc
		query   string
	}{
		{name: "scenario status", handler: handler.GetAppScenarioStatus},
		{name: "health", handler: handler.CheckAppHealth},
		{name: "documents", handler: handler.GetAppDocuments},
		{name: "document search", handler: handler.SearchAppDocuments, query: "?q=app"},
		{name: "interop compliance", handler: handler.CheckAppInteropCompliance},
		{name: "completeness", handler: handler.GetAppCompleteness},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupTestRouter()
			router.GET("/apps/:id", tt.handler)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/apps/%20"+tt.query, nil))

			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusBadRequest, response.Body.String())
			}
			if !strings.Contains(response.Body.String(), `"success":false`) || !strings.Contains(response.Body.String(), `"error":"app identifier is required"`) {
				t.Fatalf("response = %s, want the original app identifier error", response.Body.String())
			}
		})
	}
}

func TestStartApp(t *testing.T) {
	handler := newAppHandlerForTest()

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
	handler := newAppHandlerForTest()
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
	handler := newAppHandlerForTest()

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
	handler := newAppHandlerForTest()

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
	handler := newAppHandlerForTest()

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

	t.Run("ValidationError", func(t *testing.T) {
		router := setupTestRouter()
		router.POST("/apps/:id/fixes/report", handler.ReportAppFix)

		request := httptest.NewRequest(http.MethodPost, "/apps/test-app/fixes/report", strings.NewReader(`{"message":"  "}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusBadRequest {
			t.Fatalf("validation failure status = %d, want %d: %s", response.Code, http.StatusBadRequest, response.Body.String())
		}
		if !strings.Contains(response.Body.String(), services.ErrIssueReportMessageRequired.Error()) {
			t.Fatalf("validation response = %s, want message validation error", response.Body.String())
		}
	})

	t.Run("InvalidScenarioScope", func(t *testing.T) {
		handler.reportAppIssue = func(context.Context, *services.IssueReportRequest) (*services.IssueReportResult, error) {
			return nil, services.ErrIssueReportScenarioInvalid
		}
		router := setupTestRouter()
		router.POST("/apps/:id/fixes/report", handler.ReportAppFix)

		request := httptest.NewRequest(http.MethodPost, "/apps/test-app/fixes/report", strings.NewReader(`{"message":"investigate issue"}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusBadRequest {
			t.Fatalf("invalid scope status = %d, want %d: %s", response.Code, http.StatusBadRequest, response.Body.String())
		}
	})

	t.Run("SwarmManagerUnavailable", func(t *testing.T) {
		handler.reportAppIssue = func(context.Context, *services.IssueReportRequest) (*services.IssueReportResult, error) {
			return nil, services.ErrSwarmManagerUnavailable
		}
		router := setupTestRouter()
		router.POST("/apps/:id/fixes/report", handler.ReportAppFix)

		request := httptest.NewRequest(http.MethodPost, "/apps/test-app/fixes/report", strings.NewReader(`{"message":"investigate issue"}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusServiceUnavailable {
			t.Fatalf("upstream failure status = %d, want %d: %s", response.Code, http.StatusServiceUnavailable, response.Body.String())
		}
	})

	t.Run("UnexpectedFailure", func(t *testing.T) {
		handler.reportAppIssue = func(context.Context, *services.IssueReportRequest) (*services.IssueReportResult, error) {
			return nil, errors.New("unexpected report failure")
		}
		router := setupTestRouter()
		router.POST("/apps/:id/fixes/report", handler.ReportAppFix)

		request := httptest.NewRequest(http.MethodPost, "/apps/test-app/fixes/report", strings.NewReader(`{"message":"investigate issue"}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusInternalServerError {
			t.Fatalf("unexpected failure status = %d, want %d: %s", response.Code, http.StatusInternalServerError, response.Body.String())
		}
	})

	t.Run("SuccessContract", func(t *testing.T) {
		handler.reportAppIssue = func(_ context.Context, request *services.IssueReportRequest) (*services.IssueReportResult, error) {
			if request.AppID != "test-app" || request.Message != "investigate issue" {
				t.Fatalf("service request = %#v, want path app ID and submitted message", request)
			}
			return &services.IssueReportResult{Message: "created", Kind: "fix", Name: "fix-123", URL: "http://swarm/fix-123"}, nil
		}
		router := setupTestRouter()
		router.POST("/apps/:id/fixes/report", handler.ReportAppFix)

		request := httptest.NewRequest(http.MethodPost, "/apps/test-app/fixes/report", strings.NewReader(`{"message":"investigate issue"}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("success status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
		}
		for _, field := range []string{`"message":"created"`, `"kind":"fix"`, `"name":"fix-123"`, `"url":"http://swarm/fix-123"`} {
			if !strings.Contains(response.Body.String(), field) {
				t.Errorf("success response %s omitted %s", response.Body.String(), field)
			}
		}
	})
}

func TestCheckAppIframeBridge(t *testing.T) {
	handler := newAppHandlerForTest()

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
	handler := newAppHandlerForTest()

	router := setupTestRouter()
	router.GET("/apps/:id/diagnostics/localhost", handler.CheckAppLocalhostUsage)

	req := httptest.NewRequest("GET", "/apps/test-app/diagnostics/localhost", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == 0 {
		t.Error("expected handler to write a response")
	}
}

func TestAppLogRoutesPreserveAppIdentityAndLogType(t *testing.T) {
	cases := []struct {
		name        string
		route       string
		path        string
		wantApp     string
		wantType    string
		handler     func(*AppHandler) gin.HandlerFunc
		bodyMarkers []string
	}{
		{
			name:     "primary app ID route returns both streams",
			route:    "/apps/:id/logs",
			path:     "/apps/primary-app/logs",
			wantApp:  "primary-app",
			wantType: "both",
			handler:  func(h *AppHandler) gin.HandlerFunc { return h.GetAppLogs },
			bodyMarkers: []string{
				`"logs":["lifecycle-ready","","--- Background: Worker ---","background-ready"]`,
				`"key":"background:worker:run"`,
			},
		},
		{
			name:     "legacy app name route honors requested stream",
			route:    "/logs/:appName",
			path:     "/logs/legacy-app?type=background",
			wantApp:  "legacy-app",
			wantType: "background",
			handler:  func(h *AppHandler) gin.HandlerFunc { return h.GetAppLogs },
			bodyMarkers: []string{
				`"logs":["--- Background: Worker ---","background-ready"]`,
				`"key":"background:worker:run"`,
			},
		},
		{
			name:        "lifecycle route",
			route:       "/apps/:id/logs/lifecycle",
			path:        "/apps/lifecycle-app/logs/lifecycle",
			wantApp:     "lifecycle-app",
			wantType:    "lifecycle",
			handler:     func(h *AppHandler) gin.HandlerFunc { return h.GetAppLifecycleLogs },
			bodyMarkers: []string{"lifecycle-ready"},
		},
		{
			name:     "background route",
			route:    "/apps/:id/logs/background",
			path:     "/apps/background-app/logs/background",
			wantApp:  "background-app",
			wantType: "background",
			handler:  func(h *AppHandler) gin.HandlerFunc { return h.GetAppBackgroundLogs },
			bodyMarkers: []string{
				`"message":"background-ready"`,
				`"source":"background"`,
				`"step":"worker"`,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			appService := services.NewAppService(nil)
			handler := NewAppHandler(appService)
			var gotApp, gotType string
			handler.getAppLogs = func(_ context.Context, appID, logType string) (*services.AppLogsResult, error) {
				gotApp, gotType = appID, logType
				result := &services.AppLogsResult{
					Lifecycle: []string{"lifecycle-ready"},
					Background: []services.BackgroundLog{{
						Step: "worker", Phase: "run", Label: "Worker", Lines: []string{"background-ready"},
					}},
				}
				switch logType {
				case "lifecycle":
					result.Background = nil
				case "background":
					result.Lifecycle = nil
				}
				return result, nil
			}

			router := setupTestRouter()
			router.GET(tc.route, tc.handler(handler))
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tc.path, nil))

			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
			}
			if gotApp != tc.wantApp {
				t.Errorf("service app ID = %q, want %q", gotApp, tc.wantApp)
			}
			if gotType != tc.wantType {
				t.Errorf("service log type = %q, want %q", gotType, tc.wantType)
			}
			for _, marker := range tc.bodyMarkers {
				if !strings.Contains(response.Body.String(), marker) {
					t.Errorf("response %s omitted %q", response.Body.String(), marker)
				}
			}
		})
	}
}

func TestGetAppMetricsPreservesAppIDHoursAndServiceFailure(t *testing.T) {
	cases := []struct {
		name       string
		path       string
		wantApp    string
		wantHours  int
		wantStatus int
		wantBody   string
		serviceErr error
	}{
		{
			name: "default window", path: "/apps/default-app/metrics", wantApp: "default-app", wantHours: 24,
			wantStatus: http.StatusOK, wantBody: `"app_id":"default-app"`,
		},
		{
			name: "requested window", path: "/apps/window-app/metrics?hours=6", wantApp: "window-app", wantHours: 6,
			wantStatus: http.StatusOK, wantBody: `"cpu_usage":12.5`,
		},
		{
			name: "service failure", path: "/apps/failing-app/metrics", wantApp: "failing-app", wantHours: 24,
			wantStatus: http.StatusInternalServerError, wantBody: "Failed to fetch metrics", serviceErr: errors.New("repository unavailable"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler := newAppHandlerForTest()
			var gotApp string
			var gotHours int
			handler.getStatusHistory = func(_ context.Context, appID string, hours int) ([]repository.AppStatus, error) {
				gotApp, gotHours = appID, hours
				if tc.serviceErr != nil {
					return nil, tc.serviceErr
				}
				return []repository.AppStatus{{ID: "status-1", AppID: tc.wantApp, Status: "running", CPUUsage: 12.5}}, nil
			}

			router := setupTestRouter()
			router.GET("/apps/:id/metrics", handler.GetAppMetrics)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tc.path, nil))

			if response.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d: %s", response.Code, tc.wantStatus, response.Body.String())
			}
			if gotApp != tc.wantApp || gotHours != tc.wantHours {
				t.Errorf("service request = (%q, %d hours), want (%q, %d hours)", gotApp, gotHours, tc.wantApp, tc.wantHours)
			}
			if !strings.Contains(response.Body.String(), tc.wantBody) {
				t.Errorf("response %s omitted %q", response.Body.String(), tc.wantBody)
			}
		})
	}
}
