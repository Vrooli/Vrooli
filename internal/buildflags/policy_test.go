package buildflags

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadAndSelectPolicy(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".vrooli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "repo-contract.json"), []byte(`{"build.go_flags":{"develop":["-trimpath"],"distribution":["-trimpath","-buildvcs=false"],"scenario":["-trimpath"],"rationale":"one policy"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	policy, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := policy.For("distribution"), []string{"-trimpath", "-buildvcs=false"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("distribution flags = %v, want %v", got, want)
	}
	got := policy.For("scenario")
	got[0] = "mutated"
	if policy.Scenario[0] != "-trimpath" {
		t.Fatal("For returned the policy's backing slice")
	}
}

func TestLoadRejectsUnsupportedFlag(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".vrooli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "repo-contract.json"), []byte(`{"build.go_flags":{"develop":["-race"]}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil {
		t.Fatal("expected unsupported build flag to be rejected")
	}
}
