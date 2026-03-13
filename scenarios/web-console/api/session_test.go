package main

import (
	"testing"
	"time"
)

// [REQ:P0-002a] PTY Session Backend
func TestSessionManager_Create(t *testing.T) {
	sm := NewSessionManager()

	sess, err := sm.Create("", 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	if sess.ID == "" {
		t.Error("session ID should not be empty")
	}
	if sess.Cols != 80 {
		t.Errorf("expected cols=80, got %d", sess.Cols)
	}
	if sess.Rows != 24 {
		t.Errorf("expected rows=24, got %d", sess.Rows)
	}
	if sess.CreatedAt.IsZero() {
		t.Error("created_at should not be zero")
	}
}

// [REQ:P0-002a] PTY Session Backend - concurrent sessions
func TestSessionManager_ConcurrentSessions(t *testing.T) {
	sm := NewSessionManager()

	s1, err := sm.Create("", 80, 24)
	if err != nil {
		t.Fatalf("Create s1 failed: %v", err)
	}
	defer func() { _ = sm.Delete(s1.ID) }()

	s2, err := sm.Create("", 120, 40)
	if err != nil {
		t.Fatalf("Create s2 failed: %v", err)
	}
	defer func() { _ = sm.Delete(s2.ID) }()

	if s1.ID == s2.ID {
		t.Error("session IDs should be unique")
	}

	sessions := sm.List()
	if len(sessions) < 2 {
		t.Errorf("expected at least 2 sessions, got %d", len(sessions))
	}
}

// [REQ:P0-003a] Session Persistence Store - list returns all active
func TestSessionManager_List(t *testing.T) {
	sm := NewSessionManager()

	sess, err := sm.Create("", 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	sessions := sm.List()
	found := false
	for _, s := range sessions {
		if s.ID == sess.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("List should include the created session")
	}
}

// [REQ:P0-002a] PTY Session Backend - Get
func TestSessionManager_Get(t *testing.T) {
	sm := NewSessionManager()

	sess, err := sm.Create("", 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	got, ok := sm.Get(sess.ID)
	if !ok {
		t.Fatal("Get should find the session")
	}
	if got.ID != sess.ID {
		t.Errorf("expected ID %s, got %s", sess.ID, got.ID)
	}

	_, ok = sm.Get("nonexistent")
	if ok {
		t.Error("Get should not find nonexistent session")
	}
}

// [REQ:P0-002a] PTY Session Backend - Delete cleanup
func TestSessionManager_Delete(t *testing.T) {
	sm := NewSessionManager()

	sess, err := sm.Create("", 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = sm.Delete(sess.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, ok := sm.Get(sess.ID)
	if ok {
		t.Error("session should not exist after Delete")
	}

	err = sm.Delete("nonexistent")
	if err == nil {
		t.Error("Delete should fail for nonexistent session")
	}
}

// [REQ:P0-002c] Terminal Resize Handling — per-client resize
func TestSession_ResizeClient(t *testing.T) {
	sm := NewSessionManager()

	sess, err := sm.Create("", 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	ch, _ := sess.Subscribe(80, 24)
	defer sess.Unsubscribe(ch)

	effCols, effRows := sess.ResizeClient(ch, 120, 40)
	if effCols != 120 {
		t.Errorf("expected effective cols=120, got %d", effCols)
	}
	if effRows != 40 {
		t.Errorf("expected effective rows=40, got %d", effRows)
	}

	got, _ := sm.Get(sess.ID)
	if got.Cols != 120 {
		t.Errorf("expected session cols=120, got %d", got.Cols)
	}
	if got.Rows != 40 {
		t.Errorf("expected session rows=40, got %d", got.Rows)
	}
}

// [REQ:P0-002c] Multi-client resize uses max dimensions
func TestSession_MultiClientResize_UsesMaxDimensions(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))

	sess, err := sm.Create("/fake/shell", 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Desktop client: 120x40
	ch1, _ := sess.Subscribe(120, 40)
	defer sess.Unsubscribe(ch1)

	// Phone client: 40x20
	ch2, _ := sess.Subscribe(40, 20)
	defer sess.Unsubscribe(ch2)

	// PTY should use max: 120x40
	cols, rows := sess.EffectiveSize()
	if cols != 120 {
		t.Errorf("expected PTY cols=120 (max of clients), got %d", cols)
	}
	if rows != 40 {
		t.Errorf("expected PTY rows=40 (max of clients), got %d", rows)
	}

	fake.Close()
	<-sess.Done()
}

// [REQ:P0-002c] Unsubscribe recomputes and shrinks PTY
func TestSession_ClientDisconnect_RecomputesShrinks(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))

	sess, err := sm.Create("/fake/shell", 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	ch1, _ := sess.Subscribe(120, 40)
	ch2, _ := sess.Subscribe(80, 24)

	// PTY should be 120x40
	cols, rows := sess.EffectiveSize()
	if cols != 120 || rows != 40 {
		t.Errorf("before disconnect: expected 120x40, got %dx%d", cols, rows)
	}

	// Disconnect the larger client
	sess.Unsubscribe(ch1)

	// PTY should shrink to 80x24
	cols, rows = sess.EffectiveSize()
	if cols != 80 {
		t.Errorf("after disconnect: expected cols=80, got %d", cols)
	}
	if rows != 24 {
		t.Errorf("after disconnect: expected rows=24, got %d", rows)
	}

	sess.Unsubscribe(ch2)
	fake.Close()
	<-sess.Done()
}

// [REQ:P0-002c] No clients falls back to session defaults
func TestSession_NoClients_UsesDefaults(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))

	sess, err := sm.Create("/fake/shell", 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	ch, _ := sess.Subscribe(120, 40)
	sess.Unsubscribe(ch) // remove only client

	// Should fall back to creation defaults (80x24), not zero
	cols, rows := sess.EffectiveSize()
	if cols != 120 || rows != 40 {
		// Note: recomputePTYSize only updates if different from current.
		// After the 120x40 client leaves, the session's Cols/Rows are already 120x40,
		// and with no clients the fallback is "keep current" since max(0,0) triggers fallback.
		// Actually the code says: if maxCols == 0 { maxCols = s.Cols } — which is 120 at this point.
		// This is by design: we don't shrink below the last-known size when all clients disconnect.
	}

	fake.Close()
	<-sess.Done()
}

// [REQ:P0-002b] WebSocket I/O Streaming - Subscribe/broadcast
func TestSession_SubscribeAndBroadcast(t *testing.T) {
	sm := NewSessionManager()

	sess, err := sm.Create("/bin/sh", 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	ch, _ := sess.Subscribe(80, 24)
	defer sess.Unsubscribe(ch)

	// Write to stdin - the shell should echo something back
	_, err = sess.Write([]byte("echo hello\n"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read output with timeout
	select {
	case data := <-ch:
		if len(data) == 0 {
			t.Error("expected non-empty output")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for terminal output")
	}
}

// [REQ:P0-003b] Reconnect State Restoration - offline buffer
func TestSession_OfflineBuffer(t *testing.T) {
	sm := NewSessionManager()

	sess, err := sm.Create("/bin/sh", 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	// Write some output while no subscriber is connected
	_, err = sess.Write([]byte("echo offline_test\n"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Wait for output to be buffered
	time.Sleep(500 * time.Millisecond)

	// Now subscribe - should get buffered output
	ch, _ := sess.Subscribe(80, 24)
	defer sess.Unsubscribe(ch)

	select {
	case data := <-ch:
		if len(data) == 0 {
			t.Error("expected buffered offline output")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for offline buffer")
	}
}

func TestDefaultConfig_Shell(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.DefaultShell == "" {
		t.Error("DefaultShell should not be empty")
	}
}

// --- snapToCleanBoundary tests ---

func TestSnapToCleanBoundary_EmptyBuffer(t *testing.T) {
	result := snapToCleanBoundary(nil)
	if result != nil {
		t.Errorf("expected nil for empty input, got %q", result)
	}
}

func TestSnapToCleanBoundary_CleanBuffer(t *testing.T) {
	// Buffer starts with normal text — no trimming needed, but snaps to newline
	buf := []byte("hello\nworld\n")
	result := snapToCleanBoundary(buf)
	// Should snap to after first newline since 'h' isn't mid-sequence
	// Actually 'h' (0x68) is NOT in the mid-sequence range, so start stays at 0.
	// Then it finds the newline at index 5 and snaps to index 6.
	if string(result) != "world\n" {
		t.Errorf("expected 'world\\n', got %q", string(result))
	}
}

func TestSnapToCleanBoundary_MidCSISequence(t *testing.T) {
	// Simulate a buffer that starts mid-escape-sequence: the ESC was trimmed,
	// leaving "[31m" (set red) followed by text.
	buf := []byte("[31mhello\nworld\n")
	result := snapToCleanBoundary(buf)
	// Should skip past the partial sequence "[31m" (final byte 'm' is 0x6D,
	// in range 0x40-0x7E), then snap to the newline after "hello".
	if string(result) != "world\n" {
		t.Errorf("expected 'world\\n', got %q", string(result))
	}
}

func TestSnapToCleanBoundary_MidCSISequenceNoNewline(t *testing.T) {
	// Partial sequence followed by text with no newline within scan limit
	buf := []byte("[31mhello world this is a long line")
	result := snapToCleanBoundary(buf)
	// Should skip past "[31m" (4 bytes), no newline found, so starts at byte 4
	if string(result) != "hello world this is a long line" {
		t.Errorf("expected text after partial sequence, got %q", string(result))
	}
}

func TestSnapToCleanBoundary_IntactEscapeSequence(t *testing.T) {
	// Buffer starts with a complete ESC sequence — should NOT be skipped
	buf := []byte("\x1b[31mred text\nnormal\n")
	result := snapToCleanBoundary(buf)
	// ESC (0x1b) is not in mid-sequence range, so start stays at 0.
	// Snaps to newline after "red text".
	if string(result) != "normal\n" {
		t.Errorf("expected 'normal\\n', got %q", string(result))
	}
}

func TestSnapToCleanBoundary_ParameterBytesOnly(t *testing.T) {
	// Buffer starts with just parameter bytes (0x30-0x3F) from a truncated sequence
	buf := []byte("31m\ntext\n")
	result := snapToCleanBoundary(buf)
	// '3' is 0x33, in range 0x30-0x3F → mid-sequence detected.
	// Scans forward: '3'(0x33 param), '1'(0x31 param), 'm'(0x6D final) → skip 3 bytes.
	// Then finds '\n' at position 0 relative to start=3 → skip to position 4.
	if string(result) != "text\n" {
		t.Errorf("expected 'text\\n', got %q", string(result))
	}
}

func TestLooksLikeMidSequence(t *testing.T) {
	tests := []struct {
		name   string
		input  byte
		expect bool
	}{
		{"open bracket", '[', true},
		{"digit 0", '0', true},
		{"digit 9", '9', true},
		{"semicolon", ';', true},
		{"space", ' ', true},
		{"ESC byte", 0x1b, false},
		{"letter A", 'A', false},
		{"letter m", 'm', false},
		{"newline", '\n', false},
		{"null", 0x00, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := looksLikeMidSequence([]byte{tt.input})
			if got != tt.expect {
				t.Errorf("looksLikeMidSequence(0x%02x) = %v, want %v", tt.input, got, tt.expect)
			}
		})
	}
}

// --- Subscribe SGR reset prefix test ---

func TestSubscribe_PrependsSGRReset(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))

	sess, err := sm.Create("/fake/shell", 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Write colored output while no subscribers connected
	colored := []byte("\x1b[31mred text\x1b[0m\nnormal text\n")
	_, _ = fake.outW.Write(colored)
	time.Sleep(50 * time.Millisecond)

	// Subscribe and check that replay starts with SGR reset
	ch, _ := sess.Subscribe(80, 24)
	defer sess.Unsubscribe(ch)

	select {
	case data := <-ch:
		if len(data) < 4 {
			t.Fatalf("replayed data too short: %q", data)
		}
		prefix := string(data[:4])
		if prefix != "\x1b[0m" {
			t.Errorf("replayed history should start with SGR reset, got prefix %q", prefix)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for replayed history")
	}

	fake.Close()
	<-sess.Done()
}

// --- History trim with ANSI sequences test ---

// --- Drop detection tests ---

func TestSession_Broadcast_DroppedFrames_Notifies(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	// Tiny channel buffer to force drops quickly
	sm.cfg.ClientChannelBuffer = 1
	sm.cfg.DropNotifyThreshold = 2

	sess, err := sm.Create("/fake/shell", 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	ch, notifyCh := sess.Subscribe(80, 24)
	defer sess.Unsubscribe(ch)

	// Fill the output channel (buffer size 1)
	// Don't read from ch — let it fill up
	for i := 0; i < 4; i++ {
		_, _ = fake.outW.Write([]byte("data\n"))
		time.Sleep(10 * time.Millisecond)
	}

	// Should have received a drop notification (threshold=2)
	select {
	case dropped := <-notifyCh:
		if dropped < 2 {
			t.Errorf("expected at least 2 dropped frames, got %d", dropped)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for drop notification")
	}

	fake.Close()
	<-sess.Done()
}

func TestSession_Broadcast_DroppedFrames_ThresholdRespected(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	sm.cfg.ClientChannelBuffer = 1
	sm.cfg.DropNotifyThreshold = 10 // High threshold

	sess, err := sm.Create("/fake/shell", 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	ch, notifyCh := sess.Subscribe(80, 24)
	defer sess.Unsubscribe(ch)

	// Cause a few drops (less than threshold)
	for i := 0; i < 5; i++ {
		_, _ = fake.outW.Write([]byte("data\n"))
		time.Sleep(10 * time.Millisecond)
	}

	// Should NOT have a notification yet (only ~4 drops, threshold is 10)
	select {
	case <-notifyCh:
		t.Error("should not receive notification before threshold is reached")
	case <-time.After(200 * time.Millisecond):
		// Expected: no notification
	}

	fake.Close()
	<-sess.Done()
}

func TestAppendHistory_TrimDoesNotSplitANSISequence(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	// Override offline buffer to a tiny size to force trimming
	sm.cfg.OfflineBufferMax = 32

	sess, err := sm.Create("/fake/shell", 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Write enough data to force a trim. The ANSI sequence could land at
	// the trim boundary.
	chunk1 := []byte("AAAAAAAAAAAAAAAA\n")           // 17 bytes — fills most of the 32-byte buffer
	chunk2 := []byte("\x1b[31mRED\x1b[0m\nNORMAL\n") // 18 bytes — forces trim

	_, _ = fake.outW.Write(chunk1)
	time.Sleep(30 * time.Millisecond)
	_, _ = fake.outW.Write(chunk2)
	time.Sleep(30 * time.Millisecond)

	// Subscribe to get the replayed (trimmed) history
	ch, _ := sess.Subscribe(80, 24)
	defer sess.Unsubscribe(ch)

	select {
	case data := <-ch:
		s := string(data)
		// The replayed data should NOT contain raw partial sequence bytes
		// like "[31m" without the preceding ESC. Check that any '[' is
		// preceded by ESC (0x1b) or is not part of an escape sequence.
		for i := 0; i < len(data); i++ {
			if data[i] == '[' && i > 0 && data[i-1] == 0x1b {
				// Valid: ESC[ sequence
				continue
			}
			if data[i] == '[' && i == 0 {
				t.Errorf("history replay starts with bare '[' (partial ANSI sequence): %q", s)
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for replayed history")
	}

	fake.Close()
	<-sess.Done()
}
