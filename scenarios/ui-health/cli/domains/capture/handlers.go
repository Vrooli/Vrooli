package capture

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/cli-core/cliapp"

	repocontract "github.com/vrooli/repo-contract-go"
	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/ai"
	aiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/ai/aiconnect"
	basev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
	captureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture/captureconnect"
	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/search"
	searchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/search/search_v1connect"
	visualv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth"
	visualconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth/visualhealth_v1connect"
)

type handlers struct {
	core     *cliapp.ScenarioApp
	resolver *discovery.Resolver
	search   searchconnect.SearchServiceClient
	visual   visualconnect.VisualHealthServiceClient
	ai       aiconnect.AIServiceClient
	http     connect.HTTPClient
}

// CommandGroup exposes capture as the top-level `ui-health capture` command.
// The other ui-health domains are hierarchical Connect groups, but capture is
// intentionally a one-verb operator entry point.
func CommandGroup(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	h := newHandlers(core)
	return cliapp.CommandGroup{
		Title: "capture",
		Commands: []cliapp.Command{{
			Name:        "capture",
			Description: "Resolve a surface through selector, observed-route, or corpus search, then capture it",
			NeedsAPI:    true,
			Args:        commandArgs(),
			RunCtx:      h.run,
		}},
	}
}

func commandArgs() cliapp.ArgSchema {
	return cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "scenario", Description: "Scenario id whose running UI owns the surface", Required: true},
		{Name: "surface", Description: "Exact selector id or natural-language surface request", Required: true},
		{Name: "viewport", Description: "Repeatable viewport: desktop, tablet, or mobile", Values: []string{"desktop", "tablet", "mobile"}},
		{Name: "theme", Description: "Repeatable theme: light or dark", Values: []string{"light", "dark"}},
		{Name: "strict-tokens", Description: "Set every declared CSS custom property to the loud magenta token diagnostic sentinel", Bool: true},
		{Name: "fail-on", Description: "Exit non-zero when a finding reaches INFO, WARNING, or ERROR"},
	}}
}

type candidate struct {
	ID          string  `json:"id"`
	Route       string  `json:"route"`
	DisplayName string  `json:"displayName"`
	FilePath    string  `json:"filePath"`
	Score       float64 `json:"score"`
}

type resolution struct {
	Rung       int         `json:"rung"`
	SurfaceID  string      `json:"surfaceId"`
	Route      string      `json:"route"`
	Selector   string      `json:"selector,omitempty"`
	Candidates []candidate `json:"candidates,omitempty"`
	FollowUp   string      `json:"followUp,omitempty"`
}

type captureArtifact struct {
	Viewport                 string   `json:"viewport"`
	Theme                    string   `json:"theme"`
	ExecutionID              string   `json:"executionId"`
	ScreenshotRef            string   `json:"screenshotRef,omitempty"`
	SnapshotRef              string   `json:"snapshotRef,omitempty"`
	Readiness                any      `json:"readiness,omitempty"`
	StrictTokenSentinelCount int      `json:"strictTokenSentinelCount,omitempty"`
	NonSentinelColorCount    int      `json:"nonSentinelColorCount,omitempty"`
	NonSentinelColorSamples  []string `json:"nonSentinelColorSamples,omitempty"`
}

type finding struct {
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	Selector string `json:"selector,omitempty"`
	Message  string `json:"message"`
	Viewport string `json:"viewport"`
	Theme    string `json:"theme"`
}

