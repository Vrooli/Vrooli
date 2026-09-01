package handlers

// DOC: docs/reference/api-endpoints.md#metrics

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/nodereach"
	"github.com/vrooli/api-core/operatorsession"
	metricspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/metrics"
	metricsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/metrics/metricsconnect"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/apierrors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/convert"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/httputil"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
)

// MetricsHandler handles metrics-related requests
type MetricsHandler struct {
	log        *slog.Logger
	config     *config.Config
	monitorSvc MonitorQuerier
	bridge     *nodereach.Client
}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler(cfg *config.Config, monitorSvc MonitorQuerier, log *slog.Logger) *MetricsHandler {
	return &MetricsHandler{
		log:        log,
		config:     cfg,
		monitorSvc: monitorSvc,
		bridge: nodereach.New(nodereach.Config{
			Token:         firstNonEmpty(os.Getenv("VROOLI_BRIDGE_API_TOKEN"), os.Getenv("VROOLI_API_TOKEN")),
			TokenProvider: resolveLocalOwnerToken,
		}),
	}
}

// resolveLocalOwnerToken lets an installed local Vrooli app use the enrolled
// operator session without copying a long-lived Bridge credential into its
// environment. The node client calls this only when no explicit compatibility
// token is configured, and requests a fresh short-lived session per operation.
func resolveLocalOwnerToken(_ context.Context) (string, error) {
	store, err := operatorsession.DefaultFileStore()
	if err != nil {
		return "", nil
	}
	resolution, err := (operatorsession.LocalResolver{Store: store}).Resolve()
	if err != nil || strings.TrimSpace(resolution.Token) == "" {
		return "", nil
	}
	return operatorsession.LocalSessionScheme + " " + resolution.Token, nil
}

// GetCurrentMetrics handles the typed Connect-RPC metrics snapshot contract.
func (h *MetricsHandler) GetCurrentMetrics(ctx context.Context, req *connect.Request[metricspb.GetCurrentMetricsRequest]) (*connect.Response[metricspb.GetCurrentMetricsResponse], error) {
	var (
		metrics *models.MetricsResponse
		err     error
	)
	if req.Msg.GetFresh() {
		metrics, err = h.monitorSvc.GetCurrentMetricsFresh(ctx)
	} else {
		metrics, err = h.monitorSvc.GetCurrentMetrics(ctx)
	}
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&metricspb.GetCurrentMetricsResponse{
		Metrics: convert.MetricsResponseToProto(metrics),
	}), nil
}

// GetDetailedMetrics handles the typed Connect-RPC detailed metrics contract.
func (h *MetricsHandler) GetDetailedMetrics(ctx context.Context, _ *connect.Request[metricspb.GetDetailedMetricsRequest]) (*connect.Response[metricspb.GetDetailedMetricsResponse], error) {
	metrics, err := h.monitorSvc.GetDetailedMetrics(ctx)
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&metricspb.GetDetailedMetricsResponse{
		Metrics: convert.DetailedMetricsToProto(metrics),
	}), nil
}

// GetProcessMonitor handles the typed Connect-RPC process monitor contract.
func (h *MetricsHandler) GetProcessMonitor(ctx context.Context, _ *connect.Request[metricspb.GetProcessMonitorRequest]) (*connect.Response[metricspb.GetProcessMonitorResponse], error) {
	data, err := h.monitorSvc.GetProcessMonitorData(ctx)
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&metricspb.GetProcessMonitorResponse{
		Data: convert.ProcessMonitorDataToProto(data),
	}), nil
}

// GetProcessTimeline handles the typed Connect-RPC process attribution timeline contract.
func (h *MetricsHandler) GetProcessTimeline(ctx context.Context, req *connect.Request[metricspb.GetProcessTimelineRequest]) (*connect.Response[metricspb.GetProcessTimelineResponse], error) {
	window := 5 * time.Minute
	if req.Msg.GetWindowSeconds() > 0 {
		window = time.Duration(req.Msg.GetWindowSeconds()) * time.Second
	}
	top := 20
	if req.Msg.GetTop() > 0 {
		top = int(req.Msg.GetTop())
	}
	owner := req.Msg.GetOwner()
	rank := req.Msg.GetRank()
	if rank == "" {
		rank = "cpu"
	}

	entries, err := h.monitorSvc.GetProcessTimelineRanked(ctx, window, owner, top, rank)
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&metricspb.GetProcessTimelineResponse{
		Timeline: convert.ProcessTimelineResponseToProto(int(window.Seconds()), owner, top, entries),
	}), nil
}

