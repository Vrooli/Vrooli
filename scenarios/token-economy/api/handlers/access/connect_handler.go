// Package access owns the two authenticated public RPC surfaces and delegates
// implemented operations to their domain transport adapters.
package access

import (
	"context"
	"errors"

	internalaccess "token-economy/internal/access"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/provenance"

	accessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/access"
	accessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/access/accessv1connect"
	earningv1 "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/earning"
	earningconnect "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/earning/earningv1connect"
	grantsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/grants"
	mintsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/mints"
)

type MintsDelegate interface {
	CreateTokenType(context.Context, *connect.Request[mintsv1.CreateTokenTypeRequest]) (*connect.Response[mintsv1.CreateTokenTypeResponse], error)
	GetTokenType(context.Context, *connect.Request[mintsv1.GetTokenTypeRequest]) (*connect.Response[mintsv1.GetTokenTypeResponse], error)
	ListTokenTypes(context.Context, *connect.Request[mintsv1.ListTokenTypesRequest]) (*connect.Response[mintsv1.ListTokenTypesResponse], error)
	RetireTokenType(context.Context, *connect.Request[mintsv1.RetireTokenTypeRequest]) (*connect.Response[mintsv1.RetireTokenTypeResponse], error)
	MintSupply(context.Context, *connect.Request[mintsv1.MintSupplyRequest]) (*connect.Response[mintsv1.MintSupplyResponse], error)
}

type GrantsDelegate interface {
	CreateGrant(context.Context, *connect.Request[grantsv1.CreateGrantRequest]) (*connect.Response[grantsv1.CreateGrantResponse], error)
	GetGrant(context.Context, *connect.Request[grantsv1.GetGrantRequest]) (*connect.Response[grantsv1.GetGrantResponse], error)
	ListGrants(context.Context, *connect.Request[accessv1.ListGrantsRequest]) (*connect.Response[accessv1.ListGrantsResponse], error)
	RevokeGrant(context.Context, *connect.Request[accessv1.RevokeGrantRequest]) (*connect.Response[accessv1.RevokeGrantResponse], error)
}

type HolderDelegate interface {
	CreateHolder(context.Context, *connect.Request[accessv1.CreateHolderRequest]) (*connect.Response[accessv1.CreateHolderResponse], error)
	GetHolder(context.Context, *connect.Request[accessv1.GetHolderRequest]) (*connect.Response[accessv1.GetHolderResponse], error)
	ListHolders(context.Context, *connect.Request[accessv1.ListHoldersRequest]) (*connect.Response[accessv1.ListHoldersResponse], error)
	ViewEconomy(context.Context, string, *connect.Request[accessv1.ViewEconomyRequest]) (*connect.Response[accessv1.ViewEconomyResponse], error)
}

type EarningDelegate interface {
	SubmitEarning(context.Context, string, *connect.Request[earningv1.SubmitEarningRequest]) (*connect.Response[earningv1.SubmitEarningResponse], error)
	ListEarnings(context.Context, *connect.Request[earningv1.ListEarningsRequest]) (*connect.Response[earningv1.ListEarningsResponse], error)
}

type CatalogDelegate interface {
	CreateCatalogEntry(context.Context, *connect.Request[accessv1.CreateCatalogEntryRequest]) (*connect.Response[accessv1.CreateCatalogEntryResponse], error)
	UpdateCatalogEntry(context.Context, *connect.Request[accessv1.UpdateCatalogEntryRequest]) (*connect.Response[accessv1.UpdateCatalogEntryResponse], error)
	GetCatalogEntry(context.Context, *connect.Request[accessv1.GetCatalogEntryRequest]) (*connect.Response[accessv1.GetCatalogEntryResponse], error)
	ListCatalogEntries(context.Context, *connect.Request[accessv1.ListCatalogEntriesRequest]) (*connect.Response[accessv1.ListCatalogEntriesResponse], error)
	RetireCatalogEntry(context.Context, *connect.Request[accessv1.RetireCatalogEntryRequest]) (*connect.Response[accessv1.RetireCatalogEntryResponse], error)
	BrowseCatalog(context.Context, *connect.Request[accessv1.BrowseCatalogRequest]) (*connect.Response[accessv1.BrowseCatalogResponse], error)
	RequireAvailable(context.Context, string) error
}

