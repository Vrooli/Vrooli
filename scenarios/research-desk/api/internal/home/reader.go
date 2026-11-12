package home

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/schedule"
)

// Config locates the research home and the Codex session records.
type Config struct {
	// ResearchDir is the research home, ~/.vrooli/research in production.
	ResearchDir string
	// CodexDir holds one Codex home per Web Console session,
	// ~/.vrooli/state/vrooli/web-console/sessions/codex in production.
	CodexDir string
	// OtherCodexHomes are glob patterns naming the other Codex homes the desk
	// can read (the operator's ~/.codex, Agent Manager run homes). Their goals
	// are read only to prove a packet was never started.
	OtherCodexHomes []string
	// UserHome expands "~/" spellings of the charter path; may be empty.
	UserHome string
}

// DefaultConfig returns the production locations under the user's home.
func DefaultConfig(userHome string) Config {
	return Config{
		ResearchDir: filepath.Join(userHome, ".vrooli", "research"),
		CodexDir:    filepath.Join(userHome, ".vrooli", "state", "vrooli", "web-console", "sessions", "codex"),
		OtherCodexHomes: []string{
			filepath.Join(userHome, ".codex"),
			filepath.Join(userHome, ".vrooli", "state", "vrooli", "agent-manager", "runs", "*", "codex"),
		},
		UserHome: userHome,
	}
}

// Deps are the reader's seams (docs/internal/SEAMS.md).
type Deps struct {
	Clock schedule.NowSource
	Procs Procs
	// Missions runs mission_check.py; nil leaves mission checks unknown.
	Missions MissionChecker
	// Ceilings reads the read-set ceilings; nil or an error leaves them unknown.
	Ceilings func() (*Ceilings, error)
}

// Reader builds consistent snapshots of the research home.
type Reader struct {
	cfg   Config
	deps  Deps
	codex *codexIndex
	// others caches the goals of OtherCodexHomes apart from codex, whose
	// read forgets every file outside the Web Console session folder.
	others *codexIndex
	walks  *walkCache
}

// NewReader returns a Reader.
func NewReader(cfg Config, deps Deps) *Reader {
	return &Reader{cfg: cfg, deps: deps, codex: newCodexIndex(), others: newCodexIndex(), walks: &walkCache{}}
}

// Config returns the reader's locations.
func (r *Reader) Config() Config { return r.cfg }

// LimitsLens is the name of the lens for the skill's limits.json.
const LimitsLens = "limits.json"

// Lens is one source the projection reads, and whether it was readable.
type Lens struct {
	Name      string
	Available bool
	// Reason says what is wrong in words, never a command.
	Reason string
	// Command is a command-line step that would make it available, for the
	// CLI; the UI ignores it.
	Command string
}

// Snapshot is one consistent read of the research home.
type Snapshot struct {
	Now         time.Time
	ResearchDir string
	Commissions []*Commission // newest last write first
	Topics      []*Topic      // newest activity first
	Lenses      []Lens
	// Ceilings are nil when limits.json could not be read.
	Ceilings *Ceilings
	// Attention and AttentionLegacy are ATTENTION.md's "## Actionable now".
	Attention       []NeedsYouItem
	AttentionLegacy []Legacy
	Reviews         []*Review
	// Unreadable lists ATTENTION.md and the reviews folder when they exist
	// but cannot be read; every commission, topic and review
	// keeps its own list too, and AllUnreadable gathers them.
	Unreadable []Evidence
}

// Options selects the optional, slower parts of a snapshot.
type Options struct {
	// MissionCheck runs mission_check.py for the topics' workstream summary.
	MissionCheck bool
}

