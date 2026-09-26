package reconcile

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"compute-manager/internal/clock"
	"compute-manager/internal/module"
	internalreconcile "compute-manager/internal/reconcile"
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
	db      db
	run     func(context.Context) ([]*reconcilev1.Finding, error)
	destroy func(context.Context, string, string) error
}

func NewModule(database db) (module.Module, error) {
	return newModule(database, nil)
}

func NewModuleWithRunner(database db, runner func(context.Context) ([]*reconcilev1.Finding, error)) (module.Module, error) {
	return newModule(database, runner)
}

func NewModuleWithRunnerAndDestroy(database db, runner func(context.Context) ([]*reconcilev1.Finding, error), destroy func(context.Context, string, string) error) (module.Module, error) {
	return newModuleWithDestroy(database, runner, destroy)
}

func newModule(database db, runner func(context.Context) ([]*reconcilev1.Finding, error)) (module.Module, error) {
	return newModuleWithDestroy(database, runner, nil)
}

func newModuleWithDestroy(database db, runner func(context.Context) ([]*reconcilev1.Finding, error), destroy func(context.Context, string, string) error) (module.Module, error) {
	if database == nil {
		return module.Module{}, errors.New("reconcile database is required")
	}
	s := &service{db: database, run: runner, destroy: destroy}
	path, h := reconcileconnect.NewReconcileServiceHandler(s)
	return module.Module{Name: "reconcile", Mount: func(r *mux.Router) { r.PathPrefix(path).Handler(h) }, Endpoints: Endpoints}, nil
}

func (s *service) RunReconciliation(ctx context.Context, _ *connect.Request[reconcilev1.RunReconciliationRequest]) (*connect.Response[reconcilev1.RunReconciliationResponse], error) {
	if s.run != nil {
		findings, err := s.run(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeUnavailable, err)
		}
		return connect.NewResponse(&reconcilev1.RunReconciliationResponse{Findings: findings}), nil
	}
	result, err := s.list(ctx, "")
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&reconcilev1.RunReconciliationResponse{Findings: result}), nil
}

// SweepRunner builds the provider/local comparison and records each finding
// durably. It never destroys a provider resource.
func SweepRunner(database db, provider internalreconcile.Service, settle func(context.Context, string, float64) error) func(context.Context) ([]*reconcilev1.Finding, error) {
	return func(ctx context.Context) ([]*reconcilev1.Finding, error) {
		rows, err := database.QueryContext(ctx, `SELECT id,provider_instance_id,state,reservation_id,created_at FROM instances WHERE state <> 'destroyed'`)
		if err != nil {
			return nil, err
		}
		local := []internalreconcile.Local{}
		for rows.Next() {
			var item internalreconcile.Local
			var created string
			if err := rows.Scan(&item.InstanceID, &item.ProviderID, &item.State, &item.ReservationID, &created); err != nil {
				_ = rows.Close()
				return nil, err
			}
			item.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
			local = append(local, item)
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, err
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
		provider.Settle = func(ctx context.Context, item internalreconcile.Local) error {
			if item.ReservationID != "" {
				amount := clock.System{}.Now().Sub(item.CreatedAt).Minutes()
				if amount < 1 {
					amount = 1
				}
				if settle == nil {
					return fmt.Errorf("settlement is unavailable")
				}
				if err := settle(ctx, item.ReservationID, amount); err != nil {
					return err
				}
			}
			if item.InstanceID != "" {
				_, err := database.ExecContext(ctx, `UPDATE instances SET state='destroyed',destroyed_at=? WHERE id=?`, clock.System{}.Now().UTC().Format(time.RFC3339Nano), item.InstanceID)
				return err
			}
			return nil
		}
		observed, err := provider.Sweep(ctx, local)
		if err != nil {
			return nil, err
		}
		out := make([]*reconcilev1.Finding, 0, len(observed))
		for _, finding := range observed {
			kind := reconcilev1.FindingKind_FINDING_KIND_STATE_DIVERGENCE
			if finding.Kind == internalreconcile.ProviderOnly {
				kind = reconcilev1.FindingKind_FINDING_KIND_UNACCOUNTED_AT_PROVIDER
			}
			if finding.Kind == internalreconcile.LocalOnly {
				kind = reconcilev1.FindingKind_FINDING_KIND_DESTROYED_OUT_OF_BAND
			}
			now := clock.System{}.Now()
			item := &reconcilev1.Finding{Id: fmt.Sprintf("%d", now.UnixNano()), Kind: kind, Provider: provider.Provider.Name(), ProviderInstanceId: finding.ProviderID, Status: "open", Detail: finding.Detail, ObservedAt: timestamppb.New(now)}
			if _, err := database.ExecContext(ctx, `INSERT INTO reconcile_findings (id,observed_at,kind,provider,provider_instance_id,instance_id,status,detail_json) VALUES (?,?,?,?,?,?,?,?)`, item.Id, item.ObservedAt.AsTime().Format(time.RFC3339Nano), strings.ToLower(strings.TrimPrefix(kind.String(), "FINDING_KIND_")), item.Provider, item.ProviderInstanceId, item.InstanceId, item.Status, fmt.Sprintf(`{"detail":%q}`, finding.Detail)); err != nil {
				return nil, err
			}
			out = append(out, item)
		}
		return out, nil
	}
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
	response, err := s.GetFinding(ctx, connect.NewRequest(&reconcilev1.GetFindingRequest{Id: req.Msg.GetId()}))
	if err != nil {
		return nil, err
	}
	finding := response.Msg.GetFinding()
	if finding.GetKind() != reconcilev1.FindingKind_FINDING_KIND_UNACCOUNTED_AT_PROVIDER || finding.GetProviderInstanceId() == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("only provider-only findings can be quarantined for destruction"))
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE reconcile_findings SET status='quarantined' WHERE id=?`, req.Msg.GetId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if s.destroy == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("provider destruction is unavailable"))
	}
	if err := s.destroy(ctx, finding.GetProvider(), finding.GetProviderInstanceId()); err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("destroy quarantined provider instance: %w", err))
	}
	response.Msg.GetFinding().Status = "quarantined"
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
