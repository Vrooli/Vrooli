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
	// MaxModeTTL bounds every log-only interval, from the terminal or from an
	// answered decision. A permanent change belongs in the shipped policy.
	MaxModeTTL         = 30 * 24 * time.Hour
	maxGuardModeTTL    = MaxModeTTL
	maxGuardModeAudits = 10000
	answerLockWait     = 5 * time.Second
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
	// ConsumedAnswers is permanent replay protection for verified Hub answers.
	// Answer IDs are retained after mode expiry, disable, and audit compaction.
	ConsumedAnswers map[string]AnswerConsumption `json:"consumed_answers,omitempty"`
}

type AnswerConsumption struct {
	Check      CheckID   `json:"check"`
	ExpiresAt  time.Time `json:"expires_at"`
	ConsumedAt time.Time `json:"consumed_at"`
}

// Store owns temporary check relaxations and their bounded JSONL audit.
// State is stored beside the policy bundle so all resource hooks share it.
type Store struct {
	Dir string
	Now func() time.Time
	// writeState is an injectable persistence seam for failure-ordering tests.
	writeState func(guardModeState) error
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

// SetLogOnlyFromAnswer atomically consumes a stable Notification Hub answer
// ID with one bounded mode interval. Callers must first read the exact
// answered structural ask and validate its requested check and human identity.
func (s *Store) SetLogOnlyFromAnswer(check CheckID, expiresAt time.Time, reason, actor, answerID string) error {
	answerID = strings.TrimSpace(answerID)
	if answerID == "" {
		return errors.New("Notification Hub answer ID is required")
	}
	if s == nil || strings.TrimSpace(s.Dir) == "" {
		return errors.New("guard mode directory is required")
	}
	if !validLoosenableCheck(check) {
		return fmt.Errorf("check %q cannot be put in log-only mode", check)
	}
	now := s.now()
	expiresAt = expiresAt.UTC()
	if !expiresAt.After(now) || expiresAt.After(now.Add(MaxModeTTL)) {
		return fmt.Errorf("answer-authorized expiry must be after now and within %s", MaxModeTTL)
	}
	reason, actor = strings.TrimSpace(reason), strings.TrimSpace(actor)
	if reason == "" || actor == "" {
		return errors.New("reason and actor are required for an audited log-only mode")
	}
	mode := GuardMode{Check: check, Mode: "log_only", EnabledAt: now, ExpiresAt: expiresAt, Actor: actor, Reason: reason, AuthorizationID: answerID}
	mutate := func(state *guardModeState) error {
		if _, used := state.ConsumedAnswers[answerID]; used {
			return fmt.Errorf("Notification Hub answer %q was already consumed", answerID)
		}
		// The synced journal record is the durable claim. If the state rename
		// fails or the process crashes before it, loadLocked restores the
		// consumed tombstone; retry is refused rather than extending the mode.
		if err := s.appendAuditLocked(GuardModeAudit{At: now, Action: "answer_consumed", Check: check,
			ExpiresAt: expiresAt, Actor: actor, Reason: reason, AuthorizationID: answerID}); err != nil {
			return err
		}
		state.ConsumedAnswers[answerID] = AnswerConsumption{Check: check, ExpiresAt: expiresAt, ConsumedAt: now}
		state.Active[check] = mode
		return nil
	}
	return s.updateWithBusyRetry(answerLockWait, mutate)
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
	if !validLoosenableCheck(check) {
		return GuardMode{}, false, fmt.Errorf("check %q cannot suppress enforcement", check)
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

// ModeFor reports, without taking the store lock, the unexpired mode for one
// check. The state file is only ever replaced by an atomic rename, so a hook
// reads a whole state and concurrent agents never contend for the lock. A
// missing file means no mode. A mode the journal has already ended may still
// show until the next locked read rewrites the state.
func (s *Store) ModeFor(check CheckID, now time.Time) (GuardMode, bool, error) {
	if s == nil || strings.TrimSpace(s.Dir) == "" {
		return GuardMode{}, false, errors.New("guard mode directory is required")
	}
	data, err := os.ReadFile(filepath.Join(s.Dir, guardModeStateFile))
	if errors.Is(err, os.ErrNotExist) {
		return GuardMode{}, false, nil
	}
	if err != nil {
		return GuardMode{}, false, err
	}
	var state guardModeState
	if err := json.Unmarshal(data, &state); err != nil {
		return GuardMode{}, false, fmt.Errorf("parse guard mode state: %w", err)
	}
	mode, ok := state.Active[check]
	if !ok || !mode.ExpiresAt.After(now) {
		return GuardMode{}, false, nil
	}
	return mode, true, nil
}

func (s *Store) updateWithBusyRetry(timeout time.Duration, mutate func(*guardModeState) error) error {
	deadline := time.Now().Add(timeout)
	for {
		err := s.update(mutate)
		if !errors.Is(err, ErrStoreBusy) || !time.Now().Before(deadline) {
			return err
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
	if s.writeState != nil {
		return s.writeState(state)
	}
	return s.writeLocked(state)
}

func (s *Store) loadLocked() (guardModeState, error) {
	data, err := os.ReadFile(filepath.Join(s.Dir, guardModeStateFile))
	var state guardModeState
	if errors.Is(err, os.ErrNotExist) {
		state = guardModeState{SchemaVersion: 2, Active: map[CheckID]GuardMode{}, ConsumedAnswers: map[string]AnswerConsumption{}}
	} else if err != nil {
		return guardModeState{}, err
	} else {
		if err := json.Unmarshal(data, &state); err != nil {
			return guardModeState{}, fmt.Errorf("parse guard mode state: %w", err)
		}
	}
	if (state.SchemaVersion != 1 && state.SchemaVersion != 2) || state.Active == nil {
		return guardModeState{}, errors.New("guard mode state has unsupported schema or missing active map")
	}
	if state.ConsumedAnswers == nil {
		state.ConsumedAnswers = map[string]AnswerConsumption{}
	}
	for check, mode := range state.Active {
		if !validStoredCheck(check) || mode.Check != check || mode.Mode != "log_only" || mode.ExpiresAt.IsZero() {
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
	// Recover answer claims that were synced to the journal before a crash or
	// failed atomic state replacement. Such an answer is burned but does not
	// recreate a mode that was never committed to state.
	for _, entry := range audit {
		if (entry.Action == "answer_consumed" || entry.Action == "enabled") && entry.AuthorizationID != "" {
			state.ConsumedAnswers[entry.AuthorizationID] = AnswerConsumption{Check: entry.Check, ExpiresAt: entry.ExpiresAt, ConsumedAt: entry.At}
		}
	}
	for check, mode := range state.Active {
		journaled := false
		for i := len(audit) - 1; i >= 0; i-- {
			entry := audit[i]
			switch {
			case (entry.Action == "enabled" || entry.Action == "answer_consumed") && entry.Check == check:
				journaled = true
				if !sameGuardModeAudit(entry, mode) {
					if entry.Action == "enabled" || entry.AuthorizationID == mode.AuthorizationID {
						delete(state.Active, check)
					}
				}
				if entry.Action == "answer_consumed" && entry.AuthorizationID != mode.AuthorizationID {
					journaled = false
					continue
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
	state.SchemaVersion = 2
	if state.ConsumedAnswers == nil {
		state.ConsumedAnswers = map[string]AnswerConsumption{}
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(s.Dir, fmt.Sprintf(".%s.%d.tmp", guardModeStateFile, time.Now().UnixNano()))
	file, err := os.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		_ = file.Close()
		_ = os.Remove(tmp)
		return err
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
	if err := os.Rename(tmp, filepath.Join(s.Dir, guardModeStateFile)); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	dir, err := os.Open(s.Dir)
	if err != nil {
		return err
	}
	if err := dir.Sync(); err != nil {
		_ = dir.Close()
		return err
	}
	if err := dir.Close(); err != nil {
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
		case "answer_consumed":
			// This is a permanent one-use tombstone, including failed commits.
			// It must survive compaction even if state is later damaged.
			continue
		case "enabled":
			if entry.AuthorizationID != "" {
				// Retain G11-era answer-backed enables during migration.
				continue
			}
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
		if !keep[i] && entry.Action != "answer_consumed" && !(entry.Action == "enabled" && entry.AuthorizationID != "") {
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
	// New modes must target an owner-issued, provider-namespaced evaluator ID.
	// The former RiskClass strings remain readable from disk for migration and
	// expiry/audit, but cannot be enabled, disabled, or used for suppression.
	if strings.TrimSpace(check) != check || strings.ContainsAny(check, " \t\r\n") {
		return false
	}
	parts := strings.SplitN(check, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}
	for _, r := range parts[0] + parts[1] {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("._/-", r)) {
			return false
		}
	}
	switch CheckID(check) {
	case RiskInspection, RiskFrozenReproduce, RiskDependencyAdd, RiskDependencyUpgrade, RiskDependencyRemove, RiskLifecycle, RiskPublish, RiskOpaque, RiskUnknown:
		return false
	}
	return true
}

// validStoredCheck keeps historical RiskClass records readable so they can
// expire or be cleared by the hard-stop tripwire. They remain invalid for
// enable, disable, and suppression operations.
func validStoredCheck(check CheckID) bool {
	return validLoosenableCheck(check) || check == RiskInspection || check == RiskFrozenReproduce || check == RiskDependencyAdd || check == RiskDependencyUpgrade || check == RiskDependencyRemove || check == RiskLifecycle || check == RiskPublish || check == RiskOpaque || check == RiskUnknown
}
