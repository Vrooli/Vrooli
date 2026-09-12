package execution

import (
	"errors"
	"sort"
	"strings"
	"sync"

	"swarm-manager/internal/runtimepaths"
	"swarm-manager/internal/storage"
)

// errEngagementConflict is the sentinel for a per-scenario exclusivity violation
// (a scenario already engaged under a different owner). Wrapped into a 409 by
// the start-time gate so callers can distinguish it from other failures.
var errEngagementConflict = errors.New("scenario already has an open baseline engagement under a different owner")

// Baseline Modes engagement ownership (shadow-mode rework — plan P-b).
//
// An engagement is the git-free restore point + @shadow instance that isolates
// a live scenario from an in-progress candidate. The plan's contract (§8):
//
//	"Engagement keyed by scenario; owned by item/initiative-step. A scenario has
//	 at most one open engagement (one working tree → one candidate)."
//
// Today every execution path (the member-item workflow strategy and every
// registered operating mode, e.g. holistic-loop / phased-plan-drain)
// executes and reviews per backlog item — initiative modes queue items but each
// still reaches review-decide independently — so the owner is the backlog item.
// The owner holds a SET of per-scenario engagements that survives the main run,
// every fixup, and the gap until review-decide closes it. This is deliberately
// NOT on the execution Record (which is per-run): keying by owner is what lets a
// fixup transparently inherit the set and review-decide find it.

// engagementMode is the Baseline Modes mode swarm-manager opens engagements in.
// Shadow mode runs the candidate on an @shadow instance while the live instance
// keeps serving the captured baseline (the isolation floor, plan P-a). This is
// the single source of truth for the mode and replaces the Part-28 live-mode v1.
const engagementMode = "shadow"

// ExclusivityPolicy decides what happens when an owner wants to engage a
// scenario already engaged under a DIFFERENT owner. It is a named policy (not an
// inline conditional) so switching the project from block-at-start to queueing
// later is a one-value change — see the §8a control-levers contract.
type ExclusivityPolicy int

const (
	// ExclusivityBlockAtStart refuses to start an owner whose projected scope
	// intersects another owner's open engagement (user decision 2026-06-06).
	ExclusivityBlockAtStart ExclusivityPolicy = iota
	// ExclusivityQueue would defer the owner until the conflict clears. Not yet
	// wired; present so the policy is a value, not a hardcoded branch.
	ExclusivityQueue
)

// engagementExclusivityPolicy is the active policy. The plan locks block-at-start;
// this constant is the one place to change it.
const engagementExclusivityPolicy = ExclusivityBlockAtStart

// EngagementCloseDecision is the terminal verdict applied to an owner's whole
// engagement set at review-decide (plan P-c). The backlog→execution adapter maps
// the review decision onto it (accept→Promote, fail/reject→Abandon,
// followup→Leave) so the execution package never imports backlog statuses.
type EngagementCloseDecision int

const (
	// EngagementLeaveOpen keeps the set open (followup: work continues under the
	// same owner, the next run's hold expands it).
	EngagementLeaveOpen EngagementCloseDecision = iota
	// EngagementPromote blesses every candidate into its baseline (accept).
	EngagementPromote
	// EngagementAbandon discards every candidate, restoring each baseline (reject).
	EngagementAbandon
)

// EngagementSet is one owner's open Baseline Modes engagements: scenario → mode.
type EngagementSet struct {
	Owner       string            `json:"owner"`
	Engagements map[string]string `json:"engagements"`
	OpenedAt    string            `json:"opened_at"`
	UpdatedAt   string            `json:"updated_at"`
}

// scenarios returns the engaged scenario names, sorted for deterministic order.
func (e EngagementSet) scenarios() []string {
	out := make([]string, 0, len(e.Engagements))
	for scenario := range e.Engagements {
		out = append(out, scenario)
	}
	sort.Strings(out)
	return out
}

// ownerKeyFor is the SINGLE SOURCE OF TRUTH for "what owns an engagement set."
// Every current operating mode reviews per backlog item, so the owner key is the
// item ref "kind/name". If initiative-step ownership is ever needed, only this
// function changes — no caller re-derives ownership.
func ownerKeyFor(kind, name string) string {
	return strings.TrimSpace(kind) + "/" + strings.TrimSpace(name)
}

