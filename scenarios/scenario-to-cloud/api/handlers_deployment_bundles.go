package main

import (
	"context"
	"net/http"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/bundle"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/httputil"
)

// The deployment-scoped release inventory is the owner's listing
// (`vrooli cloud-target release list`) read through reach; retention is
// planned on the cloud side with lease protection and executed by the owner
// (`release prune`), which keeps active and previous whatever is asked.

// handleListDeploymentVPSBundles lists the releases the target owner holds
// for a deployment.
// GET /api/v1/deployments/{id}/bundles/vps
func (s *Server) handleListDeploymentVPSBundles(w http.ResponseWriter, r *http.Request) {
	dc := s.FetchDeploymentContext(w, r)
	if dc == nil {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	listing, err := bundle.ListTargetReleases(ctx, s.reachFor(dc), s.targetFor(dc), dc.ID)
	if err != nil {
		httputil.WriteJSON(w, http.StatusBadGateway, domain.VPSBundleListResponse{
			OK:        false,
			Bundles:   []domain.VPSBundleInfo{},
			Error:     err.Error(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}
	rows, total := bundle.InventoryFromListing(listing, dc.Manifest.Scenario.ID)
	if rows == nil {
		rows = []domain.VPSBundleInfo{}
	}
	httputil.WriteJSON(w, http.StatusOK, domain.VPSBundleListResponse{
		OK:             true,
		Bundles:        rows,
		TotalSizeBytes: total,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
	})
}

// handleGCDeploymentVPSBundles garbage-collects retired releases on the
// deployment target through the owner.
// POST /api/v1/deployments/{id}/bundles/vps/gc
func (s *Server) handleGCDeploymentVPSBundles(w http.ResponseWriter, r *http.Request) {
	dc := s.FetchDeploymentContext(w, r)
	if dc == nil {
		return
	}
	var req domain.VPSBundleGCRequest
	if !httputil.DecodeRequestBody(w, r, &req) {
		return
	}
	if req.KeepLatest <= 0 {
		req.KeepLatest = bundle.DefaultVPSBundleKeepLatest
	}
	if req.ScenarioID == "" {
		req.ScenarioID = dc.Manifest.Scenario.ID
	}
	// Protect the bundle currently associated with this deployment (if recorded).
	if dc.Deployment.BundleSHA256 != nil && *dc.Deployment.BundleSHA256 != "" {
		req.ProtectSHA256 = append(req.ProtectSHA256, *dc.Deployment.BundleSHA256)
	}
	for _, sha := range req.ProtectSHA256 {
		if len(sha) != 64 {
			apierrors.Write(w, apierrors.New(apierrors.CodeInvalidRequest, "protect_sha256 entries must be 64-hex bundle digests"))
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	resp := bundle.GCTargetReleases(ctx, s.reachFor(dc), s.targetFor(dc), dc.ID, dc.Manifest.Scenario.ID, req)
	httputil.WriteJSON(w, http.StatusOK, resp)
}
