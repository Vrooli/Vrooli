// Package annotationscheduler wires the shared capacity-aware scheduler to the
// run annotation service.
//
// It owns only the agent-manager side of the contract:
//
//   - a durable schedule database beside the capacity ledger (the scheduler
//     package owns its schema);
//   - one background job kind, "annotate.run", whose executor calls the
//     annotation service for a bounded run set;
//   - a producer that keeps one annotation pass queued when the configured mode
//     allows background work;
//   - an operator status endpoint that reports queue depth, deferral reasons and
//     the rolling-hour budget.
//
// On-demand annotation (the HTTP surface and the CLI) never depends on this
// package. If the capacity ledger or scheduler store is unavailable, Start
// returns an error and the caller logs it; the annotation service still works.
package annotationscheduler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/annotationfacets"
	"agent-manager/internal/runannotations"
	"agent-manager/internal/sqlcompat"

	"github.com/gorilla/mux"
	capacity "github.com/vrooli/vrooli/packages/capacity"
	"github.com/vrooli/vrooli/packages/capacity/schedule"
)

const (
	// JobKind is the one deferred work kind this package produces.
	JobKind = "annotate.run"
	// Owner names every claim and job this runner creates.
	Owner = "agent-manager:annotations"
	// DefaultInterval is how often the producer checks whether a pass is due.
	DefaultInterval = 15 * time.Minute
	// DefaultBackfillHorizon is how far back a pass selects candidate runs.
	DefaultBackfillHorizon = 30 * 24 * time.Hour
	// DefaultMaxRunsPerPass bounds one annotation pass.
	DefaultMaxRunsPerPass = 50
	// defaultEstimateRAM is a deliberately conservative per-pass claim.
	defaultEstimateRAM = int64(512 << 20)
	// defaultEstimateSeconds is the budget charge for one pass.
	defaultEstimateSeconds = 30.0
)

// ModeEnv overrides the pack's default scheduling mode without editing data.
// Accepted values: off, passive, idle, balanced, eager.
const ModeEnv = "AGENT_MANAGER_ANNOTATIONS_SCHEDULE_MODE"

// Options configures the annotation scheduler.
type Options struct {
	// Service is the annotation engine. Required.
	Service *runannotations.Service
	// DB is the Agent Manager database used to project labels into the
	// conversation-search catalog's tags. Optional; without it, facets are not
	// refreshed by the scheduler.
	DB sqlcompat.DB
	// DBPath overrides the scheduler database. Empty uses the shared default.
	DBPath string
	// Mode overrides ModeEnv and the default. Empty means passive.
	Mode schedule.Mode
	// Getenv reads ModeEnv when Mode is empty. Nil uses the process
	// environment; tests inject a map so no test touches ambient state.
	Getenv func(string) string
	// Interval is the producer cadence. Zero uses DefaultInterval.
	Interval time.Duration
	// BackfillHorizon is how far back a pass selects runs. Zero uses the
	// default. A negative value means unbounded by this package (the service
	// still applies its own pack horizon).
	BackfillHorizon time.Duration
	// MaxRunsPerPass bounds one Evaluate call. Zero uses the default.
	MaxRunsPerPass int
	// Log receives lifecycle and error messages. Nil uses slog.Default().
	Log *slog.Logger
}

// Runner owns the scheduler, its producer and its status surface.
type Runner struct {
	options   Options
	store     *schedule.Store
	broker    *capacity.Broker
	scheduler *schedule.Scheduler
	log       *slog.Logger
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	closeOnce sync.Once
	closeErr  error
}

