//go:build darwin

package accel

import (
	"fmt"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

// observeMetalProcess reads Metal placement on macOS. Every process on a Metal
// host that opened a GPU device holds an IOAccelerator region; the host
// inventory does not enumerate those regions today, so this reports the honest
// answer rather than a guess.
//
// Phase 13 of the accelerator plan adds the IOAccelerator region enumeration to
// internal/hostinventory. Until it lands, a darwin host reports unknown with a
// named reason and never reports a pass.
func observeMetalProcess(snapshot hostinventory.Snapshot, process HostProcess) (Backend, AccessState, string, error) {
	if len(snapshot.GPUs) == 0 {
		return BackendCPU, AccessUnknown, "the host enumerated no Metal-capable device", nil
	}
	return "", AccessUnknown, fmt.Sprintf("per-process Metal attribution is not yet readable from the host inventory, so placement of pid %d is unknown", process.PID), nil
}
