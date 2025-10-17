package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/creack/pty"
	"github.com/google/uuid"
)

type closeReason string

const (
	reasonClientRequested closeReason = "client_requested"
	reasonIdleTimeout     closeReason = "idle_timeout"
	reasonTTLExpired      closeReason = "ttl_expired"
	reasonPanicStop       closeReason = "panic_stop"
	reasonInternalError   closeReason = "internal_error"
)

type session struct {
	id        string
	createdAt time.Time
	expiresAt time.Time

	cfg     config
	metrics *metricsRegistry
	manager *sessionManager

	commandName string
	commandArgs []string

	ctx    context.Context
	cancel context.CancelFunc

	command *exec.Cmd
	ptyFile *os.File

	done     chan struct{}
	doneOnce sync.Once

	clientsMu sync.Mutex
	clients   map[*wsClient]struct{}

	ttlTimer *time.Timer

	lastActivity atomic.Int64 // unix nano

	transcriptMu     sync.Mutex
	transcriptFile   *os.File
	transcriptWriter *bufio.Writer

	readBuffer []byte

	// Rolling output buffer for reconnection replay (stores recent output chunks)
	outputBufferMu      sync.RWMutex
	outputBuffer        []outputPayload
	outputBufferDropped bool
	maxBufferSize       int
	maxBufferBytes      int
	currentBufferBytes  int

	inputSeqMu   sync.Mutex
	lastInputSeq map[string]uint64

	termSizeMu sync.RWMutex
	termRows   int
	termCols   int
}

func newSession(manager *sessionManager, cfg config, metrics *metricsRegistry, req createSessionRequest) (*session, error) {
	ctx, cancel := context.WithCancel(context.Background())

	id := uuid.NewString()
	s := &session{
		id:                  id,
		createdAt:           time.Now().UTC(),
		expiresAt:           time.Now().UTC().Add(cfg.sessionTTL),
		cfg:                 cfg,
		metrics:             metrics,
		manager:             manager,
		ctx:                 ctx,
		cancel:              cancel,
		done:                make(chan struct{}),
		clients:             make(map[*wsClient]struct{}),
		readBuffer:          make([]byte, cfg.readBufferSizeBytes),
		outputBuffer:        make([]outputPayload, 0, 500), // Keep last 500 chunks for better reconnect experience
		outputBufferDropped: false,
		maxBufferSize:       500,
		maxBufferBytes:      1024 * 1024, // 1MB max buffer
		currentBufferBytes:  0,
		lastInputSeq:        make(map[string]uint64),
		termRows:            cfg.defaultTTYRows,
		termCols:            cfg.defaultTTYCols,
	}

	command := strings.TrimSpace(req.Command)
	var args []string
	if command == "" {
		command = cfg.defaultCommand
		args = append([]string{}, cfg.defaultArgs...)
	} else {
		args = append([]string{}, req.Args...)
	}
	if command == "" {
		s.cleanupOnInitFailure()
		return nil, errors.New("no command provided")
	}

	s.commandName = command
	s.commandArgs = append([]string{}, args...)

	if err := os.MkdirAll(cfg.storagePath, 0o755); err != nil {
		s.cleanupOnInitFailure()
		return nil, fmt.Errorf("create storage directory: %w", err)
	}

	filePath := filepath.Join(cfg.storagePath, fmt.Sprintf("session-%s.ndjson", id))
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o640)
	if err != nil {
		s.cleanupOnInitFailure()
		return nil, fmt.Errorf("open transcript file: %w", err)
	}
	s.transcriptFile = file
	s.transcriptWriter = bufio.NewWriter(file)

	cmd := exec.CommandContext(ctx, command, args...)
	envExtras := []string{
		"TERM=xterm-256color",
		"WEB_CONSOLE_SESSION_ID=" + id,
		"TERMINAL_CONSOLE_SESSION_ID=" + id,
		"TERMINAL_CONSOLE_COMMAND=" + command,
		"TERMINAL_CONSOLE_COMMAND_LINE=" + formatCommandLine(command, args),
	}
	cmd.Env = append(os.Environ(), envExtras...)
	if cfg.defaultWorkingDir != "" {
		cmd.Dir = cfg.defaultWorkingDir
	}

	ptyFile, err := pty.Start(cmd)
	if err != nil {
		s.cleanupOnInitFailure()
		return nil, fmt.Errorf("start command: %w", err)
	}
	if err := pty.Setsize(ptyFile, &pty.Winsize{Rows: uint16(cfg.defaultTTYRows), Cols: uint16(cfg.defaultTTYCols)}); err != nil {
		s.cleanupOnInitFailure()
		return nil, fmt.Errorf("set initial pty size: %w", err)
	}

	s.command = cmd
	s.ptyFile = ptyFile
	s.lastActivity.Store(time.Now().UTC().UnixNano())

	s.recordStatus("started", formatCommandLine(command, args))

	if req.Operator != "" {
		s.writeTranscript(transcriptEntry{
			Timestamp: time.Now().UTC(),
			Direction: directionStatus,
			Message:   "operator:" + req.Operator,
		})
	}
	if req.Reason != "" {
		s.writeTranscript(transcriptEntry{
			Timestamp: time.Now().UTC(),
			Direction: directionStatus,
			Message:   "reason:" + req.Reason,
		})
	}
	if len(req.Metadata) > 0 {
		s.writeTranscript(transcriptEntry{
			Timestamp: time.Now().UTC(),
			Direction: directionStatus,
			Message:   "metadata:" + string(req.Metadata),
		})
	}

	s.ttlTimer = time.AfterFunc(cfg.sessionTTL, func() {
		s.Close(reasonTTLExpired)
	})

	go s.streamOutput()
	go s.waitForExit()

	return s, nil
}

