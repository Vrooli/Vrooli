// Package retention contains the storage-manager-owned adapter for directory
// budgets declared in storage.entries. It delegates selection and cleanup to
// api-core/retention's builtin directory pruner; storage-manager only resolves
// the owner path and records whether the bounded operation completed.
package retention

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"runtime"
	"strings"
	"time"

	coreRetention "github.com/vrooli/api-core/retention"
	coreStorage "github.com/vrooli/api-core/storage"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/packages/artifactledger"
)

// Enforcer applies directory budgets from the normalized owner inventory.
// Non-directory entries are intentionally left to their owner-specific
// retention manager; storage-manager must never guess how to prune a file or
// SQLite table it does not own.
type Enforcer struct {
	RepoRoot string
	Platform coreStorage.Platform
	// Ledger receives one removal receipt per pruned entry. A nil Ledger keeps
	// the pruner's default unrecorded os.RemoveAll, which is acceptable only in
	// tests: production wiring supplies one.
	Ledger *artifactledger.Ledger
}

// Result records the outcome for one owner entry. A successful result means
// the builtin provider measured and reconciled the directory, including the
// no-op case where it was already within its ceiling.
//
// A refused result means the entry was measured but deliberately not pruned.
// That is still a governed outcome: the budget keeps working as an alarm, it
// just never works as a deleter.
type Result struct {
	Owner     string
	Entry     string
	Deleted   int
	Freed     int64
	Error     string
	Refused   bool
	Reason    string
	UsedBytes int64
	OverBytes int64
	// EntryResults preserves the outcome for every budgeted entry. The other
	// fields remain an owner-level rollup for compatibility with existing
	// consumers that only need the governance state.
	EntryResults []Result `json:"entry_results,omitempty"`
}

type budgetPruner interface {
	Measure(context.Context) (coreRetention.Usage, error)
	Prune(context.Context, coreRetention.Budget) (coreRetention.Result, error)
}

