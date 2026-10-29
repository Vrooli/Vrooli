package providers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	coreRetention "github.com/vrooli/api-core/retention"
	"storage-manager/internal/cleanup"
)

type FileProviderConfig struct {
	ID          string
	Name        string
	Roots       []string
	Description string
	// Tier is supplied by the governed root specification. Empty preserves the
	// historical conditional cache default for callers that have not migrated.
	Tier cleanup.SafetyTier

	// TopLevelEntries makes each immediate child of a root the unit of
	// cleanup, instead of each individual file.
	//
	// This matters for temp and trash roots, where the meaningful artifact is
	// a whole staging directory rather than the files inside it. Per-file
	// granularity there is wrong twice over: it reclaims the bytes but leaves
	// the directory skeleton behind — the incident host accumulated 24,882
	// top-level temp entries, whose inodes alone are non-trivial — and it ages
	// each file independently, so a directory could be half-deleted with its
	// newer files spared, leaving a corrupt fragment that is worse than either
	// keeping or removing the whole thing.
	//
	// Caches with independently reclaimable entries leave this false. Content
	// addressing alone does not establish that safety: shared Go build outputs
	// can still be needed by a later build step, even without an open handle.
	TopLevelEntries bool

	// MeasureBudget caps the wall-clock time spent measuring one root.
	//
	// Zero selects defaultMeasureBudget. The cap exists because measuring a
	// top-level entry means stat-ing its whole subtree, and real hosts have
	// trees far larger than anyone anticipates — the machine this was built on
	// held 5,064,705 files in its trash alone, which took long enough to blow
	// the API write timeout and fail the plan outright.
	MeasureBudget time.Duration

	// Retention limits are used by contract-backed providers. Generic file
	// providers leave these zero and continue to use the active policy only.
	RetentionMaxAge   time.Duration
	RetentionMaxBytes int64
	ProtectedGlobs    []string
	ProtectActive     bool
	LeaseCheck        string
	// OpenHandleSkipsNonDumpable lets an open_handle check pass over
	// processes this user cannot inspect. Only regenerable roots opt in.
	OpenHandleSkipsNonDumpable bool
	RepairClass                string
	OwnershipRepairer          cleanup.OwnershipRepairer
	ActivityObserver           cleanup.TemporaryActivityObserver
}

// defaultMeasureBudget bounds measurement of a single root.
//
// The API server's default WriteTimeout is 30s, and a plan measures several
// roots in sequence, so each one gets a fraction of that. Ten seconds is enough
// to fully measure an ordinary temp directory (measured: 0.9s for 251,051
// entries) while leaving headroom for the response to be written. A plan that
// returns a partial estimate with a warning is strictly more useful than one
// that times out and returns nothing at all.
const defaultMeasureBudget = 10 * time.Second

type FileProvider struct {
	meta              cleanup.ProviderMetadata
	files             cleanup.FileSystem
	clock             cleanup.Clock
	roots             []string
	description       string
	action            string
	topLevelEntries   bool
	recoveryTopLevel  bool
	measureBudget     time.Duration
	retentionMaxAge   time.Duration
	retentionMaxBytes int64
	protectActive     bool
	protectedGlobs    []string
	leaseCheck        string
	skipNonDumpable   bool
	repairClass       string
	repairer          cleanup.OwnershipRepairer
	activityObserver  cleanup.TemporaryActivityObserver

	// foreignRoots are roots governed by another provider or storage owner
	// that lie strictly beneath this provider's roots. See
	// ExcludeGovernedRoots.
	foreignRoots []string

	// memo holds the most recent measurement so a single plan does not walk
	// the same trees twice. See previewMemo.
	memoMu        sync.Mutex
	memo          *previewMemo
	recoveryMu    sync.Mutex
	recoveryWalks map[string]*recoveryWalkState

	// failed remembers entries whose removal or effect-boundary revalidation
	// failed, keyed by path and bound to the entry's identity. See
	// rememberFailedCandidate.
	failedMu sync.Mutex
	failed   map[string]failedCandidate
	// elevation remembers top-level entries holding content this user cannot
	// read (another user's subtree, a mode-000 directory). See
	// rememberNeedsElevation.
	elevation map[string]elevationRecord
}

// failedCandidateBackoff is how long a candidate whose removal failed is kept
// out of later previews.
//
// Without it the oldest unremovable entry is re-offered first on every pass:
// one read-only test fixture in /tmp was retried 130 times over eleven days
// while 57 GB of removable entries behind it were never reached. The failure
// is remembered in process memory only; a restart retries once, which is the
// right trade for a condition (ownership, permissions) that rarely heals on
// its own but costs nothing to re-check occasionally.
const failedCandidateBackoff = 12 * time.Hour

type failedCandidate struct {
	identity string
	until    time.Time
}

// previewMemo caches one measurement result.
//
// orchestrator.Plan calls Estimate and then Preview on every provider, and a
// file provider's Estimate is itself a full preview — the estimated bytes ARE
// the sum of the preview items. Without a memo every plan measures every tree
// exactly twice, which on this project's development host meant 22.5s instead
// of 12s and would exceed the API's 30s WriteTimeout outright once the cache
// providers are enabled.
//
// The entry is deliberately short-lived. Preview results feed Apply, so a
// measurement must reflect something close to the current filesystem; the TTL
// is sized to span the Estimate-then-Preview pair within one plan and nothing
// more. Staleness beyond that window is already handled downstream, where Apply
// re-checks containment and RemoveAll re-checks ownership at the moment of
// deletion.
type previewMemo struct {
	key       string
	computed  time.Time
	preview   cleanup.Preview
	previewer error
}

// memoTTL bounds how long a measurement may be reused.
const memoTTL = 5 * time.Second

// desktopPlatforms are the platforms whose ordinary user-owned directories the
// file providers can walk and remove from with nothing but the standard
// library.
var desktopPlatforms = []string{"linux", "darwin", "windows"}

// trashPlatforms excludes windows deliberately.
//
// The Recycle Bin is not a directory that can be emptied by removing paths; it
// is a SID-scoped store with its own metadata whose supported lifecycle runs
// through the shell API. hostpaths.trashRoots reports no root there, so
// claiming windows support would advertise a capability that always returns
// nothing. See hostpaths/trash_windows.go.
var trashPlatforms = []string{"linux", "darwin"}

func NewTrashProvider(files cleanup.FileSystem, clock cleanup.Clock, cfg FileProviderConfig) *FileProvider {
	// Trash is already an explicit user discard. It has no owner lease or
	// durable scenario state, so it belongs to the autonomous regenerable rung.
	return newFileProvider(files, clock, cfg, cleanup.SafetyTierRegenerable, cleanup.ProviderModeDisabled, cleanup.ApprovalModeNone, "trash-remove", trashPlatforms)
}

