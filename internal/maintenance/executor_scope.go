package maintenance

import (
	"context"
	"fmt"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/supervision"
)

// ExecutorScopeRequest is a read-only exclusion request. Labels must come from
// the durable run, never from an empty live-run list. It does not authorize
// recovery: the caller must hold its admission/continuation fence across this
// observation and dispatch. All fields are non-secret process identities.
type ExecutorScopeRequest struct {
	Executors []ExecutorScopeRef `json:"executors"`
}

type ExecutorScopeRef struct {
	RunID     string     `json:"runId"`
	Tag       string     `json:"tag"`
	LegacyTag string     `json:"legacyTag"`
	PID       int        `json:"pid"`
	PGID      int        `json:"pgid"`
	StartedAt *time.Time `json:"startedAt,omitempty"`
	EndedAt   *time.Time `json:"endedAt,omitempty"`
}

type ExecutorScopeEvidence struct {
	RunID   string   `json:"runId"`
	State   string   `json:"state"`
	PIDs    []int    `json:"pids"`
	Reasons []string `json:"reasons"`
}

type ExecutorScopeReport struct {
	SchemaVersion string                  `json:"schemaVersion"`
	Complete      bool                    `json:"complete"`
	Executors     []ExecutorScopeEvidence `json:"executors"`
}

// Only these labels survive reading a process environment. In particular no
// identity token, credential, command line or arbitrary environment is retained.
type scopeIdentity struct {
	RunID, Scenario, Variant, InstanceID string
	Tags                                 []string
	TagLabels                            map[string]string
}

var scopeProcessTable = listProcessTableContext
var scopeReadIdentity = readScopeIdentity
var scopeProcessBirths = (supervision.NativeProcessTableSource{}).ProcessesContext

const (
	ExecutorScopeMaxReferences = 16384
	ExecutorScopeMaxBytes      = 8 << 20
	ExecutorScopeTimeout       = 4 * time.Second
	executorScopeSampleBudget  = 65536
)

// Index only requested identities. Samples are bounded independently of host
// size and shared tags: matching is O(references + processes + labels), plus a
// fixed total evidence-sample budget, not references times host processes.
type executorScopeIndex struct {
	runs        map[string][]int
	tags        map[string][]int
	serviceTags map[string][]int
	groups      map[int]int
}

func indexExecutorScopes(ctx context.Context, refs []ExecutorScopeRef, entries map[int]processTableEntry, identities map[int]scopeIdentity, inherited map[int]map[string]bool, sampleLimit int) (executorScopeIndex, error) {
	index := executorScopeIndex{runs: make(map[string][]int, len(refs)), tags: make(map[string][]int, 2*len(refs)), serviceTags: make(map[string][]int, 2*len(refs)), groups: make(map[int]int)}
	for _, ref := range refs {
		index.runs[ref.RunID] = nil
		index.tags[ref.Tag] = nil
		index.tags[ref.LegacyTag] = nil
		index.serviceTags[ref.Tag] = nil
		index.serviceTags[ref.LegacyTag] = nil
		if ref.PGID > 0 {
			index.groups[ref.PGID] = 0
		}
	}
	for pid, entry := range entries {
		if err := ctx.Err(); err != nil {
			return index, err
		}
		if strings.HasPrefix(entry.State, "Z") {
			continue
		}
		identity := identities[pid]
		if pids, wanted := index.runs[identity.RunID]; wanted && len(pids) < sampleLimit {
			index.runs[identity.RunID] = append(pids, pid)
		}
		for _, tag := range identity.Tags {
			if err := ctx.Err(); err != nil {
				return index, err
			}
			// Classify before sampling: many inherited service labels cannot
			// crowd a true executor out of the bounded positive evidence.
			tags := index.tags
			if inherited[pid][tag] {
				tags = index.serviceTags
			}
			if pids, wanted := tags[tag]; wanted && len(pids) < sampleLimit {
				tags[tag] = append(pids, pid)
			}
		}
		if member, wanted := index.groups[entry.PGID]; wanted && member == 0 {
			index.groups[entry.PGID] = pid
		}
	}
	return index, nil
}

