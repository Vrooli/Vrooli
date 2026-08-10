package planworkshop

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"swarm-manager/internal/storage"
)

type Store struct {
	root string
	mu   sync.Mutex
}

func NewStore(dataRoot string) *Store { return &Store{root: filepath.Join(dataRoot, "plan-workshops")} }

func (s *Store) path(id string) (string, error) {
	if strings.TrimSpace(id) == "" || filepath.Base(id) != id {
		return "", fmt.Errorf("invalid workshop id")
	}
	return filepath.Join(s.root, id+".json"), nil
}

func (s *Store) Load(id string) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	path, err := s.path(id)
	if err != nil {
		return Session{}, err
	}
	var session Session
	found, err := storage.ReadJSON(path, &session)
	if err != nil {
		return Session{}, err
	}
	if !found {
		return Session{}, os.ErrNotExist
	}
	return session, nil
}

func (s *Store) Save(session Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := session.Validate(); err != nil {
		return err
	}
	path, err := s.path(session.ID)
	if err != nil {
		return err
	}
	return storage.WriteJSONAtomic(path, session)
}
