package shell

import (
	"bytes"
	"testing"
)

func TestCommandUsesProvidedSpec(t *testing.T) {
	var stdout bytes.Buffer
	cmd := Command(Spec{
		Name:   "echo",
		Args:   []string{"hello"},
		Dir:    "/tmp",
		Env:    []string{"A=B"},
		Stdout: &stdout,
	})

	if cmd.Path == "" || cmd.Args[0] != "echo" {
		t.Fatalf("cmd.Path = %q", cmd.Path)
	}
	if len(cmd.Args) != 2 || cmd.Args[1] != "hello" {
		t.Fatalf("cmd.Args = %#v", cmd.Args)
	}
	if cmd.Dir != "/tmp" {
		t.Fatalf("cmd.Dir = %q", cmd.Dir)
	}
	if len(cmd.Env) != 1 || cmd.Env[0] != "A=B" {
		t.Fatalf("cmd.Env = %#v", cmd.Env)
	}
	if cmd.Stdout != &stdout {
		t.Fatal("expected stdout writer to be preserved")
	}
}

func TestBashCommandWrapsCommandLine(t *testing.T) {
	cmd := BashCommand("echo hello", Spec{Dir: "/fixture"})

	if cmd.Path == "" || cmd.Args[0] != "bash" {
		t.Fatalf("cmd.Path = %q", cmd.Path)
	}
	if len(cmd.Args) != 3 || cmd.Args[1] != "-lc" || cmd.Args[2] != "echo hello" {
		t.Fatalf("cmd.Args = %#v", cmd.Args)
	}
	if cmd.Dir != "/fixture" {
		t.Fatalf("cmd.Dir = %q", cmd.Dir)
	}
}
