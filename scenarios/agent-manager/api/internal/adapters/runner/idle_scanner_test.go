package runner

import (
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestIdleScanner_NormalEOF(t *testing.T) {
	// Process that writes 3 lines and exits
	cmd := exec.Command("printf", "line1\nline2\nline3\n")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	is := newIdleScanner(stdout, cmd, 2*time.Second)
	defer is.Stop()

	var lines []string
	for is.Scan() {
		lines = append(lines, is.Text())
	}

	if err := is.Err(); err != nil {
		t.Fatalf("unexpected scanner error: %v", err)
	}
	if is.TimedOut() {
		t.Fatal("should not have timed out on normal EOF")
	}
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}
	_ = cmd.Wait()
}

func TestIdleScanner_IdleTimeout(t *testing.T) {
	// Process that writes one line, then hangs indefinitely
	cmd := exec.Command("bash", "-c", `echo "hello"; sleep 300`)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	is := newIdleScanner(stdout, cmd, 500*time.Millisecond)
	defer is.Stop()

	var lines []string
	start := time.Now()
	for is.Scan() {
		lines = append(lines, is.Text())
	}
	elapsed := time.Since(start)

	if !is.TimedOut() {
		t.Fatal("expected idle timeout to fire")
	}
	if len(lines) != 1 || lines[0] != "hello" {
		t.Fatalf("expected [hello], got %v", lines)
	}
	// Should have timed out in roughly 500ms, not 300s
	if elapsed > 5*time.Second {
		t.Fatalf("idle timeout took too long: %v", elapsed)
	}
	_ = cmd.Wait()
}

func TestIdleScanner_NoTimeout(t *testing.T) {
	// Timeout=0 means disabled
	cmd := exec.Command("printf", "a\nb\n")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	is := newIdleScanner(stdout, cmd, 0)
	defer is.Stop()

	var lines []string
	for is.Scan() {
		lines = append(lines, is.Text())
	}

	if is.TimedOut() {
		t.Fatal("should not time out when timeout is 0")
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	_ = cmd.Wait()
}

func TestIdleScanner_ResetOnOutput(t *testing.T) {
	// Process that writes a line every 200ms — should NOT time out with 500ms idle timeout
	cmd := exec.Command("bash", "-c", `for i in 1 2 3 4 5; do echo "line$i"; sleep 0.2; done`)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	is := newIdleScanner(stdout, cmd, 500*time.Millisecond)
	defer is.Stop()

	var lines []string
	for is.Scan() {
		lines = append(lines, is.Text())
	}

	if is.TimedOut() {
		t.Fatal("should not have timed out — output was arriving regularly")
	}
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines, got %d: %v", len(lines), lines)
	}
	_ = cmd.Wait()
}

func TestReapProcess_AlreadyExited(t *testing.T) {
	// Process that exits immediately — reapProcess should return the real exit status
	cmd := exec.Command("true")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	// Wait for it to exit naturally
	time.Sleep(100 * time.Millisecond)

	err := reapProcess(cmd)
	if err != nil {
		t.Fatalf("expected nil error for clean exit, got: %v", err)
	}
}

func TestReapProcess_StuckProcess(t *testing.T) {
	// Process that hangs forever — reapProcess should kill it and return nil
	cmd := exec.Command("sleep", "300")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	err := reapProcess(cmd)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected nil error (we killed it ourselves), got: %v", err)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("reapProcess took too long: %v", elapsed)
	}
}

func TestReapProcess_FailedExit(t *testing.T) {
	// Process that exits with error code — reapProcess should return the error
	cmd := exec.Command("false")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)

	err := reapProcess(cmd)
	if err == nil {
		t.Fatal("expected error for non-zero exit, got nil")
	}
}

func TestIdleScanner_WithBuffer(t *testing.T) {
	// Verify WithBuffer works (large output)
	longLine := strings.Repeat("x", 100000) // 100KB line
	cmd := exec.Command("printf", longLine+"\n")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	is := newIdleScanner(stdout, cmd, 2*time.Second).
		WithBuffer(make([]byte, 64*1024), 512*1024)
	defer is.Stop()

	var lines []string
	for is.Scan() {
		lines = append(lines, is.Text())
	}

	if is.TimedOut() {
		t.Fatal("should not have timed out")
	}
	if len(lines) != 1 || len(lines[0]) != 100000 {
		t.Fatalf("expected 1 line of 100000 chars, got %d lines", len(lines))
	}
	_ = cmd.Wait()
}
