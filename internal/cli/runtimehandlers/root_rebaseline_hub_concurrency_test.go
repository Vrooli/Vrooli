package runtimehandlers

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestConcurrentHubPreparationsResumeTheirOwnSnapshots(t *testing.T) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, name := range []string{"mk", "tools", "Makefile"} {
		if err := os.Symlink(filepath.Join(root, name), filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
	build := filepath.Join(dir, "build")
	if err := os.Mkdir(build, 0700); err != nil {
		t.Fatal(err)
	}
	// Preparation writes distinct snapshots; the first activation waits until the
	// second has begun, forcing the global pointer to change while it is active.
	installer := `#!/bin/sh
set -eu
if [ "$1" = --snapshot-root ]; then printf '%s\n' "$SNAPSHOT_ROOT"; exit 0; fi
if [ "$1" = --snapshot-lock-exec ]; then shift 3; exec "$@"; fi
if [ "$1" = --snapshot-lease-exec ] || [ "$1" = --try-snapshot-lease-exec ]; then shift 4; exec "$@"; fi
if [ "$1" = --prune-hub-snapshots ] || [ "$1" = --prune-install-snapshots ]; then exit 0; fi
if [ "$1" = --verify-runtime-paths ]; then exit 0; fi
if [ "$1" = --publish-source ]; then
 printf '%s\n' "$5" > "$6"
 exit 0
fi
snapshot=$3
mkdir "$snapshot"
cp "$FIXTURE" "$snapshot/vrooli"
cp "$0" "$snapshot/vrooli-atomic-install"
(cd "$snapshot" && if command -v sha256sum >/dev/null 2>&1; then sha256sum vrooli vrooli-atomic-install > SHA256SUMS; else shasum -a 256 vrooli vrooli-atomic-install > SHA256SUMS; fi)
touch "$snapshot/.ready"
printf '%s\n' "$snapshot" > "$4"
`
	fixture := `#!/bin/sh
set -eu
[ "$4" != --capability ] || exit 2
printf '%s\n' "$4" >> "$TRACE"
if mkdir "$FIRST" 2>/dev/null; then
 while [ ! -f "$RELEASE" ]; do sleep 0.02; done
fi
`
	for name, data := range map[string]string{"vrooli-atomic-install": installer, "fixture": fixture} {
		if err := os.WriteFile(filepath.Join(build, name), []byte(data), 0700); err != nil {
			t.Fatal(err)
		}
	}
	wrapper := filepath.Join(dir, "fixture-make")
	if err := os.WriteFile(wrapper, []byte(`#!/bin/sh
if [ "$1" = --no-print-directory ] && [ "$2" = prepare-install ]; then
 cohort=${3#BUILD_DIR=}
 cp "$FIXTURE_INSTALLER" "$cohort/vrooli-atomic-install"
 exit 0
fi
exec make "$@"
`), 0700); err != nil {
		t.Fatal(err)
	}
	trace, release := filepath.Join(dir, "trace"), filepath.Join(dir, "release")
	command := func() *exec.Cmd {
		cmd := exec.Command("make", "-f", filepath.Join(root, "Makefile"), "rebaseline-hub", "MAKE="+wrapper, "BUILD_DIR=build", "INSTALL_DIR="+filepath.Join(dir, "install"), "SOURCE_ROOT_POINTER="+filepath.Join(dir, "source"), "BUNDLE_PUBLISHER="+filepath.Join(build, "vrooli-atomic-install"))
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "SNAPSHOT_ROOT="+build, "FIXTURE="+filepath.Join(build, "fixture"), "FIXTURE_INSTALLER="+filepath.Join(build, "vrooli-atomic-install"), "TRACE="+trace, "FIRST="+filepath.Join(dir, "first"), "RELEASE="+release)
		return cmd
	}
	first, second := command(), command()
	if err := first.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.WriteFile(release, nil, 0600)
		if first.ProcessState == nil {
			_ = first.Process.Kill()
			_ = first.Wait()
		}
		if second.Process != nil && second.ProcessState == nil {
			_ = second.Process.Kill()
			_ = second.Wait()
		}
	}()
	waitLines := func(n int) []string {
		t.Helper()
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			data, _ := os.ReadFile(trace)
			lines := strings.Fields(string(data))
			if len(lines) >= n {
				return lines
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatalf("timed out waiting for %d activations", n)
		return nil
	}
	waitLines(1)
	if err := second.Start(); err != nil {
		t.Fatal(err)
	}
	lines := waitLines(2)
	if lines[0] == lines[1] {
		t.Fatalf("both invocations resumed %s", lines[0])
	}
	if err := os.WriteFile(release, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := first.Wait(); err != nil {
		t.Fatalf("first: %v", err)
	}
	if err := second.Wait(); err != nil {
		t.Fatalf("second: %v", err)
	}
	// The pointer now names the second snapshot. A pending first decision must
	// still be resumable by its exact retained path.
	resume := exec.Command("make", "-f", filepath.Join(root, "Makefile"), "rebaseline-hub-resume", "BUILD_DIR=build", "INSTALL_DIR="+filepath.Join(dir, "install"), "SOURCE_ROOT_POINTER="+filepath.Join(dir, "source"), "BUNDLE_PUBLISHER="+filepath.Join(build, "vrooli-atomic-install"))
	resume.Dir = dir
	resume.Env = append(os.Environ(), "SNAPSHOT_ROOT="+build, "HUB_SNAPSHOT="+filepath.Dir(lines[0]), "TRACE="+trace, "FIRST="+filepath.Join(dir, "first"), "RELEASE="+release)
	if out, err := resume.CombinedOutput(); err != nil {
		t.Fatalf("resume first: %v %s", err, out)
	}
	resumed := waitLines(3)
	if resumed[2] != lines[0] {
		t.Fatalf("resumed %q, want %q", resumed[2], lines[0])
	}
}
