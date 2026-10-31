package cleanup

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/storage"

	"github.com/vrooli/api-core/retention"
	corestorage "github.com/vrooli/api-core/storage"
	"github.com/vrooli/vrooli/packages/scheduler"
)

// OutputBudgetName is the storage entry in .vrooli/service.json that declares
// the job-output age and byte bounds. The manifest is the single source of
// truth; environment variables only override it for an operator.
const OutputBudgetName = "job_outputs"

// Protection reasons reported for reclaimable outputs the owner refused to
// delete. They make a cycle that could not reach its budget explain itself.
const (
	ProtectedPinned         = "pinned"
	ProtectedActiveInput    = "active_job_input"
	ProtectedRecentGrace    = "recent_grace"
	ProtectedUnindexed      = "unindexed"
	defaultOutputGrace      = time.Hour
	defaultOutputMaxDeletes = 5000
	defaultOutputJobLimit   = 200000
	outputCycleReceiptName  = "job-outputs-cycle.json"
)

// deleteMu serializes every deletion of managed outputs, whether driven by the
// periodic owner loop or by the storage-manager owner contract, so a reviewed
// apply never races an enforcement cycle over the same keys.
var deleteMu sync.Mutex

type outputIndex interface {
	ListManagedOutputs(context.Context, int) ([]internaljobs.Job, error)
	ListActive(context.Context) ([]internaljobs.Job, error)
}

// OutputRetention enforces the job_outputs budget. Eligibility comes from the
// jobs table, not file mtimes: placement migrations rewrite mtimes, while the
// job's finish time is when the result was produced. A key shared by several
// jobs (content-addressed results) takes the age of its newest reference.
type OutputRetention struct {
	Jobs  outputIndex
	Store *storage.Store
	Now   func() time.Time
	// Grace is the minimum age before byte pressure may delete an output that
	// is still inside the age horizon, so a caller always has time to fetch a
	// result it was just handed.
	Grace time.Duration
	// MaxDeletes bounds one cycle's deletions; a larger backlog completes over
	// later cycles and the cycle reports itself incomplete.
	MaxDeletes int
	// JobLimit bounds how many job rows one cycle reads.
	JobLimit int
}

// OutputResult reports one enforcement cycle.
type OutputResult struct {
	BeforeBytes int64          `json:"before_bytes"`
	AfterBytes  int64          `json:"after_bytes"`
	BeforeItems int            `json:"before_items"`
	Deleted     int            `json:"deleted"`
	FreedBytes  int64          `json:"freed_bytes"`
	Incomplete  bool           `json:"incomplete"`
	Protected   map[string]int `json:"protected"`
}

// ProtectedTotal is the number of reclaimable outputs the owner refused.
func (r OutputResult) ProtectedTotal() int {
	total := 0
	for _, n := range r.Protected {
		total += n
	}
	return total
}

// ProtectedSummary renders Protected deterministically, e.g. "pinned=2,unindexed=1".
func (r OutputResult) ProtectedSummary() string {
	reasons := make([]string, 0, len(r.Protected))
	for reason, n := range r.Protected {
		if n > 0 {
			reasons = append(reasons, reason)
		}
	}
	sort.Strings(reasons)
	parts := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		parts = append(parts, fmt.Sprintf("%s=%d", reason, r.Protected[reason]))
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ",")
}

// OverBudget reports whether the cycle left outputs above the byte ceiling.
func (r OutputResult) OverBudget(b retention.Budget) bool {
	return b.HasByteBound() && r.AfterBytes > b.MaxBytes
}

type outputKey struct {
	key      string
	newest   time.Time
	pinned   bool
	bytes    int64
	present  bool
	consumed bool
}

