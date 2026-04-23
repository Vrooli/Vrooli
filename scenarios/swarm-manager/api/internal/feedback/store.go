package feedback

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"swarm-manager/internal/storage"
)

// Store persists feedback rounds under an initiative's folder. The backing
// layout is:
//
//	{initiativeDir}/feedback/
//	  round-001-{slug}/feedback.json
//	  round-002-{slug}/feedback.json
//	  ...
//
// The store is stateless — rounds are discovered by scanning the feedback
// dir on each call, so concurrent writers can safely add rounds without
// invalidating other readers' views.
type Store struct {
	// InitDir returns the absolute path to a named initiative's folder.
	// Supplied by the caller (typically initiatives.Service.InitDir) to
	// avoid importing the initiatives package from here.
	InitDir func(initiativeName string) string
}

// NewStore returns a Store backed by the given initiative-dir resolver.
func NewStore(initDir func(string) string) *Store {
	return &Store{InitDir: initDir}
}

// ErrRoundNotFound is returned when LoadRound cannot locate the requested round.
var ErrRoundNotFound = errors.New("feedback round not found")

const (
	feedbackDirName  = "feedback"
	feedbackJSONName = "feedback.json"
)

var (
	roundDirRE  = regexp.MustCompile(`^round-(\d{3})(?:-(.+))?$`)
	slugRE      = regexp.MustCompile(`[^a-z0-9-]+`)
	hyphenRunRE = regexp.MustCompile(`-+`)
)

// RoundDir returns the absolute path to a round's folder, whether or not
// it exists on disk yet.
func (s *Store) RoundDir(initiativeName string, number int, slug string) string {
	feedbackDir := s.FeedbackDir(initiativeName)
	return filepath.Join(feedbackDir, roundFolderName(number, slug))
}

// FeedbackDir returns the initiative's feedback subfolder path.
func (s *Store) FeedbackDir(initiativeName string) string {
	return filepath.Join(s.InitDir(initiativeName), feedbackDirName)
}

// roundFolderName formats the canonical folder name: `round-001-{slug}`.
// Empty slug yields `round-001`.
func roundFolderName(number int, slug string) string {
	if slug == "" {
		return fmt.Sprintf("round-%03d", number)
	}
	return fmt.Sprintf("round-%03d-%s", number, slug)
}

// Sanitize produces a folder-safe slug. Used by the service when promoting
// a user-supplied title into a round folder name; exposed so tests can
// assert the same rules.
func Sanitize(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = slugRE.ReplaceAllString(s, "-")
	s = hyphenRunRE.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 40 {
		s = strings.TrimRight(s[:40], "-")
	}
	return s
}

// NextRoundNumber returns one more than the highest existing round number
// for the initiative. Counts every `round-NNN(-slug)` directory on disk,
// including reserved-but-not-yet-persisted ones — otherwise ReserveRound
// would deadlock retrying the same number after creating the dir but
// before SaveRound finishes.
func (s *Store) NextRoundNumber(initiativeName string) (int, error) {
	dir := s.FeedbackDir(initiativeName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 1, nil
		}
		return 0, fmt.Errorf("read feedback dir: %w", err)
	}
	max := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		m := roundDirRE.FindStringSubmatch(entry.Name())
		if m == nil {
			continue
		}
		var n int
		fmt.Sscanf(m[1], "%d", &n)
		if n > max {
			max = n
		}
	}
	return max + 1, nil
}

