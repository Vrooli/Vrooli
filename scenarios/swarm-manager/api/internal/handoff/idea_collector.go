package handoff

import (
	"os"
	"sort"
	"strings"

	"swarm-manager/internal/jsonutil"
	"swarm-manager/internal/planworkshop"
)

func summarizeDecisions(session planworkshop.Session) ([]DecisionSummary, []OpenDecision) {
	type decisionState struct {
		key    string
		locked DecisionSummary
		open   OpenDecision
		isOpen bool
		round  int
	}

	decisions := make(map[string]decisionState)

	answers := make(map[string]string)
	for _, response := range session.Responses {
		for id, answer := range response.Answers {
			answers[strings.TrimSpace(id)] = strings.TrimSpace(answer)
		}
	}
	for _, question := range session.Packet.Questions {
		id := strings.TrimSpace(question.ID)
		if id == "" {
			continue
		}
		answer := strings.TrimSpace(answers[id])
		if answer != "" {
			decisions[id] = decisionState{
				key: id,
				locked: DecisionSummary{
					Round:         1,
					ID:            id,
					Topic:         strings.TrimSpace(question.Question),
					SelectedKey:   answer,
					SelectedLabel: planWorkshopAnswerLabel(question, answer),
				},
				round: 1,
			}
			continue
		}
		decisions[id] = decisionState{
			key: id,
			open: OpenDecision{
				Round: 1,
				ID:    id,
				Topic: strings.TrimSpace(question.Question),
			},
			isOpen: true,
			round:  1,
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

func planWorkshopAnswerLabel(question planworkshop.DecisionQuestion, answer string) string {
	for _, option := range question.Options {
		if strings.EqualFold(strings.TrimSpace(option), answer) {
			return strings.TrimSpace(option)
		}
	}
	return ""
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
	return jsonutil.WriteFileRedacted(path, value)
}
