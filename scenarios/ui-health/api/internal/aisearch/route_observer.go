package aisearch

// This file owns the runtime half of UI-surface discovery. Static inventory
// remains authoritative for what a scenario declares; this observer records
// what a running UI actually exposes through same-origin links.

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	urlpkg "net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/ai"
	aiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/ai/aiconnect"
)

const (
	maxObservedRoutes = 50
	maxObservedDepth  = 3
	observationSettle = 800
)

var titleRE = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)

// RouteObservation is the durable observation attached to an indexed surface
// (or retained in the reindex report when no static surface matched it).
type RouteObservation struct {
	Scenario   string `json:"scenario"`
	Route      string `json:"route"`
	LinkText   string `json:"linkText,omitempty"`
	PageTitle  string `json:"pageTitle,omitempty"`
	ObservedAt string `json:"observedAt"`
	Reachable  bool   `json:"reachable"`
	HTTPStatus int    `json:"httpStatus"`
}

// RouteReport is the operator-facing result of one bounded crawl.
type RouteReport struct {
	Scenario       string             `json:"scenario"`
	Observations   []RouteObservation `json:"observations"`
	OrphanSurfaces []string           `json:"orphanSurfaces"`
	DeadRoutes     []string           `json:"deadRoutes"`
	Warning        string             `json:"warning,omitempty"`
}

type routeQueueItem struct {
	URL   string
	Depth int
}

type observedDiscovery struct {
	base     DiscoverySource
	resolver *discovery.Resolver
	client   *http.Client

	mu      sync.RWMutex
	byJob   map[string]RouteReport
	byRoute map[string]map[string]RouteObservation
}

func newObservedDiscovery(base DiscoverySource) *observedDiscovery {
	return &observedDiscovery{
		base:     base,
		resolver: discovery.NewResolver(discovery.ResolverConfig{}),
		client:   &http.Client{Timeout: 30 * time.Second},
		byJob:    map[string]RouteReport{},
		byRoute:  map[string]map[string]RouteObservation{},
	}
}

func (d *observedDiscovery) ListScenarios(ctx context.Context) ([]string, error) {
	return d.base.ListScenarios(ctx)
}

func (d *observedDiscovery) Discover(ctx context.Context, scenario string) ([]SurfaceRecord, error) {
	records, err := d.base.Discover(ctx, scenario)
	if err != nil {
		return nil, err
	}
	d.mu.RLock()
	observations := d.byRoute[scenario]
	d.mu.RUnlock()
	if len(observations) == 0 {
		return records, nil
	}
	for i := range records {
		if obs, ok := matchObservation(records[i], observations); ok {
			records[i].ObservedRoute = obs.Route
			records[i].ObservedLinkText = obs.LinkText
			records[i].ObservedPageTitle = obs.PageTitle
			records[i].ObservedAt = obs.ObservedAt
			records[i].Reachable = obs.Reachable
			records[i].HTTPStatus = obs.HTTPStatus
		}
	}
	return records, nil
}

func (d *observedDiscovery) Observe(ctx context.Context, scenario string) RouteReport {
	report := RouteReport{Scenario: scenario, Observations: []RouteObservation{}, OrphanSurfaces: []string{}, DeadRoutes: []string{}}
	uiURL, err := d.resolver.ResolveScenarioURL(ctx, scenario, "UI_PORT")
	if err != nil || strings.TrimSpace(uiURL) == "" {
		report.Warning = fmt.Sprintf("target UI is not running: %v", err)
		return report
	}
	basURL, err := d.resolver.ResolveScenarioURLDefault(ctx, "browser-automation-studio")
	if err != nil || strings.TrimSpace(basURL) == "" {
		report.Warning = fmt.Sprintf("browser-automation-studio is not running: %v", err)
		return report
	}

	bas := aiconnect.NewAIServiceClient(d.client, strings.TrimRight(basURL, "/"))
	baseURL, err := urlpkg.Parse(strings.TrimRight(uiURL, "/"))
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" {
		report.Warning = fmt.Sprintf("invalid target UI URL %q", uiURL)
		return report
	}

	queue := []routeQueueItem{{URL: baseURL.String(), Depth: 0}}
	visited := map[string]struct{}{}
	for len(queue) > 0 && len(visited) < maxObservedRoutes {
		item := queue[0]
		queue = queue[1:]
		canonical := canonicalURL(item.URL)
		if _, seen := visited[canonical]; seen {
			continue
		}
		visited[canonical] = struct{}{}

		obs, links, statusErr := d.observeURL(ctx, bas, baseURL, item.URL, scenario)
		report.Observations = append(report.Observations, obs)
		if obs.HTTPStatus < 200 || obs.HTTPStatus >= 300 {
			report.DeadRoutes = append(report.DeadRoutes, obs.Route)
		}
		if statusErr != nil && report.Warning == "" {
			report.Warning = statusErr.Error()
		}
		if item.Depth >= maxObservedDepth || obs.HTTPStatus < 200 || obs.HTTPStatus >= 300 {
			continue
		}
		for _, link := range links {
			next, ok := sameOriginLink(baseURL, item.URL, link.href)
			if !ok || next == "" {
				continue
			}
			if _, seen := visited[canonicalURL(next)]; seen {
				continue
			}
			queue = append(queue, routeQueueItem{URL: next, Depth: item.Depth + 1})
		}
	}

	report = d.enrichReport(ctx, scenario, report)
	d.applyReport(scenario, report)
	return report
}

