package routing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

// TelemetrySample is the per-query record the router emits for the metrics
// domain (Phase 7). QueryHash is the HASHED query text — the router never hands
// the raw text to telemetry (the plan's privacy note). ProviderResults maps
// each provider_id the query fanned out to → that leaf's hit count, latency, and
// normalized degradation category. This struct is owned by the consumer
// (routing) per seam-discovery;
// the metrics store is bridged to it at the wiring edge so internal/metrics
// never imports internal/routing.
type TelemetrySample struct {
	QueryHash       string
	RoutedTypes     []string
	ProviderResults map[string]ProviderTelemetry
	ResultCount     int
	Degraded        bool
	Reranked        bool
	LatencyMs       int64
	// RoutingMode and the fan-out counts explain why a query paid the latency it
	// did without storing its text. Eligible is the active fleet seen by the
	// router; selected is the final live fan-out (including a permitted
	// escalation); withheld_external is deliberately excluded automatic scope;
	// queued is the excess over the bounded fan-out worker budget.
	RoutingMode           string
	EligibleProviderCount int
	SelectedProviderCount int
	// SelectedLeafCount is the number of exact classifier selections before
	// uncertainty widening; WidenedLeafCount is the number added by sibling
	// widening or the bounded fallback. FanoutWidthBoundReached records whether
	// the configured automatic width prevented further widening.
	SelectedLeafCount       int
	WidenedLeafCount        int
	FanoutWidthBoundReached bool
	WithheldExternalCount   int
	QueuedProviderCount     int
	ClassifierLatencyMs     int64
	ResolverLatencyMs       int64
	ResolverCacheHits       int64
	ResolverCacheMisses     int64
	FanoutLatencyMs         int64
	RerankLatencyMs         int64
	RerankCandidateCount    int
	ResponseDegradeReason   string
	// AutoRoutedExternal is true when the automatic path folded a SCOPE_EXTERNAL
	// provider into the fan-out because the query was judged web-shaped
	// (OT-P2-002). Lets the metrics surface measure the auto-routed-external rate.
	AutoRoutedExternal bool
	// Escalated is true when the project corpus was empty/weak and the router
	// escalated to a withheld external provider (OT-P2-002 fallback). Lets the
	// metrics surface measure the escalation rate.
	Escalated bool
}

// ResponseDegradeReason records the response-level cause categories. It is a
// compact, closed vocabulary (comma-separated only when causes coexist), not
// a copy of unbounded provider error text.
func ResponseDegradeReason(classifierFailed, rerankerFailed bool, groups []*routingv1.ProviderResultGroup, resultCount int) string {
	reasons := make([]string, 0, 4)
	if classifierFailed {
		reasons = append(reasons, "classifier")
	}
	if rerankerFailed {
		reasons = append(reasons, "reranker")
	}
	for _, g := range groups {
		if g.GetDegraded() {
			reasons = append(reasons, "provider_leg")
			break
		}
	}
	if resultCount == 0 {
		reasons = append(reasons, "zero_result")
	}
	return strings.Join(reasons, ",")
}

// ProviderTelemetry is one provider leg's telemetry projection. DegradeReason
// is a closed category derived from ProviderResultGroup.Note; raw notes are not
// persisted because they are freeform provider output.
type ProviderTelemetry struct {
	HitCount      int
	LatencyMs     int64
	Degraded      bool
	DegradeReason string
}

// TelemetryRecorder is the telemetry write seam. Production wires a bridge over
// the metrics SQLite store; tests wire a fake. Record is best-effort by
// contract — it returns no error, so the router's hot path never has to handle
// a telemetry failure (the bridge logs and swallows the store error).
type TelemetryRecorder interface {
	Record(ctx context.Context, s TelemetrySample)
}

// hashQuery returns the SHA-256 hex of the trimmed query text. Telemetry stores
// only this hash so the table carries no recoverable user input while still
// letting repeated identical queries be correlated.
func hashQuery(query string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(query)))
	return hex.EncodeToString(sum[:])
}

// buildSample derives the telemetry row from a completed query: the unique set
// of types actually fanned out to, the per-provider leg telemetry, and the
// response-level flags. It does not include the routing latency — the caller
// stamps that from the response so recording time is never counted.
func buildSample(query string, targets []*registryv1.ProviderDescriptor, resp *routingv1.QueryResponse) TelemetrySample {
	providerResults := make(map[string]ProviderTelemetry, len(resp.GetGroups()))
	total := 0
	for _, g := range resp.GetGroups() {
		n := len(g.GetHits())
		degraded := g.GetDegraded()
		reason := ""
		if degraded {
			reason = classifyDegradeReason(g.GetNote())
		}
		providerResults[g.GetProviderId()] = ProviderTelemetry{
			HitCount:      n,
			LatencyMs:     g.GetLatencyMs(),
			Degraded:      degraded,
			DegradeReason: reason,
		}
		total += n
	}

	return TelemetrySample{
		QueryHash:       hashQuery(query),
		RoutedTypes:     uniqueTypes(targets),
		ProviderResults: providerResults,
		ResultCount:     total,
		Degraded:        resp.GetDegraded(),
		Reranked:        resp.GetReranked(),
		LatencyMs:       resp.GetLatencyMs(),
	}
}

func classifyDegradeReason(note string) string {
	note = strings.ToLower(strings.TrimSpace(note))
	switch {
	case note == "":
		return "other"
	case strings.Contains(note, "timeout"), strings.Contains(note, "deadline exceeded"):
		return "timeout"
	case strings.Contains(note, "unreachable"), strings.Contains(note, "connection refused"),
		strings.Contains(note, "no such host"), strings.Contains(note, "not running"):
		return "unreachable"
	case strings.Contains(note, "http "):
		return "http_error"
	case strings.Contains(note, "reranker"):
		return "reranker_unavailable"
	default:
		return "other"
	}
}

// uniqueTypes returns the distinct leaf types among the fanned-out targets, in
// first-seen (provider_id) order, so telemetry records what the query routed to
// regardless of explicit vs classifier selection.
func uniqueTypes(targets []*registryv1.ProviderDescriptor) []string {
	seen := make(map[string]struct{}, len(targets))
	out := make([]string, 0, len(targets))
	for _, t := range targets {
		typ := t.GetType()
		if typ == "" {
			continue
		}
		if _, ok := seen[typ]; ok {
			continue
		}
		seen[typ] = struct{}{}
		out = append(out, typ)
	}
	return out
}
