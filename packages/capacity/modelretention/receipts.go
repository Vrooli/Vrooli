package modelretention

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	platform "github.com/vrooli/platform-go"
)

// Receipt statuses. An owner appends "evicting" before it deletes and
// "evicted" or "failed" after, so an interrupted eviction still leaves the
// reacquisition command on disk.
const (
	ReceiptEvicting = "evicting"
	ReceiptEvicted  = "evicted"
	ReceiptFailed   = "failed"
)

// Receipt is one line of model-evictions.jsonl.
type Receipt struct {
	Resource      string    `json:"resource"`
	Model         string    `json:"model"`
	Digest        string    `json:"digest"`
	Bytes         int64     `json:"bytes"`
	UniqueBytes   int64     `json:"unique_bytes"`
	Status        string    `json:"status"`
	At            time.Time `json:"at"`
	LastRequestAt time.Time `json:"last_request_at,omitempty"`
	// Reacquire is the exact argv that brings the model back.
	Reacquire []string `json:"reacquire"`
	// ReacquireExact is true when reacquisition reproduces the same digest
	// (a digest-pinned pull); false when it re-resolves a registry tag that
	// may have moved since the evicted copy was pulled.
	ReacquireExact bool   `json:"reacquire_exact"`
	Reason         string `json:"reason,omitempty"`
	RequestedBy    string `json:"requested_by,omitempty"`
	Error          string `json:"error,omitempty"`
}

// AppendReceipt appends one receipt durably (fsync) under the file's lock.
func AppendReceipt(ctx context.Context, path string, receipt Receipt) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("model eviction receipts path is not configured")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create model eviction receipts directory: %w", err)
	}
	release, err := platform.AcquireFileLockContext(ctx, path+".lock")
	if err != nil {
		return fmt.Errorf("lock model eviction receipts: %w", err)
	}
	defer release()
	line, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open model eviction receipts: %w", err)
	}
	if _, err := file.Write(append(line, '\n')); err != nil {
		_ = file.Close()
		return fmt.Errorf("write model eviction receipt: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync model eviction receipt: %w", err)
	}
	return file.Close()
}

// LatestReceipt returns the newest receipt for model, or false when none.
func LatestReceipt(path, model string) (Receipt, bool, error) {
	key := CanonicalName(model)
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return Receipt{}, false, nil
	}
	if err != nil {
		return Receipt{}, false, err
	}
	defer file.Close()
	var latest Receipt
	found := false
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1<<20)
	for scanner.Scan() {
		var r Receipt
		if json.Unmarshal(scanner.Bytes(), &r) != nil {
			continue
		}
		if CanonicalName(r.Model) == key {
			latest, found = r, true
		}
	}
	return latest, found, scanner.Err()
}