// GetInfrastructureMonitor handles the typed Connect-RPC infrastructure metrics contract.
func (h *MetricsHandler) GetInfrastructureMonitor(ctx context.Context, _ *connect.Request[metricspb.GetInfrastructureMonitorRequest]) (*connect.Response[metricspb.GetInfrastructureMonitorResponse], error) {
	data, err := h.monitorSvc.GetInfrastructureMonitorData(ctx)
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&metricspb.GetInfrastructureMonitorResponse{
		Data: convert.InfrastructureMonitorDataToProto(data),
	}), nil
}

// GetMetricsTimeline handles the typed Connect-RPC metrics timeline contract.
func (h *MetricsHandler) GetMetricsTimeline(ctx context.Context, req *connect.Request[metricspb.GetMetricsTimelineRequest]) (*connect.Response[metricspb.GetMetricsTimelineResponse], error) {
	windowSeconds := 120
	if req.Msg.GetWindowSeconds() > 0 {
		windowSeconds = int(req.Msg.GetWindowSeconds())
	}
	sampleInterval := 5
	if req.Msg.GetSampleIntervalSeconds() > 0 {
		sampleInterval = int(req.Msg.GetSampleIntervalSeconds())
	}

	timeline, err := h.monitorSvc.GetMetricsTimeline(ctx, windowSeconds, sampleInterval)
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&metricspb.GetMetricsTimelineResponse{
		Timeline: convert.MetricsTimelineResponseToProto(timeline),
	}), nil
}

// GetDiskDetail handles the typed Connect-RPC disk detail contract.
func (h *MetricsHandler) GetDiskDetail(ctx context.Context, _ *connect.Request[metricspb.GetDiskDetailRequest]) (*connect.Response[metricspb.GetDiskDetailResponse], error) {
	detail, err := h.monitorSvc.GetDiskDetail(ctx)
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&metricspb.GetDiskDetailResponse{
		Data: convert.DiskDetailResponseToProto(detail),
	}), nil
}

// HandleGetCurrentMetrics handles GET /api/v1/metrics/current.
func (h *MetricsHandler) HandleGetCurrentMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if nodeID := strings.TrimSpace(r.URL.Query().Get("node")); nodeID != "" {
		h.handleRemoteCurrentMetrics(w, r, nodeID)
		return
	}

	fresh := r.URL.Query().Get("fresh")
	var (
		metrics *models.MetricsResponse
		err     error
	)
	if fresh == "1" || fresh == "true" {
		metrics, err = h.monitorSvc.GetCurrentMetricsFresh(ctx)
	} else {
		metrics, err = h.monitorSvc.GetCurrentMetrics(ctx)
	}
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	httputil.SafeProtoJSON(w, h.log, r, convert.MetricsResponseToProto(metrics))
}

// HandleGetMachines exposes the same provider-neutral readiness facts used by
// other target surfaces. The local machine is implicit; Bridge nodes are
// listed from the durable registry with freshness already computed by Bridge.
func (h *MetricsHandler) HandleGetMachines(w http.ResponseWriter, r *http.Request) {
	if h.bridge == nil {
		http.Error(w, "Bridge client is unavailable", http.StatusServiceUnavailable)
		return
	}
	nodes, err := h.bridge.List(r.Context(), 5*time.Second)
	if err != nil {
		h.handleNodeClientError(w, r, err)
		return
	}
	result := make([]machineView, 0, len(nodes)+1)
	result = append(result, machineView{ID: "", Name: "This machine", OS: runtimeOS(), Arch: runtimeArch(), Online: true, HeartbeatFresh: true, Dispatchable: true, Status: "local"})
	for _, node := range nodes {
		if node == nil {
			continue
		}
		result = append(result, machineView{
			ID: node.GetId(), Name: firstNonEmpty(node.GetName(), node.GetId()), OS: node.GetOs(), Arch: node.GetArch(),
			Online: node.GetOnline(), HeartbeatFresh: node.GetHeartbeatFresh(), HeartbeatAgeSeconds: heartbeatAge(node),
			Dispatchable: node.GetDispatchable(), Status: node.GetStatus().String(), Grant: grantSummary(node.GetScopes()), Scopes: append([]string(nil), node.GetScopes()...), Readiness: nodeReadiness(node),
		})
	}
	_ = httputil.JSON(w, result)
}

