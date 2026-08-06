// Package sessions owns the lifetime and governance state of a program kernel.
// Kernel memory is intentionally process-local; this package stores only the
// session lease and its grant/sandbox metadata.
package sessions

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound      = errors.New("session not found")
	ErrReclaimed     = errors.New("session reclaimed")
	ErrLimitExceeded = errors.New("session limit exceeded")
	ErrKernelStart   = errors.New("kernel failed to start")
)

type Clock interface{ Now() time.Time }
type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

type Options struct {
	IdleTimeout   time.Duration
	WallTimeout   time.Duration
	MemoryLimit   int64
	MaxSessions   int
	Clock         Clock
	KernelFactory KernelFactory
}

type Kernel interface{ Close() error }
type KernelFactory func(sessionID string) (Kernel, error)

type Session struct {
	ID               string
	Name             string
	State            string
	CreatedAt        time.Time
	LastActivityAt   time.Time
	Grants           map[string]struct{}
	SandboxWorkspace string
	ReclaimedReason  string
	MemoryBytes      int64
	Kernel           Kernel
}

type Manager struct {
	mu       sync.RWMutex
	options  Options
	sessions map[string]*Session
}

func NewManager(options Options) *Manager {
	if options.Clock == nil {
		options.Clock = systemClock{}
	}
	if options.IdleTimeout <= 0 {
		options.IdleTimeout = 30 * time.Minute
	}
	if options.WallTimeout <= 0 {
		options.WallTimeout = 4 * time.Hour
	}
	return &Manager{options: options, sessions: make(map[string]*Session)}
}

func (m *Manager) Create(name, sandbox string, grants []string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.options.MaxSessions > 0 && len(m.sessions) >= m.options.MaxSessions {
		return nil, ErrLimitExceeded
	}
	id := "sess_" + uuid.NewString()
	now := m.options.Clock.Now()
	s := &Session{ID: id, Name: strings.TrimSpace(name), State: "running", CreatedAt: now, LastActivityAt: now, Grants: grantSet(grants), SandboxWorkspace: strings.TrimSpace(sandbox)}
	if m.options.KernelFactory != nil {
		kernel, err := m.options.KernelFactory(id)
		if err != nil {
			return nil, fmt.Errorf("%w for %s: %v", ErrKernelStart, id, err)
		}
		s.Kernel = kernel
	}
	m.sessions[id] = s
	return clone(s), nil
}

func (m *Manager) Get(id string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	if !ok {
		return nil, ErrNotFound
	}
	return clone(s), nil
}

func (m *Manager) List() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		out = append(out, clone(s))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

func (m *Manager) Delete(id, reason string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	if !ok {
		return nil, ErrNotFound
	}
	if s.Kernel != nil {
		_ = s.Kernel.Close()
	}
	s.State = "reclaimed"
	s.ReclaimedReason = strings.TrimSpace(reason)
	if s.ReclaimedReason == "" {
		s.ReclaimedReason = "deleted"
	}
	delete(m.sessions, id)
	return clone(s), nil
}

func (m *Manager) Grant(id string, grants []string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	if !ok {
		return nil, ErrNotFound
	}
	for _, grant := range grants {
		if strings.TrimSpace(grant) != "" {
			s.Grants[strings.TrimSpace(grant)] = struct{}{}
		}
	}
	s.LastActivityAt = m.options.Clock.Now()
	return clone(s), nil
}

func (m *Manager) HasGrant(id, grant string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	if !ok {
		return false
	}
	_, ok = s.Grants[grant]
	return ok
}

func (m *Manager) Touch(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	if !ok {
		return ErrNotFound
	}
	s.LastActivityAt = m.options.Clock.Now()
	return nil
}

func (m *Manager) SetMemoryBytes(id string, bytes int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	if !ok {
		return ErrNotFound
	}
	s.MemoryBytes = bytes
	if m.options.MemoryLimit > 0 && bytes > m.options.MemoryLimit {
		return m.reclaimLocked(id, "memory ceiling exceeded")
	}
	return nil
}

func (m *Manager) ReclaimIdle(now time.Time) []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var reclaimed []string
	for id, s := range m.sessions {
		reason := ""
		if now.Sub(s.LastActivityAt) >= m.options.IdleTimeout {
			reason = "idle timeout exceeded"
		}
		if now.Sub(s.CreatedAt) >= m.options.WallTimeout {
			reason = "wall-clock ceiling exceeded"
		}
		if m.options.MemoryLimit > 0 && s.MemoryBytes > m.options.MemoryLimit {
			reason = "memory ceiling exceeded"
		}
		if reason != "" {
			_ = m.reclaimLocked(id, reason)
			reclaimed = append(reclaimed, id)
		}
	}
	sort.Strings(reclaimed)
	return reclaimed
}

func (m *Manager) reclaimLocked(id, reason string) error {
	s, ok := m.sessions[id]
	if !ok {
		return ErrNotFound
	}
	if s.Kernel != nil {
		_ = s.Kernel.Close()
	}
	s.State = "reclaimed"
	s.ReclaimedReason = reason
	delete(m.sessions, id)
	return ErrReclaimed
}

func grantSet(grants []string) map[string]struct{} {
	out := make(map[string]struct{}, len(grants))
	for _, g := range grants {
		if g = strings.TrimSpace(g); g != "" {
			out[g] = struct{}{}
		}
	}
	return out
}
func clone(s *Session) *Session {
	c := *s
	c.Grants = grantSet(nil)
	for g := range s.Grants {
		c.Grants[g] = struct{}{}
	}
	c.Kernel = nil
	return &c
}
