package secrets

import (
	"regexp"
	"scenario-to-cloud/domain"
	"testing"
)

func TestGeneratorInterface(t *testing.T) {
	t.Parallel()

	// Verify Generator implements GeneratorFunc interface
	var _ GeneratorFunc = (*Generator)(nil)
}

func TestNewGenerator(t *testing.T) {
	t.Parallel()

	gen := NewGenerator()
	if gen == nil {
		t.Fatal("NewGenerator() returned nil")
	}
}

func TestGenerator_GenerateSecrets_IgnoresNonPerInstallGenerated(t *testing.T) {
	t.Parallel()

	gen := NewGenerator()
	plans := []domain.BundleSecretPlan{
		{
			ID:    "env-secret",
			Class: "env",
			Target: domain.BundleSecretTarget{
				Name: "API_KEY",
			},
		},
		{
			ID:    "vault-secret",
			Class: "vault",
			Target: domain.BundleSecretTarget{
				Name: "DB_PASSWORD",
			},
		},
	}

	generated, err := gen.GenerateSecrets(plans)
	if err != nil {
		t.Fatalf("GenerateSecrets() error = %v", err)
	}

	if len(generated) != 0 {
		t.Errorf("GenerateSecrets() generated %d secrets, want 0", len(generated))
	}
}

func TestGenerator_GenerateSecrets_ProcessesPerInstallGenerated(t *testing.T) {
	t.Parallel()

	gen := NewGenerator()
	plans := []domain.BundleSecretPlan{
		{
			ID:    "db-password",
			Class: "per_install_generated",
			Target: domain.BundleSecretTarget{
				Name: "POSTGRES_PASSWORD",
			},
		},
	}

	generated, err := gen.GenerateSecrets(plans)
	if err != nil {
		t.Fatalf("GenerateSecrets() error = %v", err)
	}

	if len(generated) != 1 {
		t.Fatalf("GenerateSecrets() generated %d secrets, want 1", len(generated))
	}

	secret := generated[0]
	if secret.ID != "db-password" {
		t.Errorf("secret.ID = %q, want %q", secret.ID, "db-password")
	}
	if secret.Key != "POSTGRES_PASSWORD" {
		t.Errorf("secret.Key = %q, want %q", secret.Key, "POSTGRES_PASSWORD")
	}
	if len(secret.Value) != 25 {
		t.Errorf("secret.Value length = %d, want 25 (default)", len(secret.Value))
	}
}

func TestGenerator_GenerateSecrets_WithCustomLength(t *testing.T) {
	t.Parallel()

	gen := NewGenerator()
	plans := []domain.BundleSecretPlan{
		{
			ID:    "short-secret",
			Class: "per_install_generated",
			Target: domain.BundleSecretTarget{
				Name: "SHORT_SECRET",
			},
			Generator: map[string]interface{}{
				"type":   "random",
				"length": float64(10),
			},
		},
	}

	generated, err := gen.GenerateSecrets(plans)
	if err != nil {
		t.Fatalf("GenerateSecrets() error = %v", err)
	}

	if len(generated) != 1 {
		t.Fatalf("GenerateSecrets() generated %d secrets, want 1", len(generated))
	}

	if len(generated[0].Value) != 10 {
		t.Errorf("secret.Value length = %d, want 10", len(generated[0].Value))
	}
}

func TestGenerator_GenerateSecrets_UUID(t *testing.T) {
	t.Parallel()

	gen := NewGenerator()
	plans := []domain.BundleSecretPlan{
		{
			ID:    "uuid-secret",
			Class: "per_install_generated",
			Target: domain.BundleSecretTarget{
				Name: "SESSION_ID",
			},
			Generator: map[string]interface{}{
				"type": "uuid",
			},
		},
	}

	generated, err := gen.GenerateSecrets(plans)
	if err != nil {
		t.Fatalf("GenerateSecrets() error = %v", err)
	}

	if len(generated) != 1 {
		t.Fatalf("GenerateSecrets() generated %d secrets, want 1", len(generated))
	}

	// UUID v4 format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !uuidRegex.MatchString(generated[0].Value) {
		t.Errorf("secret.Value = %q, want UUID v4 format", generated[0].Value)
	}
}

func TestGenerator_AlphanumericCharset(t *testing.T) {
	t.Parallel()

	gen := NewGenerator()
	plans := []domain.BundleSecretPlan{
		{
			ID:    "alnum-secret",
			Class: "per_install_generated",
			Target: domain.BundleSecretTarget{
				Name: "ALNUM_SECRET",
			},
			Generator: map[string]interface{}{
				"type":    "random",
				"length":  float64(50),
				"charset": "alnum",
			},
		},
	}

	generated, err := gen.GenerateSecrets(plans)
	if err != nil {
		t.Fatalf("GenerateSecrets() error = %v", err)
	}

	// Verify all characters are alphanumeric
	alnumRegex := regexp.MustCompile(`^[A-Za-z0-9]+$`)
	if !alnumRegex.MatchString(generated[0].Value) {
		t.Errorf("secret.Value = %q, want only alphanumeric characters", generated[0].Value)
	}
}

func TestGenerator_MultipleSecrets(t *testing.T) {
	t.Parallel()

	gen := NewGenerator()
	plans := []domain.BundleSecretPlan{
		{
			ID:    "secret-1",
			Class: "per_install_generated",
			Target: domain.BundleSecretTarget{
				Name: "SECRET_1",
			},
		},
		{
			ID:    "secret-2",
			Class: "per_install_generated",
			Target: domain.BundleSecretTarget{
				Name: "SECRET_2",
			},
		},
		{
			ID:    "ignored",
			Class: "env",
			Target: domain.BundleSecretTarget{
				Name: "IGNORED",
			},
		},
	}

	generated, err := gen.GenerateSecrets(plans)
	if err != nil {
		t.Fatalf("GenerateSecrets() error = %v", err)
	}

	if len(generated) != 2 {
		t.Errorf("GenerateSecrets() generated %d secrets, want 2", len(generated))
	}

	// Verify secrets are unique
	if generated[0].Value == generated[1].Value {
		t.Error("Generated secrets should be unique")
	}
}
