package handlers

// DOC: docs/reference/api-endpoints.md#health

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	repocontract "github.com/vrooli/repo-contract-go"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	healthpb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/health"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/healthutil"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/httputil"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	config      *config.Config
	monitorSvc  MonitorQuerier
	settingsMgr SettingsProvider
	startTime   time.Time
}

type selfMetricsProvider interface {
	SelfMetrics() map[string]interface{}
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(cfg *config.Config, monitorSvc MonitorQuerier, settingsMgr SettingsProvider) *HealthHandler {
	return &HealthHandler{
		config:      cfg,
		monitorSvc:  monitorSvc,
		settingsMgr: settingsMgr,
		startTime:   time.Now(),
	}
}

// Handle processes health check requests
func (h *HealthHandler) Handle(w http.ResponseWriter, r *http.Request) {
	httputil.JSON(w, h.buildHealthResponse(r.Context())) //nolint:errcheck
}

// Health handles the typed Connect-RPC health contract while preserving the
// same source data as the REST health probes.
func (h *HealthHandler) Health(ctx context.Context, _ *connect.Request[healthpb.HealthRequest]) (*connect.Response[healthpb.HealthResponse], error) {
	return connect.NewResponse(h.healthResponseToProto(h.buildHealthResponse(ctx))), nil
}

func (h *HealthHandler) buildHealthResponse(ctx context.Context) map[string]interface{} {
	overallStatus := "healthy"

	// Schema-compliant health response
	healthResponse := map[string]interface{}{
		"status":    overallStatus,
		"service":   h.config.Server.ServiceName,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"readiness": true, // Service is ready to accept requests
		"version":   h.config.Server.Version,
		"processor_active": func() bool {
			if h.settingsMgr == nil {
				return true
			}
			return h.settingsMgr.IsActive()
		}(),
		"maintenance_state": func() string {
			if h.settingsMgr == nil {
				return "unknown"
			}
			return h.settingsMgr.GetMaintenanceState()
		}(),
		"dependencies": map[string]interface{}{},
		"metrics": map[string]interface{}{
			"uptime_seconds": time.Since(h.startTime).Seconds(),
		},
	}

	dependencies := healthResponse["dependencies"].(map[string]interface{})

	// 1. Check system metrics collection capability (core functionality)
	metricsHealth := h.checkMetricsCollection(ctx)
	dependencies["metrics_collection"] = metricsHealth
	if metricsHealth["connected"] == false {
		overallStatus = "degraded"
	}

	// 2. Check investigation system (file I/O for investigations and results)
	investigationHealth := h.checkInvestigationSystem()
	dependencies["investigation_system"] = investigationHealth
	if investigationHealth["connected"] == false {
		if overallStatus == "healthy" {
			overallStatus = "degraded"
		}
	}

	// 3. Check external services (if configured)
	if h.config.Resources.NodeRedURL != "" {
		nodeRedHealth := h.checkNodeRed()
		dependencies["node_red"] = nodeRedHealth
		if nodeRedHealth["connected"] == false {
			// Node-RED is optional, so only degrade if we were healthy
			if overallStatus == "healthy" {
				overallStatus = "degraded"
			}
		}
	}

	ollamaHealth := h.checkOllama()
	dependencies["ollama"] = ollamaHealth
	if ollamaHealth["connected"] == false {
		if overallStatus == "healthy" {
			overallStatus = "degraded"
		}
	}

	// 5. Check alert systems (if enabled)
	if h.config.Alerts.EnableWebhooks && h.config.Alerts.WebhookURL != "" {
		webhookHealth := h.checkWebhookEndpoint()
		dependencies["webhooks"] = webhookHealth
		if webhookHealth["connected"] == false {
			if overallStatus == "healthy" {
				overallStatus = "degraded"
			}
		}
	}

	// Update overall status and metrics
	healthResponse["status"] = overallStatus

	// Add system resource metrics if available
	if metrics, err := h.monitorSvc.GetCurrentMetrics(ctx); err == nil && metrics != nil {
		systemMetrics := healthResponse["metrics"].(map[string]interface{})
		systemMetrics["cpu_usage_percent"] = metrics.CPUUsage
		systemMetrics["memory_usage_percent"] = metrics.MemoryUsage
		if h.settingsMgr != nil {
			systemMetrics["active_monitoring"] = h.settingsMgr.IsActive()
		} else {
			systemMetrics["active_monitoring"] = true
		}
	}
	if provider, ok := h.monitorSvc.(selfMetricsProvider); ok {
		systemMetrics := healthResponse["metrics"].(map[string]interface{})
		systemMetrics["self"] = provider.SelfMetrics()
	}

	return healthResponse
}

func (h *HealthHandler) healthResponseToProto(response map[string]interface{}) *healthpb.HealthResponse {
	return &healthpb.HealthResponse{
		Status:           healthStatusToProto(stringValue(response["status"])),
		Service:          stringValue(response["service"]),
		Timestamp:        stringValue(response["timestamp"]),
		Readiness:        boolValue(response["readiness"]),
		Version:          stringValue(response["version"]),
		ProcessorActive:  boolValue(response["processor_active"]),
		MaintenanceState: stringValue(response["maintenance_state"]),
		Dependencies:     mapToProtoJSONValues(mapValue(response["dependencies"])),
		Metrics:          mapToProtoJSONValues(mapValue(response["metrics"])),
	}
}

func healthStatusToProto(status string) commonv1.HealthStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "healthy":
		return commonv1.HealthStatus_HEALTH_STATUS_HEALTHY
	case "degraded":
		return commonv1.HealthStatus_HEALTH_STATUS_DEGRADED
	case "unhealthy":
		return commonv1.HealthStatus_HEALTH_STATUS_UNHEALTHY
	default:
		return commonv1.HealthStatus_HEALTH_STATUS_UNSPECIFIED
	}
}

