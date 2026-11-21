// Responsibility: bind a relay to a live Linux kernel process, not a recycled PID.
//go:build linux

package programs

import "golang.org/x/sys/unix"

func openKernelProcess(pid int) (int, error) { return unix.PidfdOpen(pid, 0) }

func kernelProcessAlive(fd int) bool {
	if fd < 0 {
		return false
	}
	fds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
	n, err := unix.Poll(fds, 0)
	return err == nil && n == 0
}

func closeKernelProcess(fd int) {
	if fd >= 0 {
		_ = unix.Close(fd)
	}
}
