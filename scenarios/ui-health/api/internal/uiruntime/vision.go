package uiruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"connectrpc.com/connect"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	"ui-health/internal/services/manifestvalidation"
)

// ux.visual-critique is the AI Gateway role for screenshot UX findings.
// locate.visual has a distinct coordinate-grounding contract and normalizes
// output to bounds, so it cannot serve this findings schema.
const visionCriticRole = "ux.visual-critique"

type visionJudge interface {
	Check(context.Context, []byte, int, int, string, string) []manifestvalidation.Finding
}

type visionRunClient interface {
	Run(context.Context, *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error)
}

type gatewayVisionJudge struct {
	client func(context.Context) (visionRunClient, error)
}

type visionRunCircuitKey struct{}

type visionRunCircuit struct {
	mu     sync.Mutex
	open   bool
	reason string
}

func withVisionRunCircuit(ctx context.Context) context.Context {
	return context.WithValue(ctx, visionRunCircuitKey{}, &visionRunCircuit{})
}

func visionCircuitFrom(ctx context.Context) *visionRunCircuit {
	circuit, _ := ctx.Value(visionRunCircuitKey{}).(*visionRunCircuit)
	return circuit
}

func (c *visionRunCircuit) isOpen() bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.open
}

func (c *visionRunCircuit) trip(reason string) bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.open {
		return false
	}
	c.open = true
	c.reason = reason
	return true
}

func newGatewayVisionJudge(client func(context.Context) (visionRunClient, error)) visionJudge {
	if client == nil {
		return nil
	}
	return &gatewayVisionJudge{client: client}
}

type visionFinding struct {
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Confidence  float64 `json:"confidence"`
	Severity    string  `json:"severity"`
}

type visionJudgment struct {
	Findings []visionFinding `json:"findings"`
}

func (j *gatewayVisionJudge) Check(ctx context.Context, screenshot []byte, width, height int, pageURL, screenshotRef string) []manifestvalidation.Finding {
	if len(screenshot) == 0 {
		return nil
	}
	if circuit := visionCircuitFrom(ctx); circuit.isOpen() {
		return nil
	}
	client, err := j.client(ctx)
	if err != nil {
		return []manifestvalidation.Finding{visionUnavailableFinding(pageURL, visionRunFailureReason(ctx, "ai-gateway is unavailable"))}
	}
	request := connect.NewRequest(&inferencev1.RunRequest{
		Role:            visionCriticRole,
		Profile:         sharedv1.Profile_PROFILE_LOCAL_ONLY,
		PrivacyClass:    sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL,
		Instruction:     "Inspect this screenshot of a locally served Vrooli scenario page at the supplied viewport. Report only visible UX defects that can be identified from this image. Choose each category from: navigation_crowding, touch_target, typography_hierarchy, spacing_layout, safe_area_chrome, color_contrast, visual_polish, functionality_regression, design_system. Look for crowded or inconsistent navigation, overly small touch targets, unsafe edge spacing, poor text hierarchy, excessive or inconsistent typography, low contrast, awkward spacing, visual imbalance, and controls that appear unclear. Do not infer hidden behavior, personal data, or defects that cannot be seen. Return an empty findings list when the page looks sound. Confidence must be 0..1.",
		SchemaJson:      `{"type":"object","required":["findings"],"properties":{"findings":{"type":"array","items":{"type":"object","required":["category","description","confidence","severity"],"properties":{"category":{"type":"string"},"description":{"type":"string"},"confidence":{"type":"number"},"severity":{"type":"string"}}}}}}`,
		MaxOutputTokens: 700,
		Attachments: []*sharedv1.Attachment{{
			Modality:  sharedv1.Modality_MODALITY_IMAGE,
			MediaType: "image/png",
			Width:     uint32(width),
			Height:    uint32(height),
			Bytes:     uint64(len(screenshot)),
			Payload:   &sharedv1.Attachment_InlineBytes{InlineBytes: screenshot},
		}},
	})
	response, err := client.Run(ctx, request)
	if err != nil || response == nil || response.Msg == nil {
		return []manifestvalidation.Finding{visionUnavailableFinding(pageURL, visionRunFailureReason(ctx, "local vision judge could not complete"))}
	}
	if response.Msg.GetError() != nil {
		code := strings.TrimPrefix(response.Msg.GetError().GetCode().String(), "INFERENCE_ERROR_CODE_")
		if code == "UNSPECIFIED" || code == "" {
			code = "UNKNOWN"
		}
		reason := summarizeGatewayVisionError(response.Msg.GetError().GetMessage())
		if code != "VALIDATION_FAILED" {
			reason = visionRunFailureReason(ctx, reason)
		}
		return []manifestvalidation.Finding{visionUnavailableFinding(pageURL, "local vision request failed ("+code+": "+reason+")")}
	}
	if !response.Msg.GetValidated() {
		return []manifestvalidation.Finding{visionUnavailableFinding(pageURL, visionRunFailureReason(ctx, "local vision result did not pass gateway validation"))}
	}
	var judgment visionJudgment
	if err := json.Unmarshal([]byte(response.Msg.GetValueJson()), &judgment); err != nil {
		return []manifestvalidation.Finding{visionUnavailableFinding(pageURL, visionRunFailureReason(ctx, "local vision judge returned an unreadable result"))}
	}
	type rankedFinding struct {
		finding    manifestvalidation.Finding
		confidence float64
	}
	ranked := make([]rankedFinding, 0, len(judgment.Findings))
	for _, finding := range judgment.Findings {
		category := strings.TrimSpace(finding.Category)
		description := strings.TrimSpace(finding.Description)
		codeCategory := normalizeUXCategory(category)
		if category == "" || codeCategory == "" || description == "" || finding.Confidence < 0.65 || finding.Confidence > 1 {
			continue
		}
		severity := manifestvalidation.SeverityWarning
		if strings.EqualFold(finding.Severity, "error") && finding.Confidence >= 0.9 {
			severity = manifestvalidation.SeverityError
		}
		location := pageURL
		pageContext := ""
		if strings.TrimSpace(screenshotRef) != "" {
			location = strings.TrimSpace(screenshotRef)
			pageContext = " on served page " + pageURL
		}
		ranked = append(ranked, rankedFinding{finding: manifestvalidation.Finding{
			Severity:   severity,
			Code:       "vision_ux_" + codeCategory,
			Location:   location,
			Message:    fmt.Sprintf("Vision UX critic (confidence %.2f): %s%s", finding.Confidence, description, pageContext),
			Suggestion: "Review the linked served screenshot and correct the visible issue if confirmed.",
		}, confidence: finding.Confidence})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].finding.Severity != ranked[j].finding.Severity {
			return ranked[i].finding.Severity == manifestvalidation.SeverityError
		}
		return ranked[i].confidence > ranked[j].confidence
	})
	out := make([]manifestvalidation.Finding, 0, len(ranked))
	for _, item := range ranked {
		out = append(out, item.finding)
	}
	return out
}