func NewTmpProvider(files cleanup.FileSystem, clock cleanup.Clock, cfg FileProviderConfig) *FileProvider {
	provider := newFileProvider(files, clock, cfg, cleanup.SafetyTierSafe, cleanup.ProviderModeDisabled, cleanup.ApprovalModeOperator, "tmp-remove", desktopPlatforms)
	// Legacy previews did not retain descendant-age and identity evidence.
	// They must be regenerated rather than replayed under the stricter contract.
	provider.meta.Version = "v2"
	if cfg.LeaseCheck == "temporary" {
		provider.meta.SupportedPlatforms = []string{"linux"}
		provider.meta.RequiredPrivileges = []string{"privilege-broker"}
		if cfg.TopLevelEntries && cfg.ProtectActive {
			// The balanced profile may automatically reclaim these guarded,
			// aged whole trees. Missing activity proof still blocks every effect.
			provider.meta.DefaultApproval = cleanup.ApprovalModeNone
		}
	}
	return provider
}

// NewScratchProvider reaps the repository's agent-scratch directory.
//
// It is tiered Safe rather than SafeWithOwner: scratch has no owning scenario
// to consult, because the agents that write there are precisely the ones that
// did not route through an owner. Approval stays with the operator, and the
// provider ships disabled by default like every other file provider, so a host
// only reaps scratch once someone turns it on.
func NewScratchProvider(files cleanup.FileSystem, clock cleanup.Clock, cfg FileProviderConfig) *FileProvider {
	return newFileProvider(files, clock, cfg, cleanup.SafetyTierSafe, cleanup.ProviderModeDisabled, cleanup.ApprovalModeOperator, "scratch-remove", desktopPlatforms)
}

func NewCacheProvider(files cleanup.FileSystem, clock cleanup.Clock, cfg FileProviderConfig) *FileProvider {
	tier := cfg.Tier
	if tier == "" || isGoBuildCacheProvider(cfg.ID) {
		tier = cleanup.SafetyTierConditional
	}
	approval := cleanup.ApprovalModeOperator
	if tier == cleanup.SafetyTierRegenerable {
		approval = cleanup.ApprovalModeNone
	}
	return newFileProvider(files, clock, cfg, tier, cleanup.ProviderModeDisabled, approval, "cache-remove", desktopPlatforms)
}

func isGoBuildCacheProvider(id string) bool {
	// Both are canonical registry identities: the current root-spec provider
	// and its legacy adapter. Do not let an old regenerable declaration restore
	// autonomous authority while it still advertises a false NoLease proof.
	return id == "go-build-cache" || id == "spec-go-build-cache"
}

func (p *FileProvider) reclaimBlockedReason() string {
	if isGoBuildCacheProvider(p.meta.ID) {
		// Go promises a Get/Put output remains available through cache Close.
		// File age, a byte cap, and a snapshot of open handles cannot establish
		// that all builds have finished using it. This provider has no tool-owner
		// build-use protocol; even operator approval is not that evidence.
		return "Go build cache reclaim requires tool-owner build-use or quiescence proof; generic filesystem cleanup cannot provide it"
	}
	return ""
}

func newFileProvider(files cleanup.FileSystem, clock cleanup.Clock, cfg FileProviderConfig, tier cleanup.SafetyTier, mode cleanup.ProviderMode, approval cleanup.ApprovalMode, action string, platforms []string) *FileProvider {
	return &FileProvider{
		meta: cleanup.ProviderMetadata{
			ID:                  cfg.ID,
			Name:                cfg.Name,
			Version:             "v1",
			OwnerScenario:       "storage-manager",
			SafetyTier:          tier,
			DefaultMode:         mode,
			DefaultApproval:     approval,
			SupportedPlatforms:  append([]string(nil), platforms...),
			IrreversibleEffects: []string{"filesystem entries are removed from configured cleanup roots"},
			TestSubstitute:      "fake-filesystem",
			NoLease:             tier == cleanup.SafetyTierRegenerable,
			RegenerableProof: cleanup.RegenerableProof{
				Derived: true, ToolRecreates: true, ExactRoot: true,
				NoLease: tier == cleanup.SafetyTierRegenerable,
			},
		},
		files:             files,
		clock:             clock,
		roots:             cleanRoots(cfg.Roots),
		description:       cfg.Description,
		action:            action,
		topLevelEntries:   cfg.TopLevelEntries,
		recoveryTopLevel:  tier == cleanup.SafetyTierRegenerable && cfg.RetentionMaxBytes > 0,
		measureBudget:     cfg.MeasureBudget,
		retentionMaxAge:   cfg.RetentionMaxAge,
		retentionMaxBytes: cfg.RetentionMaxBytes,
		protectedGlobs:    append([]string(nil), cfg.ProtectedGlobs...),
		leaseCheck:        strings.TrimSpace(cfg.LeaseCheck),
		skipNonDumpable:   cfg.OpenHandleSkipsNonDumpable,
		protectActive:     cfg.ProtectActive,
		repairClass:       cfg.RepairClass,
		repairer:          cfg.OwnershipRepairer,
		activityObserver:  cfg.ActivityObserver,
	}
}

func (p *FileProvider) Metadata() cleanup.ProviderMetadata { return p.meta }

func (p *FileProvider) Estimate(ctx context.Context, req cleanup.EstimateRequest) (cleanup.Estimate, error) {
	preview, err := p.preview(ctx, req.Scope, req.Policy)
	if err != nil {
		return cleanup.Estimate{}, err
	}
	return cleanup.Estimate{
		ProviderID:       p.meta.ID,
		ProviderVersion:  p.meta.Version,
		EstimatedBytes:   sumPreviewBytes(preview.Items),
		ItemCount:        len(preview.Items),
		RequiresApproval: req.Policy.ApprovalMode != cleanup.ApprovalModeNone,
		BlockedReason:    preview.BlockedReason,
		ObservedAt:       p.now(req.Scope),
		Notices:          preview.Notices,
	}, nil
}

func (p *FileProvider) Preview(ctx context.Context, req cleanup.PreviewRequest) (cleanup.Preview, error) {
	// Estimate and Preview are one observation, even when traversal is paused.
	// Independent bounded observations must advance rather than reuse an empty
	// partial memo indefinitely.
	if req.Scope.Recovery && req.Estimate.ProviderID == p.meta.ID {
		p.memoMu.Lock()
		if p.memo != nil && p.memo.key == memoKey(req.Scope, req.Policy) {
			out, err := p.memo.preview.Clone(), p.memo.previewer
			p.memoMu.Unlock()
			return out, err
		}
		p.memoMu.Unlock()
	}
	return p.preview(ctx, req.Scope, req.Policy)
}

