package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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

	ctx    context.Context
	cancel context.CancelFunc

	command *exec.Cmd
	ptyFile *os.File

	done     chan struct{}
	doneOnce sync.Once

	clientsMu sync.Mutex
	clients   map[*wsClient]struct{}

	idleReset chan struct{}
	ttlTimer  *time.Timer

	lastActivity atomic.Int64 // unix nano

	transcriptMu     sync.Mutex
	transcriptFile   *os.File
	transcriptWriter *bufio.Writer

	readBuffer []byte
}

func newSession(manager *sessionManager, cfg config, metrics *metricsRegistry, req createSessionRequest) (*session, error) {
	ctx, cancel := context.WithCancel(context.Background())

	id := uuid.NewString()
	s := &session{
		id:         id,
		createdAt:  time.Now().UTC(),
		expiresAt:  time.Now().UTC().Add(cfg.sessionTTL),
		cfg:        cfg,
		metrics:    metrics,
		manager:    manager,
		ctx:        ctx,
		cancel:     cancel,
		done:       make(chan struct{}),
		clients:    make(map[*wsClient]struct{}),
		idleReset:  make(chan struct{}, 1),
		readBuffer: make([]byte, cfg.readBufferSizeBytes),
	}

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

	cmd := exec.CommandContext(ctx, cfg.cliPath)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "CODEX_CONSOLE_SESSION_ID="+id)

	ptyFile, err := pty.Start(cmd)
	if err != nil {
		s.cleanupOnInitFailure()
		return nil, fmt.Errorf("start codex CLI: %w", err)
	}

	s.command = cmd
	s.ptyFile = ptyFile
	s.lastActivity.Store(time.Now().UTC().UnixNano())

	s.recordStatus("started", "")

	if req.Operator != "" {
		s.writeTranscript(transcriptEntry{
			Timestamp: time.Now().UTC(),
			Direction: directionStatus,
			Message:   "operator:" + req.Operator,
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

	go s.watchIdle()
	go s.streamOutput()
	go s.waitForExit()

	return s, nil
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

func (s *session) watchIdle() {
	timer := time.NewTimer(s.cfg.idleTimeout)
	defer timer.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-timer.C:
			s.Close(reasonIdleTimeout)
			return
		case <-s.idleReset:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(s.cfg.idleTimeout)
		}
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
		s.recordStatus("command_exit_error", fmt.Sprintf("%v", err))
	}
	s.Close(reasonClientRequested)
}

func (s *session) handleOutput(chunk []byte) {
	s.touch()

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

	s.broadcast(websocketEnvelope{
		Type:    "output",
		Payload: mustJSON(payload),
	})
}

func (s *session) handleInput(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if s.ptyFile == nil {
		return errors.New("pty closed")
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

func (s *session) removeClient(client *wsClient) {
	s.clientsMu.Lock()
	delete(s.clients, client)
	s.clientsMu.Unlock()
}

func (s *session) broadcast(msg websocketEnvelope) {
	s.clientsMu.Lock()
	for client := range s.clients {
		client.enqueue(msg)
	}
	s.clientsMu.Unlock()
}

func (s *session) touch() {
	s.lastActivity.Store(time.Now().UTC().UnixNano())
	select {
	case s.idleReset <- struct{}{}:
	default:
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
