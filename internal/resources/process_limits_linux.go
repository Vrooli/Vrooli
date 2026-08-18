//go:build linux

package resources

import (
	"fmt"
	"os"
	"strconv"

	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

// applyManagedServiceProcessLimits applies the declared controls that Linux can
// actually enforce on a bare PID.
//
// It deliberately does NOT translate memory_high_percent / memory_max_percent
// into an rlimit. Linux has no rlimit that bounds *resident* memory: RLIMIT_RSS
// has been a no-op since 2.6, and RLIMIT_AS / RLIMIT_DATA bound the virtual
// address space, which for reservation-heavy runtimes is many times RSS. An
// earlier version of this function set RLIMIT_AS to those percentages, which
// delivered none of the declared host protection (a process can hold 40 GiB of
// address space against 2 GiB resident) while breaking every GPU workload:
// llama.cpp reserves a 32 GiB CUDA VMM pool on first inference, so a
// 60%-of-RAM address-space cap failed that reservation with "CUDA error: out of
// memory" on an otherwise idle GPU, killing the model runner on every load.
//
// The memory percentages therefore remain unapplied here, by design and not
// silently: the ollama-resource-controls safeguard reports the gap in its notes
// so an operator sees an honest "not enforced" rather than a limit that is
// present but meaningless. Enforcing them as written requires placing each
// managed service in its own delegated cgroup v2 scope (memory.high /
// memory.max), which the supervisor cannot do while it launches processes into a
// shared session scope that already holds other processes.
func applyManagedServiceProcessLimits(pid int, limits *resourcedeployment.ProcessLimits) error {
	if limits == nil {
		return nil
	}
	if limits.MemoryHighPercent > 0 && limits.MemoryMaxPercent > 0 &&
		limits.MemoryHighPercent > limits.MemoryMaxPercent {
		return fmt.Errorf("memory high limit exceeds memory max limit")
	}
	if limits.OOMScoreAdjust != 0 {
		path := "/proc/" + strconv.Itoa(pid) + "/oom_score_adj"
		if err := os.WriteFile(path, []byte(strconv.Itoa(limits.OOMScoreAdjust)), 0o600); err != nil {
			return fmt.Errorf("set oom_score_adj: %w", err)
		}
	}
	return nil
}
