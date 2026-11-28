package vrooli

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	// Match the resource check's 60-second cadence. The start watermark
	// remains conservative, and post-action recovery always forces a fresh read.
	defaultSnapshotTTL          = 60 * time.Second
	defaultSnapshotStaleCeil    = 2 * time.Minute
	defaultSnapshotRetryBackoff = 5 * time.Second
	// Resource status is a bounded fleet operation. The control plane now
	// uses a worker pool, but a busy host can still need more than the generic
	// 30-second health-check budget. Resource checks have a matching 60-second
	// per-check timeout, leaving room for the fleet command to finish cleanly.
	defaultSnapshotCycleBudget = 45 * time.Second
)

// SnapshotProviderOption configures the bounded snapshot cache.
type SnapshotProviderOption func(*SnapshotProvider)

func WithSnapshotTTL(ttl time.Duration) SnapshotProviderOption {
	return func(p *SnapshotProvider) {
		if ttl > 0 {
			p.ttl = ttl
		}
	}
}

func WithSnapshotStaleCeiling(ceiling time.Duration) SnapshotProviderOption {
	return func(p *SnapshotProvider) {
		if ceiling > 0 {
			p.staleCeiling = ceiling
		}
	}
}

func WithSnapshotRetryBackoff(backoff time.Duration) SnapshotProviderOption {
	return func(p *SnapshotProvider) {
		if backoff > 0 {
			p.retryBackoff = backoff
		}
	}
}

func WithSnapshotCycleBudget(budget time.Duration) SnapshotProviderOption {
	return func(p *SnapshotProvider) {
		if budget > 0 {
			p.cycleBudget = budget
		}
	}
}

func WithSnapshotClock(now func() time.Time) SnapshotProviderOption {
	return func(p *SnapshotProvider) {
		if now != nil {
			p.now = now
		}
	}
}

// SnapshotProvider shares one fleet refresh across all resource checks. The
// cache is deliberately process-local: it is an optimization over the typed
// control-plane boundary, not a second source of resource truth.
type SnapshotProvider struct {
	client       *Client
	ttl          time.Duration
	staleCeiling time.Duration
	retryBackoff time.Duration
	cycleBudget  time.Duration
	now          func() time.Time

	mu         sync.Mutex
	snapshot   *ResourceStatusSnapshot
	nextRetry  time.Time
	inflight   chan struct{}
	deep       map[string]deepStatusResult
	deepFlight map[string]chan struct{}
	metrics    SnapshotMetrics
}

type deepStatusResult struct {
	status   ResourceStatus
	err      error
	observed time.Time
}

func NewSnapshotProvider(client *Client, opts ...SnapshotProviderOption) *SnapshotProvider {
	if client == nil {
		client = NewClient(nil)
	}
	p := &SnapshotProvider{
		client:       client,
		ttl:          defaultSnapshotTTL,
		staleCeiling: defaultSnapshotStaleCeil,
		retryBackoff: defaultSnapshotRetryBackoff,
		cycleBudget:  defaultSnapshotCycleBudget,
		now:          time.Now,
		deep:         make(map[string]deepStatusResult),
		deepFlight:   make(map[string]chan struct{}),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(p)
		}
	}
	return p
}

// Snapshot returns only requested resources. An empty name list returns the
// entire typed fleet response; Autoheal normally supplies one supervised name.
func (p *SnapshotProvider) Snapshot(ctx context.Context, names []string) (ResourceStatusSnapshot, error) {
	requested := normalizedNames(names)
	p.mu.Lock()
	p.metrics.SnapshotRequests++
	p.mu.Unlock()
	for {
		now := p.now()
		p.mu.Lock()
		if p.snapshot != nil && now.Before(p.snapshot.ExpiresAt) {
			snapshot := selectSnapshot(*p.snapshot, requested)
			p.mu.Unlock()
			return snapshot, nil
		}
		if p.inflight != nil {
			wait := p.inflight
			p.metrics.CoalescedCalls++
			p.mu.Unlock()
			select {
			case <-wait:
				continue
			case <-ctx.Done():
				return undeterminedSnapshot(now, ctx.Err()), ctx.Err()
			}
		}
		if now.Before(p.nextRetry) {
			snapshot := fallbackSnapshot(now, p.snapshot, "refresh backoff active", p.staleCeiling)
			p.metrics.StaleReads++
			p.mu.Unlock()
			return selectSnapshot(snapshot, requested), fmt.Errorf("resource status snapshot refresh backoff until %s", p.nextRetry.UTC().Format(time.RFC3339))
		}
		p.inflight = make(chan struct{})
		wait := p.inflight
		previous := cloneSnapshot(p.snapshot)
		p.mu.Unlock()

		snapshot, err := p.refresh(ctx, previous)
		p.mu.Lock()
		if err == nil && !p.now().Before(snapshot.ExpiresAt) {
			// Observations can be made anywhere during a fleet scan. If even
			// the conservative start watermark is already too old, do not
			// let every coalesced waiter immediately launch another scan or
			// pretend the old observation was made at completion.
			err = fmt.Errorf("resource status snapshot expired before refresh completed")
			snapshot = fallbackSnapshot(p.now(), &snapshot, err.Error(), p.staleCeiling)
		}
		if err != nil {
			p.nextRetry = p.now().Add(p.retryBackoff)
			snapshot.NextRefreshAt = p.nextRetry
			p.metrics.RefreshFailures++
		} else {
			p.nextRetry = time.Time{}
		}
		p.snapshot = &snapshot
		close(wait)
		p.inflight = nil
		p.mu.Unlock()
		return selectSnapshot(snapshot, requested), err
	}
}

