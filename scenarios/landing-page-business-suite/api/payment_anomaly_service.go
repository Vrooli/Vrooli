package main

import (
	"context"
	"fmt"

	"landing-page-business-suite-api/internal/commerce"
)

// LogPaymentAnomaly is the server-facing entry point for producers that need
// the composed commerce service. Payment policy lives in internal/commerce.
func LogPaymentAnomaly(ctx context.Context, srv *Server, anomaly commerce.PaymentAnomaly) (int64, error) {
	if srv == nil || srv.paymentAnomaly == nil {
		return 0, fmt.Errorf("payment anomaly service not initialized")
	}
	return srv.paymentAnomaly.Log(ctx, anomaly)
}
