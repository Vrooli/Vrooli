package guardmodes

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	guardModeStateFile = "guard-modes.json"
	guardModeAuditFile = "guard-modes.audit.jsonl"
	guardModeLockFile  = ".guard-modes.lock"
	maxGuardModeTTL    = 7 * 24 * time.Hour
	maxGuardModeAudits = 10000
	restoreLockWait    = 500 * time.Millisecond
)

type GuardMode struct {
	Check           CheckID   `json:"check"`
	Mode            string    `json:"mode"`
	EnabledAt       time.Time `json:"enabled_at"`
	ExpiresAt       time.Time `json:"expires_at"`
	Actor           string    `json:"actor"`
	Reason          string    `json:"reason"`
	AuthorizationID string    `json:"authorization_id,omitempty"`
}

type GuardModeAudit struct {
	At              time.Time `json:"at"`
	Action          string    `json:"action"`
	Check           CheckID   `json:"check,omitempty"`
	Checks          []CheckID `json:"checks,omitempty"`
	OriginalAction  Action    `json:"original_action,omitempty"`
	ExpiresAt       time.Time `json:"expires_at,omitempty"`
	Actor           string    `json:"actor,omitempty"`
	Reason          string    `json:"reason,omitempty"`
	AuthorizationID string    `json:"authorization_id,omitempty"`
	HardStop        HardStop  `json:"hard_stop,omitempty"`
	EventID         string    `json:"event_id,omitempty"`
	RestoredChecks  []CheckID `json:"restored_checks,omitempty"`
}

type guardModeState struct {
	SchemaVersion int                   `json:"schema_version"`
	Active        map[CheckID]GuardMode `json:"active"`
}

// Store owns temporary check relaxations and their bounded JSONL audit.
// State is stored beside the policy bundle so all resource hooks share it.
type Store struct {
	Dir string
	Now func() time.Time
}

func NewStore(dir string) *Store {
	return &Store{Dir: dir, Now: time.Now}
}

func DefaultDataDir() (string, error) {
	if override := strings.TrimSpace(os.Getenv("VROOLI_AGENT_POLICY_HOME")); override != "" {
		return filepath.Clean(override), nil
	}
	root, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve application config directory: %w", err)
	}
	return filepath.Join(root, "vrooli", "agent-policy"), nil
}

func (s *Store) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

// SetLogOnly temporarily converts one provider check into an audited
// observation. Floor checks (operator-only commands and filesystem deletion)
// cannot be loosened. Expiries are limited to seven days.
func (s *Store) SetLogOnly(check CheckID, expiresAt time.Time, reason, actor string) error {
	return s.setLogOnly(check, expiresAt, reason, actor, "")
}

// SetLogOnlyFromAnswer records the stable Notification Hub answer ID that
// authorized a steward-applied mode. Callers must first read the exact
// answered structural ask and validate its requested check and expiry.
func (s *Store) SetLogOnlyFromAnswer(check CheckID, expiresAt time.Time, reason, actor, answerID string) error {
	answerID = strings.TrimSpace(answerID)
	if answerID == "" {
		return errors.New("Notification Hub answer ID is required")
	}
	return s.setLogOnly(check, expiresAt, reason, actor, answerID)
}

func (s *Store) setLogOnly(check CheckID, expiresAt time.Time, reason, actor, authorizationID string) error {
	if s == nil || strings.TrimSpace(s.Dir) == "" {
		return errors.New("guard mode directory is required")
	}
	if !validLoosenableCheck(check) {
		return fmt.Errorf("check %q cannot be put in log-only mode", check)
	}
	now := s.now()
	expiresAt = expiresAt.UTC()
	if !expiresAt.After(now) || expiresAt.After(now.Add(maxGuardModeTTL)) {
		return fmt.Errorf("expiry must be after now and within %s", maxGuardModeTTL)
	}
	reason, actor = strings.TrimSpace(reason), strings.TrimSpace(actor)
	if reason == "" || actor == "" {
		return errors.New("reason and actor are required for an audited log-only mode")
	}
	mode := GuardMode{Check: check, Mode: "log_only", EnabledAt: now, ExpiresAt: expiresAt, Actor: actor, Reason: reason, AuthorizationID: authorizationID}
	return s.update(func(state *guardModeState) error {
		if err := s.appendAuditLocked(GuardModeAudit{At: now, Action: "enabled", Check: check, ExpiresAt: expiresAt, Actor: actor, Reason: reason, AuthorizationID: authorizationID}); err != nil {
			return err
		}
		state.Active[check] = mode
		return nil
	})
}

func (s *Store) Disable(check CheckID, actor, reason string) error {
	if s == nil || strings.TrimSpace(s.Dir) == "" {
		return errors.New("guard mode directory is required")
	}
	if !validLoosenableCheck(check) {
		return fmt.Errorf("check %q cannot be put in log-only mode", check)
	}
	actor, reason = strings.TrimSpace(actor), strings.TrimSpace(reason)
	if actor == "" || reason == "" {
		return errors.New("actor and reason are required for an audited mode change")
	}
	now := s.now()
	return s.update(func(state *guardModeState) error {
		if _, ok := state.Active[check]; !ok {
			return fmt.Errorf("check %q has no active log-only mode", check)
		}
		if err := s.appendAuditLocked(GuardModeAudit{At: now, Action: "disabled", Check: check, Actor: actor, Reason: reason}); err != nil {
			return err
		}
		delete(state.Active, check)
		return nil
	})
}

