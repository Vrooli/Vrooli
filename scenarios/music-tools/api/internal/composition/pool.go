package composition

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"music-tools/internal/jobs"
)

var ErrUnavailable = errors.New("take is not available")

type Pool struct {
	mu               sync.Mutex
	takes            map[string]Take
	target           map[string]int
	threshold        map[string]int
	lastReplenishJob map[string]string
	inflight         map[string]bool
	reservationTTL   time.Duration
	db               jobs.SQLExecutor
	trigger          func(styleID string, needed int)
	blobPolicy       BlobPolicy
}

type BlobPolicy interface {
	SetProtected(key string, protected bool)
	SetEvictionPriority(key string, priority int)
}

func NewPool() *Pool {
	return &Pool{takes: map[string]Take{}, target: map[string]int{}, threshold: map[string]int{}, lastReplenishJob: map[string]string{}, inflight: map[string]bool{}, reservationTTL: 30 * time.Minute}
}

const poolSchema = `
CREATE TABLE IF NOT EXISTS music_pool_takes (
  id TEXT PRIMARY KEY, job_id TEXT NOT NULL, style_id TEXT NOT NULL, pool_state TEXT NOT NULL,
  blob_ref TEXT NOT NULL, reserved_by TEXT NOT NULL DEFAULT '', reserved_at TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL, model_id TEXT NOT NULL, license_lane TEXT NOT NULL, applied_rung TEXT NOT NULL,
  seed INTEGER NOT NULL, caption_as_authored TEXT NOT NULL, caption_as_sent TEXT NOT NULL, times_offered INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS style_inventory (
	  style_id TEXT PRIMARY KEY, target INTEGER NOT NULL, threshold INTEGER NOT NULL, last_replenish_job TEXT NOT NULL DEFAULT ''
);`

func Schema() string { return poolSchema }

func NewPoolWithStore(db jobs.SQLExecutor) *Pool {
	p := NewPool()
	p.db = db
	if db == nil {
		return p
	}
	if _, err := db.ExecContext(context.Background(), poolSchema); err != nil {
		return p
	}
	// style_inventory predates last_replenish_job in already-running installs.
	// SQLite has no portable IF NOT EXISTS form for ADD COLUMN, so ignore the
	// duplicate-column error and retain the additive migration for old stores.
	_, _ = db.ExecContext(context.Background(), `ALTER TABLE style_inventory ADD COLUMN last_replenish_job TEXT NOT NULL DEFAULT ''`)
	rows, err := db.QueryContext(context.Background(), `SELECT id, job_id, style_id, pool_state, blob_ref, reserved_by, reserved_at, created_at, model_id, license_lane, applied_rung, seed, caption_as_authored, caption_as_sent, times_offered FROM music_pool_takes`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var t Take
			var reservedAt, createdAt string
			if err := rows.Scan(&t.ID, &t.JobID, &t.StyleID, &t.PoolState, &t.BlobRef, &t.ReservedBy, &reservedAt, &createdAt, &t.Provenance.ModelID, &t.Provenance.LicenseLane, &t.Provenance.AppliedRung, &t.Provenance.Seed, &t.Provenance.CaptionAsAuthored, &t.Provenance.CaptionAsSent, &t.TimesOffered); err != nil {
				continue
			}
			t.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
			if reservedAt != "" {
				if parsed, parseErr := time.Parse(time.RFC3339Nano, reservedAt); parseErr == nil {
					t.ReservedAt = &parsed
				}
			}
			p.takes[t.ID] = t
		}
		rows.Close()
	}
	policy, err := db.QueryContext(context.Background(), `SELECT style_id, target, threshold FROM style_inventory`)
	if err == nil {
		defer policy.Close()
		for policy.Next() {
			var style string
			var target, threshold int
			if policy.Scan(&style, &target, &threshold) != nil {
				continue
			}
			p.target[style], p.threshold[style] = target, threshold
		}
	}
	// Read the optional job column separately so databases created by an older
	// revision remain readable after the additive schema change.
	if policy, err := db.QueryContext(context.Background(), `SELECT style_id, last_replenish_job FROM style_inventory`); err == nil {
		defer policy.Close()
		for policy.Next() {
			var style, jobID string
			if policy.Scan(&style, &jobID) == nil {
				p.lastReplenishJob[style] = jobID
			}
		}
	}
	return p
}

