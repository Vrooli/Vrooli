package handlers

import (
	"encoding/json"
	"net/http"
	"sort"

	"agent-manager/internal/domain"
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/orchestration"

	"github.com/gorilla/mux"
)

type CanaryHandler struct {
	store invocationreadmodel.CanaryStore
}

func NewCanaryHandler(stores ...invocationreadmodel.Store) *CanaryHandler {
	h := &CanaryHandler{}
	if len(stores) > 0 {
		h.store, _ = stores[0].(invocationreadmodel.CanaryStore)
	}
	return h
}

func (h *CanaryHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/model-policy/canary/arm", h.Arm).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/model-policy/canary/compare", h.Compare).Methods(http.MethodPost)
}

// armRequest is intentionally a snapshot-shaped request. The route computes
// an arm from the immutable candidate evidence supplied at run creation; it
// does not read the mutable resource catalog.
type armRequest struct {
	Seed      string                    `json:"seed"`
	Candidate domain.ExecutionCandidate `json:"candidate"`
}

type compareRequest struct {
	Role               string                         `json:"role"`
	IncumbentModel     string                         `json:"incumbent_model"`
	ChallengerModel    string                         `json:"challenger_model"`
	Incumbent          orchestration.CanaryArmMetrics `json:"incumbent"`
	Challenger         orchestration.CanaryArmMetrics `json:"challenger"`
	IncludedRunClasses []string                       `json:"included_run_classes,omitempty"`
	Source             string                         `json:"source,omitempty"`
}

// Arm handles the deterministic arm decision as a small transport seam for
// CLI/UI callers and integration tests.
func (h *CanaryHandler) Arm(w http.ResponseWriter, r *http.Request) {
	var request armRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"arm": orchestration.SelectCanaryArm(request.Seed, request.Candidate)})
}

func (h *CanaryHandler) Compare(w http.ResponseWriter, r *http.Request) {
	var request compareRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	for _, class := range request.IncludedRunClasses {
		if class == "imported" || class == "interactive" {
			writeJSONError(w, http.StatusBadRequest, "canary comparison excludes imported and interactive runs")
			return
		}
	}
	if request.Source == "durable" && h.store != nil {
		rows, err := h.store.CanaryRuns(r.Context(), invocationreadmodel.Filter{})
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		request.Incumbent, request.Challenger = durableCanaryMetrics(rows, request.Role)
	}
	comparison := orchestration.CompareCanary(request.Role, request.IncumbentModel, request.ChallengerModel, request.Incumbent, request.Challenger)
	writeJSON(w, http.StatusOK, map[string]any{"comparison": comparison, "source": map[bool]string{true: "durable_terminal_projection", false: "supplied_metrics"}[request.Source == "durable"]})
}

func durableCanaryMetrics(rows []invocationreadmodel.CanaryRun, role string) (orchestration.CanaryArmMetrics, orchestration.CanaryArmMetrics) {
	values := map[string][]invocationreadmodel.CanaryRun{"incumbent": {}, "challenger": {}}
	for _, row := range rows {
		if (role == "" || row.Role == role) && row.Arm != "" {
			values[row.Arm] = append(values[row.Arm], row)
		}
	}
	return metricsForCanaryRows(values["incumbent"]), metricsForCanaryRows(values["challenger"])
}

func metricsForCanaryRows(rows []invocationreadmodel.CanaryRun) orchestration.CanaryArmMetrics {
	if len(rows) == 0 {
		return orchestration.CanaryArmMetrics{}
	}
	durations := make([]float64, 0, len(rows))
	var successes int64
	var cost float64
	for _, row := range rows {
		durations = append(durations, row.DurationMS)
		if row.Status == "complete" {
			successes++
		}
		cost += row.CostUSD
	}
	sort.Float64s(durations)
	median := durations[len(durations)/2]
	if len(durations)%2 == 0 {
		median = (durations[len(durations)/2-1] + durations[len(durations)/2]) / 2
	}
	return orchestration.CanaryArmMetrics{Count: int64(len(rows)), SuccessRate: float64(successes) / float64(len(rows)), MedianMS: median, CostPerRun: cost / float64(len(rows))}
}
