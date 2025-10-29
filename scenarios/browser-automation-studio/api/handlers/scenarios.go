package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
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
		http.Error(w, "Scenario name is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	portInfo, err := h.getScenarioPortInfo(ctx, scenarioName)
	if err != nil {
		h.log.WithError(err).WithField("scenario", scenarioName).Error("Failed to get scenario port")
		http.Error(w, "Failed to get scenario port information", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(portInfo)
}

func (h *Handler) getScenarioPortInfo(ctx context.Context, scenarioName string) (*ScenarioPortInfo, error) {
	// For most scenarios, we'll try UI_PORT first, then API_PORT as fallback
	portNames := []string{"UI_PORT", "API_PORT"}

	var port int
	var portName string
	var err error

	// Try each port name until we find one that works
	for _, name := range portNames {
		cmd := exec.CommandContext(ctx, "vrooli", "scenario", "port", scenarioName, name)
		output, portErr := cmd.Output()
		if portErr == nil {
			portStr := strings.TrimSpace(string(output))
			port, err = strconv.Atoi(portStr)
			if err == nil {
				portName = name
				break
			}
		}
	}

	if err != nil || port == 0 {
		h.log.WithError(err).WithField("scenario", scenarioName).Error("Failed to get any port for scenario")
		return nil, fmt.Errorf("failed to get port for scenario %s: no valid ports found", scenarioName)
	}

	// Construct URL
	host := "localhost"
	url := fmt.Sprintf("http://%s:%d", host, port)

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
		URL:    url,
	}, nil
}
