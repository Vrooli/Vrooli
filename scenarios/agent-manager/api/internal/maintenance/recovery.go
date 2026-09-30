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
type Recovery struct {
	mu     sync.RWMutex
	state  RecoveryState
	once   sync.Once
	cancel context.CancelFunc
	done   chan struct{}
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
		ctx, cancel := context.WithCancel(parent)
		r.mu.Lock()
		r.cancel = cancel
		r.mu.Unlock()
		go func() {
			defer close(r.done)
			criticalFailure := false
			for _, step := range steps {
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
				if err != nil {
					criticalFailure = criticalFailure || !step.NonCritical
					r.mu.Lock()
					r.state.Failures = append(r.state.Failures, step.Name+": "+err.Error())
					r.mu.Unlock()
				}
			}
			r.mu.Lock()
			defer r.mu.Unlock()
			r.state.Phase = ""
			switch {
			case ctx.Err() != nil:
				r.state.Status = "stopping"
				r.state.Readiness = false
			case len(r.state.Failures) > 0:
				r.state.Status = "degraded"
				r.state.Readiness = !criticalFailure
			default:
				r.state.Status = "ready"
				r.state.Readiness = true
			}
		}()
	})
}

// Stop cancels and joins startup before its storage is closed. Long-lived workers
// still use the existing owners' shutdown methods after this join.
func (r *Recovery) Stop() {
	r.mu.Lock()
	cancel := r.cancel
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