func visionRunFailureReason(ctx context.Context, reason string) string {
	if circuit := visionCircuitFrom(ctx); circuit.trip(reason) {
		return reason + "; remaining page requests were skipped for this UI Health run"
	}
	return reason
}

func summarizeGatewayVisionError(message string) string {
	message = strings.ToLower(message)
	switch {
	case strings.Contains(message, "not declared"):
		return "gateway role is not declared"
	case strings.Contains(message, "not pulled"), strings.Contains(message, "not installed"):
		return "local model is not installed"
	case strings.Contains(message, "profile"), strings.Contains(message, "local candidate"):
		return "local route is not eligible for this request"
	case strings.Contains(message, "unavailable"), strings.Contains(message, "unreachable"):
		return "local provider is unavailable"
	default:
		return "gateway could not route the request"
	}
}

func visionUnavailableFinding(pageURL, reason string) manifestvalidation.Finding {
	return manifestvalidation.Finding{
		Severity: manifestvalidation.SeverityInfo, Code: "vision_ux_judge_skipped", Location: pageURL,
		Message:    "Local vision UX critique was skipped: " + reason + ". Structural and geometric findings remain available.",
		Suggestion: "Check the local AI Gateway vision.default route and rerun the served UI Health phase.",
	}
}

func normalizeUXCategory(value string) string {
	value = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), "_", " "))
	switch {
	case strings.Contains(value, "navigation"), strings.Contains(value, "nav "), strings.Contains(value, "crowd"), strings.Contains(value, "clutter"):
		return "navigation_crowding"
	case strings.Contains(value, "touch"), strings.Contains(value, "tap"), strings.Contains(value, "target"), strings.Contains(value, "button"):
		return "touch_target"
	case strings.Contains(value, "typograph"), strings.Contains(value, "font"), strings.Contains(value, "text size"), strings.Contains(value, "hierarchy"):
		return "typography_hierarchy"
	case strings.Contains(value, "safe area"), strings.Contains(value, "status bar"), strings.Contains(value, "viewport"), strings.Contains(value, "chrome"):
		return "safe_area_chrome"
	case strings.Contains(value, "contrast"), strings.Contains(value, "color"), strings.Contains(value, "colour"):
		return "color_contrast"
	case strings.Contains(value, "spacing"), strings.Contains(value, "layout"), strings.Contains(value, "margin"), strings.Contains(value, "padding"):
		return "spacing_layout"
	case strings.Contains(value, "function"), strings.Contains(value, "missing feature"), strings.Contains(value, "regression"):
		return "functionality_regression"
	case strings.Contains(value, "design system"), strings.Contains(value, "component library"), strings.Contains(value, "rcl"):
		return "design_system"
	default:
		return "visual_polish"
	}
}
