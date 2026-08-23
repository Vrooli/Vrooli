package routing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	internalregistry "search-hub/internal/registry"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	statusCacheTTL          = 5 * time.Second
	statusProbeDefault      = 2 * time.Second
	statusProbeFloor        = 250 * time.Millisecond
	statusProbeCeiling      = 2 * time.Second
	statusProbeLatencyScale = 2
)

// Status reports federation health (Phase 7): per-provider reachability plus
// whether the classifier and reranker models are available. It lists the ACTIVE
// leaves (capability_gap stubs carry no endpoint and are excluded), resolves
// each leaf's scenario URL to gauge reachability, and probes the optional
// Classifier/Reranker Available seams.
//
// It never fails on an individual provider — an unresolvable leaf is reported
// as unreachable+degraded rather than erroring the whole call. It returns an
// error only on a registry read failure.
func (r *Router) Status(ctx context.Context) (*routingv1.StatusResponse, error) {
	r.providerBreakers.restore(ctx)
	r.statusMu.Lock()
	if r.statusCache != nil && time.Since(r.statusCacheAt) < statusCacheTTL {
		cached := proto.Clone(r.statusCache).(*routingv1.StatusResponse)
		r.statusMu.Unlock()
		return cached, nil
	}
	r.statusMu.Unlock()

	active, err := r.deps.Lister.List(ctx, internalregistry.ListFilter{
		State: int32(registryv1.ProviderState_PROVIDER_STATE_ACTIVE),
	})
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}

	// Provider status probes are independent, but the worker pool is bounded by
	// the same concurrency factor as query fan-out. This keeps a fleet-sized
	// status read from creating an unbounded goroutine/probe burst.
	health := make([]*routingv1.ProviderHealth, len(active))
	workerCount := r.deps.Concurrency
	if workerCount <= 0 {
		workerCount = r.factors.Concurrency
	}
	if workerCount > len(active) {
		workerCount = len(active)
	}
	jobs := make(chan int)
	var wg sync.WaitGroup
	for worker := 0; worker < workerCount; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				health[i] = r.providerHealth(ctx, active[i])
			}
		}()
	}
	for i := range active {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	resp := &routingv1.StatusResponse{Providers: health}
	resp.ActiveStrategy = r.strategy.Name
	resp.Strategies = strategyInfos(r.strategies)
	registered := make(map[string]struct{}, len(active))
	for _, provider := range active {
		registered[provider.GetProviderId()] = struct{}{}
	}
	for providerID, stats := range r.providerBreakers.states() {
		if _, ok := registered[providerID]; ok {
			continue
		}
		resp.AuditProviders = append(resp.AuditProviders, &routingv1.ProviderHealth{
			ProviderId: providerID, Reachable: false, Degraded: true,
			AutomaticEligible: false, AutomaticExclusionReason: "unregistered accounting key; retained for audit",
			Reachability: "no registered provider descriptor", CircuitState: "unregistered",
			RecoveryState: "unregistered", TimesRouted: stats.routed, TotalHits: stats.hits,
			Stuck: proto.Bool(false), Lifecycle: "unregistered",
		})
	}
	for i, p := range active {
		if p.GetLifecycle() != registryv1.Lifecycle_LIFECYCLE_EXPERIMENTAL || health[i] == nil {
			continue
		}
		item := &registryv1.IncubatingProvider{
			ProviderId:  p.GetProviderId(),
			DeclaredAt:  p.GetDeclaredAt(),
			TimesRouted: health[i].GetTimesRouted(),
			TotalHits:   health[i].GetTotalHits(),
		}
		if r.deps.EvalQuality != nil {
			if evidence, evidenceErr := r.deps.EvalQuality.LatestProviderEval(ctx, p.GetProviderId()); evidenceErr == nil {
				item.SuitePresent = evidence.SuitePresent
				item.NextAction = incubatingNextAction(evidence)
			}
		}
		if item.GetNextAction() == "" {
			item.NextAction = "establish a reviewed suite and passing evidence"
		}
		resp.Incubating = append(resp.Incubating, item)
	}
	// The proto field is retained for wire compatibility with older clients;
	// the interactive LLM classifier was retired in favor of the lexical
	// strategy, so it is always false.
	resp.ClassifierAvailable = false
	resp.RerankerAvailable = r.deps.Reranker != nil && r.deps.Reranker.Available(ctx)
	resp.RerankerLeg = activeRerankerLeg(ctx, r.deps.Reranker)
	open, breached, err := r.CircuitOpenQuorum(ctx)
	if err != nil {
		return nil, err
	}
	resp.CircuitOpenShare = open
	resp.CircuitOpenQuorum = CircuitOpenQuorumThreshold
	resp.FederationDegraded = breached
	r.statusMu.Lock()
	r.statusCache = proto.Clone(resp).(*routingv1.StatusResponse)
	r.statusCacheAt = time.Now()
	r.statusMu.Unlock()
	return resp, nil
}

