package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/buildinfo"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

const (
	invalidInvocationExitCode = 2
)

func main() {
	os.Exit(run(os.Args[1:]))
}

//nolint:gocyclo // distribution CLI sequencing has distinct validation, build, and publication failures.
func run(args []string) int {
	flags := flag.NewFlagSet("vrooli-dist", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	goos := flags.String("goos", "", "target operating system")
	goarch := flags.String("goarch", "", "target architecture")
	output := flags.String("output", "", "output binary path")
	outDir := flags.String("out-dir", "dist", "output directory used with --all")
	root := flags.String("root", "", "repository root")
	version := flags.String("version", "", "release version (normally the git tag)")
	all := flags.Bool("all", false, "build every supported target")
	targets := flags.String("targets", "", "comma-separated os/arch subset to build (for example darwin/amd64,darwin/arm64)")
	skipExisting := flags.Bool("skip-existing", false, "skip targets whose output binary is already staged, so a release can adopt binaries built on another runner")
	allowMissingDarwinKeychain := flags.Bool("allow-missing-darwin-keychain", false,
		"permit building darwin from a non-darwin host, producing a CLI with no macOS credential backend; never use for a release")
	matrixJSON := flags.Bool("matrix-json", false, "print the supported target matrix as JSON")
	resourceArtifacts := flags.Bool("resource-artifacts", false, "build and stage release-signable resource controller and service artifacts")
	resourceSelection := flags.String("resource", "", "comma-separated resource names to stage (defaults to every supported resource)")
	platformSelection := flags.String("platform", "", "comma-separated resource platforms to stage (for example linux-amd64)")
	toolArtifacts := flags.Bool("tool-artifacts", false, "stage release-signable vendored tool artifacts")
	rendererProfiles := flags.String("renderer-profiles", "", "comma-separated managed renderer packages for a development-local desktop release (blender, ffmpeg-video)")
	verifyReleaseManifest := flags.Bool("verify-release-manifest", false, "verify a staged generic release manifest before it is bundled")
	trustMode := flags.String("trust-mode", "", "artifact trust mode for manifest verification: development-local or production")
	releaseArtifactRoot := flags.String("release-artifact-root", "", "staged release artifact directory to verify")
	releasePublicKey := flags.String("release-public-key", "", "PEM public key used to verify a production release manifest")
	if err := flags.Parse(args); err != nil {
		return invalidInvocationExitCode
	}
	if *matrixJSON {
		payload, err := json.Marshal(buildinfo.DistributionTargets())
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println(string(payload))
		return 0
	}
	if *verifyReleaseManifest {
		if strings.TrimSpace(*releaseArtifactRoot) == "" || strings.TrimSpace(*trustMode) == "" {
			fmt.Fprintln(os.Stderr, "--verify-release-manifest requires --release-artifact-root and --trust-mode")
			return invalidInvocationExitCode
		}
		mode := resourcedeployment.ArtifactTrustMode(*trustMode)
		publicKeyPath := *releasePublicKey
		if mode == resourcedeployment.ArtifactTrustProduction && strings.TrimSpace(publicKeyPath) == "" {
			if strings.TrimSpace(*root) == "" {
				fmt.Fprintln(os.Stderr, "production verification requires --release-public-key or --root (for install/vrooli-release.pub)")
				return invalidInvocationExitCode
			}
			publicKeyPath = filepath.Join(*root, "install", "vrooli-release.pub")
		}
		manifest, signature, err := resourcedeployment.VerifyReleaseDirectory(*releaseArtifactRoot, mode, publicKeyPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if signature == nil {
			fmt.Printf("verified %d artifacts in development-local mode (non-promotable)\n", len(manifest.Artifacts))
		} else {
			fmt.Printf("verified %d artifacts in production mode (key %s)\n", len(manifest.Artifacts), signature.KeyID)
		}
		return 0
	}
	stagedReleaseArtifacts := false
	if *resourceArtifacts || *toolArtifacts || strings.TrimSpace(*rendererProfiles) != "" {
		if strings.TrimSpace(*root) == "" || strings.TrimSpace(*outDir) == "" {
			fmt.Fprintln(os.Stderr, "artifact staging requires --root and --out-dir")
			return invalidInvocationExitCode
		}
		var stageErrors []error
		if *resourceArtifacts {
			resources, err := splitArtifactSelection(*resourceSelection)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return invalidInvocationExitCode
			}
			platforms, err := splitArtifactSelection(*platformSelection)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return invalidInvocationExitCode
			}
			if err := stageResourceArtifactsSelected(context.Background(), *root, *outDir, resources, platforms); err != nil {
				stageErrors = append(stageErrors, err)
			}
		}
		if *toolArtifacts {
			if err := stageToolArtifacts(context.Background(), *root, *outDir); err != nil {
				stageErrors = append(stageErrors, err)
			}
		}
		if strings.TrimSpace(*rendererProfiles) != "" {
			selected, err := splitArtifactSelection(*rendererProfiles)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return invalidInvocationExitCode
			}
			if err := stageManagedRendererProfiles(context.Background(), selected, *outDir); err != nil {
				stageErrors = append(stageErrors, err)
			}
		}
		if err := finalizeReleaseArtifactStaging(*outDir, stageErrors); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		stagedReleaseArtifacts = true
	}
	if stagedReleaseArtifacts {
		return 0
	}
	if *all || strings.TrimSpace(*targets) != "" {
		selected, err := selectTargets(*targets)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return invalidInvocationExitCode
		}
		for _, target := range selected {
			path := filepath.Join(*outDir, buildinfo.DistributionAssetName(target))
			if *skipExisting {
				if _, err := os.Stat(path); err == nil {
					fmt.Printf("%s/%s: adopting already-staged %s\n", target.OS, target.Arch, path)
					continue
				}
			}
			if code := buildOne(*root, path, *version, target, *allowMissingDarwinKeychain); code != 0 {
				return code
			}
		}
		return 0
	}
	if strings.TrimSpace(*goos) == "" || strings.TrimSpace(*goarch) == "" || strings.TrimSpace(*output) == "" {
		fmt.Fprintln(os.Stderr, "--goos, --goarch, and --output are required (or use --all/--targets/--matrix-json)")
		return invalidInvocationExitCode
	}
	return buildOne(*root, *output, *version, buildinfo.DistributionTarget{OS: *goos, Arch: *goarch}, *allowMissingDarwinKeychain)
}

