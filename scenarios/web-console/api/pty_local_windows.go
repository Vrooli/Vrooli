//go:build windows

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"unsafe"

	"web-console/backends/codex"
	"web-console/backends/grok"
	"web-console/internal/config"
	"web-console/internal/pty"
	"web-console/session"

	"golang.org/x/sys/windows"
)

var errWindowsPTYClosed = errors.New("Windows ConPTY is closed")

func localPTYAvailable() bool { return true }

// conptyPTY is the host side of a Windows pseudo console. ConPTY owns the
// child console and exposes two anonymous pipes: host writes to input and
// reads from output. This gives the session package the same PTY contract as
// Unix without pretending that ordinary stdout pipes are interactive.
type conptyPTY struct {
	input   *os.File
	output  *os.File
	process windows.Handle
	console windows.Handle
	cwd     string
	mu      sync.Mutex
	closed  bool
}

// realPTY remains the package-main name used by the shared test seams.
type realPTY = conptyPTY

func (p *conptyPTY) Read(buf []byte) (int, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return 0, io.EOF
	}
	out := p.output
	p.mu.Unlock()
	return out.Read(buf)
}

func (p *conptyPTY) WriteInput(data []byte, _ pty.InputKind) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return errWindowsPTYClosed
	}
	in := p.input
	p.mu.Unlock()
	_, err := in.Write(data)
	if err != nil {
		return fmt.Errorf("ConPTY input: %w", err)
	}
	return nil
}

func (p *conptyPTY) SetSize(cols, rows uint16) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return errWindowsPTYClosed
	}
	console := p.console
	p.mu.Unlock()
	return windows.ResizePseudoConsole(console, windows.Coord{X: int16(cols), Y: int16(rows)})
}

func (p *conptyPTY) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	in, out, process, console := p.input, p.output, p.process, p.console
	p.mu.Unlock()
	if in != nil {
		_ = in.Close()
	}
	if out != nil {
		_ = out.Close()
	}
	if process != 0 {
		_ = windows.CloseHandle(process)
	}
	if console != 0 {
		windows.ClosePseudoConsole(console)
	}
	return nil
}

func (p *conptyPTY) Kill() error {
	p.mu.Lock()
	process := p.process
	p.mu.Unlock()
	if process == 0 {
		return nil
	}
	if err := windows.TerminateProcess(process, 1); err != nil && err != windows.ERROR_INVALID_HANDLE {
		return err
	}
	return nil
}

func (p *conptyPTY) ExitCode() int {
	p.mu.Lock()
	process := p.process
	p.mu.Unlock()
	if process == 0 {
		return -1
	}
	var code uint32
	if err := windows.GetExitCodeProcess(process, &code); err != nil || code == 259 {
		return -1
	}
	return int(code)
}

func (p *conptyPTY) ProbeReady(context.Context) error { return nil }

func (p *conptyPTY) CurrentDir(context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.cwd, nil
}

func (p *conptyPTY) TerminalEchoState() (session.EchoState, error) {
	return session.EchoState{}, session.ErrEchoStateUnsupported
}

