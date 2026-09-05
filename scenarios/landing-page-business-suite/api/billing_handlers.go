package main

import (
	billinghttp "landing-page-business-suite-api/handlers/commerce"
	"landing-page-business-suite-api/internal/logx"
)

func billingLimitsDependencies() billinghttp.LimitsDependencies {
	return billinghttp.LimitsDependencies{WriteError: writeJSONError, Log: logx.Error}
}

func billingWebhookDependencies(service *StripeService) billinghttp.WebhookDependencies {
	return billinghttp.WebhookDependencies{Handle: service.HandleWebhook, WriteError: writeJSONError, WriteJSON: writeJSONSuccessData, Log: logx.Error}
}

func billingConnectDependencies(service *StripeService) billinghttp.ConnectDependencies {
	return billinghttp.ConnectDependencies{Payments: service, ValidateEmail: ValidateEmail, NormalizeRedirect: NormalizeRedirectURL, ValidateOptionalURL: ValidateURLOptional, UserEmail: getUserEmail}
}
