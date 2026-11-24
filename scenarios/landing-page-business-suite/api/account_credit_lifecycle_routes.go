package main

import (
	"errors"
	"net/http"
	"strings"

	"github.com/vrooli/api-core/lpbsidentity"
	billinghttp "landing-page-business-suite-api/handlers/commerce"
	"landing-page-business-suite-api/internal/commerce"
)

func registerAccountCreditLifecycleRoutes(s *Server) {
	verifier := lpbsidentity.AccountCreditLifecycle.NewVerifier()
	service := commerce.NewAccountCreditLifecycleService(s.usageService)
	billinghttp.RegisterAccountCreditLifecycleRoutes(s.router, billinghttp.AccountCreditLifecycleDependencies{
		Service:      service,
		UserID:       func(r *http.Request) string { return getUserID(r.Context()) },
		UserIdentity: func(r *http.Request) string { return getUserEmail(r.Context()) },
		ResolveAccount: func(r *http.Request, requestedID string) (string, error) {
			account, err := s.businessAccounts.ResolveForUser(r.Context(), getUserID(r.Context()), getUserEmail(r.Context()), requestedID)
			if err != nil {
				return "", err
			}
			return account.ID, nil
		},
		AccountTier: func(r *http.Request, actor, accountID string) (string, error) {
			subscription, err := s.accountService.GetSubscriptionForBusinessAccountContext(r.Context(), actor, accountID)
			if err != nil {
				return "", err
			}
			if subscription == nil || subscription.PlanTier == nil || strings.TrimSpace(*subscription.PlanTier) == "" {
				return "free", nil
			}
			return strings.TrimSpace(*subscription.PlanTier), nil
		},
		ResolveApp: func(r *http.Request) (string, error) {
			raw := strings.TrimSpace(r.Header.Get(lpbsidentity.AccountCreditLifecycleAuthorizationHeader))
			token, found := strings.CutPrefix(raw, "Bearer ")
			if !found || strings.TrimSpace(token) == "" || strings.ContainsAny(token, " \t\r\n") {
				return "", errors.New("account-credit service bearer required")
			}
			principal, err := verifier.Verify(r.Context(), strings.TrimSpace(token))
			if err != nil {
				return "", err
			}
			return lpbsidentity.AccountCreditLifecycle.Source(principal)
		},
		WriteError: writeJSONError,
	}, s.requireUserAuth)
}
