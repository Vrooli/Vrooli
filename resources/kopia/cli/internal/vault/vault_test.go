package vault_test

import (
	"context"
	"errors"
	"testing"

	"resource-kopia/cli/internal/vault"
	"resource-kopia/cli/internal/vault/mocks"
)

func TestS3CredentialsRoundTrip(t *testing.T) {
	v := mocks.NewFakeVault()
	want := vault.S3Credentials{AccessKeyID: "AKIA", SecretAccessKey: "shhh"}
	if err := vault.PutS3Credentials(context.Background(), v, "offsite", want); err != nil {
		t.Fatalf("PutS3Credentials error = %v", err)
	}
	got, found, err := vault.S3CredentialsFor(context.Background(), v, "offsite")
	if err != nil || !found {
		t.Fatalf("S3CredentialsFor found=%v err=%v", found, err)
	}
	if got != want {
		t.Fatalf("round-trip mismatch: got %+v want %+v", got, want)
	}
}

func TestCLIVaultGetSecretClassifiesMissingOnly(t *testing.T) {
	ctx := context.Background()
	c := &vault.CLIVault{
		Command:  "resource-vault",
		LookPath: func(string) (string, error) { return "/bin/resource-vault", nil },
		Run: func(context.Context, string, ...string) (string, error) {
			return "", &vault.CLIError{Command: "resource-vault", Stderr: "No value found at secret/x", Err: errors.New("exit status 2")}
		},
	}
	if _, found, err := c.GetSecret(ctx, "secret/x", "value"); err != nil || found {
		t.Fatalf("missing secret found=%v err=%v, want found=false err=nil", found, err)
	}

	c.Run = func(context.Context, string, ...string) (string, error) {
		return "", &vault.CLIError{Command: "resource-vault", Stderr: "Cannot connect to Docker daemon", Err: errors.New("exit status 1")}
	}
	if _, _, err := c.GetSecret(ctx, "secret/x", "value"); err == nil {
		t.Fatal("expected docker/vault outage to remain a hard error")
	}
}
