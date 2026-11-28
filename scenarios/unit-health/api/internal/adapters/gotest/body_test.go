package gotest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadTestBodyRedactsSecretsAndCapsExcerpt(t *testing.T) {
	root := t.TempDir()
	source := `package p
import "testing"
func TestBody(t *testing.T) {
    apiKey := "AKIA1234567890ABCDEF"
    t.Log(apiKey)
    t.Log(strings.Repeat("x", 8000))
}
`
	if err := os.WriteFile(filepath.Join(root, "body_test.go"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := ReadTestBody(BodyRequest{Root: root, Workspace: "api", File: "body_test.go", TestID: "TestBody", MaxBytes: 100})
	if err != nil {
		t.Fatal(err)
	}
	if got.BodyBytes <= 100 || len(got.BodyExcerpt) > 100 || got.Redactions != 1 {
		t.Fatalf("bounded body = %+v", got)
	}
	if strings.Contains(got.BodyExcerpt, "AKIA") || !strings.Contains(got.BodyExcerpt, "[redacted]") {
		t.Fatalf("secret was not redacted: %q", got.BodyExcerpt)
	}
}

func TestReadTestBodyRefusesPrivacyPatternAndReportsMissingTest(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "secrets"), 0o700); err != nil {
		t.Fatal(err)
	}
	if got, err := ReadTestBody(BodyRequest{Root: root, Workspace: "api", File: "secrets/test.go", TestID: "TestSecret"}); err != nil || !got.Refused || got.RefusalReason != "privacy_pattern" {
		t.Fatalf("privacy refusal = %+v, err=%v", got, err)
	}
	if err := os.WriteFile(filepath.Join(root, "body_test.go"), []byte("package p\nimport \"testing\"\nfunc TestBody(t *testing.T) {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadTestBody(BodyRequest{Root: root, Workspace: "api", File: "body_test.go", TestID: "TestMissing"}); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("missing test error = %v", err)
	}
}
