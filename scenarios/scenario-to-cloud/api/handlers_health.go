package main

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/dns"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/health"
	"scenario-to-cloud/internal/httputil"
	"scenario-to-cloud/sshidentity"
	"scenario-to-cloud/tlsinfo"
	"scenario-to-cloud/vps"

	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/health"
)

// healthInspectionTimeout bounds one live-state + DNS + TLS inspection. The
// console is an operator surface, so an unreachable target must settle to a
// typed unknown/degraded observation within a bounded interactive interval.
const healthInspectionTimeout = 15 * time.Second

// registerHealthRoutes mounts the typed observation surface beside the
// legacy report. The legacy GET /deployments/{id}/health route stays where
// setupRoutes registers it; this adds the observation route and the Connect
// HealthService.
func (s *Server) registerHealthRoutes(api *mux.Router) {
	api.HandleFunc("/deployments/{id}/health/observation", s.handleGetDeploymentHealthObservation).Methods("GET")
	path, handler := health.NewService(s).Handler()
	s.router.PathPrefix(path).Handler(handler)
}

// evaluateFreshnessForHealth is the release-parity seam. It hashes the local
// scenario bundle, which is slow; tests replace it.
var evaluateFreshnessForHealth = func(s *Server, ctx context.Context, dep *domain.Deployment, manifest domain.CloudManifest) *domain.FreshnessStatus {
	return s.evaluateDeploymentFreshness(ctx, dep, manifest)
}

func (s *Server) alerter() *health.Alerter {
	if s == nil {
		return nil
	}
	return s.healthSvc
}

// healthInspection is one producer pass over a deployment: the legacy
// composed report and the typed observation built from the same evidence.
type healthInspection struct {
	report      domain.HealthResponse
	observation *healthv1.HealthObservation
}

// inspectDeploymentHealth runs live-state, DNS and TLS checks in parallel and
// composes both the legacy report and the typed observation. observed_at on
// the observation is the live-state inspection's own timestamp, never the
// time this function returns.
func (s *Server) inspectDeploymentHealth(parent context.Context, dc *DeploymentContext) healthInspection {
	ctx, cancel := context.WithTimeout(parent, healthInspectionTimeout)
	defer cancel()
	identity := s.resolveCanonicalIdentity(ctx, dc.Deployment)

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
		result := vps.RunLiveStateInspection(ctx, dc.Manifest, identity, s.proberFor(dc))
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
		if edgeDomain != "" && vpsHost != "" && s.dnsService != nil {
			eval := dns.Evaluate(ctx, s.dnsService, edgeDomain, vpsHost)
			dnsEval = &eval
		}
	}()

	// 3. TLS snapshot
	wg.Add(1)
	go func() {
		defer wg.Done()
		edgeDomain := dc.Manifest.Edge.Domain
		if edgeDomain != "" && s.tlsService != nil {
			snap, err := tlsinfo.RunSnapshot(ctx, edgeDomain, s.tlsService, s.tlsALPNRunner)
			tlsSnap = &snap
			tlsErr = err
		}
	}()

	wg.Wait()
	if liveState != nil && liveState.OK && liveState.System != nil {
		verified := sshidentity.ApplyVerificationResult(
			identity,
			sshidentity.VerificationState(liveState.System.SSH.VerificationState),
			time.Now().UTC(),
		)
		identity = verified
		s.persistCanonicalIdentity(ctx, dc.Deployment.ID, verified)
	}

	// Compute the legacy report
	report := vps.ComputeHealth(dc.Deployment, dc.Manifest, identity, liveState, dnsEval, tlsSnap, tlsErr)
	report.Freshness = evaluateFreshnessForHealth(s, ctx, dc.Deployment, dc.Manifest)
	if report.Freshness != nil && report.Freshness.Status == domain.FreshnessOutdated {
		report.Recommendations = append(report.Recommendations, domain.Recommendation{
			Priority: 3,
			Category: "freshness",
			Summary:  "Deployment is healthy but outdated compared to local scenario state",
			Command:  "scenario-to-cloud deployment execute " + dc.Deployment.ID + " --force-bundle",
		})
	}

	observation := health.Build(health.Input{
		Deployment: dc.Deployment,
		Report:     report,
		LiveState:  liveState,
		Now:        time.Now().UTC(),
	}, health.DefaultPolicy())
	if raw, err := health.MarshalJSON(observation); err == nil {
		report.Observation = raw
	}
	if alerter := s.alerter(); alerter != nil {
		alerter.Observe(ctx, observation, report)
	}
	return healthInspection{report: report, observation: observation}
}

// Observe implements health.Producer for the Connect HealthService.
func (s *Server) Observe(ctx context.Context, deploymentID string) (*healthv1.HealthObservation, error) {
	dc, err := s.loadDeploymentContext(ctx, deploymentID)
	if err != nil {
		return nil, err
	}
	return s.inspectDeploymentHealth(ctx, dc).observation, nil
}

// handleGetDeploymentHealth runs all health checks in parallel and returns
// the unified legacy report with the typed observation embedded.
// GET /api/v1/deployments/{id}/health
func (s *Server) handleGetDeploymentHealth(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	dc := s.FetchDeploymentContext(w, r)
	if dc == nil {
		return // Error already written
	}

	resp := s.inspectDeploymentHealth(r.Context(), dc).report
	resp.DurationMs = time.Since(start).Milliseconds()

	httputil.WriteJSON(w, http.StatusOK, resp)
}

// handleGetDeploymentHealthObservation returns only the typed observation in
// its versioned envelope. HTTP 200 means the observation was produced; the
// deployment verdict is observation.status together with
// observation.freshness.
// GET /api/v1/deployments/{id}/health/observation
func (s *Server) handleGetDeploymentHealthObservation(w http.ResponseWriter, r *http.Request) {
	dc := s.FetchDeploymentContext(w, r)
	if dc == nil {
		return
	}
	observation := s.inspectDeploymentHealth(r.Context(), dc).observation
	raw, err := health.MarshalResponseJSON(observation)
	if err != nil {
		apierrors.Write(w, apierrors.Internal("Failed to encode health observation", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}
