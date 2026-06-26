package visualhealth

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	visualpb "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth"
)

const (
	overflowSlackPx     = 8
	overlayAreaFraction = 0.75
	mobileWidthPx       = 480
	minMobileFontPx     = 16
)

type layoutSnapshot struct {
	ViewportWidth  float64
	ViewportHeight float64
	DocumentWidth  float64
	DocumentHeight float64
	Elements       []layoutElement
}

type layoutElement struct {
	Selector        string
	Tag             string
	Role            string
	Text            string
	Type            string
	X               float64
	Y               float64
	Width           float64
	Height          float64
	ClientWidth     float64
	ClientHeight    float64
	ScrollWidth     float64
	ScrollHeight    float64
	FontSize        float64
	Position        string
	OverflowX       string
	OverflowY       string
	PointerEvents   string
	Visibility      string
	Display         string
	Opacity         float64
	AriaModal       bool
	Interactive     bool
	ContentEditable bool
}

func analyzeLayout(step *visualpb.VisualStepArtifact) ([]*visualpb.VisualFinding, []*visualpb.VisualMetric) {
	if step == nil || strings.TrimSpace(step.GetLayoutJson()) == "" {
		return nil, nil
	}
	snap, err := parseLayoutSnapshot(step)
	if err != nil {
		return []*visualpb.VisualFinding{{
			Code:        "visual_layout_unreadable",
			Severity:    severityWarning,
			Category:    categoryLayout,
			Message:     "layout snapshot could not be decoded for visual health analysis",
			Location:    locationFor(step),
			Evidence:    err.Error(),
			Remediation: "Verify the layout_json artifact is a JSON object captured by browser-automation-studio.",
			StepId:      step.GetStepId(),
		}}, nil
	}

	var findings []*visualpb.VisualFinding
	var metrics []*visualpb.VisualMetric
	if snap.ViewportWidth > 0 && snap.DocumentWidth > 0 {
		overflow := math.Max(0, snap.DocumentWidth-snap.ViewportWidth)
		metrics = append(metrics, metric("visual_horizontal_overflow_px", overflow))
		if overflow > overflowSlackPx {
			findings = append(findings, &visualpb.VisualFinding{
				Code:        "visual_viewport_overflow",
				Severity:    severityError,
				Category:    categoryLayout,
				Message:     "layout width exceeds the visible viewport",
				Location:    locationFor(step),
				Evidence:    fmt.Sprintf("document width %.0fpx exceeds viewport width %.0fpx", snap.DocumentWidth, snap.ViewportWidth),
				Remediation: "Constrain page width to the viewport and move wide content into an intentional scroll container.",
				StepId:      step.GetStepId(),
				Metrics: []*visualpb.VisualMetric{
					metric("overflow_px", overflow),
				},
			})
		}
	}

	for _, el := range snap.Elements {
		if !el.visible() {
			continue
		}
		if el.isInteractive() && snap.hasViewport() && el.offscreen(snap) {
			findings = append(findings, &visualpb.VisualFinding{
				Code:        "visual_offscreen_interactive",
				Severity:    severityError,
				Category:    categoryLayout,
				Message:     "interactive element is outside the visible viewport",
				Location:    firstNonEmpty(el.Selector, locationFor(step)),
				Evidence:    el.describeRect(),
				Remediation: "Keep interactive controls inside the visible viewport or inside reachable scroll containers.",
				StepId:      step.GetStepId(),
			})
		}
		if el.hasClippedText() {
			findings = append(findings, &visualpb.VisualFinding{
				Code:        "visual_text_clipped",
				Severity:    severityWarning,
				Category:    categoryLayout,
				Message:     "text element appears clipped by its box",
				Location:    firstNonEmpty(el.Selector, locationFor(step)),
				Evidence:    el.describeOverflow(),
				Remediation: "Allow text containers to grow, wrap, or scroll instead of clipping meaningful text.",
				StepId:      step.GetStepId(),
			})
		}
		if snap.hasViewport() && el.blocksViewport(snap) {
			findings = append(findings, &visualpb.VisualFinding{
				Code:        "visual_blocking_overlay",
				Severity:    severityError,
				Category:    categoryLayout,
				Message:     "large pointer-blocking overlay covers the viewport without modal semantics",
				Location:    firstNonEmpty(el.Selector, locationFor(step)),
				Evidence:    el.describeRect(),
				Remediation: "Remove unexpected full-screen overlays or mark intentional modals with dialog semantics.",
				StepId:      step.GetStepId(),
			})
		}
		if snap.mobileLike() && el.isTextEntry() && el.FontSize > 0 && el.FontSize < minMobileFontPx {
			findings = append(findings, &visualpb.VisualFinding{
				Code:        "visual_focus_zoom_risk",
				Severity:    severityWarning,
				Category:    categoryFocus,
				Message:     "mobile text-entry control uses a font size that can trigger browser zoom",
				Location:    firstNonEmpty(el.Selector, locationFor(step)),
				Evidence:    fmt.Sprintf("font size %.1fpx < %dpx", el.FontSize, minMobileFontPx),
				Remediation: "Use at least 16px computed font size for mobile text-entry controls to avoid browser zoom.",
				StepId:      step.GetStepId(),
			})
		}
	}
	return findings, metrics
}

