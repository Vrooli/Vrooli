package support

import (
	"time"
)

// ModelSelectResponse mirrors the response from POST /api/v1/ai/select-model.
// Alternatives, SystemMetrics, and ModelInfo vary in shape between Ollama
// backends, so they are kept as raw maps and rendered via MapRows.
type ModelSelectResponse struct {
	RequestID     string                 `json:"requestId"`
	SelectedModel string                 `json:"selectedModel"`
	TaskType      string                 `json:"taskType"`
	FallbackUsed  bool                   `json:"fallbackUsed"`
	Alternatives  []string               `json:"alternatives"`
	SystemMetrics map[string]interface{} `json:"systemMetrics"`
	ModelInfo     map[string]interface{} `json:"modelInfo"`
}

// RouteResponse mirrors the response from POST /api/v1/ai/route-request.
type RouteResponse struct {
	RequestID     string                 `json:"requestId"`
	SelectedModel string                 `json:"selectedModel"`
	Response      string                 `json:"response"`
	FallbackUsed  bool                   `json:"fallbackUsed"`
	Metrics       map[string]interface{} `json:"metrics"`
}

// ModelMetric mirrors a row from the models status endpoint. All fields are
// optional because models without persisted metrics report zero values and the
// Ollama-derived capabilities may be absent.
type ModelMetric struct {
	ModelName         string     `json:"model_name"`
	RequestCount      int        `json:"request_count"`
	SuccessCount      int        `json:"success_count"`
	ErrorCount        int        `json:"error_count"`
	AvgResponseTimeMs float64    `json:"avg_response_time_ms"`
	CurrentLoad       float64    `json:"current_load"`
	MemoryUsageMB     float64    `json:"memory_usage_mb"`
	Healthy           bool       `json:"healthy"`
	LastUsed          *time.Time `json:"last_used,omitempty"`
	Capabilities      []string   `json:"capabilities"`
	Speed             string     `json:"speed"`
	QualityTier       string     `json:"quality_tier"`
	CostPer1KTokens   float64    `json:"cost_per_1k_tokens"`
	RamRequiredGB     float64    `json:"ram_required_gb"`
}

// ModelsStatusResponse mirrors GET /api/v1/ai/models/status.
type ModelsStatusResponse struct {
	Models        []ModelMetric          `json:"models"`
	TotalModels   int                    `json:"totalModels"`
	HealthyModels int                    `json:"healthyModels"`
	SystemHealth  map[string]interface{} `json:"systemHealth"`
}

// ResourceMetricsResponse mirrors GET /api/v1/ai/resources/metrics.
// Current and History entries vary in shape (different backends surface
// different fields), so they stay as raw maps.
type ResourceMetricsResponse struct {
	Current        map[string]interface{}   `json:"current"`
	History        []map[string]interface{} `json:"history"`
	MemoryPressure float64                  `json:"memoryPressure"`
}
