package intelligence

import (
	"context"
	"errors"
	"testing"

	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

type credentialAuthorityAPIKeyClient struct {
	value string
	err   error
}

func (c credentialAuthorityAPIKeyClient) Provision(context.Context, credentialclient.ProvisionRequest) (credentialclient.ProvisionResponse, error) {
	return credentialclient.ProvisionResponse{}, nil
}

func (c credentialAuthorityAPIKeyClient) Resolve(context.Context, string, string) (string, error) {
	return c.value, c.err
}

func (c credentialAuthorityAPIKeyClient) Delete(context.Context, string, string) error { return nil }
func (c credentialAuthorityAPIKeyClient) Status(context.Context, string, string) (credentialclient.CredentialStatus, error) {
	return credentialclient.CredentialStatus{}, nil
}
func (c credentialAuthorityAPIKeyClient) List(context.Context) ([]credentialclient.CredentialRef, error) {
	return nil, nil
}
func (c credentialAuthorityAPIKeyClient) Doctor(context.Context) (credentialclient.DoctorResponse, error) {
	return credentialclient.DoctorResponse{}, nil
}
func (c credentialAuthorityAPIKeyClient) KeyringInspect(context.Context, string) (credentialclient.KeyringReport, error) {
	return credentialclient.KeyringReport{}, nil
}
func (c credentialAuthorityAPIKeyClient) KeyringRepair(context.Context, string) (credentialclient.KeyringReport, error) {
	return credentialclient.KeyringReport{}, nil
}
func (c credentialAuthorityAPIKeyClient) RecoveryExport(context.Context, credentialclient.RecoveryExportRequest) (credentialclient.RecoveryExportResponse, error) {
	return credentialclient.RecoveryExportResponse{}, nil
}
func (c credentialAuthorityAPIKeyClient) RecoveryVerify(context.Context, credentialclient.RecoveryVerifyRequest) (credentialclient.RecoveryVerifyResponse, error) {
	return credentialclient.RecoveryVerifyResponse{}, nil
}
func (c credentialAuthorityAPIKeyClient) RecoveryRestore(context.Context, credentialclient.RecoveryRestoreRequest) error {
	return nil
}
func (c credentialAuthorityAPIKeyClient) StoreStatus(context.Context) (credentialclient.StoreStatus, error) {
	return credentialclient.StoreStatus{}, nil
}

func TestCredentialAuthorityAPIKeyServiceResolvesSharedProviderCredential(t *testing.T) {
	service := CredentialAuthorityAPIKeyService{Client: credentialAuthorityAPIKeyClient{value: " sk-or-shared "}}
	value, err := service.Get(context.Background(), "openrouter")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if value != "sk-or-shared" {
		t.Fatalf("value = %q, want trimmed shared credential", value)
	}
}

func TestCredentialAuthorityAPIKeyServiceRefusesOtherProviderStores(t *testing.T) {
	service := CredentialAuthorityAPIKeyService{Client: credentialAuthorityAPIKeyClient{value: "secret"}}
	if _, err := service.Get(context.Background(), "openai"); err == nil {
		t.Fatal("expected non-OpenRouter provider to be refused")
	}
}

func TestCredentialAuthorityAPIKeyServicePropagatesAuthorityFailure(t *testing.T) {
	want := errors.New("credential store unavailable")
	service := CredentialAuthorityAPIKeyService{Client: credentialAuthorityAPIKeyClient{err: want}}
	if _, err := service.Get(context.Background(), "openrouter"); !errors.Is(err, want) {
		t.Fatalf("error = %v, want authority error %v", err, want)
	}
}
