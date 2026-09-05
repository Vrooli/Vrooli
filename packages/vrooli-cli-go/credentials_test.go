package vroolicli

import (
	"context"
	"testing"
)

func TestCredentialsStatusDecodesWithoutSecretFields(t *testing.T) {
	runner := &stubRunner{responses: []stubResponse{{output: []byte(`{"checked_at":"2026-08-26T00:00:00Z","configured":true,"field":"api-key","identity":"vrooli/openai","provider":"encrypted-file","provider_state":"available"}`)}}}
	client := New(WithRunner(runner))
	resp, err := client.CredentialsStatus(context.Background(), "vrooli/openai", "api-key")
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetConfigured() || resp.GetIdentity() != "vrooli/openai" {
		t.Fatalf("unexpected status: %+v", resp)
	}
	if len(runner.calls) != 1 || runner.calls[0].args[len(runner.calls[0].args)-1] != "json" {
		t.Fatalf("calls = %+v", runner.calls)
	}
}

func TestBreakGlassStatusDecodesMetadataOnly(t *testing.T) {
	runner := &stubRunner{responses: []stubResponse{{output: []byte(`{"account_id":"operator","audience":"vrooli:uninstall","complete":true,"metadata":true,"provisioned_at":"1700000000","public":true,"scopes":["vrooli:uninstall"],"wrapped_private":true}`)}}}
	client := New(WithRunner(runner))
	resp, err := client.BreakGlassStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetComplete() || resp.GetAccountId() != "operator" || len(resp.GetScopes()) != 1 {
		t.Fatalf("unexpected status: %+v", resp)
	}
}
