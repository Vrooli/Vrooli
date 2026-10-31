package control

import (
	"context"
	"fmt"
	"io"
	"time"

	authdomain "device-control/internal/auth"
	"device-control/strategy"

	"github.com/google/uuid"
)

func (s *Service) AuthProfiles(ctx context.Context) []authdomain.Profile {
	if s.auth == nil {
		return []authdomain.Profile{}
	}
	return s.auth.List(ctx)
}

func (s *Service) CreateAuthProfile(ctx context.Context, profile authdomain.Profile, actor string) (authdomain.Profile, error) {
	if s.auth == nil {
		return authdomain.Profile{}, fmt.Errorf("authentication profile store is unavailable")
	}
	created, err := s.auth.Create(ctx, profile)
	if err == nil {
		s.recordAuthAudit(ctx, actor, created.DeviceID, "auth_profile_create", "success", "", nil)
	}
	return created, err
}

func (s *Service) UpdateAuthProfile(ctx context.Context, profileID string, profile authdomain.Profile, actor string) (authdomain.Profile, error) {
	if s.auth == nil {
		return authdomain.Profile{}, fmt.Errorf("authentication profile store is unavailable")
	}
	updated, err := s.auth.Update(ctx, profileID, profile)
	if err == nil {
		s.recordAuthAudit(ctx, actor, updated.DeviceID, "auth_profile_update", "success", "", nil)
	}
	return updated, err
}

func (s *Service) ProvisionAuthCredential(ctx context.Context, profileID string, input io.Reader) (authdomain.ProviderStatus, error) {
	if s.auth == nil {
		return authdomain.ProviderStatus{}, fmt.Errorf("authentication profile store is unavailable")
	}
	return s.auth.Provision(ctx, profileID, input)
}

func (s *Service) DeleteAuthCredential(ctx context.Context, profileID, actor string) (authdomain.ProviderStatus, error) {
	if s.auth == nil {
		return authdomain.ProviderStatus{}, fmt.Errorf("authentication profile store is unavailable")
	}
	status, err := s.auth.DeleteCredential(ctx, profileID)
	if err == nil {
		profile, _ := s.auth.Get(ctx, profileID)
		s.recordAuthAudit(ctx, actor, profile.DeviceID, "auth_credential_delete", "success", "", nil)
	}
	return status, err
}

func (s *Service) RevokeAuthProfile(ctx context.Context, profileID, actor string) (authdomain.Profile, error) {
	if s.auth == nil {
		return authdomain.Profile{}, fmt.Errorf("authentication profile store is unavailable")
	}
	p, err := s.auth.Revoke(ctx, profileID)
	if err == nil {
		s.recordAuthAudit(ctx, actor, p.DeviceID, "auth_profile_revoke", "success", "", nil)
	}
	return p, err
}

func (s *Service) AuthProfileStatus(ctx context.Context, profileID string) (authdomain.Profile, authdomain.ProviderStatus, error) {
	if s.auth == nil {
		return authdomain.Profile{}, authdomain.ProviderStatus{}, fmt.Errorf("authentication profile store is unavailable")
	}
	p, ok := s.auth.Get(ctx, profileID)
	if !ok {
		return authdomain.Profile{}, authdomain.ProviderStatus{}, fmt.Errorf("authentication profile %q not found", profileID)
	}
	return p, s.auth.ProviderStatus(ctx, p), nil
}

func (s *Service) UnlockDevice(ctx context.Context, profileID, deviceID, actor, leaseToken string) (authdomain.UnlockResponse, error) {
	if leaseToken == "" {
		return authdomain.UnlockResponse{}, fmt.Errorf("unlock requires an active device lease")
	}
	sess, err := s.sessionForLease(ctx, deviceID, leaseToken)
	if err != nil {
		return authdomain.UnlockResponse{}, err
	}
	// Authentication is a device operation, so it must use the device's
	// effective transport. Flow execution intentionally defaults to USB for
	// release-grade compatibility, but that default is unsafe here: a
	// promoted wireless device would be probed through an unscoped USB adapter
	// and report an unknown state even while the wireless state endpoint works.
	transport := ""
	if record, found := s.devices.Get(deviceID); found {
		transport = record.Transport
	}
	adapter, ok := s.strategyForFlow(deviceID, transport)
	if !ok {
		return authdomain.UnlockResponse{}, fmt.Errorf("unknown or unavailable device %q", deviceID)
	}
	reader, ok := adapter.(strategy.StateReader)
	if !ok {
		return authdomain.UnlockResponse{}, fmt.Errorf("strategy does not expose live device state")
	}
	state, err := reader.ReadState(ctx)
	if err != nil {
		return authdomain.UnlockResponse{ProfileID: profileID, DeviceID: deviceID, Outcome: authdomain.OutcomeUnknownState, NextAction: "re-probe the device state"}, err
	}
	unlocker, ok := adapter.(strategy.Unlocker)
	if !ok {
		return authdomain.UnlockResponse{ProfileID: profileID, DeviceID: deviceID, Outcome: authdomain.OutcomeUnsupported, NextAction: "use a strategy with an unlock adapter"}, nil
	}
	response, unlockErr := s.auth.Unlock(ctx, profileID, deviceID, state, unlocker, reader.ReadState)
	outcome := response.Outcome
	if unlockErr != nil {
		outcome = authdomain.OutcomeTransport
	}
	s.auth.RecordOutcome(ctx, profileID, outcome)
	s.recordAuthAudit(ctx, actor, deviceID, "device_unlock", outcome, sess.ID, &response)
	return response, unlockErr
}

// ProviderStatus is kept on the store so handlers never need a credential
// value or an authority implementation detail.
func (s *Service) recordAuthAudit(ctx context.Context, actor, deviceID, verb, outcome, leaseID string, unlock *authdomain.UnlockResponse) {
	record := Audit{ID: uuid.NewString(), Actor: actor, DeviceID: deviceID, LeaseID: leaseID, Verb: verb, Outcome: outcome, CreatedAt: time.Now().UTC(), RedactionVerified: true}
	if unlock != nil {
		record.ProfileID = unlock.ProfileID
		record.Method = unlock.Method
		record.Attempts = unlock.Attempts
		record.ProviderState = unlock.ProviderState
		record.BeforeLockState = unlock.BeforeLockState
		record.AfterLockState = unlock.AfterLockState
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		_, _ = s.db.ExecContext(ctx, `INSERT INTO device_control_audits (id, actor, device_id, lease_id, verb, outcome, created_at, redaction_verified, redaction_opted_out, profile_id, method, attempts, provider_state, before_lock_state, after_lock_state) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, record.ID, record.Actor, record.DeviceID, record.LeaseID, record.Verb, record.Outcome, record.CreatedAt.Format(time.RFC3339Nano), 1, 0, record.ProfileID, record.Method, record.Attempts, record.ProviderState, record.BeforeLockState, record.AfterLockState)
	}
	s.audits = append(s.audits, record)
}
