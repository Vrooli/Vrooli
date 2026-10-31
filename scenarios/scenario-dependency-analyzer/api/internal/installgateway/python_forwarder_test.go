package installgateway

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveEmbeddedPythonLockUsesItsCanonicalOwner(t *testing.T) {
	repo := t.TempDir()
	mkSurface(t, repo, "embedded", "api", map[string]string{"requirements.txt": "# compatibility entry point\n-r internal/pydeps/requirements.lock\n"})
	owner := filepath.Join(repo, "scenarios", "embedded", "api", "internal", "pydeps")
	if err := os.MkdirAll(owner, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{"requirements.in": "Pillow>=10\n", "requirements.lock": "pillow==11.2.0\n"} {
		if err := os.WriteFile(filepath.Join(owner, name), []byte(value), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r, err := Resolve(repo, "embedded", "api", "pip", "trimesh", "4.7.4")
	if err != nil {
		t.Fatal(err)
	}
	if r.SurfaceRoot != owner || r.PackageManager != "uv-lock" || r.ManifestPath != filepath.Join(owner, "requirements.lock") || len(r.PythonLockfiles) != 1 {
		t.Fatalf("embedded closure became a host pip install: %+v", r)
	}
	if !strings.Contains(r.Command(), "--only-binary :all:") {
		t.Fatalf("embedded owner lost governed wheel policy: %s", r.Command())
	}
}

func TestPythonLockForwarderRejectsEscapesAndMissingOwner(t *testing.T) {
	for _, tc := range []string{"missing", "outside", "symlink", "no-input"} {
		t.Run(tc, func(t *testing.T) {
			parent := t.TempDir()
			root := filepath.Join(parent, "surface")
			if err := os.MkdirAll(root, 0o755); err != nil {
				t.Fatal(err)
			}
			path := "child/requirements.lock"
			if tc == "outside" {
				path = "../requirements.lock"
			}
			if err := os.WriteFile(filepath.Join(root, "requirements.txt"), []byte("-r "+path+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if tc != "missing" {
				if err := os.MkdirAll(filepath.Join(root, "child"), 0o755); err != nil {
					t.Fatal(err)
				}
				target := filepath.Join(root, path)
				if tc == "symlink" {
					outside := filepath.Join(parent, "requirements.lock")
					if err := os.WriteFile(outside, []byte("pillow==11\n"), 0o644); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink(outside, target); err != nil {
						t.Fatal(err)
					}
				} else if err := os.WriteFile(target, []byte("pillow==11\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := pythonLockSurface(root); err == nil {
				t.Fatalf("unsafe %s forwarder accepted", tc)
			}
		})
	}
}

func TestPythonInputUpdatePreservesRangesCommentsAndPackageIdentity(t *testing.T) {
	source := "# model runtime\nPillow>=10,<13\ntrimesh[easy]>=4 # mesh tools\nfoo-bar>=1\n"
	updated, err := updatePythonInput(source, "trimesh", "4.7.4")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Pillow>=10,<13", "trimesh[easy]==4.7.4 # mesh tools", "foo-bar>=1"} {
		if !strings.Contains(updated, want) {
			t.Fatalf("input changed unrelated declaration %q: %s", want, updated)
		}
	}
	updated, err = updatePythonInput(updated, "foobar", "2.0")
	if err != nil || !strings.Contains(updated, "foo-bar>=1") || !strings.Contains(updated, "foobar==2.0") {
		t.Fatalf("distinct normalized package was overwritten: %s, %v", updated, err)
	}
	if _, err := updatePythonInput("foo.bar>=1\nfoo-bar>=2\n", "foo_bar", "3"); err == nil {
		t.Fatal("duplicate canonical package accepted")
	}
	if _, err := updatePythonInput("pillow>=10; python_version > '3.10'\n", "pillow", "11"); err == nil {
		t.Fatal("conditional dependency silently broadened")
	}
}

func TestPythonLockUpdateKeepsDistinctPackageNames(t *testing.T) {
	updated, err := replacePythonRequirement("foo-bar==1\nfoobar==2\n", "foo_bar", "3")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(updated, "foo_bar==3") || !strings.Contains(updated, "foobar==2") {
		t.Fatalf("unrelated package overwritten: %s", updated)
	}
}