type envelope struct {
	Status       string            `json:"status"`
	Scenario     string            `json:"scenario"`
	Surface      string            `json:"surface"`
	Resolved     *resolution       `json:"resolved,omitempty"`
	Artifacts    []captureArtifact `json:"artifacts"`
	Findings     []finding         `json:"findings"`
	Candidates   []candidate       `json:"candidates,omitempty"`
	StrictTokens bool              `json:"strictTokens,omitempty"`
	Sentinel     string            `json:"sentinel,omitempty"`
	Error        string            `json:"error,omitempty"`
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:     core,
		resolver: discovery.NewResolver(discovery.ResolverConfig{}),
		http:     httpClient,
		search:   searchconnect.NewSearchServiceClient(httpClient, baseURL),
		visual:   visualconnect.NewVisualHealthServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) run(ctx cliapp.RunContext) error {
	scenario := strings.TrimSpace(ctx.Flag("scenario"))
	surface := strings.TrimSpace(ctx.Flag("surface"))
	if scenario == "" || surface == "" {
		return errors.New("--scenario and --surface are required")
	}
	viewports := normalizeValues(ctx.FlagValues("viewport"), []string{"desktop", "mobile"})
	themes := normalizeValues(ctx.FlagValues("theme"), []string{"light", "dark"})
	strictTokens := ctx.FlagDeclared("strict-tokens") && ctx.BoolFlag("strict-tokens")
	env := envelope{Status: "partial", Scenario: scenario, Surface: surface, Artifacts: []captureArtifact{}, Findings: []finding{}, StrictTokens: strictTokens}
	if strictTokens {
		env.Sentinel = strictTokenSentinel
	}

	baseURL, err := h.resolver.ResolveScenarioURL(context.Background(), scenario, "UI_PORT")
	if err != nil || strings.TrimSpace(baseURL) == "" {
		env.Status = "unavailable"
		env.Error = fmt.Sprintf("scenario %q is not running; start it with `vrooli scenario start %s`", scenario, scenario)
		return h.render(ctx, env, nil)
	}

	resolved, err := h.resolve(context.Background(), scenario, surface, baseURL)
	if err != nil {
		env.Error = err.Error()
		return h.render(ctx, env, nil)
	}
	if resolved == nil {
		env.Status = "partial"
		env.Candidates = nil
		env.Error = unresolvedRouteReason
		return h.render(ctx, env, nil)
	}
	env.Resolved = resolved
	if resolved.Route == "" {
		env.Status = "partial"
		env.Candidates = resolved.Candidates
		env.Error = firstError(resolved.FollowUp, unresolvedRouteReason)
		return h.render(ctx, env, nil)
	}

	basURL, err := h.resolver.ResolveScenarioURLDefault(context.Background(), "browser-automation-studio")
	if err != nil || strings.TrimSpace(basURL) == "" {
		env.Status = "unavailable"
		env.Error = "browser-automation-studio is not running; start it with `vrooli scenario start browser-automation-studio`"
		return h.render(ctx, env, nil)
	}
	captures := captureconnect.NewCaptureServiceClient(h.http, strings.TrimRight(basURL, "/"))

	for _, viewport := range viewports {
		for _, theme := range themes {
			artifact, finds, captureErr := h.captureOne(context.Background(), captures, baseURL, resolved, viewport, theme, strictTokens)
			if captureErr != nil {
				env.Error = firstError(env.Error, captureErr.Error())
				continue
			}
			env.Artifacts = append(env.Artifacts, artifact)
			env.Findings = append(env.Findings, finds...)
		}
	}
	if len(env.Artifacts) == len(viewports)*len(themes) {
		env.Status = "ok"
	}
	if ctx.JSON() {
		if err := cliapp.PrintJSON(ctx.Stdout(), env); err != nil {
			return err
		}
	} else {
		return h.renderHuman(ctx, env)
	}
	if shouldFail(ctx.Flag("fail-on"), env.Findings) {
		return fmt.Errorf("capture findings reached --fail-on %s", ctx.Flag("fail-on"))
	}
	return nil
}

