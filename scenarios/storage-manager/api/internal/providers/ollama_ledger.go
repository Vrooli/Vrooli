package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type ollamaLedgerEntry struct {
	FirstObserved time.Time `json:"first_observed"`
	LastUsed      time.Time `json:"last_used"`
}

// FileOllamaUsageLedger is the durable usage record for model retention. It
// records live residency observations, not modified_at (which only describes
// when a model was pulled). Writes use a sibling temporary file and rename so
// a storage-manager restart cannot reset or partially write the safety window.
type FileOllamaUsageLedger struct {
	mu      sync.Mutex
	path    string
	Entries map[string]ollamaLedgerEntry `json:"entries"`
}

func NewFileOllamaUsageLedger(path string) (*FileOllamaUsageLedger, error) {
	l := &FileOllamaUsageLedger{path: path, Entries: map[string]ollamaLedgerEntry{}}
	if err := l.reload(); err != nil {
		return nil, err
	}
	return l, nil
}

func (l *FileOllamaUsageLedger) reload() error {
	data, err := os.ReadFile(l.path)
	if os.IsNotExist(err) {
		l.Entries = map[string]ollamaLedgerEntry{}
		return nil
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, l); err != nil {
		return fmt.Errorf("decode Ollama usage ledger: %w", err)
	}
	if l.Entries == nil {
		l.Entries = map[string]ollamaLedgerEntry{}
	}
	return nil
}

func (l *FileOllamaUsageLedger) Record(_ context.Context, now time.Time, models []OllamaModel) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.reload(); err != nil {
		return err
	}
	for _, model := range models {
		name := model.Name
		if name == "" {
			continue
		}
		entry := l.Entries[name]
		if entry.FirstObserved.IsZero() {
			entry.FirstObserved = now
		}
		entry.LastUsed = now
		l.Entries[name] = entry
	}
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(l.path), ".ollama-ledger-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	removeTempFile := os.Remove
	defer func() { _ = removeTempFile(tmpPath) }()
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, l.path)
}

func (l *FileOllamaUsageLedger) Eligible(model string, now time.Time, window time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	entry, ok := l.Entries[model]
	return ok && !entry.FirstObserved.IsZero() && !entry.LastUsed.IsZero() &&
		now.Sub(entry.FirstObserved) >= window && now.Sub(entry.LastUsed) >= window
}
