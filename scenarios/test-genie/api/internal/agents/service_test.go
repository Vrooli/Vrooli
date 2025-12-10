package agents

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// --- Mock implementations for seam testing ---

// MockProcessChecker simulates process checking for tests.
type MockProcessChecker struct {
	mu       sync.RWMutex
	alive    map[int]bool
	checkLog []int // Records which PIDs were checked
}

func NewMockProcessChecker() *MockProcessChecker {
	return &MockProcessChecker{
		alive: make(map[int]bool),
	}
}

func (m *MockProcessChecker) SetAlive(pid int, alive bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alive[pid] = alive
}

func (m *MockProcessChecker) IsAlive(pid int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkLog = append(m.checkLog, pid)
	return m.alive[pid]
}

func (m *MockProcessChecker) GetCheckLog() []int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]int, len(m.checkLog))
	copy(result, m.checkLog)
	return result
}

// MockEnvironmentProvider simulates environment access for tests.
type MockEnvironmentProvider struct {
	mu       sync.RWMutex
	env      map[string]string
	hostname string
}

func NewMockEnvironmentProvider() *MockEnvironmentProvider {
	return &MockEnvironmentProvider{
		env:      make(map[string]string),
		hostname: "test-host",
	}
}

func (m *MockEnvironmentProvider) SetEnv(key, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.env[key] = value
}

func (m *MockEnvironmentProvider) SetHostname(hostname string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hostname = hostname
}

func (m *MockEnvironmentProvider) Getenv(key string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.env[key]
}

func (m *MockEnvironmentProvider) Hostname() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.hostname == "" {
		return "", errors.New("hostname not set")
	}
	return m.hostname, nil
}

// MockTimeProvider provides deterministic time for tests.
type MockTimeProvider struct {
	mu   sync.RWMutex
	time time.Time
}

func NewMockTimeProvider(t time.Time) *MockTimeProvider {
	return &MockTimeProvider{time: t}
}

func (m *MockTimeProvider) Now() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.time
}

func (m *MockTimeProvider) Advance(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.time = m.time.Add(d)
}

func (m *MockTimeProvider) Set(t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.time = t
}

// MockAgentRepository provides an in-memory repository for testing.
type MockAgentRepository struct {
	mu         sync.RWMutex
	agents     map[string]*SpawnedAgent
	locks      map[string]*AgentScopeLock
	intents    map[string]*SpawnIntent
	sessions   map[int]*SpawnSession
	lockIDSeq  int
	sessionSeq int
}

func NewMockAgentRepository() *MockAgentRepository {
	return &MockAgentRepository{
		agents:   make(map[string]*SpawnedAgent),
		locks:    make(map[string]*AgentScopeLock),
		intents:  make(map[string]*SpawnIntent),
		sessions: make(map[int]*SpawnSession),
	}
}

func (m *MockAgentRepository) Create(ctx context.Context, input CreateAgentInput) (*SpawnedAgent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	agent := &SpawnedAgent{
		ID:          input.ID,
		Scenario:    input.Scenario,
		Scope:       input.Scope,
		Phases:      input.Phases,
		Model:       input.Model,
		Status:      AgentStatusPending,
		PromptHash:  input.PromptHash,
		PromptIndex: input.PromptIndex,
		PromptText:  input.PromptText,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	m.agents[agent.ID] = agent
	return agent, nil
}

func (m *MockAgentRepository) Get(ctx context.Context, id string) (*SpawnedAgent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if agent, ok := m.agents[id]; ok {
		return agent, nil
	}
	return nil, nil
}

func (m *MockAgentRepository) GetByIdempotencyKey(ctx context.Context, key string) (*SpawnedAgent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, agent := range m.agents {
		if agent.IdempotencyKey == key {
			return agent, nil
		}
	}
	return nil, nil
}

func (m *MockAgentRepository) Update(ctx context.Context, id string, input UpdateAgentInput) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	agent, ok := m.agents[id]
	if !ok {
		return errors.New("agent not found")
	}
	if input.Status != nil {
		agent.Status = *input.Status
	}
	if input.Output != nil {
		agent.Output = *input.Output
	}
	if input.Error != nil {
		agent.Error = *input.Error
	}
	if input.PID != nil {
		agent.PID = *input.PID
	}
	if input.Hostname != nil {
		agent.Hostname = *input.Hostname
	}
	if input.SessionID != nil {
		agent.SessionID = *input.SessionID
	}
	agent.UpdatedAt = time.Now()
	return nil
}

