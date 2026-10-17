package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrMiss    = errors.New("evidence: cache miss")
	ErrCorrupt = errors.New("evidence: corrupt record")
	ErrStale   = errors.New("evidence: stale record")
)

type Record struct {
	KeyDigest string    `json:"key_digest"`
	Complete  bool      `json:"complete"`
	CreatedAt time.Time `json:"created_at"`
	Payload   []byte    `json:"payload"`
	Integrity string    `json:"integrity"`
}

type Store struct {
	root     string
	maxBytes int64
	maxAge   time.Duration
	mu       sync.Mutex
}

func NewStore(root string, maxBytes int64, maxAge time.Duration) (*Store, error) {
	if strings.TrimSpace(root) == "" || maxBytes <= 0 || maxAge <= 0 {
		return nil, fmt.Errorf("evidence: root, maxBytes, and maxAge are required")
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, fmt.Errorf("evidence: create root: %w", err)
	}
	return &Store{root: root, maxBytes: maxBytes, maxAge: maxAge}, nil
}

func (s *Store) Get(key Key, now time.Time) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	path := s.path(key.Digest)
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Record{}, ErrMiss
		}
		return Record{}, fmt.Errorf("evidence: read: %w", err)
	}
	var record Record
	if err := json.Unmarshal(raw, &record); err != nil || !record.Complete || record.KeyDigest != key.Digest {
		s.quarantine(path)
		return Record{}, fmt.Errorf("%w: %w", ErrCorrupt, ErrMiss)
	}
	if now.Sub(record.CreatedAt) > s.maxAge {
		s.quarantine(path)
		return Record{}, fmt.Errorf("%w: %w", ErrStale, ErrMiss)
	}
	digest := sha256.Sum256(record.Payload)
	if record.Integrity != hex.EncodeToString(digest[:]) {
		s.quarantine(path)
		return Record{}, fmt.Errorf("%w: %w", ErrCorrupt, ErrMiss)
	}
	return record, nil
}

func (s *Store) Put(key Key, payload []byte, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	digest := sha256.Sum256(payload)
	record := Record{KeyDigest: key.Digest, Complete: true, CreatedAt: now.UTC(), Payload: append([]byte(nil), payload...), Integrity: hex.EncodeToString(digest[:])}
	raw, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("evidence: encode: %w", err)
	}
	tmp, err := os.CreateTemp(s.root, ".evidence-*.tmp")
	if err != nil {
		return fmt.Errorf("evidence: temporary file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err = tmp.Chmod(0o600); err == nil {
		_, err = tmp.Write(raw)
	}
	if err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("evidence: write: %w", err)
	}
	if err := os.Rename(tmpPath, s.path(key.Digest)); err != nil {
		return fmt.Errorf("evidence: commit: %w", err)
	}
	return s.evict(now.UTC())
}

func (s *Store) path(digest string) string { return filepath.Join(s.root, digest+".json") }

func (s *Store) quarantine(path string) {
	_ = os.Rename(path, path+fmt.Sprintf(".corrupt-%d", time.Now().UnixNano()))
}

func (s *Store) evict(now time.Time) error {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return err
	}
	type item struct {
		path    string
		size    int64
		created time.Time
	}
	items := make([]item, 0)
	var total int64
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(s.root, entry.Name())
		info, statErr := entry.Info()
		if statErr != nil {
			continue
		}
		var record Record
		if raw, readErr := os.ReadFile(path); readErr == nil {
			_ = json.Unmarshal(raw, &record)
		}
		if !record.Complete || now.Sub(record.CreatedAt) > s.maxAge {
			_ = os.Remove(path)
			continue
		}
		items = append(items, item{path: path, size: info.Size(), created: record.CreatedAt})
		total += info.Size()
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].created.Equal(items[j].created) {
			return items[i].path < items[j].path
		}
		return items[i].created.Before(items[j].created)
	})
	for _, item := range items {
		if total <= s.maxBytes {
			break
		}
		_ = os.Remove(item.path)
		total -= item.size
	}
	return nil
}
