package repocontract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync/atomic"
)

const maxRuntimeHomeSnapshotBytes = 64 << 10

type runtimeHomeSnapshot struct {
	Version     int             `json:"version"`
	RuntimeHome RuntimeHomeSpec `json:"runtime_home"`
}

var pinnedRuntimeHome atomic.Pointer[Contract]

// ValidateRuntimeHomeSnapshot checks build metadata without pinning process
// authority or reading source or runtime files.
func ValidateRuntimeHomeSnapshot(data []byte) error {
	_, err := decodeRuntimeHomeSnapshot(data)
	return err
}

// RuntimeHomeSnapshotEntry resolves a compiled entry without pinning global
// authority or reading a source checkout. This supports pre-initialization
// fences which must inspect state before any ordinary command side effects.
func RuntimeHomeSnapshotEntry(data []byte, home, key string) (HomeEntry, error) {
	c, err := decodeRuntimeHomeSnapshot(data)
	if err != nil {
		return HomeEntry{}, err
	}
	return c.RuntimeHomeEntry(home, key)
}

// RuntimeHomeSnapshot exports the validated runtime-home authority for a
// governed build. It contains structure and retention policy, never runtime
// values, credentials, a selected user home or source-discovery overrides.
func (c *Contract) RuntimeHomeSnapshot() ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("repo-contract: runtime-home authority is required")
	}
	if err := validateRuntimeHome(c.doc.RuntimeHome); err != nil {
		return nil, err
	}
	data, err := json.Marshal(runtimeHomeSnapshot{Version: 1, RuntimeHome: cloneRuntimeHomeSpec(c.doc.RuntimeHome)})
	if err != nil {
		return nil, err
	}
	if len(data) > maxRuntimeHomeSnapshotBytes {
		return nil, fmt.Errorf("repo-contract: runtime-home snapshot exceeds its bound")
	}
	return data, nil
}

// PinRuntimeHomeSnapshot selects the immutable authority embedded by a native
// artifact's build owner. It affects only runtime-home path helpers, not repo
// discovery or source operations. A process may repeat the same pin but cannot
// replace it with another layout. Invalid input never falls back to source.
// Call this before starting native workers or resolving any runtime-home path.
func PinRuntimeHomeSnapshot(data []byte) error {
	next, err := decodeRuntimeHomeSnapshot(data)
	if err != nil {
		return err
	}
	for {
		previous := pinnedRuntimeHome.Load()
		if previous != nil {
			prior, err := previous.RuntimeHomeSnapshot()
			if err != nil {
				return err
			}
			if !bytes.Equal(prior, data) {
				return fmt.Errorf("repo-contract: runtime-home authority is already pinned to another snapshot")
			}
			return nil
		}
		if pinnedRuntimeHome.CompareAndSwap(nil, next) {
			return nil
		}
	}
}

func decodeRuntimeHomeSnapshot(data []byte) (*Contract, error) {
	if len(data) == 0 || len(data) > maxRuntimeHomeSnapshotBytes {
		return nil, fmt.Errorf("repo-contract: invalid bounded runtime-home snapshot")
	}
	var snapshot runtimeHomeSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil || snapshot.Version != 1 {
		return nil, fmt.Errorf("repo-contract: invalid runtime-home snapshot version or encoding")
	}
	// Only the canonical encoding emitted by the build owner is accepted.
	// Re-encoding rejects unknown/duplicate/case-alias fields recursively,
	// trailing values and ambiguous representations without a second schema.
	canonical, err := json.Marshal(snapshot)
	if err != nil || !bytes.Equal(canonical, data) {
		return nil, fmt.Errorf("repo-contract: runtime-home snapshot is not canonical")
	}
	if err := validateRuntimeHome(snapshot.RuntimeHome); err != nil {
		return nil, err
	}
	return &Contract{doc: contractDoc{RuntimeHome: cloneRuntimeHomeSpec(snapshot.RuntimeHome)}}, nil
}
