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

	baseName := filepath.Base(relPath)
	isModuleFile := baseName == "module.json"

	group := RequirementGroup{
		ID:       filepath.ToSlash(relPath),
		Name:     groupNameFromPath(relPath, file),
		FilePath: filepath.ToSlash(absPath),
		IsModule: isModuleFile,
	}

	// Use explicit description from module files, or fall back to metadata
	if file.Description != "" {
		group.Description = file.Description
	} else if desc, ok := file.Metadata["description"].(string); ok {
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
			Criticality: getCriticalityFromPRDRef(req.PRDRef),
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

	// Always return just the group itself - children are nested within it
	return []RequirementGroup{group}, nil
}

func groupNameFromPath(relPath string, file requirementsFile) string {
	baseName := filepath.Base(relPath)

	// For module.json files, prefer the title field
	if baseName == "module.json" && file.Title != "" {
		return file.Title
	}

	// For index.json at root, use "Requirements"
	if relPath == "index.json" {
		return "Requirements"
	}

	// Otherwise use the base filename without extension
	base := filepath.Base(relPath)
	name := strings.TrimSuffix(base, filepath.Ext(base))

	// If it's "module" or "index", try to use the parent directory name instead
	if name == "module" || name == "index" {
		dir := filepath.Dir(relPath)
		if dir != "." && dir != "/" {
			parentDir := filepath.Base(dir)
			// Clean up numbered prefixes like "01-template-management" -> "Template Management"
			cleaned := cleanFolderName(parentDir)
			if cleaned != "" {
				return cleaned
			}
		}
	}

	return name
}

// cleanFolderName removes number prefixes and converts kebab-case to Title Case
func cleanFolderName(folderName string) string {
	// Remove leading numbers and dash (e.g., "01-template-management" -> "template-management")
	parts := strings.SplitN(folderName, "-", 2)
	if len(parts) == 2 && isNumeric(parts[0]) {
		folderName = parts[1]
	}

	// Convert kebab-case to space-separated words
	words := strings.Split(folderName, "-")

	// Title case each word
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}

	return strings.Join(words, " ")
}

func isNumeric(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}
