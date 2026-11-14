// Package search provides skill search functionality.
package search

import (
	"errors"
	"regexp"
	"strings"

	"prompt-manager/internal/skills"
)

// Service handles search logic.
type Service struct {
	store skills.SkillStore
}

// ErrInvalidPattern indicates the search pattern could not be compiled.
var ErrInvalidPattern = errors.New("invalid search pattern")

// NewService creates a new search service.
func NewService(store skills.SkillStore) *Service {
	return &Service{store: store}
}

// Search performs quick metadata search across skills.
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

// SearchContent performs content-only search across skill bodies.
func (s *Service) SearchContent(query ContentSearchQuery) (*ContentSearchResponse, error) {
	trimmed := strings.TrimSpace(query.Query)
	if trimmed == "" {
		return &ContentSearchResponse{
			Matches: []ContentSearchMatch{},
			Total:   0,
			Query:   query.Query,
		}, nil
	}

	pattern, err := compileContentPattern(trimmed, query)
	if err != nil {
		return nil, err
	}

	allSkills, err := s.store.GetAll()
	if err != nil {
		return nil, err
	}

	tagSet := normalizeTagSet(query.Tags)
	folderSet := normalizeFolderSet(query.Folders)

	limit := query.Limit
	if limit <= 0 {
		limit = 200
	}

	matches := make([]ContentSearchMatch, 0)
	limitReached := false

	for _, skill := range allSkills {
		folder, filename := s.extractFolderAndFile(skill.File)
		if len(folderSet) > 0 && !folderSet[folder] {
			continue
		}
		if len(tagSet) > 0 && !skillHasAnyTag(skill, tagSet) {
			continue
		}

		content, err := s.store.GetContent(folder, filename)
		if err != nil {
			continue
		}

		lines := strings.Split(content, "\n")
		for i, line := range lines {
			indices := pattern.FindAllStringIndex(line, -1)
			if len(indices) == 0 {
				continue
			}

			ranges := make([]MatchRange, 0, len(indices))
			for _, idx := range indices {
				if len(idx) != 2 || idx[0] == idx[1] {
					continue
				}
				ranges = append(ranges, MatchRange{Start: idx[0], End: idx[1]})
			}
			if len(ranges) == 0 {
				continue
			}

			matches = append(matches, ContentSearchMatch{
				SkillID:     skill.ID,
				SkillName:   skill.Name,
				File:        skill.File,
				Folder:      folder,
				LineNumber:  i + 1,
				Line:        line,
				MatchRanges: ranges,
			})

			if len(matches) >= limit {
				limitReached = true
				break
			}
		}

		if limitReached {
			break
		}
	}

	return &ContentSearchResponse{
		Matches: matches,
		Total:   len(matches),
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

	// Mode matches
	for _, mode := range skill.Modes {
		modeLower := strings.ToLower(mode)
		if modeLower == queryLower {
			score += 2.5
		} else if strings.Contains(modeLower, queryLower) {
			score += 1.25
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

func compileContentPattern(query string, opts ContentSearchQuery) (*regexp.Regexp, error) {
	pattern := query
	if !opts.Regex {
		pattern = regexp.QuoteMeta(pattern)
	}
	if opts.WholeWord {
		pattern = `\b` + pattern + `\b`
	}
	if !opts.CaseSensitive {
		pattern = `(?i)` + pattern
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, errors.Join(ErrInvalidPattern, err)
	}
	return re, nil
}

func normalizeTagSet(tags []string) map[string]bool {
	if len(tags) == 0 {
		return nil
	}
	set := make(map[string]bool, len(tags))
	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed == "" {
			continue
		}
		set[trimmed] = true
	}
	return set
}

func normalizeFolderSet(folders []string) map[string]bool {
	if len(folders) == 0 {
		return nil
	}
	set := make(map[string]bool, len(folders))
	for _, folder := range folders {
		trimmed := strings.TrimSpace(folder)
		if trimmed == "" {
			continue
		}
		set[trimmed] = true
	}
	return set
}

func skillHasAnyTag(skill skills.Metadata, tagSet map[string]bool) bool {
	for _, tag := range skill.Tags {
		if tagSet[tag] {
			return true
		}
	}
	return false
}
