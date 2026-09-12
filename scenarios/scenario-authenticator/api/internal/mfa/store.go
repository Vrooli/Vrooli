package mfa

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/schedule"
)

const (
	enrollmentTTL = 10 * time.Minute
	challengeTTL  = 5 * time.Minute
	recoveryCount = 10
)

var (
	ErrEnrollmentExpired = errors.New("MFA enrollment has expired")
	ErrChallengeInvalid  = errors.New("MFA challenge is invalid or expired")
	ErrCodeInvalid       = errors.New("MFA code is invalid")
)

// SQLDB is the transaction-capable surface used by the durable MFA store.
// api-core's RoutedDB and database/sql.DB both satisfy it.
type SQLDB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

// SeedCustodian is the production custody seam for TOTP seeds. The database
// stores only the slot returned by this seam; the authority implementation
// decides where the encrypted value lives.
type SeedCustodian interface {
	Put(userID, slot, secret string) error
	Resolve(userID, slot string) (string, error)
	Delete(userID, slot string) error
}

// Store owns encrypted TOTP seed custody, enrollment state, login challenges,
// and single-use recovery-code consumption. It deliberately exposes no
// plaintext seed or stored recovery hash to callers.
type Store struct {
	db        SQLDB
	key       []byte
	custodian SeedCustodian
	clock     schedule.Clock
}

// NewStore requires a stable 32-byte key derived from the authenticator's
// persisted authority key. Refusing an absent key prevents plaintext MFA seed
// storage or a silently changing encryption key.
func NewStore(db SQLDB, key []byte, clock schedule.Clock) (*Store, error) {
	if db == nil {
		return nil, errors.New("MFA store requires a database")
	}
	if len(key) != 32 {
		return nil, errors.New("MFA store requires a 32-byte encryption key")
	}
	if clock == nil {
		clock = schedule.System()
	}
	return &Store{db: db, key: append([]byte(nil), key...), clock: clock}, nil
}

// NewStoreWithCustodian constructs the production form. A nil custodian is
// rejected so a deployment cannot silently fall back to database seed
// storage after selecting the authority-backed path.
func NewStoreWithCustodian(db SQLDB, custodian SeedCustodian, clock schedule.Clock) (*Store, error) {
	if db == nil {
		return nil, errors.New("MFA store requires a database")
	}
	if custodian == nil {
		return nil, errors.New("MFA store requires a seed custodian")
	}
	if clock == nil {
		clock = schedule.System()
	}
	return &Store{db: db, custodian: custodian, clock: clock}, nil
}

// Schema is owned by this package so the storage shape stays beside its
// interpretation.
func Schema() string {
	return `
CREATE TABLE IF NOT EXISTS mfa_pending_enrollments (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  realm_id TEXT NOT NULL,
  secret_ciphertext TEXT NOT NULL DEFAULT '',
  secret_ref TEXT NOT NULL DEFAULT '',
  expires_at TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_mfa_pending_user ON mfa_pending_enrollments(user_id, expires_at);
CREATE TABLE IF NOT EXISTS mfa_enrollments (
  user_id TEXT PRIMARY KEY,
  realm_id TEXT NOT NULL,
  secret_ciphertext TEXT NOT NULL DEFAULT '',
  secret_ref TEXT NOT NULL DEFAULT '',
  enrolled_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS mfa_recovery_codes (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  code_hash TEXT NOT NULL,
  used_at TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_mfa_recovery_user ON mfa_recovery_codes(user_id, used_at);
CREATE TABLE IF NOT EXISTS mfa_login_challenges (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  challenge_digest TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  used_at TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_mfa_challenge_digest ON mfa_login_challenges(challenge_digest);
`
}

type Enrollment struct {
	ID              string
	ProvisioningURI string
	ExpiresAt       time.Time
}

