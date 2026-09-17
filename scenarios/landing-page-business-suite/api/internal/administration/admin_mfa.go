package administration

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" // #nosec G505 -- RFC 6238 authenticator apps require HMAC-SHA1.
	"crypto/subtle"
	"database/sql"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"landing-page-business-suite-api/internal/securevalue"
)

// Admin two-factor authentication failures.
var (
	ErrMFARequired       = errors.New("two-factor code required")
	ErrMFAInvalid        = errors.New("two-factor code is invalid")
	ErrMFANotEnrolling   = errors.New("two-factor enrollment has not been started")
	ErrMFAAlreadyEnabled = errors.New("two-factor authentication is already enabled")
	ErrMFAUnavailable    = errors.New("two-factor encryption key is unavailable")
)

const (
	totpPeriod        = 30
	totpDigits        = 6
	totpSkewSteps     = 1
	recoveryCodeCount = 10
)

// AdminMFAStatus is the non-secret two-factor state for one administrator.
type AdminMFAStatus struct {
	Enabled             bool       `json:"enabled"`
	EnabledAt           *time.Time `json:"enabled_at,omitempty"`
	RecoveryCodesLeft   int        `json:"recovery_codes_left"`
	EnrollmentInProcess bool       `json:"enrollment_in_progress"`
}

// AdminMFAEnrollment is shown once while the administrator scans the secret.
type AdminMFAEnrollment struct {
	Secret     string `json:"secret"`
	OTPAuthURI string `json:"otpauth_uri"`
}

// AdminMFA owns TOTP enrollment and verification. Secrets are sealed with a
// dedicated key ring; recovery codes are stored as bcrypt hashes.
type AdminMFA struct {
	store  AdminAuthStore
	ring   func() (securevalue.Ring, error)
	issuer func() string
	now    func() time.Time
}

// NewAdminMFA creates the admin two-factor service. issuer names the site in
// authenticator apps; it is read at enrollment so a rebrand applies to new
// enrollments without a restart.
func NewAdminMFA(store AdminAuthStore, ring func() (securevalue.Ring, error), issuer func() string) *AdminMFA {
	return &AdminMFA{store: store, ring: ring, issuer: issuer, now: time.Now}
}

func (m *AdminMFA) issuerName() string {
	if m.issuer != nil {
		if name := strings.TrimSpace(m.issuer()); name != "" {
			return name
		}
	}
	return "Landing Page Business Suite"
}

// UseClock replaces the clock in tests.
func (m *AdminMFA) UseClock(now func() time.Time) { m.now = now }

// Status reports whether two-factor authentication is enabled.
func (m *AdminMFA) Status(ctx context.Context, email string) (*AdminMFAStatus, error) {
	var enabledAt sql.NullTime
	var pending, codes sql.NullString
	err := m.store.QueryRowContext(ctx, `
		SELECT totp_enabled_at, totp_pending_secret_encrypted, recovery_code_hashes::text FROM admin_users WHERE email = $1
	`, email).Scan(&enabledAt, &pending, &codes)
	if err != nil {
		return nil, err
	}
	status := &AdminMFAStatus{Enabled: enabledAt.Valid, EnrollmentInProcess: pending.Valid && pending.String != ""}
	if enabledAt.Valid {
		status.EnabledAt = &enabledAt.Time
		status.RecoveryCodesLeft = len(decodeRecoveryHashes(codes.String))
	}
	return status, nil
}

// Enabled reports whether an administrator must present a second factor.
func (m *AdminMFA) Enabled(ctx context.Context, email string) (bool, error) {
	var enabled bool
	err := m.store.QueryRowContext(ctx, `SELECT (totp_enabled_at IS NOT NULL OR EXISTS (SELECT 1 FROM admin_passkeys p WHERE p.admin_id = admin_users.id AND p.revoked_at IS NULL)) FROM admin_users WHERE email = $1`, email).Scan(&enabled)
	if err != nil {
		return false, err
	}
	return enabled, nil
}

// BeginEnrollment creates a pending secret. It does not change login policy
// until ConfirmEnrollment proves the authenticator app produces valid codes.
func (m *AdminMFA) BeginEnrollment(ctx context.Context, email string) (*AdminMFAEnrollment, error) {
	var enabled bool
	err := m.store.QueryRowContext(ctx, `SELECT totp_enabled_at IS NOT NULL FROM admin_users WHERE email = $1`, email).Scan(&enabled)
	if err != nil {
		return nil, err
	}
	if enabled {
		return nil, ErrMFAAlreadyEnabled
	}
	raw := make([]byte, 20)
	if _, err := rand.Read(raw); err != nil {
		return nil, fmt.Errorf("generate totp secret: %w", err)
	}
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)
	sealed, err := m.seal(secret)
	if err != nil {
		return nil, err
	}
	if _, err := m.store.ExecContext(ctx, `UPDATE admin_users SET totp_pending_secret_encrypted = $1 WHERE email = $2`, sealed, email); err != nil {
		return nil, fmt.Errorf("store pending totp secret: %w", err)
	}
	issuer := m.issuerName()
	label := url.PathEscape(issuer + ":" + email)
	query := url.Values{"secret": {secret}, "issuer": {issuer}, "algorithm": {"SHA1"}, "digits": {"6"}, "period": {"30"}}
	return &AdminMFAEnrollment{Secret: secret, OTPAuthURI: "otpauth://totp/" + label + "?" + query.Encode()}, nil
}

