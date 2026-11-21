package resources

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/vrooli/internal/operatorstate"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
)

func TestOpenRGBAdoptedArtifactInspectionUsesExactNativeProfile(t *testing.T) {
	path := os.Getenv("VROOLI_OPENRGB_ADOPTION_ENVELOPE")
	if path == "" {
		t.Skip("explicit offline current-host envelope qualification only")
	}
	p, err := openRGBReleaseProfileForCurrentHost()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	m, err := manifestpkg.Load(filepath.Join(projectRootForResourcesTest(t), "resources", "openrgb", "resource.json"))
	if err != nil {
		t.Fatal(err)
	}
	writeResourceManifest(t, root, "openrgb", m)
	copyOperatorStateSchemaForTest(t, root)
	c := NewController(root, root)
	c.operatorState = operatorstate.New(operatorstate.Config{RepoRoot: root, StateRoot: filepath.Join(root, "operator-state")})
	choice := operatorstate.ResourceSourceChoice{Mode: "host", Executable: path, Version: openRGBAppImageVersion, SHA256: p.EnvelopeSHA256}
	if err := c.persistManagedServiceSource(t.Context(), "openrgb", choice); err != nil {
		t.Fatal(err)
	}
	actual, err := c.InspectManagedServiceArtifact(t.Context(), "openrgb")
	if err != nil || actual.Path != path || actual.SHA256 != p.EnvelopeSHA256 {
		t.Fatalf("exact adopted native profile not inspectable: %+v %v", actual, err)
	}
	choice.SHA256 = updateTestDigest("different-native-source")
	if err := c.persistManagedServiceSource(t.Context(), "openrgb", choice); err != nil {
		t.Fatal(err)
	}
	if _, err := c.InspectManagedServiceArtifact(t.Context(), "openrgb"); err == nil {
		t.Fatal("changed adopted native identity passed artifact inspection")
	}
}

func TestOpenRGBAdoptionNeverRunsUnqualifiedProgram(t *testing.T) {
	root := t.TempDir()
	m, err := manifestpkg.Load(filepath.Join(projectRootForResourcesTest(t), "resources", "openrgb", "resource.json"))
	if err != nil {
		t.Fatal(err)
	}
	writeResourceManifest(t, root, "openrgb", m)
	bin := t.TempDir()
	marker := filepath.Join(root, "vendor-program-ran")
	path := filepath.Join(bin, "openrgb")
	if err := os.WriteFile(path, []byte("#!/bin/sh\ntouch '"+marker+"'\necho 'OpenRGB 1.1'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	if _, err := NewController(root, t.TempDir()).SetManagedServiceSource(t.Context(), "openrgb", "host"); err == nil {
		t.Fatal("arbitrary matching-version executable adopted")
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("unqualified vendor program was executed during adoption")
	}
}

func TestOpenRGBAdoptionCopiesExactClosureAndPreservesSource(t *testing.T) {
	source := filepath.Join(t.TempDir(), "AppDir")
	if err := os.MkdirAll(filepath.Join(source, "usr", "bin"), 0o700); err != nil {
		t.Fatal(err)
	}
	image := make([]byte, 64)
	copy(image, []byte{0x7f, 'E', 'L', 'F', 2, 1, 1})
	binary.LittleEndian.PutUint16(image[16:], 2)
	binary.LittleEndian.PutUint16(image[18:], 62)
	binary.LittleEndian.PutUint32(image[20:], 1)
	binary.LittleEndian.PutUint16(image[52:], 64)
	payload := filepath.Join(source, openRGBAppDirPayload)
	if err := os.WriteFile(payload, image, 0o700); err != nil {
		t.Fatal(err)
	}
	dependency := filepath.Join(source, "dependency")
	if err := os.WriteFile(dependency, []byte("pinned dependency"), 0o600); err != nil {
		t.Fatal(err)
	}
	tree, err := binaryfetch.TreeDigest(source)
	if err != nil {
		t.Fatal(err)
	}
	p := openRGBReleaseProfile{OS: "linux", Arch: "amd64", Payload: openRGBAppDirPayload, TreeSHA256: tree}
	if err := p.verifyPayloadABI(payload); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "stage")
	if err := copyOpenRGBAdoptedBundle(t.Context(), source, destination, p); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(payload)
	if err != nil || !bytes.Equal(image, after) {
		t.Fatal("adoption modified source executable")
	}
	if err := os.WriteFile(dependency, []byte("changed dependency"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := copyOpenRGBAdoptedBundle(t.Context(), source, filepath.Join(t.TempDir(), "bad"), p); err == nil {
		t.Fatal("standalone payload digest concealed incompatible bundle dependency")
	}
}
