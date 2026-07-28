package execution

import (
	"context"
	"time"

	"github.com/gorilla/mux"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
)

// DOC: docs/concepts/ARCHITECTURE.md#api-boundaries
// DOC: docs/reference/operational-targets.md
// DOC: docs/internal/TEMPORAL-FLOWS.md

// Handler exposes execution-control endpoints.
type Handler struct {
	service *Service
}

// NewHandler creates a handler with filesystem-backed storage.
func NewHandler(cfg ServiceConfig) *Handler {
	return &Handler{service: NewService(cfg)}
}

// NewHandlerFromService creates a handler from an existing Service.
func NewHandlerFromService(svc *Service) *Handler {
	return &Handler{service: svc}
}

// RegisterRoutes registers execution routes.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/execution", h.List).Methods("GET")
	r.HandleFunc("/api/v1/execution", h.Create).Methods("POST")
	r.HandleFunc("/api/v1/execution/strategies", h.Strategies).Methods("GET")
	r.HandleFunc("/api/v1/execution/{execution_id}", h.Get).Methods("GET")
	r.HandleFunc("/api/v1/execution/{execution_id}/prompt-trace", h.GetPromptTrace).Methods("GET")
	r.HandleFunc("/api/v1/execution/{execution_id}/progress", h.GetProgress).Methods("GET")
	r.HandleFunc("/api/v1/execution/{execution_id}/start", h.Start).Methods("POST")
	r.HandleFunc("/api/v1/execution/{execution_id}/cancel", h.Cancel).Methods("POST")
	// Deprecated transition aliases: use TransitionService.StartTransition/ApplyTransition.
	r.HandleFunc("/api/v1/execution/{execution_id}/workflow/apply", h.ApplyWorkflow).Methods("POST")
	r.HandleFunc("/api/v1/execution/{execution_id}/workflow/approve", h.ApprovePhasedPlanWorkflow).Methods("POST")
	r.HandleFunc("/api/v1/execution/{execution_id}/retry", h.Retry).Methods("POST")
	r.HandleFunc("/api/v1/execution/{execution_id}/follow-up", h.FollowUp).Methods("POST")
	r.HandleFunc("/api/v1/execution/{execution_id}/trigger-review", h.TriggerReview).Methods("POST")
	r.HandleFunc("/api/v1/execution/circuit-breaker/reset", h.ResetCircuitBreaker).Methods("POST")
	r.HandleFunc("/api/v1/gct/status", h.GCTStatus).Methods("GET")
}

// StartBackgroundWorker launches the background worker for active execution
// progression.
func (h *Handler) StartBackgroundWorker(stop <-chan struct{}) {
	_ = h.service.ProcessActiveExecutions(context.Background())
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			_ = h.service.ProcessActiveExecutions(context.Background())
		}
	}
}

func executionResponse(record Record) *apipb.ExecutionResponse {
	return &apipb.ExecutionResponse{Execution: recordToProto(record)}
}