func (p *FileProvider) Apply(ctx context.Context, req cleanup.ApplyRequest) (cleanup.ApplyResult, error) {
	if req.ProviderVersion != p.meta.Version {
		return cleanup.ApplyResult{}, fmt.Errorf("provider %s version mismatch: got %q want %q", p.meta.ID, req.ProviderVersion, p.meta.Version)
	}
	if req.IdempotencyKey == "" {
		return cleanup.ApplyResult{}, fmt.Errorf("provider %s apply requires idempotency key", p.meta.ID)
	}
	// Recheck at the effect boundary: a persisted preview or standing approval
	// from before the safety repair must not authorize deletion on replay.
	if reason := p.reclaimBlockedReason(); reason != "" {
		return cleanup.ApplyResult{ProviderID: p.meta.ID, SkippedItems: previewItemIDs(req.Preview.Items), Warnings: []string{reason}}, nil
	}
	if req.ApprovalMode == cleanup.ApprovalModeDisabled {
		return cleanup.ApplyResult{ProviderID: p.meta.ID, Applied: false, SkippedItems: previewItemIDs(req.Preview.Items), Warnings: []string{"provider disabled by policy"}}, nil
	}
	var reclaimed int64
	var skipped []string
	var warnings []string
	var repairAttempted bool
	var repairAttempts, repairs, retryAttempts uint64
	var appliedItems []string
	heldAtApply := p.heldAtApply()
	for _, item := range req.Preview.Items {
		if err := ctx.Err(); err != nil {
			return cleanup.ApplyResult{}, err
		}
		if !p.withinConfiguredRoot(item.Path) {
			skipped = append(skipped, item.ID)
			continue
		}
		if p.isConfiguredRoot(item.Path) {
			skipped = append(skipped, item.ID)
			warnings = append(warnings, fmt.Sprintf("%s: configured root %s is protected", item.ID, cleanup.Redact(item.Path)))
			continue
		}
		if p.isProtectedPath(item.Path) {
			skipped = append(skipped, item.ID)
			warnings = append(warnings, fmt.Sprintf("%s: protected path", item.ID))
			continue
		}
		if heldAtApply != nil && heldAtApply(item.Path) {
			skipped = append(skipped, item.ID)
			warnings = append(warnings, fmt.Sprintf("%s: held open by a live process", item.ID))
			continue
		}
		// Re-checked at the effect boundary: a persisted preview from before a
		// nested root was governed must not remove that root or its parent.
		if p.insideForeignRoot(item.Path) || p.containsForeignRoot(item.Path) {
			skipped = append(skipped, item.ID)
			warnings = append(warnings, fmt.Sprintf("%s: overlaps a root governed by another owner", item.ID))
			continue
		}
		info, err := p.files.Stat(ctx, item.Path)
		if err != nil {
			if isFileMissing(err) {
				// A prior apply or an operator may have removed the item after
				// preview. Do not count planned bytes for an entry that is already
				// gone; this keeps retries honest.
				continue
			}
			skipped = append(skipped, item.ID)
			warnings = append(warnings, fmt.Sprintf("%s: %s", item.ID, cleanup.Redact(err.Error())))
			continue
		}
		if item.Identity != "" && info.Identity != item.Identity {
			skipped = append(skipped, item.ID)
			warnings = append(warnings, fmt.Sprintf("%s: entry identity changed", item.ID))
			continue
		}
		if p.topLevelEntries && p.meta.SafetyTier == cleanup.SafetyTierSafe {
			// Invalidate acceleration after every apply attempt, including a
			// safety refusal, so one changed entry cannot starve later entries.
			p.forgetRecoveryCandidate(item.Path)
			if err := p.validateTemporaryApply(ctx, item, req.Preview.MinAge); err != nil {
				skipped = append(skipped, item.ID)
				warnings = append(warnings, fmt.Sprintf("%s: %s", item.ID, cleanup.Redact(err.Error())))
				p.rememberFailedCandidate(item.Path, info.Identity)
				continue
			}
			current, statErr := p.files.Stat(ctx, item.Path)
			if statErr != nil || (info.Identity != "" && current.Identity != info.Identity) || !current.ModTime.Equal(info.ModTime) {
				skipped = append(skipped, item.ID)
				warnings = append(warnings, fmt.Sprintf("%s: temporary entry changed during revalidation", item.ID))
				p.rememberFailedCandidate(item.Path, info.Identity)
				continue
			}
		}
		if err := p.files.RemoveAll(ctx, item.Path); err != nil {
			if !repairAttempted && p.repairer != nil && p.repairClass != "" && isPermissionError(err) {
				repairAttempted = true
				repairAttempts++
				repairCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				repair, repairErr := p.repairer.Repair(repairCtx, p.repairClass)
				cancel()
				if repairErr != nil {
					warnings = append(warnings, fmt.Sprintf("ownership repair unavailable for %s: %s", p.repairClass, cleanup.Redact(repairErr.Error())))
				} else if repair.Failed > 0 || repair.Repaired == 0 {
					code := repair.Code
					if code == "" {
						code = "runtime_home_repair_refused"
					}
					warnings = append(warnings, fmt.Sprintf("ownership repair refused for %s: %s", p.repairClass, code))
				} else {
					repairs += repair.Repaired
					retryAttempts++
					if retryErr := p.files.RemoveAll(ctx, item.Path); retryErr == nil {
						reclaimed += item.Bytes
						appliedItems = append(appliedItems, item.ID)
						continue
					} else {
						warnings = append(warnings, fmt.Sprintf("cleanup retry failed for %s: %s", item.ID, cleanup.Redact(retryErr.Error())))
					}
				}
			}
			// A single unremovable entry must not abandon the rest of the run.
			//
			// Previously this returned immediately with a zero-valued result,
			// which was wrong twice: it gave up on thousands of removable
			// entries because of one, and it discarded the byte count for
			// everything already deleted, so the audit trail under-reported
			// real deletions. Under disk pressure the whole point is to
			// reclaim what can be reclaimed, and hitting an entry owned by
			// another user or vanished mid-run is routine, not exceptional.
			skipped = append(skipped, item.ID)
			warnings = append(warnings, fmt.Sprintf("%s: %s", item.ID, cleanup.Redact(err.Error())))
			// The same entry will fail the same way next pass. Remembering it is
			// what lets the next preview reach the entries behind it.
			p.rememberFailedCandidate(item.Path, info.Identity)
			continue
		}
		reclaimed += item.Bytes
		appliedItems = append(appliedItems, item.ID)
	}
	return cleanup.ApplyResult{
		ProviderID:     p.meta.ID,
		Applied:        reclaimed > 0,
		AppliedItems:   appliedItems,
		ReclaimedBytes: reclaimed,
		SkippedItems:   skipped,
		Warnings:       warnings,
		RepairAttempts: repairAttempts,
		Repairs:        repairs,
		RetryAttempts:  retryAttempts,
	}, nil
}

