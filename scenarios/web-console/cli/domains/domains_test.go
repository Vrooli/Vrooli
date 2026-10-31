package domains

import (
	"encoding/json"
	"os"
	"sort"
	"testing"
)

// codeRegisteredGroups are the groups that cannot be expressed as
// one-command-per-method manifest bindings and are therefore registered in
// code. Their service methods are declared in cli/manifest.json's omitted[].
var codeRegisteredGroups = []string{"hooks"}

// TestSubcommandGroupsCoverTheManifest asserts coverage rather than a count.
// A pinned number tells you that something changed but not what, and it fails
// for a correct addition exactly as loudly as for a dropped group. Comparing
// the registered names against the manifest's own groups names the difference.
func TestSubcommandGroupsCoverTheManifest(t *testing.T) {
	manifest, err := os.ReadFile("../manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var declared struct {
		Groups []struct {
			Name string `json:"name"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(manifest, &declared); err != nil {
		t.Fatal(err)
	}
	if len(declared.Groups) == 0 {
		t.Fatal("cli/manifest.json declares no command groups")
	}

	want := make(map[string]struct{}, len(declared.Groups)+len(codeRegisteredGroups))
	for _, group := range declared.Groups {
		want[group.Name] = struct{}{}
	}
	for _, name := range codeRegisteredGroups {
		want[name] = struct{}{}
	}

	groups, err := SubcommandGroups(nil, manifest)
	if err != nil {
		t.Fatal(err)
	}
	got := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		got[group.Name] = struct{}{}
	}

	for _, name := range sortedKeys(want) {
		if _, ok := got[name]; !ok {
			t.Errorf("group %q is declared but not registered", name)
		}
	}
	for _, name := range sortedKeys(got) {
		if _, ok := want[name]; !ok {
			t.Errorf("group %q is registered but neither declared in the manifest nor listed as code-registered", name)
		}
	}

	if CommandGroups(nil) != nil {
		t.Fatal("CommandGroups should be empty after manifest migration")
	}
}

func sortedKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
