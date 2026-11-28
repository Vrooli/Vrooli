//go:build linux

package platform

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

const (
	// holdShell is the gate interpreter. POSIX requires it; the gate uses no
	// extension beyond `read`, `exec` and fd redirection.
	holdShell = "/bin/sh"
	// holdToken is what the parent writes to open the gate. Anything else,
	// including EOF when the parent dies, exits without running the target.
	holdToken = "exec"
	// holdAbortExit is the gate's exit status when it never ran the target.
	holdAbortExit = 125
	// holdVar is the shell variable the gate reads into.
	holdVar = "vrooli_exec_hold"
	// holdMaxFD is the highest descriptor a POSIX shell must redirect.
	holdMaxFD = 9
)

// holdShellManagedEnv are the variables a POSIX shell may create or rewrite
// at startup (dash sets PWD; bash also SHLVL and OLDPWD) plus the gate's own
// read variable. The gate restores each to the target's exact value or
// unsets it, so the target's environment is the one the caller configured.
var holdShellManagedEnv = []string{"PWD", "OLDPWD", "SHLVL", holdVar}

type linuxHold struct {
	gate *os.File
	// heldArgs is the gate's observed argv; a different argv is the witness
	// that the target has replaced the gate.
	heldArgs []string
}

func startHeld(cmd *exec.Cmd, start func(*exec.Cmd) error) (processHold, error) {
	if err := checkHeldTarget(cmd); err != nil {
		return nil, err
	}
	fd := 3 + len(cmd.ExtraFiles)
	if fd > holdMaxFD {
		return nil, fmt.Errorf("%w: %d extra files leave no shell-addressable gate descriptor", ErrHoldUnavailable, len(cmd.ExtraFiles))
	}
	if _, err := os.Stat(holdShell); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrHoldUnavailable, err)
	}
	env := cmd.Environ()
	read, write, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrHoldUnavailable, err)
	}
	target, targetArgs := cmd.Path, append([]string(nil), cmd.Args[1:]...)
	cmd.Path = holdShell
	cmd.Args = append([]string{holdShell, "-c", holdScript(fd, env), target}, targetArgs...)
	cmd.Env = env
	cmd.ExtraFiles = append(cmd.ExtraFiles, read)
	startErr := start(cmd)
	_ = read.Close()
	if startErr != nil {
		_ = write.Close()
		return nil, startErr
	}
	if cmd.Process == nil {
		_ = write.Close()
		return nil, errors.New("platform: start returned without a process")
	}
	// exec.Cmd.Start returns after the gate image is in place, so this argv is
	// stable until the gate opens.
	held, err := ProcessArguments(cmd.Process.Pid)
	if err != nil {
		held = nil
	}
	return &linuxHold{gate: write, heldArgs: held}, nil
}

// checkHeldTarget keeps exec failure synchronous: a direct cmd.Start reports
// a missing or non-executable target, but behind the gate it would only be the
// gate's exit status after Release.
func checkHeldTarget(cmd *exec.Cmd) error {
	path := cmd.Path
	if !filepath.IsAbs(path) && cmd.Dir != "" {
		path = filepath.Join(cmd.Dir, path)
	}
	info, err := os.Stat(path)
	if err == nil && info.IsDir() {
		err = unix.EACCES
	}
	if err == nil {
		err = unix.Access(path, unix.X_OK)
	}
	if err != nil {
		var pathErr *fs.PathError
		if errors.As(err, &pathErr) {
			err = pathErr.Err
		}
		return &fs.PathError{Op: "fork/exec", Path: cmd.Path, Err: err}
	}
	return nil
}

func holdScript(fd int, env []string) string {
	gate := strconv.Itoa(fd)
	parts := []string{
		"IFS= read -r " + holdVar + " <&" + gate + " || exit " + strconv.Itoa(holdAbortExit),
		`[ "$` + holdVar + `" = ` + holdToken + " ] || exit " + strconv.Itoa(holdAbortExit),
		"exec " + gate + "<&-",
	}
	for _, name := range holdShellManagedEnv {
		if value, ok := envValue(env, name); ok {
			parts = append(parts, name+"="+shellQuote(value), "export "+name)
		} else {
			parts = append(parts, "unset "+name)
		}
	}
	parts = append(parts, `exec "$0" "$@"`)
	return strings.Join(parts, "; ")
}

func envValue(env []string, name string) (string, bool) {
	value, found := "", false
	for _, entry := range env {
		if key, v, ok := strings.Cut(entry, "="); ok && key == name {
			value, found = v, true // the last entry wins, as in exec.Cmd
		}
	}
	return value, found
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func (h *linuxHold) abort() { _ = h.gate.Close() }

func (h *linuxHold) release(cmd *exec.Cmd, budget time.Duration) error {
	_, writeErr := h.gate.WriteString(holdToken + "\n")
	closeErr := h.gate.Close()
	if writeErr != nil {
		// The gate already exited; the caller's liveness checks report it.
		if errors.Is(writeErr, unix.EPIPE) {
			return nil
		}
		return fmt.Errorf("platform: open exec hold: %w", writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("platform: open exec hold: %w", closeErr)
	}
	if h.heldArgs == nil {
		return nil
	}
	pid := cmd.Process.Pid
	deadline := time.Now().Add(budget)
	pause := 200 * time.Microsecond
	for {
		args, err := ProcessArguments(pid)
		// An unreadable argv means the process exited (a zombie has none).
		if err != nil || !slices.Equal(args, h.heldArgs) {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("platform: held process %d did not execute its target within %s", pid, budget)
		}
		time.Sleep(pause)
		pause = min(pause*2, 20*time.Millisecond)
	}
}
