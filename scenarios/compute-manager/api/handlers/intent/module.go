package intent

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
	intentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/intent"
	intentconnect "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/intent/intent_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type db interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type service struct {
	intentconnect.UnimplementedIntentServiceHandler
	db db
}

func NewModule(database db) (module.Module, error) {
	if database == nil {
		return module.Module{}, errors.New("intent database is required")
	}
	s := &service{db: database}
	path, handler := intentconnect.NewIntentServiceHandler(s)
	return module.Module{Name: "intent", Mount: func(r *mux.Router) { r.PathPrefix(path).Handler(handler) }, Endpoints: Endpoints}, nil
}

func (s *service) ListIntents(ctx context.Context, req *connect.Request[intentv1.ListIntentsRequest]) (*connect.Response[intentv1.ListIntentsResponse], error) {
	query := `SELECT id,idempotency_key,requested_by,provider,reservation_id,instance_id,state,created_at,resolved_at FROM instance_intents WHERE (?='' OR state=?) AND (?='' OR provider=?) ORDER BY created_at DESC`
	rows, err := s.db.QueryContext(ctx, query, req.Msg.GetState(), req.Msg.GetState(), req.Msg.GetProvider(), req.Msg.GetProvider())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer rows.Close()
	result := &intentv1.ListIntentsResponse{Intents: []*intentv1.Intent{}}
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		result.Intents = append(result.Intents, item)
	}
	if err := rows.Err(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(result), nil
}

func (s *service) GetIntent(ctx context.Context, req *connect.Request[intentv1.GetIntentRequest]) (*connect.Response[intentv1.GetIntentResponse], error) {
	row := s.db.QueryRowContext(ctx, `SELECT id,idempotency_key,requested_by,provider,reservation_id,instance_id,state,created_at,resolved_at FROM instance_intents WHERE id=? OR idempotency_key=? LIMIT 1`, req.Msg.GetIdOrKey(), req.Msg.GetIdOrKey())
	item, err := scan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("intent %q not found", req.Msg.GetIdOrKey()))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&intentv1.GetIntentResponse{Intent: item}), nil
}

type scanner interface{ Scan(...any) error }

func scan(row scanner) (*intentv1.Intent, error) {
	var id, key, requestedBy, provider, reservationID, instanceID, state, created, resolved string
	if err := row.Scan(&id, &key, &requestedBy, &provider, &reservationID, &instanceID, &state, &created, &resolved); err != nil {
		return nil, err
	}
	item := &intentv1.Intent{Id: id, IdempotencyKey: key, RequestedBy: requestedBy, Provider: provider, ReservationId: reservationID, InstanceId: instanceID, State: intentv1.IntentState(intentv1.IntentState_value["INTENT_STATE_"+strings.ToUpper(state)])}
	if t, err := time.Parse(time.RFC3339Nano, created); err == nil {
		item.CreatedAt = timestamppb.New(t)
	}
	if resolved != "" {
		if t, err := time.Parse(time.RFC3339Nano, resolved); err == nil {
			item.ResolvedAt = timestamppb.New(t)
		}
	}
	return item, nil
}