func (p *FileProvider) Verify(context.Context, cleanup.VerifyRequest) (cleanup.VerifyResult, error) {
	return cleanup.VerifyResult{Verified: true, Message: "filesystem provider applied through FileSystem seam"}, nil
}

// rememberFailedCandidate keeps an entry whose removal or revalidation failed
// out of previews for failedCandidateBackoff. The record is bound to the
// entry's identity: a new entry that reuses the name is a different candidate
// and is measured normally. Skipping a candidate never authorizes anything, so
// this is acceleration only; Apply still revalidates whatever it is handed.
func (p *FileProvider) rememberFailedCandidate(path, identity string) {
	p.forgetRecoveryCandidate(path)
	p.failedMu.Lock()
	defer p.failedMu.Unlock()
	now := p.failureNow()
	if p.failed == nil {
		p.failed = make(map[string]failedCandidate)
	}
	// Bound the memory: expired records are dropped whenever a new one lands.
	for key, record := range p.failed {
		if !now.Before(record.until) {
			delete(p.failed, key)
		}
	}
	p.failed[filepath.Clean(path)] = failedCandidate{identity: identity, until: now.Add(failedCandidateBackoff)}
}

// recentlyFailed reports whether path is inside its failure back-off. An
// unknown identity on either side is treated as the same entry, because the
// conservative direction for a skip is to keep skipping.
func (p *FileProvider) recentlyFailed(path, identity string) bool {
	p.failedMu.Lock()
	defer p.failedMu.Unlock()
	record, ok := p.failed[filepath.Clean(path)]
	if !ok {
		return false
	}
	if !p.failureNow().Before(record.until) {
		delete(p.failed, filepath.Clean(path))
		return false
	}
	return record.identity == "" || identity == "" || record.identity == identity
}

// failureNow reads the injected clock when there is one. Unlike observedNow it
// never falls back to a zero instant, which would make a back-off permanent.
func (p *FileProvider) failureNow() time.Time {
	if p.clock != nil {
		return p.clock.Now()
	}
	return time.Now()
}

// preview measures the configured roots, reusing a very recent identical
// measurement when one exists.
func (p *FileProvider) preview(ctx context.Context, scope cleanup.ObservationScope, policy cleanup.ProviderPolicy) (cleanup.Preview, error) {
	if reason := p.reclaimBlockedReason(); reason != "" {
		return cleanup.Preview{ProviderID: p.meta.ID, ProviderVersion: p.meta.Version, BlockedReason: reason}, nil
	}
	key := memoKey(scope, policy)

	p.memoMu.Lock()
	if !scope.Recovery && p.memo != nil && p.memo.key == key && p.observedNow(scope.Now).Sub(p.memo.computed) < memoTTL {
		cached := p.memo
		p.memoMu.Unlock()
		return cached.preview.Clone(), cached.previewer
	}
	p.memoMu.Unlock()

	out, err := p.measure(ctx, scope, policy)

	p.memoMu.Lock()
	p.memo = &previewMemo{key: key, computed: p.observedNow(scope.Now), preview: out, previewer: err}
	p.memoMu.Unlock()

	return out.Clone(), err
}

// memoKey identifies a measurement by everything that changes its result.
func memoKey(scope cleanup.ObservationScope, policy cleanup.ProviderPolicy) string {
	return fmt.Sprintf("%v|%v|%v|%v|%v|%v|%v|%v|%v",
		scope.Now.UnixNano(), scope.RootPaths, scope.CompleteCensus, scope.Recovery,
		policy.Enabled, policy.MinAge, policy.MaxBytes, policy.ApprovalMode, policy.AllowFreshReclaim)
}

func (p *FileProvider) measure(ctx context.Context, scope cleanup.ObservationScope, policy cleanup.ProviderPolicy) (cleanup.Preview, error) {
	out := cleanup.Preview{ProviderID: p.meta.ID, ProviderVersion: p.meta.Version}
	if scope.Recovery && p.meta.SafetyTier == cleanup.SafetyTierRegenerable {
		// Regenerable roots are explicitly safe to recreate. Under pressure the
		// declared byte budget is the authority, so fresh entries may be
		// reclaimed; waiting for the normal age horizon defeats the R1 rung.
		policy.AllowFreshReclaim = true
		policy.MinAge = 0
	}
	if p.retentionMaxAge > 0 && !policy.AllowFreshReclaim && (policy.MinAge == 0 || policy.MinAge < p.retentionMaxAge) {
		policy.MinAge = p.retentionMaxAge
	}
	if p.retentionMaxBytes > 0 && (policy.MaxBytes == 0 || policy.MaxBytes > p.retentionMaxBytes) {
		policy.MaxBytes = p.retentionMaxBytes
	}
	if !policy.Enabled {
		out.BlockedReason = "provider disabled by policy"
		return out, nil
	}
	if p.files == nil {
		out.BlockedReason = "filesystem seam unavailable"
		return out, nil
	}
	if p.topLevelEntries && p.meta.SafetyTier == cleanup.SafetyTierSafe && !scope.CompleteCensus {
		if resumable, ok := p.files.(cleanup.ResumableFileSystem); ok {
			return p.measureTemporaryRecovery(ctx, resumable, scope, policy)
		}
	}

	roots := p.roots
	if len(scope.RootPaths) > 0 {
		roots = intersectRoots(p.roots, cleanRoots(scope.RootPaths))
	}
	now := p.now(scope)
	measureCtx := ctx
	var cancel context.CancelFunc
	if !scope.CompleteCensus {
		measureCtx, cancel = context.WithTimeout(ctx, p.budget())
		defer cancel()
	}
	// One budget for the whole provider, not one per root. A per-root budget
	// would scale the worst case with however many roots a provider happens to
	// have — the trash alone has two (files/ and info/), which doubled its
	// share of the API write timeout for no reason.
	var deadline time.Time
	if !scope.CompleteCensus {
		deadline = p.observedNow(now).Add(p.budget())
	}
	for _, root := range roots {
		var err error
		if p.topLevelEntries || (scope.Recovery && p.recoveryTopLevel) {
			err = p.collectTopLevelEntries(measureCtx, root, now, deadline, policy, &out)
		} else {
			err = p.collectFiles(measureCtx, root, now, policy, &out)
		}
		if err != nil {
			// The provider's own measurement budget expiring is a bounded
			// observation, not a failed one. Every item already in out is a
			// complete unit (top-level entries are appended only after a full
			// subtree walk; per-file selection appends nothing on error), so
			// returning them with a warning lets pressure recovery act on what
			// was measured instead of discarding it as "estimate failed".
			// A caller's own cancellation or deadline is still an error.
			if ctx.Err() == nil && measureCtx.Err() != nil && errors.Is(err, context.DeadlineExceeded) {
				out.Warnings = append(out.Warnings, fmt.Sprintf("measurement budget of %s exhausted under %s; entries measured so far are retained", p.budget(), cleanup.Redact(root)))
				break
			}
			return cleanup.Preview{}, err
		}
	}
	// Plans present items in a stable path order. Recovery keeps the
	// collectors' least-recently-written-first order, because when only one
	// indivisible unit fits a batch it must be the stalest one, not the one
	// whose name sorts first.
	if !scope.Recovery {
		sort.Slice(out.Items, func(i, j int) bool { return out.Items[i].Path < out.Items[j].Path })
	}
	out.MinAge = policy.MinAge
	return out, nil
}

