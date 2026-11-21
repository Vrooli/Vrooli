package home

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Process is one running Codex process and the Codex home it uses.
type Process struct {
	PID       int
	CodexHome string
	// AppServer is true for the background app-server daemon, which outlives
	// the terminal window and does not mean a supervisor is open.
	AppServer bool
	// WebConsoleSession is WC_WEB_CONSOLE_SESSION_ID, which Web Console sets
	// in its session's terminal; every child inherits it.
	WebConsoleSession string
	// InTerminal is true when the process descends from the terminal of a
	// Web Console session (its TMUX server is an ancestor). A process that
	// inherited the session's environment but was detached from it is not.
	InTerminal bool
}

// Procs lists running Codex processes. Known is false when the process
// table cannot be read (another OS, or /proc hidden); window state is then
// unknown rather than "closed".
type Procs interface {
	Codex() (processes []Process, known bool, reason string)
}

// FixedProcs is a process table given in advance, always known: the tests'
// table, or a reader that must not look at /proc.
type FixedProcs []Process

// Codex implements Procs.
func (p FixedProcs) Codex() ([]Process, bool, string) { return p, true, "" }

// ProcTable reads /proc. Only processes whose command line names codex are
// inspected; their environment gives CODEX_HOME.
type ProcTable struct {
	Root string // "/proc" in production
}

// Codex implements Procs.
func (p ProcTable) Codex() ([]Process, bool, string) {
	root := p.Root
	if root == "" {
		root = "/proc"
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, false, "process table unreadable: " + err.Error()
	}
	var out []Process
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		cmdline, err := os.ReadFile(filepath.Join(root, entry.Name(), "cmdline"))
		if err != nil || !bytes.Contains(cmdline, []byte("codex")) {
			continue
		}
		environ, err := os.ReadFile(filepath.Join(root, entry.Name(), "environ"))
		if err != nil {
			continue
		}
		process := Process{PID: pid, AppServer: bytes.Contains(cmdline, []byte("app-server"))}
		tmuxServer := 0
		for _, kv := range bytes.Split(environ, []byte{0}) {
			name, value, _ := strings.Cut(string(kv), "=")
			switch name {
			case "CODEX_HOME":
				process.CodexHome = filepath.Clean(value)
			case "WC_WEB_CONSOLE_SESSION_ID":
				process.WebConsoleSession = value
			case "TMUX":
				// <socket>,<server pid>,<session>
				if parts := strings.Split(value, ","); len(parts) == 3 {
					tmuxServer, _ = strconv.Atoi(parts[1])
				}
			}
		}
		if process.CodexHome == "" {
			continue
		}
		process.InTerminal = tmuxServer > 0 && descends(root, pid, tmuxServer)
		out = append(out, process)
	}
	return out, true, ""
}

// descends reports whether ancestor is a parent, grandparent, … of pid.
func descends(root string, pid, ancestor int) bool {
	for range 64 {
		parent, ok := parentOf(root, pid)
		if !ok || parent <= 1 {
			return false
		}
		if parent == ancestor {
			return true
		}
		pid = parent
	}
	return false
}

// parentOf reads the parent pid from /proc/<pid>/stat. The command name in
// field 2 may hold spaces, so fields are counted after its closing paren.
func parentOf(root string, pid int) (int, bool) {
	stat, err := os.ReadFile(filepath.Join(root, strconv.Itoa(pid), "stat"))
	if err != nil {
		return 0, false
	}
	end := bytes.LastIndexByte(stat, ')')
	if end < 0 {
		return 0, false
	}
	fields := strings.Fields(string(stat[end+1:]))
	if len(fields) < 2 {
		return 0, false
	}
	parent, err := strconv.Atoi(fields[1])
	return parent, err == nil
}

// windowsOpen returns the Codex homes with a terminal window (a non
// app-server process) running.
func windowsOpen(processes []Process) map[string]bool {
	open := map[string]bool{}
	for _, process := range processes {
		if !process.AppServer {
			open[process.CodexHome] = true
		}
	}
	return open
}
