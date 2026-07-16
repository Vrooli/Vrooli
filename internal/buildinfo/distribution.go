package buildinfo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/shell"
)

// DistributionTarget is one supported prebuilt Vrooli CLI platform. This list
// is the single source of truth consumed by the release workflow and bridge.
type DistributionTarget struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

var distributionTargets = []DistributionTarget{
	{OS: "linux", Arch: "amd64"},
	{OS: "linux", Arch: "arm64"},
	{OS: "darwin", Arch: "amd64"},
	{OS: "darwin", Arch: "arm64"},
	{OS: "windows", Arch: "amd64"},
	{OS: "windows", Arch: "arm64"},
}

// DistributionTargets returns a copy of the supported release matrix.
func DistributionTargets() []DistributionTarget {
	return append([]DistributionTarget(nil), distributionTargets...)
}

// DistributionAssetName returns the canonical release asset name.
func DistributionAssetName(target DistributionTarget) string {
	suffix := ""
	if target.OS == "windows" {
		suffix = ".exe"
	}
	return fmt.Sprintf("vrooli_%s_%s%s", target.OS, target.Arch, suffix)
}

// DistributionBuildOptions describes one cross-compiled CLI artifact.
type DistributionBuildOptions struct {
	Root      string
	Output    string
	Target    DistributionTarget
	Version   string
	GitCommit string
	BuildTime time.Time
	Stdout    *os.File
	Stderr    *os.File
}

// DistributionArtifact is the binary and freshness metadata emitted by a build.
type DistributionArtifact struct {
	BinaryPath  string
	SidecarPath string
	Fingerprint string
	Target      DistributionTarget
}

var distributionRun = shell.Run

// BuildDistribution cross-compiles the project-level vrooli CLI with CGO
// disabled and writes the exact freshness fingerprint consumed by CheckStaleness.
func BuildDistribution(ctx context.Context, options DistributionBuildOptions) (DistributionArtifact, error) {
	if !isDistributionTarget(options.Target) {
		return DistributionArtifact{}, fmt.Errorf("unsupported distribution target %s/%s", options.Target.OS, options.Target.Arch)
	}
	root := strings.TrimSpace(options.Root)
	if root == "" {
		var err error
		root, err = ResolveSourceRoot()
		if err != nil {
			return DistributionArtifact{}, err
		}
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return DistributionArtifact{}, fmt.Errorf("resolve distribution root: %w", err)
	}
	output := strings.TrimSpace(options.Output)
	if output == "" {
		return DistributionArtifact{}, errors.New("distribution output path is required")
	}
	if !filepath.IsAbs(output) {
		output = filepath.Join(root, output)
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return DistributionArtifact{}, fmt.Errorf("create distribution directory: %w", err)
	}

	report, err := ComputeSourceFingerprintReport(root, FingerprintOptions{
		RequireExistingTargets: true,
		RequireGoFiles:         true,
	}, FingerprintTargetsForExecutable("vrooli")...)
	if err != nil {
		return DistributionArtifact{}, fmt.Errorf("compute vrooli fingerprint: %w", err)
	}

	version := strings.TrimPrefix(strings.TrimSpace(options.Version), "v")
	if version == "" {
		version = "dev"
	}
	gitCommit := strings.TrimSpace(options.GitCommit)
	if gitCommit == "" {
		gitCommit = resolveDistributionGitCommit(root)
	}
	buildTime := options.BuildTime.UTC()
	if buildTime.IsZero() {
		buildTime = time.Now().UTC()
	}
	ldflags := fmt.Sprintf(
		"-s -w -X github.com/vrooli/vrooli/internal/buildinfo.Fingerprint=%s -X github.com/vrooli/vrooli/internal/buildinfo.GitCommit=%s -X github.com/vrooli/vrooli/internal/buildinfo.BuildTime=%s -X main.vrooliVersion=%s",
		report.Fingerprint, gitCommit, buildTime.Format(time.RFC3339), version,
	)
	env := append([]string(nil), os.Environ()...)
	env = setEnvValue(env, "CGO_ENABLED", "0")
	env = setEnvValue(env, "GOOS", options.Target.OS)
	env = setEnvValue(env, "GOARCH", options.Target.Arch)
	buildArgs := []string{"build", "-trimpath"}
	overlayArgs, cleanupOverlay, err := distributionOverlay(root, options.Target)
	if err != nil {
		return DistributionArtifact{}, err
	}
	defer cleanupOverlay()
	buildArgs = append(buildArgs, overlayArgs...)
	buildArgs = append(buildArgs, "-ldflags", ldflags, "-o", output, "./cmd/vrooli")
	if err := distributionRun(shell.Spec{
		Context: ctx,
		Name:    "go",
		Args:    buildArgs,
		Dir:     root,
		Env:     env,
		Stdout:  options.Stdout,
		Stderr:  options.Stderr,
	}); err != nil {
		_ = os.Remove(output)
		return DistributionArtifact{}, fmt.Errorf("cross-compile vrooli for %s/%s: %w", options.Target.OS, options.Target.Arch, err)
	}
	if err := WriteSidecarFingerprint(output, report.Fingerprint); err != nil {
		_ = os.Remove(output)
		_ = os.Remove(output + ".fp")
		return DistributionArtifact{}, fmt.Errorf("write distribution fingerprint: %w", err)
	}
	return DistributionArtifact{
		BinaryPath:  output,
		SidecarPath: output + ".fp",
		Fingerprint: report.Fingerprint,
		Target:      options.Target,
	}, nil
}

