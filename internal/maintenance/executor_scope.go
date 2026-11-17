package maintenance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/cli-core/cliutil"
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
	RunID            string                            `json:"runId"`
	HarnessSessionID string                            `json:"harnessSessionId,omitempty"`
	HarnessKind      string                            `json:"harnessKind,omitempty"`
	BindingProof     *cliutil.AgentSessionBindingProof `json:"bindingProof,omitempty"`
	// HomePaths opts into exact process-environment matching for harness homes.
	// In-scope unreadable identities make a negative home verdict unknown.
	HomePaths []string   `json:"homePaths,omitempty"`
	Tag       string     `json:"tag"`
	LegacyTag string     `json:"legacyTag"`
	PID       int        `json:"pid"`
	PGID      int        `json:"pgid"`
	StartedAt *time.Time `json:"startedAt,omitempty"`
	EndedAt   *time.Time `json:"endedAt,omitempty"`
}

func validExecutorScopeBindingProof(ref ExecutorScopeRef) bool {
	proof := ref.BindingProof
	if proof == nil || ref.PID != 0 || ref.PGID != 0 ||
		proof.ProofVersion != cliutil.AgentSessionBindingProofVersion ||
		proof.RunID != ref.RunID || proof.HarnessSessionID != ref.HarnessSessionID ||
		proof.LeaseSessionID == "" || proof.WebConsoleSessionID == "" ||
		proof.HarnessKind != ref.HarnessKind || proof.OwnerKind != agentsession.KindWebConsoleSession {
		return false
	}
	binding := proof.Binding
	if binding.PID <= 1 || binding.UID < 0 || len(binding.Cgroup) > 1024 ||
		!strings.HasPrefix(binding.Cgroup, "0::/user.slice/user-") ||
		!strings.Contains(binding.Cgroup, "/vrooli.slice/vrooli-agents.slice/") ||
		strings.ContainsAny(binding.Cgroup, "\r\n\t ") {
		return false
	}
	start, err := strconv.ParseUint(binding.Start, 10, 64)
	if err != nil || start == 0 {
		return false
	}
	boot, err := uuid.Parse(binding.Boot)
	return err == nil && boot != uuid.Nil
}

func executorScopeReferencePID(ref ExecutorScopeRef) int {
	if ref.PID > 1 {
		return ref.PID
	}
	if proof := ref.BindingProof; proof != nil {
		return proof.Binding.PID
	}
	return ref.PID
}

func executorScopeLeaseSessionID(ref ExecutorScopeRef) string {
	if proof := ref.BindingProof; proof != nil && proof.LeaseSessionID != "" {
		return proof.LeaseSessionID
	}
	return ref.HarnessSessionID
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
	HomePaths                            []string
	TagLabels                            map[string]string
}

