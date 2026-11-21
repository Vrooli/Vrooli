package volumeremediation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/artifactlock"
)

// AcquireOperationLease serializes cooperating operations for one physical
// volume. The identity must include a UUID or serial; device paths alone are
// unstable across re-enumeration and are never used as a lock identity.
// Observe is called both before and after lock acquisition so a swapped device
// cannot inherit a lease while queued.
func AcquireOperationLease(ctx context.Context, expected Device, observe func(context.Context, string) (State, error)) (func(), error) {
	if ctx == nil || observe == nil {
		return nil, fmt.Errorf("volume lease requires context and identity observer")
	}
	if strings.TrimSpace(expected.Path) == "" || !expected.StableIdentity() {
		return nil, fmt.Errorf("volume lease requires a device path and stable UUID or serial")
	}
	verify := func() error {
		state, err := observe(ctx, expected.Path)
		if err != nil {
			return fmt.Errorf("observe volume for lease: %w", err)
		}
		if !state.Device.StableIdentity() || !leaseIdentityMatches(expected, state.Device) {
			return ErrDeviceChanged
		}
		return nil
	}
	if err := verify(); err != nil {
		return nil, err
	}
	identity := strings.ToLower(strings.TrimSpace(expected.UUID)) + "\x00" + strings.TrimSpace(expected.Serial)
	sum := sha256.Sum256([]byte(identity))
	release, err := artifactlock.AcquireContext(ctx, "host-volume-operation:"+hex.EncodeToString(sum[:]))
	if err != nil {
		return nil, err
	}
	if err := verify(); err != nil {
		release()
		return nil, err
	}
	return release, nil
}

func leaseIdentityMatches(expected, observed Device) bool {
	if strings.TrimSpace(expected.Path) != strings.TrimSpace(observed.Path) {
		return false
	}
	if strings.TrimSpace(expected.Filesystem) != "" && !sameFilesystemIdentity(expected.Filesystem, observed.Filesystem) {
		return false
	}
	if strings.TrimSpace(expected.UUID) != "" && !equalFoldOptional(expected.UUID, observed.UUID) {
		return false
	}
	if strings.TrimSpace(expected.Serial) != "" && !equalOptional(expected.Serial, observed.Serial) {
		return false
	}
	return expected.StableIdentity()
}
