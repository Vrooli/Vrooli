package agentharness

import (
	"time"

	"github.com/vrooli/cli-core/guardmodes"
)

type HardStopClass string

const (
	HardStopMoney          HardStopClass = HardStopClass(guardmodes.HardStopMoney)
	HardStopOutsideMessage HardStopClass = HardStopClass(guardmodes.HardStopOutsideMessage)
	HardStopSecret         HardStopClass = HardStopClass(guardmodes.HardStopSecret)
	HardStopOutsideDelete  HardStopClass = HardStopClass(guardmodes.HardStopOutsideDelete)
	HardStopUnknownEffect  HardStopClass = HardStopClass(guardmodes.HardStopUnknownEffect)
)

type GuardMode struct {
	Check           RiskClass `json:"check"`
	Mode            string    `json:"mode"`
	EnabledAt       time.Time `json:"enabled_at"`
	ExpiresAt       time.Time `json:"expires_at"`
	Actor           string    `json:"actor"`
	Reason          string    `json:"reason"`
	AuthorizationID string    `json:"authorization_id,omitempty"`
}

type GuardModeAudit struct {
	At              time.Time      `json:"at"`
	Action          string         `json:"action"`
	Check           RiskClass      `json:"check,omitempty"`
	Checks          []RiskClass    `json:"checks,omitempty"`
	OriginalAction  DecisionAction `json:"original_action,omitempty"`
	ExpiresAt       time.Time      `json:"expires_at,omitempty"`
	Actor           string         `json:"actor,omitempty"`
	Reason          string         `json:"reason,omitempty"`
	AuthorizationID string         `json:"authorization_id,omitempty"`
	HardStop        HardStopClass  `json:"hard_stop,omitempty"`
	EventID         string         `json:"event_id,omitempty"`
	RestoredChecks  []RiskClass    `json:"restored_checks,omitempty"`
}

// GuardModeStore is a runtime-typed adapter over cli-core's shared durable
// guard mode contract. Root CLI and hooks use the same files and locking.
type GuardModeStore struct {
	Dir string
	Now func() time.Time
}

func NewGuardModeStore(dir string) *GuardModeStore {
	return &GuardModeStore{Dir: dir, Now: time.Now}
}

func (s *GuardModeStore) core() *guardmodes.Store {
	if s == nil {
		return nil
	}
	store := guardmodes.NewStore(s.Dir)
	store.Now = s.Now
	return store
}

func (s *GuardModeStore) SetLogOnly(check RiskClass, expiresAt time.Time, reason, actor string) error {
	return s.core().SetLogOnly(string(check), expiresAt, reason, actor)
}

// SetLogOnlyFromAnswer is for the steward answer path only. The caller must
// validate the exact answered structural ask before passing its stable ID.
func (s *GuardModeStore) SetLogOnlyFromAnswer(check RiskClass, expiresAt time.Time, reason, actor, answerID string) error {
	return s.core().SetLogOnlyFromAnswer(string(check), expiresAt, reason, actor, answerID)
}

func (s *GuardModeStore) Disable(check RiskClass, actor, reason string) error {
	return s.core().Disable(string(check), actor, reason)
}

func (s *GuardModeStore) Active(now time.Time) ([]GuardMode, error) {
	modes, err := s.core().Active(now)
	if err != nil {
		return nil, err
	}
	out := make([]GuardMode, len(modes))
	for i, mode := range modes {
		out[i] = GuardMode{Check: RiskClass(mode.Check), Mode: mode.Mode, EnabledAt: mode.EnabledAt, ExpiresAt: mode.ExpiresAt, Actor: mode.Actor, Reason: mode.Reason, AuthorizationID: mode.AuthorizationID}
	}
	return out, nil
}

func (s *GuardModeStore) RecordSuppressedAction(check RiskClass, eventID string, original DecisionAction, now time.Time) (GuardMode, bool, error) {
	mode, active, err := s.core().RecordSuppressedAction(string(check), eventID, string(original), now)
	return GuardMode{Check: RiskClass(mode.Check), Mode: mode.Mode, EnabledAt: mode.EnabledAt, ExpiresAt: mode.ExpiresAt, Actor: mode.Actor, Reason: mode.Reason, AuthorizationID: mode.AuthorizationID}, active, err
}

func (s *GuardModeStore) Audit() ([]GuardModeAudit, error) {
	entries, err := s.core().Audit()
	if err != nil {
		return nil, err
	}
	out := make([]GuardModeAudit, len(entries))
	for i, entry := range entries {
		out[i] = GuardModeAudit{At: entry.At, Action: entry.Action, Check: RiskClass(entry.Check), OriginalAction: DecisionAction(entry.OriginalAction), ExpiresAt: entry.ExpiresAt, Actor: entry.Actor, Reason: entry.Reason, AuthorizationID: entry.AuthorizationID, HardStop: HardStopClass(entry.HardStop), EventID: entry.EventID}
		for _, check := range entry.Checks {
			out[i].Checks = append(out[i].Checks, RiskClass(check))
		}
		for _, check := range entry.RestoredChecks {
			out[i].RestoredChecks = append(out[i].RestoredChecks, RiskClass(check))
		}
	}
	return out, nil
}

func (s *GuardModeStore) RestoreAll(hardStop HardStopClass, eventID string, now time.Time) ([]RiskClass, error) {
	checks, err := s.core().RestoreAll(guardmodes.HardStop(hardStop), eventID, now)
	out := make([]RiskClass, len(checks))
	for i, check := range checks {
		out[i] = RiskClass(check)
	}
	return out, err
}
