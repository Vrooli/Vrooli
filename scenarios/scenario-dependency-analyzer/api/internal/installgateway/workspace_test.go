package installgateway

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkspaceResolutionPreservesExternalPathAndSafePolicy(t *testing.T) {
	repo, workspace := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "package.json"), []byte(`{"packageManager":"pnpm@10.19.0"}`), 0600); err != nil {
		t.Fatal(err)
	}
	r, err := ResolveWorkspace(repo, workspace, "npm", "@scope/package", "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if r.SurfaceRoot != workspace || !r.ExternalWorkspace || !r.Profile.ScriptsDisabled || r.ManifestPath != filepath.Join(workspace, "package.json") {
		t.Fatalf("incorrect target or policy: %+v", r)
	}
	for _, flag := range []string{"--ignore-scripts", "--ignore-pnpmfile", "--ignore-workspace", "--save-exact", "@scope/package@1.2.3"} {
		if !strings.Contains(r.Command(), flag) {
			t.Errorf("missing %s: %s", flag, r.Command())
		}
	}
	entries, err := os.ReadDir(workspace)
	if err != nil || len(entries) != 1 {
		t.Fatalf("resolution mutated workspace: %v %v", entries, err)
	}
}

func TestWorkspaceRejectsUnsafeOrAmbiguousTargets(t *testing.T) {
	for _, mode := range []string{"relative", "repository", "repository-link", "manifest-link", "config", "membership", "missing-manager", "local-dependency", "range", "other-ecosystem"} {
		t.Run(mode, func(t *testing.T) {
			repo, workspace := t.TempDir(), t.TempDir()
			manifest, ecosystem, version := `{"packageManager":"pnpm@10.19.0"}`, "npm", "1.2.3"
			switch mode {
			case "repository":
				workspace = repo
			case "repository-link":
				link := filepath.Join(workspace, "repo-link")
				if err := os.Symlink(repo, link); err != nil {
					t.Fatal(err)
				}
				workspace = link
			case "membership":
				manifest = `{"packageManager":"pnpm@10.19.0","workspaces":["*"]}`
			case "missing-manager":
				manifest = `{}`
			case "local-dependency":
				manifest = `{"packageManager":"pnpm@10.19.0","dependencies":{"other":"file:../other"}}`
			case "range":
				version = "^1.2.3"
			case "other-ecosystem":
				ecosystem = "pip"
			}
			p := filepath.Join(workspace, "package.json")
			if mode == "manifest-link" {
				target := filepath.Join(repo, "manifest.json")
				if err := os.WriteFile(target, []byte(manifest), 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, p); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(p, []byte(manifest), 0600); err != nil {
				t.Fatal(err)
			}
			if mode == "config" {
				if err := os.WriteFile(filepath.Join(workspace, ".npmrc"), []byte("ignore-scripts=false"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "relative" {
				workspace = "relative"
			}
			if _, err := ResolveWorkspace(repo, workspace, ecosystem, "package", version); err == nil {
				t.Fatal("unsafe target admitted")
			}
		})
	}
}
