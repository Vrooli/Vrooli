package installgateway

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func pnpmPatchManifest(path string) string {
	return `{"name":"demo-ui","packageManager":"pnpm@10.34.9","dependencies":{"alpha":"1.0.0"},"pnpm":{"patchedDependencies":{"rebrowser-playwright-core@1.52.0":` + fmt.Sprintf("%q", path) + `}}}`
}

func writePatchFixture(t *testing.T, fixture pnpmFixture, path, content string) string {
	t.Helper()
	full := filepath.Join(fixture.surface, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return full
}

func TestStageCopiesDeclaredPnpmPatchAndMeasuresTemporaryBytes(t *testing.T) {
	const patchPath = "patches/rebrowser-playwright-core.patch"
	const patchContent = "--- a/file.js\n+++ b/file.js\n@@ change\n"
	fixture := newPnpmFixture(t, pnpmPatchManifest(patchPath), testBaseLock, "beta", "2.0.0")
	writePatchFixture(t, fixture, patchPath, patchContent)
	fake := &fakePnpm{add: func(call pnpmCall) ([]byte, error) {
		staged, err := os.ReadFile(filepath.Join(call.Dir, filepath.FromSlash(patchPath)))
		if err != nil {
			return nil, fmt.Errorf("resolver could not read declared patch: %w", err)
		}
		if string(staged) != patchContent {
			return nil, errors.New("staged patch bytes differ from the source")
		}
		return resolvesTo(testNextManifest, testNextLock)(call)
	}}
	useFakePnpm(t, fake)
	useFakeSizer(t, &fakeSizer{registry: map[string]artifactSize{"alpha@1.0.0": {Unpacked: 10, Archive: 5}, "beta@2.0.0": {Unpacked: 100, Archive: 50}}})

	stage, err := stageJSResolution(context.Background(), fixture.resolution)
	if err != nil {
		t.Fatal(err)
	}
	stageDir := stage.dir
	if string(mustReadFile(t, filepath.Join(stageDir, filepath.FromSlash(patchPath)))) != patchContent {
		t.Fatal("stage did not retain the exact declared patch")
	}
	if stage.patchBytes != int64(len(patchContent)) || stage.Measured.InputBytes < int64(len(patchContent)) {
		t.Fatalf("staged patch bytes were not included in the bounded measurement: bytes=%d measurement=%+v", stage.patchBytes, stage.Measured)
	}
	stage.Release()
	if _, err := os.Lstat(stageDir); !os.IsNotExist(err) {
		t.Fatalf("release left staged patch tree at %s: %v", stageDir, err)
	}
}

func TestStageFailureAfterCopyCleansDeclaredPnpmPatch(t *testing.T) {
	const patchPath = "patches/local.patch"
	fixture := newPnpmFixture(t, pnpmPatchManifest(patchPath), testBaseLock, "beta", "2.0.0")
	writePatchFixture(t, fixture, patchPath, "patch bytes")
	useFakePnpm(t, &fakePnpm{add: func(call pnpmCall) ([]byte, error) {
		if _, err := os.Stat(filepath.Join(call.Dir, filepath.FromSlash(patchPath))); err != nil {
			return nil, fmt.Errorf("patch was not staged before resolution: %w", err)
		}
		return nil, errors.New("synthetic resolver failure")
	}})
	useFakeSizer(t, &fakeSizer{})

	if _, err := stageJSResolution(context.Background(), fixture.resolution); err == nil {
		t.Fatal("expected resolver failure")
	}
	if left := stageLeftovers(t, fixture); len(left) != 0 {
		t.Fatalf("failed resolution left staged patch data behind: %v", left)
	}
}

func TestPublishRejectsChangedPnpmPatchBeforePublishingInputs(t *testing.T) {
	const patchPath = "patches/local.patch"
	fixture := newPnpmFixture(t, pnpmPatchManifest(patchPath), testBaseLock, "beta", "2.0.0")
	patchFile := writePatchFixture(t, fixture, patchPath, "before")
	useFakePnpm(t, &fakePnpm{add: resolvesTo(testNextManifest, testNextLock)})
	useFakeSizer(t, &fakeSizer{registry: map[string]artifactSize{"alpha@1.0.0": {Unpacked: 10, Archive: 5}, "beta@2.0.0": {Unpacked: 100, Archive: 50}}})

	stage, err := stageJSResolution(context.Background(), fixture.resolution)
	if err != nil {
		t.Fatal(err)
	}
	defer stage.Release()
	if err := os.WriteFile(patchFile, []byte("changed after resolution"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = stage.publish(context.Background(), fixture.resolution)
	if err == nil || !strings.Contains(err.Error(), "changed after dependency resolution") {
		t.Fatalf("changed patch publication = %v", err)
	}
	if fixture.read(t, "package.json") != pnpmPatchManifest(patchPath) || fixture.read(t, "pnpm-lock.yaml") != testBaseLock {
		t.Fatal("changed patch must refuse before publishing manifest or lock")
	}
}

func TestPnpmPatchPathRejectsExternalAndTraversalLocations(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "outside.patch")
	for name, raw := range map[string]string{
		"parent traversal": "../outside.patch",
		"nested traversal": "patches/../../outside.patch",
		"absolute":         outside,
		"windows drive":    `C:/outside.patch`,
		"backslash":        `patches\\outside.patch`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := cleanPnpmPatchPath(raw); err == nil {
				t.Fatalf("accepted unsafe patch path %q", raw)
			}
		})
	}
}

