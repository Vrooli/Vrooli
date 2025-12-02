package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"test-genie/internal/orchestrator"
	"test-genie/internal/queue"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := "healthy"
	dbStatus := "connected"

	if err := s.db.PingContext(r.Context()); err != nil {
		status = "unhealthy"
		dbStatus = "disconnected"
	}

	operations := map[string]interface{}{}
	if s.suiteRequests != nil {
		if snapshot, err := s.suiteRequests.StatusSnapshot(r.Context()); err == nil {
			operations["queue"] = queueSnapshotPayload(snapshot)
		} else if err != nil {
			s.log("queue snapshot failed", map[string]interface{}{"error": err.Error()})
		}
	}
	if s.executionHistory != nil {
		if latest, err := s.executionHistory.Latest(r.Context()); err == nil && latest != nil {
			operations["lastExecution"] = executionSummaryPayload(latest)
		} else if err != nil {
			s.log("latest execution lookup failed", map[string]interface{}{"error": err.Error()})
		}
	}

	response := map[string]interface{}{
		"status":    status,
		"service":   s.serviceName(),
		"version":   "1.0.0",
		"readiness": status == "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"dependencies": map[string]string{
			"database": dbStatus,
		},
		"operations": operations,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func queueSnapshotPayload(snapshot queue.SuiteRequestSnapshot) map[string]interface{} {
	payload := map[string]interface{}{
		"total":     snapshot.Total,
		"queued":    snapshot.Queued,
		"delegated": snapshot.Delegated,
		"running":   snapshot.Running,
		"completed": snapshot.Completed,
		"failed":    snapshot.Failed,
		"pending":   snapshot.Queued + snapshot.Delegated,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	if snapshot.OldestQueuedAt != nil {
		payload["oldestQueuedAt"] = snapshot.OldestQueuedAt.Format(time.RFC3339)
		age := time.Since(*snapshot.OldestQueuedAt).Seconds()
		if age < 0 {
			age = 0
		}
		payload["oldestQueuedAgeSeconds"] = int(age)
	}
	return payload
}

func executionSummaryPayload(result *orchestrator.SuiteExecutionResult) map[string]interface{} {
	if result == nil {
		return nil
	}
	return map[string]interface{}{
		"executionId":  result.ExecutionID,
		"scenario":     result.ScenarioName,
		"success":      result.Success,
		"completedAt":  result.CompletedAt.Format(time.RFC3339),
		"startedAt":    result.StartedAt.Format(time.RFC3339),
		"phaseSummary": result.PhaseSummary,
		"preset":       result.PresetUsed,
	}
}
