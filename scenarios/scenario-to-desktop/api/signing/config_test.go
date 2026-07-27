package signing

import (
	"os"
	"path/filepath"
	"scenario-to-desktop-api/signing/types"
	"testing"
	"time"
)

func TestSigningConfigDefaultsRepresentSafeDisabledPosture(t *testing.T) {
	config := NewSigningConfig()
	if config.SchemaVersion != types.SchemaVersion || config.Enabled {
		t.Fatalf("signing config = %#v", config)
	}
	windows := NewDefaultWindowsConfig()
	if windows.CertificateSource != types.CertSourceFile || windows.SignAlgorithm != types.SignAlgorithmSHA256 || windows.DualSign {
		t.Fatalf("windows defaults = %#v", windows)
	}
	mac := NewDefaultMacOSConfig()
	if !mac.HardenedRuntime || !mac.GatekeeperAssess || mac.Notarize {
		t.Fatalf("mac defaults = %#v", mac)
	}
	if linux := NewDefaultLinuxConfig(); linux == nil {
		t.Fatal("linux defaults must be allocated")
	}
	result := NewValidationResult()
	if !result.Valid || result.Platforms == nil || result.Errors == nil || result.Warnings == nil {
		t.Fatalf("validation result = %#v", result)
	}
}

func TestCertificateExpiryThresholds(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	if !IsCertificateExpired(now.Add(-time.Hour), now) || IsCertificateExpired(now.Add(time.Hour), now) {
		t.Fatal("expiry classification is incorrect")
	}
	if !IsCertificateExpiringWarning(now.AddDate(0, 0, CertExpiryWarningDays-1), now) || IsCertificateExpiringWarning(now.AddDate(0, 0, CertExpiryWarningDays+1), now) {
		t.Fatal("warning threshold is incorrect")
	}
	if !IsCertificateExpiringCritical(now.AddDate(0, 0, CertExpiryCriticalDays-1), now) || IsCertificateExpiringCritical(now.AddDate(0, 0, CertExpiryCriticalDays+1), now) {
		t.Fatal("critical threshold is incorrect")
	}
	if got := CalculateDaysToExpiry(now.Add(49*time.Hour), now); got != 2 {
		t.Fatalf("days to expiry = %d, want 2", got)
	}
}

func TestRealSigningEnvironmentAndFilesystemAdapters(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "nested", "signing.json")
	fs := NewRealFileSystem()
	if err := fs.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := fs.WriteFile(path, []byte("config"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !fs.Exists(path) {
		t.Fatal("written config was not found")
	}
	data, err := fs.ReadFile(path)
	if err != nil || string(data) != "config" {
		t.Fatalf("read config = %q, %v", data, err)
	}
	info, err := fs.Stat(path)
	if err != nil || info.Size() != int64(len(data)) {
		t.Fatalf("stat = %#v, %v", info, err)
	}
	if err := fs.Remove(path); err != nil {
		t.Fatal(err)
	}
	if fs.Exists(path) {
		t.Fatal("removed config still exists")
	}

	t.Setenv("SCENARIO_TO_DESKTOP_SIGNING_TEST", "available")
	env := NewRealEnvironmentReader()
	if got := env.GetEnv("SCENARIO_TO_DESKTOP_SIGNING_TEST"); got != "available" {
		t.Fatalf("GetEnv = %q", got)
	}
	if got, ok := env.LookupEnv("SCENARIO_TO_DESKTOP_SIGNING_TEST"); !ok || got != "available" {
		t.Fatalf("LookupEnv = %q, %v", got, ok)
	}
	if _, ok := env.LookupEnv("SCENARIO_TO_DESKTOP_MISSING"); ok {
		t.Fatal("missing variable unexpectedly found")
	}
	if now := NewRealTimeProvider().Now(); now.Before(time.Now().Add(-time.Minute)) || now.After(time.Now().Add(time.Minute)) {
		t.Fatalf("unexpected current time: %v", now)
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatal(err)
	}
}
