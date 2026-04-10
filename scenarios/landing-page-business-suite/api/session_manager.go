package main

import (
	"net/http"

	"github.com/gorilla/sessions"
)

// SessionManager provides an interface for session management operations.
// This abstraction allows for easy mocking in tests.
type SessionManager interface {
	// GetSession retrieves or creates a session with the given name.
	GetSession(r *http.Request, name string) (*sessions.Session, error)
	// SaveSession persists the session to the response.
	SaveSession(r *http.Request, w http.ResponseWriter, session *sessions.Session) error
}

// cookieSessionManager implements SessionManager using gorilla/sessions CookieStore.
type cookieSessionManager struct {
	store *sessions.CookieStore
}

// NewCookieSessionManager creates a new session manager backed by a cookie store.
func NewCookieSessionManager(secret string) SessionManager {
	return &cookieSessionManager{
		store: sessions.NewCookieStore([]byte(secret)),
	}
}

// GetSession retrieves or creates a session with the given name.
func (m *cookieSessionManager) GetSession(r *http.Request, name string) (*sessions.Session, error) {
	return m.store.Get(r, name)
}

// SaveSession persists the session to the response.
func (m *cookieSessionManager) SaveSession(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	return session.Save(r, w)
}

// mockSessionManager is a test implementation of SessionManager.
// It's exported so tests in other packages can use it if needed.
type MockSessionManager struct {
	Sessions  map[string]*sessions.Session
	SaveError error
	GetError  error
}

// NewMockSessionManager creates a new mock session manager for testing.
func NewMockSessionManager() *MockSessionManager {
	return &MockSessionManager{
		Sessions: make(map[string]*sessions.Session),
	}
}

// GetSession returns a mock session.
func (m *MockSessionManager) GetSession(r *http.Request, name string) (*sessions.Session, error) {
	if m.GetError != nil {
		return nil, m.GetError
	}
	if session, ok := m.Sessions[name]; ok {
		return session, nil
	}
	// Create new empty session
	session := sessions.NewSession(nil, name)
	session.Values = make(map[interface{}]interface{})
	session.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
	}
	m.Sessions[name] = session
	return session, nil
}

// SaveSession is a no-op for mock (or returns configured error).
func (m *MockSessionManager) SaveSession(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	return m.SaveError
}

// SetSessionValues sets values on a named session for testing.
func (m *MockSessionManager) SetSessionValues(name string, values map[interface{}]interface{}) {
	session, _ := m.GetSession(nil, name)
	for k, v := range values {
		session.Values[k] = v
	}
}
