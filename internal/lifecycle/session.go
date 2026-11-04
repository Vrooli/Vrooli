package lifecycle

import (
	"context"
	"slices"
	"sync"
)

// startSession owns the mutable state shared by one recursive start graph.
// Keeping the context, readiness set, freshness cache, and dependency stack in
// one synchronized object makes fan-out safe without exposing raw maps to
// lifecycle functions.
type startSession struct {
	state *sessionState
	stack []string
	ctx   context.Context
	// state owns the maps and mutex; stack is immutable for this recursive
	// branch and therefore safe to copy into a child session.
}

type sessionState struct {
	mu                         sync.Mutex
	readySet                   map[string]struct{}
	cache                      setupCheckCache
	dependencyLeaseIDs         map[string]struct{}
	dependencyLeaseConsumers   map[string]string
	priorDependencyLeaseIDs    map[string]map[string]struct{}
	stoppedDependencyConsumers map[string]struct{}
}

type setupCheckCache map[string]setupCheckResult

type setupCheckResult struct {
	needed  bool
	reasons []string
}

func newStartSession(ctx context.Context) *startSession {
	if ctx == nil {
		ctx = context.Background()
	}
	return &startSession{ctx: ctx, state: &sessionState{
		readySet:                   map[string]struct{}{},
		cache:                      setupCheckCache{},
		dependencyLeaseIDs:         map[string]struct{}{},
		dependencyLeaseConsumers:   map[string]string{},
		priorDependencyLeaseIDs:    map[string]map[string]struct{}{},
		stoppedDependencyConsumers: map[string]struct{}{},
	}}
}

func (s *startSession) recordDependencyLease(consumerID, leaseID string) {
	if s == nil || s.state == nil || leaseID == "" {
		return
	}
	s.state.mu.Lock()
	s.state.dependencyLeaseIDs[leaseID] = struct{}{}
	s.state.dependencyLeaseConsumers[leaseID] = consumerID
	s.state.mu.Unlock()
}

func (s *startSession) rememberPriorDependencyLeases(consumerID string, leaseIDs []string) {
	if s == nil || s.state == nil || consumerID == "" {
		return
	}
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	if _, captured := s.state.priorDependencyLeaseIDs[consumerID]; captured {
		return
	}
	prior := make(map[string]struct{}, len(leaseIDs))
	for _, leaseID := range leaseIDs {
		if leaseID != "" {
			prior[leaseID] = struct{}{}
		}
	}
	s.state.priorDependencyLeaseIDs[consumerID] = prior
}

func (s *startSession) markDependencyConsumerStopped(consumerID string) {
	if s == nil || s.state == nil || consumerID == "" {
		return
	}
	s.state.mu.Lock()
	s.state.stoppedDependencyConsumers[consumerID] = struct{}{}
	s.state.mu.Unlock()
}

func (s *startSession) shouldPreservePriorDependencyLease(consumerID, leaseID string) bool {
	if s == nil || s.state == nil || consumerID == "" || leaseID == "" {
		return false
	}
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	if _, stopped := s.state.stoppedDependencyConsumers[consumerID]; stopped {
		return false
	}
	_, prior := s.state.priorDependencyLeaseIDs[consumerID][leaseID]
	return prior
}

func (s *startSession) shouldPreserveRecordedDependencyLease(leaseID string) bool {
	if s == nil || s.state == nil || leaseID == "" {
		return false
	}
	s.state.mu.Lock()
	consumerID := s.state.dependencyLeaseConsumers[leaseID]
	s.state.mu.Unlock()
	return s.shouldPreservePriorDependencyLease(consumerID, leaseID)
}

func (s *startSession) dependencyLeaseSetForConsumer(consumerID string) map[string]struct{} {
	if s == nil || s.state == nil || consumerID == "" {
		return nil
	}
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	set := make(map[string]struct{})
	for leaseID := range s.state.dependencyLeaseIDs {
		if s.state.dependencyLeaseConsumers[leaseID] == consumerID {
			set[leaseID] = struct{}{}
		}
	}
	return set
}

func (s *startSession) dependencyLeases() []string {
	if s == nil || s.state == nil {
		return nil
	}
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	ids := make([]string, 0, len(s.state.dependencyLeaseIDs))
	for id := range s.state.dependencyLeaseIDs {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

func (s *startSession) dependencyLeaseSet() map[string]struct{} {
	if s == nil || s.state == nil {
		return nil
	}
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	set := make(map[string]struct{}, len(s.state.dependencyLeaseIDs))
	for id := range s.state.dependencyLeaseIDs {
		set[id] = struct{}{}
	}
	return set
}

func (s *startSession) forgetDependencyLeases(ids []string) {
	if s == nil || s.state == nil {
		return
	}
	s.state.mu.Lock()
	for _, id := range ids {
		delete(s.state.dependencyLeaseIDs, id)
		delete(s.state.dependencyLeaseConsumers, id)
	}
	s.state.mu.Unlock()
}

func (s *startSession) context() context.Context {
	if s == nil || s.ctx == nil {
		return context.Background()
	}
	return s.ctx
}

func (s *startSession) markReady(name string) {
	if s == nil || s.state == nil {
		return
	}
	s.state.mu.Lock()
	s.state.readySet[name] = struct{}{}
	s.state.mu.Unlock()
}

func (s *startSession) isReady(name string) bool {
	if s == nil || s.state == nil {
		return false
	}
	s.state.mu.Lock()
	_, ok := s.state.readySet[name]
	s.state.mu.Unlock()
	return ok
}

func (s *startSession) setupNeeded(item string, force bool, evaluate func() (bool, []string, error)) (bool, []string, error) {
	if s == nil || s.state == nil {
		return evaluate()
	}
	key := item + "|force=" + boolString(force)
	s.state.mu.Lock()
	if cached, ok := s.state.cache[key]; ok {
		s.state.mu.Unlock()
		return cached.needed, append([]string(nil), cached.reasons...), nil
	}
	s.state.mu.Unlock()
	needed, reasons, err := evaluate()
	if err != nil {
		return false, nil, err
	}
	s.state.mu.Lock()
	s.state.cache[key] = setupCheckResult{needed: needed, reasons: append([]string(nil), reasons...)}
	s.state.mu.Unlock()
	return needed, append([]string(nil), reasons...), nil
}

func boolString(value bool) string {
	if value {
		return phasesTrue
	}
	return "false"
}

func (s *startSession) contains(name string) bool {
	if s == nil || s.state == nil {
		return false
	}
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	return slices.Contains(s.stack, name)
}

func (s *startSession) childStack(name string) *startSession {
	if s == nil {
		return nil
	}
	stack := append([]string(nil), s.stack...)
	stack = append(stack, name)
	return &startSession{ctx: s.context(), state: s.state, stack: stack}
}

func (s *startSession) withContext(ctx context.Context) *startSession {
	if s == nil {
		return newStartSession(ctx)
	}
	if ctx == nil {
		ctx = s.context()
	}
	return &startSession{ctx: ctx, state: s.state, stack: append([]string(nil), s.stack...)}
}
