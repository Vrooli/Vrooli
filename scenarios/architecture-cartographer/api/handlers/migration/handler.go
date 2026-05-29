// Package migration is the Connect-RPC surface for the migration domain —
// the stateful tracker that ingests the shared ArchitectureFinding
// photograph, tracks each finding through a lifecycle, hands the agent an
// ordered worklist, and reconciles re-audits by stable id.
package migration

import (
	"context"
	"errors"

	"architecture-cartographer/internal/migration"

	"connectrpc.com/connect"
	migrationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/migration"
	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/migration/migration_v1connect"
)

// Handler implements migration_v1connect.MigrationServiceHandler.
type Handler struct {
	migration_v1connect.UnimplementedMigrationServiceHandler
	svc migration.Service
}

// NewHandler constructs the Connect handler.
func NewHandler(svc migration.Service) *Handler { return &Handler{svc: svc} }

var _ migration_v1connect.MigrationServiceHandler = (*Handler)(nil)

// errorToConnectCode maps domain errors to Connect codes.
func errorToConnectCode(err error) connect.Code {
	var notFoundM migration.ErrMigrationNotFound
	var notFoundF migration.ErrFindingNotFound
	var invalid migration.ErrInvalidInput
	switch {
	case errors.As(err, &notFoundM), errors.As(err, &notFoundF):
		return connect.CodeNotFound
	case errors.As(err, &invalid):
		return connect.CodeInvalidArgument
	default:
		return connect.CodeInternal
	}
}

func (h *Handler) CreateMigration(ctx context.Context, req *connect.Request[migrationv1.CreateMigrationRequest]) (*connect.Response[migrationv1.CreateMigrationResponse], error) {
	st, err := h.svc.Create(ctx, req.Msg.GetScenario(), req.Msg.GetName(), req.Msg.GetFindings())
	if err != nil {
		return nil, connect.NewError(errorToConnectCode(err), err)
	}
	return connect.NewResponse(&migrationv1.CreateMigrationResponse{Status: statusProjectionToProto(st)}), nil
}

func (h *Handler) ListMigrations(ctx context.Context, req *connect.Request[migrationv1.ListMigrationsRequest]) (*connect.Response[migrationv1.ListMigrationsResponse], error) {
	ms, err := h.svc.List(ctx, req.Msg.GetScenario())
	if err != nil {
		return nil, connect.NewError(errorToConnectCode(err), err)
	}
	out := make([]*migrationv1.Migration, 0, len(ms))
	for _, m := range ms {
		out = append(out, migrationToProto(m))
	}
	return connect.NewResponse(&migrationv1.ListMigrationsResponse{Migrations: out}), nil
}

func (h *Handler) GetMigrationStatus(ctx context.Context, req *connect.Request[migrationv1.GetMigrationStatusRequest]) (*connect.Response[migrationv1.GetMigrationStatusResponse], error) {
	st, err := h.svc.Status(ctx, req.Msg.GetMigrationId())
	if err != nil {
		return nil, connect.NewError(errorToConnectCode(err), err)
	}
	return connect.NewResponse(&migrationv1.GetMigrationStatusResponse{Status: statusProjectionToProto(st)}), nil
}

func (h *Handler) NextMigrationStep(ctx context.Context, req *connect.Request[migrationv1.NextMigrationStepRequest]) (*connect.Response[migrationv1.NextMigrationStepResponse], error) {
	findings, err := h.svc.Next(ctx, req.Msg.GetMigrationId())
	if err != nil {
		return nil, connect.NewError(errorToConnectCode(err), err)
	}
	return connect.NewResponse(&migrationv1.NextMigrationStepResponse{Findings: findingsToProto(findings)}), nil
}

func (h *Handler) ResolveFinding(ctx context.Context, req *connect.Request[migrationv1.ResolveFindingRequest]) (*connect.Response[migrationv1.ResolveFindingResponse], error) {
	f, err := h.svc.Resolve(ctx, req.Msg.GetMigrationId(), req.Msg.GetStableId(), req.Msg.GetNote())
	if err != nil {
		return nil, connect.NewError(errorToConnectCode(err), err)
	}
	return connect.NewResponse(&migrationv1.ResolveFindingResponse{Finding: findingToProto(f)}), nil
}

func (h *Handler) ApplyFinding(ctx context.Context, req *connect.Request[migrationv1.ApplyFindingRequest]) (*connect.Response[migrationv1.ApplyFindingResponse], error) {
	// v1: apply records a hand-fix as a status-only transition (no file
	// write). Auto-execution of file-op fixes stays deferred to the
	// apply-execution plan; Phase 6 refines fix-kind discrimination.
	f, err := h.svc.Resolve(ctx, req.Msg.GetMigrationId(), req.Msg.GetStableId(), "applied (manual)")
	if err != nil {
		return nil, connect.NewError(errorToConnectCode(err), err)
	}
	return connect.NewResponse(&migrationv1.ApplyFindingResponse{Finding: findingToProto(f)}), nil
}

func (h *Handler) ReauditMigration(ctx context.Context, req *connect.Request[migrationv1.ReauditMigrationRequest]) (*connect.Response[migrationv1.ReauditMigrationResponse], error) {
	res, err := h.svc.Reaudit(ctx, req.Msg.GetMigrationId(), req.Msg.GetFindings())
	if err != nil {
		return nil, connect.NewError(errorToConnectCode(err), err)
	}
	return connect.NewResponse(&migrationv1.ReauditMigrationResponse{
		Validated:   findingsToProto(res.Validated),
		StillOpen:   findingsToProto(res.StillOpen),
		Regressions: findingsToProto(res.Regressions),
		Status:      statusProjectionToProto(res.Status),
	}), nil
}

func (h *Handler) CloseMigration(ctx context.Context, req *connect.Request[migrationv1.CloseMigrationRequest]) (*connect.Response[migrationv1.CloseMigrationResponse], error) {
	st, err := h.svc.Close(ctx, req.Msg.GetMigrationId())
	if err != nil {
		return nil, connect.NewError(errorToConnectCode(err), err)
	}
	return connect.NewResponse(&migrationv1.CloseMigrationResponse{Status: statusProjectionToProto(st)}), nil
}
