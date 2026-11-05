// Package mocks provides test fakes for the domains domain seams:
// DomainSourceExtractor and ScenarioLocator. Production code never imports
// this package.
package mocks

import (
	"context"
	"sync/atomic"

	"architecture-cartographer/internal/domains"
)

// FakeExtractor is a programmable DomainSourceExtractor. It returns the
// configured Extraction (or Err) regardless of the scenario directory, so
// ladder and service tests can drive resolution without touching the
// filesystem.
type FakeExtractor struct {
	Src        domains.Source
	Extraction domains.Extraction
	Err        error
	// Calls records each scenarioDir passed to Extract, for assertions.
	Calls []string
}

var _ domains.DomainSourceExtractor = (*FakeExtractor)(nil)

// Source returns the configured source.
func (f *FakeExtractor) Source() domains.Source { return f.Src }

// Extract returns the configured extraction or error.
func (f *FakeExtractor) Extract(_ context.Context, scenarioDir string) (domains.Extraction, error) {
	f.Calls = append(f.Calls, scenarioDir)
	if f.Err != nil {
		return domains.Extraction{}, f.Err
	}
	ex := f.Extraction
	ex.Source = f.Src
	return ex, nil
}

// FakeLocator is a programmable ScenarioLocator.
type FakeLocator struct {
	Dir string
	Err error
}

var _ domains.ScenarioLocator = (*FakeLocator)(nil)

// Locate returns the configured directory or error.
func (f *FakeLocator) Locate(string) (string, error) {
	if f.Err != nil {
		return "", f.Err
	}
	return f.Dir, nil
}

// FakeService satisfies domains.Service for handler tests. Both methods
// return Map/Err and bump GetCalls so callers can assert the domain map
// was fetched.
type FakeService struct {
	Map      domains.DerivedDomainMap
	Err      error
	GetCalls atomic.Int64
}

var _ domains.Service = (*FakeService)(nil)

func (f *FakeService) ExtractDomains(_ context.Context, _ string) (domains.DerivedDomainMap, error) {
	f.GetCalls.Add(1)
	if f.Err != nil {
		return domains.DerivedDomainMap{}, f.Err
	}
	return f.Map, nil
}

func (f *FakeService) GetDomainMap(_ context.Context, _ string) (domains.DerivedDomainMap, error) {
	f.GetCalls.Add(1)
	if f.Err != nil {
		return domains.DerivedDomainMap{}, f.Err
	}
	return f.Map, nil
}

func (f *FakeService) DraftDomains(_ context.Context, _ string) (domains.DomainDraft, error) {
	f.GetCalls.Add(1)
	if f.Err != nil {
		return domains.DomainDraft{}, f.Err
	}
	return domains.DraftFromMap(f.Map), nil
}

// NewExtraction is a small constructor that mirrors the common test shape:
// a single-source extraction declaring the named domains (paths defaulted
// to "<name>/").
func NewExtraction(src domains.Source, names ...string) domains.Extraction {
	ex := domains.Extraction{Source: src}
	for _, n := range names {
		ex.Domains = append(ex.Domains, domains.ExtractedDomain{Name: n, Paths: []string{n + "/"}})
	}
	return ex
}
