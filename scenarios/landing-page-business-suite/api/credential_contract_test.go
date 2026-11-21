package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestCredentialDescriptorsMatchRuntimeOwnership(t *testing.T) {
	data, err := os.ReadFile("../.vrooli/service.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Credentials struct {
			Descriptors []struct {
				Field        string `json:"field"`
				Env          string `json:"env"`
				Required     bool   `json:"required"`
				Provisioning string `json:"provisioning"`
			} `json:"descriptors"`
			Consumers []struct {
				Field    string `json:"field"`
				Required bool   `json:"required"`
			} `json:"consumers"`
		} `json:"credentials"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	byField := make(map[string]struct {
		env          string
		required     bool
		provisioning string
	})
	for _, descriptor := range manifest.Credentials.Descriptors {
		byField[descriptor.Field] = struct {
			env          string
			required     bool
			provisioning string
		}{descriptor.Env, descriptor.Required, descriptor.Provisioning}
	}
	for _, field := range []string{"session-secret", "service-secret", "consumer-auth-private-key", "api-key-encryption-key", "remote-profile-encryption-key"} {
		descriptor, ok := byField[field]
		if !ok {
			t.Fatalf("missing generated descriptor %q", field)
		}
		if descriptor.env != "" || descriptor.provisioning != "generated" {
			t.Fatalf("generated descriptor %q = %#v; generated LPBS values must stay in-process", field, descriptor)
		}
	}
	admin, ok := byField["admin-default-password"]
	if !ok || !admin.required || admin.env != "" || admin.provisioning != "operator" {
		t.Fatalf("admin password descriptor = %#v; want required authority-backed descriptor", admin)
	}
	if sendgrid, ok := byField["sendgrid-api-key"]; !ok || sendgrid.provisioning != "operator" {
		t.Fatalf("SendGrid descriptor = %#v; want explicit operator provisioning", sendgrid)
	}
	for _, field := range []string{"stripe-secret-key", "stripe-webhook-secret"} {
		descriptor, ok := byField[field]
		if !ok || descriptor.env != "" || descriptor.provisioning != "operator" {
			t.Fatalf("Stripe server credential %q has wrong descriptor: %#v", field, descriptor)
		}
	}
	for _, field := range []string{
		"stripe-test-secret-key", "stripe-test-webhook-secret", "stripe-test-publishable-key",
		"stripe-live-secret-key", "stripe-live-webhook-secret", "stripe-live-publishable-key",
	} {
		descriptor, ok := byField[field]
		if !ok || descriptor.provisioning != "operator" {
			t.Fatalf("mode-specific Stripe credential %q must be declared as operator-provisioned: %#v", field, descriptor)
		}
	}
	consumerRequired := make(map[string]bool)
	for _, consumer := range manifest.Credentials.Consumers {
		if consumer.Field != "" {
			consumerRequired[consumer.Field] = consumer.Required
		}
	}
	for _, field := range []string{
		"stripe-test-secret-key", "stripe-test-webhook-secret", "stripe-test-publishable-key",
		"stripe-live-secret-key", "stripe-live-webhook-secret", "stripe-live-publishable-key",
	} {
		if required, ok := consumerRequired[field]; !ok || !required {
			t.Fatalf("active mode-specific Stripe consumer %q must be required in its selected mode", field)
		}
	}
	for _, field := range []string{"stripe-secret-key", "stripe-webhook-secret", "stripe-publishable-key"} {
		if required := consumerRequired[field]; required {
			t.Fatalf("legacy Stripe consumer %q must remain migration-only", field)
		}
	}
	if stripe, ok := byField["stripe-publishable-key"]; !ok || stripe.env != "STRIPE_PUBLISHABLE_KEY" {
		t.Fatalf("publishable Stripe descriptor must retain browser env injection: %#v", stripe)
	}
	for _, field := range []string{"admin-default-email", "auth-magic-link-base-url"} {
		if strings.TrimSpace(field) == "" {
			t.Fatal("test fixture field unexpectedly empty")
		}
		if _, ok := byField[field]; ok {
			t.Fatalf("configuration field %q must not be a credential descriptor", field)
		}
	}
}
