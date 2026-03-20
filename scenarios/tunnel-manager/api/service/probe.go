package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"tunnel-manager/domain"
)

// ProbeService runs liveness probes against published routes.
type ProbeService struct {
	routeLister RouteLister
	probeWriter ProbeResultWriter
	httpClient  *http.Client
}

func NewProbeService(routeLister RouteLister, probeWriter ProbeResultWriter) *ProbeService {
	return &ProbeService{
		routeLister: routeLister,
		probeWriter: probeWriter,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// RunAll executes internal and external probes for all enabled routes concurrently.
func (ps *ProbeService) RunAll(ctx context.Context) ([]domain.ProbeResult, error) {
	routes, err := ps.routeLister.List()
	if err != nil {
		return nil, fmt.Errorf("probes: %w", err)
	}

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		results []domain.ProbeResult
	)

	for _, route := range routes {
		if !route.Enabled {
			continue
		}
		wg.Add(1)
		go func(r domain.Route) {
			defer wg.Done()
			// Internal probe (local port)
			internal := ps.probeInternal(ctx, r)
			// External probe (public URL)
			external := ps.probeExternal(ctx, r)

			mu.Lock()
			results = append(results, internal, external)
			mu.Unlock()

			// Persist results (log errors but don't fail the probe cycle)
			if err := ps.probeWriter.PersistResult(internal); err != nil {
				slog.Error("persist probe result", "subdomain", r.Subdomain, "type", "internal", "error", err)
			}
			if err := ps.probeWriter.PersistResult(external); err != nil {
				slog.Error("persist probe result", "subdomain", r.Subdomain, "type", "external", "error", err)
			}
		}(route)
	}

	wg.Wait()
	return results, nil
}

func (ps *ProbeService) probeInternal(ctx context.Context, route domain.Route) domain.ProbeResult {
	url := fmt.Sprintf("http://localhost:%d%s", route.LocalPort, route.HealthPath)
	return ps.doProbe(ctx, route.ID, route.Subdomain, "internal", url)
}

func (ps *ProbeService) probeExternal(ctx context.Context, route domain.Route) domain.ProbeResult {
	if route.PublicURL == "" {
		r := newProbeResult(route.ID, route.Subdomain, "external", "error")
		r.ErrorMsg = "no public_url configured"
		return r
	}
	url := route.PublicURL + route.HealthPath
	return ps.doProbe(ctx, route.ID, route.Subdomain, "external", url)
}

// newProbeResult creates a base ProbeResult with the common identity fields filled in.
func newProbeResult(routeID int, subdomain, probeType, status string) domain.ProbeResult {
	return domain.ProbeResult{
		RouteID:   routeID,
		Subdomain: subdomain,
		ProbeType: probeType,
		Status:    status,
	}
}

func (ps *ProbeService) doProbe(ctx context.Context, routeID int, subdomain, probeType, url string) domain.ProbeResult {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		r := newProbeResult(routeID, subdomain, probeType, "error")
		r.ErrorMsg = err.Error()
		return r
	}

	resp, err := ps.httpClient.Do(req)
	latency := int(time.Since(start).Milliseconds())
	if err != nil {
		status := "down"
		if ctx.Err() != nil {
			status = "timeout"
		}
		r := newProbeResult(routeID, subdomain, probeType, status)
		r.LatencyMs = latency
		r.ErrorMsg = err.Error()
		return r
	}
	defer resp.Body.Close()

	status := "up"
	if resp.StatusCode >= 400 {
		status = "down"
	}
	r := newProbeResult(routeID, subdomain, probeType, status)
	r.LatencyMs = latency
	r.StatusCode = resp.StatusCode
	return r
}

// ClassifyProbeResults takes a set of probe results and classifies each route's status
// based on the combination of internal and external probe outcomes.
func ClassifyProbeResults(results []domain.ProbeResult) []domain.RouteClassification {
	// Group results by route
	type pair struct {
		internal *domain.ProbeResult
		external *domain.ProbeResult
	}
	byRoute := make(map[int]*pair)
	subdomains := make(map[int]string)

	for i := range results {
		r := &results[i]
		if _, ok := byRoute[r.RouteID]; !ok {
			byRoute[r.RouteID] = &pair{}
		}
		subdomains[r.RouteID] = r.Subdomain
		switch r.ProbeType {
		case "internal":
			byRoute[r.RouteID].internal = r
		case "external":
			byRoute[r.RouteID].external = r
		}
	}

	var classifications []domain.RouteClassification
	for routeID, p := range byRoute {
		c := domain.RouteClassification{
			RouteID:   routeID,
			Subdomain: subdomains[routeID],
		}

		internalUp := p.internal != nil && p.internal.Status == "up"
		externalUp := p.external != nil && p.external.Status == "up"

		if p.internal != nil {
			c.Internal = p.internal.Status
		}
		if p.external != nil {
			c.External = p.external.Status
		}

		switch {
		case internalUp && externalUp:
			c.Status = "up"
			c.Assessment = "route is fully operational"
		case internalUp && !externalUp:
			c.Status = "tunnel-issue"
			c.Assessment = "scenario is running locally but not reachable via tunnel"
		case !internalUp && externalUp:
			c.Status = "scenario-down"
			c.Assessment = "internal probe failed but external still resolves (stale cache possible)"
		case !internalUp && !externalUp:
			c.Status = "unknown"
			c.Assessment = "both internal and external probes failed; investigate scenario and tunnel"
		}

		classifications = append(classifications, c)
	}
	return classifications
}
