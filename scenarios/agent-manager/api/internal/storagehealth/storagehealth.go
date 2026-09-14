// Package storagehealth keeps Agent Manager's SQLite file inside its declared
// storage budget. It measures the file against that budget, returns freed pages
// to the filesystem in bounded incremental-vacuum batches, and runs the fenced
// one-time compaction that rewrites a database into incremental auto-vacuum
// mode.
//
// Only pages SQLite already holds on its freelist are returned; no row is ever
// deleted here. Deciding what history to keep belongs to the retention owners.
// With auto_vacuum=NONE a deleted row's pages stay in the file forever, which is
// how a database with 8.7 GB of live rows grew to a 293 GB file (2026-09-14).
package storagehealth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/retention"
)

// Level places the file's occupied bytes against the declared budget.
type Level string

const (
	LevelOK         Level = "ok"
	LevelReclaim    Level = "reclaim"
	LevelAlarm      Level = "alarm"
	LevelUnbudgeted Level = "unbudgeted"
)

// Action is what the owner should do next about the file's footprint.
type Action string

const (
	ActionNone Action = "none"
	// ActionIncrementalVacuum: freed pages can be returned online in bounded
	// batches without an outage.
	ActionIncrementalVacuum Action = "incremental_vacuum"
	// ActionCompactionRequired: freed pages exist but the file is not in
	// incremental mode, so only the fenced full compaction can return them.
	ActionCompactionRequired Action = "compaction_required"
	// ActionRetentionOrBudget: live rows alone approach the budget. Reclaiming
	// free pages cannot help; retention or a reviewed budget change must.
	ActionRetentionOrBudget Action = "retention_or_budget"
)

// Budget is the declared ceiling the thresholds are measured against.
type Budget struct {
	Bytes  int64  `json:"bytes"`
	Source string `json:"source,omitempty"`
}

// Policy holds the thresholds and batch bounds. Ratios apply to the budget.
type Policy struct {
	// ReclaimRatio is where the owner starts returning free pages even below
	// the free-page trigger; AlarmRatio is where it escalates.
	ReclaimRatio float64 `json:"reclaimRatio"`
	AlarmRatio   float64 `json:"alarmRatio"`
	// Free pages above max(FreeTriggerBytes, FreeTriggerFraction*live) are
	// returned; FreeFloorBytes stays on the freelist as reuse slack.
	FreeTriggerBytes    int64   `json:"freeTriggerBytes"`
	FreeTriggerFraction float64 `json:"freeTriggerFraction"`
	FreeFloorBytes      int64   `json:"freeFloorBytes"`
	// Each incremental_vacuum transaction holds the writer while it relocates
	// live pages from the end of the file. The batch adapts between
	// MinBatchPages and BatchPages so one hold stays near BatchHoldTarget: a
	// fixed 2048-page batch held it ~4 s on the 10 GB rehearsal copy
	// (2026-09-14). TickBudget bounds one maintenance pass and ReclaimBudget
	// one reclaim request.
	BatchPages      int64         `json:"batchPages"`
	MinBatchPages   int64         `json:"minBatchPages"`
	BatchHoldTarget time.Duration `json:"batchHoldTarget"`
	TickBudget      time.Duration `json:"tickBudget"`
	ReclaimBudget   time.Duration `json:"reclaimBudget"`
	BatchPause      time.Duration `json:"batchPause"`
	// MaintainEvery spaces idle measurements; a pass that found work runs again
	// on the next reconcile cycle.
	MaintainEvery time.Duration `json:"maintainEvery"`
	AlarmLogEvery time.Duration `json:"alarmLogEvery"`
	// SpaceMarginBytes is kept free beyond what compaction is estimated to use.
	SpaceMarginBytes int64 `json:"spaceMarginBytes"`
}

func DefaultPolicy() Policy {
	return Policy{
		ReclaimRatio: 0.70, AlarmRatio: 0.85,
		FreeTriggerBytes: 64 << 20, FreeTriggerFraction: 0.05, FreeFloorBytes: 16 << 20,
		BatchPages: 2048, MinBatchPages: 16, BatchHoldTarget: 250 * time.Millisecond,
		TickBudget: 5 * time.Second, ReclaimBudget: 30 * time.Second, BatchPause: 20 * time.Millisecond,
		MaintainEvery: 5 * time.Minute, AlarmLogEvery: time.Hour, SpaceMarginBytes: 1 << 30,
	}
}