// Snapshot reads every commission and topic. It fails only when the research
// home itself cannot be read; every other gap becomes an unknown value.
func (r *Reader) Snapshot(ctx context.Context, opts Options) (*Snapshot, error) {
	now := r.deps.Clock.Now()
	index, runs, err := readTopics(r.cfg.ResearchDir)
	topics := index.topics
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v (set the research home or create it)", ErrNoResearchHome, r.cfg.ResearchDir, err)
	}
	cohortsDir := filepath.Join(r.cfg.ResearchDir, "cohorts")
	entries, err := os.ReadDir(cohortsDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s: %v", ErrNoResearchHome, cohortsDir, err)
	}

	var ids []string
	for _, entry := range entries {
		if entry.IsDir() && entry.Name()[0] != '.' {
			ids = append(ids, entry.Name())
		}
	}
	owned := map[string][]RunFolder{}
	var unowned []RunFolder
	for name, list := range runs {
		if owner := runOwner(name, ids); owner != "" {
			owned[owner] = append(owned[owner], list...)
		} else {
			unowned = append(unowned, list...)
		}
	}
	walk := func(dir string, cutoff time.Time) newestWrite { return r.walks.newest(dir, "", cutoff, now) }
	for i := range unowned {
		unowned[i].LastWrite = walk(unowned[i].Path, time.Time{}).All
	}

	records := r.codex.read(ctx, r.cfg.CodexDir)
	processes, procsKnown, procsReason := r.deps.Procs.Codex()
	env := deriveEnv{
		now:         now,
		researchDir: r.cfg.ResearchDir,
		codexDir:    filepath.Clean(r.cfg.CodexDir),
		userHome:    r.cfg.UserHome,
		records:     records,
		windows:     windowsOpen(processes),
		procsKnown:  procsKnown,
		procsReason: procsReason,
		walk:        walk,
		others:      sync.OnceValue(func() OtherGoals { return r.others.others(ctx, r.cfg.OtherCodexHomes) }),
	}

	snap := &Snapshot{Now: now, ResearchDir: r.cfg.ResearchDir}
	for _, id := range ids {
		runs := owned[id]
		sort.Slice(runs, func(i, j int) bool { return runs[i].Path < runs[j].Path })
		c := readCommission(r.cfg.ResearchDir, id, runs, index.of)
		c.derive(env)
		snap.Commissions = append(snap.Commissions, c)
	}
	sortCommissions(snap.Commissions)
	attention := filepath.Join(r.cfg.ResearchDir, AttentionFile)
	if snap.Attention, snap.AttentionLegacy, err = readNeedsYou(attention, AttentionNeedsSection); err != nil {
		snap.Unreadable = append(snap.Unreadable, Unreadable(attention, err))
	}
	if snap.Reviews, err = readReviews(r.cfg.ResearchDir, index.of); err != nil {
		snap.Unreadable = append(snap.Unreadable, Unreadable(filepath.Join(r.cfg.ResearchDir, ReviewsDir), err))
	}
	linkTopics(topics, snap.Commissions, unowned, snap.Reviews)
	snap.Topics = topics

	var missionLens *Lens
	if opts.MissionCheck {
		missionLens = &Lens{Name: "mission_check.py"}
		if r.deps.Missions == nil {
			missionLens.Reason = "no mission checker configured"
			applyMissionCheck(topics, nil, "", now, fmt.Errorf("%s", missionLens.Reason))
		} else {
			output, source, err := r.deps.Missions.MissionCheck(ctx, r.cfg.ResearchDir)
			applyMissionCheck(topics, output, source, now, err)
			missionLens.Available, missionLens.Reason = err == nil, source
			if err != nil {
				missionLens.Reason = err.Error()
			}
		}
	}

	codexLens := Lens{Name: "codex sessions", Available: records.Known, Reason: r.cfg.CodexDir}
	if !records.Known {
		codexLens.Reason = records.Reason
	} else if n := len(records.Unreadable); n > 0 {
		codexLens.Reason = fmt.Sprintf("%d session files unreadable, first %s", n, records.Unreadable[0])
	}
	procLens := Lens{Name: "process table", Available: procsKnown, Reason: procsReason}
	limitsLens := Lens{Name: LimitsLens, Reason: "no ceilings source configured"}
	if r.deps.Ceilings != nil {
		ceilings, err := r.deps.Ceilings()
		if err != nil {
			limitsLens.Reason = err.Error()
		} else {
			snap.Ceilings = ceilings
			limitsLens.Available, limitsLens.Reason = true, ceilings.Source
		}
	}
	snap.Lenses = []Lens{
		{Name: "research home", Available: true, Reason: r.cfg.ResearchDir},
		codexLens,
		procLens,
		limitsLens,
	}
	if missionLens != nil {
		snap.Lenses = append(snap.Lenses, *missionLens)
	}
	return snap, nil
}