// Start composes the scheduler. A nil service is an error; an unavailable
// capacity ledger is also an error, and the caller must treat background
// annotation as unavailable rather than bypassing admission.
func Start(ctx context.Context, opts Options) (*Runner, error) {
	if opts.Service == nil {
		return nil, errors.New("annotation scheduler requires the annotation service")
	}
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	mode, err := resolveMode(opts.Mode, opts.Getenv)
	if err != nil {
		return nil, err
	}
	interval := opts.Interval
	if interval <= 0 {
		interval = DefaultInterval
	}
	horizon := opts.BackfillHorizon
	if horizon == 0 {
		horizon = DefaultBackfillHorizon
	}
	limit := opts.MaxRunsPerPass
	if limit <= 0 {
		limit = DefaultMaxRunsPerPass
	}

	path := strings.TrimSpace(opts.DBPath)
	if path == "" {
		path, err = schedule.DefaultPath("")
		if err != nil {
			return nil, fmt.Errorf("resolve annotation scheduler database: %w", err)
		}
	}
	store, err := schedule.NewStore(ctx, schedule.Config{Path: path})
	if err != nil {
		return nil, fmt.Errorf("open annotation scheduler store: %w", err)
	}

	broker, err := capacity.NewBroker(ctx, "")
	if err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("open capacity broker for annotation scheduling: %w", err)
	}
	admission, err := schedule.NewBrokerAdmission(broker, schedule.BrokerAdmissionOptions{
		OwnerPrefix: "schedule:" + Owner,
	})
	if err != nil {
		_ = store.Close()
		_ = broker.Close()
		return nil, fmt.Errorf("compose annotation admission: %w", err)
	}

	runner := &Runner{options: opts, store: store, broker: broker, log: log}
	executor := schedule.ExecutorFunc(func(ctx context.Context, job schedule.Job) (schedule.Result, error) {
		started := time.Now()
		since := time.Time{}
		if horizon > 0 {
			since = started.Add(-horizon)
		}
		result, evalErr := opts.Service.Evaluate(ctx, runannotations.EvaluateOptions{
			Since: since,
			Until: started,
			Limit: limit,
			Force: false,
		})
		actual := time.Since(started).Seconds()
		if evalErr != nil {
			return schedule.Result{ActualSeconds: actual, Items: len(result.Runs)}, evalErr
		}
		if opts.DB != nil && len(result.Runs) > 0 {
			if refreshErr := (&annotationfacets.Refresher{DB: opts.DB, Service: opts.Service}).RefreshViews(ctx, result.Runs); refreshErr != nil {
				log.Warn("annotation facet refresh failed; annotations remain authoritative", "error", refreshErr.Error())
			}
		}
		return schedule.Result{
			ActualSeconds: actual,
			Items:         len(result.Runs),
		}, nil
	})

	policy := schedule.Policy{
		Mode:                 mode,
		MaxSecondsPerHour:    600,
		MaxJobsPerHour:       12,
		MaxModelLoadsPerHour: 6,
		ModelDwell:           15 * time.Minute,
		DefaultMaxBatch:      1,
		MaxSharePerOwner:     1,
		Interval:             30 * time.Second,
	}
	if mode == schedule.ModeIdle {
		policy.RequireIdle = true
	}
	scheduler, err := schedule.New(schedule.SchedulerConfig{
		Store:      store,
		Policy:     policy,
		Admission:  admission,
		Executor:   executor,
		Observer:   schedule.BrokerObserver{Source: broker},
		LeaseOwner: "agent-manager-annotations",
	})
	if err != nil {
		_ = store.Close()
		_ = broker.Close()
		return nil, fmt.Errorf("compose annotation scheduler: %w", err)
	}
	runner.scheduler = scheduler

	runCtx, cancel := context.WithCancel(ctx)
	runner.cancel = cancel
	runner.wg.Add(2)
	go func() {
		defer runner.wg.Done()
		scheduler.Run(runCtx)
	}()
	go func() {
		defer runner.wg.Done()
		runner.produce(runCtx, mode, interval, limit)
	}()

	log.Info("annotation scheduler started",
		"mode", string(mode), "interval", interval.String(),
		"backfill_horizon", horizon.String(), "max_runs_per_pass", limit,
		"db", path)
	return runner, nil
}

