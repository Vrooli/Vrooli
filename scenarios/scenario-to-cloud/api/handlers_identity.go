package main

import (
	"context"
	"net/http"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/identity"
	"scenario-to-cloud/internal/httputil"
)

// deploymentNotFound is the typed not-found for a deployment id.
func deploymentNotFound(id string) *apierrors.Error {
	return apierrors.New(apierrors.CodeDeploymentNotFound, "Deployment not found").
		WithDetail("id", id).
		WithNextAction(apierrors.NextAction{Owner: "scenario-to-cloud", Kind: "endpoint", Reference: "/api/v1/deployments", Label: "List deployments"})
}

// findDeploymentForManifest returns the deployment that already owns the
// (scenario, environment, target) identity a manifest describes, or nil when
// the manifest would create a new deployment. The host facet is used because
// manifests describe SSH targets; a Bridge-bound deployment keeps its locator
// host, so the same manifest keeps resolving to it after enrollment.
func (s *Server) findDeploymentForManifest(ctx context.Context, scenarioID, environment string, target identity.TargetRef) (*domain.Deployment, error) {
	if target.Locator.Host == "" {
		return nil, nil
	}
	matches, err := s.repo.ResolveDeployments(ctx, identity.Selector{ScenarioID: scenarioID, Environment: environment, Host: target.Locator.Host})
	if err != nil {
		return nil, apierrors.Internal("Failed to check for existing deployment", err)
	}
	if len(matches) == 0 {
		return nil, nil
	}
	if len(matches) > 1 {
		return nil, apierrors.New(apierrors.CodeDeploymentSelectorAmbiguous, "More than one deployment matches this manifest's scenario, environment and host").
			WithDetail("scenario_id", scenarioID).
			WithDetail("environment", environment).
			WithDetail("host", target.Locator.Host)
	}
	existing, err := s.repo.GetDeployment(ctx, matches[0].ID)
	if err != nil {
		return nil, apierrors.Internal("Failed to load existing deployment", err)
	}
	return existing, nil
}

// handleResolveDeployment maps a selector from query parameters to exactly
// one deployment identity.
// GET /deployments/resolve?id=&scenario=&environment=&domain=&host=
func (s *Server) handleResolveDeployment(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	selector := identity.Selector{
		ID:          query.Get("id"),
		ScenarioID:  query.Get("scenario"),
		Environment: query.Get("environment"),
		Domain:      query.Get("domain"),
		Host:        query.Get("host"),
	}
	ref, err := identity.Resolve(r.Context(), s.repo, selector)
	if err != nil {
		apierrors.Write(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"schema_version": "1",
		"ref":            ref,
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
	})
}