// Enforce applies every supported storage-entry budget and returns successful
// owner IDs. A failure is returned immediately: retention status must not call
// an owner governed when one of its declared directories could not be checked.
func (e Enforcer) Enforce(ctx context.Context, inventory coreStorage.OwnerInventory) (map[string]Result, error) {
	platform := e.Platform
	if platform == "" {
		platform = coreStorage.Platform(runtime.GOOS)
	}
	// Resolving this first, and failing the whole cycle when it cannot be
	// resolved, is the point. A retention pass that cannot enumerate what it
	// must not delete has no business deleting anything, and the previous
	// version returned an empty set on error -- which read as "nothing is
	// protected" rather than "protection is unavailable".
	protectedRoots, err := protectedRuntimeRoots(e.RepoRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve protected runtime roots: %w", err)
	}
	results := make(map[string]Result)
	for _, owner := range inventory.Owners {
		for _, entry := range owner.StorageEntries {
			if entry.Budget == nil || (entry.Kind != "dir" && entry.Kind != "file") {
				continue
			}
			path, err := coreStorage.ResolveOwnerStoragePath(e.RepoRoot, owner, entry, platform, coreStorage.PlatformSeams{})
			if err != nil {
				var notApplicable *coreStorage.NotApplicable
				if errors.As(err, &notApplicable) {
					continue
				}
				addResult(results, owner.ID, Result{Owner: owner.ID, Entry: entry.Name, Error: fmt.Errorf("resolve storage path: %w", err).Error()})
				continue
			}
			// Contract-protected runtime-home entries are control-plane
			// substrate shared by every scenario and resource. An owner may
			// declare a budget over one -- by mistake, or because it
			// contributes a few files to a shared directory and named the whole
			// directory -- and that declaration must never become a licence to
			// prune it. Refusing here catches the ancestor case too: a budget on
			// ~/.vrooli would otherwise remove ~/.vrooli/bin as one top-level
			// entry.
			if coreRetention.ProtectedPathOverlap(path, protectedRoots) {
				addResult(results, owner.ID, Result{
					Owner: owner.ID, Entry: entry.Name, Refused: true,
					Reason: "path overlaps a contract-protected runtime-home entry, which is never retention-managed",
				})
				continue
			}
			budget := coreRetention.Budget{Name: entry.Name}
			if strings.TrimSpace(entry.Budget.MaxBytes) != "" {
				budget.MaxBytes, err = coreRetention.ParseBytes(entry.Budget.MaxBytes)
				if err != nil {
					addResult(results, owner.ID, Result{Owner: owner.ID, Entry: entry.Name, Error: fmt.Errorf("parse max_bytes: %w", err).Error()})
					continue
				}
			}
			if strings.TrimSpace(entry.Budget.MaxAge) != "" {
				budget.MaxAge, err = coreRetention.ParseAge(entry.Budget.MaxAge)
				if err != nil {
					addResult(results, owner.ID, Result{Owner: owner.ID, Entry: entry.Name, Error: fmt.Errorf("parse max_age: %w", err).Error()})
					continue
				}
			}
			var pruner budgetPruner
			if entry.Kind == "dir" {
				pruner, err = coreRetention.NewDirectoryPruner(coreRetention.DirectoryConfig{
					Path:              path,
					ProtectedRoots:    protectedRoots,
					MaxDeleteFraction: MaxDeleteFraction,
					RemoveHook:        e.recordRemoval(owner.ID, entry.Name),
				})
			} else {
				pruner, err = coreRetention.NewFilePruner(coreRetention.FileConfig{Path: path, ProtectedRoots: protectedRoots})
			}
			if err != nil {
				addResult(results, owner.ID, Result{Owner: owner.ID, Entry: entry.Name, Error: fmt.Errorf("build provider: %w", err).Error()})
				continue
			}
			// Pruning deletes files. For an entry its owner declared it cannot
			// rebuild, the deleted bytes are the only copy, so a budget here is
			// an accountability signal and never a licence to destroy. Measure
			// and report instead: the owner still learns it is over ceiling.
			if !entry.Regenerable {
				usage, measureErr := pruner.Measure(ctx)
				if measureErr != nil {
					addResult(results, owner.ID, Result{Owner: owner.ID, Entry: entry.Name, Error: fmt.Errorf("measure non-regenerable entry: %w", measureErr).Error()})
					continue
				}
				result := Result{
					Owner: owner.ID, Entry: entry.Name, Refused: true, UsedBytes: usage.Bytes,
					Reason: "entry is declared regenerable=false; budgets on non-regenerable data alarm but never prune",
				}
				if budget.MaxBytes > 0 && usage.Bytes > budget.MaxBytes {
					result.OverBytes = usage.Bytes - budget.MaxBytes
				}
				addResult(results, owner.ID, result)
				continue
			}
			out, err := pruner.Prune(ctx, budget)
			if err != nil {
				addResult(results, owner.ID, Result{Owner: owner.ID, Entry: entry.Name, Error: fmt.Errorf("enforce: %w", err).Error()})
				continue
			}
			if out.Refused {
				addResult(results, owner.ID, Result{
					Owner: owner.ID, Entry: entry.Name, Refused: true,
					Reason: out.RefusedReason, UsedBytes: out.Before.Bytes,
					OverBytes: overBytes(out.Before.Bytes, budget.MaxBytes),
				})
				continue
			}
			addResult(results, owner.ID, Result{Owner: owner.ID, Entry: entry.Name, Deleted: int(out.Deleted), Freed: out.FreedBytes})
		}
		var ownerErr error
		if result, ok := results[owner.ID]; ok && result.Error != "" {
			ownerErr = errors.New(result.Error)
		}
		if err := coreRetention.RecordEnforcementReceipt(owner.ID, time.Now().UTC(), ownerErr); err != nil {
			results[owner.ID] = Result{Owner: owner.ID, Error: fmt.Errorf("record enforcement receipt: %w", err).Error()}
		}
	}
	return results, nil
}

// protectedRuntimeRoots returns every runtime-home entry the repository
// contract marks protected, resolved against the invoking user's home.
//
// Two properties matter here, and the previous implementation had neither.
//
// It fails closed. Every failure path used to return an empty slice, which is
// indistinguishable at the call site from "this host has nothing to protect".
// The caller now cannot proceed without a resolved set.
//
// It resolves from the repository root the Enforcer was constructed with,
// rather than re-deriving one from the process environment and working
// directory. That asymmetry was the real hazard: the target path resolves from
// $HOME alone and needs no repository, while the protection needed to locate a
// contract on disk. A storage-manager process started outside a checkout, or
// with a working directory that had been deleted under it, therefore kept
// targeting ~/.vrooli/bin while silently losing every guard on it. One
// authority for both, or they drift again.
func protectedRuntimeRoots(repoRoot string) ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home directory: %w", err)
	}
	contract, err := repocontract.LoadDefault(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("load repo contract from %s: %w", repoRoot, err)
	}
	entries, err := contract.RuntimeHomeEntries(home)
	if err != nil {
		return nil, fmt.Errorf("resolve runtime-home entries: %w", err)
	}
	roots := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.Protected {
			continue
		}
		roots = append(roots, entry.AbsPath)
	}
	if len(roots) == 0 {
		// The canonical contract marks several entries protected. None at all
		// means a contract this build does not understand, and guessing is the
		// wrong direction when the guess authorizes deletion.
		return nil, fmt.Errorf("repo contract at %s declares no protected runtime-home entries", repoRoot)
	}
	return coreRetention.NormalizeProtectedRoots(roots)
}

