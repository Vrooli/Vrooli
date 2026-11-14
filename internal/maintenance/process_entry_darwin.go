//go:build darwin

package maintenance

import (
	"strconv"

	"github.com/vrooli/vrooli/internal/shell"
)

// readProcessEntry shells out to ps on darwin (no /proc). This path only runs
// on the small-N orphan kill-guard, so one fork per re-validated PID is
// acceptable.
func readProcessEntry(pid int) (processTableEntry, bool) {
	if pid <= 0 {
		return processTableEntry{}, false
	}
	output, err := shell.Output(shell.Spec{
		Name: "ps",
		Args: []string{"-p", strconv.Itoa(pid), "-o", "pid=,ppid=,pgid=,sid=,state=,command="},
	})
	if err != nil {
		// ps returns a non-zero exit when the PID doesn't exist — the caller
		// should treat that as "process gone".
		return processTableEntry{}, false
	}
	entry, ok := parseProcessTableLine(string(output))
	if !ok || entry.PID != pid {
		return processTableEntry{}, false
	}
	return entry, true
}
