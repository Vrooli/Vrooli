package infra

import (
	"os"
	"os/exec"

	platform "github.com/vrooli/platform-go"
)

func configureProcessCommand(cmd *exec.Cmd) {
	_ = platform.ConfigureCommand(cmd, platform.ProcessOptions{Detached: true})
}

func assignProcessContainment(process *os.Process) (func(), error) {
	return platform.AssignProcessContainment(process)
}

func gracefulStopProcess(process *os.Process) error {
	return platform.GracefulStopProcess(process)
}