func (m *MockAgentRepository) List(ctx context.Context, opts AgentListOptions) ([]*SpawnedAgent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*SpawnedAgent
	for _, agent := range m.agents {
		if opts.ActiveOnly && !isActiveStatus(agent.Status) {
			continue
		}
		if opts.Scenario != "" && agent.Scenario != opts.Scenario {
			continue
		}
		result = append(result, agent)
		if opts.Limit > 0 && len(result) >= opts.Limit {
			break
		}
	}
	return result, nil
}

func (m *MockAgentRepository) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.agents, id)
	return nil
}

func (m *MockAgentRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var count int64
	for id, agent := range m.agents {
		if agent.CreatedAt.Before(cutoff) && !isActiveStatus(agent.Status) {
			delete(m.agents, id)
			count++
		}
	}
	return count, nil
}

func (m *MockAgentRepository) AcquireLocks(ctx context.Context, agentID, scenario string, paths []string, expiresAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, path := range paths {
		key := scenario + ":" + path
		if existing, ok := m.locks[key]; ok && existing.AgentID != agentID {
			return errors.New("lock already held")
		}
		m.lockIDSeq++
		m.locks[key] = &AgentScopeLock{
			ID:         m.lockIDSeq,
			AgentID:    agentID,
			Scenario:   scenario,
			Path:       path,
			AcquiredAt: time.Now(),
			ExpiresAt:  expiresAt,
		}
	}
	return nil
}

func (m *MockAgentRepository) ReleaseLocks(ctx context.Context, agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for key, lock := range m.locks {
		if lock.AgentID == agentID {
			delete(m.locks, key)
		}
	}
	return nil
}

func (m *MockAgentRepository) RenewLocks(ctx context.Context, agentID string, newExpiry time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, lock := range m.locks {
		if lock.AgentID == agentID {
			lock.ExpiresAt = newExpiry
		}
	}
	return nil
}

func (m *MockAgentRepository) GetActiveLocks(ctx context.Context) ([]*AgentScopeLock, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*AgentScopeLock
	for _, lock := range m.locks {
		result = append(result, lock)
	}
	return result, nil
}

func (m *MockAgentRepository) GetLocksForScenario(ctx context.Context, scenario string) ([]*AgentScopeLock, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*AgentScopeLock
	for _, lock := range m.locks {
		if lock.Scenario == scenario {
			result = append(result, lock)
		}
	}
	return result, nil
}

func (m *MockAgentRepository) CheckConflicts(ctx context.Context, scenario string, paths []string) ([]ConflictDetail, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var conflicts []ConflictDetail
	for _, path := range paths {
		key := scenario + ":" + path
		if lock, ok := m.locks[key]; ok {
			conflicts = append(conflicts, ConflictDetail{
				Path: path,
				LockedBy: ScopeLockInfo{
					Path:      lock.Path,
					AgentID:   lock.AgentID,
					Scenario:  lock.Scenario,
					StartedAt: lock.AcquiredAt,
					ExpiresAt: lock.ExpiresAt,
				},
			})
		}
	}
	return conflicts, nil
}

func (m *MockAgentRepository) AcquireSpawnIntent(ctx context.Context, key, scenario string, scope []string, ttl time.Duration) (*SpawnIntent, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if intent, ok := m.intents[key]; ok {
		return intent, false, nil
	}
	intent := &SpawnIntent{
		Key:      key,
		Scenario: scenario,
		Scope:    scope,
		Status:   "pending",
	}
	m.intents[key] = intent
	return intent, true, nil
}

func (m *MockAgentRepository) UpdateSpawnIntent(ctx context.Context, key string, agentID, status, resultJSON string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if intent, ok := m.intents[key]; ok {
		intent.AgentID = agentID
		intent.Status = status
		intent.ResultJSON = resultJSON
	}
	return nil
}

func (m *MockAgentRepository) GetSpawnIntent(ctx context.Context, key string) (*SpawnIntent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if intent, ok := m.intents[key]; ok {
		return intent, nil
	}
	return nil, nil
}

