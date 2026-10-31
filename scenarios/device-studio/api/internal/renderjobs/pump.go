package renderjobs

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"device-studio/internal/jobs"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
)

const progressBatch = 24

// Pump owns bounded local progress for queued/claimed jobs. It rotates an
// immutable keyset cursor per storage route, so one unresolved old operation
// cannot starve later work. Browser reads never invoke it.
type Pump struct {
	coordinator *Coordinator
	jobs        jobs.Service
	db          *database.RoutedDB
	logger      *log.Logger
	advance     func(context.Context, string) error
	interval    time.Duration
	wake        chan struct{}
	roots       *filerouting.RoutedRoots
	cancel      context.CancelFunc
	done        chan struct{}
	once        sync.Once
	mu          sync.Mutex
	testLease   *testLeaseActivation
}

type testLeaseActivation struct {
	leaseID, dataRoot string
	// routeCtx retains the exact DB pool and file-root generation accepted by
	// the routed request. Never pin a fresh lease from a later lifecycle tick.
	routeCtx context.Context
}

// lifecycleRouteContext keeps the captured routing values while giving each
// pump step the lifecycle's cancellation and deadline.
type lifecycleRouteContext struct {
	context.Context
	lifecycle context.Context
}

func (c lifecycleRouteContext) Deadline() (time.Time, bool) { return c.lifecycle.Deadline() }
func (c lifecycleRouteContext) Done() <-chan struct{}       { return c.lifecycle.Done() }
func (c lifecycleRouteContext) Err() error                  { return c.lifecycle.Err() }

func NewPump(coordinator *Coordinator, service jobs.Service, db *database.RoutedDB, roots *filerouting.RoutedRoots, logger *log.Logger) *Pump {
	if logger == nil {
		logger = log.Default()
	}
	return &Pump{coordinator: coordinator, jobs: service, db: db, roots: roots, logger: logger, interval: 750 * time.Millisecond, wake: make(chan struct{}, 1), done: make(chan struct{}), advance: func(ctx context.Context, id string) error { _, err := coordinator.Advance(ctx, id); return err }}
}

// ActivateTestRoute is called only after a successful routed submit or cancel.
// An installed but unrelated test pool never activates test progress.
func (p *Pump) ActivateTestRoute(ctx context.Context) error {
	if !database.IsTestMode(ctx) {
		p.wakeProgress()
		return nil
	}
	if p.db == nil || p.roots == nil {
		return errors.New("test render route is not active")
	}
	leaseID := p.db.ActiveLeaseID()
	if leaseID == "" {
		return errors.New("test render database route has no active lease")
	}
	pinned, err := p.db.PinTestPool(ctx, leaseID)
	if err != nil {
		return err
	}
	pinned, err = p.roots.PinTestRoots(pinned, leaseID)
	if err != nil {
		return err
	}
	root, err := p.roots.PickRequired(pinned, storage.ClassData)
	if err != nil {
		return err
	}
	p.mu.Lock()
	p.testLease = &testLeaseActivation{leaseID: leaseID, dataRoot: root, routeCtx: context.WithoutCancel(pinned)}
	p.mu.Unlock()
	p.wakeProgress()
	return nil
}

// A wake carries no routing authority. Progress still resolves the primary
// route and validates the exact activated test lease before every step.
func (p *Pump) wakeProgress() {
	select {
	case p.wake <- struct{}{}:
	default:
	}
}

func (p *Pump) Start(parent context.Context) {
	p.once.Do(func() {
		ctx, cancel := context.WithCancel(parent)
		p.cancel = cancel
		go p.run(ctx)
	})
}

func (p *Pump) Stop(ctx context.Context) error {
	if p.cancel == nil {
		return nil
	}
	p.cancel()
	select {
	case <-p.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *Pump) run(ctx context.Context) {
	defer close(p.done)
	var primaryCursor, testCursor jobs.Cursor
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		var primaryActive, testActive bool
		primaryCursor, primaryActive = p.advanceRoute(ctx, ctx, primaryCursor, "primary")
		if testCtx, activation, ok := p.activeTestContext(ctx); ok {
			testCursor, testActive = p.advanceRoute(ctx, testCtx, testCursor, "test", activation)
		} else {
			testCursor = jobs.Cursor{}
		}
		delay := p.interval
		if (primaryActive || testActive) && delay > 250*time.Millisecond {
			delay = 250 * time.Millisecond
		}
		ticker.Reset(delay)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-p.wake:
		}
	}
}

func (p *Pump) activeTestContext(lifecycle context.Context) (context.Context, *testLeaseActivation, bool) {
	p.mu.Lock()
	activation := p.testLease
	p.mu.Unlock()
	if activation == nil || p.db == nil || p.roots == nil {
		p.retireTestActivation(activation)
		return nil, nil, false
	}
	testCtx := lifecycleRouteContext{Context: activation.routeCtx, lifecycle: lifecycle}
	root, err := p.roots.PickRequired(testCtx, storage.ClassData)
	if err != nil || root != activation.dataRoot {
		p.retireTestActivation(activation)
		return nil, nil, false
	}
	return testCtx, activation, true
}

func (p *Pump) retireTestActivation(expected *testLeaseActivation) {
	if expected == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.testLease == expected {
		p.testLease = nil
	}
}

func (p *Pump) testActivationValid(expected *testLeaseActivation, ctx context.Context) bool {
	if expected == nil || p.db == nil || p.roots == nil {
		return false
	}
	if _, err := p.db.PoolForContext(expected.routeCtx); err != nil {
		return false
	}
	root, err := p.roots.PickRequired(ctx, storage.ClassData)
	return err == nil && root == expected.dataRoot
}

func (p *Pump) advanceRoute(lifecycle context.Context, routeCtx context.Context, cursor jobs.Cursor, routeName string, activation ...*testLeaseActivation) (jobs.Cursor, bool) {
	if lifecycle.Err() != nil {
		return cursor, false
	}
	queryCtx, cancel := context.WithTimeout(routeCtx, 3*time.Second)
	rows, err := p.jobs.ListProgress(queryCtx, cursor, progressBatch)
	cancel()
	if err != nil {
		p.logger.Printf("render job %s progress listing failed: %v", routeName, err)
		if routeName == "test" && len(activation) > 0 && !p.testActivationValid(activation[0], routeCtx) {
			p.retireTestActivation(activation[0])
		}
		return jobs.Cursor{}, false
	}
	if len(rows) == 0 {
		return jobs.Cursor{}, false
	}
	for _, job := range rows {
		if lifecycle.Err() != nil {
			return cursor, false
		}
		if routeName == "test" && (len(activation) == 0 || !p.testActivationValid(activation[0], routeCtx)) {
			if len(activation) > 0 {
				p.retireTestActivation(activation[0])
			}
			return jobs.Cursor{}, false
		}
		if err := p.advance(routeCtx, job.ID); err != nil {
			p.logger.Printf("render job %s progress step unresolved: %v", job.ID, err)
		}
		cursor = jobs.Cursor{CreatedAt: job.CreatedAt, ID: job.ID}
	}
	if len(rows) < progressBatch {
		return jobs.Cursor{}, true
	}
	return cursor, true
}
