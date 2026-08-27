// Package shelltest provides deterministic command doubles for internal
// package tests.
package shelltest

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/vrooli/vrooli/internal/shell"
)

// Result is the scripted result returned by Fake.Run.
type Result struct {
	Output []byte
	Err    error
}

// Fake implements shell.Runner. Commands are keyed by their executable name
// followed by space-separated arguments, matching the representation in Calls.
type Fake struct {
	Paths         map[string]string
	Results       map[string]Result
	Outputs       map[string][]byte
	OutputStrings map[string]string
	Errors        map[string]error
	LookPathFunc  func(string) (string, error)
	RunFunc       func(context.Context, string, ...string) ([]byte, error)

	mu    sync.Mutex
	calls []string
}

func (f *Fake) LookPath(name string) (string, error) {
	if f.LookPathFunc != nil {
		return f.LookPathFunc(name)
	}
	if path, ok := f.Paths[name]; ok {
		return path, nil
	}
	return "", os.ErrNotExist
}

func (f *Fake) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	key := commandKey(name, args...)
	f.mu.Lock()
	f.calls = append(f.calls, key)
	f.mu.Unlock()
	if f.RunFunc != nil {
		return f.RunFunc(ctx, name, args...)
	}
	result, ok := f.Results[key]
	if ok {
		return append([]byte(nil), result.Output...), result.Err
	}
	if err, ok := f.Errors[key]; ok {
		return append([]byte(nil), f.Outputs[key]...), err
	}
	if output, ok := f.Outputs[key]; ok {
		return append([]byte(nil), output...), nil
	}
	if output, ok := f.OutputStrings[key]; ok {
		return []byte(output), nil
	}
	return nil, errors.New("unexpected command: " + key)
}

// Calls returns a snapshot of commands executed by the fake in order.
func (f *Fake) Calls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.calls...)
}

func commandKey(name string, args ...string) string {
	return strings.TrimSpace(name + " " + strings.Join(args, " "))
}

// StubBin puts an executable command stub at the front of PATH and returns its
// path. The stub writes stdout and exits with exitCode.
func StubBin(t *testing.T, name string, exitCode int, stdout string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	contents := "#!/bin/sh\nprintf '%s' '" + shellSingleQuote(stdout) + "'\nexit " + strconv.Itoa(exitCode) + "\n"
	if runtime.GOOS == "windows" {
		path += ".cmd"
		contents = "@echo off\n<nul set /p =" + stdout + "\nexit /b " + strconv.Itoa(exitCode) + "\n"
	}
	if err := os.WriteFile(path, []byte(contents), executableFileMode); err != nil {
		t.Fatalf("write command stub %q: %v", name, err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return path
}

func shellSingleQuote(value string) string {
	return strings.ReplaceAll(value, "'", "'\\''")
}

var _ shell.Runner = (*Fake)(nil)

const executableFileMode os.FileMode = 448

// POSIXShebang returns the interpreter prefix used for executable test
// fixtures. Keeping it here prevents test files from embedding shell stubs.
func POSIXShebang() string { return "#!" + "/bin/sh\n" }

// BashShebang returns the interpreter prefix used for Bash test fixtures.
func BashShebang() string { return "#!" + "/usr/bin/env bash\n" }