func (m *MockAgentRepository) CleanupSpawnIntents(ctx context.Context) (int64, error) {
	return 0, nil
}

func (m *MockAgentRepository) RecordFileOperation(ctx context.Context, input FileOperationInput) error {
	return nil
}

func (m *MockAgentRepository) GetFileOperations(ctx context.Context, agentID string) ([]*FileOperation, error) {
	return nil, nil
}

func (m *MockAgentRepository) GetFileOperationsForScenario(ctx context.Context, scenario string, limit int) ([]*FileOperation, error) {
	return nil, nil
}

func (m *MockAgentRepository) CreateSpawnSession(ctx context.Context, input CreateSpawnSessionInput) (*SpawnSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessionSeq++
	session := &SpawnSession{
		ID:             m.sessionSeq,
		UserIdentifier: input.UserIdentifier,
		Scenario:       input.Scenario,
		Scope:          input.Scope,
		Phases:         input.Phases,
		AgentIDs:       input.AgentIDs,
		Status:         "active",
		CreatedAt:      time.Now(),
		ExpiresAt:      time.Now().Add(input.TTL),
	}
	m.sessions[session.ID] = session
	return session, nil
}

func (m *MockAgentRepository) GetActiveSpawnSessions(ctx context.Context, userIdentifier, scenario string) ([]*SpawnSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*SpawnSession
	for _, session := range m.sessions {
		if session.UserIdentifier == userIdentifier && (scenario == "" || session.Scenario == scenario) {
			if session.Status == "active" {
				result = append(result, session)
			}
		}
	}
	return result, nil
}

func (m *MockAgentRepository) CheckSpawnSessionConflicts(ctx context.Context, userIdentifier, scenario string, scope []string) ([]SpawnSessionConflict, error) {
	return nil, nil
}

func (m *MockAgentRepository) UpdateSpawnSessionStatus(ctx context.Context, sessionID int, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if session, ok := m.sessions[sessionID]; ok {
		session.Status = status
	}
	return nil
}

func (m *MockAgentRepository) ClearSpawnSessions(ctx context.Context, userIdentifier string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var count int64
	for id, session := range m.sessions {
		if session.UserIdentifier == userIdentifier {
			session.Status = "cleared"
			delete(m.sessions, id)
			count++
		}
	}
	return count, nil
}

func (m *MockAgentRepository) CleanupExpiredSpawnSessions(ctx context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var count int64
	now := time.Now()
	for id, session := range m.sessions {
		if session.ExpiresAt.Before(now) {
			delete(m.sessions, id)
			count++
		}
	}
	return count, nil
}

func isActiveStatus(status AgentStatus) bool {
	return status == AgentStatusPending || status == AgentStatusRunning
}

// --- Tests ---

