package resources

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/vrooli/internal/credentialauthority"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
	vaultbootstrap "github.com/vrooli/vrooli/packages/vaultbootstrap-go"
)

// The Vault half of the disaster drill.
//
// Every piece of this chain is unit-tested somewhere, and that is exactly why
// this test exists: the pieces live in three packages and the failure that
// matters is a break *between* them. A key stored under one address and
// enumerated under another, or exported and restored into a form Vault will not
// accept, passes every unit test and loses the instance anyway.
//
// The chain, end to end: a live instance stores its unseal key -> the recovery
// inventory finds it without any manifest naming it -> export -> total store
// loss -> restore -> unseal -> mint a fresh root token from the recovered key.

const drillUnsealKey = "hG8vQm2xR7nP1wZ4tY6bK3sL9dF5cJ0a="

// fakeVault answers the three exchanges a restore performs: seal-status, unseal,
// and the root-generation dance. It masks the token exactly as Vault does.
func fakeVault(t *testing.T, rootToken, otp string) *httptest.Server {
	t.Helper()
	if len(rootToken) != len(otp) {
		t.Fatalf("fixture error: root token and one-time pad must be the same length")
	}
	masked := make([]byte, len(rootToken))
	for i := range masked {
		masked[i] = rootToken[i] ^ otp[i]
	}
	encoded := base64.RawStdEncoding.EncodeToString(masked)

	sealed := true
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/sys/seal-status":
			_, _ = fmt.Fprintf(w, `{"initialized":true,"sealed":%t}`, sealed)
		case r.URL.Path == "/v1/sys/unseal":
			sealed = false
			_, _ = fmt.Fprint(w, `{"sealed":false}`)
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/sys/generate-root/attempt":
			w.WriteHeader(http.StatusNoContent)
		case r.URL.Path == "/v1/sys/generate-root/attempt":
			_, _ = fmt.Fprintf(w, `{"nonce":"drill-nonce","otp":%q}`, otp)
		case r.URL.Path == "/v1/sys/generate-root/update":
			_, _ = fmt.Fprintf(w, `{"complete":true,"encoded_token":%q}`, encoded)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestVaultUnsealKeySurvivesTotalStoreLossAndRestoresAWorkingInstance(t *testing.T) {
	const instanceID = "drill-instance"
	const rootToken = "hvs.DRILLROOTTOKENVAL"
	const otp = "0123456789abcdefghijk"

	// --- a live instance stores its material -----------------------------
	sourceStore := &memorySecureStore{values: map[string]string{}}
	sourceAuthority, err := credentialauthority.NewAuthority(sourceStore)
	if err != nil {
		t.Fatal(err)
	}
	material := vaultbootstrap.Material{RootToken: "hvs.ORIGINALROOTTOKEN", UnsealKey: drillUnsealKey}
	if err := vaultbootstrap.Save(sourceStore, sourceAuthority, instanceID, material); err != nil {
		t.Fatalf("store vault material: %v", err)
	}

	// --- the inventory finds it with no manifest naming it ---------------
	broker := &Broker{instances: map[string]ManagedInstance{
		instanceID: {ID: instanceID, Resource: "vault", Provider: resourcedeployment.ProviderManagedPrivate},
	}}
	inventory := VaultUnsealKeyEntries(broker)
	if len(inventory) != 1 {
		t.Fatalf("inventory = %+v, want the live instance", inventory)
	}
	entries := make([]credentialauthority.RecoveryEntry, 0, len(inventory))
	for _, found := range inventory {
		identity, err := credentialauthority.ParseIdentity(found.LogicalID)
		if err != nil {
			t.Fatal(err)
		}
		entries = append(entries, credentialauthority.RecoveryEntry{Identity: identity, Field: found.Field})
	}

	// --- export ----------------------------------------------------------
	const passphrase = "drill-recovery-passphrase"
	bundle, err := sourceAuthority.ExportRecovery(entries, passphrase)
	if err != nil {
		t.Fatalf("export recovery bundle: %v", err)
	}
	if bytes.Contains(bundle, []byte(drillUnsealKey)) {
		t.Fatal("the bundle carries the unseal key in plaintext")
	}
	// The root token must not be in the bundle. Vault re-mints one from the
	// unseal key, so carrying it would widen the blast radius for nothing.
	if bytes.Contains(bundle, []byte(material.RootToken)) {
		t.Fatal("the bundle carries the root token, which is regenerable and must not be backed up")
	}

	// --- total store loss ------------------------------------------------
	targetStore := &memorySecureStore{values: map[string]string{}}
	targetAuthority, err := credentialauthority.NewAuthority(targetStore)
	if err != nil {
		t.Fatal(err)
	}
	if _, found, _ := vaultbootstrap.LoadUnsealKey(targetAuthority, instanceID); found {
		t.Fatal("the replacement store already held the key; this is not a restore")
	}

	// --- restore ---------------------------------------------------------
	if err := targetAuthority.RestoreRecovery(bundle, passphrase); err != nil {
		t.Fatalf("restore recovery bundle: %v", err)
	}
	recovered, found, err := vaultbootstrap.LoadUnsealKey(targetAuthority, instanceID)
	if err != nil || !found {
		t.Fatalf("LoadUnsealKey after restore = %v, %v", found, err)
	}
	if recovered != drillUnsealKey {
		t.Fatalf("recovered key is %d bytes, want %d — the bundle did not round-trip",
			len(recovered), len(drillUnsealKey))
	}

	// --- the recovered key actually reopens the instance ------------------
	// This is the part no unit test covers: a key that round-trips through the
	// bundle but that Vault will not accept is a backup that only looks like one.
	server := fakeVault(t, rootToken, otp)
	defer server.Close()
	client := vaultbootstrap.Client{Endpoint: server.URL}

	state, err := client.LifecycleState(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if state != vaultbootstrap.StateSealed {
		t.Fatalf("state = %q, want a sealed instance to recover", state)
	}
	if err := client.Unseal(context.Background(), vaultbootstrap.Material{UnsealKey: recovered}); err != nil {
		t.Fatalf("unseal with the recovered key: %v", err)
	}

	// --- and mints a fresh root token, which the bundle deliberately lacks --
	minted, err := client.GenerateRootToken(context.Background(), recovered)
	if err != nil {
		t.Fatalf("generate root token from the recovered key: %v", err)
	}
	if minted != rootToken {
		t.Fatalf("minted token is %d bytes, want %d", len(minted), len(rootToken))
	}
	if minted == material.RootToken {
		t.Fatal("recovery returned the original root token; it must mint a new one")
	}
}

// An export that silently omitted a live instance's key would hand the operator
// a bundle that looks complete and cannot restore the instance.
func TestRecoveryExportFailsRatherThanSkippingAnUnstoredUnsealKey(t *testing.T) {
	store := &memorySecureStore{values: map[string]string{}}
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := vaultbootstrap.UnsealKeyIdentity("never-stored")
	if err != nil {
		t.Fatal(err)
	}
	_, err = authority.ExportRecovery(
		[]credentialauthority.RecoveryEntry{{Identity: identity, Field: vaultbootstrap.UnsealKeyField}},
		"passphrase")
	if err == nil {
		t.Fatal("export produced a bundle for an instance whose key was never stored")
	}
}

// recordingBootstrapper implements the control plane's Vault interfaces and
// records what was asked of it, so a test can prove a restore recovered rather
// than re-initialized. Re-initializing is the failure that matters: it succeeds,
// looks healthy, and replaces the key to data still sealed under the old one.
type recordingBootstrapper struct {
	bootstrapCalled bool
	unsealed        bool
	generated       bool
	mintedToken     string
}

func (r *recordingBootstrapper) Bootstrap(context.Context, string) (VaultBootstrapMaterial, error) {
	r.bootstrapCalled = true
	return VaultBootstrapMaterial{RootToken: "hvs.FRESHLY-INITIALIZED", UnsealKey: "brand-new-key"}, nil
}

func (r *recordingBootstrapper) Unseal(context.Context, string, VaultBootstrapMaterial) error {
	r.unsealed = true
	return nil
}

func (r *recordingBootstrapper) GenerateRootToken(_ context.Context, _, unsealKey string) (string, error) {
	r.generated = true
	if unsealKey != drillUnsealKey {
		return "", fmt.Errorf("root generation used the wrong unseal key")
	}
	r.mintedToken = "hvs.REMINTED-ROOT-TOKEN"
	return r.mintedToken, nil
}

// The control-plane restore path: material blob gone, unseal key present. This
// is exactly the state a host is in after restoring a recovery bundle, and it
// used to be reported as unrecoverable even though everything needed was in the
// credential store.
func TestControlPlaneRecoversFromAnUnsealKeyRatherThanReinitializing(t *testing.T) {
	const instanceID = "restored-instance"
	store := &memorySecureStore{values: map[string]string{}}
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	previous := unsealKeyStore
	unsealKeyStore = func() vaultbootstrap.UnsealKeyStore { return authority }
	t.Cleanup(func() { unsealKeyStore = previous })

	// The bundle restored the key; the blob is deliberately absent.
	if err := vaultbootstrap.SaveUnsealKey(authority, instanceID, drillUnsealKey); err != nil {
		t.Fatal(err)
	}
	if _, found, _ := loadVaultBootstrapMaterial(store, instanceID); found {
		t.Fatal("fixture error: the material blob should be absent")
	}

	bootstrap := &recordingBootstrapper{}
	material, didRecover, err := recoverVaultFromUnsealKey(
		context.Background(), bootstrap, store, "http://127.0.0.1:8200", instanceID)
	if err != nil {
		t.Fatalf("recoverVaultFromUnsealKey() = %v", err)
	}
	if !didRecover {
		t.Fatal("the restore path did not recover despite a stored unseal key")
	}
	if bootstrap.bootstrapCalled {
		t.Fatal("the restore re-initialized the instance, destroying the key to its own data")
	}
	if !bootstrap.unsealed || !bootstrap.generated {
		t.Fatalf("restore skipped a step (unsealed=%t generated=%t)", bootstrap.unsealed, bootstrap.generated)
	}
	if material.UnsealKey != drillUnsealKey || material.RootToken != bootstrap.mintedToken {
		t.Fatalf("recovered material is wrong: unseal %d bytes, root %q",
			len(material.UnsealKey), material.RootToken)
	}

	// The re-minted pair is persisted, so the next start is an ordinary
	// recovery rather than another root generation.
	stored, found, err := loadVaultBootstrapMaterial(store, instanceID)
	if err != nil || !found {
		t.Fatalf("recovered material was not persisted: %v, %v", found, err)
	}
	if stored != material {
		t.Fatal("the persisted material differs from what recovery produced")
	}
}

// With no stored key this is a fresh instance, not a restore, and the caller
// must bootstrap normally rather than fail.
func TestControlPlaneTreatsNoStoredKeyAsAFreshInstance(t *testing.T) {
	store := &memorySecureStore{values: map[string]string{}}
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	previous := unsealKeyStore
	unsealKeyStore = func() vaultbootstrap.UnsealKeyStore { return authority }
	t.Cleanup(func() { unsealKeyStore = previous })

	bootstrap := &recordingBootstrapper{}
	_, didRecover, err := recoverVaultFromUnsealKey(
		context.Background(), bootstrap, store, "http://127.0.0.1:8200", "never-seen")
	if err != nil {
		t.Fatalf("recoverVaultFromUnsealKey() = %v, want a clean not-a-restore", err)
	}
	if didRecover {
		t.Fatal("recovery claimed to restore an instance with no stored key")
	}
	if bootstrap.generated {
		t.Fatal("recovery attempted root generation with no key")
	}
}
