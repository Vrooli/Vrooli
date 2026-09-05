package accounts

import (
	"context"
	"errors"
	"time"

	domain "persona/internal/accounts"

	"connectrpc.com/connect"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/accounts"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type connectHandler struct{ service domain.Service }

func NewConnectHandler(service domain.Service) *connectHandler {
	return &connectHandler{service: service}
}

func (h *connectHandler) LinkAccount(ctx context.Context, req *connect.Request[accountsv1.LinkAccountRequest]) (*connect.Response[accountsv1.LinkAccountResponse], error) {
	a, err := h.service.Link(ctx, domain.AccountInput{PersonaID: req.Msg.GetPersonaId(), Site: req.Msg.GetSite(), LoginSeam: req.Msg.GetLoginSeam(), RecoveryPath: req.Msg.GetRecoveryPath()})
	if err != nil {
		return nil, accountError(err)
	}
	return connect.NewResponse(&accountsv1.LinkAccountResponse{Account: toAccount(a)}), nil
}

func (h *connectHandler) ListAccounts(ctx context.Context, req *connect.Request[accountsv1.ListAccountsRequest]) (*connect.Response[accountsv1.ListAccountsResponse], error) {
	items, err := h.service.ListAccounts(ctx, req.Msg.GetPersonaId())
	if err != nil {
		return nil, accountError(err)
	}
	out := &accountsv1.ListAccountsResponse{Accounts: make([]*accountsv1.AccountLink, 0, len(items))}
	for _, item := range items {
		out.Accounts = append(out.Accounts, toAccount(item))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) AddAddress(ctx context.Context, req *connect.Request[accountsv1.AddAddressRequest]) (*connect.Response[accountsv1.AddAddressResponse], error) {
	if req.Msg.GetAddress() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, domain.ErrInvalidAddress)
	}
	a, err := h.service.AddAddress(ctx, domain.AddressInput{PersonaID: req.Msg.GetPersonaId(), Address: fromAddress(req.Msg.GetAddress())})
	if err != nil {
		return nil, accountError(err)
	}
	return connect.NewResponse(&accountsv1.AddAddressResponse{Address: toAddress(a)}), nil
}

func (h *connectHandler) ListAddresses(ctx context.Context, req *connect.Request[accountsv1.ListAddressesRequest]) (*connect.Response[accountsv1.ListAddressesResponse], error) {
	items, err := h.service.ListAddresses(ctx, req.Msg.GetPersonaId())
	if err != nil {
		return nil, accountError(err)
	}
	out := &accountsv1.ListAddressesResponse{Addresses: make([]*accountsv1.Address, 0, len(items))}
	for _, item := range items {
		out.Addresses = append(out.Addresses, toAddress(item))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) AddObligation(ctx context.Context, req *connect.Request[accountsv1.AddObligationRequest]) (*connect.Response[accountsv1.AddObligationResponse], error) {
	o, err := h.service.AddObligation(ctx, domain.ObligationInput{PersonaID: req.Msg.GetPersonaId(), AccountLinkID: req.Msg.GetAccountLinkId(), Description: req.Msg.GetDescription(), RenewalAt: timeFromProto(req.Msg.GetRenewalAt()), CancelPath: req.Msg.GetCancelPath()})
	if err != nil {
		return nil, accountError(err)
	}
	return connect.NewResponse(&accountsv1.AddObligationResponse{Obligation: toObligation(o)}), nil
}

func (h *connectHandler) ListObligations(ctx context.Context, req *connect.Request[accountsv1.ListObligationsRequest]) (*connect.Response[accountsv1.ListObligationsResponse], error) {
	items, err := h.service.ListObligations(ctx, req.Msg.GetPersonaId())
	if err != nil {
		return nil, accountError(err)
	}
	out := &accountsv1.ListObligationsResponse{Obligations: make([]*accountsv1.Obligation, 0, len(items))}
	for _, item := range items {
		out.Obligations = append(out.Obligations, toObligation(item))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) CancelObligation(ctx context.Context, req *connect.Request[accountsv1.CancelObligationRequest]) (*connect.Response[accountsv1.CancelObligationResponse], error) {
	o, err := h.service.CancelObligation(ctx, req.Msg.GetObligationId())
	if err != nil {
		return nil, accountError(err)
	}
	return connect.NewResponse(&accountsv1.CancelObligationResponse{Obligation: toObligation(o)}), nil
}

func (h *connectHandler) ReleaseAddress(ctx context.Context, req *connect.Request[accountsv1.ReleaseAddressRequest]) (*connect.Response[accountsv1.ReleaseAddressResponse], error) {
	a, err := h.service.ReleaseAddress(ctx, domain.AddressReleaseInput{PersonaID: req.Msg.GetPersonaId(), AddressID: req.Msg.GetAddressId(), TargetKind: req.Msg.GetTargetKind(), TargetID: req.Msg.GetTargetId()})
	if err != nil {
		return nil, accountError(err)
	}
	return connect.NewResponse(&accountsv1.ReleaseAddressResponse{Address: toAddress(a), TargetKind: req.Msg.GetTargetKind(), TargetId: req.Msg.GetTargetId()}), nil
}

func accountError(err error) error {
	code := connect.CodeInternal
	if errors.Is(err, domain.ErrMissingPersona) || errors.Is(err, domain.ErrMissingAddress) || errors.Is(err, domain.ErrMissingObligation) || errors.Is(err, domain.ErrInvalidAccount) || errors.Is(err, domain.ErrInvalidAddress) || errors.Is(err, domain.ErrInvalidObligation) || errors.Is(err, domain.ErrAddressReleaseTarget) {
		code = connect.CodeInvalidArgument
	}
	return connect.NewError(code, err)
}

func timeFromProto(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}

func toAccount(a domain.AccountLink) *accountsv1.AccountLink {
	return &accountsv1.AccountLink{Id: a.ID, PersonaId: a.PersonaID, Site: a.Site, LoginSeam: a.LoginSeam, RecoveryPath: a.RecoveryPath, CreatedAt: timestamppb.New(a.CreatedAt)}
}

func fromAddress(a *accountsv1.Address) domain.Address {
	return domain.Address{ID: a.GetId(), Label: a.GetLabel(), Line1: a.GetLine1(), Line2: a.GetLine2(), City: a.GetCity(), Region: a.GetRegion(), PostalCode: a.GetPostalCode(), Country: a.GetCountry()}
}

func toAddress(a domain.Address) *accountsv1.Address {
	return &accountsv1.Address{Id: a.ID, PersonaId: a.PersonaID, Label: a.Label, Line1: a.Line1, Line2: a.Line2, City: a.City, Region: a.Region, PostalCode: a.PostalCode, Country: a.Country, CreatedAt: timestamppb.New(a.CreatedAt)}
}

func toObligation(o domain.Obligation) *accountsv1.Obligation {
	return &accountsv1.Obligation{Id: o.ID, PersonaId: o.PersonaID, AccountLinkId: o.AccountLinkID, Description: o.Description, RenewalAt: timestamppb.New(o.RenewalAt), CancelPath: o.CancelPath, Cancelled: o.Cancelled, CreatedAt: timestamppb.New(o.CreatedAt)}
}
