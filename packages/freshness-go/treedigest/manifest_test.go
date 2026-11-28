package treedigest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func manifestRequest(root string) ManifestRequest {
	return ManifestRequest{
		Primary:       RootSpec{Name: "scenario", Path: root, Selections: []InputSelection{{Glob: "src/**", Required: true}}},
		Configuration: map[string]string{"preset": "quick"},
		Toolchain:     map[string]string{"go": "1.25"},
		Attribution:   ManifestAttribution{Commit: "one", Branch: "agi", Dirty: true},
	}
}

func TestOwnerEnumeratedFilesAreLiteralRequiredAndIndependentOfGit(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "src/[id].tsx", "route")
	writeFile(t, root, "src/i.tsx", "not the route")
	request := ManifestRequest{Primary: RootSpec{Name: "owner", Path: root, Files: []string{"src/[id].tsx"}}}
	noEnumeration := func(string, string, ...string) ([]byte, error) {
		t.Fatal("owner files must not be enumerated again")
		return nil, nil
	}
	manifest, err := BuildInputManifestWithRunner(request, noEnumeration)
	if err != nil || len(manifest.Roots) != 1 || len(manifest.Roots[0].Files) != 1 || manifest.Roots[0].Files[0].Path != "src/[id].tsx" {
		t.Fatalf("literal owner inputs: %+v, %v", manifest, err)
	}
	for _, invalid := range []string{"../outside", "/outside", "C:/outside", "missing.go"} {
		request.Primary.Files = []string{invalid}
		if _, err := BuildInputManifestWithRunner(request, noEnumeration); err == nil {
			t.Fatalf("accepted invalid required file %q", invalid)
		}
	}
	if err := os.Symlink(filepath.Join(root, "src"), filepath.Join(root, "linked")); err == nil {
		request.Primary.Files = []string{"linked/[id].tsx"}
		if _, err := BuildInputManifestWithRunner(request, noEnumeration); err == nil {
			t.Fatal("accepted symlink ancestor")
		}
	}
}

func TestInputManifestIdentityIgnoresAttributionMtimeAndUnrelatedEdits(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-IDENTITY-P0]
	root := t.TempDir()
	writeFile(t, root, "src/main.go", "package main\n")
	writeFile(t, root, "docs/readme.md", "before\n")
	req := manifestRequest(root)
	before, err := BuildInputManifestWithRunner(req, walkOnlyRunner)
	if err != nil {
		t.Fatal(err)
	}

	req.Attribution = ManifestAttribution{Commit: "random-commit", Branch: "other", Dirty: false}
	writeFile(t, root, "docs/readme.md", "after\n")
	path := filepath.Join(root, "src/main.go")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, info.ModTime().AddDate(0, 0, -1), info.ModTime().AddDate(0, 0, -1)); err != nil {
		t.Fatal(err)
	}
	after, err := BuildInputManifestWithRunner(req, walkOnlyRunner)
	if err != nil {
		t.Fatal(err)
	}
	if before.Identity != after.Identity {
		t.Fatalf("attribution, mtime, or unrelated edit changed identity: %s != %s", before.Identity, after.Identity)
	}
}

func TestInputManifestRelevantAndDeclaredDependencyChangesAreScoped(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-IDENTITY-P0]
	consumer := t.TempDir()
	dependency := t.TempDir()
	writeFile(t, consumer, "src/main.go", "package main\n")
	writeFile(t, dependency, "schema/api.proto", "before\n")
	req := manifestRequest(consumer)
	withoutDependency, err := BuildInputManifestWithRunner(req, walkOnlyRunner)
	if err != nil {
		t.Fatal(err)
	}
	req.Dependencies = []RootSpec{{Name: "shared-proto", Path: dependency, Selections: []InputSelection{{Glob: "schema/**", Required: true}}}}
	withDependency, err := BuildInputManifestWithRunner(req, walkOnlyRunner)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, dependency, "schema/api.proto", "after\n")
	changedDependency, err := BuildInputManifestWithRunner(req, walkOnlyRunner)
	if err != nil {
		t.Fatal(err)
	}
	withoutDependencyAgain, err := BuildInputManifestWithRunner(manifestRequest(consumer), walkOnlyRunner)
	if err != nil {
		t.Fatal(err)
	}
	if withDependency.Identity == changedDependency.Identity {
		t.Fatal("declared dependency edit did not change consumer identity")
	}
	if withoutDependency.Identity != withoutDependencyAgain.Identity {
		t.Fatal("dependency edit changed a consumer that did not declare it")
	}
	writeFile(t, consumer, "src/main.go", "package changed\n")
	relevant, err := BuildInputManifestWithRunner(req, walkOnlyRunner)
	if err != nil || relevant.Identity == changedDependency.Identity {
		t.Fatalf("relevant edit was not reflected: %v", err)
	}
}

