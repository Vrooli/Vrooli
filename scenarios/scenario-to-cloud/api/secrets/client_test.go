package secrets

import (
	"testing"

	"scenario-to-cloud/domain"
)

func TestTransformSecretsPreservesGeneratedCredentialIdentity(t *testing.T) {
	plans := transformSecrets([]ManagerSecret{{
		ID:               "postgres-password",
		SecretKey:        "POSTGRES_PASSWORD",
		SecretType:       "password",
		Classification:   "infrastructure",
		HandlingStrategy: "generate",
		LogicalID:        "vrooli/postgres",
		Field:            "password",
		Required:         true,
	}}, "tier-4-saas")

	if len(plans) != 1 {
		t.Fatalf("got %d plans, want 1", len(plans))
	}
	plan := plans[0]
	if plan.Class != domain.SecretClassPerInstallGenerated {
		t.Fatalf("class = %q, want %q", plan.Class, domain.SecretClassPerInstallGenerated)
	}
	if plan.Descriptor == nil {
		t.Fatal("generated plan lost its credential descriptor")
	}
	if plan.Descriptor.LogicalID != "vrooli/postgres" || plan.Descriptor.Field != "password" {
		t.Fatalf("descriptor = %+v, want logical_id=vrooli/postgres field=password", plan.Descriptor)
	}
	if got, want := plan.Target.Name, "POSTGRES_PASSWORD"; got != want {
		t.Fatalf("target = %q, want %q", got, want)
	}
}
