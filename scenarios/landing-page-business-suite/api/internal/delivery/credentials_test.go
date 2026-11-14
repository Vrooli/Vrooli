package delivery

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

func TestDefaultBucketForProducesS3SafeName(t *testing.T) {
	cases := map[string]string{
		"business_suite":              "business-suite-downloads",
		"landing-page-business-suite": "landing-page-business-suite-downloads",
		"  Mixed Case.Key  ":          "mixed-case-key-downloads",
		"":                            "vrooli-downloads",
	}
	for input, want := range cases {
		if got := DefaultBucketFor(input); got != want {
			t.Errorf("DefaultBucketFor(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestStorageSettingsWithDefaultsFillsBlankFields(t *testing.T) {
	resolved := storageSettingsWithDefaults(StorageSettings{BundleKey: "business_suite"}, "business_suite")

	if resolved.Provider != "s3" {
		t.Errorf("provider = %q, want s3", resolved.Provider)
	}
	if resolved.Bucket != "business-suite-downloads" {
		t.Errorf("bucket = %q, want default", resolved.Bucket)
	}
	if resolved.Region != DefaultRegion {
		t.Errorf("region = %q, want %q", resolved.Region, DefaultRegion)
	}
	if resolved.DefaultPrefix != DefaultObjectPrefix {
		t.Errorf("prefix = %q, want %q", resolved.DefaultPrefix, DefaultObjectPrefix)
	}
	if resolved.SignedURLTTLSeconds != DefaultSignedURLTTLSeconds {
		t.Errorf("ttl = %d, want %d", resolved.SignedURLTTLSeconds, DefaultSignedURLTTLSeconds)
	}
}

func TestStorageSettingsWithDefaultsKeepsOperatorValues(t *testing.T) {
	operator := StorageSettings{Provider: "s3", Bucket: "mine", Region: "eu-west-1", DefaultPrefix: "releases", SignedURLTTLSeconds: 120}
	resolved := storageSettingsWithDefaults(operator, "business_suite")
	if resolved != operator {
		t.Fatalf("resolved = %#v, want %#v", resolved, operator)
	}
}

func TestApplyCredentialUpdatesWritesThroughAndClears(t *testing.T) {
	stored := map[string]string{}
	var deleted []string
	service := NewService(nil).WithCredentialIO(CredentialIO{
		Write: func(_ context.Context, field, value string) error {
			stored[field] = value
			return nil
		},
		Delete: func(_ context.Context, field string) error {
			deleted = append(deleted, field)
			delete(stored, field)
			return nil
		},
	})

	accessKey := "AKIAEXAMPLE"
	secretKey := "secret-value"
	clearToken := ""
	if err := service.applyCredentialUpdates(context.Background(), StorageSettingsUpdate{
		AccessKeyID:     &accessKey,
		SecretAccessKey: &secretKey,
		SessionToken:    &clearToken,
	}); err != nil {
		t.Fatalf("applyCredentialUpdates: %v", err)
	}
	if stored[CredentialFieldAccessKeyID] != accessKey || stored[CredentialFieldSecretAccessKey] != secretKey {
		t.Fatalf("stored = %#v", stored)
	}
	if len(deleted) != 1 || deleted[0] != CredentialFieldSessionToken {
		t.Fatalf("deleted = %#v, want session token cleared", deleted)
	}
}

func TestApplyCredentialUpdatesRefusesWithoutAuthorityIO(t *testing.T) {
	value := "AKIAEXAMPLE"
	err := NewService(nil).applyCredentialUpdates(context.Background(), StorageSettingsUpdate{AccessKeyID: &value})
	if err == nil {
		t.Fatal("expected refusal when no credential authority boundary is installed")
	}
}

func TestCredentialPresentDistinguishesAbsence(t *testing.T) {
	present := NewService(nil).WithCredentialIO(CredentialIO{
		Read: func(context.Context, string) (string, error) { return "AKIAEXAMPLE", nil },
	})
	if !present.credentialPresent(context.Background(), CredentialFieldAccessKeyID) {
		t.Error("expected present credential to report true")
	}

	absent := NewService(nil).WithCredentialIO(CredentialIO{
		Read: func(context.Context, string) (string, error) { return "", credentialauthority.ErrUnconfigured },
	})
	if absent.credentialPresent(context.Background(), CredentialFieldAccessKeyID) {
		t.Error("expected unconfigured credential to report false")
	}

	if NewService(nil).credentialPresent(context.Background(), CredentialFieldAccessKeyID) {
		t.Error("expected nil credential IO to report false")
	}
}

func TestSettingsSnapshotReportsPresenceAndDefaults(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT id, bundle_key, provider, bucket, region, endpoint, force_path_style, default_prefix,").
		WithArgs("business_suite").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "bundle_key", "provider", "bucket", "region", "endpoint", "force_path_style",
			"default_prefix", "signed_url_ttl_seconds", "public_base_url", "created_at", "updated_at",
		}))

	service := NewService(db).WithCredentialIO(CredentialIO{
		Read: func(_ context.Context, field string) (string, error) {
			if field == CredentialFieldAccessKeyID || field == CredentialFieldSecretAccessKey {
				return "value", nil
			}
			return "", credentialauthority.ErrUnconfigured
		},
	})

	snapshot, err := service.SettingsSnapshot(context.Background(), "business_suite")
	if err != nil {
		t.Fatalf("SettingsSnapshot: %v", err)
	}
	if !snapshot.AccessKeyIDSet || !snapshot.SecretAccessKeySet {
		t.Errorf("expected access and secret presence flags, got %#v", snapshot)
	}
	if snapshot.SessionTokenSet {
		t.Errorf("expected session token to be absent, got %#v", snapshot)
	}
	if snapshot.Bucket != "business-suite-downloads" || snapshot.Region != DefaultRegion || snapshot.DefaultPrefix != DefaultObjectPrefix {
		t.Errorf("expected defaults in snapshot, got %#v", snapshot)
	}
	if snapshot.SettingsRowAvailable {
		t.Errorf("expected SettingsRowAvailable=false, got true")
	}
	if !snapshot.CredentialsFromAuthority {
		t.Errorf("expected CredentialsFromAuthority=true")
	}
}

