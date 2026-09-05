package rewrite

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	intgraph "typescript-code-graph/internal/graph"
	"typescript-code-graph/internal/sidecar"
)

// Service orchestrates the Plan + Apply flow. Validate → normalize →
// store on Plan; validate → load → check sidecar → (sidecar.RewriteApply
// OR dry-run short-circuit) on Apply.
//
// The Service reuses graph.PathMutex (not its own) so Extract and
// Apply serialize on the same per-project_path lock — OT-P0-007.
//
// Substrate rule: this file does NOT import time, os, net/http, or
// os/exec. Wall-clock timing belongs to the handler; filesystem
// mutation belongs to the sidecar.
type Service struct {
	store  PlanStore
	client sidecar.SidecarClient
	mu     *intgraph.PathMutex
}

// NewService wires the production Service. All three arguments are
// required; the caller (api/main.go) owns construction so the mutex
// can be shared with the graph domain.
func NewService(store PlanStore, client sidecar.SidecarClient, mu *intgraph.PathMutex) *Service {
	return &Service{store: store, client: client, mu: mu}
}

// Plan validates the input, normalizes the operations, derives the
// PlanID, and persists the (ProjectPath, PlanID) → Plan mapping.
//
// Empty / non-absolute project_path, empty operations, or any
// individual invalid operation surface as
// RewriteError{Kind:RewriteErrorInvalidInput|RewriteErrorInvalidOperation}
// — both map to InvalidArgument at the handler.
func (s *Service) Plan(ctx context.Context, in PlanInput) (PlanOutput, error) {
	abs, err := validateProjectPath(in.ProjectPath)
	if err != nil {
		return PlanOutput{}, err
	}
	if len(in.Operations) == 0 {
		return PlanOutput{}, RewriteError{
			Kind:    RewriteErrorInvalidInput,
			Path:    abs,
			Message: "operations list is empty",
		}
	}
	normalized, err := Normalize(in.Operations)
	if err != nil {
		return PlanOutput{}, err
	}
	id := DerivePlanID(normalized)
	if err := s.store.Save(Plan{ID: id, ProjectPath: abs, Operations: normalized}); err != nil {
		return PlanOutput{}, RewriteError{
			Kind:    RewriteErrorInternal,
			Path:    abs,
			PlanID:  id,
			Cause:   err,
			Message: "plan store save failed",
		}
	}
	return PlanOutput{PlanID: id, NormalizedOperations: normalized}, nil
}

// Apply loads the stored plan, acquires the per-path mutex, checks
// sidecar readiness, then either short-circuits with synthetic OK
// results (dry-run, no IPC call per §8.6) or delegates to
// sidecar.RewriteApply for real execution.
func (s *Service) Apply(ctx context.Context, in ApplyInput) (ApplyOutput, error) {
	abs, err := validateProjectPath(in.ProjectPath)
	if err != nil {
		return ApplyOutput{}, err
	}
	if strings.TrimSpace(string(in.PlanID)) == "" {
		return ApplyOutput{}, RewriteError{
			Kind:    RewriteErrorInvalidInput,
			Path:    abs,
			Message: "plan_id is required",
		}
	}

	plan, err := s.store.Get(abs, in.PlanID)
	if err != nil {
		if errors.Is(err, ErrPlanNotFound) {
			return ApplyOutput{}, RewriteError{
				Kind:    RewriteErrorPlanNotFound,
				Path:    abs,
				PlanID:  in.PlanID,
				Message: "plan not found for scenario",
				Cause:   err,
			}
		}
		return ApplyOutput{}, RewriteError{
			Kind:    RewriteErrorInternal,
			Path:    abs,
			PlanID:  in.PlanID,
			Cause:   err,
			Message: "plan store get failed",
		}
	}

	unlock := s.mu.Lock(abs)
	defer unlock()

	if in.DryRun {
		return ApplyOutput{
			PlanID:  plan.ID,
			Results: synthesizeDryRunResults(plan.Operations),
			DryRun:  true,
		}, nil
	}

	if st := s.client.Status(); st != sidecar.StatusReady {
		return ApplyOutput{}, RewriteError{
			Kind:    RewriteErrorSidecarUnavailable,
			Path:    abs,
			PlanID:  plan.ID,
			Message: "sidecar status " + string(st),
		}
	}

	sideOps := toSidecarOperations(plan.Operations)
	sideResults, err := s.client.RewriteApply(ctx, abs, sideOps)
	if err != nil {
		return ApplyOutput{}, fromSidecarError(abs, plan.ID, err)
	}

	return ApplyOutput{
		PlanID:  plan.ID,
		Results: fromSidecarResults(plan.Operations, sideResults),
		DryRun:  false,
	}, nil
}

