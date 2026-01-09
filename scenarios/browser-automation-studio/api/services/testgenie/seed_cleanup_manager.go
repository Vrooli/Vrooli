package testgenie

import (
	"context"
	"fmt"
	"sync"
	"time"
	"strings"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
)

const (
	defaultCleanupRetryInterval = 10 * time.Second
	defaultCleanupTimeout       = 30 * time.Second
)

// ExecutionStatusReader reads execution status for cleanup scheduling.
type ExecutionStatusReader interface {
	GetExecution(ctx context.Context, id uuid.UUID) (*database.ExecutionIndex, error)
}

// SeedCleanupManager tracks async executions that require seed cleanup.
type SeedCleanupManager struct {
	exec          ExecutionStatusReader
	client        *Client
	log           *logrus.Logger
	pollInterval  time.Duration
	maxWait       time.Duration
	retryInterval time.Duration

	mu    sync.Mutex
	jobs  map[uuid.UUID]*seedCleanupJob
	start sync.Once
}

type seedCleanupJob struct {
	ExecutionID  uuid.UUID
	SeedScenario string
	CleanupToken string
	StartedAt    time.Time
	LastAttempt  time.Time
	Attempts     int
}

// NewSeedCleanupManager creates a cleanup manager and starts the background worker.
func NewSeedCleanupManager(exec ExecutionStatusReader, client *Client, log *logrus.Logger, pollInterval time.Duration, maxWait time.Duration) *SeedCleanupManager {
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}
	if maxWait <= 0 {
		maxWait = 15 * time.Minute
	}
	if log == nil {
		log = logrus.StandardLogger()
	}
	if client == nil {
		client = NewClient(nil, nil)
	}
	mgr := &SeedCleanupManager{
		exec:          exec,
		client:        client,
		log:           log,
		pollInterval:  pollInterval,
		maxWait:       maxWait,
		retryInterval: defaultCleanupRetryInterval,
		jobs:          map[uuid.UUID]*seedCleanupJob{},
	}
	mgr.start.Do(func() {
		go mgr.run()
	})
	return mgr
}

// Schedule registers an execution for async seed cleanup.
func (m *SeedCleanupManager) Schedule(executionID string, seedScenario string, cleanupToken string) error {
	if m == nil {
		return fmt.Errorf("seed cleanup manager not configured")
	}
	if strings.TrimSpace(executionID) == "" {
		return fmt.Errorf("execution id is required")
	}
	id, err := uuid.Parse(strings.TrimSpace(executionID))
	if err != nil {
		return fmt.Errorf("invalid execution id: %w", err)
	}
	if strings.TrimSpace(cleanupToken) == "" {
		return fmt.Errorf("cleanup token is required")
	}
	if strings.TrimSpace(seedScenario) == "" {
		seedScenario = defaultScenarioName
	}

	m.mu.Lock()
	m.jobs[id] = &seedCleanupJob{
		ExecutionID:  id,
		SeedScenario: seedScenario,
		CleanupToken: cleanupToken,
		StartedAt:    time.Now().UTC(),
	}
	m.mu.Unlock()

	m.log.WithFields(logrus.Fields{
		"execution_id":  id.String(),
		"seed_scenario": seedScenario,
	}).Info("Seed cleanup scheduled")
	return nil
}

func (m *SeedCleanupManager) run() {
	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()

	for range ticker.C {
		m.processJobs()
	}
}

func (m *SeedCleanupManager) processJobs() {
	if m == nil {
		return
	}
	m.mu.Lock()
	jobs := make([]*seedCleanupJob, 0, len(m.jobs))
	for _, job := range m.jobs {
		jobs = append(jobs, job)
	}
	m.mu.Unlock()

	for _, job := range jobs {
		m.processJob(job)
	}
}

func (m *SeedCleanupManager) processJob(job *seedCleanupJob) {
	if job == nil {
		return
	}
	if m.exec == nil {
		m.log.Warn("Seed cleanup manager missing execution reader")
		return
	}
	if time.Since(job.StartedAt) > m.maxWait {
		m.log.WithFields(logrus.Fields{
			"execution_id":  job.ExecutionID.String(),
			"seed_scenario": job.SeedScenario,
		}).Warn("Seed cleanup max wait exceeded; attempting cleanup")
		m.cleanup(job, true)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	exec, err := m.exec.GetExecution(ctx, job.ExecutionID)
	cancel()
	if err != nil {
		m.log.WithFields(logrus.Fields{
			"execution_id": job.ExecutionID.String(),
			"error":        err.Error(),
		}).Warn("Seed cleanup could not read execution status")
		return
	}
	if exec == nil {
		m.log.WithField("execution_id", job.ExecutionID.String()).Warn("Seed cleanup missing execution record")
		return
	}

	switch exec.Status {
	case database.ExecutionStatusCompleted, database.ExecutionStatusFailed:
		m.cleanup(job, false)
	}
}

func (m *SeedCleanupManager) cleanup(job *seedCleanupJob, forced bool) {
	if job == nil {
		return
	}
	if time.Since(job.LastAttempt) < m.retryInterval {
		return
	}

	job.LastAttempt = time.Now().UTC()
	job.Attempts++

	ctx, cancel := context.WithTimeout(context.Background(), defaultCleanupTimeout)
	defer cancel()

	_, err := m.client.CleanupSeed(ctx, job.SeedScenario, job.CleanupToken)
	if err != nil {
		m.log.WithFields(logrus.Fields{
			"execution_id":  job.ExecutionID.String(),
			"seed_scenario": job.SeedScenario,
			"attempts":      job.Attempts,
			"forced":        forced,
			"error":         err.Error(),
		}).Warn("Seed cleanup failed; will retry")
		return
	}

	m.log.WithFields(logrus.Fields{
		"execution_id":  job.ExecutionID.String(),
		"seed_scenario": job.SeedScenario,
		"attempts":      job.Attempts,
	}).Info("Seed cleanup completed")

	m.mu.Lock()
	delete(m.jobs, job.ExecutionID)
	m.mu.Unlock()
}