func strategyInfos(catalog map[string]RetrievalStrategy) []*routingv1.RetrievalStrategyInfo {
	items := make([]RetrievalStrategy, 0, len(catalog))
	for _, strategy := range catalog {
		items = append(items, strategy)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	result := make([]*routingv1.RetrievalStrategyInfo, 0, len(items))
	for _, strategy := range items {
		info := &routingv1.RetrievalStrategyInfo{Name: strategy.Name, Description: strategy.Description}
		for _, stage := range strategy.Stages {
			params, _ := json.Marshal(stage.Params)
			info.Stages = append(info.Stages, &routingv1.RetrievalStageInfo{Kind: string(stage.Kind), ParamsJson: string(params)})
		}
		result = append(result, info)
	}
	return result
}

func activeRerankerLeg(ctx context.Context, reranker Reranker) string {
	if reranker == nil {
		return "none"
	}
	if named, ok := reranker.(interface{ ActiveName(context.Context) string }); ok {
		if leg := strings.TrimSpace(named.ActiveName(ctx)); leg != "" {
			return leg
		}
	}
	return "none"
}

// providerHealth resolves one leaf's reachability and, only on this status
// path, probes the provider-declared status endpoint for index age. The probe
// is deliberately absent from Query, so status inspection cannot add query
// latency or turn Search Hub into the owner of provider corpus state.
func (r *Router) providerHealth(ctx context.Context, p *registryv1.ProviderDescriptor) *routingv1.ProviderHealth {
	h := &routingv1.ProviderHealth{ProviderId: p.GetProviderId(), AutomaticEligible: true, Stuck: proto.Bool(false), Lifecycle: lifecycleName(p.GetLifecycle()), DeclaredAt: p.GetDeclaredAt()}
	budget := effectiveFreshnessBudget(p)
	h.FreshnessBudget = budget.String()
	if tuning := p.GetTuning(); tuning != nil {
		h.EmbeddingModel = tuning.GetEmbedModel()
	}
	h.RecoveryState = "healthy"
	h.CircuitState = "closed"
	if reason := strings.TrimSpace(p.GetJunkLeakOptOutReason()); reason != "" {
		h.QualityGateOptedOut = true
		h.QualityGateOptOutReason = reason
	}
	if lifecycle := p.GetLifecycle(); lifecycle != registryv1.Lifecycle_LIFECYCLE_UNSPECIFIED && lifecycle != registryv1.Lifecycle_LIFECYCLE_PRODUCTION {
		h.AutomaticEligible = false
		h.AutomaticExclusionReason = fmt.Sprintf("lifecycle=%s; explicit selector required", lifecycleName(lifecycle))
	}
	if r.deps.EvalQuality != nil {
		if evidence, err := r.deps.EvalQuality.LatestProviderEval(ctx, p.GetProviderId()); err == nil {
			if evidence.EvidenceAvailable {
				if reason := automaticEvidenceExclusion(evidence); reason != "" {
					h.AutomaticEligible = false
					h.AutomaticExclusionReason = "evidence: " + reason
				}
			}
			if evidence.Fresh && !evidence.Degraded && QualityJunkLeak(evidence.MaxGibberishScore, evidence.MeanStrongTop1) && !h.QualityGateOptedOut {
				h.AutomaticEligible = false
				h.QualityWithheld = true
				h.QualityEvidenceRunId = evidence.RunID
				h.QualityWithheldReason = fmt.Sprintf("withheld (junk leak): gibberish=%.3f exceeds strongest real=%.3f by at least %.3f", evidence.MaxGibberishScore, evidence.MeanStrongTop1, QualityJunkLeakMargin)
				h.AutomaticExclusionReason = h.QualityWithheldReason + "; run=" + evidence.RunID
			}
		}
	}
	r.providerBreakers.clearExpiredProbation(p.GetProviderId(), r.deps.Now())
	h.Demoted, h.TimesRouted, h.TotalHits = r.providerBreakers.demotion(p.GetProviderId())
	stats := r.providerBreakers.state(p.GetProviderId())
	var stuck bool
	h.RecoveryState, stuck = providerRecoveryState(stats, h.Demoted, r.deps.Now())
	h.Stuck = proto.Bool(stuck)
	if h.Demoted {
		h.AutomaticEligible = false
		deadline := r.providerBreakers.demotionDeadline(p.GetProviderId())
		if stats != nil && !deadline.IsZero() && !r.deps.Now().Before(deadline) && !stats.probation {
			h.AutomaticEligible = true
			h.AutomaticExclusionReason = "graded-empty demotion decay elapsed; next automatic route is a recovery probe"
		}
		emptyStreak := int64(0)
		if stats != nil {
			emptyStreak = stats.emptyStreak
		}
		h.DemotionReason = fmt.Sprintf("demoted from automatic routing after %d successful empty responses (demotion-window routed=%d, demotion-window hits=%d); decay deadline=%s", emptyStreak, h.TimesRouted, h.TotalHits, deadline.Format(time.RFC3339))
		if stats != nil && stats.trigger != "" {
			h.DemotionReason += "; trigger=" + stats.trigger
		}
		if h.AutomaticExclusionReason == "" {
			h.AutomaticExclusionReason = h.DemotionReason
		}
	}
	if open, note := r.providerBreakers.status(p.GetProviderId(), r.deps.Now()); open {
		h.Degraded = true
		if strings.Contains(note, "recovery probe is due") {
			h.Reachable = true
			h.CircuitState = "probe_due"
			h.Reachability = "circuit_cooldown_elapsed_probe_due"
		} else {
			h.Reachable = false
			h.CircuitState = "open"
			h.Reachability = "circuit_open"
		}
		h.IndexAge = "unreported: provider circuit unavailable"
		return h
	}

	hj := p.GetEndpoint().GetHttpJson()
	if hj == nil {
		h.Reachable = false
		h.Degraded = true
		h.Reachability = "no http endpoint registered"
		h.IndexAge = "unreported: no search endpoint registered"
		return h
	}

	if _, err := r.deps.Resolver.ResolveScenarioURL(ctx, hj.GetScenarioId()); err != nil {
		h.Reachable = false
		h.Degraded = true
		h.Reachability = fmt.Sprintf("scenario %q unreachable: %s", hj.GetScenarioId(), oneLine(err.Error()))
		h.IndexAge = "unreported: provider unreachable"
		return h
	}

	h.Reachable = true
	h.Reachability = "endpoint resolved"
	snapshot := r.probeIndexStatus(ctx, p)
	h.IndexAge, h.PointCount = snapshot.age, snapshot.documents
	h.ActiveGeneration, h.SourceFiles = snapshot.generation, snapshot.sourceFiles
	h.SemanticCards, h.GraphFacts = snapshot.semanticCards, snapshot.graphFacts
	h.IndexState = snapshot.state
	h.DegradedStages = append([]string(nil), snapshot.degraded...)
	h.Drifted = snapshot.drifted
	if snapshot.state == "uninitialized" || len(snapshot.degraded) > 0 {
		h.Degraded = true
	}
	lastIndexedAt := snapshot.timestamp
	if !lastIndexedAt.IsZero() {
		h.LastIndexedAt = timestamppb.New(lastIndexedAt)
		age := r.deps.Now().Sub(lastIndexedAt)
		if age > budget {
			h.AutomaticEligible = false
			h.AutomaticExclusionReason = fmt.Sprintf("stale index: age=%s exceeds freshness budget=%s", age.Round(time.Second), budget)
		}
	}
	return h
}

func effectiveFreshnessBudget(p *registryv1.ProviderDescriptor) time.Duration {
	if p != nil && p.GetFreshnessBudget() != nil {
		if budget := p.GetFreshnessBudget().AsDuration(); budget > 0 {
			return budget
		}
	}
	return internalregistry.DefaultFreshnessBudget
}

func incubatingNextAction(evidence EvalQualityEvidence) string {
	if !evidence.SuitePresent {
		return "register a reviewed provider suite"
	}
	if !evidence.LiveReviewedPositive {
		return "add a live reviewed positive case"
	}
	if evidence.CorpusAllStale {
		return "re-anchor stale reviewed positives"
	}
	if !evidence.RecentPassingRun {
		return "run a recent passing evaluation"
	}
	return "declare production lifecycle"
}

func lifecycleName(value registryv1.Lifecycle) string {
	switch value {
	case registryv1.Lifecycle_LIFECYCLE_PRODUCTION:
		return "production"
	case registryv1.Lifecycle_LIFECYCLE_FIXTURE:
		return "fixture"
	case registryv1.Lifecycle_LIFECYCLE_EXPERIMENTAL:
		return "experimental"
	default:
		return "unspecified"
	}
}

func providerRecoveryState(stats *providerYieldStats, demoted bool, now time.Time) (string, bool) {
	if !demoted || stats == nil {
		return "healthy", false
	}
	if stats.probation {
		if !stats.decayDeadline.IsZero() && !now.Before(stats.decayDeadline) {
			return "stuck", true
		}
		return "probing", false
	}
	if !stats.decayDeadline.IsZero() && !now.Before(stats.decayDeadline) {
		return "probe_due", false
	}
	return "demoted", false
}

type providerIndexSnapshot struct {
	age           string
	documents     int64
	timestamp     time.Time
	generation    string
	sourceFiles   int64
	semanticCards int64
	graphFacts    int64
	state         string
	degraded      []string
	drifted       bool
}

func (r *Router) probeIndexStatus(ctx context.Context, p *registryv1.ProviderDescriptor) providerIndexSnapshot {
	result := providerIndexSnapshot{}
	hj := p.GetStatusEndpoint().GetHttpJson()
	if hj == nil {
		result.age = "not_applicable: provider has no status_endpoint"
		return result
	}
	if r.deps.Doer == nil {
		result.age = "unreported: status probe transport unavailable"
		return result
	}
	base, err := r.deps.Resolver.ResolveScenarioURL(ctx, hj.GetScenarioId())
	if err != nil {
		result.age = fmt.Sprintf("unreported: status endpoint unreachable: %s", oneLine(err.Error()))
		return result
	}
	body := strings.TrimSpace(hj.GetBodyTemplate())
	if body == "" {
		body = "{}"
	}
	probeCtx, cancel := context.WithTimeout(ctx, r.statusProbeTimeout(p.GetProviderId()))
	defer cancel()
	started := time.Now()
	defer func() { r.observeStatusProbe(p.GetProviderId(), time.Since(started)) }()
	url := strings.TrimRight(base, "/") + hj.GetPath()
	req, err := http.NewRequestWithContext(probeCtx, httpMethod(hj.GetMethod()), url, bytes.NewReader([]byte(body)))
	if err != nil {
		result.age = fmt.Sprintf("unreported: status request invalid: %s", oneLine(err.Error()))
		return result
	}
	for key, value := range hj.GetHeaders() {
		req.Header.Set(key, value)
	}
	resp, err := r.deps.Doer.Do(req)
	if err != nil {
		result.age = fmt.Sprintf("unreported: status probe failed: %s", oneLine(err.Error()))
		return result
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		result.age = fmt.Sprintf("unreported: status probe returned HTTP %d", resp.StatusCode)
		return result
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		result.age = fmt.Sprintf("unreported: status response unreadable: %s", oneLine(err.Error()))
		return result
	}
	readProviderIndexFields(raw, &result)
	timestamp, pointCount, ok := parseDeclaredIndexStatus(raw, p.GetIndexTimestampField())
	if !ok {
		if r.deps.Logger != nil {
			field := p.GetIndexTimestampField()
			if field == "" {
				field = "<missing declaration>"
			}
			r.deps.Logger.Printf("provider %q did not return declared index timestamp field %q; using deprecated key-sniff fallback", p.GetProviderId(), field)
		}
		timestamp, pointCount, ok = parseIndexStatus(raw)
	}
	if !ok {
		result.age, result.documents = "unreported: status response has no usable declared index timestamp", pointCount
		return result
	}
	age := r.deps.Now().Sub(timestamp)
	if age < 0 {
		age = 0
	}
	result.age, result.timestamp = age.Round(time.Second).String(), timestamp
	if result.documents == 0 {
		result.documents = pointCount
	}
	return result
}

func readProviderIndexFields(raw []byte, result *providerIndexSnapshot) {
	var payload map[string]any
	if result == nil || json.Unmarshal(raw, &payload) != nil {
		return
	}
	result.generation = firstStatusString(payload, "activeGeneration", "active_generation")
	result.state = firstStatusString(payload, "state", "indexState", "index_state")
	result.documents = firstStatusInt(payload, "searchDocuments", "search_documents", "indexedCount", "indexed_count")
	result.sourceFiles = firstStatusInt(payload, "sourceFiles", "source_files")
	result.semanticCards = firstStatusInt(payload, "semanticCards", "semantic_cards")
	result.graphFacts = firstStatusInt(payload, "graphFacts", "graph_facts")
	for _, key := range []string{"degradedStages", "degraded_stages"} {
		if values, ok := payload[key].([]any); ok {
			for _, value := range values {
				if stage, ok := value.(string); ok && strings.TrimSpace(stage) != "" {
					result.degraded = append(result.degraded, stage)
					if strings.Contains(strings.ToLower(stage), "drift") {
						result.drifted = true
					}
				}
			}
		}
	}
	if value, ok := payload["drifted"].(bool); ok {
		result.drifted = value
	}
}

func firstStatusString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := payload[key].(string); ok {
			return value
		}
	}
	return ""
}