// Prune runs one enforcement cycle against budget.
func (o OutputRetention) Prune(ctx context.Context, budget retention.Budget) (OutputResult, error) {
	deleteMu.Lock()
	defer deleteMu.Unlock()

	res := OutputResult{Protected: map[string]int{}}
	if o.Jobs == nil || o.Store == nil {
		return res, errors.New("output retention: jobs index and blob store are required")
	}
	now := time.Now
	if o.Now != nil {
		now = o.Now
	}
	grace := o.Grace
	if grace <= 0 {
		grace = defaultOutputGrace
	}
	maxDeletes := o.MaxDeletes
	if maxDeletes <= 0 {
		maxDeletes = defaultOutputMaxDeletes
	}
	jobLimit := o.JobLimit
	if jobLimit <= 0 {
		jobLimit = defaultOutputJobLimit
	}

	probe, err := o.Store.PathForKeyContext(ctx, "out/.retention-probe")
	if err != nil {
		return res, fmt.Errorf("output retention: resolve output root: %w", err)
	}
	root := filepath.Dir(probe)
	files, err := measureOutputs(root)
	if err != nil {
		return res, err
	}
	for _, f := range files {
		res.BeforeBytes += f.bytes
	}
	res.BeforeItems = len(files)
	used := res.BeforeBytes

	jobs, err := o.Jobs.ListManagedOutputs(ctx, jobLimit)
	if err != nil {
		return res, fmt.Errorf("output retention: list managed outputs: %w", err)
	}
	if len(jobs) >= jobLimit {
		res.Incomplete = true
	}
	active, err := o.Jobs.ListActive(ctx)
	if err != nil {
		return res, fmt.Errorf("output retention: list active jobs: %w", err)
	}

	byKey := map[string]*outputKey{}
	order := make([]*outputKey, 0, len(jobs))
	for _, job := range jobs {
		key := strings.TrimSpace(job.ResultRef)
		if !strings.HasPrefix(key, "out/") {
			continue
		}
		entry, ok := byKey[key]
		if !ok {
			entry = &outputKey{key: key}
			byKey[key] = entry
			order = append(order, entry)
		}
		if at := finishedAtOrCreated(job); at.After(entry.newest) {
			entry.newest = at
		}
		entry.pinned = entry.pinned || jobPinned(job)
	}
	for _, entry := range order {
		rel := strings.TrimPrefix(entry.key, "out/")
		if f, ok := files[rel]; ok {
			entry.present, entry.bytes = true, f.bytes
			f.indexed = true
		}
		for _, job := range active {
			if bytes.Contains(job.Payload, []byte(entry.key)) {
				entry.consumed = true
				break
			}
		}
	}
	// Oldest effective age first, so age expiry and byte pressure both remove
	// the least recently produced results.
	sort.SliceStable(order, func(i, j int) bool { return order[i].newest.Before(order[j].newest) })

	at := now()
	for _, entry := range order {
		if !entry.present {
			continue
		}
		age := at.Sub(entry.newest)
		expired := budget.HasAgeBound() && age >= budget.MaxAge
		pressure := budget.HasByteBound() && used > budget.MaxBytes
		if !expired && !pressure {
			// Later entries are younger, and nothing is over the ceiling.
			break
		}
		switch {
		case entry.pinned:
			res.Protected[ProtectedPinned]++
			continue
		case entry.consumed:
			res.Protected[ProtectedActiveInput]++
			continue
		case !expired && age < grace:
			res.Protected[ProtectedRecentGrace]++
			continue
		}
		if res.Deleted >= maxDeletes {
			res.Incomplete = true
			break
		}
		if err := ctx.Err(); err != nil {
			res.Incomplete = true
			break
		}
		if err := o.Store.Delete(ctx, entry.key); err != nil {
			res.AfterBytes = used
			return res, fmt.Errorf("output retention: %w", err)
		}
		used -= entry.bytes
		res.FreedBytes += entry.bytes
		res.Deleted++
	}

	// A file no successful job indexes may belong to a job still writing its
	// result, or to history the jobs table no longer describes. The owner
	// cannot prove either way, so it keeps it and reports the expired ones.
	for _, f := range files {
		if f.indexed {
			continue
		}
		if budget.HasAgeBound() && at.Sub(f.modTime) >= budget.MaxAge {
			res.Protected[ProtectedUnindexed]++
		}
	}
	res.AfterBytes = used
	return res, nil
}

type outputFile struct {
	bytes   int64
	modTime time.Time
	indexed bool
}

// measureOutputs maps each blob (relative to the out/ root, without its .meta
// sidecar) to its size including the sidecar.
func measureOutputs(root string) (map[string]*outputFile, error) {
	files := map[string]*outputFile{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if errors.Is(walkErr, fs.ErrNotExist) {
				return nil
			}
			return walkErr
		}
		if d.IsDir() || !d.Type().IsRegular() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		name := strings.TrimSuffix(rel, ".meta")
		f, ok := files[name]
		if !ok {
			f = &outputFile{}
			files[name] = f
		}
		f.bytes += info.Size()
		if rel == name || f.modTime.IsZero() {
			f.modTime = info.ModTime()
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("output retention: measure %s: %w", root, err)
	}
	return files, nil
}

// OutputBudget loads the job_outputs budget from the owner manifest and applies
// operator overrides. Both an age and a byte bound are required: a budget
// missing either makes no promise worth enforcing.
func OutputBudget(manifestPath, maxAge, maxBytes string) (retention.Budget, error) {
	specs, err := retention.LoadSpecs(retention.ScenarioConfig{ManifestPath: manifestPath})
	if err != nil {
		return retention.Budget{}, err
	}
	for _, spec := range specs {
		if spec.Budget.Name != OutputBudgetName {
			continue
		}
		budget, err := retention.ConfigureBudget(spec.Budget, strings.TrimSpace(maxAge), strings.TrimSpace(maxBytes))
		if err != nil {
			return retention.Budget{}, err
		}
		if !budget.HasAgeBound() || !budget.HasByteBound() {
			return retention.Budget{}, fmt.Errorf("%s requires age and byte bounds", OutputBudgetName)
		}
		return budget, nil
	}
	return retention.Budget{}, fmt.Errorf("%s retention declaration is required", OutputBudgetName)
}

