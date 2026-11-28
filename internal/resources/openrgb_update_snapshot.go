package resources

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/operatorstate"
)

// Each role freezes the catalog separately from effective merged settings.
// A later checkout or selected policy cannot silently redefine rollback.
type openRGBFrozenUpdateRole struct {
	ContractDigest string                          `json:"contract_digest"`
	Policy         hostreqkit.OpenRGBLinuxPolicy   `json:"policy"`
	NativePolicy   *hostreqkit.OpenRGBNativePolicy `json:"native_policy,omitempty"`
	Catalog        []byte                          `json:"detector_catalog"`
}
type openRGBManagedUpdatePolicies struct {
	Source   *operatorstate.ResourceSourceChoice `json:"source,omitempty"`
	Original openRGBFrozenUpdateRole             `json:"original"`
	Target   openRGBFrozenUpdateRole             `json:"target"`
	Rollback openRGBFrozenUpdateRole             `json:"rollback"`
}

// Structural reads stay available for exact revocation after safety policy
// evolves. Only launch/stop qualification uses binding's current scope parser.
func (r openRGBFrozenUpdateRole) structuralBinding(contract ManagedUpdateContract) (openRGBExecPolicy, error) {
	if err := contract.validate(); err != nil {
		return openRGBExecPolicy{}, err
	}
	id, artifact, catalog, err := r.policyIdentity()
	if err != nil {
		return openRGBExecPolicy{}, err
	}
	if r.ContractDigest != contract.Digest || len(r.Catalog) == 0 || len(r.Catalog) > managedUpdateConfigMaxBytes || artifact != contract.Artifact.SHA256 || fmt.Sprintf("%x", sha256.Sum256(r.Catalog)) != catalog {
		return openRGBExecPolicy{}, errors.New("invalid frozen OpenRGB structural policy binding")
	}
	b := openRGBExecPolicy{ID: id, ArtifactSHA256: contract.Artifact.SHA256, DetectorCatalogSHA256: catalog, ConfigSHA256: contract.ConfigSHA256}
	if r.NativePolicy != nil {
		b.Platform = r.NativePolicy.Platform
		b.Account = r.NativePolicy.Account
	}
	return b, nil
}
func (r openRGBFrozenUpdateRole) binding(contract ManagedUpdateContract) (openRGBExecPolicy, error) {
	if r.ContractDigest != contract.Digest || len(r.Catalog) == 0 || len(r.Catalog) > managedUpdateConfigMaxBytes {
		return openRGBExecPolicy{}, errors.New("OpenRGB frozen role catalog or contract identity is invalid")
	}
	return r.record().freezeExecPolicy(contract.Artifact.SHA256, r.Catalog, contract.ConfigBytes)
}
func openRGBFrozenUpdateRoleFor(u ManagedServiceUpdate, contract ManagedUpdateContract) (openRGBFrozenUpdateRole, error) {
	if u.Resource != openRGBResourceName || u.OpenRGBPolicies == nil {
		return openRGBFrozenUpdateRole{}, errors.New("OpenRGB update has no frozen role policy and detector catalog")
	}
	for _, r := range []openRGBFrozenUpdateRole{u.OpenRGBPolicies.Original, u.OpenRGBPolicies.Target, u.OpenRGBPolicies.Rollback} {
		if r.ContractDigest == contract.Digest {
			if _, err := r.binding(contract); err != nil {
				return openRGBFrozenUpdateRole{}, err
			}
			return r, nil
		}
	}
	return openRGBFrozenUpdateRole{}, errors.New("OpenRGB contract is outside the frozen role policies")
}
func (c *Controller) loadOpenRGBFrozenUpdateRolePolicy(ctx context.Context, u ManagedServiceUpdate, contract ManagedUpdateContract) (openRGBUpdateRolePolicy, error) {
	if _, err := c.verifyOpenRGBRoleArtifact(ctx, contract.Artifact); err != nil {
		return openRGBUpdateRolePolicy{}, err
	}
	r, err := openRGBFrozenUpdateRoleFor(u, contract)
	if err != nil {
		return openRGBUpdateRolePolicy{}, err
	}
	b, err := r.binding(contract)
	if err != nil {
		return openRGBUpdateRolePolicy{}, err
	}
	if r.NativePolicy == nil {
		if err := hostreqkit.ValidateOpenRGBLinuxDetectorScope(r.Policy); err != nil {
			return openRGBUpdateRolePolicy{}, err
		}
	}
	return openRGBUpdateRolePolicy{binding: b, policy: r.Policy, nativePolicy: r.NativePolicy}, nil
}
func (c *Controller) openRGBFrozenExecRole(ctx context.Context, fence managedUpdateExecFence, contract ManagedUpdateContract) (openRGBFrozenUpdateRole, error) {
	if err := ctx.Err(); err != nil {
		return openRGBFrozenUpdateRole{}, err
	}
	store, err := newManagedUpdateStore(openRGBResourceName)
	if err != nil {
		return openRGBFrozenUpdateRole{}, err
	}
	ledger, err := store.read()
	if err != nil {
		return openRGBFrozenUpdateRole{}, err
	}
	for _, entry := range ledger.Records {
		u := entry.State
		if u.ID != fence.OperationID {
			continue
		}
		p, admitted, err := managedUpdateAdmittedRole(u, fence.Intent.InstanceID)
		if err != nil {
			return openRGBFrozenUpdateRole{}, err
		}
		if p.Intent == nil || *p.Intent != fence.Intent || admitted.Digest != contract.Digest {
			return openRGBFrozenUpdateRole{}, errors.New("OpenRGB exec differs from frozen admitted role")
		}
		return openRGBFrozenUpdateRoleFor(u, contract)
	}
	return openRGBFrozenUpdateRole{}, ErrManagedUpdateNotFound
}
func validateOpenRGBUpdateSnapshot(u ManagedServiceUpdate) error {
	if u.OpenRGBPolicies == nil {
		return nil
	}
	if u.Resource != openRGBResourceName {
		return errors.New("non-OpenRGB update contains OpenRGB role policies")
	}
	old := u.OpenRGBPolicies.Original
	_, artifact, catalog, err := old.policyIdentity()
	if err != nil {
		return err
	}
	if !updateDigest(old.ContractDigest) || artifact != u.Original.ArtifactSHA256 || len(old.Catalog) == 0 || len(old.Catalog) > managedUpdateConfigMaxBytes || fmt.Sprintf("%x", sha256.Sum256(old.Catalog)) != catalog {
		return errors.New("invalid retained OpenRGB original policy")
	}
	for _, pair := range []struct {
		r openRGBFrozenUpdateRole
		c ManagedUpdateContract
	}{{u.OpenRGBPolicies.Target, u.Target}, {u.OpenRGBPolicies.Rollback, u.Rollback}} {
		if _, err := pair.r.structuralBinding(pair.c); err != nil {
			return err
		}
	}
	if !old.samePhysicalGrant(u.OpenRGBPolicies.Target) || !old.samePhysicalGrant(u.OpenRGBPolicies.Rollback) {
		return errors.New("OpenRGB update changes the approved physical device grant")
	}
	return nil
}
func (c *Controller) freezeOpenRGBUpdateSnapshot(ctx context.Context, u ManagedServiceUpdate) (*openRGBManagedUpdatePolicies, error) {
	old, err := c.openRGBFrozenOriginalRole(ctx, u.Original)
	if err != nil {
		return nil, err
	}
	targetCatalog, err := c.openRGBUpdateTargetCatalog(u)
	if err != nil {
		return nil, err
	}
	targetPolicy := old.Policy
	// Artifact/catalog binding changes only metadata. Physical selectors and
	// installed grant remain exactly the already approved set.
	targetPolicy.ArtifactSHA256 = u.Target.Artifact.SHA256
	targetPolicy.DetectorSHA256 = fmt.Sprintf("%x", sha256.Sum256(targetCatalog))
	target := openRGBFrozenUpdateRole{ContractDigest: u.Target.Digest, Policy: targetPolicy, Catalog: targetCatalog}
	if old.NativePolicy != nil {
		p := *old.NativePolicy
		p.ArtifactSHA256 = u.Target.Artifact.SHA256
		p.DetectorSHA256 = fmt.Sprintf("%x", sha256.Sum256(targetCatalog))
		target.Policy = hostreqkit.OpenRGBLinuxPolicy{}
		target.NativePolicy = &p
	}
	if _, err := target.binding(u.Target); err != nil {
		return nil, fmt.Errorf("qualify target OpenRGB detector policy: %w", err)
	}
	rollback := old
	rollback.ContractDigest = u.Rollback.Digest
	if _, err := rollback.binding(u.Rollback); err != nil {
		return nil, err
	}
	state, err := c.operatorStateService().Load(ctx)
	if err != nil {
		return nil, err
	}
	return &openRGBManagedUpdatePolicies{Source: state.Resources[openRGBResourceName].Source, Original: old, Target: target, Rollback: rollback}, nil
}

