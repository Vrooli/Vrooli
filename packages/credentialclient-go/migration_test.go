package credentialclient

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

type failingRecoveryClient struct {
	Client
	err error
}

func (c failingRecoveryClient) RecoveryExport(context.Context, RecoveryExportRequest) (RecoveryExportResponse, error) {
	return RecoveryExportResponse{}, c.err
}

func migrationClient(t *testing.T, descriptors []CredentialRef) Client {
	t.Helper()
	authority, err := credentialauthority.NewAuthority(&testStore{})
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewInProcess(InProcessOptions{
		Authority:   authority,
		StateDir:    t.TempDir(),
		Descriptors: func() ([]CredentialRef, error) { return descriptors, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestMigrateLegacyJSONReportsUnmappedKeysWithoutDeletingSource(t *testing.T) {
	path := "legacy.json"
	content, err := json.Marshal(map[string]string{"OPENROUTER_API_KEY": "value", "UNKNOWN": "orphan"})
	if err != nil {
		t.Fatal(err)
	}
	removed := false
	report, err := migrateLegacyJSONData(context.Background(), migrationClient(t, []CredentialRef{{Env: "OPENROUTER_API_KEY", LogicalID: "vrooli/openrouter", Field: "api-key", Required: true}}), path, filepath.Join(t.TempDir(), "recovery.bundle"), "passphrase", content, func(string) error {
		removed = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Unmapped) != 1 || report.Unmapped[0] != "UNKNOWN" {
		t.Fatalf("report = %+v, want UNKNOWN to remain visible", report)
	}
	if report.Deleted {
		t.Fatal("migration deleted a source containing an unmapped key")
	}
	if removed {
		t.Fatal("migration attempted to remove a source containing an unmapped key")
	}
}

func TestMigrateLegacyJSONDeletesOnlyAfterVerifiedRecoveryExport(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "legacy.json")
	bundle := filepath.Join(root, "recovery.bundle")
	removedPath := ""
	report, err := migrateLegacyJSONData(context.Background(), migrationClient(t, []CredentialRef{{Env: "OPENROUTER_API_KEY", LogicalID: "vrooli/openrouter", Field: "api-key", Required: true}}), path, bundle, "passphrase", []byte(`{"OPENROUTER_API_KEY":"value"}`), func(path string) error {
		removedPath = path
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Deleted || report.BundlePath != bundle {
		t.Fatalf("report = %+v, want verified deletion and bundle path", report)
	}
	if removedPath != path {
		t.Fatalf("removed source = %q, want %q", removedPath, path)
	}
	if _, err := os.Stat(bundle); err != nil {
		t.Fatalf("recovery bundle missing: %v", err)
	}
}

func TestMigrateLegacyJSONRefusesDeletionWhenRecoveryExportFails(t *testing.T) {
	path := "legacy.json"
	removed := false
	client := failingRecoveryClient{Client: migrationClient(t, []CredentialRef{{Env: "OPENROUTER_API_KEY", LogicalID: "vrooli/openrouter", Field: "api-key", Required: true}}), err: errors.New("recovery backend unavailable")}
	_, err := migrateLegacyJSONData(context.Background(), client, path, filepath.Join(t.TempDir(), "recovery.bundle"), "passphrase", []byte(`{"OPENROUTER_API_KEY":"value"}`), func(string) error {
		removed = true
		return nil
	})
	if err == nil {
		t.Fatal("migration succeeded without a covering recovery bundle")
	}
	if removed {
		t.Fatal("migration removed the source after recovery export failed")
	}
}
