package review

import (
	"encoding/json"
	"fmt"
	"sync"

	"swarm-manager/internal/attemptstore"
)

var roundSaveMu sync.Mutex

// readRounds reads all review/round-*.json files from the item directory,
// sorted by round number ascending. Durable I/O itself is owned by
// attemptstore; this package only supplies review decoding and normalization.
func readRounds(itemDir string) ([]Round, error) {
	rounds, err := attemptstore.LoadRounds(itemDir, "review", decodeRound)
	if err != nil {
		return nil, err
	}
	for i := range rounds {
		normalized, normalizeErr := normalizeRound(rounds[i])
		if normalizeErr != nil {
			return nil, fmt.Errorf("normalize review round %d: %w", rounds[i].RoundNum, normalizeErr)
		}
		rounds[i] = normalized
	}
	return rounds, nil
}

// ReadLatestRound returns the most recent review projection and total count.
func ReadLatestRound(itemDir string) (*Round, int, error) {
	rounds, err := readRounds(itemDir)
	if err != nil {
		return nil, 0, err
	}
	if len(rounds) == 0 {
		return nil, 0, nil
	}
	return &rounds[len(rounds)-1], len(rounds), nil
}

// readRound loads a specific review projection. Returns nil when absent.
func readRound(itemDir string, roundNum int) (*Round, error) {
	round, err := attemptstore.LoadRound(itemDir, "review", roundNum, decodeRound)
	if err != nil || round == nil {
		return round, err
	}
	normalized, err := normalizeRound(*round)
	if err != nil {
		return nil, fmt.Errorf("normalize review round %d: %w", round.RoundNum, err)
	}
	return &normalized, nil
}

// ReadRound returns a durable review projection for a subject-specific
// decision boundary. File I/O remains owned by attemptstore.
func ReadRound(itemDir string, roundNum int) (*Round, error) { return readRound(itemDir, roundNum) }

// saveRound persists a review projection through the shared attempt store.
func saveRound(itemDir string, round Round) error {
	roundSaveMu.Lock()
	defer roundSaveMu.Unlock()
	current, err := attemptstore.LoadRound(itemDir, "review", round.RoundNum, decodeRound)
	if err != nil {
		return err
	}
	if current != nil {
		if err := current.TransitionStatus(round.Status); err != nil {
			return err
		}
		if err := validateRequestThreadSave(current.RequestThreads, round.RequestThreads); err != nil {
			return fmt.Errorf("validate review request thread transitions: %w", err)
		}
	} else if err := validateRequestThreadSave(nil, round.RequestThreads); err != nil {
		return fmt.Errorf("validate new review request threads: %w", err)
	}
	return attemptstore.SaveRound(itemDir, "review", round)
}

// nextRoundNumber returns the next sequential durable attempt number.
func nextRoundNumber(itemDir string) (int, error) {
	return attemptstore.NextRoundNumber(itemDir, "review")
}

func loadCapture(itemDir, relativePath string) ([]byte, error) {
	return attemptstore.LoadCapture(itemDir, "review", relativePath)
}

func decodeRound(data []byte) (Round, error) {
	var round Round
	return round, json.Unmarshal(data, &round)
}
