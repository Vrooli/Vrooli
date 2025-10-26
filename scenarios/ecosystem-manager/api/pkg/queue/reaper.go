package queue

import (
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

// InitProcessReaper sets up SIGCHLD handler to automatically reap zombie processes
// This prevents zombie processes from accumulating when child processes exit
func InitProcessReaper() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGCHLD)

	go func() {
		for range sigChan {
			// Reap all available zombie children
			for {
				var status syscall.WaitStatus
				pid, err := syscall.Wait4(-1, &status, syscall.WNOHANG, nil)
				if err != nil || pid <= 0 {
					break
				}
				log.Printf("Reaped zombie process: PID %d, exit status: %d", pid, status.ExitStatus())
			}
		}
	}()

	log.Println("Process reaper initialized")
}

// SetProcessGroup sets the process to run in its own process group
// This allows us to kill the entire process tree when needed
func SetProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

// KillProcessGroup terminates an entire process group
func KillProcessGroup(pid int) error {
	// Send SIGTERM to the entire process group
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		return err
	}

	// Negative pgid means kill the entire process group
	if err := syscall.Kill(-pgid, syscall.SIGTERM); err != nil {
		return err
	}

	return nil
}
