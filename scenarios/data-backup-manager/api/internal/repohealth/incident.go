package repohealth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// IncidentState is where a backup-failure incident is in its life.
type IncidentState string

const (
	// IncidentAwaiting: a question is in front of the operator.
	IncidentAwaiting IncidentState = "awaiting_decision"
	// IncidentDeclined: the operator chose to leave it, dismissed the question,
	// or the answer was not a verified human approval. The question returns after
	// the cool-down.
	IncidentDeclined IncidentState = "declined"
	// IncidentLapsed: the hub closed the question unanswered at its deadline.
	// Nothing was changed; the monitor asks again at once while the condition lasts.
	IncidentLapsed IncidentState = "lapsed"
	// IncidentApplying: a verified answer is being carried out.
	IncidentApplying IncidentState = "applying"
	// IncidentApplied: the answer was applied and a retry run was requested; the
	// incident closes when a backup succeeds or a changed condition supersedes it.
	IncidentApplied IncidentState = "answer_applied"
	// IncidentAgentRequested: the operator chose the option an agent must carry
	// out; nothing was changed.
	IncidentAgentRequested IncidentState = "agent_requested"
	// IncidentFailed: a verified answer was applied and the action failed.
	IncidentFailed IncidentState = "action_failed"
	// IncidentResolved: backups succeeded again.
	IncidentResolved IncidentState = "resolved"
	// IncidentSuperseded: a different condition replaced this one.
	IncidentSuperseded IncidentState = "superseded"
)

func (s IncidentState) Valid() bool {
	switch s {
	case IncidentAwaiting, IncidentDeclined, IncidentLapsed, IncidentApplying, IncidentApplied,
		IncidentAgentRequested, IncidentFailed, IncidentResolved, IncidentSuperseded:
		return true
	default:
		return false
	}
}

func (s IncidentState) CanTransitionTo(next IncidentState) bool {
	if !s.Valid() || !next.Valid() {
		return false
	}
	if s == next {
		return true // metadata refreshes may persist without changing lifecycle state.
	}
	switch s {
	case IncidentAwaiting:
		return next == IncidentDeclined || next == IncidentLapsed || next == IncidentApplying ||
			next == IncidentAgentRequested || next == IncidentResolved || next == IncidentSuperseded
	case IncidentDeclined, IncidentLapsed, IncidentFailed:
		return next == IncidentAwaiting || next == IncidentResolved || next == IncidentSuperseded
	case IncidentApplying:
		return next == IncidentApplied || next == IncidentFailed || next == IncidentAgentRequested || next == IncidentSuperseded
	case IncidentApplied:
		return next == IncidentAwaiting || next == IncidentResolved || next == IncidentSuperseded
	case IncidentAgentRequested:
		return next == IncidentResolved || next == IncidentSuperseded
	case IncidentResolved, IncidentSuperseded:
		return next == IncidentAwaiting
	default:
		return false
	}
}

type ErrInvalidIncidentTransition struct {
	Current IncidentState
	Next    IncidentState
}

func (e ErrInvalidIncidentTransition) Error() string {
	return fmt.Sprintf("invalid backup incident transition %q -> %q", e.Current, e.Next)
}

// Open reports whether the incident still needs the monitor's attention.
func (s IncidentState) Open() bool {
	switch s {
	case IncidentResolved, IncidentSuperseded:
		return false
	default:
		return true
	}
}

