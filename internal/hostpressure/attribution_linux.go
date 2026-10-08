package hostpressure

import (
	"os"
	"strconv"
	"strings"

	platformgo "github.com/vrooli/platform-go"
)

// parentScope reads the parent's cgroup path only when the pid is alive and
// still carries the snapshot's name, so a fixture's or a recycled pid never
// borrows a live process's scope.
func parentScope(pid int64, name string) string {
	comm, err := os.ReadFile("/proc/" + strconv.FormatInt(pid, 10) + "/comm")
	if err != nil || strings.TrimSpace(string(comm)) != name {
		return ""
	}
	scope, err := platformgo.ProcessScope(int(pid))
	if err != nil {
		return ""
	}
	return scope
}