func recordToProto(r Record) *domainpb.ExecutionRecord {
	pb := &domainpb.ExecutionRecord{
		ExecutionId: r.ExecutionID,
		BacklogKind: r.BacklogKind,
		BacklogName: r.BacklogName,
		Status:      string(r.Status),
		Mode:        string(r.Mode),
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
	if r.TaskID != "" {
		pb.TaskId = &r.TaskID
	}
	if r.RunID != "" {
		pb.RunId = &r.RunID
	}
	if r.StartedAt != "" {
		pb.StartedAt = &r.StartedAt
	}
	if r.FinishedAt != "" {
		pb.FinishedAt = &r.FinishedAt
	}
	if r.FailureReason != "" {
		pb.FailureReason = &r.FailureReason
	}
	if r.StartedBy != "" {
		pb.StartedBy = &r.StartedBy
	}
	if r.Operation != "" {
		pb.Operation = &r.Operation
	}
	if r.ArchiveContext != nil {
		ac := r.ArchiveContext
		pbAc := &domainpb.ArchiveContext{
			ScenarioName:  ac.ScenarioName,
			ScenarioPath:  ac.ScenarioPath,
			PreservePaths: ac.PreservePaths,
		}
		if ac.PresetOrCustom != "" {
			pbAc.PresetOrCustom = &ac.PresetOrCustom
		}
		if ac.PreservePreset != "" {
			pbAc.PreservePreset = &ac.PreservePreset
		}
		pb.ArchiveContext = pbAc
	}
	if r.ParentExecutionID != "" {
		pb.ParentExecutionId = &r.ParentExecutionID
	}
	pb.FixupAttempt = int32(r.FixupAttempt)
	if finalization := r.Finalization; finalization != nil {
		pb.Finalization = finalizationToProto(finalization)
	}
	return pb
}

func reviewResultToProto(rr *ReviewResult) *domainpb.ReviewResult {
	pb := &domainpb.ReviewResult{
		JobId:          rr.JobID,
		Classification: rr.Classification,
		Summary:        rr.Summary,
		ReviewedAt:     rr.ReviewedAt,
	}
	if len(rr.RawDimensions) > 0 {
		s := string(rr.RawDimensions)
		pb.RawDimensions = &s
	}
	for _, dim := range rr.Dimensions {
		pbDim := &domainpb.ReviewDimension{
			Name:   dim.Name,
			Status: dim.Status,
		}
		if dim.Details != "" {
			pbDim.Details = &dim.Details
		}
		pb.Dimensions = append(pb.Dimensions, pbDim)
	}
	return pb
}

func finalizationToProto(finalization *Finalization) *domainpb.Finalization {
	if finalization == nil {
		return nil
	}
	pb := &domainpb.Finalization{
		Eligible:                finalization.Eligible,
		Status:                  string(finalization.Status),
		Phase:                   finalization.Phase,
		ScopeSource:             finalization.ScopeSource,
		AffectedScenarios:       append([]string(nil), finalization.AffectedScenarios...),
		AggregateClassification: finalization.AggregateClassification,
	}
	if finalization.SkipReason != "" {
		pb.SkipReason = &finalization.SkipReason
	}
	if finalization.StartedAt != "" {
		pb.StartedAt = &finalization.StartedAt
	}
	if finalization.CompletedAt != "" {
		pb.CompletedAt = &finalization.CompletedAt
	}
	if finalization.AggregateSummary != "" {
		pb.AggregateSummary = &finalization.AggregateSummary
	}
	for _, warning := range finalization.Warnings {
		pbWarning := &domainpb.FinalizationWarning{
			Code:      warning.Code,
			Message:   warning.Message,
			Retryable: warning.Retryable,
			CreatedAt: warning.CreatedAt,
		}
		if warning.ScenarioName != "" {
			pbWarning.ScenarioName = &warning.ScenarioName
		}
		pb.Warnings = append(pb.Warnings, pbWarning)
	}
	for _, scenario := range finalization.Scenarios {
		pbScenario := &domainpb.ScenarioFinalization{
			ScenarioName: scenario.ScenarioName,
			ChangedPaths: append([]string(nil), scenario.ChangedPaths...),
			Restart: &domainpb.RestartResult{
				Status:   string(scenario.Restart.Status),
				Attempts: int32(scenario.Restart.Attempts),
			},
			Health: &domainpb.HealthCheckResult{
				Status:      string(scenario.Health.Status),
				SchemaValid: scenario.Health.SchemaValid,
			},
			Review: &domainpb.ScenarioReview{
				Status: string(scenario.Review.Status),
			},
		}
		if scenario.Restart.LastError != "" {
			pbScenario.Restart.LastError = &scenario.Restart.LastError
		}
		if scenario.Restart.StartedAt != "" {
			pbScenario.Restart.StartedAt = &scenario.Restart.StartedAt
		}
		if scenario.Restart.FinishedAt != "" {
			pbScenario.Restart.FinishedAt = &scenario.Restart.FinishedAt
		}
		if scenario.Health.ScenarioStatus != "" {
			pbScenario.Health.ScenarioStatus = &scenario.Health.ScenarioStatus
		}
		if scenario.Health.HealthStatus != "" {
			pbScenario.Health.HealthStatus = &scenario.Health.HealthStatus
		}
		if scenario.Health.Details != "" {
			pbScenario.Health.Details = &scenario.Health.Details
		}
		if scenario.Health.CheckedAt != "" {
			pbScenario.Health.CheckedAt = &scenario.Health.CheckedAt
		}
		if scenario.Review.JobID != "" {
			pbScenario.Review.JobId = &scenario.Review.JobID
		}
		if scenario.Review.SkipReason != "" {
			pbScenario.Review.SkipReason = &scenario.Review.SkipReason
		}
		if scenario.Review.Result != nil {
			pbScenario.Review.Result = reviewResultToProto(scenario.Review.Result)
		}
		pb.Scenarios = append(pb.Scenarios, pbScenario)
	}
	return pb
}