func formatCommandLine(command string, args []string) string {
	parts := append([]string{command}, args...)
	return strings.Join(parts, " ")
}

func (s *session) cleanupOnInitFailure() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.ptyFile != nil {
		_ = s.ptyFile.Close()
	}
	if s.command != nil && s.command.Process != nil {
		_ = s.command.Process.Kill()
	}
	if s.transcriptWriter != nil {
		_ = s.transcriptWriter.Flush()
	}
	if s.transcriptFile != nil {
		_ = s.transcriptFile.Close()
	}
}

func (s *session) streamOutput() {
	for {
		select {
		case <-s.done:
			return
		default:
		}

		n, err := s.ptyFile.Read(s.readBuffer)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, s.readBuffer[:n])
			s.handleOutput(chunk)
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				s.Close(reasonClientRequested)
			}
			return
		}
	}
}

func (s *session) waitForExit() {
	err := s.command.Wait()
	if err != nil {
		s.recordStatus("command_exit_error", fmt.Sprintf("%s: %v", formatCommandLine(s.commandName, s.commandArgs), err))
	}
	s.Close(reasonClientRequested)
}

func (s *session) handleOutput(chunk []byte) {
	s.touch()

	if bytes.Contains(chunk, []byte("\x1b[6n")) {
		s.respondCursorQuery()
	}

	payload := outputPayload{
		Data:      base64.StdEncoding.EncodeToString(chunk),
		Encoding:  "base64",
		Direction: "stdout",
		Timestamp: time.Now().UTC(),
	}

	s.writeTranscript(transcriptEntry{
		Timestamp: payload.Timestamp,
		Direction: directionStdout,
		Data:      payload.Data,
		Encoding:  payload.Encoding,
	})

	// Add to rolling buffer for replay on reconnect
	s.outputBufferMu.Lock()
	chunkSize := len(chunk)
	s.outputBuffer = append(s.outputBuffer, payload)
	s.currentBufferBytes += chunkSize

	// Trim buffer if it exceeds size or byte limits
	for len(s.outputBuffer) > s.maxBufferSize || s.currentBufferBytes > s.maxBufferBytes {
		if len(s.outputBuffer) == 0 {
			break
		}
		// Remove oldest chunk
		removed := s.outputBuffer[0]
		s.outputBuffer = s.outputBuffer[1:]
		s.outputBufferDropped = true

		// Decode to get actual size
		if removed.Data != "" {
			decoded, err := base64.StdEncoding.DecodeString(removed.Data)
			if err == nil {
				s.currentBufferBytes -= len(decoded)
			}
		}
	}
	s.outputBufferMu.Unlock()

	s.broadcast(websocketEnvelope{
		Type:    "output",
		Payload: mustJSON(payload),
	})
}

func (s *session) handleInput(client *wsClient, data []byte, seq uint64, source string) error {
	if len(data) == 0 {
		return nil
	}
	if s.ptyFile == nil {
		return errors.New("pty closed")
	}
	if seq > 0 {
		key := source
		if key == "" && client != nil {
			key = client.id
		}
		if key != "" && !s.shouldProcessInputSeq(key, seq) {
			return nil
		}
	}
	if _, err := s.ptyFile.Write(data); err != nil {
		return err
	}
	s.touch()

	s.writeTranscript(transcriptEntry{
		Timestamp: time.Now().UTC(),
		Direction: directionStdin,
		Data:      base64.StdEncoding.EncodeToString(data),
		Encoding:  "base64",
	})
	return nil
}

func (s *session) respondCursorQuery() {
	if s.ptyFile == nil {
		return
	}
	s.termSizeMu.RLock()
	rows := s.termRows
	cols := s.termCols
	s.termSizeMu.RUnlock()
	if rows <= 0 {
		rows = 1
	}
	if cols <= 0 {
		cols = 80
	}
	response := fmt.Sprintf("\x1b[%d;%dR", rows, cols)
	_, _ = s.ptyFile.Write([]byte(response))
}

