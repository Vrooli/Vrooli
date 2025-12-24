package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	exportservices "github.com/vrooli/browser-automation-studio/services/export"
)

func TestReplayConfigHandlers(t *testing.T) {
	repo := NewMockRepository()
	handler := &Handler{repo: repo, log: logrus.New()}

	t.Run("get empty config", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/replay-config", nil)
		rr := httptest.NewRecorder()
		handler.GetReplayConfig(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}

		var payload ReplayConfigResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if len(payload.Config) != 0 {
			t.Fatalf("expected empty config, got %+v", payload.Config)
		}
	})

	t.Run("put and get config", func(t *testing.T) {
		body := `{"config":{"chromeTheme":"aurora","cursorSpeedProfile":"easeInOut"}}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/replay-config", strings.NewReader(body))
		rr := httptest.NewRecorder()
		handler.PutReplayConfig(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/api/v1/replay-config", nil)
		rr = httptest.NewRecorder()
		handler.GetReplayConfig(rr, req)

		var payload ReplayConfigResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if payload.Config["chromeTheme"] != "aurora" {
			t.Fatalf("expected chromeTheme aurora, got %v", payload.Config["chromeTheme"])
		}
	})

	t.Run("delete config", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/replay-config", nil)
		rr := httptest.NewRecorder()
		handler.DeleteReplayConfig(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/api/v1/replay-config", nil)
		rr = httptest.NewRecorder()
		handler.GetReplayConfig(rr, req)

		var payload ReplayConfigResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if len(payload.Config) != 0 {
			t.Fatalf("expected empty config after delete, got %+v", payload.Config)
		}
	})
}

func TestReplayConfigOverridePrecedence(t *testing.T) {
	spec := &exportservices.ReplayMovieSpec{
		Decor: exportservices.ExportDecor{
			ChromeTheme:     "aurora",
			BackgroundTheme: "aurora",
		},
	}
	config := map[string]any{
		"chromeTheme": "midnight",
		"background": map[string]any{
			"type": "theme",
			"id":   "nebula",
		},
	}
	applyExportOverrides(spec, replayConfigToOverrides(config))

	requestOverrides := &executionExportOverrides{
		ThemePreset: &themePresetOverride{
			ChromeTheme:     "solar",
			BackgroundTheme: "dawn",
		},
	}
	applyExportOverrides(spec, requestOverrides)

	if spec.Decor.ChromeTheme != "solar" {
		t.Fatalf("expected chrome theme solar, got %s", spec.Decor.ChromeTheme)
	}
	if spec.Decor.BackgroundTheme != "dawn" {
		t.Fatalf("expected background theme dawn, got %s", spec.Decor.BackgroundTheme)
	}
}

func TestReplayConfigBrowserScale(t *testing.T) {
	spec := &exportservices.ReplayMovieSpec{
		Presentation: exportservices.ExportPresentation{
			Canvas: exportservices.ExportDimensions{
				Width:  1000,
				Height: 800,
			},
		},
	}
	config := map[string]any{
		"browser_scale": 0.4,
	}

	applyReplayConfigToSpec(spec, config)

	frame := spec.Presentation.BrowserFrame
	if frame.Width != 600 || frame.Height != 480 {
		t.Fatalf("expected browser frame 600x480, got %dx%d", frame.Width, frame.Height)
	}
	if frame.X != 200 || frame.Y != 160 {
		t.Fatalf("expected centered browser frame at 200,160 got %d,%d", frame.X, frame.Y)
	}
	if frame.Radius != 24 {
		t.Fatalf("expected default browser frame radius 24, got %d", frame.Radius)
	}
}
