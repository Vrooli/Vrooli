package maintenance

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/health"
)

type RecoveryStep struct {
	Name    string
	Timeout time.Duration
	Run     func(context.Context) error
	// NonCritical is limited to rebuildable historical projections. Required
	// ownership recovery remains readiness-blocking by default.
	NonCritical bool
}

type RecoveryState struct {
	Status    string   `json:"status"`
	Readiness bool     `json:"readiness"`
	Phase     string   `json:"phase,omitempty"`
	Failures  []string `json:"failures,omitempty"`
}

// Recovery owns only startup sequencing and its observable readiness. Historical
// accounting stays with its existing owner and is never deleted or zero-filled.
type recoveryStepResult struct {
	name        string
	nonCritical bool
	failure     string
}

type Recovery struct {
	mu              sync.RWMutex
	state           RecoveryState
	once            sync.Once
	cancel          context.CancelFunc
	ctx             context.Context
	done            chan struct{}
	steps           []recoveryStepResult
	startupFinished bool
	stopped         bool
}

func NewRecovery() *Recovery {
	return &Recovery{state: RecoveryState{Status: "initializing"}, done: make(chan struct{})}
}

func (r *Recovery) Snapshot() RecoveryState {
	r.mu.RLock()
	defer r.mu.RUnlock()
	state := r.state
	state.Failures = append([]string(nil), state.Failures...)
	return state
}

func (r *Recovery) Done() <-chan struct{} { return r.done }

func (r *Recovery) Start(parent context.Context, steps []RecoveryStep) {
	r.once.Do(func() {
		r.mu.Lock()
		if r.stopped {
			r.mu.Unlock()
			close(r.done)
			return
		}
		// Install cancellation state before releasing the lock. Stop either
		// marks the recovery stopped first, or sees this cancel function and
		// joins the worker after cancelling it.
		ctx, cancel := context.WithCancel(parent)
		r.cancel = cancel
		r.ctx = ctx
		r.steps = make([]recoveryStepResult, len(steps))
		for i, step := range steps {
			r.steps[i] = recoveryStepResult{name: step.Name, nonCritical: step.NonCritical}
		}
		r.mu.Unlock()
		go func() {
			defer close(r.done)
			for i, step := range steps {
				if ctx.Err() != nil {
					break
				}
				r.mu.Lock()
				r.state.Phase = step.Name
				r.mu.Unlock()
				stepCtx := ctx
				stepCancel := func() {}
				if step.Timeout > 0 {
					stepCtx, stepCancel = context.WithTimeout(ctx, step.Timeout)
				}
				err := step.Run(stepCtx)
				if err == nil {
					err = stepCtx.Err()
				}
				stepCancel()
				r.mu.Lock()
				if err != nil {
					r.steps[i].failure = err.Error()
				}
				r.mu.Unlock()
			}
			r.mu.Lock()
			defer r.mu.Unlock()
			r.state.Phase = ""
			if r.stopped || ctx.Err() != nil {
				r.stopped = true
				r.state.Status = "stopping"
				r.state.Readiness = false
				return
			}
			r.startupFinished = true
			r.refreshStateLocked()
		}()
	})
}

// ReconcileDeclarationStep clears a retained startup failure after its
// declaration retry owner succeeds. Retry errors are logged by that owner and
// do not add to or rewrite startup recovery state. Only these two steps have
// deferred retry owners; updates are ignored during startup and after shutdown.
func (r *Recovery) ReconcileDeclarationStep(name string, err error) bool {
	if name != "self_declarations" && name != "scenario_declarations" {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.startupFinished || r.stopped {
		return false
	}
	if r.ctx != nil && r.ctx.Err() != nil {
		r.stopped = true
		r.state.Status = "stopping"
		r.state.Readiness = false
		return false
	}
	matched := -1
	for i := range r.steps {
		if r.steps[i].name != name {
			continue
		}
		if matched >= 0 {
			return false // An ambiguous step name must not clear multiple results.
		}
		matched = i
	}
	if matched < 0 {
		return false
	}
	if err != nil || r.steps[matched].failure == "" {
		return true
	}
	r.steps[matched].failure = ""
	r.refreshStateLocked()
	return true
}

// refreshStateLocked derives the public state from per-step outcomes while
// preserving each step's original criticality and failure identity.
func (r *Recovery) refreshStateLocked() {
	r.state.Failures = r.state.Failures[:0]
	criticalFailure := false
	for _, step := range r.steps {
		if step.failure == "" {
			continue
		}
		r.state.Failures = append(r.state.Failures, step.name+": "+step.failure)
		criticalFailure = criticalFailure || !step.nonCritical
	}
	if len(r.state.Failures) > 0 {
		r.state.Status = "degraded"
		r.state.Readiness = !criticalFailure
		return
	}
	r.state.Status = "ready"
	r.state.Readiness = true
}

// Stop cancels and joins startup before its storage is closed. Long-lived workers
// still use the existing owners' shutdown methods after this join.
func (r *Recovery) Stop() {
	r.mu.Lock()
	cancel := r.cancel
	r.stopped = true
	r.state.Status = "stopping"
	r.state.Readiness = false
	r.mu.Unlock()
	if cancel != nil {
		cancel()
		<-r.done
	}
}

// Health is the lifecycle liveness surface. Initializing/degraded recovery must
// not provoke a restart loop: HTTP succeeds, but readiness remains false. Keep
// the shared health schema and build identity for the control-plane probe.
// Once ready, the existing dependency checks apply.
func (r *Recovery) Health(next http.Handler) http.Handler {
	return r.health(next, false, false)
}

// Readiness is the strict data-plane readiness surface, distinct from liveness.
func (r *Recovery) Readiness(next http.Handler) http.Handler {
	return r.health(next, false, true)
}

// ConnectHealth retains Connect's error envelope for the CLI/RPC health surface.
func (r *Recovery) ConnectHealth(next http.Handler) http.Handler {
	return r.health(next, true, true)
}

func (r *Recovery) health(next http.Handler, rpc, strict bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		state := r.Snapshot()
		if state.Readiness {
			next.ServeHTTP(w, req)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		if strict || state.Status == "stopping" {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		if rpc {
			_ = json.NewEncoder(w).Encode(map[string]string{"code": "unavailable", "message": "startup " + state.Status + ": " + state.Phase + " " + strings.Join(state.Failures, "; ")})
			return
		}
		_ = json.NewEncoder(w).Encode(struct {
			health.Response
			Recovery RecoveryState `json:"recovery"`
		}{Response: health.Response{
			Status: health.StatusDegraded, Service: "agent-manager",
			Timestamp: time.Now().UTC().Format(time.RFC3339), Readiness: false,
			BuildIdentity: os.Getenv("VROOLI_BUILD_IDENTITY"),
			Functional:    &health.FunctionalStatus{Healthy: false, Reason: "startup " + state.Status + ": " + state.Phase + " " + strings.Join(state.Failures, "; ")},
		}, Recovery: state})
	})
}
