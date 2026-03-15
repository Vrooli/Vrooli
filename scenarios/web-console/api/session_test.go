package main

import (
	"bytes"
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

// [REQ:P0-002c] Terminal Resize Handling — last writer wins
func TestSession_Resize(t *testing.T) {
	sm := NewSessionManager()

	sess, err := sm.Create("", 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	ch, _ := sess.Subscribe()
	defer sess.Unsubscribe(ch)

	sess.Resize(120, 40)

	cols, rows := sess.EffectiveSize()
	if cols != 120 {
		t.Errorf("expected cols=120, got %d", cols)
	}
	if rows != 40 {
		t.Errorf("expected rows=40, got %d", rows)
	}
}

// [REQ:P0-002c] Last resize wins regardless of which client sent it
func TestSession_Resize_LastWriterWins(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))

	sess, err := sm.Create("/fake/shell", 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	ch1, _ := sess.Subscribe()
	defer sess.Unsubscribe(ch1)
	ch2, _ := sess.Subscribe()
	defer sess.Unsubscribe(ch2)

	// Desktop resizes to 120x40
	sess.Resize(120, 40)
	cols, rows := sess.EffectiveSize()
	if cols != 120 || rows != 40 {
		t.Errorf("after first resize: expected 120x40, got %dx%d", cols, rows)
	}

	// Phone resizes to 60x20 — last writer wins (not max)
	sess.Resize(60, 20)
	cols, rows = sess.EffectiveSize()
	if cols != 60 || rows != 20 {
		t.Errorf("after second resize: expected 60x20, got %dx%d", cols, rows)
	}

	fake.Close()
	<-sess.Done()
}

// [REQ:P0-002c] PTY size unchanged when all clients disconnect
func TestSession_NoClients_SizeUnchanged(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))

	sess, err := sm.Create("/fake/shell", 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	ch, _ := sess.Subscribe()
	sess.Resize(120, 40)
	sess.Unsubscribe(ch)

	// PTY size should remain at last resize value
	cols, rows := sess.EffectiveSize()
	if cols != 120 || rows != 40 {
		t.Errorf("expected 120x40 after client leaves, got %dx%d", cols, rows)
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

	ch, _ := sess.Subscribe()
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
	ch, _ := sess.Subscribe()
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
	ch, _ := sess.Subscribe()
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

// --- Coalescing tests ---

func TestSession_Broadcast_Coalescing_Notifies(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	// Tiny channel buffer to trigger coalescing quickly.
	sm.cfg.ClientChannelBuffer = 1
	sm.cfg.CoalesceNotifyThreshold = 2

	sess, err := sm.Create("/fake/shell", 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	ch, notifyCh := sess.Subscribe()
	defer sess.Unsubscribe(ch)

	// Fill the output channel (buffer size 1).
	// Don't read from ch — let coalescing kick in.
	for i := 0; i < 4; i++ {
		_, _ = fake.outW.Write([]byte("data\n"))
		time.Sleep(10 * time.Millisecond)
	}

	// Should have received a coalescing notification (threshold=2).
	select {
	case coalesced := <-notifyCh:
		if coalesced < 2 {
			t.Errorf("expected at least 2 coalesced frames, got %d", coalesced)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for coalescing notification")
	}

	fake.Close()
	<-sess.Done()
}

func TestSession_Broadcast_CoalescingThreshold(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	sm.cfg.ClientChannelBuffer = 1
	sm.cfg.CoalesceNotifyThreshold = 10 // High threshold

	sess, err := sm.Create("/fake/shell", 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	ch, notifyCh := sess.Subscribe()
	defer sess.Unsubscribe(ch)

	// Cause a few coalesced frames (less than threshold).
	for i := 0; i < 5; i++ {
		_, _ = fake.outW.Write([]byte("data\n"))
		time.Sleep(10 * time.Millisecond)
	}

	// Should NOT have a notification yet (only ~4 coalesced, threshold is 10).
	select {
	case <-notifyCh:
		t.Error("should not receive notification before threshold is reached")
	case <-time.After(200 * time.Millisecond):
		// Expected: no notification.
	}

	fake.Close()
	<-sess.Done()
}

func TestSession_Broadcast_CoalescingPreservesAllData(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	// Tiny buffer to force coalescing on every frame after the first.
	sm.cfg.ClientChannelBuffer = 1
	sm.cfg.CoalesceNotifyThreshold = 100 // suppress notifications

	sess, err := sm.Create("/fake/shell", 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	ch, _ := sess.Subscribe()
	defer sess.Unsubscribe(ch)

	// Write 20 small messages without reading from the channel.
	var expected []byte
	for i := 0; i < 20; i++ {
		msg := []byte("msg\n")
		expected = append(expected, msg...)
		_, _ = fake.outW.Write(msg)
		time.Sleep(5 * time.Millisecond)
	}

	// Drain: read what's in the channel, then flush pending.
	var received []byte
	for {
		select {
		case data := <-ch:
			received = append(received, data...)
			sess.FlushPending(ch)
		case <-time.After(200 * time.Millisecond):
			goto done
		}
	}
done:
	// Drain anything flushed.
	for {
		select {
		case data := <-ch:
			received = append(received, data...)
		case <-time.After(100 * time.Millisecond):
			goto verify
		}
	}
verify:
	if string(received) != string(expected) {
		t.Errorf("coalesced data mismatch:\n  got  %d bytes\n  want %d bytes", len(received), len(expected))
	}

	fake.Close()
	<-sess.Done()
}

func TestSession_Broadcast_CoalescingCapRespected(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	sm.cfg.ClientChannelBuffer = 1
	sm.cfg.OfflineBufferMax = 32 // tiny cap for pending coalesced data
	sm.cfg.CoalesceNotifyThreshold = 100

	sess, err := sm.Create("/fake/shell", 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	ch, _ := sess.Subscribe()
	defer sess.Unsubscribe(ch)

	// Write more data than the cap allows while consumer is blocked.
	for i := 0; i < 10; i++ {
		_, _ = fake.outW.Write([]byte("ABCDEFGH")) // 8 bytes × 10 = 80 bytes total
		time.Sleep(5 * time.Millisecond)
	}

	// Drain channel + flush.
	var received []byte
	for {
		select {
		case data := <-ch:
			received = append(received, data...)
			sess.FlushPending(ch)
		case <-time.After(200 * time.Millisecond):
			goto done2
		}
	}
done2:
	for {
		select {
		case data := <-ch:
			received = append(received, data...)
		case <-time.After(100 * time.Millisecond):
			goto verify2
		}
	}
verify2:
	// The most recent data should be preserved (the tail of the stream).
	if len(received) == 0 {
		t.Fatal("expected some data after coalescing")
	}
	// With cap=32, pending is trimmed to last 32 bytes. Combined with what
	// was in the channel before coalescing, we should have less than full 80 bytes.
	if len(received) >= 80 {
		t.Errorf("expected data to be capped, but got all %d bytes", len(received))
	}

	fake.Close()
	<-sess.Done()
}

func TestSession_FlushPending_UnknownChannel(t *testing.T) {
	fake := newFakePTYWithOutput()
	defer fake.Close()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	sess, err := sm.Create("/fake/shell", 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// FlushPending with an unsubscribed channel should not panic.
	unknownCh := make(chan []byte, 1)
	sess.FlushPending(unknownCh)
}

// --- UTF-8 boundary tests ---

func TestSplitCompleteUTF8(t *testing.T) {
	tests := []struct {
		name          string
		input         []byte
		wantComplete  []byte
		wantRemainder []byte
	}{
		{"ascii_only", []byte("hello"), []byte("hello"), nil},
		{"complete_2byte", []byte{0xc3, 0xa9}, []byte{0xc3, 0xa9}, nil},                            // é
		{"split_2byte_1of2", []byte{'A', 0xc3}, []byte{'A'}, []byte{0xc3}},                         // A + incomplete é
		{"complete_3byte", []byte{0xe2, 0x9c, 0x93}, []byte{0xe2, 0x9c, 0x93}, nil},                // ✓
		{"split_3byte_1of3", []byte{'A', 0xe2}, []byte{'A'}, []byte{0xe2}},                         // A + 1 byte of ✓
		{"split_3byte_2of3", []byte{'A', 0xe2, 0x9c}, []byte{'A'}, []byte{0xe2, 0x9c}},             // A + 2 bytes of ✓
		{"complete_4byte", []byte{0xf0, 0x9f, 0x98, 0x80}, []byte{0xf0, 0x9f, 0x98, 0x80}, nil},    // 😀
		{"split_4byte_1of4", []byte{'A', 0xf0}, []byte{'A'}, []byte{0xf0}},                         // A + 1 byte of 😀
		{"split_4byte_2of4", []byte{'A', 0xf0, 0x9f}, []byte{'A'}, []byte{0xf0, 0x9f}},             // A + 2 bytes of 😀
		{"split_4byte_3of4", []byte{'A', 0xf0, 0x9f, 0x98}, []byte{'A'}, []byte{0xf0, 0x9f, 0x98}}, // A + 3 bytes of 😀
		{"empty", nil, nil, nil},
		{"orphan_continuation", []byte{0x80}, []byte{0x80}, nil},                                                                 // pass through
		{"only_leading_byte", []byte{0xc3}, nil, []byte{0xc3}},                                                                   // just a leading byte
		{"ascii_then_complete_multibyte", []byte{'H', 'i', 0xc3, 0xa9}, []byte{'H', 'i', 0xc3, 0xa9}, nil},                       // "Hié"
		{"multibyte_then_split", []byte{0xe2, 0x9c, 0x93, 0xf0, 0x9f, 0x98}, []byte{0xe2, 0x9c, 0x93}, []byte{0xf0, 0x9f, 0x98}}, // ✓ + 3/4 of 😀
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotComplete, gotRemainder := splitCompleteUTF8(tt.input)
			if string(gotComplete) != string(tt.wantComplete) {
				t.Errorf("complete: got %v, want %v", gotComplete, tt.wantComplete)
			}
			if string(gotRemainder) != string(tt.wantRemainder) {
				t.Errorf("remainder: got %v, want %v", gotRemainder, tt.wantRemainder)
			}
		})
	}
}

func TestReadLoop_UTF8BoundarySafety(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))

	sess, err := sm.Create("/fake/shell", 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	ch, _ := sess.Subscribe()
	defer sess.Unsubscribe(ch)

	// Write a 3-byte UTF-8 char (✓ = 0xe2 0x9c 0x93) split across two writes.
	// First write: ASCII + first 2 bytes of the char.
	_, _ = fake.outW.Write([]byte{'A', 0xe2, 0x9c})
	time.Sleep(50 * time.Millisecond)
	// Second write: final byte + more ASCII.
	_, _ = fake.outW.Write([]byte{0x93, 'B'})
	time.Sleep(50 * time.Millisecond)

	// Collect messages from the channel.
	var received []byte
	for {
		select {
		case data := <-ch:
			received = append(received, data...)
		case <-time.After(300 * time.Millisecond):
			goto verify
		}
	}
verify:
	expected := "A\xe2\x9c\x93B"
	if string(received) != expected {
		t.Errorf("UTF-8 boundary: got %q, want %q", received, expected)
	}
	// Verify no replacement characters.
	if bytes.Contains(received, []byte{0xef, 0xbf, 0xbd}) {
		t.Error("output contains U+FFFD replacement character — UTF-8 split was not handled")
	}

	fake.Close()
	<-sess.Done()
}

// --- History trim tests ---

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
	ch, _ := sess.Subscribe()
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