func TestInputManifestFailsClosedForMissingAndSymlinkInputs(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-IDENTITY-P0]
	root := t.TempDir()
	writeFile(t, root, "other.txt", "x")
	_, err := BuildInputManifestWithRunner(manifestRequest(root), walkOnlyRunner)
	if err == nil || !strings.Contains(err.Error(), "matched no eligible files") {
		t.Fatalf("missing required input error = %v", err)
	}

	writeFile(t, root, "target.go", "package target\n")
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "target.go"), filepath.Join(root, "src", "linked.go")); err != nil {
		t.Fatal(err)
	}
	_, err = BuildInputManifestWithRunner(manifestRequest(root), walkOnlyRunner)
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlink input error = %v", err)
	}
}

func TestInputManifestTreatsTrackedWorktreeDeletionAsAbsentContent(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-IDENTITY-P0]
	root := t.TempDir()
	writeFile(t, root, "src/current.go", "package current\n")
	runner := func(_ string, name string, args ...string) ([]byte, error) {
		if name == "git" && len(args) > 0 && args[0] == "ls-files" {
			// `git ls-files --cached` reports index entries that are intentionally
			// deleted in the working tree. The manifest describes working-tree
			// content, so the deleted path must not become an unreadable input.
			return []byte("src/current.go\nsrc/removed.go\n"), nil
		}
		return nil, errors.New("unexpected command")
	}

	req := manifestRequest(root)
	req.Primary.Selections = []InputSelection{{Glob: "**", Required: true}}
	manifest, err := BuildInputManifestWithRunner(req, runner)
	if err != nil {
		t.Fatalf("tracked worktree deletion must be representable: %v", err)
	}
	files := manifest.Roots[0].Files
	if len(files) != 1 || files[0].Path != "src/current.go" {
		t.Fatalf("manifest should contain only current working-tree bytes: %+v", files)
	}
}

func TestInputManifestExcludesNestedNodeModulesBeforeSymlinkValidation(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-IDENTITY-P0]
	root := t.TempDir()
	writeFile(t, root, "src/main.go", "package main\n")
	link := filepath.Join(root, "ui", "node_modules", "@vrooli", "iframe-bridge")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../../../../packages/iframe-bridge", link); err != nil {
		t.Fatal(err)
	}
	runner := func(_ string, name string, args ...string) ([]byte, error) {
		if name == "git" && len(args) > 0 && args[0] == "ls-files" {
			return []byte("src/main.go\nui/node_modules/@vrooli/iframe-bridge\n"), nil
		}
		return nil, errors.New("unexpected command")
	}

	req := manifestRequest(root)
	req.Primary.Selections = []InputSelection{{Glob: "**", Required: true}}
	manifest, err := BuildInputManifestWithRunner(req, runner)
	if err != nil {
		t.Fatalf("nested node_modules must be excluded from content identity: %v", err)
	}
	files := manifest.Roots[0].Files
	if len(files) != 1 || files[0].Path != "src/main.go" {
		t.Fatalf("manifest should exclude nested node_modules: %+v", files)
	}
}

func TestInputManifestRejectsPortableCaseCollisions(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-IDENTITY-P0]
	root := t.TempDir()
	writeFile(t, root, "src/A.go", "package a\n")
	writeFile(t, root, "src/a.go", "package a\n")
	_, err := BuildInputManifestWithRunner(manifestRequest(root), walkOnlyRunner)
	if err == nil || !strings.Contains(err.Error(), "portable path collision") {
		t.Fatalf("case collision error = %v", err)
	}
}

func TestInputManifestRejectsMidReadMutation(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-IDENTITY-P0]
	root := t.TempDir()
	writeFile(t, root, "src/main.go", "before\n")
	deps := manifestDeps{run: walkOnlyRunner, lstat: os.Lstat}
	deps.readFile = func(path string) ([]byte, error) {
		data, err := os.ReadFile(path)
		if err == nil {
			err = os.WriteFile(path, []byte("after mutation\n"), 0o644)
		}
		return data, err
	}
	_, err := buildInputManifest(manifestRequest(root), deps)
	if err == nil || !strings.Contains(err.Error(), "changed while") {
		t.Fatalf("mid-read mutation error = %v", err)
	}
}

