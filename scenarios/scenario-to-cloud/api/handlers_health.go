package main

import (
	"context"
	"net/http"
	"sync"
	"time"

	"scenario-to-cloud/dns"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/httputil"
	"scenario-to-cloud/tlsinfo"
	"scenario-to-cloud/vps"
)

// handleGetDeploymentHealth runs all health checks in parallel and returns a unified health report.
// GET /api/v1/deployments/{id}/health
func (s *Server) handleGetDeploymentHealth(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	dc := s.FetchDeploymentContext(w, r)
	if dc == nil {
		return // Error already written
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	// Run checks in parallel
	var (
		liveState *domain.LiveStateResult
		dnsEval   *dns.Evaluation
		tlsSnap   *tlsinfo.Snapshot
		tlsErr    error
		wg        sync.WaitGroup
	)

	// 1. Live state inspection (includes Caddy TLS enrichment)
	wg.Add(1)
	go func() {
		defer wg.Done()
		result := vps.RunLiveStateInspection(ctx, dc.Manifest, s.sshRunner)
		s.enrichCaddyTLS(ctx, &result)
		liveState = &result
	}()

	// 2. DNS evaluation
	wg.Add(1)
	go func() {
		defer wg.Done()
		edgeDomain := dc.Manifest.Edge.Domain
		vpsHost := ""
		if dc.Manifest.Target.VPS != nil {
			vpsHost = dc.Manifest.Target.VPS.Host
		}
		if edgeDomain != "" && vpsHost != "" {
			eval := dns.Evaluate(ctx, s.dnsService, edgeDomain, vpsHost)
			dnsEval = &eval
		}
	}()

	// 3. TLS snapshot
	wg.Add(1)
	go func() {
		defer wg.Done()
		edgeDomain := dc.Manifest.Edge.Domain
		if edgeDomain != "" {
			snap, err := tlsinfo.RunSnapshot(ctx, edgeDomain, s.tlsService, s.tlsALPNRunner)
			tlsSnap = &snap
			tlsErr = err
		}
	}()

	wg.Wait()

	// Compute health report
	resp := vps.ComputeHealth(dc.Deployment, dc.Manifest, liveState, dnsEval, tlsSnap, tlsErr)
	resp.Freshness = s.evaluateDeploymentFreshness(ctx, dc.Deployment, dc.Manifest)
	if resp.Freshness != nil && resp.Freshness.Status == domain.FreshnessOutdated {
		resp.Recommendations = append(resp.Recommendations, domain.Recommendation{
			Priority: 3,
			Category: "freshness",
			Summary:  "Deployment is healthy but outdated compared to local scenario state",
			Command:  "scenario-to-cloud deployment execute " + dc.Deployment.ID + " --force-bundle",
		})
	}
	resp.DurationMs = time.Since(start).Milliseconds()

	httputil.WriteJSON(w, http.StatusOK, resp)
}
