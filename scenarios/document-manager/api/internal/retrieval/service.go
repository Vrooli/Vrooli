package retrieval

import (
	"context"
	"sort"
	"strings"
)

func NewService(repo Repository, vector VectorStore) Service {
	return Service{Repo: repo, Vector: vector}
}

func (s Service) Query(ctx context.Context, query Query) (Response, error) {
	candidates, err := s.Repo.Candidates(ctx, query)
	if err != nil {
		return Response{}, err
	}
	if query.Limit <= 0 {
		query.Limit = 20
	}
	// Candidates are privacy- and collection-filtered before either score leg
	// runs. No post-ranking security filter exists by construction.
	lexical := make(map[string]float64, len(candidates))
	terms := strings.Fields(strings.ToLower(query.Text))
	for _, unit := range candidates {
		score := 0.0
		body := strings.ToLower(unit.Text)
		for _, term := range terms {
			if strings.Contains(body, term) {
				score++
			}
		}
		if score > 0 {
			lexical[unit.ID] = score
		}
	}
	semantic := map[string]float64{}
	partial := len(query.Vector) == 0 || s.Vector == nil
	if !partial {
		semantic = s.Vector.Similar(ctx, query.Vector, candidates, query.Limit)
	}
	lexRank := rank(lexical)
	semRank := rank(semantic)
	byID := make(map[string]Unit, len(candidates))
	for _, unit := range candidates {
		byID[unit.ID] = unit
	}
	scores := make(map[string]float64)
	for id, n := range lexRank {
		scores[id] += 1.0 / float64(60+n)
	}
	for id, n := range semRank {
		scores[id] += 1.0 / float64(60+n)
	}
	ids := make([]string, 0, len(scores))
	for id := range scores {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		if scores[ids[i]] == scores[ids[j]] {
			return ids[i] < ids[j]
		}
		return scores[ids[i]] > scores[ids[j]]
	})
	if len(ids) > query.Limit {
		ids = ids[:query.Limit]
	}
	out := Response{Partial: partial, Results: make([]Result, 0, len(ids))}
	for _, id := range ids {
		unit := byID[id]
		out.Results = append(out.Results, Result{UnitID: unit.ID, DocumentHash: unit.DocumentHash, AnchorURI: unit.AnchorURI, Score: scores[id]})
	}
	return out, nil
}

func rank(scores map[string]float64) map[string]int {
	ids := make([]string, 0, len(scores))
	for id := range scores {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		if scores[ids[i]] == scores[ids[j]] {
			return ids[i] < ids[j]
		}
		return scores[ids[i]] > scores[ids[j]]
	})
	out := make(map[string]int, len(ids))
	for i, id := range ids {
		out[id] = i + 1
	}
	return out
}
