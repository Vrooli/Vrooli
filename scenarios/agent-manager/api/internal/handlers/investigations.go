package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/investigation"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/runreport"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

const maxInvestigationRequestBytes = 1 << 20

func (h *Handler) StartInvestigation(w http.ResponseWriter, r *http.Request) {
	if h.investigations == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "investigation lifecycle is unavailable")
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxInvestigationRequestBytes+1))
	if err != nil || len(raw) > maxInvestigationRequestBytes {
		writeSimpleError(w, r, "body", "investigation request is missing or exceeds 1 MiB")
		return
	}
	request, err := investigation.DecodeRequest(raw)
	if err != nil {
		writeInvestigationError(w, r, err)
		return
	}
	item, reused, err := h.admitInvestigation(r.Context(), request)
	if err != nil {
		writeInvestigationError(w, r, err)
		return
	}
	status := http.StatusCreated
	if reused {
		status = http.StatusOK
	}
	writeJSON(w, status, map[string]any{"investigation": item, "reused": reused})
}

// admitInvestigation makes admission and dispatch one idempotent boundary.
// The repository is authoritative for admission; the workflow service is
// optional only for presentation-layer tests and older embedders. Production
// wiring always supplies it, so a queued request is immediately attached to a
// durable diagnosis-only workflow.
func (h *Handler) admitInvestigation(ctx context.Context, request investigation.Request) (*investigation.Lifecycle, bool, error) {
	item, reused, err := h.investigations.Reserve(ctx, request)
	if err != nil {
		return nil, false, err
	}
	if h.svc.WorkflowService == nil || item == nil || item.WorkflowRef != "" || investigationTerminal(item.OperationStatus) {
		return item, reused, nil
	}

	input, err := investigationWorkflowInput(ctx, request, h.svc.RunReportService)
	if err != nil {
		_, _ = h.investigations.UpdateStatus(ctx, item.ID, investigation.OperationFailed, "")
		return nil, false, err
	}
	execution, err := h.svc.StartWorkflowExecution(ctx, orchestration.StartWorkflowExecutionRequest{
		Owner:          "agent-manager",
		WorkflowKey:    investigation.WorkflowKey,
		Input:          input,
		IdempotencyKey: "typed-investigation/" + item.ID,
		Initiator:      domain.WorkflowInitiatorProgrammatic,
	})
	if err != nil {
		_, _ = h.investigations.UpdateStatus(ctx, item.ID, investigation.OperationFailed, "")
		return nil, false, err
	}
	if execution == nil {
		_, _ = h.investigations.UpdateStatus(ctx, item.ID, investigation.OperationFailed, "")
		return nil, false, errors.New("investigation workflow did not return an execution")
	}
	item, err = h.investigations.UpdateStatus(ctx, item.ID, investigation.OperationCollecting, execution.ID.String())
	if err != nil {
		return nil, false, err
	}
	// The workflow can settle between StartWorkflowExecution returning and the
	// lifecycle linkage being persisted. Re-read the authoritative execution
	// after linking so that a fast terminal workflow cannot strand the typed
	// investigation in collecting/diagnosing.
	authoritative := execution
	if reader, ok := h.svc.WorkflowService.(interface {
		GetWorkflowExecution(context.Context, uuid.UUID) (*domain.WorkflowExecution, error)
	}); ok {
		if settled, readErr := reader.GetWorkflowExecution(ctx, execution.ID); readErr == nil && settled != nil {
			authoritative = settled
		}
	}
	if authoritative.Status.Terminal() {
		if settler, ok := h.svc.WorkflowService.(interface {
			ReconcileTypedInvestigation(context.Context, *domain.WorkflowExecution)
		}); ok {
			settler.ReconcileTypedInvestigation(ctx, authoritative)
			item, _ = h.investigations.Get(ctx, item.ID)
		}
	}
	return item, reused, nil
}

const maxTypedInvestigationContextBytes = 56000

func investigationWorkflowInput(ctx context.Context, request investigation.Request, reports orchestration.RunReportService) (json.RawMessage, error) {
	depth := "standard"
	runIDs := request.Subject.RunIDs
	if runIDs == nil {
		runIDs = []string{}
	}
	switch {
	case request.Budget.MaxTurns > 0 && request.Budget.MaxTurns <= 3:
		depth = "quick"
	case request.Budget.MaxTurns > 12:
		depth = "deep"
	}
	selection := map[string]any{
		"subjectKind":   request.Subject.Kind,
		"subjectRef":    request.Subject.Ref,
		"callerKey":     request.RequestKey,
		"diagnosisOnly": true,
	}
	evidenceRefs := request.DomainEvidence
	if evidenceRefs == nil {
		evidenceRefs = []investigation.EvidenceReference{}
	}
	contextText := strings.TrimSpace(request.Question)
	if len(runIDs) > 0 {
		if reports == nil {
			return nil, errors.New("typed investigation requires the bounded run-report service")
		}
		var evidence strings.Builder
		if contextText != "" {
			fmt.Fprintf(&evidence, "Question: %s\n\n", contextText)
		}
		evidence.WriteString("Authoritative bounded run evidence follows. Classify only what these projections support.\n")
		for _, rawID := range runIDs {
			runID, err := uuid.Parse(rawID)
			if err != nil {
				return nil, fmt.Errorf("subject run %q is not a UUID: %w", rawID, err)
			}
			report, err := reports.BuildRunReport(ctx, runID)
			if err != nil {
				return nil, fmt.Errorf("build bounded report for subject run %s: %w", runID, err)
			}
			evidence.WriteString("\n## Run evidence\n\n")
			evidence.WriteString(runreport.Text(report))
		}
		contextText = evidence.String()
		if len(contextText) > maxTypedInvestigationContextBytes {
			contextText = contextText[:maxTypedInvestigationContextBytes] + "\n\n... (typed investigation context truncated; treat coverage as bounded)\n"
		}
	}
	if contextText == "" && request.MethodRef != nil {
		contextText = request.MethodRef.SkillID + "@" + request.MethodRef.Revision
	}
	payload := map[string]any{
		"context":      contextText,
		"depth":        depth,
		"runIds":       runIDs,
		"selection":    selection,
		"evidenceRefs": evidenceRefs,
	}
	raw, _ := json.Marshal(payload)
	return raw, nil
}

