package resources

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/vrooli/vrooli/internal/config"
)

func TestAdversarialBNativeMaterializationRejectsForeignStateBeforeFilesystem(t *testing.T) {
	_, m, _, _, _ := nativeTransportFixture(t)
	rootB := filepath.Join(t.TempDir(), "never-create-root-b")
	m.Runtime.StateDir = filepath.Join(rootB, "state")
	if err := materializeOpenRGBNativeCapsule(m, m.Invocation); err == nil {
		t.Fatal("foreign namespace materialized")
	}
	if _, err := os.Lstat(rootB); !os.IsNotExist(err) {
		t.Fatal("rejected namespace touched filesystem", err)
	}
}

// This fixture exercises real native ACL/file publication, but only in a
// disposable tree. A Linux pass cannot qualify the native ACL implementation.
func TestAdversarialBNativeCustomRootMaterializationWritesOnlyFrozenTree(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		t.Skip("requires actual native private-path ACL implementation")
	}
	_, m, _, _, _ := nativeTransportFixture(t)
	root := filepath.Join(t.TempDir(), "custom-native")
	info, err := os.Stat(filepath.Dir(root))
	if err != nil {
		t.Fatal(err)
	}
	principal, err := config.PrivatePathOwner(filepath.Dir(root), info)
	if err != nil {
		t.Fatal(err)
	}
	owner := openRGBNativeOwner{SchemaVersion: openRGBNativeOwnerVersion, OwnerPrincipal: principal, RuntimePrincipal: principal, NativeAccount: principal, resourceRoot: root}
	if err := owner.validate(); err != nil {
		t.Skip("fixture requires an ordinary native operator:", err)
	}
	policy, err := owner.policyFor(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := config.ConfigurePrivateDirectoryAccess(root, policy); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(owner)
	if err != nil {
		t.Fatal(err)
	}
	policy, err = owner.policyFor(filepath.Join(root, "owner.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := config.WritePrivateFileDurable(filepath.Join(root, "owner.json"), data, 0600, policy); err != nil {
		t.Fatal(err)
	}
	m.Runtime.StateDir = filepath.Join(root, "state")
	m.Invocation, err = renderManagedUpdateInvocation(m.Contract, m.Runtime, "fixture", "role", "private-token")
	if err != nil {
		t.Fatal(err)
	}
	// The wrapper starts without inherited resource namespace metadata.
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", "")
	if err := materializeOpenRGBNativeCapsule(m, m.Invocation); err != nil {
		t.Fatal("custom-root materialization depended on ambient root", err)
	}
	actual, err := os.ReadFile(m.Invocation.ConfigPath)
	if err != nil || string(actual) != m.Invocation.ConfigContent {
		t.Fatal("configuration did not reach frozen runtime", err)
	}
	for _, path := range nativeCapsuleRuntimeDirs(m.Invocation) {
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() || !nativeCapsulePathWithin(root, path) {
			t.Fatal("private runtime directory missing or outside root", path, err)
		}
	}
}

func TestAdversarialBNativeCustomRootMaterialIsClosedOverEmptyAmbientEnv(t *testing.T) {
	c, m, _, _, _ := nativeTransportFixture(t)
	rootA := filepath.Join(t.TempDir(), "custom-native")
	m.Runtime.StateDir = filepath.Join(rootA, "state")
	var err error
	m.Invocation, err = renderManagedUpdateInvocation(m.Contract, m.Runtime, "fixture", "role", "private-token")
	if err != nil {
		t.Fatal(err)
	}
	c.RuntimeRoot = m.Invocation.RuntimeRoot
	c.MaterialSHA256 = m.digest()
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", "")
	for _, key := range []string{"HOME", "APPDATA", "LOCALAPPDATA", "XDG_CONFIG_HOME", "RESOURCE_CONFIG_DIR", "RESOURCE_STATE_DIR"} {
		t.Setenv(key, "")
	}
	if err := m.validate(c); err != nil {
		t.Fatal("frozen invocation followed empty ambient environment", err)
	}
	for _, path := range nativeCapsuleRuntimeDirs(m.Invocation) {
		if !nativeCapsulePathWithin(c.RuntimeRoot, path) {
			t.Fatal("rendered private directory escaped custom runtime", path)
		}
	}
	for _, env := range [][]string{nil, {}, {managedServiceOwnershipTokenEnv + "=wrong-token"}, append(slices.Clone(m.Invocation.Env), m.Invocation.Env[0])} {
		copy := m
		copy.Invocation.Env = env
		capsule := c
		capsule.MaterialSHA256 = copy.digest()
		if err := copy.validate(capsule); err == nil {
			t.Fatal("missing, wrong or ambiguous explicit environment admitted", env)
		}
	}
	if _, err := os.Lstat(rootA); !os.IsNotExist(err) {
		t.Fatal("metadata validation materialized custom runtime", err)
	}
}
