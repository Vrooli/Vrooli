package schedule

import (
	"math"
	"sort"
	"time"
)

// groupKey is the batching identity. Kind is included on purpose: two jobs that
// name the same model must not be executed as one batch if they do different
// work (an embedding batch and an annotation batch have different payloads).
// Owner and resource class are included because one capacity claim can only be
// attributed to one owner and one resource dimension.
type groupKey struct {
	Kind          string
	Owner         string
	ResourceClass ResourceClass
	Model         string
	BatchKey      string
}

// candidate is one batch group awaiting a decision.
type candidate struct {
	Key      groupKey
	Jobs     []Job
	Value    float64
	Seconds  float64
	Oldest   time.Time
	Hot      bool
	Density  float64
	Priority Priority
}

// groupCandidates groups ready jobs by (kind, model affinity, batch key).
func groupCandidates(jobs []Job) []*candidate {
	index := map[groupKey]*candidate{}
	order := make([]*candidate, 0, len(jobs))
	for _, job := range jobs {
		key := groupKey{
			Kind:          job.Kind,
			Owner:         job.Owner,
			ResourceClass: job.ResourceClass,
			Model:         job.ModelAffinity,
			BatchKey:      job.BatchKey,
		}
		group, ok := index[key]
		if !ok {
			group = &candidate{Key: key}
			index[key] = group
			order = append(order, group)
		}
		group.Jobs = append(group.Jobs, job)
		group.Value += job.Value
		group.Seconds += job.Estimate.Seconds
		if group.Oldest.IsZero() || job.NotBefore.Before(group.Oldest) {
			group.Oldest = job.NotBefore
		}
		if job.Priority.rank() > group.Priority.rank() {
			group.Priority = job.Priority
		}
	}
	for _, group := range order {
		group.Density = ValueDensity(group.Value, group.Seconds)
		sortJobs(group.Jobs)
	}
	return order
}

// sortJobs orders jobs inside a group so the most valuable, highest-priority and
// oldest work is taken first when the batch is trimmed.
func sortJobs(jobs []Job) {
	sort.SliceStable(jobs, func(i, j int) bool {
		a, b := jobs[i], jobs[j]
		if a.Priority.rank() != b.Priority.rank() {
			return a.Priority.rank() > b.Priority.rank()
		}
		aDensity := ValueDensity(a.Value, a.Estimate.Seconds)
		bDensity := ValueDensity(b.Value, b.Estimate.Seconds)
		if aDensity != bDensity {
			return aDensity > bDensity
		}
		if !a.NotBefore.Equal(b.NotBefore) {
			return a.NotBefore.Before(b.NotBefore)
		}
		return a.ID < b.ID
	})
}

// rankCandidates orders groups for admission. The order is the documented
// selection ladder: priority tier, model still resident, value density, age,
// then fair-share deficit.
func rankCandidates(groups []*candidate, budget budgetSnapshot) {
	sort.SliceStable(groups, func(i, j int) bool {
		a, b := groups[i], groups[j]
		if a.Priority.rank() != b.Priority.rank() {
			return a.Priority.rank() > b.Priority.rank()
		}
		if a.Hot != b.Hot {
			return a.Hot
		}
		if a.Density != b.Density {
			return a.Density > b.Density
		}
		if !a.Oldest.Equal(b.Oldest) {
			return a.Oldest.Before(b.Oldest)
		}
		aShare, bShare := groupShareOf(a, budget), groupShareOf(b, budget)
		if aShare != bShare {
			return aShare < bShare
		}
		return a.Key.Model < b.Key.Model
	})
}

// ownerShareOf is an owner's fraction of the seconds started in the rolling
// budget window.
func ownerShareOf(owner string, budget budgetSnapshot) float64 {
	if budget.Seconds <= 0 {
		return 0
	}
	return budget.ByOwner[owner] / budget.Seconds
}

// groupShareOf is the most-consumed owner's share inside a group.
func groupShareOf(group *candidate, budget budgetSnapshot) float64 {
	share := 0.0
	for _, job := range group.Jobs {
		if candidateShare := ownerShareOf(job.Owner, budget); candidateShare > share {
			share = candidateShare
		}
	}
	return share
}

