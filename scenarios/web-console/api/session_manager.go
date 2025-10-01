package main

import (
	"errors"
	"net/http"
	"strings"
	"sync"
)

type sessionManager struct {
	cfg       config
	metrics   *metricsRegistry
	workspace *workspace

	mu       sync.RWMutex
	sessions map[string]*session
	slots    chan struct{}
}

func newSessionManager(cfg config, metrics *metricsRegistry, ws *workspace) *sessionManager {
	return &sessionManager{
		cfg:       cfg,
		metrics:   metrics,
		workspace: ws,
		sessions:  make(map[string]*session),
		slots:     make(chan struct{}, cfg.maxConcurrent),
	}
}

func (m *sessionManager) createSession(req createSessionRequest) (*session, error) {
	select {
	case m.slots <- struct{}{}:
	default:
		return nil, errors.New("session capacity reached")
	}

	s, err := newSession(m, m.cfg, m.metrics, req)
	if err != nil {
		m.releaseSlot()
		return nil, err
	}

	m.mu.Lock()
	m.sessions[s.id] = s
	m.mu.Unlock()

	m.metrics.activeSessions.Add(1)
	m.metrics.totalSessions.Add(1)

	return s, nil
}

func (m *sessionManager) getSession(id string) (*session, bool) {
	m.mu.RLock()
	s, ok := m.sessions[id]
	m.mu.RUnlock()
	return s, ok
}

func (m *sessionManager) deleteSession(id string, reason closeReason) {
	m.mu.Lock()
	s, ok := m.sessions[id]
	if ok {
		delete(m.sessions, id)
	}
	m.mu.Unlock()
	if ok {
		s.Close(reason)
	}
}

func (m *sessionManager) deleteAllSessions(reason closeReason) int {
	m.mu.Lock()
	toClose := make([]*session, 0, len(m.sessions))
	for id, s := range m.sessions {
		toClose = append(toClose, s)
		delete(m.sessions, id)
	}
	m.mu.Unlock()

	for _, s := range toClose {
		s.Close(reason)
	}

	return len(toClose)
}

func (m *sessionManager) onSessionClosed(s *session, _ closeReason) {
	m.mu.Lock()
	delete(m.sessions, s.id)
	m.mu.Unlock()
	m.releaseSlot()

	// Detach session from any tabs in workspace
	if m.workspace != nil {
		m.workspace.mu.RLock()
		for _, tab := range m.workspace.Tabs {
			if tab.SessionID != nil && *tab.SessionID == s.id {
				m.workspace.mu.RUnlock()
				_ = m.workspace.detachSession(tab.ID)
				return
			}
		}
		m.workspace.mu.RUnlock()
	}
}

func (m *sessionManager) releaseSlot() {
	select {
	case <-m.slots:
	default:
	}
}

func (m *sessionManager) listSummaries() []sessionSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]sessionSummary, 0, len(m.sessions))
	for _, s := range m.sessions {
		result = append(result, sessionSummary{
			ID:           s.id,
			CreatedAt:    s.createdAt,
			ExpiresAt:    s.expiresAt,
			LastActivity: s.lastActivityTime(),
			State:        "active",
			Command:      s.commandName,
			Args:         append([]string{}, s.commandArgs...),
		})
	}
	return result
}

func (m *sessionManager) makeProxyGuard(next http.Handler) http.Handler {
	if !m.cfg.enableProxyGuard {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwardedFor := r.Header.Get("X-Forwarded-For")
		forwardedProto := r.Header.Get("X-Forwarded-Proto")
		if strings.TrimSpace(forwardedFor) == "" || strings.TrimSpace(forwardedProto) == "" {
			writeJSONError(w, http.StatusForbidden, "web console must be behind an authenticated proxy; missing required forwarding headers")
			return
		}
		next.ServeHTTP(w, r)
	})
}
