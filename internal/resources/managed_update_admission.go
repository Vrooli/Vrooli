package resources

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"time"

	"github.com/vrooli/vrooli/internal/capacity"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
)

type ManagedServiceUpdateAdmission struct {
	RequestID       string `json:"request_id"`
	ExpectedVersion string `json:"expected_version"`
	ExpectedSHA256  string `json:"expected_sha256"`
	ExplicitRetry   bool   `json:"explicit_retry"`
	RollbackOf      string `json:"rollback_of,omitempty"`
}

// AdmitManagedServiceUpdate freezes a verified, already staged release and the
// exact serving owner. It never stops or launches a service. Legacy launches
// must qualify the retained image in isolation before target candidate launch.
// Reusing the request reconciles the same operation, including after caller loss.
func (c *Controller) AdmitManagedServiceUpdate(ctx context.Context, name string, opts ManagedServiceUpdateAdmission) (ManagedServiceUpdate, bool, error) {

	if name != openRGBResourceName {
		return c.admitManagedServiceUpdate(ctx, name, opts)
	}
	var replayed bool
	u, err := c.runOpenRGBPublicUpdate(ctx, func(ctx context.Context) (ManagedServiceUpdate, error) {
		u, r, err := c.admitManagedServiceUpdate(ctx, name, opts)
		replayed = r
		return u, err
	})
	return u, replayed, err
}

