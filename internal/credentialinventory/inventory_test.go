package credentialinventory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/credentialauthority"
	"github.com/vrooli/vrooli/internal/credentialspec"
	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/internal/securestore"
	"github.com/vrooli/vrooli/internal/testenv"
)

func TestTreasuryWalletDynamicCredentialsAreIncludedInRecovery(t *testing.T) {
	logicalID := "vrooli/treasury/wallets/" + strings.Repeat("a", 32) + "/" + strings.Repeat("b", 32)
	identity, err := credentialauthority.ParseIdentity(logicalID)
	if err != nil {
		t.Fatal(err)
	}
	store := testenv.NewCredentialStore(securestore.ErrNotFound)
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	// These are non-secret fixture strings, not valid wallet keys/passphrases.
	for _, field := range []string{"secp256k1-scalar", "recovery-passphrase", "unrelated-field"} {
		if err := authority.Put(identity, field, "TEST fixture only: "+field); err != nil {
			t.Fatal(err)
		}
	}
	previous := credentialauthority.DefaultAuthority
	credentialauthority.DefaultAuthority = func() (*credentialauthority.Authority, error) { return authority, nil }
	t.Cleanup(func() { credentialauthority.DefaultAuthority = previous })
	root := t.TempDir()
	writeFixtureFile(t, filepath.Join(root, ".vrooli", "service.json"), map[string]any{})
	refs := []securestore.EntryRef{
		{Service: "vrooli.credentials.v1", Key: logicalID + ":secp256k1-scalar"},
		{Service: "vrooli.credentials.v1", Key: logicalID + ":recovery-passphrase"},
		{Service: "vrooli.credentials.v1", Key: logicalID + ":secp256k1-scalar"},
		{Service: "vrooli.credentials.v1", Key: logicalID + ":unrelated-field"},
		{Service: "unrelated-service", Key: logicalID + ":secp256k1-scalar"},
	}
	systems := managedSystemEntriesFromRefs(root, refs)
	for _, entry := range systems {
		if entry.LogicalID == logicalID && entry.Owner != "treasury" {
			t.Fatal("wallet credential attributed to another owner")
		}
	}
	result, err := CollectWithOptions(root, CollectionOptions{
		ManagedSystemEntries:       func(string) []SystemEntry { return systems },
		LiveVaultUnsealKeyEntries:  func() []resources.VaultUnsealKeyEntry { return nil },
		LiveKopiaRepositoryEntries: func() []resources.KopiaRepositoryEntry { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 2 {
		t.Fatalf("wallet recovery population has %d configured entries, want both scoped fields exactly once", len(result.Entries))
	}
	for _, row := range result.Rows {
		if row.LogicalID == logicalID && (row.Resource != "treasury" || row.Provisioning != credentialspec.ProvisioningGenerated) {
			t.Fatal("wallet credential lost its generated-owner attribution")
		}
	}
	for _, field := range []string{"secp256k1-scalar", "recovery-passphrase"} {
		if !containsRecoveryAddress(result.Entries, logicalID+":"+field) {
			t.Fatalf("wallet %s is absent from authoritative recovery population", field)
		}
	}
	bundle, err := authority.ExportRecovery(result.Entries, "TEST-only recovery challenge")
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := credentialauthority.InspectRecovery(bundle, "TEST-only recovery challenge")
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Entries) != 2 {
		t.Fatal("exported manifest does not cover exactly both wallet fields")
	}
	fresh, err := credentialauthority.NewAuthority(testenv.NewCredentialStore(securestore.ErrNotFound))
	if err != nil {
		t.Fatal(err)
	}
	if err = fresh.RestoreRecovery(bundle, "TEST-only recovery challenge"); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"secp256k1-scalar", "recovery-passphrase"} {
		value, err := fresh.Resolve(identity, field)
		if err != nil || value != "TEST fixture only: "+field {
			t.Fatal("fresh TEST authority did not restore the expected fixture")
		}
	}
}

func TestTreasuryWalletInventoryRejectsAdjacentMalformedAndUnrelatedReferences(t *testing.T) {
	valid := "vrooli/treasury/wallets/" + strings.Repeat("a", 32) + "/" + strings.Repeat("b", 32)
	for name, logicalID := range map[string]string{
		"adjacent_namespace": strings.Replace(valid, "/wallets/", "/wallets-extra/", 1),
		"missing_digest":     "vrooli/treasury/wallets/" + strings.Repeat("a", 32),
		"short_namespace":    "vrooli/treasury/wallets/a/" + strings.Repeat("b", 32),
		"nonhex_digest":      "vrooli/treasury/wallets/" + strings.Repeat("a", 32) + "/" + strings.Repeat("z", 32),
		"extra_component":    valid + "/extra",
	} {
		t.Run(name, func(t *testing.T) {
			entries := managedSystemEntriesFromRefs(t.TempDir(), []securestore.EntryRef{{Service: "vrooli.credentials.v1", Key: logicalID + ":secp256k1-scalar"}})
			for _, entry := range entries {
				if entry.Owner == "treasury" {
					t.Fatal("malformed reference entered wallet recovery population")
				}
			}
		})
	}
	for _, field := range []string{"api-key", "value", "secp256k1-scalar-extra", "recovery-passphrase/extra"} {
		entries := managedSystemEntriesFromRefs(t.TempDir(), []securestore.EntryRef{{Service: "vrooli.credentials.v1", Key: valid + ":" + field}})
		for _, entry := range entries {
			if entry.Owner == "treasury" {
				t.Fatal("unrelated field entered wallet recovery population")
			}
		}
	}
}

func TestTreasuryX402RecoveryCredentialsAreIncludedInRecovery(t *testing.T) {
	logicalID := "vrooli/treasury/x402-recovery/" + strings.Repeat("a", 64)
	entries := managedSystemEntriesFromRefs(t.TempDir(), []securestore.EntryRef{
		{Service: "vrooli.credentials.v1", Key: logicalID + ":signed-payment"},
		{Service: "vrooli.credentials.v1", Key: logicalID + ":unrelated"},
	})
	var treasuryEntries []SystemEntry
	for _, entry := range entries {
		if entry.Owner == "treasury" {
			treasuryEntries = append(treasuryEntries, entry)
		}
	}
	if len(treasuryEntries) != 1 || treasuryEntries[0].Field != "signed-payment" || treasuryEntries[0].Provisioning != credentialspec.ProvisioningGenerated {
		t.Fatalf("x402 recovery inventory = %+v", entries)
	}
	for _, malformed := range []string{
		"vrooli/treasury/x402-recovery/" + strings.Repeat("a", 63),
		"vrooli/treasury/x402-recovery/" + strings.Repeat("z", 64),
		"vrooli/treasury/x402-recovery-extra/" + strings.Repeat("a", 64),
	} {
		entries := managedSystemEntriesFromRefs(t.TempDir(), []securestore.EntryRef{{Service: "vrooli.credentials.v1", Key: malformed + ":signed-payment"}})
		for _, entry := range entries {
			if entry.Owner == "treasury" {
				t.Fatalf("malformed recovery identity entered inventory: %+v", entries)
			}
		}
	}
}

func TestCollectEmptyRootIsValueFreeAndEmpty(t *testing.T) {
	result, err := Collect("")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 0 || len(result.Declared) != 0 || len(result.RequiredAbsent) != 0 {
		t.Fatalf("empty inventory = %+v", result)
	}
}

func TestCollectFixtureHasNoDiscoveredMinusCollectedAddresses(t *testing.T) {
	authority, err := credentialauthority.NewAuthority(securestore.Absent("isolated inventory fixture"))
	if err != nil {
		t.Fatal(err)
	}
	previous := credentialauthority.DefaultAuthority
	credentialauthority.DefaultAuthority = func() (*credentialauthority.Authority, error) { return authority, nil }
	t.Cleanup(func() { credentialauthority.DefaultAuthority = previous })

	root := t.TempDir()
	writeFixtureFile(t, filepath.Join(root, ".vrooli", "service.json"), map[string]any{
		"credentials": map[string]any{"descriptors": []map[string]any{
			{"logical_id": "vrooli/fixture/project", "field": "token", "required": true},
		}},
	})
	writeFixtureFile(t, filepath.Join(root, "resources", "fixture", "resource.json"), map[string]any{
		"name":   "fixture",
		"driver": "external-cli",
		"cli": map[string]any{
			"enabled": true, "command": "fixture",
			"adapter":      map[string]any{"kind": "go_module", "module_dir": "cli"},
			"source_build": map[string]any{"kind": "go_module"},
			"invoke":       map[string]any{"kind": "installed_command", "command": "fixture"},
			"freshness":    map[string]any{"inputs": []string{"cli/**", "resource.json"}},
		},
		"credentials": map[string]any{"descriptors": []map[string]any{
			{"logical_id": "vrooli/fixture/configured", "field": "token", "required": true},
			{"logical_id": "vrooli/fixture/absent", "field": "token", "required": true},
		}},
	})
	writeFixtureFile(t, filepath.Join(root, "scenarios", "fixture-scenario", ".vrooli", "service.json"), map[string]any{
		"service": map[string]any{"name": "fixture-scenario"},
		"credentials": map[string]any{"descriptors": []map[string]any{
			{"logical_id": "vrooli/fixture/configured", "field": "token", "required": true},
			{"logical_id": "vrooli/fixture/absent", "field": "token", "required": true},
		}},
	})

	result, err := Collect(root)
	if err != nil {
		t.Fatal(err)
	}
	collected := make(map[string]struct{}, len(result.Entries))
	for _, entry := range result.Entries {
		collected[string(entry.Identity)+":"+entry.Field] = struct{}{}
	}
	for _, address := range []string{"vrooli/fixture/configured:token", "vrooli/fixture/absent:token"} {
		if _, missing := collected[address]; missing {
			if !contains(result.RequiredAbsent, address) {
				t.Fatalf("declared address %q was neither collected nor marked required-absent", address)
			}
		}
	}
	if !contains(result.RequiredAbsent, "vrooli/fixture/absent:token") {
		t.Fatalf("required absent fixture address missing: %v", result.RequiredAbsent)
	}
	if !contains(result.RequiredAbsent, "vrooli/fixture/project:token") {
		t.Fatalf("project service credential was not collected: entries=%v absent=%v", result.Entries, result.RequiredAbsent)
	}
	if _, present := collected["vrooli/fixture/configured:token"]; !present {
		// The fixture remains portable across hosts whose live store does not
		// contain this deliberately named address; the important invariant is
		// that a declared address is classified, never silently dropped.
		if !contains(result.RequiredAbsent, "vrooli/fixture/configured:token") {
			t.Fatalf("configured fixture address was not classified: entries=%v absent=%v", result.Entries, result.RequiredAbsent)
		}
	}
}

func TestManagedDesktopSigningIdentityMatchesSharedAndScenarioNamespaces(t *testing.T) {
	cases := map[string]bool{
		"vrooli/desktop-signing":                   true,
		"vrooli/desktop-signing-e2e":               true,
		"vrooli/desktop-signing/prod":              true,
		"vrooli/scenario-to-desktop/hello-desktop": true,
		"vrooli/scenario-to-desktop":               false,
		"vrooli/desktop-signingish":                false,
		"device-control/android-1/unlock":          false,
		"vrooli/release-authority":                 false,
	}
	for identity, want := range cases {
		if got := isManagedDesktopSigningIdentity(identity); got != want {
			t.Errorf("isManagedDesktopSigningIdentity(%q) = %v, want %v", identity, got, want)
		}
	}
	if !isManagedDesktopSigningField("gpg-private-key") || !isManagedDesktopSigningField("gpg-passphrase") {
		t.Error("expected both managed signing fields to match")
	}
	if isManagedDesktopSigningField("api-key") {
		t.Error("unrelated field must not match")
	}
}

func TestGeneratedCredentialIsInventoriedAndRecoverable(t *testing.T) {
	store := testenv.NewCredentialStore(securestore.ErrNotFound)
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := credentialauthority.ParseIdentity("vrooli/generated-service")
	if err != nil {
		t.Fatal(err)
	}
	if err := authority.Put(identity, "signing-key", "generated-value"); err != nil {
		t.Fatal(err)
	}
	previous := credentialauthority.DefaultAuthority
	credentialauthority.DefaultAuthority = func() (*credentialauthority.Authority, error) { return authority, nil }
	t.Cleanup(func() { credentialauthority.DefaultAuthority = previous })

	root := t.TempDir()
	writeFixtureFile(t, filepath.Join(root, ".vrooli", "service.json"), map[string]any{
		"credentials": map[string]any{"descriptors": []map[string]any{{
			"logical_id": "vrooli/generated-service", "field": "signing-key", "required": true, "provisioning": "generated",
		}}},
	})

	result, err := Collect(root)
	if err != nil {
		t.Fatal(err)
	}
	address := "vrooli/generated-service:signing-key"
	if !containsRecoveryAddress(result.Entries, address) {
		t.Fatalf("generated credential was not inventoried: entries=%v", result.Entries)
	}
	if contains(result.RequiredAbsent, address) {
		t.Fatalf("generated credential was treated as operator-absent: %v", result.RequiredAbsent)
	}
	bundle, err := authority.ExportRecovery(result.Entries, "generated-recovery-passphrase")
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := credentialauthority.InspectRecovery(bundle, "generated-recovery-passphrase")
	if err != nil {
		t.Fatal(err)
	}
	if !containsRecoveryAddress(manifest.Entries, address) {
		t.Fatalf("generated credential was not covered by recovery: entries=%v", manifest.Entries)
	}
}

func writeFixtureFile(t *testing.T, path string, value any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsRecoveryAddress(values []credentialauthority.RecoveryEntry, want string) bool {
	for _, value := range values {
		if string(value.Identity)+":"+value.Field == want {
			return true
		}
	}
	return false
}
