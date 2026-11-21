package entitlementclient

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LeaseRecord contains the original signed token. Verification keys are
// resolved through IssuerKeyStore, a separate trust boundary, and are never
// accepted from this mutable application-data record.
type LeaseRecord struct {
	Lease string `json:"lease"`
}

// LeaseStore is the persistence boundary for signed, untrusted lease records.
// Implementations must not store access or refresh credentials.
type LeaseStore interface {
	Load(identity string) (LeaseRecord, error)
	Save(identity string, record LeaseRecord) error
	Delete(identity string) error
	Clear() error
}

// IssuerKeyStore persists the public issuer key set outside the lease cache.
// Implementations must protect it independently of LeaseStore because both
// values together would let a modified cache file invent its own trust root.
type IssuerKeyStore interface {
	Load(ctx context.Context, authority string) ([]byte, error)
	Save(ctx context.Context, authority string, jwks []byte) error
}

// FileLeaseStore stores one private record per normalized identity. Records
// are written atomically with owner-only permissions so a crash cannot leave
// a partial signed token.
type FileLeaseStore struct {
	dir string
}

// NewFileLeaseStore creates a store under dir. The directory is private to the
// current user; callers should place it in their application data directory.
func NewFileLeaseStore(dir string) (*FileLeaseStore, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, errors.New("lease store directory is required")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create lease store directory: %w", err)
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return nil, fmt.Errorf("inspect lease store directory: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("lease store path must be a directory, not a symlink")
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return nil, fmt.Errorf("restrict lease store directory permissions: %w", err)
	}
	return &FileLeaseStore{dir: dir}, nil
}

func (s *FileLeaseStore) Load(identity string) (LeaseRecord, error) {
	path, err := s.path(identity)
	if err != nil {
		return LeaseRecord{}, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return LeaseRecord{}, ErrLeaseUnavailable
		}
		return LeaseRecord{}, fmt.Errorf("inspect cached lease: %w", err)
	}
	if !info.Mode().IsRegular() {
		return LeaseRecord{}, errors.New("cached lease must be a regular file")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return LeaseRecord{}, fmt.Errorf("read cached lease: %w", err)
	}
	var record LeaseRecord
	if err := json.Unmarshal(body, &record); err != nil {
		return LeaseRecord{}, fmt.Errorf("decode cached lease: %w", err)
	}
	return record, nil
}

func (s *FileLeaseStore) Save(identity string, record LeaseRecord) error {
	path, err := s.path(identity)
	if err != nil {
		return err
	}
	if record.Lease == "" {
		return errors.New("signed lease is required")
	}
	body, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encode cached lease: %w", err)
	}
	tmp, err := os.CreateTemp(s.dir, ".lease-*.tmp")
	if err != nil {
		return fmt.Errorf("create cached lease temporary file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("restrict cached lease permissions: %w", err)
	}
	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write cached lease: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync cached lease: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close cached lease: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("publish cached lease: %w", err)
	}
	return nil
}

func (s *FileLeaseStore) Delete(identity string) error {
	path, err := s.path(identity)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete cached lease: %w", err)
	}
	return nil
}

// Clear removes lease records from this dedicated store directory.
func (s *FileLeaseStore) Clear() error {
	if s == nil || s.dir == "" {
		return errors.New("lease store is not configured")
	}
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return fmt.Errorf("list cached leases: %w", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if filepath.Ext(name) != ".json" && !strings.HasPrefix(name, ".lease-") {
			continue
		}
		if err := os.Remove(filepath.Join(s.dir, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove cached lease %q: %w", name, err)
		}
	}
	return nil
}

func (s *FileLeaseStore) path(identity string) (string, error) {
	identity = strings.ToLower(strings.TrimSpace(identity))
	if identity == "" {
		return "", ErrLeaseIdentityMissing
	}
	if s == nil || s.dir == "" {
		return "", errors.New("lease store is not configured")
	}
	digest := sha256.Sum256([]byte(identity))
	return filepath.Join(s.dir, hex.EncodeToString(digest[:])+".json"), nil
}
