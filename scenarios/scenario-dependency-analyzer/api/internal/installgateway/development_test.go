package installgateway

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDevelopmentPlacementPreservesDefaultsAndSupportsEveryMutatingJSManager(t *testing.T) {
	for manager, lock := range map[string]string{"npm": "package-lock.json", "pnpm": "pnpm-lock.yaml", "yarn": "yarn.lock"} {
		for _, classification := range []string{"new", "existing-dev", "existing-runtime"} {
			t.Run(manager+"/"+classification, func(t *testing.T) {
				root := t.TempDir()
				ui := filepath.Join(root, "scenarios", "demo", "ui")
				if err := os.MkdirAll(ui, 0o700); err != nil {
					t.Fatal(err)
				}
				manifest := `{}`
				if classification == "existing-dev" {
					manifest = `{"devDependencies":{"test-library":"1.0.0"}}`
				} else if classification == "existing-runtime" {
					manifest = `{"dependencies":{"test-library":"1.0.0"}}`
				}
				for name, data := range map[string]string{lock: "", "package.json": manifest} {
					if err := os.WriteFile(filepath.Join(ui, name), []byte(data), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				r, err := Resolve(root, "demo", "ui", "npm", "test-library", "1.2.3")
				if err != nil {
					t.Fatal(err)
				}
				flag := map[string]string{"npm": "--save-dev", "pnpm": "-D", "yarn": "--dev"}[manager]
				if strings.Contains(r.Command(), flag) != (classification == "existing-dev") {
					t.Fatal("default placement changed", r.Command())
				}
				if err := ApplyDevelopmentDependency(&r); err != nil {
					t.Fatal(err)
				}
				if err := ApplyDevelopmentDependency(&r); err != nil {
					t.Fatal(err)
				}
				if err := ApplyExactVersion(&r, "1.2.3"); err != nil {
					t.Fatal(err)
				}
				count := 0
				for _, arg := range r.Argv {
					if arg == flag {
						count++
					}
				}
				if count != 1 || !r.Profile.ScriptsDisabled || !r.ExactVersion || !strings.Contains(r.Command(), "--ignore-scripts") {
					t.Fatalf("unsafe development plan %+v", r)
				}
				got, _ := os.ReadFile(filepath.Join(ui, "package.json"))
				if string(got) != manifest {
					t.Fatal("planning hand-edited the manifest")
				}
			})
		}
	}
}

func TestDevelopmentPlacementRejectsNonJSAndFrozenOperations(t *testing.T) {
	for _, r := range []Resolution{{PackageManager: "go", Argv: []string{"go", "get"}}, {HostTool: true, PackageManager: "npm", Argv: []string{"npm", "ci"}}, {HostToolRefresh: true, PackageManager: "npm", Argv: []string{"npm", "install"}}, {}} {
		if err := ApplyDevelopmentDependency(&r); err == nil {
			t.Fatal("unsupported development placement accepted", r)
		}
	}
}