// ConfirmEnrollment enables two-factor authentication and returns recovery
// codes. The codes are shown exactly once.
func (m *AdminMFA) ConfirmEnrollment(ctx context.Context, email, code string) ([]string, error) {
	var pending sql.NullString
	if err := m.store.QueryRowContext(ctx, `SELECT totp_pending_secret_encrypted FROM admin_users WHERE email = $1`, email).Scan(&pending); err != nil {
		return nil, err
	}
	if !pending.Valid || pending.String == "" {
		return nil, ErrMFANotEnrolling
	}
	secret, err := m.open(pending.String)
	if err != nil {
		return nil, err
	}
	step, ok := matchTOTP(secret, code, m.now(), -1)
	if !ok {
		return nil, ErrMFAInvalid
	}
	codes, hashes, err := newRecoveryCodes()
	if err != nil {
		return nil, err
	}
	encoded, _ := json.Marshal(hashes)
	_, err = m.store.ExecContext(ctx, `
		UPDATE admin_users
		SET totp_secret_encrypted = totp_pending_secret_encrypted, totp_pending_secret_encrypted = NULL,
		    totp_enabled_at = NOW(), totp_last_step = $1, recovery_code_hashes = $2::jsonb
		WHERE email = $3
	`, step, string(encoded), email)
	if err != nil {
		return nil, fmt.Errorf("enable totp: %w", err)
	}
	return codes, nil
}

// Verify accepts a current TOTP code (never the same time step twice) or an
// unused recovery code, which is consumed.
func (m *AdminMFA) Verify(ctx context.Context, email, code string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return ErrMFARequired
	}
	var sealed, codes sql.NullString
	var lastStep sql.NullInt64
	err := m.store.QueryRowContext(ctx, `
		SELECT totp_secret_encrypted, totp_last_step, recovery_code_hashes::text FROM admin_users WHERE email = $1 AND totp_enabled_at IS NOT NULL
	`, email).Scan(&sealed, &lastStep, &codes)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrMFAInvalid
	}
	if err != nil {
		return err
	}
	digits := onlyDigits(code)
	numeric := strings.Trim(strings.ReplaceAll(code, " ", ""), "0123456789") == ""
	if numeric && len(digits) == totpDigits && sealed.Valid {
		secret, openErr := m.open(sealed.String)
		if openErr != nil {
			return openErr
		}
		last := int64(-1)
		if lastStep.Valid {
			last = lastStep.Int64
		}
		if step, ok := matchTOTP(secret, digits, m.now(), last); ok {
			// The conditional update makes replay of the same step lose a race.
			result, err := m.store.ExecContext(ctx, `
				UPDATE admin_users SET totp_last_step = $1 WHERE email = $2 AND (totp_last_step IS NULL OR totp_last_step < $1)
			`, step, email)
			if err != nil {
				return err
			}
			if affected, _ := result.RowsAffected(); affected == 1 {
				return nil
			}
			return ErrMFAInvalid
		}
		return ErrMFAInvalid
	}
	return m.consumeRecoveryCode(ctx, email, code, codes.String)
}

// Disable turns two-factor authentication off after re-verifying a code.
func (m *AdminMFA) Disable(ctx context.Context, email, code string) error {
	if err := m.Verify(ctx, email, code); err != nil {
		return err
	}
	_, err := m.store.ExecContext(ctx, `
		UPDATE admin_users SET totp_secret_encrypted = NULL, totp_pending_secret_encrypted = NULL, totp_enabled_at = NULL, totp_last_step = NULL, recovery_code_hashes = NULL WHERE email = $1
	`, email)
	return err
}

// RegenerateRecoveryCodes replaces every recovery code after verifying a code.
func (m *AdminMFA) RegenerateRecoveryCodes(ctx context.Context, email, code string) ([]string, error) {
	if err := m.Verify(ctx, email, code); err != nil {
		return nil, err
	}
	codes, hashes, err := newRecoveryCodes()
	if err != nil {
		return nil, err
	}
	encoded, _ := json.Marshal(hashes)
	if _, err := m.store.ExecContext(ctx, `UPDATE admin_users SET recovery_code_hashes = $1::jsonb WHERE email = $2`, string(encoded), email); err != nil {
		return nil, err
	}
	return codes, nil
}

