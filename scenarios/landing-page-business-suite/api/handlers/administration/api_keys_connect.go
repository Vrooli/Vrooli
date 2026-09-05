package administration

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	admin "landing-page-business-suite-api/internal/administration"
)

// APIKeyConnectService is the transport-neutral administration capability used
// by the generated Connect edge. Raw credentials never leave this boundary.
type APIKeyConnectService interface {
	List(context.Context) ([]admin.APIKey, error)
	Store(context.Context, string, string) (*admin.APIKey, error)
	Delete(context.Context, string) error
	Test(context.Context, string) (bool, string, error)
	SetActive(context.Context, string, bool) error
}

type APIKeyConnectHandler struct{ service APIKeyConnectService }

func NewAPIKeyConnectHandler(service APIKeyConnectService) *APIKeyConnectHandler {
	return &APIKeyConnectHandler{service: service}
}

func (h *APIKeyConnectHandler) ListAPIKeys(ctx context.Context, _ *connect.Request[lpbsv1.ListAPIKeysRequest]) (*connect.Response[lpbsv1.ListAPIKeysResponse], error) {
	keys, err := h.service.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("list API keys"))
	}
	response := &lpbsv1.ListAPIKeysResponse{Keys: make([]*lpbsv1.APIKey, 0, len(keys))}
	for _, key := range keys {
		response.Keys = append(response.Keys, apiKeyProto(key))
	}
	return connect.NewResponse(response), nil
}

func (h *APIKeyConnectHandler) CreateAPIKey(ctx context.Context, request *connect.Request[lpbsv1.CreateAPIKeyRequest]) (*connect.Response[lpbsv1.CreateAPIKeyResponse], error) {
	if request == nil || request.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("API key request is required"))
	}
	key, err := h.service.Store(ctx, request.Msg.GetProvider(), request.Msg.GetKey())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid API key request"))
	}
	return connect.NewResponse(&lpbsv1.CreateAPIKeyResponse{Key: apiKeyProto(*key)}), nil
}

func (h *APIKeyConnectHandler) DeleteAPIKey(ctx context.Context, request *connect.Request[lpbsv1.DeleteAPIKeyRequest]) (*connect.Response[lpbsv1.DeleteAPIKeyResponse], error) {
	if request == nil || request.Msg == nil || strings.TrimSpace(request.Msg.GetProvider()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("provider is required"))
	}
	if err := h.service.Delete(ctx, request.Msg.GetProvider()); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("API key not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, errors.New("delete API key"))
	}
	return connect.NewResponse(&lpbsv1.DeleteAPIKeyResponse{}), nil
}

func (h *APIKeyConnectHandler) TestAPIKey(ctx context.Context, request *connect.Request[lpbsv1.TestAPIKeyRequest]) (*connect.Response[lpbsv1.TestAPIKeyResponse], error) {
	if request == nil || request.Msg == nil || strings.TrimSpace(request.Msg.GetProvider()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("provider is required"))
	}
	success, message, err := h.service.Test(ctx, request.Msg.GetProvider())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("test API key"))
	}
	return connect.NewResponse(&lpbsv1.TestAPIKeyResponse{Success: success, Message: message, Provider: request.Msg.GetProvider()}), nil
}

func (h *APIKeyConnectHandler) SetAPIKeyActive(ctx context.Context, request *connect.Request[lpbsv1.SetAPIKeyActiveRequest]) (*connect.Response[lpbsv1.SetAPIKeyActiveResponse], error) {
	if request == nil || request.Msg == nil || strings.TrimSpace(request.Msg.GetProvider()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("provider is required"))
	}
	if err := h.service.SetActive(ctx, request.Msg.GetProvider(), request.Msg.GetActive()); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("API key not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, errors.New("update API key"))
	}
	return connect.NewResponse(&lpbsv1.SetAPIKeyActiveResponse{}), nil
}

func apiKeyProto(key admin.APIKey) *lpbsv1.APIKey {
	result := &lpbsv1.APIKey{Id: key.ID, Provider: key.Provider, KeyHint: key.KeyHint, IsActive: key.IsActive, CreatedAt: key.CreatedAt.Format(time.RFC3339), UpdatedAt: key.UpdatedAt.Format(time.RFC3339)}
	if key.LastVerifiedAt != nil {
		result.LastVerifiedAt = key.LastVerifiedAt.Format(time.RFC3339)
	}
	return result
}

func RegisterAPIKeyConnectRoutes(router *mux.Router, service APIKeyConnectService, requireAdmin func(http.HandlerFunc) http.HandlerFunc) {
	_, generated := lpbsconnect.NewAdministrationServiceHandler(NewAPIKeyConnectHandler(service))
	router.Handle(lpbsconnect.AdministrationServiceListAPIKeysProcedure, requireAdmin(generated.ServeHTTP)).Methods(http.MethodPost)
	router.Handle(lpbsconnect.AdministrationServiceCreateAPIKeyProcedure, requireAdmin(generated.ServeHTTP)).Methods(http.MethodPost)
	router.Handle(lpbsconnect.AdministrationServiceDeleteAPIKeyProcedure, requireAdmin(generated.ServeHTTP)).Methods(http.MethodPost)
	router.Handle(lpbsconnect.AdministrationServiceTestAPIKeyProcedure, requireAdmin(generated.ServeHTTP)).Methods(http.MethodPost)
	router.Handle(lpbsconnect.AdministrationServiceSetAPIKeyActiveProcedure, requireAdmin(generated.ServeHTTP)).Methods(http.MethodPost)
}

var _ lpbsconnect.AdministrationServiceHandler = (*APIKeyConnectHandler)(nil)
