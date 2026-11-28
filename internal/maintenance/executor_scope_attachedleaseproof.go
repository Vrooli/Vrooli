package maintenance

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	platformgo "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/agentscope"
	"github.com/vrooli/vrooli/internal/agentsession"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/supervision"
)

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
