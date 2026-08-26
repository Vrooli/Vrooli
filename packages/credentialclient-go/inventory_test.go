package credentialclient

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/vrooli/vrooli/internal/credentialinventory"
	"github.com/vrooli/vrooli/internal/resources"
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
    {"logical_id": "vrooli/remote-desktop", "field": "username", "label": "Remote desktop username", "required": false},
    {"logical_id": "vrooli/remote-desktop", "field": "password", "label": "Remote desktop password", "required": false}
  ]}
}`)
	write(filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), `{
  "service": {"name": "alpha", "description": "Alpha"},
  "credentials": {"descriptors": [{"logical_id": "vrooli/alpha", "field": "token", "required": true}]}
}`)
	write(filepath.Join(root, "scenarios", "beta", ".vrooli", "service.json"), `{
  "service": {"name": "beta", "description": "Beta"},
  "credentials": {"descriptors": [{"logical_id": "vrooli/beta", "field": "token", "required": false}]}
}`)
	write(filepath.Join(root, "resources", "one", "resource.json"), `{
  "name": "one", "display_name": "One", "description": "One", "category": "general",
  "driver": "manual",
  "cli": {"enabled": false},
  "credentials": {"descriptors": [{"logical_id": "vrooli/one", "field": "password", "required": true}]}
}`)
	write(filepath.Join(root, "resources", "two", "resource.json"), `{
  "name": "two", "display_name": "Two", "description": "Two", "category": "general",
  "driver": "manual",
  "cli": {"enabled": false},
  "credentials": {"descriptors": [{"logical_id": "vrooli/two", "field": "password", "required": false}]}
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
		if ref.LogicalID != "vrooli/remote-desktop" {
			continue
		}
		if ref.Resource != ProjectScopeOwner {
			t.Fatalf("project descriptor owner = %q, want %q", ref.Resource, ProjectScopeOwner)
		}
		if ref.Label == "" {
			t.Fatalf("project descriptor %s carries no label", ref.Field)
		}
	}
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
