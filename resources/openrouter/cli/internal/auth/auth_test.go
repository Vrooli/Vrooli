package auth

import (
	"context"
	"errors"
	"testing"
)

func TestResolveReadsVaultSecretsExport(t *testing.T) {
	resolver := Resolver{
		LookupEnv:    func(string) (string, bool) { return "", false },
		LookPathFunc: func(string) (string, error) { return "/bin/resource-vault", nil },
		RunCommand: func(_ context.Context, _ string, args ...string) (string, error) {
			if len(args) >= 2 && args[0] == "content" {
				return "", errors.New("not found")
			}
			if len(args) == 3 && args[0] == "secrets" && args[1] == "export" && args[2] == "openrouter" {
				return `export OPENROUTER_API_KEY="sk-or-v1-from-vault-export"`, nil
			}
			return "", errors.New("unexpected command")
		},
	}

	creds, err := resolver.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if creds.APIKey != "sk-or-v1-from-vault-export" {
		t.Fatalf("APIKey = %q", creds.APIKey)
	}
	if creds.Source != "vault-export-openrouter" {
		t.Fatalf("Source = %q", creds.Source)
	}
}

func TestParseExportedKey(t *testing.T) {
	out := "export OTHER=1\nexport OPENROUTER_API_KEY='sk-or-v1-abc'\n"
	if got := parseExportedKey(out, APIKeyEnv); got != "sk-or-v1-abc" {
		t.Fatalf("parseExportedKey = %q", got)
	}
}
