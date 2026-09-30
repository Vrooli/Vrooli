package config

import (
	"context"
	"errors"
	"testing"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

type derivedBootstrapAuthority struct{ values map[string]string }

func (a *derivedBootstrapAuthority) Resolve(_ credentialauthority.Identity, field string) (string, error) {
	value := a.values[field]
	if value == "" {
		return "", errors.New("missing")
	}
	return value, nil
}

func (a *derivedBootstrapAuthority) Require(id credentialauthority.Identity, field string) (string, error) {
	return a.Resolve(id, field)
}

func (a *derivedBootstrapAuthority) Put(_ credentialauthority.Identity, field, value string) error {
	a.values[field] = value
	return nil
}

func (*derivedBootstrapAuthority) Delete(credentialauthority.Identity, string) error { return nil }
func (*derivedBootstrapAuthority) Status(credentialauthority.Identity, string) credentialauthority.Status {
	return credentialauthority.Status{}
}
func (*derivedBootstrapAuthority) Provider() string { return "test" }

type derivedBootstrapAPI struct{ calls int }

func (a *derivedBootstrapAPI) VerifyToken(context.Context, string) error { a.calls++; return nil }
func (a *derivedBootstrapAPI) ListAccounts(context.Context, string) ([]CloudflareAccount, error) {
	a.calls++
	return []CloudflareAccount{{ID: "account-1"}}, nil
}

func (a *derivedBootstrapAPI) ListTunnels(context.Context, string, string) ([]CloudflareTunnel, error) {
	a.calls++
	return []CloudflareTunnel{{ID: "tunnel-1", Name: "vrooli"}}, nil
}

func (*derivedBootstrapAPI) CreateTunnel(context.Context, string, string, string) (CloudflareTunnel, error) {
	return CloudflareTunnel{}, errors.New("unexpected create")
}

func (a *derivedBootstrapAPI) ConnectorToken(context.Context, string, string, string) (string, error) {
	a.calls++
	return "connector-1", nil
}

func TestBootstrapConfiguredCloudflareIsOwnerLifecycleSeam(t *testing.T) {
	authority := &derivedBootstrapAuthority{values: map[string]string{"cloudflare-api-token": "api-token"}}
	api := &derivedBootstrapAPI{}
	svc := NewService(Deps{BootstrapAPI: api, BootstrapAuthority: authority})

	result, err := svc.(DerivedCredentialBootstrapper).BootstrapConfiguredCloudflare(context.Background())
	if err != nil {
		t.Fatalf("BootstrapConfiguredCloudflare() error = %v", err)
	}
	if !result.Written || result.AccountID != "account-1" || result.TunnelID != "tunnel-1" {
		t.Fatalf("bootstrap result = %+v", result)
	}
	for field, want := range map[string]string{
		"cloudflare-api-token":       "api-token",
		"cloudflare-account-id":      "account-1",
		"cloudflare-tunnel-id":       "tunnel-1",
		"cloudflare-connector-token": "connector-1",
	} {
		if authority.values[field] != want {
			t.Fatalf("authority[%q] = %q, want %q", field, authority.values[field], want)
		}
	}
}

func TestBootstrapConfiguredCloudflareMissingOperatorCredentialIsNoop(t *testing.T) {
	api := &derivedBootstrapAPI{}
	svc := NewService(Deps{BootstrapAPI: api, BootstrapAuthority: &derivedBootstrapAuthority{values: map[string]string{}}})
	result, err := svc.(DerivedCredentialBootstrapper).BootstrapConfiguredCloudflare(context.Background())
	if err != nil {
		t.Fatalf("BootstrapConfiguredCloudflare() error = %v", err)
	}
	if result.Written || api.calls != 0 {
		t.Fatalf("missing operator credential caused bootstrap: result=%+v calls=%d", result, api.calls)
	}
}
