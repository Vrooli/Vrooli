package scenarios

import (
	"context"
	"database/sql"
	"log"
	"sort"
	"strings"
)

type scenarioSummaryStore interface {
	List(ctx context.Context) ([]ScenarioSummary, error)
	Get(ctx context.Context, scenario string) (*ScenarioSummary, error)
}

// ScenarioLister exposes scenario metadata from the Vrooli CLI.
type ScenarioLister interface {
	ListScenarios(ctx context.Context) ([]ScenarioMetadata, error)
}

// ScenarioMetadata captures high-level scenario info returned by the CLI.
type ScenarioMetadata struct {
	Name        string
	Description string
	Status      string
	Tags        []string
}

// ScenarioDirectoryService orchestrates catalog lookups.
type ScenarioDirectoryService struct {
	repo   scenarioSummaryStore
	lister ScenarioLister
}

func NewScenarioDirectoryService(repo scenarioSummaryStore, lister ScenarioLister) *ScenarioDirectoryService {
	return &ScenarioDirectoryService{repo: repo, lister: lister}
}

// ListSummaries returns every tracked scenario with queue + execution metadata and hydrates
// missing scenarios with CLI metadata so the catalog includes the entire workspace.
func (s *ScenarioDirectoryService) ListSummaries(ctx context.Context) ([]ScenarioSummary, error) {
	if s == nil || s.repo == nil {
		return nil, sql.ErrConnDone
	}

	summaries, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	if s.lister == nil {
		return summaries, nil
	}

	metadata, err := s.lister.ListScenarios(ctx)
	if err != nil {
		log.Printf("scenario lister failed: %v", err)
		return summaries, nil
	}

	if len(metadata) == 0 {
		return summaries, nil
	}

	summaryMap := make(map[string]ScenarioSummary, len(summaries))
	for _, summary := range summaries {
		key := strings.ToLower(summary.ScenarioName)
		if key == "" {
			continue
		}
		summaryMap[key] = summary
	}

	merged := make([]ScenarioSummary, 0, len(metadata)+len(summaries))
	for _, meta := range metadata {
		name := strings.TrimSpace(meta.Name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if summary, ok := summaryMap[key]; ok {
			merged = append(merged, applyScenarioMetadata(summary, meta))
			delete(summaryMap, key)
			continue
		}
		merged = append(merged, applyScenarioMetadata(ScenarioSummary{
			ScenarioName:    name,
			PendingRequests: 0,
			TotalRequests:   0,
			TotalExecutions: 0,
		}, meta))
	}

	// Preserve any tracked scenarios not currently returned by the CLI.
	if len(summaryMap) > 0 {
		leftovers := make([]ScenarioSummary, 0, len(summaryMap))
		for _, summary := range summaryMap {
			leftovers = append(leftovers, summary)
		}
		sort.Slice(leftovers, func(i, j int) bool {
			return strings.ToLower(leftovers[i].ScenarioName) < strings.ToLower(leftovers[j].ScenarioName)
		})
		merged = append(merged, leftovers...)
	}

	return merged, nil
}

// GetSummary loads a single scenario summary by name and falls back to CLI metadata if needed.
func (s *ScenarioDirectoryService) GetSummary(ctx context.Context, scenario string) (*ScenarioSummary, error) {
	if s == nil || s.repo == nil {
		return nil, sql.ErrConnDone
	}
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return nil, sql.ErrNoRows
	}

	summary, err := s.repo.Get(ctx, scenario)
	if err == nil {
		if meta := s.lookupScenarioMetadata(ctx, scenario); meta != nil {
			result := applyScenarioMetadata(*summary, *meta)
			return &result, nil
		}
		return summary, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	meta := s.lookupScenarioMetadata(ctx, scenario)
	if meta == nil {
		return nil, err
	}

	placeholder := applyScenarioMetadata(ScenarioSummary{
		ScenarioName:    meta.Name,
		PendingRequests: 0,
		TotalRequests:   0,
		TotalExecutions: 0,
	}, *meta)
	return &placeholder, nil
}

func (s *ScenarioDirectoryService) lookupScenarioMetadata(ctx context.Context, scenario string) *ScenarioMetadata {
	if s.lister == nil {
		return nil
	}
	metadata, err := s.lister.ListScenarios(ctx)
	if err != nil {
		log.Printf("scenario metadata lookup failed: %v", err)
		return nil
	}
	target := strings.ToLower(strings.TrimSpace(scenario))
	for _, item := range metadata {
		if strings.ToLower(strings.TrimSpace(item.Name)) == target {
			meta := item
			return &meta
		}
	}
	return nil
}

func applyScenarioMetadata(summary ScenarioSummary, meta ScenarioMetadata) ScenarioSummary {
	summary.ScenarioDescription = strings.TrimSpace(meta.Description)
	summary.ScenarioStatus = strings.TrimSpace(meta.Status)
	if len(meta.Tags) > 0 {
		summary.ScenarioTags = append([]string(nil), meta.Tags...)
	} else {
		summary.ScenarioTags = nil
	}
	return summary
}