func (p *SnapshotProvider) refresh(ctx context.Context, previous *ResourceStatusSnapshot) (ResourceStatusSnapshot, error) {
	started := p.now()
	wallStarted := time.Now()
	p.mu.Lock()
	p.metrics.FleetRefreshes++
	p.mu.Unlock()
	refreshCtx, cancel := context.WithTimeout(ctx, p.cycleBudget)
	defer cancel()
	statuses, success, discoveryFailures, _, err := p.client.ResourceStatuses(refreshCtx)
	if err != nil {
		p.mu.Lock()
		p.metrics.RefreshLatency += time.Since(wallStarted)
		p.mu.Unlock()
		return fallbackSnapshot(started, previous, boundedError(err), p.staleCeiling), err
	}

	resources := make(map[string]ResourceStatus, len(statuses))
	for _, status := range statuses {
		if status.Name != "" {
			resources[status.Name] = status
		}
	}
	complete := success && discoveryFailures == 0
	refreshError := ""
	if !success {
		refreshError = "fleet status response was unsuccessful"
	} else if discoveryFailures > 0 {
		refreshError = fmt.Sprintf("fleet status omitted %d resource(s)", discoveryFailures)
	}
	p.mu.Lock()
	p.metrics.RefreshLatency += time.Since(wallStarted)
	if !complete {
		p.metrics.IncompleteResponses++
	}
	p.mu.Unlock()
	return ResourceStatusSnapshot{
		ObservedAt:    started,
		ExpiresAt:     started.Add(p.ttl),
		Complete:      complete,
		Source:        SnapshotSourceFleet,
		ProbeLevel:    ProbeLevelFast,
		Resources:     resources,
		RefreshError:  refreshError,
		NextRefreshAt: started.Add(p.ttl),
	}, nil
}

func (p *SnapshotProvider) DeepStatus(ctx context.Context, name string) (ResourceStatus, error) {
	return p.deepStatus(ctx, name, false)
}

// ForceDeepStatus bypasses the named-status cache for recovery verification or
// another explicit safety-sensitive read. Concurrent forced readers coalesce
// onto the in-flight request that produced the fresh result.
func (p *SnapshotProvider) ForceDeepStatus(ctx context.Context, name string) (ResourceStatus, error) {
	return p.deepStatus(ctx, name, true)
}

func (p *SnapshotProvider) deepStatus(ctx context.Context, name string, force bool) (ResourceStatus, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return ResourceStatus{}, fmt.Errorf("resource status: name is required")
	}
	for {
		now := p.now()
		p.mu.Lock()
		if cached, ok := p.deep[name]; ok && (!force && now.Sub(cached.observed) < p.ttl) {
			status, err := cloneResourceStatus(cached.status), cached.err
			p.mu.Unlock()
			return status, err
		}
		if wait, ok := p.deepFlight[name]; ok {
			waitStarted := now
			p.mu.Unlock()
			select {
			case <-wait:
				if force {
					p.mu.Lock()
					cached, ok := p.deep[name]
					if ok && !cached.observed.Before(waitStarted) {
						status, err := cloneResourceStatus(cached.status), cached.err
						p.mu.Unlock()
						return status, err
					}
					p.mu.Unlock()
				}
				continue
			case <-ctx.Done():
				return ResourceStatus{}, ctx.Err()
			}
		}
		flight := make(chan struct{})
		p.deepFlight[name] = flight
		p.metrics.NamedDeepChecks++
		p.mu.Unlock()

		started := time.Now()
		status, _, err := p.client.ResourceStatus(ctx, name)
		p.mu.Lock()
		p.deep[name] = deepStatusResult{status: cloneResourceStatus(status), err: err, observed: p.now()}
		p.metrics.DeepLatency += time.Since(started)
		close(flight)
		delete(p.deepFlight, name)
		p.mu.Unlock()
		return status, err
	}
}