var (
	scopeProcessTable        = listProcessTableContext
	scopeReadIdentity        = readScopeIdentity
	scopeReadProcessBinding  = cliutil.ObserveProcessBinding
	scopeReadProcessFact     = func(pid int) (agentsession.ProcessFact, error) { return agentsession.NewHostFacts().Process(pid) }
	scopeProcessBirths       = (supervision.NativeProcessTableSource{}).ProcessesContext
	scopeEditorLeaseBindings = listScopeEditorLeaseBindings
	scopeEditorScopeMembers  = func(unit string) ([]int, error) {
		return agentsession.NewHostFacts().ScopeMembers(unit)
	}
	scopeSessionCgroupPath                                                                                               = agentsession.ScopeCgroupPath
	scopeEditorScopeOccupancy                                                                                            = platformgo.ScopeOccupancy
	scopeEditorScopeProcesses                                                                                            = platformgo.ScopeProcessesContext
	scopeEditorScopeProcessesBatch                                                                                       = platformgo.ScopeProcessesBatchContext
	scopeEditorLeaseProvenance      func(context.Context, string, string) (scenarioruntime.EditorLeaseProvenance, error) = getScopeEditorLeaseProvenance
	scopeEditorLeaseProvenanceBatch                                                                                      = getScopeEditorLeaseProvenances
	scopeEditorScopeGone                                                                                                 = func(ctx context.Context, unit string) (bool, error) {
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

// Kept separate from the exported wire-contract timeout so focused tests can
// exercise budget exhaustion without waiting for a full observation window.
var executorScopeObservationTimeout = ExecutorScopeTimeout

const (
	// Retained terminal identities are inspected with the same single host
	// observation as live runs. Requests are bounded by bytes; reference work is
	// paged so a large history does not require a larger fixed count ceiling.
	ExecutorScopeBatchSize = 8192
	ExecutorScopeMaxBytes  = 32 << 20
	// A complete scope proof can include tens of thousands of retained runs and
	// exact attached-session checks. Keep it bounded, but allow normal large
	// inventories enough time to finish and return per-run evidence.
	ExecutorScopeTimeout             = 20 * time.Second
	ExecutorScopeMaxProcesses        = 262144
	ExecutorScopeIdentityWorkers     = 16
	executorScopeProcessTablePasses  = 3
	ExecutorScopeLeaseCheckBatchSize = 128

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
	homes       map[string][]int
	serviceTags map[string][]int
	groups      map[int]int
}

func normalizeExecutorHomePath(value string) (string, error) {
	if value == "" || len(value) > 4096 || strings.ContainsAny(value, "\x00\r\n") || !path.IsAbs(value) {
		return "", fmt.Errorf("executor home path must be a bounded absolute path")
	}
	return path.Clean(value), nil
}

func normalizeExecutorHomePaths(paths []string) ([]string, error) {
	if len(paths) > 8 {
		return nil, fmt.Errorf("executor home path limit exceeded")
	}
	seen := make(map[string]struct{}, len(paths))
	normalized := make([]string, 0, len(paths))
	for _, candidate := range paths {
		value, err := normalizeExecutorHomePath(candidate)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	slices.Sort(normalized)
	return normalized, nil
}

// executorScopeHomePathsFromEnvironment retains only the two harness-home
// values relevant to exact native-home inventory. Arbitrary environment
// entries are consumed by the caller but never stored in scopeIdentity.
func executorScopeHomePathsFromEnvironment(env map[string]string) []string {
	paths := make([]string, 0, 2)
	for _, key := range []string{"CODEX_HOME", "GROK_HOME"} {
		value := env[key]
		if value == "" {
			continue
		}
		clean, err := normalizeExecutorHomePath(value)
		if err != nil || slices.Contains(paths, clean) {
			continue
		}
		paths = append(paths, clean)
	}
	return paths
}

func indexExecutorScopes(ctx context.Context, refs []ExecutorScopeRef, entries map[int]processTableEntry, identities map[int]scopeIdentity, inherited map[int]map[string]bool, sampleLimit int) (executorScopeIndex, error) {
	index := executorScopeIndex{runs: make(map[string][]int, len(refs)), tags: make(map[string][]int, 2*len(refs)), homes: make(map[string][]int), serviceTags: make(map[string][]int, 2*len(refs)), groups: make(map[int]int)}
	for i := range refs {
		normalized, err := normalizeExecutorHomePaths(refs[i].HomePaths)
		if err != nil {
			return index, err
		}
		refs[i].HomePaths = normalized
		ref := refs[i]
		index.runs[ref.RunID] = nil
		index.tags[ref.Tag] = nil
		index.tags[ref.LegacyTag] = nil
		index.serviceTags[ref.Tag] = nil
		index.serviceTags[ref.LegacyTag] = nil
		for _, home := range ref.HomePaths {
			index.homes[home] = nil
		}
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
		for _, home := range identity.HomePaths {
			if pids, wanted := index.homes[home]; wanted && len(pids) < sampleLimit {
				index.homes[home] = append(pids, pid)
			}
		}
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
	if len(req.Executors) < 1 {
		return report, fmt.Errorf("executor-scope requires at least one executor")
	}
	for i := range req.Executors {
		homes, err := normalizeExecutorHomePaths(req.Executors[i].HomePaths)
		if err != nil {
			return report, fmt.Errorf("executor-scope invalid exact home reference: %w", err)
		}
		req.Executors[i].HomePaths = homes
	}
	requestBytes, err := json.Marshal(req)
	if err != nil {
		return report, fmt.Errorf("encode executor-scope request: %w", err)
	}
	if len(requestBytes) > ExecutorScopeMaxBytes {
		return report, fmt.Errorf("executor-scope request exceeds %d bytes", ExecutorScopeMaxBytes)
	}
	ctx, cancel := context.WithTimeout(parent, executorScopeObservationTimeout)
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
				len(ref.HarnessSessionID) > 512 || len(ref.HarnessKind) > 100 || (ref.PID <= 1 && !validExecutorScopeBindingProof(ref)) {
				return report, fmt.Errorf("executor-scope attached identity requires an exact harnessSessionId, harnessKind, and persisted or owner-proved pid")
			}
		}
		if ref.BindingProof != nil && !validExecutorScopeBindingProof(ref) {
			return report, fmt.Errorf("executor-scope attached owner proof does not match its exact run and session identity")
		}
		seen[ref.RunID] = true
		if ref.StartedAt != nil && ref.EndedAt != nil && ref.EndedAt.Before(*ref.StartedAt) {
			return report, fmt.Errorf("executor-scope execution interval is reversed")
		}
		report.Executors = append(report.Executors, ExecutorScopeEvidence{RunID: ref.RunID, State: "absent", PIDs: []int{}, Reasons: []string{}})
	}
	// Read exact lease keys before walking the host process table. On a busy
	// host, process identity inspection can consume the whole observation
	// budget; leaving the lease read until afterward turns that unrelated cost
	// into a misleading "registry unavailable" result for every run.
	leasePages := make([]scopeLeasePage, 0, (len(req.Executors)+ExecutorScopeBatchSize-1)/ExecutorScopeBatchSize)
	for pageStart := 0; pageStart < len(req.Executors); pageStart += ExecutorScopeBatchSize {
		pageEnd := min(pageStart+ExecutorScopeBatchSize, len(req.Executors))
		pageRefs := req.Executors[pageStart:pageEnd]
		leasePages = append(leasePages, loadScopeLeasePage(ctx, c.Home, pageRefs))
	}
	beforeBirths, beforeErr := scopeProcessBirths(ctx)
	entries, identities, identityErrors, err := observeScopeProcesses(ctx)
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
	for pageIndex, pageStart := 0, 0; pageStart < len(req.Executors); pageIndex, pageStart = pageIndex+1, pageStart+ExecutorScopeBatchSize {
		pageEnd := min(pageStart+ExecutorScopeBatchSize, len(req.Executors))
		pageRefs := req.Executors[pageStart:pageEnd]
		attached, attachedUncertain, attachedReasons, attachedErr := c.attachedExecutorLeases(ctx, pageRefs, leasePages[pageIndex], entries, identities, identityErrors, beforeBirths, afterBirths, beforeErr, afterErr, err)
		sampleLimit := max(1, min(64, executorScopeSampleBudget/len(pageRefs)))
		index, indexErr := indexExecutorScopes(ctx, pageRefs, entries, identities, inherited, sampleLimit)
		pageErr := err
		if indexErr != nil {
			pageErr = fmt.Errorf("scope observation deadline or cancellation")
		}
		for pageIndex, ref := range pageRefs {
			evidence := &report.Executors[pageStart+pageIndex]
			matches := [][]int{index.runs[ref.RunID], index.tags[ref.Tag], index.tags[ref.LegacyTag]}
			for _, home := range ref.HomePaths {
				matches = append(matches, index.homes[home])
			}
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
			if len(ref.HomePaths) > 0 {
				unreadable := make([]int, 0)
				for pid := range identityErrors {
					entry, ok := entries[pid]
					if ok && executorHomeIdentityInScope(entry) {
						unreadable = append(unreadable, pid)
					}
				}
				if len(unreadable) > 0 {
					slices.Sort(unreadable)
					const unreadablePIDSampleLimit = 16
					sample := unreadable[:min(len(unreadable), unreadablePIDSampleLimit)]
					report.Complete = false
					if evidence.State != "present" {
						evidence.State = "unknown"
					}
					evidence.Reasons = append(evidence.Reasons, fmt.Sprintf("exact-home inventory has %d unreadable in-scope process identity reads; sample_pids=%v; omitted_pids=%d", len(unreadable), sample, max(0, len(unreadable)-len(sample))))
				}
			}
			if reason := attachedReasons[ref.RunID]; reason != "" {
				evidence.Reasons = append(evidence.Reasons, reason)
			}
			slices.Sort(evidence.PIDs)
			if pageErr != nil {
				report.Complete = false
				if evidence.State != "present" {
					evidence.State = "unknown"
				}
				evidence.Reasons = append(evidence.Reasons, pageErr.Error())
			}
			if evidence.State == "unknown" {
				report.Complete = false
			}
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
type attachedLeaseProof struct {
	ref           ExecutorScopeRef
	lease         scenarioruntime.EditorLease
	pid           int
	changedReason string
}

type attachedLeaseResolution struct {
	lease scenarioruntime.EditorLease
	err   error
}

type scopeGoneCheck struct {
	runID string
	unit  string
}

type scopeLeasePage struct {
	bySession scopeLeaseSnapshot
	err       error
}

func loadScopeLeasePage(ctx context.Context, home string, refs []ExecutorScopeRef) scopeLeasePage {
	page := scopeLeasePage{bySession: scopeLeaseSnapshot{}}
	keys := make([]string, 0, 2*len(refs))
	for _, ref := range refs {
		keys = append(keys, ref.RunID)
		if sessionID := executorScopeLeaseSessionID(ref); sessionID != "" {
			keys = append(keys, sessionID)
		}
	}
	leases, err := scopeEditorLeaseBindings(ctx, home, keys)
	if err != nil {
		page.err = fmt.Errorf("attached executor lease registry unavailable: %w", err)
		return page
	}
	for _, lease := range leases {
		page.bySession[lease.SessionID] = append(page.bySession[lease.SessionID], lease)
	}
	return page
}

func observeAttachedScopeGone(ctx context.Context, refs []ExecutorScopeRef, resolved map[string]attachedLeaseResolution, provenance map[string]scenarioruntime.EditorLeaseProvenance, provenanceErrors map[string]error, entries map[int]processTableEntry, bootID string) map[string]scopeGoneResult {
	checks := make([]scopeGoneCheck, 0)
	for _, ref := range refs {
		resolution := resolved[ref.RunID]
		lease, anchored := resolution.lease, ref.HarnessSessionID != ""
		if resolution.err != nil || (!anchored && lease.Status != scenarioruntime.EditorLeaseActive) ||
			(anchored && (lease.PID != executorScopeReferencePID(ref) || lease.Harness != ref.HarnessKind)) ||
			lease.HostBootID == "" || lease.HostBootID != bootID || lease.PID <= 0 || lease.Scope == "" {
			continue
		}
		entry, exists := entries[lease.PID]
		if exists && !strings.HasPrefix(entry.State, "Z") {
			continue
		}
		unit := ""
		if anchored {
			proof, ok := provenance[lease.BindingID]
			if !ok || provenanceErrors[lease.BindingID] != nil || !establishedLeaseContainmentMatches(lease, proof) {
				continue
			}
			var err error
			unit, err = establishedContainmentScopeUnit(lease, proof.Scope)
			if err != nil {
				continue
			}
		} else {
			unit = (agentsession.Session{Scope: lease.Scope}).ScopeUnit()
			if unit == "" || !agentscope.IsSessionScope(unit) {
				continue
			}
		}
		checks = append(checks, scopeGoneCheck{runID: ref.RunID, unit: unit})
	}
	return readScopeGoneChecks(ctx, checks)
}

type scopeGoneResult struct {
	gone bool
	err  error
}

func readScopeGoneChecks(ctx context.Context, checks []scopeGoneCheck) map[string]scopeGoneResult {
	results := make(map[string]scopeGoneResult, len(checks))
	if len(checks) == 0 {
		return results
	}
	type exactScope struct {
		ref    platformgo.ScopeRef
		runIDs []string
	}
	byUnit := make(map[string]*exactScope, len(checks))
	for _, check := range checks {
		if err := ctx.Err(); err != nil {
			results[check.runID] = scopeGoneResult{err: err}
			continue
		}
		cgroupPath, err := scopeSessionCgroupPath(check.unit)
		if err != nil {
			results[check.runID] = scopeGoneResult{err: err}
			continue
		}
		ref := agentscope.Ref(cgroupPath)
		ref.Name = strings.TrimSuffix(check.unit, agentscope.ScopeSuffix)
		_, err = scopeEditorScopeOccupancy(ref)
		if err == nil {
			results[check.runID] = scopeGoneResult{gone: false}
			continue
		}
		if !errors.Is(err, os.ErrNotExist) {
			results[check.runID] = scopeGoneResult{err: err}
			continue
		}
		item := byUnit[check.unit]
		if item == nil {
			item = &exactScope{ref: ref}
			byUnit[check.unit] = item
		}
		item.runIDs = append(item.runIDs, check.runID)
	}
	units := make([]string, 0, len(byUnit))
	for unit := range byUnit {
		units = append(units, unit)
	}
	slices.Sort(units)
	refs := make([]platformgo.ScopeRef, 0, len(units))
	for _, unit := range units {
		refs = append(refs, byUnit[unit].ref)
	}
	observations := scopeEditorScopeProcessesBatch(ctx, refs)
	for i, unit := range units {
		item := byUnit[unit]
		observation := platformgo.ScopeProcessObservation{}
		if i < len(observations) {
			observation = observations[i]
		} else {
			observation.Err = fmt.Errorf("exact attached scope batch omitted a requested unit")
		}
		gone := observation.Err == nil && len(observation.PIDs) == 0
		if gone {
			// A readable empty cgroup is not enough. Recheck the exact path after
			// the manager state query so a reappearing scope is not absence.
			_, err := scopeEditorScopeOccupancy(item.ref)
			if err == nil {
				gone = false
			} else if !errors.Is(err, os.ErrNotExist) {
				observation.Err = err
				gone = false
			}
		}
		for _, runID := range item.runIDs {
			results[runID] = scopeGoneResult{gone: gone, err: observation.Err}
		}
	}
	return results
}

func (c *Controller) attachedExecutorLeases(ctx context.Context, refs []ExecutorScopeRef, page scopeLeasePage, entries map[int]processTableEntry, identities map[int]scopeIdentity, identityErrors map[int]error, before, after map[int]supervision.ProcessInfo, beforeErr, afterErr, processObservationErr error) (map[string]int, map[string]bool, map[string]string, error) {
	matched := map[string]int{}
	uncertain := map[string]bool{}
	reasons := map[string]string{}
	if beforeErr != nil || afterErr != nil {
		return matched, uncertain, reasons, errors.Join(fmt.Errorf("attached executor birth evidence unavailable"), page.err)
	}
	bootID := scopeCurrentBootID()
	if bootID == "" {
		return matched, uncertain, reasons, errors.Join(fmt.Errorf("attached executor host boot identity unavailable"), page.err)
	}
	if page.err != nil {
		return matched, uncertain, reasons, page.err
	}
	// Preserve one bounded registry snapshot for this reference page, but join
	// each attached run by its immutable launcher key. Only a missing canonical
	// key permits the older run-ID key; conflicting canonical rows never do.
	bySession := page.bySession
	resolved := make(map[string]attachedLeaseResolution, len(refs))
	provenanceIDs := make([]string, 0, len(refs))
	provenanceSeen := map[string]bool{}
	for _, ref := range refs {
		lease, lookupErr := scenarioruntime.ResolveAttachedEditorLease(ctx, bySession, ref.RunID, executorScopeLeaseSessionID(ref))
		resolved[ref.RunID] = attachedLeaseResolution{lease: lease, err: lookupErr}
		if lookupErr != nil || ref.HarnessSessionID == "" || lease.PID != executorScopeReferencePID(ref) || lease.Harness != ref.HarnessKind ||
			lease.BindingID == "" || lease.HostBootID != bootID || processObservationErr != nil {
			continue
		}
		if !provenanceSeen[lease.BindingID] {
			provenanceSeen[lease.BindingID] = true
			provenanceIDs = append(provenanceIDs, lease.BindingID)
		}
	}
	provenanceByID, provenanceErrors := scopeEditorLeaseProvenanceBatch(ctx, c.Home, provenanceIDs)
	scopeGoneByRun := map[string]scopeGoneResult{}
	if processObservationErr == nil {
		scopeGoneByRun = observeAttachedScopeGone(ctx, refs, resolved, provenanceByID, provenanceErrors, entries, bootID)
	}
	pendingProofs := make([]attachedLeaseProof, 0, ExecutorScopeLeaseCheckBatchSize)
	flushProofs := func() {
		if len(pendingProofs) == 0 {
			return
		}
		keys := make([]string, 0, 2*len(pendingProofs))
		for _, proof := range pendingProofs {
			keys = append(keys, proof.ref.RunID)
			if sessionID := executorScopeLeaseSessionID(proof.ref); sessionID != "" {
				keys = append(keys, sessionID)
			}
		}
		currentRows, readErr := scopeEditorLeaseBindings(ctx, c.Home, keys)
		currentBySession := scopeLeaseSnapshot{}
		if readErr == nil {
			for _, current := range currentRows {
				currentBySession[current.SessionID] = append(currentBySession[current.SessionID], current)
			}
		}
		for _, proof := range pendingProofs {
			current, currentErr := scenarioruntime.EditorLease{}, readErr
			if currentErr == nil {
				current, currentErr = scenarioruntime.ResolveAttachedEditorLease(ctx, currentBySession, proof.ref.RunID, executorScopeLeaseSessionID(proof.ref))
			}
			if currentErr != nil || !scenarioruntime.EditorLeaseBindingEqual(current, proof.lease) {
				reasons[proof.ref.RunID] = proof.changedReason
				continue
			}
			delete(uncertain, proof.ref.RunID)
			if proof.pid > 0 {
				matched[proof.ref.RunID] = proof.pid
			}
		}
		pendingProofs = pendingProofs[:0]
	}
	queueProof := func(proof attachedLeaseProof) {
		pendingProofs = append(pendingProofs, proof)
		if len(pendingProofs) >= ExecutorScopeLeaseCheckBatchSize {
			flushProofs()
		}
	}
	for _, ref := range refs {
		if err := ctx.Err(); err != nil {
			flushProofs()
			return matched, uncertain, reasons, fmt.Errorf("attached executor observation deadline or cancellation")
		}
		resolution := resolved[ref.RunID]
		lease, lookupErr := resolution.lease, resolution.err
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
		if anchored && (lease.PID != executorScopeReferencePID(ref) || lease.Harness != ref.HarnessKind) {
			reasons[ref.RunID] = "attached-session lease conflicts with persisted PID or harness identity"
			continue
		}
		if lease.HostBootID != "" && lease.HostBootID != bootID {
			// The old boot cannot still hold a process. Re-resolve the exact
			// binding before excluding it, retaining all independent process
			// evidence and refusing a concurrent canonical-key registration.
			queueProof(attachedLeaseProof{ref: ref, lease: lease, changedReason: "attached-session lease changed during prior-boot proof"})
			continue
		}
		if lease.HostBootID == "" || lease.PID <= 0 || lease.Scope == "" {
			reasons[ref.RunID] = "attached-session lease is missing required boot, PID, or scope metadata"
			continue
		}
		entry, exists := entries[lease.PID]
		identity := identities[lease.PID]
		if exists && identityErrors[lease.PID] != nil && !strings.HasPrefix(entry.State, "Z") {
			reasons[ref.RunID] = "leased PID process identity is unreadable; exact attached executor cannot be verified"
			continue
		}
		birth, stable := stableScopeBirth(lease.PID, before, after)
		if !exists || strings.HasPrefix(entry.State, "Z") {
			if !exists && processObservationErr != nil {
				reason := "attached-session leader absence cannot be proven because host process identity inventory is incomplete; retry after host process inspection succeeds"
				if lease.Owner.Kind == agentsession.KindWebConsoleSession && strings.TrimSpace(lease.Owner.Ref) != "" && len(lease.Owner.Ref) <= 128 {
					reason += fmt.Sprintf("; inspect Web Console session %s and its tmux pane", lease.Owner.Ref)
				}
				reasons[ref.RunID] = reason
				continue
			}
			if anchored {
				provenance, provenanceFound := provenanceByID[lease.BindingID]
				if !provenanceFound || provenanceErrors[lease.BindingID] != nil || !establishedLeaseContainmentMatches(lease, provenance) {
					reasons[ref.RunID] = "owner scope unknown; containment provenance missing; " + attachedScopeDiagnostic(lease, provenance, provenanceFound, provenanceErrors[lease.BindingID])
					continue
				}
				_, scopeErr := establishedContainmentScopeUnit(lease, provenance.Scope)
				if scopeErr != nil {
					reasons[ref.RunID] = "attached-session established native scope is not supported"
					continue
				}
				goneResult, checked := scopeGoneByRun[ref.RunID]
				if !checked {
					goneResult.err = fmt.Errorf("exact attached scope check was not scheduled")
				}
				scopeGone, goneErr := goneResult.gone, goneResult.err
				if goneErr != nil {
					reasons[ref.RunID] = "attached-session native scope gone state could not be verified"
					flushProofs()
					return matched, uncertain, reasons, fmt.Errorf("attached executor native scope gone-state unavailable")
				}
				if !scopeGone {
					reasons[ref.RunID] = "established attached-session native scope is not positively gone/inactive"
					continue
				}
				queueProof(attachedLeaseProof{ref: ref, lease: lease, changedReason: "attached-session lease changed during native scope proof"})
				continue
			}
			session := agentsession.Session{Scope: lease.Scope}
			unit := session.ScopeUnit()
			if unit == "" || !agentscope.IsSessionScope(unit) {
				reasons[ref.RunID] = "attached-session lease scope is not a supported session scope"
				continue
			}
			goneResult, checked := scopeGoneByRun[ref.RunID]
			if !checked {
				goneResult.err = fmt.Errorf("exact attached scope check was not scheduled")
			}
			scopeGone, goneErr := goneResult.gone, goneResult.err
			if goneErr != nil {
				reasons[ref.RunID] = "attached-session native scope gone state could not be verified"
				flushProofs()
				return matched, uncertain, reasons, fmt.Errorf("attached executor native scope gone-state unavailable")
			}
			if scopeGone {
				// Reattach refreshes the exact SessionID lease in place. Re-read
				// that row after the native proof so an old PID/scope snapshot
				// cannot classify a newer attached executor as absent. The exact
				// binding is re-read in bounded batches below.
				queueProof(attachedLeaseProof{ref: ref, lease: lease, changedReason: "attached-session lease changed during native scope proof"})
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
			flushProofs()
			return matched, uncertain, reasons, fmt.Errorf("attached executor native scope membership unavailable")
		}
		if slices.Contains(members, lease.PID) {
			// Native membership and a list snapshot are not enough if a launcher
			// rebinds the session during the observation. Revalidate the exact
			// selected key, including its owner, in a bounded batch.
			queueProof(attachedLeaseProof{ref: ref, lease: lease, pid: lease.PID, changedReason: "attached-session lease changed or became unavailable during native scope proof"})
		} else if evidencePID, ok := proveAttachedScopedChild(ctx, ref, lease, provenanceByID[lease.BindingID], provenanceErrors[lease.BindingID], members, entries, before, after); ok {
			// spawn-and-wait keeps the exact launch process outside its session
			// cgroup. A live direct child in the canonical scope is positive
			// executor evidence, with immutable provenance checked when available.
			queueProof(attachedLeaseProof{ref: ref, lease: lease, pid: evidencePID, changedReason: "attached-session lease changed during scoped-child proof"})
		} else {
			reasons[ref.RunID] = "leased PID is not a member of the recorded native session scope (outside the recorded native session scope), and no exact scoped child proof was established"
		}
	}
	flushProofs()
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

// attachedScopeDiagnostic carries bounded, non-secret owner evidence so an
// operator can resolve an unknown attached scope through its owning session.
// It is diagnostic only: these fields never turn an unknown into absent.
func attachedScopeDiagnostic(lease scenarioruntime.EditorLease, provenance scenarioruntime.EditorLeaseProvenance, found bool, lookupErr error) string {
	state := provenance.ContainmentState
	if !found {
		state = "missing"
	} else if lookupErr != nil {
		state = "unavailable"
	}
	diagnostic := struct {
		Owner       string `json:"owner,omitempty"`
		Session     string `json:"session"`
		Lease       string `json:"lease"`
		PID         int    `json:"pid"`
		Boot        string `json:"boot,omitempty"`
		Scope       string `json:"scope,omitempty"`
		Binding     string `json:"binding,omitempty"`
		Containment string `json:"containment"`
		ProofScope  string `json:"proofScope,omitempty"`
		Method      string `json:"method,omitempty"`
		EvidencePID int    `json:"evidencePid,omitempty"`
	}{
		Owner:   lease.Owner.Kind + ":" + lease.Owner.Ref,
		Session: lease.SessionID, Lease: lease.Status, PID: lease.PID,
		Boot: lease.HostBootID, Scope: lease.Scope, Binding: lease.BindingID,
		Containment: state, ProofScope: provenance.Scope,
		Method: provenance.Method, EvidencePID: provenance.EvidencePID,
	}
	data, err := json.Marshal(diagnostic)
	if err != nil {
		return "attached-scope-diagnostic=unavailable"
	}
	return "attached-scope-diagnostic=" + string(data)
}

// establishedContainmentScopeUnit converts the recorded cgroup ref to a
// session unit only when it names this lease's exact session and canonical
// current-user agent-slice path. The path is retained in provenance so a
// similarly named scope elsewhere cannot be mistaken for this attachment.
func establishedContainmentScopeUnit(lease scenarioruntime.EditorLease, recorded string) (string, error) {
	ref, err := platformgo.ParseScopeRef(recorded)
	if err != nil || ref.Kind != platformgo.ScopeKindCgroup || path.Clean(ref.Path) != ref.Path {
		return "", fmt.Errorf("invalid established cgroup provenance")
	}
	unit := path.Base(ref.Path)
	if !agentscope.IsSessionScope(unit) || (agentsession.Session{Scope: lease.Scope}).ScopeUnit() != unit {
		return "", fmt.Errorf("established cgroup provenance does not identify the attached session")
	}
	canonicalPath, err := scopeSessionCgroupPath(unit)
	if err != nil || canonicalPath != ref.Path {
		return "", fmt.Errorf("established cgroup provenance is not under the canonical agent slice")
	}
	return unit, nil
}

// establishedLeaseContainmentMatches accepts only the immutable provenance
// for this exact lease attempt. Intended scope names and PIDs alone are not
// evidence that an executor ever entered a native scope.
func establishedLeaseContainmentMatches(lease scenarioruntime.EditorLease, binding scenarioruntime.EditorLeaseProvenance) bool {
	return lease.BindingID != "" && binding.BindingID == lease.BindingID && binding.SessionID == lease.SessionID &&
		binding.PID == lease.PID && binding.HostBootID == lease.HostBootID && binding.IntendedScope == lease.Scope &&
		binding.ContainmentState == scenarioruntime.EditorContainmentEstablished && binding.Scope != "" &&
		binding.Method != "" && binding.EvidencePID > 1 && binding.EvidenceUID == binding.UID &&
		binding.EvidenceBootID == binding.HostBootID && binding.EvidenceProcessStart != "" && binding.Failure == ""
}

// proveAttachedScopedChild resolves the spawn-and-wait form of an attached
// session. The lease PID remains the wrapper, while its direct live child is
// the executor root inside the exact canonical session cgroup. A supplied
// identity-binding proof is required when the historical lease has no
// containment provenance; established provenance must match the current
// binding exactly whenever a binding ID exists. This is positive evidence
// only: it never proves absence.
func proveAttachedScopedChild(ctx context.Context, ref ExecutorScopeRef, lease scenarioruntime.EditorLease, provenance scenarioruntime.EditorLeaseProvenance, provenanceErr error, members []int, entries map[int]processTableEntry, before, after map[int]supervision.ProcessInfo) (int, bool) {
	if ctx.Err() != nil || ref.HarnessSessionID == "" || lease.PID <= 1 || lease.Harness != ref.HarnessKind || lease.SessionID != executorScopeLeaseSessionID(ref) {
		return 0, false
	}
	proof := ref.BindingProof
	if proof != nil {
		if !validExecutorScopeBindingProof(ref) || proof.Binding.PID != lease.PID || lease.Status != scenarioruntime.EditorLeaseActive {
			return 0, false
		}
		owner, ok := agentsession.OwnerFromLease(lease.Owner)
		if !ok || owner.Kind != agentsession.KindWebConsoleSession || owner.Ref != proof.WebConsoleSessionID || owner.Source != agentsession.SourceLaunch ||
			owner.Evidence.ReadFromPID != lease.PID || owner.Evidence.WebConsoleSessionID != proof.WebConsoleSessionID {
			return 0, false
		}
	}
	unit := (agentsession.Session{Scope: lease.Scope}).ScopeUnit()
	if unit == "" || !agentscope.IsSessionScope(unit) {
		return 0, false
	}
	canonicalPath, err := scopeSessionCgroupPath(unit)
	if err != nil || canonicalPath == "" || !strings.HasPrefix(canonicalPath, "/") {
		return 0, false
	}
	wantCgroup := "0::" + canonicalPath

	provenancePresent := lease.BindingID != ""
	if provenancePresent {
		if provenanceErr != nil || provenance.State != scenarioruntime.EditorBindingActive ||
			!establishedLeaseContainmentMatches(lease, provenance) {
			return 0, false
		}
		provenanceUnit, unitErr := establishedContainmentScopeUnit(lease, provenance.Scope)
		if unitErr != nil || provenanceUnit != unit {
			return 0, false
		}
	} else if proof == nil {
		// An owner label or a live PID cannot recreate a missing historical
		// launch proof. The exact owner proof is the only supported fallback.
		return 0, false
	}

	wrapperBinding, err := scopeReadProcessBinding(lease.PID)
	if err != nil || wrapperBinding.PID != lease.PID || wrapperBinding.Boot != lease.HostBootID || wrapperBinding.Cgroup == wantCgroup {
		return 0, false
	}
	if proof != nil {
		if wrapperBinding != proof.Binding {
			return 0, false
		}
	} else if wrapperBinding.UID != provenance.UID || wrapperBinding.Start != provenance.ProcessStart {
		return 0, false
	}
	wrapperFact, err := scopeReadProcessFact(lease.PID)
	if err != nil || wrapperFact.PID != lease.PID || wrapperFact.StartTime != wrapperBinding.Start {
		return 0, false
	}

	childPID := 0
	if provenancePresent {
		childPID = provenance.EvidencePID
	} else {
		// Legacy leases have no immutable containment row. A current owner
		// proof plus one exact direct child in the exact session cgroup proves
		// live work conservatively; it still cannot prove eventual absence.
		for _, member := range members {
			entry, ok := entries[member]
			if !ok || strings.HasPrefix(entry.State, "Z") || entry.PPID != lease.PID || entry.Cgroup != wantCgroup {
				continue
			}
			if childPID != 0 {
				return 0, false
			}
			childPID = member
		}
	}
	if childPID <= 1 || !slices.Contains(members, childPID) {
		return 0, false
	}
	entry, exists := entries[childPID]
	if !exists || strings.HasPrefix(entry.State, "Z") || entry.PPID != lease.PID || entry.Cgroup != wantCgroup {
		return 0, false
	}
	childBinding, err := scopeReadProcessBinding(childPID)
	if err != nil || childBinding.PID != childPID || childBinding.UID != wrapperBinding.UID ||
		childBinding.Boot != lease.HostBootID || childBinding.Cgroup != wantCgroup {
		return 0, false
	}
	childFact, err := scopeReadProcessFact(childPID)
	if err != nil || childFact.PID != childPID || childFact.PPID != lease.PID || childFact.StartTime != childBinding.Start {
		return 0, false
	}
	if provenancePresent && (childBinding.UID != provenance.EvidenceUID || childBinding.Boot != provenance.EvidenceBootID || childBinding.Start != provenance.EvidenceProcessStart) {
		return 0, false
	}
	if _, stable := stableScopeBirth(childPID, before, after); !stable {
		return 0, false
	}
	childBirth := before[childPID].StartedAt
	if ref.StartedAt != nil && childBirth.Before(ref.StartedAt.Add(-2*time.Second)) {
		return 0, false
	}
	if !lease.CreatedAt.IsZero() && childBirth.Before(lease.CreatedAt.Add(-2*time.Second)) {
		return 0, false
	}
	// Recheck exact cgroup occupancy after process identity reads to close a
	// scope-move race; process start stamps and owner lease are checked again
	// by the caller's final snapshot and lease revalidation.
	currentMembers, err := scopeEditorScopeMembers(unit)
	if err != nil || !slices.Contains(currentMembers, childPID) {
		return 0, false
	}
	return childPID, true
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

func getScopeEditorLeaseProvenance(ctx context.Context, home, bindingID string) (scenarioruntime.EditorLeaseProvenance, error) {
	if bindingID == "" {
		return scenarioruntime.EditorLeaseProvenance{}, scenarioruntime.ErrNotFound
	}
	path, err := scenarioruntime.DefaultDBPath(home)
	if err != nil {
		return scenarioruntime.EditorLeaseProvenance{}, err
	}
	if _, err := os.Stat(path); err != nil {
		return scenarioruntime.EditorLeaseProvenance{}, err
	}
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home, DBPath: path, ReadOnly: true})
	if err != nil {
		return scenarioruntime.EditorLeaseProvenance{}, err
	}
	defer store.Close()
	return store.GetEditorLeaseProvenance(ctx, bindingID)
}

func getScopeEditorLeaseProvenances(ctx context.Context, home string, bindingIDs []string) (map[string]scenarioruntime.EditorLeaseProvenance, map[string]error) {
	provenance := make(map[string]scenarioruntime.EditorLeaseProvenance, len(bindingIDs))
	provenanceErrors := map[string]error{}
	if len(bindingIDs) == 0 {
		return provenance, provenanceErrors
	}
	path, err := scenarioruntime.DefaultDBPath(home)
	if err != nil {
		for _, id := range bindingIDs {
			provenanceErrors[id] = err
		}
		return provenance, provenanceErrors
	}
	if _, err := os.Stat(path); err != nil {
		for _, id := range bindingIDs {
			provenanceErrors[id] = err
		}
		return provenance, provenanceErrors
	}
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home, DBPath: path, ReadOnly: true})
	if err != nil {
		for _, id := range bindingIDs {
			provenanceErrors[id] = err
		}
		return provenance, provenanceErrors
	}
	defer store.Close()
	for start := 0; start < len(bindingIDs); start += scenarioruntime.EditorLeaseProvenanceQueryBatchSize {
		end := min(start+scenarioruntime.EditorLeaseProvenanceQueryBatchSize, len(bindingIDs))
		batch := bindingIDs[start:end]
		items, readErr := store.GetEditorLeaseProvenances(ctx, batch)
		if readErr != nil {
			for _, id := range batch {
				provenanceErrors[id] = readErr
			}
			continue
		}
		for id, item := range items {
			provenance[id] = item
		}
	}
	return provenance, provenanceErrors
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
func observeScopeProcesses(ctx context.Context) (map[int]processTableEntry, map[int]scopeIdentity, map[int]error, error) {
	identities := map[int]scopeIdentity{}
	identityErrors := map[int]error{}
	observed := map[int]processTableEntry{}
	if runtime.GOOS != "linux" {
		return observed, identities, identityErrors, fmt.Errorf("complete managed executor identity inspection is unsupported on this host")
	}
	for pass := 0; pass < executorScopeProcessTablePasses; pass++ {
		if ctx.Err() != nil {
			return observed, identities, identityErrors, fmt.Errorf("scope observation deadline or cancellation")
		}
		table, err := scopeProcessTable(ctx)
		if err != nil {
			return observed, identities, identityErrors, fmt.Errorf("process table unavailable")
		}
		if len(table) > ExecutorScopeMaxProcesses {
			return observed, identities, identityErrors, fmt.Errorf("process table exceeds bounded executor identity scan")
		}
		pending := false
		identityReads := make([]processTableEntry, 0, len(table))
		for pid, entry := range table {
			if err := ctx.Err(); err != nil {
				return observed, identities, identityErrors, fmt.Errorf("scope observation deadline or cancellation")
			}
			old, exists := observed[pid]
			changed := !exists || old.PGID != entry.PGID || old.PPID != entry.PPID || old.Cgroup != entry.Cgroup
			if changed {
				pending = true
			}
			observed[pid] = entry
			if len(observed) > ExecutorScopeMaxProcesses {
				return observed, identities, identityErrors, fmt.Errorf("observed process set exceeds bounded executor identity scan")
			}
			if strings.HasPrefix(entry.State, "Z") {
				delete(identities, pid)
				delete(identityErrors, pid)
				continue
			}
			identityReads = append(identityReads, entry)
		}
		if err := readScopeProcessIdentities(ctx, identityReads, identities, identityErrors); err != nil {
			return observed, identities, identityErrors, err
		}
		if !pending {
			return observed, identities, identityErrors, nil
		}
	}
	// Host-wide process churn is normal on a development machine. The requested
	// run identity, PID/group identity and birth-time checks below are the
	// exclusion proof; requiring every unrelated host process to remain still
	// makes maintenance permanently unavailable under ordinary scheduler/UI
	// activity. Return the latest bounded snapshot and let matching evidence
	// remain positive or unknown as appropriate.
	return observed, identities, identityErrors, nil
}

type scopeIdentityReadResult struct {
	pid      int
	identity scopeIdentity
	err      error
}

// readScopeProcessIdentities keeps every process-table pass complete while
// limiting concurrent /proc reads and queued work.
func readScopeProcessIdentities(ctx context.Context, entries []processTableEntry, identities map[int]scopeIdentity, identityErrors map[int]error) error {
	if len(entries) == 0 {
		return nil
	}
	workers := min(ExecutorScopeIdentityWorkers, len(entries))
	jobs := make(chan processTableEntry)
	results := make(chan scopeIdentityReadResult, workers)
	var work sync.WaitGroup
	work.Add(workers)
	for range workers {
		go func() {
			defer work.Done()
			for entry := range jobs {
				identity, err := scopeReadIdentity(entry)
				result := scopeIdentityReadResult{pid: entry.PID, identity: identity, err: err}
				select {
				case results <- result:
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, entry := range entries {
			select {
			case jobs <- entry:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		work.Wait()
		close(results)
	}()
	for result := range results {
		if result.err != nil {
			delete(identities, result.pid)
			identityErrors[result.pid] = result.err
			continue
		}
		identities[result.pid] = result.identity
		delete(identityErrors, result.pid)
	}
	if ctx.Err() != nil {
		return fmt.Errorf("scope observation deadline or cancellation")
	}
	return nil
}
