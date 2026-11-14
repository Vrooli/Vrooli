//go:build linux

package resources

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/binaryfetch"
)

func TestOpenRGBMacDerivedManagedAndAdoptedRecovery(t *testing.T) {
	root := os.Getenv("VROOLI_OPENRGB_PROFILE_MEASUREMENT_ROOT")
	if root == "" {
		t.Skip("explicit original Mac archive qualification only")
	}
	for _, arch := range []string{"amd64", "arm64"} {
		for _, adopted := range []bool{false, true} {
			t.Run(arch+map[bool]string{false: "/managed", true: "/adopted"}[adopted], func(t *testing.T) {
				p, _ := openRGBReleaseProfileFor("darwin", arch)
				name := "OpenRGB_1.1_MacOS_Intel_8e16de2.zip"
				if arch == "arm64" {
					name = "OpenRGB_1.1_MacOS_ARM64_8e16de2.zip"
				}
				archive := filepath.Join(root, name)
				if err := binaryfetch.VerifyArtifactContext(t.Context(), archive, "file", p.EnvelopeSHA256); err != nil {
					t.Fatal(err)
				}
				f, err := os.Open(archive)
				if err != nil {
					t.Fatal(err)
				}
				defer f.Close()
				original := filepath.Join(t.TempDir(), "OpenRGB.app")
				if err = os.Mkdir(original, 0o700); err != nil {
					t.Fatal(err)
				}
				if err = extractOpenRGBZip(t.Context(), f, original, p); err != nil {
					t.Fatal(err)
				}
				source, final, p, owner, extractions := openRGBReleasePreparationFixture(t, p, original, archive, adopted)
				sync := owner.syncDir
				lost := true
				owner.syncDir = func(path string) error {
					if path == filepath.Dir(final) && lost {
						lost = false
						return errors.New("injected unknown publication acknowledgement")
					}
					return sync(path)
				}
				if _, err = prepareOpenRGBAppImageLocked(t.Context(), source, final, p.servingTreeSHA256(), p.PayloadSHA256, owner); !errors.Is(err, ErrOpenRGBStageCommitUnknown) {
					t.Fatalf("lost ack not preserved: %v", err)
				}
				bundle, err := prepareOpenRGBAppImageLocked(t.Context(), source, final, p.servingTreeSHA256(), p.PayloadSHA256, owner)
				if err != nil {
					t.Fatal(err)
				}
				if *extractions != 1 {
					t.Fatal("verified Mac recovery re-extracted candidate")
				}
				if err = binaryfetch.VerifyArtifactContext(t.Context(), original, "dir", p.TreeSHA256); err != nil {
					t.Fatalf("adoption changed original: %v", err)
				}
				if err = binaryfetch.VerifyArtifactContext(t.Context(), bundle, "dir", p.servingTreeSHA256()); err != nil {
					t.Fatal(err)
				}
				if err = p.verifyServingStageLayout(t.Context(), final); err != nil {
					t.Fatal(err)
				}
				if err = p.verifyServingProfile(t.Context(), final, source); err != nil {
					t.Fatal(err)
				}
				if _, err = p.darwinLoaderSystemImages(t.Context(), bundle); err != nil {
					t.Fatal(err)
				}
				if err = os.Chmod(final, 0o700); err != nil {
					t.Fatal(err)
				}
				if err = os.Mkdir(filepath.Join(final, "lib"), 0o700); err != nil {
					t.Fatal(err)
				}
				if err = os.Chmod(final, 0o555); err != nil {
					t.Fatal(err)
				}
				if _, err = prepareOpenRGBAppImageLocked(t.Context(), source, final, p.servingTreeSHA256(), p.PayloadSHA256, owner); err == nil {
					t.Fatal("undeclared adjacent plugin search directory accepted on recovery")
				}
			})
		}
	}
}
