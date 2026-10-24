package ptyfake

import (
	"context"
	"errors"
	"io"
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
	if _, err := f.CurrentDir(context.Background()); err != nil {
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

func TestFakePTYReadProbeAndRepeatedClose(t *testing.T) {
	f := NewFakePTYWithOutput()
	if err := f.ProbeReady(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := f.StdoutReader.CloseWithError(io.EOF); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Read(make([]byte, 1)); err == nil {
		t.Fatal("closed reader should return an error")
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal("repeated close should be harmless")
	}
}