func (m *AdminMFA) consumeRecoveryCode(ctx context.Context, email, code, stored string) error {
	normalized := normalizeRecoveryCode(code)
	if len(normalized) != 10 {
		return ErrMFAInvalid
	}
	hashes := decodeRecoveryHashes(stored)
	for index, hash := range hashes {
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(normalized)) != nil {
			continue
		}
		remaining := append(append([]string{}, hashes[:index]...), hashes[index+1:]...)
		encoded, _ := json.Marshal(remaining)
		result, err := m.store.ExecContext(ctx, `
			UPDATE admin_users SET recovery_code_hashes = $1::jsonb WHERE email = $2 AND recovery_code_hashes::text = $3
		`, string(encoded), email, stored)
		if err != nil {
			return err
		}
		if affected, _ := result.RowsAffected(); affected == 1 {
			return nil
		}
		return ErrMFAInvalid
	}
	return ErrMFAInvalid
}

func (m *AdminMFA) seal(secret string) (string, error) {
	if m.ring == nil {
		return "", ErrMFAUnavailable
	}
	ring, err := m.ring()
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrMFAUnavailable, err)
	}
	return securevalue.EncryptRing(ring, secret)
}

func (m *AdminMFA) open(sealed string) (string, error) {
	if m.ring == nil {
		return "", ErrMFAUnavailable
	}
	ring, err := m.ring()
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrMFAUnavailable, err)
	}
	return securevalue.DecryptRing(ring, sealed)
}

// TOTPCode computes the RFC 6238 code for a base32 secret at a time.
func TOTPCode(secret string, at time.Time) (string, error) {
	return totpAt(secret, at.Unix()/totpPeriod)
}

func totpAt(secret string, step int64) (string, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.TrimRight(secret, "=")))
	if err != nil {
		return "", err
	}
	var counter [8]byte
	binary.BigEndian.PutUint64(counter[:], uint64(step))
	mac := hmac.New(sha1.New, key)
	mac.Write(counter[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	value := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", value%1000000), nil
}

func matchTOTP(secret, code string, now time.Time, after int64) (int64, bool) {
	code = onlyDigits(code)
	if len(code) != totpDigits {
		return 0, false
	}
	current := now.Unix() / totpPeriod
	for delta := -totpSkewSteps; delta <= totpSkewSteps; delta++ {
		step := current + int64(delta)
		if step <= after {
			continue
		}
		expected, err := totpAt(secret, step)
		if err == nil && subtle.ConstantTimeCompare([]byte(expected), []byte(code)) == 1 {
			return step, true
		}
	}
	return 0, false
}

const recoveryAlphabet = "abcdefghjkmnpqrstuvwxyz23456789"

func newRecoveryCodes() ([]string, []string, error) {
	codes := make([]string, 0, recoveryCodeCount)
	hashes := make([]string, 0, recoveryCodeCount)
	for len(codes) < recoveryCodeCount {
		raw := make([]byte, 10)
		if _, err := rand.Read(raw); err != nil {
			return nil, nil, err
		}
		var builder strings.Builder
		for _, b := range raw {
			builder.WriteByte(recoveryAlphabet[int(b)%len(recoveryAlphabet)])
		}
		normalized := builder.String()
		hash, err := bcrypt.GenerateFromPassword([]byte(normalized), bcrypt.DefaultCost)
		if err != nil {
			return nil, nil, err
		}
		codes = append(codes, normalized[:5]+"-"+normalized[5:])
		hashes = append(hashes, string(hash))
	}
	return codes, hashes, nil
}

func normalizeRecoveryCode(code string) string {
	return strings.ToLower(strings.NewReplacer("-", "", " ", "").Replace(strings.TrimSpace(code)))
}

func decodeRecoveryHashes(stored string) []string {
	var hashes []string
	if strings.TrimSpace(stored) == "" {
		return nil
	}
	_ = json.Unmarshal([]byte(stored), &hashes)
	return hashes
}

func onlyDigits(value string) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, value)
}

// Reset clears two-factor state without a code. It is reserved for the
// operator recovery path.
func (m *AdminMFA) Reset(ctx context.Context, email string) error {
	result, err := m.store.ExecContext(ctx, `
		UPDATE admin_users SET totp_secret_encrypted = NULL, totp_pending_secret_encrypted = NULL, totp_enabled_at = NULL, totp_last_step = NULL, recovery_code_hashes = NULL WHERE email = $1
	`, email)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}
