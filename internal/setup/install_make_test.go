package setup

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
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
				rootAction := "install:bundle"
				if target == "rebaseline" {
					rootAction = "candidate:rebaseline"
				}
				want = append(want, "prepare:bundle", rootAction)
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
					if strings.HasPrefix(event, "install:") && (failure == "candidate-build" || event != "install:bundle") {
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

func TestRebaselineRejectsInvalidRuntimePathsBeforeActivation(t *testing.T) {
	f := newInstallMakeFixture(t)
	events, output, err := f.run("rebaseline", "runtime-paths")
	if err == nil || !strings.Contains(output, "invalid runtime paths") {
		t.Fatalf("rebaseline did not reject invalid destinations: %v\n%s", err, output)
	}
	for _, event := range events {
		if event == "candidate:rebaseline" {
			t.Fatalf("rebaseline activated despite invalid destinations: %v", events)
		}
	}
	f.assertNoSourcePointer(t)
}

func TestInstallPreparationDoesNotCreateCheckoutBuildDirectory(t *testing.T) {
	for _, target := range []string{"install", "rebaseline"} {
		t.Run(target, func(t *testing.T) {
			f := newInstallMakeFixture(t)
			_, output, err := f.run(target, "")
			if err != nil {
				t.Fatalf("%s: %v\n%s", target, err, output)
			}
			if _, err := os.Stat(filepath.Join(f.root, ".vrooli", "build")); !os.IsNotExist(err) {
				t.Fatalf("preparation created checkout build directory: %v", err)
			}
		})
	}
}

func TestInstallSnapshotCleanupAndPrune(t *testing.T) {
	f := newInstallMakeFixture(t)
	for _, failure := range []string{"root-action", "interrupt"} {
		_, out, err := f.run("install", failure)
		if err == nil {
			t.Fatalf("expected %s failure: %s", failure, out)
		}
	}
	matches, err := filepath.Glob(filepath.Join(f.root, "snapshots", "install-bundle.*"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("failed install left snapshots %v: %v", matches, err)
	}
	stale := filepath.Join(f.root, "snapshots", "install-bundle.abcdefgh")
	active := filepath.Join(f.root, "snapshots", "install-bundle.ijklmnop")
	for _, path := range []string{stale, active} {
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, ".lease"), nil, 0600); err != nil {
			t.Fatal(err)
		}
		old := time.Now().Add(-25 * time.Hour)
		if err := os.Chtimes(path, old, old); err != nil {
			t.Fatal(err)
		}
	}
	// Hold a lease as an interrupted parent with a still-running child would.
	lease := exec.Command("flock", "-x", filepath.Join(active, ".lease"), "sh", "-c", "echo ready; cat")
	stdin, err := lease.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := lease.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := lease.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { stdin.Close(); lease.Wait() }()
	var ready [6]byte
	if _, err := io.ReadFull(stdout, ready[:]); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-25 * time.Hour)
	if err := os.Chtimes(active, old, old); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("make", "--no-print-directory", "install-snapshot-prune", "BUNDLE_PUBLISHER="+filepath.Join(f.bin, "publisher"))
	cmd.Dir = f.root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("prune: %v %s", err, out)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale snapshot retained: %v", err)
	}
	if _, err := os.Stat(active); err != nil {
		t.Fatalf("active snapshot removed: %v", err)
	}
}

