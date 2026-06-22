// Package advisor mounts storage-health's AdvisorService — migration-hygiene
// grading and Postgres→SQLite engine-fitness ranking across the fleet.
package advisor

import (
	"context"
	"log"

	"connectrpc.com/connect"

	internaladvisor "storage-health/internal/advisor"

	advisorv1 "github.com/vrooli/vrooli/packages/proto/gen/go/storage-health/v1/advisor"
	advisorconnect "github.com/vrooli/vrooli/packages/proto/gen/go/storage-health/v1/advisor/advisor_v1connect"
)

// Handler implements the generated AdvisorServiceHandler.
type Handler struct {
	advisorconnect.UnimplementedAdvisorServiceHandler
	svc    *internaladvisor.Service
	logger *log.Logger
}

// NewHandler builds an advisor Handler.
func NewHandler(svc *internaladvisor.Service, logger *log.Logger) *Handler {
	if logger == nil {
		logger = log.Default()
	}
	return &Handler{svc: svc, logger: logger}
}

var _ advisorconnect.AdvisorServiceHandler = (*Handler)(nil)

// AnalyzeMigrations grades migration hygiene per scenario.
func (h *Handler) AnalyzeMigrations(ctx context.Context, req *connect.Request[advisorv1.AnalyzeMigrationsRequest]) (*connect.Response[advisorv1.AnalyzeMigrationsResponse], error) {
	res, err := h.svc.AnalyzeMigrations(ctx, req.Msg.GetScenarios())
	if err != nil {
		h.logger.Printf("advisor.AnalyzeMigrations: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &advisorv1.AnalyzeMigrationsResponse{
		ScenarioCount:       int32(res.ScenarioCount),
		WithMigrationsCount: int32(res.WithMigrationsCount),
		DebtCount:           int32(res.DebtCount),
	}
	for _, e := range res.Entries {
		out.Entries = append(out.Entries, &advisorv1.MigrationHygiene{
			Scenario:            e.Scenario,
			StorageStage:        e.StorageStage,
			HasMigrations:       e.HasMigrations,
			HasAlterInSchema:    e.HasAlterInSchema,
			NonIdempotentSchema: e.NonIdempotentSchema,
			MigrationDebt:       int32(e.MigrationDebt),
			Notes:               e.Notes,
		})
	}
	for _, e := range res.Errors {
		out.Errors = append(out.Errors, &advisorv1.AdvisorScanError{Scenario: e.Scenario, Reason: e.Reason})
	}
	return connect.NewResponse(out), nil
}

// AdviseEngines ranks Postgres→SQLite migration candidates by fitness.
func (h *Handler) AdviseEngines(ctx context.Context, req *connect.Request[advisorv1.AdviseEnginesRequest]) (*connect.Response[advisorv1.AdviseEnginesResponse], error) {
	res, err := h.svc.AdviseEngines(ctx, req.Msg.GetScenarios())
	if err != nil {
		h.logger.Printf("advisor.AdviseEngines: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &advisorv1.AdviseEnginesResponse{ScenarioCount: int32(res.ScenarioCount)}
	for _, c := range res.Candidates {
		out.Candidates = append(out.Candidates, &advisorv1.EngineCandidate{
			Scenario:          c.Scenario,
			CurrentEngine:     c.CurrentEngine,
			RecommendedEngine: c.RecommendedEngine,
			FitnessScore:      c.FitnessScore,
			Rationale:         c.Rationale,
			Autofixable:       c.Autofixable,
			Blockers:          c.Blockers,
		})
	}
	for _, e := range res.Errors {
		out.Errors = append(out.Errors, &advisorv1.AdvisorScanError{Scenario: e.Scenario, Reason: e.Reason})
	}
	return connect.NewResponse(out), nil
}
