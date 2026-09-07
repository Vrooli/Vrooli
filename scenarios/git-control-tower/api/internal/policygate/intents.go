package policygate

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/cli-core/cliutil"
)

const DefaultIntentTTL = 2 * time.Minute

var (
	ErrIntentRequired         = errors.New("mutation intent is required")
	ErrIntentExpired          = errors.New("mutation intent expired")
	ErrIntentReplay           = errors.New("mutation intent has already been consumed")
	ErrIntentMismatch         = errors.New("mutation intent does not match the requested operation")
	ErrStepUpRequired         = errors.New("step-up authentication is required for this operation")
	ErrMutationAuthentication = errors.New("verified authentication is required")
	ErrMutationPermission     = errors.New("principal is not permitted to mutate this repository")
)

// Intent is the safe receipt returned to a browser or CLI. The opaque ID is
// never persisted in plaintext; only its SHA-256 hash is stored.
type Intent struct {
	ID               string
	PrincipalID      string
	RepositoryID     string
	Operation        string
	ExpectedRevision string
	SubjectDigest    string
	PolicyVersion    string
	IssuedAt         time.Time
	ExpiresAt        time.Time
	ConsumedAt       *time.Time
	StepUpRequired   bool
}

type IntentRequest struct {
	RepositoryID     string
	Operation        string
	ExpectedRevision string
	SubjectDigest    string
	PolicyVersion    string
	TTL              time.Duration
	StepUpRequired   bool
}

type IntentStore interface {
	Save(context.Context, Intent) error
	Consume(context.Context, string, string, string, string, string, string, time.Time) (Intent, error)
	Find(context.Context, string) (Intent, error)
}

// SQLIntentStore is restart-safe and uses a conditional UPDATE so only one
// concurrent request can consume an intent.
type SQLIntentStore struct {
	DB interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
		QueryRowContext(context.Context, string, ...any) *sql.Row
	}
}

func NewSQLIntentStore(db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}) *SQLIntentStore {
	return &SQLIntentStore{DB: db}
}