func (p *FileProvider) isConfiguredRoot(path string) bool {
	for _, root := range p.roots {
		if filepath.Clean(path) == root {
			return true
		}
	}
	return false
}

func (p *FileProvider) validateTemporaryApply(ctx context.Context, item cleanup.PreviewItem, minAge time.Duration) error {
	root := ""
	for _, configured := range p.roots {
		if pathContains(configured, item.Path) {
			root = configured
			break
		}
	}
	var active func(string) bool
	if p.leaseCheck == "temporary" {
		verified, err := p.temporaryActivityFilter(ctx, root)
		if err != nil {
			return err
		}
		active = verified
	} else {
		active = p.activePredicate(root)
	}
	if active(item.Path) {
		return fmt.Errorf("temporary entry is active")
	}
	cutoff := p.observedNow(time.Time{}).Add(-minAge)
	check := func(info cleanup.FileInfo) error {
		if info.Mode&(fs.ModeSocket|fs.ModeNamedPipe|fs.ModeDevice|fs.ModeCharDevice) != 0 {
			return fmt.Errorf("temporary entry contains a protected special file")
		}
		// The enclosing entry's process snapshot protects every held descendant.
		// Walking already visits every lease marker; probing ancestor markers for
		// each file would turn a large tree into millions of redundant stat calls.
		if info.Path != item.Path && (p.temporaryDescendantActive(info.Path) || p.isProtectedPath(info.Path)) {
			return fmt.Errorf("temporary descendant is active or protected")
		}
		if minAge > 0 && info.ModTime.After(cutoff) {
			return fmt.Errorf("temporary descendant is newer than retention")
		}
		return nil
	}
	if resumable, ok := p.files.(cleanup.ResumableFileSystem); ok {
		walk, err := resumable.OpenWalk(ctx, item.Path)
		if err != nil {
			return err
		}
		defer walk.Close()
		for {
			info, err := walk.Next(ctx)
			if errors.Is(err, io.EOF) {
				if p.leaseCheck == "temporary" {
					current, observeErr := p.temporaryActivityFilter(ctx, root)
					if observeErr != nil {
						return observeErr
					}
					if current(item.Path) {
						return fmt.Errorf("temporary entry became active during revalidation")
					}
				}
				return nil
			}
			if err != nil {
				return err
			}
			if err := check(info); err != nil {
				return err
			}
		}
	}
	return p.files.Walk(ctx, item.Path, check)
}

// collectFiles adds one candidate per individual file beneath root.
func (p *FileProvider) collectFiles(ctx context.Context, root string, now time.Time, policy cleanup.ProviderPolicy, out *cleanup.Preview) error {
	remaining := policy.MaxBytes
	if p.retentionMaxBytes > 0 {
		used, err := p.rootBytes(ctx, root)
		if err != nil {
			return err
		}
		remaining = used - p.retentionMaxBytes
		if remaining <= 0 {
			return nil
		}
		if policy.MaxBytes > 0 && policy.MaxBytes < remaining {
			remaining = policy.MaxBytes
		}
		// A declared byte ceiling is an enforced size policy, not an age
		// policy. Once a regenerable root exceeds its ceiling, its oldest
		// entries are eligible even when they are newer than the ordinary
		// retention horizon; otherwise a fresh cache can remain permanently
		// over budget and pressure recovery cannot make progress.
		policy.MinAge = 0
	}
	if remaining > 0 {
		remaining -= sumPreviewBytes(out.Items)
		if remaining <= 0 {
			out.Warnings = append(out.Warnings, "max reclaim limit reached")
			return nil
		}
	}
	selected, err := coreRetention.SelectFiles(ctx, cleanupFileWalker{files: p.files}, coreRetention.FileSelectionConfig{
		Root:                 root,
		Now:                  now,
		MinAge:               policy.MinAge,
		MaxBytes:             remaining,
		AllowSingleOvershoot: p.retentionMaxBytes > 0,
		IsActive:             p.excludingForeignRoots(p.activePredicate(root)),
		ProtectedGlobs:       p.protectedGlobs,
	})
	if err != nil {
		return err
	}
	for _, candidate := range selected {
		out.Items = append(out.Items, p.previewItem(candidate.Path, candidate.Bytes))
	}
	if policy.MaxBytes > 0 && sumPreviewBytes(out.Items) >= policy.MaxBytes {
		out.Warnings = append(out.Warnings, "max reclaim limit reached")
	}
	return nil
}

func (p *FileProvider) rootBytes(ctx context.Context, root string) (int64, error) {
	var total int64
	err := p.files.Walk(ctx, root, func(info cleanup.FileInfo) error {
		// Another owner's nested root is measured against that owner's budget;
		// counting it here would hold this root permanently over its ceiling.
		if !info.IsDir && info.Size > 0 && !p.insideForeignRoot(info.Path) {
			total += info.Size
		}
		return nil
	})
	return total, err
}

// cleanupFileWalker adapts storage-manager's injected filesystem, including
// its privileged host implementation and test fakes, to the engine-neutral
// retention selector. The adapter carries facts only; it cannot delete.
type cleanupFileWalker struct{ files cleanup.FileSystem }

func (w cleanupFileWalker) Walk(ctx context.Context, root string, visit func(coreRetention.FileEntry) error) error {
	return w.files.Walk(ctx, root, func(info cleanup.FileInfo) error {
		return visit(coreRetention.FileEntry{Path: info.Path, Size: info.Size, Mode: info.Mode, ModTime: info.ModTime, IsDir: info.IsDir})
	})
}

// entryAggregate accumulates one top-level entry's subtree.
type entryAggregate struct {
	bytes     int64
	newest    time.Time
	protected bool
}

