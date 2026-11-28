// Process-termination responsibility: delegate graceful shutdown and
// process-group cleanup to the platform lifecycle seam.
package orchestration

import (
	"os"

	platform "github.com/vrooli/platform-go"
)

func gracefulTerminateProcess(process *os.Process) bool {
	return requestGracefulProcessStop(process) == nil
}

func requestGracefulProcessStop(process *os.Process) error {
	return platform.GracefulStopProcess(process)
}

func processGroupID(pid int) int {
	group, err := platform.ProcessGroupID(pid)
	if err != nil {
		return 0
	}
	return group
}

func killProcessGroupID(pgid int) bool {
	return platform.SignalProcessGroup(pgid, true) == nil
}