// Stats is one cheap measurement: pragmas and file sizes, no page scan.
type Stats struct {
	Path          string    `json:"path"`
	PageSize      int64     `json:"pageSize"`
	PageCount     int64     `json:"pageCount"`
	FreelistCount int64     `json:"freelistCount"`
	AutoVacuum    string    `json:"autoVacuum"`
	JournalMode   string    `json:"journalMode"`
	FileBytes     int64     `json:"fileBytes"`
	WALBytes      int64     `json:"walBytes"`
	LiveBytes     int64     `json:"liveBytes"`
	FreeBytes     int64     `json:"freeBytes"`
	OccupiedBytes int64     `json:"occupiedBytes"`
	BudgetBytes   int64     `json:"budgetBytes"`
	BudgetSource  string    `json:"budgetSource,omitempty"`
	UsageRatio    float64   `json:"usageRatio"`
	Level         Level     `json:"level"`
	Action        Action    `json:"action"`
	ObservedAt    time.Time `json:"observedAt"`
}

// FenceState is the owner-maintenance admission fence as the compaction sees it.
type FenceState struct {
	Closed   bool
	Drained  bool
	Revision int64
}

// FenceObserver reads the live fence; Pauser stops Agent Manager's own
// background writers and returns the function that restarts them.
type (
	FenceObserver func(context.Context) (FenceState, error)
	Pauser        func(context.Context) (resume func(), err error)
)

type Options struct {
	// DB returns the production pool. Compaction and vacuum never run against a
	// routed test pool.
	DB     func() *sql.DB
	Path   string
	Budget func() Budget
	Policy Policy
	Fence  FenceObserver
	Pause  Pauser
	// Watermarks are in-memory positions over run_events.rowid that a VACUUM
	// could invalidate; see rowids.go.
	Watermarks []NamedWatermark
	FreeSpace  func(dir string) (uint64, error)
	SameDevice func(a, b string) bool
	TempDir    func() string
	Clock      func() time.Time
	Logger     *slog.Logger
}

var (
	ErrFenceNotDrained   = errors.New("storage compaction requires the maintenance fence to be closed and drained")
	ErrBusy              = errors.New("storage maintenance is already running")
	ErrInsufficientSpace = errors.New("insufficient free disk space for storage compaction")
)

type CompactionState string

const (
	CompactionIdle      CompactionState = "idle"
	CompactionRunning   CompactionState = "running"
	CompactionSucceeded CompactionState = "succeeded"
	CompactionFailed    CompactionState = "failed"
)

// CompactionReceipt is the evidence one compaction leaves behind.
type CompactionReceipt struct {
	Actor            string `json:"actor"`
	Reason           string `json:"reason"`
	FenceRevision    int64  `json:"fenceRevision"`
	Before           Stats  `json:"before"`
	After            Stats  `json:"after"`
	AutoVacuumBefore string `json:"autoVacuumBefore"`
	AutoVacuumAfter  string `json:"autoVacuumAfter"`
	ReclaimedBytes   int64  `json:"reclaimedBytes"`
	QuickCheck       string `json:"quickCheck"`
	CheckpointBusy   int64  `json:"checkpointBusy"`
	CheckpointLog    int64  `json:"checkpointLogFrames"`
	CheckpointDone   int64  `json:"checkpointedFrames"`
	TempDir          string `json:"tempDir"`
	// RemappedWatermarks are the run_events rowid positions carried across
	// the VACUUM; SearchBefore/SearchAfter prove the lexical index still
	// matches its catalog.
	RemappedWatermarks []RemappedWatermark `json:"remappedWatermarks"`
	SearchBefore       SearchCheck         `json:"searchBefore"`
	SearchAfter        SearchCheck         `json:"searchAfter"`
	VacuumMillis       int64               `json:"vacuumMillis"`
	CheckMillis        int64               `json:"checkMillis"`
	TotalMillis        int64               `json:"totalMillis"`
}

// CompactionStatus is the asynchronous compaction's latest outcome.
type CompactionStatus struct {
	State      CompactionState    `json:"state"`
	Actor      string             `json:"actor,omitempty"`
	Reason     string             `json:"reason,omitempty"`
	StartedAt  *time.Time         `json:"startedAt,omitempty"`
	FinishedAt *time.Time         `json:"finishedAt,omitempty"`
	Receipt    *CompactionReceipt `json:"receipt,omitempty"`
	Error      string             `json:"error,omitempty"`
}

