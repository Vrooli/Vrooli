package uiruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"

	"ui-health/internal/evidence"

	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api/apiconnect"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basexec "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	commonpb "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

const (
	maxRuntimeConsoleEntries      = 200
	maxRuntimeNetworkEntries      = 200
	maxRuntimeConsoleEntryBytes   = 4096
	maxRuntimeNetworkPreviewBytes = 4096
)

// errBASUnavailable signals the BAS engine could not be reached or driven. The
// runner maps it to a skipped (resource unavailable) finding — never a failure.
var errBASUnavailable = errors.New("browser-automation-studio unavailable")

// runResult is the thin reduction of a BAS ExecutionTimeline the runtime group
// needs: the handshake assert outcome, a screenshot-present flag, and
// best-effort console/network observations.
type runResult struct {
	loaded            bool
	loadError         string
	handshakeSignaled bool
	handshakeError    string
	screenshotRef     string
	screenshotPNG     []byte
	domHTML           string
	layoutJSON        string
	viewportWidth     int32
	viewportHeight    int32
	console           []evidence.ConsoleEntry
	network           []evidence.NetworkEntry
}

// evidence converts the run result into the engine-agnostic evidence the shared
// analyzer verdicts on.
func (r *runResult) evidenceFor(url string) evidence.Evidence {
	if r == nil {
		return evidence.Evidence{URL: url, Loaded: false, LoadError: "no BAS result"}
	}
	return evidence.Evidence{
		URL:           url,
		Loaded:        r.loaded,
		LoadError:     r.loadError,
		Handshake:     evidence.Handshake{Signaled: r.handshakeSignaled, TimedOut: !r.handshakeSignaled, Error: r.handshakeError},
		Console:       r.console,
		Network:       r.network,
		ScreenshotRef: r.screenshotRef,
	}
}

// basRunner drives one handshake workflow on BAS and returns the thin result.
type basRunner interface {
	Run(ctx context.Context, def map[string]any) (*runResult, error)
}

// connectRunner is the production basRunner over BAS's WorkflowsService +
// ExecutionsService Connect-RPCs.
type connectRunner struct {
	// resolveBAS returns the BAS scenario base URL (scheme://host:port, no
	// /api/v1 suffix). nil is not valid — New wires a discovery-backed resolver.
	resolveBAS func(ctx context.Context) (string, error)
	httpClient connect.HTTPClient
	// pollInterval / pollTimeout bound the execution wait loop.
	pollInterval time.Duration
	pollTimeout  time.Duration
}

func (c *connectRunner) Run(ctx context.Context, def map[string]any) (*runResult, error) {
	baseURL, err := c.resolveBAS(ctx)
	if err != nil || strings.TrimSpace(baseURL) == "" {
		return nil, errBASUnavailable
	}
	baseURL = strings.TrimRight(baseURL, "/")

	httpClient := c.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 120 * time.Second}
	}
	wf := apiconnect.NewWorkflowsServiceClient(httpClient, baseURL)
	ex := apiconnect.NewExecutionsServiceClient(httpClient, baseURL)

	proto, err := definitionToProto(def)
	if err != nil {
		return nil, fmt.Errorf("build workflow definition: %w", err)
	}

	artifactConfig := runtimeArtifactConfig()
	resp, err := wf.ExecuteAdhocWorkflow(ctx, connect.NewRequest(&basexec.ExecuteAdhocRequest{
		FlowDefinition: proto,
		Metadata:       &basexec.ExecutionMetadata{Name: "ui-health-runtime", Description: "ui-health runtime/render"},
		Parameters: &basexec.ExecutionParameters{
			ArtifactConfig: artifactConfig,
		},
	}))
	if err != nil {
		return nil, errBASUnavailable
	}
	execID := resp.Msg.GetExecutionId()
	if strings.TrimSpace(execID) == "" {
		return nil, errBASUnavailable
	}

	// Poll to terminal. A workflow *failure* (e.g. the handshake assert timing
	// out) is an expected runtime outcome, not a transport error — the timeline
	// is the source of truth, so a non-completed terminal state is not fatal here.
	c.waitTerminal(ctx, ex, execID)

	tlResp, err := ex.GetExecutionTimeline(ctx, connect.NewRequest(&basapi.GetExecutionTimelineRequest{ExecutionId: execID}))
	if err != nil {
		return nil, errBASUnavailable
	}
	result := readTimeline(tlResp.Msg)
	if png, ref := c.downloadExecutionScreenshot(ctx, ex, baseURL, execID); len(png) > 0 {
		result.screenshotPNG = png
		result.screenshotRef = firstNonEmpty(ref, result.screenshotRef)
	}
	return result, nil
}