// Incident is the durable journal of one backup-failure condition: what was
// diagnosed, what was asked, what was answered and what was done.
type Incident struct {
	Version                  string        `json:"version"`
	ID                       string        `json:"id"`
	DestinationID            string        `json:"destination_id"`
	Signature                string        `json:"signature"`
	Generation               int           `json:"generation"`
	State                    IncidentState `json:"state"`
	PlanIDs                  []string      `json:"plan_ids"`
	Subject                  string        `json:"subject"`
	Diagnosis                Diagnosis     `json:"diagnosis"`
	Offers                   Offers        `json:"offers"`
	AskID                    string        `json:"ask_id,omitempty"`
	DiagnosedRunID           string        `json:"diagnosed_run_id,omitempty"`
	CreatedAt                time.Time     `json:"created_at"`
	AskedAt                  time.Time     `json:"asked_at,omitempty"`
	UpdatedAt                time.Time     `json:"updated_at"`
	DeclinedAt               time.Time     `json:"declined_at,omitempty"`
	AnsweredAt               time.Time     `json:"answered_at,omitempty"`
	AnsweredBy               string        `json:"answered_by,omitempty"`
	AnsweredByIdentityKind   string        `json:"answered_by_identity_kind,omitempty"`
	AnsweredByIdentitySource string        `json:"answered_by_identity_source,omitempty"`
	Answer                   string        `json:"answer,omitempty"`
	Result                   string        `json:"result,omitempty"`
	RepairJournalID          string        `json:"repair_journal_id,omitempty"`
	QuarantinedPaths         []string      `json:"quarantined_paths,omitempty"`
	// FreshMigrationStage records the callback's durable handoff state. A fresh
	// repository cannot be provisioned until the operator supplies a destination
	// location and the owner proves capacity/credentials; the approval itself
	// must never be treated as permission to guess those inputs.
	FreshMigrationStage                  string    `json:"fresh_migration_stage,omitempty"`
	FreshMigrationID                     string    `json:"fresh_migration_id,omitempty"`
	FreshMigrationDetail                 string    `json:"fresh_migration_detail,omitempty"`
	FreshMigrationPlanID                 string    `json:"fresh_migration_plan_id,omitempty"`
	FreshMigrationDestinationID          string    `json:"fresh_migration_destination_id,omitempty"`
	FreshMigrationTargets                []string  `json:"fresh_migration_targets,omitempty"`
	FreshMigrationSelectionAskID         string    `json:"fresh_migration_selection_ask_id,omitempty"`
	FreshMigrationSelectionGeneration    int       `json:"fresh_migration_selection_generation,omitempty"`
	FreshMigrationSelectionBy            string    `json:"fresh_migration_selection_by,omitempty"`
	FreshMigrationSelectionRetryAt       time.Time `json:"fresh_migration_selection_retry_at,omitempty"`
	FreshMigrationProposedLocation       string    `json:"fresh_migration_proposed_location,omitempty"`
	FreshMigrationRegistrationAskID      string    `json:"fresh_migration_registration_ask_id,omitempty"`
	FreshMigrationRegistrationGeneration int       `json:"fresh_migration_registration_generation,omitempty"`
	FreshMigrationRegistrationBy         string    `json:"fresh_migration_registration_by,omitempty"`
	FreshMigrationRegisteredDestID       string    `json:"fresh_migration_registered_destination_id,omitempty"`
	ResolvedAt                           time.Time `json:"resolved_at,omitempty"`
	ExceptionPaths                       []string  `json:"exception_paths,omitempty"`
}

// FreshMigrationID binds the migration journal to the incident generation
// whose verified operator answer authorized the request. It contains no raw
// incident or destination identifiers and is stable across callback replay.
func FreshMigrationID(incidentID string, generation int) string {
	identity := fmt.Sprintf("%s\x00%d", strings.TrimSpace(incidentID), generation)
	sum := sha256.Sum256([]byte(identity))
	return "db4-" + hex.EncodeToString(sum[:16])
}

// Transition is the single lifecycle mutation owner. Reopening starts a new
// generation; every other event retains the current generation.
func (i *Incident) Transition(next IncidentState) error {
	if i == nil || !i.State.CanTransitionTo(next) {
		current := IncidentState("<nil>")
		if i != nil {
			current = i.State
		}
		return ErrInvalidIncidentTransition{Current: current, Next: next}
	}
	if i.State == next {
		return nil
	}
	if next == IncidentAwaiting {
		i.Generation++
	}
	i.State = next
	return nil
}

