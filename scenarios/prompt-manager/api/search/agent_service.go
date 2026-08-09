package search

import (
	"context"
	"strings"

	"prompt-manager/store"
)

// AgentFileReader abstracts file operations for agent content search.
type AgentFileReader interface {
	ListFiles(ctx context.Context, id string) ([]store.AgentFileEntry, error)
	ReadFile(ctx context.Context, id, path string) (string, error)
}

// AgentSearchService provides text search over agents.
type AgentSearchService struct {
	store store.AgentStore
	files AgentFileReader
}

// NewAgentSearchService creates a new agent search service.
func NewAgentSearchService(agentStore store.AgentStore, files AgentFileReader) *AgentSearchService {
	return &AgentSearchService{store: agentStore, files: files}
}

// Search performs text search across agents.
func (s *AgentSearchService) Search(ctx context.Context, query AgentSearchQuery) (*AgentSearchResponse, error) {
	agents, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}

	// Apply filters
	filtered := s.filterAgents(agents, query)

	// If no query text, return all filtered results
	if query.Query == "" {
		results := make([]AgentSearchResult, 0, len(filtered))
		for _, a := range filtered {
			results = append(results, toAgentSearchResult(a))
		}
		return &AgentSearchResponse{
			Results: results,
			Total:   len(results),
			Query:   query.Query,
		}, nil
	}

	queryLower := strings.ToLower(query.Query)
	var results []AgentSearchResult

	for _, a := range filtered {
		score := scoreAgent(a, queryLower)
		if score > 0 {
			result := toAgentSearchResult(a)
			result.Score = score
			result.Highlight = agentHighlight(a, queryLower)
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

	return &AgentSearchResponse{
		Results: results,
		Total:   len(results),
		Query:   query.Query,
	}, nil
}

// SearchContent performs line-level content search across agent files.
func (s *AgentSearchService) SearchContent(ctx context.Context, query AgentContentSearchQuery) (*AgentContentSearchResponse, error) {
	trimmed := strings.TrimSpace(query.Query)
	if trimmed == "" {
		return &AgentContentSearchResponse{
			Matches: []AgentContentSearchMatch{},
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

	agents, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}

	tagSet := normalizeTagSet(query.Tags)

	limit := query.Limit
	if limit <= 0 {
		limit = 200
	}

	matches := make([]AgentContentSearchMatch, 0)
	limitReached := false

	for _, agent := range agents {
		if len(tagSet) > 0 && !agentHasAnyTag(agent, tagSet) {
			continue
		}

		files, err := s.files.ListFiles(ctx, agent.ID)
		if err != nil {
			continue
		}

		for _, f := range files {
			if f.IsDir {
				continue
			}

			content, err := s.files.ReadFile(ctx, agent.ID, f.Path)
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

				matches = append(matches, AgentContentSearchMatch{
					AgentID:     agent.ID,
					AgentName:   agent.DisplayName,
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

	return &AgentContentSearchResponse{
		Matches: matches,
		Total:   len(matches),
		Query:   query.Query,
	}, nil
}

func (s *AgentSearchService) filterAgents(agents []store.Agent, query AgentSearchQuery) []store.Agent {
	if query.Tag == "" && query.Status == "" {
		return agents
	}

	var filtered []store.Agent
	for _, a := range agents {
		if query.Status != "" && !strings.EqualFold(a.Status, query.Status) {
			continue
		}
		if query.Tag != "" {
			hasTag := false
			for _, t := range a.Tags {
				if strings.EqualFold(t, query.Tag) {
					hasTag = true
					break
				}
			}
			if !hasTag {
				continue
			}
		}
		filtered = append(filtered, a)
	}
	return filtered
}

func scoreAgent(a store.Agent, queryLower string) float64 {
	var score float64

	nameLower := strings.ToLower(a.DisplayName)
	descLower := strings.ToLower(a.Description)

	if nameLower == queryLower {
		score += 10.0
	} else if strings.Contains(nameLower, queryLower) {
		score += 5.0
	}

	if strings.Contains(descLower, queryLower) {
		score += 2.0
	}

	for _, tag := range a.Tags {
		if strings.ToLower(tag) == queryLower {
			score += 3.0
		} else if strings.Contains(strings.ToLower(tag), queryLower) {
			score += 1.5
		}
	}

	if strings.Contains(strings.ToLower(a.ID), queryLower) {
		score += 1.0
	}

	return score
}

func toAgentSearchResult(a store.Agent) AgentSearchResult {
	return AgentSearchResult{
		ID:          a.ID,
		DisplayName: a.DisplayName,
		Description: a.Description,
		Status:      a.Status,
		Tags:        a.Tags,
	}
}

func agentHighlight(a store.Agent, queryLower string) string {
	descLower := strings.ToLower(a.Description)
	if idx := strings.Index(descLower, queryLower); idx >= 0 {
		return extractSnippet(a.Description, idx, len(queryLower), 100)
	}
	return ""
}

func agentHasAnyTag(a store.Agent, tagSet map[string]bool) bool {
	for _, tag := range a.Tags {
		if tagSet[tag] {
			return true
		}
	}
	return false
}