// Performed names what one reclaim request actually did.
const (
	PerformedPreview           = "preview"
	PerformedNone              = "none"
	PerformedIncrementalVacuum = "incremental_vacuum"
)

// ReclaimReceipt answers the storage-manager reclaim contract. Byte fields
// count the database file plus its WAL. ProjectedBytesAfter is where the file
// lands once every free page is returned: online for an incremental-mode file,
// or only after the fenced compaction when CompactionRequired is true.
type ReclaimReceipt struct {
	DryRun              bool   `json:"dryRun"`
	Action              Action `json:"action"`
	Performed           string `json:"performed"`
	CompactionRequired  bool   `json:"compactionRequired"`
	BytesBefore         int64  `json:"bytesBefore"`
	BytesAfter          int64  `json:"bytesAfter"`
	ProjectedBytesAfter int64  `json:"projectedBytesAfter"`
	ReclaimedBytes      int64  `json:"reclaimedBytes"`
	Complete            bool   `json:"complete"`
	DurationMillis      int64  `json:"durationMillis"`
	Guidance            string `json:"guidance,omitempty"`
	Before              Stats  `json:"before"`
	After               *Stats `json:"after,omitempty"`
}

type Service struct {
	opts Options
	// run serializes every mutation of the file's footprint: a maintenance
	// pass, an explicit reclaim, and a compaction never overlap.
	run sync.Mutex

	// afterVacuum runs between VACUUM and the rowid remap; tests use it to
	// renumber rowids the way SQLite documents VACUUM may.
	afterVacuum func(context.Context, *sql.Conn) error

	mu           sync.Mutex
	lastMaintain time.Time
	lastAction   Action
	lastAlarmLog time.Time
	compaction   CompactionStatus
}

func New(opts Options) (*Service, error) {
	if opts.DB == nil || strings.TrimSpace(opts.Path) == "" {
		return nil, errors.New("storage health requires the production pool and database path")
	}
	if opts.Policy == (Policy{}) {
		opts.Policy = DefaultPolicy()
	}
	if opts.Budget == nil {
		opts.Budget = func() Budget { return Budget{} }
	}
	if opts.FreeSpace == nil {
		opts.FreeSpace = freeBytes
	}
	if opts.SameDevice == nil {
		opts.SameDevice = sameDevice
	}
	if opts.TempDir == nil {
		opts.TempDir = SQLiteTempDir
	}
	if opts.Clock == nil {
		opts.Clock = time.Now
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	return &Service{opts: opts, compaction: CompactionStatus{State: CompactionIdle}}, nil
}

func (s *Service) Policy() Policy { return s.opts.Policy }

// Stats measures the file with pragmas and stat calls only.
func (s *Service) Stats(ctx context.Context) (Stats, error) {
	db := s.opts.DB()
	if db == nil {
		return Stats{}, errors.New("database pool is closed")
	}
	st := Stats{Path: s.opts.Path, ObservedAt: s.opts.Clock().UTC()}
	var autoVacuum int64
	for _, read := range []struct {
		pragma string
		into   *int64
	}{{"page_size", &st.PageSize}, {"page_count", &st.PageCount}, {"freelist_count", &st.FreelistCount}, {"auto_vacuum", &autoVacuum}} {
		if err := db.QueryRowContext(ctx, "PRAGMA "+read.pragma).Scan(read.into); err != nil {
			return Stats{}, fmt.Errorf("read %s: %w", read.pragma, err)
		}
	}
	if err := db.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&st.JournalMode); err != nil {
		return Stats{}, fmt.Errorf("read journal_mode: %w", err)
	}
	st.AutoVacuum = autoVacuumName(autoVacuum)
	st.FileBytes = fileSize(s.opts.Path)
	st.WALBytes = fileSize(s.opts.Path + "-wal")
	st.LiveBytes = (st.PageCount - st.FreelistCount) * st.PageSize
	st.FreeBytes = st.FreelistCount * st.PageSize
	budget := s.opts.Budget()
	st.BudgetBytes, st.BudgetSource = budget.Bytes, budget.Source
	assess(&st, s.opts.Policy)
	return st, nil
}

