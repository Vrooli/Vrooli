package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/vrooli/platform-go"
)

type ollamaLedgerEntry struct {
	FirstObserved time.Time `json:"first_observed"`
	LastUsed      time.Time `json:"last_used"`
}

// ollamaInventoryTotals retain high-water accounting for the complete tag
// inventory. These are logical tag sizes: Ollama may share blobs between
// tags, so they must never be presented as physical filesystem usage.
type ollamaInventoryTotals struct {
	BaselineLogicalBytes int64     `json:"baseline_logical_bytes"`
	CurrentLogicalBytes  int64     `json:"current_logical_bytes"`
	PeakLogicalBytes     int64     `json:"peak_logical_bytes"`
	FirstObservedAt      time.Time `json:"first_observed_at"`
	LastObservedAt       time.Time `json:"last_observed_at"`
}

// FileOllamaUsageLedger is the durable usage record for model retention. It
// records live residency observations, not modified_at (which only describes
// when a model was pulled. A platform-go lock serializes read-modify-write
// across ledger handles/processes; writes use a sibling temporary file and
// rename so restart cannot reset or partially write the safety window.
type FileOllamaUsageLedger struct {
	mu        sync.Mutex
	path      string
	Entries   map[string]ollamaLedgerEntry `json:"entries"`
	Inventory ollamaInventoryTotals        `json:"inventory"`
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

func (l *FileOllamaUsageLedger) Record(ctx context.Context, now time.Time, models []OllamaModel) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	unlock, err := l.acquireFileLock(ctx)
	if err != nil {
		return err
	}
	defer unlock()
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
	return l.persistLocked()
}

func (l *FileOllamaUsageLedger) RecordInventory(ctx context.Context, now time.Time, models []OllamaModel) (ollamaInventoryTotals, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	unlock, err := l.acquireFileLock(ctx)
	if err != nil {
		return ollamaInventoryTotals{}, err
	}
	defer unlock()
	if err := l.reload(); err != nil {
		return ollamaInventoryTotals{}, err
	}
	var current int64
	for _, model := range models {
		current = saturatingAddNonnegative(current, model.Size)
	}
	if l.Inventory.FirstObservedAt.IsZero() || now.Before(l.Inventory.FirstObservedAt) {
		l.Inventory.FirstObservedAt = now
		l.Inventory.BaselineLogicalBytes = current
	}
	if l.Inventory.LastObservedAt.IsZero() || !now.Before(l.Inventory.LastObservedAt) {
		l.Inventory.LastObservedAt = now
		l.Inventory.CurrentLogicalBytes = current
	}
	if current > l.Inventory.PeakLogicalBytes {
		l.Inventory.PeakLogicalBytes = current
	}
	if err := l.persistLocked(); err != nil {
		return ollamaInventoryTotals{}, err
	}
	return l.Inventory, nil
}

func (l *FileOllamaUsageLedger) acquireFileLock(ctx context.Context) (func(), error) {
	if ctx == nil {
		return nil, fmt.Errorf("Ollama usage ledger lock context is unavailable")
	}
	if err := os.MkdirAll(filepath.Dir(l.path), 0o700); err != nil {
		return nil, fmt.Errorf("create Ollama usage ledger directory: %w", err)
	}
	path := l.path + ".lock"
	file, err := platform.OpenLockFile(path, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open Ollama usage ledger lock: %w", err)
	}
	unlock, err := platform.LockFileContext(ctx, file)
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("lock Ollama usage ledger: %w", err)
	}
	if err := platform.VerifyLockFile(file, path); err != nil {
		unlock()
		_ = file.Close()
		return nil, fmt.Errorf("verify Ollama usage ledger lock: %w", err)
	}
	return func() { unlock(); _ = file.Close() }, nil
}

func (l *FileOllamaUsageLedger) persistLocked() error {
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
