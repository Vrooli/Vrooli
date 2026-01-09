package secrets

import (
	"strings"
	"testing"

	"scenario-to-desktop-runtime/manifest"
)

func ptrBool(v bool) *bool { return &v }

func TestValidator_Validate_RequiredMissing(t *testing.T) {
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{ID: "required_secret", Required: ptrBool(true)},
		},
	}

	v := NewValidator(m)
	errs := v.Validate(map[string]string{})
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if errs[0].SecretID != "required_secret" {
		t.Fatalf("expected error for required_secret, got %s", errs[0].SecretID)
	}
	if !strings.Contains(errs[0].Message, "missing") {
		t.Fatalf("expected 'missing' in error message, got %s", errs[0].Message)
	}
}

func TestValidator_Validate_RequiredPresent(t *testing.T) {
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{ID: "required_secret", Required: ptrBool(true)},
		},
	}

	v := NewValidator(m)
	errs := v.Validate(map[string]string{"required_secret": "value"})
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func TestValidator_Validate_OptionalMissing(t *testing.T) {
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{ID: "optional_secret", Required: ptrBool(false)},
		},
	}

	v := NewValidator(m)
	errs := v.Validate(map[string]string{})
	if len(errs) != 0 {
		t.Fatalf("expected no errors for optional secret, got %v", errs)
	}
}

func TestValidator_Validate_DefaultRequiredIsTrue(t *testing.T) {
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{ID: "default_required"},
		},
	}

	v := NewValidator(m)
	errs := v.Validate(map[string]string{})
	if len(errs) != 1 {
		t.Fatalf("expected 1 error (default required=true), got %d", len(errs))
	}
}

func TestValidator_Validate_FormatValid(t *testing.T) {
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{ID: "api_key", Required: ptrBool(true), Format: `^[A-Za-z0-9]{10,}$`},
		},
	}

	v := NewValidator(m)
	errs := v.Validate(map[string]string{"api_key": "abcdefghij1234"})
	if len(errs) != 0 {
		t.Fatalf("expected no errors for valid format, got %v", errs)
	}
}

func TestValidator_Validate_FormatInvalid(t *testing.T) {
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{ID: "api_key", Required: ptrBool(true), Format: `^[A-Za-z0-9]{10,}$`},
		},
	}

	v := NewValidator(m)
	errs := v.Validate(map[string]string{"api_key": "short"})
	if len(errs) != 1 {
		t.Fatalf("expected 1 error for invalid format, got %d", len(errs))
	}
	if !strings.Contains(errs[0].Message, "format") {
		t.Fatalf("expected 'format' in error message, got %s", errs[0].Message)
	}
}

func TestValidator_Validate_FormatInvalidRegex(t *testing.T) {
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{ID: "api_key", Required: ptrBool(true), Format: `[invalid`},
		},
	}

	v := NewValidator(m)
	errs := v.Validate(map[string]string{"api_key": "value"})
	if len(errs) != 1 {
		t.Fatalf("expected 1 error for invalid regex, got %d", len(errs))
	}
	if !strings.Contains(errs[0].Message, "invalid format pattern") {
		t.Fatalf("expected 'invalid format pattern' in error, got %s", errs[0].Message)
	}
}

func TestValidator_Validate_FormatSkippedForEmpty(t *testing.T) {
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{ID: "optional", Required: ptrBool(false), Format: `^[0-9]+$`},
		},
	}

	v := NewValidator(m)
	// Empty optional secret should not be validated for format
	errs := v.Validate(map[string]string{})
	if len(errs) != 0 {
		t.Fatalf("expected no errors for empty optional with format, got %v", errs)
	}
}

func TestValidator_Validate_MultipleErrors(t *testing.T) {
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{ID: "secret1", Required: ptrBool(true)},
			{ID: "secret2", Required: ptrBool(true), Format: `^[0-9]+$`},
			{ID: "secret3", Required: ptrBool(true)},
		},
	}

	v := NewValidator(m)
	errs := v.Validate(map[string]string{
		"secret2": "not-a-number", // format error
		// secret1 missing
		// secret3 missing
	})
	if len(errs) != 3 {
		t.Fatalf("expected 3 errors, got %d: %v", len(errs), errs)
	}
}

func TestValidationErrors_Error(t *testing.T) {
	errs := ValidationErrors{
		{SecretID: "a", Message: "missing"},
		{SecretID: "b", Message: "format error"},
	}
	msg := errs.Error()
	if !strings.Contains(msg, "secret a") || !strings.Contains(msg, "secret b") {
		t.Fatalf("error message should contain secret IDs: %s", msg)
	}
}

func TestValidationErrors_ErrorEmpty(t *testing.T) {
	errs := ValidationErrors{}
	if errs.Error() != "" {
		t.Fatalf("empty errors should return empty string, got %q", errs.Error())
	}
}

func TestValidator_MissingRequired(t *testing.T) {
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{ID: "r1", Required: ptrBool(true)},
			{ID: "r2", Required: ptrBool(true)},
			{ID: "o1", Required: ptrBool(false)},
		},
	}

	v := NewValidator(m)
	missing := v.MissingRequired(map[string]string{"r1": "value"})
	if len(missing) != 1 || missing[0] != "r2" {
		t.Fatalf("expected [r2], got %v", missing)
	}
}

func TestValidator_InvalidFormats(t *testing.T) {
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{ID: "num", Format: `^[0-9]+$`},
			{ID: "alpha", Format: `^[a-z]+$`},
		},
	}

	v := NewValidator(m)
	errs := v.InvalidFormats(map[string]string{
		"num":   "abc",   // invalid
		"alpha": "hello", // valid
	})
	if len(errs) != 1 {
		t.Fatalf("expected 1 format error, got %d", len(errs))
	}
	if errs[0].SecretID != "num" {
		t.Fatalf("expected error for 'num', got %s", errs[0].SecretID)
	}
}

func TestValidator_ValidateSecret(t *testing.T) {
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{ID: "api_key", Required: ptrBool(true), Format: `^key_`},
		},
	}

	v := NewValidator(m)

	// Unknown secret
	if err := v.ValidateSecret("unknown", "value"); err == nil {
		t.Fatal("expected error for unknown secret")
	}

	// Missing required
	if err := v.ValidateSecret("api_key", ""); err == nil {
		t.Fatal("expected error for empty required secret")
	}

	// Invalid format
	if err := v.ValidateSecret("api_key", "invalid"); err == nil {
		t.Fatal("expected error for invalid format")
	}

	// Valid
	if err := v.ValidateSecret("api_key", "key_abc123"); err != nil {
		t.Fatalf("unexpected error for valid secret: %v", err)
	}
}
