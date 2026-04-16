package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
		LookPathFunc: func(name string) (string, error) { return "", exec.ErrNotFound },
	}

	creds, err := resolver.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if creds.Source != "env" || creds.APIKey != "env-key" {
		t.Fatalf("Resolve() = %+v", creds)
	}
}

func TestResolverResolveFromVault(t *testing.T) {
	t.Parallel()

	resolver := Resolver{
		LookupEnv:    func(key string) (string, bool) { return "", false },
		LookPathFunc: func(name string) (string, error) { return "/usr/bin/resource-vault", nil },
		RunCommand: func(_ context.Context, _ string, args ...string) (string, error) {
			path := args[3]
			key := args[5]
			if path == defaultSecretPath && key == defaultSecretKey {
				return "vault-key", nil
			}
			return "", fmt.Errorf("unexpected request: %v", args)
		},
	}

	creds, err := resolver.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if creds.Source != "vault" || creds.APIKey != "vault-key" {
		t.Fatalf("Resolve() = %+v", creds)
	}
}

func TestResolverResolveFromFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")
	if err := os.WriteFile(path, []byte("{\"data\":{\"apiKey\":\"file-key\"}}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	resolver := Resolver{
		LookupEnv:           func(key string) (string, bool) { return "", false },
		LookPathFunc:        func(name string) (string, error) { return "", exec.ErrNotFound },
		CredentialsFilePath: path,
	}

	creds, err := resolver.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if creds.Source != "file" || creds.APIKey != "file-key" {
		t.Fatalf("Resolve() = %+v", creds)
	}
}

func TestResolverMissing(t *testing.T) {
	t.Parallel()

	resolver := Resolver{
		LookupEnv:    func(key string) (string, bool) { return "", false },
		LookPathFunc: func(name string) (string, error) { return "", exec.ErrNotFound },
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
