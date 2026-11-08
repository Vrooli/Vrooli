package coreactivation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// ErrStateContractChanged marks a candidate whose persistent-state contract
// differs from the serving image's: the prior image could not be relied on
// to read what the candidate writes, so automatic rollback is unavailable.
var ErrStateContractChanged = errors.New("candidate changes the persistent-state contract")

// ErrUnqualifiedStateContract marks an image that reports no persistent-state
// contract: an ungoverned build, or a governed build whose source checks
// failed when it was stamped. Its refusal carries the image's own reason.
var ErrUnqualifiedStateContract = errors.New("image has no qualified persistent-state contract")

// QualifyRollbackIdentities compares metadata from the actual retained images.
// A runtime-home digest is a separate layout constraint, never a substitute for
// persistence compatibility. Unstamped images remain ineligible.
func QualifyRollbackIdentities(prior, candidate *cliv1.CliVersion) error {
	if prior == nil || candidate == nil {
		return fmt.Errorf("%w: retained executable identities are required", ErrConflict)
	}
	for _, side := range []struct {
		role     string
		identity *cliv1.CliVersion
	}{{"serving", prior}, {"candidate", candidate}} {
		identity := side.identity
		if !digestPattern.MatchString(identity.GetRuntimeHomeContractSha256()) || !digestPattern.MatchString(identity.GetBuildIdentity()) {
			return fmt.Errorf("%w: %s image lacks qualified build or layout metadata (unstamped or ungoverned build)", ErrConflict, side.role)
		}
		if !digestPattern.MatchString(identity.GetPersistentStateContractSha256()) {
			return fmt.Errorf("%w: %w: the %s image", ErrConflict, ErrUnqualifiedStateContract, side.role)
		}
	}
	if prior.GetRuntimeHomeContractSha256() != candidate.GetRuntimeHomeContractSha256() {
		return fmt.Errorf("%w: candidate changes the runtime-home layout contract", ErrConflict)
	}
	if prior.GetPersistentStateContractSha256() != candidate.GetPersistentStateContractSha256() {
		return fmt.Errorf("%w: %w (serving %.12s, candidate %.12s)", ErrConflict, ErrStateContractChanged, prior.GetPersistentStateContractSha256(), candidate.GetPersistentStateContractSha256())
	}
	return nil
}

// QualifyFrozenCompatibility derives admission evidence from the frozen image
// bytes. Record fields alone cannot assert compatibility. Call under admission's
// preparation lock; this method performs no publication or host initialization.
func (a Assets) QualifyFrozenCompatibility(ctx context.Context, r Record) error {
	identities := make([]*cliv1.CliVersion, 0, 2)
	paths := make([]string, 0, 2)
	for _, name := range []string{"prior", "candidate"} {
		path, err := a.Image(ctx, r, name)
		if err != nil {
			return err
		}
		paths = append(paths, path)
		if err := ValidateRootExecutableRole(path); err != nil {
			return err
		}
		identity, err := ProbeRootIdentity(ctx, path)
		if err != nil {
			return err
		}
		// Recheck frozen byte identity after invoking the read-only probe.
		if _, err := a.Image(ctx, r, name); err != nil {
			return err
		}
		identities = append(identities, identity)
	}
	if err := QualifyRollbackIdentities(identities[0], identities[1]); err != nil {
		return ExplainRollbackRefusal(ctx, err, paths[0], paths[1])
	}
	if identities[0].GetBuildIdentity() != r.PriorBuild || identities[1].GetBuildIdentity() != r.CandidateBuild || identities[0].GetPersistentStateContractSha256() != r.PriorStateContract || identities[1].GetPersistentStateContractSha256() != r.CandidateStateContract {
		return fmt.Errorf("%w: recorded compatibility is not the frozen images' identity", ErrConflict)
	}
	return nil
}

// GuardEntrypoint prevents an unverified candidate from running the general CLI
// (sessions, shims, setup and other unrelated state writers). The fixed native
// supervisor is allowed only at its admitted activation/verification edge and
// with its actual build and persistence contract. Prior-image commands remain
// available for owner recovery. This read must precede normal CLI initialization.
func (s *Store) GuardEntrypoint(ctx context.Context, actualSHA, build, state string, nativeSupervisor bool) error {
	if ctx == nil {
		return ErrConflict
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	r, exists, err := s.Current()
	if err != nil {
		return err
	}
	if !exists || r.Terminal() {
		return nil
	}
	if !digestPattern.MatchString(actualSHA) {
		return ErrConflict
	}
	if actualSHA != r.CandidateSHA {
		if actualSHA == r.PriorSHA {
			return nil
		}
		return fmt.Errorf("%w: active root update does not identify this executable", ErrConflict)
	}
	if !nativeSupervisor || !time.Now().Before(time.Unix(0, r.DeadlineUnixNano)) || r.Fence != Claimed || (r.Phase != Activating && r.Phase != Verifying) || build != r.CandidateBuild || state != r.CandidateStateContract {
		return fmt.Errorf("%w: candidate command must wait for native verification", ErrConflict)
	}
	return nil
}

// GuardCurrentEntrypoint opens the actual running image through the platform
// owner. The installed pathname may already refer to a replacement executable.
func (s *Store) GuardCurrentEntrypoint(ctx context.Context, build, state string, nativeSupervisor bool) error {
	if ctx == nil {
		return ErrConflict
	}
	r, exists, err := s.Current()
	if err != nil {
		return err
	}
	if !exists || r.Terminal() {
		return nil
	}
	sha, err := runningImageDigest(ctx, os.Getpid())
	if err != nil {
		return err
	}
	return s.GuardEntrypoint(ctx, sha, build, state, nativeSupervisor)
}
