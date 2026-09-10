package lifecycle

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/packagegov"
	"github.com/vrooli/vrooli/packages/proto/protogen"
)

func TestSharedPackageDependenciesDeriveGovernedFileDependencies(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	dependencies, err := sharedPackageDependencies(repoRoot, filepath.Join(repoRoot, "scenarios", "test-genie", "ui", "package.json"))
	if err != nil {
		t.Fatalf("sharedPackageDependencies: %v", err)
	}
	var names []string
	for _, dependency := range dependencies {
		names = append(names, dependency.Name)
	}
	if got := strings.Join(names, ","); got != "@vrooli/api-base,@vrooli/iframe-bridge" {
		t.Fatalf("governed file dependencies = %q", got)
	}
}

// helper: digest of the current declared-output path set.
func outputsDigestFor(t *testing.T, root string, patterns []string) string {
	t.Helper()
	outputs, err := declaredOutputFiles(root, patterns)
	if err != nil {
		t.Fatal(err)
	}
	return sharedPackageOutputsListDigest(root, outputs)
}

// helper: a package with one source and one generated output.
func writeSharedPackageFixture(t *testing.T, root, sourceBody, outputBody string) (string, string) {
	t.Helper()
	source := filepath.Join(root, "src", "index.ts")
	output := filepath.Join(root, "dist", "index.js")
	for _, dir := range []string{filepath.Dir(source), filepath.Dir(output)} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(source, []byte(sourceBody), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, []byte(outputBody), 0o644); err != nil {
		t.Fatal(err)
	}
	return source, output
}

