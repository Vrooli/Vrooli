package main

import (
	"context"
	"strings"
	"testing"
)

// A deployment that never configured commerce promised nothing, so the
// payments dependency must stay quiet rather than degrading health forever.
func TestPaymentsCheckSilentWhenCommerceIsNotConfigured(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()
	t.Setenv("STRIPE_MODE", "")

	result := server.paymentsCheck().Check(context.Background())

	if !result.Connected || result.Error != nil {
		t.Fatalf("an unconfigured deployment must not degrade health: %v", result.Error)
	}
}

// The failure this covers: a deployment declares live payments but has no
// usable credentials, and before this check its /health body said nothing.
func TestPaymentsCheckReportsDeclaredButUnusableCommerce(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()
	t.Setenv("STRIPE_MODE", "live")

	result := server.paymentsCheck().Check(context.Background())

	if result.Connected || result.Error == nil {
		t.Fatal("declared live payments with no credentials must degrade health")
	}
	if !strings.Contains(result.Error.Error(), "stripe-live") {
		t.Errorf("error = %v, want the missing credential fields named", result.Error)
	}
	if result.Name != "payments" {
		t.Errorf("name = %q, want payments", result.Name)
	}
}

// An unreadable STRIPE_MODE is a configuration error an operator must see,
// not a silent default.
func TestPaymentsConfiguredRejectsAnUnknownMode(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()
	t.Setenv("STRIPE_MODE", "sandbox")

	err := server.stripePaymentsConfigured(context.Background())

	if err == nil || !strings.Contains(err.Error(), "STRIPE_MODE") {
		t.Fatalf("error = %v, want an explicit STRIPE_MODE complaint", err)
	}
}
