package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var requirementsCache sync.Map

type requirementsCacheEntry struct {
	groups   []RequirementGroup
	loadedAt time.Time
}

func loadRequirementsForEntity(entityType, entityName string) ([]RequirementGroup, error) {
	key := fmt.Sprintf("%s:%s", entityType, entityName)
	if entry, ok := requirementsCache.Load(key); ok {
		cacheEntry := entry.(requirementsCacheEntry)
		if time.Since(cacheEntry.loadedAt) < 2*time.Minute {
			return cacheEntry.groups, nil
		}
	}

	vrooliRoot, err := getVrooliRoot()
	if err != nil {
		return nil, err
	}
	baseDir := filepath.Join(vrooliRoot, entityType+"s", entityName, "requirements")
	indexPath := filepath.Join(baseDir, "index.json")
	if _, err := os.Stat(indexPath); errors.Is(err, os.ErrNotExist) {
		return []RequirementGroup{}, nil
	}

	groups, err := parseRequirementGroups(baseDir, "index.json", map[string]bool{})
	if err != nil {
		return nil, err
	}

	requirementsCache.Store(key, requirementsCacheEntry{groups: groups, loadedAt: time.Now()})
	return groups, nil
}

func parseRequirementGroups(baseDir, relPath string, visited map[string]bool) ([]RequirementGroup, error) {
	absPath := filepath.Join(baseDir, relPath)
	if visited[absPath] {
		return nil, fmt.Errorf("circular requirement import detected: %s", relPath)
	}
	visited[absPath] = true

	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	var file requirementsFile
	if err := json.Unmarshal(content, &file); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", relPath, err)
	}

	group := RequirementGroup{
		ID:       filepath.ToSlash(relPath),
		Name:     groupNameFromPath(relPath),
		FilePath: filepath.ToSlash(absPath),
	}
	if desc, ok := file.Metadata["description"].(string); ok {
		group.Description = desc
	}

	for _, req := range file.Requirements {
		group.Requirements = append(group.Requirements, RequirementRecord{
			ID:          req.ID,
			Category:    req.Category,
			PRDRef:      req.PRDRef,
			Title:       req.Title,
			Description: req.Description,
			Status:      req.Status,
			Criticality: req.Criticality,
			FilePath:    filepath.ToSlash(absPath),
			Validations: req.Validations,
		})
	}

	for _, child := range file.Imports {
		childGroups, err := parseRequirementGroups(baseDir, child, visited)
		if err != nil {
			return nil, err
		}
		group.Children = append(group.Children, childGroups...)
	}

	if relPath == "index.json" {
		return append([]RequirementGroup{group}, group.Children...), nil
	}
	return []RequirementGroup{group}, nil
}

func groupNameFromPath(relPath string) string {
	base := filepath.Base(relPath)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
