package setup

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
)

// Execute the real make dependency graph, with every build and installer
// replaced inside a private checkout. These checks never activate a host image.
func TestInstallPreparationStopsBeforeBuildOnSupervisoryFailure(t *testing.T) {
	for _, target := range []string{"install", "rebaseline"} {
		for _, failure := range []string{"root-build", "root-vet", "module-build", "module-vet"} {
			t.Run(target+"/"+failure, func(t *testing.T) {
				fixture := newInstallMakeFixture(t)
				events, output, err := fixture.run(target, failure)
				if err == nil {
					t.Fatalf("failed qualification allowed %s: events=%v\n%s", target, events, output)
				}
				for _, event := range events {
					if !strings.HasPrefix(event, "qualify:") {
						t.Fatalf("failed qualification reached candidate build or activation: %v\n%s", events, output)
					}
				}
				if !strings.Contains(output, "refusing to continue") {
					t.Fatalf("qualification refusal missing: %s", output)
				}
				fixture.assertNoSourcePointer(t)
			})
		}
	}
}

func TestInstallPreparationQualifiesBeforeBuildUnderParallelMake(t *testing.T) {
	for _, target := range []string{"prepare-install", "install", "rebaseline"} {
		t.Run(target, func(t *testing.T) {
			fixture := newInstallMakeFixture(t)
			events, output, err := fixture.run(target, "")
			if err != nil {
				t.Fatalf("%s: %v\n%s", target, err, output)
			}
			want := []string{"qualify:root:build", "qualify:root:vet", "qualify:module:build", "qualify:module:vet", "build"}
			if target != "prepare-install" {
				rootAction := "install:vrooli"
				if target == "rebaseline" {
					rootAction = "candidate:runtime update rebaseline " + filepath.Join(fixture.root, ".vrooli/build/vrooli")
				}
				want = append(want, rootAction, "install:vrooli-api", "install:vrooli-agent-launcher", "install:vrooli-policy-runner")
			}
			if !reflect.DeepEqual(events, want) {
				t.Fatalf("%s events = %v, want %v\n%s", target, events, want, output)
			}
			if target == "prepare-install" {
				fixture.assertNoSourcePointer(t)
			} else {
				pointer, err := os.ReadFile(fixture.sourcePointer)
				if err != nil || strings.TrimSpace(string(pointer)) != fixture.root {
					t.Fatalf("completed installation source pointer = %q, %v", pointer, err)
				}
			}
		})
	}
}

func TestInstallationStopsBeforeCompanionsWhenBuildOrRootActionFails(t *testing.T) {
	for _, target := range []string{"install", "rebaseline"} {
		for _, failure := range []string{"candidate-build", "root-action"} {
			t.Run(target+"/"+failure, func(t *testing.T) {
				fixture := newInstallMakeFixture(t)
				events, output, err := fixture.run(target, failure)
				if err == nil {
					t.Fatalf("%s accepted %s: %v\n%s", target, failure, events, output)
				}
				for _, event := range events {
					if strings.HasPrefix(event, "install:") && (failure == "candidate-build" || event != "install:vrooli") {
						t.Fatalf("failure reached installation: %v\n%s", events, output)
					}
					if failure == "candidate-build" && strings.HasPrefix(event, "candidate:") {
						t.Fatalf("failed build invoked candidate: %v", events)
					}
				}
				fixture.assertNoSourcePointer(t)
			})
		}
	}
}

type installMakeFixture struct {
	root          string
	bin           string
	sourcePointer string
}

func newInstallMakeFixture(t *testing.T) installMakeFixture {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("root make installation uses POSIX shell")
	}
	if _, err := exec.LookPath("make"); err != nil {
		t.Fatal("make is required to verify installation ordering")
	}
	fixture := installMakeFixture{root: t.TempDir()}
	fixture.bin = filepath.Join(fixture.root, "test-bin")
	fixture.sourcePointer = filepath.Join(fixture.root, "source-root")
	write := func(path, body string, mode os.FileMode) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), mode); err != nil {
			t.Fatal(err)
		}
	}
	repo := testkitgo.ProjectRoot(t)
	for _, path := range []string{"Makefile", "mk/toolchain.mk"} {
		body, err := os.ReadFile(filepath.Join(repo, path))
		if err != nil {
			t.Fatal(err)
		}
		write(filepath.Join(fixture.root, path), string(body), 0o600)
	}
	write(filepath.Join(fixture.root, "tools/supervisory-modules.txt"), "supervisory-module\n", 0o600)
	if err := os.Mkdir(filepath.Join(fixture.root, "supervisory-module"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(filepath.Join(fixture.bin, "go"), `#!/bin/sh
set -eu
case "$1" in
build|vet)
  location=root
  case "$PWD" in */supervisory-module) location=module;; esac
  printf 'qualify:%s:%s\n' "$location" "$1" >> "$TEST_EVENTS"
  [ "$TEST_FAILURE" != "$location-$1" ]
  ;;
*) echo "unexpected source compilation or invocation: go $*" >&2; exit 90;;
esac
`, 0o700)
	write(filepath.Join(fixture.bin, "build-candidates"), `#!/bin/sh
set -eu
[ "$*" = build ]
printf 'build\n' >> "$TEST_EVENTS"
[ "$TEST_FAILURE" != candidate-build ]
mkdir -p .vrooli/build
cat > .vrooli/build/vrooli <<'CANDIDATE'
#!/bin/sh
set -eu
printf 'candidate:%s\n' "$*" >> "$TEST_EVENTS"
[ "$TEST_FAILURE" != root-action ]
CANDIDATE
cat > .vrooli/build/vrooli-atomic-install <<'INSTALLER'
#!/bin/sh
set -eu
name="${1##*/}"
printf 'install:%s\n' "$name" >> "$TEST_EVENTS"
if [ "$name" = vrooli ]; then [ "$TEST_FAILURE" != root-action ]; fi
INSTALLER
chmod +x .vrooli/build/vrooli .vrooli/build/vrooli-atomic-install
`, 0o700)
	return fixture
}

func (f installMakeFixture) run(target, failure string) ([]string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "make", "--no-print-directory", "-j4", target,
		"VROOLI="+filepath.Join(f.bin, "build-candidates"),
		"INSTALL_DIR="+filepath.Join(f.root, "installed"), "SOURCE_ROOT_POINTER="+f.sourcePointer)
	cmd.Dir = f.root
	cmd.Env = append(os.Environ(), "PATH="+f.bin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"TEST_EVENTS="+filepath.Join(f.root, "events"), "TEST_FAILURE="+failure,
		"MAKEFLAGS=", "MFLAGS=")
	output, err := cmd.CombinedOutput()
	events, _ := os.ReadFile(filepath.Join(f.root, "events"))
	return strings.FieldsFunc(strings.TrimSpace(string(events)), func(r rune) bool { return r == '\n' }), string(output), err
}

func (f installMakeFixture) assertNoSourcePointer(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(f.sourcePointer); !os.IsNotExist(err) {
		t.Fatalf("incomplete installation published source pointer: %v", err)
	}
}