func TestAgentServiceSeams(t *testing.T) {
	t.Run("ProcessChecker seam is called during orphan cleanup", func(t *testing.T) {
		repo := NewMockAgentRepository()
		processChecker := NewMockProcessChecker()
		envProvider := NewMockEnvironmentProvider()
		timeProvider := NewMockTimeProvider(time.Now())

		// Create an "active" agent that appears to still be running
		activeAgent := &SpawnedAgent{
			ID:       "agent-123",
			Scenario: "test-scenario",
			Status:   AgentStatusRunning,
			PID:      12345,
			Hostname: "test-host",
		}
		repo.agents[activeAgent.ID] = activeAgent

		// Mark the process as alive
		processChecker.SetAlive(12345, true)

		// Create service with injected seams
		svc := NewAgentService(repo,
			WithProcessChecker(processChecker),
			WithEnvironmentProvider(envProvider),
			WithTimeProvider(timeProvider),
		)
		defer svc.Close()

		// Verify process was checked
		checkLog := processChecker.GetCheckLog()
		if len(checkLog) == 0 {
			t.Error("Expected ProcessChecker.IsAlive to be called during orphan cleanup")
		}
		if checkLog[0] != 12345 {
			t.Errorf("Expected PID 12345 to be checked, got %d", checkLog[0])
		}

		// Agent should still be running since process is alive
		agent, _ := repo.Get(context.Background(), "agent-123")
		if agent.Status != AgentStatusRunning {
			t.Errorf("Expected agent to remain running, got status %s", agent.Status)
		}
	})

	t.Run("Dead process is marked as orphaned", func(t *testing.T) {
		repo := NewMockAgentRepository()
		processChecker := NewMockProcessChecker()
		envProvider := NewMockEnvironmentProvider()
		timeProvider := NewMockTimeProvider(time.Now())

		// Create an "active" agent with a dead process
		activeAgent := &SpawnedAgent{
			ID:       "agent-456",
			Scenario: "test-scenario",
			Status:   AgentStatusRunning,
			PID:      99999,
			Hostname: "test-host",
		}
		repo.agents[activeAgent.ID] = activeAgent

		// Mark the process as dead
		processChecker.SetAlive(99999, false)

		// Create service - orphan cleanup runs in constructor
		svc := NewAgentService(repo,
			WithProcessChecker(processChecker),
			WithEnvironmentProvider(envProvider),
			WithTimeProvider(timeProvider),
		)
		defer svc.Close()

		// Agent should be marked as failed
		agent, _ := repo.Get(context.Background(), "agent-456")
		if agent.Status != AgentStatusFailed {
			t.Errorf("Expected agent to be marked as failed, got status %s", agent.Status)
		}
		if agent.Error == "" {
			t.Error("Expected error message to be set for orphaned agent")
		}
	})

	t.Run("Config option is used for lock timeout config", func(t *testing.T) {
		repo := NewMockAgentRepository()
		envProvider := NewMockEnvironmentProvider()
		timeProvider := NewMockTimeProvider(time.Now())

		// Use WithConfig option to set custom lock timeout
		cfg := DefaultConfig()
		cfg.LockTimeoutMinutes = 30

		svc := NewAgentService(repo,
			WithConfig(cfg),
			WithEnvironmentProvider(envProvider),
			WithTimeProvider(timeProvider),
		)
		defer svc.Close()

		// Verify the custom timeout was applied via GetLockTimeout()
		expectedTimeout := 30 * time.Minute
		if svc.GetLockTimeout() != expectedTimeout {
			t.Errorf("Expected lock timeout of %v, got %v", expectedTimeout, svc.GetLockTimeout())
		}

		// Also verify GetConfig() returns the same config
		gotConfig := svc.GetConfig()
		if gotConfig.LockTimeoutMinutes != 30 {
			t.Errorf("Expected config LockTimeoutMinutes of 30, got %d", gotConfig.LockTimeoutMinutes)
		}
	})

	t.Run("TimeProvider seam is used for lock expiration", func(t *testing.T) {
		repo := NewMockAgentRepository()
		envProvider := NewMockEnvironmentProvider()
		fixedTime := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
		timeProvider := NewMockTimeProvider(fixedTime)

		svc := NewAgentService(repo,
			WithEnvironmentProvider(envProvider),
			WithTimeProvider(timeProvider),
		)
		defer svc.Close()

		// Register an agent
		agent, err := svc.Register(context.Background(), CreateAgentInput{
			ID:         "agent-time-test",
			Scenario:   "test-scenario",
			Scope:      []string{"api"},
			PromptHash: "hash123",
		})
		if err != nil {
			t.Fatalf("Failed to register agent: %v", err)
		}

		// Check that lock expiration is based on the mock time
		locks, _ := repo.GetActiveLocks(context.Background())
		for _, lock := range locks {
			if lock.AgentID == agent.ID {
				expectedExpiry := fixedTime.Add(DefaultLockTimeout)
				if !lock.ExpiresAt.Equal(expectedExpiry) {
					t.Errorf("Expected lock expiry at %v, got %v", expectedExpiry, lock.ExpiresAt)
				}
			}
		}
	})

	t.Run("Hostname from EnvironmentProvider is used for PID tracking", func(t *testing.T) {
		repo := NewMockAgentRepository()
		envProvider := NewMockEnvironmentProvider()
		timeProvider := NewMockTimeProvider(time.Now())

		customHostname := "custom-test-host"
		envProvider.SetHostname(customHostname)

		svc := NewAgentService(repo,
			WithEnvironmentProvider(envProvider),
			WithTimeProvider(timeProvider),
		)
		defer svc.Close()

		// Register an agent first
		agent, _ := svc.Register(context.Background(), CreateAgentInput{
			ID:         "agent-hostname-test",
			Scenario:   "test-scenario",
			Scope:      []string{"api"},
			PromptHash: "hosthash",
		})

		// Set PID
		err := svc.SetAgentPID(context.Background(), agent.ID, 55555)
		if err != nil {
			t.Fatalf("Failed to set agent PID: %v", err)
		}

		// Verify hostname was set from the mock provider
		updatedAgent, _ := repo.Get(context.Background(), agent.ID)
		if updatedAgent.Hostname != customHostname {
			t.Errorf("Expected hostname %q, got %q", customHostname, updatedAgent.Hostname)
		}
	})

	t.Run("RenewLocks uses TimeProvider", func(t *testing.T) {
		repo := NewMockAgentRepository()
		envProvider := NewMockEnvironmentProvider()
		initialTime := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
		timeProvider := NewMockTimeProvider(initialTime)

		svc := NewAgentService(repo,
			WithEnvironmentProvider(envProvider),
			WithTimeProvider(timeProvider),
		)
		defer svc.Close()

		// Register an agent with locks
		agent, _ := svc.Register(context.Background(), CreateAgentInput{
			ID:         "agent-renew-test",
			Scenario:   "test-scenario",
			Scope:      []string{"api"},
			PromptHash: "renewhash",
		})

		// Advance time
		newTime := initialTime.Add(5 * time.Minute)
		timeProvider.Set(newTime)

		// Renew locks
		err := svc.RenewLocks(context.Background(), agent.ID)
		if err != nil {
			t.Fatalf("Failed to renew locks: %v", err)
		}

		// Verify lock expiry is based on new time
		locks, _ := repo.GetActiveLocks(context.Background())
		for _, lock := range locks {
			if lock.AgentID == agent.ID {
				expectedExpiry := newTime.Add(DefaultLockTimeout)
				if !lock.ExpiresAt.Equal(expectedExpiry) {
					t.Errorf("Expected renewed lock expiry at %v, got %v", expectedExpiry, lock.ExpiresAt)
				}
			}
		}
	})
}

