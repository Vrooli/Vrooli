package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// [REQ:P0-002a] PTY Session Backend
func TestSessionManager_Create(t *testing.T) {
	sm := NewSessionManager()

	sess, err := sm.Create("", 80, 24, "", nil)
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

	s1, err := sm.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create s1 failed: %v", err)
	}
	defer func() { _ = sm.Delete(s1.ID) }()

	s2, err := sm.Create("", 120, 40, "", nil)
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

	sess, err := sm.Create("", 80, 24, "", nil)
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

	sess, err := sm.Create("", 80, 24, "", nil)
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

	sess, err := sm.Create("", 80, 24, "", nil)
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

	sess, err := sm.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	sub := sess.Subscribe(0)
	defer sess.Unsubscribe(sub.OutputCh)

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

	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	sub1 := sess.Subscribe(0)
	defer sess.Unsubscribe(sub1.OutputCh)
	sub2 := sess.Subscribe(0)
	defer sess.Unsubscribe(sub2.OutputCh)

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

	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	sub := sess.Subscribe(0)
	sess.Resize(120, 40)
	sess.Unsubscribe(sub.OutputCh)

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

	sess, err := sm.Create("/bin/sh", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	sub := sess.Subscribe(0)
	defer sess.Unsubscribe(sub.OutputCh)

	// Write to stdin - the shell should echo something back
	err = sess.WriteInput([]byte("echo hello\n"), InputKindKeystroke)
	if err != nil {
		t.Fatalf("WriteInput failed: %v", err)
	}

	// Read output with timeout
	select {
	case data := <-sub.OutputCh:
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

	sess, err := sm.Create("/bin/sh", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	// Write some output while no subscriber is connected
	err = sess.WriteInput([]byte("echo offline_test\n"), InputKindKeystroke)
	if err != nil {
		t.Fatalf("WriteInput failed: %v", err)
	}

	// Wait for output to be buffered
	time.Sleep(500 * time.Millisecond)

	// Now subscribe - should get buffered output
	sub := sess.Subscribe(0)
	defer sess.Unsubscribe(sub.OutputCh)

	select {
	case data := <-sub.OutputCh:
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
		input  []byte
		expect bool
	}{
		{"open bracket", []byte("["), true},
		{"bracket_K", []byte("[K"), true},
		{"digit_then_m", []byte("31m"), true},
		{"cursor_position", []byte("1;1H"), true},
		{"lone_digit", []byte("5"), false},
		{"lone_space", []byte(" "), false},
		{"semicolon_alone", []byte(";"), false},
		{"ESC_byte", []byte{0x1b}, false},
		{"letter_A", []byte("A"), false},
		{"letter_m", []byte("m"), false},
		{"newline", []byte("\n"), false},
		{"null", []byte{0x00}, false},
		{"empty", []byte{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := looksLikeMidSequence(tt.input)
			if got != tt.expect {
				t.Errorf("looksLikeMidSequence(%q) = %v, want %v", tt.input, got, tt.expect)
			}
		})
	}
}

// --- Subscribe SGR reset prefix test ---

func TestSubscribe_PrependsSGRReset(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))

	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Write colored output while no subscribers connected
	colored := []byte("\x1b[31mred text\x1b[0m\nnormal text\n")
	_, _ = fake.outW.Write(colored)
	time.Sleep(50 * time.Millisecond)

	// Subscribe and check that replay starts with SGR reset
	sub := sess.Subscribe(0)
	defer sess.Unsubscribe(sub.OutputCh)

	select {
	case data := <-sub.OutputCh:
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

	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	sub := sess.Subscribe(0)
	defer sess.Unsubscribe(sub.OutputCh)

	// Fill the output channel (buffer size 1).
	// Don't read from sub.OutputCh — let coalescing kick in.
	for i := 0; i < 4; i++ {
		_, _ = fake.outW.Write([]byte("data\n"))
		time.Sleep(10 * time.Millisecond)
	}

	// Should have received a coalescing notification (threshold=2).
	select {
	case coalesced := <-sub.NotifyCh:
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

	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	sub := sess.Subscribe(0)
	defer sess.Unsubscribe(sub.OutputCh)

	// Cause a few coalesced frames (less than threshold).
	for i := 0; i < 5; i++ {
		_, _ = fake.outW.Write([]byte("data\n"))
		time.Sleep(10 * time.Millisecond)
	}

	// Should NOT have a notification yet (only ~4 coalesced, threshold is 10).
	select {
	case <-sub.NotifyCh:
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

	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	sub := sess.Subscribe(0)
	defer sess.Unsubscribe(sub.OutputCh)

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
		case data := <-sub.OutputCh:
			received = append(received, data...)
			sess.FlushPending(sub.OutputCh)
		case <-time.After(200 * time.Millisecond):
			goto done
		}
	}
done:
	// Drain anything flushed.
	for {
		select {
		case data := <-sub.OutputCh:
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

	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	sub := sess.Subscribe(0)
	defer sess.Unsubscribe(sub.OutputCh)

	// Write more data than the cap allows while consumer is blocked.
	for i := 0; i < 10; i++ {
		_, _ = fake.outW.Write([]byte("ABCDEFGH")) // 8 bytes × 10 = 80 bytes total
		time.Sleep(5 * time.Millisecond)
	}

	// Drain channel + flush.
	var received []byte
	for {
		select {
		case data := <-sub.OutputCh:
			received = append(received, data...)
			sess.FlushPending(sub.OutputCh)
		case <-time.After(200 * time.Millisecond):
			goto done2
		}
	}
done2:
	for {
		select {
		case data := <-sub.OutputCh:
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
	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
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

	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	sub := sess.Subscribe(0)
	defer sess.Unsubscribe(sub.OutputCh)

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
		case data := <-sub.OutputCh:
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

	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
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
	sub := sess.Subscribe(0)
	defer sess.Unsubscribe(sub.OutputCh)

	select {
	case data := <-sub.OutputCh:
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

// --- Coalescing cap ANSI boundary tests ---

func TestSession_Broadcast_CoalescingCapSnapsToCleanBoundary(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	sm.cfg.ClientChannelBuffer = 1
	sm.cfg.OfflineBufferMax = 32 // tiny cap to force trimming
	sm.cfg.CoalesceNotifyThreshold = 100

	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	sub := sess.Subscribe(0)
	defer sess.Unsubscribe(sub.OutputCh)

	// Write data containing ANSI sequences. The coalescing cap should trim
	// at an ANSI-clean boundary rather than slicing mid-escape-sequence.
	for i := 0; i < 10; i++ {
		_, _ = fake.outW.Write([]byte("\x1b[31mRED\x1b[0m\n"))
		time.Sleep(5 * time.Millisecond)
	}

	// Drain channel + flush pending.
	var received []byte
	for {
		select {
		case data := <-sub.OutputCh:
			received = append(received, data...)
			sess.FlushPending(sub.OutputCh)
		case <-time.After(200 * time.Millisecond):
			goto coalesceDone
		}
	}
coalesceDone:
	for {
		select {
		case data := <-sub.OutputCh:
			received = append(received, data...)
		case <-time.After(100 * time.Millisecond):
			goto coalesceVerify
		}
	}
coalesceVerify:
	if len(received) == 0 {
		t.Fatal("expected data after coalescing cap")
	}
	// Verify no orphaned partial ANSI sequences: every '[' should be
	// preceded by ESC (0x1b), not appear bare as part of a CSI sequence.
	for i, b := range received {
		if b == '[' && (i == 0 || received[i-1] != 0x1b) {
			if looksLikeMidSequence(received[i:]) {
				t.Errorf("coalesced data contains orphaned CSI at byte %d", i)
			}
		}
	}

	fake.Close()
	<-sess.Done()
}

// --- Slice aliasing tests ---

func TestAppendHistory_NoSliceAliasing(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	sm.cfg.OfflineBufferMax = 32

	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Write data that fills the buffer and triggers the trim path.
	_, _ = fake.outW.Write([]byte("AAAAAAAAAAAAAAAA\n")) // 17 bytes
	time.Sleep(30 * time.Millisecond)
	_, _ = fake.outW.Write([]byte("BBBBBBBBBBBBBBBB\n")) // 17 bytes — triggers trim
	time.Sleep(30 * time.Millisecond)

	// Subscribe to see the current history state.
	sub1 := sess.Subscribe(0)
	select {
	case data := <-sub1.OutputCh:
		if !bytes.Contains(data, []byte("BBBB")) {
			t.Errorf("history should contain second write, got %q", data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for history")
	}
	sess.Unsubscribe(sub1.OutputCh)

	// Write more data to trigger another trim cycle.
	_, _ = fake.outW.Write([]byte("CCCCCCCCCCCCCCCC\n")) // 17 bytes — triggers trim again
	time.Sleep(30 * time.Millisecond)

	// Verify the history is not corrupted by slice aliasing.
	sub2 := sess.Subscribe(0)
	defer sess.Unsubscribe(sub2.OutputCh)

	select {
	case data := <-sub2.OutputCh:
		if !bytes.Contains(data, []byte("CCCC")) {
			t.Errorf("history should contain third write after re-trim, got %q", data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for history after second trim")
	}

	fake.Close()
	<-sess.Done()
}

// --- History chunking tests ---

func TestSubscribe_ChunksLargeHistory(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	// Allow large history so it exceeds historyChunkSize.
	sm.cfg.OfflineBufferMax = 200 * 1024 // 200 KB

	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Write 100KB of data (exceeds historyChunkSize of 64KB).
	bigData := bytes.Repeat([]byte("X"), 100*1024)
	_, _ = fake.outW.Write(bigData)
	time.Sleep(100 * time.Millisecond)

	sub := sess.Subscribe(0)
	defer sess.Unsubscribe(sub.OutputCh)

	var chunks [][]byte
	for {
		select {
		case data := <-sub.OutputCh:
			if data == nil {
				continue // skip nil sentinel
			}
			chunks = append(chunks, data)
		case <-time.After(500 * time.Millisecond):
			goto verifyChunks
		}
	}
verifyChunks:
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks for 100KB history, got %d", len(chunks))
	}
	// First chunk should start with SGR reset.
	if !bytes.HasPrefix(chunks[0], sgrReset) {
		t.Errorf("first chunk should start with SGR reset, got prefix %q", chunks[0][:min(8, len(chunks[0]))])
	}
	// Each chunk should be at most historyChunkSize bytes.
	for i, chunk := range chunks {
		if len(chunk) > historyChunkSize {
			t.Errorf("chunk %d is %d bytes, exceeds historyChunkSize (%d)", i, len(chunk), historyChunkSize)
		}
	}
	// Concatenation should equal SGR reset + original data.
	var combined []byte
	for _, chunk := range chunks {
		combined = append(combined, chunk...)
	}
	expected := append(append([]byte(nil), sgrReset...), bigData...)
	if !bytes.Equal(combined, expected) {
		t.Errorf("chunked replay mismatch: got %d bytes, want %d bytes", len(combined), len(expected))
	}

	fake.Close()
	<-sess.Done()
}

func TestSubscribe_SmallHistoryNotChunked(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))

	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Write small data (well under historyChunkSize).
	_, _ = fake.outW.Write([]byte("small output\n"))
	time.Sleep(50 * time.Millisecond)

	sub := sess.Subscribe(0)
	defer sess.Unsubscribe(sub.OutputCh)

	var chunks [][]byte
	for {
		select {
		case data := <-sub.OutputCh:
			if data == nil {
				continue // skip nil sentinel
			}
			chunks = append(chunks, data)
		case <-time.After(300 * time.Millisecond):
			goto verifySmall
		}
	}
verifySmall:
	if len(chunks) != 1 {
		t.Errorf("expected exactly 1 chunk for small history, got %d", len(chunks))
	}

	fake.Close()
	<-sess.Done()
}

// --- Subscribe hadHistory return value + nil sentinel tests ---

func TestSubscribe_ReturnsHadHistoryFalse(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))

	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Fresh session with no output history.
	sub := sess.Subscribe(0)
	if sub.HadData {
		t.Error("expected HadData=false for session with no output")
	}

	fake.Close()
	<-sess.Done()
}

func TestSubscribe_ReturnsHadHistoryTrue(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))

	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Write some output so history is non-empty.
	_, _ = fake.outW.Write([]byte("some output"))
	time.Sleep(50 * time.Millisecond)

	sub := sess.Subscribe(0)
	if !sub.HadData {
		t.Error("expected HadData=true for session with output history")
	}

	fake.Close()
	<-sess.Done()
}

func TestSubscribe_NilSentinelAfterHistoryChunks(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))

	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	_, _ = fake.outW.Write([]byte("hello world"))
	time.Sleep(50 * time.Millisecond)

	sub := sess.Subscribe(0)
	defer sess.Unsubscribe(sub.OutputCh)

	if !sub.HadData {
		t.Fatal("expected HadData=true")
	}

	// First message should be the history chunk (non-nil).
	select {
	case data := <-sub.OutputCh:
		if data == nil {
			t.Fatal("expected history chunk, got nil sentinel")
		}
		if !bytes.Contains(data, []byte("hello world")) {
			t.Errorf("history chunk should contain 'hello world', got %q", data)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for history chunk")
	}

	// Next message should be the nil sentinel.
	select {
	case data := <-sub.OutputCh:
		if data != nil {
			t.Errorf("expected nil sentinel after history, got %d bytes", len(data))
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for nil sentinel")
	}

	fake.Close()
	<-sess.Done()
}

func TestSubscribe_NoSentinelWithoutHistory(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))

	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// No output written — channel should be empty (no sentinel).
	sub := sess.Subscribe(0)
	defer sess.Unsubscribe(sub.OutputCh)

	if sub.HadData {
		t.Fatal("expected HadData=false")
	}

	select {
	case data := <-sub.OutputCh:
		t.Errorf("expected empty channel, got %d bytes (nil=%v)", len(data), data == nil)
	case <-time.After(100 * time.Millisecond):
		// Expected: nothing in channel.
	}

	fake.Close()
	<-sess.Done()
}

func TestSubscribe_ChannelCapacityFitsChunksAndSentinel(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	// Tiny buffer — history chunks + sentinel should still fit without deadlock.
	sm.cfg.ClientChannelBuffer = 1
	sm.cfg.OfflineBufferMax = 200 * 1024

	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Write 100KB to produce multiple chunks (>1 × 64KB).
	bigData := bytes.Repeat([]byte("X"), 100*1024)
	_, _ = fake.outW.Write(bigData)
	time.Sleep(100 * time.Millisecond)

	// Subscribe must not deadlock despite ClientChannelBuffer=1.
	done := make(chan struct{})
	go func() {
		defer close(done)
		sub := sess.Subscribe(0)
		defer sess.Unsubscribe(sub.OutputCh)
		if !sub.HadData {
			t.Error("expected HadData=true")
		}
		// Drain all chunks and verify we get the nil sentinel.
		var gotSentinel bool
		for {
			select {
			case data := <-sub.OutputCh:
				if data == nil {
					gotSentinel = true
					goto verify
				}
			case <-time.After(2 * time.Second):
				goto verify
			}
		}
	verify:
		if !gotSentinel {
			t.Error("expected nil sentinel after history chunks")
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Subscribe appears to have deadlocked")
	}

	fake.Close()
	<-sess.Done()
}

func TestFlushPending_ChunksLargeData(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	sm.cfg.ClientChannelBuffer = 1 // force coalescing
	sm.cfg.OfflineBufferMax = 200 * 1024
	sm.cfg.CoalesceNotifyThreshold = 10000

	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	sub := sess.Subscribe(0)
	defer sess.Unsubscribe(sub.OutputCh)

	// Drain history replay if any
	drainWithTimeout := func() {
		for {
			select {
			case <-sub.OutputCh:
			case <-time.After(50 * time.Millisecond):
				return
			}
		}
	}
	drainWithTimeout()

	// Write 150KB in small chunks to force coalescing
	chunkData := make([]byte, 1024)
	for i := range chunkData {
		chunkData[i] = byte('A' + (i % 26))
	}
	for i := 0; i < 150; i++ {
		_, _ = fake.outW.Write(chunkData)
		time.Sleep(2 * time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond)

	// Drain the one message in the channel (first broadcast that fit)
	select {
	case <-sub.OutputCh:
	case <-time.After(200 * time.Millisecond):
	}

	// Now flush pending data — channel has room
	sess.FlushPending(sub.OutputCh)

	// Collect all flushed chunks
	var chunks [][]byte
	for {
		select {
		case data := <-sub.OutputCh:
			chunks = append(chunks, data)
			sess.FlushPending(sub.OutputCh) // try to flush more
		case <-time.After(200 * time.Millisecond):
			goto verify
		}
	}
verify:
	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk from FlushPending")
	}
	for i, chunk := range chunks {
		if len(chunk) > historyChunkSize {
			t.Errorf("chunk %d is %d bytes, exceeds historyChunkSize (%d)", i, len(chunk), historyChunkSize)
		}
	}

	fake.Close()
	<-sess.Done()
}

func TestFlushPending_StopsWhenChannelFull(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	sm.cfg.ClientChannelBuffer = 1 // tiny channel — fills immediately
	sm.cfg.OfflineBufferMax = 200 * 1024
	sm.cfg.CoalesceNotifyThreshold = 10000

	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	sub := sess.Subscribe(0)
	defer sess.Unsubscribe(sub.OutputCh)

	// Drain initial history replay
	for {
		select {
		case <-sub.OutputCh:
		case <-time.After(50 * time.Millisecond):
			goto write
		}
	}
write:
	// Write 150KB to force coalescing
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte('X')
	}
	for i := 0; i < 150; i++ {
		_, _ = fake.outW.Write(data)
		time.Sleep(2 * time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond)

	// Drain the one message that fit in the channel
	select {
	case <-sub.OutputCh:
	case <-time.After(200 * time.Millisecond):
	}

	// Fill the channel with a dummy message so FlushPending can't send everything
	sub.OutputCh <- []byte("blocker")

	// FlushPending should send one chunk (if channel drains the blocker first) or stop
	sess.FlushPending(sub.OutputCh)

	// Drain everything and call FlushPending repeatedly
	var total int
	for {
		select {
		case d := <-sub.OutputCh:
			total += len(d)
			sess.FlushPending(sub.OutputCh)
		case <-time.After(200 * time.Millisecond):
			goto done
		}
	}
done:
	// We should have received some data (the blocker + flushed chunks)
	if total == 0 {
		t.Fatal("expected some data from FlushPending after draining")
	}

	fake.Close()
	<-sess.Done()
}

// --- Coalescing trim recovery tests ---

// TestDeliver_PrependsSGRResetOnTrim verifies that when the coalesced pending
// buffer exceeds offlineBufferMax and is trimmed, the remaining data is
// prefixed with an SGR reset sequence (\x1b[0m) to clear dangling color state.
func TestDeliver_PrependsSGRResetOnTrim(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	sm.cfg.ClientChannelBuffer = 1
	sm.cfg.OfflineBufferMax = 64 // tiny cap to force trimming
	sm.cfg.CoalesceNotifyThreshold = 100

	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	sub := sess.Subscribe(0)
	defer sess.Unsubscribe(sub.OutputCh)

	// Write enough data with ANSI color codes to overflow the coalescing cap.
	for i := 0; i < 20; i++ {
		_, _ = fake.outW.Write([]byte("\x1b[31mRED\x1b[0m\n"))
		time.Sleep(5 * time.Millisecond)
	}

	// Drain channel + flush pending to collect all coalesced data.
	var received []byte
	for {
		select {
		case data := <-sub.OutputCh:
			received = append(received, data...)
			sess.FlushPending(sub.OutputCh)
		case <-time.After(200 * time.Millisecond):
			goto sgrDrain
		}
	}
sgrDrain:
	for {
		select {
		case data := <-sub.OutputCh:
			received = append(received, data...)
		case <-time.After(100 * time.Millisecond):
			goto sgrVerify
		}
	}
sgrVerify:
	if len(received) == 0 {
		t.Fatal("expected data after coalescing")
	}

	// The coalesced data should contain an SGR reset from the trim.
	if !bytes.Contains(received, sgrReset) {
		t.Error("coalesced trimmed data should contain SGR reset prefix")
	}

	fake.Close()
	<-sess.Done()
}

// TestFlushPending_TriggersSIGWINCHAfterTrimmedDrain verifies that when the
// coalesced pending buffer was trimmed, FlushPending triggers a SIGWINCH
// (via pty.SetSize) after fully draining, so the shell redraws its screen.
func TestFlushPending_TriggersSIGWINCHAfterTrimmedDrain(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	sm.cfg.ClientChannelBuffer = 1
	sm.cfg.OfflineBufferMax = 64
	sm.cfg.CoalesceNotifyThreshold = 100

	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	sub := sess.Subscribe(0)
	defer sess.Unsubscribe(sub.OutputCh)

	// Record initial SetSize calls (from session creation).
	fake.mu.Lock()
	initialCalls := fake.setSizeCalls
	fake.mu.Unlock()

	// Write enough data to trigger coalescing trim.
	for i := 0; i < 20; i++ {
		_, _ = fake.outW.Write([]byte("\x1b[31mRED LINE DATA\x1b[0m\n"))
		time.Sleep(5 * time.Millisecond)
	}

	// Drain channel and flush pending until fully drained.
	for {
		select {
		case <-sub.OutputCh:
			sess.FlushPending(sub.OutputCh)
		case <-time.After(200 * time.Millisecond):
			goto sigwinchDrainRemainder
		}
	}
sigwinchDrainRemainder:
	for {
		select {
		case <-sub.OutputCh:
		case <-time.After(100 * time.Millisecond):
			goto sigwinchCheck
		}
	}
sigwinchCheck:
	fake.mu.Lock()
	extraCalls := fake.setSizeCalls - initialCalls
	gotCols := fake.cols
	gotRows := fake.rows
	fake.mu.Unlock()

	if extraCalls < 1 {
		t.Errorf("expected at least 1 extra SetSize call (SIGWINCH), got %d", extraCalls)
	}
	if gotCols != 80 || gotRows != 24 {
		t.Errorf("SIGWINCH SetSize should use session dims (80x24), got %dx%d", gotCols, gotRows)
	}

	fake.Close()
	<-sess.Done()
}

// TestFlushPending_NoSIGWINCHWithoutTrim verifies that FlushPending does NOT
// trigger a SIGWINCH when the coalesced data was not trimmed.
func TestFlushPending_NoSIGWINCHWithoutTrim(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	sm.cfg.ClientChannelBuffer = 1
	sm.cfg.OfflineBufferMax = 1 << 20 // 1MB — large enough to never trim
	sm.cfg.CoalesceNotifyThreshold = 100

	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	sub := sess.Subscribe(0)
	defer sess.Unsubscribe(sub.OutputCh)

	fake.mu.Lock()
	initialCalls := fake.setSizeCalls
	fake.mu.Unlock()

	// Write some data — not enough to trigger trim.
	for i := 0; i < 5; i++ {
		_, _ = fake.outW.Write([]byte("hello world\n"))
		time.Sleep(5 * time.Millisecond)
	}

	// Drain + flush.
	for {
		select {
		case <-sub.OutputCh:
			sess.FlushPending(sub.OutputCh)
		case <-time.After(200 * time.Millisecond):
			goto noTrimDrain
		}
	}
noTrimDrain:
	for {
		select {
		case <-sub.OutputCh:
		case <-time.After(100 * time.Millisecond):
			goto noTrimCheck
		}
	}
noTrimCheck:
	fake.mu.Lock()
	extraCalls := fake.setSizeCalls - initialCalls
	fake.mu.Unlock()

	if extraCalls != 0 {
		t.Errorf("expected 0 extra SetSize calls (no trim occurred), got %d", extraCalls)
	}

	fake.Close()
	<-sess.Done()
}

// TestFlushPending_SIGWINCHOnlyOncePerTrim verifies that when a trim occurs
// and FlushPending partially drains (channel fills mid-flush), the SIGWINCH
// is triggered exactly once — on the final drain, not the partial one.
func TestFlushPending_SIGWINCHOnlyOncePerTrim(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	sm.cfg.ClientChannelBuffer = 1
	// Set OfflineBufferMax > historyChunkSize so trimmed pending data requires
	// multiple 64KB chunks to drain, enabling a partial-flush scenario.
	sm.cfg.OfflineBufferMax = historyChunkSize + historyChunkSize/2 // 96KB
	sm.cfg.CoalesceNotifyThreshold = 100

	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	sub := sess.Subscribe(0)
	defer sess.Unsubscribe(sub.OutputCh)

	fake.mu.Lock()
	initialCalls := fake.setSizeCalls
	fake.mu.Unlock()

	// Write enough data to exceed OfflineBufferMax and trigger trim.
	// Each write is 1KB; we need > 96KB total coalesced.
	chunk := make([]byte, 1024)
	for i := range chunk {
		chunk[i] = byte('A' + (i % 26))
	}
	chunk[len(chunk)-1] = '\n'
	for i := 0; i < 120; i++ {
		_, _ = fake.outW.Write(chunk)
		time.Sleep(2 * time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond)

	// Drain one message from the channel — this makes room for FlushPending
	// to send one 64KB chunk, but the channel is too small for the rest.
	select {
	case <-sub.OutputCh:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected at least one message in channel")
	}

	// First FlushPending: partial drain (channel capacity = 1, pending > 64KB).
	sess.FlushPending(sub.OutputCh)

	fake.mu.Lock()
	callsAfterPartial := fake.setSizeCalls - initialCalls
	fake.mu.Unlock()

	if callsAfterPartial != 0 {
		t.Errorf("expected 0 extra SetSize calls after partial drain, got %d", callsAfterPartial)
	}

	// Now fully drain everything.
	for {
		select {
		case <-sub.OutputCh:
			sess.FlushPending(sub.OutputCh)
		case <-time.After(200 * time.Millisecond):
			goto onceFinalDrain
		}
	}
onceFinalDrain:
	for {
		select {
		case <-sub.OutputCh:
		case <-time.After(100 * time.Millisecond):
			goto onceFinalCheck
		}
	}
onceFinalCheck:
	fake.mu.Lock()
	totalExtraCalls := fake.setSizeCalls - initialCalls
	fake.mu.Unlock()

	if totalExtraCalls != 1 {
		t.Errorf("expected exactly 1 SIGWINCH SetSize call after full drain, got %d", totalExtraCalls)
	}

	fake.Close()
	<-sess.Done()
}

// --- Byte-offset resume tests ---

// TestSubscribe_TotalOutputBytesTracking verifies that TotalBytes in
// SubscribeResult correctly reflects the cumulative bytes written to the PTY.
func TestSubscribe_TotalOutputBytesTracking(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Write "hello" (5 bytes) and wait for readLoop to process.
	_, _ = fake.outW.Write([]byte("hello"))
	time.Sleep(100 * time.Millisecond)

	sub1 := sess.Subscribe(0)
	sess.Unsubscribe(sub1.OutputCh)
	if sub1.TotalBytes != 5 {
		t.Errorf("expected TotalBytes=5 after first write, got %d", sub1.TotalBytes)
	}

	// Write "world" (5 more bytes).
	_, _ = fake.outW.Write([]byte("world"))
	time.Sleep(100 * time.Millisecond)

	sub2 := sess.Subscribe(0)
	sess.Unsubscribe(sub2.OutputCh)
	if sub2.TotalBytes != 10 {
		t.Errorf("expected TotalBytes=10 after second write, got %d", sub2.TotalBytes)
	}

	fake.Close()
	<-sess.Done()
}

// TestSubscribe_ResumeValidOffset verifies that subscribing with a valid
// offset within the history buffer returns only the delta (no SGR reset).
func TestSubscribe_ResumeValidOffset(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Write 100 bytes of known output.
	payload := strings.Repeat("A", 100)
	_, _ = fake.outW.Write([]byte(payload))
	time.Sleep(100 * time.Millisecond)

	// Get full history to learn TotalBytes.
	full := sess.Subscribe(0)
	sess.Unsubscribe(full.OutputCh)
	if full.TotalBytes != 100 {
		t.Fatalf("expected TotalBytes=100, got %d", full.TotalBytes)
	}

	// Resume from offset 50 — should get last 50 bytes as delta.
	sub := sess.Subscribe(50)
	defer sess.Unsubscribe(sub.OutputCh)

	if !sub.Resumed {
		t.Error("expected Resumed=true for valid offset")
	}
	if !sub.HadData {
		t.Error("expected HadData=true for valid offset with data remaining")
	}

	// Drain the output channel: collect all data chunks before the nil sentinel.
	var collected []byte
	for {
		select {
		case data := <-sub.OutputCh:
			if data == nil {
				goto doneValidResume
			}
			collected = append(collected, data...)
		case <-time.After(2 * time.Second):
			t.Fatal("timed out draining output channel")
		}
	}
doneValidResume:

	// Delta should be exactly the last 50 bytes, no SGR reset prefix.
	expected := strings.Repeat("A", 50)
	if string(collected) != expected {
		t.Errorf("expected delta of 50 'A' bytes, got %d bytes: %q", len(collected), string(collected))
	}
	if bytes.HasPrefix(collected, []byte("\x1b[0m")) {
		t.Error("delta data should NOT have SGR reset prefix")
	}

	fake.Close()
	<-sess.Done()
}

// TestSubscribe_ResumeExactMatch verifies that resuming at exactly
// TotalBytes returns Resumed=true, HadData=false, and an empty channel.
func TestSubscribe_ResumeExactMatch(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	_, _ = fake.outW.Write([]byte("some output"))
	time.Sleep(100 * time.Millisecond)

	full := sess.Subscribe(0)
	totalBytes := full.TotalBytes
	sess.Unsubscribe(full.OutputCh)

	sub := sess.Subscribe(totalBytes)
	defer sess.Unsubscribe(sub.OutputCh)

	if !sub.Resumed {
		t.Error("expected Resumed=true when offset equals TotalBytes")
	}
	if sub.HadData {
		t.Error("expected HadData=false when offset equals TotalBytes")
	}

	// Channel should be empty — no data, no sentinel.
	select {
	case data := <-sub.OutputCh:
		t.Errorf("expected empty channel, got data: %v", data)
	case <-time.After(100 * time.Millisecond):
		// Expected: nothing in channel.
	}

	fake.Close()
	<-sess.Done()
}

// TestSubscribe_ResumeStaleOffset verifies that an offset before the history
// buffer's start falls back to full history replay with SGR reset.
func TestSubscribe_ResumeStaleOffset(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Override offlineBufferMax to a small value so history gets trimmed.
	sess.mu.Lock()
	sess.offlineBufferMax = 100
	sess.mu.Unlock()

	// Write 200+ bytes so history is trimmed.
	payload := strings.Repeat("B", 250)
	_, _ = fake.outW.Write([]byte(payload))
	time.Sleep(100 * time.Millisecond)

	// Subscribe with offset 10 — before historyStart, should get full history.
	sub := sess.Subscribe(10)
	defer sess.Unsubscribe(sub.OutputCh)

	if sub.Resumed {
		t.Error("expected Resumed=false for stale offset")
	}
	if !sub.HadData {
		t.Error("expected HadData=true for stale offset with history")
	}

	// Drain and verify SGR reset prefix.
	var collected []byte
	for {
		select {
		case data := <-sub.OutputCh:
			if data == nil {
				goto doneStale
			}
			collected = append(collected, data...)
		case <-time.After(2 * time.Second):
			t.Fatal("timed out draining output channel")
		}
	}
doneStale:

	if !bytes.HasPrefix(collected, []byte("\x1b[0m")) {
		t.Error("full history replay should have SGR reset prefix")
	}

	fake.Close()
	<-sess.Done()
}

// TestSubscribe_ResumeZeroOffset verifies that offset=0 always triggers
// full history replay with SGR reset prefix.
func TestSubscribe_ResumeZeroOffset(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	_, _ = fake.outW.Write([]byte("zero offset test"))
	time.Sleep(100 * time.Millisecond)

	sub := sess.Subscribe(0)
	defer sess.Unsubscribe(sub.OutputCh)

	if sub.Resumed {
		t.Error("expected Resumed=false for zero offset")
	}

	// Drain and verify SGR reset prefix.
	var collected []byte
	for {
		select {
		case data := <-sub.OutputCh:
			if data == nil {
				goto doneZero
			}
			collected = append(collected, data...)
		case <-time.After(2 * time.Second):
			t.Fatal("timed out draining output channel")
		}
	}
doneZero:

	if !bytes.HasPrefix(collected, []byte("\x1b[0m")) {
		t.Error("zero offset should produce full history with SGR reset prefix")
	}
	// The data after the SGR reset should contain our original output.
	if !bytes.Contains(collected, []byte("zero offset test")) {
		t.Error("full history should contain the original output")
	}

	fake.Close()
	<-sess.Done()
}

// TestSubscribe_ResumeFutureOffset verifies that an offset beyond the
// current total falls back to full history replay.
func TestSubscribe_ResumeFutureOffset(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	_, _ = fake.outW.Write([]byte("future test"))
	time.Sleep(100 * time.Millisecond)

	full := sess.Subscribe(0)
	totalBytes := full.TotalBytes
	sess.Unsubscribe(full.OutputCh)

	// Resume with an offset beyond totalBytes.
	sub := sess.Subscribe(totalBytes + 100)
	defer sess.Unsubscribe(sub.OutputCh)

	if sub.Resumed {
		t.Error("expected Resumed=false for future offset")
	}
	if !sub.HadData {
		t.Error("expected HadData=true for future offset (full history sent)")
	}

	// Drain and verify full history with SGR reset prefix.
	var collected []byte
	for {
		select {
		case data := <-sub.OutputCh:
			if data == nil {
				goto doneFuture
			}
			collected = append(collected, data...)
		case <-time.After(2 * time.Second):
			t.Fatal("timed out draining output channel")
		}
	}
doneFuture:

	if !bytes.HasPrefix(collected, []byte("\x1b[0m")) {
		t.Error("future offset should produce full history with SGR reset prefix")
	}

	fake.Close()
	<-sess.Done()
}

// TestSubscribe_ResultTotalBytes verifies that TotalBytes in the
// SubscribeResult accurately reflects the number of bytes written.
func TestSubscribe_ResultTotalBytes(t *testing.T) {
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	sess, err := sm.Create("/fake/shell", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Write known bytes in two batches.
	_, _ = fake.outW.Write([]byte("abcde")) // 5 bytes
	time.Sleep(100 * time.Millisecond)
	_, _ = fake.outW.Write([]byte("fghij")) // 5 bytes
	time.Sleep(100 * time.Millisecond)

	sub := sess.Subscribe(0)
	sess.Unsubscribe(sub.OutputCh)

	if sub.TotalBytes != 10 {
		t.Errorf("expected TotalBytes=10, got %d", sub.TotalBytes)
	}

	fake.Close()
	<-sess.Done()
}
