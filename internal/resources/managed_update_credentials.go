package resources

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/credentialauthority"
)

// ManagedUpdateCredentialSnapshot binds private, operation-lifetime runtime
// files. It is not a new credential authority or an authority-version receipt.
// Only logical references and a keyed integrity digest enter the ledger.
type ManagedUpdateCredentialSnapshot struct {
	SchemaVersion  int                           `json:"schema_version"`
	Resource       string                        `json:"resource"`
	OperationID    string                        `json:"operation_id"`
	SourceRevision string                        `json:"source_revision"`
	TargetDigest   string                        `json:"target_digest"`
	RollbackDigest string                        `json:"rollback_digest"`
	Items          []ManagedUpdateCredentialItem `json:"items"`
	Digest         string                        `json:"digest"`
}

type ManagedUpdateCredentialItem struct {
	LogicalID string `json:"logical_id"`
	Field     string `json:"field"`
	FileKey   string `json:"file_key"`
}

// managedUpdateCredentialDefaultField is the credential field a declaration
// without one names; snapshot items record it and both images match on it.
const managedUpdateCredentialDefaultField = "value"

func managedUpdateCredentialItems(u ManagedServiceUpdate) ([]ManagedUpdateCredentialItem, error) {
	byKey := map[string]ManagedUpdateCredentialItem{}
	for _, c := range []ManagedUpdateContract{u.Target, u.Rollback} {
		if c.Manifest.ManagedService == nil {
			return nil, fmt.Errorf("credential snapshot requires frozen launch contracts")
		}
		if err := c.Manifest.ManagedService.ValidateCredentialFiles(); err != nil {
			return nil, err
		}
		for _, d := range c.Manifest.ManagedService.CredentialFiles {
			if _, err := credentialauthority.ParseIdentity(d.LogicalID); err != nil {
				return nil, err
			}
			field := strings.TrimSpace(d.Field)
			if field == "" {
				field = managedUpdateCredentialDefaultField
			}
			hash := sha256.Sum256([]byte(d.LogicalID + "\x00" + field))
			key := hex.EncodeToString(hash[:])
			byKey[key] = ManagedUpdateCredentialItem{LogicalID: d.LogicalID, Field: field, FileKey: key}
		}
	}
	if len(byKey) > 64 {
		return nil, fmt.Errorf("credential snapshot exceeds its 64-field bound")
	}
	items := make([]ManagedUpdateCredentialItem, 0, len(byKey))
	for _, v := range byKey {
		items = append(items, v)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].FileKey < items[j].FileKey })
	return items, nil
}

func (s ManagedUpdateCredentialSnapshot) validate(u ManagedServiceUpdate) error {
	items, err := managedUpdateCredentialItems(u)
	if err != nil {
		return err
	}
	if s.SchemaVersion != managedUpdateCredentialSchemaVersion || s.Resource != u.Resource || s.OperationID != u.ID || s.SourceRevision != u.SourceRevision || s.TargetDigest != u.Target.Digest || s.RollbackDigest != u.Rollback.Digest || !updateDigest(s.Digest) || len(items) == 0 || len(s.Items) != len(items) {
		return fmt.Errorf("credential snapshot differs from its operation")
	}
	for i, item := range items {
		if s.Items[i] != item {
			return fmt.Errorf("credential snapshot logical references changed")
		}
	}
	return nil
}

type managedUpdateCredentialStore struct {
	dir     string
	write   func(string, []byte, os.FileMode) error
	resolve func(context.Context, ManagedUpdateCredentialItem) (string, error)
}

func newManagedUpdateCredentialStore(stateDir, operation string) (*managedUpdateCredentialStore, error) {
	if !filepath.IsAbs(stateDir) || filepath.Clean(stateDir) != stateDir || !managedUpdateKey.MatchString(operation) {
		return nil, fmt.Errorf("invalid credential snapshot owner path")
	}
	return &managedUpdateCredentialStore{
		dir: filepath.Join(stateDir, "managed-update-exec", operation, "credentials"), write: config.WriteOwnedFileDurable,
		resolve: func(ctx context.Context, item ManagedUpdateCredentialItem) (string, error) {
			if err := ctx.Err(); err != nil {
				return "", err
			}
			authority, err := credentialauthority.DefaultAuthority()
			if err != nil {
				return "", fmt.Errorf("credential snapshot authority unavailable")
			}
			identity, err := credentialauthority.ParseIdentity(item.LogicalID)
			if err != nil {
				return "", err
			}
			value, err := authority.Resolve(identity, item.Field)
			if err != nil {
				return "", fmt.Errorf("credential snapshot logical field unavailable")
			}
			return value, nil
		},
	}, nil
}