func TestInputManifestRoundTripValidationDetectsTampering(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-IDENTITY-P0]
	root := t.TempDir()
	writeFile(t, root, "src/main.go", "package main\n")
	manifest, err := BuildInputManifestWithRunner(manifestRequest(root), walkOnlyRunner)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var decoded InputManifest
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if err := decoded.Validate(); err != nil {
		t.Fatalf("valid round trip rejected: %v", err)
	}
	decoded.Roots[0].Files[0].Digest = "sha256:" + strings.Repeat("0", 64)
	if err := decoded.Validate(); err == nil || !strings.Contains(err.Error(), "root identity mismatch") {
		t.Fatalf("tampered manifest validation error = %v", err)
	}
}

func TestInputManifestGoldenIncludesTrackedAndUntrackedNotIgnoredFiles(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-IDENTITY-P0]
	root := t.TempDir()
	writeFile(t, root, "src/tracked.go", "package golden\n")
	writeFile(t, root, "src/untracked.txt", "included\n")
	writeFile(t, root, "src/ignored.tmp", "excluded\n")
	runner := func(string, string, ...string) ([]byte, error) {
		return []byte("src/tracked.go\nsrc/untracked.txt\n"), nil
	}
	manifest, err := BuildInputManifestWithRunner(ManifestRequest{Primary: RootSpec{Name: "scenario", Path: root, Selections: []InputSelection{{Glob: "src/**", Required: true}}}}, runner)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	actual = append(actual, '\n')
	expected, err := os.ReadFile(filepath.Join("testdata", "manifest-golden.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, expected) {
		t.Fatalf("golden manifest mismatch\nactual:\n%s", actual)
	}
}

func TestManifestBuilderCacheCannotHideRelevantEdit(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-IDENTITY-P0]
	root := t.TempDir()
	writeFile(t, root, "src/a.txt", "same bytes\n")
	writeFile(t, root, "src/b.txt", "same bytes\n")
	builder := NewManifestBuilder(1024)
	req := ManifestRequest{Primary: RootSpec{Name: "scenario", Path: root, Selections: []InputSelection{{Glob: "src/**", Required: true}}}}
	before, err := builder.Build(req)
	if err != nil {
		t.Fatal(err)
	}
	if stats := builder.CacheStats(); stats.Hits == 0 || stats.Misses == 0 {
		t.Fatalf("cache did not exercise exact-byte hit and miss: %#v", stats)
	}
	writeFile(t, root, "src/a.txt", "new! bytes\n")
	after, err := builder.Build(req)
	if err != nil {
		t.Fatal(err)
	}
	if before.Identity == after.Identity {
		t.Fatal("exact-byte cache hid a relevant edit")
	}
}

func TestPortableManifestPathNormalization(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-IDENTITY-P0]
	cases := map[string]string{
		`./src\\nested\\file.go`: "src/nested/file.go",
		"src//nested/./file.go":  "src/nested/file.go",
		"src/../file.go":         "file.go",
	}
	for input, expected := range cases {
		if actual := normalizeManifestPath(input); actual != expected {
			t.Errorf("normalizeManifestPath(%q) = %q, want %q", input, actual, expected)
		}
	}
	for _, absolute := range []string{"/tmp/file", `C:\\repo\\file`, `z:/repo/file`} {
		if !portableAbsolutePath(normalizeManifestPath(absolute)) {
			t.Errorf("portable absolute path not recognized: %q", absolute)
		}
	}
}

func TestInputManifestContextCancellationIsExplicit(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-IDENTITY-P0]
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := BuildInputManifestContext(ctx, manifestRequest(t.TempDir()))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled capture error = %v", err)
	}
}

func BenchmarkBuildInputManifestColdWarm(b *testing.B) {
	root := b.TempDir()
	for i := 0; i < 64; i++ {
		writeFileBenchmark(b, root, filepath.Join("src", fmt.Sprintf("file-%03d.bin", i)), strings.Repeat(string(rune('a'+i%26)), 16*1024))
	}
	req := ManifestRequest{Primary: RootSpec{Name: "scenario", Path: root, Selections: []InputSelection{{Glob: "src/**", Required: true}}}}
	b.Run("cold-64-files-1MiB", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if _, err := NewManifestBuilder(2 << 20).Build(req); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("warm-64-files-1MiB", func(b *testing.B) {
		builder := NewManifestBuilder(2 << 20)
		if _, err := builder.Build(req); err != nil {
			b.Fatal(err)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := builder.Build(req); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func writeFileBenchmark(b *testing.B, root, rel, value string) {
	b.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		b.Fatal(err)
	}
}
