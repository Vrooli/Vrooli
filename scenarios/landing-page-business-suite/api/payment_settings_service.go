package main

import "landing-page-business-suite-api/internal/commerce"

type PaymentSettingsStore = commerce.PaymentSettingsStore
type PaymentSettingsService = commerce.PaymentSettingsService
type StripeSettingsInput = commerce.StripeSettingsInput

func NewPaymentSettingsService(db PaymentSettingsStore) *PaymentSettingsService {
	return commerce.NewPaymentSettingsService(db)
}
