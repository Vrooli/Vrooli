package desktopoptimize

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// FileJournal is the scenario-owned append-only receipt ledger. Appends are
// flushed before returning; reopening it and calling Find(runID) recovers the
// latest receipt after a process restart. A single adapter instance serializes
// concurrent appends to prevent interleaved JSONL records.
type FileJournal struct {
	Path string
	mu   sync.Mutex
}

func (j *FileJournal) Append(r Receipt) error {
	if strings.TrimSpace(j.Path) == "" {
		return errors.New("receipt journal path is required")
	}
	if r.Schema != ReceiptSchema || strings.TrimSpace(r.RunID) == "" || r.Disposition == "" {
		return errors.New("receipt schema, run_id, and disposition are required")
	}
	data, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("encode receipt: %w", err)
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(j.Path), 0o700); err != nil {
		return fmt.Errorf("create receipt directory: %w", err)
	}
	f, err := os.OpenFile(j.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open receipt journal: %w", err)
	}
	defer f.Close()
	if _, err = f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("append receipt: %w", err)
	}
	if err = f.Sync(); err != nil {
		return fmt.Errorf("sync receipt journal: %w", err)
	}
	return nil
}

func (j *FileJournal) Find(runID string) (Receipt, bool, error) {
	if strings.TrimSpace(j.Path) == "" {
		return Receipt{}, false, errors.New("receipt journal path is required")
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	f, err := os.Open(j.Path)
	if errors.Is(err, os.ErrNotExist) {
		return Receipt{}, false, nil
	}
	if err != nil {
		return Receipt{}, false, fmt.Errorf("open receipt journal: %w", err)
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	var latest Receipt
	found := false
	for {
		line, readErr := reader.ReadBytes('\n')
		if len(line) > 0 {
			var record Receipt
			if err := json.Unmarshal(line, &record); err != nil {
				return Receipt{}, false, fmt.Errorf("decode receipt journal record: %w", err)
			}
			if record.Schema != ReceiptSchema || record.RunID == "" {
				return Receipt{}, false, errors.New("receipt journal contains an invalid record")
			}
			if record.RunID == runID {
				latest, found = record, true
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return Receipt{}, false, fmt.Errorf("read receipt journal: %w", readErr)
		}
	}
	return latest, found, nil
}
