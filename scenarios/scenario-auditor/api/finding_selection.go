package main

import (
	"sort"
	"strings"
)

func filterFindingsBySeverity[T any](items []T, allowed map[string]struct{}, severityOf func(T) string) []T {
	if len(items) == 0 {
		return nil
	}
	filtered := make([]T, 0, len(items))
	for _, item := range items {
		severity := strings.ToLower(strings.TrimSpace(severityOf(item)))
		if len(allowed) > 0 {
			if _, ok := allowed[severity]; !ok {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func uniqueFindingIDs[T any](items []T, idOf func(T) string) []string {
	ids := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		id := strings.TrimSpace(idOf(item))
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func orderedUniqueFindings[T any](items []T, strategy string, idOf, severityOf func(T) string) ([]string, map[string]string) {
	if len(items) == 0 {
		return nil, map[string]string{}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return compareSeverities(severityOf(items[i]), severityOf(items[j]), strategy) < 0
	})
	severities := make(map[string]string, len(items))
	ordered := make([]string, 0, len(items))
	for _, item := range items {
		id := strings.TrimSpace(idOf(item))
		if id == "" {
			continue
		}
		if _, exists := severities[id]; exists {
			continue
		}
		severities[id] = strings.ToLower(strings.TrimSpace(severityOf(item)))
		ordered = append(ordered, id)
	}
	return ordered, severities
}
