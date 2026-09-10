package credentialclient

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/credentialinventory"
	"github.com/vrooli/vrooli/internal/resources"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// writeScopeFixture builds a repository whose credential population is known
// exactly: a project manifest, two scenarios, and two resources. Tests assert
// against this fixture rather than the live repository, whose manifests change
// for reasons unrelated to scope resolution.
func writeScopeFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(path, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(root, ".vrooli", "service.json"), `{
  "service": {"name": "vrooli", "description": "Project scope"},
  "credentials": {"descriptors": [
    {"logical_id": "vrooli/remote-desktop", "field": "username", "label": "Remote desktop username", "obtain_url": "https://example.test/remote-desktop", "provisioning": "operator", "required": false},
    {"logical_id": "vrooli/remote-desktop", "field": "password", "label": "Remote desktop password", "description": "Derived host password", "obtain_url": "https://example.test/remote-desktop", "provisioning": "derived", "derived_from": "username", "required": false}
  ]}
}`)
	write(filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), `{
  "service": {"name": "alpha", "description": "Alpha"},
  "credentials": {
    "descriptors": [{"version": "credential-descriptor/v1", "logical_id": "vrooli/alpha", "field": "token", "obtain_url": "https://example.test/alpha", "provisioning": "operator", "required": true, "owner": "alpha-owner", "kind": "secret", "provider": "alpha-provider", "requirement_group": "alpha.auth", "consumer_refs": ["alpha.session"], "companion_settings": ["alpha.region"], "acquisition_ref": "alpha.connect", "verification_ref": "alpha.verify", "recovery_ref": "alpha.recover", "help_ref": "alpha-help", "evidence_policy": "release-required", "applies_when": {"all": [{"eq": {"setting": "alpha.mode", "value": "enabled"}}]}}],
    "consumers": [{"logical_id": "vrooli/alpha", "field": "token", "kind": "delegated", "consumer": "alpha session broker", "source_ref": "api/session.go:7"}]
  }
}`)
	write(filepath.Join(root, "scenarios", "beta", ".vrooli", "service.json"), `{
  "service": {"name": "beta", "description": "Beta"},
  "credentials": {"descriptors": [{"logical_id": "vrooli/beta", "field": "token", "obtain_url": "https://example.test/beta", "provisioning": "operator", "required": false, "tiers": ["tier-1-local"]}]}
}`)
	write(filepath.Join(root, "resources", "one", "resource.json"), `{
  "name": "one", "display_name": "One", "description": "One", "category": "general",
  "driver": "external-cli",
  "binary": "one",
  "cli": {"enabled": true, "command": "one", "adapter": {"kind": "go_module", "module_dir": "cli"}, "source_build": {"kind": "go_module"}, "invoke": {"kind": "installed_command", "command": "one"}, "freshness": {"inputs": ["cli/**", "resource.json"]}},
  "credentials": {"descriptors": [{"logical_id": "vrooli/one", "field": "password", "description": "One password", "obtain_url": "https://example.test/one", "provisioning": "operator", "required": true}]}
}`)
	write(filepath.Join(root, "resources", "two", "resource.json"), `{
  "name": "two", "display_name": "Two", "description": "Two", "category": "general",
  "driver": "external-cli",
  "binary": "two",
  "cli": {"enabled": true, "command": "two", "adapter": {"kind": "go_module", "module_dir": "cli"}, "source_build": {"kind": "go_module"}, "invoke": {"kind": "installed_command", "command": "two"}, "freshness": {"inputs": ["cli/**", "resource.json"]}},
  "credentials": {"descriptors": [{"logical_id": "vrooli/two", "field": "password", "obtain_url": "https://example.test/two", "provisioning": "operator", "required": false}]}
}`)
	return root
}

func addressSet(refs []CredentialRef) []string {
	addresses := make([]string, 0, len(refs))
	for _, ref := range refs {
		addresses = append(addresses, ref.LogicalID+":"+ref.Field)
	}
	sort.Strings(addresses)
	return addresses
}

func requireAddresses(t *testing.T, got []CredentialRef, want ...string) {
	t.Helper()
	sort.Strings(want)
	have := addressSet(got)
	if len(have) != len(want) {
		t.Fatalf("addresses = %v, want %v", have, want)
	}
	for i := range want {
		if have[i] != want[i] {
			t.Fatalf("addresses = %v, want %v", have, want)
		}
	}
}