func TestPnpmPatchPathRejectsSymlinkComponents(t *testing.T) {
	fixture := newPnpmFixture(t, testBaseManifest, testBaseLock, "beta", "2.0.0")
	outsideDir := t.TempDir()
	outsideFile := filepath.Join(outsideDir, "outside.patch")
	if err := os.WriteFile(outsideFile, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideFile, filepath.Join(fixture.surface, "linked.patch")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := os.Symlink(outsideDir, filepath.Join(fixture.surface, "linked-dir")); err != nil {
		t.Skipf("directory symlink unavailable: %v", err)
	}
	for _, path := range []string{"linked.patch", "linked-dir/outside.patch"} {
		t.Run(path, func(t *testing.T) {
			if _, _, err := readPnpmPatchInputs(fixture.surface, []byte(pnpmPatchManifest(path))); err == nil || !strings.Contains(err.Error(), "symbolic link") {
				t.Fatalf("symlink path result = %v", err)
			}
		})
	}
}

func TestPnpmPatchFilesHaveIndividualAndAggregateSizeBounds(t *testing.T) {
	fixture := newPnpmFixture(t, testBaseManifest, testBaseLock, "beta", "2.0.0")
	tooLarge := strings.Repeat("x", maxPnpmPatchFileBytes+1)
	writePatchFixture(t, fixture, "patches/too-large.patch", tooLarge)
	if _, _, err := readPnpmPatchInputs(fixture.surface, []byte(pnpmPatchManifest("patches/too-large.patch"))); err == nil {
		t.Fatal("accepted an oversized individual patch")
	}

	paths := []string{"patches/one.patch", "patches/two.patch", "patches/three.patch", "patches/four.patch", "patches/five.patch"}
	manifest := `{"name":"demo-ui","pnpm":{"patchedDependencies":{`
	for i, path := range paths {
		writePatchFixture(t, fixture, path, strings.Repeat("y", (maxPnpmPatchTotalBytes/4)+1))
		if i > 0 {
			manifest += ","
		}
		manifest += fmt.Sprintf("%q:%q", fmt.Sprintf("pkg-%d@1.0.0", i), path)
	}
	manifest += "}}}"
	if _, _, err := readPnpmPatchInputs(fixture.surface, []byte(manifest)); err == nil || !strings.Contains(err.Error(), "aggregate limit") {
		t.Fatalf("aggregate patch size result = %v", err)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
