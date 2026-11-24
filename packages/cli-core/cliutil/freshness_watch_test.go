package cliutil

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
	"time"
)

// watchFixtureSpec is the installed-resource spec over replaceFixture plus a
// glob input and a declared input that does not exist yet.
func watchFixtureSpec(t *testing.T) (repo, resourceRoot string, spec FreshnessSpec) {
	t.Helper()
	repo, resourceRoot, modulePath := replaceFixture(t)
	mustWriteFile(t, filepath.Join(resourceRoot, "docs", "guide.md"), "# guide\n")
	if err := os.MkdirAll(filepath.Join(modulePath, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	spec = CanonicalResourceGoModuleFreshnessSpec(resourceRoot, modulePath, "resource-demo", []string{"cli/**", "resource.json", "docs/*.md", "README.md"})
	return repo, resourceRoot, spec
}

// watchTrustedAt is a manifest write time far enough after the fixture was
// written that no watched object falls inside the racy window.
func watchTrustedAt() int64 { return time.Now().Add(time.Hour).UnixNano() }

func TestFreshnessWatchProvesAnUnchangedTree(t *testing.T) {
	_, _, spec := watchFixtureSpec(t)
	manifest, err := ComputeFreshnessManifest(spec, "cli", nil, watchTrustedAt())
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Watch == nil || !manifest.Watch.ExpansionStable {
		t.Fatalf("expected an expansion-stable watch, got %+v", manifest.Watch)
	}
	if !manifest.Watch.Holds(spec, manifest.WrittenAtNS) {
		t.Fatal("watch must hold for the unchanged tree it recorded")
	}
	// Every hashed file is watched, so a content change cannot hide.
	watched := map[string]bool{}
	for _, entry := range manifest.Watch.Entries {
		watched[entry.Rel] = true
	}
	for _, file := range manifest.Files {
		if !watched[file.Rel] {
			t.Errorf("hashed input %s is not watched", file.Rel)
		}
	}
	if verdict, err := EvaluateFreshness(spec, manifest, nil); err != nil || verdict.Stale {
		t.Fatalf("EvaluateFreshness = %+v, %v; want fresh", verdict, err)
	}
}

// Each mutation must defeat the watch. Mutations that change the input set or
// content must also be reported stale by the full evaluation the watch falls
// back to; build-output noise must not.
func TestFreshnessWatchRefusesEveryChange(t *testing.T) {
	type mutation struct {
		name      string
		mutate    func(t *testing.T, repo, resourceRoot string)
		wantStale bool
	}
	mutations := []mutation{
		{"same-size content edit", func(t *testing.T, _, root string) {
			mustWriteFile(t, filepath.Join(root, "resource.json"), "[]\n")
		}, true},
		{"new file in a walked directory", func(t *testing.T, _, root string) {
			mustWriteFile(t, filepath.Join(root, "cli", "extra.go"), "package main\n")
		}, true},
		{"first file in a directory that was empty", func(t *testing.T, _, root string) {
			mustWriteFile(t, filepath.Join(root, "cli", "empty", "late.go"), "package empty\n")
		}, true},
		{"declared input that did not exist appears", func(t *testing.T, _, root string) {
			mustWriteFile(t, filepath.Join(root, "README.md"), "# demo\n")
		}, true},
		{"new glob match", func(t *testing.T, _, root string) {
			mustWriteFile(t, filepath.Join(root, "docs", "faq.md"), "# faq\n")
		}, true},
		{"input removed", func(t *testing.T, _, root string) {
			if err := os.Remove(filepath.Join(root, "docs", "guide.md")); err != nil {
				t.Fatal(err)
			}
		}, true},
		{"shared package edit", func(t *testing.T, repo, _ string) {
			mustWriteFile(t, filepath.Join(repo, "packages", "cli-core", "core.go"), "package core // edited\n")
		}, true},
		{"build output beside the sources", func(t *testing.T, _, root string) {
			mustWriteFile(t, filepath.Join(root, "cli", "coverage.out"), "mode: set\n")
		}, false},
	}
	for _, tc := range mutations {
		t.Run(tc.name, func(t *testing.T) {
			repo, resourceRoot, spec := watchFixtureSpec(t)
			manifest, err := ComputeFreshnessManifest(spec, "cli", nil, watchTrustedAt())
			if err != nil {
				t.Fatal(err)
			}
			tc.mutate(t, repo, resourceRoot)
			if manifest.Watch.Holds(spec, manifest.WrittenAtNS) {
				t.Fatal("watch still holds after the change")
			}
			verdict, err := EvaluateFreshness(spec, manifest, nil)
			if err != nil {
				t.Fatal(err)
			}
			if verdict.Stale != tc.wantStale {
				t.Fatalf("EvaluateFreshness stale = %v (%+v), want %v", verdict.Stale, verdict, tc.wantStale)
			}
		})
	}
}

// A rewrite that restores size and mtime still moves ctime, which user space
// cannot set back; the stat-cache alone would miss it.
func TestFreshnessWatchSeesContentRewriteWithRestoredMTime(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("ctime is only recorded on linux")
	}
	_, resourceRoot, spec := watchFixtureSpec(t)
	path := filepath.Join(resourceRoot, "resource.json")
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := ComputeFreshnessManifest(spec, "cli", nil, watchTrustedAt())
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(path, []byte("[]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, before.ModTime(), before.ModTime()); err != nil {
		t.Fatal(err)
	}
	if manifest.Watch.Holds(spec, manifest.WrittenAtNS) {
		t.Fatal("watch holds after a content rewrite that restored mtime")
	}
}

// The replace closure is derived from imports in files outside the declared
// inputs (the proto tree is represented by its stamp). Those files are
// watched, so a new import there cannot leave a stale expansion trusted.
func TestFreshnessWatchCoversReplaceClosureDerivation(t *testing.T) {
	repo, _, spec := watchFixtureSpec(t)
	manifest, err := ComputeFreshnessManifest(spec, "cli", nil, watchTrustedAt())
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(repo, "packages", "proto", "gen", "go", "x", "x.pb.go"), "package x\n\nimport _ \"example.com/unused\"\n\nconst Name = \"x\"\n")
	if manifest.Watch.Holds(spec, manifest.WrittenAtNS) {
		t.Fatal("watch holds after the import closure changed")
	}
	if expanded := WithGoReplaceFreshnessInputs(spec); slices.Equal(expanded.Inputs, spec.Inputs) {
		t.Fatal("fixture should change the expansion; the test proves nothing otherwise")
	}
}

func TestFreshnessWatchIsBoundToItsSpecAndWriteTime(t *testing.T) {
	_, _, spec := watchFixtureSpec(t)
	manifest, err := ComputeFreshnessManifest(spec, "cli", nil, watchTrustedAt())
	if err != nil {
		t.Fatal(err)
	}
	other := spec
	other.SkipFiles = append(append([]string(nil), spec.SkipFiles...), "another-binary")
	if manifest.Watch.Holds(other, manifest.WrittenAtNS) {
		t.Fatal("watch must not vouch for a different spec")
	}
	reordered := spec
	reordered.SkipFiles = append([]string{"resource-demo"}, spec.SkipFiles...)
	if !manifest.Watch.Holds(reordered, manifest.WrittenAtNS) {
		t.Fatal("skip lists are sets; a duplicate entry must not change the key")
	}

	racy, err := ComputeFreshnessManifest(spec, "cli", nil, time.Now().UnixNano())
	if err != nil {
		t.Fatal(err)
	}
	if racy.Watch.Holds(spec, racy.WrittenAtNS) {
		t.Fatal("objects changed inside the racy window must not be trusted")
	}
}

func TestFreshnessWatchMarksUnexpandedSpecsUnstable(t *testing.T) {
	_, resourceRoot, _ := watchFixtureSpec(t)
	unexpanded := FreshnessSpec{
		SourceRoot:  filepath.Join(resourceRoot, "cli"),
		ContextRoot: resourceRoot,
		Inputs:      []string{"cli/**", "resource.json", "../../packages/cli-core", "../../packages/proto/gen/.vrooli-proto-artifact.json"},
		SkipFiles:   []string{"resource-demo"},
	}
	manifest, err := ComputeFreshnessManifest(unexpanded, "cli", nil, watchTrustedAt())
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Watch == nil || manifest.Watch.ExpansionStable {
		t.Fatalf("a spec the replace closure would extend must not claim a stable expansion: %+v", manifest.Watch)
	}
}

// installedChecker returns a StaleChecker for the fixture's resource CLI with
// a manifest written beside a stand-in installed binary.
func installedChecker(t *testing.T, resourceRoot string) (*StaleChecker, FreshnessSpec) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	checker := &StaleChecker{
		AppName:           "resource-demo",
		BuildSourceRoot:   filepath.Join(resourceRoot, "cli"),
		SourceContextPath: "..",
		FreshnessInputs:   []string{"cli/**", "resource.json", "docs/*.md", "README.md", "../../packages/cli-core", "../../packages/proto/gen/.vrooli-proto-artifact.json"},
		InstallKind:       "resource",
		InstallName:       "demo",
		InstalledManifest: filepath.Join(t.TempDir(), "resource-demo.freshness.json"),
		LookPathFunc:      func(string) (string, error) { return "/usr/bin/go", nil },
	}
	// The installer bakes the expanded inputs; the runtime starts from them.
	spec := checker.freshnessSpec(checker.BuildSourceRoot)
	checker.FreshnessInputs = spec.Inputs
	spec.SkipFiles = []string{filepath.Base(executable)}
	return checker, spec
}

