package composition

import (
	"context"
	_ "embed"
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
	ProtectAndCommit(key string, commit func() error) (bool, error)
	SetEvictionPriority(key string, priority int)
}

func NewPool() *Pool {
	return &Pool{takes: map[string]Take{}, target: map[string]int{}, threshold: map[string]int{}, lastReplenishJob: map[string]string{}, inflight: map[string]bool{}, reservationTTL: 30 * time.Minute}
}

//go:embed schema.sql
var poolSchema string

func Schema() string { return poolSchema }

// NewPoolWithStore reads a schema installed by production EnsureSchemas.
// Initialization fails explicitly; existing data is never replaced or migrated here.
func NewPoolWithStore(db jobs.SQLExecutor) (*Pool, error) {
	p := NewPool()
	p.db = db
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if db == nil {
		return p, nil
	}
	rows, err := db.QueryContext(ctx, `SELECT id, job_id, style_id, pool_state, blob_ref, reserved_by, reserved_at, created_at, model_id, license_lane, applied_rung, seed, caption_as_authored, caption_as_sent, times_offered FROM music_pool_takes`)
	if err != nil {
		return nil, fmt.Errorf("load takes (apply owner schema first): %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var t Take
		var reserved, created string
		if err = rows.Scan(&t.ID, &t.JobID, &t.StyleID, &t.PoolState, &t.BlobRef, &t.ReservedBy, &reserved, &created, &t.Provenance.ModelID, &t.Provenance.LicenseLane, &t.Provenance.AppliedRung, &t.Provenance.Seed, &t.Provenance.CaptionAsAuthored, &t.Provenance.CaptionAsSent, &t.TimesOffered); err != nil {
			return nil, err
		}
		if t.CreatedAt, err = time.Parse(time.RFC3339Nano, created); err != nil {
			return nil, fmt.Errorf("take %s created_at: %w", t.ID, err)
		}
		if reserved != "" {
			parsed, e := time.Parse(time.RFC3339Nano, reserved)
			if e != nil {
				return nil, e
			}
			t.ReservedAt = &parsed
		}
		p.takes[t.ID] = t
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	policy, err := db.QueryContext(ctx, `SELECT style_id, target, threshold, last_replenish_job FROM style_inventory`)
	if err != nil {
		return nil, fmt.Errorf("load inventory (owner schema compatibility required): %w", err)
	}
	defer policy.Close()
	for policy.Next() {
		var style, jobID string
		var target, threshold int
		if err = policy.Scan(&style, &target, &threshold, &jobID); err != nil {
			return nil, err
		}
		if target < 1 || target > 100 || threshold < 0 || threshold >= target {
			return nil, fmt.Errorf("inventory policy %s exceeds supported 1–100 target/threshold bounds; owner maintenance required", style)
		}
		p.target[style], p.threshold[style], p.lastReplenishJob[style] = target, threshold, jobID
	}
	return p, policy.Err()
}

func (p *Pool) persist(t Take) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if p.db == nil {
		return nil
	}
	reservedAt := ""
	if t.ReservedAt != nil {
		reservedAt = t.ReservedAt.UTC().Format(time.RFC3339Nano)
	}
	_, err := p.db.ExecContext(ctx, `INSERT INTO music_pool_takes (id, job_id, style_id, pool_state, blob_ref, reserved_by, reserved_at, created_at, model_id, license_lane, applied_rung, seed, caption_as_authored, caption_as_sent, times_offered) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET job_id=excluded.job_id, style_id=excluded.style_id, pool_state=excluded.pool_state, blob_ref=excluded.blob_ref, reserved_by=excluded.reserved_by, reserved_at=excluded.reserved_at, created_at=excluded.created_at, model_id=excluded.model_id, license_lane=excluded.license_lane, applied_rung=excluded.applied_rung, seed=excluded.seed, caption_as_authored=excluded.caption_as_authored, caption_as_sent=excluded.caption_as_sent, times_offered=excluded.times_offered`, t.ID, t.JobID, t.StyleID, t.PoolState, t.BlobRef, t.ReservedBy, reservedAt, t.CreatedAt.UTC().Format(time.RFC3339Nano), t.Provenance.ModelID, t.Provenance.LicenseLane, t.Provenance.AppliedRung, t.Provenance.Seed, t.Provenance.CaptionAsAuthored, t.Provenance.CaptionAsSent, t.TimesOffered)
	return err
}

func (p *Pool) persistPolicy(styleID string, target, threshold int, jobID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if p.db == nil {
		return nil
	}
	_, err := p.db.ExecContext(ctx, `INSERT INTO style_inventory (style_id, target, threshold, last_replenish_job) VALUES (?, ?, ?, ?) ON CONFLICT(style_id) DO UPDATE SET target=excluded.target, threshold=excluded.threshold, last_replenish_job=excluded.last_replenish_job`, styleID, target, threshold, jobID)
	return err
}

func (p *Pool) SetBlobPolicy(policy BlobPolicy) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.blobPolicy = policy
	for _, take := range p.takes {
		priority := 1
		if take.PoolState == "consumed" {
			priority = 2
		}
		if take.PoolState == "discarded" {
			priority = 0
		}
		p.setBlobState(take, take.PoolState == "reserved", priority)
	}
}

