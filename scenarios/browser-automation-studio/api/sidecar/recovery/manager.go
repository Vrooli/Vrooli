package recovery

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// CheckpointManager manages checkpoint lifecycle for recording sessions.
type CheckpointManager struct {
	config       Config
	store        Store
	actionSource ActionSource
	log          *logrus.Logger

	activeSessions map[string]*checkpointWorker
	mu             sync.RWMutex

	stopCh  chan struct{}
	stopped bool
}

// checkpointWorker manages checkpointing for a single session.
type checkpointWorker struct {
	sessionID     string
	workflowID    string
	browserConfig BrowserConfig
	stopCh        chan struct{}
}

// NewCheckpointManager creates a new CheckpointManager.
//
// Parameters:
//   - config: checkpoint configuration
//   - store: persistent storage for checkpoints
//   - actionSource: function to get current actions from sidecar
//   - log: logger for checkpoint events
func NewCheckpointManager(
	config Config,
	store Store,
	actionSource ActionSource,
	log *logrus.Logger,
) *CheckpointManager {
	return &CheckpointManager{
		config:         config,
		store:          store,
		actionSource:   actionSource,
		log:            log,
		activeSessions: make(map[string]*checkpointWorker),
	}
}

// Start initializes the checkpoint manager and starts cleanup.
func (m *CheckpointManager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.stopped {
		m.mu.Unlock()
		return nil
	}
	m.stopCh = make(chan struct{})
	m.mu.Unlock()

	// Start cleanup goroutine
	go m.cleanupLoop()

	m.log.Info("Checkpoint manager started")
	return nil
}

// Stop stops the checkpoint manager.
func (m *CheckpointManager) Stop() error {
	m.mu.Lock()
	if m.stopped {
		m.mu.Unlock()
		return nil
	}
	m.stopped = true
	close(m.stopCh)

	// Stop all active workers
	for _, worker := range m.activeSessions {
		close(worker.stopCh)
	}
	m.activeSessions = make(map[string]*checkpointWorker)
	m.mu.Unlock()

	m.log.Info("Checkpoint manager stopped")
	return nil
}

// cleanupLoop periodically cleans up old checkpoints.
func (m *CheckpointManager) cleanupLoop() {
	ticker := time.NewTicker(m.config.Retention / 2)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.doCleanup()
		}
	}
}

// doCleanup removes old checkpoints.
func (m *CheckpointManager) doCleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	count, err := m.store.Cleanup(ctx, m.config.Retention)
	if err != nil {
		m.log.WithError(err).Warn("Failed to cleanup old checkpoints")
		return
	}

	if count > 0 {
		m.log.WithField("count", count).Info("Cleaned up old checkpoints")
	}
}

// StartCheckpointing begins periodic checkpointing for a session.
func (m *CheckpointManager) StartCheckpointing(sessionID string, workflowID string, browserConfig BrowserConfig) {
	if !m.config.Enabled {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already checkpointing this session
	if _, exists := m.activeSessions[sessionID]; exists {
		return
	}

	worker := &checkpointWorker{
		sessionID:     sessionID,
		workflowID:    workflowID,
		browserConfig: browserConfig,
		stopCh:        make(chan struct{}),
	}

	m.activeSessions[sessionID] = worker

	go m.checkpointLoop(worker)

	m.log.WithField("session_id", sessionID).Info("Started checkpointing for session")
}

// StopCheckpointing stops checkpointing and deletes the checkpoint.
func (m *CheckpointManager) StopCheckpointing(sessionID string) {
	m.mu.Lock()
	worker, exists := m.activeSessions[sessionID]
	if exists {
		close(worker.stopCh)
		delete(m.activeSessions, sessionID)
	}
	m.mu.Unlock()

	if exists {
		// Delete the checkpoint since recording completed successfully
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := m.store.Delete(ctx, sessionID); err != nil {
			m.log.WithError(err).WithField("session_id", sessionID).Warn("Failed to delete checkpoint")
		}

		m.log.WithField("session_id", sessionID).Info("Stopped checkpointing for session")
	}
}

// checkpointLoop periodically saves checkpoints for a session.
func (m *CheckpointManager) checkpointLoop(worker *checkpointWorker) {
	ticker := time.NewTicker(m.config.CheckpointInterval)
	defer ticker.Stop()

	for {
		select {
		case <-worker.stopCh:
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.saveCheckpoint(worker)
		}
	}
}

// saveCheckpoint saves the current state of a recording session.
func (m *CheckpointManager) saveCheckpoint(worker *checkpointWorker) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get current actions from the sidecar
	actions, currentURL, err := m.actionSource(ctx, worker.sessionID)
	if err != nil {
		m.log.WithError(err).WithField("session_id", worker.sessionID).Warn("Failed to get actions for checkpoint")
		return
	}

	// Skip if no actions
	if len(actions) == 0 {
		return
	}

	checkpoint := &SessionCheckpoint{
		ID:              uuid.New().String(),
		SessionID:       worker.sessionID,
		WorkflowID:      worker.workflowID,
		RecordedActions: actions,
		CurrentURL:      currentURL,
		BrowserConfig:   worker.browserConfig,
	}

	if err := m.store.Save(ctx, checkpoint); err != nil {
		m.log.WithError(err).WithField("session_id", worker.sessionID).Warn("Failed to save checkpoint")
		return
	}

	m.log.WithFields(logrus.Fields{
		"session_id":   worker.sessionID,
		"action_count": len(actions),
	}).Debug("Saved checkpoint")
}

// GetRecoveryData retrieves a checkpoint for potential recovery.
func (m *CheckpointManager) GetRecoveryData(ctx context.Context, sessionID string) (*SessionCheckpoint, error) {
	return m.store.Get(ctx, sessionID)
}

// ListRecoverableSessions returns sessions with available checkpoints.
func (m *CheckpointManager) ListRecoverableSessions(ctx context.Context) ([]*SessionCheckpoint, error) {
	return m.store.ListActive(ctx)
}

// DeleteCheckpoint explicitly deletes a checkpoint.
func (m *CheckpointManager) DeleteCheckpoint(ctx context.Context, sessionID string) error {
	return m.store.Delete(ctx, sessionID)
}

// compile-time check
var _ Manager = (*CheckpointManager)(nil)