func (c *Controller) openRGBFrozenOriginalRole(ctx context.Context, state ManagedServiceState) (openRGBFrozenUpdateRole, error) {
	original, _, err := openRGBOriginalLaunch(state)
	if err != nil {
		return openRGBFrozenUpdateRole{}, err
	}
	var old openRGBFrozenUpdateRole
	if state.InitialLaunch != nil {
		a, _, err := readOpenRGBInitialServing(state)
		if err != nil {
			return openRGBFrozenUpdateRole{}, err
		}
		old = openRGBFrozenUpdateRole{ContractDigest: original.Digest, Catalog: a.Catalog}
		if a.NativeDevicePolicy != nil {
			old.NativePolicy = a.NativeDevicePolicy
		} else if a.DevicePolicy != nil {
			old.Policy = *a.DevicePolicy
		} else {
			record, err := c.loadOpenRGBDevicePolicy(original.Manifest)
			if err != nil || record == nil {
				return openRGBFrozenUpdateRole{}, errors.Join(errors.New("original OpenRGB device policy is not retained"), err)
			}
			old.Policy = record.Policy
			old.NativePolicy = record.NativePolicy
		}
	} else if state.UpdateLaunch != nil {
		store, err := newManagedUpdateStore(openRGBResourceName)
		if err != nil {
			return openRGBFrozenUpdateRole{}, err
		}
		prior, err := store.Lookup(state.UpdateLaunch.RequestID)
		if err != nil {
			return openRGBFrozenUpdateRole{}, err
		}
		if prior.ID != state.UpdateLaunch.OperationID {
			return openRGBFrozenUpdateRole{}, ErrManagedUpdateConflict
		}

		if prior.OpenRGBPolicies == nil {
			return openRGBFrozenUpdateRole{}, errors.New("retained OpenRGB update has no frozen role policy")
		}
		for _, role := range []openRGBFrozenUpdateRole{prior.OpenRGBPolicies.Original, prior.OpenRGBPolicies.Target, prior.OpenRGBPolicies.Rollback} {
			if role.ContractDigest == original.Digest {
				old = role
				break
			}
		}
		_, err = old.structuralBinding(original)
		if err != nil {
			return openRGBFrozenUpdateRole{}, err
		}
	} else {
		return openRGBFrozenUpdateRole{}, errors.New("OpenRGB update requires an exact retained native original")
	}
	if len(old.Catalog) == 0 {
		old.Catalog, err = readManagedUpdateConfigFile(filepath.Join(c.Root, "resources", openRGBResourceName), original.ConfigFile)
		if err != nil {
			return openRGBFrozenUpdateRole{}, err
		}
	}
	if _, err := old.structuralBinding(original); err != nil {
		return openRGBFrozenUpdateRole{}, fmt.Errorf("qualify retained original OpenRGB policy: %w", err)
	}
	return old, nil
}

