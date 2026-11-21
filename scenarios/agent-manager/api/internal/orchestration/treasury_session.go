// This file owns per-run Treasury session-key issuance and revocation.
package orchestration

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"agent-manager/internal/adapters/runner"
	runidentity "agent-manager/internal/identity"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/scopecatalog"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

const (
	treasurySessionIdentityPrefix = "vrooli/agent-manager-run-"
	treasurySessionKeyField       = "treasury-agent-session-key"
	treasurySessionKeyEnv         = "VROOLI_TREASURY_SESSION_PRIVATE_KEY"
)

type treasurySessionKeyState struct {
	Version         int    `json:"version"`
	Key             string `json:"key"`
	RotationRequest string `json:"rotation_request,omitempty"`
	Revoked         bool   `json:"revoked,omitempty"`
}

// This mutex serializes the Agent Manager process's read-modify-write key
// lifecycle. Production runs have one Agent Manager owner process; secure-store
// mutations themselves are atomic at their backend boundary, but the generic
// Store contract does not provide a cross-process transaction/CAS primitive.
var treasurySessionKeyMu sync.Mutex

// The issuance gate is shared with run lifecycle commits. A terminal transition
// holds its run-specific write lock across the durable status write and
// revocation; key leases hold that run's read lock through broker response
// delivery or process creation. Independent runs never block each other's
// launch or cleanup path.
var treasurySessionKeyGate treasurySessionKeyGateRegistry

type treasurySessionKeyGateEntry struct {
	gate sync.RWMutex
	refs int
}

type treasurySessionKeyGateRegistry struct {
	mu      sync.Mutex
	entries map[uuid.UUID]*treasurySessionKeyGateEntry
}

func (r *treasurySessionKeyGateRegistry) lock(runID uuid.UUID, write bool) func() {
	r.mu.Lock()
	if r.entries == nil {
		r.entries = make(map[uuid.UUID]*treasurySessionKeyGateEntry)
	}
	entry := r.entries[runID]
	if entry == nil {
		entry = &treasurySessionKeyGateEntry{}
		r.entries[runID] = entry
	}
	entry.refs++
	r.mu.Unlock()
	if write {
		entry.gate.Lock()
	} else {
		entry.gate.RLock()
	}
	var once sync.Once
	return func() {
		once.Do(func() {
			if write {
				entry.gate.Unlock()
			} else {
				entry.gate.RUnlock()
			}
			r.mu.Lock()
			entry.refs--
			if entry.refs == 0 && r.entries[runID] == entry {
				delete(r.entries, runID)
			}
			r.mu.Unlock()
		})
	}
}

func (r *treasurySessionKeyGateRegistry) readLock(runID uuid.UUID) func() {
	return r.lock(runID, false)
}

func (r *treasurySessionKeyGateRegistry) writeLock(runID uuid.UUID) func() {
	return r.lock(runID, true)
}

func resolveTreasurySessionKey(ctx context.Context, store TreasurySessionKeyStore, runID uuid.UUID, identityToken string, identitySecret []byte) (string, error) {
	if strings.TrimSpace(identityToken) == "" || len(identitySecret) == 0 {
		return "", nil
	}
	claims, err := runidentity.VerifyToken(identityToken, identitySecret)
	if err != nil {
		return "", fmt.Errorf("verify run identity before Treasury session-key injection: %w", err)
	}
	if !scopecatalog.MatchCapability(claims.Scopes, "treasury:write") {
		return "", nil
	}
	if store == nil {
		return "", fmt.Errorf("run has treasury:write capability but the run-owned Treasury session-key store is unavailable")
	}
	key, release, err := acquireTreasurySessionKey(ctx, store, runID, identityToken, identitySecret)
	if err != nil {
		return "", fmt.Errorf("provision run-owned Treasury session key: %w", err)
	}
	if release != nil {
		release()
	}
	if key == "" {
		return "", nil
	}
	if len(key) != 64 {
		return "", fmt.Errorf("run-owned Treasury session key has invalid encoding")
	}
	if _, err := hex.DecodeString(key); err != nil {
		return "", fmt.Errorf("run-owned Treasury session key has invalid encoding")
	}
	return strings.ToLower(key), nil
}

func acquireTreasurySessionKey(ctx context.Context, store TreasurySessionKeyStore, runID uuid.UUID, identityToken string, identitySecret []byte) (string, func(), error) {
	return acquireTreasurySessionKeyWithContinuation(ctx, store, runID, identityToken, identitySecret, false)
}

func acquireTreasurySessionKeyForContinuation(ctx context.Context, store TreasurySessionKeyStore, runID uuid.UUID, identityToken string, identitySecret []byte) (string, func(), error) {
	return acquireTreasurySessionKeyWithContinuation(ctx, store, runID, identityToken, identitySecret, true)
}

