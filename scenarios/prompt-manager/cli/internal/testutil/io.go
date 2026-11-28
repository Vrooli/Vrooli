package testutil

import (
	"bytes"
	"io"
	"os"
	"sync"
	"testing"
)

var processIOMu sync.Mutex

// Output captures stdout and stderr while fn runs.
func Output(t testing.TB, fn func() error) (stdout string, stderr string, err error) {
	t.Helper()
	return IO(t, "", fn)
}

// IO captures stdout/stderr and optionally provides stdin while fn runs.
func IO(t testing.TB, stdin string, fn func() error) (stdout string, stderr string, err error) {
	t.Helper()
	processIOMu.Lock()
	defer processIOMu.Unlock()

	oldStdout := os.Stdout
	oldStderr := os.Stderr
	oldStdin := os.Stdin

	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}
	var stdinFile *os.File
	if stdin != "" {
		stdinReader, stdinWriter, err := os.Pipe()
		if err != nil {
			t.Fatalf("create stdin pipe: %v", err)
		}
		if _, err := stdinWriter.WriteString(stdin); err != nil {
			t.Fatalf("write stdin fixture: %v", err)
		}
		if err := stdinWriter.Close(); err != nil {
			t.Fatalf("close stdin writer: %v", err)
		}
		stdinFile = stdinReader
		os.Stdin = stdinReader
	}

	os.Stdout = stdoutWriter
	os.Stderr = stderrWriter

	outCh := make(chan string, 1)
	errCh := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, stdoutReader)
		outCh <- buf.String()
	}()
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, stderrReader)
		errCh <- buf.String()
	}()

	runErr := fn()

	os.Stdout = oldStdout
	os.Stderr = oldStderr
	os.Stdin = oldStdin

	if err := stdoutWriter.Close(); err != nil {
		t.Fatalf("close stdout writer: %v", err)
	}
	if err := stderrWriter.Close(); err != nil {
		t.Fatalf("close stderr writer: %v", err)
	}
	if stdinFile != nil {
		if err := stdinFile.Close(); err != nil {
			t.Fatalf("close stdin reader: %v", err)
		}
	}
	stdout = <-outCh
	stderr = <-errCh
	if err := stdoutReader.Close(); err != nil {
		t.Fatalf("close stdout reader: %v", err)
	}
	if err := stderrReader.Close(); err != nil {
		t.Fatalf("close stderr reader: %v", err)
	}

	return stdout, stderr, runErr
}