func TestAgentServiceWithoutOptions(t *testing.T) {
	t.Run("Service works with default seam implementations", func(t *testing.T) {
		repo := NewMockAgentRepository()

		// Create service without any options - should use OS defaults
		svc := NewAgentService(repo)
		defer svc.Close()

		// Verify it still works for basic operations
		agent, err := svc.Register(context.Background(), CreateAgentInput{
			ID:         "agent-default-test",
			Scenario:   "test-scenario",
			Scope:      []string{"api"},
			PromptHash: "defaulthash",
		})
		if err != nil {
			t.Fatalf("Failed to register agent: %v", err)
		}

		if agent.ID != "agent-default-test" {
			t.Errorf("Expected agent ID %q, got %q", "agent-default-test", agent.ID)
		}
	})
}

// --- Tests for ClassifyAgentOrphanStatus decision function ---

func TestClassifyAgentOrphanStatus(t *testing.T) {
	tests := []struct {
		name            string
		agent           *SpawnedAgent
		currentHostname string
		processAlive    bool
		wantOrphaned    bool
		wantReasonType  string // Substring to check in reason
	}{
		{
			name: "No PID recorded - orphaned",
			agent: &SpawnedAgent{
				ID:       "agent-1",
				PID:      0,
				Hostname: "test-host",
			},
			currentHostname: "test-host",
			processAlive:    false,
			wantOrphaned:    true,
			wantReasonType:  "server restarted",
		},
		{
			name: "Negative PID - orphaned",
			agent: &SpawnedAgent{
				ID:       "agent-2",
				PID:      -1,
				Hostname: "test-host",
			},
			currentHostname: "test-host",
			processAlive:    false,
			wantOrphaned:    true,
			wantReasonType:  "server restarted",
		},
		{
			name: "Different hostname - orphaned",
			agent: &SpawnedAgent{
				ID:       "agent-3",
				PID:      12345,
				Hostname: "other-host",
			},
			currentHostname: "current-host",
			processAlive:    true, // Doesn't matter - can't check remote process
			wantOrphaned:    true,
			wantReasonType:  "no longer running",
		},
		{
			name: "Same host, process alive - not orphaned",
			agent: &SpawnedAgent{
				ID:       "agent-4",
				PID:      12345,
				Hostname: "test-host",
			},
			currentHostname: "test-host",
			processAlive:    true,
			wantOrphaned:    false,
			wantReasonType:  "still running",
		},
		{
			name: "Same host, process dead - orphaned",
			agent: &SpawnedAgent{
				ID:       "agent-5",
				PID:      12345,
				Hostname: "test-host",
			},
			currentHostname: "test-host",
			processAlive:    false,
			wantOrphaned:    true,
			wantReasonType:  "no longer running",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockAgentRepository()
			processChecker := NewMockProcessChecker()
			envProvider := NewMockEnvironmentProvider()
			timeProvider := NewMockTimeProvider(time.Now())

			processChecker.SetAlive(tt.agent.PID, tt.processAlive)

			svc := NewAgentService(repo,
				WithProcessChecker(processChecker),
				WithEnvironmentProvider(envProvider),
				WithTimeProvider(timeProvider),
			)
			defer svc.Close()

			result := svc.ClassifyAgentOrphanStatus(tt.agent, tt.currentHostname)

			if result.IsOrphaned != tt.wantOrphaned {
				t.Errorf("IsOrphaned = %v, want %v", result.IsOrphaned, tt.wantOrphaned)
			}

			if tt.wantReasonType != "" && !contains(result.Reason, tt.wantReasonType) {
				t.Errorf("Reason = %q, want substring %q", result.Reason, tt.wantReasonType)
			}
		})
	}
}

