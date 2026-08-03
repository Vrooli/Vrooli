//go:build !(aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris)

package orchestration

import "os"

func gracefulTerminateProcess(process *os.Process) bool {
	return process.Kill() == nil
}

func processGroupID(_ int) int { return 0 }

func killProcessGroupID(_ int) bool { return false }
