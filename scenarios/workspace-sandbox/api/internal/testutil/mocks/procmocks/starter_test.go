package procmocks_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"workspace-sandbox/internal/process"
	"workspace-sandbox/internal/testutil/mocks/procmocks"
)

func TestFakeStarter_LookPath_TableLookup(t *testing.T) {
	s := procmocks.NewFakeStarter()
	s.SetLookPath("foo", "/usr/bin/foo")
	got, err := s.LookPath("foo")
	if err != nil {
		t.Fatalf("LookPath(foo): %v", err)
	}
	if got != "/usr/bin/foo" {
		t.Errorf("got %q, want /usr/bin/foo", got)
	}
	_, err = s.LookPath("bar")
	if !errors.Is(err, process.ErrBinaryNotFound) {
		t.Errorf("LookPath(bar) err=%v, want ErrBinaryNotFound", err)
	}
}

func TestFakeStarter_Start_FailOnUnmatchedByDefault(t *testing.T) {
	s := procmocks.NewFakeStarter()
	_, err := s.Start(context.Background(), process.StartOpts{Path: "echo"})
	if err == nil {
		t.Fatal("expected fail-on-unmatched")
	}
	if !strings.Contains(err.Error(), "echo") {
		t.Errorf("error mentions command? got %v", err)
	}
}

func TestFakeStarter_Start_DefaultBehaviorFallback(t *testing.T) {
	s := procmocks.NewFakeStarter()
	s.SetDefault(procmocks.CommandBehavior{
		Exit: process.ProcessExit{ExitCode: 7},
	})
	h, err := s.Start(context.Background(), process.StartOpts{Path: "anything"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	exit, _ := h.Wait(context.Background())
	if exit.ExitCode != 7 {
		t.Errorf("ExitCode: got %d, want 7", exit.ExitCode)
	}
}

func TestFakeStarter_Start_LongestPrefixMatchWins(t *testing.T) {
	s := procmocks.NewFakeStarter()
	s.AddCommand("git", procmocks.CommandBehavior{Exit: process.ProcessExit{ExitCode: 1}})
	s.AddCommand("git status", procmocks.CommandBehavior{Exit: process.ProcessExit{ExitCode: 2}})
	s.AddCommand("git status --porcelain", procmocks.CommandBehavior{Exit: process.ProcessExit{ExitCode: 3}})

	cases := []struct {
		path string
		args []string
		want int
	}{
		{"git", []string{"--version"}, 1},
		{"git", []string{"status"}, 2},
		{"git", []string{"status", "--porcelain"}, 3},
	}
	for _, c := range cases {
		h, err := s.Start(context.Background(), process.StartOpts{Path: c.path, Args: c.args})
		if err != nil {
			t.Fatalf("Start %v: %v", c, err)
		}
		exit, _ := h.Wait(context.Background())
		if exit.ExitCode != c.want {
			t.Errorf("Start(%v) ExitCode=%d, want %d", c, exit.ExitCode, c.want)
		}
	}
}

func TestFakeStarter_Start_StartErrPropagates(t *testing.T) {
	s := procmocks.NewFakeStarter()
	s.AddCommand("foo", procmocks.CommandBehavior{StartErr: errors.New("boom")})
	_, err := s.Start(context.Background(), process.StartOpts{Path: "foo"})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("expected boom, got %v", err)
	}
}

func TestFakeStarter_Wait_StreamsScriptedOutput(t *testing.T) {
	s := procmocks.NewFakeStarter()
	s.AddCommand("greet", procmocks.CommandBehavior{
		Stdout: []byte("hello"),
		Stderr: []byte("warn"),
		Exit:   process.ProcessExit{ExitCode: 0},
	})

	var stdout, stderr bytes.Buffer
	h, err := s.Start(context.Background(), process.StartOpts{
		Path:   "greet",
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if _, err := h.Wait(context.Background()); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if stdout.String() != "hello" {
		t.Errorf("stdout=%q, want hello", stdout.String())
	}
	if stderr.String() != "warn" {
		t.Errorf("stderr=%q, want warn", stderr.String())
	}
}

func TestFakeStarter_Wait_ContextCancelOverridesHold(t *testing.T) {
	s := procmocks.NewFakeStarter()
	s.AddCommand("sleeper", procmocks.CommandBehavior{Hold: true})

	h, err := s.Start(context.Background(), process.StartOpts{Path: "sleeper"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	exit, waitErr := h.Wait(ctx)
	if waitErr == nil {
		t.Error("expected ctx error")
	}
	if exit.Signal == 0 {
		t.Errorf("Signal: got 0, want SIGKILL")
	}
}

func TestFakeStarter_RunCombinedOutput(t *testing.T) {
	s := procmocks.NewFakeStarter()
	s.AddCommand("merge", procmocks.CommandBehavior{
		Stdout: []byte("out"),
		Stderr: []byte("err"),
	})
	res, err := process.RunCombinedOutput(context.Background(), s, process.StartOpts{Path: "merge"})
	if err != nil {
		t.Fatalf("RunCombinedOutput: %v", err)
	}
	out := string(res.Stdout)
	if !strings.Contains(out, "out") || !strings.Contains(out, "err") {
		t.Errorf("combined: %q, want out+err", out)
	}
}

func TestFakeStarter_Kill_FlipsKilled(t *testing.T) {
	s := procmocks.NewFakeStarter()
	s.AddCommand("hold", procmocks.CommandBehavior{Hold: true})
	h, err := s.Start(context.Background(), process.StartOpts{Path: "hold"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	fh, ok := h.(*procmocks.FakeHandle)
	if !ok {
		t.Fatalf("Start returned %T, want *FakeHandle", h)
	}
	if fh.Killed() {
		t.Error("Killed() should start false")
	}
	if err := fh.Kill(); err != nil {
		t.Errorf("Kill: %v", err)
	}
	if !fh.Killed() {
		t.Error("Killed() should be true after Kill")
	}
}

func TestFakeStarter_Calls_RecordedInOrder(t *testing.T) {
	s := procmocks.NewFakeStarter()
	s.SetDefault(procmocks.CommandBehavior{})
	for _, p := range []string{"a", "b", "c"} {
		_, _ = s.Start(context.Background(), process.StartOpts{Path: p})
	}
	if len(s.Calls) != 3 {
		t.Fatalf("Calls: got %d, want 3", len(s.Calls))
	}
	for i, want := range []string{"a", "b", "c"} {
		if s.Calls[i].Path != want {
			t.Errorf("Calls[%d].Path=%q, want %q", i, s.Calls[i].Path, want)
		}
	}
}

func TestFakeStarter_StartErrFromGlobal(t *testing.T) {
	s := procmocks.NewFakeStarter()
	s.SetStartErr(errors.New("boom"))
	_, err := s.Start(context.Background(), process.StartOpts{Path: "foo"})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("expected boom, got %v", err)
	}
}
