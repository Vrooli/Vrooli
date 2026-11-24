package instance

import (
	"errors"
	"time"
)

var ErrInstanceNotFound = errors.New("instance was not found")

type State string

const (
	StatePlanned  State = "planned"
	StateStopped  State = "stopped"
	StateRunning  State = "running"
	StateFailed   State = "failed"
	StateDeleting State = "deleting"
)

type Profile string

const (
	ProfileHeadlessLinux Profile = "headless-linux"
	ProfileDesktopLinux  Profile = "desktop-linux"
)

// Instance is retained only for reading and maintaining pre-migration records.
// New QEMU guests are owned and addressed through Compute Manager references.
type Instance struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Provider  string    `json:"provider"`
	State     State     `json:"state"`
	Image     string    `json:"image"`
	Workdir   string    `json:"workdir"`
	Address   string    `json:"address,omitempty"`
	SSHPort   int       `json:"ssh_port,omitempty"`
	Profile   Profile   `json:"profile"`
	PID       int       `json:"pid,omitempty"`
	Command   []string  `json:"command,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	LastError string    `json:"last_error,omitempty"`
}