func TestSharedPackageOutputsFreshTracksSourceContent(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	patterns := []string{"dist/**"}
	source, _ := writeSharedPackageFixture(t, root, "export const value = 1;\n", "export const value = 1;\n")

	// Nothing recorded yet: the outputs exist but nothing vouches for them.
	fresh, err := sharedPackageOutputsFresh(home, root, "build", patterns)
	if err != nil {
		t.Fatal(err)
	}
	if fresh {
		t.Fatal("outputs with no recorded stamp must not be reported fresh")
	}

	digest, err := sharedPackageSourceDigest(root, patterns)
	if err != nil {
		t.Fatal(err)
	}
	stampPath, err := sharedPackageStampPath(home, root, "build")
	if err != nil {
		t.Fatal(err)
	}
	if err := writeSharedPackageStamp(stampPath, "@vrooli/example", "build", digest, outputsDigestFor(t, root, patterns)); err != nil {
		t.Fatal(err)
	}

	fresh, err = sharedPackageOutputsFresh(home, root, "build", patterns)
	if err != nil {
		t.Fatal(err)
	}
	if !fresh {
		t.Fatal("unchanged sources with a recorded stamp must be fresh")
	}

	// A source edit must invalidate, whatever the mtimes say.
	if err := os.WriteFile(source, []byte("export const value = 2;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fresh, err = sharedPackageOutputsFresh(home, root, "build", patterns)
	if err != nil {
		t.Fatal(err)
	}
	if fresh {
		t.Fatal("changed source content must invalidate freshness")
	}
}

// TestSharedPackageOutputsFreshIgnoresFrozenOutputMtime pins the production
// bug this gate replaced. A content-addressed publish leaves an unchanged
// output's mtime far in the past; under the previous mtime comparison that one
// frozen file reported the whole package stale forever, so every scenario
// start paid a full regeneration.
func TestSharedPackageOutputsFreshIgnoresFrozenOutputMtime(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	patterns := []string{"dist/**"}
	_, output := writeSharedPackageFixture(t, root, "export const value = 1;\n", "export const value = 1;\n")

	// A marker file whose bytes never change, so publish never rewrites it.
	marker := filepath.Join(root, "dist", "py.typed")
	if err := os.WriteFile(marker, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	digest, err := sharedPackageSourceDigest(root, patterns)
	if err != nil {
		t.Fatal(err)
	}
	stampPath, err := sharedPackageStampPath(home, root, "build")
	if err != nil {
		t.Fatal(err)
	}
	if err := writeSharedPackageStamp(stampPath, "@vrooli/example", "build", digest, outputsDigestFor(t, root, patterns)); err != nil {
		t.Fatal(err)
	}

	// Age both outputs well past every source. mtime says "stale"; content
	// says "unchanged", and content is what must win.
	ancient := time.Now().Add(-30 * 24 * time.Hour)
	for _, file := range []string{output, marker} {
		if err := os.Chtimes(file, ancient, ancient); err != nil {
			t.Fatal(err)
		}
	}

	fresh, err := sharedPackageOutputsFresh(home, root, "build", patterns)
	if err != nil {
		t.Fatal(err)
	}
	if !fresh {
		t.Fatal("an output older than its sources must still be fresh when sources are unchanged")
	}
}

// A stamp must not vouch for outputs that are no longer on disk.
func TestSharedPackageOutputsFreshRequiresOutputsToExist(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	patterns := []string{"dist/**"}
	_, output := writeSharedPackageFixture(t, root, "export const value = 1;\n", "export const value = 1;\n")

	digest, err := sharedPackageSourceDigest(root, patterns)
	if err != nil {
		t.Fatal(err)
	}
	stampPath, err := sharedPackageStampPath(home, root, "build")
	if err != nil {
		t.Fatal(err)
	}
	if err := writeSharedPackageStamp(stampPath, "@vrooli/example", "build", digest, outputsDigestFor(t, root, patterns)); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(output); err != nil {
		t.Fatal(err)
	}

	fresh, err := sharedPackageOutputsFresh(home, root, "build", patterns)
	if err != nil {
		t.Fatal(err)
	}
	if fresh {
		t.Fatal("missing outputs must never be reported fresh")
	}
}

// Without a runtime home there is nowhere to record a stamp, so the gate must
// fail closed rather than assume freshness.
func TestSharedPackageOutputsFreshFailsClosedWithoutHome(t *testing.T) {
	root := t.TempDir()
	writeSharedPackageFixture(t, root, "export const value = 1;\n", "export const value = 1;\n")
	fresh, err := sharedPackageOutputsFresh("", root, "build", []string{"dist/**"})
	if err != nil {
		t.Fatal(err)
	}
	if fresh {
		t.Fatal("no runtime home must mean not fresh")
	}
}

// The digest must ignore the generated tree; otherwise generating would always
// invalidate the very stamp it just wrote.
func TestSharedPackageSourceDigestExcludesDeclaredOutputs(t *testing.T) {
	root := t.TempDir()
	patterns := []string{"dist/**"}
	_, output := writeSharedPackageFixture(t, root, "export const value = 1;\n", "export const value = 1;\n")

	before, err := sharedPackageSourceDigest(root, patterns)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, []byte("export const value = 999;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := sharedPackageSourceDigest(root, patterns)
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatal("changing a declared output must not change the source digest")
	}
}

// Renaming a source file changes the inputs even when total bytes are equal.
func TestSharedPackageSourceDigestIsPathSensitive(t *testing.T) {
	root := t.TempDir()
	patterns := []string{"dist/**"}
	source, _ := writeSharedPackageFixture(t, root, "export const value = 1;\n", "out\n")

	before, err := sharedPackageSourceDigest(root, patterns)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(source, filepath.Join(filepath.Dir(source), "renamed.ts")); err != nil {
		t.Fatal(err)
	}
	after, err := sharedPackageSourceDigest(root, patterns)
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Fatal("renaming a source must change the source digest")
	}
}

func TestSharedPackageOutputsFreshTracksSelectedArtifactPointer(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	patterns := []string{"dist/**"}
	writeSharedPackageFixture(t, root, "export const value = 1;\n", "export const value = 1;\n")
	selectionRoot := filepath.Join(home, ".vrooli", "artifacts", "react-component-library")
	if err := os.MkdirAll(selectionRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	selectionPath := filepath.Join(selectionRoot, "current.json")
	if err := os.WriteFile(selectionPath, []byte(`{"id":"sha256:one"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	commandInputs := [][]string{nil, nil, {"@vrooli/react-component-library"}}
	digest, err := sharedPackageSourceDigestWithHome(home, root, patterns, commandInputs...)
	if err != nil {
		t.Fatal(err)
	}
	stampPath, err := sharedPackageStampPath(home, root, "build")
	if err != nil {
		t.Fatal(err)
	}
	if err := writeSharedPackageStamp(stampPath, "@vrooli/react-component-library", "build", digest, outputsDigestFor(t, root, patterns)); err != nil {
		t.Fatal(err)
	}
	fresh, err := sharedPackageOutputsFresh(home, root, "build", patterns, commandInputs...)
	if err != nil || !fresh {
		t.Fatalf("unchanged selected artifact should be fresh: fresh=%v err=%v", fresh, err)
	}
	draftRoot := filepath.Join(filepath.Dir(root), "draft-catalog")
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(draftRoot, "unpublished-component.tsx"), []byte("broken draft\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fresh, err = sharedPackageOutputsFresh(home, root, "build", patterns, commandInputs...)
	if err != nil || !fresh {
		t.Fatalf("unrelated external draft should not invalidate selected artifact: fresh=%v err=%v", fresh, err)
	}
	if err := os.WriteFile(selectionPath, []byte(`{"id":"sha256:two"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	fresh, err = sharedPackageOutputsFresh(home, root, "build", patterns, commandInputs...)
	if err != nil {
		t.Fatal(err)
	}
	if fresh {
		t.Fatal("changed selected artifact pointer must invalidate package freshness")
	}
}

// A stamp recorded under an older hashing scheme must not be trusted.
func TestSharedPackageOutputsFreshRejectsStaleStampVersion(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	patterns := []string{"dist/**"}
	writeSharedPackageFixture(t, root, "export const value = 1;\n", "export const value = 1;\n")

	digest, err := sharedPackageSourceDigest(root, patterns)
	if err != nil {
		t.Fatal(err)
	}
	stampPath, err := sharedPackageStampPath(home, root, "build")
	if err != nil {
		t.Fatal(err)
	}
	if err := writeSharedPackageStamp(stampPath, "@vrooli/example", "build", digest, outputsDigestFor(t, root, patterns)); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(stampPath)
	if err != nil {
		t.Fatal(err)
	}
	current := fmt.Sprintf(`"version":%d`, sharedPackageStampVersion)
	downgraded := strings.Replace(string(raw), current, `"version":0`, 1)
	if downgraded == string(raw) {
		t.Fatalf("could not downgrade stamp version in %s", raw)
	}
	if err := os.WriteFile(stampPath, []byte(downgraded), 0o644); err != nil {
		t.Fatal(err)
	}

	fresh, err := sharedPackageOutputsFresh(home, root, "build", patterns)
	if err != nil {
		t.Fatal(err)
	}
	if fresh {
		t.Fatal("a stamp from an older version must not be trusted")
	}
}

// Two commands in one package must not share a stamp.
func TestSharedPackageStampPathIsPerCommand(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	generate, err := sharedPackageStampPath(home, root, "generate")
	if err != nil {
		t.Fatal(err)
	}
	build, err := sharedPackageStampPath(home, root, "build")
	if err != nil {
		t.Fatal(err)
	}
	if generate == build {
		t.Fatal("stamp paths for different commands must differ")
	}
}

func TestProvisionSharedPackageReportsMissingDeclaredOutput(t *testing.T) {
	dependency := sharedPackageDependency{
		Name:  "@vrooli/example",
		Root:  t.TempDir(),
		Build: []packagegov.CommandSpec{{Name: "build", Run: []string{"sh", "-c", "true"}, Outputs: []string{"dist/**"}}},
	}
	err := provisionSharedPackage(dependency, os.Stdout, os.Stdout)
	var provisioningErr *SharedPackageProvisioningError
	if !errors.As(err, &provisioningErr) {
		t.Fatalf("error = %v, want SharedPackageProvisioningError", err)
	}
	if !strings.Contains(provisioningErr.Error(), "@vrooli/example") || !strings.Contains(provisioningErr.Error(), "sh -c true") {
		t.Fatalf("error = %v, want package and command", provisioningErr)
	}
}

func TestProvisionSelectedProtoArtifactDoesNotRunMutableSourceGeneration(t *testing.T) {
	home := t.TempDir()
	packageRoot := t.TempDir()
	generated := filepath.Join(t.TempDir(), "generated")
	output := filepath.Join(generated, "typescript", "demo", "demo.ts")
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, []byte("export const selected = true;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	metadata, err := protogen.BuildArtifactMetadata(generated, "sha256:source", "sha256:generator", []string{"schemas/demo/v1/demo.proto"}, []string{"demo"}, "")
	if err != nil {
		t.Fatal(err)
	}
	store, err := protogen.NewArtifactStore(protogen.DefaultArtifactRoot(home))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Publish(context.Background(), protogen.Candidate{Metadata: metadata, SourceRoot: generated}); err != nil {
		t.Fatal(err)
	}
	dependency := sharedPackageDependency{Name: "proto", Root: packageRoot, Generation: []packagegov.CommandSpec{{Name: "generate", Run: []string{"this-command-must-not-run"}, Outputs: []string{"gen/**"}}}}
	var log bytes.Buffer
	if err := provisionSharedPackageWithOptions(dependency, &log, &log, sharedPackageProvisionOptions{Context: context.Background(), Home: home, Scenario: "command-center"}); err != nil {
		t.Fatal(err)
	}
	selected, err := os.ReadFile(filepath.Join(packageRoot, "gen", "typescript", "demo", "demo.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if string(selected) != "export const selected = true;\n" {
		t.Fatalf("materialized Proto output = %q", selected)
	}
	if !strings.Contains(log.String(), `proto-artifact event=selected`) {
		t.Fatalf("selection event missing from lifecycle log: %s", log.String())
	}
}

func TestProvisionSharedPackagePromotesRuntimeDegradedEvidence(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	var log bytes.Buffer
	dependency := sharedPackageDependency{
		Name: "@vrooli/react-component-library",
		Root: root,
		Build: []packagegov.CommandSpec{{
			Name:    "build",
			Run:     []string{"sh", "-c", `mkdir -p dist; printf built > dist/index.js; printf '%s\n' '{"built":"@vrooli/react-component-library","status":"degraded","reason":"candidate compile failed","candidate":"candidate:test","selectedArtifact":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","compatibility":"verified"}'`},
			Outputs: []string{"dist/**"},
		}},
	}
	var output bytes.Buffer
	if err := provisionSharedPackageWithOptions(dependency, &output, &log, sharedPackageProvisionOptions{Home: home}); err != nil {
		t.Fatalf("provision shared package: %v", err)
	}
	if !strings.Contains(log.String(), `shared-package-runtime event=status package="@vrooli/react-component-library" status="degraded"`) {
		t.Fatalf("lifecycle log did not promote degraded evidence:\n%s", log.String())
	}
	if !strings.Contains(log.String(), `candidate="candidate:test"`) || !strings.Contains(log.String(), `selected_artifact="sha256:aaaaaaaa`) {
		t.Fatalf("lifecycle log omitted artifact identities:\n%s", log.String())
	}
}

func TestProvisionSourceOnlySharedPackageIsNoop(t *testing.T) {
	dependency := sharedPackageDependency{Name: "@vrooli/source-only", Root: t.TempDir()}
	if err := provisionSharedPackage(dependency, os.Stdout, os.Stdout); err != nil {
		t.Fatalf("source-only package should not require a generated artifact: %v", err)
	}
}

func TestProvisionSharedPackageSerializesCommandsAcrossConsumers(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	countFile := filepath.Join(root, "invocations")
	dependency := sharedPackageDependency{
		Name: "@vrooli/example",
		Root: root,
		Build: []packagegov.CommandSpec{{
			Name:    "build",
			Run:     []string{"/bin/sh", "-c", `printf x >> "$1"; sleep 0.15; mkdir -p dist; printf built > dist/index.js`, "sh", countFile},
			Outputs: []string{"dist/**"},
		}},
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var output bytes.Buffer
			errs <- provisionSharedPackageWithOptions(dependency, &output, &output, sharedPackageProvisionOptions{
				Home:  home,
				Stdin: strings.NewReader(""),
			})
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("provision shared package: %v", err)
		}
	}

	contents, err := os.ReadFile(countFile)
	if err != nil {
		t.Fatalf("read invocation count: %v", err)
	}
	if got := string(contents); got != "x" {
		t.Fatalf("build invocation marker = %q, want exactly one invocation", got)
	}
}

// TestSharedPackageOutputsFreshDetectsDeletedOutput pins self-healing. A
// generator publishes by mirroring a staging tree, so an interrupted or
// partial publish deletes outputs it should have kept. If unchanged sources
// alone meant "fresh", that gap would never be regenerated.
func TestSharedPackageOutputsFreshDetectsDeletedOutput(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	patterns := []string{"dist/**"}
	writeSharedPackageFixture(t, root, "export const value = 1;\n", "export const value = 1;\n")
	extra := filepath.Join(root, "dist", "extra.js")
	if err := os.WriteFile(extra, []byte("extra\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	digest, err := sharedPackageSourceDigest(root, patterns)
	if err != nil {
		t.Fatal(err)
	}
	stampPath, err := sharedPackageStampPath(home, root, "build")
	if err != nil {
		t.Fatal(err)
	}
	if err := writeSharedPackageStamp(stampPath, "@vrooli/example", "build", digest, outputsDigestFor(t, root, patterns)); err != nil {
		t.Fatal(err)
	}
	fresh, err := sharedPackageOutputsFresh(home, root, "build", patterns)
	if err != nil {
		t.Fatal(err)
	}
	if !fresh {
		t.Fatal("complete outputs with unchanged sources must be fresh")
	}

	// Lose one generated file, as a partial publish would.
	if err := os.Remove(extra); err != nil {
		t.Fatal(err)
	}
	fresh, err = sharedPackageOutputsFresh(home, root, "build", patterns)
	if err != nil {
		t.Fatal(err)
	}
	if fresh {
		t.Fatal("a deleted output must invalidate freshness so it can be regenerated")
	}
}