func investigationTerminal(status string) bool {
	return status == investigation.OperationCompleted || status == investigation.OperationFailed || status == investigation.OperationCancelled
}

func (h *Handler) GetInvestigation(w http.ResponseWriter, r *http.Request) {
	if h.investigations == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "investigation lifecycle is unavailable")
		return
	}
	item, err := h.investigations.Get(r.Context(), strings.TrimSpace(mux.Vars(r)["id"]))
	if err != nil {
		writeInvestigationError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) ListInvestigations(w http.ResponseWriter, r *http.Request) {
	if h.investigations == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "investigation lifecycle is unavailable")
		return
	}
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 200 {
			writeSimpleError(w, r, "limit", "must be between 1 and 200")
			return
		}
		limit = parsed
	}
	items, err := h.investigations.List(r.Context(), r.URL.Query().Get("status"), limit)
	if err != nil {
		writeInvestigationError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"investigations": items})
}

func (h *Handler) WaitInvestigation(w http.ResponseWriter, r *http.Request) {
	if h.investigations == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "investigation lifecycle is unavailable")
		return
	}
	timeout := 30 * time.Second
	if raw := strings.TrimSpace(r.URL.Query().Get("timeoutSeconds")); raw != "" {
		seconds, err := strconv.Atoi(raw)
		if err != nil || seconds < 1 || seconds > 1800 {
			writeSimpleError(w, r, "timeoutSeconds", "must be between 1 and 1800")
			return
		}
		timeout = time.Duration(seconds) * time.Second
	}
	item, terminal, err := h.investigations.Wait(r.Context(), strings.TrimSpace(mux.Vars(r)["id"]), timeout)
	if err != nil {
		writeInvestigationError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"investigation": item, "terminal": terminal})
}

func (h *Handler) CompleteInvestigation(w http.ResponseWriter, r *http.Request) {
	if h.investigations == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "investigation lifecycle is unavailable")
		return
	}
	var result investigation.Result
	decoder := json.NewDecoder(io.LimitReader(r.Body, maxInvestigationRequestBytes+1))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		writeSimpleError(w, r, "body", "invalid investigation result JSON")
		return
	}
	item, err := h.investigations.Complete(r.Context(), strings.TrimSpace(mux.Vars(r)["id"]), result)
	if err != nil {
		writeInvestigationError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) CancelInvestigation(w http.ResponseWriter, r *http.Request) {
	if h.investigations == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "investigation lifecycle is unavailable")
		return
	}
	item, err := h.cancelInvestigation(r.Context(), strings.TrimSpace(mux.Vars(r)["id"]))
	if err != nil {
		writeInvestigationError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) RetryInvestigationLearning(w http.ResponseWriter, r *http.Request) {
	if h.investigations == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "investigation lifecycle is unavailable")
		return
	}
	retrier, ok := h.svc.WorkflowService.(interface {
		RetryPendingInvestigationLearning(context.Context, string) (*investigation.Lifecycle, error)
	})
	if !ok {
		writeJSONError(w, http.StatusServiceUnavailable, "investigation learning retry is unavailable")
		return
	}
	item, err := retrier.RetryPendingInvestigationLearning(r.Context(), strings.TrimSpace(mux.Vars(r)["id"]))
	if err != nil {
		writeInvestigationError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) cancelInvestigation(ctx context.Context, id string) (*investigation.Lifecycle, error) {
	item, err := h.investigations.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if h.svc.WorkflowService != nil && item != nil && item.WorkflowRef != "" && !investigationTerminal(item.OperationStatus) {
		if executionID, parseErr := uuid.Parse(item.WorkflowRef); parseErr == nil {
			// Lifecycle cancellation is authoritative even if the workflow was
			// already settled or disappeared during recovery. The workflow call
			// is therefore best effort, while the durable lifecycle transition
			// remains idempotent and observable.
			_, _ = h.svc.CancelWorkflowExecution(ctx, orchestration.WorkflowExecutionOperationRequest{ExecutionID: executionID, IdempotencyKey: "cancel-typed-investigation/" + item.ID, Reason: "typed investigation cancelled"})
		}
	}
	return h.investigations.Cancel(ctx, id)
}

func writeInvestigationError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, investigation.ErrInvalidRequest), errors.Is(err, investigation.ErrInvalidResult):
		status = http.StatusBadRequest
	case errors.Is(err, investigation.ErrRequestKeyConflict):
		status = http.StatusConflict
	case errors.Is(err, investigation.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, investigation.ErrInvalidTransition):
		status = http.StatusConflict
	}
	if status == http.StatusInternalServerError {
		writeError(w, r, err)
		return
	}
	writeJSON(w, status, map[string]any{"error": err.Error()})
}
