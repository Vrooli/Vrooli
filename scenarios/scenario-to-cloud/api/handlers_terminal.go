package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin for development
		// In production, this should be more restrictive
		return true
	},
}

// TerminalMessage represents a message sent over the terminal WebSocket.
type TerminalMessage struct {
	Type string `json:"type"` // "data", "resize", "ping", "error"
	Data string `json:"data,omitempty"`
	Cols int    `json:"cols,omitempty"`
	Rows int    `json:"rows,omitempty"`
}

// handleTerminalWebSocket handles WebSocket connections for web-based SSH terminal.
// WS /api/v1/deployments/{id}/terminal
func (s *Server) handleTerminalWebSocket(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	// Get deployment
	deployment, err := s.repo.GetDeployment(r.Context(), id)
	if err != nil {
		log.Printf("Terminal: failed to get deployment %s: %v", id, err)
		http.Error(w, "Failed to get deployment", http.StatusInternalServerError)
		return
	}

	if deployment == nil {
		http.Error(w, "Deployment not found", http.StatusNotFound)
		return
	}

	// Parse manifest
	var manifest CloudManifest
	if err := json.Unmarshal(deployment.Manifest, &manifest); err != nil {
		log.Printf("Terminal: failed to parse manifest: %v", err)
		http.Error(w, "Failed to parse manifest", http.StatusInternalServerError)
		return
	}

	normalized, _ := ValidateAndNormalizeManifest(manifest)
	if normalized.Target.VPS == nil {
		http.Error(w, "Deployment does not have a VPS target", http.StatusBadRequest)
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Terminal: failed to upgrade WebSocket: %v", err)
		return
	}
	defer conn.Close()

	// Create SSH connection
	sshClient, err := createSSHClient(normalized)
	if err != nil {
		sendTerminalError(conn, "Failed to connect to VPS: "+err.Error())
		return
	}
	defer sshClient.Close()

	// Create SSH session
	session, err := sshClient.NewSession()
	if err != nil {
		sendTerminalError(conn, "Failed to create SSH session: "+err.Error())
		return
	}
	defer session.Close()

	// Get stdin, stdout, stderr pipes
	stdin, err := session.StdinPipe()
	if err != nil {
		sendTerminalError(conn, "Failed to get stdin: "+err.Error())
		return
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		sendTerminalError(conn, "Failed to get stdout: "+err.Error())
		return
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		sendTerminalError(conn, "Failed to get stderr: "+err.Error())
		return
	}

	// Request PTY
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	// Default terminal size
	if err := session.RequestPty("xterm-256color", 24, 80, modes); err != nil {
		sendTerminalError(conn, "Failed to request PTY: "+err.Error())
		return
	}

	// Start shell
	workdir := normalized.Target.VPS.Workdir
	if err := session.Shell(); err != nil {
		sendTerminalError(conn, "Failed to start shell: "+err.Error())
		return
	}

	// Change to workdir
	if _, err := stdin.Write([]byte("cd " + shellQuoteSingle(workdir) + " && clear\n")); err != nil {
		log.Printf("Terminal: failed to change to workdir: %v", err)
	}

	// Create context for cleanup
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	var wg sync.WaitGroup

	// Read from SSH stdout and send to WebSocket
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := stdout.Read(buf)
				if err != nil {
					if err != io.EOF {
						log.Printf("Terminal: stdout read error: %v", err)
					}
					cancel()
					return
				}
				if n > 0 {
					msg := TerminalMessage{
						Type: "data",
						Data: string(buf[:n]),
					}
					if err := conn.WriteJSON(msg); err != nil {
						log.Printf("Terminal: WebSocket write error: %v", err)
						cancel()
						return
					}
				}
			}
		}
	}()

	// Read from SSH stderr and send to WebSocket
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := stderr.Read(buf)
				if err != nil {
					if err != io.EOF {
						log.Printf("Terminal: stderr read error: %v", err)
					}
					return
				}
				if n > 0 {
					msg := TerminalMessage{
						Type: "data",
						Data: string(buf[:n]),
					}
					if err := conn.WriteJSON(msg); err != nil {
						log.Printf("Terminal: WebSocket write error: %v", err)
						cancel()
						return
					}
				}
			}
		}
	}()

	// Read from WebSocket and write to SSH stdin
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				var msg TerminalMessage
				if err := conn.ReadJSON(&msg); err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						log.Printf("Terminal: WebSocket read error: %v", err)
					}
					cancel()
					return
				}

				switch msg.Type {
				case "data":
					if _, err := stdin.Write([]byte(msg.Data)); err != nil {
						log.Printf("Terminal: stdin write error: %v", err)
						cancel()
						return
					}
				case "resize":
					if msg.Cols > 0 && msg.Rows > 0 {
						session.WindowChange(msg.Rows, msg.Cols)
					}
				case "ping":
					// Respond with pong
					pong := TerminalMessage{Type: "pong"}
					conn.WriteJSON(pong)
				}
			}
		}
	}()

	// Wait for session to complete
	session.Wait()
	cancel()
	wg.Wait()
}

// createSSHClient creates an SSH client connection to the VPS.
func createSSHClient(manifest CloudManifest) (*ssh.Client, error) {
	vps := manifest.Target.VPS

	// Read private key
	key, err := readSSHPrivateKey(vps.KeyPath)
	if err != nil {
		return nil, err
	}

	// Parse private key
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}

	config := &ssh.ClientConfig{
		User: vps.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Use known_hosts in production
		Timeout:         30 * time.Second,
	}

	// Connect
	port := vps.Port
	if port == 0 {
		port = 22
	}
	addr := vps.Host + ":" + intToStr(port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// sendTerminalError sends an error message over the terminal WebSocket.
func sendTerminalError(conn *websocket.Conn, message string) {
	msg := TerminalMessage{
		Type: "error",
		Data: message,
	}
	conn.WriteJSON(msg)
}

// readSSHPrivateKey reads an SSH private key from a file.
func readSSHPrivateKey(keyPath string) ([]byte, error) {
	return os.ReadFile(keyPath)
}