type domLink struct {
	href string
	text string
}

func (d *observedDiscovery) observeURL(ctx context.Context, bas aiconnect.AIServiceClient, base *urlpkg.URL, rawURL, scenario string) (RouteObservation, []domLink, error) {
	u, _ := urlpkg.Parse(rawURL)
	obs := RouteObservation{
		Scenario:   scenario,
		Route:      routePath(u),
		ObservedAt: time.Now().UTC().Format(time.RFC3339),
	}
	pageTitle, status, body, err := d.fetchPage(ctx, rawURL)
	obs.PageTitle = pageTitle
	obs.HTTPStatus = status
	obs.Reachable = err == nil && status >= 200 && status < 300
	if err != nil {
		return obs, nil, err
	}
	if status < 200 || status >= 300 {
		return obs, nil, nil
	}

	resp, err := bas.GetDOMTree(ctx, connect.NewRequest(&aiv1.GetDOMTreeRequest{
		Url:       rawURL,
		WaitUntil: aiv1.WaitUntil_WAIT_UNTIL_LOAD,
		SettleMs:  observationSettle,
		Computed:  false,
		MaxNodes:  4000,
	}))
	if err != nil {
		return obs, nil, fmt.Errorf("capture %s: %w", obs.Route, err)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Tree == nil {
		return obs, nil, fmt.Errorf("capture %s returned no DOM tree", obs.Route)
	}
	links := collectDOMLinks(resp.Msg.Tree.AsMap())
	for i := range links {
		if obs.LinkText == "" && canonicalURL(links[i].href) == canonicalURL(rawURL) {
			obs.LinkText = links[i].text
		}
	}
	_ = base
	_ = body
	return obs, links, nil
}

func (d *observedDiscovery) fetchPage(ctx context.Context, rawURL string) (string, int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", 0, nil, err
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return "", 0, nil, err
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if readErr != nil {
		return "", resp.StatusCode, body, readErr
	}
	title := ""
	if match := titleRE.FindSubmatch(body); len(match) == 2 {
		title = strings.TrimSpace(html.UnescapeString(string(match[1])))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return title, resp.StatusCode, body, fmt.Errorf("GET %s returned HTTP %d", rawURL, resp.StatusCode)
	}
	return title, resp.StatusCode, body, nil
}

func collectDOMLinks(node map[string]any) []domLink {
	var out []domLink
	var walk func(map[string]any)
	walk = func(current map[string]any) {
		if href, ok := current["href"].(string); ok && strings.TrimSpace(href) != "" {
			text, _ := current["text"].(string)
			out = append(out, domLink{href: strings.TrimSpace(href), text: strings.TrimSpace(text)})
		}
		children, _ := current["children"].([]any)
		for _, child := range children {
			if childMap, ok := child.(map[string]any); ok {
				walk(childMap)
			}
		}
	}
	walk(node)
	return out
}

func sameOriginLink(base *urlpkg.URL, current, href string) (string, bool) {
	ref, err := urlpkg.Parse(strings.TrimSpace(href))
	if err != nil || ref.Scheme == "mailto" || ref.Scheme == "javascript" {
		return "", false
	}
	if ref.Fragment != "" {
		ref.Fragment = ""
	}
	cur, err := urlpkg.Parse(current)
	if err != nil {
		return "", false
	}
	next := cur.ResolveReference(ref)
	if next.Scheme != base.Scheme || next.Host != base.Host {
		return "", false
	}
	if strings.HasPrefix(next.Path, "/assets/") || strings.HasSuffix(strings.ToLower(next.Path), ".js") || strings.HasSuffix(strings.ToLower(next.Path), ".css") {
		return "", false
	}
	next.Fragment = ""
	return next.String(), true
}

func canonicalURL(raw string) string {
	u, err := urlpkg.Parse(raw)
	if err != nil {
		return raw
	}
	u.Fragment = ""
	if u.Path == "" {
		u.Path = "/"
	}
	if u.Path != "/" {
		u.Path = strings.TrimRight(u.Path, "/")
	}
	return u.String()
}

func routePath(u *urlpkg.URL) string {
	if u == nil {
		return "/"
	}
	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}
	if u.RawQuery != "" {
		path += "?" + u.RawQuery
	}
	return path
}

