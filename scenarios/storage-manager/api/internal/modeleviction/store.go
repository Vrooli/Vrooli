package modeleviction

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/vrooli/api-core/operatordecision"
	corestorage "github.com/vrooli/api-core/storage"
	platform "github.com/vrooli/platform-go"
)

// FileStore is the durable record of model-eviction decisions: the
// operatordecision PendingStore plus the last notice sent. A handful of
// records a week, so one JSON file under the scenario state directory.
type FileStore struct {
	path string
	mu   sync.Mutex
}

type storeDocument struct {
	Asks          map[string]operatordecision.PendingAsk `json:"asks"`
	LastNoticeKey string                                 `json:"last_notice_key,omitempty"`
}

// NewFileStore opens (or will create) the store at path.
func NewFileStore(path string) *FileStore { return &FileStore{path: path} }

func (s *FileStore) update(ctx context.Context, fn func(*storeDocument) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	release, err := platform.AcquireFileLockContext(ctx, s.path+".lock")
	if err != nil {
		return fmt.Errorf("lock model eviction store: %w", err)
	}
	defer release()
	doc, err := s.read()
	if err != nil {
		return err
	}
	if err := fn(&doc); err != nil {
		return err
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return corestorage.WriteFileAtomic(s.path, append(data, '\n'), 0o600)
}

func (s *FileStore) read() (storeDocument, error) {
	doc := storeDocument{Asks: map[string]operatordecision.PendingAsk{}}
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return doc, nil
	}
	if err != nil {
		return doc, err
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return doc, fmt.Errorf("decode model eviction store: %w", err)
	}
	if doc.Asks == nil {
		doc.Asks = map[string]operatordecision.PendingAsk{}
	}
	return doc, nil
}

func (s *FileStore) snapshot() (storeDocument, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.read()
}

// Get returns the ask for a correlation id, or nil when none exists.
func (s *FileStore) Get(_ context.Context, id string) (*operatordecision.PendingAsk, error) {
	doc, err := s.snapshot()
	if err != nil {
		return nil, err
	}
	ask, ok := doc.Asks[id]
	if !ok {
		return nil, nil
	}
	return &ask, nil
}

// Put inserts or replaces an ask by id.
func (s *FileStore) Put(ctx context.Context, ask *operatordecision.PendingAsk) error {
	return s.update(ctx, func(doc *storeDocument) error {
		doc.Asks[ask.ID] = *ask
		return nil
	})
}

// CountOutstandingForSubject counts un-decided, un-expired asks for a subject.
func (s *FileStore) CountOutstandingForSubject(_ context.Context, subject string, now time.Time) (int, error) {
	doc, err := s.snapshot()
	if err != nil {
		return 0, err
	}
	count := 0
	for _, ask := range doc.Asks {
		if ask.Subject == subject && ask.Outstanding(now) {
			count++
		}
	}
	return count, nil
}

// Pending returns every ask still awaiting an answer, oldest first.
func (s *FileStore) Pending(_ context.Context) ([]operatordecision.PendingAsk, error) {
	doc, err := s.snapshot()
	if err != nil {
		return nil, err
	}
	var out []operatordecision.PendingAsk
	for _, ask := range doc.Asks {
		if ask.State == operatordecision.PendingStatePending {
			out = append(out, ask)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RaisedAt.Before(out[j].RaisedAt) })
	return out, nil
}

// SwapNoticeKey records key and reports whether it differs from the last one.
func (s *FileStore) SwapNoticeKey(ctx context.Context, key string) (bool, error) {
	changed := false
	err := s.update(ctx, func(doc *storeDocument) error {
		changed = doc.LastNoticeKey != key
		doc.LastNoticeKey = key
		return nil
	})
	return changed, err
}

// ForgetNoticeKey lets the next tick resend after a failed send.
func (s *FileStore) ForgetNoticeKey(ctx context.Context) error {
	return s.update(ctx, func(doc *storeDocument) error {
		doc.LastNoticeKey = ""
		return nil
	})
}