type RedemptionDelegate interface {
	RequestRedemption(context.Context, string, *connect.Request[accessv1.RequestRedemptionRequest]) (*connect.Response[accessv1.RequestRedemptionResponse], error)
	ListPendingRedemptions(context.Context, *connect.Request[accessv1.ListPendingRedemptionsRequest]) (*connect.Response[accessv1.ListPendingRedemptionsResponse], error)
	ApproveRedemption(context.Context, string, *connect.Request[accessv1.ApproveRedemptionRequest]) (*connect.Response[accessv1.ApproveRedemptionResponse], error)
	DenyRedemption(context.Context, string, *connect.Request[accessv1.DenyRedemptionRequest]) (*connect.Response[accessv1.DenyRedemptionResponse], error)
}

type holderRedemptionDelegate interface {
	ListHolderRedemptions(context.Context, string) ([]*accessv1.Redemption, error)
}

type JournalDelegate interface {
	ListJournalEvents(context.Context, *connect.Request[accessv1.ListJournalEventsRequest]) (*connect.Response[accessv1.ListJournalEventsResponse], error)
	ShowBalance(context.Context, *connect.Request[accessv1.ShowBalanceRequest]) (*connect.Response[accessv1.ShowBalanceResponse], error)
	ExportJournal(context.Context, *connect.Request[accessv1.ExportJournalRequest]) (*connect.Response[accessv1.ExportJournalResponse], error)
	ReverseEvent(context.Context, string, *connect.Request[accessv1.ReverseEventRequest]) (*connect.Response[accessv1.ReverseEventResponse], error)
}

type connectHandler struct {
	accessconnect.UnimplementedMinterServiceHandler
	accessconnect.UnimplementedHolderServiceHandler
	earningconnect.UnimplementedEarningServiceHandler
	mints      MintsDelegate
	grants     GrantsDelegate
	holder     HolderDelegate
	earning    EarningDelegate
	catalog    CatalogDelegate
	redemption RedemptionDelegate
	journal    JournalDelegate
}

func NewConnectHandler(mints MintsDelegate, grants GrantsDelegate, holder HolderDelegate, earning EarningDelegate, catalog CatalogDelegate, redemption RedemptionDelegate, journal JournalDelegate) *connectHandler {
	return &connectHandler{mints: mints, grants: grants, holder: holder, earning: earning, catalog: catalog, redemption: redemption, journal: journal}
}

func (h *connectHandler) CreateTokenType(ctx context.Context, req *connect.Request[accessv1.CreateTokenTypeRequest]) (*connect.Response[accessv1.CreateTokenTypeResponse], error) {
	if h.mints == nil {
		return nil, unavailableDelegate("mints")
	}
	response, err := h.mints.CreateTokenType(ctx, connect.NewRequest(&mintsv1.CreateTokenTypeRequest{
		Name: req.Msg.Name, Symbol: req.Msg.Symbol, Color: req.Msg.Color,
		SupplyPolicy: mintSupplyPolicy(req.Msg.SupplyPolicy), CapAmount: req.Msg.CapAmount,
		MinterSubject: req.Msg.MinterSubject,
	}))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&accessv1.CreateTokenTypeResponse{TokenType: accessTokenType(response.Msg.TokenType)}), nil
}

func (h *connectHandler) GetTokenType(ctx context.Context, req *connect.Request[accessv1.GetTokenTypeRequest]) (*connect.Response[accessv1.GetTokenTypeResponse], error) {
	if h.mints == nil {
		return nil, unavailableDelegate("mints")
	}
	response, err := h.mints.GetTokenType(ctx, connect.NewRequest(&mintsv1.GetTokenTypeRequest{Id: req.Msg.Id}))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&accessv1.GetTokenTypeResponse{TokenType: accessTokenType(response.Msg.TokenType)}), nil
}

