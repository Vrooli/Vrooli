//go:build linux

package securestore

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
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
