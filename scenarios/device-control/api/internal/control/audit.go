package control

import (
	"context"
	"encoding/json"
	"time"
)

func (s *Service) addAuditAlias(canonicalID, aliasID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.auditAliases == nil {
		s.auditAliases = map[string]map[string]struct{}{}
	}
	if s.auditAliases[canonicalID] == nil {
		s.auditAliases[canonicalID] = map[string]struct{}{}
	}
	s.auditAliases[canonicalID][canonicalID] = struct{}{}
	s.auditAliases[canonicalID][aliasID] = struct{}{}
}

func (s *Service) Audit() []Audit {
	return s.AuditContext(context.Background())
}

func (s *Service) AuditContext(ctx context.Context) []Audit {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		rows, err := s.db.QueryContext(ctx, `SELECT id, actor, device_id, transport, causation_id, operation_id, lease_id, verb, outcome, profile_id, method, attempts, provider_state, before_lock_state, after_lock_state, created_at, redaction_verified, redaction_opted_out, interactive, evidence_backed, details_json FROM device_control_audits ORDER BY created_at DESC`)
		if err == nil {
			defer rows.Close()
			out := make([]Audit, 0)
			for rows.Next() {
				var v Audit
				var created, details string
				var verified, optedOut, interactive, evidenceBacked int
				if err := rows.Scan(&v.ID, &v.Actor, &v.DeviceID, &v.Transport, &v.CausationID, &v.OperationID, &v.LeaseID, &v.Verb, &v.Outcome, &v.ProfileID, &v.Method, &v.Attempts, &v.ProviderState, &v.BeforeLockState, &v.AfterLockState, &created, &verified, &optedOut, &interactive, &evidenceBacked, &details); err != nil {
					continue
				}
				if details != "" {
					_ = json.Unmarshal([]byte(details), &v.Details)
				}
				v.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
				v.RedactionVerified = verified != 0
				v.RedactionOptedOut = optedOut != 0
				v.Interactive = interactive != 0
				v.EvidenceBacked = evidenceBacked != 0
				out = append(out, v)
			}
			return out
		}
	}
	return append([]Audit{}, s.audits...)
}
