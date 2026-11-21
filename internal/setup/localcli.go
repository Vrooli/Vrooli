package setup

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/buildflags"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/shell"
)

// BuildLocalCLI builds ./cmd/vrooli from root's current compiler inputs and
// publishes it atomically at outputPath.
//
// It is a development build, not an activation: the managed image in the
// runtime home is owned by root activation and is the rollback target, so it is
// never written here. The published binary carries the same build identity,
// runtime-home pin and persistent-state descriptor as a governed build, because
// it is compiled by the same function.
func BuildLocalCLI(ctx context.Context, root, outputPath string) error {
	buildDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return fmt.Errorf("prepare the local CLI directory: %w", err)
	}
	targets := buildinfo.FingerprintTargetsForExecutable("vrooli")
	// The snapshot must carry every module replacement the build environment
	// resolves, not only the files that participate in this binary's
	// fingerprint: `go env` fails if a replaced module directory is absent.
	return buildFromSnapshot(ctx, root, buildCohortSourcePaths, func(buildRoot string) error {
		stageDir, err := os.MkdirTemp(buildDir, ".local-cli-*")
		if err != nil {
			return fmt.Errorf("create the local CLI stage: %w", err)
		}
		defer os.RemoveAll(stageDir)

		buildEnv, err := freezeGoBuildEnvironment(buildRoot, stageDir)
		if err != nil {
			return fmt.Errorf("capture Go build environment: %w", err)
		}
		goFlags := []string{"-trimpath"}
		if policy, policyErr := buildflags.Load(buildRoot); policyErr == nil && len(policy.For("develop")) > 0 {
			goFlags = policy.For("develop")
		}
		if err := rejectUnsupportedGoFlags(goFlags); err != nil {
			return err
		}
		// The snapshot is a temporary directory, so Go's VCS discovery would
		// otherwise walk out of it and stamp an unrelated repository.
		if !slices.Contains(goFlags, "-buildvcs=false") {
			goFlags = append(goFlags, "-buildvcs=false")
		}

		gitCommit := "unknown"
		if output, cmdErr := shell.Output(shell.Spec{Dir: root, Name: "git", Args: []string{"rev-parse", "HEAD"}}); cmdErr == nil {
			if value := strings.TrimSpace(string(output)); value != "" {
				gitCommit = value
			}
		}
		stagedPath := filepath.Join(stageDir, "vrooli")
		// Build output is discarded: a refresh is routine, and its noise reads
		// like destructive work to an agent's command guard.
		if err := buildProjectBinary(buildRoot, stagedPath, "./cmd/vrooli", targets, gitCommit, time.Now().UTC().Format(time.RFC3339), buildEnv.root, buildEnv.goCommand, goFlags, io.Discard, io.Discard); err != nil {
			return err
		}
		if err := buildEnv.verifyNoVendor(); err != nil {
			return err
		}
		if err := platform.AtomicReplace(stagedPath, outputPath); err != nil {
			return fmt.Errorf("publish the local CLI: %w", err)
		}
		fingerprint, err := buildinfo.ComputeSourceFingerprintForPaths(buildRoot, targets...)
		if err != nil {
			return err
		}
		return buildinfo.WriteSidecarFingerprint(outputPath, fingerprint)
	})
}