func readManagedUpdatePrivateFile(path string, limit int) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 || info.Size() > int64(limit) {
		return nil, fmt.Errorf("private update file has unsafe type, permissions or size")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	actual, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !os.SameFile(info, actual) {
		return nil, fmt.Errorf("private update file identity changed")
	}
	data, err := io.ReadAll(io.LimitReader(file, int64(limit)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > limit {
		return nil, fmt.Errorf("private update file exceeds its bound")
	}
	return data, nil
}

func (s *managedUpdateCredentialStore) read(u ManagedServiceUpdate) (ManagedUpdateCredentialSnapshot, error) {
	data, err := readManagedUpdatePrivateFile(filepath.Join(s.dir, "snapshot.json"), 64<<10)
	if err != nil {
		return ManagedUpdateCredentialSnapshot{}, err
	}
	if err := validateManagedUpdateJSON(data); err != nil {
		return ManagedUpdateCredentialSnapshot{}, err
	}
	var snapshot ManagedUpdateCredentialSnapshot
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&snapshot); err != nil {
		return snapshot, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return snapshot, fmt.Errorf("credential snapshot has trailing data")
	}
	if err := snapshot.validate(u); err != nil {
		return snapshot, err
	}
	if err := s.verify(snapshot); err != nil {
		return snapshot, err
	}
	return snapshot, nil
}

func (s *managedUpdateCredentialStore) digest(snapshot ManagedUpdateCredentialSnapshot) (string, error) {
	key, err := readManagedUpdatePrivateFile(filepath.Join(s.dir, "key"), 32)
	if err != nil {
		return "", err
	}
	return credentialSnapshotDigest(key, snapshot, func(fileKey string) ([]byte, error) {
		return readManagedUpdatePrivateFile(filepath.Join(s.dir, fileKey+".value"), 1<<20)
	})
}

// credentialSnapshotDigest selects, orders and reads the snapshot's values
// and computes its stored integrity digest. readValue is the only effect, so
// the persistent-state sample runs this exact derivation in memory.
func credentialSnapshotDigest(key []byte, snapshot ManagedUpdateCredentialSnapshot, readValue func(string) ([]byte, error)) (string, error) {
	if len(key) != 32 {
		return "", fmt.Errorf("credential snapshot key is incomplete")
	}
	values := make([][]byte, 0, len(snapshot.Items))
	for _, item := range snapshot.Items {
		if !updateDigest(item.FileKey) {
			return "", fmt.Errorf("invalid credential snapshot file key")
		}
		value, err := readValue(item.FileKey)
		if err != nil {
			return "", err
		}
		if len(value) == 0 {
			return "", fmt.Errorf("credential snapshot field is empty")
		}
		values = append(values, value)
	}
	return credentialSnapshotMAC(key, snapshot, values)
}

// credentialSnapshotMAC is the stored integrity digest: HMAC-SHA256 over the
// snapshot JSON without its digest, then each value framed by its big-endian
// length. A release that reads snapshots recomputes it, so its framing is a
// persistent format (sampled by PersistentState).
func credentialSnapshotMAC(key []byte, snapshot ManagedUpdateCredentialSnapshot, values [][]byte) (string, error) {
	snapshot.Digest = ""
	metadata, err := json.Marshal(snapshot)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(metadata)
	for _, value := range values {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(value)))
		_, _ = mac.Write(size[:])
		_, _ = mac.Write(value)
	}
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func (s *managedUpdateCredentialStore) verify(snapshot ManagedUpdateCredentialSnapshot) error {
	digest, err := s.digest(snapshot)
	if err != nil {
		return err
	}
	if !hmac.Equal([]byte(digest), []byte(snapshot.Digest)) {
		return fmt.Errorf("credential snapshot integrity mismatch")
	}
	return nil
}

