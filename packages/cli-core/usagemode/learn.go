package usagemode

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Mode kinds. An absent kind reads as save.
const (
	KindSave  = "save"
	KindLearn = "learn"
)

// Arms of a Learn comparison. The incumbent is what the role runs on today; the
// challenger is the lower model being tested against it.
const (
	ArmIncumbent  = "incumbent"
	ArmChallenger = "challenger"
)

// Environment markers the launcher sets so a session's arm is known without
// reading its transcript.
const (
	EnvArm          = "VROOLI_USAGE_ARM"
	EnvFullPower    = "VROOLI_FULL_POWER"
	ArmExceptionKey = "VROOLI_USAGE_ARM_EXCEPTION"
)

// Exceptions recorded when an operator session on a challenger day did not run
// on the challenger. Analysis stays by assigned arm (intention to treat).
const (
	ExceptionFullPower     = "full_power"
	ExceptionExplicitModel = "explicit_model"
)

const (
	// DefaultSplitPercent is the challenger share: 50/50 is the fastest design.
	DefaultSplitPercent = 50
	// DefaultBlockDays is the length of one randomized session block.
	DefaultBlockDays = 1
)

// Learn is the part of a mode that makes it a controlled comparison. The
// challenger's settings live here and not in Mode.Runners, so a reader that
// does not know Learn finds no settings and leaves the launch alone.
type Learn struct {
	// SplitPercent is the share of work sent to the challenger, 1 to 99.
	SplitPercent int `json:"split_percent,omitempty"`
	// Runners holds the challenger's concrete settings per runner: ModelMap maps
	// the incumbent model to the challenger model.
	Runners map[string]RunnerSettings `json:"runners"`
	// Sessions opts operator sessions in as randomized day blocks.
	Sessions *SessionBlocks `json:"sessions,omitempty"`
	// Paused stops new assignments while the forecast says a window will run
	// out. Work already assigned keeps its arm.
	Paused       bool       `json:"paused,omitempty"`
	PausedReason string     `json:"paused_reason,omitempty"`
	PausedAt     *time.Time `json:"paused_at,omitempty"`

	extra map[string]json.RawMessage
}

// SessionBlocks is the opt-in for operator sessions.
type SessionBlocks struct {
	OptIn     bool      `json:"opt_in"`
	BlockDays int       `json:"block_days,omitempty"`
	OptedInBy string    `json:"opted_in_by,omitempty"`
	OptedInAt time.Time `json:"opted_in_at,omitempty"`
}

// Split returns the challenger share in (0,1).
func (l Learn) Split() float64 {
	percent := l.SplitPercent
	if percent <= 0 || percent >= 100 {
		percent = DefaultSplitPercent
	}
	return float64(percent) / 100
}

func (l Learn) blockDays() int {
	if l.Sessions == nil || l.Sessions.BlockDays <= 0 {
		return DefaultBlockDays
	}
	return l.Sessions.BlockDays
}

// IsLearn reports a learn-kind mode.
func (m Mode) IsLearn() bool { return m.Kind == KindLearn }

// ChallengerSettings returns the challenger's settings for a runner.
func (m Mode) ChallengerSettings(runner string) (RunnerSettings, bool) {
	if m.Learn == nil {
		return RunnerSettings{}, false
	}
	want := NormalizeRunner(runner)
	for key, settings := range m.Learn.Runners {
		if NormalizeRunner(key) == want {
			return settings, true
		}
	}
	return RunnerSettings{}, false
}

// RoleExempt reports a role that never joins a comparison, whatever the
// mode's scope: exempt roles stay out.
func (m Mode) RoleExempt(role string) bool {
	if role == "" {
		return false
	}
	for _, list := range [][]string{m.ExemptRoles, DefaultExemptRoles} {
		for _, exempt := range list {
			if exempt == role {
				return true
			}
		}
	}
	return false
}

// SessionArm is the arm of a randomized day block, fixed by the mode id and
// the block's first UTC day alone, so the launcher, the ledger and a reader
// with no database all agree and nothing needs to be stored to know it.
func SessionArm(modeID string, at time.Time, split float64, blockDays int) string {
	if blockDays <= 0 {
		blockDays = DefaultBlockDays
	}
	if split <= 0 || split >= 1 {
		split = DefaultSplitPercent / 100.0
	}
	block := at.UTC().Unix() / 86400 / int64(blockDays)
	digest := sha256.Sum256([]byte(modeID + "|day-block|" + strconv.FormatInt(block, 10)))
	value := float64(binary.BigEndian.Uint64(digest[:8])) / float64(math.MaxUint64)
	if value < split {
		return ArmChallenger
	}
	return ArmIncumbent
}

