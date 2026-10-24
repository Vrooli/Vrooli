package session

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	shared "github.com/vrooli/api-core/operatorsession"
)

func TestLocalEnrollmentRoundTripKeepsBearerMaterialOutOfMetadata(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("VROOLI_OPERATOR_SESSION_DIR", dir)
	private, err := shared.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	enrollment := shared.Enrollment{
		OperatorID:       "operator-1",
		IdentityProvider: "scenario-authenticator",
		Mode:             shared.ModePersonal,
		Reference:        "enrollment-1",
		EnrolledAt:       time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC),
		ScopeCeiling:     []string{"vrooli-bridge:read"},
	}
	if err := saveLocalEnrollment(private, enrollment); err != nil {
		t.Fatal(err)
	}

	loadedPrivate, loadedEnrollment, err := loadLocalEnrollment()
	if err != nil {
		t.Fatal(err)
	}
	if string(loadedPrivate) != string(private) {
		t.Fatalf("private key changed during round trip")
	}
	if loadedEnrollment.Reference != enrollment.Reference || loadedEnrollment.OperatorID != enrollment.OperatorID {
		t.Fatalf("enrollment changed during round trip: %#v", loadedEnrollment)
	}
	token, _, err := mintLocalSession(time.Date(2026, 8, 16, 12, 1, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(token, "OS1.") {
		t.Fatalf("unexpected local session token scheme: %q", token)
	}
	metadata, err := os.ReadFile(filepath.Join(dir, "enrollment.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(metadata), token) || strings.Contains(string(metadata), "OS1.") {
		t.Fatalf("enrollment metadata contains bearer material: %s", metadata)
	}
	if runtime.GOOS != "windows" {
		for _, name := range []string{"private.key", "enrollment.json"} {
			info, statErr := os.Stat(filepath.Join(dir, name))
			if statErr != nil {
				t.Fatal(statErr)
			}
			if got := info.Mode().Perm(); got != 0o600 {
				t.Fatalf("%s permissions = %o, want 600", name, got)
			}
		}
	}
}

func TestLocalEnrollmentRejectsMalformedPrivateKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("VROOLI_OPERATOR_SESSION_DIR", dir)
	if err := os.WriteFile(filepath.Join(dir, "private.key"), make([]byte, ed25519.PrivateKeySize-1), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := loadLocalEnrollment(); err == nil || !strings.Contains(err.Error(), "invalid size") {
		t.Fatalf("load malformed key error = %v", err)
	}
}
