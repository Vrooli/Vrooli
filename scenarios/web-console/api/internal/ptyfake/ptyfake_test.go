package ptyfake

import (
	"errors"
	"testing"

	"web-console/internal/pty"
)

func TestFakePTYStateAndFactory(t *testing.T) {
	f := NewFakePTYWithOutput()
	if err := f.SetSize(120, 40); err != nil {
		t.Fatal(err)
	}
	f.SetExitCode(7)
	if f.ExitCode() != 7 {
		t.Fatal("exit code not retained")
	}
	f.SetWriteInputErr(errors.New("blocked"))
	if err := f.WriteInput([]byte("x"), 0); err == nil {
		t.Fatal("forced write error missing")
	}
	if _, err := f.CurrentDir(nil); err != nil {
		t.Fatal(err)
	}
	if err := f.Kill(); err != nil || !f.Killed {
		t.Fatalf("kill err=%v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := NewFactory()(pty.LaunchSpec{}); err != nil {
		t.Fatal(err)
	}
}
