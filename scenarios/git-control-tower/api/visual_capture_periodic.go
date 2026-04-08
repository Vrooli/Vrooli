package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// PeriodicCaptureConfig configures the periodic capture goroutine.
type PeriodicCaptureConfig struct {
	Interval     time.Duration
	MaxSnapshots int
}

// PeriodicCapture runs background captures for changed scenarios.
type PeriodicCapture struct {
	config       PeriodicCaptureConfig
	capabilities *CapabilityRegistry
	basClient    *BrowserAutomationClient
	storage      *VisualCaptureStorage
	repos        *RepoService
	git          GitRunner
	repoLock     *RepoLock
	fs           FileIO
	mu           sync.Mutex
	cancel       context.CancelFunc
	running      bool
}

// NewPeriodicCapture creates a new periodic capture instance.
func NewPeriodicCapture(
	config PeriodicCaptureConfig,
	capabilities *CapabilityRegistry,
	basClient *BrowserAutomationClient,
	storage *VisualCaptureStorage,
	repos *RepoService,
	git GitRunner,
	repoLock *RepoLock,
) *PeriodicCapture {
	if config.Interval == 0 {
		config.Interval = 1 * time.Hour
	}
	if config.MaxSnapshots == 0 {
		config.MaxSnapshots = 10
	}
	return &PeriodicCapture{
		config:       config,
		capabilities: capabilities,
		basClient:    basClient,
		storage:      storage,
		repos:        repos,
		git:          git,
		repoLock:     repoLock,
		fs:           OSFileIO{},
	}
}

// Start begins the background periodic capture goroutine.
func (p *PeriodicCapture) Start() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	p.running = true

	go p.run(ctx)
}

// Stop cancels the background goroutine.
func (p *PeriodicCapture) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return
	}

	p.cancel()
	p.running = false
}

func (p *PeriodicCapture) run(ctx context.Context) {
	ticker := time.NewTicker(p.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.tick(ctx)
		}
	}
}

// acquireRepoLockIfNeeded acquires the per-repo lock if a RepoLock is configured.
// Returns the unlock function (may be nil) and any error.
func (p *PeriodicCapture) acquireRepoLockIfNeeded(ctx context.Context, repoPath string) (func(), error) {
	if p.repoLock == nil {
		return nil, nil
	}
	unlock, err := p.repoLock.Acquire(ctx, repoPath)
	if err != nil {
		return nil, fmt.Errorf("lock acquisition timed out for %s: %w", repoPath, err)
	}
	return unlock, nil
}

// shouldCaptureScope returns true if the scope is a scenario and its last capture
// is older than the configured interval.
func (p *PeriodicCapture) shouldCaptureScope(scope string, repoID int64) (string, bool) {
	if !isScenarioScope(scope) {
		return "", false
	}
	slug := scopeSlug(scope)
	existing, err := p.storage.ListSnapshotSets(repoID, slug)
	if err == nil && len(existing) > 0 && time.Since(existing[0].CreatedAt) < p.config.Interval {
		return slug, false
	}
	return slug, true
}

func (p *PeriodicCapture) tick(ctx context.Context) {
	// DESIGN DECISION: Periodic timer captures screenshots ONLY.
	// Workflow execution is too heavy for background periodic runs.
	// Workflows are triggered manually via POST /api/v1/repo/workflow-capture.

	if !p.capabilities.IsAvailable(ctx, "browser-automation-studio") {
		return
	}

	active, err := p.repos.GetActive(ctx)
	if err != nil || active == nil {
		return
	}

	unlock, err := p.acquireRepoLockIfNeeded(ctx, active.Path)
	if err != nil {
		log.Printf("periodic capture: %v", err)
		return
	}

	status, err := GetRepoStatus(ctx, RepoStatusDeps{
		Git:     p.git,
		RepoDir: active.Path,
	})

	if unlock != nil {
		unlock()
	}

	if err != nil {
		log.Printf("periodic capture: failed to get repo status: %v", err)
		return
	}

	for scope := range status.Scopes {
		slug, ok := p.shouldCaptureScope(scope, active.ID)
		if !ok {
			continue
		}

		meta, captureErr := CaptureScenario(ctx, VisualCaptureDeps{
			BAS:     p.basClient,
			Storage: p.storage,
			FS:      p.fs,
			RepoDir: active.Path,
			RepoID:  active.ID,
		}, VisualCaptureRequest{
			ScenarioSlug: slug,
			Mode:         CaptureModeCapture,
			TriggerType:  "periodic",
		})
		if captureErr != nil {
			log.Printf("periodic capture: failed for %s: %v", slug, captureErr)
			continue
		}
		log.Printf("periodic capture: captured %s (%d screenshots)", slug, meta.ScreenshotCount)
	}
}

func isScenarioScope(scope string) bool {
	return len(scope) > 9 && scope[:9] == "scenario:"
}

func scopeSlug(scope string) string {
	if idx := len("scenario:"); idx < len(scope) {
		return scope[idx:]
	}
	return scope
}
