package lightingruntime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/config"
)

func TestNativeIntentAuthorityUsesLogicalHomeOwner(t *testing.T) {
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	controller, err := NewController(home, Owners{})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", filepath.Join(t.TempDir(), "machine-broker-home"))
	policy, err := controller.nativeStatePolicy()
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(home)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := config.PrivatePathOwner(home, info)
	if err != nil {
		t.Fatal(err)
	}
	if policy.OwnerPrincipal != actual || policy.RuntimePrincipal != "" {
		t.Fatal("intent storage authority changed with machine broker HOME", policy)
	}
}

func TestMissingIntentDoesNotResolveOrCreateNativeOwnerHome(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	controller := &Controller{path: filepath.Join(root, "not-created", "intent.json"), home: filepath.Join(root, "missing-operator-home")}
	intent, err := controller.Current()
	if err != nil || intent.Phase != Disabled {
		t.Fatalf("missing intent must remain disabled without owner-directory effects: %+v %v", intent, err)
	}
	if _, err = os.Lstat(filepath.Dir(controller.path)); !os.IsNotExist(err) {
		t.Fatal("missing intent created native state directory", err)
	}
	if _, err = os.Lstat(controller.home); !os.IsNotExist(err) {
		t.Fatal("missing intent created operator namespace", err)
	}
}