func (h *handlers) resolve(ctx context.Context, scenario, text, targetURL string) (*resolution, error) {
	// A preview URL is already an observed, deterministic surface. Accepting
	// it keeps bounded story capture from pretending that a story query is a
	// natural-language surface lookup, while still requiring the URL to belong
	// to the scenario selected by the caller.
	if direct, parseErr := url.Parse(text); parseErr == nil && direct.IsAbs() && direct.Host != "" {
		base, baseErr := url.Parse(targetURL)
		if baseErr == nil && direct.Host == base.Host {
			route := direct.EscapedPath()
			if route == "" {
				route = "/"
			}
			if direct.RawQuery != "" {
				route += "?" + direct.RawQuery
			}
			return &resolution{Rung: 1, SurfaceID: "direct:" + route, Route: route}, nil
		}
	}
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		return nil, err
	}
	manifestPath := filepath.Join(root, "scenarios", scenario, "ui", "src", "consts", "selectors.manifest.json")
	var manifest struct {
		Selectors map[string]struct {
			TestID   string `json:"testId"`
			Selector string `json:"selector"`
			Route    string `json:"route"`
		} `json:"selectors"`
	}
	if raw, readErr := os.ReadFile(manifestPath); readErr == nil {
		if json.Unmarshal(raw, &manifest) == nil {
			if item, ok := manifest.Selectors[text]; ok && concreteRoute(item.Route) {
				return &resolution{Rung: 1, SurfaceID: text, Route: item.Route, Selector: item.Selector}, nil
			}
		}
	}
	// A source declaration does not depend on corpus availability or recency.
	// Only exact, unambiguous static routes qualify; no page-name-to-URL guess.
	if route, routeErr := declaredRoute(ctx, root, scenario, text); routeErr == nil && route != "" {
		return &resolution{Rung: 4, SurfaceID: text, Route: route}, nil
	}

	// Rung 2 is a deterministic lookup over the observed route payloads. Force
	// the text leg so a dense top-k result from an unrelated scenario cannot
	// hide an exact observed link/title match.
	observedResp, err := h.search.Search(ctx, connect.NewRequest(&searchv1.SearchRequest{Query: text, Limit: 2500, Mode: searchv1.Mode_MODE_TEXT}))
	if err != nil {
		return nil, fmt.Errorf("selector and declared-route lookup did not resolve; observed-route search failed: %w", err)
	}
	var observed []candidate
	for _, hit := range observedResp.Msg.GetResults() {
		if hit.GetScenario() != scenario || !concreteRoute(hit.GetObservedRoute()) {
			continue
		}
		score := observedScore(text, hit)
		if score <= 0 {
			continue
		}
		observed = append(observed, candidate{ID: hit.GetFilePath(), Route: hit.GetObservedRoute(), DisplayName: hit.GetDisplayName(), FilePath: hit.GetFilePath(), Score: score})
	}
	observed = dedupeCandidates(observed)
	sort.Slice(observed, func(i, j int) bool { return observed[i].Score > observed[j].Score })
	if len(observed) > 0 && observed[0].Score >= 0.8 {
		if len(observed) > 1 && observed[0].Score-observed[1].Score < 0.08 {
			return &resolution{Rung: 2, Candidates: observed[:min(5, len(observed))], FollowUp: disambiguationCommand(scenario, observed[0].ID)}, nil
		}
		return &resolution{Rung: 2, SurfaceID: observed[0].ID, Route: observed[0].Route}, nil
	}
	// A route can be observed even when static inventory has no page component
	// for it (for example the RCL catalog workspace). Re-read the running base
	// page through BAS as the deterministic route-table fallback; this is still
	// a bounded anchor observation, not vision navigation.
	if link, ok, err := h.observeBaseLink(ctx, targetURL, text); err != nil {
		return nil, err
	} else if ok {
		return &resolution{Rung: 2, SurfaceID: "observed:" + link.route, Route: link.route}, nil
	}

	// Rung 3 uses the normal AI-first search contract and its explicit score
	// threshold/margin. It is intentionally separate from the exact observed
	// route pass above.
	resp, err := h.search.Search(ctx, connect.NewRequest(&searchv1.SearchRequest{Query: text, Limit: 50, Mode: searchv1.Mode_MODE_UNSPECIFIED}))
	if err != nil {
		return nil, fmt.Errorf("search surface corpus: %w", err)
	}
	var corpus []candidate
	for _, hit := range resp.Msg.GetResults() {
		if hit.GetScenario() != scenario {
			continue
		}
		corpus = append(corpus, candidate{ID: hit.GetFilePath(), Route: hit.GetObservedRoute(), DisplayName: hit.GetDisplayName(), FilePath: hit.GetFilePath(), Score: hit.GetScore()})
	}
	sort.Slice(corpus, func(i, j int) bool { return corpus[i].Score > corpus[j].Score })
	if len(corpus) == 0 || corpus[0].Score <= 0.60 {
		return nil, nil
	}
	if len(corpus) > 1 && corpus[0].Score-corpus[1].Score < 0.08 {
		return &resolution{Rung: 3, Candidates: corpus[:min(5, len(corpus))], FollowUp: disambiguationCommand(scenario, corpus[0].ID)}, nil
	}
	route := corpus[0].Route
	if !concreteRoute(route) {
		if declared, routeErr := declaredRoute(ctx, root, scenario, corpus[0].FilePath); routeErr == nil && declared != "" {
			return &resolution{Rung: 4, SurfaceID: corpus[0].ID, Route: declared}, nil
		}
		return &resolution{Rung: 3, SurfaceID: corpus[0].ID, Candidates: corpus[:1], FollowUp: unresolvedRouteReason}, nil
	}
	return &resolution{Rung: 3, SurfaceID: corpus[0].ID, Route: route}, nil
}

