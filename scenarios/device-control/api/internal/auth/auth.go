// Package auth owns device authentication policy and the safe boundary to the
// Vrooli credential authority. It persists references and policy only; secret
// material exists solely during one unlock transaction.
package auth

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"device-control/strategy"
	"github.com/google/uuid"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

const CredentialNamespace = "device-control/"

const (
	MethodPIN              = "pin"
	MethodNumeric          = "numeric_passcode"
	MethodPassword         = "password" // #nosec G101 -- public method label, never credential material; gitleaks:allow
	MethodPattern          = "pattern"
	MethodBiometric        = "biometric"
	MethodHumanGated       = "human_gated"
	MethodUnsupported      = "unsupported"
	ProfileActive          = "active"
	ProfileRevoked         = "revoked"
	OutcomeAlreadyUnlocked = "already_unlocked"
	OutcomeUnlocked        = "unlocked"
	OutcomeProfileMissing  = "profile_missing"
	OutcomeProfileRevoked  = "profile_revoked"
	OutcomeProviderAbsent  = "credential_provider_absent"
	OutcomeProviderDown    = "credential_provider_unavailable"
	OutcomeUnconfigured    = "credential_unconfigured"
	OutcomeUnknownState    = "unknown_device_state"
	OutcomeUnsupported     = "unsupported_method"
	OutcomeHumanRequired   = "human_required"
	OutcomeWrongCredential = "wrong_credential" // #nosec G101 -- public failure code, never credential material
	OutcomeTimeout         = "timeout"
	OutcomeTransport       = "transport_error"
	OutcomeCancelled       = "cancelled"
	OutcomeVerifiedFailed  = "verification_failed"
	OutcomeInvalidProfile  = "invalid_profile"
)

type Policy struct {
	MaxAttempts  int           `json:"max_attempts"`
	AttemptLimit time.Duration `json:"attempt_limit"`
	Settle       time.Duration `json:"settle"`
}

