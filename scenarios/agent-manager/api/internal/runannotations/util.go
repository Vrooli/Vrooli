package runannotations

import (
	"crypto/sha256"
	"encoding/binary"
	"sort"
	"strings"
)

func sampled(runID string, label Label) bool {
	rate := 1.0
	if label.Sampling.Rate != nil {
		rate = *label.Sampling.Rate
	}
	if rate >= 1 {
		return true
	}
	if rate <= 0 {
		return false
	}
	sum := sha256.Sum256([]byte(runID + "|" + label.ID))
	value := binary.BigEndian.Uint64(sum[:8])
	fraction := float64(value) / float64(^uint64(0))
	return fraction < rate
}

func optionLabels(options []Option) []string {
	out := make([]string, len(options))
	for i, option := range options {
		out[i] = option.Label
	}
	return out
}

func maxProbability(probabilities map[string]float64) (float64, bool) {
	maximum, found := 0.0, false
	for _, probability := range probabilities {
		if !found || probability > maximum {
			maximum, found = probability, true
		}
	}
	return maximum, found
}

func numericObservation(observation Annotation) (float64, bool) {
	if observation.State != StateObserved && observation.State != StateCompatible {
		return 0, false
	}
	if observation.Probability != nil {
		return *observation.Probability, true
	}
	if observation.Value.Noul != nil {
		return *observation.Value.Noul, true
	}
	if observation.Value.Score != nil {
		return *observation.Value.Score, true
	}
	if observation.Value.Number != nil {
		return *observation.Value.Number, true
	}
	if observation.Value.Bool != nil {
		if *observation.Value.Bool {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

func cloneActivation(activation map[string]any) map[string]any {
	out := make(map[string]any, len(activation)+2)
	for key, value := range activation {
		out[key] = value
	}
	return out
}

func chunkStrings(values []string, size int) [][]string {
	if size <= 0 {
		size = len(values)
	}
	var out [][]string
	for start := 0; start < len(values); start += size {
		end := min(start+size, len(values))
		out = append(out, values[start:end])
	}
	return out
}

func chunkWorks(values []*runWork, size int) [][]*runWork {
	if size <= 0 {
		size = len(values)
	}
	var out [][]*runWork
	for start := 0; start < len(values); start += size {
		end := min(start+size, len(values))
		out = append(out, values[start:end])
	}
	return out
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func remove(values []string, target string) []string {
	out := values[:0]
	for _, value := range values {
		if value != target {
			out = append(out, value)
		}
	}
	return out
}

func addCount(counts *StateCounts, state State) {
	switch state {
	case StateObserved:
		counts.Observed++
	case StateCompatible:
		counts.Compatible++
	case StateNotObserved:
		counts.NotObserved++
	case StateUnavailable:
		counts.Unavailable++
	}
}

func hasAnnotation(annotations []Annotation, labelID string) bool {
	for _, annotation := range annotations {
		if annotation.LabelID == labelID {
			return true
		}
	}
	return false
}

func skippedFor(skipped []SkippedLabel, labelID string) bool {
	for _, entry := range skipped {
		if entry.LabelID == labelID {
			return true
		}
	}
	return false
}

func shortHash(hash string) string {
	if len(hash) <= 12 {
		return hash
	}
	return hash[:12]
}

func boundIDs(ids []string, max int) []string {
	out := make([]string, 0, max)
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		out = append(out, id)
		if len(out) >= max {
			break
		}
	}
	return out
}

// evidenceIDsFor resolves declarative selectors to event ids. Selectors that
// name an aggregate (timings, workload, terminal) have no event id; the stored
// digest already carries their values.
func evidenceIDsFor(label Label, facts RunFacts) []string {
	var ids []string
	for _, selector := range label.Evidence {
		switch selector.Source {
		case "operator_messages":
			for _, message := range tailMessages(facts.OperatorMessages, 4) {
				ids = append(ids, message.EventID)
			}
		case "agent_text":
			for _, message := range tailMessages(facts.AgentText, 2) {
				ids = append(ids, message.EventID)
			}
		case "existing_signals":
			if selector.Label == "" {
				for _, signal := range facts.ExistingSignals {
					ids = append(ids, signal.LatestEventID)
				}
			} else if signal, ok := facts.ExistingSignals[selector.Label]; ok {
				ids = append(ids, signal.LatestEventID)
			}
		}
	}
	return boundIDs(ids, 8)
}

func sortModelKeys(keys []ModelKey) {
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Model != keys[j].Model {
			return keys[i].Model < keys[j].Model
		}
		return keys[i].ResourceRole < keys[j].ResourceRole
	})
}
