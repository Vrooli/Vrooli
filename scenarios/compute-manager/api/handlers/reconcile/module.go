package reconcile

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"compute-manager/internal/module"
	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	reconcilev1 "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/reconcile"
	reconcileconnect "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/reconcile/reconcile_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type db interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}
type service struct {
	reconcileconnect.UnimplementedReconcileServiceHandler
	db db
}

func NewModule(database db) (module.Module, error) {
	if database == nil {
		return module.Module{}, errors.New("reconcile database is required")
	}
	s := &service{db: database}
	path, h := reconcileconnect.NewReconcileServiceHandler(s)
	return module.Module{Name: "reconcile", Mount: func(r *mux.Router) { r.PathPrefix(path).Handler(h) }, Endpoints: Endpoints}, nil
}

func (s *service) RunReconciliation(ctx context.Context, _ *connect.Request[reconcilev1.RunReconciliationRequest]) (*connect.Response[reconcilev1.RunReconciliationResponse], error) {
	result, err := s.list(ctx, "")
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&reconcilev1.RunReconciliationResponse{Findings: result}), nil
}
func (s *service) ListFindings(ctx context.Context, req *connect.Request[reconcilev1.ListFindingsRequest]) (*connect.Response[reconcilev1.ListFindingsResponse], error) {
	result, err := s.list(ctx, req.Msg.GetStatus())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&reconcilev1.ListFindingsResponse{Findings: result}), nil
}
func (s *service) list(ctx context.Context, status string) ([]*reconcilev1.Finding, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,observed_at,kind,provider,provider_instance_id,instance_id,status,detail_json FROM reconcile_findings WHERE (?='' OR status=?) ORDER BY observed_at DESC`, status, status)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer rows.Close()
	out := []*reconcilev1.Finding{}
	for rows.Next() {
		var f reconcilev1.Finding
		var observed, kind, detail string
		if err := rows.Scan(&f.Id, &observed, &kind, &f.Provider, &f.ProviderInstanceId, &f.InstanceId, &f.Status, &detail); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		f.Kind = reconcilev1.FindingKind(reconcilev1.FindingKind_value["FINDING_KIND_"+strings.ToUpper(kind)])
		f.Detail = detail
		f.ObservedAt = stamp(observed)
		out = append(out, &f)
	}
	if err := rows.Err(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return out, nil
}
func (s *service) GetFinding(ctx context.Context, req *connect.Request[reconcilev1.GetFindingRequest]) (*connect.Response[reconcilev1.GetFindingResponse], error) {
	row := s.db.QueryRowContext(ctx, `SELECT id,observed_at,kind,provider,provider_instance_id,instance_id,status,detail_json FROM reconcile_findings WHERE id=?`, req.Msg.GetId())
	var f reconcilev1.Finding
	var observed, kind, detail string
	if err := row.Scan(&f.Id, &observed, &kind, &f.Provider, &f.ProviderInstanceId, &f.InstanceId, &f.Status, &detail); errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("finding %q not found", req.Msg.GetId()))
	} else if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	f.Kind = reconcilev1.FindingKind(reconcilev1.FindingKind_value["FINDING_KIND_"+strings.ToUpper(kind)])
	f.Detail = detail
	f.ObservedAt = stamp(observed)
	return connect.NewResponse(&reconcilev1.GetFindingResponse{Finding: &f}), nil
}
func (s *service) QuarantineFinding(ctx context.Context, req *connect.Request[reconcilev1.QuarantineFindingRequest]) (*connect.Response[reconcilev1.QuarantineFindingResponse], error) {
	if _, err := s.db.ExecContext(ctx, `UPDATE reconcile_findings SET status='quarantined' WHERE id=?`, req.Msg.GetId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response, err := s.GetFinding(ctx, connect.NewRequest(&reconcilev1.GetFindingRequest{Id: req.Msg.GetId()}))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&reconcilev1.QuarantineFindingResponse{Finding: response.Msg.GetFinding()}), nil
}
func stamp(value string) *timestamppb.Timestamp {
	if value == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return nil
	}
	return timestamppb.New(t)
}
