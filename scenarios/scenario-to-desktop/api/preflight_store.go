package main

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"

	bundlemanifest "scenario-to-desktop-runtime/manifest"
	bundleruntime "scenario-to-desktop-runtime"
)

// PreflightSessionStore defines the interface for preflight session management.
// This enables testing seams - implementations can be swapped for tests.
type PreflightSessionStore interface {
	// Create creates a new preflight session with the given manifest and bundle root.
	Create(manifest *bundlemanifest.Manifest, bundleRoot string, ttlSeconds int) (*preflightSession, error)

	// Get retrieves a session by ID, checking expiry.
	// Returns nil and false if session not found or expired.
	Get(id string) (*preflightSession, bool)

	// Refresh extends the TTL of an existing session.
	Refresh(session *preflightSession, ttlSeconds int)

	// Stop stops and cleans up a preflight session.
	// Returns true if session was found and stopped.
	Stop(id string) bool

	// Cleanup removes expired sessions. Called periodically by janitor.
	Cleanup()
}

// PreflightJobStore defines the interface for preflight job management.
// This enables testing seams for async job operations.
type PreflightJobStore interface {
	// Create creates a new async preflight job and returns it.
	Create() *preflightJob

	// Get retrieves a job by ID.
	Get(id string) (*preflightJob, bool)

	// Update updates a job using a modifier function.
	Update(id string, fn func(job *preflightJob))

	// SetStep updates the state of a specific step in a job.
	SetStep(id, stepID, state, detail string)

	// SetResult updates the result of a job using an updater function.
	SetResult(id string, updater func(prev *BundlePreflightResponse) *BundlePreflightResponse)

	// Finish marks a job as complete with final status.
	Finish(id string, status, errMsg string)

	// Cleanup removes completed jobs older than the expiration threshold.
	Cleanup()
}

// inMemorySessionStore is the default in-memory implementation of PreflightSessionStore.
type inMemorySessionStore struct {
	sessions     map[string]*preflightSession
	mux          sync.Mutex
	createSupervisor func(manifest *bundlemanifest.Manifest, bundleRoot, appData string) (*bundleruntime.Supervisor, error)
}

// NewInMemorySessionStore creates a new in-memory session store.
func NewInMemorySessionStore() *inMemorySessionStore {
	return &inMemorySessionStore{
		sessions: make(map[string]*preflightSession),
		createSupervisor: defaultCreateSupervisor,
	}
}

func defaultCreateSupervisor(manifest *bundlemanifest.Manifest, bundleRoot, appData string) (*bundleruntime.Supervisor, error) {
	return bundleruntime.NewSupervisor(bundleruntime.Options{
		Manifest:   manifest,
		BundlePath: bundleRoot,
		AppDataDir: appData,
		DryRun:     false,
	})
}

func (s *inMemorySessionStore) Create(manifest *bundlemanifest.Manifest, bundleRoot string, ttlSeconds int) (*preflightSession, error) {
	if ttlSeconds <= 0 {
		ttlSeconds = 120
	}
	if ttlSeconds > 900 {
		ttlSeconds = 900
	}

	appData, err := os.MkdirTemp("", "s2d-preflight-*")
	if err != nil {
		return nil, err
	}

	supervisor, err := s.createSupervisor(manifest, bundleRoot, appData)
	if err != nil {
		_ = os.RemoveAll(appData)
		return nil, err
	}

	ctx := context.Background()
	if err := supervisor.Start(ctx); err != nil {
		_ = os.RemoveAll(appData)
		return nil, err
	}

	session := &preflightSession{
		id:         uuid.NewString(),
		manifest:   manifest,
		bundleDir:  bundleRoot,
		appData:    appData,
		supervisor: supervisor,
		createdAt:  time.Now(),
		expiresAt:  time.Now().Add(time.Duration(ttlSeconds) * time.Second),
	}

	s.mux.Lock()
	s.sessions[session.id] = session
	s.mux.Unlock()

	return session, nil
}

func (s *inMemorySessionStore) Get(id string) (*preflightSession, bool) {
	s.mux.Lock()
	defer s.mux.Unlock()
	session, ok := s.sessions[id]
	if !ok {
		return nil, false
	}
	if time.Now().After(session.expiresAt) {
		delete(s.sessions, id)
		go shutdownSession(session)
		return nil, false
	}
	return session, true
}