const unresolvedRouteReason = "surface route unresolved after selector, declared-route, observed-route, and corpus lookup; supply an exact same-origin URL or declare the page route"

func concreteRoute(route string) bool {
	u, err := url.Parse(route)
	return err == nil && strings.HasPrefix(route, "/") && !strings.HasPrefix(route, "//") && u.Host == "" && !strings.ContainsAny(u.Path, ":*")
}

type observedLink struct {
	route string
}

func (h *handlers) observeBaseLink(ctx context.Context, targetURL, query string) (observedLink, bool, error) {
	if h.ai == nil {
		basURL, err := h.resolver.ResolveScenarioURLDefault(ctx, "browser-automation-studio")
		if err != nil {
			return observedLink{}, false, nil
		}
		h.ai = aiconnect.NewAIServiceClient(h.http, strings.TrimRight(basURL, "/"))
	}
	resp, err := h.ai.GetDOMTree(ctx, connect.NewRequest(&aiv1.GetDOMTreeRequest{Url: targetURL, WaitUntil: aiv1.WaitUntil_WAIT_UNTIL_LOAD, SettleMs: 800, MaxNodes: 4000}))
	if err != nil || resp == nil || resp.Msg == nil || resp.Msg.Tree == nil {
		return observedLink{}, false, nil
	}
	query = strings.ToLower(strings.TrimSpace(query))
	base, _ := url.Parse(targetURL)
	var found observedLink
	var walk func(map[string]any)
	walk = func(node map[string]any) {
		href, _ := node["href"].(string)
		label, _ := node["text"].(string)
		role, _ := node["role"].(string)
		if found.route == "" && (strings.EqualFold(role, "link") || strings.EqualFold(role, "navigation")) && matchesIntent(query, label) && strings.TrimSpace(href) != "" {
			u, parseErr := url.Parse(href)
			if parseErr == nil && (u.Host == "" || (base != nil && u.Host == base.Host)) {
				found.route = u.Path
				if found.route == "" {
					found.route = "/"
				}
			}
		}
		children, _ := node["children"].([]any)
		for _, child := range children {
			if childMap, ok := child.(map[string]any); ok {
				walk(childMap)
			}
		}
	}
	walk(resp.Msg.Tree.AsMap())
	return found, found.route != "", nil
}