func mapToProtoJSONValues(values map[string]interface{}) map[string]*commonv1.JsonValue {
	out := make(map[string]*commonv1.JsonValue, len(values))
	for key, value := range values {
		out[key] = interfaceToProtoJSONValue(value)
	}
	return out
}

func interfaceToProtoJSONValue(value interface{}) *commonv1.JsonValue {
	switch typed := value.(type) {
	case nil:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE}}
	case bool:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_BoolValue{BoolValue: typed}}
	case string:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: typed}}
	case int:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(typed)}}
	case int64:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: typed}}
	case float64:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_DoubleValue{DoubleValue: typed}}
	case map[string]interface{}:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_ObjectValue{ObjectValue: &commonv1.JsonObject{Fields: mapToProtoJSONValues(typed)}}}
	case []interface{}:
		values := make([]*commonv1.JsonValue, 0, len(typed))
		for _, item := range typed {
			values = append(values, interfaceToProtoJSONValue(item))
		}
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_ListValue{ListValue: &commonv1.JsonList{Values: values}}}
	default:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: fmt.Sprint(typed)}}
	}
}

func stringValue(value interface{}) string {
	if typed, ok := value.(string); ok {
		return typed
	}
	return ""
}

func boolValue(value interface{}) bool {
	if typed, ok := value.(bool); ok {
		return typed
	}
	return false
}

func mapValue(value interface{}) map[string]interface{} {
	if typed, ok := value.(map[string]interface{}); ok {
		return typed
	}
	return nil
}

// checkMetricsCollection tests the core system monitoring capability
func (h *HealthHandler) checkMetricsCollection(ctx context.Context) map[string]interface{} {
	result := healthutil.NewResult()

	// Test if we can collect system metrics
	metrics, err := h.monitorSvc.GetCurrentMetrics(ctx)
	if err != nil {
		return healthutil.WithError(result, "METRICS_COLLECTION_FAILED",
			fmt.Sprintf("Cannot collect system metrics: %v", err), "resource", true)
	}

	if metrics == nil {
		return healthutil.WithError(result, "METRICS_NULL",
			"Metrics collection returned null data", "internal", true)
	}

	result["latency_ms"] = nil // Could measure collection time if needed
	return healthutil.MarkConnected(result)
}

