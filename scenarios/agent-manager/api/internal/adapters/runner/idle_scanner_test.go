package runner

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestManagedProcess_NormalEOF(t *testing.T) {
	cmd := exec.Command("printf", "line1\nline2\nline3\n")
	cmd.SysProcAttr = managedProcessSysProcAttr()

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

// TestManagedProcess_StderrRemainsReadableAfterChildExit pins the ownership
// boundary needed by the durable transcript drainer: the background cmd.Wait
// may finish before a consumer reads stderr, but it must not close the
// consumer's pipe first.
func TestManagedProcess_StderrRemainsReadableAfterChildExit(t *testing.T) {
	cmd := exec.Command("sh", "-c", "printf retained >&2")
	cmd.SysProcAttr = managedProcessSysProcAttr()

	mp, err := startManagedProcess(cmd, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer mp.Wait()

	// Give the short-lived child enough time to exit and the background
	// cmd.Wait to run. With exec.Cmd.StderrPipe this read would fail because
	// cmd.Wait closes the pipe before the transcript drainer can consume it.
	time.Sleep(20 * time.Millisecond)
	got, err := io.ReadAll(mp.Stderr())
	if err != nil {
		t.Fatalf("read stderr after child exit: %v", err)
	}
	if string(got) != "retained" {
		t.Fatalf("stderr = %q, want retained", got)
	}
}

func TestManagedProcess_GrandchildPipe(t *testing.T) {
	// Process that exits but leaves a grandchild holding stdout open.
	// With old StdoutPipe: scanner blocks forever.
	// With managedProcess: scanner gets EOF promptly after main process exits.
	cmd := exec.Command("bash", "-c", `bash -c "sleep 300" & echo "done"`)
	cmd.SysProcAttr = managedProcessSysProcAttr()

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
	cmd.SysProcAttr = managedProcessSysProcAttr()

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
	cmd.SysProcAttr = managedProcessSysProcAttr()

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
	cmd.SysProcAttr = managedProcessSysProcAttr()

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

func TestManagedProcessWithFiles_OutputGrowthResetsIdleTimeout(t *testing.T) {
	dir := t.TempDir()
	stdout, err := os.Create(dir + "/stdout.log")
	if err != nil {
		t.Fatal(err)
	}
	defer stdout.Close()
	stderr, err := os.Create(dir + "/stderr.log")
	if err != nil {
		t.Fatal(err)
	}
	defer stderr.Close()
	cmd := exec.Command("sh", "-c", "printf 'one\\n'; sleep 0.14; printf 'two\\n'; sleep 0.14; printf 'three\\n'; sleep 0.14")
	cmd.SysProcAttr = managedProcessSysProcAttr()
	cmd.Stdout, cmd.Stderr = stdout, stderr
	mp, err := startManagedProcessWithFiles(cmd, 300*time.Millisecond, stdout)
	if err != nil {
		t.Fatal(err)
	}
	if err := mp.Wait(); err != nil {
		t.Fatalf("command failed despite continued output: %v", err)
	}
	if mp.TimedOut() {
		t.Fatal("continued durable output did not reset idle timeout")
	}
	content, err := os.ReadFile(stdout.Name())
	if err != nil || string(content) != "one\ntwo\nthree\n" {
		t.Fatalf("durable output=%q err=%v", content, err)
	}
}

func TestObserveFileActivityGrowthAtDeadlineResetsBeforeTimeout(t *testing.T) {
	lastActivity := time.Unix(100, 0)
	timeout := 200 * time.Millisecond
	deadline := lastActivity.Add(timeout)
	lastSize, nextActivity, expired := observeFileActivity(11, 10, deadline, lastActivity, timeout)
	if expired || lastSize != 11 || !nextActivity.Equal(deadline) {
		t.Fatalf("fresh boundary write was treated as expired: size=%d activity=%s expired=%v", lastSize, nextActivity, expired)
	}
	_, _, expired = observeFileActivity(11, 11, deadline.Add(timeout), nextActivity, timeout)
	if !expired {
		t.Fatal("unchanged output at the next full deadline was not expired")
	}
}

func TestManagedProcessWithFiles_SilentOutputTimesOut(t *testing.T) {
	dir := t.TempDir()
	stdout, err := os.Create(dir + "/stdout.log")
	if err != nil {
		t.Fatal(err)
	}
	defer stdout.Close()
	stderr, err := os.Create(dir + "/stderr.log")
	if err != nil {
		t.Fatal(err)
	}
	defer stderr.Close()
	cmd := exec.Command("sh", "-c", "sleep 5")
	cmd.SysProcAttr = managedProcessSysProcAttr()
	cmd.Stdout, cmd.Stderr = stdout, stderr
	mp, err := startManagedProcessWithFiles(cmd, 100*time.Millisecond, stdout)
	if err != nil {
		t.Fatal(err)
	}
	_ = mp.Wait()
	if !mp.TimedOut() {
		t.Fatal("silent process did not fire configured idle timeout")
	}
}

func TestManagedProcess_Kill(t *testing.T) {
	cmd := exec.Command("bash", "-c", `echo "start"; sleep 300`)
	cmd.SysProcAttr = managedProcessSysProcAttr()

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
	cmd.SysProcAttr = managedProcessSysProcAttr()

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
	cmd.SysProcAttr = managedProcessSysProcAttr()

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
