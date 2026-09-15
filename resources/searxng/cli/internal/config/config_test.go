package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyCreatesValidRedactedConfiguration(t *testing.T) {
	dir := t.TempDir()
	report, err := Apply(dir, "http://localhost:8280", "Test SearXNG", "")
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !report.Created || report.Secret != "generated" {
		t.Fatalf("report = %#v", report)
	}
	data, err := os.ReadFile(SettingsPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "${SEARXNG_SECRET_KEY}") {
		t.Fatalf("secret placeholder survived: %s", data)
	}
	doc, _, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	summary := RedactedSummary(doc)
	if summary["secret_key"] != "[redacted]" {
		t.Fatalf("summary leaked secret: %#v", summary)
	}
}

func TestApplyPreservesUnknownSettingsAndExistingSecret(t *testing.T) {
	dir := t.TempDir()
	path := SettingsPath(dir)
	original := "use_default_settings: true\nserver:\n  secret_key: keep-me-private\n  base_url: http://old\ncustom_upstream:\n  retained: yes\nsearch:\n  formats: [html, json]\n"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := Apply(dir, "http://new", "Migrated", "new-secret")
	if err != nil {
		t.Fatal(err)
	}
	if report.Secret != "preserved" || report.BackupPath == "" {
		t.Fatalf("report = %#v", report)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	for _, expected := range []string{"keep-me-private", "custom_upstream:", "retained: yes", "base_url: http://new"} {
		if !strings.Contains(got, expected) {
			t.Fatalf("settings missing %q:\n%s", expected, got)
		}
	}
	if _, err := os.Stat(report.BackupPath); err != nil {
		t.Fatalf("backup missing: %v", err)
	}
}

func TestApplyImportsExistingSettingsWithoutASecret(t *testing.T) {
	dir := t.TempDir()
	path := SettingsPath(dir)
	original := "use_default_settings: true\ncustom_upstream:\n  retained: yes\nsearch:\n  formats: [html, json]\n"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := Apply(dir, "http://localhost:8280", "Migrated", "")
	if err != nil {
		t.Fatal(err)
	}
	if report.Secret != "generated" || report.BackupPath == "" {
		t.Fatalf("report = %#v", report)
	}
	doc, _, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(doc); err != nil {
		t.Fatalf("imported configuration invalid: %v", err)
	}
	if findScalar(doc, "server", "secret_key") == "" {
		t.Fatal("migration did not materialize a session secret")
	}
}

func TestLoadRejectsInvalidOrNonContractConfiguration(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, settingsName), []byte("- not-a-map\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Load(dir); err == nil {
		t.Fatal("Load() error = nil, want invalid mapping error")
	}
	if err := os.WriteFile(filepath.Join(dir, settingsName), []byte("server:\n  secret_key: x\nsearch:\n  formats: [html]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	document, _, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if err := Validate(document); err == nil || !strings.Contains(err.Error(), "include json") {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestApplyRefusesInvalidSettingsWithoutChangingTheOriginal(t *testing.T) {
	dir := t.TempDir()
	path := SettingsPath(dir)
	original := "server:\n  secret_key: existing\nsearch:\n  formats: [html]\n"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(dir, "http://localhost:8280", "Ignored", ""); err == nil {
		t.Fatal("Apply() error = nil, want invalid configuration rejection")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != original {
		t.Fatalf("invalid source changed:\n got %q\nwant %q", got, original)
	}
}

func TestAtomicWriteFailureDoesNotReplaceExistingTarget(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "settings.yml")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := atomicWrite(target, []byte("replacement")); err == nil {
		t.Fatal("atomicWrite() error = nil, want rename failure")
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Fatalf("target changed after atomic write failure: %#v", info)
	}
}