const incidentVersion = "dbm-backup-incident-v1"

// IncidentID is the stable identity of one condition on one destination.
func IncidentID(destinationID, signature string) string {
	return "bf-" + strings.TrimSpace(destinationID)[:minInt(8, len(strings.TrimSpace(destinationID)))] + "-" + signature
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// IncidentStore keeps incident journals.
type IncidentStore interface {
	Get(ctx context.Context, id string) (Incident, bool, error)
	Put(ctx context.Context, in Incident) error
	List(ctx context.Context) ([]Incident, error)
}

// FileIncidentStore keeps one JSON file per incident, written atomically.
type FileIncidentStore struct {
	Root string
	mu   sync.Mutex
}

var _ IncidentStore = (*FileIncidentStore)(nil)

func (s *FileIncidentStore) path(id string) (string, error) {
	if s == nil || !filepath.IsAbs(s.Root) || strings.TrimSpace(id) == "" || strings.ContainsAny(id, `/\`) {
		return "", errors.New("invalid incident store or id")
	}
	return filepath.Join(filepath.Clean(s.Root), id+".json"), nil
}

func (s *FileIncidentStore) Get(ctx context.Context, id string) (Incident, bool, error) {
	if err := ctx.Err(); err != nil {
		return Incident{}, false, err
	}
	p, err := s.path(id)
	if err != nil {
		return Incident{}, false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return readIncident(p)
}

func readIncident(p string) (Incident, bool, error) {
	data, err := os.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) {
		return Incident{}, false, nil
	}
	if err != nil {
		return Incident{}, false, err
	}
	var in Incident
	if err := json.Unmarshal(data, &in); err != nil {
		return Incident{}, false, fmt.Errorf("incident journal is corrupt: %w", err)
	}
	if !in.State.Valid() {
		return Incident{}, false, fmt.Errorf("incident journal has unknown state %q", in.State)
	}
	return in, true, nil
}

func (s *FileIncidentStore) Put(ctx context.Context, in Incident) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	in.Version = incidentVersion
	p, err := s.path(in.ID)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(in, "", "  ")
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	current, found, err := readIncident(p)
	if err != nil {
		return err
	}
	if !in.State.Valid() {
		return ErrInvalidIncidentTransition{Current: current.State, Next: in.State}
	}
	if !found {
		if in.State != IncidentAwaiting || in.Generation < 1 {
			return ErrInvalidIncidentTransition{Current: "<new>", Next: in.State}
		}
	} else {
		if current.ID != in.ID || current.DestinationID != in.DestinationID || current.Signature != in.Signature || !current.CreatedAt.Equal(in.CreatedAt) {
			return fmt.Errorf("backup incident %q identity is immutable", in.ID)
		}
		if !current.State.CanTransitionTo(in.State) {
			return ErrInvalidIncidentTransition{Current: current.State, Next: in.State}
		}
		wantGeneration := current.Generation
		if current.State != in.State && in.State == IncidentAwaiting {
			wantGeneration++
		}
		if in.Generation != wantGeneration {
			return fmt.Errorf("backup incident %q generation changed from %d to %d; want %d", in.ID, current.Generation, in.Generation, wantGeneration)
		}
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".incident-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, p)
}

func (s *FileIncidentStore) List(ctx context.Context) ([]Incident, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || !filepath.IsAbs(s.Root) {
		return nil, errors.New("invalid incident store")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := os.ReadDir(filepath.Clean(s.Root))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Incident
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		in, ok, err := readIncident(filepath.Join(filepath.Clean(s.Root), e.Name()))
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, in)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

// NoticeStateStore persists when each notification key was last sent, so a
// restart neither drops the daily cadence nor repeats a once-only notice.
type NoticeStateStore struct {
	Path string
	mu   sync.Mutex
}

type noticeFile struct {
	Version string                   `json:"version"`
	Sent    map[string]time.Time     `json:"sent"`
	Pending map[string]PendingNotice `json:"pending,omitempty"`
}

// PendingNotice is an operator-facing event persisted before publication.
// DedupeKey is stable across restarts, so event delivery can be retried safely.
type PendingNotice struct {
	Key       string    `json:"key"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Category  string    `json:"category"`
	Urgency   string    `json:"urgency"`
	DedupeKey string    `json:"dedupe_key"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *NoticeStateStore) Load() (NoticeState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}

func (s *NoticeStateStore) load() (NoticeState, error) {
	if s == nil || !filepath.IsAbs(s.Path) {
		return nil, errors.New("invalid notice state store")
	}
	data, err := os.ReadFile(s.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return NoticeState{}, nil
	}
	if err != nil {
		return nil, err
	}
	var f noticeFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("notice state is corrupt: %w", err)
	}
	if f.Sent == nil {
		f.Sent = map[string]time.Time{}
	}
	return NoticeState(f.Sent), nil
}

// Mark records that key was sent at t.
func (s *NoticeStateStore) Mark(key string, t time.Time) error {
	if s == nil || !filepath.IsAbs(s.Path) || key == "" {
		return errors.New("invalid notice state store")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.loadFile()
	if err != nil {
		return err
	}
	f.Sent[key] = t.UTC()
	return s.saveFile(f)
}

// Queue persists a one-off result notice before attempting delivery.
func (s *NoticeStateStore) Queue(n PendingNotice) error {
	if s == nil || !filepath.IsAbs(s.Path) || n.Key == "" || n.Title == "" || n.Body == "" || n.DedupeKey == "" {
		return errors.New("invalid pending notice")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.loadFile()
	if err != nil {
		return err
	}
	if f.Pending == nil {
		f.Pending = map[string]PendingNotice{}
	}
	if _, sent := f.Sent[n.Key]; sent {
		return nil
	}
	if _, exists := f.Pending[n.Key]; !exists {
		f.Pending[n.Key] = n
	}
	return s.saveFile(f)
}

func (s *NoticeStateStore) Pending() ([]PendingNotice, error) {
	if s == nil || !filepath.IsAbs(s.Path) {
		return nil, errors.New("invalid notice state store")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.loadFile()
	if err != nil {
		return nil, err
	}
	rows := make([]PendingNotice, 0, len(f.Pending))
	for _, row := range f.Pending {
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Key < rows[j].Key })
	return rows, nil
}

// MarkDelivered removes a pending notice and records its stable key as sent.
func (s *NoticeStateStore) MarkDelivered(key string, t time.Time) error {
	if s == nil || !filepath.IsAbs(s.Path) || key == "" {
		return errors.New("invalid notice state store")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.loadFile()
	if err != nil {
		return err
	}
	if f.Sent == nil {
		f.Sent = map[string]time.Time{}
	}
	if f.Pending != nil {
		delete(f.Pending, key)
	}
	f.Sent[key] = t.UTC()
	return s.saveFile(f)
}

func (s *NoticeStateStore) loadFile() (noticeFile, error) {
	data, err := os.ReadFile(s.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return noticeFile{Version: "dbm-backup-notices-v1", Sent: map[string]time.Time{}, Pending: map[string]PendingNotice{}}, nil
	}
	if err != nil {
		return noticeFile{}, err
	}
	var f noticeFile
	if err := json.Unmarshal(data, &f); err != nil {
		return noticeFile{}, fmt.Errorf("notice state is corrupt: %w", err)
	}
	if f.Sent == nil {
		f.Sent = map[string]time.Time{}
	}
	if f.Pending == nil {
		f.Pending = map[string]PendingNotice{}
	}
	return f, nil
}

func (s *NoticeStateStore) saveFile(f noticeFile) error {
	f.Version = "dbm-backup-notices-v1"
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.Path), ".notices-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, s.Path)
}