func TestDescriptorsForScopeIncludesProjectManifest(t *testing.T) {
	root := writeScopeFixture(t)
	refs, err := DescriptorsForScope(root, Scope{IncludeProject: true})
	if err != nil {
		t.Fatal(err)
	}
	requireAddresses(t, refs,
		"vrooli/alpha:token", "vrooli/beta:token", "vrooli/one:password",
		"vrooli/remote-desktop:password", "vrooli/remote-desktop:username", "vrooli/two:password")
	for _, ref := range refs {
		if ref.Provisioning == "" {
			t.Fatalf("descriptor %s:%s lost provisioning classification: %+v", ref.LogicalID, ref.Field, ref)
		}
		if ref.LogicalID == "vrooli/alpha" && (ref.Owner != "alpha-owner" || ref.SourceRef == "" || !reflect.DeepEqual(ref.ConsumerRefs, []string{"alpha session broker", "alpha.session"}) || ref.Provider != "alpha-provider" || ref.RequirementGroup != "alpha.auth" || len(ref.CompanionSettings) != 1 || ref.AcquisitionRef != "alpha.connect" || ref.VerificationRef != "alpha.verify" || ref.RecoveryRef != "alpha.recover" || ref.HelpRef != "alpha-help" || ref.EvidencePolicy != "release-required" || ref.AppliesWhen == nil || len(ref.Provenance) != 1 || ref.Provenance[0].Owner != "alpha-owner") {
			t.Fatalf("alpha descriptor lost owner/source/consumer provenance: %+v", ref)
		}
		if ref.LogicalID != "vrooli/remote-desktop" {
			if ref.LogicalID == "vrooli/one" && ref.ObtainURL != "https://example.test/one" {
				t.Fatalf("resource descriptor lost acquisition guide: %+v", ref)
			}
			continue
		}
		if ref.Resource != ProjectScopeOwner {
			t.Fatalf("project descriptor owner = %q, want %q", ref.Resource, ProjectScopeOwner)
		}
		if ref.Label == "" {
			t.Fatalf("project descriptor %s carries no label", ref.Field)
		}
		if ref.ObtainURL == "" {
			t.Fatalf("project descriptor %s carries no acquisition guide", ref.Field)
		}
		if ref.Field == "password" && (ref.Provisioning != "derived" || ref.DerivedFrom != "username") {
			t.Fatalf("project descriptor %s lost provisioning metadata: %+v", ref.Field, ref)
		}
	}
}

