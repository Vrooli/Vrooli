package livedesktop

import (
	"os/exec"
	"sync"
	"time"

	"scenario-to-desktop-api/screenrecording"
)

// SessionState represents the lifecycle state of a live desktop session.
type SessionState string

const (
	StateCreating SessionState = "creating"
	StateRunning  SessionState = "running"
	StateStopping SessionState = "stopping"
	StateStopped  SessionState = "stopped"
	StateError    SessionState = "error"
)

// Session holds the full state of a live desktop session.
type Session struct {
	ID            string                          `json:"id"`
	ScenarioName  string                          `json:"scenario_name"`
	State         SessionState                    `json:"state"`
	Display       *screenrecording.ManagedDisplay `json:"-"`
	VNCPort       int                             `json:"vnc_port"`
	WSPort        int                             `json:"ws_port"`
	X11VNCCmd     *exec.Cmd                       `json:"-"`
	WebsockifyCmd *exec.Cmd                       `json:"-"`
	AppCmd        *exec.Cmd                       `json:"-"`
	CreatedAt     time.Time                       `json:"created_at"`
	LastHeartbeat time.Time                       `json:"last_heartbeat"`
	Error         string                          `json:"error,omitempty"`
	Width         int                             `json:"width"`
	Height        int                             `json:"height"`

	mu sync.Mutex
}

// SetState updates the session state under lock.
func (s *Session) SetState(state SessionState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.State = state
}

// SetError sets the session into error state with a message.
func (s *Session) SetError(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.State = StateError
	s.Error = msg
}

// Touch updates the last heartbeat timestamp.
func (s *Session) Touch() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastHeartbeat = time.Now()
}

// SessionConfig is the configuration for creating a new session.
type SessionConfig struct {
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	ScenarioName string `json:"scenario_name"`
	AppPath      string `json:"app_path,omitempty"`
}

// SessionView is the JSON-safe view of a session for API responses.
type SessionView struct {
	ID            string       `json:"id"`
	ScenarioName  string       `json:"scenario_name"`
	State         SessionState `json:"state"`
	VNCPort       int          `json:"vnc_port"`
	WSPort        int          `json:"ws_port"`
	Width         int          `json:"width"`
	Height        int          `json:"height"`
	CreatedAt     time.Time    `json:"created_at"`
	LastHeartbeat time.Time    `json:"last_heartbeat"`
	Error         string       `json:"error,omitempty"`
}

// View returns a JSON-safe snapshot of the session.
func (s *Session) View() SessionView {
	s.mu.Lock()
	defer s.mu.Unlock()
	return SessionView{
		ID:            s.ID,
		ScenarioName:  s.ScenarioName,
		State:         s.State,
		VNCPort:       s.VNCPort,
		WSPort:        s.WSPort,
		Width:         s.Width,
		Height:        s.Height,
		CreatedAt:     s.CreatedAt,
		LastHeartbeat: s.LastHeartbeat,
		Error:         s.Error,
	}
}
