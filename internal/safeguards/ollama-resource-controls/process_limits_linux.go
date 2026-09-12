//go:build linux

package ollamaresourcecontrols

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func readProcessLimits(pid int) (uint64, uint64, error) {
	var limit unix.Rlimit
	if err := unix.Prlimit(pid, unix.RLIMIT_AS, nil, &limit); err != nil {
		return 0, 0, fmt.Errorf("read address-space limit: %w", err)
	}
	return limit.Cur, limit.Max, nil
}
