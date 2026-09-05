package auth

import (
	"context"
	"testing"
)

func TestResolveAcceptsOnlyEphemeralInjection(t *testing.T) {
	resolver := Resolver{
		LookupEnv: func(name string) (string, bool) { return "sk-or-v1-ephemeral", name == APIKeyEnv },
	}

	creds, err := resolver.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if creds.APIKey != "sk-or-v1-ephemeral" {
		t.Fatalf("APIKey = %q", creds.APIKey)
	}
	if creds.Source != "ephemeral-injection" {
		t.Fatalf("Source = %q", creds.Source)
	}
}

func TestResolveRejectsAbsentInjection(t *testing.T) {
	_, err := (Resolver{LookupEnv: func(string) (string, bool) { return "", false }}).Resolve(context.Background())
	if err != ErrCredentialsNotConfigured {
		t.Fatalf("Resolve error = %v", err)
	}
}
