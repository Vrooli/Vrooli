// Package mocks holds in-memory fakes for the manifest domain.
package mocks

import (
	"context"
	"sync"
	"sync/atomic"

	"architecture-cartographer/internal/manifest"
)

// FakeRepository is the in-memory manifest.Repository.
type FakeRepository struct {
	mu sync.Mutex

	ByScenario map[string]manifest.ManifestDefinition

	SaveErr error
	GetErr  error

	SaveCalls atomic.Int64
	GetCalls  atomic.Int64
}

func (f *FakeRepository) SaveManifest(_ context.Context, m manifest.ManifestDefinition) (manifest.ManifestDefinition, error) {
	f.SaveCalls.Add(1)
	if f.SaveErr != nil {
		return manifest.ManifestDefinition{}, f.SaveErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.ByScenario == nil {
		f.ByScenario = make(map[string]manifest.ManifestDefinition)
	}
	f.ByScenario[m.Scenario] = m
	return m, nil
}

func (f *FakeRepository) GetManifest(_ context.Context, scenario string) (manifest.ManifestDefinition, error) {
	f.GetCalls.Add(1)
	if f.GetErr != nil {
		return manifest.ManifestDefinition{}, f.GetErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.ByScenario[scenario]
	if !ok {
		return manifest.ManifestDefinition{}, manifest.ErrManifestNotFound{Scenario: scenario}
	}
	return m, nil
}

var _ manifest.Repository = (*FakeRepository)(nil)
