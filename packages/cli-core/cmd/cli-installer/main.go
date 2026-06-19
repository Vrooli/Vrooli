package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

type stringListFlag []string

func (f *stringListFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *stringListFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	moduleRoot := flag.String("module", ".", "path to the Go module directory")
	manifestSource := flag.String("manifest", "", "path to the source manifest that should be installed beside the binary")
	output := flag.String("output", "", "explicit output path for the built binary")
	installDir := flag.String("install-dir", "", "directory where the binary should be installed")
	name := flag.String("name", "", "binary name (defaults to module directory name)")
	contextRoot := flag.String("context-root", "", "path to the freshness context root (defaults to module root)")
	force := flag.Bool("force", true, "overwrite existing binary when present")
	var freshnessInputs stringListFlag
	flag.Var(&freshnessInputs, "freshness-input", "declared freshness input relative to the context root (repeatable)")
	flag.Parse()

	if *output == "" && *installDir == "" {
		*installDir = defaultInstallDir()
	}

	modulePath, err := filepath.Abs(*moduleRoot)
	if err != nil {
		return fmt.Errorf("resolve module root: %w", err)
	}

	if _, err := os.Stat(filepath.Join(modulePath, "go.mod")); err != nil {
		return fmt.Errorf("module root must contain go.mod: %w", err)
	}
	manifestPath, err := resolveManifestPath(*manifestSource)
	if err != nil {
		return err
	}

	dst, err := determineDestination(modulePath, *output, *installDir, *name)
	if err != nil {
		return err
	}

	if !*force {
		if _, err := os.Stat(dst); err == nil {
			return fmt.Errorf("target already exists: %s (use --force to overwrite)", dst)
		}
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("prepare install directory: %w", err)
	}

	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("Go toolchain is required: %w", err)
	}

	binaryName := *name
	if binaryName == "" {
		binaryName = filepath.Base(modulePath)
	}
	spec, err := buildFreshnessSpec(modulePath, *contextRoot, []string(freshnessInputs), binaryName)
	if err != nil {
		return fmt.Errorf("build freshness spec: %w", err)
	}
	fingerprint, err := cliutil.ComputeFreshnessFingerprint(spec)
	if err != nil {
		return fmt.Errorf("compute fingerprint: %w", err)
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)
	sourceRoot := filepath.ToSlash(modulePath)
	// Bake the freshness-input list into cliapp.BakedFreshnessInputs so the
	// runtime stale-checker uses the same input set the install-time
	// fingerprint was computed from. Without this bake, a CLI whose
	// manifest declares custom freshness inputs (e.g. additional docs/
	// or README entries) would have the installer compute fingerprint
	// over N inputs while the runtime stale-checker computed over only
	// the package's hardcoded default M — guaranteeing a mismatch and
	// triggering a rebuild loop on every invocation.
	//
	// Comma separator is safe because no current resource/scenario
	// freshness input contains a comma — they're all path globs and
	// filenames. Avoiding whitespace keeps the value transparent through
	// the Go linker's flag parsing (which splits the -ldflags argument on
	// whitespace before consuming each -X token).
	bakedInputs := strings.Join(spec.Inputs, ",")
	flags := fmt.Sprintf(
		"-X main.buildFingerprint=%s -X main.buildTimestamp=%s -X main.buildSourceRoot=%s -X github.com/vrooli/cli-core/cliapp.BakedFreshnessInputs=%s",
		fingerprint,
		timestamp,
		escapeLdflagValue(sourceRoot),
		escapeLdflagValue(bakedInputs),
	)

	tmpFile, err := os.CreateTemp(filepath.Dir(dst), "cli-build-*")
	if err != nil {
		return fmt.Errorf("create temporary binary: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	cmd := exec.Command("go", "build", "-ldflags", flags, "-o", tmpPath, ".")
	cmd.Dir = modulePath
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Include modulePath in the error so the caller (and the
		// operator scanning a stack of CLI installs) can tell which
		// module failed without having to bisect or grep.
		return fmt.Errorf("go build failed in %s: %w", modulePath, err)
	}

	if err := replaceBinary(tmpPath, dst); err != nil {
		return fmt.Errorf("install binary: %w", err)
	}
	if manifestPath != "" {
		if err := installManifest(manifestPath, cliutil.InstalledManifestPath(dst)); err != nil {
			return fmt.Errorf("install manifest: %w", err)
		}
	}
	if err := writeInstallMetadata(dst, installMetadata{
		BinaryName:  binaryName,
		ModulePath:  sourceRoot,
		Fingerprint: fingerprint,
		InstalledAt: timestamp,
	}); err != nil {
		return fmt.Errorf("write install metadata: %w", err)
	}

	fmt.Printf("✅ installed CLI to %s\n", dst)
	ensurePathHint(dst)
	ensureShellRefreshHint()
	return nil
}

type installMetadata struct {
	BinaryName  string `json:"binary_name"`
	ModulePath  string `json:"module_path"`
	Fingerprint string `json:"fingerprint"`
	InstalledAt string `json:"installed_at,omitempty"`
}

func buildFreshnessSpec(modulePath, contextRoot string, inputs []string, binaryName string) (cliutil.FreshnessSpec, error) {
	modulePath = strings.TrimSpace(modulePath)
	if modulePath == "" {
		return cliutil.FreshnessSpec{}, errors.New("module path must not be empty")
	}
	absModulePath, err := filepath.Abs(filepath.Clean(modulePath))
	if err != nil {
		return cliutil.FreshnessSpec{}, fmt.Errorf("resolve module path: %w", err)
	}
	modulePath = absModulePath

	if strings.TrimSpace(contextRoot) == "" {
		contextRoot = modulePath
	} else {
		absContextRoot, err := filepath.Abs(filepath.Clean(strings.TrimSpace(contextRoot)))
		if err != nil {
			return cliutil.FreshnessSpec{}, fmt.Errorf("resolve context root: %w", err)
		}
		contextRoot = absContextRoot
	}

	return cliutil.FreshnessSpec{
		SourceRoot:  modulePath,
		ContextRoot: contextRoot,
		Inputs:      append([]string(nil), inputs...),
		SkipFiles:   []string{binaryName},
	}, nil
}

func determineDestination(modulePath, explicitOutput, installDir, name string) (string, error) {
	if explicitOutput != "" {
		return filepath.Abs(explicitOutput)
	}

	if installDir == "" {
		return "", errors.New("install directory is required when --output is not set")
	}

	dir := installDir
	if !filepath.IsAbs(dir) {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return "", fmt.Errorf("resolve install directory: %w", err)
		}
		dir = absDir
	}

	binaryName := name
	if binaryName == "" {
		binaryName = filepath.Base(modulePath)
	}
	if binaryName == "" {
		return "", errors.New("failed to infer binary name")
	}

	return filepath.Join(dir, binaryName), nil
}