func (p *Pool) persist(t Take) {
	if p.db == nil {
		return
	}
	reservedAt := ""
	if t.ReservedAt != nil {
		reservedAt = t.ReservedAt.UTC().Format(time.RFC3339Nano)
	}
	_, _ = p.db.ExecContext(context.Background(), `INSERT INTO music_pool_takes (id, job_id, style_id, pool_state, blob_ref, reserved_by, reserved_at, created_at, model_id, license_lane, applied_rung, seed, caption_as_authored, caption_as_sent, times_offered) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET job_id=excluded.job_id, style_id=excluded.style_id, pool_state=excluded.pool_state, blob_ref=excluded.blob_ref, reserved_by=excluded.reserved_by, reserved_at=excluded.reserved_at, created_at=excluded.created_at, model_id=excluded.model_id, license_lane=excluded.license_lane, applied_rung=excluded.applied_rung, seed=excluded.seed, caption_as_authored=excluded.caption_as_authored, caption_as_sent=excluded.caption_as_sent, times_offered=excluded.times_offered`, t.ID, t.JobID, t.StyleID, t.PoolState, t.BlobRef, t.ReservedBy, reservedAt, t.CreatedAt.UTC().Format(time.RFC3339Nano), t.Provenance.ModelID, t.Provenance.LicenseLane, t.Provenance.AppliedRung, t.Provenance.Seed, t.Provenance.CaptionAsAuthored, t.Provenance.CaptionAsSent, t.TimesOffered)
}

func (p *Pool) persistPolicy(styleID string, target, threshold int) {
	if p.db != nil {
		_, _ = p.db.ExecContext(context.Background(), `INSERT INTO style_inventory (style_id, target, threshold, last_replenish_job) VALUES (?, ?, ?, ?) ON CONFLICT(style_id) DO UPDATE SET target=excluded.target, threshold=excluded.threshold, last_replenish_job=excluded.last_replenish_job`, styleID, target, threshold, p.lastReplenishJob[styleID])
	}
}

func (p *Pool) SetBlobPolicy(policy BlobPolicy) { p.mu.Lock(); p.blobPolicy = policy; p.mu.Unlock() }

func (p *Pool) SetLastReplenishJob(styleID, jobID string) {
	p.mu.Lock()
	p.lastReplenishJob[styleID] = jobID
	target, threshold := p.target[styleID], p.threshold[styleID]
	p.persistPolicy(styleID, target, threshold)
	p.mu.Unlock()
}

func (p *Pool) LastReplenishJob(styleID string) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.lastReplenishJob[styleID]
}

func (p *Pool) setBlobState(take Take, protected bool, priority int) {
	if p.blobPolicy == nil || take.BlobRef == "" {
		return
	}
	p.blobPolicy.SetProtected(take.BlobRef, protected)
	p.blobPolicy.SetEvictionPriority(take.BlobRef, priority)
}

func (p *Pool) SetReplenishTrigger(trigger func(styleID string, needed int)) {
	p.mu.Lock()
	p.trigger = trigger
	p.mu.Unlock()
}

func (p *Pool) maybeTrigger(styleID string, needs bool, needed int) {
	if !needs {
		return
	}
	p.mu.Lock()
	trigger := p.trigger
	if trigger == nil || p.inflight[styleID] {
		p.mu.Unlock()
		return
	}
	p.inflight[styleID] = true
	if needed < 1 {
		needed = p.target[styleID] - p.depthLocked(styleID)
	}
	p.mu.Unlock()
	if needed > 0 {
		trigger(styleID, needed)
	} else {
		p.ReplenishmentFinished(styleID)
	}
}

func (p *Pool) Add(takes ...Take) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, take := range takes {
		take.PoolState = "available"
		p.takes[take.ID] = take
		p.persist(take)
		p.setBlobState(take, false, 1)
	}
}

