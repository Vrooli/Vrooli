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
	Tokens         []string
	Chrome         chromeIntent
	SafeAreaInsets safeAreaInsets
}

type chromeIntent struct {
	ThemeColor     string
	StatusBarColor string
	SafeAreaColor  string
	StatusBarStyle string
}

type safeAreaInsets struct {
	Top    float64
	Right  float64
	Bottom float64
	Left   float64
}

func (c chromeIntent) empty() bool {
	return strings.TrimSpace(c.ThemeColor) == "" &&
		strings.TrimSpace(c.StatusBarColor) == "" &&
		strings.TrimSpace(c.SafeAreaColor) == ""
}

type layoutElement struct {
	Selector          string
	ParentSelector    string
	NodeType          string
	Tag               string
	Role              string
	Text              string
	Type              string
	X                 float64
	Y                 float64
	Width             float64
	Height            float64
	ClientWidth       float64
	ClientHeight      float64
	ScrollWidth       float64
	ScrollHeight      float64
	FontSize          float64
	Color             string
	BackgroundColor   string
	FontFamily        string
	Position          string
	OverflowX         string
	OverflowY         string
	PointerEvents     string
	Visibility        string
	Display           string
	TextOverflow      string
	WhiteSpace        string
	InlineIntent      bool
	Opacity           float64
	AriaModal         bool
	Interactive       bool
	ContentEditable   bool
	InScrollContainer bool
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
		if el.isInteractive() && !el.InScrollContainer && snap.hasViewport() && el.offscreen(snap) {
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
		if snap.hasViewport() && !el.InScrollContainer && el.overflowsViewport(snap) {
			findings = append(findings, &visualpb.VisualFinding{
				Code:        "visual_viewport_overflow",
				Severity:    severityError,
				Category:    categoryLayout,
				Message:     "visible element extends beyond the captured viewport",
				Location:    firstNonEmpty(el.Selector, locationFor(step)),
				Evidence:    el.describeRect(),
				Remediation: "Constrain page width to the viewport and move wide content into an intentional scroll container.",
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
		if edge := el.unsafeSafeAreaEdge(snap); edge != "" {
			findings = append(findings, &visualpb.VisualFinding{
				Code:        "visual_unsafe_edge_tap_zone",
				Severity:    severityError,
				Category:    categoryLayout,
				Message:     "interactive element overlaps an unsafe edge or notch zone",
				Location:    firstNonEmpty(el.Selector, locationFor(step)),
				Evidence:    fmt.Sprintf("%s overlaps %s safe area", el.describeRect(), edge),
				Remediation: "Move interactive controls out of unsafe edge and notch zones or pad the layout with env(safe-area-inset-*).",
				StepId:      step.GetStepId(),
			})
		}
	}
	findings = append(findings, headingNotBlockFindings(step, snap)...)
	findings = append(findings, tokenFallbackFindings(step, snap)...)
	findings = append(findings, adjacentTextFindings(step, snap)...)
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
	snap.Chrome = parseChromeIntent(raw)
	snap.SafeAreaInsets = parseSafeAreaInsets(raw)
	snap.Tokens = stringArray(raw, "tokens", "tokenSet", "token_set")
	for _, item := range arrayValue(raw, "elements") {
		if m, ok := item.(map[string]any); ok {
			snap.Elements = append(snap.Elements, parseLayoutElement(m))
		}
	}
	if len(snap.Elements) == 0 {
		if tree := firstMap(raw, "tree"); tree != nil {
			appendDOMTreeElements(&snap, tree, "")
		} else if _, hasChildren := raw["children"]; hasChildren {
			appendDOMTreeElements(&snap, raw, "")
		}
	}
	return snap, nil
}

func appendDOMTreeElements(snap *layoutSnapshot, node map[string]any, parentSelector string) {
	if snap == nil || node == nil {
		return
	}
	computed := firstMap(node, "computed")
	selector := firstString(node, "selector", "id")
	tag := strings.ToLower(firstString(node, "tag", "tagName", "nodeName"))
	role := strings.ToLower(firstString(node, "role"))
	if role == "" {
		switch tag {
		case "h1", "h2", "h3", "h4", "h5", "h6":
			role = "heading"
		case "a":
			role = "link"
		}
	}
	if firstMap(node, "rect", "bounds", "boundingClientRect") != nil || selector != "" || strings.TrimSpace(firstString(node, "text", "innerText")) != "" {
		m := map[string]any{
			"selector":          selector,
			"parentSelector":    parentSelector,
			"nodeType":          "element",
			"tag":               tag,
			"role":              role,
			"text":              firstString(node, "text", "innerText"),
			"rect":              firstMap(node, "rect", "bounds", "boundingClientRect"),
			"clientWidth":       firstNumber(node, "clientWidth"),
			"clientHeight":      firstNumber(node, "clientHeight"),
			"scrollWidth":       firstNumber(node, "scrollWidth"),
			"scrollHeight":      firstNumber(node, "scrollHeight"),
			"inScrollContainer": boolValue(node, "inScrollContainer", "insideScrollContainer"),
			"display":           firstString(computed, "display"),
			"color":             firstString(computed, "color"),
			"backgroundColor":   firstString(computed, "backgroundColor"),
			"fontFamily":        firstString(computed, "fontFamily", "font"),
			"overflowX":         firstString(computed, "overflowX", "overflow"),
			"overflowY":         firstString(computed, "overflowY", "overflow"),
			"textOverflow":      firstString(computed, "textOverflow"),
			"whiteSpace":        firstString(computed, "whiteSpace"),
		}
		snap.Elements = append(snap.Elements, parseLayoutElement(m))
	}
	for _, item := range arrayValue(node, "children") {
		if child, ok := item.(map[string]any); ok {
			appendDOMTreeElements(snap, child, selector)
		}
	}
}

func parseChromeIntent(raw map[string]any) chromeIntent {
	chrome := firstMap(raw, "chrome", "declaredChrome", "theme")
	safe := firstMap(raw, "safeArea", "safe_area", "safeAreaIntent")
	if chrome == nil {
		chrome = raw
	}
	out := chromeIntent{
		ThemeColor:     firstString(chrome, "themeColor", "theme_color", "declaredThemeColor"),
		StatusBarColor: firstString(chrome, "statusBarColor", "status_bar_color", "declaredStatusBarColor"),
		SafeAreaColor:  firstString(chrome, "safeAreaColor", "safe_area_color", "declaredSafeAreaColor"),
		StatusBarStyle: firstString(chrome, "statusBarStyle", "status_bar_style", "appleMobileWebAppStatusBarStyle"),
	}
	if safe != nil && out.SafeAreaColor == "" {
		out.SafeAreaColor = firstString(safe, "color", "safeAreaColor", "backgroundColor", "background")
	}
	return out
}

func parseSafeAreaInsets(raw map[string]any) safeAreaInsets {
	safe := firstMap(raw, "safeArea", "safe_area", "safeAreaInsets", "safe_area_insets")
	if safe == nil {
		safe = raw
	}
	return safeAreaInsets{
		Top:    firstNumber(safe, "top", "safeAreaTop", "safe_area_top"),
		Right:  firstNumber(safe, "right", "safeAreaRight", "safe_area_right"),
		Bottom: firstNumber(safe, "bottom", "safeAreaBottom", "safe_area_bottom"),
		Left:   firstNumber(safe, "left", "safeAreaLeft", "safe_area_left"),
	}
}

func parseLayoutElement(m map[string]any) layoutElement {
	rect := firstMap(m, "rect", "bounds", "boundingClientRect")
	if rect == nil {
		rect = m
	}
	return layoutElement{
		Selector:          firstString(m, "selector", "id", "path"),
		ParentSelector:    firstString(m, "parentSelector", "parent_selector", "parent"),
		NodeType:          strings.ToLower(firstString(m, "nodeType", "node_type")),
		Tag:               strings.ToLower(firstString(m, "tag", "tagName", "nodeName")),
		Role:              strings.ToLower(firstString(m, "role")),
		Text:              strings.TrimSpace(firstString(m, "text", "innerText", "label")),
		Type:              strings.ToLower(firstString(m, "type")),
		X:                 firstNumber(rect, "x", "left"),
		Y:                 firstNumber(rect, "y", "top"),
		Width:             firstNumber(rect, "width", "clientWidth"),
		Height:            firstNumber(rect, "height", "clientHeight"),
		ClientWidth:       firstNumber(m, "clientWidth"),
		ClientHeight:      firstNumber(m, "clientHeight"),
		ScrollWidth:       firstNumber(m, "scrollWidth"),
		ScrollHeight:      firstNumber(m, "scrollHeight"),
		FontSize:          firstNumber(m, "fontSize", "fontSizePx"),
		Color:             firstString(m, "color", "foregroundColor", "textColor"),
		BackgroundColor:   firstString(m, "backgroundColor", "background_color"),
		FontFamily:        firstString(m, "fontFamily", "font_family"),
		Position:          strings.ToLower(firstString(m, "position")),
		OverflowX:         strings.ToLower(firstString(m, "overflowX", "overflow")),
		OverflowY:         strings.ToLower(firstString(m, "overflowY", "overflow")),
		PointerEvents:     strings.ToLower(firstString(m, "pointerEvents")),
		Visibility:        strings.ToLower(firstString(m, "visibility")),
		Display:           strings.ToLower(firstString(m, "display")),
		TextOverflow:      strings.ToLower(firstString(m, "textOverflow", "text_overflow")),
		WhiteSpace:        strings.ToLower(firstString(m, "whiteSpace", "white_space")),
		InlineIntent:      boolValue(m, "inlineIntent", "inline_intent"),
		Opacity:           numberDefault(m, 1, "opacity"),
		AriaModal:         boolValue(m, "ariaModal", "aria-modal"),
		Interactive:       boolValue(m, "interactive", "focusable"),
		ContentEditable:   boolValue(m, "contentEditable", "isContentEditable"),
		InScrollContainer: boolValue(m, "inScrollContainer", "insideScrollContainer"),
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
	return (xClipped || yClipped) && (e.TextOverflow == "" || e.TextOverflow == "ellipsis" || e.WhiteSpace == "nowrap")
}

func (e layoutElement) overflowsViewport(s layoutSnapshot) bool {
	return e.X < -overflowSlackPx || e.X+e.Width > s.ViewportWidth+overflowSlackPx
}

func (e layoutElement) isTextNode() bool {
	return e.NodeType == "text" || e.Tag == "#text"
}

func (e layoutElement) inlineByIntent() bool {
	return e.InlineIntent || e.Display == "inline" || e.Display == "inline-block"
}

func headingNotBlockFindings(step *visualpb.VisualStepArtifact, snap layoutSnapshot) []*visualpb.VisualFinding {
	var findings []*visualpb.VisualFinding
	for _, el := range snap.Elements {
		if !el.visible() || el.Role != "heading" || (el.Display != "inline" && el.Display != "inline-block") {
			continue
		}
		findings = append(findings, &visualpb.VisualFinding{
			Code:        "visual_heading_not_block",
			Severity:    severityError,
			Category:    categoryLayout,
			Message:     "the heading is not a block box, so its margins are inert and it will run into the next element.",
			Location:    firstNonEmpty(el.Selector, locationFor(step)),
			Evidence:    fmt.Sprintf("role=heading display=%s", el.Display),
			Remediation: "Make the heading a block box before applying block-flow margins.",
			StepId:      step.GetStepId(),
		})
	}
	return findings
}

func tokenFallbackFindings(step *visualpb.VisualStepArtifact, snap layoutSnapshot) []*visualpb.VisualFinding {
	if len(snap.Tokens) == 0 {
		return nil
	}
	var findings []*visualpb.VisualFinding
	for _, el := range snap.Elements {
		if !el.visible() {
			continue
		}
		for property, value := range map[string]string{"color": el.Color, "backgroundColor": el.BackgroundColor, "fontFamily": el.FontFamily} {
			value = strings.TrimSpace(value)
			if value == "" || strings.Contains(strings.ToLower(value), "var(") || tokenValuePresent(snap.Tokens, value) {
				continue
			}
			findings = append(findings, &visualpb.VisualFinding{
				Code:        "visual_token_fallback_used",
				Severity:    severityWarning,
				Category:    categoryDOM,
				Message:     "computed " + property + " is not represented by a scenario token",
				Location:    firstNonEmpty(el.Selector, locationFor(step)),
				Evidence:    property + "=" + value,
				Remediation: "Replace the literal visual value with a scenario design token when the value is not intentional.",
				StepId:      step.GetStepId(),
			})
		}
	}
	return findings
}

func adjacentTextFindings(step *visualpb.VisualStepArtifact, snap layoutSnapshot) []*visualpb.VisualFinding {
	var findings []*visualpb.VisualFinding
	for i := 1; i < len(snap.Elements); i++ {
		left, right := snap.Elements[i-1], snap.Elements[i]
		if !left.visible() || !right.visible() || !left.isTextNode() || !right.isTextNode() || left.ParentSelector == "" || left.ParentSelector != right.ParentSelector || left.inlineByIntent() || right.inlineByIntent() {
			continue
		}
		if left.Y+left.Height <= right.Y || right.Y+right.Height <= left.Y {
			continue
		}
		findings = append(findings, &visualpb.VisualFinding{
			Code:        "visual_adjacent_text_collision",
			Severity:    severityWarning,
			Category:    categoryLayout,
			Message:     "adjacent text nodes share a line box without inline intent",
			Location:    firstNonEmpty(left.Selector, right.Selector, locationFor(step)),
			Evidence:    fmt.Sprintf("siblings %q and %q share parent %s", left.Text, right.Text, left.ParentSelector),
			Remediation: "Keep adjacent text inline by intent or place each block on its own line.",
			StepId:      step.GetStepId(),
		})
	}
	return findings
}

func tokenValuePresent(tokens []string, value string) bool {
	want := normalizeTokenValue(value)
	for _, token := range tokens {
		if want == normalizeTokenValue(token) {
			return true
		}
	}
	return false
}

func normalizeTokenValue(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
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

func (e layoutElement) unsafeSafeAreaEdge(s layoutSnapshot) string {
	if !e.isInteractive() || !s.hasViewport() || !s.SafeAreaInsets.any() {
		return ""
	}
	switch {
	case s.SafeAreaInsets.Top > 0 && e.Y < s.SafeAreaInsets.Top:
		return "top"
	case s.SafeAreaInsets.Bottom > 0 && e.Y+e.Height > s.ViewportHeight-s.SafeAreaInsets.Bottom:
		return "bottom"
	case s.SafeAreaInsets.Left > 0 && e.X < s.SafeAreaInsets.Left:
		return "left"
	case s.SafeAreaInsets.Right > 0 && e.X+e.Width > s.ViewportWidth-s.SafeAreaInsets.Right:
		return "right"
	default:
		return ""
	}
}

func (s safeAreaInsets) any() bool {
	return s.Top > 0 || s.Right > 0 || s.Bottom > 0 || s.Left > 0
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

func stringArray(m map[string]any, keys ...string) []string {
	for _, key := range keys {
		values, ok := m[key].([]any)
		if !ok {
			continue
		}
		out := make([]string, 0, len(values))
		for _, value := range values {
			if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
				out = append(out, text)
			}
		}
		return out
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
