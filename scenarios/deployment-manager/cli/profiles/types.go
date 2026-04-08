package profiles

import (
	"encoding/json"
	"fmt"
)

type Profile struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Scenario string                 `json:"scenario"`
	Tiers    []int                  `json:"tiers"`
	Swaps    Swaps                  `json:"swaps"`
	Secrets  map[string]interface{} `json:"secrets"`
	Settings map[string]interface{} `json:"settings"`
	Version  int                    `json:"version"`
}

// Swap mirrors the API swap payload.
type Swap struct {
	From            string   `json:"from"`
	To              string   `json:"to"`
	Reason          string   `json:"reason,omitempty"`
	Limitations     string   `json:"limitations,omitempty"`
	ApplicableTiers []string `json:"applicable_tiers,omitempty"`
	AppliedAt       string   `json:"applied_at,omitempty"`
}

// Swaps accepts either the legacy map[string]string format or the API array of Swap objects.
type Swaps []Swap

func (s *Swaps) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*s = Swaps{}
		return nil
	}
	switch data[0] {
	case '{':
		var m map[string]string
		if err := json.Unmarshal(data, &m); err != nil {
			return err
		}
		var out Swaps
		for k, v := range m {
			out = append(out, Swap{From: k, To: v})
		}
		*s = out
		return nil
	case '[':
		var arr []Swap
		if err := json.Unmarshal(data, &arr); err != nil {
			return err
		}
		*s = arr
		return nil
	default:
		return fmt.Errorf("unsupported swaps format")
	}
}

func (s Swaps) MarshalJSON() ([]byte, error) {
	type alias Swap
	arr := make([]alias, 0, len(s))
	for _, sw := range s {
		arr = append(arr, alias(sw))
	}
	return json.Marshal(arr)
}

func (s Swaps) len() int {
	return len(s)
}

func (s *Swaps) ensureInitialized() {
	if s == nil {
		return
	}
	if *s == nil {
		*s = Swaps{}
	}
}

func (s *Swaps) set(from, to string) {
	s.ensureInitialized()
	for i := range *s {
		if (*s)[i].From == from {
			(*s)[i].To = to
			return
		}
	}
	*s = append(*s, Swap{From: from, To: to})
}

func (s *Swaps) remove(from string) {
	s.ensureInitialized()
	out := (*s)[:0]
	for _, sw := range *s {
		if sw.From != from {
			out = append(out, sw)
		}
	}
	*s = out
}

type ProfileHistory struct {
	ProfileID string    `json:"profile_id"`
	Versions  []Profile `json:"versions"`
}
