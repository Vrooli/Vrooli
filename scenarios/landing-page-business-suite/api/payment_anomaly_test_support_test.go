package main

import (
	"context"

	"landing-page-business-suite-api/internal/commerce"
)

func newPaymentAnomalyServiceForTest(ctx context.Context, db commerce.PaymentAnomalyStore, shutdownCtx context.Context) *commerce.PaymentAnomalyService {
	return commerce.NewPaymentAnomalyService(ctx, db, shutdownCtx, commerce.PaymentAnomalyRuntime{
		ScenarioName:   "landing-page-business-suite",
		NormalizeEmail: NormalizeEmail,
		Log:            logStructured,
		LogError:       logStructuredError,
	})
}
