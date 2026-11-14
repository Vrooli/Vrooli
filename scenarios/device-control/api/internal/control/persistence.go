package control

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	devicedomain "device-control/internal/devices"
	identitydomain "device-control/internal/identity"
	"device-control/strategy"
)

type transportState struct {
	DeviceID    string
	Serial      string
	StrategyID  string
	Transport   string
	Endpoint    string
	ProfileJSON string
	UpdatedAt   time.Time
}

func (s *Service) loadTransportStates() error {
	if s.db == nil {
		return nil
	}
	profileRows, err := s.db.QueryContext(context.Background(), `SELECT device_id, serial, strategy_id, transport, endpoint, updated_at, profile_json FROM device_control_transport_profiles`)
	if err != nil {
		return fmt.Errorf("load device transport profiles: %w", err)
	}
	for profileRows.Next() {
		var state transportState
		var updated string
		if err := profileRows.Scan(&state.DeviceID, &state.Serial, &state.StrategyID, &state.Transport, &state.Endpoint, &updated, &state.ProfileJSON); err != nil {
			profileRows.Close()
			return fmt.Errorf("read device transport profile: %w", err)
		}
		state.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		s.transportProfiles[transportProfileKey(state)] = state
		if state.DeviceID != "" {
			s.devices.UpsertIdentity(devicedomain.Record{ID: state.DeviceID, IdentityKey: persistedIdentityKey(state), IdentityKind: persistedIdentityKind(state), Kind: "physical", Serial: state.Serial, StrategyID: state.StrategyID, Transport: state.Transport, Endpoint: state.Endpoint, Transports: []strategy.DeviceTransport{restoredTransportProfile(state)}, Status: strategy.HealthUnreachable, Health: strategy.HealthUnreachable, HealthReason: "restored transport has not been probed yet"})
		}
	}
	profileErr := profileRows.Err()
	profileRows.Close()
	if profileErr != nil {
		return profileErr
	}
	rows, err := s.db.QueryContext(context.Background(), `SELECT device_id, serial, strategy_id, transport, endpoint, updated_at FROM device_control_transports`)
	if err != nil {
		return fmt.Errorf("load device transports: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var state transportState
		var updated string
		if err := rows.Scan(&state.DeviceID, &state.Serial, &state.StrategyID, &state.Transport, &state.Endpoint, &updated); err != nil {
			return fmt.Errorf("read device transport: %w", err)
		}
		state.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		s.transportStates[state.DeviceID] = state
		s.transportProfiles[transportProfileKey(state)] = state
		if state.Transport == "wireless" {
			s.devices.UpsertIdentity(devicedomain.Record{ID: state.DeviceID, IdentityKey: persistedIdentityKey(state), IdentityKind: persistedIdentityKind(state), Kind: "physical", Serial: state.Serial, StrategyID: state.StrategyID, Transport: state.Transport, Endpoint: state.Endpoint, Status: strategy.HealthUnreachable, Health: strategy.HealthUnreachable, HealthReason: "restored wireless transport has not been probed yet"})
		}
	}
	return rows.Err()
}

func persistedIdentityKey(state transportState) string {
	if state.StrategyID == "android-tv-remote" || state.StrategyID == "google-cast" {
		return ""
	}
	return state.Serial
}

func persistedIdentityKind(state transportState) string {
	switch state.StrategyID {
	case "android-adb":
		return string(identitydomain.ADBSerial)
	case "android-tv-remote":
		return string(identitydomain.BluetoothMAC)
	case "google-cast":
		return string(identitydomain.CastID)
	default:
		return ""
	}
}

func restoredTransportProfile(state transportState) strategy.DeviceTransport {
	profile := strategy.DeviceTransport{StrategyID: state.StrategyID, Name: state.Transport, Endpoint: state.Endpoint, Health: strategy.HealthUnreachable, HealthReason: "restored transport has not been probed yet", ObservedAt: state.UpdatedAt}
	if strings.TrimSpace(state.ProfileJSON) != "" {
		if err := json.Unmarshal([]byte(state.ProfileJSON), &profile); err == nil {
			profile.StrategyID = firstNonEmpty(profile.StrategyID, state.StrategyID)
			profile.Name = firstNonEmpty(profile.Name, state.Transport)
			profile.Endpoint = firstNonEmpty(profile.Endpoint, state.Endpoint)
		}
	}
	// Persisted metadata is useful for planning after restart, but reachability
	// must still be re-probed before a write is allowed.
	profile.Health = strategy.HealthUnreachable
	profile.HealthReason = "restored transport has not been probed yet"
	return profile
}

func (s *Service) loadIdentityClaims() error {
	if s.db == nil {
		return nil
	}
	rows, err := s.db.QueryContext(context.Background(), `SELECT device_id, kind, value, strategy_id, evidence FROM device_control_identity_claims`)
	if err != nil {
		return fmt.Errorf("load device identity claims: %w", err)
	}
	defer rows.Close()
	grouped := map[string][]identitydomain.IdentityClaim{}
	for rows.Next() {
		var deviceID, kind, value, strategyID, evidence string
		if err := rows.Scan(&deviceID, &kind, &value, &strategyID, &evidence); err != nil {
			return fmt.Errorf("read device identity claim: %w", err)
		}
		claim, err := identitydomain.NewClaim(kind, value, strategyID, evidence)
		if err != nil {
			continue
		}
		grouped[deviceID] = append(grouped[deviceID], claim)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for deviceID, claims := range grouped {
		record, ok := s.devices.Get(deviceID)
		if !ok {
			record = devicedomain.Record{ID: deviceID, Kind: "physical", Status: strategy.HealthUnreachable, Health: strategy.HealthUnreachable, HealthReason: "identity claims restored before transport probe"}
		}
		record.Claims = claims
		if record.IdentityKey == "" {
			record.IdentityKey = claims[0].Value
			record.IdentityKind = string(claims[0].Kind)
		}
		s.devices.Upsert(record)
	}
	return nil
}

func (s *Service) loadIdentityMerges() error {
	if s.db == nil {
		return nil
	}
	rows, err := s.db.QueryContext(context.Background(), `SELECT canonical_id, member_id, claim_kind, claim_value, claim_strategy_id, claim_evidence, canonical_snapshot, member_snapshot FROM device_control_identity_merges`)
	if err != nil {
		return fmt.Errorf("load device identity merges: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var canonicalID, memberID, kind, value, strategyID, evidence, canonicalJSON, memberJSON string
		if err := rows.Scan(&canonicalID, &memberID, &kind, &value, &strategyID, &evidence, &canonicalJSON, &memberJSON); err != nil {
			return fmt.Errorf("read device identity merge: %w", err)
		}
		var canonical, member devicedomain.Record
		if err := json.Unmarshal([]byte(canonicalJSON), &canonical); err != nil {
			return fmt.Errorf("decode canonical identity snapshot: %w", err)
		}
		if err := json.Unmarshal([]byte(memberJSON), &member); err != nil {
			return fmt.Errorf("decode member identity snapshot: %w", err)
		}
		claim, err := identitydomain.NewClaim(kind, value, strategyID, evidence)
		if err != nil {
			return err
		}
		s.devices.RestoreMerge(canonicalID, devicedomain.MergeSnapshot{CanonicalBefore: canonical, Members: map[string]devicedomain.Record{memberID: member}, Claim: claim})
	}
	return rows.Err()
}

func (s *Service) loadIdentityAliases() error {
	if s.db == nil {
		return nil
	}
	rows, err := s.db.QueryContext(context.Background(), `SELECT canonical_id, alias_id FROM device_control_identity_aliases`)
	if err != nil {
		return fmt.Errorf("load device identity aliases: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var canonicalID, aliasID string
		if err := rows.Scan(&canonicalID, &aliasID); err != nil {
			return err
		}
		s.addAuditAlias(canonicalID, aliasID)
	}
	return rows.Err()
}

func (s *Service) persistIdentityClaims(ctx context.Context, record devicedomain.Record) error {
	if s.db == nil {
		return nil
	}
	for _, claim := range record.Claims {
		if err := identitydomain.ValidateClaim(claim); err != nil {
			continue
		}
		if _, err := s.db.ExecContext(ctx, `INSERT INTO device_control_identity_claims (device_id, kind, value, strategy_id, evidence) VALUES (?, ?, ?, ?, ?) ON CONFLICT(device_id, kind, value) DO UPDATE SET strategy_id=excluded.strategy_id, evidence=excluded.evidence`, record.ID, claim.Kind, claim.Value, claim.StrategyID, claim.Evidence); err != nil {
			return fmt.Errorf("persist device identity claim: %w", err)
		}
	}
	return nil
}

func (s *Service) persistIdentityMerge(ctx context.Context, canonicalID string, snapshot devicedomain.MergeSnapshot) error {
	if s.db == nil {
		return nil
	}
	canonicalJSON, err := json.Marshal(snapshot.CanonicalBefore)
	if err != nil {
		return err
	}
	for memberID, member := range snapshot.Members {
		memberJSON, marshalErr := json.Marshal(member)
		if marshalErr != nil {
			return marshalErr
		}
		claim := snapshot.Claim
		if _, execErr := s.db.ExecContext(ctx, `INSERT INTO device_control_identity_merges (canonical_id, member_id, claim_kind, claim_value, claim_strategy_id, claim_evidence, canonical_snapshot, member_snapshot, merged_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(canonical_id, member_id) DO UPDATE SET claim_kind=excluded.claim_kind, claim_value=excluded.claim_value, claim_strategy_id=excluded.claim_strategy_id, claim_evidence=excluded.claim_evidence, canonical_snapshot=excluded.canonical_snapshot, member_snapshot=excluded.member_snapshot, merged_at=excluded.merged_at`, canonicalID, memberID, claim.Kind, claim.Value, claim.StrategyID, claim.Evidence, string(canonicalJSON), string(memberJSON), time.Now().UTC().Format(time.RFC3339Nano)); execErr != nil {
			return execErr
		}
	}
	return nil
}

func (s *Service) persistIdentityAlias(ctx context.Context, canonicalID, aliasID string) error {
	s.addAuditAlias(canonicalID, aliasID)
	if s.db == nil {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO device_control_identity_aliases (canonical_id, alias_id, created_at) VALUES (?, ?, ?) ON CONFLICT(canonical_id, alias_id) DO NOTHING`, canonicalID, aliasID, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Service) restoreTransportStrategies() {
	for id, state := range s.transportStates {
		if state.Transport != "wireless" || state.Endpoint == "" {
			continue
		}
		base, ok := s.registry.Get(state.StrategyID)
		if !ok {
			continue
		}
		if scoped, ok := base.(strategy.DeviceScoped); ok && state.Serial != "" {
			base = scoped.ForDevice(state.Serial)
		}
		restorer, ok := base.(interface {
			RestoreWireless(string) strategy.Strategy
		})
		if !ok {
			continue
		}
		s.transportStrategies[id] = restorer.RestoreWireless(state.Endpoint)
	}
}

func (s *Service) persistTransportState(ctx context.Context, state transportState) error {
	if s.db == nil {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO device_control_transport_profiles (device_id, serial, strategy_id, transport, endpoint, updated_at) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(device_id, strategy_id, transport) DO UPDATE SET serial=excluded.serial, endpoint=excluded.endpoint, updated_at=excluded.updated_at`, state.DeviceID, state.Serial, state.StrategyID, state.Transport, state.Endpoint, state.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("persist device transport: %w", err)
	}
	// Keep the legacy selected-transport row for older local databases and
	// wireless strategy restoration; the profile table above is the durable
	// many-transport source of truth.
	_, err = s.db.ExecContext(ctx, `INSERT INTO device_control_transports (device_id, serial, strategy_id, transport, endpoint, updated_at) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(device_id) DO UPDATE SET serial=excluded.serial, strategy_id=excluded.strategy_id, transport=excluded.transport, endpoint=excluded.endpoint, updated_at=excluded.updated_at`, state.DeviceID, state.Serial, state.StrategyID, state.Transport, state.Endpoint, state.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("persist selected device transport: %w", err)
	}
	s.transportProfiles[transportProfileKey(state)] = state
	return nil
}

// persistObservedTransportProfiles records every transport discovered during
// inventory without changing the selected-transport row. A device can expose
// several independently reachable strategies at once, so writing only the
// selected transport would make the identity merge disappear on restart.
func (s *Service) persistObservedTransportProfiles(ctx context.Context, record devicedomain.Record) error {
	if s.db == nil {
		return nil
	}
	for _, profile := range record.Transports {
		strategyID := strings.TrimSpace(profile.StrategyID)
		name := strings.TrimSpace(profile.Name)
		if strategyID == "" && name == "" {
			continue
		}
		if strategyID == "" {
			strategyID = strings.TrimSpace(record.StrategyID)
		}
		if name == "" {
			name = strategyID
		}
		observedAt := profile.ObservedAt
		if observedAt.IsZero() {
			observedAt = record.ObservedAt
		}
		if observedAt.IsZero() {
			observedAt = time.Now().UTC()
		}
		profileJSON, err := json.Marshal(profile)
		if err != nil {
			return fmt.Errorf("encode observed device transport: %w", err)
		}
		state := transportState{
			DeviceID:    record.ID,
			Serial:      record.Serial,
			StrategyID:  strategyID,
			Transport:   name,
			Endpoint:    profile.Endpoint,
			ProfileJSON: string(profileJSON),
			UpdatedAt:   observedAt,
		}
		if _, err := s.db.ExecContext(ctx, `INSERT INTO device_control_transport_profiles (device_id, serial, strategy_id, transport, endpoint, updated_at, profile_json) VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT(device_id, strategy_id, transport) DO UPDATE SET serial=excluded.serial, endpoint=excluded.endpoint, updated_at=excluded.updated_at, profile_json=excluded.profile_json`, state.DeviceID, state.Serial, state.StrategyID, state.Transport, state.Endpoint, state.UpdatedAt.Format(time.RFC3339Nano), state.ProfileJSON); err != nil {
			return fmt.Errorf("persist observed device transport: %w", err)
		}
		s.transportProfiles[transportProfileKey(state)] = state
	}
	return nil
}

func transportProfileKey(state transportState) string {
	return state.DeviceID + "\x00" + state.StrategyID + "\x00" + state.Transport
}