func (h *connectHandler) ListTokenTypes(ctx context.Context, req *connect.Request[accessv1.ListTokenTypesRequest]) (*connect.Response[accessv1.ListTokenTypesResponse], error) {
	if h.mints == nil {
		return nil, unavailableDelegate("mints")
	}
	response, err := h.mints.ListTokenTypes(ctx, connect.NewRequest(&mintsv1.ListTokenTypesRequest{IncludeRetired: req.Msg.IncludeRetired}))
	if err != nil {
		return nil, err
	}
	out := &accessv1.ListTokenTypesResponse{TokenTypes: make([]*accessv1.TokenType, 0, len(response.Msg.TokenTypes))}
	for _, tokenType := range response.Msg.TokenTypes {
		out.TokenTypes = append(out.TokenTypes, accessTokenType(tokenType))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) RetireTokenType(ctx context.Context, req *connect.Request[accessv1.RetireTokenTypeRequest]) (*connect.Response[accessv1.RetireTokenTypeResponse], error) {
	if h.mints == nil {
		return nil, unavailableDelegate("mints")
	}
	response, err := h.mints.RetireTokenType(ctx, connect.NewRequest(&mintsv1.RetireTokenTypeRequest{Id: req.Msg.Id}))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&accessv1.RetireTokenTypeResponse{TokenType: accessTokenType(response.Msg.TokenType)}), nil
}

func (h *connectHandler) MintSupply(ctx context.Context, req *connect.Request[accessv1.MintSupplyRequest]) (*connect.Response[accessv1.MintSupplyResponse], error) {
	if h.mints == nil {
		return nil, unavailableDelegate("mints")
	}
	response, err := h.mints.MintSupply(ctx, connect.NewRequest(&mintsv1.MintSupplyRequest{TokenTypeId: req.Msg.TokenTypeId, Amount: req.Msg.Amount}))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&accessv1.MintSupplyResponse{TokenType: accessTokenType(response.Msg.TokenType)}), nil
}

func (h *connectHandler) CreateGrant(ctx context.Context, req *connect.Request[accessv1.CreateGrantRequest]) (*connect.Response[accessv1.CreateGrantResponse], error) {
	if h.grants == nil {
		return nil, unavailableDelegate("grants")
	}
	rules := make([]*grantsv1.GrantRule, 0, len(req.Msg.Rules))
	for _, rule := range req.Msg.Rules {
		if rule != nil {
			rules = append(rules, &grantsv1.GrantRule{Id: rule.Id, Condition: grantRuleCondition(rule.Condition), Operands: rule.Operands, AmountLimit: rule.AmountLimit})
		}
	}
	ctx, _, actorErr := authenticatedActorContext(ctx)
	if actorErr != nil {
		return nil, actorErr
	}
	response, err := h.grants.CreateGrant(ctx, connect.NewRequest(&grantsv1.CreateGrantRequest{
		TokenTypeId: req.Msg.TokenTypeId, GrantSourceId: req.Msg.GrantSourceId,
		Authorizer: req.Msg.Authorizer, HolderId: req.Msg.HolderId, AmountMinor: req.Msg.AmountMinor,
		AllowedCatalogScopes: req.Msg.AllowedCatalogScopes, DeniedCatalogScopes: req.Msg.DeniedCatalogScopes,
		ExpiresAt: req.Msg.ExpiresAt, IdempotencyKey: req.Msg.IdempotencyKey,
		RequiredEvidence: req.Msg.RequiredEvidence, Rules: rules,
	}))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&accessv1.CreateGrantResponse{Grant: accessGrant(response.Msg.Grant)}), nil
}

func (h *connectHandler) GetGrant(ctx context.Context, req *connect.Request[accessv1.GetGrantRequest]) (*connect.Response[accessv1.GetGrantResponse], error) {
	if h.grants == nil {
		return nil, unavailableDelegate("grants")
	}
	response, err := h.grants.GetGrant(ctx, connect.NewRequest(&grantsv1.GetGrantRequest{Id: req.Msg.Id}))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&accessv1.GetGrantResponse{Grant: accessGrant(response.Msg.Grant)}), nil
}

func (h *connectHandler) ListGrants(ctx context.Context, req *connect.Request[accessv1.ListGrantsRequest]) (*connect.Response[accessv1.ListGrantsResponse], error) {
	if h.grants == nil {
		return nil, unavailableDelegate("grants")
	}
	return h.grants.ListGrants(ctx, req)
}

