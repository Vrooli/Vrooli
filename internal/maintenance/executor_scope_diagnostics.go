package maintenance

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	platformgo "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/agentscope"
	"github.com/vrooli/vrooli/internal/agentsession"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/supervision"
)

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