func (s *SQLIntentStore) Save(ctx context.Context, intent Intent) error {
	if s == nil || s.DB == nil {
		return errors.New("intent store is not configured")
	}
	_, err := s.DB.ExecContext(ctx, `INSERT INTO git_mutation_intents
		(intent_hash, principal_id, repository_id, operation, expected_revision, subject_digest, policy_version, issued_at, expires_at, step_up_required)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, hashIntentID(intent.ID), intent.PrincipalID, intent.RepositoryID, intent.Operation, intent.ExpectedRevision, intent.SubjectDigest, intent.PolicyVersion, intent.IssuedAt.UTC().Format(time.RFC3339Nano), intent.ExpiresAt.UTC().Format(time.RFC3339Nano), boolInt(intent.StepUpRequired))
	return err
}

func (s *SQLIntentStore) Find(ctx context.Context, rawID string) (Intent, error) {
	if s == nil || s.DB == nil {
		return Intent{}, errors.New("intent store is not configured")
	}
	return s.scan(ctx, `SELECT principal_id, repository_id, operation, expected_revision, subject_digest, policy_version, issued_at, expires_at, consumed_at, step_up_required FROM git_mutation_intents WHERE intent_hash = ?`, hashIntentID(rawID))
}

func (s *SQLIntentStore) Consume(ctx context.Context, rawID, principalID, repositoryID, operation, expectedRevision, subjectDigest string, now time.Time) (Intent, error) {
	if s == nil || s.DB == nil {
		return Intent{}, errors.New("intent store is not configured")
	}
	result, err := s.DB.ExecContext(ctx, `UPDATE git_mutation_intents SET consumed_at = ?
		WHERE intent_hash = ? AND principal_id = ? AND repository_id = ? AND operation = ? AND expected_revision = ?
		AND subject_digest = ? AND consumed_at IS NULL AND expires_at > ?`, now.UTC().Format(time.RFC3339Nano), hashIntentID(rawID), principalID, repositoryID, operation, expectedRevision, subjectDigest, now.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return Intent{}, err
	}
	rows, err := result.RowsAffected()
	if err != nil || rows != 1 {
		intent, findErr := s.Find(ctx, rawID)
		if findErr != nil {
			return Intent{}, ErrIntentRequired
		}
		if intent.ConsumedAt != nil {
			return Intent{}, ErrIntentReplay
		}
		if !intent.ExpiresAt.After(now) {
			return Intent{}, ErrIntentExpired
		}
		return Intent{}, ErrIntentMismatch
	}
	intent, err := s.Find(ctx, rawID)
	if err != nil {
		return Intent{}, err
	}
	// Find reads only the hashed identifier because the raw opaque ID is never
	// persisted. Restore the caller-held ID on the receipt returned from a
	// successful consume so it can be attached to the request context and pass
	// the domain writer's consumed-intent guard.
	intent.ID = rawID
	return intent, nil
}

func (s *SQLIntentStore) scan(ctx context.Context, query string, args ...any) (Intent, error) {
	var intent Intent
	var issued, expires, consumed sql.NullString
	var step int
	err := s.DB.QueryRowContext(ctx, query, args...).Scan(&intent.PrincipalID, &intent.RepositoryID, &intent.Operation, &intent.ExpectedRevision, &intent.SubjectDigest, &intent.PolicyVersion, &issued, &expires, &consumed, &step)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Intent{}, ErrIntentRequired
		}
		return Intent{}, err
	}
	intent.IssuedAt, _ = time.Parse(time.RFC3339Nano, issued.String)
	intent.ExpiresAt, _ = time.Parse(time.RFC3339Nano, expires.String)
	intent.StepUpRequired = step != 0
	if consumed.Valid {
		if parsed, parseErr := time.Parse(time.RFC3339Nano, consumed.String); parseErr == nil {
			intent.ConsumedAt = &parsed
		}
	}
	return intent, nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

type MemoryIntentStore struct {
	mu      sync.Mutex
	intents map[string]Intent
}

func NewMemoryIntentStore() *MemoryIntentStore {
	return &MemoryIntentStore{intents: make(map[string]Intent)}
}
func (s *MemoryIntentStore) Save(_ context.Context, intent Intent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.intents[hashIntentID(intent.ID)] = intent
	return nil
}
func (s *MemoryIntentStore) Find(_ context.Context, rawID string) (Intent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	intent, ok := s.intents[hashIntentID(rawID)]
	if !ok {
		return Intent{}, ErrIntentRequired
	}
	return intent, nil
}
func (s *MemoryIntentStore) Consume(_ context.Context, rawID, principalID, repositoryID, operation, expectedRevision, subjectDigest string, now time.Time) (Intent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	intent, ok := s.intents[hashIntentID(rawID)]
	if !ok {
		return Intent{}, ErrIntentRequired
	}
	if intent.ConsumedAt != nil {
		return Intent{}, ErrIntentReplay
	}
	if !intent.ExpiresAt.After(now) {
		return Intent{}, ErrIntentExpired
	}
	if intent.PrincipalID != principalID || intent.RepositoryID != repositoryID || intent.Operation != operation || intent.ExpectedRevision != expectedRevision || intent.SubjectDigest != subjectDigest {
		return Intent{}, ErrIntentMismatch
	}
	consumed := now.UTC()
	intent.ConsumedAt = &consumed
	s.intents[hashIntentID(rawID)] = intent
	return intent, nil
}

type IntentService struct {
	store IntentStore
	now   func() time.Time
	ttl   time.Duration
}

func NewIntentService(store IntentStore) *IntentService {
	return &IntentService{store: store, now: time.Now, ttl: DefaultIntentTTL}
}
func (s *IntentService) WithClock(now func() time.Time) *IntentService {
	if now != nil {
		s.now = now
	}
	return s
}
func (s *IntentService) WithTTL(ttl time.Duration) *IntentService {
	if ttl > 0 {
		s.ttl = ttl
	}
	return s
}

func (s *IntentService) Issue(ctx context.Context, principal Principal, req IntentRequest) (Intent, error) {
	if !principal.Verified || strings.TrimSpace(principal.Subject) == "" {
		return Intent{}, ErrMutationAuthentication
	}
	if principal.Kind != cliutil.CallerKindHuman {
		return Intent{}, ErrMutationPermission
	}
	if strings.TrimSpace(req.RepositoryID) == "" || strings.TrimSpace(req.Operation) == "" || strings.TrimSpace(req.ExpectedRevision) == "" || strings.TrimSpace(req.SubjectDigest) == "" {
		return Intent{}, ErrIntentMismatch
	}
	ttl := req.TTL
	if ttl <= 0 {
		ttl = s.ttl
	}
	now := s.now().UTC()
	rawID := "gct-intent-" + uuid.NewString()
	intent := Intent{ID: rawID, PrincipalID: principal.Subject, RepositoryID: req.RepositoryID, Operation: req.Operation, ExpectedRevision: req.ExpectedRevision, SubjectDigest: req.SubjectDigest, PolicyVersion: req.PolicyVersion, IssuedAt: now, ExpiresAt: now.Add(ttl), StepUpRequired: req.StepUpRequired}
	if err := s.store.Save(ctx, intent); err != nil {
		return Intent{}, err
	}
	return intent, nil
}

func (s *IntentService) Consume(ctx context.Context, principal Principal, rawID, repositoryID, operation, expectedRevision, subjectDigest string) (Intent, error) {
	if !principal.Verified || strings.TrimSpace(principal.Subject) == "" {
		return Intent{}, ErrMutationAuthentication
	}
	intent, err := s.store.Consume(ctx, rawID, principal.Subject, repositoryID, operation, expectedRevision, subjectDigest, s.now().UTC())
	if err != nil {
		return Intent{}, err
	}
	return intent, nil
}

func hashIntentID(raw string) string {
	digest := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(digest[:])
}

// NewOpaqueIntentID exists for tests and callers that need to supply a stable
// ID without exposing a credential. Production Issue uses UUIDs.
func NewOpaqueIntentID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "gct-intent-fallback"
	}
	return "gct-intent-" + hex.EncodeToString(raw[:])
}

func (i Intent) AsHumanIntent() HumanIntent {
	return HumanIntent{ID: i.ID, PrincipalID: i.PrincipalID, RepositoryID: i.RepositoryID, Operation: i.Operation, ExpectedRevision: i.ExpectedRevision, SubjectDigest: i.SubjectDigest, PolicyVersion: i.PolicyVersion, ExpiresAt: i.ExpiresAt, ConsumedAt: i.ConsumedAt, Consumed: i.ConsumedAt != nil}
}

func (i Intent) String() string {
	return fmt.Sprintf("%s:%s:%s", i.RepositoryID, i.Operation, i.ExpectedRevision)
}