func (h *handlers) captureOne(ctx context.Context, client captureconnect.CaptureServiceClient, baseURL string, resolved *resolution, viewport, theme string, strictTokens bool) (captureArtifact, []finding, error) {
	if resolved == nil || !concreteRoute(resolved.Route) {
		return captureArtifact{}, nil, errors.New(unresolvedRouteReason)
	}
	dimensions, err := dimensionsFor(viewport)
	if err != nil {
		return captureArtifact{}, nil, err
	}
	url := strings.TrimRight(baseURL, "/") + resolved.Route
	colorScheme := theme
	request := &capturev1.CaptureRequest{
		Url:                url,
		Captures:           []capturev1.CaptureType{capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT, capturev1.CaptureType_CAPTURE_TYPE_DOM_TREE},
		Dimensions:         dimensions,
		WaitFor:            &capturev1.WaitFor{Spec: &capturev1.WaitFor_TimeoutMs{TimeoutMs: 800}},
		Label:              fmt.Sprintf("ui-health:%s:%s:%s", resolved.SurfaceID, viewport, theme),
		InlineDom:          true,
		InlineDomTree:      true,
		BrowserProfile:     &basev1.BrowserProfile{Fingerprint: &basev1.FingerprintSettings{ColorScheme: &colorScheme}},
		ScreenshotSelector: resolved.Selector,
	}
	if strictTokens {
		request.InteractionFlowJson, err = strictTokenInteractionFlowJSON()
		if err != nil {
			return captureArtifact{}, nil, err
		}
	}
	resp, err := client.Capture(ctx, connect.NewRequest(request))
	if err != nil {
		return captureArtifact{}, nil, fmt.Errorf("capture %s/%s: %w", viewport, theme, err)
	}
	if resp == nil || resp.Msg == nil {
		return captureArtifact{}, nil, fmt.Errorf("capture %s/%s returned no response", viewport, theme)
	}
	artifact := captureArtifact{Viewport: viewport, Theme: theme, ExecutionID: resp.Msg.GetExecutionId()}
	var screenshot []byte
	for _, item := range resp.Msg.GetArtifacts() {
		switch item.GetType() {
		case capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT:
			if item.GetPrimary() || len(screenshot) == 0 {
				artifact.ScreenshotRef = item.GetReference()
				if item.GetPath() != "" {
					screenshot, _ = os.ReadFile(item.GetPath())
				}
			}
		case capturev1.CaptureType_CAPTURE_TYPE_DOM_TREE:
			artifact.SnapshotRef = item.GetReference()
		}
	}
	if resp.Msg.GetReadiness() != nil {
		artifact.Readiness = map[string]any{"outcome": resp.Msg.GetReadiness().GetOutcome(), "selectedStrategy": resp.Msg.GetReadiness().GetSelectedStrategy()}
	}
	if strictTokens {
		artifact.StrictTokenSentinelCount, artifact.NonSentinelColorCount, artifact.NonSentinelColorSamples = strictTokenColorObservation(resp.Msg.GetDomTreeJson())
	}

	visualResp, err := h.visual.AnalyzeArtifacts(ctx, connect.NewRequest(&visualv1.AnalyzeArtifactsRequest{
		Scenario: "ui-health",
		RunId:    resp.Msg.GetExecutionId(),
		Steps: []*visualv1.VisualStepArtifact{{
			StepId:        "capture",
			Label:         fmt.Sprintf("%s %s", viewport, theme),
			Url:           url,
			Viewport:      viewportFor(viewport),
			LayoutJson:    resp.Msg.GetDomTreeJson(),
			ScreenshotPng: screenshot,
			DomHtml:       resp.Msg.GetDomHtml(),
		}},
	}))
	if err != nil {
		return artifact, nil, fmt.Errorf("analyze %s/%s: %w", viewport, theme, err)
	}
	findings := make([]finding, 0)
	for _, step := range visualResp.Msg.GetSteps() {
		for _, item := range step.GetFindings() {
			findings = append(findings, finding{Rule: item.GetCode(), Severity: item.GetSeverity().String(), Selector: item.GetLocation(), Message: item.GetMessage(), Viewport: viewport, Theme: theme})
		}
	}
	return artifact, findings, nil
}

const strictTokenSentinel = "#ff00ff"

