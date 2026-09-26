//go:build linux

package securestore

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/testenv"
)

func TestInspectReportsStaleKeyringDaemon(t *testing.T) {
	path := writeKeyring(t, keyringFixture(keyringEntry("1", "Vrooli", "secret", "vrooli.credentials.v1", "k")))
	previous := keyringDaemonStartTime
	keyringDaemonStartTime = func() (time.Time, bool) { return time.Now().Add(-time.Hour), true }
	t.Cleanup(func() { keyringDaemonStartTime = previous })

	report, err := InspectKeyringFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !report.StaleDaemon || report.StaleDaemonDetail == "" || report.StaleDaemonCheck != "checked" {
		t.Fatalf("report = %+v, want stale daemon detail", report)
	}
}

func TestInspectDoesNotReportFreshKeyringAsStale(t *testing.T) {
	path := writeKeyring(t, keyringFixture(keyringEntry("1", "Vrooli", "secret", "vrooli.credentials.v1", "k")))
	previous := keyringDaemonStartTime
	keyringDaemonStartTime = func() (time.Time, bool) { return time.Now().Add(time.Hour), true }
	t.Cleanup(func() { keyringDaemonStartTime = previous })

	report, err := InspectKeyringFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if report.StaleDaemon || report.StaleDaemonCheck != "checked" {
		t.Fatalf("report = %+v, did not expect stale daemon", report)
	}
}

func TestInspectOmitsStaleClaimWhenDaemonTimeCannotBeRead(t *testing.T) {
	path := writeKeyring(t, keyringFixture(keyringEntry("1", "Vrooli", "secret", "vrooli.credentials.v1", "k")))
	previous := keyringDaemonStartTime
	keyringDaemonStartTime = func() (time.Time, bool) { return time.Time{}, false }
	t.Cleanup(func() { keyringDaemonStartTime = previous })

	report, err := InspectKeyringFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if report.StaleDaemon || report.StaleDaemonDetail != "" || report.StaleDaemonCheck != "not-run" {
		t.Fatalf("report = %+v, want no stale claim", report)
	}
}

func TestNativeStorageStrengthClassifiesKeyringFile(t *testing.T) {
	dir := filepath.Join(testenv.RuntimeHome(t), ".local", "share")
	keyrings := filepath.Join(dir, "keyrings")
	if err := os.MkdirAll(keyrings, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(keyrings, "login.keyring")
	if err := os.WriteFile(path, []byte("[keyring]\nsecret=readable\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	strength, caveat := nativeStorageStrength()
	if strength != "unencrypted-keyring" || caveat == "" {
		t.Fatalf("strength/caveat = %q/%q", strength, caveat)
	}
	if err := os.WriteFile(path, []byte{0, 1, 2, 3}, 0o600); err != nil {
		t.Fatal(err)
	}
	strength, _ = nativeStorageStrength()
	if strength != "encrypted-keyring" {
		t.Fatalf("binary strength = %q, want encrypted-keyring", strength)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	strength, caveat = nativeStorageStrength()
	if strength != "" || caveat != "" {
		t.Fatalf("absent strength/caveat = %q/%q, want empty", strength, caveat)
	}
}

func TestRetireKeyringBackupRequiresExactRegularBackup(t *testing.T) {
	dir := t.TempDir()
	backup := filepath.Join(dir, "login.keyring.corrupt-backup")
	if err := os.WriteFile(backup, []byte("backup"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := RetireKeyringBackup(backup); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(backup); !os.IsNotExist(err) {
		t.Fatalf("backup stat error = %v, want not exist", err)
	}

	regular := filepath.Join(dir, "login.keyring")
	if err := os.WriteFile(regular, []byte("live"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := RetireKeyringBackup(regular); err == nil {
		t.Fatal("retired live keyring, want refusal")
	}
}
