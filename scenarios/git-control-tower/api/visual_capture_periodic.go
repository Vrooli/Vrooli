package main

import (
	"context"
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

func (p *PeriodicCapture) tick(ctx context.Context) {
	// Check BAS availability
	if !p.capabilities.IsAvailable(ctx, "browser-automation-studio") {
		return
	}

	// Get active repo
	active, err := p.repos.GetActive(ctx)
	if err != nil || active == nil {
		return
	}

	// Get repo status to find changed scenarios
	status, err := GetRepoStatus(ctx, RepoStatusDeps{
		Git:     p.git,
		RepoDir: active.Path,
	})
	if err != nil {
		log.Printf("periodic capture: failed to get repo status: %v", err)
		return
	}

	// Extract scenario scopes with changes
	for scope := range status.Scopes {
		if !isScenarioScope(scope) {
			continue
		}
		slug := scopeSlug(scope)

		// Check if last capture is recent enough
		existing, err := p.storage.ListSnapshotSets(active.ID, slug)
		if err == nil && len(existing) > 0 {
			if time.Since(existing[0].CreatedAt) < p.config.Interval {
				continue
			}
		}

		req := VisualCaptureRequest{
			ScenarioSlug: slug,
			Mode:         CaptureModeCapture,
			TriggerType:  "periodic",
		}
		meta, err := CaptureScenario(ctx, VisualCaptureDeps{
			BAS:     p.basClient,
			Storage: p.storage,
			FS:      p.fs,
			RepoDir: active.Path,
			RepoID:  active.ID,
		}, req)
		if err != nil {
			log.Printf("periodic capture: failed for %s: %v", slug, err)
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
