package operatorsession

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLocalSessionRoundTripAndExpiry(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	private, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	public, err := PublicKey(private)
	if err != nil {
		t.Fatal(err)
	}
	token, err := Mint(private, "enrollment-1", "operator-1", []string{"vrooli-bridge:read"}, now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := Verify(public, token, now.Add(30*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if claims.EnrollmentReference != "enrollment-1" || claims.OperatorID != "operator-1" {
		t.Fatalf("claims = %#v", claims)
	}
	if _, err := Verify(public, token, now.Add(61*time.Second)); !errors.Is(err, ErrLocalSessionExpired) {
		t.Fatalf("expired token error = %v", err)
	}
}

func TestContainsAll(t *testing.T) {
	if !ContainsAll([]string{"read", "write"}, []string{"read"}) {
		t.Fatal("read should be within the ceiling")
	}
	if ContainsAll([]string{"read"}, []string{"write"}) {
		t.Fatal("write should exceed the ceiling")
	}
}

func TestFileStorePersistsEnrollmentButNeverMintedSession(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	private, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	enrollment := Enrollment{OperatorID: "operator-1", IdentityProvider: "scenario-authenticator", Mode: ModePersonal, Reference: "enrollment-1", EnrolledAt: time.Unix(1_700_000_000, 0).UTC(), ScopeCeiling: []string{"vrooli-bridge:read"}}
	if err := store.Save(private, enrollment); err != nil {
		t.Fatal(err)
	}
	resolution, err := (LocalResolver{Store: store, Now: func() time.Time { return enrollment.EnrolledAt.Add(time.Minute) }}).Resolve()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(resolution.Token, "OS1.") {
		t.Fatalf("token = %q", resolution.Token)
	}
	metadata, err := os.ReadFile(filepath.Join(dir, "enrollment.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(metadata), resolution.Token) || strings.Contains(string(metadata), "access_token") || strings.Contains(string(metadata), "refresh_token") {
		t.Fatalf("enrollment metadata contains bearer material: %s", metadata)
	}
	loadedKey, loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if string(loadedKey) != string(private) || loaded.Reference != enrollment.Reference {
		t.Fatalf("stored enrollment changed: key=%v enrollment=%#v", string(loadedKey) == string(private), loaded)
	}
}
