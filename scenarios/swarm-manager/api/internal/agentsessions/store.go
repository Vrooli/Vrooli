package agentsessions

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"swarm-manager/internal/storage"
)

var ErrNotFound = errors.New("agent session not found")

const (
	sessionFileName   = "session.json"
	messagesFileName  = "messages.jsonl"
	artifactsFileName = "artifacts.jsonl"
	proposalsDirName  = "proposals"
)

type Store interface {
	CreateSession(session Session) error
	SaveSession(session Session) error
	DeleteSession(sessionID string) error
	LoadSession(sessionID string) (Session, error)
	ListSessions(filters ListFilters) ([]Session, error)
	AppendMessage(sessionID string, message Message) error
	SaveProposal(sessionID string, proposal Proposal) error
	AppendArtifact(sessionID string, artifact Artifact) error
	AppendArtifacts(sessionID string, artifacts []Artifact) error
	ListArtifacts(sessionID string) ([]Artifact, error)
	ListArtifactsByEntity(artifactType ArtifactType, entityRef string) ([]Artifact, error)
}

type ListFilters struct {
	Kind       Kind
	Status     Status
	ActiveOnly bool
	Limit      int
}

type FileStore struct {
	root string
	mu   sync.Mutex
}

func NewFileStore(root string) *FileStore {
	return &FileStore{root: filepath.Join(root, "agent-sessions")}
}

func (s *FileStore) CreateSession(session Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session = snapshotOnly(session)
	if err := session.Validate(); err != nil {
		return err
	}
	dir := s.sessionDir(session.ID)
	if _, err := os.Stat(filepath.Join(dir, sessionFileName)); err == nil {
		return fmt.Errorf("%w: %s", ErrValidation, "session already exists")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dir, proposalsDirName), 0o755); err != nil {
		return err
	}
	return storage.WriteJSONAtomic(filepath.Join(dir, sessionFileName), session)
}

func (s *FileStore) SaveSession(session Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveSessionLocked(session)
}

func (s *FileStore) DeleteSession(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	safeID, err := safeSessionID(sessionID)
	if err != nil {
		return err
	}
	if _, err := s.loadSessionLocked(safeID); err != nil {
		return err
	}
	return os.RemoveAll(s.sessionDir(safeID))
}

func (s *FileStore) LoadSession(sessionID string) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadSessionLocked(sessionID)
}

func (s *FileStore) ListSessions(filters ListFilters) ([]Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Session{}, nil
		}
		return nil, err
	}

	sessions := make([]Session, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		session, err := s.loadSessionLocked(entry.Name())
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				continue
			}
			return nil, err
		}
		if !matchesListFilters(session, filters) {
			continue
		}
		sessions = append(sessions, session)
	}

	sort.Slice(sessions, func(i, j int) bool {
		if sessions[i].UpdatedAt == sessions[j].UpdatedAt {
			return sessions[i].ID > sessions[j].ID
		}
		return sessions[i].UpdatedAt > sessions[j].UpdatedAt
	})
	if filters.Limit > 0 && len(sessions) > filters.Limit {
		sessions = sessions[:filters.Limit]
	}
	return sessions, nil
}

func (s *FileStore) AppendMessage(sessionID string, message Message) error {
	if err := message.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	session, err := s.loadSessionLocked(sessionID)
	if err != nil {
		return err
	}
	if err := appendJSONL(filepath.Join(s.sessionDir(session.ID), messagesFileName), message); err != nil {
		return err
	}
	session.UpdatedAt = message.CreatedAt
	return s.saveSessionLocked(session)
}

func (s *FileStore) SaveProposal(sessionID string, proposal Proposal) error {
	if err := proposal.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	session, err := s.loadSessionLocked(sessionID)
	if err != nil {
		return err
	}
	path := filepath.Join(s.sessionDir(session.ID), proposalsDirName, proposal.ID+".json")
	if err := storage.WriteJSONAtomic(path, proposal); err != nil {
		return err
	}
	session.UpdatedAt = proposal.UpdatedAt
	if proposal.Status == ProposalStatusReady {
		session.Status = StatusProposalReady
	}
	return s.saveSessionLocked(session)
}

func (s *FileStore) AppendArtifact(sessionID string, artifact Artifact) error {
	if err := artifact.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	session, err := s.loadSessionLocked(sessionID)
	if err != nil {
		return err
	}
	if artifact.SessionID != session.ID {
		return validationError("artifact session_id does not match session")
	}
	if err := appendJSONL(filepath.Join(s.sessionDir(session.ID), artifactsFileName), artifact); err != nil {
		return err
	}
	session.UpdatedAt = artifact.CreatedAt
	return s.saveSessionLocked(session)
}

