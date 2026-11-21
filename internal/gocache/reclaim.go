package gocache

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Schema identifiers for the documents this package emits.
const (
	PlanSchema    = "vrooli.go-cache.plan/v1"
	ReceiptSchema = "vrooli.go-cache.reclaim/v1"
	StatusSchema  = "vrooli.go-cache.status/v1"
)

const (
	// DefaultMinIdle is the recency floor. Live sessions are protected by
	// pins regardless of age; the floor additionally protects outputs a
	// finished command handed to another reader (go list -export paths read
	// by go/packages tools) and covers cmd/go's own one-hour mtime lag for
	// entries last touched natively.
	DefaultMinIdle = 2 * time.Hour
	bucketWidth    = 5 * time.Minute
	maxBucketAge   = 120 * 24 * time.Hour
	// planTTL lets successive bounded applies continue one plan's sweep
	// instead of rescanning a million entries per batch.
	planTTL = 15 * time.Minute
	// staleTempAge bounds how long an abandoned staged write may remain.
	staleTempAge = 24 * time.Hour
	shardCount   = 256
)

// Stop reasons a receipt reports.
const (
	StopComplete    = "complete"
	StopMaxBytes    = "max_bytes"
	StopDeadline    = "deadline"
	StopBlocked     = "blocked"
	StopBusy        = "busy"
	StopLockTimeout = "lock_timeout"
)

// Policy bounds what reclaim may remove.
type Policy struct {
	// BudgetBytes is the size the store is trimmed down to.
	BudgetBytes int64
	// MinIdle is the minimum time since last use before an entry is eligible.
	MinIdle time.Duration
}

func (p Policy) validate() error {
	if p.BudgetBytes < 0 {
		return errors.New("budget must not be negative")
	}
	if p.MinIdle < bucketWidth {
		return fmt.Errorf("minimum idle time must be at least %s", bucketWidth)
	}
	return nil
}

// Plan is a reclaim proposal bound to one store by its token.
type Plan struct {
	Schema             string       `json:"schema"`
	Cache              string       `json:"cache"`
	StoreID            string       `json:"store_id"`
	Token              string       `json:"token,omitempty"`
	ObservedAt         time.Time    `json:"observed_at"`
	ScannedAt          *time.Time   `json:"scanned_at,omitempty"`
	StoreBytes         int64        `json:"store_bytes"`
	StoreEntries       int          `json:"store_entries"`
	BudgetBytes        int64        `json:"budget_bytes"`
	MinIdleSeconds     int64        `json:"min_idle_seconds"`
	Cutoff             *time.Time   `json:"cutoff,omitempty"`
	ReclaimableBytes   int64        `json:"reclaimable_bytes"`
	ReclaimableEntries int          `json:"reclaimable_entries"`
	Activation         Activation   `json:"activation"`
	LiveSessions       int          `json:"live_sessions"`
	Audit              *AuditReport `json:"audit,omitempty"`
	BlockedReason      string       `json:"blocked_reason,omitempty"`
}

// Receipt records one apply.
type Receipt struct {
	Schema              string       `json:"schema"`
	Cache               string       `json:"cache"`
	StoreID             string       `json:"store_id,omitempty"`
	Token               string       `json:"token,omitempty"`
	Cutoff              *time.Time   `json:"cutoff,omitempty"`
	StartedAt           time.Time    `json:"started_at"`
	FinishedAt          time.Time    `json:"finished_at"`
	FreedBytes          int64        `json:"freed_bytes"`
	RemovedEntries      int          `json:"removed_entries"`
	SkippedPinned       int          `json:"skipped_pinned"`
	SkippedRecentlyUsed int          `json:"skipped_recently_used"`
	FailedRemovals      int          `json:"failed_removals"`
	FirstFailure        string       `json:"first_failure,omitempty"`
	ShardsSwept         int          `json:"shards_swept"`
	NextShard           int          `json:"next_shard"`
	PlanExhausted       bool         `json:"plan_exhausted"`
	StopReason          string       `json:"stop_reason"`
	BlockedReason       string       `json:"blocked_reason,omitempty"`
	LiveSessions        int          `json:"live_sessions"`
	Audit               *AuditReport `json:"audit,omitempty"`
}