func (c *Controller) openRGBUpdateTargetCatalog(u ManagedServiceUpdate) ([]byte, error) {
	if u.RollbackOf != "" {
		store, err := newManagedUpdateStore(openRGBResourceName)
		if err != nil {
			return nil, err
		}
		source, err := store.Lookup(u.RollbackOf)
		if err != nil {
			return nil, err
		}
		if source.OpenRGBPolicies == nil {
			return nil, errors.New("retained OpenRGB rollback has no qualified detector catalog")
		}
		return source.OpenRGBPolicies.Rollback.Catalog, nil
	}
	return readManagedUpdateConfigFile(filepath.Join(c.Root, "resources", openRGBResourceName), u.Target.ConfigFile)
}
func (c *Controller) mergeOpenRGBUpdateTarget(ctx context.Context, target ManagedUpdateContract, source managedUpdateAdmissionSource, rollbackOf string) (ManagedUpdateContract, error) {
	u := ManagedServiceUpdate{Resource: openRGBResourceName, Original: source.Process, Target: target, RollbackOf: rollbackOf}
	old, err := c.openRGBFrozenOriginalRole(ctx, source.Process)
	if err != nil {
		return ManagedUpdateContract{}, err
	}
	catalog, err := c.openRGBUpdateTargetCatalog(u)
	if err != nil {
		return ManagedUpdateContract{}, err
	}
	decoded, err := decodeOpenRGBDetectorCatalog(catalog)
	if err != nil {
		return ManagedUpdateContract{}, err
	}
	names, err := old.record().selectedDetectors(decoded)
	if err != nil {
		return ManagedUpdateContract{}, err
	}
	settings, err := mergeOpenRGBConfigContentWithSelection(source.Contract.ConfigBytes, string(catalog), names)
	if err != nil {
		return ManagedUpdateContract{}, err
	}
	if old.NativePolicy != nil {
		data, e := enforceNativeOpenRGBSettings([]byte(settings))
		if e != nil {
			return ManagedUpdateContract{}, e
		}
		settings = string(data)
	}
	return target.withConfigBytes(target.ConfigFile, []byte(settings))
}
