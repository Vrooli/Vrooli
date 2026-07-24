package baseline

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// CollectionSchemaVersion is independent from per-scenario baseline manifests:
// a collection only references those immutable manifests and never copies their
// Test Genie evidence.
const CollectionSchemaVersion = 1

type CollectionMemberStatus string

const (
	CollectionMemberPending CollectionMemberStatus = "pending"
	CollectionMemberReady   CollectionMemberStatus = "ready"
	CollectionMemberFailed  CollectionMemberStatus = "failed"
	CollectionMemberSkipped CollectionMemberStatus = "skipped"
	CollectionMemberStale   CollectionMemberStatus = "stale"
)

// CollectionMember records one existing per-scenario baseline selected by a
// caller. Required is a coverage gate: skipped/failed/stale required members
// can never collapse to a clean collection result.
type CollectionMember struct {
	Scenario     string                 `json:"scenario"`
	BaselineName string                 `json:"baseline_name"`
	Required     bool                   `json:"required"`
	Status       CollectionMemberStatus `json:"status"`
	RunID        string                 `json:"run_id,omitempty"`
	GitSHA       string                 `json:"git_sha,omitempty"`
	Error        string                 `json:"error,omitempty"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// CollectionManifest aggregates immutable baseline references under one durable
// identity. It owns no Test Genie run and is intentionally repository/branch
// scoped, matching the existing baseline store.
type CollectionManifest struct {
	Name          string             `json:"name"`
	Branch        string             `json:"branch"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
	SchemaVersion int                `json:"schema_version"`
	Members       []CollectionMember `json:"members"`
	// PathSnapshots are source-evidence references only. They do not enter the
	// behavioral collection coverage or verdict algebra.
	PathSnapshots  []string `json:"path_snapshots,omitempty"`
	Generation     int      `json:"generation,omitempty"`
	Reanchored     bool     `json:"reanchored,omitempty"`
	ReanchorDetail string   `json:"reanchor_detail,omitempty"`
}

func (m CollectionManifest) Validate() error {
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("collection name is required")
	}
	if strings.TrimSpace(m.Branch) == "" {
		return fmt.Errorf("collection branch is required")
	}
	if m.SchemaVersion != CollectionSchemaVersion {
		return fmt.Errorf("unsupported collection schema version %d", m.SchemaVersion)
	}
	if m.Generation < 0 {
		return fmt.Errorf("collection generation cannot be negative")
	}
	if len(m.Members) == 0 {
		return fmt.Errorf("collection requires at least one member")
	}
	seen := map[string]struct{}{}
	for _, member := range m.Members {
		scenario := strings.TrimSpace(member.Scenario)
		if scenario == "" {
			return fmt.Errorf("collection member scenario is required")
		}
		if _, duplicate := seen[scenario]; duplicate {
			return fmt.Errorf("collection has duplicate scenario %q", scenario)
		}
		seen[scenario] = struct{}{}
		if strings.TrimSpace(member.BaselineName) == "" {
			return fmt.Errorf("collection member %q baseline name is required", scenario)
		}
		switch member.Status {
		case CollectionMemberPending, CollectionMemberReady, CollectionMemberFailed, CollectionMemberSkipped, CollectionMemberStale:
		default:
			return fmt.Errorf("collection member %q has invalid status %q", scenario, member.Status)
		}
		if member.Status == CollectionMemberReady && strings.TrimSpace(member.RunID) == "" {
			return fmt.Errorf("ready collection member %q requires baseline run id", scenario)
		}
	}
	pathSeen := map[string]struct{}{}
	for _, snapshot := range m.PathSnapshots {
		snapshot = strings.TrimSpace(snapshot)
		if snapshot == "" {
			return fmt.Errorf("collection path snapshot name is required")
		}
		if _, duplicate := pathSeen[snapshot]; duplicate {
			return fmt.Errorf("collection has duplicate path snapshot %q", snapshot)
		}
		pathSeen[snapshot] = struct{}{}
	}
	return nil
}

// Normalized returns a deterministic collection representation for stable
// storage/diffs. It does not invent member state or silently widen selection.
func (m CollectionManifest) Normalized() CollectionManifest {
	m.Name = strings.TrimSpace(m.Name)
	m.ReanchorDetail = strings.TrimSpace(m.ReanchorDetail)
	m.Branch = strings.TrimSpace(m.Branch)
	for i := range m.Members {
		m.Members[i].Scenario = strings.TrimSpace(m.Members[i].Scenario)
		m.Members[i].BaselineName = strings.TrimSpace(m.Members[i].BaselineName)
		m.Members[i].GitSHA = strings.TrimSpace(m.Members[i].GitSHA)
		m.Members[i].Error = strings.TrimSpace(m.Members[i].Error)
	}
	for i := range m.PathSnapshots {
		m.PathSnapshots[i] = strings.TrimSpace(m.PathSnapshots[i])
	}
	sort.SliceStable(m.Members, func(i, j int) bool { return m.Members[i].Scenario < m.Members[j].Scenario })
	sort.Strings(m.PathSnapshots)
	return m
}

// CollectionCoverage is the honest aggregate used by callers to distinguish a
// complete ready set from partial/degraded capture before attempting any diff.
type CollectionCoverage struct {
	Required int
	Ready    int
	Pending  int
	Failed   int
	Skipped  int
	Stale    int
}

func (m CollectionManifest) Coverage() CollectionCoverage {
	var out CollectionCoverage
	for _, member := range m.Members {
		if !member.Required {
			continue
		}
		out.Required++
		switch member.Status {
		case CollectionMemberReady:
			out.Ready++
		case CollectionMemberPending:
			out.Pending++
		case CollectionMemberFailed:
			out.Failed++
		case CollectionMemberSkipped:
			out.Skipped++
		case CollectionMemberStale:
			out.Stale++
		}
	}
	return out
}