func parseLayoutSnapshot(step *visualpb.VisualStepArtifact) (layoutSnapshot, error) {
	var raw map[string]any
	if err := json.Unmarshal([]byte(step.GetLayoutJson()), &raw); err != nil {
		return layoutSnapshot{}, err
	}
	snap := layoutSnapshot{}
	if vp := mapValue(raw, "viewport"); vp != nil {
		snap.ViewportWidth = firstNumber(vp, "width", "clientWidth", "innerWidth")
		snap.ViewportHeight = firstNumber(vp, "height", "clientHeight", "innerHeight")
	}
	if step.GetViewport() != nil {
		if step.GetViewport().GetWidth() > 0 {
			snap.ViewportWidth = float64(step.GetViewport().GetWidth())
		}
		if step.GetViewport().GetHeight() > 0 {
			snap.ViewportHeight = float64(step.GetViewport().GetHeight())
		}
	}
	if snap.ViewportWidth == 0 {
		snap.ViewportWidth = firstNumber(raw, "viewportWidth", "clientWidth", "innerWidth")
	}
	if snap.ViewportHeight == 0 {
		snap.ViewportHeight = firstNumber(raw, "viewportHeight", "clientHeight", "innerHeight")
	}
	if doc := firstMap(raw, "document", "body", "root"); doc != nil {
		snap.DocumentWidth = firstNumber(doc, "scrollWidth", "width", "clientWidth")
		snap.DocumentHeight = firstNumber(doc, "scrollHeight", "height", "clientHeight")
	}
	if snap.DocumentWidth == 0 {
		snap.DocumentWidth = firstNumber(raw, "scrollWidth", "documentWidth")
	}
	if snap.DocumentHeight == 0 {
		snap.DocumentHeight = firstNumber(raw, "scrollHeight", "documentHeight")
	}
	for _, item := range arrayValue(raw, "elements") {
		if m, ok := item.(map[string]any); ok {
			snap.Elements = append(snap.Elements, parseLayoutElement(m))
		}
	}
	return snap, nil
}

func parseLayoutElement(m map[string]any) layoutElement {
	rect := firstMap(m, "rect", "bounds", "boundingClientRect")
	if rect == nil {
		rect = m
	}
	return layoutElement{
		Selector:        firstString(m, "selector", "id", "path"),
		Tag:             strings.ToLower(firstString(m, "tag", "tagName", "nodeName")),
		Role:            strings.ToLower(firstString(m, "role")),
		Text:            strings.TrimSpace(firstString(m, "text", "innerText", "label")),
		Type:            strings.ToLower(firstString(m, "type")),
		X:               firstNumber(rect, "x", "left"),
		Y:               firstNumber(rect, "y", "top"),
		Width:           firstNumber(rect, "width", "clientWidth"),
		Height:          firstNumber(rect, "height", "clientHeight"),
		ClientWidth:     firstNumber(m, "clientWidth"),
		ClientHeight:    firstNumber(m, "clientHeight"),
		ScrollWidth:     firstNumber(m, "scrollWidth"),
		ScrollHeight:    firstNumber(m, "scrollHeight"),
		FontSize:        firstNumber(m, "fontSize", "fontSizePx"),
		Position:        strings.ToLower(firstString(m, "position")),
		OverflowX:       strings.ToLower(firstString(m, "overflowX", "overflow")),
		OverflowY:       strings.ToLower(firstString(m, "overflowY", "overflow")),
		PointerEvents:   strings.ToLower(firstString(m, "pointerEvents")),
		Visibility:      strings.ToLower(firstString(m, "visibility")),
		Display:         strings.ToLower(firstString(m, "display")),
		Opacity:         numberDefault(m, 1, "opacity"),
		AriaModal:       boolValue(m, "ariaModal", "aria-modal"),
		Interactive:     boolValue(m, "interactive", "focusable"),
		ContentEditable: boolValue(m, "contentEditable", "isContentEditable"),
	}
}

