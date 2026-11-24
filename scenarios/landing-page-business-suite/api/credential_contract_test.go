package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/api-core/lpbsidentity"
)

func TestCredentialDescriptorsMatchRuntimeOwnership(t *testing.T) {
	data, err := os.ReadFile("../.vrooli/service.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Credentials struct {
			Descriptors []struct {
				LogicalID    string `json:"logical_id"`
				Field        string `json:"field"`
				Env          string `json:"env"`
				Required     bool   `json:"required"`
				Provisioning string `json:"provisioning"`
			} `json:"descriptors"`
			Consumers []struct {
				LogicalID string `json:"logical_id"`
				Field     string `json:"field"`
				SourceRef string `json:"source_ref"`
				Required  bool   `json:"required"`
			} `json:"consumers"`
		} `json:"credentials"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	byField := make(map[string]struct {
		logicalID    string
		env          string
		required     bool
		provisioning string
	})
	for _, descriptor := range manifest.Credentials.Descriptors {
		byField[descriptor.Field] = struct {
			logicalID    string
			env          string
			required     bool
			provisioning string
		}{descriptor.LogicalID, descriptor.Env, descriptor.Required, descriptor.Provisioning}
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
	for _, field := range []string{
		"stripe-test-secret-key", "stripe-test-webhook-secret", "stripe-test-publishable-key",
		"stripe-live-secret-key", "stripe-live-webhook-secret", "stripe-live-publishable-key",
	} {
		descriptor, ok := byField[field]
		if !ok || descriptor.provisioning != "operator" {
			t.Fatalf("mode-specific Stripe credential %q must be declared as operator-provisioned: %#v", field, descriptor)
		}
	}
	if _, ok := byField["money-ledger-service-token"]; ok {
		t.Fatal("Money Ledger writer must use its Scenario Authenticator service-principal credential, not the duplicate legacy LPBS field")
	}
	principalData, err := os.ReadFile("../.vrooli/service-principals.json")
	if err != nil {
		t.Fatal(err)
	}
	var principalManifest struct {
		Principals []struct {
			Name     string   `json:"name"`
			Resource string   `json:"resource"`
			Audience string   `json:"audience"`
			Scopes   []string `json:"scopes"`
		} `json:"principals"`
	}
	if err := json.Unmarshal(principalData, &principalManifest); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, principal := range principalManifest.Principals {
		if principal.Name == "landing-page-business-suite-money-ledger-writer" {
			found = principal.Resource == "money-ledger" && principal.Audience == "scenario-authenticator:money-ledger" && len(principal.Scopes) == 1 && principal.Scopes[0] == "money-ledger:write"
		}
	}
	if !found {
		t.Fatal("Money Ledger writer service principal must remain restricted to the declared money-ledger:write scope")
	}
	bridgeDispatchFound := false
	for _, principal := range principalManifest.Principals {
		if principal.Name == "landing-page-business-suite-bridge-dispatch" {
			bridgeDispatchFound = principal.Resource == "vrooli-bridge" && principal.Audience == "scenario-authenticator:vrooli-bridge-dispatch" && len(principal.Scopes) == 1 && principal.Scopes[0] == "vrooli-bridge:dispatch"
		}
	}
	if !bridgeDispatchFound {
		t.Fatal("LPBS Bridge dispatcher must use the dedicated exact vrooli-bridge:dispatch service principal")
	}
	browserSettlementPrincipal := lpbsidentity.AgentManagerBrowserSessionSettlement
	if browserSettlementPrincipal.Principal("agent-manager") != "agent-manager-browser-session-settler" || browserSettlementPrincipal.CredentialIdentity("agent-manager") != "scenario-authenticator/service-principal/agent-manager-browser-session-settler/landing-page-business-suite" {
		t.Fatal("AM browser settlement role must address the exact dedicated Scenario Authenticator identity")
	}
	settlementFound := false
	for _, principal := range principalManifest.Principals {
		if principal.Name == browserSettlementPrincipal.Principal("agent-manager") {
			settlementFound = principal.Resource == browserSettlementPrincipal.Resource && principal.Audience == browserSettlementPrincipal.Audience && len(principal.Scopes) == 1 && principal.Scopes[0] == browserSettlementPrincipal.Scope
		}
	}
	if !settlementFound {
		t.Fatal("AM browser settlement service principal must match the exact role and sole scope")
	}
	consumerByField := make(map[string]struct{ logicalID, source string })
	for _, consumer := range manifest.Credentials.Consumers {
		consumerByField[consumer.Field] = struct{ logicalID, source string }{consumer.LogicalID, consumer.SourceRef}
	}
	for _, field := range []string{
		"stripe-test-secret-key", "stripe-test-webhook-secret", "stripe-test-publishable-key",
		"stripe-live-secret-key", "stripe-live-webhook-secret", "stripe-live-publishable-key",
		"delivery-s3-access-key-id", "delivery-s3-secret-access-key", "delivery-s3-session-token",
		"mailgun-api-key", "sendgrid-api-key", "sendgrid-webhook-public-key", "smtp-password",
		"session-secret", "session-secret-previous", "consumer-auth-private-key",
		"service-secret", "metrics-reader-token", "admin-default-password",
	} {
		descriptor, declared := byField[field]
		if !declared || descriptor.logicalID != "vrooli/landing-page-business-suite" || descriptor.env != "" {
			t.Fatalf("credential %q must be declared under LPBS authority identity without an environment injection: %#v", field, descriptor)
		}
		consumer, found := consumerByField[field]
		if !found || consumer.logicalID != descriptor.logicalID || strings.TrimSpace(consumer.source) == "" {
			t.Fatalf("credential %q must bind to an exact LPBS authority consumer and source reference: %#v", field, consumer)
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
		if descriptor, ok := byField[field]; ok {
			t.Fatalf("retired Stripe credential %q must not remain in the active descriptor manifest: %#v", field, descriptor)
		}
		if _, ok := consumerRequired[field]; ok {
			t.Fatalf("retired Stripe credential %q must not remain in the consumer manifest", field)
		}
	}
	for _, field := range []string{"admin-default-email", "auth-magic-link-base-url"} {
		if strings.TrimSpace(field) == "" {
			t.Fatal("test fixture field unexpectedly empty")
		}
		if _, ok := byField[field]; ok {
			t.Fatalf("configuration field %q must not be a credential descriptor", field)
		}
	}
	cloudData, err := os.ReadFile("../cloud-manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var cloud struct {
		Secrets struct {
			Summary struct {
				Total      int `json:"total_secrets"`
				UserPrompt int `json:"user_prompt"`
			} `json:"summary"`
			BundleSecrets []struct {
				ID         string `json:"id"`
				Descriptor struct {
					LogicalID string `json:"logical_id"`
					Field     string `json:"field"`
				} `json:"descriptor"`
				Target struct {
					Type string `json:"type"`
					Name string `json:"name"`
				} `json:"target"`
			} `json:"bundle_secrets"`
		} `json:"secrets"`
	}
	if err := json.Unmarshal(cloudData, &cloud); err != nil {
		t.Fatal(err)
	}
	cloudByID := make(map[string]struct{ descriptor, target string })
	for _, secret := range cloud.Secrets.BundleSecrets {
		cloudByID[secret.ID] = struct{ descriptor, target string }{secret.Descriptor.LogicalID + ":" + secret.Descriptor.Field, secret.Target.Type + ":" + secret.Target.Name}
	}
	cloudTargets := map[string]string{
		"admin-default-password": "ADMIN_DEFAULT_PASSWORD",
		"stripe-test-secret-key": "STRIPE_TEST_SECRET_KEY", "stripe-test-webhook-secret": "STRIPE_TEST_WEBHOOK_SECRET", "stripe-test-publishable-key": "STRIPE_TEST_PUBLISHABLE_KEY",
		"stripe-live-secret-key": "STRIPE_LIVE_SECRET_KEY", "stripe-live-webhook-secret": "STRIPE_LIVE_WEBHOOK_SECRET", "stripe-live-publishable-key": "STRIPE_LIVE_PUBLISHABLE_KEY",
		"delivery-s3-access-key-id": "DELIVERY_S3_ACCESS_KEY_ID", "delivery-s3-secret-access-key": "DELIVERY_S3_SECRET_ACCESS_KEY", "delivery-s3-session-token": "DELIVERY_S3_SESSION_TOKEN",
		"mailgun-api-key": "MAILGUN_API_KEY", "sendgrid-api-key": "SENDGRID_API_KEY", "sendgrid-webhook-public-key": "SENDGRID_WEBHOOK_PUBLIC_KEY", "smtp-password": "SMTP_PASSWORD",
	}
	for field, expectedTarget := range cloudTargets {
		entry, ok := cloudByID[field]
		if !ok || entry.descriptor != "vrooli/landing-page-business-suite:"+field || entry.target != "env:"+expectedTarget {
			t.Fatalf("cloud manifest must bind %q to its LPBS descriptor and exact owner target env %q: %#v", field, expectedTarget, entry)
		}
	}
	if cloud.Secrets.Summary.Total != len(cloud.Secrets.BundleSecrets) || cloud.Secrets.Summary.UserPrompt != len(cloud.Secrets.BundleSecrets) {
		t.Fatalf("cloud secret summary disagrees with explicit descriptors: summary=%#v bundle_count=%d", cloud.Secrets.Summary, len(cloud.Secrets.BundleSecrets))
	}
	if strings.Contains(string(cloudData), `"value":`) {
		t.Fatal("cloud manifest must contain descriptors only, never credential values")
	}
	for _, secret := range cloud.Secrets.BundleSecrets {
		switch secret.ID {
		case "stripe-secret-key", "stripe-webhook-secret", "service-secret":
			t.Fatalf("credential %q must not be distributed into an LPBS process environment: %#v", secret.ID, secret.Target)
		}
	}
}

func TestAgentManagerBrowserSettlementPrincipalMatchesLPBSRole(t *testing.T) {
	role := lpbsidentity.AgentManagerBrowserSessionSettlement
	if role.Principal("agent-manager") != "agent-manager-browser-session-settler" || role.CredentialIdentity("agent-manager") != "scenario-authenticator/service-principal/agent-manager-browser-session-settler/landing-page-business-suite" {
		t.Fatalf("principal=%q identity=%q", role.Principal("agent-manager"), role.CredentialIdentity("agent-manager"))
	}
	data, err := os.ReadFile("../.vrooli/service-principals.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Principals []struct {
			Name     string   `json:"name"`
			Resource string   `json:"resource"`
			Audience string   `json:"audience"`
			Scopes   []string `json:"scopes"`
		} `json:"principals"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, principal := range manifest.Principals {
		if principal.Name == role.Principal("agent-manager") {
			if principal.Resource == role.Resource && principal.Audience == role.Audience && len(principal.Scopes) == 1 && principal.Scopes[0] == role.Scope {
				return
			}
			t.Fatalf("manifest principal does not match exact LPBS role: %+v", principal)
		}
	}
	t.Fatal("missing AM browser settlement service principal")
}