func (s *session) resize(cols, rows int) error {
	if s.ptyFile == nil {
		return errors.New("pty not available")
	}
	if cols <= 0 || rows <= 0 {
		return errors.New("invalid size")
	}
	win := &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)}
	if err := pty.Setsize(s.ptyFile, win); err != nil {
		return fmt.Errorf("set pty size: %w", err)
	}
	s.termSizeMu.Lock()
	s.termRows = rows
	s.termCols = cols
	s.termSizeMu.Unlock()
	return nil
}

func (s *session) Close(reason closeReason) {
	s.doneOnce.Do(func() {
		close(s.done)
		if s.ttlTimer != nil {
			s.ttlTimer.Stop()
		}

		s.cancel()
		if s.command != nil && s.command.Process != nil {
			_ = s.command.Process.Signal(os.Interrupt)
			time.AfterFunc(s.cfg.panicKillGrace, func() {
				if s.command.ProcessState == nil || !s.command.ProcessState.Exited() {
					_ = s.command.Process.Kill()
				}
			})
		}

		s.recordStatus("closed", string(reason))

		s.clientsMu.Lock()
		for client := range s.clients {
			client.close()
		}
		s.clients = map[*wsClient]struct{}{}
		s.clientsMu.Unlock()

		s.transcriptMu.Lock()
		if s.transcriptWriter != nil {
			_ = s.transcriptWriter.Flush()
		}
		if s.transcriptFile != nil {
			_ = s.transcriptFile.Close()
		}
		s.transcriptMu.Unlock()

		s.metrics.activeSessions.Add(-1)
		s.incrementReasonMetric(reason)
		s.manager.onSessionClosed(s, reason)
	})
}

func (s *session) incrementReasonMetric(reason closeReason) {
	switch reason {
	case reasonPanicStop:
		s.metrics.panicStops.Add(1)
	case reasonIdleTimeout:
		s.metrics.idleTimeOuts.Add(1)
	case reasonTTLExpired:
		s.metrics.ttlExpirations.Add(1)
	}
}

func (s *session) recordStatus(status string, reason string) {
	entry := transcriptEntry{
		Timestamp: time.Now().UTC(),
		Direction: directionStatus,
		Message:   fmt.Sprintf("%s:%s", status, reason),
	}
	s.writeTranscript(entry)
	s.broadcast(websocketEnvelope{
		Type:    "status",
		Payload: mustJSON(statusPayload{Status: status, Reason: reason, Timestamp: entry.Timestamp}),
	})
}

func (s *session) writeTranscript(entry transcriptEntry) {
	s.transcriptMu.Lock()
	defer s.transcriptMu.Unlock()
	if s.transcriptWriter == nil {
		return
	}
	encoded, err := json.Marshal(entry)
	if err != nil {
		return
	}
	if _, err := s.transcriptWriter.Write(encoded); err == nil {
		_, _ = s.transcriptWriter.WriteString("\n")
	}
}

func (s *session) addClient(client *wsClient) {
	s.clientsMu.Lock()
	s.clients[client] = struct{}{}
	s.clientsMu.Unlock()
}

// replayBufferToClient sends recent output history to a newly connected client
func (s *session) replayBufferToClient(client *wsClient) {
	s.outputBufferMu.RLock()
	buffer := make([]outputPayload, len(s.outputBuffer))
	copy(buffer, s.outputBuffer)
	truncated := s.outputBufferDropped
	s.outputBufferMu.RUnlock()

	client.enqueue(websocketEnvelope{
		Type: "output_replay",
		Payload: mustJSON(replayPayload{
			Chunks:    buffer,
			Count:     len(buffer),
			Complete:  true,
			Truncated: truncated,
			Generated: time.Now().UTC(),
		}),
	})
}

func (s *session) removeClient(client *wsClient) {
	s.clientsMu.Lock()
	delete(s.clients, client)
	s.clientsMu.Unlock()
}

func (s *session) shouldProcessInputSeq(key string, seq uint64) bool {
	s.inputSeqMu.Lock()
	defer s.inputSeqMu.Unlock()
	last := s.lastInputSeq[key]
	if seq <= last {
		return false
	}
	s.lastInputSeq[key] = seq
	return true
}

func (s *session) broadcast(msg websocketEnvelope) {
	s.clientsMu.Lock()
	for client := range s.clients {
		client.enqueue(msg)
	}
	s.clientsMu.Unlock()
}

func (s *session) touch() {
	now := time.Now().UTC()
	s.lastActivity.Store(now.UnixNano())
	if s.manager != nil {
		s.manager.recordActivity()
	}
}

func (s *session) lastActivityTime() time.Time {
	unix := s.lastActivity.Load()
	if unix == 0 {
		return s.createdAt
	}
	return time.Unix(0, unix).UTC()
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return b
}
