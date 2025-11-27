package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/internal/scenarioport"
)

// ScenarioPortInfo represents port information for a scenario
type ScenarioPortInfo struct {
	Port   int    `json:"port"`
	Status string `json:"status"`
	URL    string `json:"url"`
}

// GetScenarioPort handles GET /api/v1/scenarios/{name}/port
func (h *Handler) GetScenarioPort(w http.ResponseWriter, r *http.Request) {
	scenarioName := chi.URLParam(r, "name")
	if scenarioName == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "name"}))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.ExtendedRequestTimeout)
	defer cancel()

	portInfo, err := h.getScenarioPortInfo(ctx, scenarioName)
	if err != nil {
		h.log.WithError(err).WithField("scenario", scenarioName).Error("Failed to get scenario port")
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "get_scenario_port", "scenario": scenarioName}))
		return
	}

	h.respondSuccess(w, http.StatusOK, portInfo)
}

func (h *Handler) getScenarioPortInfo(ctx context.Context, scenarioName string) (*ScenarioPortInfo, error) {
	resolvedURL, portInfo, err := scenarioport.ResolveURL(ctx, scenarioName, "")
	if err != nil {
		h.log.WithError(err).WithField("scenario", scenarioName).Error("Failed to get scenario port")
		return nil, fmt.Errorf("failed to get port for scenario %s: %w", scenarioName, err)
	}

	portName := ""
	port := 0
	if portInfo != nil {
		portName = portInfo.Name
		port = portInfo.Port
	}

	// Check if the scenario is running by trying to get its status
	statusCmd := exec.CommandContext(ctx, "vrooli", "scenario", "status", scenarioName)
	statusOutput, err := statusCmd.Output()
	status := "unknown"
	if err == nil {
		statusStr := strings.TrimSpace(string(statusOutput))
		if strings.Contains(strings.ToLower(statusStr), "running") {
			status = "running"
		} else if strings.Contains(strings.ToLower(statusStr), "stopped") {
			status = "stopped"
		}
	}

	h.log.WithFields(logrus.Fields{
		"scenario":  scenarioName,
		"port_name": portName,
		"port":      port,
		"status":    status,
	}).Info("Successfully retrieved scenario port info")

	return &ScenarioPortInfo{
		Port:   port,
		Status: status,
		URL:    resolvedURL,
	}, nil
}

// ListScenarios handles GET /api/v1/scenarios
func (h *Handler) ListScenarios(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	scenarios, err := scenarioport.ListScenarios(ctx)
	if err != nil {
		h.log.WithError(err).Error("Failed to list scenarios")
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "list_scenarios"}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"scenarios": scenarios,
	})
}
