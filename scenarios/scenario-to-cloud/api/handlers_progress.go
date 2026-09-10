package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/deployment"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/httputil"
	"scenario-to-cloud/operations"

	"github.com/gorilla/mux"
)

// handleDeploymentProgress streams deployment progress events via SSE.
func (s *Server) handleDeploymentProgress(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	// Check if deployment exists
	dep, err := s.repo.GetDeployment(r.Context(), id)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "get_failed",
			Message: "Failed to get deployment",
			Hint:    err.Error(),
		})
		return
	}

	if dep == nil {
		httputil.WriteAPIError(w, http.StatusNotFound, httputil.APIError{
			Code:    "not_found",
			Message: "Deployment not found",
		})
		return
	}

	// The stream may be keyed by operation id (DL-05: the durable operation
	// replaced run_id). A terminal operation replays its outcome and closes;
	// a live one streams until the operation's terminal event.
	var op *domain.CloudOperation
	if opID := r.URL.Query().Get("operation_id"); opID != "" {
		if s.operations == nil {
			apierrors.Write(w, apierrors.New(apierrors.CodeInternal, "Operation owner is not configured"))
			return
		}
		found, err := s.operations.Get(r.Context(), opID)
		if err != nil {
			apierrors.Write(w, err)
			return
		}
		if found.DeploymentID != id {
			apierrors.Write(w, apierrors.New(apierrors.CodeOperationNotFound, "Operation does not belong to this deployment").WithDetail("operation_id", opID))
			return
		}
		op = found
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	flusher, ok := w.(http.Flusher)
	if !ok {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "sse_not_supported",
			Message: "Streaming not supported",
		})
		return
	}

	// Helper to send an SSE event
	sendEvent := func(event deployment.Event) {
		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
		flusher.Flush()
	}

	var preflightResult *domain.PreflightResponse
	if dep.PreflightResult.Valid && len(dep.PreflightResult.Data) > 0 {
		var result domain.PreflightResponse
		if err := json.Unmarshal(dep.PreflightResult.Data, &result); err == nil {
			preflightResult = &result
		}
	}

	sendPreflightResult := func() {
		if preflightResult == nil {
			return
		}
		sendEvent(deployment.Event{
			Type:            "preflight_result",
			Step:            "preflight",
			StepTitle:       "Running preflight checks",
			PreflightResult: preflightResult,
			Progress:        dep.ProgressPercent,
			Timestamp:       time.Now().UTC().Format(time.RFC3339),
		})
	}

	if op != nil && op.State.IsTerminal() {
		sendPreflightResult()
		sendOperationTerminal(sendEvent, op, dep)
		return
	}
	if op != nil {
		// A live operation: stream its events until terminal, regardless of
		// the deployment status projection.
		s.streamOperation(w, r, flusher, sendEvent, op, dep)
		return
	}

	// Check if deployment is already complete
	switch dep.Status {
	case domain.StatusDeployed:
		sendPreflightResult()
		sendEvent(deployment.Event{
			Type:      "completed",
			Progress:  100,
			Message:   "Deployment complete",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return

	case domain.StatusFailed:
		sendPreflightResult()
		errMsg := "Deployment failed"
		if dep.ErrorMessage != nil {
			errMsg = *dep.ErrorMessage
		}
		// Include the step that failed so UI can reconstruct step states
		step := ""
		if dep.ProgressStep != nil {
			step = *dep.ProgressStep
		}
		sendEvent(deployment.Event{
			Type:      "deployment_error",
			Step:      step,
			Error:     errMsg,
			Progress:  dep.ProgressPercent,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return

	case domain.StatusStopped:
		sendPreflightResult()
		sendEvent(deployment.Event{
			Type:      "deployment_error",
			Error:     "Deployment was stopped",
			Progress:  0,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	// Send current progress state for reconnecting clients
	if dep.ProgressStep != nil && *dep.ProgressStep != "" {
		sendEvent(deployment.Event{
			Type:      "progress_update",
			Step:      *dep.ProgressStep,
			Progress:  dep.ProgressPercent,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
	}
	sendPreflightResult()

	// Subscribe to live updates
	ch := s.progressHub.Subscribe(id)
	defer s.progressHub.Unsubscribe(id, ch)

	// Keep-alive ticker to detect disconnected clients
	keepAlive := time.NewTicker(15 * time.Second)
	defer keepAlive.Stop()

	for {
		select {
		case event, ok := <-ch:
			if !ok {
				// Channel closed, deployment complete
				return
			}

			sendEvent(event)

			// Close connection after terminal events
			if event.Type == "completed" || event.Type == "error" {
				return
			}

		case <-keepAlive.C:
			// Send a comment to keep the connection alive
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()

		case <-r.Context().Done():
			// Client disconnected
			return
		}
	}
}

// sendOperationTerminal replays a terminal operation as the legacy terminal
// SSE event so existing consumers close correctly.
func sendOperationTerminal(sendEvent func(deployment.Event), op *domain.CloudOperation, dep *domain.Deployment) {
	standing := operations.StandingOf(op)
	switch op.State {
	case operations.Succeeded:
		sendEvent(deployment.Event{Type: "completed", Progress: 100, Message: "Deployment complete", Timestamp: time.Now().UTC().Format(time.RFC3339)})
	default:
		msg := string(op.State)
		if standing.Error != nil {
			msg = standing.Error.Message
		} else if standing.Result != nil && standing.Result.Message != "" {
			msg = standing.Result.Message
		}
		sendEvent(deployment.Event{Type: "deployment_error", Step: standing.ActiveStep, Error: msg, Progress: dep.ProgressPercent, Timestamp: time.Now().UTC().Format(time.RFC3339)})
	}
}

// streamOperation forwards hub events for the deployment and the operation's
// own events until the operation is terminal.
func (s *Server) streamOperation(w http.ResponseWriter, r *http.Request, flusher http.Flusher, sendEvent func(deployment.Event), op *domain.CloudOperation, dep *domain.Deployment) {
	if dep.ProgressStep != nil && *dep.ProgressStep != "" {
		sendEvent(deployment.Event{Type: "progress_update", Step: *dep.ProgressStep, Progress: dep.ProgressPercent, Timestamp: time.Now().UTC().Format(time.RFC3339)})
	}
	sendEvent(deployment.Event{Type: "operation", Step: operations.StandingOf(op).ActiveStep, Message: string(op.State), Timestamp: time.Now().UTC().Format(time.RFC3339)})
	hubCh := s.progressHub.Subscribe(dep.ID)
	defer s.progressHub.Unsubscribe(dep.ID, hubCh)
	opCh, unsubscribe := s.operations.Subscribe(op.ID)
	defer unsubscribe()
	done, cancelWaiter := s.operations.RegisterWaiter(op.ID)
	defer cancelWaiter()
	// Recheck after registering so a terminal transition racing the
	// subscription is not missed.
	if current, err := s.operations.Get(r.Context(), op.ID); err == nil && current.State.IsTerminal() {
		sendOperationTerminal(sendEvent, current, dep)
		return
	}
	keepAlive := time.NewTicker(15 * time.Second)
	defer keepAlive.Stop()
	for {
		select {
		case event, ok := <-hubCh:
			if !ok {
				return
			}
			if event.Type == "completed" || event.Type == "error" {
				// Terminal projection events are replayed from the record below.
				continue
			}
			sendEvent(event)
		case ev := <-opCh:
			sendEvent(deployment.Event{Type: "operation", Step: ev.Step, Message: ev.Message, Timestamp: ev.Timestamp})
		case <-done:
			if current, err := s.operations.Get(r.Context(), op.ID); err == nil {
				sendOperationTerminal(sendEvent, current, dep)
			}
			return
		case <-keepAlive.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