func defaultInstallDir() string {
	if runtime.GOOS == "windows" {
		if dir := os.Getenv("USERPROFILE"); dir != "" {
			return filepath.Join(dir, "bin")
		}
		if dir, err := os.UserHomeDir(); err == nil && dir != "" {
			return filepath.Join(dir, "bin")
		}
		return "."
	}
	if dir := os.Getenv("HOME"); dir != "" {
		return filepath.Join(dir, ".local", "bin")
	}
	if dir, err := os.UserHomeDir(); err == nil && dir != "" {
		return filepath.Join(dir, ".local", "bin")
	}
	return "."
}

// replaceBinary atomically and durably replaces dst with the file at src.
//
// Why fsync explicitly rather than relying on rename(2) alone:
//
//	On Linux ext4 with the default data=ordered journal mode, rename(2) is
//	atomic for the directory entry but says NOTHING about when the source
//	file's data blocks reach disk. The metadata commit (which makes the
//	rename visible after a crash) can land in the journal before the data
//	blocks are flushed. If power is cut in that window, dst ends up with
//	valid metadata — correct size, exec bit, mtime — but ZERO-FILLED
//	contents. The kernel refuses to exec it ("Exec format error") even
//	though `ls` and `stat` look fine. We were burned by exactly this on
//	2026-05-07: an in-flight install left ~/.vrooli/bin/vrooli as 10 MB of
//	nulls, and the orphaned `cli-build-*` zero-byte files in that directory
//	show this had been silently happening to scenario installs too.
//
// Crash-safe sequence:
//  1. fsync(src) — force the freshly-built binary's data blocks to stable
//     storage BEFORE we hand it dst's name. Without this step, step 2 is
//     racing the writeback path.
//  2. rename(src, dst) — atomic within a single filesystem. dst points at
//     either the old inode (if we crash before this commits) or the fully
//     fsynced new inode, never a partial state.
//  3. fsync(parent_dir) — persist the directory-entry change itself, so a
//     crash after rename(2) can't roll the entry back to the previous
//     inode (or, on a brand-new dst, leave it dangling).
//
// Together these guarantee that after any crash, dst either still exists
// as the previous-good binary or is the fully-written new binary — never
// the zero-filled stub that broke us today.
func replaceBinary(src, dst string) error {
	if err := fsyncPath(src); err != nil {
		return fmt.Errorf("fsync new binary before rename: %w", err)
	}
	renameErr := os.Rename(src, dst)
	if renameErr != nil && runtime.GOOS == "windows" {
		// Windows rename(2) refuses to overwrite an existing file; fall
		// back to remove-then-rename. This is non-atomic on Windows but
		// matches the prior behavior — the Linux path (our production
		// target) gets the strong guarantees above.
		_ = os.Remove(dst)
		renameErr = os.Rename(src, dst)
	}
	if renameErr != nil {
		return fmt.Errorf("replace binary: %w", renameErr)
	}
	if err := fsyncPath(filepath.Dir(dst)); err != nil {
		return fmt.Errorf("fsync install directory after rename: %w", err)
	}
	return nil
}

