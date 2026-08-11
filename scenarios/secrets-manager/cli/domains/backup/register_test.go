package backup

import (
	"context"
	"testing"

	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

type backupClient struct {
	statuses map[string]bool
}

func (c backupClient) Provision(context.Context, credentialclient.ProvisionRequest) (credentialclient.ProvisionResponse, error) {
	return credentialclient.ProvisionResponse{}, nil
}
func (c backupClient) Delete(context.Context, string, string) error { return nil }
func (c backupClient) Status(_ context.Context, identity, field string) (credentialclient.CredentialStatus, error) {
	return credentialclient.CredentialStatus{Identity: identity, Field: field, Configured: c.statuses[identity+"\x00"+field]}, nil
}

func (c backupClient) List(context.Context) ([]credentialclient.CredentialRef, error) {
	return nil, nil
}

func (c backupClient) Doctor(context.Context) (credentialclient.DoctorResponse, error) {
	return credentialclient.DoctorResponse{}, nil
}

func (c backupClient) KeyringInspect(context.Context, string) (credentialclient.KeyringReport, error) {
	return credentialclient.KeyringReport{}, nil
}

func (c backupClient) KeyringRepair(context.Context, string) (credentialclient.KeyringReport, error) {
	return credentialclient.KeyringReport{}, nil
}

func (c backupClient) RecoveryExport(context.Context, credentialclient.RecoveryExportRequest) (credentialclient.RecoveryExportResponse, error) {
	return credentialclient.RecoveryExportResponse{}, nil
}

func (c backupClient) RecoveryVerify(context.Context, credentialclient.RecoveryVerifyRequest) (credentialclient.RecoveryVerifyResponse, error) {
	return credentialclient.RecoveryVerifyResponse{}, nil
}

func (c backupClient) RecoveryRestore(context.Context, credentialclient.RecoveryRestoreRequest) error {
	return nil
}

func (c backupClient) StoreStatus(context.Context) (credentialclient.StoreStatus, error) {
	return credentialclient.StoreStatus{}, nil
}

func TestConfiguredEntriesSkipsUnconfiguredDeclarationsAndDeduplicates(t *testing.T) {
	entries := []credentialclient.CredentialRef{
		{LogicalID: "vrooli/openrouter", Field: "api-key"},
		{LogicalID: "vrooli/openrouter", Field: "api-key"},
		{LogicalID: "vrooli/postgres", Field: "password"},
	}
	got, err := configuredEntries(context.Background(), backupClient{statuses: map[string]bool{"vrooli/openrouter\x00api-key": true}}, entries)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].LogicalID != "vrooli/openrouter" || got[0].Field != "api-key" {
		t.Fatalf("configured entries = %#v, want one configured credential", got)
	}
}
