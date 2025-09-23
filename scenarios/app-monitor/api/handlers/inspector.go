package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"os/exec"
	"time"

	"github.com/gin-gonic/gin"
)

// InspectorHandler exposes scenario/resource inspection data by delegating to the Vrooli CLI.
type InspectorHandler struct{}

// NewInspectorHandler creates a new InspectorHandler.
func NewInspectorHandler() *InspectorHandler {
	return &InspectorHandler{}
}

// scenarioStatusResponse mirrors the output of `vrooli scenario status --json`.
type scenarioStatusResponse struct {
	Success bool `json:"success"`
	Summary struct {
		TotalScenarios int `json:"total_scenarios"`
		Running        int `json:"running"`
		Stopped        int `json:"stopped"`
	} `json:"summary"`
	Scenarios      []map[string]any `json:"scenarios"`
	SystemHealth   string           `json:"system_health"`
	SystemWarnings []map[string]any `json:"system_warnings"`
}

// GetScenarioSummary returns the orchestrator summary section from the CLI output.
func (h *InspectorHandler) GetScenarioSummary(c *gin.Context) {
	resp, err := fetchScenarioStatus(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch scenario status",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"summary":         resp.Summary,
		"system_health":   resp.SystemHealth,
		"system_warnings": resp.SystemWarnings,
	})
}

// GetScenarios returns the full scenario list reported by the orchestrator.
func (h *InspectorHandler) GetScenarios(c *gin.Context) {
	resp, err := fetchScenarioStatus(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch scenarios",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp.Scenarios)
}

func fetchScenarioStatus(parentCtx context.Context) (*scenarioStatusResponse, error) {
	ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "status", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var resp scenarioStatusResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
