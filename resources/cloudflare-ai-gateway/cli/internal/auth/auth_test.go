package auth

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"testing"
)

func TestResolverResolveFromEnv(t *testing.T) {
	t.Parallel()

	resolver := Resolver{
		LookupEnv: func(key string) (string, bool) {
			values := map[string]string{
				AccountIDEnv: "acct-123",
				APITokenEnv:  "token-456",
			}
			value, ok := values[key]
			return value, ok
		},
		LookPathFunc: func(name string) (string, error) {
			return "", exec.ErrNotFound
		},
	}

	creds, err := resolver.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if creds.Source != "env" {
		t.Fatalf("Resolve() source = %q, want env", creds.Source)
	}
	if creds.AccountID != "acct-123" || creds.APIToken != "token-456" {
		t.Fatalf("Resolve() creds = %+v", creds)
	}
}

func TestResolverPrefersVaultWhenAvailable(t *testing.T) {
	t.Parallel()

	resolver := Resolver{
		LookupEnv: func(key string) (string, bool) {
			return "", false
		},
		LookPathFunc: func(name string) (string, error) {
			return "/usr/bin/resource-vault", nil
		},
		RunCommand: func(_ context.Context, _ string, args ...string) (string, error) {
			path := args[3]
			switch path {
			case defaultAccountIDPath:
				return "vault-account", nil
			case defaultAPITokenPath:
				return "vault-token", nil
			default:
				return "", fmt.Errorf("unexpected path %q", path)
			}
		},
	}

	creds, err := resolver.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if creds.Source != "vault" {
		t.Fatalf("Resolve() source = %q, want vault", creds.Source)
	}
}

func TestResolverReturnsConfiguredErrorWhenMissing(t *testing.T) {
	t.Parallel()

	resolver := Resolver{
		LookupEnv: func(key string) (string, bool) { return "", false },
		LookPathFunc: func(name string) (string, error) {
			return "", exec.ErrNotFound
		},
		RunCommand: func(_ context.Context, _ string, args ...string) (string, error) {
			return "", errors.New("should not run")
		},
	}

	_, err := resolver.Resolve(context.Background())
	if !errors.Is(err, ErrCredentialsNotConfigured) {
		t.Fatalf("Resolve() error = %v, want %v", err, ErrCredentialsNotConfigured)
	}
}

func TestCredentialsHelpers(t *testing.T) {
	t.Parallel()

	creds := Credentials{AccountID: "acct", APIToken: "abcdef123456"}
	if !creds.Valid() {
		t.Fatal("Valid() = false, want true")
	}
	if got := creds.AuthorizationHeader(); got != "Bearer abcdef123456" {
		t.Fatalf("AuthorizationHeader() = %q", got)
	}
	if got := creds.RedactedToken(); got != "abcd...3456" {
		t.Fatalf("RedactedToken() = %q", got)
	}
}
