// Package routing is the search-hub router core: it resolves which registered
// providers a query targets, fans out to each over HTTP+JSON with bounded
// concurrency and a per-provider timeout, maps every heterogeneous response
// through the generic providers.MapResults adapter, and groups the results by
// provider.
//
// It is the load-bearing realization of the thin-router invariant: this package
// holds NO corpus data and NO provider-specific code. A provider is reached
// entirely through its registered ProviderDescriptor (logical scenario_id +
// path + body_template + ResultMapping); the live base URL is resolved at
// call-time via the cross-scenario URLResolver seam (never client-computed).
//
// Phase 4 ships explicit-type fan-out and honest by-provider grouping — there
// is no cross-provider score interleave until the cross-encoder reranker lands
// (Phase 6), and no automatic classifier until Phase 5 (a query with neither
// explicit types, --all, nor --group is rejected, not silently widened).
package routing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"search-hub/internal/httpc"
	"search-hub/internal/providers"
	internalregistry "search-hub/internal/registry"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

const (
	// defaultPerProviderTimeout bounds a single provider's HTTP round-trip so a
	// slow or hung leaf never blocks the whole fan-out (Risk: latency/fan-out
	// cost — partial results return within the bound).
	defaultPerProviderTimeout = 10 * time.Second
	// defaultConcurrency caps how many providers are queried at once.
	defaultConcurrency = 6
	// defaultLimit is the per-provider result cap when the request omits one.
	defaultLimit = 10
	// maxResponseBytes caps how much of a provider response the router reads,
	// so a misbehaving leaf cannot exhaust memory.
	maxResponseBytes = 8 << 20 // 8 MiB
)

// ProviderLister is the registry read seam the router depends on. The
// SQLite-backed registry Store satisfies it in production; tests substitute a
// fake. Declared at the consumer (seam-discovery) so the router never imports
// a concrete store.
type ProviderLister interface {
	List(ctx context.Context, filter internalregistry.ListFilter) ([]*registryv1.ProviderDescriptor, error)
}

// URLResolver turns a logical scenario_id into the scenario's live API base URL
// (scheme://host:port) at call-time. Production wraps api-core/discovery's
// Resolver (which shells out to `vrooli scenario port`); tests inject a static
// or recording resolver. This is the project rule "never compute proxy URLs
// client-side; resolve via the backend" made into a seam.
type URLResolver interface {
	ResolveScenarioURL(ctx context.Context, scenarioID string) (baseURL string, err error)
}

// ErrInvalidQuery is the typed sentinel for caller mistakes the router can
// detect before any fan-out (empty query text; no routing selector while the
// classifier is still Phase 5). The Connect handler translates it into
// InvalidArgument.
type ErrInvalidQuery struct{ Reason string }

func (e ErrInvalidQuery) Error() string { return e.Reason }

// Deps wires the router's seams. Lister/Resolver/Doer are required; the rest
// default in NewRouter.
type Deps struct {
	Lister             ProviderLister
	Resolver           URLResolver
	Doer               httpc.Doer
	Logger             *log.Logger
	Concurrency        int
	PerProviderTimeout time.Duration
}

// Router executes federated queries across registered providers.
type Router struct {
	deps Deps
}

// NewRouter constructs a Router, applying defaults for the optional Deps
// fields. Logger defaults to log.Default(); concurrency and timeout fall back
// to the package defaults when non-positive.
func NewRouter(d Deps) *Router {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	if d.Concurrency <= 0 {
		d.Concurrency = defaultConcurrency
	}
	if d.PerProviderTimeout <= 0 {
		d.PerProviderTimeout = defaultPerProviderTimeout
	}
	return &Router{deps: d}
}

// Query fans out text to the providers selected by req (explicit types, --all,
// and/or --group), collects each provider's hits behind the generic adapter,
// and returns them grouped by provider. A provider that is unreachable, times
// out, returns a non-2xx, or yields an unmappable body is reported as a
// degraded group with a human note — it never fails the whole query (graceful
// degradation, partial results).
//
// It returns an error only for caller mistakes (ErrInvalidQuery) or a registry
// read failure — never for an individual provider's runtime failure.
func (r *Router) Query(ctx context.Context, req *routingv1.QueryRequest) (*routingv1.QueryResponse, error) {
	start := time.Now()

	query := strings.TrimSpace(req.GetQuery())
	if query == "" {
		return nil, ErrInvalidQuery{Reason: "query text is required"}
	}
	group := strings.TrimSpace(req.GetGroup())
	if !req.GetAll() && len(nonEmpty(req.GetTypes())) == 0 && group == "" {
		return nil, ErrInvalidQuery{Reason: "no routing target: pass explicit --type <types>, --all, or --group <scenario> (automatic routing lands in Phase 5)"}
	}

	limit := req.GetLimit()
	if limit <= 0 {
		limit = defaultLimit
	}

	// Only ACTIVE leaves are callable; capability_gap stubs carry no endpoint
	// and are intentionally excluded from fan-out.
	active, err := r.deps.Lister.List(ctx, internalregistry.ListFilter{
		State: int32(registryv1.ProviderState_PROVIDER_STATE_ACTIVE),
	})
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}

	targets := selectTargets(active, req)

	groups := r.fanOut(ctx, targets, query, limit)

	resp := &routingv1.QueryResponse{
		Groups:   groups,
		Reranked: false, // honest by-provider grouping until Phase 6 rerank
	}
	for _, g := range groups {
		resp.CorporaSearched = append(resp.CorporaSearched, g.GetProviderId())
		if g.GetDegraded() {
			resp.Degraded = true
		}
	}
	if req.GetExplain() {
		resp.RoutingExplanation = explain(req, targets)
	}
	resp.LatencyMs = time.Since(start).Milliseconds()
	return resp, nil
}

