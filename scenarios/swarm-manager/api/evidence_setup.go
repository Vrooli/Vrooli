package main

import (
	"context"
	"errors"
	"log/slog"

	"swarm-manager/internal/evidence"
	"swarm-manager/internal/planclient"
)

// wireEvidence registers the durable evidence ledger for interactive Session
// Runs. Programmatic workflow evidence is applied by its owning domain adapter;
// operating-mode executions no longer own live Runs after the workflow cutover.
func (s *Server) wireEvidence() {
	if s.evidenceStore == nil || s.agentSessionSvc == nil {
		return
	}
	s.evidenceSvc = evidence.NewService(s.evidenceStore, evidence.RunOwnerResolver{
		Sessions: s.agentSessionSvc,
	})
	s.agentSessionSvc.SetEvidenceService(s.evidenceSvc)
	if err := s.agentSessionSvc.MigrateArtifactEvidence(context.Background()); err != nil {
		// Keep the legacy read path active; an incomplete migration must never
		// hide artifacts from operators.
		slog.Warn("session artifact evidence migration incomplete; keeping legacy projection", "error", err)
	} else {
		s.agentSessionSvc.EnableEvidenceProjection()
	}
	reconciler := evidenceReconciler{service: s.evidenceSvc, planReader: planclient.NewConnectClient(nil, nil), agentReader: s.agentSvc}
	s.agentSessionSvc.SetEvidenceReconciler(reconciler)
	s.router.Use(s.evidenceInvocationMiddleware)
	evidence.RegisterConnectService(s.router, s.evidenceSvc, reconciler)
}

type evidenceReconciler struct {
	service     *evidence.Service
	planReader  planclient.PlanAuditReader
	agentReader evidence.AgentManagerReader
}

func (r evidenceReconciler) Reconcile(ctx context.Context, runID string) error {
	_, planErr := r.service.ReconcilePlanManager(ctx, r.planReader, runID)
	_, agentErr := r.service.ReconcileAgentManager(ctx, r.agentReader, runID)
	return errors.Join(planErr, agentErr)
}