// defaultPTYFactory starts the configured shell inside a native Windows
// pseudo console. The process attribute is the documented ConPTY handoff;
// no pipe fallback is used because a pipe is not a terminal.
func defaultPTYFactory(spec pty.LaunchSpec) (pty.PTY, error) {
	var inputRead, inputWrite windows.Handle
	var outputRead, outputWrite windows.Handle
	if err := windows.CreatePipe(&inputRead, &inputWrite, nil, 0); err != nil {
		return nil, fmt.Errorf("create ConPTY input pipe: %w", err)
	}
	if err := windows.CreatePipe(&outputRead, &outputWrite, nil, 0); err != nil {
		_ = windows.CloseHandle(inputRead)
		_ = windows.CloseHandle(inputWrite)
		return nil, fmt.Errorf("create ConPTY output pipe: %w", err)
	}
	cleanup := func() {
		_ = windows.CloseHandle(inputRead)
		_ = windows.CloseHandle(inputWrite)
		_ = windows.CloseHandle(outputRead)
		_ = windows.CloseHandle(outputWrite)
	}

	var console windows.Handle
	if err := windows.CreatePseudoConsole(
		windows.Coord{X: int16(spec.Cols), Y: int16(spec.Rows)},
		inputRead, outputWrite, 0, &console,
	); err != nil {
		cleanup()
		return nil, fmt.Errorf("create ConPTY: %w", err)
	}
	// ConPTY has taken ownership of these endpoints. The host retains the
	// opposite sides for session I/O.
	_ = windows.CloseHandle(inputRead)
	_ = windows.CloseHandle(outputWrite)

	attrs, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		windows.ClosePseudoConsole(console)
		_ = windows.CloseHandle(inputWrite)
		_ = windows.CloseHandle(outputRead)
		return nil, fmt.Errorf("create ConPTY process attributes: %w", err)
	}
	defer attrs.Delete()
	if err := attrs.Update(windows.PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE, unsafe.Pointer(&console), unsafe.Sizeof(console)); err != nil {
		windows.ClosePseudoConsole(console)
		_ = windows.CloseHandle(inputWrite)
		_ = windows.CloseHandle(outputRead)
		return nil, fmt.Errorf("attach ConPTY process attribute: %w", err)
	}

	commandLine, err := windows.UTF16FromString(windows.EscapeArg(spec.Shell))
	if err != nil {
		windows.ClosePseudoConsole(console)
		_ = windows.CloseHandle(inputWrite)
		_ = windows.CloseHandle(outputRead)
		return nil, fmt.Errorf("encode shell command: %w", err)
	}
	envBlock, err := windows.UTF16FromString(strings.Join(buildSessionEnv(spec), "\x00") + "\x00")
	if err != nil {
		windows.ClosePseudoConsole(console)
		_ = windows.CloseHandle(inputWrite)
		_ = windows.CloseHandle(outputRead)
		return nil, fmt.Errorf("encode shell environment: %w", err)
	}
	cwd := resolveLaunchDir(spec)
	cwdPtr, err := windows.UTF16PtrFromString(cwd)
	if err != nil {
		windows.ClosePseudoConsole(console)
		_ = windows.CloseHandle(inputWrite)
		_ = windows.CloseHandle(outputRead)
		return nil, fmt.Errorf("encode shell working directory: %w", err)
	}

	startup := windows.StartupInfoEx{}
	startup.StartupInfo.Cb = uint32(unsafe.Sizeof(startup))
	startup.StartupInfo.Flags = windows.STARTF_USESHOWWINDOW
	startup.StartupInfo.ShowWindow = windows.SW_HIDE
	startup.ProcThreadAttributeList = attrs.List()
	var processInfo windows.ProcessInformation
	if err := windows.CreateProcess(
		nil, &commandLine[0], nil, nil, false,
		windows.EXTENDED_STARTUPINFO_PRESENT|windows.CREATE_UNICODE_ENVIRONMENT|windows.CREATE_NO_WINDOW,
		&envBlock[0], cwdPtr, &startup.StartupInfo, &processInfo,
	); err != nil {
		windows.ClosePseudoConsole(console)
		_ = windows.CloseHandle(inputWrite)
		_ = windows.CloseHandle(outputRead)
		return nil, fmt.Errorf("start ConPTY shell: %w", err)
	}
	_ = windows.CloseHandle(processInfo.Thread)

	return &conptyPTY{
		input:   os.NewFile(uintptr(inputWrite), "web-console-conpty-input"),
		output:  os.NewFile(uintptr(outputRead), "web-console-conpty-output"),
		process: processInfo.Process,
		console: console,
		cwd:     cwd,
	}, nil
}

// These helpers are referenced by shared environment and lifecycle code. The
// imports above deliberately keep this file self-contained on Windows; the
// native implementation can replace this file without changing its callers.
func resolveLaunchDir(spec pty.LaunchSpec) string {
	if spec.WorkingDir != "" {
		return spec.WorkingDir
	}
	return config.ResolveWorkingDir()
}

func exitCodeOnce(*exec.Cmd) int { return -1 }

func ensureTermEnv(env []string) []string {
	const term = "TERM=xterm-256color"
	for i, entry := range env {
		if len(entry) >= 5 && entry[:5] == "TERM=" {
			env[i] = term
			return env
		}
	}
	return append(env, term)
}

func filterServiceEnv(env []string) []string { return env }

func applySessionEnv(base []string, extra map[string]string) []string {
	result := append([]string(nil), base...)
	for name, value := range extra {
		found := false
		for i, entry := range result {
			if len(entry) > len(name) && entry[:len(name)] == name && entry[len(name)] == '=' {
				result[i] = name + "=" + value
				found = true
				break
			}
		}
		if !found {
			result = append(result, name+"="+value)
		}
	}
	return result
}

func sessionCodexHome(sessionID string) string {
	return codex.SessionHomePath(resolveSessionStateRoot(), sessionID)
}

func sessionCodexSessionsDir(sessionID string) string {
	return codex.SessionsDirPath(resolveSessionStateRoot(), sessionID)
}

func sessionGrokHome(sessionID string) string {
	return grok.SessionHomePath(resolveSessionStateRoot(), sessionID)
}

func sessionGrokSessionsDir(sessionID string) string {
	return grok.SessionsDirPath(resolveSessionStateRoot(), sessionID)
}

func buildSessionEnv(spec pty.LaunchSpec) []string {
	return applySessionEnv(ensureTermEnv(filterServiceEnv(os.Environ())), spec.Env)
}
