package main

import (
	"context"

	billinghttp "landing-page-business-suite-api/handlers/commerce"
)

func billingLimitsDependencies() billinghttp.LimitsDependencies {
	return billinghttp.LimitsDependencies{WriteError: writeJSONError, Log: logStructuredError}
}

func billingDependencies(service *StripeService) billinghttp.Dependencies {
	return billinghttp.Dependencies{
		ValidateEmail:       ValidateEmailForHandler,
		NormalizeRedirect:   NormalizeRedirectURLForHandler,
		ValidateOptionalURL: ValidateURLOptional,
		CreateCheckout:      service.CreateCheckoutSession,
		CreatePortal: func(ctx context.Context, user, returnURL string) (any, error) {
			return service.CreateBillingPortalSession(ctx, user, returnURL)
		},
		UserEmail:     getUserEmail,
		ClassifyError: classifyStripeError,
		WriteJSON:     writeJSON,
		WriteError:    writeJSONError,
		Log:           logStructuredError,
	}
}

func billingWebhookDependencies(service *StripeService) billinghttp.WebhookDependencies {
	return billinghttp.WebhookDependencies{
		Handle:     service.HandleWebhook,
		WriteError: writeJSONError,
		WriteJSON:  writeJSONSuccessData,
		Log:        logStructuredError,
	}
}

func billingConnectDependencies(service *StripeService) billinghttp.ConnectDependencies {
	return billinghttp.ConnectDependencies{
		Payments:            service,
		ValidateEmail:       ValidateEmail,
		NormalizeRedirect:   NormalizeRedirectURL,
		ValidateOptionalURL: ValidateURLOptional,
		UserEmail:           getUserEmail,
	}
}