// Prepare is called under the resource lifecycle lock before admission. A
// committed snapshot is immutable: retries verify/recommit its metadata, never
// refetch or replace its values. Partial pre-admission files may be rebuilt;
// their presence alone never gives a wrapper execution permission.
func (s *managedUpdateCredentialStore) Prepare(ctx context.Context, u ManagedServiceUpdate) (*ManagedUpdateCredentialSnapshot, error) {
	items, err := managedUpdateCredentialItems(u)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	_, metadataErr := os.Lstat(filepath.Join(s.dir, "snapshot.json"))
	if metadataErr == nil {
		snapshot, err := s.read(u)
		if err != nil {
			return nil, err
		}
		data, _ := json.Marshal(snapshot)
		if err := s.write(filepath.Join(s.dir, "snapshot.json"), data, 0o600); err != nil {
			return nil, fmt.Errorf("%w: credential snapshot acknowledgement: %w", ErrManagedUpdateCommitUnknown, err)
		}
		return &snapshot, nil
	} else if !os.IsNotExist(metadataErr) {
		return nil, metadataErr
	}
	owner, err := newManagedUpdateStore(u.Resource)
	if err != nil {
		return nil, err
	}
	ledger, err := owner.read()
	if err != nil {
		return nil, err
	}
	for _, entry := range ledger.Records {
		if entry.State.ID == u.ID || entry.State.RequestID == u.RequestID {
			return nil, fmt.Errorf("refuse to replace missing credentials for an admitted update")
		}
	}
	stateRoot := filepath.Dir(filepath.Dir(filepath.Dir(s.dir)))
	if err := ensureManagedUpdatePrivateDirectory(stateRoot, s.dir); err != nil {
		return nil, err
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	if err := s.write(filepath.Join(s.dir, "key"), key, 0o600); err != nil {
		return nil, err
	}
	for _, item := range items {
		value, err := s.resolve(ctx, item)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(value) == "" || len(value) > 1<<20 {
			return nil, fmt.Errorf("credential snapshot field is empty or exceeds its bound")
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := s.write(filepath.Join(s.dir, item.FileKey+".value"), []byte(value), 0o600); err != nil {
			return nil, err
		}
	}
	snapshot := ManagedUpdateCredentialSnapshot{SchemaVersion: managedUpdateCredentialSchemaVersion, Resource: u.Resource, OperationID: u.ID, SourceRevision: u.SourceRevision, TargetDigest: u.Target.Digest, RollbackDigest: u.Rollback.Digest, Items: items}
	snapshot.Digest, err = s.digest(snapshot)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(snapshot)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := s.write(filepath.Join(s.dir, "snapshot.json"), data, 0o600); err != nil {
		return nil, fmt.Errorf("%w: credential snapshot acknowledgement: %w", ErrManagedUpdateCommitUnknown, err)
	}
	return &snapshot, nil
}

func materializeManagedUpdateCredentials(u ManagedServiceUpdate, c ManagedUpdateContract, invocation managedUpdateInvocation) error {
	if len(c.Manifest.ManagedService.CredentialFiles) == 0 {
		return nil
	}
	if u.Credentials == nil {
		return fmt.Errorf("resource exec requires operation-bound credentials")
	}
	if err := u.Credentials.validate(u); err != nil {
		return err
	}
	store, err := newManagedUpdateCredentialStore(invocation.StateRoot, u.ID)
	if err != nil {
		return err
	}
	snapshot, err := store.read(u)
	if err != nil {
		return err
	}
	if snapshot.Digest != u.Credentials.Digest {
		return fmt.Errorf("resource exec credential snapshot differs from admission")
	}
	values := managedServiceEnvValues(invocation.Env)
	for _, declaration := range c.Manifest.ManagedService.CredentialFiles {
		field := strings.TrimSpace(declaration.Field)
		if field == "" {
			field = managedUpdateCredentialDefaultField
		}
		hash := sha256.Sum256([]byte(declaration.LogicalID + "\x00" + field))
		value, err := readManagedUpdatePrivateFile(filepath.Join(store.dir, hex.EncodeToString(hash[:])+".value"), 1<<20)
		if err != nil {
			return err
		}
		path := filepath.Join(values["RESOURCE_STATE_DIR"], filepath.FromSlash(declaration.Path))
		if err := config.WriteOwnedFileDurable(path, append(value, '\n'), 0o600); err != nil {
			return err
		}
	}
	return store.verify(snapshot)
}
