// Package sessions owns the lifetime and governance state of a program
// kernel. Persistent lease metadata is stored through Repository; the live
// process handle remains in this manager because operating-system handles are
// intentionally not serialized.
package sessions

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound      = errors.New("session not found")
	ErrReclaimed     = errors.New("session reclaimed")
	ErrLimitExceeded = errors.New("session limit exceeded")
	ErrKernelStart   = errors.New("kernel failed to start")
)

const (
	defaultWallBudget = 4 * time.Hour
	defaultCPUBudget  = 4 * time.Hour
)

type ExecutionBudget struct {
	WallBudget   time.Duration
	WallConsumed time.Duration
	CPUBudget    time.Duration
	CPUConsumed  time.Duration
}

type ExecutionBudgetExceededError struct {
	Kind     string
	Ceiling  time.Duration
	Consumed time.Duration
}

func (e *ExecutionBudgetExceededError) Error() string {
	return fmt.Sprintf("%s budget exhausted: ceiling=%s consumed=%s", e.Kind, e.Ceiling, e.Consumed)
}

// SpendExceededError identifies the ceiling that prevented a new metered
// operation from starting.
type SpendExceededError struct {
	Kind        string
	Ceiling     int64
	Accumulated int64
}

func (e *SpendExceededError) Error() string {
	return fmt.Sprintf("%s_spend_exceeded: ceiling=%d micros accumulated=%d micros", e.Kind, e.Ceiling, e.Accumulated)
}

type (
	Clock       interface{ Now() time.Time }
	systemClock struct{}
)

func (systemClock) Now() time.Time { return time.Now().UTC() }

type Options struct {
	Store                   SQLExecutor
	IdleTimeout             time.Duration
	WallTimeout             time.Duration
	WallBudget              time.Duration
	CPUBudget               time.Duration
	MemoryLimit             int64
	MaxSessions             int
	InferenceCeilingMicros  int64
	DelegationCeilingMicros int64
	Clock                   Clock
	KernelFactory           KernelFactory
	OnReclaimed             func(string)
	WorkspaceResolver       WorkspaceResolver
	OnWorkspaceResolved     func(string, string)
}

type (
	Kernel        interface{ Close() error }
	KernelFactory func(sessionID string) (Kernel, error)
)

type Session struct {
	ID                      string
	Name                    string
	State                   string
	CreatedAt               time.Time
	LastActivityAt          time.Time
	Grants                  map[string]struct{}
	SandboxWorkspace        string
	ReclaimedReason         string
	MemoryBytes             int64
	InferenceCostMicros     int64
	InferenceTokens         int64
	DelegationCostMicros    int64
	InferenceCeilingMicros  int64
	DelegationCeilingMicros int64
	DelegationSpendMeasured bool
	DelegationSpendNote     string
	WallBudget              time.Duration
	WallConsumed            time.Duration
	CPUBudget               time.Duration
	CPUConsumed             time.Duration
	Kernel                  Kernel
}

type Manager struct {
	mu      sync.Mutex
	options Options
	repo    Repository
	kernels map[string]Kernel
}

func NewManager(options Options) *Manager {
	if options.Clock == nil {
		options.Clock = systemClock{}
	}
	if options.IdleTimeout <= 0 {
		options.IdleTimeout = 30 * time.Minute
	}
	if options.WallTimeout <= 0 {
		options.WallTimeout = 4 * time.Hour
	}
	if options.WallBudget <= 0 {
		options.WallBudget = defaultWallBudget
	}
	if options.CPUBudget <= 0 {
		options.CPUBudget = defaultCPUBudget
	}
	repo := Repository(newMemoryRepository())
	if options.Store != nil {
		repo = NewRepository(options.Store)
	}
	if options.WorkspaceResolver == nil {
		options.WorkspaceResolver = &localWorkspaceResolver{}
	}
	return &Manager{options: options, repo: repo, kernels: make(map[string]Kernel)}
}

func (m *Manager) Create(ctx context.Context, name, sandbox string, grants []string) (*Session, error) {
	return m.CreateWithBudgets(ctx, name, sandbox, grants, 0, 0)
}

func (m *Manager) CreateWithBudgets(ctx context.Context, name, sandbox string, grants []string, inferenceCeilingMicros, delegationCeilingMicros int64) (*Session, error) {
	return m.CreateWithExecutionBudgets(ctx, name, sandbox, grants, inferenceCeilingMicros, delegationCeilingMicros, 0, 0)
}