// resolveMode applies the explicit option, then the environment, then passive.
// Passive is the safe default: on-demand annotation still works, but no
// background inference starts until an operator opts in.
func resolveMode(explicit schedule.Mode, getenv func(string) string) (schedule.Mode, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	raw := explicit
	if raw == "" {
		raw = schedule.Mode(strings.TrimSpace(getenv(ModeEnv)))
	}
	if raw == "" {
		raw = schedule.ModePassive
	}
	normalized := schedule.Mode(strings.ToLower(strings.TrimSpace(string(raw))))
	if !normalized.Valid() {
		return "", fmt.Errorf("annotation scheduler: unknown mode %q (want off, passive, idle, balanced or eager)", raw)
	}
	return normalized, nil
}

// produce keeps at most one annotation pass queued. It deliberately does not
// submit background work in off or passive mode: the scheduler would reject it,
// and a growing queue is not a plan.
func (r *Runner) produce(ctx context.Context, mode schedule.Mode, interval time.Duration, limit int) {
	if mode == schedule.ModeOff || mode == schedule.ModePassive {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	r.submitIfIdle(ctx, limit)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.submitIfIdle(ctx, limit)
		}
	}
}

func (r *Runner) submitIfIdle(ctx context.Context, limit int) {
	if r == nil || r.scheduler == nil {
		return
	}
	status, err := r.scheduler.Status(ctx)
	if err != nil {
		r.log.Warn("annotation scheduler status failed", "error", err.Error())
		return
	}
	if status.Queue.Depth > 0 || status.Queue.Running > 0 {
		return
	}
	job := schedule.Job{
		Kind:          JobKind,
		Owner:         Owner,
		Priority:      schedule.PriorityBatch,
		ResourceClass: schedule.ResourceClassCompute,
		ModelAffinity: r.options.Service.Affinity(),
		Estimate: schedule.Estimate{
			Seconds:  defaultEstimateSeconds,
			RAMBytes: defaultEstimateRAM,
			Items:    limit,
		},
		Value:       1,
		BatchKey:    "annotations",
		MaxBatch:    1,
		MaxAttempts: 3,
	}
	if _, err := r.store.Submit(ctx, job); err != nil {
		r.log.Warn("annotation scheduler submit failed", "error", err.Error())
	}
}

// Status returns the scheduler's operator view.
func (r *Runner) Status(ctx context.Context) (schedule.Status, error) {
	if r == nil || r.scheduler == nil {
		return schedule.Status{}, errors.New("annotation scheduler is unavailable")
	}
	return r.scheduler.Status(ctx)
}

// RegisterRoutes mounts the scheduler status endpoint. It is separate from the
// annotation data routes so a missing scheduler does not hide the vocabulary.
func (r *Runner) RegisterRoutes(router *mux.Router) {
	if r == nil || router == nil {
		return
	}
	router.HandleFunc("/api/v1/annotations/scheduler", r.handleStatus).Methods(http.MethodGet)
}

func (r *Runner) handleStatus(w http.ResponseWriter, req *http.Request) {
	status, err := r.Status(req.Context())
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": "annotation scheduler is unavailable", "detail": err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// Stop cancels the producer and the scheduler loop. The store is closed by the
// caller's process lifetime; an explicit Close is provided for tests.
func (r *Runner) Stop() {
	if r != nil && r.cancel != nil {
		r.cancel()
	}
}

// Close stops the runner, waits briefly for the scheduler and producer to
// release the store, then closes it. Closing the store under a live scheduler
// would turn a clean shutdown into avoidable store errors.
func (r *Runner) Close() error {
	if r == nil {
		return nil
	}
	r.closeOnce.Do(func() {
		r.closeErr = r.close()
	})
	return r.closeErr
}

func (r *Runner) close() error {
	r.Stop()
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
	}
	var first error
	if r.store != nil {
		first = r.store.Close()
	}
	if r.broker != nil {
		if err := r.broker.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}
