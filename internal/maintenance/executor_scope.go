package maintenance

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"slices"
	"strings"
	"time"

	platformgo "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/agentscope"
	"github.com/vrooli/vrooli/internal/agentsession"
	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
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
	RunID            string     `json:"runId"`
	HarnessSessionID string     `json:"harnessSessionId,omitempty"`
	HarnessKind      string     `json:"harnessKind,omitempty"`
	Tag              string     `json:"tag"`
	LegacyTag        string     `json:"legacyTag"`
	PID              int        `json:"pid"`
	PGID             int        `json:"pgid"`
	StartedAt        *time.Time `json:"startedAt,omitempty"`
	EndedAt          *time.Time `json:"endedAt,omitempty"`
}

type ExecutorScopeEvidence struct {
	RunID   string   `json:"runId"`
	State   string   `json:"state"`
	PIDs    []int    `json:"pids"`
	Reasons []string `json:"reasons"`
}

type ExecutorScopeReport struct {
	IdentityVersion string                  `json:"identityVersion"`
	SchemaVersion   string                  `json:"schemaVersion"`
	Complete        bool                    `json:"complete"`
	Executors       []ExecutorScopeEvidence `json:"executors"`
}

// Only these labels survive reading a process environment. In particular no
// identity token, credential, command line or arbitrary environment is retained.
type scopeIdentity struct {
	RunID, Scenario, Variant, InstanceID string
	Tags                                 []string
	TagLabels                            map[string]string
}

var (
	scopeProcessTable        = listProcessTableContext
	scopeReadIdentity        = readScopeIdentity
	scopeProcessBirths       = (supervision.NativeProcessTableSource{}).ProcessesContext
	scopeEditorLeaseBindings = listScopeEditorLeaseBindings
	scopeEditorScopeMembers  = func(unit string) ([]int, error) {
		return agentsession.NewHostFacts().ScopeMembers(unit)
	}
	scopeSessionCgroupPath    = agentsession.ScopeCgroupPath
	scopeEditorScopeOccupancy = platformgo.ScopeOccupancy
	scopeEditorScopeProcesses = platformgo.ScopeProcessesContext
	scopeEditorLeaseBySession = getScopeEditorLeaseBySession
	scopeEditorScopeGone      = func(ctx context.Context, unit string) (bool, error) {
		cgroupPath, err := scopeSessionCgroupPath(unit)
		if err != nil {
			return false, err
		}
		ref := agentscope.Ref(cgroupPath)
		ref.Name = strings.TrimSuffix(unit, agentscope.ScopeSuffix)
		_, occupancyErr := scopeEditorScopeOccupancy(ref)
		if occupancyErr == nil {
			return false, nil
		}
		if !errors.Is(occupancyErr, os.ErrNotExist) {
			return false, occupancyErr
		}
		// A missing cgroup is only a positive gone result when the platform
		// owner also proves the exact unit is not-found/inactive. This second
		// bounded probe rejects empty-but-active and inconsistent scope states.
		members, err := scopeEditorScopeProcesses(ctx, ref)
		if err != nil {
			return false, err
		}
		if len(members) != 0 {
			return false, nil
		}
		// The unit probe may observe an empty readable cgroup after the first
		// ENOENT. Recheck the exact cgroup so a reappeared scope is not absence.
		_, finalOccupancyErr := scopeEditorScopeOccupancy(ref)
		if finalOccupancyErr == nil {
			return false, nil
		}
		if !errors.Is(finalOccupancyErr, os.ErrNotExist) {
			return false, finalOccupancyErr
		}
		return true, nil
	}
)
var scopeCurrentBootID = hostsession.CurrentBootID

