package providers

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"storage-manager/internal/cleanup"
)

type FileProviderConfig struct {
	ID          string
	Name        string
	Roots       []string
	Description string

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
	// Content-addressed caches (Go's build cache, Playwright's browser cache)
	// want the opposite: their entries are independent by construction and
	// per-file aging is exactly right, so they leave this false.
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
	ProtectActive     bool
	RepairClass       string
	OwnershipRepairer cleanup.OwnershipRepairer
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
	meta            cleanup.ProviderMetadata
	files           cleanup.FileSystem
	clock           cleanup.Clock
	roots           []string
	description     string
	action          string
	topLevelEntries bool
	measureBudget   time.Duration
	protectActive   bool
	repairClass     string
	repairer        cleanup.OwnershipRepairer

	// memo holds the most recent measurement so a single plan does not walk
	// the same trees twice. See previewMemo.
	memoMu sync.Mutex
	memo   *previewMemo
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
	return newFileProvider(files, clock, cfg, cleanup.SafetyTierSafeWithOwner, cleanup.ProviderModeDisabled, cleanup.ApprovalModeOwner, "trash-remove", trashPlatforms)
}

func NewTmpProvider(files cleanup.FileSystem, clock cleanup.Clock, cfg FileProviderConfig) *FileProvider {
	return newFileProvider(files, clock, cfg, cleanup.SafetyTierSafe, cleanup.ProviderModeDisabled, cleanup.ApprovalModeOperator, "tmp-remove", desktopPlatforms)
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
	return newFileProvider(files, clock, cfg, cleanup.SafetyTierConditional, cleanup.ProviderModeDisabled, cleanup.ApprovalModeOperator, "cache-remove", desktopPlatforms)
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
		},
		files:           files,
		clock:           clock,
		roots:           cleanRoots(cfg.Roots),
		description:     cfg.Description,
		action:          action,
		topLevelEntries: cfg.TopLevelEntries,
		measureBudget:   cfg.MeasureBudget,
		protectActive:   cfg.ProtectActive,
		repairClass:     cfg.RepairClass,
		repairer:        cfg.OwnershipRepairer,
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
	}, nil
}

func (p *FileProvider) Preview(ctx context.Context, req cleanup.PreviewRequest) (cleanup.Preview, error) {
	return p.preview(ctx, req.Scope, req.Policy)
}

func (p *FileProvider) Apply(ctx context.Context, req cleanup.ApplyRequest) (cleanup.ApplyResult, error) {
	if req.ProviderVersion != p.meta.Version {
		return cleanup.ApplyResult{}, fmt.Errorf("provider %s version mismatch: got %q want %q", p.meta.ID, req.ProviderVersion, p.meta.Version)
	}
	if req.IdempotencyKey == "" {
		return cleanup.ApplyResult{}, fmt.Errorf("provider %s apply requires idempotency key", p.meta.ID)
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
	for _, item := range req.Preview.Items {
		if err := ctx.Err(); err != nil {
			return cleanup.ApplyResult{}, err
		}
		if !p.withinConfiguredRoot(item.Path) {
			skipped = append(skipped, item.ID)
			continue
		}
		if _, err := p.files.Stat(ctx, item.Path); err != nil {
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
			continue
		}
		reclaimed += item.Bytes
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

// preview measures the configured roots, reusing a very recent identical
// measurement when one exists.
func (p *FileProvider) preview(ctx context.Context, scope cleanup.ObservationScope, policy cleanup.ProviderPolicy) (cleanup.Preview, error) {
	key := memoKey(scope, policy)

	p.memoMu.Lock()
	if p.memo != nil && p.memo.key == key && p.observedNow(scope.Now).Sub(p.memo.computed) < memoTTL {
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
	return fmt.Sprintf("%v|%v|%v|%v|%v|%v|%v",
		scope.Now.UnixNano(), scope.RootPaths, scope.CompleteCensus,
		policy.Enabled, policy.MinAge, policy.MaxBytes, policy.ApprovalMode)
}

func (p *FileProvider) measure(ctx context.Context, scope cleanup.ObservationScope, policy cleanup.ProviderPolicy) (cleanup.Preview, error) {
	out := cleanup.Preview{ProviderID: p.meta.ID, ProviderVersion: p.meta.Version}
	if !policy.Enabled {
		out.BlockedReason = "provider disabled by policy"
		return out, nil
	}
	if p.files == nil {
		out.BlockedReason = "filesystem seam unavailable"
		return out, nil
	}

	roots := p.roots
	if len(scope.RootPaths) > 0 {
		roots = intersectRoots(p.roots, cleanRoots(scope.RootPaths))
	}
	now := p.now(scope)
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
		if p.topLevelEntries {
			err = p.collectTopLevelEntries(ctx, root, now, deadline, policy, &out)
		} else {
			err = p.collectFiles(ctx, root, now, policy, &out)
		}
		if err != nil {
			return cleanup.Preview{}, err
		}
	}
	sort.Slice(out.Items, func(i, j int) bool { return out.Items[i].Path < out.Items[j].Path })
	return out, nil
}

// collectFiles adds one candidate per individual file beneath root.
func (p *FileProvider) collectFiles(ctx context.Context, root string, now time.Time, policy cleanup.ProviderPolicy, out *cleanup.Preview) error {
	return p.files.Walk(ctx, root, func(info cleanup.FileInfo) error {
		if info.Path == root || info.IsDir || !p.withinConfiguredRoot(info.Path) || p.isActivePath(info.Path) {
			return nil
		}
		if policy.MinAge > 0 && now.Sub(info.ModTime) < policy.MinAge {
			return nil
		}
		if policy.MaxBytes > 0 && sumPreviewBytes(out.Items)+info.Size > policy.MaxBytes {
			out.Warnings = append(out.Warnings, "max reclaim limit reached")
			return nil
		}
		out.Items = append(out.Items, p.previewItem(info.Path, info.Size))
		return nil
	})
}

// entryAggregate accumulates one top-level entry's subtree.
type entryAggregate struct {
	bytes  int64
	newest time.Time
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
	entries, err := p.files.ReadDir(ctx, root)
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

	var unmeasured int

	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !p.withinConfiguredRoot(entry.Path) || p.isActivePath(entry.Path) {
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
		if policy.MinAge > 0 && now.Sub(agg.newest) < policy.MinAge {
			continue
		}
		if policy.MaxBytes > 0 && sumPreviewBytes(out.Items)+agg.bytes > policy.MaxBytes {
			out.Warnings = append(out.Warnings, "max reclaim limit reached")
			continue
		}
		out.Items = append(out.Items, p.previewItem(entry.Path, agg.bytes))
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
			if root == scope || strings.HasPrefix(root, scope+string(filepath.Separator)) || strings.HasPrefix(scope, root+string(filepath.Separator)) {
				out = append(out, root)
				break
			}
		}
	}
	return out
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
