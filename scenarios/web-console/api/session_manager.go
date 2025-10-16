package main

import (
	"errors"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type sessionManager struct {
	cfg       config
	metrics   *metricsRegistry
	workspace *workspace

	mu       sync.RWMutex
	sessions map[string]*session
	slots    chan struct{}

	idleTimeout    atomic.Int64
	globalActivity atomic.Int64
}

func newSessionManager(cfg config, metrics *metricsRegistry, ws *workspace) *sessionManager {
	m := &sessionManager{
		cfg:       cfg,
		metrics:   metrics,
		workspace: ws,
		sessions:  make(map[string]*session),
		slots:     make(chan struct{}, cfg.maxConcurrent),
	}
	m.idleTimeout.Store(int64(cfg.idleTimeout))
	m.globalActivity.Store(time.Now().UTC().UnixNano())
	go m.monitorGlobalIdle()
	return m
}

func (m *sessionManager) createSession(req createSessionRequest) (*session, error) {
	select {
	case m.slots <- struct{}{}:
	default:
		return nil, errors.New("session capacity reached")
	}

	cfg := m.cfg
	cfg.idleTimeout = m.getIdleTimeout()
	s, err := newSession(m, cfg, m.metrics, req)
	if err != nil {
		m.releaseSlot()
		return nil, err
	}

	m.mu.Lock()
	m.sessions[s.id] = s
	m.mu.Unlock()

	m.metrics.activeSessions.Add(1)
	m.metrics.totalSessions.Add(1)
	m.recordActivity()

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
	m.recordActivity()

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

func (m *sessionManager) capacity() int {
	return m.cfg.maxConcurrent
}

func (m *sessionManager) getIdleTimeout() time.Duration {
	value := time.Duration(m.idleTimeout.Load())
	if value < 0 {
		return 0
	}
	return value
}

func (m *sessionManager) recordActivity() {
	m.globalActivity.Store(time.Now().UTC().UnixNano())
}

func (m *sessionManager) updateIdleTimeout(d time.Duration) {
	if d < 0 {
		d = 0
	}
	m.idleTimeout.Store(int64(d))
	m.recordActivity()
}

func (m *sessionManager) monitorGlobalIdle() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		timeout := m.getIdleTimeout()
		if timeout <= 0 {
			continue
		}
		last := time.Unix(0, m.globalActivity.Load()).UTC()
		if time.Since(last) < timeout {
			continue
		}
		if count := m.deleteAllSessions(reasonIdleTimeout); count > 0 {
			m.recordActivity()
		} else {
			m.recordActivity()
		}
	}
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
