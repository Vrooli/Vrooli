package evidence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"swarm-manager/internal/attemptstore"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/review"
)

// LoadReviewRoundSources reads review projections only from registered backlog
// kinds. It deliberately does not walk arbitrary data directories: a migration
// must have a small, inspectable source boundary.
func LoadReviewRoundSources(dataRoot string) ([]ReviewRoundSource, error) {
	kinds := make([]backlog.BacklogKind, 0, len(backlog.KindConfig))
	for kind := range backlog.KindConfig {
		kinds = append(kinds, kind)
	}
	sort.Slice(kinds, func(i, j int) bool { return kinds[i] < kinds[j] })

	sources := make([]ReviewRoundSource, 0)
	for _, kind := range kinds {
		kindRoot := filepath.Join(dataRoot, backlog.KindConfig[kind].Dir)
		items, err := os.ReadDir(kindRoot)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read %s review source root: %w", kind, err)
		}
		for _, item := range items {
			if !item.IsDir() {
				continue
			}
			rounds, err := attemptstore.LoadRounds(filepath.Join(kindRoot, item.Name()), "review", decodeReviewRound)
			if err != nil {
				return nil, fmt.Errorf("read %s/%s review rounds: %w", kind, item.Name(), err)
			}
			for _, round := range rounds {
				sources = append(sources, ReviewRoundSource{Kind: string(kind), Name: item.Name(), Round: round})
			}
		}
	}
	sort.Slice(sources, func(i, j int) bool {
		left := fmt.Sprintf("%s/%s/%09d", sources[i].Kind, sources[i].Name, sources[i].Round.RoundNum)
		right := fmt.Sprintf("%s/%s/%09d", sources[j].Kind, sources[j].Name, sources[j].Round.RoundNum)
		return left < right
	})
	return sources, nil
}

func decodeReviewRound(data []byte) (review.Round, error) {
	var round review.Round
	return round, json.Unmarshal(data, &round)
}