// checkInvestigationSystem tests the investigation file system access
func (h *HealthHandler) checkInvestigationSystem() map[string]interface{} {
	result := healthutil.NewResult()
	scenarioRoot, err := resolveSystemMonitorScenarioRoot()
	if err != nil {
		return healthutil.WithError(result, "SCENARIO_ROOT_RESOLUTION",
			fmt.Sprintf("Cannot resolve system-monitor scenario root: %v", err), "resource", false)
	}

	// Check if investigations directory exists and is accessible
	investigationsDir := filepath.Join(scenarioRoot, "investigations")
	if _, err := os.Stat(investigationsDir); err != nil {
		return healthutil.WithError(result, "INVESTIGATION_DIR_ACCESS",
			fmt.Sprintf("Cannot access investigations directory: %v", err), "resource", false)
	}

	// Check if results directory exists and is writable
	resultsDir := filepath.Join(investigationsDir, "results")
	if err := os.MkdirAll(resultsDir, 0o755); err != nil {
		return healthutil.WithError(result, "RESULTS_DIR_WRITE",
			fmt.Sprintf("Cannot create/write results directory: %v", err), "resource", false)
	}

	// Test writing a small test file
	testFile := filepath.Join(resultsDir, ".health_check_test")
	if err := os.WriteFile(testFile, []byte("test"), 0o644); err != nil {
		return healthutil.WithError(result, "FILESYSTEM_WRITE_TEST",
			fmt.Sprintf("Cannot write test file: %v", err), "resource", true)
	}

	// Clean up test file
	os.Remove(testFile)

	return healthutil.MarkConnected(result)
}

func resolveSystemMonitorScenarioRoot() (string, error) {
	repoRoot, err := repocontract.ResolveRepoRoot()
	if err != nil {
		return "", err
	}
	return repocontract.ResolveScenarioPath(repoRoot, "system-monitor")
}

// checkNodeRed tests Node-RED connectivity
func (h *HealthHandler) checkNodeRed() map[string]interface{} {
	result := healthutil.NewResult()

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(h.config.Resources.NodeRedURL)
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			return healthutil.WithError(result, "CONNECTION_REFUSED",
				"Node-RED service not running", "network", true)
		}
		return healthutil.WithError(result, "NODE_RED_CONNECTION_FAILED",
			fmt.Sprintf("Cannot connect to Node-RED: %v", err), "network", true)
	}
	defer resp.Body.Close()

	result["latency_ms"] = nil // Could measure if needed
	return healthutil.MarkConnected(result)
}

// checkOllama tests Ollama over its HTTP health surface. Health handlers must
// not fork a control-plane CLI: that turns every liveness request into a
// process-creation event and makes the handler unavailable when the CLI is.
func (h *HealthHandler) checkOllama() map[string]interface{} {
	result := healthutil.NewResult()
	base := strings.TrimRight(os.Getenv("OLLAMA_BASE_URL"), "/")
	if base == "" { base = "http://127.0.0.1:11434" }
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/tags", nil)
	if err != nil { return healthutil.WithError(result,"OLLAMA_CONNECTION_FAILED",err.Error(),"network",true) }
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return healthutil.WithError(result, "OLLAMA_CONNECTION_FAILED",
			fmt.Sprintf("Ollama HTTP health request failed: %v", err), "network", true)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 { return healthutil.WithError(result,"OLLAMA_CONNECTION_FAILED",fmt.Sprintf("Ollama returned HTTP %d",resp.StatusCode),"network",true) }
	return healthutil.MarkConnected(result)
}

// checkWebhookEndpoint tests webhook URL accessibility
func (h *HealthHandler) checkWebhookEndpoint() map[string]interface{} {
	result := healthutil.NewResult()

	// For webhook, we just check if URL is reachable (don't actually send alert)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Head(h.config.Alerts.WebhookURL)
	if err != nil {
		return healthutil.WithError(result, "WEBHOOK_UNREACHABLE",
			fmt.Sprintf("Webhook URL unreachable: %v", err), "network", true)
	}
	defer resp.Body.Close()

	return healthutil.MarkConnected(result)
}
