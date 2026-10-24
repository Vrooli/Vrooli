package livedesktop

import (
	"fmt"
	"sync"
)

// Store manages session persistence.
type Store interface {
	Create(session *Session) error
	Get(id string) (*Session, error)
	Update(session *Session) error
	Delete(id string) error
	List() []*Session
	ActiveSessions() []*Session
}

// InMemoryStore implements Store with an in-memory map.
type InMemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewInMemoryStore creates a new in-memory session store.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		sessions: make(map[string]*Session),
	}
}

func (s *InMemoryStore) Create(session *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.sessions[session.ID]; exists {
		return fmt.Errorf("session %s already exists", session.ID)
	}
	s.sessions[session.ID] = session
	return nil
}

func (s *InMemoryStore) Get(id string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session %s not found", id)
	}
	return session, nil
}

func (s *InMemoryStore) Update(session *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.sessions[session.ID]; !exists {
		return fmt.Errorf("session %s not found", session.ID)
	}
	s.sessions[session.ID] = session
	return nil
}

func (s *InMemoryStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.sessions[id]; !exists {
		return fmt.Errorf("session %s not found", id)
	}
	delete(s.sessions, id)
	return nil
}

func (s *InMemoryStore) List() []*Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Session, 0, len(s.sessions))
	for _, session := range s.sessions {
		result = append(result, session)
	}
	return result
}

func (s *InMemoryStore) ActiveSessions() []*Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*Session
	for _, session := range s.sessions {
		if session.State == StateCreating || session.State == StateRunning {
			result = append(result, session)
		}
	}
	return result
}
