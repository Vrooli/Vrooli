//go:build linux

package resources

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/binaryfetch"
)

func TestOpenRGBARM64OfflineExtraction(t *testing.T) {
	root := os.Getenv("VROOLI_OPENRGB_PROFILE_MEASUREMENT_ROOT")
	if root == "" {
		t.Skip("explicit offline acquired release proof")
	}
	p, _ := openRGBReleaseProfileFor("linux", "arm64")
	archive := filepath.Join(root, "OpenRGB_1.1_arm64_8e16de2.AppImage")
	if err := binaryfetch.VerifyArtifactContext(t.Context(), archive, "file", p.EnvelopeSHA256); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(archive)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	dir := filepath.Join(t.TempDir(), "AppDir")
	if err := extractOpenRGBStageForProfile(t.Context(), file, dir, p); err != nil {
		t.Fatal(err)
	}
	if err := binaryfetch.VerifyArtifactContext(t.Context(), dir, "dir", p.TreeSHA256); err != nil {
		t.Fatal(err)
	}
	if err := p.verifyPayloadABI(filepath.Join(dir, p.Payload)); err != nil {
		t.Fatal(err)
	}
}
