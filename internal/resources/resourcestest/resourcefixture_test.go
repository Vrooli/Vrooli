package resourcefixture

import (
	"os"
	"path/filepath"
	"testing"

	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
	resourceenv "github.com/vrooli/vrooli/internal/resources/env"
	"github.com/vrooli/vrooli/internal/shell/shelltest"
	"github.com/vrooli/vrooli/internal/testenv"
)

func TestWritePortRegistryCreatesFixtureFile(t *testing.T) {
	root := t.TempDir()
	WritePortRegistry(t, root, map[string]int{"redis": 6379})

	for _, rel := range []string{".vrooli/test-port-registry.json"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("expected %s: %v", rel, err)
		}
	}
}

func TestWritePortRegistryStatePersistsReservedRanges(t *testing.T) {
	root := t.TempDir()
	WritePortRegistryState(t, root, resourceenv.PortRegistry{
		ResourcePorts:  map[string]int{"postgres": 5433},
		ReservedRanges: map[string]string{"db": "5432-5499"},
	})

	registry := testkitgo.ReadJSONFileInto[resourceenv.PortRegistry](t, filepath.Join(root, ".vrooli", "test-port-registry.json"))
	if got := registry.ResourcePorts["postgres"]; got != 5433 {
		t.Fatalf("postgres port = %d, want 5433", got)
	}
	if got := registry.ReservedRanges["db"]; got != "5432-5499" {
		t.Fatalf("reserved range = %q, want 5432-5499", got)
	}
}

func TestWriteResourceRegistryEntryCreatesRegistryFile(t *testing.T) {
	root := t.TempDir()
	WriteResourceRegistryEntry(t, root, "redis")

	if _, err := os.Stat(filepath.Join(root, ".vrooli", "resource-registry", "redis.json")); err != nil {
		t.Fatalf("expected registry entry: %v", err)
	}
}

func TestWriteFakeDockerInstallsDockerShimOnPath(t *testing.T) {
	stateFile := WriteFakeDocker(t)
	if _, err := os.Stat(filepath.Join(filepath.Dir(stateFile), "docker")); err != nil {
		t.Fatalf("expected docker shim: %v", err)
	}
	if got := os.Getenv("FAKE_DOCKER_STATE"); got != stateFile {
		t.Fatalf("FAKE_DOCKER_STATE = %q, want %q", got, stateFile)
	}
}

func TestWriteExternalCLIResourceFixtureCreatesManifestAndBinary(t *testing.T) {
	root := t.TempDir()
	path := WriteExternalCLIResourceFixture(t, root, "redis", shelltest.BashShebang()+"exit 0\n")

	manifest := ReadResourceManifest(t, root, "redis")
	if manifest.Driver != "external-cli" {
		t.Fatalf("driver = %q, want external-cli", manifest.Driver)
	}
	if manifest.Binary != "resource-redis" {
		t.Fatalf("binary = %q, want resource-redis", manifest.Binary)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected installed binary: %v", err)
	}
}

func TestResourceTemplateUsesTypedDefaults(t *testing.T) {
	manifest := ResourceTemplate("docker-service")
	if manifest.Name != "docker-service" {
		t.Fatalf("name = %q", manifest.Name)
	}
	if manifest.Driver != "docker-service" {
		t.Fatalf("driver = %q", manifest.Driver)
	}
	if manifest.RequiredVars["RESOURCE_NAME"].Flag != "name" {
		t.Fatalf("required RESOURCE_NAME flag = %q", manifest.RequiredVars["RESOURCE_NAME"].Flag)
	}
}

func TestWriteResourceTemplateManifestPersistsCanonicalPath(t *testing.T) {
	root := t.TempDir()
	WriteResourceTemplateManifest(t, root, "docker-service", ResourceTemplate(
		"docker-service",
		WithTemplateDisplayName("Docker Service"),
		WithTemplateDocs(map[string]string{"operations": "docs/OPERATIONS.md"}),
	))

	manifest := testkitgo.ReadJSONFile(t, filepath.Join(root, "templates", "resources", "docker-service", "template.json"))
	if manifest["displayName"] != "Docker Service" {
		t.Fatalf("displayName = %v", manifest["displayName"])
	}
	docs := manifest["docs"].(map[string]any)
	if docs["operations"] != "docs/OPERATIONS.md" {
		t.Fatalf("docs.operations = %v", docs["operations"])
	}
}

func TestWriteResourceCLIGoModCreatesCanonicalPath(t *testing.T) {
	tree := testenv.NewRepositoryTree(t, "fixture")
	root := tree.Root
	WriteResourceCLIGoMod(t, root, "redis", "")

	path := filepath.Join(root, "resources", "redis", "cli", "go.mod")
	testenv.AssertFileContents(t, path, "module resource-redis/cli\n")
}
