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
// Routing is explicit (--type/--all/--group) or automatic: when no selector is
// given and a Classifier is wired, the router asks the classifier
// which provider types to hit — reading only the registry's NL descriptions —
// and widens on uncertainty (over-fetch, recall over precision) so the bare
// `search-hub query "…"` routes on its own. Results are always grouped honestly
// by provider for provenance; when a Reranker is wired the fused
// shortlist is additionally reranked into one comparable cross-provider list,
// degrading back to grouping-only if the reranker is unavailable.
package routing

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
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
	// defaultRerankTimeout bounds the single reranker round-trip so a slow or
	// thinky model degrades to honest grouping fast instead of hanging the whole
	// query (the reranker sits on the hot path, unlike per-provider fan-out which
	// is already bounded). The underlying gateway has its own ~60s cap; this is
	// the tighter, router-owned bound.
	defaultRerankTimeout = 30 * time.Second
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
// default in NewRouter. Classifier is optional: when nil, a query with no
// explicit selector is rejected; when set, such a query
// is routed automatically. Reranker is optional: when nil, results
// stay grouped by provider; when set, the fused
// shortlist is reranked into one comparable list, degrading back to grouping if
// the reranker fails. Recorder is optional: when set, each completed
// query emits a TelemetrySample; when nil, no telemetry is
// recorded.
type Deps struct {
	Lister             ProviderLister
	Resolver           URLResolver
	Doer               httpc.Doer
	Classifier         Classifier
	Reranker           Reranker
	Recorder           TelemetryRecorder
	Logger             *log.Logger
	Concurrency        int
	PerProviderTimeout time.Duration
	RerankTimeout      time.Duration
	// AutoRouteExternal gates OT-P2-002: when true, the automatic (classifier)
	// path may fold SCOPE_EXTERNAL providers back into the fan-out — either
	// because the classifier judged the query web-shaped (above
	// autoExternalThreshold) or as a fallback escalation when the project corpus
	// returned no hits. DEFAULT FALSE: a plain federated query never auto-hits a
	// rate-limited/paid external corpus unless the operator opts in. Explicit
	// --all/--type always reach external providers regardless of this flag.
	AutoRouteExternal bool
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
	if d.RerankTimeout <= 0 {
		d.RerankTimeout = defaultRerankTimeout
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
	hasExplicit := req.GetAll() || len(nonEmpty(req.GetTypes())) > 0 || strings.TrimSpace(req.GetGroup()) != ""
	if !hasExplicit && r.deps.Classifier == nil {
		// No selector and no classifier wired ⇒ reject rather than silently widen.
		return nil, ErrInvalidQuery{Reason: "no routing target: pass explicit --type <types>, --all, or --group <scenario> (automatic routing requires a classifier)"}
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

	var (
		targets            []*registryv1.ProviderDescriptor
		autoExplain        []string
		classifierError    bool
		pendingExternal    []*registryv1.ProviderDescriptor
		autoRoutedExternal bool
		escalated          bool
	)
	if hasExplicit {
		// Explicit selectors (--all / --type / --group) reach every active
		// provider, including SCOPE_EXTERNAL ones — the operator asked for them.
		targets = selectTargets(active, req)
	} else {
		// Automatic/classifier routing never auto-hits an external (e.g.
		// rate-limited / paid) corpus UNLESS the operator opted in
		// (AutoRouteExternal) and the classifier judged the query web-shaped.
		// External providers always stay reachable via the explicit path above.
		autoCandidates, withheldExternal := partitionByScope(active)
		var webShaped bool
		targets, autoExplain, classifierError, webShaped = r.autoSelect(ctx, autoCandidates, query)
		pendingExternal = withheldExternal
		switch {
		case r.deps.AutoRouteExternal && webShaped && len(withheldExternal) > 0:
			// OT-P2-002: a confidently web-shaped query folds the withheld
			// external provider(s) back into the fan-out — driven purely off
			// descriptor scope + the generic web-shaped label (no per-provider
			// code). Rate-safety is the provider's own governor's job.
			targets = append(targets, withheldExternal...)
			autoRoutedExternal = true
			autoExplain = append(autoExplain, autoRoutedExternalLine(withheldExternal))
		case len(withheldExternal) > 0:
			autoExplain = append(autoExplain, withheldExternalLine(withheldExternal))
		}
	}

	groups := r.fanOut(ctx, targets, query, limit)

	// OT-P2-002 fallback escalation: if the project corpus returned nothing and
	// the operator opted in, escalate to the withheld external provider(s) via
	// the same governor-protected path (only on the automatic path, and only
	// when we did not already auto-route external above).
	if !hasExplicit && r.deps.AutoRouteExternal && !autoRoutedExternal && len(pendingExternal) > 0 && resultsWeak(groups) {
		escalationGroups := r.fanOut(ctx, pendingExternal, query, limit)
		groups = append(groups, escalationGroups...)
		targets = append(targets, pendingExternal...)
		escalated = true
		autoExplain = append(autoExplain, escalationLine(pendingExternal))
	}

	ranked, reranked, rerankDegraded, rerankExplain := r.maybeRerank(ctx, query, groups)

	resp := &routingv1.QueryResponse{
		Ranked:   ranked,
		Groups:   groups,
		Reranked: reranked,
		Degraded: classifierError || rerankDegraded,
	}
	for _, g := range groups {
		resp.CorporaSearched = append(resp.CorporaSearched, g.GetProviderId())
		if g.GetDegraded() {
			resp.Degraded = true
		}
	}
	if req.GetExplain() {
		if hasExplicit {
			resp.RoutingExplanation = explain(req, targets)
		} else {
			resp.RoutingExplanation = append(autoExplain, matchedProvidersLine(targets))
		}
		resp.RoutingExplanation = append(resp.RoutingExplanation, rerankExplain...)
	}
	resp.LatencyMs = time.Since(start).Milliseconds()

	// Phase-7 telemetry: record the query's outcome (best-effort; the recorder
	// swallows its own errors so a telemetry failure never affects the query).
	// LatencyMs is already stamped, so recording time is not counted.
	if r.deps.Recorder != nil {
		sample := buildSample(query, targets, resp)
		sample.AutoRoutedExternal = autoRoutedExternal
		sample.Escalated = escalated
		r.deps.Recorder.Record(ctx, sample)
	}
	return resp, nil
}

// maybeRerank fuses the per-provider groups into one shortlist and reranks it
// into a unified, cross-provider ordered list. It never fails the query:
//
//   - no reranker wired ⇒ keep honest by-provider grouping (Phase-4 behavior).
//   - nothing to rank (zero hits) ⇒ same — grouping only, not a degradation.
//   - reranker errors (model down, bad output) ⇒ keep grouping and flag the
//     response degraded (the mandatory Phase-6 graceful-degradation path,
//     mirroring the classifier's).
//
// On success it returns the ranked list (each hit carrying RerankScore),
// reranked=true, and a one-line --explain note.
func (r *Router) maybeRerank(ctx context.Context, query string, groups []*routingv1.ProviderResultGroup) (ranked []*routingv1.SearchHit, reranked, degraded bool, explain []string) {
	if r.deps.Reranker == nil {
		return nil, false, false, nil
	}
	candidates := fuseGroups(groups)
	if len(candidates) == 0 {
		return nil, false, false, nil
	}
	rctx, cancel := context.WithTimeout(ctx, r.deps.RerankTimeout)
	defer cancel()
	ranked, err := r.deps.Reranker.Rerank(rctx, query, candidates)
	if err != nil {
		r.deps.Logger.Printf("routing.maybeRerank: reranker failed, keeping by-provider grouping: %v", err)
		return nil, false, true, []string{
			fmt.Sprintf("reranker unavailable (%s) — showing honest by-provider grouping", oneLine(err.Error())),
		}
	}
	return ranked, true, false, []string{
		fmt.Sprintf("reranked %d candidate(s) into one unified cross-provider list", len(ranked)),
	}
}

// autoSelect runs the classifier over the active providers' descriptions and
// returns the leaves to fan out to. It never fails the query: if the classifier
// errors (model down, unparseable output) it widens to every active provider
// and reports classifierError=true so the response is flagged degraded. The
// widen-on-uncertainty policy (low confidence ⇒ broaden) lives in widenPolicy.
func (r *Router) autoSelect(ctx context.Context, active []*registryv1.ProviderDescriptor, query string) (targets []*registryv1.ProviderDescriptor, explain []string, classifierError, webShaped bool) {
	available := availableTypes(active)
	profiles := buildProfiles(active)

	result, err := r.deps.Classifier.Classify(ctx, query, profiles)
	if err != nil {
		// Graceful degradation: route to everything and let the operator see the
		// classifier failed (never a hard error). A failed classify is never
		// treated as web-shaped — external opt-in requires a positive judgment.
		r.deps.Logger.Printf("routing.autoSelect: classifier failed, widening to all active providers: %v", err)
		return providersByType(active, available), []string{
			"automatic routing requested (no explicit selector)",
			fmt.Sprintf("classifier unavailable (%s) — widened to all active providers", oneLine(err.Error())),
		}, true, false
	}

	chosen, widened := widenPolicy(result, available)
	targets = providersByType(active, chosen)

	explain = []string{"automatic routing via classifier (no explicit selector)"}
	if r := strings.TrimSpace(result.Rationale); r != "" {
		explain = append(explain, "classifier rationale: "+r)
	}
	explain = append(explain, fmt.Sprintf("classifier confidence: %.2f", result.Confidence))
	routed := fmt.Sprintf("routed to types: %s", strings.Join(chosen, ", "))
	if widened {
		routed += " (widened on uncertainty — over-fetching for recall)"
	}
	explain = append(explain, routed)
	// Web-shaped only counts when the classifier is confident enough to justify
	// reaching a rate-limited/paid external corpus (a higher bar than widening).
	webShaped = result.WebShaped && result.Confidence >= autoExternalThreshold()
	return targets, explain, false, webShaped
}

// providersByType returns the active leaves whose type is in the chosen set,
// preserving the registry's provider_id order. Leaves without a callable
// endpoint are skipped defensively.
func providersByType(active []*registryv1.ProviderDescriptor, types []string) []*registryv1.ProviderDescriptor {
	want := make(map[string]struct{}, len(types))
	for _, t := range types {
		want[t] = struct{}{}
	}
	out := make([]*registryv1.ProviderDescriptor, 0, len(active))
	for _, p := range active {
		if p.GetEndpoint() == nil {
			continue
		}
		if _, ok := want[p.GetType()]; ok {
			out = append(out, p)
		}
	}
	return out
}

// matchedProvidersLine renders the final --explain line naming the leaves the
// router fanned out to (shared by the explicit and classifier paths' output).
func matchedProvidersLine(targets []*registryv1.ProviderDescriptor) string {
	if len(targets) == 0 {
		return "matched providers: none"
	}
	ids := make([]string, 0, len(targets))
	for _, t := range targets {
		ids = append(ids, t.GetProviderId())
	}
	return fmt.Sprintf("matched providers: %s", strings.Join(ids, ", "))
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

// partitionByScope splits the active providers into the project-scope candidate
// set the classifier/auto path may route to, and the SCOPE_EXTERNAL providers
// withheld from automatic routing. SCOPE_UNSPECIFIED is treated as project-scope
// (a provider that never declared a scope is assumed internal, not external) so
// a legacy provider keeps its default-routable behavior. Order is preserved.
func partitionByScope(active []*registryv1.ProviderDescriptor) (project, external []*registryv1.ProviderDescriptor) {
	for _, p := range active {
		if p.GetScope() == registryv1.Scope_SCOPE_EXTERNAL {
			external = append(external, p)
			continue
		}
		project = append(project, p)
	}
	return project, external
}

// withheldExternalLine is the --explain note emitted when external providers
// were kept out of the automatic candidate set.
func withheldExternalLine(external []*registryv1.ProviderDescriptor) string {
	ids := make([]string, 0, len(external))
	for _, p := range external {
		ids = append(ids, p.GetProviderId())
	}
	return fmt.Sprintf("withheld %d external provider(s) from auto-routing (reach with --all or --type <type>): %s",
		len(external), strings.Join(ids, ", "))
}

// autoRoutedExternalLine is the --explain note emitted when a web-shaped query
// (with the operator opt-in) folded the external provider(s) into auto-routing.
func autoRoutedExternalLine(external []*registryv1.ProviderDescriptor) string {
	return fmt.Sprintf("auto-routed %d external provider(s) — query judged web-shaped (opt-in enabled): %s",
		len(external), strings.Join(providerIDs(external), ", "))
}

// escalationLine is the --explain note emitted when the project corpus returned
// no hits and the router escalated to the withheld external provider(s).
func escalationLine(external []*registryv1.ProviderDescriptor) string {
	return fmt.Sprintf("escalated to %d external provider(s) — project results were empty (opt-in enabled): %s",
		len(external), strings.Join(providerIDs(external), ", "))
}

// providerIDs projects descriptors to their provider_id list.
func providerIDs(ps []*registryv1.ProviderDescriptor) []string {
	ids := make([]string, 0, len(ps))
	for _, p := range ps {
		ids = append(ids, p.GetProviderId())
	}
	return ids
}

// resultsWeak reports whether the fanned-out groups produced no usable hits: a
// non-degraded group with at least one hit makes the round non-weak. It is the
// trigger for OT-P2-002 fallback escalation (empty/weak project corpus).
func resultsWeak(groups []*routingv1.ProviderResultGroup) bool {
	for _, g := range groups {
		if !g.GetDegraded() && len(g.GetHits()) > 0 {
			return false
		}
	}
	return true
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

	body, err := providers.RenderBody(hj.GetBodyTemplate(), query, limit, d.GetType())
	if err != nil {
		return degrade(g, fmt.Sprintf("request build failed: %s", err))
	}

	cctx, cancel := context.WithTimeout(ctx, r.deps.PerProviderTimeout)
	defer cancel()

	url := strings.TrimRight(base, "/") + hj.GetPath()
	httpReq, err := http.NewRequestWithContext(cctx, providers.HTTPMethod(hj.GetMethod()), url, bytes.NewReader([]byte(body)))
	if err != nil {
		return degrade(g, fmt.Sprintf("request build failed: %s", oneLine(err.Error())))
	}
	providers.ApplyHeaders(httpReq, hj.GetHeaders())

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

// explain renders the human-readable routing rationale for --explain on the
// explicit-selector path: the caller named --all/--type/--group, so the
// explanation states which selector chose the targets (the classifier path has
// its own explanation, built in autoSelect).
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
		"explicit routing (selector provided; classifier not consulted)",
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
