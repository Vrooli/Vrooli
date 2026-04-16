package install

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/resources/sqlite/cli/internal/discovery"
)

// Installer owns build/install and test execution concerns for the sqlite binary.
type Installer struct {
	Rebuild func() error
}

// New wires the default installer behavior for the discovered runtime.
func New(runtime discovery.Runtime) Installer {
	i := Installer{}
	i.Rebuild = func() error { return i.rebuildBinary(runtime) }
	return i
}

func (i Installer) rebuildBinary(runtime discovery.Runtime) error {
	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("Go toolchain not found; skipping binary rebuild")
	}

	srcRoot := runtime.ResolveSourceRoot()
	if srcRoot == "" {
		return fmt.Errorf("unable to locate SQLite source root; set SQLITE_CLI_SOURCE_ROOT or VROOLI_CLI_SOURCE_ROOT")
	}

	installDir := os.Getenv("VROOLI_BIN")
	if strings.TrimSpace(installDir) == "" {
		home, _ := os.UserHomeDir()
		if home == "" {
			home = "."
		}
		installDir = filepath.Join(home, ".vrooli", "bin")
	}
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return fmt.Errorf("prepare install dir: %w", err)
	}
	target := filepath.Join(installDir, "resource-sqlite")

	repoRoot, err := repocontract.FindRepoRootFromPath(srcRoot)
	if err != nil {
		return fmt.Errorf("unable to locate repository root from %s: %w", srcRoot, err)
	}

	cmd := exec.Command("go", "run", "./cmd/cli-installer",
		"--module", filepath.Join(srcRoot, "cli"),
		"--manifest", filepath.Join(srcRoot, "resource.json"),
		"--output", target,
		"--name", "resource-sqlite",
		"--force", "true",
	)
	cmd.Dir = filepath.Join(repoRoot, "packages", "cli-core")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build resource-sqlite binary via cli-core: %w", err)
	}

	return nil
}

// RunGoTests executes the sqlite Go test suite from the module root.
func (Installer) RunGoTests(ctx context.Context, root string) error {
	if _, err := exec.LookPath("go"); err != nil {
		fmt.Fprintln(os.Stderr, "Go toolchain not found. Install Go or run `go test ./...` inside resources/sqlite.")
		return err
	}
	cmd := exec.CommandContext(ctx, "go", "test", "./...")
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
