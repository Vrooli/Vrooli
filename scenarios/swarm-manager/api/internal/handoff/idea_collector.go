package handoff

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"swarm-manager/internal/pathredact"
	"swarm-manager/internal/workshop"
)

func summarizeDecisions(rounds []workshop.Round) ([]DecisionSummary, []OpenDecision) {
	type decisionState struct {
		key    string
		locked DecisionSummary
		open   OpenDecision
		isOpen bool
		round  int
	}

	decisions := make(map[string]decisionState)

	for _, round := range rounds {
		for _, item := range round.Items {
			if item.Type != "decision" {
				continue
			}

			key := decisionKey(item)
			if item.Selected != nil && strings.TrimSpace(*item.Selected) != "" {
				decisions[key] = decisionState{
					key: key,
					locked: DecisionSummary{
						Round:         round.RoundNum,
						ID:            strings.TrimSpace(item.ID),
						Topic:         strings.TrimSpace(item.Topic),
						SelectedKey:   strings.TrimSpace(*item.Selected),
						SelectedLabel: selectedLabel(item),
						Freeform:      trimmedPtr(item.Freeform),
						Notes:         trimmedPtr(item.Notes),
					},
					round: round.RoundNum,
				}
				continue
			}

			if existing, ok := decisions[key]; ok && existing.round >= round.RoundNum {
				continue
			}
			decisions[key] = decisionState{
				key: key,
				open: OpenDecision{
					Round:   round.RoundNum,
					ID:      strings.TrimSpace(item.ID),
					Topic:   strings.TrimSpace(item.Topic),
					Context: strings.TrimSpace(item.Context),
				},
				isOpen: true,
				round:  round.RoundNum,
			}
		}
	}

	locked := make([]DecisionSummary, 0, len(decisions))
	open := make([]OpenDecision, 0, len(decisions))
	for _, state := range decisions {
		if state.isOpen {
			open = append(open, state.open)
			continue
		}
		locked = append(locked, state.locked)
	}

	sort.Slice(locked, func(i, j int) bool {
		if locked[i].Round != locked[j].Round {
			return locked[i].Round < locked[j].Round
		}
		return locked[i].Topic < locked[j].Topic
	})
	sort.Slice(open, func(i, j int) bool {
		if open[i].Round != open[j].Round {
			return open[i].Round < open[j].Round
		}
		return open[i].Topic < open[j].Topic
	})

	return locked, open
}

func buildRoundPaths(itemFolder string, rounds []workshop.Round) []string {
	paths := make([]string, 0, len(rounds))
	for _, round := range rounds {
		paths = append(paths, filepath.Join(itemFolder, "workshop", fmt.Sprintf("round-%03d.json", round.RoundNum)))
	}
	return paths
}

func redactPaths(redact func(string) string, paths []string) []string {
	if len(paths) == 0 {
		return nil
	}
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		out = append(out, redact(path))
	}
	return out
}

func selectedLabel(item workshop.Item) string {
	selected := trimmedPtr(item.Selected)
	for _, option := range item.Options {
		if strings.TrimSpace(option.Key) == selected {
			return strings.TrimSpace(option.Label)
		}
	}
	return ""
}

func decisionKey(item workshop.Item) string {
	topic := strings.ToLower(strings.TrimSpace(item.Topic))
	if topic != "" {
		return topic
	}
	return strings.ToLower(strings.TrimSpace(item.ID))
}

func trimmedPtr(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func fileIfExists(path string) string {
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return path
	}
	return ""
}

func dirIfExists(path string) string {
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return path
	}
	return ""
}

func writeJSONFile(path string, value any) error {
	if redacted, changed, err := pathredact.NewForArtifactPath(path).RedactJSONValue(value); err != nil {
		return fmt.Errorf("redact %s: %w", filepath.Base(path), err)
	} else if changed {
		value = redacted
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", filepath.Base(path), err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", filepath.Base(path), err)
	}
	return nil
}