func TestStaleCheckerConfirmsInstalledBinaryFromWatchWithoutHashing(t *testing.T) {
	_, resourceRoot, _ := watchFixtureSpec(t)
	checker, spec := installedChecker(t, resourceRoot)
	manifest, err := ComputeFreshnessManifest(spec, "cli", nil, watchTrustedAt())
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteFreshnessManifest(checker.InstalledManifest, manifest); err != nil {
		t.Fatal(err)
	}
	checker.BuildFingerprint = manifest.Digest
	hashed := 0
	checker.FingerprintFunc = func(FreshnessSpec) (string, error) { hashed++; return manifest.Digest, nil }
	checker.CommandRunner = func(*exec.Cmd) error { t.Fatal("no refresh expected"); return nil }

	if checkStaleForTest(t, checker) || hashed != 0 {
		t.Fatalf("unchanged tree: restarted or hashed %d times; want the watch to decide", hashed)
	}

	mustWriteFile(t, filepath.Join(resourceRoot, "cli", "extra.go"), "package main\n")
	refreshed := false
	checker.FingerprintFunc = func(FreshnessSpec) (string, error) { hashed++; return "changed", nil }
	checker.CommandRunner = func(*exec.Cmd) error { refreshed = true; return nil }
	checker.Reexec = func(string, []string) error { return nil }
	if !checkStaleForTest(t, checker) || hashed != 1 || !refreshed {
		t.Fatalf("changed tree: hashed %d times, refreshed %v; want a full check and a refresh", hashed, refreshed)
	}
}