// fsyncPath opens path and calls fsync(2). Works for both regular files
// and directories on Linux/macOS — fsync of a directory FD is the standard
// way to make a rename(2) durable. On Windows, fsync of a directory handle
// is not a defined operation; NTFS already orders rename metadata durably
// enough for our needs, so skip it there.
func fsyncPath(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func resolveManifestPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve manifest path: %w", err)
	}
	if info, err := os.Stat(absPath); err != nil {
		return "", fmt.Errorf("manifest source must exist: %w", err)
	} else if info.IsDir() {
		return "", fmt.Errorf("manifest source must be a file: %s", absPath)
	}
	return absPath, nil
}

func installManifest(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("prepare manifest directory: %w", err)
	}
	tmpFile, err := os.CreateTemp(filepath.Dir(dst), "cli-manifest-*")
	if err != nil {
		return fmt.Errorf("create temporary manifest: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if err := copyFileContents(src, tmpFile); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temporary manifest: %w", err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(tmpPath, 0o644); err != nil {
			return fmt.Errorf("chmod temporary manifest: %w", err)
		}
	}
	if err := replaceBinary(tmpPath, dst); err != nil {
		return fmt.Errorf("replace manifest: %w", err)
	}
	return nil
}

func copyFileContents(src string, dst *os.File) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open manifest source: %w", err)
	}
	defer in.Close()
	if _, err := io.Copy(dst, in); err != nil {
		return fmt.Errorf("copy manifest source: %w", err)
	}
	return nil
}

func writeInstallMetadata(binaryPath string, meta installMetadata) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(cliutil.InstalledBuildMetadataPath(binaryPath), data, 0o644)
}

func ensurePathHint(binaryPath string) {
	installDir := filepath.Dir(binaryPath)
	pathEnv := os.Getenv("PATH")
	if strings.Contains(pathEnv, installDir) {
		return
	}

	if runtime.GOOS == "windows" {
		fmt.Printf("⚠️  Add to PATH (PowerShell): $Env:Path = \"%s;$Env:Path\"\n", installDir)
		return
	}

	fmt.Printf("⚠️  Add to PATH: export PATH=\"%s:$PATH\"\n", installDir)
}

func ensureShellRefreshHint() {
	if runtime.GOOS == "windows" {
		fmt.Println("ℹ️  If you are replacing an existing CLI command, open a new shell so command lookup refreshes.")
		return
	}
	fmt.Println("ℹ️  If you are replacing an existing CLI command in this shell, run: hash -r")
}

func escapeLdflagValue(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, " ", `\ `)
	return value
}