type Profile struct {
	ID                 string    `json:"id"`
	DeviceID           string    `json:"device_id"`
	Method             string    `json:"method"`
	CredentialIdentity string    `json:"credential_identity"`
	CredentialField    string    `json:"credential_field"`
	Verification       string    `json:"verification"`
	Policy             Policy    `json:"policy"`
	Status             string    `json:"status"`
	LastOutcome        string    `json:"last_outcome,omitempty"`
	RevokedAt          time.Time `json:"revoked_at,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type ProviderStatus struct {
	Provider      string `json:"provider"`
	ProviderState string `json:"provider_state"`
	Configured    bool   `json:"configured"`
	Detail        string `json:"provider_detail,omitempty"`
}

type Resolver interface {
	Provision(context.Context, string, string, string) error
	Resolve(context.Context, string, string) (string, error)
	Delete(context.Context, string, string) error
	Status(context.Context, string, string) ProviderStatus
}

type authorityResolver struct {
	authority *credentialauthority.Authority
}

func NewAuthorityResolver() (Resolver, error) {
	authority, err := credentialauthority.Default()
	if err != nil {
		return nil, err
	}
	return authorityResolver{authority: authority}, nil
}

func (r authorityResolver) Provision(_ context.Context, identity, field, value string) error {
	parsed, err := credentialauthority.ParseIdentity(identity)
	if err != nil {
		return err
	}
	return r.authority.Put(parsed, field, value)
}

func (r authorityResolver) Resolve(_ context.Context, identity, field string) (string, error) {
	parsed, err := credentialauthority.ParseIdentity(identity)
	if err != nil {
		return "", err
	}
	return r.authority.Resolve(parsed, field)
}

func (r authorityResolver) Delete(_ context.Context, identity, field string) error {
	parsed, err := credentialauthority.ParseIdentity(identity)
	if err != nil {
		return err
	}
	return r.authority.Delete(parsed, field)
}

func (r authorityResolver) Status(_ context.Context, identity, field string) ProviderStatus {
	parsed, err := credentialauthority.ParseIdentity(identity)
	if err != nil {
		return ProviderStatus{ProviderState: "available", Detail: err.Error()}
	}
	status := r.authority.Status(parsed, field)
	return ProviderStatus{Provider: status.Provider, ProviderState: string(status.ProviderState), Configured: status.Configured, Detail: status.ProviderDetail}
}

type dbLike interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type Store struct {
	db       dbLike
	resolver Resolver
	mu       sync.RWMutex
	profiles map[string]Profile
}

func NewStore(db dbLike, resolver Resolver) (*Store, error) {
	if resolver == nil {
		resolver, _ = NewAuthorityResolver()
	}
	s := &Store{db: db, resolver: resolver, profiles: map[string]Profile{}}
	if db == nil {
		return s, nil
	}
	if _, err := db.ExecContext(context.Background(), `
CREATE TABLE IF NOT EXISTS device_control_auth_profiles (
 id TEXT PRIMARY KEY, device_id TEXT NOT NULL, method TEXT NOT NULL,
 credential_identity TEXT NOT NULL, credential_field TEXT NOT NULL,
 verification TEXT NOT NULL, max_attempts INTEGER NOT NULL,
 attempt_limit_ms INTEGER NOT NULL, settle_ms INTEGER NOT NULL,
 status TEXT NOT NULL, last_outcome TEXT NOT NULL DEFAULT '',
 revoked_at TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL, updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS device_control_auth_profiles_device ON device_control_auth_profiles(device_id, status);`); err != nil {
		return nil, fmt.Errorf("initialize authentication profiles: %w", err)
	}
	return s, s.load(context.Background())
}

func (s *Store) load(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id, device_id, method, credential_identity, credential_field, verification, max_attempts, attempt_limit_ms, settle_ms, status, last_outcome, revoked_at, created_at, updated_at FROM device_control_auth_profiles`)
	if err != nil {
		return fmt.Errorf("load authentication profiles: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var p Profile
		var attemptMS, settleMS int64
		var revoked, created, updated string
		if err := rows.Scan(&p.ID, &p.DeviceID, &p.Method, &p.CredentialIdentity, &p.CredentialField, &p.Verification, &p.Policy.MaxAttempts, &attemptMS, &settleMS, &p.Status, &p.LastOutcome, &revoked, &created, &updated); err != nil {
			return err
		}
		p.Policy.AttemptLimit, p.Policy.Settle = time.Duration(attemptMS)*time.Millisecond, time.Duration(settleMS)*time.Millisecond
		p.RevokedAt, _ = time.Parse(time.RFC3339Nano, revoked)
		p.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		p.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		s.profiles[p.ID] = p
	}
	return rows.Err()
}

func ValidateProfile(p Profile) error {
	p.DeviceID = strings.TrimSpace(p.DeviceID)
	p.Method = strings.TrimSpace(strings.ToLower(p.Method))
	p.CredentialIdentity = strings.TrimSpace(strings.ToLower(p.CredentialIdentity))
	p.CredentialField = strings.TrimSpace(p.CredentialField)
	if p.DeviceID == "" || p.CredentialIdentity == "" || p.CredentialField == "" {
		return fmt.Errorf("device_id, credential_identity, and credential_field are required")
	}
	if !strings.HasPrefix(p.CredentialIdentity, CredentialNamespace) {
		return fmt.Errorf("credential identity must use the %q namespace", CredentialNamespace)
	}
	switch p.Method {
	case MethodPIN, MethodNumeric, MethodPassword, MethodPattern, MethodBiometric, MethodHumanGated:
	default:
		return fmt.Errorf("unsupported authentication method %q", p.Method)
	}
	if p.Verification == "" {
		return fmt.Errorf("verification policy is required")
	}
	if strings.ContainsAny(p.CredentialField, "/\\") {
		return fmt.Errorf("credential_field cannot contain a path separator")
	}
	return nil
}

func normalizeProfile(p Profile) Profile {
	if p.ID == "" {
		p.ID = "auth-" + uuid.NewString()
	}
	if p.Verification == "" {
		p.Verification = "fresh_lock_state_unlocked"
	}
	if p.Policy.MaxAttempts <= 0 || p.Policy.MaxAttempts > 3 {
		p.Policy.MaxAttempts = 1
	}
	if p.Policy.AttemptLimit <= 0 || p.Policy.AttemptLimit > time.Minute {
		p.Policy.AttemptLimit = 15 * time.Second
	}
	if p.Policy.Settle <= 0 || p.Policy.Settle > 10*time.Second {
		p.Policy.Settle = 750 * time.Millisecond
	}
	if p.Status == "" {
		p.Status = ProfileActive
	}
	now := time.Now().UTC()
	p.CreatedAt, p.UpdatedAt = now, now
	return p
}

func (s *Store) Create(ctx context.Context, input Profile) (Profile, error) {
	input = normalizeProfile(input)
	if err := ValidateProfile(input); err != nil {
		return Profile{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.profiles[input.ID]; exists {
		return Profile{}, fmt.Errorf("authentication profile %q already exists", input.ID)
	}
	if s.db != nil {
		if _, err := s.db.ExecContext(ctx, `INSERT INTO device_control_auth_profiles (id, device_id, method, credential_identity, credential_field, verification, max_attempts, attempt_limit_ms, settle_ms, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, input.ID, input.DeviceID, input.Method, input.CredentialIdentity, input.CredentialField, input.Verification, input.Policy.MaxAttempts, input.Policy.AttemptLimit.Milliseconds(), input.Policy.Settle.Milliseconds(), input.Status, input.CreatedAt.Format(time.RFC3339Nano), input.UpdatedAt.Format(time.RFC3339Nano)); err != nil {
			return Profile{}, err
		}
	}
	s.profiles[input.ID] = input
	return input, nil
}

// Update changes profile metadata and policy only. It never accepts or
// resolves credential material; the authority reference remains the only
// secret boundary.
func (s *Store) Update(ctx context.Context, id string, input Profile) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.profiles[id]
	if !ok {
		return Profile{}, fmt.Errorf("authentication profile %q not found", id)
	}
	if existing.Status != ProfileActive {
		return Profile{}, fmt.Errorf("authentication profile %q is revoked", id)
	}
	input.ID = id
	if input.DeviceID == "" {
		input.DeviceID = existing.DeviceID
	}
	if input.Method == "" {
		input.Method = existing.Method
	}
	if input.CredentialIdentity == "" {
		input.CredentialIdentity = existing.CredentialIdentity
	}
	if input.CredentialField == "" {
		input.CredentialField = existing.CredentialField
	}
	if input.Verification == "" {
		input.Verification = existing.Verification
	}
	if input.Policy.MaxAttempts <= 0 {
		input.Policy.MaxAttempts = existing.Policy.MaxAttempts
	}
	if input.Policy.AttemptLimit <= 0 {
		input.Policy.AttemptLimit = existing.Policy.AttemptLimit
	}
	if input.Policy.Settle <= 0 {
		input.Policy.Settle = existing.Policy.Settle
	}
	input.Status = ProfileActive
	input.CreatedAt = existing.CreatedAt
	input.UpdatedAt = time.Now().UTC()
	if err := ValidateProfile(input); err != nil {
		return Profile{}, err
	}
	if s.db != nil {
		_, err := s.db.ExecContext(ctx, `UPDATE device_control_auth_profiles SET device_id = ?, method = ?, credential_identity = ?, credential_field = ?, verification = ?, max_attempts = ?, attempt_limit_ms = ?, settle_ms = ?, status = ?, revoked_at = ?, updated_at = ? WHERE id = ?`, input.DeviceID, input.Method, input.CredentialIdentity, input.CredentialField, input.Verification, input.Policy.MaxAttempts, input.Policy.AttemptLimit.Milliseconds(), input.Policy.Settle.Milliseconds(), input.Status, "", input.UpdatedAt.Format(time.RFC3339Nano), id)
		if err != nil {
			return Profile{}, err
		}
	}
	s.profiles[id] = input
	return input, nil
}

func (s *Store) Get(_ context.Context, id string) (Profile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.profiles[id]
	return p, ok
}

func (s *Store) ProviderStatus(ctx context.Context, p Profile) ProviderStatus {
	if s.resolver == nil {
		return ProviderStatus{ProviderState: "absent"}
	}
	return s.resolver.Status(ctx, p.CredentialIdentity, p.CredentialField)
}

func (s *Store) List(_ context.Context) []Profile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Profile, 0, len(s.profiles))
	for _, p := range s.profiles {
		out = append(out, p)
	}
	return out
}

func (s *Store) Revoke(ctx context.Context, id string) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.profiles[id]
	if !ok {
		return Profile{}, fmt.Errorf("authentication profile %q not found", id)
	}
	p.Status, p.RevokedAt, p.UpdatedAt = ProfileRevoked, time.Now().UTC(), time.Now().UTC()
	if s.db != nil {
		if _, err := s.db.ExecContext(ctx, `UPDATE device_control_auth_profiles SET status = ?, revoked_at = ?, updated_at = ? WHERE id = ?`, p.Status, p.RevokedAt.Format(time.RFC3339Nano), p.UpdatedAt.Format(time.RFC3339Nano), id); err != nil {
			return Profile{}, err
		}
	}
	s.profiles[id] = p
	return p, nil
}

func (s *Store) RecordOutcome(ctx context.Context, id, outcome string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.profiles[id]
	if !ok {
		return
	}
	p.LastOutcome, p.UpdatedAt = outcome, time.Now().UTC()
	if s.db != nil {
		_, _ = s.db.ExecContext(ctx, `UPDATE device_control_auth_profiles SET last_outcome = ?, updated_at = ? WHERE id = ?`, p.LastOutcome, p.UpdatedAt.Format(time.RFC3339Nano), id)
	}
	s.profiles[id] = p
}

func (s *Store) Provision(ctx context.Context, id string, valueReader io.Reader) (ProviderStatus, error) {
	p, ok := s.Get(ctx, id)
	if !ok {
		return ProviderStatus{}, fmt.Errorf("authentication profile %q not found", id)
	}
	if p.Status != ProfileActive {
		return ProviderStatus{}, fmt.Errorf("authentication profile %q is revoked", id)
	}
	raw, err := io.ReadAll(io.LimitReader(valueReader, 4097))
	if err != nil {
		return ProviderStatus{}, err
	}
	defer func() {
		for i := range raw {
			raw[i] = 0
		}
	}()
	value := bytes.TrimSpace(raw)
	if len(value) == 0 || len(value) > 4096 {
		return ProviderStatus{}, fmt.Errorf("credential input is empty or exceeds the bounded input size")
	}
	if s.resolver == nil {
		return ProviderStatus{ProviderState: "absent"}, errors.New("credential authority is unavailable")
	}
	if err := s.resolver.Provision(ctx, p.CredentialIdentity, p.CredentialField, string(value)); err != nil {
		return s.resolver.Status(ctx, p.CredentialIdentity, p.CredentialField), err
	}
	return s.resolver.Status(ctx, p.CredentialIdentity, p.CredentialField), nil
}

// DeleteCredential removes the authority-held value for a profile reference.
// Profile metadata remains available for audit and can be revoked separately.
func (s *Store) DeleteCredential(ctx context.Context, id string) (ProviderStatus, error) {
	p, ok := s.Get(ctx, id)
	if !ok {
		return ProviderStatus{}, fmt.Errorf("authentication profile %q not found", id)
	}
	if s.resolver == nil {
		return ProviderStatus{ProviderState: "absent"}, errors.New("credential authority is unavailable")
	}
	if err := s.resolver.Delete(ctx, p.CredentialIdentity, p.CredentialField); err != nil {
		return s.resolver.Status(ctx, p.CredentialIdentity, p.CredentialField), err
	}
	return s.resolver.Status(ctx, p.CredentialIdentity, p.CredentialField), nil
}

type UnlockResponse struct {
	ProfileID       string `json:"profile_id"`
	DeviceID        string `json:"device_id"`
	Method          string `json:"method"`
	Outcome         string `json:"outcome"`
	NextAction      string `json:"next_action"`
	Attempts        int    `json:"attempts"`
	ProviderState   string `json:"provider_state,omitempty"`
	BeforeLockState string `json:"before_lock_state"`
	AfterLockState  string `json:"after_lock_state"`
}

func (s *Store) Unlock(ctx context.Context, profileID, deviceID string, before strategy.DeviceState, unlocker strategy.Unlocker, verify func(context.Context) (strategy.DeviceState, error)) (UnlockResponse, error) {
	p, ok := s.Get(ctx, profileID)
	if !ok || p.DeviceID != deviceID {
		return UnlockResponse{ProfileID: profileID, DeviceID: deviceID, Outcome: OutcomeProfileMissing, NextAction: "create an active profile bound to this device"}, nil
	}
	response := UnlockResponse{ProfileID: p.ID, DeviceID: deviceID, Method: p.Method, BeforeLockState: before.LockState, NextAction: "inspect the typed outcome"}
	if p.Status != ProfileActive {
		response.Outcome, response.NextAction = OutcomeProfileRevoked, "create or reactivate an authentication profile"
		return response, nil
	}
	if before.LockState == "unlocked" {
		response.Outcome, response.AfterLockState, response.NextAction = OutcomeAlreadyUnlocked, "continue the flow", "continue the flow"
		return response, nil
	}
	if before.LockState != "locked" {
		response.Outcome, response.NextAction = OutcomeUnknownState, "re-probe the device state; do not actuate while unknown"
		return response, nil
	}
	if unlocker == nil {
		response.Outcome, response.NextAction = OutcomeUnsupported, "use a strategy with an unlock adapter"
		return response, nil
	}
	switch p.Method {
	case MethodPIN, MethodNumeric:
		// The Android adapter currently supports only numeric key-event input.
	case MethodBiometric, MethodHumanGated:
		response.Outcome, response.NextAction = OutcomeHumanRequired, nextForOutcome(OutcomeHumanRequired)
		return response, nil
	default:
		response.Outcome, response.NextAction = OutcomeUnsupported, nextForOutcome(OutcomeUnsupported)
		return response, nil
	}
	if s.resolver == nil {
		response.Outcome, response.NextAction = OutcomeProviderAbsent, "configure the Vrooli credential authority"
		return response, nil
	}
	secret, err := s.resolver.Resolve(ctx, p.CredentialIdentity, p.CredentialField)
	if err != nil {
		response.ProviderState = s.resolver.Status(ctx, p.CredentialIdentity, p.CredentialField).ProviderState
		response.Outcome, response.NextAction = outcomeForCredentialError(err), nextForOutcome(outcomeForCredentialError(err))
		return response, nil
	}
	secretBytes := []byte(secret)
	for i := range secretBytes {
		defer func(i int) { secretBytes[i] = 0 }(i)
	}
	result, unlockErr := unlocker.Unlock(ctx, strategy.UnlockRequest{Method: p.Method, Secret: secretBytes, MaxAttempts: p.Policy.MaxAttempts, AttemptLimit: p.Policy.AttemptLimit, Settle: p.Policy.Settle})
	response.Attempts, response.Outcome = result.Attempts, result.Outcome
	if unlockErr != nil {
		response.Outcome, response.NextAction = OutcomeTransport, "verify the device connection and retry once"
		return response, unlockErr
	}
	response.NextAction = nextForOutcome(response.Outcome)
	if response.Outcome != OutcomeUnlocked && response.Outcome != OutcomeAlreadyUnlocked {
		return response, nil
	}
	after, verifyErr := verify(ctx)
	if verifyErr != nil {
		response.Outcome, response.NextAction = OutcomeVerifiedFailed, "the postcondition could not be verified; keep the flow stopped"
		return response, verifyErr
	}
	response.AfterLockState = after.LockState
	if after.LockState != "unlocked" {
		response.Outcome, response.NextAction = OutcomeVerifiedFailed, "the fresh keyguard probe did not prove unlocked"
	}
	return response, nil
}

func outcomeForCredentialError(err error) string {
	switch {
	case errors.Is(err, credentialauthority.ErrProviderAbsent):
		return OutcomeProviderAbsent
	case errors.Is(err, credentialauthority.ErrProviderUnavailable):
		return OutcomeProviderDown
	case errors.Is(err, credentialauthority.ErrUnconfigured):
		return OutcomeUnconfigured
	default:
		return OutcomeProviderDown
	}
}

func nextForOutcome(outcome string) string {
	switch outcome {
	case OutcomeUnlocked, OutcomeAlreadyUnlocked:
		return "continue the flow"
	case OutcomeProviderAbsent:
		return "install or enable a supported Vrooli credential provider"
	case OutcomeProviderDown:
		return "repair the credential provider and inspect its diagnosis"
	case OutcomeUnconfigured:
		return "provision the profile credential through stdin"
	case OutcomeHumanRequired:
		return "have an operator complete authentication, then re-probe"
	case OutcomeUnsupported:
		return "use a supported Android PIN method or an operator gate"
	case OutcomeWrongCredential:
		return "verify the credential; automatic retries are disabled"
	case OutcomeUnknownState:
		return "re-probe device state; do not actuate while unknown"
	default:
		return "keep the flow stopped and inspect the typed outcome"
	}
}
