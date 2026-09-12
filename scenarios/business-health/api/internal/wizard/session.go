package wizard

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Session is one resumable interview. Persisted as JSON under the
// engine's data directory, keyed by target scenario (one live session per
// target — resuming is the default, reset is explicit).
type Session struct {
	ID       string `json:"id"`
	Scenario string `json:"scenario"`
	// TargetDir is the scenario tree the scaffold writes into.
	TargetDir string            `json:"target_dir"`
	Answers   map[string]Answer `json:"answers"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// Hinter surfaces "similar capability exists in scenario X" before
// scaffolding. Implementations MUST degrade silently (return nil) when
// their backend is unavailable — the interview never blocks on search.
type Hinter interface {
	Hints(scenario string, targets []OTAnswer) []CapabilityHint
}

// CapabilityHint is one dedup pointer (anchors only, never synthesized).
type CapabilityHint struct {
	Scenario   string
	Capability string
	Anchor     string
	Score      float64
}

// NoHints is the default Hinter (the search leaf ships in plan phase 8).
type NoHints struct{}

func (NoHints) Hints(string, []OTAnswer) []CapabilityHint { return nil }

// Engine owns session persistence and scaffold rendering.
type Engine struct {
	// dataDir is business-health's own runtime data directory.
	dataDir string
	hinter  Hinter
	now     func() time.Time
}

// NewEngine builds an Engine persisting sessions under dataDir. nil
// hinter means no dedup hints; nil now means time.Now.
func NewEngine(dataDir string, hinter Hinter, now func() time.Time) *Engine {
	if hinter == nil {
		hinter = NoHints{}
	}
	if now == nil {
		now = time.Now
	}
	return &Engine{dataDir: dataDir, hinter: hinter, now: now}
}

func (e *Engine) sessionPath(scenario string) string {
	return filepath.Join(e.dataDir, "wizard-sessions", scenario+".json")
}

// StartSession creates or resumes the session for one target scenario.
func (e *Engine) StartSession(scenario, targetDir string, reset bool) (Session, error) {
	if strings.TrimSpace(scenario) == "" {
		return Session{}, fmt.Errorf("scenario is required")
	}
	path := e.sessionPath(scenario)
	if !reset {
		if data, err := os.ReadFile(path); err == nil {
			var s Session
			if err := json.Unmarshal(data, &s); err == nil {
				return s, nil
			}
			// A corrupt session file falls through to a fresh session.
		}
	}
	s := Session{
		ID:        fmt.Sprintf("wiz-%s-%d", scenario, e.now().UTC().UnixNano()),
		Scenario:  scenario,
		TargetDir: targetDir,
		Answers:   map[string]Answer{},
		CreatedAt: e.now().UTC(),
		UpdatedAt: e.now().UTC(),
	}
	if err := e.save(s); err != nil {
		return Session{}, err
	}
	return s, nil
}

// LoadSession resumes the session for a scenario (error when none).
func (e *Engine) LoadSession(scenario string) (Session, error) {
	data, err := os.ReadFile(e.sessionPath(scenario))
	if err != nil {
		return Session{}, fmt.Errorf("no wizard session for %q (start one first): %w", scenario, err)
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return Session{}, fmt.Errorf("session for %q is corrupt: %w", scenario, err)
	}
	return s, nil
}

// LoadSessionByID resumes a session by its ID (scans the session dir).
func (e *Engine) LoadSessionByID(id string) (Session, error) {
	entries, err := os.ReadDir(filepath.Join(e.dataDir, "wizard-sessions"))
	if err != nil {
		return Session{}, fmt.Errorf("no wizard sessions exist yet")
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		s, err := e.LoadSession(strings.TrimSuffix(entry.Name(), ".json"))
		if err == nil && s.ID == id {
			return s, nil
		}
	}
	return Session{}, fmt.Errorf("wizard session %q not found", id)
}

// SubmitAnswers validates and records answers, returning per-answer
// invalid reasons keyed by question id (empty map = all valid).
func (e *Engine) SubmitAnswers(s *Session, answers []Answer) (map[string]string, error) {
	byID := questionIndex()
	invalid := map[string]string{}
	for _, a := range answers {
		q, ok := byID[a.QuestionID]
		if !ok {
			invalid[a.QuestionID] = "unknown question id"
			continue
		}
		if reason := ValidateAnswer(q, a); reason != "" {
			invalid[a.QuestionID] = reason
			continue
		}
		s.Answers[a.QuestionID] = a
	}
	s.UpdatedAt = e.now().UTC()
	if err := e.save(*s); err != nil {
		return nil, err
	}
	return invalid, nil
}

// Remaining lists unanswered-or-invalid required question ids in ask order.
func Remaining(s Session) []string {
	var out []string
	for _, q := range Questions() {
		a, ok := s.Answers[q.ID]
		if !ok {
			if q.Required {
				out = append(out, q.ID)
			}
			continue
		}
		if ValidateAnswer(q, a) != "" {
			out = append(out, q.ID)
		}
	}
	return out
}

// Complete reports whether every required question has a valid answer.
func Complete(s Session) bool { return len(Remaining(s)) == 0 }

// Hints runs the dedup hook over the answered targets (silent no-op when
// the hinter's backend is down).
func (e *Engine) Hints(s Session) []CapabilityHint {
	var targets []OTAnswer
	for _, id := range []string{"targets_p0", "targets_p1", "targets_p2"} {
		if a, ok := s.Answers[id]; ok {
			targets = append(targets, a.Targets...)
		}
	}
	if len(targets) == 0 {
		return nil
	}
	return e.hinter.Hints(s.Scenario, targets)
}

func (e *Engine) save(s Session) error {
	path := e.sessionPath(s.Scenario)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func questionIndex() map[string]Question {
	out := map[string]Question{}
	for _, q := range Questions() {
		out[q.ID] = q
	}
	return out
}

// sortedAnswerIDs gives deterministic iteration for rendering.
func sortedAnswerIDs(s Session) []string {
	out := make([]string, 0, len(s.Answers))
	for id := range s.Answers {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