func (p *SnapshotProvider) SnapshotMetrics() SnapshotMetrics {
	p.mu.Lock()
	defer p.mu.Unlock()
	metrics := p.metrics
	metrics.EscalationReasons = make(map[string]uint64, len(p.metrics.EscalationReasons))
	for reason, count := range p.metrics.EscalationReasons {
		metrics.EscalationReasons[reason] = count
	}
	return metrics
}

func (p *SnapshotProvider) RecordUndetermined() {
	p.mu.Lock()
	p.metrics.UndeterminedResults++
	p.mu.Unlock()
}

func (p *SnapshotProvider) RecordEscalation(reason string) {
	if !validEscalationReason(reason) {
		return
	}
	p.mu.Lock()
	if p.metrics.EscalationReasons == nil {
		p.metrics.EscalationReasons = make(map[string]uint64)
	}
	p.metrics.EscalationReasons[reason]++
	p.mu.Unlock()
}

func validEscalationReason(reason string) bool {
	switch reason {
	case "anomaly", "recovery", "mode_drift", "reacquire", "explicit_force", "missing_semantics", "operator_diagnostic":
		return true
	default:
		return false
	}
}

func (p *SnapshotProvider) SnapshotNow() time.Time { return p.now() }

func normalizedNames(names []string) []string {
	set := make(map[string]struct{}, len(names))
	for _, name := range names {
		if name = strings.TrimSpace(name); name != "" {
			set[name] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for name := range set {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func selectSnapshot(snapshot ResourceStatusSnapshot, names []string) ResourceStatusSnapshot {
	selected := cloneSnapshot(&snapshot)
	if len(names) == 0 {
		return *selected
	}
	selected.Resources = make(map[string]ResourceStatus, len(names))
	for _, name := range names {
		if status, ok := snapshot.Resources[name]; ok {
			selected.Resources[name] = cloneResourceStatus(status)
		}
	}
	return *selected
}

func cloneSnapshot(snapshot *ResourceStatusSnapshot) *ResourceStatusSnapshot {
	if snapshot == nil {
		return &ResourceStatusSnapshot{Resources: map[string]ResourceStatus{}}
	}
	copyOf := *snapshot
	copyOf.Resources = make(map[string]ResourceStatus, len(snapshot.Resources))
	for name, status := range snapshot.Resources {
		copyOf.Resources[name] = cloneResourceStatus(status)
	}
	return &copyOf
}

func cloneResourceStatus(status ResourceStatus) ResourceStatus {
	copyOf := status
	if status.Healthy != nil {
		value := *status.Healthy
		copyOf.Healthy = &value
	}
	if status.Serving != nil {
		value := *status.Serving
		copyOf.Serving = &value
	}
	return copyOf
}

func fallbackSnapshot(now time.Time, previous *ResourceStatusSnapshot, reason string, staleCeiling time.Duration) ResourceStatusSnapshot {
	if previous == nil {
		return undeterminedSnapshot(now, errorsText(reason))
	}
	snapshot := *cloneSnapshot(previous)
	snapshot.Source = SnapshotSourceFallback
	snapshot.ProbeLevel = ProbeLevelUndetermined
	snapshot.Complete = false
	snapshot.RefreshError = boundedString(reason)
	if staleCeiling > 0 {
		snapshot.StaleUntil = snapshot.ObservedAt.Add(staleCeiling)
	}
	return snapshot
}

func undeterminedSnapshot(now time.Time, reason error) ResourceStatusSnapshot {
	message := "status snapshot unavailable"
	if reason != nil {
		message = boundedError(reason)
	}
	return ResourceStatusSnapshot{
		ObservedAt:    now,
		ExpiresAt:     now,
		Complete:      false,
		Source:        SnapshotSourceFallback,
		ProbeLevel:    ProbeLevelUndetermined,
		Resources:     map[string]ResourceStatus{},
		RefreshError:  message,
		NextRefreshAt: now,
	}
}

func errorsText(message string) error {
	return fmt.Errorf("%s", message)
}

func boundedError(err error) string {
	if err == nil {
		return ""
	}
	return boundedString(err.Error())
}

func boundedString(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 512 {
		return value[:512]
	}
	return value
}
