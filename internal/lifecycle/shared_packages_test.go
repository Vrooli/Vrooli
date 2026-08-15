package lifecycle

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/packagegov"
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

func TestSharedPackageOutputsFreshAndDigestChanges(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "src", "index.ts")
	output := filepath.Join(root, "dist", "index.js")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("export const value = 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, []byte("export const value = 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := os.Chtimes(source, now.Add(-time.Minute), now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(output, now, now); err != nil {
		t.Fatal(err)
	}
	fresh, err := sharedPackageOutputsFresh(root, []string{"dist/**"})
	if err != nil {
		t.Fatal(err)
	}
	if !fresh {
		t.Fatal("matching output should be fresh")
	}
	first, err := sharedPackageOutputDigest(root, []string{"dist/**"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, []byte("export const value = 2;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := sharedPackageOutputDigest(root, []string{"dist/**"})
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("output digest did not change after output content changed")
	}
	if err := os.Chtimes(source, now.Add(time.Minute), now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	fresh, err = sharedPackageOutputsFresh(root, []string{"dist/**"})
	if err != nil {
		t.Fatal(err)
	}
	if fresh {
		t.Fatal("output older than source should be stale")
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