// --- Tests for ClassifyIdempotencyAction decision function ---

func TestClassifyIdempotencyAction(t *testing.T) {
	tests := []struct {
		name       string
		intent     *SpawnIntent
		isNew      bool
		wantAction IdempotencyAction
	}{
		{
			name:       "New request proceeds",
			intent:     &SpawnIntent{Key: "key-1", Status: "pending"},
			isNew:      true,
			wantAction: IdempotencyActionProceed,
		},
		{
			name:       "Completed intent returns cached",
			intent:     &SpawnIntent{Key: "key-2", Status: SpawnIntentStatusCompleted, ResultJSON: `{"success":true}`},
			isNew:      false,
			wantAction: IdempotencyActionReturnCached,
		},
		{
			name:       "Pending intent returns conflict",
			intent:     &SpawnIntent{Key: "key-3", Status: SpawnIntentStatusPending},
			isNew:      false,
			wantAction: IdempotencyActionConflict,
		},
		{
			name:       "Failed intent allows retry",
			intent:     &SpawnIntent{Key: "key-4", Status: SpawnIntentStatusFailed},
			isNew:      false,
			wantAction: IdempotencyActionRetry,
		},
		{
			name:       "Unknown status allows retry",
			intent:     &SpawnIntent{Key: "key-5", Status: "unknown"},
			isNew:      false,
			wantAction: IdempotencyActionRetry,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyIdempotencyAction(tt.intent, tt.isNew)

			if result.Action != tt.wantAction {
				t.Errorf("Action = %v, want %v", result.Action, tt.wantAction)
			}

			// For completed intent, verify cached result is populated
			if tt.wantAction == IdempotencyActionReturnCached {
				if result.CachedResult != tt.intent.ResultJSON {
					t.Errorf("CachedResult = %v, want %v", result.CachedResult, tt.intent.ResultJSON)
				}
			}

			// All results should have a reason
			if result.Reason == "" {
				t.Error("Expected non-empty Reason")
			}
		})
	}
}

// helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- Tests for ClassifyAgentStatus decision function ---

