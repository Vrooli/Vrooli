package repohealth

import (
	"context"
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
	Version         string        `json:"version"`
	ID              string        `json:"id"`
	DestinationID   string        `json:"destination_id"`
	Signature       string        `json:"signature"`
	Generation      int           `json:"generation"`
	State           IncidentState `json:"state"`
	PlanIDs         []string      `json:"plan_ids"`
	Subject         string        `json:"subject"`
	Diagnosis       Diagnosis     `json:"diagnosis"`
	Offers          Offers        `json:"offers"`
	AskID           string        `json:"ask_id,omitempty"`
	DiagnosedRunID  string        `json:"diagnosed_run_id,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	AskedAt         time.Time     `json:"asked_at,omitempty"`
	UpdatedAt       time.Time     `json:"updated_at"`
	DeclinedAt      time.Time     `json:"declined_at,omitempty"`
	AnsweredAt      time.Time     `json:"answered_at,omitempty"`
	AnsweredBy      string        `json:"answered_by,omitempty"`
	Answer          string        `json:"answer,omitempty"`
	Result          string        `json:"result,omitempty"`
	RepairJournalID string        `json:"repair_journal_id,omitempty"`
	ResolvedAt      time.Time     `json:"resolved_at,omitempty"`
	ExceptionPaths  []string      `json:"exception_paths,omitempty"`
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
	Version string               `json:"version"`
	Sent    map[string]time.Time `json:"sent"`
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
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.load()
	if err != nil {
		return err
	}
	state[key] = t.UTC()
	data, err := json.MarshalIndent(noticeFile{Version: "dbm-backup-notices-v1", Sent: state}, "", "  ")
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
