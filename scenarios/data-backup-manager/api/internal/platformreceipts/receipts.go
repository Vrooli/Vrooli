// Package platformreceipts records privacy-safe native execution evidence for
// the DBM cross-platform support matrix.
package platformreceipts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Receipt struct {
	ID           string `json:"id"`
	Platform     string `json:"platform"`
	Architecture string `json:"architecture"`
	Host         string `json:"host"`
	ObservedAt   string `json:"observed_at"`
	Capability   string `json:"capability"`
	Outcome      string `json:"outcome"`
	EvidenceKind string `json:"evidence_kind"`
	Command      string `json:"command"`
	Evidence     string `json:"evidence"`
}

type Input struct {
	Capability    string
	Outcome       string
	Operation     string
	TargetID      string
	DestinationID string
	Proof         string
	ObservedAt    time.Time
}

type Sink interface {
	Record(context.Context, Input) error
}

type FileSink struct {
	Path string
	mu   sync.Mutex
}

func NewFileSink() *FileSink {
	if path := strings.TrimSpace(os.Getenv("DBM_PLATFORM_RECEIPTS_PATH")); path != "" {
		return &FileSink{Path: path}
	}
	root := strings.TrimSpace(os.Getenv("VROOLI_HOME"))
	if root == "" {
		if home, err := os.UserHomeDir(); err == nil {
			root = filepath.Join(home, ".vrooli")
		}
	}
	return &FileSink{Path: filepath.Join(root, "state", "data-backup-manager", "platform-receipts.jsonl")}
}

func (s *FileSink) Record(ctx context.Context, in Input) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || strings.TrimSpace(s.Path) == "" {
		return errors.New("DBM platform receipt path is required")
	}
	if in.Capability != "backup" && in.Capability != "scratch_restore" {
		return fmt.Errorf("unsupported DBM receipt capability %q", in.Capability)
	}
	if in.Outcome != "passed" && in.Outcome != "failed" {
		return fmt.Errorf("unsupported DBM receipt outcome %q", in.Outcome)
	}
	if in.Operation == "" || in.TargetID == "" || in.DestinationID == "" {
		return errors.New("operation, target and destination references are required")
	}
	if in.ObservedAt.IsZero() {
		in.ObservedAt = time.Now().UTC()
	}
	host, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("read execution host identity: %w", err)
	}
	proofDigest := digest(in.Proof)
	referenceDigest := digest(strings.Join([]string{in.Operation, in.TargetID, in.DestinationID}, "\x00"))
	command := "DBM native backup execution"
	if in.Capability == "scratch_restore" {
		command = "DBM native scratch restore and verification"
	}
	r := Receipt{
		ID: "dbm-" + uuid.NewString(), Platform: receiptPlatform(runtime.GOOS),
		Architecture: runtime.GOARCH, Host: "host-" + digest(strings.TrimSpace(host))[:16],
		ObservedAt: in.ObservedAt.UTC().Format(time.RFC3339Nano), Capability: in.Capability,
		Outcome: in.Outcome, EvidenceKind: "native-runtime", Command: command,
		Evidence: "operation_sha256=" + referenceDigest + "; proof_sha256=" + proofDigest,
	}
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return err
	}
	if err := os.Chmod(filepath.Dir(s.Path), 0o700); err != nil {
		return fmt.Errorf("restrict DBM receipt directory permissions: %w", err)
	}
	f, err := os.OpenFile(s.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := f.Chmod(0o600); err != nil {
		return fmt.Errorf("restrict DBM receipt file permissions: %w", err)
	}
	if _, err := f.Write(append(data, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

func receiptPlatform(goos string) string {
	if goos == "darwin" {
		return "macos"
	}
	return goos
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
