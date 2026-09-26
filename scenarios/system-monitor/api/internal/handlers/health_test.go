package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	handlermocks "github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/handlers/mocks"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
)

func TestHealthResponseReportsLifecycleBuildIdentity(t *testing.T) {
	t.Setenv("VROOLI_BUILD_IDENTITY", "sha256:test-system-monitor")
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ollama.Close()
	t.Setenv("OLLAMA_BASE_URL", ollama.URL)

	handler := NewHealthHandler(
		&config.Config{Server: config.ServerConfig{ServiceName: "system-monitor", Version: "test"}},
		handlermocks.NewMonitorQuerier().WithCurrentMetrics(&models.MetricsResponse{}),
		nil,
	)

	response := handler.buildHealthResponse(t.Context())
	if got := response["build_identity"]; got != "sha256:test-system-monitor" {
		t.Fatalf("build_identity = %v, want sha256:test-system-monitor", got)
	}
}

func TestParseHealthDependencyURL(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "operator configured http", value: "http://nodered:1880"},
		{name: "operator configured https path", value: "https://ollama.example.test/api"},
		{name: "reject unsupported scheme", value: "file:///etc/passwd", wantErr: true},
		{name: "reject missing host", value: "http:///missing-host", wantErr: true},
		{name: "reject credentials", value: "http://user:secret@example.test", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseHealthDependencyURL(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseHealthDependencyURL(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}