// assess sets occupancy, level, and action from measured fields.
func assess(st *Stats, p Policy) {
	st.OccupiedBytes = st.FileBytes + st.WALBytes
	st.Level = LevelUnbudgeted
	if st.BudgetBytes > 0 {
		st.UsageRatio = float64(st.OccupiedBytes) / float64(st.BudgetBytes)
		switch {
		case st.UsageRatio >= p.AlarmRatio:
			st.Level = LevelAlarm
		case st.UsageRatio >= p.ReclaimRatio:
			st.Level = LevelReclaim
		default:
			st.Level = LevelOK
		}
	}
	trigger := max(p.FreeTriggerBytes, int64(p.FreeTriggerFraction*float64(st.LiveBytes)))
	pressured := st.Level == LevelReclaim || st.Level == LevelAlarm
	switch {
	case st.FreeBytes > trigger || (pressured && st.FreeBytes > p.FreeFloorBytes):
		if st.AutoVacuum == "incremental" {
			st.Action = ActionIncrementalVacuum
		} else {
			st.Action = ActionCompactionRequired
		}
	case st.BudgetBytes > 0 && float64(st.LiveBytes+st.WALBytes) >= p.AlarmRatio*float64(st.BudgetBytes):
		st.Action = ActionRetentionOrBudget
	default:
		st.Action = ActionNone
	}
}

// Maintain is the reconciler's bounded pass. It measures at most every
// MaintainEvery while idle, returns free pages down to the floor within
// TickBudget when the file is in incremental mode, and escalates in the log
// when only a compaction, retention, or a budget decision can help.
func (s *Service) Maintain(ctx context.Context) error {
	if !s.run.TryLock() {
		return nil
	}
	defer s.run.Unlock()
	now := s.opts.Clock()
	s.mu.Lock()
	idle := s.lastAction == ActionNone || s.lastAction == ""
	draining := s.lastAction == ActionIncrementalVacuum
	due := s.lastMaintain.IsZero() || !idle || now.Sub(s.lastMaintain) >= s.opts.Policy.MaintainEvery
	s.mu.Unlock()
	if !due {
		return nil
	}
	st, err := s.Stats(ctx)
	if err != nil {
		return err
	}
	// The trigger starts a drain and the floor ends it: a drain cut short by
	// TickBudget continues on later passes even once the remainder is below
	// the trigger, rather than stranding up to the trigger's worth of pages.
	work := st.Action == ActionIncrementalVacuum ||
		(draining && st.AutoVacuum == "incremental" && st.FreeBytes > s.opts.Policy.FreeFloorBytes)
	next := st.Action
	if work {
		next = ActionIncrementalVacuum
	}
	s.mu.Lock()
	s.lastMaintain, s.lastAction = now, next
	s.mu.Unlock()
	s.escalate(st)
	if !work {
		return nil
	}
	_, err = s.vacuumTo(ctx, s.opts.Policy.FreeFloorBytes, s.opts.Policy.TickBudget)
	return err
}

func (s *Service) escalate(st Stats) {
	if st.Level != LevelAlarm && st.Action != ActionCompactionRequired && st.Action != ActionRetentionOrBudget {
		return
	}
	now := s.opts.Clock()
	s.mu.Lock()
	if !s.lastAlarmLog.IsZero() && now.Sub(s.lastAlarmLog) < s.opts.Policy.AlarmLogEvery {
		s.mu.Unlock()
		return
	}
	s.lastAlarmLog = now
	s.mu.Unlock()
	s.opts.Logger.Warn("agent-manager database storage needs owner action",
		"level", st.Level, "action", st.Action, "occupied_bytes", st.OccupiedBytes, "live_bytes", st.LiveBytes,
		"free_bytes", st.FreeBytes, "budget_bytes", st.BudgetBytes, "auto_vacuum", st.AutoVacuum,
		"guidance", guidance(st.Action))
}

func guidance(action Action) string {
	switch action {
	case ActionIncrementalVacuum:
		return "free pages are returned online by the reconciler or POST /api/v1/storage/reclaim"
	case ActionCompactionRequired:
		return "the file is not in incremental auto-vacuum mode; run agent-manager maintenance begin/drain, then agent-manager storage compact --local-owner, then maintenance resume"
	case ActionRetentionOrBudget:
		return "live rows approach the declared budget; tighten retention or review the budget in .vrooli/service.json"
	default:
		return ""
	}
}

