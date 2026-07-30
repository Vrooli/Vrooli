package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/buildinfo"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

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
	matrixJSON := flags.Bool("matrix-json", false, "print the supported target matrix as JSON")
	vaultServer := flags.Bool("vault-server", false, "stage a checksum-verified Vault server from the checked-in catalog")
	vaultTarget := flags.String("vault-target", "", "managed Vault target (for example linux-amd64)")
	resourceArtifacts := flags.Bool("resource-artifacts", false, "build and stage release-signable resource controller and service artifacts")
	verifyReleaseManifest := flags.Bool("verify-release-manifest", false, "verify a staged generic release manifest before it is bundled")
	trustMode := flags.String("trust-mode", "", "artifact trust mode for manifest verification: development-local or production")
	releaseArtifactRoot := flags.String("release-artifact-root", "", "staged release artifact directory to verify")
	releasePublicKey := flags.String("release-public-key", "", "PEM public key used to verify a production release manifest")
	if err := flags.Parse(args); err != nil {
		return 2
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
			return 2
		}
		mode := resourcedeployment.ArtifactTrustMode(*trustMode)
		publicKeyPath := *releasePublicKey
		if mode == resourcedeployment.ArtifactTrustProduction && strings.TrimSpace(publicKeyPath) == "" {
			if strings.TrimSpace(*root) == "" {
				fmt.Fprintln(os.Stderr, "production verification requires --release-public-key or --root (for install/vrooli-release.pub)")
				return 2
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
	if *vaultServer {
		if strings.TrimSpace(*root) == "" || strings.TrimSpace(*outDir) == "" || strings.TrimSpace(*vaultTarget) == "" {
			fmt.Fprintln(os.Stderr, "--vault-server requires --root, --out-dir, and --vault-target")
			return 2
		}
		if err := stageVaultServer(context.Background(), *root, *outDir, *vaultTarget); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		stagedReleaseArtifacts = true
	}
	if *resourceArtifacts {
		if strings.TrimSpace(*root) == "" || strings.TrimSpace(*outDir) == "" {
			fmt.Fprintln(os.Stderr, "--resource-artifacts requires --root and --out-dir")
			return 2
		}
		if err := stageResourceArtifacts(context.Background(), *root, *outDir); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if err := writeReleaseChecksumManifest(*outDir); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		stagedReleaseArtifacts = true
	}
	if stagedReleaseArtifacts {
		return 0
	}
	if *all {
		for _, target := range buildinfo.DistributionTargets() {
			path := filepath.Join(*outDir, buildinfo.DistributionAssetName(target))
			if code := buildOne(*root, path, *version, target); code != 0 {
				return code
			}
		}
		return 0
	}
	if strings.TrimSpace(*goos) == "" || strings.TrimSpace(*goarch) == "" || strings.TrimSpace(*output) == "" {
		fmt.Fprintln(os.Stderr, "--goos, --goarch, and --output are required (or use --all/--matrix-json)")
		return 2
	}
	return buildOne(*root, *output, *version, buildinfo.DistributionTarget{OS: *goos, Arch: *goarch})
}

func buildOne(root, output, version string, target buildinfo.DistributionTarget) int {
	artifact, err := buildinfo.BuildDistribution(context.Background(), buildinfo.DistributionBuildOptions{
		Root: root, Output: output, Version: version, Target: target,
		Stdout: os.Stdout, Stderr: os.Stderr,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("%s/%s: %s (fingerprint %s)\n", target.OS, target.Arch, artifact.BinaryPath, artifact.Fingerprint)
	return 0
}