const (
	// Retained terminal identities are inspected with the same single host
	// observation as live runs. Keep wire memory and positive PID samples
	// bounded independently; overflow remains an explicit incomplete inventory.
	ExecutorScopeMaxReferences = 65536
	ExecutorScopeMaxBytes      = 32 << 20
	ExecutorScopeTimeout       = 4 * time.Second

	// Older v1 readers silently ignore unknown request fields. The consumer
	// requires this version before trusting immutable attached-key evidence.
	ExecutorScopeIdentityVersion = "attached-key-v1"
	executorScopeSampleBudget    = 65536
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
	report := ExecutorScopeReport{SchemaVersion: "executor-scope-v1", IdentityVersion: ExecutorScopeIdentityVersion, Complete: true, Executors: []ExecutorScopeEvidence{}}
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
		if ref.HarnessSessionID != "" || ref.HarnessKind != "" {
			if strings.TrimSpace(ref.HarnessSessionID) == "" || strings.TrimSpace(ref.HarnessKind) == "" ||
				strings.TrimSpace(ref.HarnessSessionID) != ref.HarnessSessionID || strings.TrimSpace(ref.HarnessKind) != ref.HarnessKind ||
				len(ref.HarnessSessionID) > 512 || len(ref.HarnessKind) > 100 || ref.PID <= 1 {
				return report, fmt.Errorf("executor-scope attached identity requires an exact harnessSessionId, harnessKind, and persisted pid")
			}
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
	attached, attachedUncertain, attachedReasons, attachedErr := c.attachedExecutorLeases(ctx, req.Executors, entries, identities, beforeBirths, afterBirths, beforeErr, afterErr)
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
		} else if pid := attached[ref.RunID]; pid > 0 {
			evidence.State = "present"
			evidence.PIDs = append(evidence.PIDs, pid)
		} else if attachedUncertain[ref.RunID] {
			evidence.State = "unknown"
			evidence.Reasons = append(evidence.Reasons, "attached-session binding lacks current process/scope proof")
		} else if attachedErr != nil {
			evidence.State = "unknown"
			evidence.Reasons = append(evidence.Reasons, attachedErr.Error())
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
		if reason := attachedReasons[ref.RunID]; reason != "" {
			evidence.Reasons = append(evidence.Reasons, reason)
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

// attachedExecutorLeases joins an Agent Manager run to the exact editor lease
// recorded by the coding-agent launcher. A lease is only positive evidence
// when its process incarnation is stable and currently held by the leased
// native scope. Owner metadata, scope names, and PIDs alone never establish
// this association.
func (c *Controller) attachedExecutorLeases(ctx context.Context, refs []ExecutorScopeRef, entries map[int]processTableEntry, identities map[int]scopeIdentity, before, after map[int]supervision.ProcessInfo, beforeErr, afterErr error) (map[string]int, map[string]bool, map[string]string, error) {
	matched := map[string]int{}
	uncertain := map[string]bool{}
	reasons := map[string]string{}
	if beforeErr != nil || afterErr != nil {
		return matched, uncertain, reasons, fmt.Errorf("attached executor birth evidence unavailable")
	}
	bootID := scopeCurrentBootID()
	if bootID == "" {
		return matched, uncertain, reasons, fmt.Errorf("attached executor host boot identity unavailable")
	}
	keys := make([]string, 0, 2*len(refs))
	for _, ref := range refs {
		keys = append(keys, ref.RunID)
		if ref.HarnessSessionID != "" {
			keys = append(keys, ref.HarnessSessionID)
		}
	}
	leases, err := scopeEditorLeaseBindings(ctx, c.Home, keys)
	if err != nil {
		return matched, uncertain, reasons, fmt.Errorf("attached executor lease registry unavailable")
	}
	// Preserve one bounded registry snapshot for the whole inventory, but join
	// each attached run by its immutable launcher key. Only a missing canonical
	// key permits the older run-ID key; conflicting canonical rows never do.
	bySession := scopeLeaseSnapshot{}
	for _, lease := range leases {
		bySession[lease.SessionID] = append(bySession[lease.SessionID], lease)
	}
	for _, ref := range refs {
		if err := ctx.Err(); err != nil {
			return matched, uncertain, reasons, fmt.Errorf("attached executor observation deadline or cancellation")
		}
		lease, lookupErr := scenarioruntime.ResolveAttachedEditorLease(ctx, bySession, ref.RunID, ref.HarnessSessionID)
		anchored := ref.HarnessSessionID != ""
		if lookupErr != nil {
			if !anchored && errors.Is(lookupErr, scenarioruntime.ErrNotFound) {
				continue // No legacy lease; retain the independent process proof.
			}
			uncertain[ref.RunID] = true
			reasons[ref.RunID] = "attached-session exact lease identity could not be resolved"
			continue
		}
		if !anchored && lease.Status != scenarioruntime.EditorLeaseActive {
			continue
		}
		uncertain[ref.RunID] = true
		if anchored && (lease.PID != ref.PID || lease.Harness != ref.HarnessKind) {
			reasons[ref.RunID] = "attached-session lease conflicts with persisted PID or harness identity"
			continue
		}
		if lease.HostBootID != "" && lease.HostBootID != bootID {
			// The old boot cannot still hold a process. Re-resolve the exact
			// binding before excluding it, retaining all independent process
			// evidence and refusing a concurrent canonical-key registration.
			if c.scopeLeaseBindingCurrent(ctx, ref, lease) {
				delete(uncertain, ref.RunID)
			} else {
				reasons[ref.RunID] = "attached-session lease changed during prior-boot proof"
			}
			continue
		}
		if lease.HostBootID == "" || lease.PID <= 0 || lease.Scope == "" {
			reasons[ref.RunID] = "attached-session lease is missing required boot, PID, or scope metadata"
			continue
		}
		entry, exists := entries[lease.PID]
		identity := identities[lease.PID]
		birth, stable := stableScopeBirth(lease.PID, before, after)
		if !exists || strings.HasPrefix(entry.State, "Z") {
			if anchored {
				// Launchers historically wrote the planned scope before trying
				// best-effort containment. A missing scope cannot establish that
				// this executor or its descendants ever occupied it. Do not widen
				// the legacy exclusion policy to these newly resolved identities.
				reasons[ref.RunID] = "attached-session leader is gone but established containment provenance is unavailable"
				continue
			}
			session := agentsession.Session{Scope: lease.Scope}
			unit := session.ScopeUnit()
			if unit == "" || !agentscope.IsSessionScope(unit) {
				reasons[ref.RunID] = "attached-session lease scope is not a supported session scope"
				continue
			}
			scopeGone, goneErr := scopeEditorScopeGone(ctx, unit)
			if goneErr != nil {
				reasons[ref.RunID] = "attached-session native scope gone state could not be verified"
				return matched, uncertain, reasons, fmt.Errorf("attached executor native scope gone-state unavailable")
			}
			if scopeGone {
				// Reattach refreshes the exact SessionID lease in place. Re-read
				// that row after the native proof so an old PID/scope snapshot
				// cannot classify a newer attached executor as absent.
				if !c.scopeLeaseBindingCurrent(ctx, ref, lease) {
					reasons[ref.RunID] = "attached-session lease changed during native scope proof"
					continue
				}
				delete(uncertain, ref.RunID)
				// The existing consumer treats any reason on an absent verdict as
				// contradictory evidence. Keep this diagnostic only for unknown
				// outcomes; a verified absence carries no reason.
			} else {
				reasons[ref.RunID] = "leased PID is absent but its native scope is not positively gone/inactive"
			}
			continue
		}
		if !stable {
			reasons[ref.RunID] = "leased PID birth is unavailable or changed during sampling"
			continue
		}
		if identity.RunID != "" && identity.RunID != ref.RunID {
			reasons[ref.RunID] = "leased process carries a conflicting run identity"
			continue
		}
		if (ref.StartedAt != nil && birth.Before(ref.StartedAt.Add(-2*time.Second))) ||
			(ref.EndedAt != nil && birth.After(ref.EndedAt.Add(2*time.Second))) {
			reasons[ref.RunID] = "leased PID birth is outside the recorded Agent Manager run interval"
			continue
		}
		session := agentsession.Session{Scope: lease.Scope}
		unit := session.ScopeUnit()
		if unit == "" || !agentscope.IsSessionScope(unit) {
			reasons[ref.RunID] = "attached-session lease scope is not a supported session scope"
			continue
		}
		members, err := scopeEditorScopeMembers(unit)
		if err != nil {
			reasons[ref.RunID] = "attached-session native scope membership could not be read"
			return matched, uncertain, reasons, fmt.Errorf("attached executor native scope membership unavailable")
		}
		if slices.Contains(members, lease.PID) {
			// Native membership and a list snapshot are not enough if a launcher
			// rebinds the session during the observation. Revalidate the exact
			// selected key, including its owner, after the native proof.
			if !c.scopeLeaseBindingCurrent(ctx, ref, lease) {
				reasons[ref.RunID] = "attached-session lease changed or became unavailable during native scope proof"
				continue
			}
			matched[ref.RunID] = lease.PID
			delete(uncertain, ref.RunID)
		} else {
			reasons[ref.RunID] = "leased PID is not a member of the recorded native session scope"
		}
	}
	if len(matched) > 0 {
		// A PID may be reused after the batched process read but during native
		// scope inspection. One final batched birth read closes that interval
		// without an unbounded host scan per attached run.
		finalBirths, finalErr := scopeProcessBirths(ctx)
		for runID, pid := range matched {
			if _, stable := stableScopeBirth(pid, after, finalBirths); finalErr != nil || !stable {
				delete(matched, runID)
				uncertain[runID] = true
				reasons[runID] = "leased PID birth changed or became unavailable during native scope proof"
			}
		}
	}
	return matched, uncertain, reasons, nil
}

type scopeLeaseReader func(context.Context, string) (scenarioruntime.EditorLease, error)

func (r scopeLeaseReader) GetEditorLease(ctx context.Context, key string) (scenarioruntime.EditorLease, error) {
	return r(ctx, key)
}

func (c *Controller) scopeLeaseBindingCurrent(ctx context.Context, ref ExecutorScopeRef, selected scenarioruntime.EditorLease) bool {
	reader := scopeLeaseReader(func(ctx context.Context, key string) (scenarioruntime.EditorLease, error) {
		return scopeEditorLeaseBySession(ctx, c.Home, key)
	})
	// Re-resolve, rather than rereading only the previously selected legacy
	// row: a newly created canonical row must not be bypassed mid-proof.
	current, err := scenarioruntime.ResolveAttachedEditorLease(ctx, reader, ref.RunID, ref.HarnessSessionID)
	return err == nil && scenarioruntime.EditorLeaseBindingEqual(current, selected)
}

// scopeLeaseSnapshot is an exact-key adapter for the shared canonical/legacy
// selector. Duplicate rows are conflicting evidence, never a last-writer win.
type scopeLeaseSnapshot map[string][]scenarioruntime.EditorLease

func (s scopeLeaseSnapshot) GetEditorLease(ctx context.Context, sessionID string) (scenarioruntime.EditorLease, error) {
	if err := ctx.Err(); err != nil {
		return scenarioruntime.EditorLease{}, err
	}
	rows := s[sessionID]
	if len(rows) == 0 {
		return scenarioruntime.EditorLease{}, scenarioruntime.ErrNotFound
	}
	if len(rows) != 1 {
		return scenarioruntime.EditorLease{}, fmt.Errorf("duplicate attached-session lease identity")
	}
	return rows[0], nil
}

func getScopeEditorLeaseBySession(ctx context.Context, home, sessionID string) (scenarioruntime.EditorLease, error) {
	path, err := scenarioruntime.DefaultDBPath(home)
	if err != nil {
		return scenarioruntime.EditorLease{}, err
	}
	if _, err := os.Stat(path); err != nil {
		return scenarioruntime.EditorLease{}, err
	}
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home, DBPath: path, ReadOnly: true})
	if err != nil {
		return scenarioruntime.EditorLease{}, err
	}
	defer store.Close()
	return store.GetEditorLease(ctx, sessionID)
}

func listScopeEditorLeaseBindings(ctx context.Context, home string, keys []string) ([]scenarioruntime.EditorLease, error) {
	path, err := scenarioruntime.DefaultDBPath(home)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home, DBPath: path, ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer store.Close()
	return store.GetEditorLeaseBindings(ctx, keys)
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
	// Host-wide process churn is normal on a development machine. The requested
	// run identity, PID/group identity and birth-time checks below are the
	// exclusion proof; requiring every unrelated host process to remain still
	// makes maintenance permanently unavailable under ordinary scheduler/UI
	// activity. Return the latest bounded snapshot and let matching evidence
	// remain positive or unknown as appropriate.
	return observed, identities, nil
}