func (p *Pool) SetLastReplenishJob(styleID, jobID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.persistPolicy(styleID, p.target[styleID], p.threshold[styleID], jobID); err != nil {
		return err
	}
	p.lastReplenishJob[styleID] = jobID
	return nil
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

func (p *Pool) Add(takes ...Take) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, take := range takes {
		take.PoolState = "available"
		if err := p.persist(take); err != nil {
			return err
		}
		p.takes[take.ID] = take
		p.setBlobState(take, false, 1)
	}
	return nil
}

func (p *Pool) Configure(styleID string, target, threshold int) error {
	if styleID == "" || target < 1 || target > 100 || threshold < 0 || threshold >= target {
		return ErrInvalidBatch
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.persistPolicy(styleID, target, threshold, p.lastReplenishJob[styleID]); err != nil {
		return err
	}
	p.target[styleID], p.threshold[styleID] = target, threshold
	return nil
}

func (p *Pool) Draw(styleID, holder string) (Take, bool, error) {
	p.mu.Lock()
	for id, take := range p.takes {
		if take.PoolState == "reserved" && take.ReservedAt != nil && time.Since(*take.ReservedAt) >= p.reservationTTL {
			take.PoolState, take.ReservedBy, take.ReservedAt = "available", "", nil
			if err := p.persist(take); err != nil {
				p.mu.Unlock()
				return Take{}, false, err
			}
			p.takes[id] = take
			p.setBlobState(take, false, 1)
		}
		if take.StyleID != styleID || take.PoolState != "available" {
			continue
		}
		take, needs, needed, err := p.reserveLocked(id, holder)
		if errors.Is(err, ErrUnavailable) {
			continue
		}
		p.mu.Unlock()
		if err != nil {
			return Take{}, false, err
		}
		p.maybeTrigger(styleID, needs, needed)
		return take, needs, nil
	}
	needs := p.shouldReplenishLocked(styleID)
	needed := p.target[styleID] - p.depthLocked(styleID)
	p.mu.Unlock()
	p.maybeTrigger(styleID, needs, needed)
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
		if err := p.persist(take); err != nil {
			p.mu.Unlock()
			return Take{}, false, err
		}
		p.takes[id] = take
		p.setBlobState(take, false, 1)
	}
	if take.PoolState != "available" {
		p.mu.Unlock()
		return Take{}, false, ErrUnavailable
	}
	styleID := take.StyleID
	take, needs, needed, err := p.reserveLocked(id, holder)
	p.mu.Unlock()
	p.maybeTrigger(styleID, needs, needed)
	if err != nil {
		return Take{}, false, err
	}
	return take, needs, nil
}

func (p *Pool) reserveLocked(id, holder string) (Take, bool, int, error) {
	take := p.takes[id]
	now := time.Now().UTC()
	take.PoolState, take.ReservedBy, take.ReservedAt = "reserved", holder, &now
	// Reservation ownership is tracked separately from the immutable audio
	// blob reference; drawing a take must never destroy its artifact path.
	commit := func() error { return p.persist(take) }
	var err error
	available := true
	if p.blobPolicy != nil && take.BlobRef != "" {
		available, err = p.blobPolicy.ProtectAndCommit(take.BlobRef, commit)
	} else {
		err = commit()
	}
	if err != nil {
		return Take{}, false, 0, err
	}
	if !available {
		take.PoolState, take.ReservedBy, take.ReservedAt = "unavailable", "", nil
		if err := p.persist(take); err != nil {
			return Take{}, false, 0, err
		}
		p.takes[id] = take
		return Take{}, p.shouldReplenishLocked(take.StyleID), p.target[take.StyleID] - p.depthLocked(take.StyleID), ErrUnavailable
	}
	p.takes[id] = take
	p.setBlobState(take, true, 1)
	needs := p.shouldReplenishLocked(take.StyleID)
	needed := p.target[take.StyleID] - p.depthLocked(take.StyleID)
	return take, needs, needed, nil
}

func (p *Pool) MarkConsumed(id string) (Take, error) {
	p.mu.Lock()
	take, ok := p.takes[id]
	if !ok || take.PoolState != "reserved" {
		p.mu.Unlock()
		return Take{}, ErrUnavailable
	}
	take.PoolState, take.ReservedBy, take.ReservedAt = "consumed", "", nil
	if err := p.persist(take); err != nil {
		p.mu.Unlock()
		return Take{}, err
	}
	p.takes[id] = take
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
	if err := p.persist(take); err != nil {
		return err
	}
	p.takes[id] = take
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
	if err := p.persist(take); err != nil {
		return err
	}
	p.takes[id] = take
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
	if p.db != nil {
		if _, err := p.db.ExecContext(context.Background(), `DELETE FROM music_pool_takes WHERE id=?`, id); err != nil {
			return err
		}
	}
	delete(p.takes, id)
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