func runtimeArtifactConfig() *basexec.ArtifactCollectionConfig {
	profile := "custom"
	on := true
	off := false
	maxConsoleBytes := int32(maxRuntimeConsoleEntryBytes)
	maxNetworkBytes := int32(maxRuntimeNetworkPreviewBytes)
	return &basexec.ArtifactCollectionConfig{
		Profile:                &profile,
		CollectScreenshots:     &on,
		CollectDomSnapshots:    &off,
		CollectConsoleLogs:     &on,
		CollectNetworkEvents:   &on,
		CollectExtractedData:   &on,
		CollectAssertions:      &on,
		CollectCursorTrails:    &off,
		CollectTelemetry:       &off,
		MaxConsoleEntryBytes:   &maxConsoleBytes,
		MaxNetworkPreviewBytes: &maxNetworkBytes,
	}
}

func (c *connectRunner) downloadExecutionScreenshot(ctx context.Context, ex apiconnect.ExecutionsServiceClient, baseURL, execID string) ([]byte, string) {
	shots, err := ex.GetExecutionScreenshots(ctx, connect.NewRequest(&basapi.GetExecutionScreenshotsRequest{ExecutionId: execID}))
	if err != nil {
		return nil, ""
	}
	var selectedURL string
	for _, entry := range shots.Msg.GetScreenshots() {
		if entry.GetNodeId() == nodeScreens && entry.GetScreenshot() != nil {
			selectedURL = entry.GetScreenshot().GetUrl()
			break
		}
	}
	if selectedURL == "" {
		for i := len(shots.Msg.GetScreenshots()) - 1; i >= 0; i-- {
			if shot := shots.Msg.GetScreenshots()[i].GetScreenshot(); shot != nil && strings.TrimSpace(shot.GetUrl()) != "" {
				selectedURL = shot.GetUrl()
				break
			}
		}
	}
	if selectedURL == "" {
		return nil, ""
	}
	data, err := c.downloadAsset(ctx, baseURL, selectedURL)
	if err != nil {
		return nil, selectedURL
	}
	return data, selectedURL
}

func (c *connectRunner) downloadAsset(ctx context.Context, baseURL, assetURL string) ([]byte, error) {
	fullURL := strings.TrimSpace(assetURL)
	if fullURL == "" {
		return nil, errors.New("asset URL is empty")
	}
	if !strings.HasPrefix(fullURL, "http://") && !strings.HasPrefix(fullURL, "https://") {
		if strings.HasPrefix(fullURL, "/") {
			fullURL = strings.TrimRight(baseURL, "/") + fullURL
		} else {
			fullURL = strings.TrimRight(baseURL, "/") + "/" + fullURL
		}
	}
	httpClient := c.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("asset download failed: status=%s url=%s", resp.Status, fullURL)
	}
	return io.ReadAll(resp.Body)
}

func (c *connectRunner) waitTerminal(ctx context.Context, ex apiconnect.ExecutionsServiceClient, execID string) {
	interval := c.pollInterval
	if interval <= 0 {
		interval = time.Second
	}
	timeout := c.pollTimeout
	if timeout <= 0 {
		timeout = 90 * time.Second
	}
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		st, err := ex.GetExecution(ctx, connect.NewRequest(&basapi.GetExecutionRequest{ExecutionId: execID}))
		if err == nil && isTerminal(st.Msg.GetExecution().GetStatus()) {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if time.Now().After(deadline) {
				return
			}
		}
	}
}

func isTerminal(s basbase.ExecutionStatus) bool {
	switch s {
	case basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED,
		basbase.ExecutionStatus_EXECUTION_STATUS_FAILED,
		basbase.ExecutionStatus_EXECUTION_STATUS_CANCELLED:
		return true
	default:
		return false
	}
}