func (s layoutSnapshot) hasViewport() bool { return s.ViewportWidth > 0 && s.ViewportHeight > 0 }
func (s layoutSnapshot) mobileLike() bool {
	return s.ViewportWidth > 0 && s.ViewportWidth <= mobileWidthPx
}

func (e layoutElement) visible() bool {
	return e.Display != "none" && e.Visibility != "hidden" && e.Opacity > 0 && e.Width > 0 && e.Height > 0
}

func (e layoutElement) isInteractive() bool {
	if e.Interactive {
		return true
	}
	switch e.Tag {
	case "button", "a", "input", "select", "textarea", "summary":
		return true
	}
	switch e.Role {
	case "button", "link", "checkbox", "combobox", "menuitem", "radio", "searchbox", "slider", "switch", "tab", "textbox":
		return true
	}
	return e.ContentEditable
}

func (e layoutElement) isTextEntry() bool {
	if e.ContentEditable || e.Tag == "textarea" || e.Role == "textbox" || e.Role == "searchbox" {
		return true
	}
	if e.Tag != "input" {
		return false
	}
	switch e.Type {
	case "", "text", "search", "email", "password", "url", "tel", "number":
		return true
	default:
		return false
	}
}

func (e layoutElement) offscreen(s layoutSnapshot) bool {
	return e.X+e.Width < 0 || e.Y+e.Height < 0 || e.X > s.ViewportWidth || e.Y > s.ViewportHeight
}

func (e layoutElement) hasClippedText() bool {
	if strings.TrimSpace(e.Text) == "" {
		return false
	}
	xClipped := e.ScrollWidth > 0 && e.ClientWidth > 0 && e.ScrollWidth > e.ClientWidth+1 && clips(e.OverflowX)
	yClipped := e.ScrollHeight > 0 && e.ClientHeight > 0 && e.ScrollHeight > e.ClientHeight+1 && clips(e.OverflowY)
	return xClipped || yClipped
}

func (e layoutElement) blocksViewport(s layoutSnapshot) bool {
	if e.Role == "dialog" || e.Role == "alertdialog" || e.AriaModal {
		return false
	}
	if e.PointerEvents == "none" {
		return false
	}
	if e.Position != "fixed" && e.Position != "absolute" {
		return false
	}
	viewportArea := s.ViewportWidth * s.ViewportHeight
	if viewportArea <= 0 {
		return false
	}
	return (e.Width*e.Height)/viewportArea >= overlayAreaFraction
}

func (e layoutElement) describeRect() string {
	return fmt.Sprintf("%s rect=(%.0f,%.0f %.0fx%.0f)", firstNonEmpty(e.Selector, e.Tag, "element"), e.X, e.Y, e.Width, e.Height)
}

func (e layoutElement) describeOverflow() string {
	return fmt.Sprintf("%s client=(%.0fx%.0f) scroll=(%.0fx%.0f)", firstNonEmpty(e.Selector, e.Tag, "element"), e.ClientWidth, e.ClientHeight, e.ScrollWidth, e.ScrollHeight)
}

func clips(v string) bool { return v == "hidden" || v == "clip" }

func firstMap(m map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		if v := mapValue(m, key); v != nil {
			return v
		}
	}
	return nil
}

func mapValue(m map[string]any, key string) map[string]any {
	if v, ok := m[key].(map[string]any); ok {
		return v
	}
	return nil
}

func arrayValue(m map[string]any, key string) []any {
	if v, ok := m[key].([]any); ok {
		return v
	}
	return nil
}

func firstNumber(m map[string]any, keys ...string) float64 {
	return numberDefault(m, 0, keys...)
}

func numberDefault(m map[string]any, fallback float64, keys ...string) float64 {
	for _, key := range keys {
		switch v := m[key].(type) {
		case float64:
			return v
		case int:
			return float64(v)
		case json.Number:
			if n, err := v.Float64(); err == nil {
				return n
			}
		case string:
			var n float64
			if _, err := fmt.Sscanf(strings.TrimSuffix(v, "px"), "%f", &n); err == nil {
				return n
			}
		}
	}
	return fallback
}

func firstString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if s, ok := m[key].(string); ok {
			return s
		}
	}
	return ""
}

func boolValue(m map[string]any, keys ...string) bool {
	for _, key := range keys {
		switch v := m[key].(type) {
		case bool:
			return v
		case string:
			return strings.EqualFold(v, "true")
		}
	}
	return false
}