func TestStaleCheckerRestampsInstalledManifestAfterHarmlessChange(t *testing.T) {
	_, resourceRoot, _ := watchFixtureSpec(t)
	checker, spec := installedChecker(t, resourceRoot)
	manifest, err := ComputeFreshnessManifest(spec, "cli", nil, watchTrustedAt())
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteFreshnessManifest(checker.InstalledManifest, manifest); err != nil {
		t.Fatal(err)
	}
	checker.BuildFingerprint = manifest.Digest
	checker.CommandRunner = func(*exec.Cmd) error { t.Fatal("no refresh expected"); return nil }
	checker.now = func() time.Time { return time.Now().Add(time.Hour) }

	// Noise the watch sees but the inputs exclude: the full check confirms the
	// build and restamps, after which the watch decides again.
	mustWriteFile(t, filepath.Join(resourceRoot, "cli", "coverage.out"), "mode: set\n")
	if checkStaleForTest(t, checker) {
		t.Fatal("unexpected restart")
	}
	restamped, ok, err := ReadFreshnessManifest(checker.InstalledManifest)
	if err != nil || !ok {
		t.Fatalf("read restamped manifest: ok=%v err=%v", ok, err)
	}
	if restamped.Digest != manifest.Digest || restamped.CheckType != "cli" || !slices.Equal(restamped.Inputs, manifest.Inputs) {
		t.Fatalf("restamp changed the build identity: %+v", restamped)
	}
	checker.FingerprintFunc = func(FreshnessSpec) (string, error) { t.Fatal("watch should decide after the restamp"); return "", nil }
	if checkStaleForTest(t, checker) {
		t.Fatal("unexpected restart")
	}
}

func TestStaleCheckerNeverRestampsAnotherBuildsManifest(t *testing.T) {
	_, resourceRoot, _ := watchFixtureSpec(t)
	checker, spec := installedChecker(t, resourceRoot)
	manifest, err := ComputeFreshnessManifest(spec, "cli", nil, watchTrustedAt())
	if err != nil {
		t.Fatal(err)
	}
	checker.BuildFingerprint = manifest.Digest
	newer := manifest
	newer.Digest = "a-newer-install"
	if err := WriteFreshnessManifest(checker.InstalledManifest, newer); err != nil {
		t.Fatal(err)
	}
	checker.now = func() time.Time { return time.Now().Add(time.Hour) }
	if checkStaleForTest(t, checker) {
		t.Fatal("unexpected restart")
	}
	current, _, err := ReadFreshnessManifest(checker.InstalledManifest)
	if err != nil || current.Digest != "a-newer-install" {
		t.Fatalf("a process must not overwrite another build's manifest: digest=%q err=%v", current.Digest, err)
	}
}

