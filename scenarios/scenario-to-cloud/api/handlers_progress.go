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
	dep, op, ok := s.loadProgressTarget(w, r, id)
	if !ok {
		return
	}
	flusher, ok := prepareProgressStream(w)
	if !ok {
		return
	}
	sendEvent := progressEventSender(w, flusher)
	sendPreflightResult := progressPreflightSender(dep, sendEvent)

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

	if sendTerminalDeploymentEvent(dep, sendEvent, sendPreflightResult) {
		return
	}
	s.streamDeploymentProgress(w, r, flusher, sendEvent, sendPreflightResult, dep)
}

func (s *Server) loadProgressTarget(w http.ResponseWriter, r *http.Request, id string) (*domain.Deployment, *domain.CloudOperation, bool) {
	dep, err := s.repo.GetDeployment(r.Context(), id)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{Code: "get_failed", Message: "Failed to get deployment", Hint: err.Error()})
		return nil, nil, false
	}
	if dep == nil {
		httputil.WriteAPIError(w, http.StatusNotFound, httputil.APIError{Code: "not_found", Message: "Deployment not found"})
		return nil, nil, false
	}
	op, ok := s.progressOperation(w, r, id)
	return dep, op, ok
}

func (s *Server) progressOperation(w http.ResponseWriter, r *http.Request, deploymentID string) (*domain.CloudOperation, bool) {
	opID := r.URL.Query().Get("operation_id")
	if opID == "" {
		return nil, true
	}
	if s.operations == nil {
		apierrors.Write(w, apierrors.New(apierrors.CodeInternal, "Operation owner is not configured"))
		return nil, false
	}
	op, err := s.operations.Get(r.Context(), opID)
	if err != nil {
		apierrors.Write(w, err)
		return nil, false
	}
	if op.DeploymentID != deploymentID {
		apierrors.Write(w, apierrors.New(apierrors.CodeOperationNotFound, "Operation does not belong to this deployment").WithDetail("operation_id", opID))
		return nil, false
	}
	return op, true
}

func prepareProgressStream(w http.ResponseWriter) (http.Flusher, bool) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, ok := w.(http.Flusher)
	if !ok {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{Code: "sse_not_supported", Message: "Streaming not supported"})
		return nil, false
	}
	return flusher, true
}

func progressEventSender(w http.ResponseWriter, flusher http.Flusher) func(deployment.Event) {
	return func(event deployment.Event) {
		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
		flusher.Flush()
	}
}

func progressPreflightSender(dep *domain.Deployment, sendEvent func(deployment.Event)) func() {
	var result *domain.PreflightResponse
	if dep.PreflightResult.Valid && len(dep.PreflightResult.Data) > 0 {
		var decoded domain.PreflightResponse
		if json.Unmarshal(dep.PreflightResult.Data, &decoded) == nil {
			result = &decoded
		}
	}
	return func() {
		if result == nil {
			return
		}
		sendEvent(deployment.Event{Type: "preflight_result", Step: "preflight", StepTitle: "Running preflight checks", PreflightResult: result, Progress: dep.ProgressPercent, Timestamp: time.Now().UTC().Format(time.RFC3339)})
	}
}

func sendTerminalDeploymentEvent(dep *domain.Deployment, sendEvent func(deployment.Event), sendPreflightResult func()) bool {
	switch dep.Status {
	case domain.StatusDeployed:
		sendPreflightResult()
		sendEvent(deployment.Event{Type: "completed", Progress: 100, Message: "Deployment complete", Timestamp: time.Now().UTC().Format(time.RFC3339)})
	case domain.StatusFailed:
		sendPreflightResult()
		errMsg := "Deployment failed"
		if dep.ErrorMessage != nil {
			errMsg = *dep.ErrorMessage
		}
		step := ""
		if dep.ProgressStep != nil {
			step = *dep.ProgressStep
		}
		sendEvent(deployment.Event{Type: "deployment_error", Step: step, Error: errMsg, Progress: dep.ProgressPercent, Timestamp: time.Now().UTC().Format(time.RFC3339)})
	case domain.StatusStopped:
		sendPreflightResult()
		sendEvent(deployment.Event{Type: "deployment_error", Error: "Deployment was stopped", Progress: 0, Timestamp: time.Now().UTC().Format(time.RFC3339)})
	default:
		return false
	}
	return true
}

func (s *Server) streamDeploymentProgress(w http.ResponseWriter, r *http.Request, flusher http.Flusher, sendEvent func(deployment.Event), sendPreflightResult func(), dep *domain.Deployment) {
	if dep.ProgressStep != nil && *dep.ProgressStep != "" {
		sendEvent(deployment.Event{Type: "progress_update", Step: *dep.ProgressStep, Progress: dep.ProgressPercent, Timestamp: time.Now().UTC().Format(time.RFC3339)})
	}
	sendPreflightResult()
	ch := s.progressHub.Subscribe(dep.ID)
	defer s.progressHub.Unsubscribe(dep.ID, ch)
	keepAlive := time.NewTicker(15 * time.Second)
	defer keepAlive.Stop()
	for {
		select {
		case event, ok := <-ch:
			if !ok {
				return
			}
			sendEvent(event)
			if event.Type == "completed" || event.Type == "error" {
				return
			}
		case <-keepAlive.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		case <-r.Context().Done():
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
	sendOperationProgress(sendEvent, op, dep)
	hubCh := s.progressHub.Subscribe(dep.ID)
	defer s.progressHub.Unsubscribe(dep.ID, hubCh)
	opCh, unsubscribe := s.operations.Subscribe(op.ID)
	defer unsubscribe()
	done, cancelWaiter := s.operations.RegisterWaiter(op.ID)
	defer cancelWaiter()
	if s.sendOperationIfTerminal(r, sendEvent, op.ID, dep) {
		return
	}
	s.streamOperationEvents(w, r, flusher, sendEvent, op, dep, hubCh, opCh, done)
}

func sendOperationProgress(sendEvent func(deployment.Event), op *domain.CloudOperation, dep *domain.Deployment) {
	if dep.ProgressStep != nil && *dep.ProgressStep != "" {
		sendEvent(deployment.Event{Type: "progress_update", Step: *dep.ProgressStep, Progress: dep.ProgressPercent, Timestamp: time.Now().UTC().Format(time.RFC3339)})
	}
	sendEvent(deployment.Event{Type: "operation", Step: operations.StandingOf(op).ActiveStep, Message: string(op.State), Timestamp: time.Now().UTC().Format(time.RFC3339)})
}

func (s *Server) sendOperationIfTerminal(r *http.Request, sendEvent func(deployment.Event), operationID string, dep *domain.Deployment) bool {
	// Recheck after registering so a terminal transition racing the
	// subscription is not missed.
	if current, err := s.operations.Get(r.Context(), operationID); err == nil && current.State.IsTerminal() {
		sendOperationTerminal(sendEvent, current, dep)
		return true
	}
	return false
}

func (s *Server) streamOperationEvents(w http.ResponseWriter, r *http.Request, flusher http.Flusher, sendEvent func(deployment.Event), op *domain.CloudOperation, dep *domain.Deployment, hubCh <-chan deployment.Event, opCh <-chan operations.Event, done <-chan struct{}) {
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
			s.sendCurrentOperationTerminal(r, sendEvent, op.ID, dep)
			return
		case <-keepAlive.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) sendCurrentOperationTerminal(r *http.Request, sendEvent func(deployment.Event), operationID string, dep *domain.Deployment) {
	current, err := s.operations.Get(r.Context(), operationID)
	if err == nil {
		sendOperationTerminal(sendEvent, current, dep)
	}
}