func (s *Store) Active(now time.Time) ([]GuardMode, error) {
	if s == nil || strings.TrimSpace(s.Dir) == "" {
		return nil, errors.New("guard mode directory is required")
	}
	var active []GuardMode
	err := s.update(func(state *guardModeState) error {
		checks := make([]string, 0, len(state.Active))
		for check := range state.Active {
			checks = append(checks, string(check))
		}
		sort.Strings(checks)
		for _, raw := range checks {
			check := CheckID(raw)
			mode := state.Active[check]
			if !mode.ExpiresAt.After(now) {
				if err := s.appendAuditLocked(GuardModeAudit{At: now.UTC(), Action: "expired", Check: check, ExpiresAt: mode.ExpiresAt, Actor: mode.Actor, Reason: mode.Reason, AuthorizationID: mode.AuthorizationID}); err != nil {
					return err
				}
				delete(state.Active, check)
				continue
			}
			active = append(active, mode)
		}
		return nil
	})
	return active, err
}

// RecordSuppressedAction durably records one non-allow decision immediately
// before log-only mode suppresses it. The check and mode are revalidated under
// the same lock as the audit append, so a concurrent disable cannot be raced.
func (s *Store) RecordSuppressedAction(check CheckID, eventID string, original Action, now time.Time) (GuardMode, bool, error) {
	if s == nil || strings.TrimSpace(s.Dir) == "" {
		return GuardMode{}, false, errors.New("guard mode directory is required")
	}
	var mode GuardMode
	var active bool
	err := s.update(func(state *guardModeState) error {
		candidate, ok := state.Active[check]
		if !ok || !candidate.ExpiresAt.After(now) {
			if ok {
				if err := s.appendAuditLocked(GuardModeAudit{At: now.UTC(), Action: "expired", Check: check, ExpiresAt: candidate.ExpiresAt, Actor: candidate.Actor, Reason: candidate.Reason, AuthorizationID: candidate.AuthorizationID}); err != nil {
					return err
				}
				delete(state.Active, check)
			}
			return nil
		}
		if original == "allow" {
			return nil
		}
		if err := s.appendAuditLocked(GuardModeAudit{At: now.UTC(), Action: "observed", Check: check, EventID: eventID, OriginalAction: original, ExpiresAt: candidate.ExpiresAt}); err != nil {
			return err
		}
		mode, active = candidate, true
		return nil
	})
	return mode, active, err
}

func (s *Store) Audit() ([]GuardModeAudit, error) {
	if s == nil || strings.TrimSpace(s.Dir) == "" {
		return nil, errors.New("guard mode directory is required")
	}
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return nil, fmt.Errorf("create guard mode directory: %w", err)
	}
	unlock, err := acquireFileLock(filepath.Join(s.Dir, guardModeLockFile))
	if err != nil {
		return nil, err
	}
	defer unlock()
	return s.readAuditLocked()
}

