// Package search provides skill search functionality.
package search

import (
	"strings"

	"prompt-manager/skills"
)

// Service handles search logic.
type Service struct {
	store skills.SkillStore
}

// NewService creates a new search service.
func NewService(store skills.SkillStore) *Service {
	return &Service{store: store}
}

// Search performs full-text search across skills.
func (s *Service) Search(query SearchQuery) (*SearchResponse, error) {
	allSkills, err := s.store.GetAll()
	if err != nil {
		return nil, err
	}

	// Apply filters first
	filtered := skills.Filter(allSkills, skills.FilterOptions{
		Tag:    query.Tag,
		Folder: query.Folder,
	})

	// If no query text, return all filtered results
	if query.Query == "" {
		results := make([]SearchResult, 0, len(filtered))
		for _, skill := range filtered {
			result := s.toSearchResult(skill)
			results = append(results, result)
		}
		return &SearchResponse{
			Results: results,
			Total:   len(results),
			Query:   query.Query,
		}, nil
	}

	// Perform text search
	queryLower := strings.ToLower(query.Query)
	var results []SearchResult

	for _, skill := range filtered {
		score := s.calculateScore(skill, queryLower)
		if score > 0 {
			result := s.toSearchResult(skill)
			result.Score = score
			result.Highlight = s.extractHighlight(skill, queryLower)
			results = append(results, result)
		}
	}

	// Sort by score (highest first)
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return &SearchResponse{
		Results: results,
		Total:   len(results),
		Query:   query.Query,
	}, nil
}

// calculateScore returns a relevance score for a skill matching the query.
func (s *Service) calculateScore(skill skills.Metadata, queryLower string) float64 {
	var score float64

	nameLower := strings.ToLower(skill.Name)
	descLower := strings.ToLower(skill.Description)

	// Exact name match = highest score
	if nameLower == queryLower {
		score += 10.0
	} else if strings.Contains(nameLower, queryLower) {
		score += 5.0
	}

	// Description match
	if strings.Contains(descLower, queryLower) {
		score += 2.0
	}

	// Tag matches
	for _, tag := range skill.Tags {
		if strings.ToLower(tag) == queryLower {
			score += 3.0
		} else if strings.Contains(strings.ToLower(tag), queryLower) {
			score += 1.5
		}
	}

	// Check content (need to load it)
	folder, filename := s.extractFolderAndFile(skill.File)
	if content, err := s.store.GetContent(folder, filename); err == nil {
		contentLower := strings.ToLower(content)
		if strings.Contains(contentLower, queryLower) {
			// Count occurrences
			occurrences := strings.Count(contentLower, queryLower)
			score += float64(occurrences) * 0.5
			if occurrences > 10 {
				score += 2.0 // Cap bonus for many occurrences
			}
		}
	}

	return score
}

// extractHighlight extracts a snippet showing where the query matches.
func (s *Service) extractHighlight(skill skills.Metadata, queryLower string) string {
	// First try description
	descLower := strings.ToLower(skill.Description)
	if idx := strings.Index(descLower, queryLower); idx >= 0 {
		return extractSnippet(skill.Description, idx, len(queryLower), 100)
	}

	// Try content
	folder, filename := s.extractFolderAndFile(skill.File)
	if content, err := s.store.GetContent(folder, filename); err == nil {
		contentLower := strings.ToLower(content)
		if idx := strings.Index(contentLower, queryLower); idx >= 0 {
			return extractSnippet(content, idx, len(queryLower), 100)
		}
	}

	return ""
}

// extractSnippet extracts a snippet around the match.
func extractSnippet(text string, matchIdx, matchLen, contextLen int) string {
	start := matchIdx - contextLen/2
	if start < 0 {
		start = 0
	}
	end := matchIdx + matchLen + contextLen/2
	if end > len(text) {
		end = len(text)
	}

	snippet := text[start:end]

	// Add ellipsis if we truncated
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(text) {
		snippet = snippet + "..."
	}

	return snippet
}

// toSearchResult converts a skill metadata to a search result.
func (s *Service) toSearchResult(skill skills.Metadata) SearchResult {
	folder, _ := s.extractFolderAndFile(skill.File)

	return SearchResult{
		ID:          skill.ID,
		Name:        skill.Name,
		Description: skill.Description,
		Folder:      folder,
		Tags:        skill.Tags,
		Modes:       skill.Modes,
	}
}

// extractFolderAndFile splits a file path like "local/skill.md" into folder and filename.
func (s *Service) extractFolderAndFile(file string) (folder, filename string) {
	parts := strings.SplitN(file, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", file
}
