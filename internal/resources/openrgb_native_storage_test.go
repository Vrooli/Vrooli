package resources

import (
	"github.com/vrooli/vrooli/internal/lightingruntime"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeOpenRGBGlobalStorageOverrideKeepsAllOwnersInOneNamespace(t *testing.T) {
	base := t.TempDir()
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", base)
	for _, key := range []string{"VROOLI_RESOURCE_CONFIG_ROOT", "VROOLI_RESOURCE_DATA_ROOT", "VROOLI_RESOURCE_CACHE_ROOT", "VROOLI_RESOURCE_LOGS_ROOT", "VROOLI_RESOURCE_STATE_ROOT"} {
		t.Setenv(key, "")
	}
	root, err := OpenRGBNativeResourceRoot()
	if err != nil {
		t.Fatal(err)
	}
	for _, goos := range []string{"darwin", "windows"} {
		paths, handled, err := nativeOpenRGBStoragePathsFor(goos)
		if err != nil || !handled {
			t.Fatalf("native override was delegated away: %s %v", goos, err)
		}
		if paths.StateDir != filepath.Join(root, "state") || paths.ConfigDir != filepath.Join(root, "config") || paths.DataDir != filepath.Join(root, "data") || paths.CacheDir != filepath.Join(root, "cache") || paths.LogsDir != filepath.Join(root, "logs") {
			t.Fatalf("native actors split their root: %+v root=%s", paths, root)
		}
	}
	if _, handled, err := nativeOpenRGBStoragePathsFor("linux"); err != nil || handled {
		t.Fatal("Linux legacy resolver changed", err)
	}
	if _, err := os.Stat(filepath.Join(base, "native")); !os.IsNotExist(err) {
		t.Fatal("readonly storage resolution created namespace", err)
	}
}

func TestNativeOpenRGBPerClassOverridesCannotSplitPrivateNamespace(t *testing.T) {
	base := t.TempDir()
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", base)
	items := []struct{ key, class string }{{"VROOLI_RESOURCE_CONFIG_ROOT", "config"}, {"VROOLI_RESOURCE_DATA_ROOT", "data"}, {"VROOLI_RESOURCE_CACHE_ROOT", "cache"}, {"VROOLI_RESOURCE_LOGS_ROOT", "logs"}, {"VROOLI_RESOURCE_STATE_ROOT", "state"}}
	for _, item := range items {
		t.Setenv(item.key, "")
	}
	root, err := OpenRGBNativeResourceRoot()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		t.Setenv(item.key, filepath.Join(root, item.class))
		if _, handled, err := nativeOpenRGBStoragePathsFor("windows"); err != nil || !handled {
			t.Fatal("redundant canonical class override refused", item.key, err)
		}
		t.Setenv(item.key, filepath.Join(base, "other-"+item.class))
		if _, handled, err := nativeOpenRGBStoragePathsFor("windows"); err == nil || !handled {
			t.Fatal("noncanonical class override admitted", item.key)
		}
		if _, handled, err := nativeOpenRGBStoragePathsFor("linux"); err != nil || handled {
			t.Fatal("Linux class override semantics changed", err)
		}
		t.Setenv(item.key, "")
	}
	if _, err := os.Stat(filepath.Join(base, "native")); !os.IsNotExist(err) {
		t.Fatal("rejected class layout created namespace", err)
	}
}