// strictTokenInteractionFlowJSON is deliberately expressed as a BAS evaluate
// flow so the diagnostic runs in the same browser session and before the
// computed DOM snapshot. CSSOM is used instead of a source-side token list so
// injected library styles and scenario styles are both included.
func strictTokenInteractionFlowJSON() (string, error) {
	flow := map[string]any{
		"nodes": []any{map[string]any{
			"id": "strict-token-sentinel",
			"action": map[string]any{
				"type": "ACTION_TYPE_EVALUATE",
				"evaluate": map[string]any{"expression": `(function() {
  const names = new Set();
  const visit = (rules) => {
    for (const rule of Array.from(rules || [])) {
      if (rule.style) for (const name of Array.from(rule.style)) if (name.startsWith('--')) names.add(name);
      if (rule.cssRules) visit(rule.cssRules);
    }
  };
  for (const sheet of Array.from(document.styleSheets || [])) {
    try { visit(sheet.cssRules); } catch (_) {}
  }
  for (const style of Array.from(document.querySelectorAll('style'))) {
    for (const match of style.textContent.matchAll(/(--[A-Za-z0-9_-]+)\s*:/g)) names.add(match[1]);
  }
  for (const name of names) document.documentElement.style.setProperty(name, '#ff00ff');
  return { sentinel: '#ff00ff', tokenCount: names.size, tokens: Array.from(names).sort() };
})()`},
			},
		}},
		"edges": []any{},
	}
	data, err := json.Marshal(flow)
	if err != nil {
		return "", fmt.Errorf("encode strict-token interaction flow: %w", err)
	}
	return string(data), nil
}

type strictTokenTreeNode struct {
	Selector string `json:"selector"`
	Computed struct {
		Color           string `json:"color"`
		BackgroundColor string `json:"backgroundColor"`
	} `json:"computed"`
	Children []strictTokenTreeNode `json:"children"`
}