func TestHubPruneUsesCurrentSourceToRetirePendingDecision(t *testing.T) {
	f := newInstallMakeFixture(t)
	snapshot := filepath.Join(f.root, "snapshots", "rebaseline-hub.abcdefgh")
	if err := os.MkdirAll(snapshot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(snapshot, ".ready"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	retirer := filepath.Join(f.bin, "retirer")
	if err := os.WriteFile(retirer, []byte("#!/bin/sh\n[ \"$*\" = \"runtime update retire-snapshot $PWD/snapshots/rebaseline-hub.abcdefgh\" ]\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-25 * time.Hour)
	if err := os.Chtimes(snapshot, old, old); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("make", "--no-print-directory", "rebaseline-hub-prune", "BUNDLE_PUBLISHER="+filepath.Join(f.bin, "publisher"), "VROOLI="+retirer)
	cmd.Dir = f.root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("prune: %v\n%s", err, out)
	}
	if _, err := os.Stat(snapshot); !os.IsNotExist(err) {
		t.Fatalf("retired snapshot still exists: %v", err)
	}
}

func TestHubResumeUsesShasumWhenSha256sumIsUnavailable(t *testing.T) {
	if _, err := exec.LookPath("shasum"); err != nil {
		t.Skip("shasum is required to exercise the portable fallback")
	}
	for _, tamper := range []bool{false, true} {
		t.Run(fmt.Sprintf("tamper=%v", tamper), func(t *testing.T) {
			testHubResumeUsesShasumFallback(t, tamper)
		})
	}
}

func testHubResumeUsesShasumFallback(t *testing.T, tamper bool) {
	t.Helper()
	f := newInstallMakeFixture(t)
	snapshot := filepath.Join(f.root, "snapshots", "rebaseline-hub.abcdefgh")
	if err := os.MkdirAll(snapshot, 0o700); err != nil {
		t.Fatal(err)
	}
	candidate := `#!/bin/sh
set -eu
case "$*" in
  "runtime update rebaseline --capability --probe-hub-publication") printf 'legacy-publication-capability\n';;
  "runtime update rebaseline "*) printf 'resume\n' >> "$TEST_EVENTS";;
  *) echo "unexpected candidate command: $*" >&2; exit 90;;
esac
`
	if err := os.WriteFile(filepath.Join(snapshot, "vrooli"), []byte(candidate), 0o700); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(snapshot, "vrooli"))
	if err != nil {
		t.Fatal(err)
	}
	imageSHA := sha256.Sum256(data)
	manifest := fmt.Sprintf("%x  vrooli\n", imageSHA)
	if err := os.WriteFile(filepath.Join(snapshot, "SHA256SUMS"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if tamper {
		if err := os.WriteFile(filepath.Join(snapshot, "vrooli"), append(data, []byte("tampered")...), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{".ready", ".lease"} {
		if err := os.WriteFile(filepath.Join(snapshot, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(f.root, "snapshots", "current"), []byte(snapshot+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) string {
		t.Helper()
		path := filepath.Join(f.bin, name)
		if err := os.WriteFile(path, []byte(body), 0o700); err != nil {
			t.Fatal(err)
		}
		return path
	}
	publisher := write("publisher", `#!/bin/sh
case "$1" in
  --snapshot-root) printf '%s\n' "$PWD/snapshots";;
  --publish-source-owned) printf 'publish-source\n' >> "$TEST_EVENTS";;
  *) exit 91;;
esac
`)
	// Use a deliberately minimal PATH: `shasum` and the recipe utilities are
	// present, while `sha256sum` cannot be found even though the host has it.
	portablePath := filepath.Join(f.root, "portable-bin")
	if err := os.MkdirAll(portablePath, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"cat", "flock", "shasum", "touch"} {
		resolved, err := exec.LookPath(name)
		if err != nil {
			t.Fatalf("required recipe utility %s: %v", name, err)
		}
		if err := os.Symlink(resolved, filepath.Join(portablePath, name)); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command("make", "--no-print-directory", "rebaseline-hub-resume", "BUNDLE_PUBLISHER="+publisher)
	cmd.Dir = f.root
	cmd.Env = append(os.Environ(), "PATH="+portablePath, "TEST_EVENTS="+filepath.Join(f.root, "events"), "MAKEFLAGS=", "MFLAGS=")
	output, runErr := cmd.CombinedOutput()
	events, readErr := os.ReadFile(filepath.Join(f.root, "events"))
	if tamper {
		if runErr == nil {
			t.Fatalf("tampered bundle resumed successfully\n%s", output)
		}
		if readErr != nil && !os.IsNotExist(readErr) {
			t.Fatalf("read events: %v", readErr)
		}
		if strings.Contains(string(events), "resume") || strings.Contains(string(events), "publish-source") {
			t.Fatalf("tampered bundle reached resume or publication: %q\n%s", events, output)
		}
		if _, statErr := os.Stat(filepath.Join(snapshot, ".completed")); !os.IsNotExist(statErr) {
			t.Fatalf("tampered bundle was marked complete: %v", statErr)
		}
		return
	}
	if runErr != nil {
		t.Fatalf("resume without sha256sum: %v\n%s", runErr, output)
	}
	if readErr != nil || !strings.Contains(string(events), "resume") || !strings.Contains(string(events), "publish-source") {
		t.Fatalf("prepared candidate was not resumed and published: %q, %v\n%s", events, readErr, output)
	}
	if _, err := os.Stat(filepath.Join(snapshot, ".completed")); err != nil {
		t.Fatalf("resume did not write its completion marker: %v", err)
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
	write(filepath.Join(fixture.bin, "publisher"), `#!/bin/sh
set -eu
[ "$1" = --snapshot-root ]
mkdir -p "$PWD/snapshots"
printf '%s\n' "$PWD/snapshots"
`, 0o700)
	write(filepath.Join(fixture.bin, "build-candidates"), `#!/bin/sh
set -eu
[ "$*" = build ]
printf 'build\n' >> "$TEST_EVENTS"
[ "$TEST_FAILURE" != candidate-build ]
mkdir -p "$VROOLI_BUILD_DIR"
if [ -n "${TEST_GENERATION:-}" ]; then
  printf '%s' "$TEST_GENERATION" > "$VROOLI_BUILD_DIR/generation"
  touch "$TEST_BARRIER/$TEST_GENERATION"
  while [ "$(find "$TEST_BARRIER" -type f | wc -l)" -lt 2 ]; do sleep .02; done
  [ "${TEST_CRASH:-}" != yes ] || kill -KILL $$
fi
cat > "$VROOLI_BUILD_DIR/vrooli" <<'CANDIDATE'
#!/bin/sh
set -eu
printf 'candidate:rebaseline\n' >> "$TEST_EVENTS"
[ "$TEST_FAILURE" != root-action ]
printf '%s\n' "$PWD" > "$TEST_SOURCE_POINTER"
CANDIDATE
cat > "$VROOLI_BUILD_DIR/vrooli-atomic-install" <<'INSTALLER'
#!/bin/sh
set -eu
case "$1" in
--snapshot-root) mkdir -p "$PWD/snapshots"; printf '%s\n' "$PWD/snapshots";;
--verify-runtime-paths) [ "$TEST_FAILURE" != runtime-paths ] || { echo 'invalid runtime paths' >&2; exit 1; };;
--prepare-bundle)
  printf 'prepare:bundle\n' >> "$TEST_EVENTS"
  mkdir "$3"
  if [ -n "${TEST_GENERATION:-}" ]; then
    [ "$(cat "$2/generation")" = "$TEST_GENERATION" ] || exit 91
    printf '%s\n' "$TEST_GENERATION" >> "$TEST_BARRIER/snapshots"
  fi
  cp "$2/vrooli" "$2/vrooli-atomic-install" "$3/"
  printf 'fixture manifest\n' > "$3/SHA256SUMS"
  ;;
--install-bundle-owned)
  printf 'install:owned-bundle\n' >> "$TEST_EVENTS"
  [ "$TEST_FAILURE" != root-action ]
  printf '%s\n' "$4" > "$TEST_SOURCE_POINTER"
  ;;
--install-bundle)
  printf 'install:bundle\n' >> "$TEST_EVENTS"
  [ "$TEST_FAILURE" != interrupt ] || kill -TERM $$
  [ "$TEST_FAILURE" != root-action ]
  printf '%s\n' "$5" > "$TEST_SOURCE_POINTER"
  ;;
--publish-source)
  printf 'publish:source\n' >> "$TEST_EVENTS"
  printf '%s\n' "$5" > "$6"
  ;;
*) echo "unexpected installer invocation: $*" >&2; exit 90;;
esac
INSTALLER
chmod +x "$VROOLI_BUILD_DIR/vrooli" "$VROOLI_BUILD_DIR/vrooli-atomic-install"
`, 0o700)
	return fixture
}

func (f installMakeFixture) run(target, failure string) ([]string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "make", "--no-print-directory", "-j4", target,
		"VROOLI="+filepath.Join(f.bin, "build-candidates"),
		"BUNDLE_PUBLISHER="+filepath.Join(f.bin, "publisher"),
		"INSTALL_DIR="+filepath.Join(f.root, "installed"), "SOURCE_ROOT_POINTER="+f.sourcePointer)
	cmd.Dir = f.root
	cmd.Env = append(os.Environ(), "PATH="+f.bin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"TEST_EVENTS="+filepath.Join(f.root, "events"), "TEST_FAILURE="+failure, "TEST_SOURCE_POINTER="+f.sourcePointer,
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

// Overlap two builds at the point where shared output used to be vulnerable.
// An interrupted peer must not leave files in the surviving build's cohort.
func TestConcurrentInstallBuildCohorts(t *testing.T) {
	for _, crash := range []bool{false, true} {
		t.Run(fmt.Sprintf("crash=%v", crash), func(t *testing.T) {
			f := newInstallMakeFixture(t)
			barrier := filepath.Join(f.root, "barrier")
			if err := os.Mkdir(barrier, 0o755); err != nil {
				t.Fatal(err)
			}
			run := func(id string, die bool) *exec.Cmd {
				cmd := exec.Command("make", "--no-print-directory", "install", "VROOLI="+filepath.Join(f.bin, "build-candidates"), "BUNDLE_PUBLISHER="+filepath.Join(f.bin, "publisher"), "INSTALL_DIR="+filepath.Join(f.root, "installed"), "SOURCE_ROOT_POINTER="+f.sourcePointer)
				cmd.Dir = f.root
				cmd.Env = append(os.Environ(), "PATH="+f.bin+string(os.PathListSeparator)+os.Getenv("PATH"), "TEST_EVENTS="+filepath.Join(f.root, "events"), "TEST_FAILURE=", "TEST_SOURCE_POINTER="+f.sourcePointer, "TEST_GENERATION="+id, "TEST_BARRIER="+barrier, fmt.Sprintf("TEST_CRASH=%s", map[bool]string{true: "yes", false: "no"}[die]), "MAKEFLAGS=", "MFLAGS=")
				return cmd
			}
			a, b := run("a", crash), run("b", false)
			if err := a.Start(); err != nil {
				t.Fatal(err)
			}
			if err := b.Start(); err != nil {
				t.Fatal(err)
			}
			errA, errB := a.Wait(), b.Wait()
			if (errA != nil) != crash || errB != nil {
				t.Fatalf("a=%v b=%v", errA, errB)
			}
			data, err := os.ReadFile(filepath.Join(barrier, "snapshots"))
			if err != nil {
				t.Fatal(err)
			}
			got := strings.Fields(string(data))
			if len(got) != 2 && !(crash && len(got) == 1) {
				t.Fatalf("snapshots %v", got)
			}
			if crash && got[0] != "b" {
				t.Fatalf("surviving snapshot = %v", got)
			}
		})
	}
}

func TestMakeInstallUsesRuntimeOwnedDestinationByDefault(t *testing.T) {
	f := newInstallMakeFixture(t)
	cmd := exec.Command("make", "--no-print-directory", "install", "VROOLI="+filepath.Join(f.bin, "build-candidates"), "BUNDLE_PUBLISHER="+filepath.Join(f.bin, "publisher"))
	cmd.Dir = f.root
	cmd.Env = append(os.Environ(), "PATH="+f.bin+string(os.PathListSeparator)+os.Getenv("PATH"), "TEST_EVENTS="+filepath.Join(f.root, "events"), "TEST_FAILURE=", "TEST_SOURCE_POINTER="+f.sourcePointer, "HOME="+filepath.Join(f.root, "fake-root"), "SUDO_USER=nobody", "MAKEFLAGS=", "MFLAGS=")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	data, err := os.ReadFile(filepath.Join(f.root, "events"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "install:owned-bundle") {
		t.Fatalf("did not invoke owner destination: %s", data)
	}
	if _, err := os.Stat(filepath.Join(f.root, "fake-root", ".vrooli", "bin")); !os.IsNotExist(err) {
		t.Fatalf("make created sudo HOME bin: %v", err)
	}
}
