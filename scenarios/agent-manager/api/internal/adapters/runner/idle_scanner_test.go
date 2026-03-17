package runner

import (
	"bufio"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestManagedProcess_NormalEOF(t *testing.T) {
	cmd := exec.Command("printf", "line1\nline2\nline3\n")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	mp, err := startManagedProcess(cmd, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	scanner := bufio.NewScanner(mp.Stdout())
	var lines []string
	for scanner.Scan() {
		mp.ResetTimer()
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("unexpected scanner error: %v", err)
	}
	if mp.TimedOut() {
		t.Fatal("should not have timed out on normal EOF")
	}
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}
	if err := mp.Wait(); err != nil {
		t.Fatalf("unexpected wait error: %v", err)
	}
}

func TestManagedProcess_GrandchildPipe(t *testing.T) {
	// Process that exits but leaves a grandchild holding stdout open.
	// With old StdoutPipe: scanner blocks forever.
	// With managedProcess: scanner gets EOF promptly after main process exits.
	cmd := exec.Command("bash", "-c", `bash -c "sleep 300" & echo "done"`)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	mp, err := startManagedProcess(cmd, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	scanner := bufio.NewScanner(mp.Stdout())
	var lines []string
	for scanner.Scan() {
		mp.ResetTimer()
		lines = append(lines, scanner.Text())
	}
	elapsed := time.Since(start)

	if mp.TimedOut() {
		t.Fatal("should not have timed out — process exited normally")
	}
	if len(lines) != 1 || lines[0] != "done" {
		t.Fatalf("expected [done], got %v", lines)
	}
	// Should complete in well under the 10s timeout — the grandchild (sleep 300)
	// gets killed when the process group is killed after cmd.Wait() returns.
	if elapsed > 5*time.Second {
		t.Fatalf("grandchild pipe test took too long: %v (expected < 5s)", elapsed)
	}
	if err := mp.Wait(); err != nil {
		t.Fatalf("unexpected wait error: %v", err)
	}
}

func TestManagedProcess_SafetyTimeout(t *testing.T) {
	// Process that writes one line then hangs — short timeout fires
	cmd := exec.Command("bash", "-c", `echo "hello"; sleep 300`)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	mp, err := startManagedProcess(cmd, 500*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	scanner := bufio.NewScanner(mp.Stdout())
	var lines []string
	start := time.Now()
	for scanner.Scan() {
		mp.ResetTimer()
		lines = append(lines, scanner.Text())
	}
	elapsed := time.Since(start)

	if !mp.TimedOut() {
		t.Fatal("expected idle timeout to fire")
	}
	if len(lines) != 1 || lines[0] != "hello" {
		t.Fatalf("expected [hello], got %v", lines)
	}
	if elapsed > 5*time.Second {
		t.Fatalf("idle timeout took too long: %v", elapsed)
	}
	_ = mp.Wait()
}

func TestManagedProcess_TimerReset(t *testing.T) {
	// Process writes a line every 200ms — should NOT time out with 500ms timeout
	cmd := exec.Command("bash", "-c", `for i in 1 2 3 4 5; do echo "line$i"; sleep 0.2; done`)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	mp, err := startManagedProcess(cmd, 500*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	scanner := bufio.NewScanner(mp.Stdout())
	var lines []string
	for scanner.Scan() {
		mp.ResetTimer()
		lines = append(lines, scanner.Text())
	}

	if mp.TimedOut() {
		t.Fatal("should not have timed out — output was arriving regularly")
	}
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines, got %d: %v", len(lines), lines)
	}
	if err := mp.Wait(); err != nil {
		t.Fatalf("unexpected wait error: %v", err)
	}
}

func TestManagedProcess_NoTimeout(t *testing.T) {
	cmd := exec.Command("printf", "a\nb\n")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	mp, err := startManagedProcess(cmd, 0)
	if err != nil {
		t.Fatal(err)
	}

	scanner := bufio.NewScanner(mp.Stdout())
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if mp.TimedOut() {
		t.Fatal("should not time out when timeout is 0")
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if err := mp.Wait(); err != nil {
		t.Fatalf("unexpected wait error: %v", err)
	}
}

func TestManagedProcess_Kill(t *testing.T) {
	cmd := exec.Command("bash", "-c", `echo "start"; sleep 300`)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	mp, err := startManagedProcess(cmd, 0) // no timer
	if err != nil {
		t.Fatal(err)
	}

	scanner := bufio.NewScanner(mp.Stdout())
	var lines []string
	// Read the first line, then kill
	if scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	mp.Kill()

	// Scanner should get EOF after kill
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if len(lines) != 1 || lines[0] != "start" {
		t.Fatalf("expected [start], got %v", lines)
	}
	_ = mp.Wait()
}

func TestManagedProcess_LargeBuffer(t *testing.T) {
	longLine := strings.Repeat("x", 100000)
	cmd := exec.Command("printf", longLine+"\n")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	mp, err := startManagedProcess(cmd, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	scanner := bufio.NewScanner(mp.Stdout())
	scanner.Buffer(make([]byte, 64*1024), 512*1024)

	var lines []string
	for scanner.Scan() {
		mp.ResetTimer()
		lines = append(lines, scanner.Text())
	}

	if mp.TimedOut() {
		t.Fatal("should not have timed out")
	}
	if len(lines) != 1 || len(lines[0]) != 100000 {
		t.Fatalf("expected 1 line of 100000 chars, got %d lines", len(lines))
	}
	if err := mp.Wait(); err != nil {
		t.Fatalf("unexpected wait error: %v", err)
	}
}

func TestManagedProcess_WaitIdempotent(t *testing.T) {
	cmd := exec.Command("true")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	mp, err := startManagedProcess(cmd, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	// Drain stdout
	scanner := bufio.NewScanner(mp.Stdout())
	for scanner.Scan() {
	}

	// Call Wait multiple times — should not panic or return different results
	err1 := mp.Wait()
	err2 := mp.Wait()
	if err1 != err2 {
		t.Fatalf("expected same error from multiple Wait() calls, got %v and %v", err1, err2)
	}
}
