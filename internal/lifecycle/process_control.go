package lifecycle

import "github.com/vrooli/platform-go"

func signalProcessGroup(groupID int, force bool) error {
	return platform.SignalProcessGroup(groupID, force)
}

func signalPID(pid int, force bool) error {
	return platform.SignalPID(pid, force)
}