func (p *Pool) Configure(styleID string, target, threshold int) error {
	if styleID == "" || target < 1 || threshold < 0 || threshold >= target {
		return ErrInvalidBatch
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.target[styleID], p.threshold[styleID] = target, threshold
	p.persistPolicy(styleID, target, threshold)
	return nil
}

func (p *Pool) Draw(styleID, holder string) (Take, bool, error) {
	p.mu.Lock()
	for id, take := range p.takes {
		if take.PoolState == "reserved" && take.ReservedAt != nil && time.Since(*take.ReservedAt) >= p.reservationTTL {
			take.PoolState, take.ReservedBy, take.ReservedAt = "available", "", nil
			p.takes[id] = take
			p.persist(take)
			p.setBlobState(take, false, 1)
		}
		if take.StyleID != styleID || take.PoolState != "available" {
			continue
		}
		take, needs, needed := p.reserveLocked(id, holder)
		p.mu.Unlock()
		p.maybeTrigger(styleID, needs, needed)
		return take, needs, nil
	}
	needs := p.shouldReplenishLocked(styleID)
	p.mu.Unlock()
	return Take{}, needs, ErrUnavailable
}

// Reserve transitions one exact take to reserved. Draw is the style-level
// selection operation; callers that already selected an id must use Reserve so
// a concurrent or unrelated take cannot be reserved by mistake.
func (p *Pool) Reserve(id, holder string) (Take, bool, error) {
	p.mu.Lock()
	take, ok := p.takes[id]
	if !ok {
		p.mu.Unlock()
		return Take{}, false, ErrUnavailable
	}
	if take.PoolState == "reserved" && take.ReservedAt != nil && time.Since(*take.ReservedAt) >= p.reservationTTL {
		take.PoolState, take.ReservedBy, take.ReservedAt = "available", "", nil
		p.takes[id] = take
		p.persist(take)
		p.setBlobState(take, false, 1)
	}
	if take.PoolState != "available" {
		p.mu.Unlock()
		return Take{}, false, ErrUnavailable
	}
	take, needs, needed := p.reserveLocked(id, holder)
	p.mu.Unlock()
	p.maybeTrigger(take.StyleID, needs, needed)
	return take, needs, nil
}

func (p *Pool) reserveLocked(id, holder string) (Take, bool, int) {
	take := p.takes[id]
	now := time.Now().UTC()
	take.PoolState, take.ReservedBy, take.ReservedAt = "reserved", holder, &now
	// Reservation ownership is tracked separately from the immutable audio
	// blob reference; drawing a take must never destroy its artifact path.
	p.takes[id] = take
	p.persist(take)
	p.setBlobState(take, true, 1)
	needs := p.shouldReplenishLocked(take.StyleID)
	needed := p.target[take.StyleID] - p.depthLocked(take.StyleID)
	return take, needs, needed
}

func (p *Pool) MarkConsumed(id string) (Take, error) {
	p.mu.Lock()
	take, ok := p.takes[id]
	if !ok || take.PoolState != "reserved" {
		p.mu.Unlock()
		return Take{}, ErrUnavailable
	}
	take.PoolState, take.ReservedBy, take.ReservedAt = "consumed", "", nil
	p.takes[id] = take
	p.persist(take)
	p.setBlobState(take, false, 2)
	needs := p.shouldReplenishLocked(take.StyleID)
	needed := p.target[take.StyleID] - p.depthLocked(take.StyleID)
	p.mu.Unlock()
	p.maybeTrigger(take.StyleID, needs, needed)
	return take, nil
}

func (p *Pool) Discard(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	take, ok := p.takes[id]
	if !ok || (take.PoolState != "available" && take.PoolState != "reserved") {
		return ErrUnavailable
	}
	take.PoolState = "discarded"
	p.takes[id] = take
	p.persist(take)
	p.setBlobState(take, false, 0)
	return nil
}

func (p *Pool) Release(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	take, ok := p.takes[id]
	if !ok || take.PoolState != "reserved" {
		return ErrUnavailable
	}
	take.PoolState, take.ReservedBy, take.ReservedAt = "available", "", nil
	take.TimesOffered++
	p.takes[id] = take
	p.persist(take)
	p.setBlobState(take, false, 1)
	return nil
}

func (p *Pool) Consume(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	take, ok := p.takes[id]
	if !ok || take.PoolState != "reserved" {
		return ErrUnavailable
	}
	delete(p.takes, id)
	if p.db != nil {
		_, _ = p.db.ExecContext(context.Background(), `DELETE FROM music_pool_takes WHERE id=?`, id)
	}
	return nil
}

func (p *Pool) ConfigureReplenishment(styleID string) (bool, int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.inflight[styleID] {
		return false, 0
	}
	if !p.shouldReplenishLocked(styleID) {
		return false, 0
	}
	p.inflight[styleID] = true
	needed := p.target[styleID] - p.depthLocked(styleID)
	return true, needed
}

func (p *Pool) ReplenishmentFinished(styleID string) {
	p.mu.Lock()
	delete(p.inflight, styleID)
	p.mu.Unlock()
}

func (p *Pool) depthLocked(styleID string) int {
	depth := 0
	for _, take := range p.takes {
		if take.StyleID == styleID && (take.PoolState == "available" || take.PoolState == "reserved") {
			depth++
		}
	}
	return depth
}

func (p *Pool) shouldReplenishLocked(styleID string) bool {
	return p.depthLocked(styleID) < p.threshold[styleID]
}

func (p *Pool) Status(styleID string) (available, reserved, target, threshold int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	target, threshold = p.target[styleID], p.threshold[styleID]
	for _, take := range p.takes {
		if take.StyleID != styleID {
			continue
		}
		if take.PoolState == "available" {
			available++
		}
		if take.PoolState == "reserved" {
			reserved++
		}
	}
	return
}

func (p *Pool) String() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return fmt.Sprintf("takes=%d", len(p.takes))
}
