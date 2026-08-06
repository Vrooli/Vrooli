package auth

import (
	"context"
	"errors"
	"testing"
)

func TestResolverResolveFromEnv(t *testing.T) {
	t.Parallel()

	resolver := Resolver{
		LookupEnv: func(key string) (string, bool) {
			if key == APIKeyEnv {
				return "env-key", true
			}
			return "", false
		},
	}

	creds, err := resolver.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if creds.Source != "env" || creds.APIKey != "env-key" {
		t.Fatalf("Resolve() = %+v", creds)
	}
}

func TestResolverDoesNotInvokeVault(t *testing.T) {
	t.Parallel()

	resolver := Resolver{
		LookupEnv: func(key string) (string, bool) { return "", false },
	}

	if _, err := resolver.Resolve(context.Background()); !errors.Is(err, ErrCredentialsNotConfigured) {
		t.Fatalf("Resolve() error = %v, want %v", err, ErrCredentialsNotConfigured)
	}
}

func TestResolverMissing(t *testing.T) {
	t.Parallel()

	resolver := Resolver{
		LookupEnv: func(key string) (string, bool) { return "", false },
	}

	_, err := resolver.Resolve(context.Background())
	if !errors.Is(err, ErrCredentialsNotConfigured) {
		t.Fatalf("Resolve() error = %v, want %v", err, ErrCredentialsNotConfigured)
	}
}

func TestCredentialsRedactedAPIKey(t *testing.T) {
	t.Parallel()

	creds := Credentials{APIKey: "abcdefgh12345678"}
	if got := creds.RedactedAPIKey(); got != "abcd...5678" {
		t.Fatalf("RedactedAPIKey() = %q", got)
	}
}
