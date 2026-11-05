//go:build linux

package process

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

type processHandle struct{ fd int }

func openHandle(pid int) (Handle, error) {
	if pid <= 0 {
		return nil, fmt.Errorf("invalid process ID %d", pid)
	}
	fd, err := unix.PidfdOpen(pid, 0)
	if err != nil {
		return nil, fmt.Errorf("open stable handle for pid %d: %w", pid, err)
	}
	return &processHandle{fd: fd}, nil
}

func (h *processHandle) Alive() (bool, error) {
	if h.fd < 0 {
		return false, os.ErrClosed
	}
	fds := []unix.PollFd{{Fd: int32(h.fd), Events: unix.POLLIN}}
	for {
		_, err := unix.Poll(fds, 0)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if err != nil {
			return false, err
		}
		if fds[0].Revents&(unix.POLLNVAL|unix.POLLERR) != 0 {
			return false, fmt.Errorf("invalid process handle poll events: %d", fds[0].Revents)
		}
		return fds[0].Revents&(unix.POLLIN|unix.POLLHUP) == 0, nil
	}
}

func (h *processHandle) Signal(force bool) error {
	if h.fd < 0 {
		return os.ErrClosed
	}
	sig := unix.SIGTERM
	if force {
		sig = unix.SIGKILL
	}
	return unix.PidfdSendSignal(h.fd, sig, nil, 0)
}

func (h *processHandle) Close() error {
	if h.fd < 0 {
		return nil
	}
	fd := h.fd
	h.fd = -1
	return unix.Close(fd)
}