// collectTopLevelEntries adds one candidate per immediate child of root, sized
// by its whole subtree.
//
// The age test uses the NEWEST modification time anywhere beneath the entry,
// not the directory's own. A directory's mtime only changes when entries are
// added or removed from it directly, so a build staging directory created eight
// days ago and actively written to a minute ago still presents an eight-day-old
// mtime. Deleting it would destroy live work while every timestamp the naive
// check consulted said it was safely stale. Taking the maximum over the subtree
// is what makes "nothing has touched this in N days" actually true.
//
// Each entry is measured independently, under a shared time budget. That
// combination is what keeps a cleanup pass bounded on a real host: measuring
// this machine's trash meant stat-ing 5,064,705 files, which took long enough
// to exhaust the API write timeout and fail the entire plan with a 500. An
// entry that cannot be measured within the budget is DROPPED rather than
// estimated, so a partially-traversed subtree is never judged stale on the
// strength of whichever files the walk happened to reach first.
func (p *FileProvider) collectTopLevelEntries(ctx context.Context, root string, now time.Time, deadline time.Time, policy cleanup.ProviderPolicy, out *cleanup.Preview) error {
	isActive := p.activePredicate(root)
	if p.leaseCheck == "temporary" {
		verified, err := p.temporaryActivityFilter(ctx, root)
		if err != nil {
			out.BlockedReason = cleanup.Redact(err.Error())
			out.Warnings = append(out.Warnings, out.BlockedReason)
			return nil
		}
		isActive = verified
	}
	entries, err := p.files.ReadDir(ctx, root)
	if err != nil {
		return err
	}
	entries, err = p.expandGovernedUnits(ctx, entries)
	if err != nil {
		return err
	}

	// Spend the shared budget on the entries most likely to be eligible first.
	// Alphabetical order starves late names when a root is large; own mtime is a
	// safe prioritisation signal because it never makes a fresh entry eligible.
	// Path is the deterministic tie-breaker, independent of filesystem order.
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].ModTime.Equal(entries[j].ModTime) {
			return entries[i].Path < entries[j].Path
		}
		return entries[i].ModTime.Before(entries[j].ModTime)
	})

	var unmeasured, backedOff int
	// measuredBytes is a lower bound on the root's usage: every fully measured
	// entry, eligible or not. It is the only evidence this pass has that the
	// root exceeds its declared ceiling.
	var measuredBytes int64
	type measuredUnit struct {
		item   cleanup.PreviewItem
		newest time.Time
	}
	var eligible []measuredUnit
	defer func() {
		// Least recently written first, and the reclaim cap is applied in that
		// order. Entry mtime only orders the walk; the newest descendant is
		// what says which unit nothing has touched longest.
		sort.SliceStable(eligible, func(i, j int) bool {
			if eligible[i].newest.Equal(eligible[j].newest) {
				return eligible[i].item.Path < eligible[j].item.Path
			}
			return eligible[i].newest.Before(eligible[j].newest)
		})
		for _, unit := range eligible {
			if policy.MaxBytes > 0 && sumPreviewBytes(out.Items)+unit.item.Bytes > policy.MaxBytes {
				out.Warnings = append(out.Warnings, "max reclaim limit reached")
				continue
			}
			out.Items = append(out.Items, unit.item)
		}
		if p.meta.SafetyTier == cleanup.SafetyTierRegenerable && p.retentionMaxBytes > 0 && measuredBytes > p.retentionMaxBytes {
			// A regenerable root proven over its declared ceiling may shed one
			// whole unit even when that unit is larger than the controller's
			// batch. Its entries are indivisible (a module host, a store
			// generation), so without this a root whose units are all large can
			// never move toward its ceiling. Roots at or under the ceiling, and
			// roots whose usage was not measured, keep the ordinary batch cap.
			out.AllowSingleOvershoot = true
		}
		if backedOff > 0 {
			out.Warnings = append(out.Warnings, fmt.Sprintf("%d entries under %s skipped after a recent removal failure", backedOff, cleanup.Redact(root)))
		}
	}()

	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !p.withinConfiguredRoot(entry.Path) || isActive(entry.Path) || p.isProtectedPath(entry.Path) {
			continue
		}
		if p.recentlyFailed(entry.Path, entry.Identity) {
			backedOff++
			continue
		}
		// The newest mtime in a subtree is at least as new as its top-level
		// entry's own mtime. A fresh entry therefore cannot contain an entirely
		// stale subtree and may be skipped without walking it. The converse is
		// unsound: an old directory may contain a fresh descendant, so old
		// entries still receive the full newest-mtime walk below.
		if policy.MinAge > 0 && entry.ModTime.After(now.Add(-policy.MinAge)) {
			continue
		}
		// Checked per entry rather than per file, so an entry is always either
		// fully measured or not measured at all.
		if !deadline.IsZero() && !p.observedNow(now).Before(deadline) {
			unmeasured++
			continue
		}

		agg, err := p.aggregateEntry(ctx, entry, deadline)
		if err != nil {
			// The budget ran out partway through this entry. Checking only
			// before starting each entry bounds nothing on its own: one
			// directory holding millions of files can overshoot the budget by
			// any amount. Enforcing the deadline inside the traversal too is
			// what makes the bound real. The entry is dropped, never
			// half-measured.
			if errors.Is(err, errMeasurementBudgetExhausted) {
				unmeasured++
				continue
			}
			return err
		}
		measuredBytes += agg.bytes
		if agg.protected {
			// A unit is removed whole, so a declared protected path anywhere
			// beneath it protects the whole unit.
			continue
		}
		if policy.MinAge > 0 && now.Sub(agg.newest) < policy.MinAge {
			continue
		}
		item := p.previewItem(entry.Path, agg.bytes)
		item.Identity, item.ModTime = entry.Identity, entry.ModTime
		eligible = append(eligible, measuredUnit{item: item, newest: agg.newest})
	}

	if unmeasured > 0 {
		// Surfaced, never silent. An estimate that quietly under-reports is
		// indistinguishable from a host with nothing to clean, which is the
		// exact failure mode this provider exists to prevent.
		out.Warnings = append(out.Warnings, fmt.Sprintf(
			"measurement budget of %s exhausted: %d entries under %s were not measured and are excluded from this plan",
			p.budget(), unmeasured, root))
	}
	return nil
}

// errMeasurementBudgetExhausted signals that an entry could not be measured in
// the time available. It is internal: callers convert it into an unmeasured
// count and a preview warning, never into a failed plan.
var errMeasurementBudgetExhausted = errors.New("measurement budget exhausted")

