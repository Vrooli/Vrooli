package operatorsession

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	operatorSessionDirEnv = "VROOLI_OPERATOR_SESSION_DIR"
	privateKeyFilename    = "private.key"
	enrollmentFilename    = "enrollment.json"
)

// EnrollmentStore is the local half of the operator-session contract. It
// stores only the private signing key and non-secret enrollment metadata; it
// never stores a minted session or an authenticator token.
type EnrollmentStore interface {
	Load() (ed25519.PrivateKey, Enrollment, error)
	Save(ed25519.PrivateKey, Enrollment) error
}

// FileStore is the cross-platform filesystem implementation used by local
// owner clients. The directory is intentionally narrow and can be overridden
// for tests or a platform secure-store adapter without changing the contract.
type FileStore struct{ Dir string }

type enrollmentFile struct {
	Enrollment Enrollment `json:"enrollment"`
}

// NewFileStore constructs a store for dir. An empty directory selects the
// per-user platform config directory, never the repository or operator-state.
func NewFileStore(dir string) (*FileStore, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		config, err := os.UserConfigDir()
		if err != nil || strings.TrimSpace(config) == "" {
			return nil, fmt.Errorf("resolve operator session config directory: %w", err)
		}
		dir = filepath.Join(config, "vrooli", "operator-session")
	}
	return &FileStore{Dir: dir}, nil
}

// DefaultFileStore returns the standard per-user local enrollment store.
func DefaultFileStore() (*FileStore, error) {
	return NewFileStore(os.Getenv(operatorSessionDirEnv))
}

func (s *FileStore) paths() (string, string, error) {
	if s == nil || strings.TrimSpace(s.Dir) == "" {
		return "", "", errors.New("operator session store directory is required")
	}
	return filepath.Join(s.Dir, privateKeyFilename), filepath.Join(s.Dir, enrollmentFilename), nil
}

// Load reads a complete enrollment atomically enough for the two artifacts to
// be independently validated. A caller receives a defensive key copy.
func (s *FileStore) Load() (ed25519.PrivateKey, Enrollment, error) {
	keyPath, enrollmentPath, err := s.paths()
	if err != nil {
		return nil, Enrollment{}, err
	}
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, Enrollment{}, err
	}
	if len(keyBytes) != ed25519.PrivateKeySize {
		return nil, Enrollment{}, errors.New("operator session private key has an invalid size")
	}
	data, err := os.ReadFile(enrollmentPath)
	if err != nil {
		return nil, Enrollment{}, err
	}
	var file enrollmentFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, Enrollment{}, fmt.Errorf("decode operator enrollment: %w", err)
	}
	if err := file.Enrollment.Validate(); err != nil {
		return nil, Enrollment{}, err
	}
	return ed25519.PrivateKey(append([]byte(nil), keyBytes...)), file.Enrollment, nil
}

// Save writes the key and metadata with owner-only permissions and same-dir
// replacement. No caller-controlled path is removed and no bearer material is
// accepted by this API.
func (s *FileStore) Save(private ed25519.PrivateKey, enrollment Enrollment) error {
	if len(private) != ed25519.PrivateKeySize {
		return errors.New("operator session private key has an invalid size")
	}
	if err := enrollment.Validate(); err != nil {
		return err
	}
	if s == nil || strings.TrimSpace(s.Dir) == "" {
		return errors.New("operator session store directory is required")
	}
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return fmt.Errorf("create operator session directory: %w", err)
	}
	keyPath, enrollmentPath, err := s.paths()
	if err != nil {
		return err
	}
	if err := writePrivateFile(keyPath, private); err != nil {
		return fmt.Errorf("write operator session private key: %w", err)
	}
	data, err := json.Marshal(enrollmentFile{Enrollment: enrollment})
	if err != nil {
		return err
	}
	if err := writePrivateFile(enrollmentPath, append(data, '\n')); err != nil {
		return fmt.Errorf("write operator enrollment: %w", err)
	}
	return nil
}

// LocalResolution is the result of resolving an already-enrolled operator.
// Its token is intentionally ephemeral and is never part of Enrollment.
type LocalResolution struct {
	Token      string
	Enrollment Enrollment
	ExpiresAt  time.Time
}

type EnrollmentState string

const (
	EnrollmentStateEnrolled   EnrollmentState = "enrolled"
	EnrollmentStateUnenrolled EnrollmentState = "unenrolled"
)

// Status is the shared diagnostic view used by CLI and UI adapters. It never
// includes a bearer credential.
type Status struct {
	State      EnrollmentState `json:"state"`
	Enrollment Enrollment      `json:"enrollment,omitempty"`
	Reason     string          `json:"reason,omitempty"`
}

// LocalResolver implements the local portion of the shared contract. It has
// no network or authenticator dependency: enrollment is performed by a
// transport-specific client, then this resolver mints sessions from the
// locally held key.
type LocalResolver struct {
	Store EnrollmentStore
	Now   func() time.Time
	TTL   time.Duration
}

// Resolve mints a short-lived session locally from the enrolled key.
func (r LocalResolver) Resolve() (LocalResolution, error) {
	if r.Store == nil {
		return LocalResolution{}, errors.New("operator session enrollment is unavailable")
	}
	private, enrollment, err := r.Store.Load()
	if err != nil {
		return LocalResolution{}, err
	}
	defer clearBytes(private)
	now := time.Now
	if r.Now != nil {
		now = r.Now
	}
	current := now()
	ttl := r.TTL
	if ttl <= 0 {
		ttl = LocalSessionTTL
	}
	token, err := Mint(private, enrollment.Reference, enrollment.OperatorID, enrollment.ScopeCeiling, current, ttl)
	if err != nil {
		return LocalResolution{}, err
	}
	return LocalResolution{Token: token, Enrollment: enrollment, ExpiresAt: current.Add(ttl)}, nil
}

// Status reports whether the local machine has an enrollment. It is safe to
// call while the identity provider is stopped because it reads only local
// enrollment state.
func (r LocalResolver) Status() Status {
	if r.Store == nil {
		return Status{State: EnrollmentStateUnenrolled, Reason: "operator session enrollment is unavailable"}
	}
	_, enrollment, err := r.Store.Load()
	if err != nil {
		return Status{State: EnrollmentStateUnenrolled, Reason: err.Error()}
	}
	return Status{State: EnrollmentStateEnrolled, Enrollment: enrollment}
}

func writePrivateFile(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".operator-session-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if err := tmp.Chmod(0o600); err != nil {
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
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

func clearBytes(value []byte) {
	for i := range value {
		value[i] = 0
	}
	runtime.KeepAlive(value)
}