func (h *connectHandler) RevokeGrant(ctx context.Context, req *connect.Request[accessv1.RevokeGrantRequest]) (*connect.Response[accessv1.RevokeGrantResponse], error) {
	if h.grants == nil {
		return nil, unavailableDelegate("grants")
	}
	return h.grants.RevokeGrant(ctx, req)
}

func (h *connectHandler) CreateHolder(ctx context.Context, req *connect.Request[accessv1.CreateHolderRequest]) (*connect.Response[accessv1.CreateHolderResponse], error) {
	if h.holder == nil {
		return nil, unavailableDelegate("holders")
	}
	return h.holder.CreateHolder(ctx, req)
}

func (h *connectHandler) GetHolder(ctx context.Context, req *connect.Request[accessv1.GetHolderRequest]) (*connect.Response[accessv1.GetHolderResponse], error) {
	if h.holder == nil {
		return nil, unavailableDelegate("holders")
	}
	return h.holder.GetHolder(ctx, req)
}

func (h *connectHandler) ListHolders(ctx context.Context, req *connect.Request[accessv1.ListHoldersRequest]) (*connect.Response[accessv1.ListHoldersResponse], error) {
	if h.holder == nil {
		return nil, unavailableDelegate("holders")
	}
	return h.holder.ListHolders(ctx, req)
}

func (h *connectHandler) CreateCatalogEntry(ctx context.Context, req *connect.Request[accessv1.CreateCatalogEntryRequest]) (*connect.Response[accessv1.CreateCatalogEntryResponse], error) {
	if h.catalog == nil {
		return nil, unavailableDelegate("catalog")
	}
	return h.catalog.CreateCatalogEntry(ctx, req)
}

func (h *connectHandler) UpdateCatalogEntry(ctx context.Context, req *connect.Request[accessv1.UpdateCatalogEntryRequest]) (*connect.Response[accessv1.UpdateCatalogEntryResponse], error) {
	if h.catalog == nil {
		return nil, unavailableDelegate("catalog")
	}
	return h.catalog.UpdateCatalogEntry(ctx, req)
}

func (h *connectHandler) GetCatalogEntry(ctx context.Context, req *connect.Request[accessv1.GetCatalogEntryRequest]) (*connect.Response[accessv1.GetCatalogEntryResponse], error) {
	if h.catalog == nil {
		return nil, unavailableDelegate("catalog")
	}
	return h.catalog.GetCatalogEntry(ctx, req)
}

func (h *connectHandler) ListCatalogEntries(ctx context.Context, req *connect.Request[accessv1.ListCatalogEntriesRequest]) (*connect.Response[accessv1.ListCatalogEntriesResponse], error) {
	if h.catalog == nil {
		return nil, unavailableDelegate("catalog")
	}
	return h.catalog.ListCatalogEntries(ctx, req)
}

func (h *connectHandler) RetireCatalogEntry(ctx context.Context, req *connect.Request[accessv1.RetireCatalogEntryRequest]) (*connect.Response[accessv1.RetireCatalogEntryResponse], error) {
	if h.catalog == nil {
		return nil, unavailableDelegate("catalog")
	}
	return h.catalog.RetireCatalogEntry(ctx, req)
}

func (h *connectHandler) ListPendingRedemptions(ctx context.Context, req *connect.Request[accessv1.ListPendingRedemptionsRequest]) (*connect.Response[accessv1.ListPendingRedemptionsResponse], error) {
	if h.redemption == nil {
		return nil, unavailableDelegate("redemption")
	}
	return h.redemption.ListPendingRedemptions(ctx, req)
}

func (h *connectHandler) ApproveRedemption(ctx context.Context, req *connect.Request[accessv1.ApproveRedemptionRequest]) (*connect.Response[accessv1.ApproveRedemptionResponse], error) {
	if h.redemption == nil {
		return nil, unavailableDelegate("redemption")
	}
	identity, ok := internalaccess.IdentityFromContext(ctx)
	if !ok || identity.Subject == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, internalaccess.ErrUnauthenticated)
	}
	return h.redemption.ApproveRedemption(withActorSubject(ctx, identity.Subject), identity.Subject, req)
}