// PlanOptions configures MakePlan.
type PlanOptions struct {
	Store   Store
	Program string
	Policy  Policy
	// Fresh forces a new scan instead of continuing a recent plan.
	Fresh bool
	Now   func() time.Time
}

// ApplyOptions configures Apply.
type ApplyOptions struct {
	Store   Store
	Program string
	Policy  Policy
	Token   string
	// MaxBytes bounds this apply; zero removes everything the plan selected.
	MaxBytes int64
	// Deadline stops the sweep between batches; zero means no deadline.
	Deadline time.Time
	Now      func() time.Time
}

type savedPlan struct {
	Plan   Plan      `json:"plan"`
	Cutoff time.Time `json:"cutoff"`
}

type sweepCursor struct {
	Token     string `json:"token"`
	NextShard int    `json:"next_shard"`
	Freed     int64  `json:"freed_bytes"`
	Exhausted bool   `json:"exhausted"`
}

// MakePlan proposes the least-recently-used entries whose removal brings the
// store within budget. It never removes anything.
func MakePlan(opts PlanOptions) (Plan, error) {
	now := clock(opts.Now)
	at := now()
	if err := opts.Policy.validate(); err != nil {
		return Plan{}, err
	}
	state, err := openState(opts.Store)
	if err != nil {
		return Plan{}, err
	}
	defer state.Close()
	storeID, err := state.StoreID()
	if err != nil {
		return Plan{}, err
	}
	plan := Plan{
		Schema: PlanSchema, Cache: opts.Store.Root, StoreID: storeID, ObservedAt: at.UTC(),
		BudgetBytes: opts.Policy.BudgetBytes, MinIdleSeconds: int64(opts.Policy.MinIdle / time.Second),
		Activation: ReadActivation(opts.Program),
	}
	plan.LiveSessions, _ = countLiveSessions(state.sessionsDir())
	if !plan.Activation.Active {
		plan.BlockedReason = "build-cache protocol inactive: " + plan.Activation.Reason
		return plan, nil
	}
	reason, report := auditStore(opts.Store, opts.Program, now)
	plan.Audit = report
	if reason != "" {
		plan.BlockedReason = reason
		return plan, nil
	}
	if !opts.Fresh {
		if cached, ok := continuablePlan(state, storeID, opts.Policy, at); ok {
			cached.ObservedAt, cached.Activation, cached.Audit, cached.LiveSessions = plan.ObservedAt, plan.Activation, plan.Audit, plan.LiveSessions
			return cached, nil
		}
	}
	scan, err := scanStore(opts.Store, now)
	if err != nil {
		return Plan{}, err
	}
	scannedAt := at.UTC()
	plan.ScannedAt = &scannedAt
	plan.StoreBytes, plan.StoreEntries = scan.bytes, scan.entries
	cutoff, bytes, entries, ok := scan.cutoff(at, opts.Policy)
	if !ok {
		return plan, nil
	}
	plan.Cutoff = &cutoff
	plan.ReclaimableBytes, plan.ReclaimableEntries = bytes, entries
	plan.Token = encodeToken(storeID, cutoff, scannedAt)
	if err := writeJSONAtomic(state.path(planFileName), savedPlan{Plan: plan, Cutoff: cutoff}); err != nil {
		return Plan{}, fmt.Errorf("record plan: %w", err)
	}
	return plan, nil
}

// continuablePlan returns the last plan, less what its sweep already freed,
// while it is recent, unexhausted, and made under the same policy.
func continuablePlan(state *protocolState, storeID string, policy Policy, now time.Time) (Plan, bool) {
	var saved savedPlan
	if err := readJSON(state.path(planFileName), &saved); err != nil {
		return Plan{}, false
	}
	plan := saved.Plan
	if plan.StoreID != storeID || plan.Token == "" || plan.ScannedAt == nil || now.Sub(*plan.ScannedAt) > planTTL ||
		plan.BudgetBytes != policy.BudgetBytes || plan.MinIdleSeconds != int64(policy.MinIdle/time.Second) {
		return Plan{}, false
	}
	var cursor sweepCursor
	if err := readJSON(state.path(cursorFileName), &cursor); err == nil && cursor.Token == plan.Token {
		if cursor.Exhausted {
			return Plan{}, false
		}
		plan.ReclaimableBytes = max(plan.ReclaimableBytes-cursor.Freed, 0)
		if plan.ReclaimableBytes == 0 {
			return Plan{}, false
		}
	}
	return plan, true
}

