package rewrite

import "sync"

// MemoryPlanStore is the in-memory PlanStore impl wired in production.
// REQ-P1-002 calls for SQLite persistence as a follow-up; until then
// plans evaporate on API restart and the consumer must re-plan before
// applying.
type MemoryPlanStore struct {
	mu    sync.RWMutex
	plans map[memoryKey]Plan
}

type memoryKey struct {
	projectPath string
	id          PlanID
}

// NewMemoryPlanStore returns a ready-to-use store with zero entries.
func NewMemoryPlanStore() *MemoryPlanStore {
	return &MemoryPlanStore{plans: make(map[memoryKey]Plan)}
}

// Save stores plan under (plan.ProjectPath, plan.ID).
func (s *MemoryPlanStore) Save(plan Plan) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.plans[memoryKey{projectPath: plan.ProjectPath, id: plan.ID}] = plan
	return nil
}

// Get returns the plan or ErrPlanNotFound.
func (s *MemoryPlanStore) Get(projectPath string, id PlanID) (Plan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.plans[memoryKey{projectPath: projectPath, id: id}]
	if !ok {
		return Plan{}, ErrPlanNotFound
	}
	return p, nil
}

// Compile-time guarantee.
var _ PlanStore = (*MemoryPlanStore)(nil)
