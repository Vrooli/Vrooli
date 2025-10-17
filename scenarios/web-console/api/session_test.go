package main

import (
	"encoding/base64"
	"testing"
	"time"
)

func TestNewSession(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	manager, metrics, _ := setupTestSessionManager(t, cfg)

	t.Run("Success", func(t *testing.T) {
		req := createSessionRequest{
			Operator: "test-operator",
			Reason:   "testing",
		}

		s, err := newSession(manager, cfg, metrics, req)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
		defer cleanupSession(s)

		assertSessionState(t, s, cfg.defaultCommand)
		if s.ptyFile == nil {
			t.Error("PTY file should be initialized")
		}
		if s.command == nil {
			t.Error("Command should be initialized")
		}
		if s.transcriptFile == nil {
			t.Error("Transcript file should be initialized")
		}
	})

	t.Run("CustomCommand", func(t *testing.T) {
		req := createSessionRequest{
			Command: "/bin/echo",
			Args:    []string{"test"},
		}

		s, err := newSession(manager, cfg, metrics, req)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
		defer cleanupSession(s)

		if s.commandName != "/bin/echo" {
			t.Errorf("Expected command '/bin/echo', got '%s'", s.commandName)
		}
		if len(s.commandArgs) != 1 || s.commandArgs[0] != "test" {
			t.Errorf("Expected args ['test'], got %v", s.commandArgs)
		}
	})

	t.Run("NoCommand", func(t *testing.T) {
		badCfg := cfg
		badCfg.defaultCommand = ""

		req := createSessionRequest{}

		_, err := newSession(manager, badCfg, metrics, req)
		if err == nil {
			t.Error("Expected error when no command provided")
		}
	})
}

