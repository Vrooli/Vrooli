package installgateway

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPythonPublicationRestoresSourceWhenLockPublicationFails(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "rollback"}[fail], func(t *testing.T) {
			root := t.TempDir()
			pending := []pythonLockPublication{}
			for _, name := range []string{"requirements.in", "requirements.lock"} {
				output := filepath.Join(root, name)
				source := filepath.Join(root, name+".staged")
				if err := os.WriteFile(output, []byte("old "+name), 0o640); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(source, []byte("new "+name), 0o600); err != nil {
					t.Fatal(err)
				}
				pending = append(pending, pythonLockPublication{source, output})
			}
			calls := 0
			err := publishPythonLocks(pending, func(a, b string) error {
				calls++
				if fail && calls == 2 {
					return errors.New("injected filesystem refusal")
				}
				return os.Rename(a, b)
			})
			if (err != nil) != fail {
				t.Fatalf("error=%v", err)
			}
			prefix := "new "
			if fail {
				prefix = "old "
			}
			for _, item := range pending {
				data, err := os.ReadFile(item.output)
				if err != nil {
					t.Fatal(err)
				}
				if string(data) != prefix+filepath.Base(item.output) {
					t.Fatalf("incoherent dependency pair: %s=%s", item.output, data)
				}
				info, _ := os.Stat(item.output)
				if info.Mode().Perm() != 0o640 {
					t.Fatalf("mode changed: %v", info.Mode())
				}
			}
		})
	}
}

func TestPythonForwarderRefusesSymlinkCanonicalInput(t *testing.T) {
	root := t.TempDir()
	owner := filepath.Join(root, "closure")
	if err := os.Mkdir(owner, 0o755); err != nil {
		t.Fatal(err)
	}
	for path, data := range map[string]string{"requirements.txt": "-r closure/requirements.lock\n", "closure/requirements.lock": "pillow==11\n", "outside.in": "pillow>=10\n"} {
		if err := os.WriteFile(filepath.Join(root, path), []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(filepath.Join(root, "outside.in"), filepath.Join(owner, "requirements.in")); err != nil {
		t.Fatal(err)
	}
	if _, err := pythonLockSurface(root); err == nil {
		t.Fatal("symlink canonical input accepted")
	}
}
