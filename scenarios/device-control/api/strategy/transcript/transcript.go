package transcript

import (
	"encoding/json"
	"fmt"
	"os"
)

type Fixture struct {
	Version    int     `json:"version"`
	Strategies []Entry `json:"strategies"`
	Guarantee  string  `json:"guarantee"`
}
type Entry struct {
	StrategyID        string `json:"strategy_id"`
	Observe           string `json:"observe"`
	Actuate           string `json:"actuate"`
	UnavailableReason string `json:"unavailable_reason,omitempty"`
}

func Load(path string) (Fixture, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Fixture{}, fmt.Errorf("read transcript fixture: %w", err)
	}
	var f Fixture
	if err := json.Unmarshal(data, &f); err != nil {
		return Fixture{}, fmt.Errorf("decode transcript fixture: %w", err)
	}
	if f.Version < 1 || len(f.Strategies) == 0 {
		return Fixture{}, fmt.Errorf("transcript fixture has no strategy records")
	}
	return f, nil
}
