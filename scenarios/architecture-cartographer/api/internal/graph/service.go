package graph

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/vrooli/api-core/schedule"
)

// Service is the application-layer surface for graph operations.
type Service interface {
	ExtractGraph(ctx context.Context, req ExtractGraphInput) (GraphSnapshot, bool, error)
	GetSnapshot(ctx context.Context, id string) (GraphSnapshot, error)
	LatestSnapshotMeta(ctx context.Context, scenario string) (GraphSnapshotMeta, error)
	ListSnapshots(ctx context.Context, f ListSnapshotsFilter) (SnapshotPage, error)
	ClearSnapshots(ctx context.Context, scenario string, dryRun bool) (int, bool, error)

	// PreviewSnapshotRetention reports reclaimable snapshot storage without
	// deleting anything.
	PreviewSnapshotRetention(ctx context.Context, keepPerScenario int) (SnapshotRetentionPreview, error)

	// ApplySnapshotRetention prunes beyond the retention floor.
	ApplySnapshotRetention(ctx context.Context, keepPerScenario int) (RetentionResult, error)
}

// SnapshotRetentionPreview is the non-destructive view of what retention
// would remove.
type SnapshotRetentionPreview struct {
	ReclaimableBytes int64
	ReclaimableRows  int
	KeepPerScenario  int
	TotalSnapshots   int
	Scenarios        []ScenarioSnapshotCount
}

// ScenarioSnapshotCount is one scenario's snapshot inventory.
type ScenarioSnapshotCount struct {
	Scenario         string
	SnapshotCount    int
	ReclaimableCount int
}

// ExtractGraphInput is the explicit input DTO for Service.ExtractGraph.
type ExtractGraphInput struct {
	Scenario       string
	Languages      []Language
	IdempotencyKey string
	// When true, the TS adapter (Name()=="typescript") is skipped before
	// extraction; its name lands in GraphSnapshot.SkippedAdapters so the
	// audit layer can mark the run as partial.
	SkipTS bool
	// SkipPersistence keeps one-off repository-wide control-plane analyses from
	// materializing a multi-hundred-megabyte snapshot in the scenario cache.
	// The returned snapshot is still fully normalized for the caller.
	SkipPersistence bool
}

type service struct {
	repo     Repository
	adapters []CodeGraphAdapter
	clock    schedule.Clock
}

// NewService constructs the production Service. Adapters in priority
// order; the first adapter that supports the requested language wins.
func NewService(repo Repository, clk schedule.Clock, adapters ...CodeGraphAdapter) Service {
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
	var skipped []string
	for _, adapter := range s.adapters {
		if !adapterSupports(adapter, in.Languages) {
			continue
		}
		if in.SkipTS && adapter.Name() == "typescript" {
			skipped = append(skipped, adapter.Name())
			continue
		}
		raw, err := adapter.Extract(ctx, scenario)
		if err != nil {
			// Graceful degradation: when a backing producer scenario is
			// not running OR returns unimplemented (e.g., tscg hitting
			// a pnpm workspace it does not yet support), skip that
			// language rather than failing the whole cross-language
			// extract. Any other error (invalid input, timeout,
			// internal) still aborts.
			var ie IntegrationError
			if errors.As(err, &ie) && (ie.Kind == "scenario_unreachable" || ie.Kind == "unimplemented") {
				skipped = append(skipped, adapter.Name())
				continue
			}
			return GraphSnapshot{}, false, err
		}
		skipped = append(skipped, raw.SkippedAdapters...)
		combined.SourceFingerprint += raw.SourceFingerprint + ";"
		combined.Languages = append(combined.Languages, raw.Languages...)
		combined.Files = append(combined.Files, raw.Files...)
		combined.Packages = append(combined.Packages, raw.Packages...)
		combined.Symbols = append(combined.Symbols, raw.Symbols...)
		combined.Imports = append(combined.Imports, raw.Imports...)
		combined.ExtractionProfiles = append(combined.ExtractionProfiles, raw.ExtractionProfiles...)
		combined.OmittedInformation = append(combined.OmittedInformation, raw.OmittedInformation...)
		combined.ExtractionMS += raw.ExtractionMS
	}
	combined.ExtractionMS = time.Since(start).Milliseconds()
	combined.SkippedAdapters = skipped

	snap := Normalize(scenario, combined)
	snap.ExtractedAt = s.clock.Now().UTC()

	if !in.SkipPersistence {
		// Cache hit?
		if existing, err := s.repo.FindByHash(ctx, snap.Scenario, snap.ContentHash); err == nil {
			return existing, true, nil
		} else {
			var notFound ErrSnapshotNotFound
			if !errors.As(err, &notFound) {
				return GraphSnapshot{}, false, err
			}
		}
	}
	if in.SkipPersistence {
		return snap, false, nil
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

func (s *service) LatestSnapshotMeta(ctx context.Context, scenario string) (GraphSnapshotMeta, error) {
	return s.repo.LatestSnapshotMeta(ctx, scenario)
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
