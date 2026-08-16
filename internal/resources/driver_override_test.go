package resources

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManagedManifestHasNoDriverOverride(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, "..", "..", "resources", "ollama", "resource.json")
	controller := NewController(filepath.Join(root, "..", ".."), t.TempDir())

	t.Setenv("VROOLI_RESOURCE_DRIVER", "docker-service")
	manifest, err := controller.LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("default manifest: %v", err)
	}
	if manifest.Driver != "managed-service" || manifest.Template != "managed-service" {
		t.Fatalf("default driver/template = %q/%q", manifest.Driver, manifest.Template)
	}

	if manifest.Driver != "managed-service" || manifest.Template != "managed-service" {
		t.Fatalf("driver/template = %q/%q", manifest.Driver, manifest.Template)
	}
}
