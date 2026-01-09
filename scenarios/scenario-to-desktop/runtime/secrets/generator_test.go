package secrets

import (
	"strings"
	"testing"

	"scenario-to-desktop-runtime/manifest"
)

func TestGenerator_GenerateSecrets_SkipsNonGenerated(t *testing.T) {
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{ID: "user_provided", Class: "user_prompt"},
			{ID: "infra", Class: "infrastructure"},
		},
	}

	gen := NewGenerator()
	result, err := gen.GenerateSecrets(m, map[string]string{})
	if err != nil {
		t.Fatalf("GenerateSecrets error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected no generated secrets, got %d", len(result))
	}
}

func TestGenerator_GenerateSecrets_SkipsExisting(t *testing.T) {
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{ID: "token", Class: "per_install_generated"},
		},
	}

	gen := NewGenerator()
	existing := map[string]string{"token": "already-set"}
	result, err := gen.GenerateSecrets(m, existing)
	if err != nil {
		t.Fatalf("GenerateSecrets error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected no generated secrets for existing value, got %d", len(result))
	}
}

func TestGenerator_GenerateSecrets_GeneratesForPerInstallGenerated(t *testing.T) {
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{ID: "token", Class: "per_install_generated"},
		},
	}

	gen := NewGenerator()
	result, err := gen.GenerateSecrets(m, map[string]string{})
	if err != nil {
		t.Fatalf("GenerateSecrets error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 generated secret, got %d", len(result))
	}
	if result["token"] == "" {
		t.Fatal("expected non-empty generated token")
	}
	if len(result["token"]) != 32 {
		t.Fatalf("expected 32-char token, got %d", len(result["token"]))
	}
}

func TestGenerator_GenerateSecrets_UsesTemplateLength(t *testing.T) {
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{
				ID:    "token",
				Class: "per_install_generated",
				Generator: map[string]any{
					"type":   "random",
					"length": 64,
				},
			},
		},
	}

	gen := NewGenerator()
	result, err := gen.GenerateSecrets(m, map[string]string{})
	if err != nil {
		t.Fatalf("GenerateSecrets error: %v", err)
	}
	if len(result["token"]) != 64 {
		t.Fatalf("expected 64-char token, got %d", len(result["token"]))
	}
}

func TestGenerator_GenerateSecrets_UsesCharset(t *testing.T) {
	tests := []struct {
		name    string
		charset string
		check   func(string) bool
	}{
		{"hex", "hex", func(s string) bool {
			for _, c := range s {
				if !strings.ContainsRune("0123456789abcdef", c) {
					return false
				}
			}
			return true
		}},
		{"digits", "digits", func(s string) bool {
			for _, c := range s {
				if c < '0' || c > '9' {
					return false
				}
			}
			return true
		}},
		{"lower", "lower", func(s string) bool {
			for _, c := range s {
				if c < 'a' || c > 'z' {
					return false
				}
			}
			return true
		}},
		{"upper", "upper", func(s string) bool {
			for _, c := range s {
				if c < 'A' || c > 'Z' {
					return false
				}
			}
			return true
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := &manifest.Manifest{
				Secrets: []manifest.Secret{
					{
						ID:    "token",
						Class: "per_install_generated",
						Generator: map[string]any{
							"type":    "random",
							"length":  20,
							"charset": tc.charset,
						},
					},
				},
			}

			gen := NewGenerator()
			result, err := gen.GenerateSecrets(m, map[string]string{})
			if err != nil {
				t.Fatalf("GenerateSecrets error: %v", err)
			}
			if !tc.check(result["token"]) {
				t.Fatalf("token %q doesn't match charset %s", result["token"], tc.charset)
			}
		})
	}
}

func TestGenerator_GenerateSecrets_CustomCharset(t *testing.T) {
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{
				ID:    "token",
				Class: "per_install_generated",
				Generator: map[string]any{
					"type":    "random",
					"length":  10,
					"charset": "abc",
				},
			},
		},
	}

	gen := NewGenerator()
	result, err := gen.GenerateSecrets(m, map[string]string{})
	if err != nil {
		t.Fatalf("GenerateSecrets error: %v", err)
	}
	for _, c := range result["token"] {
		if !strings.ContainsRune("abc", c) {
			t.Fatalf("token %q contains char not in charset 'abc'", result["token"])
		}
	}
}

func TestGenerator_GenerateSecrets_UnsupportedType(t *testing.T) {
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{
				ID:    "token",
				Class: "per_install_generated",
				Generator: map[string]any{
					"type": "unsupported",
				},
			},
		},
	}

	gen := NewGenerator()
	_, err := gen.GenerateSecrets(m, map[string]string{})
	if err == nil {
		t.Fatal("expected error for unsupported generator type")
	}
	if !strings.Contains(err.Error(), "unsupported generator type") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerator_GenerateSecrets_InvalidLength(t *testing.T) {
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{
				ID:    "token",
				Class: "per_install_generated",
				Generator: map[string]any{
					"type":   "random",
					"length": -1,
				},
			},
		},
	}

	gen := NewGenerator()
	_, err := gen.GenerateSecrets(m, map[string]string{})
	if err == nil {
		t.Fatal("expected error for invalid length")
	}
}

func TestGenerator_GenerateSecrets_MultipleSecrets(t *testing.T) {
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{ID: "token1", Class: "per_install_generated"},
			{ID: "user", Class: "user_prompt"},
			{ID: "token2", Class: "per_install_generated"},
			{ID: "existing", Class: "per_install_generated"},
		},
	}

	gen := NewGenerator()
	existing := map[string]string{"existing": "already-here"}
	result, err := gen.GenerateSecrets(m, existing)
	if err != nil {
		t.Fatalf("GenerateSecrets error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 generated secrets, got %d", len(result))
	}
	if _, ok := result["token1"]; !ok {
		t.Fatal("expected token1 to be generated")
	}
	if _, ok := result["token2"]; !ok {
		t.Fatal("expected token2 to be generated")
	}
	if _, ok := result["user"]; ok {
		t.Fatal("user should not be generated (user_prompt class)")
	}
	if _, ok := result["existing"]; ok {
		t.Fatal("existing should not be generated (already has value)")
	}
}