func (s *FileStore) AppendArtifacts(sessionID string, artifacts []Artifact) error {
	if len(artifacts) == 0 {
		return nil
	}
	for i := range artifacts {
		if err := artifacts[i].Validate(); err != nil {
			return fmt.Errorf("%w: artifacts[%d]: %v", ErrValidation, i, err)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	session, err := s.loadSessionLocked(sessionID)
	if err != nil {
		return err
	}
	for i := range artifacts {
		if artifacts[i].SessionID != session.ID {
			return validationError("artifact session_id does not match session")
		}
	}
	path := filepath.Join(s.sessionDir(session.ID), artifactsFileName)
	existing, err := readJSONL[Artifact](path)
	if err != nil {
		return err
	}
	next := append(existing, artifacts...)
	if err := writeJSONLAtomic(path, next); err != nil {
		return err
	}
	session.UpdatedAt = artifacts[len(artifacts)-1].CreatedAt
	return s.saveSessionLocked(session)
}

func (s *FileStore) ListArtifacts(sessionID string) ([]Artifact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.loadSessionLocked(sessionID); err != nil {
		return nil, err
	}
	return readJSONL[Artifact](filepath.Join(s.sessionDir(sessionID), artifactsFileName))
}

func (s *FileStore) ListArtifactsByEntity(artifactType ArtifactType, entityRef string) ([]Artifact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Artifact{}, nil
		}
		return nil, err
	}
	var matches []Artifact
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		artifacts, err := readJSONL[Artifact](filepath.Join(s.root, entry.Name(), artifactsFileName))
		if err != nil {
			return nil, err
		}
		for _, artifact := range artifacts {
			if artifact.ArtifactType == artifactType && strings.TrimSpace(artifact.EntityRef) == strings.TrimSpace(entityRef) {
				matches = append(matches, artifact)
			}
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].CreatedAt == matches[j].CreatedAt {
			return matches[i].ID > matches[j].ID
		}
		return matches[i].CreatedAt > matches[j].CreatedAt
	})
	return matches, nil
}

func (s *FileStore) saveSessionLocked(session Session) error {
	session = snapshotOnly(session)
	if err := session.Validate(); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(s.sessionDir(session.ID), sessionFileName)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrNotFound, session.ID)
		}
		return err
	}
	return storage.WriteJSONAtomic(filepath.Join(s.sessionDir(session.ID), sessionFileName), session)
}

func (s *FileStore) loadSessionLocked(sessionID string) (Session, error) {
	trimmed, err := safeSessionID(sessionID)
	if err != nil {
		return Session{}, err
	}
	var session Session
	exists, err := storage.ReadJSON(filepath.Join(s.sessionDir(trimmed), sessionFileName), &session)
	if err != nil {
		return Session{}, err
	}
	if !exists {
		return Session{}, fmt.Errorf("%w: %s", ErrNotFound, trimmed)
	}
	messages, err := readJSONL[Message](filepath.Join(s.sessionDir(trimmed), messagesFileName))
	if err != nil {
		return Session{}, err
	}
	proposals, err := s.readProposalsLocked(trimmed)
	if err != nil {
		return Session{}, err
	}
	artifacts, err := readJSONL[Artifact](filepath.Join(s.sessionDir(trimmed), artifactsFileName))
	if err != nil {
		return Session{}, err
	}
	session.Messages = messages
	session.Proposals = proposals
	session.Artifacts = artifacts
	if err := session.Validate(); err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *FileStore) readProposalsLocked(sessionID string) ([]Proposal, error) {
	dir := filepath.Join(s.sessionDir(sessionID), proposalsDirName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Proposal{}, nil
		}
		return nil, err
	}
	proposals := make([]Proposal, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var proposal Proposal
		exists, err := storage.ReadJSON(filepath.Join(dir, entry.Name()), &proposal)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		proposals = append(proposals, proposal)
	}
	sort.Slice(proposals, func(i, j int) bool {
		if proposals[i].UpdatedAt == proposals[j].UpdatedAt {
			return proposals[i].ID > proposals[j].ID
		}
		return proposals[i].UpdatedAt > proposals[j].UpdatedAt
	})
	return proposals, nil
}

func (s *FileStore) sessionDir(sessionID string) string {
	return filepath.Join(s.root, strings.TrimSpace(sessionID))
}

func safeSessionID(sessionID string) (string, error) {
	trimmed := strings.TrimSpace(sessionID)
	if trimmed == "" {
		return "", validationError("session_id is required")
	}
	if !strings.HasPrefix(trimmed, "sess_") {
		return "", validationError("session_id must start with sess_")
	}
	if trimmed == "." || trimmed == ".." || strings.Contains(trimmed, "/") || strings.Contains(trimmed, "\\") {
		return "", validationError("session_id is invalid")
	}
	if filepath.Clean(trimmed) != trimmed {
		return "", validationError("session_id is invalid")
	}
	return trimmed, nil
}

func snapshotOnly(session Session) Session {
	session.Messages = nil
	session.Proposals = nil
	session.Artifacts = nil
	return session
}

func matchesListFilters(session Session, filters ListFilters) bool {
	if filters.Kind != "" && session.Kind != filters.Kind {
		return false
	}
	if filters.Status != "" && session.Status != filters.Status {
		return false
	}
	if filters.ActiveOnly && !isActiveSessionStatus(session.Status) {
		return false
	}
	return true
}

func isActiveSessionStatus(status Status) bool {
	switch status {
	case StatusDraft, StatusStarting, StatusRunning, StatusWaitingForUser, StatusProposalReady, StatusApplying:
		return true
	default:
		return false
	}
}

func appendJSONL(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(append(data, '\n')); err != nil {
		return err
	}
	return file.Sync()
}

func writeJSONLAtomic[T any](path string, values []T) error {
	parentDir := filepath.Dir(path)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return err
	}
	tempFile, err := os.CreateTemp(parentDir, "tmp-*.jsonl")
	if err != nil {
		return err
	}
	tempName := tempFile.Name()
	defer func() {
		_ = os.Remove(tempName)
	}()
	for _, value := range values {
		data, err := json.Marshal(value)
		if err != nil {
			_ = tempFile.Close()
			return err
		}
		if _, err := tempFile.Write(append(data, '\n')); err != nil {
			_ = tempFile.Close()
			return err
		}
	}
	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tempName, 0o644); err != nil {
		return err
	}
	return os.Rename(tempName, path)
}

func readJSONL[T any](path string) ([]T, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []T{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var values []T
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var value T
		if err := json.Unmarshal([]byte(line), &value); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if values == nil {
		return []T{}, nil
	}
	return values, nil
}
