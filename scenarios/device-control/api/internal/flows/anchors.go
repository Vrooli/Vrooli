package flows

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Anchor is a durable-safe visual checkpoint. Bounds are normalized to the
// capture dimensions, so a saved anchor can be replayed across device scales.
type Anchor struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Target      string    `json:"target"`
	Bounds      []float64 `json:"bounds"`
	Confidence  float64   `json:"confidence"`
	CreatedAt   time.Time `json:"created_at"`
	FrameSHA256 string    `json:"frame_sha256,omitempty"`
}

// AnchorStore is the resolver's small persistence seam. The control module can
// replace it with a database-backed implementation later without changing the
// flow or UI contract.
type AnchorStore struct {
	mu      sync.RWMutex
	anchors map[string]Anchor
	path    string
	db      AnchorDB
}

type AnchorDB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func NewAnchorStore() *AnchorStore {
	path := strings.TrimSpace(os.Getenv("DEVICE_CONTROL_ANCHORS_FILE"))
	if path == "" {
		if dir, err := os.UserConfigDir(); err == nil {
			path = filepath.Join(dir, "vrooli", "device-control", "anchors.json")
		}
	}
	return NewAnchorStoreAt(path)
}

func NewAnchorStoreAt(path string) *AnchorStore {
	s := &AnchorStore{anchors: map[string]Anchor{}, path: path}
	s.load()
	return s
}

func NewAnchorStoreWithDB(db AnchorDB) (*AnchorStore, error) {
	s := &AnchorStore{anchors: map[string]Anchor{}, db: db}
	if db == nil {
		return s, nil
	}
	if _, err := db.ExecContext(context.Background(), `CREATE TABLE IF NOT EXISTS device_control_anchors (
 id TEXT PRIMARY KEY, name TEXT NOT NULL, target TEXT NOT NULL, bounds_json TEXT NOT NULL,
 confidence REAL NOT NULL, created_at TEXT NOT NULL, frame_sha256 TEXT NOT NULL DEFAULT ''
); CREATE INDEX IF NOT EXISTS device_control_anchors_target ON device_control_anchors(target, name);`); err != nil {
		return nil, fmt.Errorf("initialize anchor store: %w", err)
	}
	_, _ = db.ExecContext(context.Background(), `ALTER TABLE device_control_anchors ADD COLUMN frame_sha256 TEXT NOT NULL DEFAULT ''`)
	rows, err := db.QueryContext(context.Background(), `SELECT id, name, target, bounds_json, confidence, created_at, frame_sha256 FROM device_control_anchors`)
	if err != nil {
		return nil, fmt.Errorf("load anchors: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var anchor Anchor
		var boundsJSON, createdAt string
		if err := rows.Scan(&anchor.ID, &anchor.Name, &anchor.Target, &boundsJSON, &anchor.Confidence, &createdAt, &anchor.FrameSHA256); err != nil {
			return nil, fmt.Errorf("read anchor: %w", err)
		}
		if err := json.Unmarshal([]byte(boundsJSON), &anchor.Bounds); err != nil {
			return nil, fmt.Errorf("decode anchor bounds: %w", err)
		}
		anchor.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, fmt.Errorf("decode anchor timestamp: %w", err)
		}
		s.anchors[anchor.ID] = anchor
	}
	return s, rows.Err()
}

func (s *AnchorStore) load() {
	if s == nil || s.path == "" {
		return
	}
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var anchors []Anchor
	if json.Unmarshal(raw, &anchors) == nil {
		for _, anchor := range anchors {
			s.anchors[anchor.ID] = anchor
		}
	}
}

func (s *AnchorStore) persistLocked() error {
	if s == nil || s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	anchors := make([]Anchor, 0, len(s.anchors))
	for _, anchor := range s.anchors {
		anchors = append(anchors, anchor)
	}
	raw, err := json.MarshalIndent(anchors, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *AnchorStore) Create(name, target string, bounds []float64, confidence float64) (Anchor, error) {
	return s.create(name, target, bounds, confidence, nil)
}

func (s *AnchorStore) CreateFromFrame(name, target string, bounds []float64, confidence float64, frame []byte) (Anchor, error) {
	return s.create(name, target, bounds, confidence, frame)
}

func (s *AnchorStore) create(name, target string, bounds []float64, confidence float64, frame []byte) (Anchor, error) {
	if s == nil || strings.TrimSpace(name) == "" || strings.TrimSpace(target) == "" || !validBounds(bounds) {
		return Anchor{}, fmt.Errorf("name, target, and normalized bounds are required")
	}
	if !finite(confidence) || confidence < 0 || confidence > 1 {
		return Anchor{}, fmt.Errorf("confidence must be between 0 and 1")
	}
	a := Anchor{ID: uuid.NewString(), Name: strings.TrimSpace(name), Target: strings.TrimSpace(target), Bounds: append([]float64(nil), bounds...), Confidence: confidence, CreatedAt: time.Now().UTC()}
	if len(frame) > 0 {
		digest := sha256.Sum256(frame)
		a.FrameSHA256 = hex.EncodeToString(digest[:])
	}
	s.mu.Lock()
	s.anchors[a.ID] = a
	err := s.persistLocked()
	if err == nil && s.db != nil {
		boundsJSON, marshalErr := json.Marshal(a.Bounds)
		if marshalErr != nil {
			err = marshalErr
		} else {
			_, err = s.db.ExecContext(context.Background(), `INSERT INTO device_control_anchors (id, name, target, bounds_json, confidence, created_at, frame_sha256) VALUES (?, ?, ?, ?, ?, ?, ?)`, a.ID, a.Name, a.Target, string(boundsJSON), a.Confidence, a.CreatedAt.Format(time.RFC3339Nano), a.FrameSHA256)
		}
	}
	s.mu.Unlock()
	if err != nil {
		return Anchor{}, fmt.Errorf("persist anchor: %w", err)
	}
	return a, nil
}

func (s *AnchorStore) List() []Anchor {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Anchor, 0, len(s.anchors))
	for _, a := range s.anchors {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (s *AnchorStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.anchors[id]; !ok {
		return fmt.Errorf("anchor %q not found", id)
	}
	delete(s.anchors, id)
	if err := s.persistLocked(); err != nil {
		return err
	}
	if s.db != nil {
		_, err := s.db.ExecContext(context.Background(), `DELETE FROM device_control_anchors WHERE id = ?`, id)
		return err
	}
	return nil
}

func (s *AnchorStore) Resolve(target string) (Anchor, bool) {
	for _, a := range s.List() {
		if a.Name == target || a.Target == target {
			return a, true
		}
	}
	return Anchor{}, false
}
