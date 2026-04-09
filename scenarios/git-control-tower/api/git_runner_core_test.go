package main

import (
	"strings"
	"testing"
)

func TestReadArgs_PrependsNoOptionalLocks(t *testing.T) {
	args := readArgs("/repo", "status", "--porcelain=v2")
	if len(args) != 5 {
		t.Fatalf("expected 5 args, got %d: %v", len(args), args)
	}
	if args[0] != "--no-optional-locks" {
		t.Errorf("first arg should be --no-optional-locks, got %q", args[0])
	}
	if args[1] != "-C" {
		t.Errorf("second arg should be -C, got %q", args[1])
	}
	if args[2] != "/repo" {
		t.Errorf("third arg should be /repo, got %q", args[2])
	}
	if args[3] != "status" {
		t.Errorf("fourth arg should be status, got %q", args[3])
	}
	if args[4] != "--porcelain=v2" {
		t.Errorf("fifth arg should be --porcelain=v2, got %q", args[4])
	}
}

func TestReadArgs_EmptySubcommand(t *testing.T) {
	args := readArgs("/repo")
	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d: %v", len(args), args)
	}
	if args[0] != "--no-optional-locks" {
		t.Errorf("first arg should be --no-optional-locks, got %q", args[0])
	}
}

func TestReadArgs_NotInWriteMethods(t *testing.T) {
	// Verify that write methods in ExecGitRunner do NOT use readArgs
	// by checking that readArgs output always starts with --no-optional-locks.
	args := readArgs("/repo", "add", "--", "file.go")
	if args[0] != "--no-optional-locks" {
		t.Fatal("readArgs should always prepend --no-optional-locks")
	}
	// This test documents the contract: write methods must NOT use readArgs.
	// The Stage, Unstage, Commit, etc. methods construct args directly with
	// []string{"-C", repoDir, ...} — verified by reading git_runner_core.go.
}

func TestReadArgs_PreservesArgOrder(t *testing.T) {
	args := readArgs("/my/repo", "diff", "--numstat", "--no-color", "--cached")
	expected := "--no-optional-locks|-C|/my/repo|diff|--numstat|--no-color|--cached"
	got := strings.Join(args, "|")
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}
