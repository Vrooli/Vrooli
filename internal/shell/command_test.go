package shell

import (
	"bytes"
	"strings"
	"testing"
)

func TestStderrTailKeepsLastNLines(t *testing.T) {
	tail := NewStderrTail(3)
	tail.Write([]byte("one\ntwo\nthree\nfour\nfive\n"))
	got := tail.Tail()
	want := []string{"three", "four", "five"}
	if len(got) != len(want) {
		t.Fatalf("Tail() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Tail()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestStderrTailHandlesPartialAndCRLF(t *testing.T) {
	tail := NewStderrTail(2)
	tail.Write([]byte("alpha\r\n"))
	tail.Write([]byte("brav"))
	tail.Write([]byte("o\r\nchar"))
	got := tail.String()
	if !strings.Contains(got, "bravo") {
		t.Fatalf("missing bravo in %q", got)
	}
	if !strings.Contains(got, "char") {
		t.Fatalf("partial trailing line should be retained: %q", got)
	}
	for _, line := range tail.Tail() {
		if strings.HasSuffix(line, "\r") {
			t.Fatalf("CR not stripped from %q", line)
		}
	}
}

func TestStderrTailZeroMaxFallsBackToTen(t *testing.T) {
	tail := NewStderrTail(0)
	for i := 0; i < 15; i++ {
		tail.Write([]byte("line\n"))
	}
	if got := len(tail.Tail()); got != 10 {
		t.Fatalf("len(Tail()) = %d, want 10", got)
	}
}

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