// distributionOverlay substitutes a Windows-only unsupported handler for the
// Linux pstore collector. Native Windows lifecycle support is intentionally out
// of scope, but the release artifact must still compile and report this
// safeguard as unsupported. The replacement lives beneath internal/buildinfo,
// so the normal source fingerprint covers it along with every other build input.
func distributionOverlay(root string, target DistributionTarget) ([]string, func(), error) {
	if target.OS != "windows" {
		return nil, func() {}, nil
	}
	replaceFrom := filepath.Join(root, "internal", "safeguards", "pstore-observability", "handler.go")
	replaceWith := filepath.Join(root, "internal", "buildinfo", "testdata", "pstore_observability_windows.go")
	for _, path := range []string{replaceFrom, replaceWith} {
		if _, err := os.Stat(path); err != nil {
			return nil, func() {}, fmt.Errorf("prepare Windows distribution overlay: %w", err)
		}
	}
	overlay, err := os.CreateTemp("", "vrooli-windows-overlay-*.json")
	if err != nil {
		return nil, func() {}, fmt.Errorf("create Windows distribution overlay: %w", err)
	}
	cleanup := func() { _ = os.Remove(overlay.Name()) }
	payload := struct {
		Replace map[string]string `json:"Replace"`
	}{Replace: map[string]string{replaceFrom: replaceWith}}
	if err := json.NewEncoder(overlay).Encode(payload); err != nil {
		_ = overlay.Close()
		cleanup()
		return nil, func() {}, fmt.Errorf("write Windows distribution overlay: %w", err)
	}
	if err := overlay.Close(); err != nil {
		cleanup()
		return nil, func() {}, fmt.Errorf("close Windows distribution overlay: %w", err)
	}
	return []string{"-overlay", overlay.Name()}, cleanup, nil
}

func isDistributionTarget(target DistributionTarget) bool {
	for _, supported := range distributionTargets {
		if target == supported {
			return true
		}
	}
	return false
}

func resolveDistributionGitCommit(root string) string {
	out, err := shell.Output(shell.Spec{Name: "git", Args: []string{"rev-parse", "HEAD"}, Dir: root})
	if err != nil || strings.TrimSpace(string(out)) == "" {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

// HostDistributionTarget returns the current runtime target when supported.
func HostDistributionTarget() (DistributionTarget, bool) {
	target := DistributionTarget{OS: runtime.GOOS, Arch: runtime.GOARCH}
	return target, isDistributionTarget(target)
}