func (m *Manager) CreateWithExecutionBudgets(ctx context.Context, name, sandbox string, grants []string, inferenceCeilingMicros, delegationCeilingMicros, wallBudgetMillis, cpuBudgetMillis int64) (*Session, error) {
	if m.options.MaxSessions > 0 {
		current, err := m.repo.List(ctx)
		if err != nil {
			return nil, err
		}
		if len(current) >= m.options.MaxSessions {
			return nil, ErrLimitExceeded
		}
	}
	id := "sess_" + uuid.NewString()
	now := m.options.Clock.Now().UTC()
	workspace := strings.TrimSpace(sandbox)
	if workspace != "" {
		resolved, err := m.options.WorkspaceResolver.Resolve(ctx, workspace)
		if err != nil {
			return nil, fmt.Errorf("%w %q: %v", ErrInvalidWorkspace, workspace, err)
		}
		workspace = resolved
	}
	if inferenceCeilingMicros <= 0 {
		inferenceCeilingMicros = m.options.InferenceCeilingMicros
	}
	if delegationCeilingMicros <= 0 {
		delegationCeilingMicros = m.options.DelegationCeilingMicros
	}
	wallBudget := m.options.WallBudget
	if wallBudgetMillis > 0 {
		wallBudget = time.Duration(wallBudgetMillis) * time.Millisecond
	}
	cpuBudget := m.options.CPUBudget
	if cpuBudgetMillis > 0 {
		cpuBudget = time.Duration(cpuBudgetMillis) * time.Millisecond
	}
	s := &Session{ID: id, Name: strings.TrimSpace(name), State: "running", CreatedAt: now, LastActivityAt: now, Grants: grantSet(grants), SandboxWorkspace: workspace, InferenceCeilingMicros: inferenceCeilingMicros, DelegationCeilingMicros: delegationCeilingMicros, WallBudget: wallBudget, CPUBudget: cpuBudget}
	if m.options.KernelFactory != nil {
		kernel, err := m.options.KernelFactory(id)
		if err != nil {
			return nil, fmt.Errorf("%w for %s: %v", ErrKernelStart, id, err)
		}
		m.mu.Lock()
		m.kernels[id] = kernel
		m.mu.Unlock()
	}
	if err := m.repo.Create(ctx, s); err != nil {
		m.closeKernel(id)
		return nil, err
	}
	if workspace != "" && m.options.OnWorkspaceResolved != nil {
		m.options.OnWorkspaceResolved(id, workspace)
	}
	return clone(s), nil
}

func (m *Manager) EnsureInferenceAvailable(ctx context.Context, id string) error {
	s, err := m.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if s.InferenceCeilingMicros > 0 && s.InferenceCostMicros >= s.InferenceCeilingMicros {
		return &SpendExceededError{Kind: "inference", Ceiling: s.InferenceCeilingMicros, Accumulated: s.InferenceCostMicros}
	}
	return nil
}

func (m *Manager) RecordInferenceUsage(ctx context.Context, id string, costMicros, tokens int64) error {
	return m.repo.RecordInferenceUsage(ctx, id, costMicros, tokens)
}

func (m *Manager) RecordDelegationUsage(ctx context.Context, id string, costMicros int64, measured bool, note string) error {
	return m.repo.RecordDelegationUsage(ctx, id, costMicros, measured, note)
}

func (m *Manager) SaveDelegation(ctx context.Context, delegation *Delegation) error {
	if delegation == nil || delegation.SessionID == "" || delegation.ExecutionID == "" {
		return errors.New("session and execution identifiers are required")
	}
	if _, err := m.Get(ctx, delegation.SessionID); err != nil {
		return err
	}
	return m.repo.SaveDelegation(ctx, delegation)
}

func (m *Manager) GetDelegation(ctx context.Context, sessionID, executionID string) (*Delegation, error) {
	if strings.TrimSpace(sessionID) == "" || strings.TrimSpace(executionID) == "" {
		return nil, errors.New("session_id and execution_id are required")
	}
	return m.repo.GetDelegation(ctx, sessionID, executionID)
}

// CountDelegations returns the retained delegation records used by the
// session_delegations measure. It is a direct projection of durable state.
func (m *Manager) CountDelegations(ctx context.Context) int {
	count, err := m.repo.CountDelegations(ctx)
	if err != nil {
		return 0
	}
	return count
}

func (m *Manager) ListDelegations(ctx context.Context) ([]*Delegation, error) {
	return m.repo.ListDelegations(ctx)
}

func (m *Manager) Get(ctx context.Context, id string) (*Session, error) {
	s, err := m.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	s.Kernel = m.kernels[id]
	m.mu.Unlock()
	if s.SandboxWorkspace != "" && m.options.OnWorkspaceResolved != nil {
		m.options.OnWorkspaceResolved(id, s.SandboxWorkspace)
	}
	return clone(s), nil
}

