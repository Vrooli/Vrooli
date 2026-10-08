// finite_profile_contract.go reads the complete persisted native profile contract for protected callers.
package orchestration

import (
	"context"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/effortauthority"
)

// ReadFiniteProfileContract reads the complete persisted native owner object.
// Only the separately protected profile handler exposes this read; no update,
// EnsureProfile, refresh or portable-projection recomputation occurs.
func (o *Orchestrator) ReadFiniteProfileContract(ctx context.Context, key string) (effortauthority.ProfileContract, error) {
	if o == nil || o.profiles == nil || key == "" {
		return effortauthority.ProfileContract{}, effortauthority.ErrRefused
	}
	p, e := o.profiles.GetByKey(ctx, key)
	if e != nil || p == nil || p.ProfileKey != key || p.ID == uuid.Nil {
		return effortauthority.ProfileContract{}, effortauthority.ErrRefused
	}
	return effortauthority.ProfileContract{ID: p.ID.String(), Key: p.ProfileKey, Digest: EffortProfileDigest(p)}, nil
}