// Commission returns one commission by id, or nil.
func (s *Snapshot) Commission(id string) *Commission {
	for _, c := range s.Commissions {
		if c.ID == id {
			return c
		}
	}
	return nil
}

// Find is Commission, or a not-found error that says how to list them.
func (s *Snapshot) Find(id string) (*Commission, error) {
	if c := s.Commission(id); c != nil {
		return c, nil
	}
	return nil, WithCommand(Cause(ErrNotFound, fmt.Errorf("no commission folder %q under %s/cohorts", id, s.ResearchDir)), "research-desk commissions")
}

// Review returns one review by id, or nil.
func (s *Snapshot) Review(id string) *Review {
	for _, r := range s.Reviews {
		if r.ID == id {
			return r
		}
	}
	return nil
}

// Topic returns one topic by slug, or nil.
func (s *Snapshot) Topic(name string) *Topic {
	for _, t := range s.Topics {
		if t.Slug == name || slices.Contains(t.Aliases, name) {
			return t
		}
	}
	return nil
}

// Overlap is a topic covered by two commissions whose intervals intersect;
// each intersecting pair is its own overlap.
type Overlap struct {
	Topic         string
	CommissionIDs []string
}

// Overlaps finds the pairs of commissions on one topic whose intervals
// intersect. Commissions without a known start are not placed in time.
func (s *Snapshot) Overlaps() []Overlap {
	return s.overlaps(func(*Commission) bool { return true })
}

// OpenOverlaps are the overlaps whose commissions both have a supervisor
// window open now: what overlaps today.
func (s *Snapshot) OpenOverlaps() []Overlap {
	return s.overlaps(func(c *Commission) bool { return c.State.Running() })
}

// intervalEnd is where a commission's interval ends: now while a window is
// open, else its known end, else its last write (an end that is not known is
// never stretched to now), else its start.
func (s *Snapshot) intervalEnd(c *Commission) time.Time {
	switch {
	case c.State.Running():
		return s.Now
	case c.Ended.Known:
		return c.Ended.Value
	case c.LastWrite.Known && c.LastWrite.Value.After(c.Started.Value):
		return c.LastWrite.Value
	}
	return c.Started.Value
}

func (s *Snapshot) overlaps(keep func(*Commission) bool) []Overlap {
	type interval struct {
		id         string
		start, end time.Time
	}
	byTopic := map[string][]interval{}
	for _, c := range s.Commissions {
		if !c.Started.Known || !keep(c) {
			continue
		}
		for _, ref := range c.Topics {
			byTopic[ref.Topic] = append(byTopic[ref.Topic], interval{c.ID, c.Started.Value, s.intervalEnd(c)})
		}
	}
	var out []Overlap
	for topic, list := range byTopic {
		for i := range list {
			for j := i + 1; j < len(list); j++ {
				if list[i].start.Before(list[j].end) && list[j].start.Before(list[i].end) {
					pair := []string{list[i].id, list[j].id}
					sort.Strings(pair)
					out = append(out, Overlap{Topic: topic, CommissionIDs: pair})
				}
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Topic != out[j].Topic {
			return out[i].Topic < out[j].Topic
		}
		return strings.Join(out[i].CommissionIDs, " ") < strings.Join(out[j].CommissionIDs, " ")
	})
	return out
}
