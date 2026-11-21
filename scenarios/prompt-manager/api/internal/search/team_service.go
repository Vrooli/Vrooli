package search

import (
	"context"
	"strings"

	"prompt-manager/internal/store"
)

// TeamFileReader abstracts file operations for team content search.
type TeamFileReader interface {
	ListSharedFiles(ctx context.Context, id string) ([]store.TeamFileEntry, error)
	ReadSharedFile(ctx context.Context, id, path string) (string, error)
}

// TeamSearchService provides text search over teams.
type TeamSearchService struct {
	store    store.TeamStore
	relStore store.RelationStore
	files    TeamFileReader
}

// NewTeamSearchService creates a new team search service.
func NewTeamSearchService(teamStore store.TeamStore, relStore store.RelationStore, files TeamFileReader) *TeamSearchService {
	return &TeamSearchService{store: teamStore, relStore: relStore, files: files}
}

// Search performs text search across teams.
func (s *TeamSearchService) Search(ctx context.Context, query TeamSearchQuery) (*TeamSearchResponse, error) {
	teams, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}

	// Apply filters
	filtered := s.filterTeams(teams, query)

	// If no query text, return all filtered results
	if query.Query == "" {
		results := make([]TeamSearchResult, 0, len(filtered))
		for _, t := range filtered {
			result := s.toTeamSearchResult(ctx, t)
			results = append(results, result)
		}
		return &TeamSearchResponse{
			Results: results,
			Total:   len(results),
			Query:   query.Query,
		}, nil
	}

	queryLower := strings.ToLower(query.Query)
	var results []TeamSearchResult

	for _, t := range filtered {
		score := scoreTeam(t, queryLower)
		if score > 0 {
			result := s.toTeamSearchResult(ctx, t)
			result.Score = score
			result.Highlight = teamHighlight(t, queryLower)
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

	return &TeamSearchResponse{
		Results: results,
		Total:   len(results),
		Query:   query.Query,
	}, nil
}

// SearchContent performs line-level content search across team shared files.
func (s *TeamSearchService) SearchContent(ctx context.Context, query TeamContentSearchQuery) (*TeamContentSearchResponse, error) {
	trimmed := strings.TrimSpace(query.Query)
	if trimmed == "" {
		return &TeamContentSearchResponse{
			Matches: []TeamContentSearchMatch{},
			Total:   0,
			Query:   query.Query,
		}, nil
	}

	pattern, err := compileContentPattern(trimmed, ContentSearchQuery{
		CaseSensitive: query.CaseSensitive,
		WholeWord:     query.WholeWord,
		Regex:         query.Regex,
	})
	if err != nil {
		return nil, err
	}

	teams, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 200
	}

	matches := make([]TeamContentSearchMatch, 0)
	limitReached := false

	for _, team := range teams {
		files, err := s.files.ListSharedFiles(ctx, team.ID)
		if err != nil {
			continue
		}

		for _, f := range files {
			if f.IsDir {
				continue
			}

			content, err := s.files.ReadSharedFile(ctx, team.ID, f.Path)
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

				matches = append(matches, TeamContentSearchMatch{
					TeamID:      team.ID,
					TeamName:    team.DisplayName,
					File:        f.Path,
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

		if limitReached {
			break
		}
	}

	return &TeamContentSearchResponse{
		Matches: matches,
		Total:   len(matches),
		Query:   query.Query,
	}, nil
}

func (s *TeamSearchService) filterTeams(teams []store.Team, query TeamSearchQuery) []store.Team {
	if query.Enabled == nil {
		return teams
	}

	var filtered []store.Team
	for _, t := range teams {
		if *query.Enabled != t.Enabled {
			continue
		}
		filtered = append(filtered, t)
	}
	return filtered
}

func scoreTeam(t store.Team, queryLower string) float64 {
	var score float64

	nameLower := strings.ToLower(t.DisplayName)
	missionLower := strings.ToLower(t.Mission)

	if nameLower == queryLower {
		score += 10.0
	} else if strings.Contains(nameLower, queryLower) {
		score += 5.0
	}

	if strings.Contains(missionLower, queryLower) {
		score += 2.0
	}

	if strings.Contains(strings.ToLower(t.ID), queryLower) {
		score += 1.0
	}

	return score
}

func (s *TeamSearchService) toTeamSearchResult(ctx context.Context, t store.Team) TeamSearchResult {
	memberCount := 0
	if s.relStore != nil {
		if members, err := s.relStore.ListTeamMembers(ctx, t.ID); err == nil {
			memberCount = len(members)
		}
	}

	return TeamSearchResult{
		ID:          t.ID,
		DisplayName: t.DisplayName,
		Mission:     t.Mission,
		Enabled:     t.Enabled,
		MemberCount: memberCount,
	}
}

func teamHighlight(t store.Team, queryLower string) string {
	missionLower := strings.ToLower(t.Mission)
	if idx := strings.Index(missionLower, queryLower); idx >= 0 {
		return extractSnippet(t.Mission, idx, len(queryLower), 100)
	}
	return ""
}
