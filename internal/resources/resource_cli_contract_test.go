package resources_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
)

func TestEnabledResourcesDeclareExplicitCLIContract(t *testing.T) {
	root := testkitgo.ProjectRoot(t)

	data, err := os.ReadFile(filepath.Join(root, ".vrooli", "service.json"))
	if err != nil {
		t.Fatalf("read service.json: %v", err)
	}

	var service struct {
		Dependencies struct {
			Resources map[string]struct {
				Enabled bool `json:"enabled"`
			} `json:"resources"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(data, &service); err != nil {
		t.Fatalf("unmarshal service.json: %v", err)
	}

	var enabled []string
	for name, entry := range service.Dependencies.Resources {
		if entry.Enabled {
			enabled = append(enabled, name)
		}
	}
	sort.Strings(enabled)

	for _, name := range enabled {
		t.Run(name, func(t *testing.T) {
			manifest, err := manifestpkg.Load(filepath.Join(root, "resources", name, "resource.json"))
			if err != nil {
				t.Fatalf("load manifest: %v", err)
			}
			if manifest.CLI == nil {
				t.Fatal("expected explicit cli block")
			}
			if !manifest.CLI.Enabled {
				t.Fatal("expected enabled resource to declare cli.enabled=true")
			}
			if got, want := manifest.CLI.Command, "resource-"+name; got != want {
				t.Fatalf("cli.command = %q, want %q", got, want)
			}
			if manifest.CLI.Artifacts.Manifest.Location != "sibling" {
				t.Fatalf("cli.artifacts.manifest.location = %q, want sibling", manifest.CLI.Artifacts.Manifest.Location)
			}
			if manifest.CLI.Artifacts.BuildMetadata.Location != "sibling" {
				t.Fatalf("cli.artifacts.build_metadata.location = %q, want sibling", manifest.CLI.Artifacts.BuildMetadata.Location)
			}
			if len(manifest.CLI.Install) == 0 {
				t.Fatal("expected explicit cli.install steps")
			}
			if manifest.CLI.Invoke.Kind != "installed_command" {
				t.Fatalf("cli.invoke.kind = %q, want installed_command", manifest.CLI.Invoke.Kind)
			}
			if manifest.CLI.Invoke.Command != manifest.CLI.Command {
				t.Fatalf("cli.invoke.command = %q, want %q", manifest.CLI.Invoke.Command, manifest.CLI.Command)
			}
			if manifest.CLI.Freshness == nil || len(manifest.CLI.Freshness.Inputs) == 0 {
				t.Fatal("expected explicit cli.freshness.inputs")
			}
			var hasResourceManifest bool
			for _, input := range manifest.CLI.Freshness.Inputs {
				if input == "resource.json" {
					hasResourceManifest = true
					break
				}
			}
			if !hasResourceManifest {
				t.Fatalf("cli.freshness.inputs = %v, want resource.json included", manifest.CLI.Freshness.Inputs)
			}
			requireDeclaredCLIAssets(t, root, name, manifest)
			if _, err := os.Stat(filepath.Join(root, "resources", name, "cli.sh")); !os.IsNotExist(err) {
				t.Fatalf("expected root-level cli.sh removal for %s, stat err=%v", name, err)
			}
			installPath := filepath.Join(root, "resources", name, "lib", "install.sh")
			data, err := os.ReadFile(installPath)
			if err == nil {
				body := string(data)
				for _, forbidden := range []string{
					"install_resource_cli",
					"uninstall_resource_cli",
					".vrooli/resource-registry",
				} {
					if strings.Contains(body, forbidden) {
						t.Fatalf("%s still references deprecated shell-era resource CLI plumbing %q", installPath, forbidden)
					}
				}
			} else if !os.IsNotExist(err) {
				t.Fatalf("read %s: %v", installPath, err)
			}
		})
	}
}

func requireDeclaredCLIAssets(t *testing.T, root, name string, manifest manifestpkg.ResourceManifest) {
	t.Helper()

	base := filepath.Join(root, "resources", name)
	switch manifest.CLI.Adapter.Kind {
	case "go_module":
		if got := manifest.CLI.Adapter.ModuleDir; got != "cli" {
			t.Fatalf("cli.adapter.module_dir = %q, want cli", got)
		}
		moduleDir := filepath.Join(base, filepath.FromSlash(manifest.CLI.Adapter.ModuleDir))
		requireFile(t, filepath.Join(moduleDir, "go.mod"))
		entries, err := filepath.Glob(filepath.Join(moduleDir, "*.go"))
		if err != nil {
			t.Fatalf("glob Go sources in %s: %v", moduleDir, err)
		}
		if len(entries) == 0 {
			t.Fatalf("expected Go sources under %s for go_module adapter", moduleDir)
		}
	case "shell_script":
		requireFile(t, filepath.Join(base, filepath.FromSlash(manifest.CLI.Adapter.ScriptPath)))
		requireFile(t, filepath.Join(base, filepath.FromSlash(manifest.CLI.Adapter.InstallScript)))
	default:
		t.Fatalf("unsupported cli.adapter.kind %q", manifest.CLI.Adapter.Kind)
	}
}

func requireFile(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s: %v", path, err)
	}
}