// SessionArm is the mode's arm for an operator session starting at at.
func (m Mode) SessionArm(at time.Time) string {
	if m.Learn == nil {
		return ArmIncumbent
	}
	return SessionArm(m.ID, at, m.Learn.Split(), m.Learn.blockDays())
}

// learnApplies is the shared eligibility of a learn mode at a time.
func (m Mode) learnApplies(now time.Time) bool {
	return m.IsLearn() && m.Learn != nil && !m.Learn.Paused && m.Active(now)
}

// ResolveLearn returns the learn mode covering a managed run's role, runner
// and profile. A save mode that applies takes priority and leaves the run
// out of the comparison; an exempt role, a paused mode and a role the mode is
// not scoped to are never randomized.
func ResolveLearn(q Query) (Mode, bool) {
	return resolveLearnFrom(Read(q.Now), q)
}

func resolveLearnFrom(modes []Mode, q Query) (Mode, bool) {
	if _, saving := resolveFrom(modes, q); saving {
		return Mode{}, false
	}
	var best Mode
	found := false
	for _, mode := range modes {
		if !mode.learnApplies(q.Now) || mode.Scope.Role == "" || mode.Scope.Role != q.Role || mode.RoleExempt(q.Role) {
			continue
		}
		if mode.Scope.Profile != "" && mode.Scope.Profile != q.Profile {
			continue
		}
		if mode.Scope.Runner != "" && NormalizeRunner(mode.Scope.Runner) != NormalizeRunner(q.Runner) {
			continue
		}
		if _, ok := mode.ChallengerSettings(q.Runner); !ok {
			continue
		}
		if !found || mode.StartedAt.After(best.StartedAt) {
			best, found = mode, true
		}
	}
	return best, found
}

// ResolveLearnSession returns the learn mode an operator session joins: only a
// mode whose operator opted sessions in, and never while a save mode applies.
func ResolveLearnSession(runner string, now time.Time) (Mode, bool) {
	modes := Read(now)
	if _, saving := resolveFrom(modes, Query{Runner: runner, Now: now}); saving {
		return Mode{}, false
	}
	var best Mode
	found := false
	for _, mode := range modes {
		if !mode.learnApplies(now) || mode.Learn.Sessions == nil || !mode.Learn.Sessions.OptIn {
			continue
		}
		if mode.Scope.Runner != "" && NormalizeRunner(mode.Scope.Runner) != NormalizeRunner(runner) {
			continue
		}
		if _, ok := mode.ChallengerSettings(runner); !ok {
			continue
		}
		if !found || mode.StartedAt.After(best.StartedAt) {
			best, found = mode, true
		}
	}
	return best, found
}

// LaunchMarker is one randomized operator-session launch, appended by the
// launcher so the ledger can tell a session's arm and any exception.
type LaunchMarker struct {
	At        time.Time `json:"at"`
	ModeID    string    `json:"mode_id"`
	Runner    string    `json:"runner"`
	Arm       string    `json:"arm"`
	Exception string    `json:"exception,omitempty"`
}

const (
	markerFile    = "arms.jsonl"
	markerMaxSize = 512 * 1024
)

func markerPath() string {
	path := Path()
	if path == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(path), markerFile)
}

// RecordLaunch appends a marker. It never fails the launch.
func RecordLaunch(marker LaunchMarker) {
	path := markerPath()
	if path == "" || marker.ModeID == "" {
		return
	}
	defer func() { _ = recover() }()
	if info, err := os.Stat(path); err == nil && info.Size() > markerMaxSize {
		pruneMarkers(path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer file.Close()
	raw, err := json.Marshal(marker)
	if err == nil {
		_, _ = file.Write(append(raw, '\n'))
	}
}

func pruneMarkers(path string) {
	markers := ReadLaunchesFile(path, time.Time{})
	if len(markers) > 1000 {
		markers = markers[len(markers)-1000:]
	}
	var out strings.Builder
	for _, marker := range markers {
		raw, _ := json.Marshal(marker)
		out.Write(raw)
		out.WriteByte('\n')
	}
	_ = os.WriteFile(path, []byte(out.String()), 0o600)
}

// ReadLaunches returns the markers at or after since. It never fails.
func ReadLaunches(since time.Time) []LaunchMarker {
	path := markerPath()
	if path == "" {
		return nil
	}
	return ReadLaunchesFile(path, since)
}

// ReadLaunchesFile is ReadLaunches for an explicit path.
func ReadLaunchesFile(path string, since time.Time) []LaunchMarker {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()
	var out []LaunchMarker
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var marker LaunchMarker
		if json.Unmarshal(scanner.Bytes(), &marker) != nil || marker.ModeID == "" || marker.At.Before(since) {
			continue
		}
		out = append(out, marker)
	}
	return out
}
