package mocks

import (
	"context"
	"sync"
	"sync/atomic"

	"architecture-cartographer/internal/manifest"
)

// FakeService satisfies manifest.Service for handler tests.
type FakeService struct {
	mu sync.Mutex

	Manifest    manifest.ManifestDefinition
	Diagnostics []manifest.Diagnostic
	NextErr     error

	ValidateCalls       atomic.Int64
	ValidateSourceCalls atomic.Int64
	GetCalls            atomic.Int64
	ListDomainsCalls    atomic.Int64
}

func (f *FakeService) ValidateManifest(_ context.Context, in manifest.ManifestDefinition) (manifest.ManifestDefinition, []manifest.Diagnostic, error) {
	f.ValidateCalls.Add(1)
	if f.NextErr != nil {
		return in, f.Diagnostics, f.NextErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Manifest = in
	return in, f.Diagnostics, nil
}

func (f *FakeService) ValidateSource(_ context.Context, scenario string, _ []byte, _ manifest.ContentType) (manifest.ManifestDefinition, []manifest.Diagnostic, error) {
	f.ValidateSourceCalls.Add(1)
	if f.NextErr != nil {
		return manifest.ManifestDefinition{}, f.Diagnostics, f.NextErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	out := f.Manifest
	if scenario != "" {
		out.Scenario = scenario
	}
	f.Manifest = out
	return out, f.Diagnostics, nil
}

func (f *FakeService) GetManifest(_ context.Context, scenario string) (manifest.ManifestDefinition, error) {
	f.GetCalls.Add(1)
	if f.NextErr != nil {
		return manifest.ManifestDefinition{}, f.NextErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.Manifest.Scenario == "" {
		return manifest.ManifestDefinition{}, manifest.ErrManifestNotFound{Scenario: scenario}
	}
	return f.Manifest, nil
}

func (f *FakeService) ListDomains(_ context.Context, _ string) ([]manifest.DomainSpec, error) {
	f.ListDomainsCalls.Add(1)
	if f.NextErr != nil {
		return nil, f.NextErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]manifest.DomainSpec, len(f.Manifest.Domains))
	copy(out, f.Manifest.Domains)
	return out, nil
}

var _ manifest.Service = (*FakeService)(nil)
