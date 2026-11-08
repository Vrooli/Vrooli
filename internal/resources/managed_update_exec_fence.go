package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/internal/config"
)

// The exec fence's states. The prior image validates a fence strictly against
// these words, so they are stored vocabulary.
const (
	managedUpdateExecFencePrepared = "prepared"
	managedUpdateExecFenceClaimed  = "claimed"
	managedUpdateExecFenceRevoked  = "revoked"
)

var (
	ErrManagedUpdateExecClaimed = errors.New("managed update resource exec has already been claimed")
	ErrManagedUpdateExecRevoked = errors.New("managed update resource exec is revoked")
)

// A native submission can arrive after its caller has disappeared. The
// resource wrapper must claim this separate, durable once-only exec fence
// immediately before replacing itself with the resource. A claimed fence is
// never replay permission, even when its acknowledgement was lost. Revocation
// prevents a late native wrapper from executing the resource at all.
type managedUpdateExecFence struct {
	SchemaVersion int                       `json:"schema_version"`
	Resource      string                    `json:"resource"`
	OperationID   string                    `json:"operation_id"`
	Intent        ManagedUpdateLaunchIntent `json:"intent"`
	State         string                    `json:"state"`
	Claim         *platform.ServiceRef      `json:"claim,omitempty"`
	ClaimedAt     *time.Time                `json:"claimed_at,omitempty"`
	RevokedAt     *time.Time                `json:"revoked_at,omitempty"`
	RejectedClaim *platform.ServiceRef      `json:"rejected_claim,omitempty"`
	RejectedAt    *time.Time                `json:"rejected_at,omitempty"`
}

func (f managedUpdateExecFence) validate() error {
	if f.SchemaVersion != managedUpdateExecFenceSchemaVersion || !managedUpdateResource.MatchString(f.Resource) || !managedUpdateKey.MatchString(f.OperationID) || !managedUpdateKey.MatchString(f.Intent.InstanceID) || !updateDigest(f.Intent.ContractDigest) || !updateDigest(f.Intent.OwnershipTokenHash) || f.Intent.NativeIntent.ValidateIdentity() != nil || f.Intent.NativeIntent.Owner != "resource-update" || f.Intent.NativeIntent.AttemptID != f.Intent.InstanceID || f.Intent.Runtime.validate() != nil {
		return fmt.Errorf("invalid resource exec fence identity")
	}
	if (f.Claim == nil) != (f.ClaimedAt == nil) {
		return fmt.Errorf("resource exec claim identity is incomplete")
	}
	if f.Claim != nil && (f.Claim.ValidateIdentity() != nil || f.Claim.Intent != platform.OperationIntent(f.Intent.NativeIntent) || f.Claim.RootPID <= 0 || f.ClaimedAt.IsZero()) {
		return fmt.Errorf("resource exec claim differs from its native intent")
	}
	if f.RevokedAt != nil && (f.RevokedAt.IsZero() || f.ClaimedAt != nil && f.RevokedAt.Before(*f.ClaimedAt)) {
		return fmt.Errorf("invalid resource exec revocation time")
	}
	if (f.RejectedClaim == nil) != (f.RejectedAt == nil) {
		return fmt.Errorf("resource exec rejection is incomplete")
	}
	if f.RejectedClaim != nil && (f.State != managedUpdateExecFenceRevoked || f.RevokedAt == nil || f.RejectedAt.IsZero() || f.RejectedAt.Before(*f.RevokedAt) || f.RejectedClaim.ValidateIdentity() != nil || f.RejectedClaim.RootPID <= 0 || f.RejectedClaim.Intent != platform.OperationIntent(f.Intent.NativeIntent)) {
		return fmt.Errorf("resource exec rejection differs from its revoked native intent")
	}
	switch f.State {
	case managedUpdateExecFencePrepared:
		if f.Claim != nil || f.RevokedAt != nil {
			return fmt.Errorf("prepared resource exec already has effects")
		}
	case managedUpdateExecFenceClaimed:
		if f.Claim == nil || f.RevokedAt != nil {
			return fmt.Errorf("claimed resource exec is incomplete or revoked")
		}
	case managedUpdateExecFenceRevoked:
		if f.RevokedAt == nil {
			return fmt.Errorf("revoked resource exec has no revocation record")
		}
	default:
		return fmt.Errorf("unknown resource exec fence state")
	}
	return nil
}

type managedUpdateExecStore struct {
	resource, operation, instance string
	path                          string
	lock                          func(context.Context) (func(), error)
	write                         func(string, []byte, os.FileMode) error
	now                           func() time.Time
}

func newManagedUpdateExecStore(resource, operation, instance string) (*managedUpdateExecStore, error) {
	if !managedUpdateResource.MatchString(resource) || !managedUpdateKey.MatchString(operation) || !managedUpdateKey.MatchString(instance) {
		return nil, fmt.Errorf("invalid resource exec store identity")
	}
	paths, err := resourceStoragePaths(resource)
	if err != nil {
		return nil, err
	}
	return &managedUpdateExecStore{
		resource: resource, operation: operation, instance: instance,
		path: filepath.Join(paths.StateDir, "managed-update-exec", operation, instance, "fence.json"),
		lock: func(ctx context.Context) (func(), error) {
			return artifactlock.AcquireContext(ctx, "resource-update-exec:"+resource+":"+operation+":"+instance)
		},
		write: config.WriteOwnedFileDurable, now: time.Now,
	}, nil
}