// vacuumTo returns free pages in BatchPages transactions until at most target
// bytes remain free or the time budget ends. It returns the pages returned.
func (s *Service) vacuumTo(ctx context.Context, target int64, budget time.Duration) (int64, error) {
	db := s.opts.DB()
	if db == nil {
		return 0, errors.New("database pool is closed")
	}
	deadline := s.opts.Clock().Add(budget)
	policy := s.opts.Policy
	minBatch := max(policy.MinBatchPages, 1)
	batch := max(min(policy.BatchPages, 256), minBatch)
	var returned int64
	for {
		var pageSize, free int64
		if err := db.QueryRowContext(ctx, "PRAGMA page_size").Scan(&pageSize); err != nil {
			return returned, err
		}
		if err := db.QueryRowContext(ctx, "PRAGMA freelist_count").Scan(&free); err != nil {
			return returned, err
		}
		excess := free - target/max(pageSize, 1)
		if excess <= 0 || !s.opts.Clock().Before(deadline) {
			return returned, nil
		}
		pages := min(excess, batch)
		held := s.opts.Clock()
		// incremental_vacuum emits one result row per page moved; a caller
		// that does not step every row returns a single page.
		rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA incremental_vacuum(%d)", pages))
		if err != nil {
			return returned, fmt.Errorf("incremental vacuum: %w", err)
		}
		for rows.Next() {
		}
		err = rows.Err()
		_ = rows.Close()
		if err != nil {
			return returned, fmt.Errorf("incremental vacuum: %w", err)
		}
		if target := policy.BatchHoldTarget; target > 0 {
			switch elapsed := s.opts.Clock().Sub(held); {
			case elapsed > target && batch > minBatch:
				batch = max(minBatch, batch/2)
			case elapsed < target/4 && batch < policy.BatchPages:
				batch = min(policy.BatchPages, batch*2)
			}
		}
		var after int64
		if err := db.QueryRowContext(ctx, "PRAGMA freelist_count").Scan(&after); err != nil {
			return returned, err
		}
		if after >= free {
			return returned, nil
		}
		returned += free - after
		timer := time.NewTimer(s.opts.Policy.BatchPause)
		select {
		case <-ctx.Done():
			timer.Stop()
			return returned, ctx.Err()
		case <-timer.C:
		}
	}
}

// Reclaim is the unauthenticated, bounded contract storage-manager calls as
// its over-budget backstop. It never rewrites the whole file: an incremental
// file returns free pages for at most ReclaimBudget and passively checkpoints
// the WAL; a none-mode file is only reported as needing the fenced compaction.
// Pages left over are returned by the reconciler's later passes.
func (s *Service) Reclaim(ctx context.Context, dryRun bool) (ReclaimReceipt, error) {
	start := s.opts.Clock()
	if !dryRun {
		if !s.run.TryLock() {
			return ReclaimReceipt{}, ErrBusy
		}
		defer s.run.Unlock()
	}
	before, err := s.Stats(ctx)
	if err != nil {
		return ReclaimReceipt{}, err
	}
	receipt := ReclaimReceipt{
		DryRun: dryRun, Action: before.Action, Performed: PerformedNone, Before: before, Guidance: guidance(before.Action),
		CompactionRequired: before.AutoVacuum != "incremental" && before.FreeBytes > 0,
		BytesBefore:        before.OccupiedBytes, BytesAfter: before.OccupiedBytes,
		ProjectedBytesAfter: before.OccupiedBytes - before.FreeBytes,
	}
	finish := func() (ReclaimReceipt, error) {
		receipt.DurationMillis = s.opts.Clock().Sub(start).Milliseconds()
		return receipt, nil
	}
	if dryRun {
		receipt.Performed = PerformedPreview
		receipt.Complete = before.FreeBytes == 0
		if !receipt.CompactionRequired && before.FreeBytes > 0 {
			receipt.Guidance = fmt.Sprintf("%d free bytes would be returned online in bounded incremental_vacuum batches", before.FreeBytes)
		}
		return finish()
	}
	if receipt.CompactionRequired || before.FreeBytes == 0 {
		receipt.Complete = before.FreeBytes == 0
		return finish()
	}
	receipt.Performed = PerformedIncrementalVacuum
	if _, err := s.vacuumTo(ctx, 0, s.opts.Policy.ReclaimBudget); err != nil {
		return receipt, err
	}
	// PASSIVE never waits on readers or blocks writers; journal_size_limit
	// then trims the WAL at its next reset.
	var busy, frames, done int64
	_ = s.opts.DB().QueryRowContext(ctx, "PRAGMA wal_checkpoint(PASSIVE)").Scan(&busy, &frames, &done)
	after, err := s.Stats(ctx)
	if err != nil {
		return receipt, err
	}
	receipt.After = &after
	receipt.BytesAfter = after.OccupiedBytes
	receipt.ProjectedBytesAfter = after.OccupiedBytes - after.FreeBytes
	receipt.ReclaimedBytes = before.OccupiedBytes - after.OccupiedBytes
	receipt.Complete = after.FreeBytes == 0
	if !receipt.Complete {
		receipt.Guidance = fmt.Sprintf("%d free bytes remain; the reconciler keeps returning them in bounded passes", after.FreeBytes)
	}
	return finish()
}

