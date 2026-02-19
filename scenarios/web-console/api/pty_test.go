package main

import (
	"io"
	"sync"
	"testing"
	"time"
)

// fakePTY is a pipe-based PTY substitute for fast, deterministic tests.
// It satisfies the PTY interface without spawning real shell processes.
// Use fakePTYWithOutput when you need to simulate PTY stdout.
type fakePTY struct {
	stdoutReader *io.PipeReader // Read() reads from this (simulates PTY output)
	stdinWriter  *io.PipeWriter // Write() writes to this (simulates keyboard input)
	mu           sync.Mutex
	cols         uint16
	rows         uint16
	killed       bool
	closed       bool
	exitCode     int
}

func (f *fakePTY) Read(p []byte) (int, error)  { return f.stdoutReader.Read(p) }
func (f *fakePTY) Write(p []byte) (int, error) { return f.stdinWriter.Write(p) }

func (f *fakePTY) SetSize(cols, rows uint16) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cols = cols
	f.rows = rows
	return nil
}

func (f *fakePTY) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return nil
	}
	f.closed = true
	f.stdoutReader.Close()
	f.stdinWriter.Close()
	return nil
}

func (f *fakePTY) Kill() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.killed = true
	return nil
}

func (f *fakePTY) ExitCode() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.exitCode
}

func (f *fakePTY) SetExitCode(code int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.exitCode = code
}

// fakePTYWithOutput extends fakePTY with a writable stdout pipe so tests
// can inject output that the session's readLoop will broadcast to subscribers.
type fakePTYWithOutput struct {
	fakePTY
	outW *io.PipeWriter
}

func newFakePTYWithOutput() *fakePTYWithOutput {
	stdoutR, stdoutW := io.Pipe()
	_, stdinW := io.Pipe()
	return &fakePTYWithOutput{
		fakePTY: fakePTY{
			stdoutReader: stdoutR,
			stdinWriter:  stdinW,
		},
		outW: stdoutW,
	}
}

func (f *fakePTYWithOutput) Close() error {
	f.outW.Close()
	f.stdinWriter.Close()
	return nil
}

// fakePTYFactory returns a PTYFactory that always returns the same PTY instance.
// Use when a test needs to inspect or control the exact PTY a session uses.
func fakePTYFactory(p PTY) PTYFactory {
	return func(shell string, cols, rows uint16) (PTY, error) {
		return p, nil
	}
}

// newFakePTYFactory returns a PTYFactory that creates a fresh fakePTYWithOutput
// for each session. Useful for tests that create multiple sessions.
func newFakePTYFactory() PTYFactory {
	return func(shell string, cols, rows uint16) (PTY, error) {
		return newFakePTYWithOutput(), nil
	}
}

// [REQ:P0-002a] PTY Session Backend - fast session tests via fake PTY seam
func TestFakePTY_CreateAndGet(t *testing.T) {
	fake := newFakePTYWithOutput()
	defer fake.Close()

	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	sess, err := sm.Create("/fake/shell", 100, 50)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if sess.Shell != "/fake/shell" {
		t.Errorf("expected shell=/fake/shell, got %s", sess.Shell)
	}
	if sess.Cols != 100 {
		t.Errorf("expected cols=100, got %d", sess.Cols)
	}
	if sess.Rows != 50 {
		t.Errorf("expected rows=50, got %d", sess.Rows)
	}

	got, ok := sm.Get(sess.ID)
	if !ok {
		t.Fatal("Get should find the session")
	}
	if got.ID != sess.ID {
		t.Errorf("expected ID %s, got %s", sess.ID, got.ID)
	}
}

// [REQ:P0-002b] WebSocket I/O Streaming - broadcast via fake PTY
func TestFakePTY_SubscribeAndBroadcast(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))

	sess, err := sm.Create("/fake/shell", 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	ch := sess.Subscribe()
	defer sess.Unsubscribe(ch)

	// Write output from fake PTY
	testData := []byte("hello from fake")
	go func() {
		_, _ = fake.outW.Write(testData)
	}()

	select {
	case data := <-ch:
		if string(data) != "hello from fake" {
			t.Errorf("expected 'hello from fake', got %q", string(data))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for broadcast data")
	}

	// Cleanup
	fake.Close()
	<-sess.Done()
}

// [REQ:P0-003b] Reconnect State Restoration - offline buffer via fake PTY
func TestFakePTY_OfflineBuffer(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))

	sess, err := sm.Create("/fake/shell", 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Write output while no subscribers connected
	_, _ = fake.outW.Write([]byte("offline data"))
	time.Sleep(50 * time.Millisecond)

	// Subscribe and expect buffered data
	ch := sess.Subscribe()
	defer sess.Unsubscribe(ch)

	select {
	case data := <-ch:
		if string(data) != "offline data" {
			t.Errorf("expected 'offline data', got %q", string(data))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for offline buffer")
	}

	fake.Close()
	<-sess.Done()
}

// [REQ:P0-002c] Terminal Resize Handling - resize delegates to PTY interface
func TestFakePTY_Resize(t *testing.T) {
	fake := newFakePTYWithOutput()
	defer fake.Close()

	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	sess, err := sm.Create("/fake/shell", 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := sm.Resize(sess.ID, 200, 60); err != nil {
		t.Fatalf("Resize failed: %v", err)
	}

	got, _ := sm.Get(sess.ID)
	if got.Cols != 200 {
		t.Errorf("expected cols=200, got %d", got.Cols)
	}
	if got.Rows != 60 {
		t.Errorf("expected rows=60, got %d", got.Rows)
	}

	// Verify the PTY seam received the resize
	fake.mu.Lock()
	if fake.cols != 200 || fake.rows != 60 {
		t.Errorf("fake PTY should have received resize: cols=%d rows=%d", fake.cols, fake.rows)
	}
	fake.mu.Unlock()
}

// [REQ:P0-002a] PTY Session Backend - delete calls Kill + Close on PTY
func TestFakePTY_DeleteCleanup(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))

	sess, err := sm.Create("/fake/shell", 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := sm.Delete(sess.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	fake.mu.Lock()
	if !fake.killed {
		t.Error("Delete should call Kill on PTY")
	}
	fake.mu.Unlock()

	_, ok := sm.Get(sess.ID)
	if ok {
		t.Error("session should not exist after Delete")
	}
}

// [REQ:P0-002b] WebSocket I/O Streaming - exit code forwarding via fake PTY
func TestFakePTY_ExitCodeForwarding(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))

	sess, err := sm.Create("/fake/shell", 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Set a non-zero exit code and close to simulate process exit
	fake.SetExitCode(42)
	fake.outW.Close()

	select {
	case <-sess.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for exit signal")
	}

	if code := sess.ExitCode(); code != 42 {
		t.Errorf("expected exit code 42, got %d", code)
	}
}

// [REQ:P0-002a] PTY Session Backend - exit signal via fake PTY
func TestFakePTY_ExitSignal(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))

	sess, err := sm.Create("/fake/shell", 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Close the output pipe to simulate process exit
	fake.outW.Close()

	select {
	case <-sess.Done():
		// Expected: session signals exit
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for exit signal")
	}

	if !sess.IsDead() {
		t.Error("session should be dead after PTY close")
	}

	// SessionManager should auto-remove the session
	time.Sleep(50 * time.Millisecond)
	_, ok := sm.Get(sess.ID)
	if ok {
		t.Error("session should be auto-removed after exit")
	}
}
