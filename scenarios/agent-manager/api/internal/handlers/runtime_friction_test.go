package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agent-manager/internal/orchestration"
	"agent-manager/internal/toolfriction"
)

type runtimeFrictionServiceStub struct {
	report toolfriction.RuntimeReport
}

func (s *runtimeFrictionServiceStub) RuntimeFrictionReport(_ context.Context, _, _ time.Time) (toolfriction.RuntimeReport, error) {
	return s.report, nil
}

func TestRuntimeFrictionValidatesAndUsesBoundedWindow(t *testing.T) {
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	service := &runtimeFrictionServiceStub{
		report: toolfriction.AnalyzeRuntime(nil, toolfriction.RuntimeWindow{Since: start, Until: end}),
	}
	handler := New(orchestration.HandlerServices{RuntimeFrictionService: service})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/runtime/friction?since="+start.Format(time.RFC3339)+"&until="+end.Format(time.RFC3339), nil)
	response := httptest.NewRecorder()
	handler.RuntimeFriction(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var report toolfriction.RuntimeReport
	if err := json.Unmarshal(response.Body.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if !report.Window.Since.Equal(start) || !report.Window.Until.Equal(end) || report.Coverage.State != "unavailable" {
		t.Fatalf("report = %+v", report)
	}
}

func TestRuntimeFrictionRejectsWindowOverThirtyDays(t *testing.T) {
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(31 * 24 * time.Hour)
	handler := New(orchestration.EmptyHandlerServices())
	request := httptest.NewRequest(http.MethodGet, "/api/v1/runtime/friction?since="+start.Format(time.RFC3339)+"&until="+end.Format(time.RFC3339), nil)
	response := httptest.NewRecorder()
	handler.RuntimeFriction(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}