func acquireTreasurySessionKeyWithContinuation(ctx context.Context, store TreasurySessionKeyStore, runID uuid.UUID, identityToken string, identitySecret []byte, continuation bool) (string, func(), error) {
	if strings.TrimSpace(identityToken) == "" || len(identitySecret) == 0 {
		return "", nil, nil
	}
	claims, err := runidentity.VerifyToken(identityToken, identitySecret)
	if err != nil {
		return "", nil, fmt.Errorf("verify run identity before Treasury session-key injection: %w", err)
	}
	if !scopecatalog.MatchCapability(claims.Scopes, "treasury:write") {
		return "", nil, nil
	}
	if store == nil {
		return "", nil, fmt.Errorf("run has treasury:write capability but the run-owned Treasury session-key store is unavailable")
	}
	var key string
	var release func()
	if continuation {
		continuationStore, ok := store.(interface {
			AcquireContinuation(context.Context, uuid.UUID) (string, func(), error)
		})
		if !ok {
			return "", nil, fmt.Errorf("run-owned Treasury session-key store does not support continuation leases")
		}
		key, release, err = continuationStore.AcquireContinuation(ctx, runID)
	} else {
		leased, ok := store.(interface {
			Acquire(context.Context, uuid.UUID) (string, func(), error)
		})
		if !ok {
			return "", nil, fmt.Errorf("run-owned Treasury session-key store does not support revocable delivery leases")
		}
		key, release, err = leased.Acquire(ctx, runID)
	}
	if err != nil {
		return "", nil, err
	}
	if len(key) != 64 {
		if release != nil {
			release()
		}
		return "", nil, fmt.Errorf("run-owned Treasury session key has invalid encoding")
	}
	if _, err := hex.DecodeString(key); err != nil {
		if release != nil {
			release()
		}
		return "", nil, fmt.Errorf("run-owned Treasury session key has invalid encoding")
	}
	return strings.ToLower(key), release, nil
}

// treasurySessionLaunchPreparer holds the run-key lease until the selected
// launcher has accepted the process. This closes the terminal-cleanup window
// between resolving a key and handing it to a child process.
func treasurySessionLaunchPreparer(store TreasurySessionKeyStore, runID uuid.UUID, token string, secret []byte) func(context.Context, *runner.LaunchRequest) (func(), error) {
	return treasurySessionLaunchPreparerForContinuation(store, runID, token, secret, false)
}

func treasurySessionContinuationLaunchPreparer(store TreasurySessionKeyStore, runID uuid.UUID, token string, secret []byte) func(context.Context, *runner.LaunchRequest) (func(), error) {
	return treasurySessionLaunchPreparerForContinuation(store, runID, token, secret, true)
}

func treasurySessionLaunchPreparerForContinuation(store TreasurySessionKeyStore, runID uuid.UUID, token string, secret []byte, continuation bool) func(context.Context, *runner.LaunchRequest) (func(), error) {
	return func(ctx context.Context, req *runner.LaunchRequest) (func(), error) {
		if req == nil {
			return nil, fmt.Errorf("launch request is required")
		}
		var key string
		var release func()
		var err error
		if continuation {
			key, release, err = acquireTreasurySessionKeyForContinuation(ctx, store, runID, token, secret)
		} else {
			key, release, err = acquireTreasurySessionKey(ctx, store, runID, token, secret)
		}
		if err != nil {
			return nil, err
		}
		if key == "" {
			return release, nil
		}
		prefix := treasurySessionKeyEnv + "="
		env := make([]string, 0, len(req.Env)+1)
		for _, item := range req.Env {
			if !strings.HasPrefix(item, prefix) {
				env = append(env, item)
			}
		}
		req.Env = append(env, prefix+key)
		return release, nil
	}
}

// CredentialAuthorityTreasurySessionKeys keeps run signing keys in the native
// credential authority. Only the scoped runner process receives the value.
// Run rows and user-controlled CustomEnv never contain private-key material.
type CredentialAuthorityTreasurySessionKeys struct {
	open     func() (*credentialauthority.Authority, error)
	generate func() (string, error)
}

func NewCredentialAuthorityTreasurySessionKeys() *CredentialAuthorityTreasurySessionKeys {
	return &CredentialAuthorityTreasurySessionKeys{open: credentialauthority.Default, generate: mintTreasurySessionKey}
}

// NewCredentialAuthorityTreasurySessionKeysWithOpener binds the same run-key
// lifecycle to an explicitly scoped credential authority. Production wiring
// uses Default; isolated integration tests use an encrypted temporary store.
func NewCredentialAuthorityTreasurySessionKeysWithOpener(open func() (*credentialauthority.Authority, error)) *CredentialAuthorityTreasurySessionKeys {
	return &CredentialAuthorityTreasurySessionKeys{open: open, generate: mintTreasurySessionKey}
}