func strictTokenColorObservation(raw string) (int, int, []string) {
	var root strictTokenTreeNode
	if strings.TrimSpace(raw) == "" {
		return 0, 0, nil
	}
	var envelope struct {
		Tree *strictTokenTreeNode `json:"tree"`
	}
	if json.Unmarshal([]byte(raw), &root) != nil {
		return 0, 0, nil
	}
	_ = json.Unmarshal([]byte(raw), &envelope)
	if root.Selector == "" && envelope.Tree != nil {
		root = *envelope.Tree
	}
	sentinelCount := 0
	nonSentinelCount := 0
	samples := make([]string, 0, 20)
	seen := map[string]bool{}
	var walk func(strictTokenTreeNode)
	walk = func(node strictTokenTreeNode) {
		color := strings.TrimSpace(node.Computed.Color)
		background := strings.TrimSpace(node.Computed.BackgroundColor)
		if (isStrictSentinelColor(color) || isStrictSentinelColor(background)) && node.Selector != "" {
			sentinelCount++
		}
		if (nonSentinelColor(color) || nonSentinelColor(background)) && node.Selector != "" {
			nonSentinelCount++
			key := node.Selector + " color=" + color + " background=" + background
			if len(samples) < 20 && !seen[key] {
				seen[key] = true
				samples = append(samples, key)
			}
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	walk(root)
	return sentinelCount, nonSentinelCount, samples
}

func nonSentinelColor(value string) bool {
	if value == "" || strings.EqualFold(value, "transparent") || strings.EqualFold(value, "rgba(0, 0, 0, 0)") {
		return false
	}
	return !isStrictSentinelColor(value)
}

func isStrictSentinelColor(value string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(value, " ", ""))
	return normalized == "#ff00ff" || normalized == "rgb(255,0,255)" || normalized == "rgba(255,0,255,1)" || strings.HasPrefix(normalized, "color(srgb101")
}

func (h *handlers) render(ctx cliapp.RunContext, env envelope, _ error) error {
	if ctx.JSON() {
		return cliapp.PrintJSON(ctx.Stdout(), env)
	}
	return h.renderHuman(ctx, env)
}

func (h *handlers) renderHuman(ctx cliapp.RunContext, env envelope) error {
	_, err := fmt.Fprintf(ctx.Stdout(), "Capture %s: %s\nResolved rung=%d route=%s artifacts=%d findings=%d\n", env.Surface, env.Status, env.ResolvedRung(), env.Route(), len(env.Artifacts), len(env.Findings))
	if err == nil && env.Error != "" {
		_, err = fmt.Fprintln(ctx.Stdout(), env.Error)
	}
	return err
}

func (e envelope) ResolvedRung() int {
	if e.Resolved == nil {
		return 0
	}
	return e.Resolved.Rung
}
func (e envelope) Route() string {
	if e.Resolved == nil {
		return ""
	}
	return e.Resolved.Route
}

func normalizeValues(values, defaults []string) []string {
	if len(values) == 0 {
		return defaults
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.ToLower(strings.TrimSpace(v))
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func dimensionsFor(name string) (*capturev1.Dimensions, error) {
	var preset capturev1.DimensionsPreset
	switch name {
	case "desktop":
		preset = capturev1.DimensionsPreset_DIMENSIONS_PRESET_DESKTOP
	case "mobile":
		preset = capturev1.DimensionsPreset_DIMENSIONS_PRESET_MOBILE
	case "tablet":
		preset = capturev1.DimensionsPreset_DIMENSIONS_PRESET_TABLET
	default:
		return nil, fmt.Errorf("unsupported viewport %q (use desktop, tablet, or mobile)", name)
	}
	return &capturev1.Dimensions{Preset: preset}, nil
}

func viewportFor(name string) *visualv1.Viewport {
	width, height := int32(0), int32(0)
	switch name {
	case "desktop":
		width, height = 1440, 900
	case "tablet":
		width, height = 1024, 768
	case "mobile":
		width, height = 390, 844
	}
	return &visualv1.Viewport{DeviceProfile: name, Width: width, Height: height}
}

func observedScore(query string, hit *searchv1.SearchResult) float64 {
	q := strings.ToLower(strings.TrimSpace(query))
	for _, value := range []string{hit.GetObservedLinkText(), hit.GetObservedPageTitle(), hit.GetObservedRoute()} {
		v := strings.ToLower(strings.TrimSpace(value))
		if v == q {
			return 1
		}
		if v != "" && strings.Contains(v, q) {
			return 0.85
		}
	}
	return 0
}

// matchesIntent keeps the base-page route fallback deterministic while still
// accepting the product's natural phrasing (for example, "the design page")
// when the observed anchor is simply labelled "Design".
func matchesIntent(query, label string) bool {
	q := intentTokens(query)
	l := intentTokens(label)
	if len(q) == 0 || len(l) == 0 {
		return false
	}
	if strings.Join(q, " ") == strings.Join(l, " ") {
		return true
	}
	return containsAll(q, l) || containsAll(l, q)
}

func intentTokens(value string) []string {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(value)))
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.Trim(field, " .,!?/\\")
		switch field {
		case "", "a", "an", "the", "page", "route", "screen", "view":
			continue
		default:
			out = append(out, field)
		}
	}
	return out
}

func containsAll(needles, haystack []string) bool {
	for _, needle := range needles {
		found := false
		for _, value := range haystack {
			if value == needle {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func dedupeCandidates(in []candidate) []candidate {
	seen := map[string]candidate{}
	for _, item := range in {
		key := item.Route + "\x00" + item.FilePath
		if prev, ok := seen[key]; !ok || item.Score > prev.Score {
			seen[key] = item
		}
	}
	out := make([]candidate, 0, len(seen))
	for _, item := range seen {
		out = append(out, item)
	}
	return out
}

func disambiguationCommand(scenario, id string) string {
	return fmt.Sprintf("ui-health capture --scenario %s --surface %s --json", scenario, id)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func firstError(current, next string) string {
	if current != "" {
		return current
	}
	return next
}

func shouldFail(threshold string, findings []finding) bool {
	order := map[string]int{"INFO": 1, "WARNING": 2, "ERROR": 3}
	threshold = strings.ToUpper(strings.TrimSpace(threshold))
	if threshold == "" {
		return false
	}
	want, ok := order[threshold]
	if !ok {
		return false
	}
	for _, item := range findings {
		if order[strings.ToUpper(item.Severity)] >= want {
			return true
		}
	}
	return false
}
