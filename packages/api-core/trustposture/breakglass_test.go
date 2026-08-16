package trustposture

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBreakGlassIsOfflineSignedTimeBoxedAndCeilingBound(t *testing.T) {
	keys, err := GenerateKeyMaterial()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_000, 0).UTC()
	token, err := IssueForAccount(keys.PrivateKey,
		[]string{"vrooli-bridge:*", "agent-manager:read"},
		[]string{"vrooli-bridge:write", "agent-manager:read"},
		BreakGlassClaims{Subject: "owner-1", Audience: "scenario-authenticator:default", Target: "host-a", IssuedAt: now.Unix(), ExpiresAt: now.Add(10 * time.Minute).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := Verify(keys.PublicKey, token, "scenario-authenticator:default", "host-a", now)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "owner-1" || len(claims.Scopes) != 2 {
		t.Fatalf("claims = %+v", claims)
	}
	if _, err := IssueForAccount(keys.PrivateKey, []string{"agent-manager:read"}, []string{"agent-manager:write"}, BreakGlassClaims{Subject: "owner-1", Audience: "a", Target: "host-a", IssuedAt: now.Unix(), ExpiresAt: now.Add(time.Minute).Unix()}); err == nil {
		t.Fatal("scope ceiling bypassed")
	}
	if _, err := Verify(keys.PublicKey, token, "scenario-authenticator:default", "host-a", now.Add(11*time.Minute)); err == nil {
		t.Fatal("expired break-glass credential accepted")
	}
}

func TestBreakGlassPrivateMaterialIsOwnerOnlyAndNotReplaced(t *testing.T) {
	keys, err := GenerateKeyMaterial()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	privatePath := filepath.Join(dir, "nested", "private.key")
	if err := WritePrivate(privatePath, keys.PrivateKey); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(privatePath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("private mode = %o", info.Mode().Perm())
	}
	if err := WritePrivate(privatePath, keys.PrivateKey); err == nil {
		t.Fatal("private key was silently replaced")
	}
}

func TestProvisionIsIdempotentForSameOwnerAndIssuesCeilingBoundCredential(t *testing.T) {
	dir := t.TempDir()
	paths := KeyPaths{Dir: dir, Private: filepath.Join(dir, "private.key"), Public: filepath.Join(dir, "public.key"), Metadata: filepath.Join(dir, "provisioning.json")}
	now := time.Unix(2_000, 0).UTC()
	if err := Provision(paths, "owner-1", "scenario-authenticator:default", []string{"vrooli-bridge:read", "vrooli-bridge:write"}, now); err != nil {
		t.Fatal(err)
	}
	if err := Provision(paths, "owner-1", "scenario-authenticator:default", []string{"vrooli-bridge:write"}, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := IssueFromProvision(paths, []string{"vrooli-bridge:read"}, now.Add(2*time.Minute), 10*time.Minute); err == nil {
		t.Fatal("legacy provision unexpectedly issued without a target claim")
	}
	if _, err := IssueFromProvision(paths, []string{"agent-manager:write"}, now.Add(2*time.Minute), time.Minute); err == nil {
		t.Fatal("provisioned scope ceiling was bypassed")
	}
}

func TestWrappedProvisionRoundTripBindsPurposeAndTarget(t *testing.T) {
	dir := t.TempDir()
	paths := KeyPaths{Dir: dir, Private: filepath.Join(dir, "private.key"), WrappedPrivate: filepath.Join(dir, "private.key"), Public: filepath.Join(dir, "public.key"), Metadata: filepath.Join(dir, "provisioning.json")}
	now := time.Unix(3_000, 0).UTC()
	if err := ProvisionWrapped(paths, "correct horse", "owner-1", "vrooli:uninstall", "host-a", []string{"vrooli:uninstall"}, now); err != nil {
		t.Fatal(err)
	}
	status, err := Status(paths)
	if err != nil || !status.Complete || !status.WrappedPrivate {
		t.Fatalf("status = %+v, err=%v", status, err)
	}
	privateRaw, err := os.ReadFile(paths.WrappedPrivate)
	if err != nil {
		t.Fatal(err)
	}
	var envelope wrappedPrivateEnvelope
	if err := json.Unmarshal(privateRaw, &envelope); err != nil || envelope.KDF != breakGlassKDF || envelope.Memory < 16*1024 {
		t.Fatal("wrapped key appears to be raw private material")
	}
	beforeWrongPassphrase := append([]byte(nil), privateRaw...)
	token, err := IssueFromWrappedProvision(paths, "correct horse", "vrooli:uninstall", "host-a", []string{"vrooli:uninstall"}, now.Add(time.Minute), 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	public, err := os.ReadFile(paths.Public)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := Verify(ed25519.PublicKey(public), token, "vrooli:uninstall", "host-a", now.Add(time.Minute))
	if err != nil || claims.Target != "host-a" || claims.Audience != "vrooli:uninstall" {
		t.Fatalf("claims = %+v, err=%v", claims, err)
	}
	for _, wrong := range []struct {
		passphrase string
		audience   string
		target     string
	}{
		{"wrong", "vrooli:uninstall", "host-a"},
		{"correct horse", "vrooli:other", "host-a"},
		{"correct horse", "vrooli:uninstall", "host-b"},
	} {
		_, err := IssueFromWrappedProvision(paths, wrong.passphrase, wrong.audience, wrong.target, []string{"vrooli:uninstall"}, now.Add(time.Minute), time.Minute)
		if err == nil {
			t.Fatalf("wrong issuance inputs accepted: %+v", wrong)
		}
		if wrong.passphrase == "wrong" && !errors.Is(err, ErrBreakGlassPassphrase) {
			t.Fatalf("wrong passphrase error = %v, want typed passphrase error", err)
		}
	}
	afterWrongPassphrase, err := os.ReadFile(paths.WrappedPrivate)
	if err != nil || !bytes.Equal(beforeWrongPassphrase, afterWrongPassphrase) {
		t.Fatalf("wrong passphrase changed wrapped key: err=%v", err)
	}
}

func TestVerifyRefusesWrongTarget(t *testing.T) {
	keys, err := GenerateKeyMaterial()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(4_000, 0).UTC()
	token, err := Issue(keys.PrivateKey, BreakGlassClaims{Subject: "owner-1", Audience: "purpose", Target: "host-a", Scopes: []string{"scope"}, IssuedAt: now.Unix(), ExpiresAt: now.Add(time.Minute).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(keys.PublicKey, token, "purpose", "host-b", now); !errors.Is(err, ErrBreakGlassTargetMismatch) {
		t.Fatal("credential for another target was accepted")
	}
}

func TestBreakGlassAudienceMismatchIsTyped(t *testing.T) {
	keys, err := GenerateKeyMaterial()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(5_000, 0).UTC()
	token, err := Issue(keys.PrivateKey, BreakGlassClaims{Subject: "owner-1", Audience: "purpose-a", Target: "host-a", Scopes: []string{"scope"}, IssuedAt: now.Unix(), ExpiresAt: now.Add(time.Minute).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(keys.PublicKey, token, "purpose-b", "host-a", now); !errors.Is(err, ErrBreakGlassAudienceMismatch) {
		t.Fatalf("error = %v, want typed audience mismatch", err)
	}
}

func TestResolveKeyPathsDoesNotExposeOperatorPrivatePath(t *testing.T) {
	paths, err := ResolveKeyPaths()
	if err != nil {
		t.Fatal(err)
	}
	if paths.Private == "" || paths.Private != paths.WrappedPrivate || filepath.Base(paths.Private) != "private.key" {
		t.Fatalf("operator paths = %+v; expected wrapped material at the private-key path", paths)
	}
}