func TestNativeCapsuleFrozenNamespaceSurvivesChangedAmbientStorageRoot(t *testing.T) {
	c, m, _, _, _ := nativeTransportFixture(t)
	// This test exercises pure namespace metadata, so a short fictional absolute
	// root avoids depending on native TEMP/user-name or test-name path lengths.
	short := filepath.Join(filepath.VolumeName(t.TempDir())+string(filepath.Separator), "vrooli-native-fixture")
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", short)
	selected, err := OpenRGBNativeResourceRoot()
	if err != nil {
		t.Fatal(err)
	}
	m.Runtime.StateDir = filepath.Join(selected, "state")
	m.Invocation, err = renderManagedUpdateInvocation(m.Contract, m.Runtime, "fixture", "role", "private-token")
	if err != nil {
		t.Fatal(err)
	}
	c.RuntimeRoot = m.Invocation.RuntimeRoot
	c.MaterialSHA256 = m.digest()
	frozen, err := openRGBNativeResourceRootFromRuntime(c.RuntimeRoot)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(frozen, "launch", c.ID, "capsule.json")
	fromPath, err := openRGBNativeResourceRootFromCapsulePath(path)
	if err != nil || fromPath != frozen {
		t.Fatal("capsule path lost namespace", fromPath, err)
	}
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", t.TempDir())
	ambient, err := OpenRGBNativeResourceRoot()
	if err != nil || ambient == frozen {
		t.Fatal("fixture did not change ambient root", err)
	}
	if err = m.validate(c); err != nil {
		t.Fatal("closed wrapper incorrectly depended on ambient resource env", err)
	}
	expected, err := openRGBNativeCapsuleFrozenEndpoint(c)
	if err != nil || filepath.Dir(expected) != filepath.Join(frozen, "ipc") {
		t.Fatal("endpoint followed ambient env", expected, err)
	}
	owner := openRGBNativeOwner{SchemaVersion: openRGBNativeOwnerVersion, OwnerPrincipal: m.NativePolicy.Account, RuntimePrincipal: m.NativePolicy.Account, NativeAccount: m.NativePolicy.Account, resourceRoot: frozen}
	if _, err := owner.policyFor(filepath.Join(frozen, "state", "ledger.json")); err != nil {
		t.Fatal("frozen owner path refused", err)
	}
	if _, err := owner.policyFor(filepath.Join(ambient, "state", "ledger.json")); err == nil {
		t.Fatal("frozen owner silently rebound to ambient root")
	}
	m.Runtime.StateDir = filepath.Join(ambient, "state")
	c.MaterialSHA256 = m.digest()
	if err = m.validate(c); err == nil {
		t.Fatal("material state outside capsule namespace admitted")
	}
}

func TestNativeCapsuleRootDerivationRejectsAliasesAndUnboundedRoles(t *testing.T) {
	root := t.TempDir()
	for _, path := range []string{"relative/launch/a/capsule.json", filepath.Join(root, "state", "a", "capsule.json"), filepath.Join(root, "launch", "a", "other.json"), filepath.Join(root, "launch", "a", "..", "a", "capsule.json") + string(filepath.Separator)} {
		if _, err := openRGBNativeResourceRootFromCapsulePath(path); err == nil {
			t.Fatal("invalid capsule namespace admitted", path)
		}
	}
	for _, path := range []string{filepath.Join(root, "runtime"), filepath.Join(root, "runtime", "initial"), filepath.Join(root, "runtime", "unknown", "a"), filepath.Join(root, "runtime", "managed-update", "op"), filepath.Join(root, "runtime", "initial", "a", "extra")} {
		if _, err := openRGBNativeResourceRootFromRuntime(path); err == nil {
			t.Fatal("unbounded runtime namespace admitted", path)
		}
	}
}

