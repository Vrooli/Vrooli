package book

import (
	"context"
	"errors"
	"net/http"

	domain "treasury/internal/book"
	"treasury/internal/module"
	"treasury/internal/operatorauth"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	bookv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/book"
	bookconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/book/book_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service interface {
	Create(context.Context, domain.CreateInput) (domain.Book, error)
	Get(context.Context, string) (domain.Book, error)
}

type connectHandler struct {
	authorizer operatorauth.Authorizer
	books      Service
}

func NewConnectHandler(authorizer operatorauth.Authorizer, books Service) bookconnect.BookAdminServiceHandler {
	return &connectHandler{authorizer: authorizer, books: books}
}

func (h *connectHandler) authorize(ctx context.Context, request interface{ Header() http.Header }) (operatorauth.Identity, error) {
	if h.authorizer == nil {
		return operatorauth.Identity{}, connect.NewError(connect.CodeFailedPrecondition, operatorauth.ErrUnavailable)
	}
	identity, err := h.authorizer.Authorize(ctx, request.Header())
	switch {
	case err == nil:
		return identity, nil
	case errors.Is(err, operatorauth.ErrRequired):
		return operatorauth.Identity{}, connect.NewError(connect.CodeUnauthenticated, err)
	case errors.Is(err, operatorauth.ErrDenied):
		return operatorauth.Identity{}, connect.NewError(connect.CodePermissionDenied, err)
	default:
		return operatorauth.Identity{}, connect.NewError(connect.CodeFailedPrecondition, err)
	}
}

func (h *connectHandler) CreateBook(ctx context.Context, request *connect.Request[bookv1.CreateBookRequest]) (*connect.Response[bookv1.CreateBookResponse], error) {
	identity, err := h.authorize(ctx, request)
	if err != nil {
		return nil, err
	}
	if h.books == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("book service unavailable"))
	}
	input := request.Msg.GetBook()
	if input == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("book is required"))
	}
	created, err := h.books.Create(ctx, domain.CreateInput{ID: input.GetId(), Name: input.GetName(), BeneficiaryIdentity: identity.Subject})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&bookv1.CreateBookResponse{Book: toProto(created)}), nil
}

func (h *connectHandler) GetBook(ctx context.Context, request *connect.Request[bookv1.GetBookRequest]) (*connect.Response[bookv1.GetBookResponse], error) {
	if _, err := h.authorize(ctx, request); err != nil {
		return nil, err
	}
	if h.books == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("book service unavailable"))
	}
	value, err := h.books.Get(ctx, request.Msg.GetBookId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&bookv1.GetBookResponse{Book: toProto(value)}), nil
}

func toProto(value domain.Book) *bookv1.Book {
	return &bookv1.Book{Id: value.ID, Name: value.Name, BeneficiaryIdentity: value.BeneficiaryIdentity, CreatedAt: timestamppb.New(value.CreatedAt)}
}

func Module(authorizer operatorauth.Authorizer, books Service) module.Module {
	path, handler := bookconnect.NewBookAdminServiceHandler(NewConnectHandler(authorizer, books))
	return module.Module{Name: "bookadmin", Mount: func(router *mux.Router) {
		connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
	}, Endpoints: Endpoints}
}
