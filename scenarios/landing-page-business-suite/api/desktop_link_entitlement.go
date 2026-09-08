package main

import (
	"context"
	"fmt"

	link "landing-page-business-suite-api/internal/desktoplink"

	entitlementclient "github.com/vrooli/vrooli/packages/entitlementclient-go"
)

func (s *Server) issueDesktopEntitlementLease(ctx context.Context, connected link.Link) (link.EntitlementLease, error) {
	user, err := s.userAuthService.GetUserByID(ctx, connected.LPBSUserID)
	if err != nil {
		return link.EntitlementLease{}, fmt.Errorf("resolve linked LPBS user: %w", err)
	}
	account, err := s.businessAccounts.ResolveForUser(ctx, connected.LPBSUserID, user.Email, connected.BusinessAccountID)
	if err != nil {
		return link.EntitlementLease{}, fmt.Errorf("resolve linked business account: %w", err)
	}
	payload, err := s.accountService.GetEntitlementsForBusinessAccountContext(ctx, account.BillingEmail, account.ID)
	if err != nil {
		return link.EntitlementLease{}, fmt.Errorf("resolve linked entitlement: %w", err)
	}
	lease, err := s.userAuthService.SignEntitlementLease(entitlementclient.Payload{
		UserIdentity: connected.LocalPrincipal, BusinessAccountID: connected.BusinessAccountID, LinkID: connected.ID,
		InstallationID: connected.InstallationID, Resource: connected.Resource, Audience: connected.Audience,
		Scopes: connected.Scopes, Status: payload.Status, PlanTier: payload.PlanTier, PlanRank: payload.PlanRank,
		PriceID: payload.PriceID, Features: payload.Features, Limits: payload.Limits, NotAfter: payload.NotAfter,
		BillingCycleStart: payload.BillingCycleStart, Credits: payload.Credits, Subscription: payload.Subscription,
	})
	if err != nil {
		return link.EntitlementLease{}, fmt.Errorf("sign linked entitlement: %w", err)
	}
	return link.EntitlementLease{Token: lease, ExpiresAt: payload.NotAfter}, nil
}
