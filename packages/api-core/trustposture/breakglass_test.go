package trustposture

import (
	"crypto/ed25519"
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
		BreakGlassClaims{Subject: "owner-1", Audience: "scenario-authenticator:default", IssuedAt: now.Unix(), ExpiresAt: now.Add(10 * time.Minute).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := Verify(keys.PublicKey, token, "scenario-authenticator:default", now)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "owner-1" || len(claims.Scopes) != 2 {
		t.Fatalf("claims = %+v", claims)
	}
	if _, err := IssueForAccount(keys.PrivateKey, []string{"agent-manager:read"}, []string{"agent-manager:write"}, BreakGlassClaims{Subject: "owner-1", Audience: "a", IssuedAt: now.Unix(), ExpiresAt: now.Add(time.Minute).Unix()}); err == nil {
		t.Fatal("scope ceiling bypassed")
	}
	if _, err := Verify(keys.PublicKey, token, "scenario-authenticator:default", now.Add(11*time.Minute)); err == nil {
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
	token, err := IssueFromProvision(paths, []string{"vrooli-bridge:read"}, now.Add(2*time.Minute), 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	public, err := os.ReadFile(paths.Public)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(ed25519.PublicKey(public), token, "scenario-authenticator:default", now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := IssueFromProvision(paths, []string{"agent-manager:write"}, now.Add(2*time.Minute), time.Minute); err == nil {
		t.Fatal("provisioned scope ceiling was bypassed")
	}
}
