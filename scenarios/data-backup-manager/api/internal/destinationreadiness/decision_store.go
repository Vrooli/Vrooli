package destinationreadiness

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/operatordecision"
)

// FileDecisionStore durably stores Notification Hub asks associated with
// volume-recovery journals. The ask ledger is separate from the host action
// journal: a decision can be retried independently without replaying an
// in-flight filesystem operation.
type FileDecisionStore struct {
	Root string
	mu   sync.Mutex
}

var _ operatordecision.PendingStore = (*FileDecisionStore)(nil)

func (s *FileDecisionStore) path(id string) (string, error) {
	if s == nil || !filepath.IsAbs(s.Root) || strings.TrimSpace(id) == "" || strings.ContainsAny(id, `/\\`) {
		return "", errors.New("invalid volume-recovery decision store or ask id")
	}
	return filepath.Join(filepath.Clean(s.Root), id+".json"), nil
}

func (s *FileDecisionStore) Get(ctx context.Context, id string) (*operatordecision.PendingAsk, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p, err := s.path(id)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ask operatordecision.PendingAsk
	if err := json.Unmarshal(data, &ask); err != nil {
		return nil, err
	}
	return &ask, nil
}

func (s *FileDecisionStore) Put(ctx context.Context, ask *operatordecision.PendingAsk) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if ask == nil {
		return errors.New("volume-recovery decision ask is required")
	}
	p, err := s.path(ask.ID)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(ask, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".decision-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, p)
}

func (s *FileDecisionStore) CountOutstandingForSubject(ctx context.Context, subject string, now time.Time) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if s == nil || !filepath.IsAbs(s.Root) {
		return 0, errors.New("invalid volume-recovery decision store")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := os.ReadDir(filepath.Clean(s.Root))
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(filepath.Clean(s.Root), entry.Name()))
		if err != nil {
			return 0, err
		}
		var ask operatordecision.PendingAsk
		if err := json.Unmarshal(data, &ask); err != nil {
			return 0, err
		}
		if ask.Subject == subject && ask.Outstanding(now) {
			count++
		}
	}
	return count, nil
}