func (c *Controller) ExecutorScope(parent context.Context, req ExecutorScopeRequest) (ExecutorScopeReport, error) {
	report := ExecutorScopeReport{SchemaVersion: "executor-scope-v1", Complete: true, Executors: []ExecutorScopeEvidence{}}
	if len(req.Executors) < 1 || len(req.Executors) > ExecutorScopeMaxReferences {
		return report, fmt.Errorf("executor-scope requires 1..%d executors", ExecutorScopeMaxReferences)
	}
	ctx, cancel := context.WithTimeout(parent, ExecutorScopeTimeout)
	defer cancel()
	report.Executors = make([]ExecutorScopeEvidence, 0, len(req.Executors))
	seen := map[string]bool{}
	for _, ref := range req.Executors {
		if strings.TrimSpace(ref.RunID) == "" || ref.Tag == "" || ref.LegacyTag == "" || ref.PID < 0 || ref.PGID < 0 ||
			len(ref.RunID) > 128 || len(ref.Tag) > 256 || len(ref.LegacyTag) > 256 || seen[ref.RunID] {
			return report, fmt.Errorf("executor-scope requires unique runId, current and legacy tags, and nonnegative pid/pgid")
		}
		seen[ref.RunID] = true
		if ref.StartedAt != nil && ref.EndedAt != nil && ref.EndedAt.Before(*ref.StartedAt) {
			return report, fmt.Errorf("executor-scope execution interval is reversed")
		}
		report.Executors = append(report.Executors, ExecutorScopeEvidence{RunID: ref.RunID, State: "absent", PIDs: []int{}, Reasons: []string{}})
	}
	beforeBirths, beforeErr := scopeProcessBirths(ctx)
	entries, identities, err := observeScopeProcesses(ctx)
	afterBirths, afterErr := scopeProcessBirths(ctx)
	var inherited map[int]map[string]bool
	if beforeErr == nil && afterErr == nil && err == nil {
		// This is optional evidence for removing inherited matches, not a
		// prerequisite for the process scan. On lookup failure retain every
		// original match; unrelated absent references remain provably absent.
		// Bound optional work independently of the overall observation budget.
		proofCtx, cancelProof := context.WithTimeout(ctx, time.Second)
		var proofErr error
		inherited, proofErr = c.inheritedServiceTags(proofCtx, req.Executors, entries, identities, beforeBirths, afterBirths)
		cancelProof()
		if proofErr != nil {
			inherited = nil
		}
	}
	if ctx.Err() != nil {
		err = fmt.Errorf("scope observation deadline or cancellation")
	}
	sampleLimit := min(64, executorScopeSampleBudget/len(req.Executors))
	index, indexErr := indexExecutorScopes(ctx, req.Executors, entries, identities, inherited, sampleLimit)
	if indexErr != nil {
		err = fmt.Errorf("scope observation deadline or cancellation")
	}
	// A different process incarnation across the observation is uncertainty,
	// not a reusable timestamp attached to an unrelated PID.
	bornOutside := func(pid int, ref ExecutorScopeRef) bool {
		before, beforeOK := beforeBirths[pid]
		after, afterOK := afterBirths[pid]
		if beforeErr != nil || afterErr != nil || !beforeOK || !afterOK || before.StartedAt.IsZero() || !before.StartedAt.Equal(after.StartedAt) {
			return false
		}
		const allowance = 2 * time.Second
		return (ref.StartedAt != nil && before.StartedAt.Before(ref.StartedAt.Add(-allowance))) ||
			(ref.EndedAt != nil && before.StartedAt.After(ref.EndedAt.Add(allowance)))
	}
	for i, ref := range req.Executors {
		evidence := &report.Executors[i]
		matches := [][]int{index.runs[ref.RunID], index.tags[ref.Tag], index.tags[ref.LegacyTag]}
		// Only ended executions can use service-label exclusion. A live or
		// undated run retains all labels until its owner supplies an interval.
		if ref.EndedAt == nil {
			matches = append(matches, index.serviceTags[ref.Tag], index.serviceTags[ref.LegacyTag])
		}
		for _, pids := range matches {
			for _, pid := range pids {
				if len(evidence.PIDs) < sampleLimit && !slices.Contains(evidence.PIDs, pid) {
					evidence.PIDs = append(evidence.PIDs, pid)
				}
			}
		}
		if len(evidence.PIDs) > 0 {
			evidence.State = "present"
		} else {
			ambiguous := 0
			if entry, exists := entries[ref.PID]; exists && ref.PID > 0 && !strings.HasPrefix(entry.State, "Z") && !bornOutside(ref.PID, ref) {
				ambiguous = ref.PID
			}
			// A later-born member does not prove the old group was replaced.
			if member := index.groups[ref.PGID]; member > 0 && !bornOutside(ref.PGID, ref) {
				ambiguous = member
			}
			if ambiguous > 0 {
				evidence.State = "unknown"
				evidence.Reasons = append(evidence.Reasons, fmt.Sprintf("pid/group %d lacks run identity or reuse proof", ambiguous))
			}
		}
		slices.Sort(evidence.PIDs)
		if err != nil {
			report.Complete = false
			if evidence.State != "present" {
				evidence.State = "unknown"
			}
			evidence.Reasons = append(evidence.Reasons, err.Error())
		}
		if evidence.State == "unknown" {
			report.Complete = false
		}
	}
	// The deadline covers indexing and matching too. Never emit a previously
	// complete-looking absence after that observation budget expired.
	if ctx.Err() != nil {
		report.Complete = false
		for i := range report.Executors {
			if report.Executors[i].State == "absent" {
				report.Executors[i].State = "unknown"
				report.Executors[i].Reasons = []string{"scope observation deadline or cancellation"}
			}
		}
	}
	return report, nil
}

