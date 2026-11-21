package cliinstall

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/coreactivation"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

const rootCLIImageLimit = 512 << 20

var rootCLIBuildIdentity = regexp.MustCompile(`^[a-f0-9]{64}$`)

// This prepares executable bytes only. Native activation, retained transaction
// receipts and independent rollback remain the activation owner's obligation.
// AcquireBinaryInstallLock serializes the fixed named staging slot. Recovery
// of interrupted atomic-write temporary inputs belongs to the future durable
// activation transaction and is not established by this preparation step.
func installRootCLICandidate(source, destination string) error {
	stage := destination + ".staged"
	if filepath.Ext(destination) == ".exe" {
		stage += ".exe"
	}
	defer retireRootCLIStage(stage)
	sha, err := copyRootCLIImage(source, stage, "", 0o700)
	if err != nil {
		return fmt.Errorf("stage root CLI: %w", err)
	}
	if err := coreactivation.ValidateRootExecutableRole(stage); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := probeRootCLIIdentity(ctx, stage); err != nil {
		return err
	}
	// Rejected candidates must not rotate the existing fallback copy.
	if err := buildinfo.PreserveRootBinaryFallback(destination); err != nil {
		return fmt.Errorf("retain root CLI fallback before installation: %w", err)
	}
	if _, err := copyRootCLIImage(stage, destination, sha, 0o755); err != nil {
		return fmt.Errorf("publish admitted root CLI bytes: %w", err)
	}
	return nil
}

// retireRootCLIStage removes the fixed staging slot installRootCLICandidate
// wrote. The slot is this function's private scratch copy, created and retired
// under the binary install lock; it is never an installed artifact, so it does
// not go through artifactledger.Guard (whose artifact lock is that same install
// lock and would self-deadlock here). Keep this helper single-purpose: the seam
// exemption that permits it names this function, not its caller.
func retireRootCLIStage(stage string) {
	_ = os.Remove(stage)
}

func copyRootCLIImage(source, destination, expected string, perm os.FileMode) (string, error) {
	before, err := os.Lstat(source)
	if err != nil {
		return "", err
	}
	if !before.Mode().IsRegular() || before.Size() <= 0 || before.Size() > rootCLIImageLimit {
		return "", fmt.Errorf("root CLI image must be a bounded nonempty regular file")
	}
	file, err := os.Open(source)
	if err != nil {
		return "", err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(before, opened) {
		return "", fmt.Errorf("root CLI source changed while opening")
	}
	var digest string
	err = config.WriteOwnedFileDurableStream(destination, perm, func(w io.Writer) error {
		h := sha256.New()
		n, err := io.Copy(io.MultiWriter(w, h), io.LimitReader(file, rootCLIImageLimit+1))
		if err != nil {
			return err
		}
		if n <= 0 || n > rootCLIImageLimit {
			return fmt.Errorf("root CLI image exceeded read bounds")
		}
		digest = hex.EncodeToString(h.Sum(nil))
		if expected != "" && digest != expected {
			return fmt.Errorf("root CLI image changed after its canary")
		}
		return nil
	})
	return digest, err
}

func probeRootCLIIdentity(ctx context.Context, path string) (*cliv1.CliVersion, error) {
	return coreactivation.ProbeRootIdentity(ctx, path)
}
