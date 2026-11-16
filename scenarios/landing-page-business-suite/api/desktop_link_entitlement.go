package main

import (
	"context"
	"fmt"

	link "landing-page-business-suite-api/internal/desktoplink"
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
	claims := payload.LeaseClaims(connected.LocalPrincipal)
	claims.BusinessAccountID = connected.BusinessAccountID
	claims.LinkID = connected.ID
	claims.InstallationID = connected.InstallationID
	claims.Resource = connected.Resource
	claims.Audience = connected.Audience
	claims.Scopes = connected.Scopes
	lease, err := s.userAuthService.SignEntitlementLease(claims)
	if err != nil {
		return link.EntitlementLease{}, fmt.Errorf("sign linked entitlement: %w", err)
	}
	return link.EntitlementLease{Token: lease, ExpiresAt: payload.NotAfter}, nil
}