func TestScopedClientKeepsListAndDoctorOnTheSamePopulation(t *testing.T) {
	root := writeScopeFixture(t)
	authority, err := credentialauthority.NewAuthority(&testStore{})
	if err != nil {
		t.Fatal(err)
	}
	scope := &Scope{IncludeProject: true, Scenarios: []string{"alpha"}, Resources: []string{}}
	client, err := NewInProcess(InProcessOptions{Authority: authority, Root: root, DescriptorScope: scope})
	if err != nil {
		t.Fatal(err)
	}
	listed, err := client.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	diagnosed, err := client.Doctor(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	listAddresses := addressSet(listed)
	doctorAddresses := addressSet(diagnosed.Credentials)
	if !reflect.DeepEqual(listAddresses, doctorAddresses) {
		t.Fatalf("list addresses = %v, doctor addresses = %v", listAddresses, doctorAddresses)
	}
	requireAddresses(t, listed, "vrooli/alpha:token", "vrooli/remote-desktop:password", "vrooli/remote-desktop:username")
}

func TestDescriptorsForScopeExcludesProjectWhenNotRequested(t *testing.T) {
	root := writeScopeFixture(t)
	refs, err := DescriptorsForScope(root, Scope{})
	if err != nil {
		t.Fatal(err)
	}
	requireAddresses(t, refs,
		"vrooli/alpha:token", "vrooli/beta:token", "vrooli/one:password", "vrooli/two:password")
}

func TestDescriptorsForScopeFiltersSelectedMembers(t *testing.T) {
	root := writeScopeFixture(t)
	refs, err := DescriptorsForScope(root, Scope{Scenarios: []string{"alpha"}, Resources: []string{"two"}})
	if err != nil {
		t.Fatal(err)
	}
	requireAddresses(t, refs, "vrooli/alpha:token", "vrooli/two:password")
}

func TestDescriptorsForScopeAppliesTierAndManagedApplicability(t *testing.T) {
	root := writeScopeFixture(t)
	refs, err := DescriptorsForScope(root, Scope{IncludeProject: true, IncludeManaged: true, Tier: "tier-2-saas", Scenarios: []string{"beta"}, Resources: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range refs {
		if ref.LogicalID == "vrooli/beta" {
			t.Fatalf("tier-inapplicable descriptor leaked into projection: %+v", ref)
		}
	}
	var managed CredentialRef
	for _, ref := range refs {
		if ref.LogicalID == "vrooli/release-authority" && ref.Field == "rsa-pkcs8-v1" {
			managed = ref
		}
	}
	if managed.Kind != "managed" || managed.Owner != "release-authority" || managed.SourceRef == "" {
		t.Fatalf("managed projection = %+v, want explicit owner/source metadata", managed)
	}
	if !filepath.IsAbs(managed.SourceRef) || len(managed.ConsumerRefs) != 1 || managed.ConsumerRefs[0] != "release metadata signer" {
		t.Fatalf("managed consumer projection = %+v, want absolute source and release consumer", managed)
	}
	if len(managed.Provenance) != 1 || len(managed.Provenance[0].Consumers) != 1 || !filepath.IsAbs(managed.Provenance[0].Consumers[0].SourceRef) {
		t.Fatalf("managed provenance = %+v, want absolute consumer provenance", managed.Provenance)
	}
}

func TestDescriptorsForScopeMergesSharedAddressesWithAllProvenance(t *testing.T) {
	root := writeScopeFixture(t)
	gammaDir := filepath.Join(root, "scenarios", "gamma")
	if err := os.MkdirAll(filepath.Join(gammaDir, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gammaDir, ".vrooli", "service.json"), []byte(`{
  "service": {"name": "gamma"},
  "credentials": {
    "descriptors": [{"logical_id": "vrooli/alpha", "field": "token", "required": false}],
    "consumers": [{"logical_id": "vrooli/alpha", "field": "token", "kind": "external", "consumer": "gamma provider", "source_ref": "api/provider.go:9"}]
  }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	refs, err := DescriptorsForScope(root, Scope{Scenarios: []string{"alpha", "gamma"}})
	if err != nil {
		t.Fatal(err)
	}
	var shared CredentialRef
	for _, ref := range refs {
		if ref.LogicalID == "vrooli/alpha" && ref.Field == "token" {
			shared = ref
		}
	}
	if len(shared.Provenance) != 2 || !shared.Required {
		t.Fatalf("shared credential = %+v, want two provenance entries and aggregate required=true", shared)
	}
	if !reflect.DeepEqual(shared.ConsumerRefs, []string{"alpha session broker", "alpha.session", "gamma provider"}) {
		t.Fatalf("shared consumer refs = %v, want descriptor and owner consumers", shared.ConsumerRefs)
	}
	if len(shared.Provenance[0].Consumers) != 1 || shared.Provenance[0].Consumers[0].Kind != "delegated" || !filepath.IsAbs(shared.Provenance[0].Consumers[0].SourceRef) {
		t.Fatalf("alpha consumer provenance = %+v, want delegated consumer with absolute source", shared.Provenance[0].Consumers)
	}
}

func TestDescriptorsForScopeBindsAddressPatternConsumerProvenance(t *testing.T) {
	root := writeScopeFixture(t)
	backupDir := filepath.Join(root, "scenarios", "backup")
	if err := os.MkdirAll(filepath.Join(backupDir, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backupDir, ".vrooli", "service.json"), []byte(`{
  "service": {"name": "backup"},
  "credentials": {
    "descriptors": [
      {"logical_id": "vrooli/backup/source-a", "field": "passphrase", "required": true},
      {"logical_id": "vrooli/backup/source-b", "field": "passphrase", "required": true}
    ],
    "consumers": [{"address_pattern": "vrooli/backup/{name}:passphrase", "kind": "backup", "consumer": "kopia restore", "source_ref": "api/restore.go:22", "required": true}]
  }
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	refs, err := DescriptorsForScope(root, Scope{Scenarios: []string{"backup"}, Resources: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 2 {
		t.Fatalf("refs = %d, want two concrete backup addresses", len(refs))
	}
	for _, ref := range refs {
		if len(ref.Provenance) != 1 || len(ref.Provenance[0].Consumers) != 1 {
			t.Fatalf("%s:%s provenance = %+v, want one pattern consumer", ref.LogicalID, ref.Field, ref.Provenance)
		}
		consumer := ref.Provenance[0].Consumers[0]
		if consumer.AddressPattern != "vrooli/backup/{name}:passphrase" || consumer.Consumer != "kopia restore" || !consumer.Required {
			t.Fatalf("pattern consumer = %+v", consumer)
		}
		if !filepath.IsAbs(consumer.SourceRef) {
			t.Fatalf("consumer source ref = %q, want absolute", consumer.SourceRef)
		}
	}
}

func TestDescriptorsForScopeRejectsIncompatibleSharedMetadata(t *testing.T) {
	root := writeScopeFixture(t)
	gammaDir := filepath.Join(root, "scenarios", "gamma")
	if err := os.MkdirAll(filepath.Join(gammaDir, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gammaDir, ".vrooli", "service.json"), []byte(`{
  "service": {"name": "gamma"},
  "credentials": {"descriptors": [{"logical_id": "vrooli/alpha", "field": "token", "provisioning": "generated"}]}
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := DescriptorsForScope(root, Scope{Scenarios: []string{"alpha", "gamma"}})
	if err == nil {
		t.Fatal("shared credential metadata conflict returned nil error")
	}
	if !strings.Contains(err.Error(), "incompatible provisioning modes") || !strings.Contains(err.Error(), "service.json") {
		t.Fatalf("conflict error = %q, want both incompatible metadata and source locations", err)
	}
}

// An empty non-nil selection means "none of that kind", which is how a caller
// asks for the project scope by itself.
func TestDescriptorsForScopeProjectOnly(t *testing.T) {
	root := writeScopeFixture(t)
	refs, err := DescriptorsForScope(root, Scope{IncludeProject: true, Scenarios: []string{}, Resources: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	requireAddresses(t, refs, "vrooli/remote-desktop:password", "vrooli/remote-desktop:username")
}

func TestDescriptorsForScopeTreatsMissingProjectManifestAsEmpty(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scenarios"), 0o755); err != nil {
		t.Fatal(err)
	}
	refs, err := DescriptorsForScope(root, Scope{IncludeProject: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 0 {
		t.Fatalf("refs = %v, want none", refs)
	}
}

func TestDiscoverDescriptorsDelegatesWithProjectScope(t *testing.T) {
	root := writeScopeFixture(t)
	discovered, err := DiscoverDescriptors(root)
	if err != nil {
		t.Fatal(err)
	}
	scoped, err := DescriptorsForScope(root, Scope{IncludeProject: true})
	if err != nil {
		t.Fatal(err)
	}
	requireAddresses(t, discovered, addressSet(scoped)...)
}

// TestDescriptorsForScopeAgreesWithCredentialInventory is the conformance test.
// Two enumerations of the same population must not drift: if either one starts
// reading a different set of manifests, this fails instead of producing a quiet
// disagreement between the wizard and the control-plane doctor.
//
// credentialinventory.Collect legitimately returns more than manifests. It adds
// live managed instances that cannot appear in any manifest: the
// release-authority key and device-control entries from the secure store's
// cleartext index, live Vault unseal keys, and live Kopia repository
// passphrases. Those are Collect's job and belong to the recovery inventory,
// not to a manifest resolver, so the comparison subtracts exactly those
// addresses and nothing else.
func TestDescriptorsForScopeAgreesWithCredentialInventory(t *testing.T) {
	root := writeScopeFixture(t)
	collected, err := credentialinventory.Collect(root)
	if err != nil {
		t.Fatal(err)
	}
	managed := map[string]bool{}
	for _, entry := range credentialinventory.ManagedSystemEntries(root) {
		managed[entry.LogicalID+":"+entry.Field] = true
	}
	for _, entry := range resources.LiveVaultUnsealKeyEntries() {
		managed[entry.LogicalID+":"+entry.Field] = true
	}
	for _, entry := range resources.LiveKopiaRepositoryEntries() {
		managed[entry.LogicalID+":"+entry.Field] = true
	}
	manifestDeclared := make([]string, 0, len(collected.Declared))
	for _, entry := range collected.Declared {
		address := string(entry.Identity) + ":" + entry.Field
		if managed[address] {
			continue
		}
		manifestDeclared = append(manifestDeclared, address)
	}
	sort.Strings(manifestDeclared)
	// Guard against a vacuous pass: if Collect stopped reading the fixture at
	// all, an empty-equals-empty comparison would look like agreement.
	if len(manifestDeclared) != 6 {
		t.Fatalf("control-plane manifest population = %v, want the six fixture addresses", manifestDeclared)
	}

	scoped, err := DescriptorsForScope(root, Scope{IncludeProject: true})
	if err != nil {
		t.Fatal(err)
	}
	requireAddresses(t, scoped, manifestDeclared...)
}
