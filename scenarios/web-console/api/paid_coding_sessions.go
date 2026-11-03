// Responsibility: scoped coding credentials for native terminal lifetimes.
package main

import (
	"context"
	"sync"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

type paidCodingBinding struct {
	authority   codingAuthority
	device      codingKey
	key         codingKey
	environment map[string]string
	sessionOnly bool
}

// Only LPBS-issued, expiring session credentials reach the child. Device
// credentials stay in Web Console and are revoked together with the session.
type paidCodingSessionRegistry struct {
	mu       sync.Mutex
	sessions map[string]paidCodingBinding
}

func newPaidCodingSessionRegistry() *paidCodingSessionRegistry {
	return &paidCodingSessionRegistry{sessions: make(map[string]paidCodingBinding)}
}

var activePaidCodingSessions = newPaidCodingSessionRegistry()

func (r *paidCodingSessionRegistry) Register(id string, binding paidCodingBinding) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[id] = binding
}

func (r *paidCodingSessionRegistry) Environment(id string) map[string]string {
	r.mu.Lock()
	defer r.mu.Unlock()
	binding, ok := r.sessions[id]
	if !ok || binding.key.ExpiresAt <= time.Now().Unix() {
		return nil
	}
	result := make(map[string]string, len(binding.environment))
	for key, value := range binding.environment {
		result[key] = value
	}
	return result
}

func (r *paidCodingSessionRegistry) RemoveSession(id string) {
	r.mu.Lock()
	binding, ok := r.sessions[id]
	delete(r.sessions, id)
	r.mu.Unlock()
	if ok {
		binding.revoke()
	}
}

func (r *paidCodingSessionRegistry) Clear() {
	r.mu.Lock()
	bindings := r.sessions
	r.sessions = make(map[string]paidCodingBinding)
	r.mu.Unlock()
	for _, binding := range bindings {
		binding.revoke()
	}
}

func (b paidCodingBinding) revoke() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Revoking the parent also kills every issued child at authentication.
	id := b.device.ID
	if b.sessionOnly {
		id = b.key.ID
	}
	_ = b.authority.revoke(ctx, b.device.Key, id)
}

func (s *Server) preparePaidCodingSession(ctx context.Context, account, harness, model string, cap int64) (func(string), func(), error) {
	var authority codingAuthority
	var device codingKey
	var err error
	var expires time.Time
	sessionOnly := false
	if launch, ok := ctx.Value(codingBackupContextKey{}).(codingBackupLaunch); ok && launch.Grant != nil {
		// Resolve and issue against the current consent while revocation and
		// replacement are excluded; the launch context can hold an older grant.
		s.codingBackupOperations.Lock()
		defer s.codingBackupOperations.Unlock()
		grant := s.backupGrant(ctx, harness)
		if grant == nil || grant.Device.ID != launch.Grant.Device.ID || !grant.Consent.Valid(time.Now()) || grant.Consent.Harness != harness || grant.Consent.Account != account || grant.Consent.Model != model || grant.Consent.MaxCredits != cap {
			return nil, nil, errBackupExpired
		}
		authority, device, expires = grant.Authority, grant.Device, grant.Consent.ExpiresAt
		sessionOnly = true
	} else {
		authority, device, err = s.codingAuthorityDevice(ctx, account, cap)
		if err != nil {
			return nil, nil, err
		}
	}
	binding := paidCodingBinding{authority: authority, device: device, sessionOnly: sessionOnly}
	record := cliutil.CodingBackupRecord{Consent: cliutil.CodingBackupConsent{Harness: harness, Account: account, Model: model, MaxCredits: cap, ExpiresAt: expires}, Owner: "explicit-session", BaseURL: authority.BaseURL, ParentID: device.ID, ParentKey: device.Key}
	if expires.IsZero() {
		record.Consent.ExpiresAt = time.Now().Add(time.Hour)
	}
	child, err := cliutil.IssueCodingBackupSession(ctx, record, record.Owner, authority.Client)
	if err != nil {
		if !sessionOnly {
			binding.revoke()
		}
		return nil, nil, err
	}
	binding.key = codingKey{ID: child.ID, Key: child.Environment[child.Source.SecretEnv], ExpiresAt: child.ExpiresAt}
	binding.environment = child.Environment
	return func(id string) { activePaidCodingSessions.Register(id, binding) }, binding.revoke, nil
}
