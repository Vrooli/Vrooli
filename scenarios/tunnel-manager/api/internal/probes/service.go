package probes

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"tunnel-manager/internal/httpc"
	"tunnel-manager/internal/manifest"

	"github.com/vrooli/api-core/schedule"
)

// RoutesReader is the narrow slice of the routes domain this service
// depends on: read the manifest to learn which routes to probe. Declared
// at the consumer (seam-discovery) and satisfied by routes.Service.
type RoutesReader interface {
	List(ctx context.Context, tier manifest.Tier) ([]manifest.Route, error)
}

// Service is the application-layer surface the probes handlers depend on.
// Owns the probe cycle, persistence, and classification policy. The
// handler is intentionally thin around it: decode → call service →
// translate errors.
type Service interface {
	// RunProbes executes one probe cycle across every enabled route —
	// internal (local port) and external (public URL) concurrently —
	// persists each result, and returns them. A persistence failure on a
	// single result is logged-by-omission (never fails the cycle); the
	// returned slice always reflects what was probed.
	RunProbes(ctx context.Context) ([]ProbeResult, error)

	// ListProbes returns recent probe history newest-first, optionally
	// filtered by subdomain. limit <= 0 applies the repository default.
	ListProbes(ctx context.Context, subdomain string, limit int) ([]ProbeResult, error)

	// Classify derives the per-route reachability diagnosis from the
	// latest stored internal+external probes.
	Classify(ctx context.Context) ([]RouteClassification, error)
}

// cycleRepository is the optional batching seam implemented by the production
// SQLite repository. Keeping it optional preserves the small Repository
// contract used by service tests and other implementations.
type cycleRepository interface {
	PersistWithoutPrune(context.Context, ProbeResult) (ProbeResult, error)
	PruneBefore(context.Context, time.Time) error
}

type service struct {
	routes RoutesReader
	repo   Repository
	doer   httpc.Doer
	clock  schedule.Clock
}

// NewService constructs the production Service. doer is the outbound HTTP
// seam (a timeout-bounded *http.Client in production); clock stamps
// latency so tests stay deterministic.
func NewService(routes RoutesReader, repo Repository, doer httpc.Doer, clk schedule.Clock) Service {
	return &service{routes: routes, repo: repo, doer: doer, clock: clk}
}

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) RunProbes(ctx context.Context) ([]ProbeResult, error) {
	routes, err := s.routes.List(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("probes: list routes: %w", err)
	}

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		results []ProbeResult
	)

	for _, route := range routes {
		if !route.Enabled {
			continue
		}
		wg.Add(1)
		go func(r manifest.Route) {
			defer wg.Done()

			internal := s.probe(ctx, r.Subdomain, ProbeKindInternal,
				fmt.Sprintf("http://localhost:%d%s", r.LocalPort, r.HealthPath))
			external := s.probe(ctx, r.Subdomain, ProbeKindExternal,
				r.PublicURL()+r.HealthPath)

			// Persist best-effort: a write failure must not lose the other
			// route's results, so swallow the error here and keep the
			// persisted ID (or zero) on the returned result.
			if stored, err := s.persistForCycle(ctx, internal); err == nil {
				internal = stored
			}
			if stored, err := s.persistForCycle(ctx, external); err == nil {
				external = stored
			}

			mu.Lock()
			results = append(results, internal, external)
			mu.Unlock()
		}(route)
	}

	wg.Wait()
	if batched, ok := s.repo.(cycleRepository); ok {
		if err := batched.PruneBefore(ctx, s.clock.Now().Add(-HistoryRetentionWindow)); err != nil {
			return results, err
		}
	}
	return results, nil
}

func (s *service) persistForCycle(ctx context.Context, result ProbeResult) (ProbeResult, error) {
	if batched, ok := s.repo.(cycleRepository); ok {
		return batched.PersistWithoutPrune(ctx, result)
	}
	return s.repo.Persist(ctx, result)
}

// probe issues a single GET and maps the outcome onto a ProbeResult.
// Status is up (<400), down (>=400 or transport error), timeout (context
// error), or error (request could not be built).
func (s *service) probe(ctx context.Context, subdomain string, kind ProbeKind, url string) ProbeResult {
	start := s.clock.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ProbeResult{Subdomain: subdomain, Kind: kind, Status: ProbeStatusError, ErrorMsg: err.Error()}
	}

	resp, err := s.doer.Do(req)
	latency := int(s.clock.Now().Sub(start).Milliseconds())
	if err != nil {
		status := ProbeStatusDown
		if ctx.Err() != nil {
			status = ProbeStatusTimeout
		}
		return ProbeResult{Subdomain: subdomain, Kind: kind, Status: status, LatencyMS: latency, ErrorMsg: err.Error()}
	}
	defer resp.Body.Close()

	status := ProbeStatusUp
	if resp.StatusCode >= 400 {
		status = ProbeStatusDown
	}
	return ProbeResult{Subdomain: subdomain, Kind: kind, Status: status, LatencyMS: latency, StatusCode: resp.StatusCode}
}

func (s *service) ListProbes(ctx context.Context, subdomain string, limit int) ([]ProbeResult, error) {
	return s.repo.List(ctx, subdomain, limit)
}

func (s *service) Classify(ctx context.Context) ([]RouteClassification, error) {
	pairs, err := s.repo.LatestPerRoute(ctx)
	if err != nil {
		return nil, fmt.Errorf("probes: latest per route: %w", err)
	}
	out := make([]RouteClassification, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, classify(p))
	}
	return out, nil
}

// classify maps the latest internal+external probe pair onto a
// FailureClass, following the proto enum's documented meanings:
//
//   - both up                 → HEALTHY (fully operational).
//   - internal up, external !up → TUNNEL_DOWN: the scenario is running
//     locally but is not reachable through the tunnel — the failure is in
//     the tunnel/edge layer, not the scenario.
//   - internal !up, external up → CONFIG_DRIFT: the public URL still
//     resolves and serves while the local port does not, so the manifest
//     port/path no longer matches where the scenario actually listens (a
//     drifted local config), rather than the scenario being wholly down.
//   - both !up                → SCENARIO_DOWN: nothing answers on either
//     surface; the scenario itself is the most likely culprit.
//
// CLOUDFLARE_OUTAGE and DNS_FAILURE require signals not available from a
// single internal/external GET pair (edge status, resolver errors) and so
// are not produced here — they remain in the enum for richer probes.
func classify(p LatestPair) RouteClassification {
	c := RouteClassification{Subdomain: p.Subdomain}
	internalUp := p.Internal != nil && p.Internal.Status == ProbeStatusUp
	externalUp := p.External != nil && p.External.Status == ProbeStatusUp
	if p.Internal != nil {
		c.Internal = p.Internal.Status
	}
	if p.External != nil {
		c.External = p.External.Status
	}

	switch {
	case internalUp && externalUp:
		c.Classification = FailureClassHealthy
		c.Assessment = "route is fully operational (internal and external both up)"
	case internalUp && !externalUp:
		c.Classification = FailureClassTunnelDown
		c.Assessment = "scenario is running locally but not reachable via the tunnel"
	case !internalUp && externalUp:
		c.Classification = FailureClassConfigDrift
		c.Assessment = "external still resolves but the local port does not answer; manifest port/path likely drifted"
	default:
		c.Classification = FailureClassScenarioDown
		c.Assessment = "both internal and external probes failed; the scenario itself is the likely culprit"
	}
	return c
}
