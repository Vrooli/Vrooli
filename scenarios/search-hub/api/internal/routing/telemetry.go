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
// the raw text to telemetry (the plan's privacy note). ProviderHits maps each
// provider_id the query fanned out to → that leaf's hit count (a degraded leaf
// records 0). This struct is owned by the consumer (routing) per seam-discovery;
// the metrics store is bridged to it at the wiring edge so internal/metrics
// never imports internal/routing.
type TelemetrySample struct {
	QueryHash    string
	RoutedTypes  []string
	ProviderHits map[string]int
	ResultCount  int
	Degraded     bool
	Reranked     bool
	LatencyMs    int64
	// AutoRoutedExternal is true when the automatic path folded a SCOPE_EXTERNAL
	// provider into the fan-out because the query was judged web-shaped
	// (OT-P2-002). Lets the metrics surface measure the auto-routed-external rate.
	AutoRoutedExternal bool
	// Escalated is true when the project corpus was empty/weak and the router
	// escalated to a withheld external provider (OT-P2-002 fallback). Lets the
	// metrics surface measure the escalation rate.
	Escalated bool
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
// of types actually fanned out to, the per-provider hit counts, and the
// response-level flags. It does not include the routing latency — the caller
// stamps that from the response so recording time is never counted.
func buildSample(query string, targets []*registryv1.ProviderDescriptor, resp *routingv1.QueryResponse) TelemetrySample {
	providerHits := make(map[string]int, len(resp.GetGroups()))
	total := 0
	for _, g := range resp.GetGroups() {
		n := len(g.GetHits())
		providerHits[g.GetProviderId()] = n
		total += n
	}

	return TelemetrySample{
		QueryHash:    hashQuery(query),
		RoutedTypes:  uniqueTypes(targets),
		ProviderHits: providerHits,
		ResultCount:  total,
		Degraded:     resp.GetDegraded(),
		Reranked:     resp.GetReranked(),
		LatencyMs:    resp.GetLatencyMs(),
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
