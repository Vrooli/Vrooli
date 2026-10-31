package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/encoding/protojson"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// InspectorHandler exposes scenario/resource inspection data by delegating to the Vrooli CLI.
type InspectorHandler struct{}

// NewInspectorHandler creates a new InspectorHandler.
func NewInspectorHandler() *InspectorHandler {
	return &InspectorHandler{}
}

// GetScenarioSummary returns the orchestrator summary section from the CLI output.
//
// `system_health`/`system_warnings` are not part of the `scenario status`
// contract (cliv1.ScenarioStatusListResponse carries summary + scenarios +
// discovery_failures). They were always empty in the previous implementation
// and are preserved as empty placeholders here to keep the response shape
// stable for the UI.
func (h *InspectorHandler) GetScenarioSummary(c *gin.Context) {
	resp, err := fetchScenarioStatus(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch scenario status",
			"details": err.Error(),
		})
		return
	}

	summary := resp.GetSummary()
	c.JSON(http.StatusOK, gin.H{
		"summary": gin.H{
			"total_scenarios": summary.GetTotalScenarios(),
			"running":         summary.GetRunning(),
			"stopped":         summary.GetStopped(),
		},
		"system_health":   "",
		"system_warnings": []any{},
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

	scenarios := make([]map[string]any, 0, len(resp.GetScenarios()))
	for _, item := range resp.GetScenarios() {
		scenarios = append(scenarios, scenarioItemToMap(item))
	}
	c.JSON(http.StatusOK, scenarios)
}

func fetchScenarioStatus(parentCtx context.Context) (*cliv1.ScenarioStatusListResponse, error) {
	ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
	defer cancel()
	return cliClient.ScenarioStatuses(ctx)
}

// scenarioItemToMap projects a typed scenario-status row back to a generic map
// (snake_case keys via protojson) for the raw passthrough the UI consumes.
func scenarioItemToMap(item *cliv1.ScenarioStatusItem) map[string]any {
	out, err := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: true}.Marshal(item)
	if err != nil {
		return map[string]any{}
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		return map[string]any{}
	}
	return m
}