// selectTargets applies the explicit routing selectors to the active provider
// set, preserving the registry's provider_id order (List orders ascending).
// Precedence within the optional --group scope: --all > explicit types > group
// alone (group with no narrower selector means "every active leaf in the
// group"). Providers without a callable endpoint are skipped defensively.
func selectTargets(active []*registryv1.ProviderDescriptor, req *routingv1.QueryRequest) []*registryv1.ProviderDescriptor {
	group := strings.TrimSpace(req.GetGroup())
	typeSet := make(map[string]struct{})
	for _, t := range nonEmpty(req.GetTypes()) {
		typeSet[t] = struct{}{}
	}

	out := make([]*registryv1.ProviderDescriptor, 0, len(active))
	for _, p := range active {
		if p.GetEndpoint() == nil {
			continue
		}
		if group != "" && p.GetProviderGroup() != group {
			continue
		}
		switch {
		case req.GetAll():
			out = append(out, p)
		case len(typeSet) > 0:
			if _, ok := typeSet[p.GetType()]; ok {
				out = append(out, p)
			}
		case group != "":
			out = append(out, p)
		}
	}
	return out
}

// fanOut queries every target concurrently (bounded by Concurrency) and returns
// one group per target, ordered by provider_id for deterministic output.
func (r *Router) fanOut(ctx context.Context, targets []*registryv1.ProviderDescriptor, query string, limit int32) []*routingv1.ProviderResultGroup {
	groups := make([]*routingv1.ProviderResultGroup, len(targets))
	sem := make(chan struct{}, r.deps.Concurrency)
	var wg sync.WaitGroup
	for i, t := range targets {
		wg.Add(1)
		go func(i int, t *registryv1.ProviderDescriptor) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			groups[i] = r.callProvider(ctx, t, query, limit)
		}(i, t)
	}
	wg.Wait()

	sort.SliceStable(groups, func(i, j int) bool {
		return groups[i].GetProviderId() < groups[j].GetProviderId()
	})
	return groups
}

// callProvider performs one provider's round-trip and always returns a group:
// on any failure it returns a degraded group carrying a human-readable note
// rather than an error, so one bad provider never sinks the query.
func (r *Router) callProvider(ctx context.Context, d *registryv1.ProviderDescriptor, query string, limit int32) *routingv1.ProviderResultGroup {
	g := &routingv1.ProviderResultGroup{ProviderId: d.GetProviderId()}

	hj := d.GetEndpoint().GetHttpJson()
	if hj == nil {
		return degrade(g, "unsupported endpoint kind (only http_json fan-out is implemented in Phase 4)")
	}

	base, err := r.deps.Resolver.ResolveScenarioURL(ctx, hj.GetScenarioId())
	if err != nil {
		return degrade(g, fmt.Sprintf("provider scenario %q unreachable: %s", hj.GetScenarioId(), oneLine(err.Error())))
	}

	body, err := renderBody(hj.GetBodyTemplate(), query, limit, d.GetType())
	if err != nil {
		return degrade(g, fmt.Sprintf("request build failed: %s", err))
	}

	cctx, cancel := context.WithTimeout(ctx, r.deps.PerProviderTimeout)
	defer cancel()

	url := strings.TrimRight(base, "/") + hj.GetPath()
	httpReq, err := http.NewRequestWithContext(cctx, httpMethod(hj.GetMethod()), url, bytes.NewReader([]byte(body)))
	if err != nil {
		return degrade(g, fmt.Sprintf("request build failed: %s", oneLine(err.Error())))
	}
	applyHeaders(httpReq, hj.GetHeaders())

	resp, err := r.deps.Doer.Do(httpReq)
	if err != nil {
		return degrade(g, fmt.Sprintf("request failed: %s", oneLine(err.Error())))
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return degrade(g, fmt.Sprintf("reading response failed: %s", oneLine(err.Error())))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return degrade(g, fmt.Sprintf("provider returned HTTP %d", resp.StatusCode))
	}

	hits, err := providers.MapResults(d, respBody)
	if err != nil {
		return degrade(g, fmt.Sprintf("result mapping failed: %s", oneLine(err.Error())))
	}
	if len(hits) > int(limit) {
		hits = hits[:limit]
	}
	g.Hits = hits
	g.Count = int32(len(hits))
	return g
}