// Apply removes, least recently used first, entries the plan selected that
// no live session has pinned and that have not been used since the plan,
// stopping at MaxBytes, the deadline, or the end of the plan.
func Apply(opts ApplyOptions) (Receipt, error) {
	now := clock(opts.Now)
	receipt := Receipt{Schema: ReceiptSchema, Cache: opts.Store.Root, Token: opts.Token, StartedAt: now().UTC()}
	finish := func(state *protocolState, reason string) (Receipt, error) {
		receipt.StopReason, receipt.FinishedAt = reason, now().UTC()
		if state != nil && receipt.StoreID != "" {
			if err := writeJSONAtomic(state.path(receiptFileName), receipt); err != nil {
				return receipt, fmt.Errorf("record reclaim receipt: %w", err)
			}
		}
		return receipt, nil
	}
	if err := opts.Policy.validate(); err != nil {
		return receipt, err
	}
	state, err := openState(opts.Store)
	if err != nil {
		return receipt, err
	}
	defer state.Close()
	storeID, err := state.StoreID()
	if err != nil {
		return receipt, err
	}
	receipt.StoreID = storeID
	tokenStore, cutoff, err := decodeToken(opts.Token)
	if err != nil {
		return receipt, err
	}
	if tokenStore != storeID {
		return receipt, fmt.Errorf("plan token is bound to store %s, not %s at %s; make a new plan", tokenStore, storeID, opts.Store.Root)
	}
	if cutoff.After(now().Add(-opts.Policy.MinIdle)) {
		return receipt, fmt.Errorf("plan cutoff %s is inside the %s minimum idle window", cutoff.Format(time.RFC3339), opts.Policy.MinIdle)
	}
	receipt.Cutoff = &cutoff

	reclaimLock, err := os.OpenFile(state.path(reclaimLockName), os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return receipt, err
	}
	defer reclaimLock.Close()
	if acquired, err := tryLock(reclaimLock, true); err != nil {
		return receipt, err
	} else if !acquired {
		receipt.BlockedReason = "another reclaim of this store is running"
		return finish(nil, StopBusy)
	}
	defer unlock(reclaimLock)

	if activation := ReadActivation(opts.Program); !activation.Active {
		receipt.BlockedReason = "build-cache protocol inactive: " + activation.Reason
		return finish(state, StopBlocked)
	}
	audit, err := newStoreAuditor(opts.Store, opts.Program, now)
	if err != nil {
		receipt.BlockedReason = err.Error()
		return finish(state, StopBlocked)
	}
	sweep := &sweeper{state: state, store: opts.Store, cutoff: cutoff, now: now, audit: audit, pins: newPinSnapshot(), receipt: &receipt, deadline: opts.Deadline}
	var cursor sweepCursor
	if err := readJSON(state.path(cursorFileName), &cursor); err != nil || cursor.Token != opts.Token {
		cursor = sweepCursor{Token: opts.Token}
	}
	if cursor.Exhausted {
		receipt.PlanExhausted, receipt.NextShard = true, shardCount
		return finish(state, StopComplete)
	}
	reason := StopComplete
	shard := cursor.NextShard
	for ; shard < shardCount; shard++ {
		if !opts.Deadline.IsZero() && !now().Before(opts.Deadline) {
			reason = StopDeadline
			break
		}
		if opts.MaxBytes > 0 && receipt.FreedBytes >= opts.MaxBytes {
			reason = StopMaxBytes
			break
		}
		before := receipt.FreedBytes
		done, stop, err := sweep.shard(byte(shard), opts.MaxBytes)
		cursor.Freed += receipt.FreedBytes - before
		if err != nil {
			return receipt, err
		}
		if stop != "" {
			reason = stop
			break
		}
		if done {
			receipt.ShardsSwept++
		} else {
			reason = StopMaxBytes
			break
		}
	}
	cursor.NextShard = shard
	cursor.Exhausted = shard >= shardCount
	receipt.NextShard, receipt.PlanExhausted = shard, cursor.Exhausted
	receipt.LiveSessions = sweep.pins.liveSessions()
	if err := writeJSONAtomic(state.path(cursorFileName), cursor); err != nil {
		return receipt, fmt.Errorf("record sweep position: %w", err)
	}
	return finish(state, reason)
}

