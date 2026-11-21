package resources

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vrooli/binaryfetch"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
)

func TestNativeOpenRGBAcquisitionNeverCollectsGPUOrOperatorPreference(t *testing.T) {
	manifest, err := manifestpkg.Load(filepath.Join(projectRootForResourcesTest(t), "resources", openRGBResourceName, "resource.json"))
	if err != nil {
		t.Fatal(err)
	}
	hostile := func(context.Context) (binaryfetch.Facts, error) {
		t.Fatal("native OpenRGB inspected generic GPU inventory or machine-home preferences")
		return nil, nil
	}
	for _, target := range [][2]string{{"darwin", "amd64"}, {"darwin", "arm64"}, {"windows", "amd64"}} {
		selected, facts, err := managedServiceAcquisitionTargetWithFactSource(t.Context(), manifest, target[0], target[1], hostile)
		profile, profileErr := openRGBReleaseProfileFor(target[0], target[1])
		if err != nil || profileErr != nil || selected.SHA256 != profile.EnvelopeSHA256 || len(facts) != 2 || facts["os"] != target[0] || facts["arch"] != target[1] {
			t.Fatalf("static native target selection failed: %+v %+v %v", selected, facts, err)
		}
	}
	unsupported, facts, err := managedServiceAcquisitionTargetWithFactSource(t.Context(), manifest, "windows", "arm64", hostile)
	if err == nil || !strings.Contains(err.Error(), "Windows ARM64") || unsupported.Unsupported == "" || len(facts) != 2 {
		t.Fatalf("Windows ARM64 did not retain explicit no-probe refusal: %+v %+v %v", unsupported, facts, err)
	}
	manifest.ManagedService.Acquisition.Targets[0].When = map[string]string{"accel.vendor": "nvidia"}
	if _, _, err := managedServiceAcquisitionTargetWithFactSource(t.Context(), manifest, "darwin", "arm64", hostile); err == nil {
		t.Fatal("native OpenRGB accepted a hardware-dependent acquisition profile")
	}
}

func TestNativeOpenRGBFactIsolationPreservesOtherAcquisitionOwners(t *testing.T) {
	for _, target := range []struct{ resource, hostOS string }{{openRGBResourceName, "linux"}, {"other-resource", "darwin"}, {"other-resource", "windows"}} {
		manifest := managedServiceTestManifest(strings.Repeat("a", 64))
		manifest.Name = target.resource
		manifest.ManagedService.Acquisition = &binaryfetch.Acquisition{Kind: "url", Targets: []binaryfetch.AcquisitionTarget{{When: map[string]string{"accel.vendor": "nvidia"}, URL: "https://example.invalid/gpu", SHA256: strings.Repeat("a", 64)}}}
		called := 0
		selected, _, err := managedServiceAcquisitionTargetWithFactSource(t.Context(), manifest, target.hostOS, "amd64", func(context.Context) (binaryfetch.Facts, error) {
			called++
			return binaryfetch.Facts{"accel.vendor": "nvidia"}, nil
		})
		if err != nil || selected.URL == "" || called != 1 {
			t.Fatalf("existing acquisition owner changed: %+v calls=%d err=%v", target, called, err)
		}
	}
}

func TestManagedServiceArtifactSnapshotRejectsChangedManifestBeforeAcquisition(t *testing.T) {
	root := t.TempDir()
	manifest := managedServiceTestManifest(strings.Repeat("a", 64))
	manifest.ManagedService.Acquisition = &binaryfetch.Acquisition{Kind: "url", Targets: []binaryfetch.AcquisitionTarget{{URL: "https://example.invalid/never-fetch", SHA256: strings.Repeat("a", 64), Layout: "file", Archive: "none"}}}
	writeResourceManifest(t, root, manifest.Name, manifest)
	path := filepath.Join(root, "resources", manifest.Name, "resource.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	admitted := fmt.Sprintf("%x", sha256.Sum256(raw))
	changed := strings.ReplaceAll(string(raw), strings.Repeat("a", 64), strings.Repeat("b", 64))
	if err := os.WriteFile(path, []byte(changed), 0600); err != nil {
		t.Fatal(err)
	}
	controller := NewController(root, filepath.Join(root, "home"))
	if _, err := controller.StageManagedServiceArtifactSnapshot(context.Background(), manifest.Name, manifest.ManagedService.Artifact.Version, admitted); err == nil || !strings.Contains(err.Error(), "manifest changed") {
		t.Fatalf("same-version declaration change crossed admission: %v", err)
	}
	if _, err := controller.StageManagedServiceArtifactSnapshot(context.Background(), manifest.Name, manifest.ManagedService.Artifact.Version, ""); err == nil {
		t.Fatal("missing source admission accepted")
	}
}

func TestOpenRGBExactDownloadProfileHasClosedByteBound(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("exact envelope profile is linux-amd64")
	}
	manifest := managedServiceTestManifest(openRGBAppImageSHA256)
	manifest.Name = openRGBResourceName
	manifest.ManagedService.Artifact.Version = openRGBAppImageVersion
	manifest.ManagedService.Acquisition = &binaryfetch.Acquisition{Kind: "url"}
	target := binaryfetch.AcquisitionTarget{URL: "https://example.invalid/openrgb", SHA256: openRGBAppImageSHA256, Layout: "file"}
	size, bounded := managedServiceExactDownloadBytes(manifest, target)
	if !bounded || size != 37528768 {
		t.Fatalf("missing exact release bound: %d %v", size, bounded)
	}
	for _, change := range []func(*binaryfetch.AcquisitionTarget){
		func(target *binaryfetch.AcquisitionTarget) { target.SHA256 = strings.Repeat("a", 64) },
		func(target *binaryfetch.AcquisitionTarget) { target.Archive = "tar.gz" },
		func(target *binaryfetch.AcquisitionTarget) { target.Layout = "dir" },
	} {
		altered := target
		change(&altered)
		if _, bounded := managedServiceExactDownloadBytes(manifest, altered); bounded {
			t.Fatal("changed acquisition shape inherited envelope bound")
		}
	}
}