// BeginEnrollment creates encrypted pending seed material. The provisioning
// URI is the only secret-bearing value returned and is valid only for this
// short-lived confirmation window.
func (s *Store) BeginEnrollment(ctx context.Context, userID, realmID, account string) (Enrollment, error) {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(realmID) == "" {
		return Enrollment{}, errors.New("user and realm are required")
	}
	secret, err := GenerateSecret()
	if err != nil {
		return Enrollment{}, err
	}
	uri, err := ProvisioningURI("Vrooli Authenticator", account, secret)
	if err != nil {
		return Enrollment{}, err
	}
	now := s.clock.Now().UTC()
	expires := now.Add(enrollmentTTL)
	id := uuid.NewString()
	ciphertext, secretRef, err := s.storeSecret(userID, "pending_"+id, secret)
	if err != nil {
		return Enrollment{}, err
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO mfa_pending_enrollments(id,user_id,realm_id,secret_ciphertext,secret_ref,expires_at,created_at) VALUES(?,?,?,?,?,?,?)`, id, userID, realmID, ciphertext, secretRef, expires.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)); err != nil {
		if s.custodian != nil {
			_ = s.custodian.Delete(userID, secretRef)
		}
		return Enrollment{}, fmt.Errorf("store MFA enrollment: %w", err)
	}
	return Enrollment{ID: id, ProvisioningURI: uri, ExpiresAt: expires}, nil
}

// ConfirmEnrollment verifies the seed before activation, then atomically
// replaces the old enrollment and recovery-code set. Plaintext recovery codes
// are returned once to the owner and never persisted.
func (s *Store) ConfirmEnrollment(ctx context.Context, userID, enrollmentID, code string) ([]string, error) {
	var realmID, ciphertext, secretRef, expiresRaw string
	err := s.db.QueryRowContext(ctx, `SELECT realm_id,secret_ciphertext,secret_ref,expires_at FROM mfa_pending_enrollments WHERE id=? AND user_id=?`, enrollmentID, userID).Scan(&realmID, &ciphertext, &secretRef, &expiresRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrEnrollmentExpired
	}
	if err != nil {
		return nil, fmt.Errorf("read MFA enrollment: %w", err)
	}
	expires, err := time.Parse(time.RFC3339Nano, expiresRaw)
	if err != nil || !s.clock.Now().Before(expires) {
		return nil, ErrEnrollmentExpired
	}
	secret, err := s.loadSecret(userID, ciphertext, secretRef)
	if err != nil || !Verify(string(secret), code, s.clock.Now()) {
		return nil, ErrCodeInvalid
	}
	recoveryCodes, err := GenerateRecoveryCodes(recoveryCount)
	if err != nil {
		return nil, err
	}
	hashes := make([]string, len(recoveryCodes))
	for i, recoveryCode := range recoveryCodes {
		hashes[i], err = HashRecoveryCode(recoveryCode)
		if err != nil {
			return nil, err
		}
	}
	now := s.clock.Now().UTC().Format(time.RFC3339Nano)
	permanentRef := secretRef
	oldRef := ""
	if s.custodian != nil {
		if err := s.db.QueryRowContext(ctx, `SELECT secret_ref FROM mfa_enrollments WHERE user_id=?`, userID).Scan(&oldRef); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("read active MFA seed reference: %w", err)
		}
		permanentRef = "seed_" + uuid.NewString()
		if err := s.custodian.Put(userID, permanentRef, string(secret)); err != nil {
			return nil, fmt.Errorf("store active MFA seed: %w", err)
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		if s.custodian != nil {
			_ = s.custodian.Delete(userID, permanentRef)
		}
		return nil, fmt.Errorf("begin MFA enrollment transaction: %w", err)
	}
	rollback := func(cause error) ([]string, error) {
		_ = tx.Rollback()
		if s.custodian != nil {
			_ = s.custodian.Delete(userID, permanentRef)
		}
		return nil, cause
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM mfa_recovery_codes WHERE user_id=?`, userID); err != nil {
		return rollback(fmt.Errorf("replace MFA recovery codes: %w", err))
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO mfa_enrollments(user_id,realm_id,secret_ciphertext,secret_ref,enrolled_at,updated_at) VALUES(?,?,?,?,?,?) ON CONFLICT(user_id) DO UPDATE SET realm_id=excluded.realm_id,secret_ciphertext=excluded.secret_ciphertext,secret_ref=excluded.secret_ref,enrolled_at=excluded.enrolled_at,updated_at=excluded.updated_at`, userID, realmID, ciphertext, permanentRef, now, now); err != nil {
		return rollback(fmt.Errorf("save MFA enrollment: %w", err))
	}
	for _, hash := range hashes {
		if _, err := tx.ExecContext(ctx, `INSERT INTO mfa_recovery_codes(id,user_id,code_hash,created_at) VALUES(?,?,?,?)`, uuid.NewString(), userID, hash, now); err != nil {
			return rollback(fmt.Errorf("save MFA recovery code: %w", err))
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM mfa_pending_enrollments WHERE id=? AND user_id=?`, enrollmentID, userID); err != nil {
		return rollback(fmt.Errorf("finish MFA enrollment: %w", err))
	}
	if err := tx.Commit(); err != nil {
		if s.custodian != nil {
			_ = s.custodian.Delete(userID, permanentRef)
		}
		return nil, fmt.Errorf("commit MFA enrollment: %w", err)
	}
	if s.custodian != nil && secretRef != permanentRef {
		_ = s.custodian.Delete(userID, secretRef)
	}
	if s.custodian != nil && oldRef != "" && oldRef != permanentRef {
		_ = s.custodian.Delete(userID, oldRef)
	}
	return recoveryCodes, nil
}

