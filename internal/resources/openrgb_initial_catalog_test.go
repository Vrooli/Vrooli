package resources

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

// Keep the real staged-selection -> initial-preparation boundary in this test:
// returning a fake backend would skip the catalog/serving-config distinction.
func TestOpenRGBInitialPreparationKeepsSourceCatalogSeparateFromServingConfig(t *testing.T) {
	for _, serving := range []string{"inline", "absolute-private-file"} {
		t.Run(serving, func(t *testing.T) {
			c, manifest, selection := openRGBHeadlessFixture(t)
			if _, err := c.operatorState.Apply(t.Context(), []byte(`{"resources":{"openrgb":{"enabled":true,"source":{"mode":"managed"}}}}`)); err != nil {
				t.Fatal(err)
			}
			artifact, err := managedServiceArtifactPath(c, manifest)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Dir(artifact), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(artifact, []byte("fixture-image"), 0o700); err != nil {
				t.Fatal(err)
			}
			owner := NewOpenRGBHeadless(c, nil)
			launch, err := owner.inspectSetupSelection(context.Background(), selection, true)
			if err != nil {
				t.Fatal(err)
			}
			sourceFile := manifest.ManagedService.Config.ContentFile
			sourceCatalog, err := readManagedUpdateConfigFile(filepath.Join(c.Root, "resources", openRGBResourceName), sourceFile)
			if err != nil {
				t.Fatal(err)
			}
			if launch.manifest.ManagedService.Config.ContentFile != "" || !bytes.Equal([]byte(launch.manifest.ManagedService.Config.Content), sourceCatalog) {
				t.Fatal("normal setup did not freeze the source catalog into its launch manifest")
			}
			if serving == "absolute-private-file" {
				file := filepath.Join(t.TempDir(), "selected-serving.json")
				if err := os.WriteFile(file, []byte(`{"Detectors":{"detectors":{"not-a-catalog":true}}}`), 0o600); err != nil {
					t.Fatal(err)
				}
				service := *launch.manifest.ManagedService
				cfg := *service.Config
				cfg.Content, cfg.ContentFile = "", file
				service.Config = &cfg
				launch.manifest.ManagedService = &service
				if _, err := readManagedUpdateConfigFile(filepath.Join(c.Root, "resources", openRGBResourceName), file); err == nil {
					t.Fatal("absolute serving path became a resource-relative source asset")
				}
			}
			attempt, err := c.prepareOpenRGBInitialAttempt(t.Context(), launch)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(attempt.Catalog, sourceCatalog) || attempt.Contract.ConfigFile != sourceFile || attempt.Contract.Manifest.ManagedService.Config.ContentFile != sourceFile {
				t.Fatal("initial contract lost its distinct source catalog provenance")
			}
			if bytes.Equal(attempt.Contract.ConfigBytes, sourceCatalog) {
				t.Fatal("effective selected serving settings remained the all-disabled catalog")
			}
			catalogMap, err := decodeOpenRGBDetectorCatalog(sourceCatalog)
			if err != nil {
				t.Fatal(err)
			}
			names, err := selectedOpenRGBDetectors(*attempt.DevicePolicy, catalogMap)
			if err != nil {
				t.Fatal(err)
			}
			if err := verifyOpenRGBExecDetectors(attempt.Contract.ConfigBytes, catalogMap, names); err != nil {
				t.Fatal(err)
			}
			if attempt.Policy.ID != selection.DevicePolicyID || attempt.Policy.DetectorCatalogSHA256 != updateTestDigest(string(sourceCatalog)) {
				t.Fatal("prepared claim no longer binds the selected policy/source catalog")
			}
		})
	}
}