func (s *inMemorySessionStore) Refresh(session *preflightSession, ttlSeconds int) {
	if ttlSeconds <= 0 {
		return
	}
	maxTTL := 900
	if ttlSeconds > maxTTL {
		ttlSeconds = maxTTL
	}
	s.mux.Lock()
	defer s.mux.Unlock()
	session.expiresAt = time.Now().Add(time.Duration(ttlSeconds) * time.Second)
}

func (s *inMemorySessionStore) Stop(id string) bool {
	s.mux.Lock()
	session, ok := s.sessions[id]
	if ok {
		delete(s.sessions, id)
	}
	s.mux.Unlock()
	if ok {
		shutdownSession(session)
	}
	return ok
}

func (s *inMemorySessionStore) Cleanup() {
	var expired []*preflightSession
	now := time.Now()
	s.mux.Lock()
	for id, session := range s.sessions {
		if now.After(session.expiresAt) {
			expired = append(expired, session)
			delete(s.sessions, id)
		}
	}
	s.mux.Unlock()
	for _, session := range expired {
		shutdownSession(session)
	}
}

// shutdownSession cleans up session resources.
func shutdownSession(session *preflightSession) {
	if session == nil {
		return
	}
	if session.supervisor != nil {
		_ = session.supervisor.Shutdown(context.Background())
	}
	if session.appData != "" {
		_ = os.RemoveAll(session.appData)
	}
}

// inMemoryJobStore is the default in-memory implementation of PreflightJobStore.
type inMemoryJobStore struct {
	jobs map[string]*preflightJob
	mux  sync.Mutex
}

// NewInMemoryJobStore creates a new in-memory job store.
func NewInMemoryJobStore() *inMemoryJobStore {
	return &inMemoryJobStore{
		jobs: make(map[string]*preflightJob),
	}
}

func (s *inMemoryJobStore) Create() *preflightJob {
	job := &preflightJob{
		id:     uuid.NewString(),
		status: "running",
		steps: map[string]BundlePreflightStep{
			"validation":  {ID: "validation", Name: "Load bundle + validate", State: "pending"},
			"runtime":     {ID: "runtime", Name: "Start runtime control API", State: "pending"},
			"secrets":     {ID: "secrets", Name: "Apply secrets", State: "pending"},
			"services":    {ID: "services", Name: "Start services", State: "pending"},
			"diagnostics": {ID: "diagnostics", Name: "Collect diagnostics", State: "pending"},
		},
		startedAt: time.Now(),
		updatedAt: time.Now(),
	}
	s.mux.Lock()
	s.jobs[job.id] = job
	s.mux.Unlock()
	return job
}

func (s *inMemoryJobStore) Get(id string) (*preflightJob, bool) {
	s.mux.Lock()
	defer s.mux.Unlock()
	job, ok := s.jobs[id]
	return job, ok
}

func (s *inMemoryJobStore) Update(id string, fn func(job *preflightJob)) {
	s.mux.Lock()
	defer s.mux.Unlock()
	job, ok := s.jobs[id]
	if !ok {
		return
	}
	fn(job)
	job.updatedAt = time.Now()
}

func (s *inMemoryJobStore) SetStep(id, stepID, state, detail string) {
	s.Update(id, func(job *preflightJob) {
		step, ok := job.steps[stepID]
		if !ok {
			step = BundlePreflightStep{ID: stepID, Name: stepID}
		}
		step.State = state
		step.Detail = detail
		job.steps[stepID] = step
	})
}

func (s *inMemoryJobStore) SetResult(id string, updater func(prev *BundlePreflightResponse) *BundlePreflightResponse) {
	s.Update(id, func(job *preflightJob) {
		job.result = updater(job.result)
	})
}

func (s *inMemoryJobStore) Finish(id string, status, errMsg string) {
	s.Update(id, func(job *preflightJob) {
		job.status = status
		job.err = errMsg
	})
}

func (s *inMemoryJobStore) Cleanup() {
	expiration := 15 * time.Minute
	now := time.Now()

	s.mux.Lock()
	for id, job := range s.jobs {
		if job.status == "running" {
			continue
		}
		if now.Sub(job.updatedAt) > expiration {
			delete(s.jobs, id)
		}
	}
	s.mux.Unlock()
}