func finalizeReleaseArtifactStaging(outDir string, stageErrors []error) error {
	if len(stageErrors) > 0 {
		return errors.Join(stageErrors...)
	}
	return writeReleaseChecksumManifest(outDir)
}

func splitArtifactSelection(value string) ([]string, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parts := strings.Split(value, ",")
	seen := make(map[string]bool, len(parts))
	selection := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			return nil, fmt.Errorf("artifact selection contains an empty value")
		}
		if seen[item] {
			return nil, fmt.Errorf("artifact selection contains duplicate %q", item)
		}
		seen[item] = true
		selection = append(selection, item)
	}
	return selection, nil
}

// selectTargets resolves an explicit os/arch subset, or the full matrix when the
// subset is empty. Splitting the matrix is what lets darwin build on a macOS
// runner while the rest cross-compile anywhere.
func selectTargets(spec string) ([]buildinfo.DistributionTarget, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return buildinfo.DistributionTargets(), nil
	}
	supported := buildinfo.DistributionTargets()
	var selected []buildinfo.DistributionTarget
	for _, raw := range strings.Split(spec, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		goos, goarch, ok := strings.Cut(raw, "/")
		if !ok {
			return nil, fmt.Errorf("target %q must use os/arch form", raw)
		}
		target := buildinfo.DistributionTarget{OS: strings.TrimSpace(goos), Arch: strings.TrimSpace(goarch)}
		found := false
		for _, candidate := range supported {
			if candidate == target {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("target %q is not in the supported release matrix", raw)
		}
		selected = append(selected, target)
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("--targets requires at least one os/arch")
	}
	return selected, nil
}

func buildOne(root, output, version string, target buildinfo.DistributionTarget, allowMissingDarwinKeychain bool) int {
	artifact, err := buildinfo.BuildDistribution(context.Background(), buildinfo.DistributionBuildOptions{
		Root: root, Output: output, Version: version, Target: target,
		Stdout: os.Stdout, Stderr: os.Stderr,
		AllowMissingDarwinKeychain: allowMissingDarwinKeychain,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("%s/%s: %s (fingerprint %s)\n", target.OS, target.Arch, artifact.BinaryPath, artifact.Fingerprint)
	return 0
}
