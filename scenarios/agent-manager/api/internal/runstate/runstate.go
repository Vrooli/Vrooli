package runstate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

const (
	metaFileName       = "meta.json"
	transcriptFileName = "transcript.ndjson"
	stderrFileName     = "stderr.log"
	cursorFileName     = "cursor.json"
)

type Meta struct {
	RunID      string            `json:"run_id"`
	RunnerType domain.RunnerType `json:"runner_type"`
	RunnerPID  int               `json:"runner_pid,omitempty"`
	RunnerPGID int               `json:"runner_pgid,omitempty"`
	WorkingDir string            `json:"working_dir"`
	StartedAt  time.Time         `json:"started_at"`
	SessionID  string            `json:"session_id,omitempty"`
}

type Cursor struct {
	TranscriptCursor  int64     `json:"transcript_cursor"`
	TranscriptLastSeq int64     `json:"transcript_last_seq"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type Snapshot struct {
	Dir            string
	MetaPath       string
	TranscriptPath string
	StderrPath     string
	CursorPath     string
	Meta           Meta
	Cursor         Cursor
}

type OpenOptions struct {
	RootDir    string
	RunnerType domain.RunnerType
	WorkingDir string
	StartedAt  time.Time
	// OnWrite is called after each successful durable state mutation.
	OnWrite func()
}

type State struct {
	snapshot   Snapshot
	transcript *os.File
	stderr     *os.File
	onWrite    func()
	mu         sync.Mutex
}

func RunDir(rootDir string, runID uuid.UUID) (string, error) {
	if rootDir == "" {
		return "", fmt.Errorf("run state root is required")
	}
	return filepath.Join(rootDir, runID.String()), nil
}

func Open(runID uuid.UUID, opts OpenOptions) (*State, error) {
	dir, err := RunDir(opts.RootDir, runID)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create run state dir: %w", err)
	}

	startedAt := opts.StartedAt
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}

	s := &State{
		onWrite: opts.OnWrite,
		snapshot: Snapshot{
			Dir:            dir,
			MetaPath:       filepath.Join(dir, metaFileName),
			TranscriptPath: filepath.Join(dir, transcriptFileName),
			StderrPath:     filepath.Join(dir, stderrFileName),
			CursorPath:     filepath.Join(dir, cursorFileName),
			Meta: Meta{
				RunID:      runID.String(),
				RunnerType: opts.RunnerType,
				WorkingDir: opts.WorkingDir,
				StartedAt:  startedAt,
			},
			Cursor: Cursor{},
		},
	}

	if err := atomicWriteJSON(s.snapshot.MetaPath, s.snapshot.Meta); err != nil {
		return nil, fmt.Errorf("write meta.json: %w", err)
	}
	if err := atomicWriteJSON(s.snapshot.CursorPath, s.snapshot.Cursor); err != nil {
		return nil, fmt.Errorf("write cursor.json: %w", err)
	}

	transcript, err := os.OpenFile(s.snapshot.TranscriptPath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open transcript: %w", err)
	}
	stderr, err := os.OpenFile(s.snapshot.StderrPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		_ = transcript.Close()
		return nil, fmt.Errorf("open stderr log: %w", err)
	}

	s.transcript = transcript
	s.stderr = stderr
	s.recordWrite()
	return s, nil
}

func Load(runID uuid.UUID, rootDir string) (*Snapshot, error) {
	dir, err := RunDir(rootDir, runID)
	if err != nil {
		return nil, err
	}
	s := &Snapshot{
		Dir:            dir,
		MetaPath:       filepath.Join(dir, metaFileName),
		TranscriptPath: filepath.Join(dir, transcriptFileName),
		StderrPath:     filepath.Join(dir, stderrFileName),
		CursorPath:     filepath.Join(dir, cursorFileName),
	}
	metaBytes, err := os.ReadFile(s.MetaPath)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(metaBytes, &s.Meta); err != nil {
		return nil, fmt.Errorf("read meta.json: %w", err)
	}
	cursorBytes, err := os.ReadFile(s.CursorPath)
	if err == nil {
		if err := json.Unmarshal(cursorBytes, &s.Cursor); err != nil {
			return nil, fmt.Errorf("read cursor.json: %w", err)
		}
	}
	return s, nil
}

func (s *State) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshot
}

func (s *State) TranscriptWriter() *os.File {
	return s.transcript
}

func (s *State) StderrWriter() *os.File {
	return s.stderr
}

func (s *State) PersistProcess(pid, pgid int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshot.Meta.RunnerPID = pid
	s.snapshot.Meta.RunnerPGID = pgid
	err := atomicWriteJSON(s.snapshot.MetaPath, s.snapshot.Meta)
	if err == nil {
		s.recordWrite()
	}
	return err
}

func (s *State) PersistSessionID(sessionID string) error {
	if sessionID == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.snapshot.Meta.SessionID == sessionID {
		return nil
	}
	s.snapshot.Meta.SessionID = sessionID
	err := atomicWriteJSON(s.snapshot.MetaPath, s.snapshot.Meta)
	if err == nil {
		s.recordWrite()
	}
	return err
}

func (s *State) PersistCursor(cursor, lastSeq int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshot.Cursor.TranscriptCursor = cursor
	s.snapshot.Cursor.TranscriptLastSeq = lastSeq
	s.snapshot.Cursor.UpdatedAt = time.Now().UTC()
	err := atomicWriteJSON(s.snapshot.CursorPath, s.snapshot.Cursor)
	if err == nil {
		s.recordWrite()
	}
	return err
}

func (s *State) recordWrite() {
	if s.onWrite != nil {
		s.onWrite()
	}
}

func (s *State) Close() error {
	var firstErr error
	if s.transcript != nil {
		if err := s.transcript.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if s.stderr != nil {
		if err := s.stderr.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func atomicWriteJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
