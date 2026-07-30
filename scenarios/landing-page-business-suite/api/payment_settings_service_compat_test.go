package main

import "landing-page-business-suite-api/internal/commerce"

// Test-only compatibility constructor for behavior-focused legacy tests.
func NewPaymentSettingsService(db commerce.PaymentSettingsStore) *commerce.PaymentSettingsService {
	return commerce.NewPaymentSettingsService(db)
}