// degrade marks a group degraded with a note and returns it (for one-line use
// in callProvider's failure branches).
func degrade(g *routingv1.ProviderResultGroup, note string) *routingv1.ProviderResultGroup {
	g.Degraded = true
	g.Note = note
	return g
}

// renderBody substitutes the {{query}}, {{limit}}, and {{type}} placeholders in
// a provider's body_template. {{query}} and {{type}} sit inside JSON string
// quotes in the template, so they are inserted as JSON-escaped *inner* strings
// (no surrounding quotes); {{limit}} is a bare integer. The result is validated
// as JSON so a malformed template surfaces as a degraded provider, not a
// confusing downstream parse error.
func renderBody(tmpl, query string, limit int32, typ string) (string, error) {
	if strings.TrimSpace(tmpl) == "" {
		return "", fmt.Errorf("empty body_template")
	}
	out := tmpl
	out = strings.ReplaceAll(out, "{{query}}", jsonStringInner(query))
	out = strings.ReplaceAll(out, "{{limit}}", strconv.FormatInt(int64(limit), 10))
	out = strings.ReplaceAll(out, "{{type}}", jsonStringInner(typ))
	if !json.Valid([]byte(out)) {
		return "", fmt.Errorf("rendered body is not valid JSON")
	}
	return out, nil
}

// jsonStringInner returns s JSON-escaped with the surrounding quotes stripped,
// so it can be dropped into a "{{placeholder}}" slot that already lives inside
// quotes in the template.
func jsonStringInner(s string) string {
	b, err := json.Marshal(s)
	if err != nil || len(b) < 2 {
		return ""
	}
	return string(b[1 : len(b)-1])
}

// httpMethod maps the descriptor's HttpMethod enum to a net/http verb,
// defaulting to POST (the dominant Connect/REST search shape).
func httpMethod(m registryv1.HttpMethod) string {
	if m == registryv1.HttpMethod_HTTP_METHOD_GET {
		return http.MethodGet
	}
	return http.MethodPost
}

// applyHeaders copies the descriptor's headers, defaulting Content-Type to
// application/json when the descriptor omits it (every search endpoint the hub
// federates speaks JSON).
func applyHeaders(req *http.Request, headers map[string]string) {
	hasContentType := false
	for k, v := range headers {
		req.Header.Set(k, v)
		if strings.EqualFold(k, "Content-Type") {
			hasContentType = true
		}
	}
	if !hasContentType {
		req.Header.Set("Content-Type", "application/json")
	}
}

// explain renders the human-readable routing rationale for --explain. Phase 4
// routing is purely explicit (no classifier), so the explanation states which
// selector chose the targets.
func explain(req *routingv1.QueryRequest, targets []*registryv1.ProviderDescriptor) []string {
	ids := make([]string, 0, len(targets))
	for _, t := range targets {
		ids = append(ids, t.GetProviderId())
	}
	var selector string
	switch {
	case req.GetAll():
		selector = "explicit --all → every active provider"
	case len(nonEmpty(req.GetTypes())) > 0:
		selector = fmt.Sprintf("explicit --type %s", strings.Join(nonEmpty(req.GetTypes()), ","))
	default:
		selector = fmt.Sprintf("explicit --group %s → every active leaf in the group", strings.TrimSpace(req.GetGroup()))
	}
	if g := strings.TrimSpace(req.GetGroup()); g != "" && !req.GetAll() && len(nonEmpty(req.GetTypes())) > 0 {
		selector += fmt.Sprintf(" scoped to --group %s", g)
	}
	out := []string{
		"automatic routing (classifier) lands in Phase 5; Phase 4 routes on explicit selectors only",
		fmt.Sprintf("selector: %s", selector),
	}
	if len(ids) == 0 {
		out = append(out, "matched providers: none")
	} else {
		out = append(out, fmt.Sprintf("matched providers: %s", strings.Join(ids, ", ")))
	}
	return out
}

// nonEmpty filters out blank entries (and trims) so a stray empty --type token
// is not treated as a selector.
func nonEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if t := strings.TrimSpace(s); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// oneLine collapses newlines so a multi-line transport error renders as a
// single clean note.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
