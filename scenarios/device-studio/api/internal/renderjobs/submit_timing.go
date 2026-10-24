package renderjobs

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type submitTimingContextKey struct{}

// SubmitTiming is request-scoped. The fixed span set bounds both memory and
// the response header, and the mutex makes recording safe if an owner later
// moves one synchronous stage to a joined helper goroutine.
type SubmitTiming struct {
	mu      sync.Mutex
	started time.Time
	spans   [len(submitTimingNames)]time.Duration
	seen    [len(submitTimingNames)]bool
}

var submitTimingNames = [...]string{
	"route_pin", "coordinator_submit", "route", "idempotency", "shot_resolve",
	"worker_pin", "blender_profile", "closure", "jobs_submit", "pump_activate",
	"handler_total",
}

func NewSubmitTiming(ctx context.Context) (context.Context, *SubmitTiming) {
	t := &SubmitTiming{started: time.Now()}
	return context.WithValue(ctx, submitTimingContextKey{}, t), t
}

// RecordSubmitTiming records one of the small fixed set of named spans. Unknown
// names are ignored so caller-controlled labels can never expand the header.
func RecordSubmitTiming(ctx context.Context, name string, elapsed time.Duration) {
	t, _ := ctx.Value(submitTimingContextKey{}).(*SubmitTiming)
	if t == nil || elapsed < 0 {
		return
	}
	for i, known := range submitTimingNames {
		if name == known {
			t.mu.Lock()
			t.spans[i] = elapsed
			t.seen[i] = true
			t.mu.Unlock()
			return
		}
	}
}

func (t *SubmitTiming) HeaderValue() string {
	if t == nil {
		return ""
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.seen[len(submitTimingNames)-1] {
		t.spans[len(submitTimingNames)-1] = time.Since(t.started)
		t.seen[len(submitTimingNames)-1] = true
	}
	var parts []string
	for i, name := range submitTimingNames {
		if !t.seen[i] {
			continue
		}
		ms := float64(t.spans[i]) / float64(time.Millisecond)
		if !math.IsNaN(ms) && !math.IsInf(ms, 0) {
			parts = append(parts, name+";dur="+strconv.FormatFloat(ms, 'f', 1, 64))
		}
	}
	return strings.Join(parts, ", ")
}

// SetSubmitTimingHeader adds the bounded timing diagnostics to a successful
// Connect response. Callers must not invoke it for failed requests.
func SetSubmitTimingHeader(header http.Header, timing *SubmitTiming) {
	if header == nil || timing == nil {
		return
	}
	value := timing.HeaderValue()
	if value != "" && len(value) <= 1024 {
		header.Set("Server-Timing", value)
	}
}

func submitTimingSpan(ctx context.Context, name string, start time.Time) {
	if start.IsZero() {
		return
	}
	RecordSubmitTiming(ctx, name, time.Since(start))
}