// validateProjectPath enforces the same discipline as the graph
// domain — non-empty, absolute, filepath.Clean'd. Symbolic links and
// existence checks belong to the sidecar (it owns the filesystem).
func validateProjectPath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", RewriteError{
			Kind:    RewriteErrorInvalidInput,
			Message: "project_path is required",
		}
	}
	if !filepath.IsAbs(p) {
		return "", RewriteError{
			Kind:    RewriteErrorInvalidInput,
			Path:    p,
			Message: "project_path must be absolute",
		}
	}
	return filepath.Clean(p), nil
}

// synthesizeDryRunResults returns one OK result per op in the same
// order as plan.Operations. No sidecar call happens — this is the
// dry-run guarantee from §8.6.
func synthesizeDryRunResults(ops []Operation) []ApplyResult {
	out := make([]ApplyResult, 0, len(ops))
	for _, op := range ops {
		out = append(out, ApplyResult{
			Operation: op,
			Status:    StatusOK,
		})
	}
	return out
}

// toSidecarOperations projects domain Operations onto the sidecar
// wire type. The sidecar.FileMove uses From/To while the proto/domain
// use FromPath/ToPath — translate explicitly so the seam contract
// stays loud.
func toSidecarOperations(ops []Operation) []sidecar.Operation {
	out := make([]sidecar.Operation, 0, len(ops))
	for _, op := range ops {
		so := sidecar.Operation{}
		if op.FileMove != nil {
			so.FileMove = &sidecar.FileMove{From: op.FileMove.FromPath, To: op.FileMove.ToPath}
		}
		if op.ImportRewrite != nil {
			so.ImportRewrite = &sidecar.ImportRewrite{OldPath: op.ImportRewrite.OldPath, NewPath: op.ImportRewrite.NewPath}
		}
		out = append(out, so)
	}
	return out
}

// fromSidecarResults zips the per-op sidecar results back onto domain
// ApplyResult. When the lengths disagree (sidecar misbehavior) we
// trust the sidecar's count for the Status/Message but always echo
// the request's Operation so the caller sees a consistent shape.
func fromSidecarResults(ops []Operation, side []sidecar.OperationResult) []ApplyResult {
	out := make([]ApplyResult, 0, len(ops))
	for i := range ops {
		ar := ApplyResult{Operation: ops[i], Status: StatusOK}
		if i < len(side) {
			ar.Status = normalizeStatus(side[i].Status)
			ar.Message = side[i].Message
		}
		out = append(out, ar)
	}
	return out
}

// normalizeStatus validates the sidecar's per-op status against the
// single canonical spelling. The sidecar emits exactly
// "OPERATION_STATUS_OK" / "OPERATION_STATUS_FAILED" (pinned in
// sidecar/src/rewrite.ts); any other value — including the empty string
// — is treated as a failure rather than silently coerced to OK, so a
// drifting or malfunctioning sidecar can never masquerade a failure as a
// success.
func normalizeStatus(s string) string {
	switch s {
	case StatusOK:
		return StatusOK
	default:
		return StatusFailed
	}
}