// definitionToProto turns the workflow-definition map into the BAS proto,
// mirroring test-genie's proven conversion (marshal → protojson unmarshal,
// discarding unknown fields for forward-compat).
func definitionToProto(def map[string]any) (*basworkflows.WorkflowDefinitionV2, error) {
	raw, err := json.Marshal(def)
	if err != nil {
		return nil, err
	}
	out := &basworkflows.WorkflowDefinitionV2{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, out); err != nil {
		return nil, err
	}
	return out, nil
}

// readTimeline reduces the BAS ExecutionTimeline to the runtime group's thin
// result: the handshake assert (the hard gate), screenshot presence, and
// best-effort console (timeline logs) / network (NETWORK_EVENT artifacts).
func readTimeline(tl *bastimeline.ExecutionTimeline) *runResult {
	r := &runResult{loaded: true}
	if tl == nil {
		return &runResult{loaded: false, loadError: "BAS produced no timeline for the workflow"}
	}
	handshakeSeen := false
	for _, e := range tl.GetEntries() {
		// A lost Playwright session is causal infrastructure failure, never an
		// iframe handshake failure. Preserve the first such timeline error even
		// if the later handshake assertion reports a timeout.
		if r.loaded && isSessionLoss(e.GetContext().GetError()) {
			r.loaded = false
			r.loadError = e.GetContext().GetError()
		}
		switch e.GetNodeId() {
		case nodeHandshake:
			handshakeSeen = true
			if a := e.GetContext().GetAssertion(); a != nil {
				r.handshakeSignaled = a.GetSuccess()
				if !a.GetSuccess() {
					r.handshakeError = firstNonEmpty(a.GetMessage(), e.GetContext().GetError())
				}
			} else {
				r.handshakeSignaled = e.GetContext().GetSuccess()
				if !r.handshakeSignaled {
					r.handshakeError = e.GetContext().GetError()
				}
			}
		case nodeScreens:
			if e.GetTelemetry().GetScreenshot() != nil {
				r.screenshotRef = "captured"
			}
			for _, art := range e.GetAggregates().GetArtifacts() {
				if art.GetType() == basbase.ArtifactType_ARTIFACT_TYPE_SCREENSHOT {
					r.screenshotRef = "captured"
				}
			}
		case nodeArtifacts:
			r.applyVisualArtifacts(e.GetContext().GetExtractedData())
			// The persisted execution writer keeps an extracted-data preview in
			// aggregates even when the live context is compacted. Runtime evidence
			// must consume that durable representation as well.
			if e.GetAggregates().GetExtractedDataPreview() != nil {
				r.applyVisualArtifactValue(e.GetAggregates().GetExtractedDataPreview())
			}
		}
	}
	if !handshakeSeen {
		// The handshake step never ran (an earlier step failed) — fail closed.
		r.handshakeError = "handshake step did not execute"
	}
	for _, l := range tl.GetLogs() {
		r.console = append(r.console, evidence.ConsoleEntry{
			Level:   normalizeLevel(l.GetLevel().String()),
			Message: l.GetMessage(),
		})
	}
	r.console = boundConsoleEntries(r.console)
	r.network = boundNetworkEntries(r.network)
	return r
}

func isSessionLoss(message string) bool {
	message = strings.ToLower(message)
	return strings.Contains(message, "session not found") ||
		strings.Contains(message, "page has been closed") ||
		strings.Contains(message, "target page, context or browser has been closed")
}

func (r *runResult) applyVisualArtifacts(values map[string]*commonpb.JsonValue) {
	if r == nil || len(values) == 0 {
		return
	}
	// BAS returns an evaluate result under "result" in the node timeline. A
	// storeResult key additionally makes it available to later workflow nodes,
	// but does not rewrite the node's extracted-data envelope. Accept both forms
	// so the artifact contract survives BAS versions that materialize either one.
	raw := values["visual_artifacts"]
	if raw == nil {
		raw = values["result"]
	}
	r.applyVisualArtifactValue(raw)
}