type machineView struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	OS             string `json:"os,omitempty"`
	Arch           string `json:"arch,omitempty"`
	Online         bool   `json:"online"`
	HeartbeatFresh bool   `json:"heartbeat_fresh"`
	// A pointer, not an int with omitempty: a heartbeat that arrived this
	// second has age 0, and `omitempty` would drop that as if the node had
	// never reported — which reads on the UI as "no age known" for the
	// freshest node in the fleet.
	HeartbeatAgeSeconds *int64           `json:"heartbeat_age_seconds,omitempty"`
	Dispatchable        bool             `json:"dispatchable"`
	Status              string           `json:"status"`
	Grant               string           `json:"grant,omitempty"`
	Scopes              []string         `json:"scopes,omitempty"`
	Readiness           []map[string]any `json:"readiness,omitempty"`
}

// heartbeatAge reports the node's heartbeat age when the registry has one.
// Nodes that never reported return nil so a surface can say "unknown" instead
// of presenting a fabricated zero.
func heartbeatAge(node *registryv1.Node) *int64 {
	if !node.GetRegistryRecordPresent() {
		return nil
	}
	age := node.GetHeartbeatAgeSeconds()
	return &age
}

// grantSummary is the operator-facing form of a node's concrete scopes. Keep
// the raw scopes beside it for audit, but make the common permission level
// legible in product controls without requiring an operator to parse scope
// syntax.
func grantSummary(scopes []string) string {
	hasRead, hasWrite, hasDestructive := false, false, false
	for _, raw := range scopes {
		parts := strings.SplitN(strings.ToLower(strings.TrimSpace(raw)), ":", 2)
		if len(parts) != 2 {
			continue
		}
		switch parts[1] {
		case "read", "*":
			hasRead = true
		case "write":
			hasWrite = true
		case "destructive":
			hasDestructive = true
		}
	}
	switch {
	case hasDestructive:
		return "Full control, including destructive actions"
	case hasWrite:
		return "Read and operate; destructive actions withheld"
	case hasRead:
		return "Read only; changes are not permitted"
	default:
		return "No remote actions granted"
	}
}

func nodeReadiness(node *registryv1.Node) []map[string]any {
	return []map[string]any{
		{"identity": "registry_record", "passed": node.GetRegistryRecordPresent()},
		{"identity": "heartbeat_fresh", "passed": node.GetHeartbeatFresh()},
		{"identity": "channel_held", "passed": node.GetChannelHeld()},
		{"identity": "protocol_compatible", "passed": node.GetProtocolCompatible()},
		{"identity": "dispatchable", "passed": node.GetDispatchable()},
	}
}

func (h *MetricsHandler) handleRemoteCurrentMetrics(w http.ResponseWriter, r *http.Request, nodeID string) {
	if h.bridge == nil {
		http.Error(w, "Bridge client is unavailable", http.StatusServiceUnavailable)
		return
	}
	fresh := r.URL.Query().Get("fresh") == "1" || r.URL.Query().Get("fresh") == "true"
	callCtx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	baseURL, err := h.bridge.ScenarioURL(callCtx, nodeID, "system-monitor")
	if err != nil {
		h.handleNodeClientError(w, r, err)
		return
	}
	client := metricsconnect.NewMetricsServiceClient(h.bridge.ConnectTransport(callCtx, baseURL), baseURL)
	resp, err := client.GetCurrentMetrics(callCtx, connect.NewRequest(&metricspb.GetCurrentMetricsRequest{Fresh: fresh}))
	if err != nil {
		h.handleNodeClientError(w, r, err)
		return
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetMetrics() == nil {
		h.handleNodeClientError(w, r, errors.New("remote metrics response was empty"))
		return
	}
	httputil.SafeProtoJSONCamel(w, h.log, r, resp.Msg.GetMetrics())
}

func (h *MetricsHandler) handleNodeClientError(w http.ResponseWriter, r *http.Request, err error) {
	httputil.HandleError(w, h.log, r, apierrors.Unavailable("remote node"))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func runtimeOS() string   { return runtime.GOOS }
func runtimeArch() string { return runtime.GOARCH }

// HandleGetPressureSnapshot handles GET /api/v1/metrics/pressure. It is a
// plain JSON operational surface because pressure is host evidence, not part
// of the existing compatibility metrics protobuf.
func (h *MetricsHandler) HandleGetPressureSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshot, err := h.monitorSvc.GetPressureSnapshot(r.Context())
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}
	httputil.JSON(w, snapshot) //nolint:errcheck
}

