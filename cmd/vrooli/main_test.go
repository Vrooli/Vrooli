package main

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"
)

func usableWorkingDirSeams() workingDirSeams {
	return workingDirSeams{
		getwd:   func() (string, error) { return "/home/example/project", nil },
		stat:    func(string) (os.FileInfo, error) { return nil, nil },
		chdir:   func(string) error { return nil },
		homeDir: func() (string, error) { return "/home/example", nil },
		tempDir: func() string { return "/tmp" },
	}
}

func TestEnsureUsableWorkingDirectoryLeavesAReadableDirectoryAlone(t *testing.T) {
	var stderr bytes.Buffer
	chdirCalls := 0
	seams := usableWorkingDirSeams()
	seams.chdir = func(string) error { chdirCalls++; return nil }

	ensureUsableWorkingDirectoryWith(seams, &stderr)

	if chdirCalls != 0 {
		t.Fatalf("chdir calls = %d, want 0 when the working directory is usable", chdirCalls)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty when the working directory is usable", stderr.String())
	}
}

// A shell that dropped privileges while sitting in a directory it cannot read
// (the /root case) keeps a resolvable path but fails to stat it.
func TestEnsureUsableWorkingDirectoryRelocatesWhenCWDIsUnreadable(t *testing.T) {
	var stderr bytes.Buffer
	var entered []string
	seams := usableWorkingDirSeams()
	seams.stat = func(string) (os.FileInfo, error) {
		return nil, &fs.PathError{Op: "stat", Path: ".", Err: fs.ErrPermission}
	}
	seams.chdir = func(dir string) error { entered = append(entered, dir); return nil }

	ensureUsableWorkingDirectoryWith(seams, &stderr)

	if len(entered) != 1 || entered[0] != "/home/example" {
		t.Fatalf("entered = %v, want exactly [/home/example]", entered)
	}
	if !strings.Contains(stderr.String(), "/home/example") {
		t.Fatalf("stderr = %q, want the relocation to be announced", stderr.String())
	}
}

// A directory deleted underneath a live shell keeps a stat-able inode, so only
// resolving its path reveals the problem. The Go toolchain refuses to run in
// this state too, so the guard has to catch it.
func TestEnsureUsableWorkingDirectoryRelocatesWhenCWDWasDeleted(t *testing.T) {
	var stderr bytes.Buffer
	var entered []string
	seams := usableWorkingDirSeams()
	seams.getwd = func() (string, error) {
		return "", &fs.PathError{Op: "getwd", Path: ".", Err: fs.ErrNotExist}
	}
	seams.chdir = func(dir string) error { entered = append(entered, dir); return nil }

	ensureUsableWorkingDirectoryWith(seams, &stderr)

	if len(entered) != 1 || entered[0] != "/home/example" {
		t.Fatalf("entered = %v, want exactly [/home/example]", entered)
	}
	if !strings.Contains(stderr.String(), "/home/example") {
		t.Fatalf("stderr = %q, want the relocation to be announced", stderr.String())
	}
}

func TestEnsureUsableWorkingDirectoryFallsBackToTempWhenHomeIsUnusable(t *testing.T) {
	var stderr bytes.Buffer
	var entered []string
	seams := usableWorkingDirSeams()
	seams.stat = func(string) (os.FileInfo, error) { return nil, errors.New("permission denied") }
	seams.homeDir = func() (string, error) { return "", errors.New("no home") }
	seams.chdir = func(dir string) error { entered = append(entered, dir); return errors.New("cannot enter") }

	ensureUsableWorkingDirectoryWith(seams, &stderr)

	if len(entered) != 1 || entered[0] != "/tmp" {
		t.Fatalf("entered = %v, want the temp-dir fallback to be attempted", entered)
	}
	if !strings.Contains(stderr.String(), "no fallback directory") {
		t.Fatalf("stderr = %q, want the unrecoverable case reported", stderr.String())
	}
}