// benchmarkResourceTree writes a resource CLI tree shaped like the large
// resource CLIs: dirs directories of filesPerDir Go files plus shared packages.
func benchmarkResourceTree(b *testing.B, dirs, filesPerDir int) (*StaleChecker, FreshnessSpec) {
	b.Helper()
	repo := b.TempDir()
	resourceRoot := filepath.Join(repo, "resources", "bench")
	write := func(path, body string) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			b.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			b.Fatal(err)
		}
	}
	write(filepath.Join(resourceRoot, "cli", "go.mod"), "module example.com/bench\n\ngo 1.25.0\n")
	write(filepath.Join(resourceRoot, "resource.json"), "{}\n")
	body := fmt.Sprintf("package p\n\n// %0*d\n", 2048, 0)
	for d := 0; d < dirs; d++ {
		for f := 0; f < filesPerDir; f++ {
			tree := "cli"
			if d%2 == 1 {
				tree = filepath.Join("..", "..", "packages", "shared")
			}
			write(filepath.Join(resourceRoot, tree, fmt.Sprintf("pkg%03d", d), fmt.Sprintf("file%02d.go", f)), body)
		}
	}
	write(filepath.Join(resourceRoot, "cli", "main.go"), "package main\n")
	checker := &StaleChecker{
		AppName:           "bench",
		BuildSourceRoot:   filepath.Join(resourceRoot, "cli"),
		SourceContextPath: "..",
		FreshnessInputs:   []string{"cli/**", "resource.json", "../../packages/shared"},
		InstallKind:       "resource",
		InstallName:       "bench",
		InstalledManifest: filepath.Join(b.TempDir(), "bench.freshness.json"),
	}
	spec := checker.freshnessSpec(checker.BuildSourceRoot)
	checker.FreshnessInputs = spec.Inputs
	executable, err := os.Executable()
	if err != nil {
		b.Fatal(err)
	}
	spec.SkipFiles = []string{filepath.Base(executable)}
	return checker, spec
}

// BenchmarkStaleCheck compares the per-invocation cost of confirming an
// installed CLI current: the full fingerprint (enumerate, parse, hash) against
// the installed manifest's watch. 300 directories x 10 files approximates
// resource-ollama's ~3,000 inputs.
func BenchmarkStaleCheck(b *testing.B) {
	for _, mode := range []string{"full-fingerprint", "installed-watch"} {
		b.Run(mode, func(b *testing.B) {
			checker, spec := benchmarkResourceTree(b, 300, 10)
			manifest, err := ComputeFreshnessManifest(spec, "cli", nil, watchTrustedAt())
			if err != nil {
				b.Fatal(err)
			}
			checker.BuildFingerprint = manifest.Digest
			if mode == "installed-watch" {
				if err := WriteFreshnessManifest(checker.InstalledManifest, manifest); err != nil {
					b.Fatal(err)
				}
			} else {
				// No installed manifest, and none can be restamped (its parent is
				// a file): every iteration takes the full path.
				blocker := filepath.Join(b.TempDir(), "not-a-directory")
				if err := os.WriteFile(blocker, nil, 0o644); err != nil {
					b.Fatal(err)
				}
				checker.InstalledManifest = filepath.Join(blocker, "bench.freshness.json")
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if restarted, err := checker.CheckAndMaybeRebuild(); restarted || err != nil {
					b.Fatalf("restarted=%v err=%v", restarted, err)
				}
			}
		})
	}
}

func TestFreshnessWatchEntriesRoundTripAnyPath(t *testing.T) {
	entries := FreshnessWatchEntries{
		{Rel: "cli/main.go", Size: 12, MTimeNS: 1, CTimeNS: 2, Inode: 3},
		{Rel: "../../packages/cli-core", Dir: true, Size: 4096, MTimeNS: 4},
		{Rel: `docs/a "quoted" \ path & <more>.md`, Size: 1, MTimeNS: 5, CTimeNS: 6, Inode: 7},
	}
	for _, subset := range []FreshnessWatchEntries{entries, entries[:2], nil} {
		data, err := json.Marshal(FreshnessWatch{SpecKey: "k", Entries: subset})
		if err != nil {
			t.Fatal(err)
		}
		var decoded FreshnessWatch
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("decode %s: %v", data, err)
		}
		if !slices.Equal(decoded.Entries, subset) {
			t.Fatalf("round trip changed entries:\n got  %+v\n want %+v", decoded.Entries, subset)
		}
	}
	if _, err := json.Marshal(FreshnessWatchEntries{{Rel: "bad\nname"}}); err == nil {
		t.Fatal("a path with a line break must not be encoded")
	}
}