type sweeper struct {
	state    *protocolState
	store    Store
	cutoff   time.Time
	now      func() time.Time
	audit    storeAuditor
	pins     *pinSnapshot
	receipt  *Receipt
	deadline time.Time
}

type candidate struct {
	id    ID
	path  string
	bytes int64
	temp  bool
}

// shard sweeps one shard. It reports whether the shard was finished and,
// when the sweep must stop, why.
func (s *sweeper) shard(shard byte, maxBytes int64) (bool, string, error) {
	candidates, err := s.candidates(shard)
	if err != nil {
		return false, "", err
	}
	if len(candidates) == 0 {
		return true, "", nil
	}
	if acquired, err := s.lockExclusive(); err != nil {
		return false, "", err
	} else if !acquired {
		s.receipt.BlockedReason = "timed out waiting for live sessions to release the coordination lock"
		return false, StopLockTimeout, nil
	}
	defer unlock(s.state.coord)
	report, err := s.audit.run()
	s.receipt.Audit = &report
	if err != nil {
		s.receipt.BlockedReason = "go-command audit failed: " + err.Error()
		return false, StopBlocked, nil
	}
	if !report.Clear() {
		s.receipt.BlockedReason = unmediatedReason(report)
		return false, StopBlocked, nil
	}
	if err := s.pins.refresh(s.state.sessionsDir()); err != nil {
		return false, "", err
	}
	if s.pins.unreadable != "" {
		s.receipt.BlockedReason = "live session journal " + s.pins.unreadable + " has an unknown format"
		return false, StopBlocked, nil
	}
	for _, c := range candidates {
		if maxBytes > 0 && s.receipt.FreedBytes >= maxBytes {
			return false, "", nil
		}
		if !c.temp && s.pins.pinned(c.id) {
			s.receipt.SkippedPinned++
			continue
		}
		info, err := os.Lstat(c.path)
		if err != nil {
			continue
		}
		if !c.temp && info.ModTime().After(s.cutoff) {
			s.receipt.SkippedRecentlyUsed++
			continue
		}
		if info.IsDir() {
			err = os.RemoveAll(c.path)
		} else {
			err = os.Remove(c.path)
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			s.receipt.FailedRemovals++
			if s.receipt.FirstFailure == "" {
				s.receipt.FirstFailure = err.Error()
			}
			continue
		}
		s.receipt.FreedBytes += c.bytes
		s.receipt.RemovedEntries++
	}
	return true, "", nil
}

// candidates lists the shard's entries last used at or before the cutoff,
// oldest first in directory order, plus abandoned staged writes. Listing
// happens without the coordination lock; every candidate is re-checked
// under it.
func (s *sweeper) candidates(shard byte) ([]candidate, error) {
	dir := filepath.Join(s.store.Root, shardName(shard))
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list cache shard %s: %w", shardName(shard), err)
	}
	staleTemp := s.now().Add(-staleTempAge)
	var out []candidate
	for _, entry := range entries {
		name := entry.Name()
		id, _, isEntry := parseEntryName(name)
		isTemp := !isEntry && strings.Contains(name, tempMarker)
		if !isEntry && !isTemp {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		path := filepath.Join(dir, name)
		switch {
		case isTemp && info.ModTime().Before(staleTemp):
			out = append(out, candidate{path: path, bytes: allocatedBytes(info), temp: true})
		case isEntry && !info.ModTime().After(s.cutoff):
			out = append(out, candidate{id: id, path: path, bytes: treeBytes(path, info)})
		}
	}
	return out, nil
}

func (s *sweeper) lockExclusive() (bool, error) {
	wait := time.Millisecond
	limit := s.now().Add(30 * time.Second)
	if !s.deadline.IsZero() && s.deadline.Before(limit) {
		limit = s.deadline
	}
	for {
		acquired, err := tryLock(s.state.coord, true)
		if err != nil || acquired {
			return acquired, err
		}
		if !s.now().Before(limit) {
			return false, nil
		}
		time.Sleep(wait)
		wait = min(wait*2, 50*time.Millisecond)
	}
}