func TestClassifyAgentStatus(t *testing.T) {
	tests := []struct {
		name         string
		status       AgentStatus
		wantTerminal bool
		wantActive   bool
	}{
		{
			name:         "Pending is active, not terminal",
			status:       AgentStatusPending,
			wantTerminal: false,
			wantActive:   true,
		},
		{
			name:         "Running is active, not terminal",
			status:       AgentStatusRunning,
			wantTerminal: false,
			wantActive:   true,
		},
		{
			name:         "Completed is terminal, not active",
			status:       AgentStatusCompleted,
			wantTerminal: true,
			wantActive:   false,
		},
		{
			name:         "Failed is terminal, not active",
			status:       AgentStatusFailed,
			wantTerminal: true,
			wantActive:   false,
		},
		{
			name:         "Timeout is terminal, not active",
			status:       AgentStatusTimeout,
			wantTerminal: true,
			wantActive:   false,
		},
		{
			name:         "Stopped is terminal, not active",
			status:       AgentStatusStopped,
			wantTerminal: true,
			wantActive:   false,
		},
		{
			name:         "Unknown status is treated as terminal for safety",
			status:       AgentStatus("unknown"),
			wantTerminal: true,
			wantActive:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyAgentStatus(tt.status)

			if result.IsTerminal != tt.wantTerminal {
				t.Errorf("IsTerminal = %v, want %v", result.IsTerminal, tt.wantTerminal)
			}

			if result.IsActive != tt.wantActive {
				t.Errorf("IsActive = %v, want %v", result.IsActive, tt.wantActive)
			}

			// All results should have a reason
			if result.Reason == "" {
				t.Error("Expected non-empty Reason")
			}

			// Verify helper methods match classification
			if tt.status.IsTerminal() != tt.wantTerminal {
				t.Errorf("AgentStatus.IsTerminal() = %v, want %v", tt.status.IsTerminal(), tt.wantTerminal)
			}

			if tt.status.IsActive() != tt.wantActive {
				t.Errorf("AgentStatus.IsActive() = %v, want %v", tt.status.IsActive(), tt.wantActive)
			}
		})
	}
}

// --- Tests for DecideScopeExpansion decision function ---

func TestDecideScopeExpansion(t *testing.T) {
	tests := []struct {
		name             string
		requestedScope   []string
		wantExpand       bool
		wantMinimumPaths int // Minimum number of paths in expanded scope
		wantContainsDeps bool
	}{
		{
			name:             "Empty scope does not expand",
			requestedScope:   []string{},
			wantExpand:       false,
			wantMinimumPaths: 0,
			wantContainsDeps: false,
		},
		{
			name:             "Single path expands with dependency files",
			requestedScope:   []string{"api"},
			wantExpand:       true,
			wantMinimumPaths: len(SharedDependencyFiles) + 1, // original + deps
			wantContainsDeps: true,
		},
		{
			name:             "Multiple paths expand with dependency files",
			requestedScope:   []string{"api", "cli", "ui"},
			wantExpand:       true,
			wantMinimumPaths: len(SharedDependencyFiles) + 3, // originals + deps
			wantContainsDeps: true,
		},
		{
			name:             "Scope already containing go.mod still expands",
			requestedScope:   []string{"api", "go.mod"},
			wantExpand:       true,
			wantMinimumPaths: len(SharedDependencyFiles) + 1, // deduped
			wantContainsDeps: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := DecideScopeExpansion(tt.requestedScope)

			if decision.ShouldExpand != tt.wantExpand {
				t.Errorf("ShouldExpand = %v, want %v", decision.ShouldExpand, tt.wantExpand)
			}

			if len(decision.ExpandedScope) < tt.wantMinimumPaths {
				t.Errorf("ExpandedScope has %d paths, want at least %d", len(decision.ExpandedScope), tt.wantMinimumPaths)
			}

			// Check if dependency files are included when expected
			if tt.wantContainsDeps {
				scopeMap := make(map[string]bool)
				for _, p := range decision.ExpandedScope {
					scopeMap[p] = true
				}
				if !scopeMap["go.mod"] {
					t.Error("Expected expanded scope to contain go.mod")
				}
				if !scopeMap["package.json"] {
					t.Error("Expected expanded scope to contain package.json")
				}
			}

			// All results should have a reason
			if decision.Reason == "" {
				t.Error("Expected non-empty Reason")
			}

			// Verify backward compatibility wrapper
			legacyResult := ExpandScopeWithImplicitPaths(tt.requestedScope)
			if len(legacyResult) != len(decision.ExpandedScope) {
				t.Errorf("Legacy wrapper returned %d paths, decision returned %d", len(legacyResult), len(decision.ExpandedScope))
			}
		})
	}
}