func (h *connectHandler) DenyRedemption(ctx context.Context, req *connect.Request[accessv1.DenyRedemptionRequest]) (*connect.Response[accessv1.DenyRedemptionResponse], error) {
	if h.redemption == nil {
		return nil, unavailableDelegate("redemption")
	}
	identity, ok := internalaccess.IdentityFromContext(ctx)
	if !ok || identity.Subject == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, internalaccess.ErrUnauthenticated)
	}
	return h.redemption.DenyRedemption(withActorSubject(ctx, identity.Subject), identity.Subject, req)
}

func (h *connectHandler) ViewEconomy(ctx context.Context, req *connect.Request[accessv1.ViewEconomyRequest]) (*connect.Response[accessv1.ViewEconomyResponse], error) {
	if h.holder == nil {
		return nil, unavailableDelegate("holders")
	}
	identity, ok := internalaccess.IdentityFromContext(ctx)
	if !ok || identity.Subject == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, internalaccess.ErrUnauthenticated)
	}
	response, err := h.holder.ViewEconomy(ctx, identity.Subject, req)
	if err != nil {
		return nil, err
	}
	if redemption, ok := h.redemption.(holderRedemptionDelegate); ok {
		response.Msg.Redemptions, err = redemption.ListHolderRedemptions(ctx, identity.Subject)
		if err != nil {
			return nil, err
		}
	}
	return response, nil
}

func (h *connectHandler) BrowseCatalog(ctx context.Context, req *connect.Request[accessv1.BrowseCatalogRequest]) (*connect.Response[accessv1.BrowseCatalogResponse], error) {
	if h.catalog == nil {
		return nil, unavailableDelegate("catalog")
	}
	return h.catalog.BrowseCatalog(ctx, req)
}

func (h *connectHandler) RequestRedemption(ctx context.Context, req *connect.Request[accessv1.RequestRedemptionRequest]) (*connect.Response[accessv1.RequestRedemptionResponse], error) {
	if req.Msg.Redemption == nil || req.Msg.Redemption.CatalogEntryId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("catalog entry id is required"))
	}
	if h.catalog != nil {
		if err := h.catalog.RequireAvailable(ctx, req.Msg.Redemption.CatalogEntryId); err != nil {
			return nil, err
		}
	}
	if h.redemption == nil {
		return nil, unavailableDelegate("redemption")
	}
	identity, ok := internalaccess.IdentityFromContext(ctx)
	if !ok || identity.Subject == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, internalaccess.ErrUnauthenticated)
	}
	return h.redemption.RequestRedemption(withActorSubject(ctx, identity.Subject), identity.Subject, req)
}

func (h *connectHandler) SubmitEarning(ctx context.Context, req *connect.Request[earningv1.SubmitEarningRequest]) (*connect.Response[earningv1.SubmitEarningResponse], error) {
	if h.earning == nil {
		return nil, unavailableDelegate("earning")
	}
	identity, ok := internalaccess.IdentityFromContext(ctx)
	if !ok || identity.Subject == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, internalaccess.ErrUnauthenticated)
	}
	return h.earning.SubmitEarning(withActorSubject(ctx, identity.Subject), identity.Subject, req)
}

func (h *connectHandler) ListEarnings(ctx context.Context, req *connect.Request[earningv1.ListEarningsRequest]) (*connect.Response[earningv1.ListEarningsResponse], error) {
	if h.earning == nil {
		return nil, unavailableDelegate("earning")
	}
	return h.earning.ListEarnings(ctx, req)
}

func (h *connectHandler) ReverseEvent(ctx context.Context, req *connect.Request[accessv1.ReverseEventRequest]) (*connect.Response[accessv1.ReverseEventResponse], error) {
	if h.journal == nil {
		return nil, unavailableDelegate("journal")
	}
	actorCtx, identity, err := authenticatedActorContext(ctx)
	if err != nil {
		return nil, err
	}
	return h.journal.ReverseEvent(actorCtx, identity.Subject, req)
}

