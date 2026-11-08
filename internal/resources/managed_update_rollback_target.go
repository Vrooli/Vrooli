package resources

import (
	"context"
	"fmt"

	"github.com/vrooli/binaryfetch"
)

// A rollback is a new reviewed operation over retained, qualified bytes. It
// never rewrites an old receipt, changes the desired manifest or reacquires an
// old release through a new URL. The currently serving image becomes its own
// rollback, and every ordinary qualification/admission gate still applies.
func (c *Controller) managedUpdateAdmissionTarget(ctx context.Context, m ResourceManifest, store *managedUpdateStore, opts ManagedServiceUpdateAdmission) (ManagedUpdateContract, error) {
	if opts.RollbackOf == "" {
		artifact, err := c.inspectManagedServiceArtifact(ctx, m)
		if err != nil {
			return ManagedUpdateContract{}, err
		}
		return newManagedUpdateContract(m, artifact)
	}
	source, err := store.Lookup(opts.RollbackOf)
	if err != nil {
		return ManagedUpdateContract{}, err
	}
	if source.Resource != m.Name || !source.Phase.Terminal() || !source.RollbackQualificationRequired || source.RollbackQualifiedAt == nil {
		return ManagedUpdateContract{}, fmt.Errorf("rollback source is not a terminal qualified update receipt")
	}
	target := source.Rollback
	if target.Artifact.Version != opts.ExpectedVersion || target.Artifact.SHA256 != opts.ExpectedSHA256 {
		return ManagedUpdateContract{}, fmt.Errorf("retained rollback differs from the explicitly selected release")
	}
	if err := target.validate(); err != nil {
		return ManagedUpdateContract{}, err
	}
	if err := binaryfetch.VerifyArtifactContext(ctx, target.Artifact.Path, "file", target.Artifact.SHA256); err != nil {
		return ManagedUpdateContract{}, fmt.Errorf("retained rollback image is unavailable or changed: %w", err)
	}
	return target, nil
}
