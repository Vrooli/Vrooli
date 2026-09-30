package main

import (
	"context"
	"fmt"

	"swarm-manager/internal/aisearch"
	"swarm-manager/internal/goals"
	"swarm-manager/internal/scenarios"
)

// captureGroundingProvider assembles read-only project context for capture
// classification. Each source is independent: one unavailable subsystem is
// represented in the packet rather than preventing intake altogether.
type captureGroundingProvider struct {
	search    *aisearch.Service
	goals     *goals.Service
	scenarios scenarios.Source
}

func (p captureGroundingProvider) BuildCaptureGrounding(ctx context.Context, query string) (map[string]any, error) {
	packet := map[string]any{
		"status":              "complete",
		"semantic_neighbours": []any{},
		"active_goals":        []any{},
		"scenario_roster":     []any{},
	}
	degraded := []string{}

	if p.search == nil {
		degraded = append(degraded, "semantic search is unavailable")
	} else {
		neighbours := make([]any, 0)
		for _, entity := range []aisearch.EntityType{aisearch.EntityBacklog, aisearch.EntityGoal, aisearch.EntityRecord} {
			response, err := p.search.Search(ctx, aisearch.AISearchRequest{Query: query, Entity: entity, Limit: 5})
			if err != nil {
				degraded = append(degraded, fmt.Sprintf("semantic %s: %v", entity, err))
				continue
			}
			for _, result := range response.Results {
				neighbours = append(neighbours, map[string]any{
					"entity":  result.Entity,
					"id":      result.ID,
					"score":   result.Score,
					"payload": result.Payload,
				})
			}
		}
		packet["semantic_neighbours"] = neighbours
	}

	if p.goals == nil {
		degraded = append(degraded, "goal service is unavailable")
	} else if listed, err := p.goals.List(); err != nil {
		degraded = append(degraded, fmt.Sprintf("goals: %v", err))
	} else {
		active := make([]any, 0, len(listed))
		for _, goal := range listed {
			if goal.Goal.Status != goals.StatusActive {
				continue
			}
			milestones := make([]any, 0, len(goal.Goal.Milestones))
			for _, milestone := range goal.Goal.Milestones {
				milestones = append(milestones, map[string]any{
					"name":                milestone.Name,
					"title":               milestone.Title,
					"description":         milestone.Description,
					"items":               milestone.Items,
					"acceptance_criteria": milestone.AcceptanceCriteria,
					"depends_on":          milestone.DependsOn,
					"spawned_from":        milestone.SpawnedFrom,
				})
			}
			active = append(active, map[string]any{
				"name":        goal.Goal.Name,
				"title":       goal.Goal.Title,
				"description": goal.Goal.Description,
				"targets":     goal.Goal.Targets,
				"milestones":  milestones,
			})
		}
		packet["active_goals"] = active
	}

	if p.scenarios == nil {
		degraded = append(degraded, "scenario roster is unavailable")
	} else if roster, err := p.scenarios.List(ctx); err != nil {
		degraded = append(degraded, fmt.Sprintf("scenario roster: %v", err))
	} else {
		entries := make([]any, 0, len(roster))
		for _, scenario := range roster {
			entries = append(entries, map[string]any{
				"name":        scenario.Name,
				"description": scenario.Description,
				"status":      scenario.Status,
				"tags":        scenario.Tags,
			})
		}
		packet["scenario_roster"] = entries
	}

	if len(degraded) > 0 {
		packet["status"] = "degraded"
		packet["degraded_reasons"] = degraded
	}
	return packet, nil
}
