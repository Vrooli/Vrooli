package securestore

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCopyConfigRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config", "copy.json")
	want := CopyConfig{Enabled: true, Sink: filepath.Join(t.TempDir(), "removable"), Interval: 7 * time.Minute}
	if err := WriteCopyConfig(path, want); err != nil {
		t.Fatalf("WriteCopyConfig: %v", err)
	}
	got, err := ReadCopyConfig(path)
	if err != nil {
		t.Fatalf("ReadCopyConfig: %v", err)
	}
	if got != want {
		t.Fatalf("config = %#v, want %#v", got, want)
	}
}

func TestCopyConfigRequiresAuthorityIdentityForS3(t *testing.T) {
	config := CopyConfig{Enabled: true, Sink: "s3://bucket/recovery", Interval: time.Minute, ObjectStoreRegion: "us-east-1"}
	if err := config.Validate(); err == nil || !strings.Contains(err.Error(), "credential identity") {
		t.Fatalf("Validate() = %v, want missing object-store identity", err)
	}
	config.ObjectStoreCredentialID = "vrooli/recovery-store"
	if err := config.Validate(); err != nil {
		t.Fatalf("Validate() with identity = %v", err)
	}
}

func TestCopyConfigMissingIsDisabled(t *testing.T) {
	got, err := ReadCopyConfig(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("ReadCopyConfig missing: %v", err)
	}
	if got.Enabled {
		t.Fatal("missing copy configuration is enabled")
	}
}
