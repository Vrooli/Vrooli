package trustposture

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"os"
	osuser "os/user"
	"path/filepath"
	"strings"
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
	if _, err := Verify(keys.PublicKey, token, "scenario-authenticator:default", "host-a", now.Add(13*time.Minute)); err == nil {
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

func TestWrappedProvisionRefusesExistingMaterialWithoutChangingIt(t *testing.T) {
	dir := t.TempDir()
	paths := KeyPaths{Dir: dir, WrappedPrivate: filepath.Join(dir, "private.key"), Public: filepath.Join(dir, "public.key"), Metadata: filepath.Join(dir, "provisioning.json")}
	now := time.Unix(3_500, 0).UTC()
	if err := ProvisionWrapped(paths, "correct horse", "owner-1", BreakGlassUninstallAudience, "host-a", []string{BreakGlassUninstallScope}, now); err != nil {
		t.Fatal(err)
	}
	beforeWrapped, err := os.ReadFile(paths.WrappedPrivate)
	if err != nil {
		t.Fatal(err)
	}
	beforePublic, err := os.ReadFile(paths.Public)
	if err != nil {
		t.Fatal(err)
	}
	err = ProvisionWrapped(paths, "correct horse", "owner-1", BreakGlassUninstallAudience, "host-a", []string{BreakGlassUninstallScope}, now.Add(time.Minute))
	if !errors.Is(err, ErrBreakGlassAlreadyProvisioned) || !strings.Contains(err.Error(), "wrapped_private") {
		t.Fatalf("existing material error = %v", err)
	}
	afterWrapped, err := os.ReadFile(paths.WrappedPrivate)
	if err != nil {
		t.Fatal(err)
	}
	afterPublic, err := os.ReadFile(paths.Public)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(beforeWrapped, afterWrapped) || !bytes.Equal(beforePublic, afterPublic) {
		t.Fatal("existing material changed after refused provisioning")
	}
}

func TestRotateWrappedRequiresCurrentPassphraseAndChangesPublicKey(t *testing.T) {
	dir := t.TempDir()
	paths := KeyPaths{Dir: dir, WrappedPrivate: filepath.Join(dir, "private.key"), Public: filepath.Join(dir, "public.key"), Metadata: filepath.Join(dir, "provisioning.json")}
	now := time.Unix(6_000, 0).UTC()
	if err := ProvisionWrapped(paths, "correct horse", "owner-1", BreakGlassUninstallAudience, "host-a", []string{BreakGlassUninstallScope}, now); err != nil {
		t.Fatal(err)
	}
	oldPublic, err := os.ReadFile(paths.Public)
	if err != nil {
		t.Fatal(err)
	}
	oldWrapped, err := os.ReadFile(paths.WrappedPrivate)
	if err != nil {
		t.Fatal(err)
	}
	if err := RotateWrapped(paths, "wrong", now.Add(time.Minute)); !errors.Is(err, ErrBreakGlassPassphrase) {
		t.Fatalf("wrong rotation error = %v", err)
	}
	unchanged, err := os.ReadFile(paths.WrappedPrivate)
	if err != nil || !bytes.Equal(oldWrapped, unchanged) {
		t.Fatalf("wrong rotation changed material: %v", err)
	}
	if err := RotateWrapped(paths, "correct horse", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	newPublic, err := os.ReadFile(paths.Public)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(oldPublic, newPublic) {
		t.Fatal("rotation retained the old public key")
	}
	if _, err := IssueFromWrappedProvision(paths, "correct horse", BreakGlassUninstallAudience, "host-a", []string{BreakGlassUninstallScope}, now.Add(2*time.Minute), time.Minute); err != nil {
		t.Fatalf("rotated material could not issue: %v", err)
	}
}

func TestVerifyBoundNamesEveryContextMismatch(t *testing.T) {
	keys, err := GenerateKeyMaterial()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(7_000, 0).UTC()
	binding := BreakGlassBinding{OperatorID: "operator-1", MachineID: "machine-1", NodeID: "node-1", Scope: "all", PlanHash: "hash-1", OperationID: "op-1"}
	token, err := Issue(keys.PrivateKey, BreakGlassClaims{Subject: "owner-1", Audience: BreakGlassUninstallAudience, Target: "host-a", Scopes: []string{BreakGlassUninstallScope}, IssuedAt: now.Unix(), ExpiresAt: now.Add(time.Minute).Unix(), OperatorID: binding.OperatorID, MachineID: binding.MachineID, NodeID: binding.NodeID, Scope: binding.Scope, PlanHash: binding.PlanHash, OperationID: binding.OperationID})
	if err != nil {
		t.Fatal(err)
	}
	for field, mutate := range map[string]func(*BreakGlassBinding){
		"operator_id":   func(v *BreakGlassBinding) { v.OperatorID = "other" },
		"machine_id":    func(v *BreakGlassBinding) { v.MachineID = "other" },
		"node_id":       func(v *BreakGlassBinding) { v.NodeID = "other" },
		"cleanup_scope": func(v *BreakGlassBinding) { v.Scope = "agent" },
		"plan_hash":     func(v *BreakGlassBinding) { v.PlanHash = "other" },
		"operation_id":  func(v *BreakGlassBinding) { v.OperationID = "other" },
	} {
		bad := binding
		mutate(&bad)
		_, err := VerifyBound(keys.PublicKey, token, BreakGlassUninstallAudience, "host-a", bad, now)
		if err == nil || !strings.Contains(err.Error(), field) {
			t.Fatalf("%s mismatch error = %v", field, err)
		}
	}
}

func TestVerifyBoundRefusesMissingContextByField(t *testing.T) {
	keys, err := GenerateKeyMaterial()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(7_500, 0).UTC()
	_, err = VerifyBound(keys.PublicKey, "", BreakGlassUninstallAudience, "host-a", BreakGlassBinding{}, now)
	if err == nil || !errors.Is(err, ErrBreakGlassBindingMissing) || !strings.Contains(err.Error(), "operator_id") {
		t.Fatalf("missing binding error = %v", err)
	}
}

func TestBreakGlassRefusalMatrixNamesEachSecurityBoundary(t *testing.T) {
	keys, err := GenerateKeyMaterial()
	requireNoError(t, err)
	otherKeys, err := GenerateKeyMaterial()
	requireNoError(t, err)
	now := time.Unix(9_000, 0).UTC()
	binding := BreakGlassBinding{OperatorID: "operator-1", MachineID: "machine-1", NodeID: "node-1", Scope: "all", PlanHash: "plan-hash", OperationID: "operation-1"}
	token, err := Issue(keys.PrivateKey, BreakGlassClaims{
		Subject: "owner-1", Audience: BreakGlassUninstallAudience, Target: "host-a", Scopes: []string{BreakGlassUninstallScope},
		IssuedAt: now.Unix(), ExpiresAt: now.Add(time.Minute).Unix(), OperatorID: binding.OperatorID, MachineID: binding.MachineID,
		NodeID: binding.NodeID, Scope: binding.Scope, PlanHash: binding.PlanHash, OperationID: binding.OperationID,
	})
	requireNoError(t, err)
	tests := []struct {
		name   string
		check  func() error
		marker error
		text   string
	}{
		{name: "target", check: func() error {
			_, err := Verify(keys.PublicKey, token, BreakGlassUninstallAudience, "host-b", now)
			return err
		}, marker: ErrBreakGlassTargetMismatch, text: "target"},
		{name: "plan_hash", check: func() error {
			bad := binding
			bad.PlanHash = "changed"
			_, err := VerifyBound(keys.PublicKey, token, BreakGlassUninstallAudience, "host-a", bad, now)
			return err
		}, marker: ErrBreakGlassBindingMismatch, text: "plan_hash"},
		{name: "expired", check: func() error {
			_, err := Verify(keys.PublicKey, token, BreakGlassUninstallAudience, "host-a", now.Add(5*time.Minute))
			return err
		}, marker: ErrBreakGlassClockSkew, text: "clock skew"},
		{name: "pinned_key", check: func() error {
			_, err := Verify(otherKeys.PublicKey, token, BreakGlassUninstallAudience, "host-a", now)
			return err
		}, marker: nil, text: "signature"},
		{name: "scope_ceiling", check: func() error {
			_, err := ScopeCeiling([]string{"vrooli:read"}, []string{BreakGlassUninstallScope})
			return err
		}, marker: nil, text: "ceiling"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.check()
			if err == nil || (test.marker != nil && !errors.Is(err, test.marker)) || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(test.text)) {
				t.Fatalf("refusal = %v, want marker %v containing %q", err, test.marker, test.text)
			}
		})
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestVerifyClockSkewToleranceIsExplicit(t *testing.T) {
	keys, err := GenerateKeyMaterial()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(8_000, 0).UTC()
	token, err := Issue(keys.PrivateKey, BreakGlassClaims{Subject: "owner-1", Audience: "purpose", Target: "host-a", Scopes: []string{"scope"}, IssuedAt: now.Add(time.Minute).Unix(), ExpiresAt: now.Add(10 * time.Minute).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(keys.PublicKey, token, "purpose", "host-a", now); err != nil {
		t.Fatalf("credential inside skew tolerance rejected: %v", err)
	}
	if _, err := Verify(keys.PublicKey, token, "purpose", "host-a", now.Add(-BreakGlassClockSkew-time.Second)); !errors.Is(err, ErrBreakGlassClockSkew) {
		t.Fatalf("early credential error = %v, want clock skew", err)
	}
	if _, err := Verify(keys.PublicKey, token, "purpose", "host-a", now.Add(10*time.Minute+BreakGlassClockSkew)); !errors.Is(err, ErrBreakGlassClockSkew) {
		t.Fatalf("late credential error = %v, want clock skew", err)
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

func TestResolveKeyPathsFallsBackWhenHomeIsUnset(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("VROOLI_BREAK_GLASS_DIR", "")

	current, err := osuser.Current()
	if err != nil {
		t.Skipf("current user unavailable: %v", err)
	}
	paths, err := ResolveKeyPaths()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(current.HomeDir, ".vrooli", "identity", "break-glass", "private.key")
	if paths.Private != want {
		t.Fatalf("private path = %q, want %q", paths.Private, want)
	}
}

func TestResetWrappedRemovesOnlyManagedRegularFiles(t *testing.T) {
	dir := t.TempDir()
	paths := KeyPaths{
		Dir:            dir,
		WrappedPrivate: filepath.Join(dir, "private.key"),
		Private:        filepath.Join(dir, "private.key"),
		Public:         filepath.Join(dir, "public.key"),
		Metadata:       filepath.Join(dir, "provisioning.json"),
		Credential:     filepath.Join(dir, "credential"),
	}
	for _, path := range []string{paths.WrappedPrivate, paths.Public, paths.Metadata, paths.Credential} {
		if err := os.WriteFile(path, []byte("managed"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ResetWrapped(paths); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{paths.WrappedPrivate, paths.Public, paths.Metadata, paths.Credential} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("managed path %s still exists: %v", path, err)
		}
	}
	if raw, err := os.ReadFile(outside); err != nil || string(raw) != "keep" {
		t.Fatalf("outside file changed: %q, %v", raw, err)
	}
	if err := ResetWrapped(paths); err != nil {
		t.Fatalf("reset should be idempotent: %v", err)
	}
}

func TestResetWrappedRefusesSymlinks(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	paths := KeyPaths{WrappedPrivate: filepath.Join(dir, "private.key")}
	if err := os.Symlink(outside, paths.WrappedPrivate); err != nil {
		t.Fatal(err)
	}
	if err := ResetWrapped(paths); err == nil {
		t.Fatal("reset followed a symlink")
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("outside target was affected: %v", err)
	}
}
