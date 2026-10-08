package agentharness

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var ErrStoreBusy = errors.New("agent policy snapshot store is busy")

const snapshotLockStaleAfter = 10 * time.Minute

type BundleStore struct {
	Dir string
	Now func() time.Time
	mu  sync.Mutex
}

func NewBundleStore(dir string) *BundleStore {
	return &BundleStore{Dir: dir, Now: time.Now}
}

func DefaultDataDir() (string, error) {
	if override := strings.TrimSpace(os.Getenv("VROOLI_AGENT_POLICY_HOME")); override != "" {
		return filepath.Clean(override), nil
	}
	root, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve application config directory: %w", err)
	}
	return filepath.Join(root, "vrooli", "agent-policy"), nil
}

func (s *BundleStore) bundlePath() string { return filepath.Join(s.Dir, "snapshot-bundle.json") }

func (s *BundleStore) Load() (SnapshotBundle, error) {
	if s == nil || strings.TrimSpace(s.Dir) == "" {
		return SnapshotBundle{}, errors.New("snapshot store directory is required")
	}
	data, err := os.ReadFile(s.bundlePath())
	if err != nil {
		return SnapshotBundle{}, err
	}
	var bundle SnapshotBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return SnapshotBundle{}, fmt.Errorf("parse snapshot bundle: %w", err)
	}
	if err := ValidateBundle(bundle); err != nil {
		return SnapshotBundle{}, err
	}
	if bundle.Integrity == "" || bundle.Integrity != BundleIntegrity(bundle) {
		return SnapshotBundle{}, errors.New("snapshot bundle integrity check failed")
	}
	return bundle, nil
}

func (s *BundleStore) Publish(bundle SnapshotBundle) error {
	if s == nil || strings.TrimSpace(s.Dir) == "" {
		return errors.New("snapshot store directory is required")
	}
	if bundle.SchemaVersion == "" {
		bundle.SchemaVersion = ContractVersion
	}
	if bundle.PublishedAt.IsZero() {
		bundle.PublishedAt = s.now().UTC()
	}
	if bundle.Snapshots == nil {
		bundle.Snapshots = map[string]ProviderSnapshot{}
	}
	if err := ValidateBundle(bundle); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return fmt.Errorf("create snapshot directory: %w", err)
	}
	unlock, err := acquireFileLock(filepath.Join(s.Dir, ".snapshot.lock"))
	if err != nil {
		return err
	}
	defer unlock()
	return s.publishLocked(bundle)
}

func (s *BundleStore) publishLocked(bundle SnapshotBundle) error {
	bundle.Integrity = BundleIntegrity(bundle)
	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return fmt.Errorf("encode snapshot bundle: %w", err)
	}
	tmp := filepath.Join(s.Dir, ".snapshot-bundle."+strconv.FormatInt(time.Now().UnixNano(), 10)+".tmp")
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write snapshot bundle: %w", err)
	}
	if err := os.Rename(tmp, s.bundlePath()); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("atomically publish snapshot bundle: %w", err)
	}
	return nil
}

func (s *BundleStore) PublishProvider(snapshot ProviderSnapshot) error {
	if err := ValidateSnapshot(snapshot); err != nil {
		return err
	}
	if s == nil || strings.TrimSpace(s.Dir) == "" {
		return errors.New("snapshot store directory is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return fmt.Errorf("create snapshot directory: %w", err)
	}
	unlock, err := acquireFileLock(filepath.Join(s.Dir, ".snapshot.lock"))
	if err != nil {
		return err
	}
	defer unlock()
	bundle, err := s.Load()
	if errors.Is(err, os.ErrNotExist) {
		bundle = SnapshotBundle{SchemaVersion: ContractVersion, Snapshots: map[string]ProviderSnapshot{}}
	} else if err != nil {
		return err
	}
	bundle.Generation++
	bundle.Snapshots[snapshot.ProviderID] = snapshot
	if bundle.PublishedAt.IsZero() {
		bundle.PublishedAt = s.now().UTC()
	}
	return s.publishLocked(bundle)
}

func (s *BundleStore) WithdrawProvider(providerID string) error {
	if s == nil || strings.TrimSpace(s.Dir) == "" {
		return errors.New("snapshot store directory is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	unlock, err := acquireFileLock(filepath.Join(s.Dir, ".snapshot.lock"))
	if err != nil {
		return err
	}
	defer unlock()
	bundle, err := s.Load()
	if err != nil {
		return err
	}
	if _, ok := bundle.Snapshots[providerID]; !ok {
		return fmt.Errorf("provider %q is not published", providerID)
	}
	delete(bundle.Snapshots, providerID)
	bundle.Generation++
	bundle.PublishedAt = s.now().UTC()
	return s.publishLocked(bundle)
}

func (s *BundleStore) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func BundleIntegrity(bundle SnapshotBundle) string {
	bundle.Integrity = ""
	data, _ := json.Marshal(bundle)
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:])
}

func acquireFileLock(path string) (func(), error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			if stale, staleErr := staleLock(path); staleErr == nil && stale {
				if removeErr := os.Remove(path); removeErr == nil {
					return acquireFileLock(path)
				}
			}
			return nil, ErrStoreBusy
		}
		return nil, fmt.Errorf("acquire snapshot lock: %w", err)
	}
	if _, err := file.WriteString(strconv.FormatInt(time.Now().UnixNano(), 10)); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return nil, fmt.Errorf("write snapshot lock: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return nil, fmt.Errorf("close snapshot lock: %w", err)
	}
	return func() { _ = os.Remove(path) }, nil
}

func staleLock(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	nanos, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return false, err
	}
	return time.Since(time.Unix(0, nanos)) > snapshotLockStaleAfter, nil
}
