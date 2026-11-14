package control

import (
	"encoding/json"
	"net/http"
	"strings"

	"device-control/strategy"
	"github.com/gorilla/mux"
)

type androidConformanceRequest struct {
	DeviceID   string `json:"device_id"`
	Actor      string `json:"actor"`
	LeaseToken string `json:"lease_token,omitempty"`
}

func (h *handler) androidConformancePlan(w http.ResponseWriter, _ *http.Request) {
	write(w, http.StatusOK, h.service.AndroidCapabilitySelfTestPlan())
}

func (h *handler) runAndroidConformance(w http.ResponseWriter, r *http.Request) {
	var in androidConformanceRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := h.service.RunAndroidCapabilitySelfTest(r.Context(), in.DeviceID, in.Actor, in.LeaseToken)
	if err != nil {
		if strings.HasPrefix(err.Error(), "unknown device ") {
			writeError(w, http.StatusBadRequest, "unknown_device", err.Error())
			return
		}
		writeError(w, http.StatusConflict, leaseErrorCode(err), err.Error())
		return
	}
	write(w, http.StatusOK, result)
}

func (h *handler) listStrategies(w http.ResponseWriter, r *http.Request) {
	decls := h.service.Strategies(r.Context())
	out := make([]map[string]any, 0, len(decls))
	for _, d := range decls {
		// Tiers are derived from the declared capabilities. Some strategies
		// intentionally leave Declaration.Tiers empty so the capability model
		// remains the single source of truth.
		out = append(out, map[string]any{"id": d.StrategyID, "description": d.Description, "status": d.Status, "reason": d.Reason, "supported_host_os": append([]string{}, d.SupportedHostOS...), "capabilities": d.Capabilities, "tiers": strategy.Tiers(d), "executable_step_kinds": append([]string{}, strategy.StepKinds(d)...), "next_actions": append([]string{}, d.NextActions...), "promotable": d.Promotable, "evidence_class": d.EvidenceClass, "minimum_useful_fps": d.MinimumUsefulFPS})
	}
	write(w, http.StatusOK, map[string]any{"strategies": out})
}

func (h *handler) verifyStrategy(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	report, err := h.service.Verify(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	write(w, http.StatusOK, report)
}
