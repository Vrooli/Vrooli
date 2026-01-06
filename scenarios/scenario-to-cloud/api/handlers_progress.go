package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"scenario-to-cloud/domain"
)

// handleDeploymentProgress streams deployment progress events via SSE.
func (s *Server) handleDeploymentProgress(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	// Check if deployment exists
	deployment, err := s.repo.GetDeployment(r.Context(), id)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "get_failed",
			Message: "Failed to get deployment",
			Hint:    err.Error(),
		})
		return
	}

	if deployment == nil {
		writeAPIError(w, http.StatusNotFound, APIError{
			Code:    "not_found",
			Message: "Deployment not found",
		})
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "sse_not_supported",
			Message: "Streaming not supported",
		})
		return
	}

	// Helper to send an SSE event
	sendEvent := func(event ProgressEvent) {
		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
		flusher.Flush()
	}

	var preflightResult *domain.PreflightResponse
	if deployment.PreflightResult.Valid && len(deployment.PreflightResult.Data) > 0 {
		var result domain.PreflightResponse
		if err := json.Unmarshal(deployment.PreflightResult.Data, &result); err == nil {
			preflightResult = &result
		}
	}

	sendPreflightResult := func() {
		if preflightResult == nil {
			return
		}
		sendEvent(ProgressEvent{
			Type:            "preflight_result",
			Step:            "preflight",
			StepTitle:       "Running preflight checks",
			PreflightResult: preflightResult,
			Progress:        deployment.ProgressPercent,
			Timestamp:       time.Now().UTC().Format(time.RFC3339),
		})
	}

	// Check if deployment is already complete
	switch deployment.Status {
	case domain.StatusDeployed:
		sendPreflightResult()
		sendEvent(ProgressEvent{
			Type:      "completed",
			Progress:  100,
			Message:   "Deployment complete",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return

	case domain.StatusFailed:
		sendPreflightResult()
		errMsg := "Deployment failed"
		if deployment.ErrorMessage != nil {
			errMsg = *deployment.ErrorMessage
		}
		// Include the step that failed so UI can reconstruct step states
		step := ""
		if deployment.ProgressStep != nil {
			step = *deployment.ProgressStep
		}
		sendEvent(ProgressEvent{
			Type:      "deployment_error",
			Step:      step,
			Error:     errMsg,
			Progress:  deployment.ProgressPercent,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return

	case domain.StatusStopped:
		sendPreflightResult()
		sendEvent(ProgressEvent{
			Type:      "deployment_error",
			Error:     "Deployment was stopped",
			Progress:  0,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	// Send current progress state for reconnecting clients
	if deployment.ProgressStep != nil && *deployment.ProgressStep != "" {
		sendEvent(ProgressEvent{
			Type:      "progress_update",
			Step:      *deployment.ProgressStep,
			Progress:  deployment.ProgressPercent,
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