func (s *managedUpdateExecStore) read() (managedUpdateExecFence, error) {
	file, err := os.Open(s.path)
	if err != nil {
		return managedUpdateExecFence{}, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, (64<<10)+1))
	if err != nil {
		return managedUpdateExecFence{}, err
	}
	if len(data) > 64<<10 {
		return managedUpdateExecFence{}, fmt.Errorf("resource exec fence exceeds its size bound")
	}
	if err := validateManagedUpdateJSON(data); err != nil {
		return managedUpdateExecFence{}, err
	}
	var f managedUpdateExecFence
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&f); err != nil {
		return managedUpdateExecFence{}, fmt.Errorf("decode resource exec fence: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return managedUpdateExecFence{}, fmt.Errorf("resource exec fence has trailing content")
	}
	if err := s.validateIdentity(f); err != nil {
		return managedUpdateExecFence{}, err
	}
	return f, nil
}

func (s *managedUpdateExecStore) validateIdentity(f managedUpdateExecFence) error {
	if err := f.validate(); err != nil {
		return err
	}
	if f.Resource != s.resource || f.OperationID != s.operation || f.Intent.InstanceID != s.instance {
		return fmt.Errorf("%w: resource exec fence belongs to another owner path", ErrManagedUpdateConflict)
	}
	return nil
}

func (s *managedUpdateExecStore) save(f managedUpdateExecFence) error {
	if err := s.validateIdentity(f); err != nil {
		return err
	}
	data, err := json.Marshal(f)
	if err != nil {
		return err
	}
	if len(data) > 64<<10 {
		return fmt.Errorf("resource exec fence exceeds its size bound")
	}
	if err := s.write(s.path, data, 0o600); err != nil {
		return fmt.Errorf("%w: %w", ErrManagedUpdateCommitUnknown, err)
	}
	return nil
}

func sameManagedExecIdentity(a, b managedUpdateExecFence) bool {
	return a.SchemaVersion == b.SchemaVersion && a.Resource == b.Resource && a.OperationID == b.OperationID && a.Intent == b.Intent
}

// Prepare precedes role-intent admission and performs no process effect. The
// resource wrapper must additionally verify the admitted role in the update
// ledger; this file alone never authorizes execution.
func (s *managedUpdateExecStore) Prepare(ctx context.Context, initial managedUpdateExecFence) (managedUpdateExecFence, error) {
	if err := s.validateIdentity(initial); err != nil {
		return managedUpdateExecFence{}, err
	}
	if initial.State != managedUpdateExecFencePrepared {
		return managedUpdateExecFence{}, fmt.Errorf("resource exec preparation requires an untouched valid identity")
	}
	release, err := s.lock(ctx)
	if err != nil {
		return managedUpdateExecFence{}, err
	}
	defer release()
	if err := ctx.Err(); err != nil {
		return managedUpdateExecFence{}, err
	}
	old, err := s.read()
	if err == nil {
		if !sameManagedExecIdentity(old, initial) {
			return managedUpdateExecFence{}, ErrManagedUpdateConflict
		}
		if old.State == managedUpdateExecFenceClaimed {
			return managedUpdateExecFence{}, ErrManagedUpdateExecClaimed
		}
		if old.State == managedUpdateExecFenceRevoked {
			return managedUpdateExecFence{}, ErrManagedUpdateExecRevoked
		}
		if err := s.save(old); err != nil {
			return managedUpdateExecFence{}, err
		}
		return old, nil
	}
	if !os.IsNotExist(err) {
		return managedUpdateExecFence{}, err
	}
	if err := s.save(initial); err != nil {
		return managedUpdateExecFence{}, err
	}
	return initial, nil
}

func (s *managedUpdateExecStore) Claim(ctx context.Context, expected managedUpdateExecFence, ref platform.ServiceRef) (managedUpdateExecFence, error) {
	return s.mutate(ctx, expected, &ref)
}

func (s *managedUpdateExecStore) Revoke(ctx context.Context, expected managedUpdateExecFence) (managedUpdateExecFence, error) {
	return s.mutate(ctx, expected, nil)
}

func (s *managedUpdateExecStore) mutate(ctx context.Context, expected managedUpdateExecFence, claim *platform.ServiceRef) (managedUpdateExecFence, error) {
	if err := s.validateIdentity(expected); err != nil {
		return managedUpdateExecFence{}, err
	}
	release, err := s.lock(ctx)
	if err != nil {
		return managedUpdateExecFence{}, err
	}
	defer release()
	if err := ctx.Err(); err != nil {
		return managedUpdateExecFence{}, err
	}
	f, err := s.read()
	if os.IsNotExist(err) && claim == nil {
		// Recreate only a revoked tombstone, never missing exec permission.
		f, err = expected, nil
	}
	if err != nil {
		return managedUpdateExecFence{}, err
	}
	if !sameManagedExecIdentity(f, expected) {
		return managedUpdateExecFence{}, ErrManagedUpdateConflict
	}
	at := s.now().UTC()
	if claim != nil {
		if f.State == managedUpdateExecFenceRevoked {
			// Retain one exact rejected native delivery for recovery diagnostics.
			// This never restores exec permission and is bounded independently of
			// repeated callers. Journald delivery is not the admission receipt.
			if f.RejectedClaim == nil {
				copy := *claim
				f.RejectedClaim, f.RejectedAt = &copy, &at
				if err := s.save(f); err != nil {
					return managedUpdateExecFence{}, err
				}
			}
			return managedUpdateExecFence{}, ErrManagedUpdateExecRevoked
		}
		if f.State == managedUpdateExecFenceClaimed {
			return managedUpdateExecFence{}, ErrManagedUpdateExecClaimed
		}
		copy := *claim
		f.State, f.Claim, f.ClaimedAt = managedUpdateExecFenceClaimed, &copy, &at
	} else {
		if f.RevokedAt == nil {
			f.RevokedAt = &at
		}
		f.State = managedUpdateExecFenceRevoked
	}
	if err := s.save(f); err != nil {
		return managedUpdateExecFence{}, err
	}
	return f, nil
}
