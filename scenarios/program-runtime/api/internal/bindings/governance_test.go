package bindings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDestructiveEffectRequiresGrant(t *testing.T) { // [REQ:PRT-P0-005]
	root := t.TempDir()
	manifest := filepath.Join(root, "cli", "manifest.json")
	if err := os.MkdirAll(filepath.Dir(manifest), 0o700); err != nil {
		t.Fatal(err)
	}
	const content = `{"name":"program-runtime","groups":[{"name":"ops","commands":[{"name":"delete","binding":{"kind":"connect-rpc","service":"BindingRegistryService","method":"ListBindings"},"governance":{"effect":"destructive","run_eligible":true,"permissions":["records:delete"]}}]}]}`
	if err := os.WriteFile(manifest, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := LoadFiles(filepath.Join(repoRoot(t), "packages", "proto", "gen", "descriptor", "image.binpb"), []string{manifest})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Authorize("program-runtime/ops/delete", nil, true); err == nil {
		t.Fatal("unguarded destructive call authorized")
	}
	if err := r.Authorize("program-runtime/ops/delete", []string{"records:delete"}, true); err != nil {
		t.Fatal(err)
	}
}

func TestGovernanceIsReadNotRedefined(t *testing.T) { // [REQ:PRT-P0-005]
	// The public Binding is a projection of manifest values; no second policy
	// table is consulted by the authorization boundary.
	r := fixtureRegistry(t, `{"name":"program-runtime","groups":[{"name":"ops","commands":[{"name":"delete","binding":{"kind":"connect-rpc","service":"BindingRegistryService","method":"ListBindings"},"governance":{"effect":"destructive","run_eligible":true,"permissions":["records:delete"]}}]}]}`)
	b, ok := r.Binding("program-runtime/ops/delete")
	if !ok || b.GetEffect() != "destructive" {
		t.Fatalf("binding=%v ok=%v", b, ok)
	}
}