// ownerKeyForRecord derives the engagement owner key from an execution Record.
func ownerKeyForRecord(r Record) string {
	return ownerKeyFor(r.BacklogKind, r.BacklogName)
}

// EngagementStore persists owner → EngagementSet as a single JSON map. It is
// self-locking (load-modify-save under its own mutex) so it can be called both
// under and outside the execution service mutex without lost updates.
type EngagementStore struct {
	path string
	mu   sync.Mutex
}

// NewEngagementStore creates an engagement-owner store at the given path,
// defaulting to the runtime state dir when empty.
func NewEngagementStore(path string) *EngagementStore {
	if strings.TrimSpace(path) == "" {
		if resolved, err := runtimepaths.StatePath("engagement-owners.json"); err == nil {
			path = resolved
		}
	}
	return &EngagementStore{path: path}
}

func (s *EngagementStore) loadLocked() (map[string]EngagementSet, error) {
	sets := map[string]EngagementSet{}
	if strings.TrimSpace(s.path) == "" {
		return sets, nil
	}
	if _, err := storage.ReadJSON(s.path, &sets); err != nil {
		return nil, err
	}
	if sets == nil {
		sets = map[string]EngagementSet{}
	}
	return sets, nil
}

func (s *EngagementStore) saveLocked(sets map[string]EngagementSet) error {
	if strings.TrimSpace(s.path) == "" {
		return nil
	}
	return storage.WriteJSONAtomic(s.path, sets)
}

// Get returns the engagement set for an owner, and whether one exists.
func (s *EngagementStore) Get(owner string) (EngagementSet, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sets, err := s.loadLocked()
	if err != nil {
		return EngagementSet{}, false, err
	}
	set, ok := sets[owner]
	return set, ok, nil
}

// HolderOf reports which owner (if any) currently holds an open engagement for
// the given scenario, and in which mode. This is the exclusivity lookup: a
// scenario has at most one open engagement across all owners.
func (s *EngagementStore) HolderOf(scenario string) (owner, mode string, ok bool, err error) {
	scenario = strings.TrimSpace(scenario)
	s.mu.Lock()
	defer s.mu.Unlock()
	sets, loadErr := s.loadLocked()
	if loadErr != nil {
		return "", "", false, loadErr
	}
	for ownerKey, set := range sets {
		if m, held := set.Engagements[scenario]; held {
			return ownerKey, m, true, nil
		}
	}
	return "", "", false, nil
}

// Add merges scenario→mode entries into an owner's set, creating it if needed,
// and returns the resulting set. Used to open/expand engagements from the actual
// diff at the pre-merge hold.
func (s *EngagementStore) Add(owner string, scenarioModes map[string]string, now string) (EngagementSet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sets, err := s.loadLocked()
	if err != nil {
		return EngagementSet{}, err
	}
	set, ok := sets[owner]
	if !ok {
		set = EngagementSet{Owner: owner, Engagements: map[string]string{}, OpenedAt: now}
	}
	if set.Engagements == nil {
		set.Engagements = map[string]string{}
	}
	for scenario, mode := range scenarioModes {
		scenario = strings.TrimSpace(scenario)
		if scenario == "" {
			continue
		}
		set.Engagements[scenario] = mode
	}
	set.UpdatedAt = now
	sets[owner] = set
	if err := s.saveLocked(sets); err != nil {
		return EngagementSet{}, err
	}
	return set, nil
}

// Remove pops an owner's engagement set (used when review-decide closes it),
// returning the removed set and whether one existed.
func (s *EngagementStore) Remove(owner string) (EngagementSet, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sets, err := s.loadLocked()
	if err != nil {
		return EngagementSet{}, false, err
	}
	set, ok := sets[owner]
	if !ok {
		return EngagementSet{}, false, nil
	}
	delete(sets, owner)
	if err := s.saveLocked(sets); err != nil {
		return EngagementSet{}, false, err
	}
	return set, true, nil
}
