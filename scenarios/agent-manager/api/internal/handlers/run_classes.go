package handlers

import (
	"net/http"
	"strings"
	"time"

	"agent-manager/internal/invocationreadmodel"

	"github.com/gorilla/mux"
)

type RunClassHandler struct{ store invocationreadmodel.Store }

func NewRunClassHandler(store invocationreadmodel.Store) *RunClassHandler {
	return &RunClassHandler{store: store}
}

type RunClassRow struct {
	Class        string  `json:"class"`
	RunCount     int64   `json:"run_count"`
	SuccessCount int64   `json:"success_count"`
	FailedCount  int64   `json:"failed_count"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	AverageMS    float64 `json:"average_duration_ms"`
}

type RunClassResponse struct {
	Classes             []RunClassRow `json:"classes"`
	ExecutedDenominator int64         `json:"executed_denominator"`
	MissingModelRuns    int64         `json:"missing_model_runs"`
	MissingModelRate    float64       `json:"missing_model_rate"`
	ExcludedClasses     []string      `json:"excluded_classes"`
}

func (h *RunClassHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/stats/run-classes", h.Get).Methods(http.MethodGet)
}

func (h *RunClassHandler) Get(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeJSON(w, http.StatusOK, RunClassResponse{Classes: []RunClassRow{}, ExcludedClasses: []string{"imported", "interactive"}})
		return
	}
	allRows, err := h.store.RunBreakdown(r.Context(), invocationreadmodel.Filter{}, "workload_kind", 100)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	classes := make([]RunClassRow, 0, len(allRows))
	var denominator, missing int64
	for _, row := range allRows {
		class := strings.TrimSpace(row.Value)
		if class == "" {
			class = "unclassified"
		}
		classes = append(classes, RunClassRow{Class: class, RunCount: row.RunCount, SuccessCount: row.SuccessCount, FailedCount: row.FailedCount, TotalCostUSD: row.TotalCostUSD, AverageMS: row.AvgDurationMS})
		if class != "imported" && class != "interactive" {
			denominator += row.RunCount
		}
	}
	filter := runClassFilter(r)
	modelRows, err := h.store.RunBreakdown(r.Context(), filter, "model", 1000)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for _, row := range modelRows {
		if strings.TrimSpace(row.Value) == "" || strings.EqualFold(strings.TrimSpace(row.Value), "unknown") {
			missing += row.RunCount
		}
	}
	rate := float64(0)
	if denominator > 0 {
		rate = float64(missing) / float64(denominator)
	}
	writeJSON(w, http.StatusOK, RunClassResponse{Classes: classes, ExecutedDenominator: denominator, MissingModelRuns: missing, MissingModelRate: rate, ExcludedClasses: []string{"imported", "interactive"}})
}

func runClassFilter(r *http.Request) invocationreadmodel.Filter {
	now := time.Now().UTC()
	filter := invocationreadmodel.Filter{ExcludedWorkloadKinds: []string{"imported", "interactive"}}
	if value := r.URL.Query().Get("preset"); value != "" {
		hours := map[string]float64{"6h": 6, "12h": 12, "24h": 24, "7d": 24 * 7, "30d": 24 * 30}[value]
		if hours > 0 {
			from := now.Add(-time.Duration(hours * float64(time.Hour)))
			filter.From, filter.To = &from, &now
		}
	}
	for name, target := range map[string]**time.Time{"start": &filter.From, "end": &filter.To} {
		if value := r.URL.Query().Get(name); value != "" {
			if parsed, err := time.Parse(time.RFC3339, value); err == nil {
				*target = &parsed
			}
		}
	}
	return filter
}
