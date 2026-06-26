package visualhealth

import (
	"sort"
	"strings"

	"ui-health/internal/visualhealth/pixel"

	visualpb "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth"
)

type Analyzer struct {
	thresholds pixel.Thresholds
}

func NewAnalyzer(thresholds pixel.Thresholds) Analyzer {
	if thresholds.GridSize <= 0 {
		thresholds = pixel.DefaultThresholds()
	}
	return Analyzer{thresholds: thresholds}
}

func DefaultAnalyzer() Analyzer {
	return NewAnalyzer(pixel.ThresholdsFromEnv())
}

func (a Analyzer) Analyze(req *visualpb.AnalyzeArtifactsRequest) *visualpb.AnalyzeArtifactsResponse {
	if req == nil {
		req = &visualpb.AnalyzeArtifactsRequest{}
	}
	resp := &visualpb.AnalyzeArtifactsResponse{
		Scenario: req.GetScenario(),
		RunId:    req.GetRunId(),
		Verdict:  "passed",
	}
	for _, step := range req.GetSteps() {
		verdict := a.analyzeStep(step)
		if hasError(verdict.Findings) {
			verdict.Status = "failed"
			resp.Verdict = "failed"
		} else if verdict.Status == "" {
			verdict.Status = "passed"
		}
		resp.Steps = append(resp.Steps, verdict)
		resp.Findings = append(resp.Findings, verdict.Findings...)
	}
	if len(req.GetSteps()) == 0 {
		resp.Verdict = "skipped"
	}
	return resp
}

func (a Analyzer) Compare(req *visualpb.CompareArtifactsRequest) *visualpb.CompareArtifactsResponse {
	if req == nil {
		return &visualpb.CompareArtifactsResponse{}
	}
	base := indexCompareArtifacts(req.GetBase())
	cur := indexCompareArtifacts(req.GetCurrent())
	pages := sortedUnion(base, cur)
	out := &visualpb.CompareArtifactsResponse{Deltas: make([]*visualpb.VisualDelta, 0, len(pages))}
	for _, page := range pages {
		baseArt, inBase := base[page]
		curArt, inCur := cur[page]
		label := curArt.GetLabel()
		if !inCur {
			label = baseArt.GetLabel()
		}
		delta := &visualpb.VisualDelta{Page: page, Label: label}
		switch {
		case inBase && inCur:
			status, fraction := a.comparePNG(baseArt.GetScreenshotPng(), curArt.GetScreenshotPng())
			delta.Status = status
			delta.ChangedFraction = fraction
		case inCur:
			delta.Status = "added"
		default:
			delta.Status = "removed"
		}
		out.Deltas = append(out.Deltas, delta)
	}
	return out
}

func Rules() []*visualpb.VisualRule {
	return []*visualpb.VisualRule{
		{
			Id:                "visual_pixel_blank",
			Category:          visualpb.VisualCategory_VISUAL_CATEGORY_PIXEL,
			Severity:          visualpb.VisualSeverity_VISUAL_SEVERITY_ERROR,
			RequiredArtifacts: []string{"screenshot_png"},
			Remediation:       "Verify the UI mounts meaningful content and that styles/assets load.",
		},
		{
			Id:                "visual_dom_blank",
			Category:          visualpb.VisualCategory_VISUAL_CATEGORY_DOM,
			Severity:          visualpb.VisualSeverity_VISUAL_SEVERITY_ERROR,
			RequiredArtifacts: []string{"dom_html"},
			Remediation:       "Verify the page rendered meaningful DOM text, labels, alt text, or accessible names.",
		},
		{
			Id:                "visual_broken_asset",
			Category:          visualpb.VisualCategory_VISUAL_CATEGORY_ASSET,
			Severity:          visualpb.VisualSeverity_VISUAL_SEVERITY_ERROR,
			RequiredArtifacts: []string{"network"},
			Remediation:       "Resolve failed image, media, font, or stylesheet requests emitted by the browser.",
		},
		{
			Id:                "visual_stuck_loading",
			Category:          visualpb.VisualCategory_VISUAL_CATEGORY_DOM,
			Severity:          visualpb.VisualSeverity_VISUAL_SEVERITY_ERROR,
			RequiredArtifacts: []string{"dom_html"},
			Remediation:       "Ensure the page leaves loading state and renders meaningful final content before capture.",
		},
		{
			Id:                "visual_viewport_overflow",
			Category:          visualpb.VisualCategory_VISUAL_CATEGORY_LAYOUT,
			Severity:          visualpb.VisualSeverity_VISUAL_SEVERITY_ERROR,
			RequiredArtifacts: []string{"layout_json", "viewport"},
			Remediation:       "Constrain page width to the viewport and move wide content into an intentional scroll container.",
		},
		{
			Id:                "visual_offscreen_interactive",
			Category:          visualpb.VisualCategory_VISUAL_CATEGORY_LAYOUT,
			Severity:          visualpb.VisualSeverity_VISUAL_SEVERITY_ERROR,
			RequiredArtifacts: []string{"layout_json", "viewport"},
			Remediation:       "Keep interactive controls inside the visible viewport or inside reachable scroll containers.",
		},
		{
			Id:                "visual_text_clipped",
			Category:          visualpb.VisualCategory_VISUAL_CATEGORY_LAYOUT,
			Severity:          visualpb.VisualSeverity_VISUAL_SEVERITY_WARNING,
			RequiredArtifacts: []string{"layout_json"},
			Remediation:       "Allow text containers to grow, wrap, or scroll instead of clipping meaningful text.",
		},
		{
			Id:                "visual_focus_zoom_risk",
			Category:          visualpb.VisualCategory_VISUAL_CATEGORY_FOCUS,
			Severity:          visualpb.VisualSeverity_VISUAL_SEVERITY_WARNING,
			RequiredArtifacts: []string{"layout_json", "viewport"},
			Remediation:       "Use at least 16px computed font size for mobile text-entry controls to avoid browser zoom.",
		},
		{
			Id:                "visual_blocking_overlay",
			Category:          visualpb.VisualCategory_VISUAL_CATEGORY_LAYOUT,
			Severity:          visualpb.VisualSeverity_VISUAL_SEVERITY_ERROR,
			RequiredArtifacts: []string{"layout_json", "viewport"},
			Remediation:       "Remove unexpected full-screen overlays or mark intentional modals with dialog semantics.",
		},
	}
}

