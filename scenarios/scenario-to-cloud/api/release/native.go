package release

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vrooli/envkit-go"

	"scenario-to-cloud/domain"
)

// Platform is the target the native control plane is built for.
type Platform struct {
	GOOS   string `json:"goos"`
	GOARCH string `json:"goarch"`
}

// String renders goos/goarch.
func (p Platform) String() string { return p.GOOS + "/" + p.GOARCH }

// SupportedPlatforms are the control-plane platforms a release may target.
// They mirror the closure's controlPlanePlatforms declaration.
var SupportedPlatforms = []Platform{{GOOS: "linux", GOARCH: "amd64"}, {GOOS: "linux", GOARCH: "arm64"}}

// Supported reports whether a release may be built for p.
func (p Platform) Supported() bool {
	for _, candidate := range SupportedPlatforms {
		if candidate == p {
			return true
		}
	}
	return false
}

// NativeCLIOptions pin the control-plane build.
type NativeCLIOptions struct {
	// ModuleDir is the module the binary is built from; defaults to the repo
	// root.
	ModuleDir string
	// Package is the main package; defaults to ./cmd/vrooli.
	Package string
	// GoBinary is the toolchain executable; defaults to "go" on PATH.
	GoBinary string
	// VerifyReproducible builds twice and compares bytes. Without it the
	// release records the native binary as "unverified" rather than claiming
	// determinism it did not check.
	VerifyReproducible bool
}

// DefaultNativeCLIPackage is the control-plane main package.
const DefaultNativeCLIPackage = "./cmd/vrooli"

// nativeBuildArgs are the reproducibility-pinned build arguments. -trimpath
// removes host paths, -buildvcs=false keeps VCS state out of the binary (it is
// recorded in inputs.json instead), and an empty buildid removes the
// content-derived id that would otherwise differ across identical inputs on
// some toolchains.
var nativeBuildArgs = []string{"build", "-trimpath", "-buildvcs=false", "-ldflags=-buildid=", "-o"}

func (o NativeCLIOptions) withDefaults(repoRoot string) NativeCLIOptions {
	if strings.TrimSpace(o.ModuleDir) == "" {
		o.ModuleDir = repoRoot
	}
	if strings.TrimSpace(o.Package) == "" {
		o.Package = DefaultNativeCLIPackage
	}
	if strings.TrimSpace(o.GoBinary) == "" {
		o.GoBinary = "go"
	}
	return o
}

// pinnedEnv is the exact variable set the build pins. Everything else is
// inherited and therefore recorded as a limitation, not as an input.
func pinnedEnv(platform Platform) []string {
	return []string{
		"CGO_ENABLED=0",
		"GOOS=" + platform.GOOS,
		"GOARCH=" + platform.GOARCH,
		"GOWORK=off",
		"GOTOOLCHAIN=local",
	}
}

// buildEnv composes the inherited environment with the pinned variables.
func buildEnv(platform Platform) envkit.Env {
	return envkit.Toolchain(envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, envkit.Env(pinnedEnv(platform))), envkit.ToolchainOptions{})
}

func envValue(env []string, key string) string {
	for _, entry := range env {
		if name, value, ok := strings.Cut(entry, "="); ok && name == key {
			return value
		}
	}
	return ""
}

func nativeBuildCommand(ctx context.Context, opts NativeCLIOptions, platform Platform, output string) *exec.Cmd {
	args := append(append([]string(nil), nativeBuildArgs...), output, opts.Package)
	cmd := exec.CommandContext(ctx, opts.GoBinary, args...)
	cmd.Dir = opts.ModuleDir
	cmd.Env = buildEnv(platform)
	return cmd
}

func goVersion(ctx context.Context, opts NativeCLIOptions) (string, error) {
	cmd := exec.CommandContext(ctx, opts.GoBinary, "version")
	cmd.Dir = opts.ModuleDir
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("go version: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// buildNativeCLI compiles the control plane for platform into output and
// returns its record. The second build (when requested) goes to a sibling
// temp file and is deleted.
func buildNativeCLI(ctx context.Context, repoRoot string, opts NativeCLIOptions, platform Platform, output string) (domain.ReleaseNativeCLI, domain.ReleaseReproducibility, error) {
	opts = opts.withDefaults(repoRoot)
	version, err := goVersion(ctx, opts)
	if err != nil {
		return domain.ReleaseNativeCLI{}, domain.ReleaseReproducibility{}, err
	}
	cmd := nativeBuildCommand(ctx, opts, platform, output)
	if out, err := cmd.CombinedOutput(); err != nil {
		return domain.ReleaseNativeCLI{}, domain.ReleaseReproducibility{}, fmt.Errorf("build native vrooli binary for %s: %w\n%s", platform, err, strings.TrimSpace(string(out)))
	}
	sum, size, err := fileSHA256(output)
	if err != nil {
		return domain.ReleaseNativeCLI{}, domain.ReleaseReproducibility{}, err
	}
	moduleDir := opts.ModuleDir
	if rel, err := filepath.Rel(repoRoot, opts.ModuleDir); err == nil && !strings.HasPrefix(rel, "..") {
		moduleDir = filepath.ToSlash(rel)
	}
	record := domain.ReleaseNativeCLI{
		FileName:  filepath.Base(output),
		SHA256:    sum,
		GOOS:      platform.GOOS,
		GOARCH:    platform.GOARCH,
		SizeBytes: size,
		Package:   opts.Package,
		ModuleDir: moduleDir,
		GoVersion: version,
		Args:      append(append([]string(nil), nativeBuildArgs...), "<output>", opts.Package),
		Env:       pinnedEnv(platform),
	}
	repro := domain.ReleaseReproducibility{Bundle: domain.ReproducibilityDeterministic, NativeCLI: domain.ReproducibilityUnverified, Reason: "native binary built once; reproducibility not exercised for this build"}
	if !opts.VerifyReproducible {
		return record, repro, nil
	}
	second := output + ".rebuild"
	defer os.Remove(second)
	if out, err := nativeBuildCommand(ctx, opts, platform, second).CombinedOutput(); err != nil {
		return record, repro, fmt.Errorf("rebuild native vrooli binary for %s: %w\n%s", platform, err, strings.TrimSpace(string(out)))
	}
	secondSum, _, err := fileSHA256(second)
	if err != nil {
		return record, repro, err
	}
	if secondSum == sum {
		repro.NativeCLI = domain.ReproducibilityReproducible
		repro.Reason = ""
		return record, repro, nil
	}
	repro.NativeCLI = domain.ReproducibilityNotReproducible
	repro.Reason = fmt.Sprintf("two builds with identical pinned inputs (%s, %s) produced different bytes (%s vs %s); the release binds the first build's sha256", version, runtime.GOOS+"/"+runtime.GOARCH, sum[:12], secondSum[:12])
	return record, repro, nil
}

func fileSHA256(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	size, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), size, nil
}