func firstStatusInt(payload map[string]any, keys ...string) int64 {
	for _, key := range keys {
		if value, ok := payload[key].(float64); ok {
			return int64(value)
		}
		if value, ok := payload[key].(string); ok {
			if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
				return parsed
			}
		}
	}
	return 0
}

func (r *Router) statusProbeTimeout(providerID string) time.Duration {
	r.statusMu.Lock()
	latency := r.probeLatencies[providerID]
	r.statusMu.Unlock()
	if latency <= 0 {
		return statusProbeDefault
	}
	timeout := time.Duration(statusProbeLatencyScale) * latency
	if timeout < statusProbeFloor {
		return statusProbeFloor
	}
	if timeout > statusProbeCeiling {
		return statusProbeCeiling
	}
	return timeout
}

func (r *Router) observeStatusProbe(providerID string, latency time.Duration) {
	if latency <= 0 {
		return
	}
	r.statusMu.Lock()
	defer r.statusMu.Unlock()
	if previous := r.probeLatencies[providerID]; previous > 0 {
		r.probeLatencies[providerID] = (previous + latency) / 2
	} else {
		r.probeLatencies[providerID] = latency
	}
}

func httpMethod(method registryv1.HttpMethod) string {
	switch method {
	case registryv1.HttpMethod_HTTP_METHOD_GET:
		return http.MethodGet
	default:
		return http.MethodPost
	}
}

