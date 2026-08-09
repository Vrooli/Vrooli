package lifecycle

import (
	"os"

	"github.com/vrooli/platform-go"
)

func signalProcessGroup(groupID int, force bool) error {
	return platform.SignalProcessGroup(groupID, force)
}

func signalPID(pid int, force bool) error {
	return platform.SignalPID(pid, force)
}

func reraiseSignal(signal os.Signal) error {
	return platform.ReraiseSignal(signal)
}
