package main

import (
	"context"

	billinghttp "landing-page-business-suite-api/handlers/commerce"
	"landing-page-business-suite-api/internal/businessaccount"
	"landing-page-business-suite-api/internal/logx"
)

func billingLimitsDependencies() billinghttp.LimitsDependencies {
	return billinghttp.LimitsDependencies{WriteError: writeJSONError, Log: logx.Error}
}

func billingWebhookDependencies(service *StripeService) billinghttp.WebhookDependencies {
	return billinghttp.WebhookDependencies{Handle: service.HandleWebhook, WriteError: writeJSONError, WriteJSON: writeJSONSuccessData, Log: logx.Error}
}

func billingConnectDependencies(service *StripeService, repositories ...businessaccount.Repository) billinghttp.ConnectDependencies {
	deps := billinghttp.ConnectDependencies{Payments: service, ValidateEmail: ValidateEmail, NormalizeRedirect: NormalizeRedirectURL, ValidateOptionalURL: ValidateURLOptional, UserEmail: getUserEmail, UserID: getUserID}
	if len(repositories) > 0 && repositories[0] != nil {
		deps.ResolveBusinessAccount = func(ctx context.Context, userID, userEmail, requestedID string) (businessaccount.Account, error) {
			return repositories[0].ResolveForUser(ctx, userID, userEmail, requestedID)
		}
	}
	return deps
}