func parseIndexStatus(raw []byte) (time.Time, int64, bool) {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return time.Time{}, 0, false
	}
	return parseIndexStatusMap(payload)
}

func parseDeclaredIndexStatus(raw []byte, field string) (time.Time, int64, bool) {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return time.Time{}, 0, false
	}
	field = strings.TrimSpace(field)
	if field == "" {
		return time.Time{}, 0, false
	}
	var value any = payload
	for _, part := range strings.Split(field, ".") {
		object, ok := value.(map[string]any)
		if !ok {
			return time.Time{}, 0, false
		}
		value, ok = object[part]
		if !ok {
			return time.Time{}, 0, false
		}
	}
	timestamp, ok := parseTimestamp(value)
	return timestamp, 0, ok
}

func parseIndexStatusMap(payload map[string]any) (time.Time, int64, bool) {
	var pointCount int64
	for _, key := range []string{"point_count", "pointCount", "indexed_count", "indexedCount", "document_count", "documentCount"} {
		if value, ok := payload[key].(float64); ok {
			pointCount = int64(value)
			break
		}
	}
	for _, key := range []string{"last_indexed_at", "lastIndexedAt", "last_index_at", "lastIndexAt", "index_updated_at", "indexUpdatedAt", "indexed_at", "indexedAt", "last_reindex_at", "lastReindexAt"} {
		if value, ok := payload[key]; ok {
			if timestamp, ok := parseTimestamp(value); ok {
				return timestamp, pointCount, true
			}
		}
	}
	for _, key := range []string{"status", "index", "data"} {
		if nested, ok := payload[key].(map[string]any); ok {
			if timestamp, nestedCount, found := parseIndexStatusMap(nested); found {
				if pointCount == 0 {
					pointCount = nestedCount
				}
				return timestamp, pointCount, true
			}
		}
	}
	return time.Time{}, pointCount, false
}

func parseTimestamp(value any) (time.Time, bool) {
	switch value := value.(type) {
	case string:
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05Z07:00"} {
			if timestamp, err := time.Parse(layout, value); err == nil {
				return timestamp, true
			}
		}
		if unix, err := strconv.ParseInt(value, 10, 64); err == nil {
			if unix > 1e12 {
				return time.UnixMilli(unix).UTC(), true
			}
			if unix > 1e9 {
				return time.Unix(unix, 0).UTC(), true
			}
		}
	case float64:
		if value > 1e12 {
			return time.UnixMilli(int64(value)).UTC(), true
		}
		if value > 1e9 {
			return time.Unix(int64(value), 0).UTC(), true
		}
	}
	return time.Time{}, false
}
