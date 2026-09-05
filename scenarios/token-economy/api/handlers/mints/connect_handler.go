package mints

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	domain "token-economy/internal/mints"

	mintsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/mints"
)

type connectHandler struct {
	service domain.Service
	logger  *log.Logger
}

func NewConnectHandler(service domain.Service, logger *log.Logger) *connectHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &connectHandler{service: service, logger: logger}
}

func (h *connectHandler) CreateTokenType(ctx context.Context, req *connect.Request[mintsv1.CreateTokenTypeRequest]) (*connect.Response[mintsv1.CreateTokenTypeResponse], error) {
	created, err := h.service.Create(ctx, domain.CreateInput{
		Name: req.Msg.Name, Symbol: req.Msg.Symbol, Color: req.Msg.Color,
		SupplyPolicy: supplyPolicyFromProto(req.Msg.SupplyPolicy),
		CapAmount:    req.Msg.CapAmount, MinterSubject: req.Msg.MinterSubject,
	})
	if err != nil {
		return nil, h.mapError("CreateTokenType", err)
	}
	return connect.NewResponse(&mintsv1.CreateTokenTypeResponse{TokenType: tokenTypeToProto(created)}), nil
}

func (h *connectHandler) GetTokenType(ctx context.Context, req *connect.Request[mintsv1.GetTokenTypeRequest]) (*connect.Response[mintsv1.GetTokenTypeResponse], error) {
	tokenType, err := h.service.Get(ctx, req.Msg.Id)
	if err != nil {
		return nil, h.mapError("GetTokenType", err)
	}
	return connect.NewResponse(&mintsv1.GetTokenTypeResponse{TokenType: tokenTypeToProto(tokenType)}), nil
}

func (h *connectHandler) ListTokenTypes(ctx context.Context, req *connect.Request[mintsv1.ListTokenTypesRequest]) (*connect.Response[mintsv1.ListTokenTypesResponse], error) {
	items, err := h.service.List(ctx, req.Msg.IncludeRetired)
	if err != nil {
		return nil, h.mapError("ListTokenTypes", err)
	}
	response := &mintsv1.ListTokenTypesResponse{TokenTypes: make([]*mintsv1.TokenType, 0, len(items))}
	for _, item := range items {
		response.TokenTypes = append(response.TokenTypes, tokenTypeToProto(item))
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) RetireTokenType(ctx context.Context, req *connect.Request[mintsv1.RetireTokenTypeRequest]) (*connect.Response[mintsv1.RetireTokenTypeResponse], error) {
	tokenType, err := h.service.Retire(ctx, req.Msg.Id)
	if err != nil {
		return nil, h.mapError("RetireTokenType", err)
	}
	return connect.NewResponse(&mintsv1.RetireTokenTypeResponse{TokenType: tokenTypeToProto(tokenType)}), nil
}

func (h *connectHandler) MintSupply(ctx context.Context, req *connect.Request[mintsv1.MintSupplyRequest]) (*connect.Response[mintsv1.MintSupplyResponse], error) {
	tokenType, err := h.service.Mint(ctx, req.Msg.TokenTypeId, req.Msg.Amount)
	if err != nil {
		return nil, h.mapError("MintSupply", err)
	}
	return connect.NewResponse(&mintsv1.MintSupplyResponse{TokenType: tokenTypeToProto(tokenType)}), nil
}

func (h *connectHandler) mapError(operation string, err error) error {
	var invalid *domain.InvalidTokenTypeError
	var capExceeded *domain.SupplyCapExceededError
	switch {
	case errors.Is(err, domain.ErrTokenTypeNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, domain.ErrTokenTypeRetired):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.As(err, &invalid):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.As(err, &capExceeded):
		return connect.NewError(connect.CodeResourceExhausted, err)
	default:
		h.logger.Printf("mints.%s: %v", operation, err)
		return connect.NewError(connect.CodeInternal, errors.New("internal error"))
	}
}

func supplyPolicyFromProto(value mintsv1.SupplyPolicy) domain.SupplyPolicy {
	switch value {
	case mintsv1.SupplyPolicy_SUPPLY_POLICY_UNBOUNDED:
		return domain.SupplyPolicyUnbounded
	case mintsv1.SupplyPolicy_SUPPLY_POLICY_CAPPED:
		return domain.SupplyPolicyCapped
	case mintsv1.SupplyPolicy_SUPPLY_POLICY_FIXED:
		return domain.SupplyPolicyFixed
	default:
		return ""
	}
}

func supplyPolicyToProto(value domain.SupplyPolicy) mintsv1.SupplyPolicy {
	switch value {
	case domain.SupplyPolicyUnbounded:
		return mintsv1.SupplyPolicy_SUPPLY_POLICY_UNBOUNDED
	case domain.SupplyPolicyCapped:
		return mintsv1.SupplyPolicy_SUPPLY_POLICY_CAPPED
	case domain.SupplyPolicyFixed:
		return mintsv1.SupplyPolicy_SUPPLY_POLICY_FIXED
	default:
		return mintsv1.SupplyPolicy_SUPPLY_POLICY_UNSPECIFIED
	}
}

func tokenTypeToProto(value domain.TokenType) *mintsv1.TokenType {
	out := &mintsv1.TokenType{
		Id: value.ID, Name: value.Name, Symbol: value.Symbol, Color: value.Color,
		SupplyPolicy: supplyPolicyToProto(value.SupplyPolicy), CapAmount: value.CapAmount,
		MintedAmount: value.MintedAmount, Retired: value.Retired,
		Authority: &mintsv1.MinterAuthority{TokenTypeId: value.Authority.TokenTypeID, Subject: value.Authority.Subject},
		CreatedAt: timestamppb.New(value.CreatedAt),
	}
	if value.RetiredAt != nil {
		out.RetiredAt = timestamppb.New(*value.RetiredAt)
	}
	return out
}