func (c *Controller) admitManagedServiceUpdate(ctx context.Context, name string, opts ManagedServiceUpdateAdmission) (result ManagedServiceUpdate, replayed bool, returnErr error) {
	if c == nil || ctx == nil || !managedUpdateResource.MatchString(name) || !managedUpdateKey.MatchString(opts.RequestID) || opts.ExpectedVersion == "" || !updateDigest(opts.ExpectedSHA256) {
		return ManagedServiceUpdate{}, false, fmt.Errorf("update admission requires resource, request and exact release identity")
	}
	if opts.RollbackOf != "" && (!managedUpdateKey.MatchString(opts.RollbackOf) || opts.RollbackOf == opts.RequestID) {
		return ManagedServiceUpdate{}, false, fmt.Errorf("rollback requires a different retained source request")
	}
	release, err := acquireManagedServiceLifecycleLock(ctx, name)
	if err != nil {
		return ManagedServiceUpdate{}, false, err
	}
	defer release()
	if name == openRGBResourceName {
		if err := c.requireOpenRGBEnabled(ctx); err != nil {
			return ManagedServiceUpdate{}, false, err
		}
	}
	store, err := newManagedUpdateStore(name)
	if err != nil {
		return ManagedServiceUpdate{}, false, err
	}
	if old, err := store.Lookup(opts.RequestID); err == nil {
		if old.Target.Artifact.Version != opts.ExpectedVersion || old.Target.Artifact.SHA256 != opts.ExpectedSHA256 || old.RollbackOf != opts.RollbackOf {
			return ManagedServiceUpdate{}, false, ErrManagedUpdateConflict
		}
		old, err = store.Confirm(ctx, opts.RequestID, old.Revision)
		return old, true, err
	} else if !errors.Is(err, ErrManagedUpdateNotFound) {
		return ManagedServiceUpdate{}, false, err
	}
	if _, err := store.Active(); !errors.Is(err, ErrManagedUpdateNotFound) {
		if err == nil {
			err = ErrManagedUpdateConflict
		}
		return ManagedServiceUpdate{}, false, err
	}
	m, err := c.managedServiceArtifactManifest(name)
	if err != nil {
		return ManagedServiceUpdate{}, false, err
	}
	if err := manifestpkg.ValidateManagedServiceUpdate(m); err != nil {
		return ManagedServiceUpdate{}, false, err
	}
	var target ManagedUpdateContract
	if name == openRGBResourceName {
		target, err = c.openRGBPublicOwners().target(ctx, m, store, opts)
	} else {
		target, err = c.managedUpdateAdmissionTarget(ctx, m, store, opts)
	}
	if err != nil {
		return ManagedServiceUpdate{}, false, err
	}
	artifact := target.Artifact
	if artifact.Version != opts.ExpectedVersion || artifact.SHA256 != opts.ExpectedSHA256 {
		return ManagedServiceUpdate{}, false, fmt.Errorf("declared staged release changed before update admission")
	}
	var observed managedUpdateAdmissionSource
	if name == openRGBResourceName {
		if err := c.openRGBPublicOwners().profile(ctx, artifact); err != nil {
			return ManagedServiceUpdate{}, false, err
		}
		observed, err = c.openRGBPublicOwners().source(ctx, m)
	} else {
		observed, err = c.observeManagedUpdateAdmissionSource(ctx, m)
	}
	if err != nil {
		return ManagedServiceUpdate{}, false, err
	}
	if artifact.SHA256 == observed.Contract.Artifact.SHA256 {
		return ManagedServiceUpdate{}, false, fmt.Errorf("target is already the serving image")
	}
	if name == openRGBResourceName {
		target, err = c.mergeOpenRGBUpdateTarget(ctx, target, observed, opts.RollbackOf)
	} else {
		target, err = mergeManagedUpdateTargetConfig(target, observed.Contract)
	}
	if err != nil {
		return ManagedServiceUpdate{}, false, err
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return ManagedServiceUpdate{}, false, err
	}
	now := time.Now().UTC()
	u := ManagedServiceUpdate{SchemaVersion: managedUpdateSchemaVersion, ID: "update-" + hex.EncodeToString(nonce[:]), RequestID: opts.RequestID, Resource: name, Revision: 1, Phase: UpdatePrepared, CreatedAt: now, UpdatedAt: now, SourceRevision: observed.SourceRevision, Target: target, Rollback: observed.Contract, Original: observed.Process, OriginalScope: observed.Scope, RollbackQualificationRequired: true}
	u.RollbackOf = opts.RollbackOf
	if name == openRGBResourceName {
		u.OpenRGBPolicies, err = c.openRGBPublicOwners().snapshot(ctx, u)
		if err != nil {
			return ManagedServiceUpdate{}, false, err
		}
		if err := c.qualifyOpenRGBResumption(ctx, u); err != nil {
			return ManagedServiceUpdate{}, false, err
		}
	}

	budget := managedUpdateCapacityBudget(u)
	budget.LedgerPath, err = capacity.DefaultLedgerReference(c.Home)
	if err != nil {
		return ManagedServiceUpdate{}, false, err
	}
	u.Capacity = &budget
	paths, err := resourceStoragePaths(name)
	if err != nil {
		return ManagedServiceUpdate{}, false, err
	}
	preserveInputs := false
	defer func() {
		if preserveInputs || returnErr == nil {
			return
		}
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		returnErr = errors.Join(returnErr, cleanupManagedUpdateRejectedAdmission(cleanup, store, u, paths.StateDir))
	}()
	if err := c.managedUpdateEffectOwner().reserve(ctx, u); err != nil {
		return ManagedServiceUpdate{}, false, err
	}
	credentials, err := newManagedUpdateCredentialStore(paths.StateDir, u.ID)
	if err != nil {
		return ManagedServiceUpdate{}, false, err
	}
	u.Credentials, err = credentials.Prepare(ctx, u)
	if err != nil {
		return ManagedServiceUpdate{}, false, err
	}
	// A current authority without version metadata can rotate during capture.
	// Compare the pinned bytes to the actual old launch's materialized files,
	// then repeat the complete owner/authority observation before admitting it.
	for _, d := range u.Rollback.Manifest.ManagedService.CredentialFiles {
		field := d.Field
		if field == "" {
			field = managedUpdateCredentialDefaultField
		}
		var item ManagedUpdateCredentialItem
		for _, candidate := range u.Credentials.Items {
			if candidate.LogicalID == d.LogicalID && candidate.Field == field {
				item = candidate
				break
			}
		}
		resolve := func() (string, error) {
			data, e := readManagedUpdatePrivateFile(filepath.Join(credentials.dir, item.FileKey+".value"), 1<<20)
			return string(data), e
		}
		if err := verifyManagedUpdateCredentialBinding(filepath.Join(observed.CredentialStateDir, filepath.FromSlash(d.Path)), resolve); err != nil {
			return ManagedServiceUpdate{}, false, fmt.Errorf("pinned rollback credential no longer matches the serving launch: %w", err)
		}
	}
	var after managedUpdateAdmissionSource
	if name == openRGBResourceName {
		after, err = c.openRGBPublicOwners().source(ctx, m)
	} else {
		after, err = c.observeManagedUpdateAdmissionSource(ctx, m)
	}
	if err != nil {
		return ManagedServiceUpdate{}, false, err
	}
	if observed.SourceRevision != after.SourceRevision || observed.Contract.Digest != after.Contract.Digest || !reflect.DeepEqual(observed.Process, after.Process) || !reflect.DeepEqual(observed.Scope, after.Scope) {
		return ManagedServiceUpdate{}, false, fmt.Errorf("serving owner changed before durable update admission")
	}
	result, replayed, returnErr = store.Create(ctx, u, opts.ExplicitRetry)
	preserveInputs = returnErr == nil || errors.Is(returnErr, ErrManagedUpdateCommitUnknown)
	return result, replayed, returnErr
}
