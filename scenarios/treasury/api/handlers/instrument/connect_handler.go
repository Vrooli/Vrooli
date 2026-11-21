package instrument

import (
	"context"
	"errors"
	"net/http"

	domain "treasury/internal/instrument"
	"treasury/internal/module"
	"treasury/internal/operatorauth"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	instrumentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/instrument"
	instrumentconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/instrument/instrument_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service interface {
	Register(context.Context, domain.RegisterInput) (domain.Instrument, error)
}

type connectHandler struct {
	authorizer  operatorauth.Authorizer
	instruments Service
}

func NewConnectHandler(authorizer operatorauth.Authorizer, instruments Service) instrumentconnect.InstrumentAdminServiceHandler {
	return &connectHandler{authorizer: authorizer, instruments: instruments}
}

func (h *connectHandler) authorize(ctx context.Context, request interface{ Header() http.Header }) error {
	if h.authorizer == nil {
		return connect.NewError(connect.CodeFailedPrecondition, operatorauth.ErrUnavailable)
	}
	_, err := h.authorizer.Authorize(ctx, request.Header())
	switch {
	case err == nil:
		return nil
	case errors.Is(err, operatorauth.ErrRequired):
		return connect.NewError(connect.CodeUnauthenticated, err)
	case errors.Is(err, operatorauth.ErrDenied):
		return connect.NewError(connect.CodePermissionDenied, err)
	default:
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
}

func (h *connectHandler) RegisterInstrument(ctx context.Context, request *connect.Request[instrumentv1.RegisterInstrumentRequest]) (*connect.Response[instrumentv1.RegisterInstrumentResponse], error) {
	if err := h.authorize(ctx, request); err != nil {
		return nil, err
	}
	if h.instruments == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("instrument service unavailable"))
	}
	input := request.Msg.GetInstrument()
	if input == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("instrument is required"))
	}
	created, err := h.instruments.Register(ctx, domain.RegisterInput{ID: input.GetId(), MandateID: input.GetMandateId(), Rail: input.GetRail(), CredentialReference: input.GetCredentialReference(), Counterparty: input.GetCounterparty()})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	// CredentialReference is an internal Secrets Manager locator and is never
	// returned by the operator-facing API.
	msg := &instrumentv1.Instrument{Id: created.ID, BookId: created.BookID, MandateId: created.MandateID, Rail: created.Rail, CapMinor: created.CapMinor, Currency: created.Currency, Counterparty: created.Counterparty, ExpiresAt: timestamppb.New(created.ExpiresAt)}
	return connect.NewResponse(&instrumentv1.RegisterInstrumentResponse{Instrument: msg}), nil
}

func Module(authorizer operatorauth.Authorizer, instruments Service) module.Module {
	path, handler := instrumentconnect.NewInstrumentAdminServiceHandler(NewConnectHandler(authorizer, instruments))
	return module.Module{Name: "instrumentadmin", Mount: func(router *mux.Router) {
		connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
	}, Endpoints: Endpoints}
}