// aggregateEntry measures one top-level entry's whole subtree.
//
// The deadline is enforced during traversal, not merely before it. A partially
// traversed subtree yields errMeasurementBudgetExhausted and no aggregate,
// because a subtree measured halfway can report a stale newest-mtime and would
// then look safe to delete while holding a file the walk never reached.
func (p *FileProvider) aggregateEntry(ctx context.Context, entry cleanup.FileInfo, deadline time.Time) (entryAggregate, error) {
	agg := entryAggregate{bytes: entry.Size, newest: entry.ModTime}
	if !entry.IsDir {
		return agg, nil
	}

	var checked int
	err := p.files.Walk(ctx, entry.Path, func(info cleanup.FileInfo) error {
		// Sampling the clock rather than reading it per file: a stat-bound
		// walk visits millions of entries and a time syscall each is real
		// overhead, while 1024 files is a small enough stride that overshoot
		// stays negligible.
		checked++
		if checked%1024 == 0 && !deadline.IsZero() && !p.observedNow(time.Time{}).Before(deadline) {
			return errMeasurementBudgetExhausted
		}
		if info.Path == entry.Path {
			return nil
		}
		// Directories contribute their own inode size and, more importantly,
		// their timestamps: an empty directory is still a real entry with a
		// real age.
		agg.bytes += info.Size
		if info.ModTime.After(agg.newest) {
			agg.newest = info.ModTime
		}
		if p.isProtectedPath(info.Path) {
			agg.protected = true
		}
		return nil
	})
	if err != nil {
		return entryAggregate{}, err
	}
	return agg, nil
}

// budget is the wall-clock ceiling for measuring a single root.
func (p *FileProvider) budget() time.Duration {
	if p.measureBudget > 0 {
		return p.measureBudget
	}
	return defaultMeasureBudget
}

// observedNow reports the current instant for budget accounting.
//
// It reads the injected clock so a test can exhaust the budget deterministically
// instead of sleeping. When a scope pinned the observation instant and no clock
// is available, budget accounting cannot advance, so the budget never expires —
// which is the right behaviour for a fake-driven unit test measuring an
// in-memory tree.
func (p *FileProvider) observedNow(fallback time.Time) time.Time {
	if p.clock != nil {
		return p.clock.Now()
	}
	return fallback
}

func (p *FileProvider) previewItem(path string, bytes int64) cleanup.PreviewItem {
	return cleanup.PreviewItem{
		ID:          stableItemID(p.meta.ID, path),
		Path:        path,
		Description: p.description,
		Bytes:       bytes,
		Action:      p.action,
		SafetyTier:  p.meta.SafetyTier,
	}
}

func (p *FileProvider) now(scope cleanup.ObservationScope) time.Time {
	if !scope.Now.IsZero() {
		return scope.Now
	}
	if p.clock != nil {
		return p.clock.Now()
	}
	return time.Time{}
}