// CompactionStatus reports the latest asynchronous compaction.
func (s *Service) CompactionStatus() CompactionStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.compaction
}

// StartCompaction runs the fence, space, and busy checks synchronously so a
// refusal is immediate, then compacts in the background: a full VACUUM
// outlives any HTTP write timeout.
func (s *Service) StartCompaction(ctx context.Context, actor, reason string) (CompactionStatus, error) {
	if !s.run.TryLock() {
		return s.CompactionStatus(), ErrBusy
	}
	plan, err := s.preflight(ctx)
	if err != nil {
		s.run.Unlock()
		return s.CompactionStatus(), err
	}
	started := s.opts.Clock().UTC()
	s.mu.Lock()
	s.compaction = CompactionStatus{State: CompactionRunning, Actor: actor, Reason: reason, StartedAt: &started}
	status := s.compaction
	s.mu.Unlock()
	go func() {
		defer s.run.Unlock()
		receipt, err := s.compact(context.WithoutCancel(ctx), plan, actor, reason)
		s.finish(receipt, err)
	}()
	return status, nil
}

// Compact is the synchronous form used by rehearsals and tests.
func (s *Service) Compact(ctx context.Context, actor, reason string) (CompactionReceipt, error) {
	if !s.run.TryLock() {
		return CompactionReceipt{}, ErrBusy
	}
	defer s.run.Unlock()
	plan, err := s.preflight(ctx)
	if err != nil {
		return CompactionReceipt{}, err
	}
	started := s.opts.Clock().UTC()
	s.mu.Lock()
	s.compaction = CompactionStatus{State: CompactionRunning, Actor: actor, Reason: reason, StartedAt: &started}
	s.mu.Unlock()
	receipt, err := s.compact(ctx, plan, actor, reason)
	s.finish(receipt, err)
	return receipt, err
}

func (s *Service) finish(receipt CompactionReceipt, err error) {
	finished := s.opts.Clock().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.compaction.FinishedAt = &finished
	if err != nil {
		s.compaction.State, s.compaction.Error = CompactionFailed, err.Error()
		s.opts.Logger.Error("agent-manager storage compaction failed", "error", err.Error())
		return
	}
	s.compaction.State, s.compaction.Receipt = CompactionSucceeded, &receipt
	s.lastAction = ""
	s.opts.Logger.Info("agent-manager storage compaction completed", "reclaimed_bytes", receipt.ReclaimedBytes,
		"file_bytes", receipt.After.FileBytes, "auto_vacuum", receipt.AutoVacuumAfter, "total_ms", receipt.TotalMillis)
}

type compactionPlan struct {
	fence   FenceState
	before  Stats
	tempDir string
}

func (s *Service) preflight(ctx context.Context) (compactionPlan, error) {
	if s.opts.Fence == nil {
		return compactionPlan{}, fmt.Errorf("%w: no fence observer is wired", ErrFenceNotDrained)
	}
	fence, err := s.opts.Fence(ctx)
	if err != nil {
		return compactionPlan{}, fmt.Errorf("%w: %v", ErrFenceNotDrained, err)
	}
	if !fence.Closed || !fence.Drained {
		return compactionPlan{}, fmt.Errorf("%w (closed=%t drained=%t revision=%d)", ErrFenceNotDrained, fence.Closed, fence.Drained, fence.Revision)
	}
	if legacy, err := tableExists(ctx, s.opts.DB(), legacyProjectionTable); err != nil {
		return compactionPlan{}, err
	} else if legacy {
		return compactionPlan{}, ErrLegacyProjection
	}
	before, err := s.Stats(ctx)
	if err != nil {
		return compactionPlan{}, err
	}
	tempDir := s.opts.TempDir()
	if err := s.checkSpace(before, tempDir); err != nil {
		return compactionPlan{}, err
	}
	return compactionPlan{fence: fence, before: before, tempDir: tempDir}, nil
}