// OutputReceiptError makes the durable receipt honest: an incomplete cycle or
// one that leaves outputs over their byte ceiling has not enforced the budget,
// and the error names what the owner protected.
func OutputReceiptError(result OutputResult, budget retention.Budget, cycleErr error) error {
	if cycleErr != nil {
		return cycleErr
	}
	if result.OverBudget(budget) {
		return fmt.Errorf("job outputs over budget after cycle: used %d of %d bytes, deleted %d, protected %d (%s)",
			result.AfterBytes, budget.MaxBytes, result.Deleted, result.ProtectedTotal(), result.ProtectedSummary())
	}
	if result.Incomplete {
		return fmt.Errorf("job output retention incomplete: deleted %d this cycle, backlog remains (protected %s)",
			result.Deleted, result.ProtectedSummary())
	}
	return nil
}

// outputCycleReceipt is the owner-detail receipt written beside the shared
// enforcement receipt: the shared one carries only time and error, this one
// carries the counts an operator needs to see why bytes were kept.
type outputCycleReceipt struct {
	Owner         string       `json:"owner"`
	Entry         string       `json:"entry"`
	CycleTime     time.Time    `json:"cycle_time"`
	MaxAgeSeconds int64        `json:"max_age_seconds"`
	MaxBytes      int64        `json:"max_bytes"`
	Result        OutputResult `json:"result"`
	Error         string       `json:"error,omitempty"`
}

// RecordOutputReceipt writes the shared enforcement receipt plus the owner
// detail receipt for namespace.
func RecordOutputReceipt(namespace string, at time.Time, budget retention.Budget, result OutputResult, cycleErr error) error {
	receiptErr := OutputReceiptError(result, budget, cycleErr)
	if err := retention.RecordEnforcementReceipt(namespace, at, receiptErr); err != nil {
		return err
	}
	path, err := retention.EnforcementReceiptPath(namespace)
	if err != nil {
		return err
	}
	detail := outputCycleReceipt{
		Owner: namespace, Entry: OutputBudgetName, CycleTime: at,
		MaxAgeSeconds: int64(budget.MaxAge / time.Second), MaxBytes: budget.MaxBytes, Result: result,
	}
	if receiptErr != nil {
		detail.Error = receiptErr.Error()
	}
	data, err := json.MarshalIndent(detail, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".job-outputs-cycle-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, filepath.Join(dir, outputCycleReceiptName))
}

// RetentionConfig wires the periodic owner loop.
type RetentionConfig struct {
	ManifestPath string
	Jobs         outputIndex
	Store        *storage.Store
	Logger       *slog.Logger
	// Interval between cycles (IMAGE_TOOLS_OUTPUT_RETENTION_INTERVAL).
	Interval time.Duration
	// StartupDelay runs the first cycle shortly after boot rather than a full
	// interval later; zero disables the early cycle.
	StartupDelay time.Duration
}

// StartOutputRetention starts the owner retention loop for managed job
// outputs. Storage-manager treats the entry's "custom" pruner as owner-enforced
// and reads the receipt this loop writes.
func StartOutputRetention(ctx context.Context, cfg RetentionConfig) error {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	budget, err := OutputBudget(cfg.ManifestPath,
		os.Getenv("IMAGE_TOOLS_OUTPUT_RETENTION_MAX_AGE"),
		os.Getenv("IMAGE_TOOLS_OUTPUT_RETENTION_MAX_BYTES"))
	if err != nil {
		return err
	}
	interval := cfg.Interval
	if raw := strings.TrimSpace(os.Getenv("IMAGE_TOOLS_OUTPUT_RETENTION_INTERVAL")); raw != "" {
		interval, err = time.ParseDuration(raw)
		if err != nil || interval <= 0 {
			return fmt.Errorf("invalid output retention interval %q", raw)
		}
	}
	if interval <= 0 {
		interval = 30 * time.Minute
	}
	namespace, err := corestorage.ScenarioNamespace("image-tools")
	if err != nil {
		return err
	}
	owner := OutputRetention{Jobs: cfg.Jobs, Store: cfg.Store}
	cycle := func(ctx context.Context) error {
		cycleCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()
		result, err := owner.Prune(cycleCtx, budget)
		logger.Info("image-tools output retention cycle", "used_bytes", result.AfterBytes,
			"max_bytes", budget.MaxBytes, "max_age", budget.MaxAge, "freed_bytes", result.FreedBytes,
			"deleted", result.Deleted, "incomplete", result.Incomplete, "protected", result.ProtectedTotal(),
			"protected_reasons", result.ProtectedSummary(), "error", err)
		if receiptErr := RecordOutputReceipt(namespace, time.Now().UTC(), budget, result, err); receiptErr != nil {
			return receiptErr
		}
		return err
	}
	runner := scheduler.New(interval, cycle)
	runner.Start(ctx)
	if cfg.StartupDelay > 0 {
		go func() {
			timer := time.NewTimer(cfg.StartupDelay)
			defer timer.Stop()
			select {
			case <-ctx.Done():
			case <-timer.C:
				_ = runner.RunOnce(ctx)
			}
		}()
	}
	logger.Info("image-tools output retention configured", "max_bytes", budget.MaxBytes,
		"max_age", budget.MaxAge, "interval", interval, "namespace", namespace)
	return nil
}