// Reuse the maintenance process table and platform process handles, not a
// scenario-private /proc walker. A second table checks births during the read;
// new entries are inspected in at most three passes. Persistent churn, hidden
// same-user identity, cancellation, and unsupported hosts are all unknown.
func observeScopeProcesses(ctx context.Context) (map[int]processTableEntry, map[int]scopeIdentity, error) {
	identities := map[int]scopeIdentity{}
	observed := map[int]processTableEntry{}
	if runtime.GOOS != "linux" {
		return observed, identities, fmt.Errorf("complete managed executor identity inspection is unsupported on this host")
	}
	for pass := 0; pass < 3; pass++ {
		if ctx.Err() != nil {
			return observed, identities, fmt.Errorf("scope observation deadline or cancellation")
		}
		table, err := scopeProcessTable(ctx)
		if err != nil {
			return observed, identities, fmt.Errorf("process table unavailable")
		}
		pending := false
		for pid, entry := range table {
			if err := ctx.Err(); err != nil {
				return observed, identities, fmt.Errorf("scope observation deadline or cancellation")
			}
			if old, exists := observed[pid]; !exists || old.PGID != entry.PGID || old.PPID != entry.PPID || old.Cgroup != entry.Cgroup {
				pending = true
			}
			observed[pid] = entry
			if strings.HasPrefix(entry.State, "Z") {
				continue
			}
			identity, err := scopeReadIdentity(entry)
			if err != nil {
				return observed, identities, fmt.Errorf("process %d identity unavailable", pid)
			}
			identities[pid] = identity
		}
		if !pending {
			return observed, identities, nil
		}
	}
	return observed, identities, fmt.Errorf("process inventory changed throughout bounded observation")
}
