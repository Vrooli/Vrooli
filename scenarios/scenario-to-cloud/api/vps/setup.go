package vps

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
)

// SetupRequest is the request body for VPS setup. Apply requires the
// plan_digest the caller reviewed; a mismatch is refused.
type SetupRequest struct {
	Manifest   domain.CloudManifest `json:"manifest"`
	BundlePath string               `json:"bundle_path"`
	// PlanDigest is the semantic digest of the reviewed plan (apply only).
	PlanDigest string `json:"plan_digest,omitempty"`
	// DeploymentID and RequestKey admit a durable operation when present.
	DeploymentID string `json:"deployment_id,omitempty"`
	RequestKey   string `json:"request_key,omitempty"`
}

// BuildSetupExecutablePlan compiles the install-scope plan for a manifest
// and local bundle. Preview and apply both call this.
func BuildSetupExecutablePlan(ctx context.Context, manifest domain.CloudManifest, bundlePath string) (*execplan.Plan, error) {
	return CompilePlan(ctx, PlanRequest{Manifest: manifest, BundlePath: bundlePath, Scope: execplan.ScopeInstall})
}

// BuildSetupPlan is the legacy step view of the install plan: id = action
// id, command = shell preview. It is derived from the same compiled plan the
// executor runs.
func BuildSetupPlan(manifest domain.CloudManifest, bundlePath string) ([]domain.VPSPlanStep, error) {
	plan, err := BuildSetupExecutablePlan(context.Background(), manifest, bundlePath)
	if err != nil {
		return nil, err
	}
	return RenderSteps(plan, manifest), nil
}

// RunSetupPlanWithProgress executes an already compiled (and reviewed)
// install plan. Results are keyed by action id.
func RunSetupPlanWithProgress(
	ctx context.Context,
	plan *execplan.Plan,
	manifest domain.CloudManifest,
	bundlePath string,
	rt Runtime,
	hub ProgressBroadcaster,
	repo ProgressRepo,
	deploymentID string,
	progress *float64,
) domain.VPSSetupResult {
	start := time.Now()
	steps := RenderSteps(plan, manifest)
	digest, _ := plan.SemanticDigest()
	trace, execErr := ExecutePlan(ctx, ExecuteRequest{Plan: plan, Manifest: manifest, BundlePath: bundlePath, Runtime: rt, Hub: hub, Repo: repo, DeploymentID: deploymentID, Progress: progress})
	result := domain.VPSSetupResult{
		OK:         execErr == nil,
		Steps:      steps,
		PlanDigest: digest,
		Actions:    trace.Actions,
		DurationMs: time.Since(start).Milliseconds(),
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}
	if execErr != nil {
		result.Error = execErr.Error()
		result.ErrorInfo = execErr.Info
		result.FailedStep = execErr.ActionID
	}
	return result
}

// AutohealScopeJSON is the minimal autoheal scope declaration delivered to
// <workdir>/.vrooli/cloud/autoheal-scope.json. It tells vrooli-autoheal which
// scenario and resources this deployment owns; the desired-state contract in
// docs/reference/activation-and-reconciliation.md tells it when not to act.
func AutohealScopeJSON(manifest domain.CloudManifest) []byte {
	payload := map[string]interface{}{
		"schema_version": "1.0.0",
		"environment":    manifest.Environment,
		"scenario_id":    manifest.Scenario.ID,
		"resources":      manifest.Dependencies.Resources,
		"scenarios":      manifest.Dependencies.Scenarios,
		"generated_at":   time.Now().UTC().Format(time.RFC3339),
		"desired_state":  "runtime-owner",
	}
	b, _ := json.MarshalIndent(payload, "", "  ")
	return b
}

// writeAutohealScope materialises the scope document in a temporary file for
// delivery through reach.
func writeAutohealScope(manifest domain.CloudManifest) (string, error) {
	dir, err := os.MkdirTemp("", "autoheal-scope-")
	if err != nil {
		return "", fmt.Errorf("stage autoheal scope: %w", err)
	}
	path := filepath.Join(dir, "autoheal-scope.json")
	if err := os.WriteFile(path, AutohealScopeJSON(manifest), 0o644); err != nil {
		return "", fmt.Errorf("stage autoheal scope: %w", err)
	}
	return path, nil
}