func (a Analyzer) analyzeStep(step *visualpb.VisualStepArtifact) *visualpb.StepVerdict {
	if step == nil {
		step = &visualpb.VisualStepArtifact{}
	}
	out := &visualpb.StepVerdict{StepId: step.GetStepId()}
	if pngBytes := step.GetScreenshotPng(); len(pngBytes) > 0 {
		health, err := pixel.RenderHealth(pngBytes, a.thresholds)
		if err != nil {
			out.Findings = append(out.Findings, &visualpb.VisualFinding{
				Code:        "visual_pixel_unreadable",
				Severity:    visualpb.VisualSeverity_VISUAL_SEVERITY_WARNING,
				Category:    visualpb.VisualCategory_VISUAL_CATEGORY_PIXEL,
				Message:     "screenshot could not be decoded for visual health analysis",
				Location:    locationFor(step),
				Evidence:    err.Error(),
				Remediation: "Verify the screenshot artifact is a PNG captured by browser-automation-studio.",
				StepId:      step.GetStepId(),
			})
		} else {
			out.Metrics = append(out.Metrics,
				metric("visual_dominant_fraction", health.DominantFraction),
				metric("visual_luminance_variance", health.Variance),
			)
			if health.Broken {
				out.Findings = append(out.Findings, &visualpb.VisualFinding{
					Code:        "visual_pixel_blank",
					Severity:    visualpb.VisualSeverity_VISUAL_SEVERITY_ERROR,
					Category:    visualpb.VisualCategory_VISUAL_CATEGORY_PIXEL,
					Message:     "screenshot appears blank or solid-color",
					Location:    locationFor(step),
					Evidence:    health.Reason,
					Remediation: "Verify the UI mounts meaningful content and that styles/assets load.",
					StepId:      step.GetStepId(),
					Metrics: []*visualpb.VisualMetric{
						metric("dominant_fraction", health.DominantFraction),
						metric("variance", health.Variance),
					},
				})
			}
		}
	}
	if html := step.GetDomHtml(); strings.TrimSpace(html) != "" && domTextBlank(html) {
		out.Findings = append(out.Findings, &visualpb.VisualFinding{
			Code:        "visual_dom_blank",
			Severity:    visualpb.VisualSeverity_VISUAL_SEVERITY_ERROR,
			Category:    visualpb.VisualCategory_VISUAL_CATEGORY_DOM,
			Message:     "DOM snapshot contains no meaningful visible text or accessible labels",
			Location:    locationFor(step),
			Remediation: "Verify the page rendered meaningful DOM text, labels, alt text, or accessible names.",
			StepId:      step.GetStepId(),
		})
	}
	out.Findings = append(out.Findings, domLoadingFindings(step)...)
	out.Findings = append(out.Findings, networkFindings(step)...)
	layoutFindings, layoutMetrics := analyzeLayout(step)
	out.Findings = append(out.Findings, layoutFindings...)
	out.Metrics = append(out.Metrics, layoutMetrics...)
	return out
}

func (a Analyzer) comparePNG(base, cur []byte) (string, float64) {
	if len(base) == 0 || len(cur) == 0 {
		return "changed", 0
	}
	result, err := pixel.Compare(base, cur, a.thresholds)
	if err != nil {
		return "changed", 0
	}
	if result.Identical {
		return "identical", 0
	}
	return "changed", result.ChangedFraction
}

func indexCompareArtifacts(in []*visualpb.CompareArtifact) map[string]*visualpb.CompareArtifact {
	out := make(map[string]*visualpb.CompareArtifact, len(in))
	for _, art := range in {
		if art == nil {
			continue
		}
		page := strings.TrimSpace(art.GetPage())
		if page == "" {
			page = art.GetScreenshotRef().GetRelPath()
		}
		out[page] = art
	}
	return out
}

func sortedUnion(a, b map[string]*visualpb.CompareArtifact) []string {
	set := make(map[string]struct{}, len(a)+len(b))
	for k := range a {
		set[k] = struct{}{}
	}
	for k := range b {
		set[k] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func hasError(findings []*visualpb.VisualFinding) bool {
	for _, f := range findings {
		if f.GetSeverity() == visualpb.VisualSeverity_VISUAL_SEVERITY_ERROR {
			return true
		}
	}
	return false
}

func metric(name string, value float64) *visualpb.VisualMetric {
	return &visualpb.VisualMetric{Name: name, Value: value}
}

func locationFor(step *visualpb.VisualStepArtifact) string {
	if strings.TrimSpace(step.GetUrl()) != "" {
		return step.GetUrl()
	}
	return step.GetStepId()
}
