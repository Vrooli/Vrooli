package main

import "landing-page-business-suite-api/internal/commerce"

func NewPaymentSettingsService(db commerce.PaymentSettingsStore) *commerce.PaymentSettingsService {
	return commerce.NewPaymentSettingsService(db)
}
