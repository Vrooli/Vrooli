// Package chatauth binds account identity before chat and message operations.
package chatauth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/owneridentity"
	"github.com/vrooli/api-core/schedule"
	"portal/internal/chat"
)

type interceptor struct {
	validator owneridentity.Validator
	now       func() time.Time
}

func New(validator owneridentity.Validator, now func() time.Time) connect.Interceptor {
	if now == nil {
		now = time.Now
	}
	return &interceptor{validator: validator, now: now}
}

func FromEnvironment(clk schedule.Clock) connect.Interceptor {
	if clk == nil {
		clk = schedule.System()
	}
	return New(owneridentity.NewClient(owneridentity.Config{Now: clk.Now}), clk.Now)
}

func (i *interceptor) authorize(ctx context.Context, header http.Header) (context.Context, context.CancelFunc, error) {
	noop := func() {}
	values := header.Values("Authorization")
	if len(values) == 0 {
		return ctx, noop, nil
	}
	denied := connect.NewError(connect.CodeUnauthenticated, errors.New("account authentication required"))
	if len(values) != 1 || !strings.HasPrefix(values[0], "Bearer ") || len(values[0]) > 16384 || len(values[0]) <= 7 || i.validator == nil {
		return nil, noop, denied
	}
	token := strings.TrimPrefix(values[0], "Bearer ")
	if strings.ContainsAny(token, " \t\r\n") {
		return nil, noop, denied
	}
	identity, err := i.validator.Validate(ctx, token)
	if err != nil || !identity.ExpiresAt.After(i.now()) {
		return nil, noop, denied
	}
	owned, err := chat.WithOwner(ctx, identity.Subject)
	if err != nil {
		return nil, noop, denied
	}
	owned, cancel := context.WithDeadline(owned, identity.ExpiresAt)
	return owned, cancel, nil
}

func (i *interceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		ctx, cancel, err := i.authorize(ctx, req.Header())
		if err != nil {
			return nil, err
		}
		defer cancel()
		response, err := next(ctx, req)
		if response != nil {
			response.Header().Set("Cache-Control", "no-store")
		}
		return response, err
	}
}

func (i *interceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		conn.ResponseHeader().Set("Cache-Control", "no-store")
		ctx, cancel, err := i.authorize(ctx, conn.RequestHeader())
		if err != nil {
			return err
		}
		defer cancel()
		return next(ctx, conn)
	}
}

func (i *interceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}