func TestSaveSettingsWritesCredentialsToAuthorityNotTheRow(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT id, bundle_key, provider, bucket, region, endpoint, force_path_style, default_prefix,").
		WithArgs("business_suite").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "bundle_key", "provider", "bucket", "region", "endpoint", "force_path_style",
			"default_prefix", "signed_url_ttl_seconds", "public_base_url", "created_at", "updated_at",
		}))
	mock.ExpectExec("INSERT INTO download_storage_settings").
		WithArgs("business_suite", "s3", "business-suite-downloads", "us-east-1", nil, false, "artifacts", DefaultSignedURLTTLSeconds, nil).
		WillReturnResult(sqlmock.NewResult(1, 1))
	// SaveSettings returns a fresh snapshot after persisting.
	now := time.Now()
	mock.ExpectQuery("SELECT id, bundle_key, provider, bucket, region, endpoint, force_path_style, default_prefix,").
		WithArgs("business_suite").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "bundle_key", "provider", "bucket", "region", "endpoint", "force_path_style",
			"default_prefix", "signed_url_ttl_seconds", "public_base_url", "created_at", "updated_at",
		}).AddRow(1, "business_suite", "s3", "business-suite-downloads", "us-east-1", nil, false, "artifacts", DefaultSignedURLTTLSeconds, nil, now, now))

	var written []string
	service := NewService(db).WithCredentialIO(CredentialIO{
		Write: func(_ context.Context, field, value string) error {
			written = append(written, field+"="+value)
			return nil
		},
		Delete: func(context.Context, string) error { return nil },
	})

	accessKey := "AKIAEXAMPLE"
	secretKey := "secret-value"
	if _, err := service.SaveSettings(context.Background(), "business_suite", StorageSettingsUpdate{
		AccessKeyID:     &accessKey,
		SecretAccessKey: &secretKey,
	}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	if len(written) != 2 {
		t.Fatalf("written = %#v, want both credentials written through the authority", written)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestSaveSettingsRefusesCredentialWriteWithoutAuthority(t *testing.T) {
	value := "AKIAEXAMPLE"
	if _, err := NewService(nil).SaveSettings(context.Background(), "business_suite", StorageSettingsUpdate{AccessKeyID: &value}); err == nil {
		t.Fatal("expected credential write refusal when no authority boundary is installed")
	}
}