func addResult(results map[string]Result, ownerID string, entryResult Result) {
	rollup, exists := results[ownerID]
	if !exists {
		rollup = Result{Owner: ownerID, Entry: entryResult.Entry}
	}
	rollup.Deleted += entryResult.Deleted
	rollup.Freed += entryResult.Freed
	rollup.UsedBytes += entryResult.UsedBytes
	rollup.OverBytes += entryResult.OverBytes
	rollup.Refused = rollup.Refused || entryResult.Refused
	// A refusal that reaches the owner level with no reason is the shape this
	// package exists to avoid: an operator sees "refused" and cannot act on it.
	// First reason wins; the per-entry detail stays in EntryResults.
	if rollup.Reason == "" {
		rollup.Reason = entryResult.Reason
	}
	if entryResult.Error != "" {
		if rollup.Error == "" {
			rollup.Error = entryResult.Error
		} else {
			rollup.Error += "; " + entryResult.Error
		}
	}
	rollup.EntryResults = append(rollup.EntryResults, entryResult)
	results[ownerID] = rollup
}

// MaxDeleteFraction bounds how much of a directory one unattended retention
// cycle may remove.
//
// 0.90 leaves ordinary retention untouched: a budget doing its job trims a
// tail, and trimming a tail does not remove nine entries in ten. It catches the
// shape that is never legitimate -- a ceiling so far below the steady-state
// size that oldest-first pruning walks the whole directory. That shape has a
// single cause worth acting on, a wrong declaration, and the right response to
// a wrong declaration is an alarm a human reads, not a directory that empties
// itself every cycle until someone notices.
const MaxDeleteFraction = 0.90

// removalPredicate is the rule recorded on every receipt this adapter writes.
const removalPredicate = "declared storage-entry budget exceeded; oldest top-level entries pruned to the ceiling"

// recordRemoval returns the pruner's removal wrapper, bracketing each deletion
// with a durable receipt.
//
// Retention deleted with no record of any kind. That is the blind spot that
// makes "what emptied this directory" unanswerable after the fact: the bytes
// are gone, and so is every trace of which rule decided they should be. A
// receipt turns that into one grep.
//
// The deletion itself stays inside the pruner. This adapter decides only
// whether a removal is permitted and what is recorded about it, which is also
// what keeps storage-manager free of its own cleanup side effects
// ([REQ:CLN-P0-002]).
//
// Record rather than Guard: Guard's lock file lands beside its subject, which
// for bulk retention would mean one lock file per victim inside the very
// directory being pruned, each becoming an entry the next cycle must account
// for. Exclusion here comes from the retention scheduler, which runs one cycle
// at a time.
//
// A ledger that cannot record fails the removal rather than proceeding without
// it. Recording is cheap and local, so an error means the state directory is
// unwritable -- and deleting while unable to record is exactly the mode this
// seam exists to prevent.
func (e Enforcer) recordRemoval(ownerID, entryName string) func(string, func() error) error {
	if e.Ledger == nil {
		return nil
	}
	component := "storage-manager.retention.Enforcer[" + ownerID + "/" + entryName + "]"
	return func(path string, remove func() error) error {
		err := e.Ledger.Record(artifactledger.Removal{
			Path:      path,
			Kind:      "retention-entry",
			Component: component,
			Predicate: removalPredicate,
		}, remove)
		if errors.Is(err, fs.ErrNotExist) {
			// Already gone is the outcome pruning wanted.
			return nil
		}
		return err
	}
}

// overBytes reports how far used exceeds ceiling, or zero when it does not.
func overBytes(used, ceiling int64) int64 {
	if ceiling <= 0 || used <= ceiling {
		return 0
	}
	return used - ceiling
}
