package main

import (
	"os"
	"testing"
)

func TestRunRejectsUnknownCommand(t *testing.T) {
	if err := run([]string{"definitely-unknown"}); err == nil {
		t.Fatal("unknown command was accepted")
	}
}

func TestMainReportsRunErrors(t *testing.T) {
	oldExit := exitProcess
	defer func() { exitProcess = oldExit }()
	called := 0
	exitProcess = func(code int) { called = code }
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"unit-health", "definitely-unknown"}
	main()
	if called != 1 {
		t.Fatalf("exit code = %d, want 1", called)
	}
}