func (m *Manager) List(ctx context.Context) []*Session {
	out, err := m.repo.List(ctx)
	if err != nil {
		return nil
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

func (m *Manager) Delete(ctx context.Context, id, reason string) (*Session, error) {
	s, err := m.repo.Delete(ctx, id, strings.TrimSpace(reason), m.options.Clock.Now().UTC())
	if err != nil {
		return nil, err
	}
	m.closeKernel(id)
	return s, nil
}

func (m *Manager) Grant(ctx context.Context, id string, grants []string) (*Session, error) {
	s, err := m.repo.Grant(ctx, id, normalizeGrants(grants), m.options.Clock.Now().UTC())
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (m *Manager) HasGrant(ctx context.Context, id, grant string) bool {
	ok, err := m.repo.HasGrant(ctx, id, grant)
	return err == nil && ok
}

func (m *Manager) Touch(ctx context.Context, id string) error {
	return m.repo.Touch(ctx, id, m.options.Clock.Now().UTC())
}

func (m *Manager) ExecutionBudget(ctx context.Context, id string) (ExecutionBudget, error) {
	s, err := m.repo.Get(ctx, id)
	if err != nil {
		return ExecutionBudget{}, err
	}
	if s.WallBudget > 0 && s.WallConsumed >= s.WallBudget {
		return ExecutionBudget{}, &ExecutionBudgetExceededError{Kind: "wall-clock", Ceiling: s.WallBudget, Consumed: s.WallConsumed}
	}
	if s.CPUBudget > 0 && s.CPUConsumed >= s.CPUBudget {
		return ExecutionBudget{}, &ExecutionBudgetExceededError{Kind: "cpu", Ceiling: s.CPUBudget, Consumed: s.CPUConsumed}
	}
	return ExecutionBudget{WallBudget: s.WallBudget, WallConsumed: s.WallConsumed, CPUBudget: s.CPUBudget, CPUConsumed: s.CPUConsumed}, nil
}

func (m *Manager) ChargeExecution(ctx context.Context, id string, wall, cpu time.Duration) error {
	if wall < 0 {
		wall = 0
	}
	if cpu < 0 {
		cpu = 0
	}
	if err := m.repo.RecordExecutionUsage(ctx, id, wall, cpu, m.options.Clock.Now().UTC()); err != nil {
		return err
	}
	s, err := m.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if s.WallBudget > 0 && s.WallConsumed >= s.WallBudget {
		reason := (&ExecutionBudgetExceededError{Kind: "wall-clock", Ceiling: s.WallBudget, Consumed: s.WallConsumed}).Error()
		_ = m.reclaim(ctx, id, reason)
		return errors.New(reason)
	}
	if s.CPUBudget > 0 && s.CPUConsumed >= s.CPUBudget {
		reason := (&ExecutionBudgetExceededError{Kind: "cpu", Ceiling: s.CPUBudget, Consumed: s.CPUConsumed}).Error()
		_ = m.reclaim(ctx, id, reason)
		return errors.New(reason)
	}
	return nil
}

func (m *Manager) SetMemoryBytes(ctx context.Context, id string, bytes int64) error {
	if err := m.repo.SetMemoryBytes(ctx, id, bytes); err != nil {
		return err
	}
	if m.options.MemoryLimit > 0 && bytes > m.options.MemoryLimit {
		return m.reclaim(ctx, id, "memory ceiling exceeded")
	}
	return nil
}

func (m *Manager) ReclaimIdle(ctx context.Context, now time.Time) []string {
	sessions, err := m.repo.List(ctx)
	if err != nil {
		return nil
	}
	var reclaimed []string
	for _, s := range sessions {
		reason := ""
		if now.Sub(s.LastActivityAt) >= m.options.IdleTimeout {
			reason = "idle timeout exceeded"
		}
		if now.Sub(s.CreatedAt) >= m.options.WallTimeout {
			reason = "wall-clock ceiling exceeded"
		}
		if m.options.MemoryLimit > 0 && s.MemoryBytes > m.options.MemoryLimit {
			reason = "memory ceiling exceeded"
		}
		if reason != "" {
			if err := m.reclaim(ctx, s.ID, reason); errors.Is(err, ErrReclaimed) {
				reclaimed = append(reclaimed, s.ID)
			}
		}
	}
	sort.Strings(reclaimed)
	return reclaimed
}

func (m *Manager) reclaim(ctx context.Context, id, reason string) error {
	if err := m.repo.Reclaim(ctx, id, reason, m.options.Clock.Now().UTC()); err != nil {
		return err
	}
	m.closeKernel(id)
	if m.options.OnReclaimed != nil {
		m.options.OnReclaimed(id)
	}
	return ErrReclaimed
}

func (m *Manager) closeKernel(id string) {
	m.mu.Lock()
	kernel := m.kernels[id]
	delete(m.kernels, id)
	m.mu.Unlock()
	if kernel != nil {
		_ = kernel.Close()
	}
}

func normalizeGrants(grants []string) []string {
	set := grantSet(grants)
	out := make([]string, 0, len(set))
	for grant := range set {
		out = append(out, grant)
	}
	sort.Strings(out)
	return out
}

func grantSet(grants []string) map[string]struct{} {
	out := make(map[string]struct{}, len(grants))
	for _, g := range grants {
		if g = strings.TrimSpace(g); g != "" {
			out[g] = struct{}{}
		}
	}
	return out
}

func clone(s *Session) *Session {
	c := *s
	c.Grants = grantSet(nil)
	for g := range s.Grants {
		c.Grants[g] = struct{}{}
	}
	c.Kernel = nil
	return &c
}