func (r *runResult) applyVisualArtifactValue(raw *commonpb.JsonValue) {
	if r == nil || raw == nil {
		return
	}
	payload, ok := jsonValueToAny(raw).(map[string]any)
	if !ok {
		return
	}
	if nested, ok := payload["visual_artifacts"].(map[string]any); ok {
		payload = nested
	} else if nested, ok := payload["result"].(map[string]any); ok {
		payload = nested
	}
	if dom, ok := payload["domHtml"].(string); ok {
		r.domHTML = dom
	}
	if layout, ok := payload["layout"].(map[string]any); ok {
		if raw, err := json.Marshal(layout); err == nil {
			r.layoutJSON = string(raw)
		}
	}
	if viewport, ok := payload["viewport"].(map[string]any); ok {
		r.viewportWidth = int32(numberFromAny(viewport["width"]))
		r.viewportHeight = int32(numberFromAny(viewport["height"]))
	}
	if network, ok := payload["network"].([]any); ok {
		r.network = append(r.network, networkEntriesFromAny(network)...)
	}
}

func jsonValueToAny(v *commonpb.JsonValue) any {
	if v == nil {
		return nil
	}
	switch kind := v.GetKind().(type) {
	case *commonpb.JsonValue_BoolValue:
		return kind.BoolValue
	case *commonpb.JsonValue_IntValue:
		return float64(kind.IntValue)
	case *commonpb.JsonValue_DoubleValue:
		return kind.DoubleValue
	case *commonpb.JsonValue_StringValue:
		return kind.StringValue
	case *commonpb.JsonValue_BytesValue:
		return kind.BytesValue
	case *commonpb.JsonValue_ListValue:
		values := kind.ListValue.GetValues()
		out := make([]any, 0, len(values))
		for _, item := range values {
			out = append(out, jsonValueToAny(item))
		}
		return out
	case *commonpb.JsonValue_ObjectValue:
		fields := kind.ObjectValue.GetFields()
		out := make(map[string]any, len(fields))
		for key, value := range fields {
			out[key] = jsonValueToAny(value)
		}
		return out
	default:
		return nil
	}
}

func networkEntriesFromAny(values []any) []evidence.NetworkEntry {
	out := make([]evidence.NetworkEntry, 0, len(values))
	for _, item := range values {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		var status *int
		if n := int(numberFromAny(m["status"])); n >= 400 {
			status = &n
		}
		errorText := stringFromAny(m["errorText"])
		// ResourceTiming exposes successful resources but not HTTP response
		// status. Do not turn those status=0 observations into failures; real
		// failures arrive from BAS network telemetry with a 4xx/5xx or error text.
		if status == nil && strings.TrimSpace(errorText) == "" {
			continue
		}
		out = append(out, evidence.NetworkEntry{
			URL:          stringFromAny(m["url"]),
			Method:       stringFromAny(m["method"]),
			ResourceType: stringFromAny(m["resourceType"]),
			Status:       status,
			ErrorText:    errorText,
		})
	}
	return out
}

func boundConsoleEntries(entries []evidence.ConsoleEntry) []evidence.ConsoleEntry {
	if len(entries) <= maxRuntimeConsoleEntries {
		return entries
	}
	out := make([]evidence.ConsoleEntry, 0, maxRuntimeConsoleEntries)
	for _, entry := range entries {
		if len(out) >= maxRuntimeConsoleEntries {
			break
		}
		switch entry.Level {
		case "error", "warn", "warning":
			out = append(out, entry)
		}
	}
	for i := len(entries) - 1; i >= 0 && len(out) < maxRuntimeConsoleEntries; i-- {
		switch entries[i].Level {
		case "error", "warn", "warning":
			continue
		default:
			out = append(out, entries[i])
		}
	}
	return out
}

func boundNetworkEntries(entries []evidence.NetworkEntry) []evidence.NetworkEntry {
	if len(entries) <= maxRuntimeNetworkEntries {
		return entries
	}
	return entries[len(entries)-maxRuntimeNetworkEntries:]
}

func stringFromAny(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func numberFromAny(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}

// normalizeLevel maps a BAS LogLevel enum string (LOG_LEVEL_ERROR) to the
// lowercase token the evidence analyzer counts ("error"/"warn"/...).
func normalizeLevel(level string) string {
	token := strings.ToLower(strings.TrimPrefix(level, "LOG_LEVEL_"))
	switch token {
	case "warning":
		return "warn"
	default:
		return token
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
