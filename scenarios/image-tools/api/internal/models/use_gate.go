package models

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

// ErrModelInUse means a model is currently in use or is being installed or
// removed, so the requested operation cannot safely access its files.
var ErrModelInUse = errors.New("models: model is in use or being changed")

type modelUseEntry struct {
	lock sync.RWMutex
	refs int
}

// ModelUseGate coordinates inference readers with install and removal writers.
// Share one gate across Installer, the AI engine, and analysis services.
type ModelUseGate struct {
	mu      sync.Mutex
	entries map[string]*modelUseEntry
}

// NewModelUseGate creates an empty per-model use gate.
func NewModelUseGate() *ModelUseGate {
	return &ModelUseGate{entries: make(map[string]*modelUseEntry)}
}

func (g *ModelUseGate) entry(modelID string) (*modelUseEntry, error) {
	if g == nil {
		return nil, errors.New("models: model use gate is unavailable")
	}
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil, errors.New("models: model id is required for use coordination")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.entries == nil {
		g.entries = make(map[string]*modelUseEntry)
	}
	e := g.entries[modelID]
	if e == nil {
		e = &modelUseEntry{}
		g.entries[modelID] = e
	}
	e.refs++
	return e, nil
}

func (g *ModelUseGate) releaseEntry(modelID string, e *modelUseEntry) {
	g.mu.Lock()
	defer g.mu.Unlock()
	e.refs--
	if e.refs == 0 && g.entries[modelID] == e {
		delete(g.entries, modelID)
	}
}

// Acquire retains a shared lease while inference reads model files. Concurrent
// inference is allowed; install and removal fail with ErrModelInUse until every
// shared lease has been released.
func (g *ModelUseGate) Acquire(modelID string) (func(), error) {
	modelID = strings.TrimSpace(modelID)
	e, err := g.entry(modelID)
	if err != nil {
		return nil, err
	}
	if !e.lock.TryRLock() {
		g.releaseEntry(modelID, e)
		return nil, fmt.Errorf("%w: %s", ErrModelInUse, modelID)
	}
	var once sync.Once
	return func() {
		once.Do(func() {
			e.lock.RUnlock()
			g.releaseEntry(modelID, e)
		})
	}, nil
}

// AcquireExclusive retains the mutation side of the gate while install or
// removal reads or changes model files. It refuses to wait behind an active
// inference operation so callers receive a retryable, typed refusal.
func (g *ModelUseGate) AcquireExclusive(modelID string) (func(), error) {
	modelID = strings.TrimSpace(modelID)
	e, err := g.entry(modelID)
	if err != nil {
		return nil, err
	}
	if !e.lock.TryLock() {
		g.releaseEntry(modelID, e)
		return nil, fmt.Errorf("%w: %s", ErrModelInUse, modelID)
	}
	var once sync.Once
	return func() {
		once.Do(func() {
			e.lock.Unlock()
			g.releaseEntry(modelID, e)
		})
	}, nil
}
