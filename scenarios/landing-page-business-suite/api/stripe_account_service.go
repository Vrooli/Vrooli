package main

import (
	"context"

	"landing-page-business-suite-api/internal/commerce"
)

// LinkUserToStripeCustomer preserves the Stripe aggregate's public contract
// while commerce owns customer identity persistence.
func (s *StripeService) LinkUserToStripeCustomer(email, customerID string) error {
	if err := commerce.NewAccountLinkService(s.db).LinkUserToStripeCustomer(email, customerID); err != nil {
		return err
	}
	logStructured("stripe_customer_linked", map[string]interface{}{
		"level":       "info",
		"email":       NormalizeEmail(email),
		"customer_id": customerID,
	})
	return nil
}

// LookupCustomerID preserves the Stripe aggregate's public account-link
// contract while delegating lookup behavior to commerce.
func (s *StripeService) LookupCustomerID(userIdentity string) string {
	return commerce.NewAccountLinkService(s.db).LookupCustomerID(userIdentity)
}

// MigrateCustomerEmail preserves the Stripe aggregate's public contract while
// commerce performs the transactional migration.
func (s *StripeService) MigrateCustomerEmail(ctx context.Context, oldEmail, newEmail, customerID string) error {
	if err := commerce.NewAccountLinkService(s.db).MigrateCustomerEmail(ctx, oldEmail, newEmail, customerID); err != nil {
		return err
	}
	logStructured("customer_email_migrated", map[string]interface{}{
		"level":       "info",
		"old_email":   NormalizeEmail(oldEmail),
		"new_email":   NormalizeEmail(newEmail),
		"customer_id": customerID,
	})
	return nil
}

// These package-local helpers retain existing internal callers and focused
// tests while routing all customer identity behavior through commerce.
func (s *StripeService) linkUserToStripeCustomer(email, customerID string) error {
	return s.LinkUserToStripeCustomer(email, customerID)
}

func (s *StripeService) lookupCustomerID(user string) string {
	return s.LookupCustomerID(user)
}