func (s *Store) RestoreAll(hardStop HardStop, eventID string, now time.Time) ([]CheckID, error) {
	if s == nil || strings.TrimSpace(s.Dir) == "" {
		return nil, errors.New("guard mode directory is required")
	}
	deadline := time.Now().Add(restoreLockWait)
	for {
		var restored []CheckID
		err := s.update(func(state *guardModeState) error {
			if len(state.Active) == 0 {
				return nil
			}
			for check := range state.Active {
				restored = append(restored, check)
			}
			sort.Slice(restored, func(i, j int) bool { return restored[i] < restored[j] })
			if err := s.appendAuditLocked(GuardModeAudit{At: now.UTC(), Action: "restored", HardStop: hardStop, EventID: eventID, RestoredChecks: restored}); err != nil {
				return err
			}
			state.Active = map[CheckID]GuardMode{}
			return nil
		})
		if !errors.Is(err, ErrStoreBusy) || !time.Now().Before(deadline) {
			return restored, err
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func (s *Store) update(mutate func(*guardModeState) error) error {
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return fmt.Errorf("create guard mode directory: %w", err)
	}
	unlock, err := acquireFileLock(filepath.Join(s.Dir, guardModeLockFile))
	if err != nil {
		return err
	}
	defer unlock()
	state, err := s.loadLocked()
	if err != nil {
		return err
	}
	if err := mutate(&state); err != nil {
		return err
	}
	return s.writeLocked(state)
}

func (s *Store) loadLocked() (guardModeState, error) {
	data, err := os.ReadFile(filepath.Join(s.Dir, guardModeStateFile))
	if errors.Is(err, os.ErrNotExist) {
		return guardModeState{SchemaVersion: 1, Active: map[CheckID]GuardMode{}}, nil
	}
	if err != nil {
		return guardModeState{}, err
	}
	var state guardModeState
	if err := json.Unmarshal(data, &state); err != nil {
		return guardModeState{}, fmt.Errorf("parse guard mode state: %w", err)
	}
	if state.SchemaVersion != 1 || state.Active == nil {
		return guardModeState{}, errors.New("guard mode state has unsupported schema or missing active map")
	}
	for check, mode := range state.Active {
		if !validLoosenableCheck(check) || mode.Check != check || mode.Mode != "log_only" || mode.ExpiresAt.IsZero() {
			return guardModeState{}, fmt.Errorf("invalid active guard mode for %q", check)
		}
	}
	// The audit is the recovery journal. If a crash occurs after an audited
	// restoration but before the atomic state replacement, later reads must not
	// resurrect the old relaxation.
	audit, err := s.readAuditLocked()
	if err != nil {
		return guardModeState{}, err
	}
	for check, mode := range state.Active {
		journaled := false
		for i := len(audit) - 1; i >= 0; i-- {
			entry := audit[i]
			switch {
			case entry.Action == "enabled" && entry.Check == check:
				journaled = true
				if !sameGuardModeAudit(entry, mode) {
					delete(state.Active, check)
				}
				i = -1
			case (entry.Action == "disabled" || entry.Action == "expired") && entry.Check == check:
				journaled = true
				delete(state.Active, check)
				i = -1
			case entry.Action == "restored" && containsRisk(entry.RestoredChecks, check):
				journaled = true
				delete(state.Active, check)
				i = -1
			}
		}
		if !journaled {
			delete(state.Active, check)
		}
	}
	return state, nil
}

func sameGuardModeAudit(entry GuardModeAudit, mode GuardMode) bool {
	return entry.Actor == mode.Actor && entry.Reason == mode.Reason && entry.AuthorizationID == mode.AuthorizationID && entry.ExpiresAt.Equal(mode.ExpiresAt)
}

func containsRisk(checks []CheckID, check CheckID) bool {
	for _, candidate := range checks {
		if candidate == check {
			return true
		}
	}
	return false
}

func (s *Store) writeLocked(state guardModeState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(s.Dir, fmt.Sprintf(".%s.%d.tmp", guardModeStateFile, time.Now().UnixNano()))
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, filepath.Join(s.Dir, guardModeStateFile)); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func (s *Store) appendAuditLocked(entry GuardModeAudit) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(filepath.Join(s.Dir, guardModeAuditFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open guard mode audit: %w", err)
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		_ = file.Close()
		return fmt.Errorf("append guard mode audit: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync guard mode audit: %w", err)
	}
	if err := file.Close(); err != nil {
		return err
	}
	return s.compactAuditLocked()
}

func (s *Store) compactAuditLocked() error {
	entries, err := s.readAuditLocked()
	if err != nil {
		return err
	}
	if len(entries) <= maxGuardModeAudits {
		return nil
	}
	activeEnables := map[CheckID]int{}
	for i, entry := range entries {
		switch entry.Action {
		case "enabled":
			activeEnables[entry.Check] = i
		case "disabled", "expired":
			delete(activeEnables, entry.Check)
		case "restored":
			for _, check := range entry.RestoredChecks {
				delete(activeEnables, check)
			}
		}
	}
	start := len(entries) - maxGuardModeAudits
	keep := make(map[int]bool, maxGuardModeAudits+len(activeEnables))
	for i := start; i < len(entries); i++ {
		keep[i] = true
	}
	for _, i := range activeEnables {
		keep[i] = true
	}
	tmp := filepath.Join(s.Dir, fmt.Sprintf(".%s.%d.tmp", guardModeAuditFile, time.Now().UnixNano()))
	file, err := os.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create compacted guard audit: %w", err)
	}
	for i, entry := range entries {
		if !keep[i] {
			continue
		}
		data, err := json.Marshal(entry)
		if err != nil {
			_ = file.Close()
			_ = os.Remove(tmp)
			return err
		}
		if _, err := file.Write(append(data, '\n')); err != nil {
			_ = file.Close()
			_ = os.Remove(tmp)
			return fmt.Errorf("write compacted guard audit: %w", err)
		}
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, filepath.Join(s.Dir, guardModeAuditFile)); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("publish compacted guard audit: %w", err)
	}
	return nil
}

func (s *Store) readAuditLocked() ([]GuardModeAudit, error) {
	file, err := os.Open(filepath.Join(s.Dir, guardModeAuditFile))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var entries []GuardModeAudit
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	for scanner.Scan() {
		var entry GuardModeAudit
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, fmt.Errorf("parse guard mode audit: %w", err)
		}
		entries = append(entries, entry)
	}
	return entries, scanner.Err()
}

func validLoosenableCheck(check CheckID) bool {
	switch check {
	case RiskInspection, RiskFrozenReproduce, RiskDependencyAdd, RiskDependencyUpgrade, RiskDependencyRemove, RiskLifecycle, RiskPublish, RiskOpaque, RiskUnknown:
		return true
	default:
		return false
	}
}