// HandleGetGPUHistory handles GET /api/v1/forensics/gpu?window=1h. It exposes
// bounded persisted GPU utilization and VRAM evidence without a new scan.
func (h *MetricsHandler) HandleGetGPUHistory(w http.ResponseWriter, r *http.Request) {
	window := time.Hour
	if raw := r.URL.Query().Get("window"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil || parsed <= 0 {
			httputil.HandleError(w, h.log, r, apierrors.Validation("window", "must be a positive duration"))
			return
		}
		window = parsed
	}
	history, err := h.monitorSvc.GetGPUHistory(r.Context(), window)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}
	httputil.JSON(w, history) //nolint:errcheck
}

// HandleGetPressureHistory handles GET /api/v1/forensics/pressure?window=1h.
func (h *MetricsHandler) HandleGetPressureHistory(w http.ResponseWriter, r *http.Request) {
	window := time.Hour
	if raw := r.URL.Query().Get("window"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil || parsed <= 0 {
			httputil.HandleError(w, h.log, r, apierrors.Validation("window", "must be a positive duration"))
			return
		}
		window = parsed
	}
	history, err := h.monitorSvc.GetPressureHistory(r.Context(), window)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}
	httputil.JSON(w, history) //nolint:errcheck
}

// HandleGetMetricsTimeline handles GET /api/v1/metrics/timeline.
func (h *MetricsHandler) HandleGetMetricsTimeline(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	windowSeconds := 120
	if ws := r.URL.Query().Get("window"); ws != "" {
		if parsed, err := strconv.Atoi(ws); err == nil && parsed > 0 {
			windowSeconds = parsed
		}
	}

	sampleInterval := 5
	if si := r.URL.Query().Get("interval"); si != "" {
		if parsed, err := strconv.Atoi(si); err == nil && parsed > 0 {
			sampleInterval = parsed
		}
	}

	timeline, err := h.monitorSvc.GetMetricsTimeline(ctx, windowSeconds, sampleInterval)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	httputil.SafeProtoJSON(w, h.log, r, convert.MetricsTimelineResponseToProto(timeline))
}

// HandleGetDetailedMetrics handles GET /api/v1/metrics/detailed.
func (h *MetricsHandler) HandleGetDetailedMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	metrics, err := h.monitorSvc.GetDetailedMetrics(ctx)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	httputil.SafeProtoJSON(w, h.log, r, convert.DetailedMetricsToProto(metrics))
}

// HandleGetProcessMonitor handles GET /api/v1/metrics/processes.
func (h *MetricsHandler) HandleGetProcessMonitor(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	data, err := h.monitorSvc.GetProcessMonitorData(ctx)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	httputil.SafeProtoJSON(w, h.log, r, convert.ProcessMonitorDataToProto(data))
}

// processTimelineEntryJSON is the wire shape for one ranked consumer. Plain
// JSON (not proto) following the forensics/logs precedent in this scenario:
// the attribution timeline has no cross-scenario clients, so adding ~2 proto
// messages + a convert layer would be disproportionate. See forensics.go.
type processTimelineEntryJSON struct {
	Owner       string  `json:"owner"`
	Comm        string  `json:"comm"`
	PID         int     `json:"pid,omitempty"`
	Aggregated  bool    `json:"aggregated"`
	CPUPct      float64 `json:"cpu_pct"`
	CPUSeconds  float64 `json:"cpu_seconds"`
	MaxCPUPct   float64 `json:"max_cpu_pct"`
	RSSKB       int64   `json:"rss_kb"`
	GPUVRAMMB   float64 `json:"gpu_vram_mb"`
	SampleCount int64   `json:"sample_count"`
	FirstSeen   string  `json:"first_seen,omitempty"`
	LastSeen    string  `json:"last_seen,omitempty"`
}

