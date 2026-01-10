package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"scenario-to-cloud/deployment"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/httputil"
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

	// If run_id is specified, wait for it to match the current deployment's run_id.
	// This prevents returning stale "completed" state from a previous run when a new
	// deployment has just been started but the DB hasn't been updated yet.
	expectedRunID := r.URL.Query().Get("run_id")
	currentRunID := ""
	if dep.RunID != nil {
		currentRunID = *dep.RunID
	}
	if expectedRunID != "" && currentRunID != expectedRunID {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		matched := false
		for !matched {
			select {
			case <-ctx.Done():
				httputil.WriteAPIError(w, http.StatusConflict, httputil.APIError{
					Code:    "run_not_started",
					Message: "Deployment run has not started yet",
					Hint:    fmt.Sprintf("Expected run_id %s but found %s", expectedRunID, currentRunID),
				})
				return
			case <-ticker.C:
				dep, err = s.repo.GetDeployment(r.Context(), id)
				if err != nil {
					httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
						Code:    "get_failed",
						Message: "Failed to get deployment while waiting for run",
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
				if dep.RunID != nil && *dep.RunID == expectedRunID {
					matched = true
				}
			}
		}
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
