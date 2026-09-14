package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"

	"agent-manager/internal/invocationreadmodel"

	"github.com/gorilla/mux"
)

func TestEfficiencyFilterDefaultsToBounded24HourProfileReport(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/stats/efficiency", nil)
	filter, window, response, compare, err := efficiencyFilter(req)
	if err != nil {
		t.Fatalf("efficiencyFilter() error = %v", err)
	}
	if response.GroupBy != "profile" || response.Limit != 20 || compare {
		t.Fatalf("response filter = %+v, want profile/20", response)
	}
	if filter.TagPrefix != "" || filter.From == nil || filter.To == nil || window.From == "" || window.To == "" {
		t.Fatalf("unexpected default filter/window: filter=%+v window=%+v", filter, window)
	}
}

func TestEfficiencyFilterRejectsUnboundedOrUnsupportedInputs(t *testing.T) {
	tests := []string{
		"/api/v1/stats/efficiency?preset=2h",
		"/api/v1/stats/efficiency?group_by=tag",
		"/api/v1/stats/efficiency?limit=101",
		"/api/v1/stats/efficiency?start=2026-09-14T01:00:00Z&end=2026-09-14T00:00:00Z",
		"/api/v1/stats/efficiency?compare=current",
	}
	for _, target := range tests {
		t.Run(target, func(t *testing.T) {
			req := httptest.NewRequest("GET", target, nil)
			if _, _, _, _, err := efficiencyFilter(req); err == nil {
				t.Fatal("efficiencyFilter() returned nil error")
			}
		})
	}
}

func TestEfficiencyFilterAcceptsPreviousComparison(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/stats/efficiency?preset=6h&compare=previous", nil)
	filter, window, _, compare, err := efficiencyFilter(req)
	if err != nil || !compare || filter.From == nil || filter.To == nil || window.From == "" {
		t.Fatalf("comparison filter = %+v %+v compare=%v err=%v", filter, window, compare, err)
	}
}

func TestEfficiencyUnavailableRouteIsExplicitlyReadOnly(t *testing.T) {
	router := mux.NewRouter()
	NewEfficiencyHandler(nil).RegisterRoutes(router)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/api/v1/stats/efficiency", nil))
	if response.Code != 200 || !strings.Contains(response.Body.String(), `"status":"unavailable"`) || !strings.Contains(response.Body.String(), `"read_only":true`) {
		t.Fatalf("response code=%d body=%s", response.Code, response.Body.String())
	}
}

func TestEfficiencyDTOsUseStableBoundedShapes(t *testing.T) {
	if got := efficiencyDurationDTO(invocationreadmodel.RunDurationStatistics{AverageDurationMS: 12, Count: 1}); got.AverageMS != 12 || got.Count != 1 {
		t.Fatalf("duration DTO = %+v", got)
	}
}