func (c CollectionCoverage) Complete() bool {
	return c.Required > 0 && c.Ready == c.Required
}

// CollectionDiffVerdict is deliberately narrower than a member Test Genie
// verdict. It answers whether an aggregate is usable as a behavioral gate:
// regressions win; required work still running is not-ready; missing or
// incomparable coverage is not-comparable; advisory changes/preexisting
// failures do not convert a clean aggregate into a regression.
type CollectionDiffVerdict string

const (
	CollectionDiffClean         CollectionDiffVerdict = "clean"
	CollectionDiffRegression    CollectionDiffVerdict = "regression"
	CollectionDiffNotReady      CollectionDiffVerdict = "not-ready"
	CollectionDiffNotComparable CollectionDiffVerdict = "not-comparable"
)

type CollectionDiffMember struct {
	Scenario     string
	BaselineName string
	Required     bool
	Status       string // ready | pending | failed | skipped | stale
	RunID        string
	Verdict      Verdict
	Detail       string
	Lifecycle    CollectionDiffChildLifecycle `json:"lifecycle,omitempty"`
	// DispatchAttempts and DispatchLeaseExpiresAt make the pre-run handoff
	// durable. A pending member without RunID is dispatching, never queued;
	// reconciliation may retry it after the lease, then terminalize a bounded
	// number of failed attempts with evidence.
	DispatchAttempts       int
	DispatchLeaseExpiresAt time.Time
}

// CollectionDiffChildLifecycle is the authoritative parent-side projection of
// a Test Genie child. Status remains the backwards-compatible verdict surface;
// lifecycle answers ownership and recovery without asking callers to infer it
// from a run id or rendered standing.
type CollectionDiffChildLifecycle string

const (
	CollectionDiffChildDispatching   CollectionDiffChildLifecycle = "dispatching"
	CollectionDiffChildAwaiting      CollectionDiffChildLifecycle = "awaiting_child"
	CollectionDiffChildReconciling   CollectionDiffChildLifecycle = "reconciling"
	CollectionDiffChildPassed        CollectionDiffChildLifecycle = "passed"
	CollectionDiffChildFailed        CollectionDiffChildLifecycle = "failed"
	CollectionDiffChildNotComparable CollectionDiffChildLifecycle = "not_comparable"
)

// CollectionDiffOperation is the durable parent for selected child diff
// handles. It permits server restart recovery and one-shot aggregate waits
// without making callers reconstruct a fan-out from CLI output.
type CollectionDiffOperation struct {
	ID         string `json:"id"`
	Collection string `json:"collection"`
	Branch     string `json:"branch"`
	RepoDir    string `json:"repo_dir,omitempty"`
	// CollectionSnapshot is the immutable collection/baseline membership that
	// this operation was created against. A status read must be able to explain
	// and finalize an already-created operation even after the mutable
	// collection has been deleted or replaced.
	CollectionSnapshot CollectionManifest     `json:"collection_snapshot"`
	Members            []CollectionDiffMember `json:"members"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
	// Lifecycle is owned by the aggregate, not inferred by a client from child
	// text. It is persisted so a restart cannot make finalization look pending.
	Lifecycle      string    `json:"lifecycle"`
	LastProgressAt time.Time `json:"last_progress_at"`
	// LastReconciliationError records the latest non-terminal failure while
	// projecting Test Genie truth. It is diagnostic only: a later successful
	// reconciliation clears it, and it must never make a durable child appear
	// complete.
	LastReconciliationError string `json:"last_reconciliation_error,omitempty"`
}

func (o CollectionDiffOperation) Aggregate(collection CollectionManifest) CollectionDiffResult {
	return AggregateCollectionDiff(collection, o.Members)
}

type CollectionDiffResult struct {
	Collection string
	Branch     string
	Coverage   CollectionCoverage
	Verdict    CollectionDiffVerdict
	Members    []CollectionDiffMember
}

// AggregateCollectionDiff projects member evidence without losing provenance.
// It never claims clean when a required member was unavailable, pending, or
// incomparable; callers can show the member diagnostics and recovery handle.
func AggregateCollectionDiff(collection CollectionManifest, members []CollectionDiffMember) CollectionDiffResult {
	result := CollectionDiffResult{Collection: collection.Name, Branch: collection.Branch, Coverage: collection.Coverage(), Members: append([]CollectionDiffMember(nil), members...), Verdict: CollectionDiffClean}
	if !result.Coverage.Complete() {
		result.Verdict = CollectionDiffNotComparable
		for _, member := range collection.Members {
			if member.Required && member.Status == CollectionMemberPending {
				result.Verdict = CollectionDiffNotReady
				break
			}
		}
		return result
	}
	for _, member := range result.Members {
		if !member.Required {
			continue
		}
		switch member.Status {
		case "pending":
			if result.Verdict != CollectionDiffRegression {
				result.Verdict = CollectionDiffNotReady
			}
		case "failed", "skipped", "stale", "not-comparable":
			if result.Verdict != CollectionDiffRegression && result.Verdict != CollectionDiffNotReady {
				result.Verdict = CollectionDiffNotComparable
			}
		case "ready":
			decision := GateDecisionForLegacyVerdict(member.Verdict)
			if decision.LegacyVerdict == VerdictRegression {
				result.Verdict = CollectionDiffRegression
			} else if decision.Blocking && result.Verdict != CollectionDiffRegression && result.Verdict != CollectionDiffNotReady {
				result.Verdict = CollectionDiffNotComparable
			}
		}
	}
	return result
}
