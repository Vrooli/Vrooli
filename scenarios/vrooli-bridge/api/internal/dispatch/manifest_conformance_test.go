package dispatch

import (
	"bytes"
	"strings"
	"testing"

	"github.com/vrooli/api-core/scopecatalog"
	repocontract "github.com/vrooli/repo-contract-go"
)

// [REQ:BRG-P0-004] The dispatch vocabulary is a projection of the shared CLI
// catalog. This test intentionally does not enumerate notification-hub (or
// any other scenario) in bridge source: adding a run-eligible catalog command
// must be enough to admit its typed verb, while its effect and verb grants are
// both required at authorization time.
func TestManifestConformsToSharedScopeCatalog(t *testing.T) {
	if bytes.Contains(dispatchManifestJSON, []byte("typed_bindings")) {
		t.Fatal("dispatch manifest must not carry a hand-maintained typed binding list")
	}
	manifest, _, err := BuildManifest()
	if err != nil {
		t.Fatal(err)
	}
	root, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := scopecatalog.Build(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, scope := range catalog.Scopes {
		if !scope.RunEligible || strings.TrimSpace(scope.Command) == "" {
			continue
		}
		verb := scope.Verb()
		if !inManifest(verb, manifest) {
			t.Fatalf("run-eligible catalog command %q is absent from the bridge manifest", verb)
		}
		matched := false
		for _, entry := range manifest {
			if manifestEntryVerb(entry) != verb {
				continue
			}
			requirements := manifestEntryScopes(entry)
			if contains(requirements, "vrooli-bridge:"+string(scope.Effect)) && contains(requirements, scope.Value) {
				matched = true
				break
			}
		}
		if !matched {
			t.Fatalf("catalog command %q lacks paired transport and namespace requirements", verb)
		}
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

// Web Console relays these exact argv shapes. The check needs only read; every
// action needs write. [REQ:BRG-P0-004]
func TestHarnessUpdateRelayIsAdmittedByItsDeclaredEffect(t *testing.T) {
	manifest, _, err := BuildManifest()
	if err != nil {
		t.Fatal(err)
	}
	scopesFor := func(verb string) []string {
		for _, entry := range manifest {
			if manifestEntryVerb(entry) == verb {
				return manifestEntryScopes(entry)
			}
		}
		t.Fatalf("%q is not in the manifest", verb)
		return nil
	}
	read, write := scopesFor("resource update-check"), scopesFor("resource update")
	if !contains(read, "vrooli-bridge:read") || contains(read, "vrooli-bridge:write") {
		t.Fatalf("update-check scopes = %v", read)
	}
	if !contains(write, "vrooli-bridge:write") {
		t.Fatalf("update scopes = %v", write)
	}
	for _, args := range [][]string{
		{"codex", "--json", "--on-behalf-of", "op@example.com"},
		{"codex", "--action", "use-managed-copy", "--json", "--on-behalf-of", "op@example.com"},
		{"codex", "--action", "link-managed-copy", "--json", "--on-behalf-of", "op@example.com"},
		{"codex", "--action", "rollback", "--json", "--on-behalf-of", "op@example.com"},
		{"codex", "--action", "refresh-catalog", "--json", "--on-behalf-of", "op@example.com"},
	} {
		if err := Allow(Job{Scenario: "vrooli", Verb: "resource update", Args: args}, write, manifest); err != nil {
			t.Errorf("write-scoped node refused %v: %v", args, err)
		}
		if err := Allow(Job{Scenario: "vrooli", Verb: "resource update", Args: args}, read, manifest); err == nil {
			t.Errorf("read-scoped node admitted %v", args)
		}
	}
	if err := Allow(Job{Scenario: "vrooli", Verb: "resource update-check", Args: []string{"codex", "--json"}}, read, manifest); err != nil {
		t.Errorf("read-scoped node refused the check: %v", err)
	}
}
