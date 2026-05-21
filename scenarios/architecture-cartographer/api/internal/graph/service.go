package graph

import (
	"context"
	"errors"
	"strings"
	"time"

	"architecture-cartographer/internal/clock"
)

// Service is the application-layer surface for graph operations.
type Service interface {
	ExtractGraph(ctx context.Context, req ExtractGraphInput) (GraphSnapshot, bool, error)
	GetSnapshot(ctx context.Context, id string) (GraphSnapshot, error)
	ListSnapshots(ctx context.Context, f ListSnapshotsFilter) (SnapshotPage, error)
	ClearSnapshots(ctx context.Context, scenario string, dryRun bool) (int, bool, error)
}

// ExtractGraphInput is the explicit input DTO for Service.ExtractGraph.
type ExtractGraphInput struct {
	Scenario       string
	Languages      []Language
	IdempotencyKey string
}

type service struct {
	repo     Repository
	adapters []CodeGraphAdapter
	clock    clock.Clock
}

// NewService constructs the production Service. Adapters in priority
// order; the first adapter that supports the requested language wins.
func NewService(repo Repository, clk clock.Clock, adapters ...CodeGraphAdapter) Service {
	return &service{repo: repo, adapters: adapters, clock: clk}
}

var _ Service = (*service)(nil)

func (s *service) ExtractGraph(ctx context.Context, in ExtractGraphInput) (GraphSnapshot, bool, error) {
	scenario := strings.TrimSpace(in.Scenario)
	if scenario == "" {
		return GraphSnapshot{}, false, ErrInvalidExtractRequest{Field: "scenario", Reason: "required"}
	}
	if len(s.adapters) == 0 {
		return GraphSnapshot{}, false, IntegrationError{
			Kind:     "no_adapter_registered",
			Scenario: scenario,
			Cause:    errors.New("no CodeGraphAdapter registered"),
		}
	}

	start := s.clock.Now()
	var combined RawGraph
	for _, adapter := range s.adapters {
		if !adapterSupports(adapter, in.Languages) {
			continue
		}
		raw, err := adapter.Extract(ctx, scenario)
		if err != nil {
			return GraphSnapshot{}, false, err
		}
		combined.Languages = append(combined.Languages, raw.Languages...)
		combined.Files = append(combined.Files, raw.Files...)
		combined.Packages = append(combined.Packages, raw.Packages...)
		combined.Symbols = append(combined.Symbols, raw.Symbols...)
		combined.Imports = append(combined.Imports, raw.Imports...)
		combined.ExtractionMS += raw.ExtractionMS
	}
	combined.ExtractionMS = time.Since(start).Milliseconds()

	snap := Normalize(scenario, combined)
	snap.ExtractedAt = s.clock.Now().UTC()

	// Cache hit?
	if existing, err := s.repo.FindByHash(ctx, snap.Scenario, snap.ContentHash); err == nil {
		return existing, true, nil
	} else {
		var notFound ErrSnapshotNotFound
		if !errors.As(err, &notFound) {
			return GraphSnapshot{}, false, err
		}
	}

	persisted, err := s.repo.SaveSnapshot(ctx, snap)
	if err != nil {
		return GraphSnapshot{}, false, err
	}
	return persisted, false, nil
}

func (s *service) GetSnapshot(ctx context.Context, id string) (GraphSnapshot, error) {
	return s.repo.GetSnapshot(ctx, id)
}

func (s *service) ListSnapshots(ctx context.Context, f ListSnapshotsFilter) (SnapshotPage, error) {
	return s.repo.ListSnapshots(ctx, f)
}

func (s *service) ClearSnapshots(ctx context.Context, scenario string, dryRun bool) (int, bool, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return 0, dryRun, ErrInvalidExtractRequest{Field: "scenario", Reason: "required"}
	}
	if dryRun {
		page, err := s.repo.ListSnapshots(ctx, ListSnapshotsFilter{Scenario: scenario, PageSize: 100000})
		if err != nil {
			return 0, true, err
		}
		return len(page.Snapshots), true, nil
	}
	n, err := s.repo.ClearSnapshots(ctx, scenario)
	return n, false, err
}

func adapterSupports(a CodeGraphAdapter, requested []Language) bool {
	if len(requested) == 0 {
		return true
	}
	supported := a.SupportedLanguages()
	for _, want := range requested {
		for _, have := range supported {
			if want == have {
				return true
			}
		}
	}
	return false
}
