package meter

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"compute-manager/internal/module"
	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	meterv1 "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/meter"
	meterconnect "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/meter/meter_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type db interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
type service struct {
	meterconnect.UnimplementedMeterServiceHandler
	db db
}

func NewModule(database db) (module.Module, error) {
	if database == nil {
		return module.Module{}, errors.New("meter database is required")
	}
	s := &service{db: database}
	path, handler := meterconnect.NewMeterServiceHandler(s)
	return module.Module{Name: "meter", Mount: func(r *mux.Router) { r.PathPrefix(path).Handler(handler) }, Endpoints: Endpoints}, nil
}

func (s *service) Usage(ctx context.Context, req *connect.Request[meterv1.UsageRequest]) (*connect.Response[meterv1.UsageResponse], error) {
	rows, err := s.db.QueryContext(ctx, `SELECT instance_id,tenant,quantity,started_at,ended_at FROM usage_records WHERE (?='' OR instance_id=?) AND (?='' OR tenant=?) ORDER BY started_at`, req.Msg.GetInstanceId(), req.Msg.GetInstanceId(), req.Msg.GetTenant(), req.Msg.GetTenant())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer rows.Close()
	result := &meterv1.UsageResponse{Usage: []*meterv1.Usage{}}
	for rows.Next() {
		var instance, tenant, started, ended string
		var quantity int64
		if err := rows.Scan(&instance, &tenant, &quantity, &started, &ended); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		result.Usage = append(result.Usage, &meterv1.Usage{InstanceId: instance, Tenant: tenant, Quantity: quantity, StartedAt: stamp(started), EndedAt: stamp(ended)})
	}
	return connect.NewResponse(result), rows.Err()
}

func (s *service) Reservations(ctx context.Context, req *connect.Request[meterv1.ReservationsRequest]) (*connect.Response[meterv1.ReservationsResponse], error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,intent_id,instance_id,supersedes,meter_key,state,held_at,settled_at,quantity FROM reservations WHERE (?='' OR instance_id=?) AND (?='' OR state=?) ORDER BY held_at DESC`, req.Msg.GetInstanceId(), req.Msg.GetInstanceId(), req.Msg.GetState(), req.Msg.GetState())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer rows.Close()
	result := &meterv1.ReservationsResponse{Reservations: []*meterv1.Reservation{}}
	for rows.Next() {
		var r meterv1.Reservation
		var held, settled string
		if err := rows.Scan(&r.Id, &r.IntentId, &r.InstanceId, &r.Supersedes, &r.MeterKey, &r.State, &held, &settled, &r.Quantity); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		r.HeldAt = stamp(held)
		r.SettledAt = stamp(settled)
		result.Reservations = append(result.Reservations, &r)
	}
	return connect.NewResponse(result), rows.Err()
}

func (s *service) Ceiling(ctx context.Context, req *connect.Request[meterv1.CeilingRequest]) (*connect.Response[meterv1.CeilingResponse], error) {
	var used int64
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(quantity),0) FROM usage_records WHERE (?='' OR tenant=?)`, req.Msg.GetTenant(), req.Msg.GetTenant()).Scan(&used); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	// Usage is intentionally reported from usage_records; account limits are owned by LPBS.
	return connect.NewResponse(&meterv1.CeilingResponse{Used: used, Limit: -1}), nil
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
