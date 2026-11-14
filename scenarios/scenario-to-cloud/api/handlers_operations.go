package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/authz"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/httputil"
	"scenario-to-cloud/operations"
)

// Durable operation surface (P07). The operation id is the wait reference a
// client keeps across disconnects; nothing here mutates an operation except
// cancel (a recorded intent) and reconcile (an owner pass).
func (s *Server) registerOperationRoutes(api *mux.Router) {
	api.HandleFunc("/operations/reconcile", s.handleReconcileOperations).Methods("POST")
	api.HandleFunc("/operations/{id}", s.handleGetOperation).Methods("GET")
	api.HandleFunc("/operations/{id}/wait", s.handleWaitOperation).Methods("GET")
	api.HandleFunc("/operations/{id}/cancel", s.handleCancelOperation).Methods("POST")
	api.HandleFunc("/deployments/{id}/operations", s.handleListDeploymentOperations).Methods("GET")
}

func (s *Server) requireOperations(w http.ResponseWriter) bool {
	if s.operations == nil || s.repo == nil {
		apierrors.Write(w, apierrors.New(apierrors.CodeInternal, "Operation owner is not configured"))
		return false
	}
	return true
}

// handleGetOperation returns the typed standing.
// GET /api/v1/operations/{id}
func (s *Server) handleGetOperation(w http.ResponseWriter, r *http.Request) {
	if !s.requireOperations(w) {
		return
	}
	op, err := s.operations.Get(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		apierrors.Write(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, operations.StandingOf(op))
}

// handleWaitOperation blocks server-side until the operation is terminal or
// the observer bound elapses. A timed-out wait returns 200 with
// still_pending and a recommended next check; it never changes the record.
// GET /api/v1/operations/{id}/wait?timeout=<seconds|duration>
func (s *Server) handleWaitOperation(w http.ResponseWriter, r *http.Request) {
	if !s.requireOperations(w) {
		return
	}
	timeout, err := parseWaitTimeout(r.URL.Query().Get("timeout"), s.operations.Config().ObserverTimeout)
	if err != nil {
		apierrors.Write(w, err)
		return
	}
	op, pending, werr := s.operations.Wait(r.Context(), mux.Vars(r)["id"], timeout)
	if werr != nil {
		apierrors.Write(w, werr)
		return
	}
	standing := operations.StandingOf(op)
	if pending {
		standing = standing.Pending(s.operations.Config().LeaseTTL)
	}
	httputil.WriteJSON(w, http.StatusOK, standing)
}

func parseWaitTimeout(raw string, bound time.Duration) (time.Duration, *apierrors.Error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return bound, nil
	}
	if secs, err := strconv.Atoi(raw); err == nil {
		if secs < 0 {
			return 0, apierrors.New(apierrors.CodeInvalidRequest, "timeout must not be negative")
		}
		return time.Duration(secs) * time.Second, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d < 0 {
		return 0, apierrors.New(apierrors.CodeInvalidRequest, "timeout must be seconds or a Go duration").WithDetail("timeout", raw)
	}
	return d, nil
}

// handleCancelOperation records the cancellation intent.
// POST /api/v1/operations/{id}/cancel
func (s *Server) handleCancelOperation(w http.ResponseWriter, r *http.Request) {
	if !s.requireOperations(w) {
		return
	}
	id := mux.Vars(r)["id"]
	op, err := s.operations.Get(r.Context(), id)
	if err != nil {
		apierrors.Write(w, err)
		return
	}
	if denied := s.authz.RequireEffect(r.Context(), op.DeploymentID, authz.EffectWorkloadMutation); denied != nil {
		apierrors.Write(w, denied)
		return
	}
	op, err = s.operations.Cancel(r.Context(), id)
	if err != nil {
		apierrors.Write(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusAccepted, operations.StandingOf(op))
}

// handleListDeploymentOperations lists operations of one deployment.
// GET /api/v1/deployments/{id}/operations
func (s *Server) handleListDeploymentOperations(w http.ResponseWriter, r *http.Request) {
	if !s.requireOperations(w) {
		return
	}
	id := mux.Vars(r)["id"]
	dep, err := s.repo.GetDeployment(r.Context(), id)
	if err != nil {
		apierrors.Write(w, apierrors.Internal("Failed to get deployment", err))
		return
	}
	if dep == nil {
		apierrors.Write(w, deploymentNotFound(id))
		return
	}
	ops, err := s.repo.ListOperationsByDeployment(r.Context(), id)
	if err != nil {
		apierrors.Write(w, apierrors.Internal("Failed to list operations", err))
		return
	}
	standings := make([]operations.Standing, 0, len(ops))
	for _, op := range ops {
		standings = append(standings, operations.StandingOf(op))
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"schema_version": operations.StandingSchemaVersion,
		"deployment_id":  id,
		"operations":     standings,
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
	})
}

// handleReconcileOperations runs one owner pass now.
// POST /api/v1/operations/reconcile
func (s *Server) handleReconcileOperations(w http.ResponseWriter, r *http.Request) {
	if !s.requireOperations(w) {
		return
	}
	if denied := s.authz.RequireEffect(r.Context(), "", authz.EffectWorkloadMutation); denied != nil {
		apierrors.Write(w, denied)
		return
	}
	taken, err := s.operations.Reconcile(r.Context())
	if err != nil {
		apierrors.Write(w, apierrors.Internal("Reconciliation pass failed", err))
		return
	}
	if taken == nil {
		taken = []string{}
	}
	httputil.WriteJSON(w, http.StatusAccepted, map[string]any{
		"schema_version": operations.StandingSchemaVersion,
		"worker_id":      s.operations.WorkerID(),
		"acquired":       taken,
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
	})
}

// admitAndSubmit admits an operation for a compiled plan under requestKey and
// hands it to the owner. It is the single path execute, start and plan/apply
// share. A replay (same key, same digest) returns the existing operation and
// submits nothing.
func (s *Server) admitAndSubmit(r *http.Request, deploymentID, requestKey string, compiled *CompiledPlan, options operations.ExecuteOptions) (*domain.CloudOperation, bool, *apierrors.Error) {
	if s.operations == nil {
		return nil, false, apierrors.New(apierrors.CodeInternal, "Operation owner is not configured")
	}
	op, created, aerr := operations.Admit(r.Context(), s.repo, deploymentID, requestKey, compiled.Plan)
	if aerr != nil {
		return nil, false, aerr
	}
	if created {
		s.operations.Submit(r.Context(), op.ID, options)
	}
	return op, created, nil
}

// writeOperationAccepted is the 202 body execute/start/apply share.
func writeOperationAccepted(w http.ResponseWriter, dep *domain.Deployment, op *domain.CloudOperation, replayed bool, message string) {
	body := map[string]any{
		"schema_version": operations.StandingSchemaVersion,
		"operation_id":   op.ID,
		"plan_digest":    op.PlanDigest,
		"state":          op.State,
		"replayed":       replayed,
		"wait":           fmt.Sprintf("/api/v1/operations/%s/wait", op.ID),
		"message":        message,
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
	}
	if dep != nil {
		body["deployment"] = dep
	}
	httputil.WriteJSON(w, http.StatusAccepted, body)
}

// decodeExecuteRequest reads the optional execute/start body.
func decodeExecuteRequest(r *http.Request, s *Server, what string) domain.ExecuteDeploymentRequest {
	var req domain.ExecuteDeploymentRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.log("failed to parse "+what+" request body", map[string]interface{}{"error": err.Error()})
		}
	}
	return req
}