func (h *connectHandler) ListJournalEvents(ctx context.Context, req *connect.Request[accessv1.ListJournalEventsRequest]) (*connect.Response[accessv1.ListJournalEventsResponse], error) {
	if h.journal == nil {
		return nil, unavailableDelegate("journal")
	}
	return h.journal.ListJournalEvents(ctx, req)
}

func (h *connectHandler) ShowBalance(ctx context.Context, req *connect.Request[accessv1.ShowBalanceRequest]) (*connect.Response[accessv1.ShowBalanceResponse], error) {
	if h.journal == nil {
		return nil, unavailableDelegate("journal")
	}
	return h.journal.ShowBalance(ctx, req)
}

func (h *connectHandler) ExportJournal(ctx context.Context, req *connect.Request[accessv1.ExportJournalRequest]) (*connect.Response[accessv1.ExportJournalResponse], error) {
	if h.journal == nil {
		return nil, unavailableDelegate("journal")
	}
	return h.journal.ExportJournal(ctx, req)
}

func authenticatedActorContext(ctx context.Context) (context.Context, internalaccess.Identity, error) {
	identity, ok := internalaccess.IdentityFromContext(ctx)
	if !ok || identity.Subject == "" {
		return ctx, internalaccess.Identity{}, connect.NewError(connect.CodeUnauthenticated, internalaccess.ErrUnauthenticated)
	}
	return withActorSubject(ctx, identity.Subject), identity, nil
}

func withActorSubject(ctx context.Context, subject string) context.Context {
	p := provenance.FromContext(ctx)
	if !p.IsVerifiedAgent() {
		p.Subject = subject
	}
	return provenance.NewContext(ctx, p)
}

func unavailableDelegate(name string) error {
	return connect.NewError(connect.CodeUnavailable, errors.New(name+" service unavailable"))
}

func mintSupplyPolicy(value accessv1.SupplyPolicy) mintsv1.SupplyPolicy {
	return mintsv1.SupplyPolicy(value)
}

func accessTokenType(value *mintsv1.TokenType) *accessv1.TokenType {
	if value == nil {
		return nil
	}
	out := &accessv1.TokenType{
		Id: value.Id, Name: value.Name, Symbol: value.Symbol, Color: value.Color,
		SupplyPolicy: accessv1.SupplyPolicy(value.SupplyPolicy), CapAmount: value.CapAmount,
		MintedAmount: value.MintedAmount, Retired: value.Retired,
		CreatedAt: value.CreatedAt, RetiredAt: value.RetiredAt,
	}
	if value.Authority != nil {
		out.Authority = &accessv1.MinterAuthority{TokenTypeId: value.Authority.TokenTypeId, Subject: value.Authority.Subject}
	}
	return out
}

func grantRuleCondition(value accessv1.RuleCondition) grantsv1.RuleCondition {
	return grantsv1.RuleCondition(value)
}

func accessGrant(value *grantsv1.Grant) *accessv1.Grant {
	if value == nil {
		return nil
	}
	out := &accessv1.Grant{
		Id: value.Id, TokenTypeId: value.TokenTypeId, GrantSourceId: value.GrantSourceId,
		Authorizer: value.Authorizer, HolderId: value.HolderId, AmountMinor: value.AmountMinor,
		AllowedCatalogScopes: value.AllowedCatalogScopes, DeniedCatalogScopes: value.DeniedCatalogScopes,
		ExpiresAt: value.ExpiresAt, IssuedAt: value.IssuedAt, Status: accessv1.GrantStatus(value.Status),
		IdempotencyKey: value.IdempotencyKey, RequiredEvidence: value.RequiredEvidence,
		RecurrenceSeconds: value.RecurrenceSeconds, NextIssueAt: value.NextIssueAt, CancelledAt: value.CancelledAt,
		Rules: make([]*accessv1.GrantRule, 0, len(value.Rules)),
	}
	for _, rule := range value.Rules {
		if rule != nil {
			out.Rules = append(out.Rules, &accessv1.GrantRule{Id: rule.Id, Condition: accessv1.RuleCondition(rule.Condition), Operands: rule.Operands, AmountLimit: rule.AmountLimit})
		}
	}
	return out
}
