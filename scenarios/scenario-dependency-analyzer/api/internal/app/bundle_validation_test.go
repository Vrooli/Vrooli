package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBundleSchemaSamplesValidate(t *testing.T) {
	samples := []string{
		"desktop-happy.json",
		"desktop-playwright.json",
	}
	for _, sample := range samples {
		t.Run(sample, func(t *testing.T) {
			path := filepath.Join("..", "..", "..", "..", "..", "docs", "deployment", "examples", "manifests", sample)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read sample %s: %v", sample, err)
			}
			if err := validateDesktopBundleManifestBytes(data); err != nil {
				t.Fatalf("sample %s did not validate: %v", sample, err)
			}
		})
	}
}

func TestBundleSchemaRejectsInvalidManifest(t *testing.T) {
	invalid := []byte(`{"schema_version":"v0.1","target":"desktop","services":[]}`)
	if err := validateDesktopBundleManifestBytes(invalid); err == nil {
		t.Fatalf("expected validation error for incomplete manifest")
	}
}
