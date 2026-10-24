//go:build !linux

package ollamaresourcecontrols

import "fmt"

// The safeguard is intentionally Linux-only. Keeping a typed implementation
// for other targets lets the control plane and node artifacts cross-compile;
// Inspect rejects those targets before this function is reached.
func readProcessLimits(_ int) (uint64, uint64, error) {
	return 0, 0, fmt.Errorf("process resource limits require the Linux managed-service supervisor")
}