func (d *observedDiscovery) applyReport(scenario string, report RouteReport) {
	observations := make(map[string]RouteObservation, len(report.Observations))
	for _, obs := range report.Observations {
		observations[obs.Route] = obs
	}
	d.mu.Lock()
	d.byRoute[scenario] = observations
	d.mu.Unlock()
}

func (d *observedDiscovery) saveJobReport(jobID string, report RouteReport) {
	d.mu.Lock()
	d.byJob[jobID] = report
	d.mu.Unlock()
}

func (d *observedDiscovery) reportForJob(jobID string) (RouteReport, bool) {
	d.mu.RLock()
	report, ok := d.byJob[jobID]
	d.mu.RUnlock()
	return report, ok
}

func matchObservation(record SurfaceRecord, observations map[string]RouteObservation) (RouteObservation, bool) {
	if record.Kind != "SURFACE_KIND_PAGE" && record.Slot != "page" {
		return RouteObservation{}, false
	}
	display := strings.ToLower(strings.TrimSuffix(record.DisplayName, "Page"))
	for _, obs := range observations {
		path := strings.Trim(obs.Route, "/")
		if path == "" && strings.Contains(strings.ToLower(record.DisplayName), "dashboard") {
			return obs, true
		}
		segment := path
		if slash := strings.IndexByte(segment, '/'); slash >= 0 {
			segment = segment[:slash]
		}
		segment = strings.ToLower(strings.ReplaceAll(segment, "-", ""))
		if segment != "" && strings.Contains(strings.ReplaceAll(display, " ", ""), segment) {
			return obs, true
		}
	}
	return RouteObservation{}, false
}

func (d *observedDiscovery) enrichReport(ctx context.Context, scenario string, report RouteReport) RouteReport {
	report.OrphanSurfaces = nil
	report.DeadRoutes = nil
	records, err := d.base.Discover(ctx, scenario)
	if err != nil {
		report.Warning = firstNonEmpty(report.Warning, err.Error())
		return report
	}
	observations := map[string]RouteObservation{}
	for _, obs := range report.Observations {
		observations[obs.Route] = obs
	}
	matched := map[string]struct{}{}
	for _, record := range records {
		if _, ok := matchObservation(record, observations); ok {
			matched[record.FilePath] = struct{}{}
		}
	}
	for _, record := range records {
		if record.Kind == "SURFACE_KIND_PAGE" || record.Slot == "page" {
			if _, ok := matched[record.FilePath]; !ok {
				report.OrphanSurfaces = append(report.OrphanSurfaces, record.DisplayName+" ("+record.FilePath+")")
			}
		}
	}
	for _, obs := range report.Observations {
		matchedRoute := false
		for _, record := range records {
			if candidate, ok := matchObservation(record, map[string]RouteObservation{obs.Route: obs}); ok && candidate.Route == obs.Route {
				matchedRoute = true
				break
			}
		}
		if !matchedRoute {
			report.OrphanSurfaces = append(report.OrphanSurfaces, "route:"+obs.Route+" (no indexed page surface)")
		}
	}
	sort.Strings(report.OrphanSurfaces)
	sort.Strings(report.DeadRoutes)
	return report
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
