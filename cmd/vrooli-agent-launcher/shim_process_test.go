//go:build !windows

// These tests cover the seam that had no coverage before: the real stdio path,
// where the agent inherits this process's own file descriptors. Every other
// test in the launcher stack substitutes a RunChild seam or a bytes.Buffer,
// which exercises Go's pipe path and can never observe terminal handling or
// process-tree shape.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// unreachableAgentManager is a port nothing listens on, so the attribution call
// fails fast and these tests never create real run records.
const unreachableAgentManager = "http://127.0.0.1:9"

// shimFixture builds the launcher, publishes it under an agent alias, and
// installs a stub agent further along PATH — the exact layout the installer
// produces on a real host.
type shimFixture struct {
	shim    string // the alias link the caller invokes
	pathEnv string
}

func newShimFixture(t *testing.T, agentScript string) shimFixture {
	t.Helper()

	buildDir := t.TempDir()
	launcher := filepath.Join(buildDir, "vrooli-agent-launcher")
	build := exec.Command("go", "build", "-o", launcher, ".")
	build.Env = append(os.Environ(), "GOWORK=off")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build launcher: %v\n%s", err, output)
	}

	shimDir := t.TempDir()
	shim := filepath.Join(shimDir, "codex")
	if err := os.Symlink(launcher, shim); err != nil {
		t.Fatalf("link shim: %v", err)
	}

	agentDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(agentDir, "codex"), []byte(agentScript), 0o755); err != nil {
		t.Fatalf("write stub agent: %v", err)
	}

	return shimFixture{
		shim: shim,
		pathEnv: "PATH=" + shimDir + string(os.PathListSeparator) + agentDir +
			string(os.PathListSeparator) + os.Getenv("PATH"),
	}
}

func (f shimFixture) env() []string {
	return append(os.Environ(),
		f.pathEnv,
		"AGENT_MANAGER_API_BASE="+unreachableAgentManager,
		"AGENT_MANAGER_ATTACH_TIMEOUT=50ms",
	)
}

// TestShimReplacesItselfWithTheAgent is the load-bearing assertion for the
// whole design: after exec, the agent must be running under the very pid the
// caller started. If the launcher forked instead, the reported pid would be a
// child's and the launcher would still be sitting in the process tree holding a
// terminal it has to restore later.
func TestShimReplacesItselfWithTheAgent(t *testing.T) {
	fixture := newShimFixture(t, "#!/bin/sh\nprintf 'AGENT_PID=%s\\n' \"$$\"\n")

	command := exec.Command(fixture.shim)
	command.Env = fixture.env()
	output, err := command.Output()
	if err != nil {
		t.Fatalf("run shim: %v", err)
	}

	reported := parseAgentPID(t, string(output))
	if reported != command.Process.Pid {
		t.Fatalf("agent ran as pid %d but the caller started pid %d — the launcher forked instead of exec'ing, so it is still interposed on the agent",
			reported, command.Process.Pid)
	}
}

// TestShimPropagatesAgentExitCode guards the other half of transparency: an
// operator's scripts must see the agent's status, not the launcher's.
func TestShimPropagatesAgentExitCode(t *testing.T) {
	fixture := newShimFixture(t, "#!/bin/sh\nexit 42\n")

	command := exec.Command(fixture.shim)
	command.Env = fixture.env()
	err := command.Run()

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected an exit error, got %v", err)
	}
	if exitErr.ExitCode() != 42 {
		t.Fatalf("exit code = %d, want 42", exitErr.ExitCode())
	}
}

// TestShimPassesAgentArgumentsThrough checks that arguments reach the agent
// unaltered and are never re-parsed as shell text.
func TestShimPassesAgentArgumentsThrough(t *testing.T) {
	fixture := newShimFixture(t, "#!/bin/sh\nfor a in \"$@\"; do printf 'ARG=[%s]\\n' \"$a\"; done\n")

	command := exec.Command(fixture.shim, "--yolo", "a b", "$(echo hi)")
	command.Env = fixture.env()
	output, err := command.Output()
	if err != nil {
		t.Fatalf("run shim: %v", err)
	}
	want := "ARG=[--yolo]\nARG=[a b]\nARG=[$(echo hi)]\n"
	if string(output) != want {
		t.Fatalf("arguments = %q, want %q", output, want)
	}
}

