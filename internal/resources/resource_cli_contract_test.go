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
			if manifest.CLI.Distribution == nil || manifest.CLI.Distribution.Kind != "prebuilt_artifact" {
				t.Fatal("expected explicit cli.distribution.kind=prebuilt_artifact")
			}
			if !strings.Contains(manifest.CLI.Distribution.ArtifactName, "${os}") || !strings.Contains(manifest.CLI.Distribution.ArtifactName, "${arch}") {
				t.Fatalf("cli.distribution.artifact_name = %q, want OS and arch placeholders", manifest.CLI.Distribution.ArtifactName)
			}
			if manifest.CLI.SourceBuild == nil || manifest.CLI.SourceBuild.Kind != "go_module" {
				t.Fatal("expected explicit cli.source_build.kind=go_module")
			}
			if manifest.CLI.Freshness == nil || len(manifest.CLI.Freshness.Inputs) == 0 {
				t.Fatal("expected cli.freshness.inputs")
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
		})
	}
}

func TestAllResourcesDeclareDesktopDeploymentContract(t *testing.T) {
	root := testkitgo.ProjectRoot(t)
	entries, err := os.ReadDir(filepath.Join(root, "resources"))
	if err != nil {
		t.Fatalf("read resources: %v", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			manifest, err := manifestpkg.Load(filepath.Join(root, "resources", name, "resource.json"))
			if err != nil {
				t.Fatalf("load manifest: %v", err)
			}
			profile, ok := manifest.Deployment.Profiles["desktop"]
			if !ok {
				t.Fatal("missing deployment.profiles.desktop")
			}
			if manifest.CLI == nil || manifest.CLI.Distribution == nil || manifest.CLI.Distribution.Kind != "prebuilt_artifact" {
				t.Fatal("every resource must declare a prebuilt CLI distribution")
			}
			if manifest.CLI.SourceBuild == nil || manifest.CLI.SourceBuild.Kind != "go_module" || manifest.CLI.Freshness == nil || len(manifest.CLI.Freshness.Inputs) == 0 {
				t.Fatal("every resource must declare a Go-native cli.source_build contract")
			}
			for platform, target := range map[string]*manifestpkg.ResourceDeploymentTarget{
				"linux": profile.Linux, "macos": profile.MacOS, "windows": profile.Windows,
			} {
				if target == nil {
					t.Fatalf("missing desktop %s target", platform)
				}
				if target.Support == "unsupported" {
					if target.Reason == "" {
						t.Fatal("unsupported target must explain why")
					}
					continue
				}
				if len(target.Architectures) == 0 || len(target.Evidence) == 0 {
					t.Fatalf("desktop %s must declare architectures and evidence", platform)
				}
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