// NewCredentialAuthorityTreasurySessionKeysWithGenerator allows owner tests to
// use known, non-production fixture scalars when they need a live stale-key
// refusal probe. Production constructors always use cryptographic randomness.
func NewCredentialAuthorityTreasurySessionKeysWithGenerator(open func() (*credentialauthority.Authority, error), generate func() (string, error)) *CredentialAuthorityTreasurySessionKeys {
	if generate == nil {
		generate = mintTreasurySessionKey
	}
	return &CredentialAuthorityTreasurySessionKeys{open: open, generate: generate}
}

func (s *CredentialAuthorityTreasurySessionKeys) ResolveOrMint(ctx context.Context, runID uuid.UUID) (string, error) {
	key, release, err := s.Acquire(ctx, runID)
	if release != nil {
		release()
	}
	return key, err
}

// Acquire returns the active key and a process-local issuance lease. The
// caller must retain the lease until the key has been delivered to its
// approved recipient (broker response write or runner process launch).
func (s *CredentialAuthorityTreasurySessionKeys) Acquire(ctx context.Context, runID uuid.UUID) (string, func(), error) {
	return s.acquire(ctx, runID, false)
}

// AcquireContinuation is only used by the explicit Agent Manager continuation
// path after its terminal→running transition. A tombstone authorizes no broker
// retrieval or resume; this path alone rotates to a fresh key for the new turn.
func (s *CredentialAuthorityTreasurySessionKeys) AcquireContinuation(ctx context.Context, runID uuid.UUID) (string, func(), error) {
	return s.acquire(ctx, runID, true)
}

func (s *CredentialAuthorityTreasurySessionKeys) acquire(ctx context.Context, runID uuid.UUID, continuation bool) (string, func(), error) {
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}
	if s == nil || s.open == nil || runID == uuid.Nil {
		return "", nil, fmt.Errorf("Treasury session-key authority and run ID are required")
	}
	releaseGate := treasurySessionKeyGate.readLock(runID)
	if err := ctx.Err(); err != nil {
		releaseGate()
		return "", nil, err
	}
	identity, err := credentialauthority.ParseIdentity(treasurySessionIdentityPrefix + runID.String())
	if err != nil {
		releaseGate()
		return "", nil, fmt.Errorf("parse run-scoped Treasury key identity: %w", err)
	}
	authority, err := s.open()
	if err != nil {
		releaseGate()
		return "", nil, fmt.Errorf("open credential authority for run-scoped Treasury key: %w", err)
	}
	treasurySessionKeyMu.Lock()
	defer treasurySessionKeyMu.Unlock()
	state, err := readTreasurySessionKeyState(authority, identity)
	if err != nil {
		releaseGate()
		return "", nil, err
	}
	if state.Revoked {
		if !continuation {
			releaseGate()
			return "", nil, fmt.Errorf("run-scoped Treasury key was revoked at terminal cleanup")
		}
		state.Key, err = s.generateKey()
		if err != nil {
			releaseGate()
			return "", nil, err
		}
		state.RotationRequest = ""
		state.Revoked = false
		if err = writeTreasurySessionKeyState(authority, identity, state); err != nil {
			releaseGate()
			return "", nil, fmt.Errorf("persist continued run Treasury session key: %w", err)
		}
	}
	if state.Key == "" {
		state.Key, err = s.generateKey()
		if err != nil {
			releaseGate()
			return "", nil, err
		}
		if err = writeTreasurySessionKeyState(authority, identity, state); err != nil {
			releaseGate()
			return "", nil, fmt.Errorf("persist run-scoped Treasury key: %w", err)
		}
	}
	value := state.Key
	value = strings.TrimSpace(value)
	if len(value) != 64 {
		releaseGate()
		return "", nil, fmt.Errorf("run-scoped Treasury key has invalid encoding")
	}
	if _, err := hex.DecodeString(value); err != nil {
		releaseGate()
		return "", nil, fmt.Errorf("run-scoped Treasury key has invalid encoding")
	}
	return value, releaseGate, nil
}

// Rotate replaces the only active run-held Treasury key in one credential
// field write. Replaying the same request ID returns that active key without
// retaining the retired private key.
func (s *CredentialAuthorityTreasurySessionKeys) Rotate(ctx context.Context, runID uuid.UUID, requestID string) (string, error) {
	key, release, err := s.AcquireRotation(ctx, runID, requestID)
	if release != nil {
		release()
	}
	return key, err
}