func (s *Store) RemoveEnrollment(ctx context.Context, userID string) error {
	var secretRef string
	if s.custodian != nil {
		if err := s.db.QueryRowContext(ctx, `SELECT secret_ref FROM mfa_enrollments WHERE user_id=?`, userID).Scan(&secretRef); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM mfa_recovery_codes WHERE user_id=?`, userID); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM mfa_enrollments WHERE user_id=?`, userID); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	if s.custodian != nil && secretRef != "" {
		return s.custodian.Delete(userID, secretRef)
	}
	return nil
}

func (s *Store) Required(ctx context.Context, realmID string) (bool, error) {
	var required int
	err := s.db.QueryRowContext(ctx, `SELECT mfa_required FROM realms WHERE id=?`, realmID).Scan(&required)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return required != 0, err
}

func (s *Store) Enrolled(ctx context.Context, userID string) (bool, error) {
	var enrolled int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM mfa_enrollments WHERE user_id=?`, userID).Scan(&enrolled)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil && enrolled == 1, err
}

// StartChallenge creates a one-time opaque challenge. Only its digest is
// stored, so a database reader cannot redeem a challenge directly.
func (s *Store) StartChallenge(ctx context.Context, userID string) (string, time.Time, error) {
	challenge := uuid.NewString() + uuid.NewString()
	digest := sha256.Sum256([]byte(challenge))
	now := s.clock.Now().UTC()
	expires := now.Add(challengeTTL)
	_, err := s.db.ExecContext(ctx, `INSERT INTO mfa_login_challenges(id,user_id,challenge_digest,expires_at,created_at) VALUES(?,?,?,?,?)`, uuid.NewString(), userID, base64.RawURLEncoding.EncodeToString(digest[:]), expires.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	if err != nil {
		return "", time.Time{}, err
	}
	return challenge, expires, nil
}

// VerifyLogin consumes the challenge only after a valid TOTP or recovery
// code. Recovery hashes are consumed with a conditional update, making replay
// fail under concurrent requests.
func (s *Store) VerifyLogin(ctx context.Context, userID, challenge, totpCode, recoveryCode string) (bool, error) {
	digest := sha256.Sum256([]byte(challenge))
	digestText := base64.RawURLEncoding.EncodeToString(digest[:])
	var challengeID, expiresRaw, usedRaw string
	err := s.db.QueryRowContext(ctx, `SELECT id,expires_at,used_at FROM mfa_login_challenges WHERE user_id=? AND challenge_digest=?`, userID, digestText).Scan(&challengeID, &expiresRaw, &usedRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return false, ErrChallengeInvalid
	}
	if err != nil {
		return false, err
	}
	expires, err := time.Parse(time.RFC3339Nano, expiresRaw)
	if err != nil || usedRaw != "" || !s.clock.Now().Before(expires) {
		return false, ErrChallengeInvalid
	}

	valid := false
	if strings.TrimSpace(totpCode) != "" {
		var ciphertext, secretRef string
		if err := s.db.QueryRowContext(ctx, `SELECT secret_ciphertext,secret_ref FROM mfa_enrollments WHERE user_id=?`, userID).Scan(&ciphertext, &secretRef); err != nil {
			return false, err
		}
		secret, err := s.loadSecret(userID, ciphertext, secretRef)
		valid = err == nil && Verify(string(secret), totpCode, s.clock.Now())
	} else if strings.TrimSpace(recoveryCode) != "" {
		rows, err := s.db.QueryContext(ctx, `SELECT id,code_hash FROM mfa_recovery_codes WHERE user_id=? AND used_at=''`, userID)
		if err != nil {
			return false, err
		}
		type recoveryCandidate struct {
			id   string
			hash string
		}
		candidates := make([]recoveryCandidate, 0, recoveryCount)
		for rows.Next() {
			var id, hash string
			if err := rows.Scan(&id, &hash); err != nil {
				_ = rows.Close()
				return false, err
			}
			candidates = append(candidates, recoveryCandidate{id: id, hash: hash})
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return false, err
		}
		if err := rows.Close(); err != nil {
			return false, err
		}
		for _, candidate := range candidates {
			if !VerifyRecoveryCode(recoveryCode, candidate.hash) {
				continue
			}
			result, updateErr := s.db.ExecContext(ctx, `UPDATE mfa_recovery_codes SET used_at=? WHERE id=? AND user_id=? AND used_at=''`, s.clock.Now().UTC().Format(time.RFC3339Nano), candidate.id, userID)
			if updateErr != nil {
				return false, updateErr
			}
			count, _ := result.RowsAffected()
			valid = count == 1
			break
		}
	}
	if !valid {
		return false, ErrCodeInvalid
	}
	result, err := s.db.ExecContext(ctx, `UPDATE mfa_login_challenges SET used_at=? WHERE id=? AND user_id=? AND used_at=''`, s.clock.Now().UTC().Format(time.RFC3339Nano), challengeID, userID)
	if err != nil {
		return false, err
	}
	count, _ := result.RowsAffected()
	return count == 1, nil
}

func (s *Store) storeSecret(userID, slot, secret string) (string, string, error) {
	if s.custodian != nil {
		if err := s.custodian.Put(userID, slot, secret); err != nil {
			return "", "", fmt.Errorf("store MFA seed: %w", err)
		}
		return "", slot, nil
	}
	ciphertext, err := s.seal([]byte(secret))
	return ciphertext, "", err
}

func (s *Store) loadSecret(userID, ciphertext, secretRef string) ([]byte, error) {
	if s.custodian != nil {
		if secretRef == "" {
			return nil, errors.New("MFA seed custody reference is missing")
		}
		secret, err := s.custodian.Resolve(userID, secretRef)
		return []byte(secret), err
	}
	return s.open(ciphertext)
}

func (s *Store) seal(plaintext []byte) (string, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.RawStdEncoding.EncodeToString(sealed), nil
}

func (s *Store) open(encoded string) ([]byte, error) {
	sealed, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil || len(sealed) < gcm.NonceSize() {
		return nil, errors.New("invalid MFA ciphertext")
	}
	return gcm.Open(nil, sealed[:gcm.NonceSize()], sealed[gcm.NonceSize():], nil)
}
