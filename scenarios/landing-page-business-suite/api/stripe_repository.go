package main

import "landing-page-business-suite-api/internal/commerce"

// StripeRepository and its records are commerce persistence types. These
// aliases retain the API composition boundary while the implementation lives
// with the subscription, checkout, and credit workflows it persists.
type (
	StripeRepository        = commerce.StripeRepository
	StripeStore             = commerce.StripeStore
	SubscriptionRecord      = commerce.SubscriptionRecord
	CheckoutSessionRecord   = commerce.CheckoutSessionRecord
	CreditWalletRecord      = commerce.CreditWalletRecord
	CreditTransactionRecord = commerce.CreditTransactionRecord
)

func NewStripeRepository(db StripeStore) *StripeRepository {
	return commerce.NewStripeRepository(db)
}
