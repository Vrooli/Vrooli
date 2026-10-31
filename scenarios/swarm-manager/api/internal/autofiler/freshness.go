package autofiler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"swarm-manager/internal/storage"
)

const DefaultReviewFreshness = 6 * time.Hour

type ReviewFreshnessStore struct {
	path string
}

func NewReviewFreshnessStore(dataRoot string) *ReviewFreshnessStore {
	return &ReviewFreshnessStore{path: filepath.Join(dataRoot, "autofiler", "review_freshness.json")}
}

func NewReviewFreshnessStorePath(path string) *ReviewFreshnessStore {
	return &ReviewFreshnessStore{path: path}
}

func (s *ReviewFreshnessStore) Fresh(scenario string, window time.Duration, now time.Time) (bool, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return false, nil
	}
	if window <= 0 {
		window = DefaultReviewFreshness
	}
	markers, err := s.LoadAll()
	if err != nil {
		return false, err
	}
	ranAt, ok := markers[scenario]
	if !ok {
		return false, nil
	}
	return now.UTC().Sub(ranAt) < window, nil
}

func (s *ReviewFreshnessStore) Mark(scenario string, at time.Time) error {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return fmt.Errorf("scenario is required")
	}
	markers, err := s.LoadAll()
	if err != nil {
		return err
	}
	markers[scenario] = at.UTC()
	return s.saveAll(markers)
}

func (s *ReviewFreshnessStore) LoadAll() (map[string]time.Time, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]time.Time{}, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return map[string]time.Time{}, nil
	}
	raw := map[string]string{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("load auto-filer review freshness: %w", err)
	}
	out := make(map[string]time.Time, len(raw))
	for scenario, value := range raw {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			continue
		}
		out[scenario] = parsed.UTC()
	}
	return out, nil
}

func (s *ReviewFreshnessStore) saveAll(markers map[string]time.Time) error {
	raw := make(map[string]string, len(markers))
	for scenario, at := range markers {
		raw[scenario] = at.UTC().Format(time.RFC3339)
	}
	return storage.WriteJSONAtomic(s.path, raw)
}