type processTimelineResponseJSON struct {
	WindowSeconds int                        `json:"window_seconds"`
	Owner         string                     `json:"owner,omitempty"`
	Top           int                        `json:"top"`
	Rank          string                     `json:"rank"`
	Count         int                        `json:"count"`
	Entries       []processTimelineEntryJSON `json:"entries"`
}

// HandleGetProcessTimeline handles GET /api/v1/metrics/processes/timeline. Query
// params: window (duration, default 5m), owner (scenario filter), top (int).
// It returns ranked consumers over the window, grouped by owner/scenario —
// the standing replacement for the manual `ps`/`top` forensic.
func (h *MetricsHandler) HandleGetProcessTimeline(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	window := 5 * time.Minute
	if ws := r.URL.Query().Get("window"); ws != "" {
		if parsed, err := time.ParseDuration(ws); err == nil && parsed > 0 {
			window = parsed
		} else if secs, err := strconv.Atoi(ws); err == nil && secs > 0 {
			// Accept a bare integer as seconds for parity with /metrics/timeline.
			window = time.Duration(secs) * time.Second
		}
	}

	owner := r.URL.Query().Get("owner")

	top := 20
	if t := r.URL.Query().Get("top"); t != "" {
		if parsed, err := strconv.Atoi(t); err == nil && parsed > 0 {
			top = parsed
		}
	}

	rank := r.URL.Query().Get("rank")
	if rank != "rss" && rank != "gpu" && rank != "cpu_seconds" {
		rank = "cpu"
	}
	entries, err := h.monitorSvc.GetProcessTimelineRanked(ctx, window, owner, top, rank)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	out := make([]processTimelineEntryJSON, 0, len(entries))
	for _, e := range entries {
		row := processTimelineEntryJSON{
			Owner:       e.Owner,
			Comm:        e.Comm,
			PID:         e.PID,
			Aggregated:  e.Aggregated,
			CPUPct:      e.CPUPct,
			CPUSeconds:  e.CPUSeconds,
			MaxCPUPct:   e.MaxCPUPct,
			RSSKB:       e.RSSKB,
			GPUVRAMMB:   e.GPUVRAMMB,
			SampleCount: e.SampleCount,
		}
		if !e.FirstSeen.IsZero() {
			row.FirstSeen = e.FirstSeen.UTC().Format(time.RFC3339)
		}
		if !e.LastSeen.IsZero() {
			row.LastSeen = e.LastSeen.UTC().Format(time.RFC3339)
		}
		out = append(out, row)
	}

	httputil.JSON(w, processTimelineResponseJSON{ //nolint:errcheck
		WindowSeconds: int(window.Seconds()),
		Owner:         owner,
		Top:           top,
		Rank:          rank,
		Count:         len(out),
		Entries:       out,
	})
}

// HandleGetInfrastructureMonitor handles GET /api/v1/metrics/infrastructure.
func (h *MetricsHandler) HandleGetInfrastructureMonitor(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	data, err := h.monitorSvc.GetInfrastructureMonitorData(ctx)
	if err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	httputil.SafeProtoJSON(w, h.log, r, convert.InfrastructureMonitorDataToProto(data))
}

func connectError(err error) error {
	var apiErr *apierrors.APIError
	if errors.As(err, &apiErr) {
		return connect.NewError(apiErrorCode(apiErr.Category), err)
	}
	if errors.Is(err, context.Canceled) {
		return connect.NewError(connect.CodeCanceled, err)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}

func apiErrorCode(category apierrors.Category) connect.Code {
	switch category {
	case apierrors.CategoryValidation:
		return connect.CodeInvalidArgument
	case apierrors.CategoryUnauthorized:
		return connect.CodeUnauthenticated
	case apierrors.CategoryForbidden:
		return connect.CodePermissionDenied
	case apierrors.CategoryNotFound:
		return connect.CodeNotFound
	case apierrors.CategoryConflict:
		return connect.CodeAborted
	case apierrors.CategoryCooldown:
		return connect.CodeResourceExhausted
	case apierrors.CategoryUnavailable:
		return connect.CodeUnavailable
	default:
		return connect.CodeInternal
	}
}