// shareBlocksGroup reports whether an owner at or above its fair share should
// yield this group to a competing group that is still below its share. When every
// remaining group is at its share, the gate stands down rather than deadlocking
// the queue.
func shareBlocksGroup(group *candidate, remaining []*candidate, policy Policy, budget budgetSnapshot) bool {
	if policy.MaxSharePerOwner <= 0 || budget.Seconds <= 0 {
		return false
	}
	if groupShareOf(group, budget) < policy.MaxSharePerOwner {
		return false
	}
	for _, other := range remaining {
		if other == group {
			continue
		}
		if groupShareOf(other, budget) < policy.MaxSharePerOwner {
			return true
		}
	}
	return false
}

// applyShareGate removes jobs whose owner already consumed its fair share of the
// rolling window, unless every owner is at its share (which would deadlock the
// queue). A zero MaxSharePerOwner disables the gate.
func applyShareGate(jobs []Job, policy Policy, budget budgetSnapshot) []Job {
	if policy.MaxSharePerOwner <= 0 || budget.Seconds <= 0 || len(jobs) == 0 {
		return jobs
	}
	type ownerState struct {
		jobs  int
		over  bool
		share float64
	}
	owners := map[string]*ownerState{}
	for _, job := range jobs {
		state, ok := owners[job.Owner]
		if !ok {
			state = &ownerState{share: budget.ByOwner[job.Owner] / budget.Seconds}
			state.over = state.share >= policy.MaxSharePerOwner
			owners[job.Owner] = state
		}
		state.jobs++
	}
	over := 0
	for _, state := range owners {
		if state.over {
			over++
		}
	}
	if over == 0 || over == len(owners) {
		return jobs
	}
	allowed := make([]Job, 0, len(jobs))
	for _, job := range jobs {
		if !owners[job.Owner].over {
			allowed = append(allowed, job)
		}
	}
	return allowed
}

// trimBatch takes up to limit jobs, stopping when the remaining seconds budget
// would be exceeded. remaining may be +Inf for an unlimited budget. A first job
// larger than the remaining budget yields an empty batch: honest budget
// exhaustion beats overshooting the operator's cap.
func trimBatch(jobs []Job, limit int, remaining float64) []Job {
	if limit <= 0 || len(jobs) == 0 {
		return nil
	}
	selected := make([]Job, 0, min(limit, len(jobs)))
	spent := 0.0
	for _, job := range jobs {
		if len(selected) >= limit {
			break
		}
		cost := job.Estimate.Seconds
		if cost < 0 {
			cost = 0
		}
		if math.IsInf(remaining, 1) || spent+cost <= remaining {
			selected = append(selected, job)
			spent += cost
			continue
		}
		if len(selected) == 0 {
			return nil
		}
		break
	}
	return selected
}

// aggregateEstimate combines a batch's costs for one admission call. Jobs run
// sequentially inside the batch, so memory is the maximum, not the sum; time,
// items, and tokens are sums.
func aggregateEstimate(jobs []Job) Estimate {
	var total Estimate
	for _, job := range jobs {
		total.Seconds += job.Estimate.Seconds
		total.Items += job.Estimate.Items
		total.Tokens += job.Estimate.Tokens
		if job.Estimate.RAMBytes > total.RAMBytes {
			total.RAMBytes = job.Estimate.RAMBytes
		}
		if job.Estimate.VRAMBytes > total.VRAMBytes {
			total.VRAMBytes = job.Estimate.VRAMBytes
		}
	}
	return total
}

// markHot flags groups whose model is still resident.
func markHot(groups []*candidate, hot map[string]time.Time) {
	for _, group := range groups {
		if group.Key.Model == "" {
			continue
		}
		_, group.Hot = hot[group.Key.Model]
	}
}

// readyForMode filters the waiting set to what the mode allows. Passive mode has
// no background fill: only explicit interactive work runs.
func readyForMode(jobs []Job, mode Mode) []Job {
	if mode != ModePassive {
		return jobs
	}
	filtered := make([]Job, 0, len(jobs))
	for _, job := range jobs {
		if job.Priority == PriorityInteractive {
			filtered = append(filtered, job)
		}
	}
	return filtered
}

// secondsRemaining returns the seconds budget left in the rolling hour, or +Inf
// when the policy sets no cap.
func secondsRemaining(policy Policy, budget budgetSnapshot) float64 {
	if policy.MaxSecondsPerHour <= 0 {
		return math.Inf(1)
	}
	remaining := policy.MaxSecondsPerHour - budget.Seconds
	if remaining < 0 {
		return 0
	}
	return remaining
}