func TestNativeEndpointRootUsesFixedUTF8ByteBoundBeforeEffects(t *testing.T) {
	volumeRoot := filepath.VolumeName(t.TempDir()) + string(filepath.Separator)
	for _, root := range []string{filepath.Join(volumeRoot, "Library", "Application Support", "Vrooli", "resources", "openrgb"), filepath.Join(volumeRoot, "ProgramData", "Vrooli", "resources", "openrgb")} {
		if err := validateOpenRGBNativeEndpointRoot(root); err != nil {
			t.Fatal("normal native namespace exceeds endpoint bound", root, err)
		}
	}
	root := filepath.Join(t.TempDir(), strings.Repeat("界", 32))
	if err := validateOpenRGBNativeEndpointRoot(root); err == nil || !strings.Contains(err.Error(), "shorter VROOLI_RESOURCE_STORAGE_ROOT") {
		t.Fatal("overlong Unicode namespace has no useful early diagnostic", err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatal("endpoint validation created namespace", err)
	}
}

func TestNativeEnrolledStorageRejectsChangedAmbientNamespace(t *testing.T) {
	base := t.TempDir()
	enrolled := filepath.Join(base, "root-a")
	other := filepath.Join(base, "root-b")
	selected := filepath.Join(enrolled, "native", "openrgb")
	for _, goos := range []string{"darwin", "windows"} {
		if err := validateNativeOpenRGBStorageEnrollment(goos, selected, enrolled, true, ""); err != nil {
			t.Fatal("enrolled namespace refused", goos, err)
		}
		if err := validateNativeOpenRGBStorageEnrollment(goos, filepath.Join(other, "native", "openrgb"), enrolled, true, ""); err == nil || !strings.Contains(err.Error(), enrolled) || !strings.Contains(err.Error(), "before normal disable or cleanup") {
			t.Fatal("ambient root switch lacks exact restoration diagnostic", goos, err)
		}
		if err := validateNativeOpenRGBStorageEnrollment(goos, selected, enrolled, false, ""); err != nil {
			t.Fatal("first enrollment root choice refused", err)
		}
	}
	if err := validateNativeOpenRGBStorageEnrollment("windows", strings.ToUpper(selected), enrolled, true, ""); err != nil {
		t.Fatal("Windows case-equivalent root refused", err)
	}
	if err := validateNativeOpenRGBStorageEnrollment("darwin", strings.ToUpper(selected), enrolled, true, ""); err == nil {
		t.Fatal("Darwin silently rebound a different spelling")
	}
	if _, err := os.Stat(enrolled); !os.IsNotExist(err) {
		t.Fatal("readonly mismatch validation created namespace", err)
	}
}

func TestNativeEnrolledDefaultRejectsNewGlobalOverride(t *testing.T) {
	base := t.TempDir()
	canonical := filepath.Join(base, "machine-native")
	changed := filepath.Join(base, "new-global", "native", "openrgb")
	if err := validateNativeOpenRGBStorageEnrollment("darwin", canonical, "", true, canonical); err != nil {
		t.Fatal("enrolled default refused", err)
	}
	if err := validateNativeOpenRGBStorageEnrollment("darwin", changed, "", true, canonical); err == nil || !strings.Contains(err.Error(), "unset VROOLI_RESOURCE_STORAGE_ROOT") {
		t.Fatal("enrolled default silently moved", err)
	}
	if err := validateNativeOpenRGBStorageEnrollment("linux", changed, "", true, canonical); err != nil {
		t.Fatal("Linux storage semantics changed", err)
	}
	if err := validateNativeOpenRGBStorageEnrollment("windows", changed, "relative", true, canonical); err == nil {
		t.Fatal("malformed enrolled root accepted")
	}
}

func TestNativeGlobalRootRejectsNoncanonicalRawEnrollmentValueWithoutEffects(t *testing.T) {
	base := t.TempDir()
	for _, root := range []string{"relative", " ", " " + base, base + "\n", base + string(filepath.Separator) + ".."} {
		t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", root)
		if _, err := OpenRGBNativeResourceRoot(); err == nil {
			t.Fatal("raw root incompatible with enrollment accepted", root)
		}
	}
	if _, err := os.Stat(filepath.Join(base, "native")); !os.IsNotExist(err) {
		t.Fatal("invalid root created namespace", err)
	}
}

func TestNativeUnknownStorageEnrollmentIsNotActiveResourceAbsence(t *testing.T) {
	if err := validateNativeOpenRGBUnknownStorageIntent(lightingruntime.Intent{Phase: lightingruntime.Disabled}); err != nil {
		t.Fatal("fresh disabled setup refused", err)
	}
	for _, phase := range []lightingruntime.Phase{lightingruntime.Enabled, lightingruntime.Disabling} {
		if err := validateNativeOpenRGBUnknownStorageIntent(lightingruntime.Intent{Phase: phase}); err == nil || !strings.Contains(err.Error(), "restore its retained operator enrollment") {
			t.Fatal("missing boot definition mistaken for no backend", phase, err)
		}
	}
}