func (p *FileProvider) withinConfiguredRoot(path string) bool {
	cleanPath := filepath.Clean(path)
	for _, root := range p.roots {
		if cleanPath == root || strings.HasPrefix(cleanPath, root+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func (p *FileProvider) isActivePath(path string) bool {
	return activePath(path) || (p.protectActive && activeLeasePath(path))
}

// heldAtApply re-reads live handles at the effect boundary for providers that
// protect open files. A preview can be minutes old under recovery, and a
// process that opened a file since then must still keep it. Nil means the
// provider has no open-handle check.
func (p *FileProvider) heldAtApply() func(string) bool {
	if p.leaseCheck != "open_handle" {
		return nil
	}
	predicates := map[string]func(string) bool{}
	return func(path string) bool {
		for _, root := range p.roots {
			if !pathContains(root, path) {
				continue
			}
			predicate, ok := predicates[root]
			if !ok {
				predicate = p.activePredicate(root)
				predicates[root] = predicate
			}
			return predicate(path)
		}
		return true
	}
}

// Hidden payloads (.cache, .git, etc.) inside an expired temporary unit are
// ordinary contents. Explicit live/pin markers remain protection signals.
//
// Under lease_check open_handle the process snapshot is the liveness proof, so
// name-only markers (*.lock, .keep, *.sock names) are advisory: a stale Go test
// work directory carries its fake home's credentials.toml.lock and .keep files
// forever, and treating those as live held an 11 GB directory for weeks with no
// process anywhere near it. Explicit lease declarations still protect.
func (p *FileProvider) temporaryDescendantActive(path string) bool {
	name := strings.ToLower(filepath.Base(path))
	switch name {
	case ".building", ".lease", ".active", ".running", ".in-progress", ".pin":
		return true
	case ".keep":
		return p.leaseCheck != "open_handle"
	}
	if p.leaseCheck == "open_handle" {
		return p.protectActive && activeLeasePath(path)
	}
	return strings.HasSuffix(name, ".lock") || strings.HasSuffix(name, ".sock") ||
		strings.HasPrefix(name, "systemd-private-") || (p.protectActive && activeLeasePath(path))
}

// activePredicate builds the lease-aware safety filter once per measured root.
// In particular, open-handle discovery must not scan /proc once per file in a
// large cache. Unsupported lease mechanisms fail closed: the provider reports
// protected candidates instead of treating unverifiable bytes as reclaimable.
func (p *FileProvider) activePredicate(root string) func(string) bool {
	base := p.isActivePath
	switch p.leaseCheck {
	case "lease_file":
		return func(path string) bool {
			return base(path) || hasLeaseFile(path, root)
		}
	case "model_pin":
		return func(string) bool { return true }
	case "open_handle", "temporary":
		if p.leaseCheck == "temporary" {
			predicate, err := p.temporaryActivityFilter(context.Background(), root)
			if err != nil {
				return func(string) bool { return true }
			}
			return predicate
		}
		handles, verified := openHandlePaths(root, p.skipNonDumpable)
		return p.handlePredicate(root, handles, verified)
	default:
		return base
	}
}

func (p *FileProvider) temporaryActivityFilter(ctx context.Context, root string) (func(string) bool, error) {
	// Same-user inspection cannot exclude files held by a root or foreign
	// process. The setup-owned observer is authoritative for the host scope.
	if p.activityObserver == nil {
		return nil, fmt.Errorf("temporary process activity is unverifiable; run vrooli setup --sudo-mode=ask to provision the privilege broker")
	}
	handles, err := p.activityObserver.ObserveTemporary(ctx, root)
	if err != nil {
		return nil, err
	}
	return p.handlePredicate(root, handles, true), nil
}

func (p *FileProvider) handlePredicate(root string, handles map[string]struct{}, verified bool) func(string) bool {
	if !verified {
		return func(string) bool { return true }
	}
	return func(path string) bool {
		if p.isActivePath(path) || (p.leaseCheck == "temporary" && hasLeaseFile(path, root)) {
			return true
		}
		clean := filepath.Clean(path)
		for held := range handles {
			if pathContains(held, clean) || pathContains(clean, held) {
				return true
			}
		}
		return false
	}
}

func (p *FileProvider) isProtectedPath(path string) bool {
	cleanPath := filepath.Clean(path)
	for _, pattern := range p.protectedGlobs {
		pattern = filepath.Clean(strings.TrimSpace(pattern))
		if pattern == "." || pattern == "" {
			continue
		}
		if matched, _ := filepath.Match(pattern, filepath.Base(cleanPath)); matched {
			return true
		}
		if matched, _ := filepath.Match(pattern, filepath.ToSlash(cleanPath)); matched {
			return true
		}
		// Contract globs such as "cache/lock" or "*/.building" are written
		// relative to the declared root. Matching only the base name or the
		// absolute path left every multi-segment declaration inert.
		for _, root := range p.roots {
			if rel, err := filepath.Rel(root, cleanPath); err == nil && pathContains(root, cleanPath) && rel != "." {
				if matched, _ := filepath.Match(filepath.ToSlash(pattern), filepath.ToSlash(rel)); matched {
					return true
				}
			}
		}
	}
	return false
}

// GovernedRoots reports the roots this provider governs, so the registry can
// keep every other provider from treating them as an ordinary descendant.
func (p *FileProvider) GovernedRoots() []string { return append([]string(nil), p.roots...) }

// ExcludeGovernedRoots records the governed roots nested strictly beneath this
// provider's own roots. A cleanup unit is removed whole, so a unit that
// contains another owner's root is never a unit: ~/.vrooli/cache/vrooli held
// 115 GB including scenario-to-desktop staging and every scenario's cache
// namespace, and the runtime-home cache provider offered it as one entry.
// Nested roots are withheld outright; their ancestors are descended into so
// sibling entries stay reclaimable.
func (p *FileProvider) ExcludeGovernedRoots(all []string) {
	var nested []string
	for _, candidate := range cleanRoots(all) {
		if p.isConfiguredRoot(candidate) {
			continue
		}
		for _, root := range p.roots {
			if pathContains(root, candidate) {
				nested = append(nested, candidate)
				break
			}
		}
	}
	sort.Strings(nested)
	p.foreignRoots = slices.Compact(nested)
}

// insideForeignRoot reports whether path is a nested governed root or lies
// beneath one.
func (p *FileProvider) insideForeignRoot(path string) bool {
	for _, root := range p.foreignRoots {
		if pathContains(root, path) {
			return true
		}
	}
	return false
}

// containsForeignRoot reports whether path is a strict ancestor of a nested
// governed root, so removing it whole would remove that root too.
func (p *FileProvider) containsForeignRoot(path string) bool {
	clean := filepath.Clean(path)
	for _, root := range p.foreignRoots {
		if root != clean && pathContains(clean, root) {
			return true
		}
	}
	return false
}

func (p *FileProvider) excludingForeignRoots(active func(string) bool) func(string) bool {
	if len(p.foreignRoots) == 0 {
		return active
	}
	return func(path string) bool { return active(path) || p.insideForeignRoot(path) }
}

// expandGovernedUnits replaces every entry that contains a nested governed
// root with its children, recursively, and drops the nested roots themselves.
// What remains are units that can be removed whole without touching another
// owner's root.
func (p *FileProvider) expandGovernedUnits(ctx context.Context, entries []cleanup.FileInfo) ([]cleanup.FileInfo, error) {
	if len(p.foreignRoots) == 0 {
		return entries, nil
	}
	out := make([]cleanup.FileInfo, 0, len(entries))
	queue := append([]cleanup.FileInfo(nil), entries...)
	for len(queue) > 0 {
		entry := queue[0]
		queue = queue[1:]
		switch {
		case p.insideForeignRoot(entry.Path):
			continue
		case p.containsForeignRoot(entry.Path):
			if !entry.IsDir {
				continue
			}
			children, err := p.files.ReadDir(ctx, entry.Path)
			if err != nil {
				return nil, err
			}
			queue = append(queue, children...)
		default:
			out = append(out, entry)
		}
	}
	return out, nil
}

func cleanRoots(roots []string) []string {
	out := make([]string, 0, len(roots))
	for _, root := range roots {
		if strings.TrimSpace(root) != "" {
			out = append(out, filepath.Clean(root))
		}
	}
	sort.Strings(out)
	return out
}

func intersectRoots(configured []string, scoped []string) []string {
	var out []string
	for _, root := range configured {
		for _, scope := range scoped {
			if pathContains(root, scope) {
				// A rate-triggered signal names a governed child. Restrict the
				// provider to that child instead of scanning/deleting siblings.
				out = append(out, scope)
				break
			}
			if pathContains(scope, root) {
				out = append(out, root)
				break
			}
		}
	}
	return out
}

func pathContains(ancestor, path string) bool {
	ancestor = filepath.Clean(ancestor)
	path = filepath.Clean(path)
	if ancestor == path {
		return true
	}
	if ancestor == string(filepath.Separator) {
		return filepath.IsAbs(path)
	}
	return strings.HasPrefix(path, ancestor+string(filepath.Separator))
}

func activePath(path string) bool {
	name := filepath.Base(path)
	if strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".lock") || strings.HasSuffix(name, ".sock") || strings.HasPrefix(name, "systemd-private-") || strings.Contains(path, string(filepath.Separator)+"proc"+string(filepath.Separator)) {
		return true
	}
	// This directory contains live agent session scratchpads, including the
	// session performing cleanup. It is a hard safety boundary, not a
	// retention decision, so keep the exact path protected across previews and
	// applies even when its contents are old.
	return filepath.Clean(path) == filepath.Join(os.TempDir(), "claude-1000")
}

func activeLeasePath(path string) bool {
	name := strings.ToLower(filepath.Base(path))
	return strings.HasSuffix(name, ".active") || strings.HasSuffix(name, ".lease") ||
		strings.HasSuffix(name, ".running") || strings.HasSuffix(name, ".in-progress") ||
		name == "current" || name == "active"
}

func isFileMissing(err error) bool {
	if errors.Is(err, fs.ErrNotExist) {
		return true
	}
	errText := strings.ToLower(err.Error())
	return strings.Contains(errText, "not found") || strings.Contains(errText, "does not exist")
}

func isPermissionError(err error) bool {
	if errors.Is(err, os.ErrPermission) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "permission denied") || strings.Contains(message, "operation not permitted") || strings.Contains(message, "eacces") || strings.Contains(message, "eperm")
}

func stableItemID(providerID, path string) string {
	id := strings.NewReplacer(string(filepath.Separator), "-", " ", "-", ".", "-").Replace(strings.Trim(filepath.Clean(path), string(filepath.Separator)))
	return providerID + ":" + id
}

func sumPreviewBytes(items []cleanup.PreviewItem) int64 {
	var total int64
	for _, item := range items {
		total += item.Bytes
	}
	return total
}

func previewItemIDs(items []cleanup.PreviewItem) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.ID)
	}
	return out
}
