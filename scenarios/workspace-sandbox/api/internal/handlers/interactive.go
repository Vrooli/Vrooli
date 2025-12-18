package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty/v2"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/types"
)

// WebSocket message types for interactive sessions
const (
	MsgTypeStdin  = "stdin"
	MsgTypeStdout = "stdout"
	MsgTypeStderr = "stderr"
	MsgTypeResize = "resize"
	MsgTypeExit   = "exit"
	MsgTypeError  = "error"
	MsgTypePing   = "ping"
	MsgTypePong   = "pong"
)

// InteractiveMessage is the WebSocket message format for interactive sessions.
type InteractiveMessage struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"` // Base64 or UTF-8 data
	Cols int    `json:"cols,omitempty"` // For resize messages
	Rows int    `json:"rows,omitempty"` // For resize messages
	Code int    `json:"code,omitempty"` // For exit messages
}

// InteractiveStartRequest is the initial message to start an interactive session.
type InteractiveStartRequest struct {
	Command        string            `json:"command"`
	Args           []string          `json:"args,omitempty"`
	IsolationLevel string            `json:"isolationLevel,omitempty"`
	AllowNetwork   bool              `json:"allowNetwork,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
	WorkingDir     string            `json:"workingDir,omitempty"`
	MemoryLimitMB  int               `json:"memoryLimitMB,omitempty"`
	CPUTimeSec     int               `json:"cpuTimeSec,omitempty"`
	MaxProcesses   int               `json:"maxProcesses,omitempty"`
	MaxOpenFiles   int               `json:"maxOpenFiles,omitempty"`
	Cols           int               `json:"cols,omitempty"` // Initial terminal width
	Rows           int               `json:"rows,omitempty"` // Initial terminal height
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin for now
		// In production, this should be restricted
		return true
	},
}

// ExecInteractive handles WebSocket-based interactive command execution.
// This allows real-time bidirectional communication with a process running in the sandbox.
//
// Protocol:
// 1. Client connects via WebSocket
// 2. Client sends InteractiveStartRequest as first message
// 3. Server starts process with PTY and streams output
// 4. Client sends stdin messages, server sends stdout/stderr messages
// 5. Client can send resize messages to change terminal size
// 6. Server sends exit message when process terminates
//
// [OT-P2-008] Phase 6 - Interactive Sessions
func (h *Handlers) ExecInteractive(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	// Get the sandbox
	sb, err := h.Service.Get(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	// Verify sandbox is active
	if sb.Status != types.StatusActive {
		h.JSONError(w, "sandbox must be active for interactive sessions", http.StatusConflict)
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// Upgrade already wrote error response
		return
	}
	defer conn.Close()

	// Read the initial start request
	var startReq InteractiveStartRequest
	if err := conn.ReadJSON(&startReq); err != nil {
		sendErrorMessage(conn, fmt.Sprintf("failed to read start request: %v", err))
		return
	}

	if startReq.Command == "" {
		sendErrorMessage(conn, "command is required")
		return
	}

	// Set default terminal size
	if startReq.Cols <= 0 {
		startReq.Cols = 80
	}
	if startReq.Rows <= 0 {
		startReq.Rows = 24
	}

	// Build bwrap config
	cfg := driver.DefaultBwrapConfig()
	if startReq.WorkingDir != "" {
		cfg.WorkingDir = startReq.WorkingDir
	}
	for k, v := range startReq.Env {
		cfg.Env[k] = v
	}

	// Set isolation level and related config
	if startReq.IsolationLevel == "vrooli-aware" {
		driver.ApplyVrooliAwareConfig(&cfg)
	} else if startReq.AllowNetwork {
		cfg.AllowNetwork = true
	}

	// Set resource limits
	cfg.ResourceLimits = driver.ResourceLimits{
		MemoryLimitMB: startReq.MemoryLimitMB,
		CPUTimeSec:    startReq.CPUTimeSec,
		MaxProcesses:  startReq.MaxProcesses,
		MaxOpenFiles:  startReq.MaxOpenFiles,
	}

	// Run the interactive session
	runInteractiveSession(conn, sb, cfg, startReq)
}

// runInteractiveSession runs a command with PTY and streams I/O over WebSocket.
func runInteractiveSession(conn *websocket.Conn, sb *types.Sandbox, cfg driver.BwrapConfig, req InteractiveStartRequest) {
	// Build the command
	executable, args := driver.BuildExecCommand(sb, cfg, req.Command, req.Args...)

	cmd := exec.Command(executable, args...)
	cmd.Dir = sb.MergedDir

	// Set up environment
	cmd.Env = os.Environ()
	for k, v := range cfg.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Start with PTY
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{
		Rows: uint16(req.Rows),
		Cols: uint16(req.Cols),
	})
	if err != nil {
		sendErrorMessage(conn, fmt.Sprintf("failed to start process: %v", err))
		return
	}
	defer ptmx.Close()

	// Use a context to manage shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// Mutex for writing to WebSocket (WebSocket is not concurrent-safe for writes)
	var writeMu sync.Mutex

	// Read from PTY and send to WebSocket
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			n, err := ptmx.Read(buf)
			if err != nil {
				// PTY closed (process exited)
				return
			}
			if n > 0 {
				writeMu.Lock()
				err := conn.WriteJSON(InteractiveMessage{
					Type: MsgTypeStdout,
					Data: string(buf[:n]),
				})
				writeMu.Unlock()
				if err != nil {
					return
				}
			}
		}
	}()

	// Read from WebSocket and handle messages
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel() // Cancel context when WebSocket read fails

		for {
			var msg InteractiveMessage
			if err := conn.ReadJSON(&msg); err != nil {
				// WebSocket closed
				return
			}

			switch msg.Type {
			case MsgTypeStdin:
				if _, err := ptmx.Write([]byte(msg.Data)); err != nil {
					return
				}
			case MsgTypeResize:
				if msg.Cols > 0 && msg.Rows > 0 {
					pty.Setsize(ptmx, &pty.Winsize{
						Rows: uint16(msg.Rows),
						Cols: uint16(msg.Cols),
					})
				}
			case MsgTypePing:
				writeMu.Lock()
				conn.WriteJSON(InteractiveMessage{Type: MsgTypePong})
				writeMu.Unlock()
			}
		}
	}()

	// Wait for process to exit
	exitCode := 0
	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			}
		}
	}

	// Cancel context to stop goroutines
	cancel()

	// Send exit message
	writeMu.Lock()
	conn.WriteJSON(InteractiveMessage{
		Type: MsgTypeExit,
		Code: exitCode,
	})
	writeMu.Unlock()

	// Give a moment for the message to be sent
	time.Sleep(100 * time.Millisecond)

	// Wait for goroutines to finish (with timeout)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		// Timeout waiting for goroutines
	}
}

// sendErrorMessage sends an error message over WebSocket.
func sendErrorMessage(conn *websocket.Conn, message string) {
	conn.WriteJSON(InteractiveMessage{
		Type: MsgTypeError,
		Data: message,
	})
}