// checkSpace requires room for VACUUM's transient copy (in SQLite's temp
// directory) plus the rewritten pages passing through the WAL beside the
// database, each with a margin; one filesystem must hold both when shared.
func (s *Service) checkSpace(before Stats, tempDir string) error {
	margin := s.opts.Policy.SpaceMarginBytes
	dbDir := dirOf(s.opts.Path)
	needDB := before.LiveBytes + before.WALBytes + margin
	needTemp := before.LiveBytes + margin
	if s.opts.SameDevice(dbDir, tempDir) {
		needDB += needTemp
		needTemp = 0
	}
	for _, check := range []struct {
		dir  string
		need int64
	}{{dbDir, needDB}, {tempDir, needTemp}} {
		if check.need <= 0 {
			continue
		}
		free, err := s.opts.FreeSpace(check.dir)
		if err != nil {
			return fmt.Errorf("%w: measure %s: %v", ErrInsufficientSpace, check.dir, err)
		}
		if int64(free) < check.need {
			return fmt.Errorf("%w: %s has %d bytes free, compaction needs %d", ErrInsufficientSpace, check.dir, free, check.need)
		}
	}
	return nil
}

func (s *Service) compact(ctx context.Context, plan compactionPlan, actor, reason string) (CompactionReceipt, error) {
	start := s.opts.Clock()
	receipt := CompactionReceipt{Actor: actor, Reason: reason, FenceRevision: plan.fence.Revision, Before: plan.before, AutoVacuumBefore: plan.before.AutoVacuum, TempDir: plan.tempDir}
	if s.opts.Pause != nil {
		resume, err := s.opts.Pause(ctx)
		if err != nil {
			return receipt, fmt.Errorf("pause background writers: %w", err)
		}
		defer resume()
	}
	// Pausing can take a reconcile cycle; the fence must still hold after it.
	if s.opts.Fence != nil {
		fence, err := s.opts.Fence(ctx)
		if err != nil || !fence.Closed || !fence.Drained || fence.Revision != plan.fence.Revision {
			return receipt, fmt.Errorf("%w: fence changed while pausing background writers", ErrFenceNotDrained)
		}
	}
	db := s.opts.DB()
	if db == nil {
		return receipt, errors.New("database pool is closed")
	}
	// auto_vacuum, temp_store, and VACUUM are per-connection state, so every
	// statement below runs on one pinned connection.
	conn, err := db.Conn(ctx)
	if err != nil {
		return receipt, fmt.Errorf("pin compaction connection: %w", err)
	}
	defer conn.Close()
	var tempStore int64
	if err := conn.QueryRowContext(ctx, "PRAGMA temp_store").Scan(&tempStore); err != nil {
		return receipt, fmt.Errorf("read temp_store: %w", err)
	}
	// The pool sets temp_store=MEMORY; VACUUM's transient copy would then be a
	// whole-database copy in RAM.
	if _, err := conn.ExecContext(ctx, "PRAGMA temp_store=FILE"); err != nil {
		return receipt, fmt.Errorf("set temp_store: %w", err)
	}
	defer func() {
		_, _ = conn.ExecContext(context.WithoutCancel(ctx), fmt.Sprintf("PRAGMA temp_store=%d", tempStore))
	}()
	if legacy, err := tableExists(ctx, conn, legacyProjectionTable); err != nil {
		return receipt, err
	} else if legacy {
		return receipt, ErrLegacyProjection
	}
	if receipt.SearchBefore, err = searchConsistency(ctx, conn); err != nil {
		return receipt, err
	}
	if !receipt.SearchBefore.consistent() {
		return receipt, fmt.Errorf("%w before compaction (catalog=%d indexed=%d integrity=%s); repair it before compacting",
			ErrSearchInconsistent, receipt.SearchBefore.CatalogDocuments, receipt.SearchBefore.IndexedDocuments, receipt.SearchBefore.Integrity)
	}
	// Hold every in-memory rowid consumer until its position is rewritten.
	held := make([]heldWatermark, 0, len(s.opts.Watermarks))
	for _, watermark := range s.opts.Watermarks {
		current, set, release := watermark.Holder.HoldWatermark()
		defer release()
		held = append(held, heldWatermark{name: watermark.Name, current: current, set: set})
	}
	anchors, err := captureAnchors(ctx, conn, held)
	if err != nil {
		return receipt, fmt.Errorf("anchor run_events rowid positions: %w", err)
	}
	vacuumStart := s.opts.Clock()
	if plan.before.AutoVacuum != "incremental" {
		err = retention.EnsureIncrementalAutoVacuum(ctx, conn, s.opts.Logger)
	} else {
		_, err = conn.ExecContext(ctx, "VACUUM")
	}
	receipt.VacuumMillis = s.opts.Clock().Sub(vacuumStart).Milliseconds()
	if err != nil {
		return receipt, fmt.Errorf("vacuum: %w", err)
	}
	if s.afterVacuum != nil {
		if err := s.afterVacuum(ctx, conn); err != nil {
			return receipt, err
		}
	}
	if receipt.RemappedWatermarks, err = remapAnchors(ctx, conn, anchors); err != nil {
		// Consumers would read renumbered rowids from stale positions.
		return receipt, fmt.Errorf("REQUIRES OWNER ACTION: run_events rowid positions were not rewritten after VACUUM (%w); keep the fence closed and rebuild stats and supervision cursors", err)
	}
	if err := conn.QueryRowContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)").Scan(&receipt.CheckpointBusy, &receipt.CheckpointLog, &receipt.CheckpointDone); err != nil {
		return receipt, fmt.Errorf("checkpoint: %w", err)
	}
	checkStart := s.opts.Clock()
	receipt.QuickCheck, err = quickCheck(ctx, conn)
	receipt.CheckMillis = s.opts.Clock().Sub(checkStart).Milliseconds()
	if err != nil {
		return receipt, err
	}
	if receipt.SearchAfter, err = searchConsistency(ctx, conn); err != nil {
		return receipt, err
	}
	if !receipt.SearchAfter.consistent() || receipt.SearchAfter.CatalogDocuments != receipt.SearchBefore.CatalogDocuments {
		return receipt, fmt.Errorf("%w after compaction (catalog %d -> %d, indexed %d, integrity=%s)", ErrSearchInconsistent,
			receipt.SearchBefore.CatalogDocuments, receipt.SearchAfter.CatalogDocuments, receipt.SearchAfter.IndexedDocuments, receipt.SearchAfter.Integrity)
	}
	after, err := s.Stats(ctx)
	if err != nil {
		return receipt, err
	}
	receipt.After, receipt.AutoVacuumAfter = after, after.AutoVacuum
	receipt.ReclaimedBytes = plan.before.OccupiedBytes - after.OccupiedBytes
	receipt.TotalMillis = s.opts.Clock().Sub(start).Milliseconds()
	if after.AutoVacuum != "incremental" {
		return receipt, fmt.Errorf("auto_vacuum is %s after compaction, want incremental", after.AutoVacuum)
	}
	if receipt.QuickCheck != "ok" {
		return receipt, fmt.Errorf("quick_check after compaction: %s", receipt.QuickCheck)
	}
	return receipt, nil
}

func quickCheck(ctx context.Context, conn *sql.Conn) (string, error) {
	rows, err := conn.QueryContext(ctx, "PRAGMA quick_check(20)")
	if err != nil {
		return "", fmt.Errorf("quick_check: %w", err)
	}
	defer rows.Close()
	var lines []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return "", err
		}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return strings.Join(lines, "; "), nil
}

func autoVacuumName(mode int64) string {
	switch mode {
	case 1:
		return "full"
	case 2:
		return "incremental"
	default:
		return "none"
	}
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

// SQLiteTempDir mirrors SQLite's unix temporary-directory search, which is
// where VACUUM writes its transient copy once temp_store is FILE.
func SQLiteTempDir() string {
	for _, key := range []string{"SQLITE_TMPDIR", "TMPDIR"} {
		if dir := strings.TrimSpace(os.Getenv(key)); dir != "" && isDir(dir) {
			return dir
		}
	}
	for _, dir := range []string{"/var/tmp", "/usr/tmp", "/tmp"} {
		if isDir(dir) {
			return dir
		}
	}
	return "."
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func dirOf(path string) string {
	if i := strings.LastIndex(path, string(os.PathSeparator)); i > 0 {
		return path[:i]
	}
	return "."
}
