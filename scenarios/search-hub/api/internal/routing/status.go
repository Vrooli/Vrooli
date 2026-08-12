package routing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	internalregistry "search-hub/internal/registry"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

const statusProbeTimeout = 2 * time.Second

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
	active, err := r.deps.Lister.List(ctx, internalregistry.ListFilter{
		State: int32(registryv1.ProviderState_PROVIDER_STATE_ACTIVE),
	})
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}

	// Provider status probes are independent. Running them concurrently keeps
	// the typed readiness read bounded by the slowest provider rather than the
	// sum of every unavailable provider's probe timeout. The result slice keeps
	// registry order stable for deterministic operator output.
	health := make([]*routingv1.ProviderHealth, len(active))
	var wg sync.WaitGroup
	for i, p := range active {
		wg.Add(1)
		go func(i int, p *registryv1.ProviderDescriptor) {
			defer wg.Done()
			health[i] = r.providerHealth(ctx, p)
		}(i, p)
	}
	wg.Wait()
	resp := &routingv1.StatusResponse{Providers: health}
	resp.ClassifierAvailable = r.deps.Classifier != nil && r.deps.Classifier.Available(ctx)
	resp.RerankerAvailable = r.deps.Reranker != nil && r.deps.Reranker.Available(ctx)
	open, breached, err := r.CircuitOpenQuorum(ctx)
	if err != nil {
		return nil, err
	}
	resp.CircuitOpenShare = open
	resp.CircuitOpenQuorum = CircuitOpenQuorumThreshold
	resp.FederationDegraded = breached
	return resp, nil
}

// providerHealth resolves one leaf's reachability and, only on this status
// path, probes the provider-declared status endpoint for index age. The probe
// is deliberately absent from Query, so status inspection cannot add query
// latency or turn Search Hub into the owner of provider corpus state.
func (r *Router) providerHealth(ctx context.Context, p *registryv1.ProviderDescriptor) *routingv1.ProviderHealth {
	h := &routingv1.ProviderHealth{ProviderId: p.GetProviderId(), AutomaticEligible: true}
	h.CircuitState = "closed"
	if reason := strings.TrimSpace(p.GetJunkLeakOptOutReason()); reason != "" {
		h.QualityGateOptedOut = true
		h.QualityGateOptOutReason = reason
	}
	if lifecycle := strings.TrimSpace(p.GetLifecycle()); lifecycle != "" && lifecycle != "production" {
		h.AutomaticEligible = false
		h.AutomaticExclusionReason = fmt.Sprintf("lifecycle=%s; explicit selector required", lifecycle)
	}
	if r.deps.EvalQuality != nil {
		if evidence, err := r.deps.EvalQuality.LatestProviderEval(ctx, p.GetProviderId()); err == nil && evidence.Fresh && !evidence.Degraded && evidence.GibberishLeak && !h.QualityGateOptedOut {
			h.AutomaticEligible = false
			h.QualityWithheld = true
			h.QualityEvidenceRunId = evidence.RunID
			h.QualityWithheldReason = fmt.Sprintf("withheld (junk leak): gibberish=%.3f >= strongest real=%.3f", evidence.MaxGibberishScore, evidence.MeanStrongTop1)
			h.AutomaticExclusionReason = h.QualityWithheldReason + "; run=" + evidence.RunID
		}
	}
	h.Demoted, h.TimesRouted, h.TotalHits = r.providerBreakers.demotion(p.GetProviderId())
	if h.Demoted {
		h.AutomaticEligible = false
		deadline := r.providerBreakers.demotionDeadline(p.GetProviderId())
		if stats := r.providerBreakers.state(p.GetProviderId()); stats != nil && !deadline.IsZero() && !r.deps.Now().Before(deadline) && !stats.probation {
			h.AutomaticEligible = true
			h.AutomaticExclusionReason = "graded-empty demotion decay elapsed; next automatic route is a recovery probe"
		}
		h.DemotionReason = fmt.Sprintf("demoted from automatic routing after %d successful empty routes (hits=%d); decay deadline=%s", h.TimesRouted, h.TotalHits, deadline.Format(time.RFC3339))
		if stats := r.providerBreakers.state(p.GetProviderId()); stats != nil && stats.trigger != "" {
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
	h.IndexAge, h.PointCount = r.probeIndexStatus(ctx, p)
	return h
}

func (r *Router) probeIndexStatus(ctx context.Context, p *registryv1.ProviderDescriptor) (string, int64) {
	hj := p.GetStatusEndpoint().GetHttpJson()
	if hj == nil {
		return "not_applicable: provider has no status_endpoint", 0
	}
	if r.deps.Doer == nil {
		return "unreported: status probe transport unavailable", 0
	}
	base, err := r.deps.Resolver.ResolveScenarioURL(ctx, hj.GetScenarioId())
	if err != nil {
		return fmt.Sprintf("unreported: status endpoint unreachable: %s", oneLine(err.Error())), 0
	}
	body := strings.TrimSpace(hj.GetBodyTemplate())
	if body == "" {
		body = "{}"
	}
	probeCtx, cancel := context.WithTimeout(ctx, statusProbeTimeout)
	defer cancel()
	url := strings.TrimRight(base, "/") + hj.GetPath()
	req, err := http.NewRequestWithContext(probeCtx, httpMethod(hj.GetMethod()), url, bytes.NewReader([]byte(body)))
	if err != nil {
		return fmt.Sprintf("unreported: status request invalid: %s", oneLine(err.Error())), 0
	}
	for key, value := range hj.GetHeaders() {
		req.Header.Set(key, value)
	}
	resp, err := r.deps.Doer.Do(req)
	if err != nil {
		return fmt.Sprintf("unreported: status probe failed: %s", oneLine(err.Error())), 0
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Sprintf("unreported: status probe returned HTTP %d", resp.StatusCode), 0
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return fmt.Sprintf("unreported: status response unreadable: %s", oneLine(err.Error())), 0
	}
	timestamp, pointCount, ok := parseIndexStatus(raw)
	if !ok {
		return "unreported: status response has no usable last-index timestamp", pointCount
	}
	age := r.deps.Now().Sub(timestamp)
	if age < 0 {
		age = 0
	}
	return age.Round(time.Second).String(), pointCount
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
	case float64:
		if value > 1e12 {
			return time.UnixMilli(int64(value)), true
		}
		if value > 1e9 {
			return time.Unix(int64(value), 0), true
		}
	}
	return time.Time{}, false
}