func unmediatedReason(report AuditReport) string {
	first := report.Unmediated[0]
	reason := fmt.Sprintf("%d go command(s) may use the store outside the protocol; pid %d: %s", len(report.Unmediated), first.PID, first.Reason)
	if first.Command != "" {
		reason += " (" + first.Command + ")"
	}
	return reason
}

// auditStore runs a one-off audit and reports why reclaim must wait, if it
// must.
func auditStore(store Store, program string, now func() time.Time) (string, *AuditReport) {
	audit, err := newStoreAuditor(store, program, now)
	if err != nil {
		return err.Error(), nil
	}
	report, err := audit.run()
	if err != nil {
		return "go-command audit failed: " + err.Error(), &report
	}
	if !report.Clear() {
		return unmediatedReason(report), &report
	}
	return "", &report
}

type bucket struct {
	bytes   int64
	entries int
}

type storeScan struct {
	bytes   int64
	entries int
	buckets []bucket // index = age / bucketWidth
}

func scanStore(store Store, now func() time.Time) (storeScan, error) {
	at := now()
	scan := storeScan{buckets: make([]bucket, int(maxBucketAge/bucketWidth)+1)}
	for shard := range shardCount {
		dir := filepath.Join(store.Root, shardName(byte(shard)))
		entries, err := os.ReadDir(dir)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return storeScan{}, fmt.Errorf("list cache shard %s: %w", shardName(byte(shard)), err)
		}
		for _, entry := range entries {
			if _, _, ok := parseEntryName(entry.Name()); !ok {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			size := treeBytes(filepath.Join(dir, entry.Name()), info)
			index := min(max(int(at.Sub(info.ModTime())/bucketWidth), 0), len(scan.buckets)-1)
			scan.buckets[index].bytes += size
			scan.buckets[index].entries++
			scan.bytes += size
			scan.entries++
		}
	}
	return scan, nil
}

// cutoff finds the newest last-use time such that removing every entry used
// at or before it brings the store within budget, never reaching into the
// minimum idle window. It reports false when nothing needs removing.
func (s storeScan) cutoff(now time.Time, policy Policy) (time.Time, int64, int, bool) {
	excess := s.bytes - policy.BudgetBytes
	if excess <= 0 {
		return time.Time{}, 0, 0, false
	}
	floor := int((policy.MinIdle + bucketWidth - 1) / bucketWidth)
	var bytes int64
	entries, chosen := 0, -1
	for index := len(s.buckets) - 1; index >= floor; index-- {
		if s.buckets[index].entries == 0 {
			continue
		}
		bytes += s.buckets[index].bytes
		entries += s.buckets[index].entries
		chosen = index
		if bytes >= excess {
			break
		}
	}
	if chosen < 0 {
		return time.Time{}, 0, 0, false
	}
	return now.Add(-time.Duration(chosen) * bucketWidth), bytes, entries, true
}

// treeBytes sizes a cache entry; native executable-cache entries are
// directories.
func treeBytes(path string, info os.FileInfo) int64 {
	if !info.IsDir() {
		return allocatedBytes(info)
	}
	var total int64
	_ = filepath.WalkDir(path, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if child, err := entry.Info(); err == nil {
			total += allocatedBytes(child)
		}
		return nil
	})
	return total
}

func encodeToken(storeID string, cutoff, scannedAt time.Time) string {
	return fmt.Sprintf("v1.%s.%d.%d", storeID, cutoff.UnixNano(), scannedAt.UnixNano())
}

func decodeToken(token string) (string, time.Time, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 4 || parts[0] != "v1" || parts[1] == "" {
		return "", time.Time{}, errors.New("malformed plan token")
	}
	cutoff, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return "", time.Time{}, errors.New("malformed plan token")
	}
	if _, err := strconv.ParseInt(parts[3], 10, 64); err != nil {
		return "", time.Time{}, errors.New("malformed plan token")
	}
	return parts[1], time.Unix(0, cutoff), nil
}

func clock(now func() time.Time) func() time.Time {
	if now == nil {
		return time.Now
	}
	return now
}
