package resources

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/binaryfetch"
)

func TestWindowsHIDServingRecipeMatchesDeclaredCanonicalSource(t *testing.T) {
	p, err := openRGBReleaseProfileFor("windows", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	if err = p.validate(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(projectRootForResourcesTest(t), "resources/openrgb/config/windows-hid-serving-recipe.json"))
	if err != nil {
		t.Fatal(err)
	}
	var declared any
	if err = json.Unmarshal(raw, &declared); err != nil {
		t.Fatal(err)
	}
	canonical, err := json.Marshal(declared)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := p.servingRecipeBytes()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(canonical, expected) {
		t.Fatal("declared serving recipe differs from the compiled pinned recipe")
	}
	if p.servingTreeSHA256() == p.TreeSHA256 {
		t.Fatal("derived serving tree was confused with complete official source")
	}
	for _, bad := range []func(*openRGBReleaseProfile){func(p *openRGBReleaseProfile) { p.ServingTreeSHA256 = p.TreeSHA256 }, func(p *openRGBReleaseProfile) { p.ServingRecipeID = "other-recipe" }, func(p *openRGBReleaseProfile) { p.TreeSHA256 = updateTestDigest("other-source") }} {
		changed := p
		bad(&changed)
		if changed.validate() == nil {
			t.Fatal("unadmitted recipe/source identity passed validation")
		}
	}
}

func windowsHIDServingSourceFixture(t *testing.T) (openRGBReleaseProfile, string, string) {
	t.Helper()
	root := os.Getenv("VROOLI_OPENRGB_PROFILE_MEASUREMENT_ROOT")
	if root == "" {
		t.Skip("explicit retained Windows profile qualification only")
	}
	p, err := openRGBReleaseProfileFor("windows", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(root, "OpenRGB_1.1_Windows_64_8e16de2.zip")
	if err = binaryfetch.VerifyArtifactContext(t.Context(), archive, "file", p.EnvelopeSHA256); err != nil {
		t.Fatal(err)
	}
	bundle := filepath.Join(t.TempDir(), "bundle")
	if err = os.Mkdir(bundle, 0o700); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(archive)
	if err != nil {
		t.Fatal(err)
	}
	err = extractOpenRGBZip(t.Context(), f, bundle, p)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	return p, bundle, archive
}

func TestWindowsHIDDerivationProvesSourceBeforeAnyOmission(t *testing.T) {
	p, bundle, _ := windowsHIDServingSourceFixture(t)
	path := filepath.Join(bundle, "PawnIOLib.dll")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	changed := append([]byte(nil), original...)
	changed[len(changed)-1] ^= 1
	if err = os.WriteFile(path, changed, 0o600); err != nil {
		t.Fatal(err)
	}
	if p.deriveServingTree(t.Context(), bundle) == nil {
		t.Fatal("modified source was accepted before derivation")
	}
	for name := range openRGBPawnIOSourceModules() {
		if _, err = os.Lstat(filepath.Join(bundle, name)); err != nil {
			t.Fatalf("source refusal removed %s: %v", name, err)
		}
	}
	if err = os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	if err = p.deriveServingTree(t.Context(), bundle); err != nil {
		t.Fatal(err)
	}
	if err = binaryfetch.VerifyArtifactContext(t.Context(), bundle, "dir", p.ServingTreeSHA256); err != nil {
		t.Fatal(err)
	}
	if err = p.verifyBundleABI(t.Context(), bundle); err != nil {
		t.Fatal(err)
	}
	if err = p.verifyVendorStartupMode(filepath.Join(bundle, p.Payload)); err != nil {
		t.Fatal(err)
	}
	// Plugin subdirectories are part of the closed tree too; counting all
	// regular files confirms that every one of the sixteen PE images remains.
	count := 0
	if err = filepath.WalkDir(bundle, func(_ string, d os.DirEntry, e error) error {
		if e == nil && d.Type().IsRegular() {
			count++
		}
		return e
	}); err != nil {
		t.Fatal(err)
	}
	if count != 16 {
		t.Fatalf("derived serving closure contains %d files, want all16 unchanged PE images", count)
	}
	for name := range openRGBPawnIOSourceModules() {
		if _, err = os.Lstat(filepath.Join(bundle, name)); !os.IsNotExist(err) {
			t.Fatalf("derived module %s remains: %v", name, err)
		}
	}
	if err = p.deriveServingTree(t.Context(), bundle); err == nil {
		t.Fatal("already-derived partial source was silently accepted as complete official input")
	}
	if err = os.WriteFile(filepath.Join(bundle, "LpcIO.bin"), []byte("unexpected module"), 0o600); err != nil {
		t.Fatal(err)
	}
	if p.verifyServingModulesAbsent(t.Context(), bundle) == nil {
		t.Fatal("present module did not invalidate HID serving profile")
	}
}

func TestWindowsAdoptedHIDDerivationPreservesOriginalAndBindsMetadata(t *testing.T) {
	p, source, _ := windowsHIDServingSourceFixture(t)
	root := t.TempDir()
	bundle := filepath.Join(root, "AppDir")
	if err := copyOpenRGBAdoptedBundle(t.Context(), source, bundle, p); err != nil {
		t.Fatal(err)
	}
	if err := p.deriveServingTree(t.Context(), bundle); err != nil {
		t.Fatal(err)
	}
	if err := binaryfetch.VerifyArtifactContext(t.Context(), source, "dir", p.TreeSHA256); err != nil {
		t.Fatalf("adoption changed original: %v", err)
	}
	selected := ManagedServiceArtifact{Resource: openRGBResourceName, Version: openRGBAppImageVersion, SHA256: p.PayloadSHA256}
	expected, err := p.servingProfileBytes(selected)
	if err != nil {
		t.Fatal(err)
	}
	record := filepath.Join(root, openRGBSealedServingProfile)
	if err = os.WriteFile(record, expected, 0o444); err != nil {
		t.Fatal(err)
	}
	if err = p.verifyServingProfile(t.Context(), root, selected); err != nil {
		t.Fatal(err)
	}
	other := selected
	other.SHA256 = p.EnvelopeSHA256
	if p.verifyServingProfile(t.Context(), root, other) == nil {
		t.Fatal("serving metadata accepted another selected source identity")
	}
	if err = os.Chmod(record, 0o600); err != nil {
		t.Fatal(err)
	}
	expected[len(expected)-2] ^= 1
	if err = os.WriteFile(record, expected, 0o444); err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(record, 0o444); err != nil {
		t.Fatal(err)
	}
	if p.verifyServingProfile(t.Context(), root, selected) == nil {
		t.Fatal("changed derivation metadata passed verification")
	}
}