// ReserveRound atomically claims the next available round number by
// retrying NextRoundNumber + os.Mkdir until Mkdir succeeds — `os.Mkdir`
// (unlike MkdirAll) fails when the target already exists, which gives us
// a filesystem-level CAS. The returned dir is empty but exists, so the
// caller can immediately drop attachments into it knowing no concurrent
// submission will reuse the same number.
//
// `attempts` caps the retry loop so a runaway listing failure can't
// spin forever — a healthy contention burst settles in <10 attempts.
func (s *Store) ReserveRound(initiativeName, slug string) (number int, dir string, err error) {
	feedbackDir := s.FeedbackDir(initiativeName)
	if err := os.MkdirAll(feedbackDir, 0o755); err != nil {
		return 0, "", fmt.Errorf("create feedback dir: %w", err)
	}
	const maxAttempts = 50
	for i := 0; i < maxAttempts; i++ {
		n, err := s.NextRoundNumber(initiativeName)
		if err != nil {
			return 0, "", err
		}
		candidate := s.RoundDir(initiativeName, n, slug)
		switch mkErr := os.Mkdir(candidate, 0o755); {
		case mkErr == nil:
			return n, candidate, nil
		case os.IsExist(mkErr):
			// Concurrent reservation already took this slot; bump and retry.
			continue
		default:
			return 0, "", fmt.Errorf("reserve round dir: %w", mkErr)
		}
	}
	return 0, "", fmt.Errorf("could not reserve round number after %d attempts", maxAttempts)
}

// ListRounds scans the feedback folder and returns every round in ascending
// number order. Malformed folder names (not matching round-NNN(-slug))
// are silently skipped so a manual on-disk edit doesn't break listings.
func (s *Store) ListRounds(initiativeName string) ([]Round, error) {
	dir := s.FeedbackDir(initiativeName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read feedback dir: %w", err)
	}
	out := make([]Round, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		m := roundDirRE.FindStringSubmatch(entry.Name())
		if m == nil {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name(), feedbackJSONName))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read %s: %w", entry.Name(), err)
		}
		round, err := decodeRound(data)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", entry.Name(), err)
		}
		out = append(out, round)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Number < out[j].Number })
	return out, nil
}

// LoadRound reads a single round by number. Slug is optional; when empty,
// the store picks the one matching folder.
func (s *Store) LoadRound(initiativeName string, number int) (Round, error) {
	dir := s.FeedbackDir(initiativeName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return Round{}, ErrRoundNotFound
		}
		return Round{}, fmt.Errorf("read feedback dir: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		m := roundDirRE.FindStringSubmatch(entry.Name())
		if m == nil {
			continue
		}
		var n int
		fmt.Sscanf(m[1], "%d", &n)
		if n != number {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name(), feedbackJSONName))
		if err != nil {
			if os.IsNotExist(err) {
				return Round{}, ErrRoundNotFound
			}
			return Round{}, fmt.Errorf("read round: %w", err)
		}
		return decodeRound(data)
	}
	return Round{}, ErrRoundNotFound
}

// SaveRound persists a round to disk atomically. Creates the round folder
// if it does not exist. Idempotent — safe to call repeatedly with the
// same round (e.g., after appending a thread message).
func (s *Store) SaveRound(round Round) error {
	if strings.TrimSpace(round.InitiativeName) == "" {
		return fmt.Errorf("round.InitiativeName is required")
	}
	if round.Number <= 0 {
		return fmt.Errorf("round.Number must be >= 1, got %d", round.Number)
	}
	dir := s.RoundDir(round.InitiativeName, round.Number, round.Slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create round dir: %w", err)
	}
	return storage.WriteJSONAtomic(filepath.Join(dir, feedbackJSONName), round)
}

// DeleteRound removes a round from disk. Used by tests and by the
// `dismiss` path when the user abandons a round before any state was worth
// keeping. Safe to call for a non-existent round.
func (s *Store) DeleteRound(initiativeName string, number int) error {
	dir := s.FeedbackDir(initiativeName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		m := roundDirRE.FindStringSubmatch(entry.Name())
		if m == nil {
			continue
		}
		var n int
		fmt.Sscanf(m[1], "%d", &n)
		if n != number {
			continue
		}
		return os.RemoveAll(filepath.Join(dir, entry.Name()))
	}
	return nil
}

func decodeRound(data []byte) (Round, error) {
	var r Round
	if err := json.Unmarshal(data, &r); err != nil {
		return Round{}, err
	}
	return r, nil
}