func (s *CredentialAuthorityTreasurySessionKeys) AcquireRotation(ctx context.Context, runID uuid.UUID, requestID string) (string, func(), error) {
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}
	requestID = strings.TrimSpace(requestID)
	if s == nil || s.open == nil || runID == uuid.Nil || requestID == "" || len(requestID) > 200 {
		return "", nil, fmt.Errorf("run ID, credential authority, and rotation request ID are required")
	}
	releaseGate := treasurySessionKeyGate.readLock(runID)
	if err := ctx.Err(); err != nil {
		releaseGate()
		return "", nil, err
	}
	identity, err := credentialauthority.ParseIdentity(treasurySessionIdentityPrefix + runID.String())
	if err != nil {
		releaseGate()
		return "", nil, fmt.Errorf("parse run-scoped Treasury key identity: %w", err)
	}
	authority, err := s.open()
	if err != nil {
		releaseGate()
		return "", nil, fmt.Errorf("open credential authority to rotate run-scoped Treasury key: %w", err)
	}
	treasurySessionKeyMu.Lock()
	defer treasurySessionKeyMu.Unlock()
	state, err := readTreasurySessionKeyState(authority, identity)
	if err != nil {
		releaseGate()
		return "", nil, err
	}
	if state.Revoked {
		releaseGate()
		return "", nil, fmt.Errorf("run-scoped Treasury key was revoked at terminal cleanup")
	}
	if state.Key != "" && state.RotationRequest == requestID {
		return state.Key, releaseGate, nil
	}
	state.Key, err = s.generateKey()
	if err != nil {
		releaseGate()
		return "", nil, err
	}
	state.RotationRequest = requestID
	if err := writeTreasurySessionKeyState(authority, identity, state); err != nil {
		releaseGate()
		return "", nil, fmt.Errorf("persist rotated run-scoped Treasury key: %w", err)
	}
	return state.Key, releaseGate, nil
}

func (s *CredentialAuthorityTreasurySessionKeys) generateKey() (string, error) {
	if s == nil || s.generate == nil {
		return "", fmt.Errorf("run-scoped Treasury key generator is unavailable")
	}
	key, err := s.generate()
	if err != nil {
		return "", err
	}
	key = strings.ToLower(strings.TrimSpace(key))
	if !validTreasurySessionKey(key) {
		return "", fmt.Errorf("generated run-scoped Treasury key has invalid encoding")
	}
	return key, nil
}

func mintTreasurySessionKey() (string, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("generate run-scoped Treasury key: %w", err)
	}
	return hex.EncodeToString(secret), nil
}

func readTreasurySessionKeyState(authority *credentialauthority.Authority, id credentialauthority.Identity) (treasurySessionKeyState, error) {
	value, err := authority.Resolve(id, treasurySessionKeyField)
	if errors.Is(err, credentialauthority.ErrUnconfigured) {
		return treasurySessionKeyState{Version: 1}, nil
	}
	if err != nil {
		return treasurySessionKeyState{}, fmt.Errorf("resolve run-scoped Treasury key: %w", err)
	}
	var state treasurySessionKeyState
	if json.Unmarshal([]byte(value), &state) == nil && state.Version == 1 && (validTreasurySessionKey(state.Key) || state.Revoked && state.Key == "") {
		return state, nil
	}
	if validTreasurySessionKey(value) {
		state = treasurySessionKeyState{Version: 1, Key: strings.ToLower(value)}
		if err := writeTreasurySessionKeyState(authority, id, state); err != nil {
			return treasurySessionKeyState{}, fmt.Errorf("migrate run-scoped Treasury key state: %w", err)
		}
		return state, nil
	}
	return treasurySessionKeyState{}, fmt.Errorf("run-scoped Treasury key state is invalid")
}

func writeTreasurySessionKeyState(authority *credentialauthority.Authority, id credentialauthority.Identity, state treasurySessionKeyState) error {
	encoded, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return authority.Put(id, treasurySessionKeyField, string(encoded))
}

func validTreasurySessionKey(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func (s *CredentialAuthorityTreasurySessionKeys) RevokeRunCredentialUse(ctx context.Context, runID uuid.UUID, _ string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.open == nil || runID == uuid.Nil {
		return fmt.Errorf("Treasury session-key authority and run ID are required")
	}
	identity, err := credentialauthority.ParseIdentity(treasurySessionIdentityPrefix + runID.String())
	if err != nil {
		return fmt.Errorf("parse run-scoped Treasury key identity: %w", err)
	}
	authority, err := s.open()
	if err != nil {
		return fmt.Errorf("open credential authority to revoke run-scoped Treasury key: %w", err)
	}
	treasurySessionKeyMu.Lock()
	defer treasurySessionKeyMu.Unlock()
	state := treasurySessionKeyState{Version: 1, Revoked: true}
	if err := writeTreasurySessionKeyState(authority, identity, state); err != nil {
		return fmt.Errorf("persist run-scoped Treasury key revocation: %w", err)
	}
	return nil
}
