package reconcile

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	visualpb "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth"
	visualconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth/visualhealth_v1connect"

	"experience-manager/internal/spec"
)

// GeometryEvaluator applies the canonical ui-health geometry pack to one
// captured artifact. Experience Manager keeps its spec checks, while geometry
// thresholds and findings come from ui-health over VisualStepArtifact.
type GeometryEvaluator interface {
	EvaluateGeometry(ctx context.Context, scenario string, target CaptureTarget, snapshot Snapshot) ([]spec.Finding, error)
}

type visualHealthAnalyzer interface {
	AnalyzeArtifacts(context.Context, *connect.Request[visualpb.AnalyzeArtifactsRequest]) (*connect.Response[visualpb.AnalyzeArtifactsResponse], error)
}

// UIHealthGeometryEvaluator adapts the retained BAS accessibility artifact to
// the canonical VisualStepArtifact contract and asks ui-health to evaluate it.
type UIHealthGeometryEvaluator struct {
	resolveBaseURL func(context.Context) (string, error)
	httpClient     *http.Client
	analyze        visualHealthAnalyzer
}

// NewUIHealthGeometryEvaluator constructs the production client. It resolves
// ui-health for each call so scenario restarts and port changes do not leave a
// stale endpoint cached in Experience Manager.
func NewUIHealthGeometryEvaluator() *UIHealthGeometryEvaluator {
	resolver := discovery.NewResolver(discovery.ResolverConfig{})
	return &UIHealthGeometryEvaluator{
		resolveBaseURL: func(ctx context.Context) (string, error) {
			return resolver.ResolveScenarioURLDefault(ctx, "ui-health")
		},
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (e *UIHealthGeometryEvaluator) EvaluateGeometry(ctx context.Context, scenario string, target CaptureTarget, snapshot Snapshot) ([]spec.Finding, error) {
	client := e.analyze
	if client == nil {
		if e == nil || e.resolveBaseURL == nil {
			return nil, fmt.Errorf("ui-health geometry endpoint resolver is unavailable")
		}
		baseURL, err := e.resolveBaseURL(ctx)
		if err != nil {
			return nil, fmt.Errorf("resolve ui-health endpoint: %w", err)
		}
		if strings.TrimSpace(baseURL) == "" {
			return nil, fmt.Errorf("resolve ui-health endpoint: empty base URL")
		}
		httpClient := e.httpClient
		if httpClient == nil {
			httpClient = &http.Client{Timeout: 15 * time.Second}
		}
		client = visualconnect.NewVisualHealthServiceClient(httpClient, baseURL)
	}
	step, err := visualStepFromAccessibilitySnapshot(target, snapshot)
	if err != nil {
		return nil, err
	}
	resp, err := client.AnalyzeArtifacts(ctx, connect.NewRequest(&visualpb.AnalyzeArtifactsRequest{
		Scenario: scenario,
		RunId:    geometryRunID(target),
		Steps:    []*visualpb.VisualStepArtifact{step},
	}))
	if err != nil {
		return nil, fmt.Errorf("ui-health geometry analysis: %w", err)
	}
	var findings []spec.Finding
	for _, finding := range resp.Msg.GetFindings() {
		if finding == nil || !isCanonicalGeometryRule(finding.GetCode()) {
			continue
		}
		severity := spec.SeverityInfo
		switch finding.GetSeverity() {
		case visualpb.VisualSeverity_VISUAL_SEVERITY_ERROR:
			severity = spec.SeverityError
		case visualpb.VisualSeverity_VISUAL_SEVERITY_WARNING:
			severity = spec.SeverityWarning
		}
		message := strings.TrimSpace(finding.GetMessage())
		if evidence := strings.TrimSpace(finding.GetEvidence()); evidence != "" {
			message += ": " + evidence
		}
		findings = append(findings, spec.Finding{
			Code:       finding.GetCode(),
			Severity:   severity,
			Message:    message,
			Locations:  nonEmptyStrings(finding.GetLocation()),
			Suggestion: finding.GetRemediation(),
		})
	}
	return findings, nil
}

func isCanonicalGeometryRule(code string) bool {
	switch code {
	case "visual_viewport_overflow", "visual_offscreen_interactive", "visual_unsafe_edge_tap_zone",
		"visual_touch_target_too_small", "visual_chrome_outside_viewport", "visual_multiline_chrome_label",
		"visual_surface_does_not_fill_viewport", "visual_geometry_unmeasured", "visual_surface_root_unmeasured":
		return true
	default:
		return false
	}
}

func geometryRunID(target CaptureTarget) string {
	return strings.Join([]string{target.PageID, target.ComponentID, target.Route, target.StateID, target.ViewportID, target.ColorScheme, target.Locale}, ":")
}

func nonEmptyStrings(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return []string{value}
}

func visualStepFromAccessibilitySnapshot(target CaptureTarget, snapshot Snapshot) (*visualpb.VisualStepArtifact, error) {
	if snapshot.Contract != snapshotContract {
		return nil, fmt.Errorf("unsupported accessibility snapshot contract %q", snapshot.Contract)
	}
	if target.ViewportWidth <= 0 || target.ViewportHeight <= 0 {
		return nil, fmt.Errorf("geometry artifact requires a positive viewport, got %dx%d", target.ViewportWidth, target.ViewportHeight)
	}
	nodes := snapshot.Flatten()
	if len(nodes) == 0 {
		return nil, fmt.Errorf("geometry artifact requires a nonempty accessibility tree")
	}
	parents := axParents(nodes)
	elements := make([]map[string]any, 0, len(nodes))
	for index, node := range nodes {
		if node == nil || node.Bounds == nil {
			continue
		}
		element := map[string]any{
			"nodeKey":     node.DOM.TestID,
			"selector":    selectorForAXNode(node),
			"testId":      node.DOM.TestID,
			"tag":         node.DOM.Tag,
			"role":        strings.ToLower(node.Role),
			"text":        node.Name,
			"rect":        map[string]float64{"x": node.Bounds.X, "y": node.Bounds.Y, "width": node.Bounds.Width, "height": node.Bounds.Height},
			"computed":    normalizeAXComputedStyle(node.ComputedStyle),
			"interactive": isInteractiveNode(node),
			"disabled":    containsState(node, "disabled"),
			"surfaceRoot": index == 0,
		}
		if isPreviewWorkspaceScrollNode(node) || clippedHorizontallyByAncestor(node, parents, target) {
			element["inScrollContainer"] = true
		}
		elements = append(elements, element)
	}
	if len(elements) == 0 {
		return nil, fmt.Errorf("geometry artifact has no accessibility nodes with measured bounds")
	}
	layout := map[string]any{
		"viewport":       map[string]int{"width": target.ViewportWidth, "height": target.ViewportHeight},
		"elements":       elements,
		"geometryPolicy": map[string]any{"overflowSlackPx": 2, "unsafeEdgeScope": "app-chrome-controls"},
	}
	if target.ViewportID == "mobile" {
		// Preserve the existing EXM mobile baseline floor. This is an explicit
		// contract margin, not a claim that BAS measured a physical device inset.
		layout["safeArea"] = map[string]any{"bottom": 24, "source": "experience-manager-mobile-floor"}
	}
	root := nodes[0]
	if root != nil {
		if width, ok := axLayoutMetric(root, "scrollWidth", "scroll-width"); ok {
			layout["document"] = map[string]float64{"scrollWidth": width}
		}
	}
	layoutJSON, err := json.Marshal(layout)
	if err != nil {
		return nil, fmt.Errorf("encode canonical geometry artifact: %w", err)
	}
	return &visualpb.VisualStepArtifact{
		StepId: target.ViewportID,
		Label:  strings.Join([]string{target.PageID, target.StateID, target.ViewportID}, "/"),
		Url:    snapshot.URL,
		Viewport: &visualpb.Viewport{
			Width: int32(target.ViewportWidth), Height: int32(target.ViewportHeight), DeviceProfile: target.ViewportID,
		},
		LayoutJson: string(layoutJSON),
	}, nil
}

func selectorForAXNode(node *AXNode) string {
	if node == nil {
		return ""
	}
	if testID := strings.TrimSpace(node.DOM.TestID); testID != "" {
		return `[data-testid="` + cssAttributeValue(testID) + `"]`
	}
	if tag := strings.TrimSpace(node.DOM.Tag); tag != "" {
		return strings.ToLower(tag)
	}
	return strings.ToLower(strings.TrimSpace(node.Role))
}

func cssAttributeValue(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return value
}

func normalizeAXComputedStyle(style map[string]string) map[string]string {
	if len(style) == 0 {
		return nil
	}
	keys := map[string]string{
		"display": "display", "visibility": "visibility", "overflow": "overflow",
		"overflow-x": "overflowX", "overflow-y": "overflowY", "position": "position",
		"pointer-events": "pointerEvents", "font-size": "fontSize", "text-overflow": "textOverflow",
		"white-space": "whiteSpace", "opacity": "opacity",
	}
	out := make(map[string]string, len(style))
	for key, value := range style {
		normalized, ok := keys[strings.ToLower(strings.TrimSpace(key))]
		if !ok {
			normalized = key
		}
		out[normalized] = value
	}
	return out
}

func axLayoutMetric(node *AXNode, keys ...string) (float64, bool) {
	if node == nil {
		return 0, false
	}
	for _, key := range keys {
		for styleKey, value := range node.ComputedStyle {
			if strings.EqualFold(strings.ReplaceAll(styleKey, "-", ""), strings.ReplaceAll(key, "-", "")) {
				parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
				if err == nil && parsed > 0 {
					return parsed, true
				}
			}
		}
	}
	return 0, false
}