// TestShimLeavesTerminalUsableOnExit runs the agent on a real controlling
// terminal and asserts the session ends cleanly with the line discipline it
// started with.
//
// This is the regression that motivated the change. When a Node-wrapped agent
// exits on a terminal it can no longer restore, it aborts rather than exiting,
// and the operator loses the session. Any launcher change that disturbs
// terminal ownership shows up here as a non-zero status or altered termios.
func TestShimLeavesTerminalUsableOnExit(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("pty allocation here uses Linux ioctls")
	}
	fixture := newShimFixture(t, "#!/bin/sh\nprintf 'AGENT_PID=%s\\n' \"$$\"\nstty -a >/dev/null 2>&1\n")

	master, slave, err := openPTY()
	if err != nil {
		t.Skipf("cannot allocate a pty here: %v", err)
	}
	defer master.Close()

	before, err := unix.IoctlGetTermios(int(master.Fd()), unix.TCGETS)
	if err != nil {
		t.Fatalf("read termios: %v", err)
	}

	command := exec.Command(fixture.shim)
	command.Env = fixture.env()
	command.Stdin, command.Stdout, command.Stderr = slave, slave, slave
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true}
	if err := command.Start(); err != nil {
		t.Fatalf("start on pty: %v", err)
	}
	launched := command.Process.Pid
	slave.Close()

	transcript := drainPTY(master)
	if err := command.Wait(); err != nil {
		t.Fatalf("agent did not exit cleanly on a terminal: %v\ntranscript:\n%s", err, transcript)
	}

	if reported := parseAgentPID(t, transcript); reported != launched {
		t.Fatalf("agent ran as pid %d, want the launched pid %d", reported, launched)
	}

	after, err := unix.IoctlGetTermios(int(master.Fd()), unix.TCGETS)
	if err != nil {
		t.Fatalf("re-read termios: %v", err)
	}
	if *before != *after {
		t.Fatalf("terminal settings changed across the session:\nbefore %+v\nafter  %+v", *before, *after)
	}
}

// openPTY allocates a pty pair without pulling in a dependency for four ioctls.
func openPTY() (*os.File, *os.File, error) {
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		return nil, nil, err
	}
	if err := unix.IoctlSetPointerInt(int(master.Fd()), unix.TIOCSPTLCK, 0); err != nil {
		master.Close()
		return nil, nil, fmt.Errorf("unlockpt: %w", err)
	}
	index, err := unix.IoctlGetInt(int(master.Fd()), unix.TIOCGPTN)
	if err != nil {
		master.Close()
		return nil, nil, fmt.Errorf("ptsname: %w", err)
	}
	slave, err := os.OpenFile("/dev/pts/"+strconv.Itoa(index), os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		master.Close()
		return nil, nil, err
	}
	return master, slave, nil
}

// drainPTY reads until the slave side is gone. A pty master reports EIO rather
// than EOF once the last slave closes, so that is the terminating condition.
func drainPTY(master *os.File) string {
	_ = master.SetReadDeadline(time.Now().Add(30 * time.Second))
	var collected strings.Builder
	buffer := make([]byte, 4096)
	for {
		n, err := master.Read(buffer)
		collected.Write(buffer[:n])
		if err != nil {
			// A pty master reports EIO, not EOF, once the last slave closes.
			return collected.String()
		}
	}
}

func parseAgentPID(t *testing.T, transcript string) int {
	t.Helper()
	for _, line := range strings.Split(transcript, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "AGENT_PID=") {
			continue
		}
		pid, err := strconv.Atoi(strings.TrimPrefix(line, "AGENT_PID="))
		if err != nil {
			t.Fatalf("unparseable pid line %q: %v", line, err)
		}
		return pid
	}
	t.Fatalf("stub agent never reported its pid; transcript:\n%s", transcript)
	return 0
}