func TestSessionHandleInput(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	manager, _, _ := setupTestSessionManager(t, cfg)

	s := createTestSession(t, manager, createSessionRequest{})
	defer cleanupSession(s)

	t.Run("ValidInput", func(t *testing.T) {
		input := []byte("test input\n")
		err := s.handleInput(nil, input, 0, "")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify activity was touched
		lastActivity := s.lastActivityTime()
		if lastActivity.IsZero() {
			t.Error("Last activity should be updated")
		}
	})

	t.Run("EmptyInput", func(t *testing.T) {
		err := s.handleInput(nil, []byte{}, 0, "")
		if err != nil {
			t.Errorf("Expected no error for empty input, got %v", err)
		}
	})

	t.Run("SequenceNumberDeduplication", func(t *testing.T) {
		input := []byte("test\n")

		// First input with seq 1
		err := s.handleInput(nil, input, 1, "client-1")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Duplicate with same seq should be ignored
		err = s.handleInput(nil, input, 1, "client-1")
		if err != nil {
			t.Errorf("Expected no error for duplicate seq, got %v", err)
		}

		// Higher seq should be processed
		err = s.handleInput(nil, input, 2, "client-1")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}

func TestSessionResize(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	manager, _, _ := setupTestSessionManager(t, cfg)

	s := createTestSession(t, manager, createSessionRequest{})
	defer cleanupSession(s)

	t.Run("ValidResize", func(t *testing.T) {
		err := s.resize(80, 24)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		s.termSizeMu.RLock()
		rows := s.termRows
		cols := s.termCols
		s.termSizeMu.RUnlock()

		if rows != 24 {
			t.Errorf("Expected rows 24, got %d", rows)
		}
		if cols != 80 {
			t.Errorf("Expected cols 80, got %d", cols)
		}
	})

	t.Run("InvalidSize", func(t *testing.T) {
		err := s.resize(0, 24)
		if err == nil {
			t.Error("Expected error for invalid cols")
		}

		err = s.resize(80, 0)
		if err == nil {
			t.Error("Expected error for invalid rows")
		}

		err = s.resize(-1, 24)
		if err == nil {
			t.Error("Expected error for negative cols")
		}
	})
}

func TestSessionClose(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	manager, metrics, _ := setupTestSessionManager(t, cfg)

	t.Run("NormalClose", func(t *testing.T) {
		s := createTestSession(t, manager, createSessionRequest{})

		s.Close(reasonClientRequested)
		time.Sleep(50 * time.Millisecond)

		// Verify done channel closed
		select {
		case <-s.done:
			// Expected
		default:
			t.Error("Done channel should be closed")
		}

		// Verify clients cleared
		s.clientsMu.Lock()
		clientCount := len(s.clients)
		s.clientsMu.Unlock()
		if clientCount != 0 {
			t.Errorf("Expected 0 clients, got %d", clientCount)
		}
	})

	t.Run("IdempotentClose", func(t *testing.T) {
		s := createTestSession(t, manager, createSessionRequest{})

		// Close multiple times should not panic
		s.Close(reasonClientRequested)
		s.Close(reasonClientRequested)
		s.Close(reasonPanicStop)
	})

	t.Run("MetricsIncremented", func(t *testing.T) {
		initialPanicStops := metrics.panicStops.Load()

		s := createTestSession(t, manager, createSessionRequest{})
		s.Close(reasonPanicStop)
		time.Sleep(50 * time.Millisecond)

		if metrics.panicStops.Load() <= initialPanicStops {
			t.Error("Panic stops metric should be incremented")
		}
	})
}

func TestSessionTouch(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	manager, _, _ := setupTestSessionManager(t, cfg)

	s := createTestSession(t, manager, createSessionRequest{})
	defer cleanupSession(s)

	t.Run("UpdatesLastActivity", func(t *testing.T) {
		before := s.lastActivityTime()
		time.Sleep(10 * time.Millisecond)

		s.touch()

		after := s.lastActivityTime()
		if !after.After(before) {
			t.Error("Last activity time should be updated")
		}
	})
}

func TestSessionBroadcast(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	manager, _, _ := setupTestSessionManager(t, cfg)

	s := createTestSession(t, manager, createSessionRequest{})
	defer func() {
		// Remove all clients before cleanup to avoid nil WebSocket issue
		s.clientsMu.Lock()
		s.clients = make(map[*wsClient]struct{})
		s.clientsMu.Unlock()
		cleanupSession(s)
	}()

	t.Run("BroadcastToClients", func(t *testing.T) {
		// Create mock clients
		client1 := &wsClient{
			id:      "client-1",
			send:    make(chan websocketEnvelope, 10),
			session: s,
			conn:    nil,
		}
		client2 := &wsClient{
			id:      "client-2",
			send:    make(chan websocketEnvelope, 10),
			session: s,
			conn:    nil,
		}

		s.addClient(client1)
		s.addClient(client2)

		// Broadcast message
		msg := websocketEnvelope{
			Type:    "test",
			Payload: []byte(`{"data":"test"}`),
		}
		s.broadcast(msg)

		// Verify both clients received message
		timeout := time.After(1 * time.Second)

		select {
		case received := <-client1.send:
			if received.Type != "test" {
				t.Errorf("Expected type 'test', got '%s'", received.Type)
			}
		case <-timeout:
			t.Error("Timeout waiting for message on client1")
		}

		select {
		case received := <-client2.send:
			if received.Type != "test" {
				t.Errorf("Expected type 'test', got '%s'", received.Type)
			}
		case <-timeout:
			t.Error("Timeout waiting for message on client2")
		}

		// Remove clients manually to avoid close issues
		s.removeClient(client1)
		s.removeClient(client2)
	})
}

func TestSessionOutputBuffer(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	manager, _, _ := setupTestSessionManager(t, cfg)

	s := createTestSession(t, manager, createSessionRequest{})
	defer cleanupSession(s)

	// Wait for some output
	if waitForSessionOutput(t, s, 2*time.Second) {
		s.outputBufferMu.RLock()
		bufferSize := len(s.outputBuffer)
		s.outputBufferMu.RUnlock()

		if bufferSize == 0 {
			t.Error("Expected output buffer to contain data")
		}

		// Verify buffer content
		s.outputBufferMu.RLock()
		for _, payload := range s.outputBuffer {
			if payload.Data == "" {
				t.Error("Output payload data should not be empty")
			}
			if payload.Encoding != "base64" {
				t.Errorf("Expected encoding 'base64', got '%s'", payload.Encoding)
			}
			if payload.Direction != "stdout" {
				t.Errorf("Expected direction 'stdout', got '%s'", payload.Direction)
			}

			// Verify data is valid base64
			_, err := base64.StdEncoding.DecodeString(payload.Data)
			if err != nil {
				t.Errorf("Invalid base64 data: %v", err)
			}
		}
		s.outputBufferMu.RUnlock()
	}
}

func TestSessionReplayBuffer(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	manager, _, _ := setupTestSessionManager(t, cfg)

	s := createTestSession(t, manager, createSessionRequest{})
	defer cleanupSession(s)

	// Wait for output
	if waitForSessionOutput(t, s, 2*time.Second) {
		// Create mock client
		client := &wsClient{
			id:      "test-client",
			send:    make(chan websocketEnvelope, 100),
			session: s,
			conn:    nil,
		}

		// Replay buffer
		s.replayBufferToClient(client)

		// Remove client before cleanup
		s.removeClient(client)

		// Verify replay message received
		timeout := time.After(1 * time.Second)
		select {
		case msg := <-client.send:
			if msg.Type != "output_replay" {
				t.Errorf("Expected type 'output_replay', got '%s'", msg.Type)
			}
		case <-timeout:
			t.Error("Timeout waiting for replay message")
		}
	}
}

func TestFormatCommandLine(t *testing.T) {
	tests := []struct {
		command  string
		args     []string
		expected string
	}{
		{"/bin/bash", nil, "/bin/bash"},
		{"/bin/echo", []string{"hello"}, "/bin/echo hello"},
		{"/bin/ls", []string{"-l", "-a"}, "/bin/ls -l -a"},
		{"command", []string{}, "command"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatCommandLine(tt.command, tt.args)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
