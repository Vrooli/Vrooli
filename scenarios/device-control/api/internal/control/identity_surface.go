package control

import (
	"context"
	"fmt"
	"strings"

	devicedomain "device-control/internal/devices"
	identitydomain "device-control/internal/identity"
)

// MergeDevices records an explicit owner assertion before combining two
// identities. Automatic inventory reconciliation remains stricter: it only
// merges observations that already share an accepted hardware claim.
func (s *Service) MergeDevices(ctx context.Context, canonicalID, memberID, claimSpec string) (Device, error) {
	canonicalID = strings.TrimSpace(canonicalID)
	memberID = strings.TrimSpace(memberID)
	kind, value, ok := strings.Cut(strings.TrimSpace(claimSpec), "=")
	if !ok || strings.TrimSpace(kind) == "" || strings.TrimSpace(value) == "" {
		return Device{}, fmt.Errorf("claim must have the form kind=value")
	}
	claim, err := identitydomain.NewClaim(kind, value, "device-control", "owner-asserted")
	if err != nil {
		return Device{}, err
	}
	merged, err := s.devices.Merge(canonicalID, memberID, claim)
	if err != nil {
		return Device{}, err
	}
	snapshot, ok := s.devices.MergeSnapshot(canonicalID)
	if !ok {
		return Device{}, fmt.Errorf("merge history was not retained for %q", canonicalID)
	}
	if err := s.persistIdentityMerge(ctx, canonicalID, snapshot); err != nil {
		return Device{}, err
	}
	if err := s.persistIdentityAlias(ctx, canonicalID, memberID); err != nil {
		return Device{}, err
	}
	if err := s.persistIdentityClaims(ctx, merged); err != nil {
		return Device{}, err
	}
	_ = s.persistObservedTransportProfiles(ctx, merged)
	if s.db != nil {
		_, _ = s.db.ExecContext(ctx, `DELETE FROM device_control_identity_claims WHERE device_id = ?`, memberID)
		_, _ = s.db.ExecContext(ctx, `DELETE FROM device_control_transport_profiles WHERE device_id = ?`, memberID)
		_, _ = s.db.ExecContext(ctx, `DELETE FROM device_control_transports WHERE device_id = ?`, memberID)
	}
	delete(s.transportStates, memberID)
	for key, state := range s.transportProfiles {
		if state.DeviceID == memberID {
			delete(s.transportProfiles, key)
		}
	}
	return deviceFromRecord(merged), nil
}

// SplitDevice restores the snapshots captured by the most recent merge.
// Historical audit aliases remain durable, so pre-merge records are visible
// from both identities after the split.
func (s *Service) SplitDevice(ctx context.Context, canonicalID string) ([]Device, error) {
	canonicalID = strings.TrimSpace(canonicalID)
	records, err := s.devices.Split(canonicalID)
	if err != nil {
		return nil, err
	}
	if s.db != nil {
		_, _ = s.db.ExecContext(ctx, `DELETE FROM device_control_identity_merges WHERE canonical_id = ?`, canonicalID)
		_, _ = s.db.ExecContext(ctx, `DELETE FROM device_control_identity_claims WHERE device_id = ?`, canonicalID)
		_, _ = s.db.ExecContext(ctx, `DELETE FROM device_control_transport_profiles WHERE device_id = ?`, canonicalID)
		_, _ = s.db.ExecContext(ctx, `DELETE FROM device_control_transports WHERE device_id = ?`, canonicalID)
	}
	for _, record := range records {
		if err := s.persistIdentityClaims(ctx, record); err != nil {
			return nil, err
		}
		_ = s.persistObservedTransportProfiles(ctx, record)
	}
	// Keep the alias relation alive after split for historical audit lookup.
	s.addAuditAlias(canonicalID, canonicalID)
	return devicesFromRecords(records), nil
}

func devicesFromRecords(records []devicedomain.Record) []Device {
	devices := make([]Device, 0, len(records))
	for _, record := range records {
		devices = append(devices, deviceFromRecord(record))
	}
	return devices
}

// AuditForDevice resolves historical merge aliases in addition to the current
// device id. This preserves audit reachability without rewriting immutable
// audit records when identities are merged or split.
func (s *Service) AuditForDevice(ctx context.Context, deviceID string) []Audit {
	ids := s.auditIDs(ctx, strings.TrimSpace(deviceID))
	if len(ids) == 0 {
		ids[deviceID] = struct{}{}
	}
	all := s.AuditContext(ctx)
	out := make([]Audit, 0)
	for _, record := range all {
		if _, ok := ids[record.DeviceID]; ok {
			out = append(out, record)
		}
	}
	return out
}

func (s *Service) auditIDs(ctx context.Context, deviceID string) map[string]struct{} {
	ids := map[string]struct{}{deviceID: {}}
	if s.db != nil {
		rows, err := s.db.QueryContext(ctx, `SELECT canonical_id, alias_id FROM device_control_identity_aliases WHERE canonical_id = ? OR alias_id = ?`, deviceID, deviceID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var canonical, alias string
				if rows.Scan(&canonical, &alias) == nil {
					ids[canonical] = struct{}{}
					ids[alias] = struct{}{}
				}
			}
			return ids
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for canonical, aliases := range s.auditAliases {
		if canonical == deviceID {
			for alias := range aliases {
				ids[alias] = struct{}{}
			}
			continue
		}
		if _, ok := aliases[deviceID]; ok {
			ids[canonical] = struct{}{}
			for alias := range aliases {
				ids[alias] = struct{}{}
			}
		}
	}
	return ids
}
