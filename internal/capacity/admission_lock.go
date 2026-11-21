package capacity

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/artifactlock"
)

// LockAdmission serializes strict no-reclaim sensing, decision and recording
// against the same canonical ledger across processes. It must not be held over
// external reclaim callbacks: those can invoke the resource lifecycle again.
// Ordinary advisory admissions retain their explicitly non-blocking policy.
func LockAdmission(ctx context.Context, store ClaimRepository) (func(), error) {
	owner, ok := store.(interface {
		LockAdmission(context.Context) (func(), error)
	})
	if !ok {
		return nil, fmt.Errorf("capacity store does not provide admission serialization")
	}
	return owner.LockAdmission(ctx)
}

func (s *SQLiteStore) LockAdmission(ctx context.Context) (func(), error) {
	if s.dbPath == ":memory:" {
		s.admissionOnce.Do(func() { s.admissionGate = make(chan struct{}, 1) })
		ch := s.admissionGate
		select {
		case ch <- struct{}{}:
			return func() { <-ch }, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return LockAdmissionPath(ctx, s.dbPath)
}

// AdmissionLockPath resolves one ledger identity for CLI, lifecycle and public
// broker callers, including construction before the SQLite file exists.
func AdmissionLockPath(dbPath string) (string, error) {
	if strings.TrimSpace(dbPath) == "" || dbPath == ":memory:" {
		return "", nil
	}
	absolute, err := filepath.Abs(dbPath)
	if err != nil {
		return "", err
	}
	path, suffix := filepath.Clean(absolute), ""
	for {
		canonical, err := filepath.EvalSymlinks(path)
		if err == nil {
			return filepath.Join(canonical, suffix) + ".admission.lock", nil
		}
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("resolve capacity admission identity: %w", err)
		}
		suffix = filepath.Join(filepath.Base(path), suffix)
		parent := filepath.Dir(path)
		if parent == path {
			return "", fmt.Errorf("capacity admission has no existing ancestor: %s", dbPath)
		}
		path = parent
	}
}

// LockAdmissionPath is also used before opening SQLite so schema initialization
// cannot race a live no-reclaim admission on the same ledger.
func LockAdmissionPath(ctx context.Context, dbPath string) (func(), error) {
	path, err := AdmissionLockPath(dbPath)
	if err != nil {
		return nil, err
	}
	if path == "" {
		return func() {}, nil
	}
	return artifactlock.AcquireWithPathContext(ctx, "capacity-admission:"+path, func(string) (string, error) {
		return path, nil
	})
}
