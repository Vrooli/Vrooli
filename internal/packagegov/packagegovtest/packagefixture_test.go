package packagefixture

import (
	"path/filepath"
	"testing"

	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
	packagegov "github.com/vrooli/vrooli/internal/packagegov"
)

func TestPackageManifestUsesTypedDefaults(t *testing.T) {
	manifest := PackageManifest("alpha")
	if manifest.Schema != "schemas/package.schema.json" {
		t.Fatalf("schema = %q", manifest.Schema)
	}
	if manifest.Package.Name != "alpha" {
		t.Fatalf("name = %q", manifest.Package.Name)
	}
	if manifest.Package.DisplayName != "@vrooli/alpha" {
		t.Fatalf("display name = %q", manifest.Package.DisplayName)
	}
	if manifest.Package.Kind != packagegov.KindJSRuntime {
		t.Fatalf("kind = %q", manifest.Package.Kind)
	}
	if manifest.Package.Refresh.Strategy != packagegov.RefreshScenarioSetup {
		t.Fatalf("refresh strategy = %q", manifest.Package.Refresh.Strategy)
	}
}

func TestWritePackageManifestPersistsCanonicalPackagePath(t *testing.T) {
	root := t.TempDir()
	WritePackageManifest(t, root, "alpha", PackageManifest("alpha", WithPackageDocs("docs/package-governance.md")))

	parsed := testkitgo.ReadJSONFileInto[packagegov.Manifest](t, filepath.Join(root, "packages", "alpha", ".vrooli", "package.json"))
	if parsed.Package.Name != "alpha" {
		t.Fatalf("name = %q", parsed.Package.Name)
	}
	if len(parsed.Package.Docs) != 1 || parsed.Package.Docs[0] != "docs/package-governance.md" {
		t.Fatalf("docs = %#v", parsed.Package.Docs)
	}
}

func TestWriteScenarioUIPackageManifestPersistsDependencies(t *testing.T) {
	root := t.TempDir()
	WriteScenarioUIPackageManifest(t, root, "alpha", NodePackageManifest{
		Name: "alpha-ui",
		Dependencies: map[string]string{
			"@vrooli/core": "file:../../../packages/core",
		},
	})

	parsed := testkitgo.ReadJSONFile(t, filepath.Join(root, "scenarios", "alpha", "ui", "package.json"))
	dependencies := parsed["dependencies"].(map[string]any)
	if dependencies["@vrooli/core"] != "file:../../../packages/core" {
		t.Fatalf("dependencies = %#v", dependencies)
	}
}
