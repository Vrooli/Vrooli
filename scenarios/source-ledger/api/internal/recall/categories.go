package recall

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"source-ledger/internal/classification"
)

var ErrUnknownCategory = errors.New("unknown memory category")

type ClassificationIndex interface {
	Categories(context.Context, string) ([]classification.Category, error)
	MatchingEntries(context.Context, string, []string) (map[string][]string, error)
}

type classificationCoverage interface {
	Coverage(context.Context, string, []string) (bool, error)
}

func (s *Service) SetClassificationIndex(index ClassificationIndex) { s.categories = index }

type categoryQuery struct {
	topic    string
	ids      []string
	hard     bool
	openOnly bool
}

func resolveCategoryQuery(query, filter string, categories []classification.Category) (categoryQuery, error) {
	result := categoryQuery{topic: query, hard: strings.TrimSpace(filter) != ""}
	if result.hard {
		wanted := strings.Join(lexicalWords(filter), " ")
		for _, category := range categories {
			for _, name := range categoryNames(category) {
				if wanted != "" && strings.Join(lexicalWords(name), " ") == wanted {
					result.ids = []string{category.ID}
					result.applyOpenFilter()
					return result, nil
				}
			}
		}
		return result, fmt.Errorf("%w: %q", ErrUnknownCategory, filter)
	}

	words := lexicalWords(query)
	var topic []string
	for i := 0; i < len(words); {
		var chosen string
		length := 0
		for _, category := range categories {
			for _, name := range categoryNames(category) {
				alias := lexicalWords(name)
				size := len(alias)
				if size > length && i+size <= len(words) && slices.Equal(words[i:i+size], alias) {
					chosen, length = category.ID, size
				}
			}
		}
		if length == 0 {
			topic = append(topic, words[i])
			i++
			continue
		}
		if !slices.Contains(result.ids, chosen) {
			result.ids = append(result.ids, chosen)
		}
		i += length
	}
	if len(result.ids) > 0 {
		result.topic = strings.Join(topic, " ")
		result.applyOpenFilter()
	}
	return result, nil
}

func (q *categoryQuery) applyOpenFilter() {
	words := lexicalWords(q.topic)
	if len(words) > 0 && words[0] == "open" {
		q.openOnly = true
		q.topic = strings.Join(words[1:], " ")
	}
}

func categoryNames(category classification.Category) []string {
	return append([]string{category.ID, category.Name}, category.Aliases...)
}

// inheritedCategories follows the same parent graph used by zoom. A category
// found on an earlier observation also makes its current summary eligible.
func inheritedCategories(nodes []Node, matches map[string][]string, parents map[string]string, historical map[string]bool) map[string][]string {
	labels := make(map[string][]string, len(matches))
	for _, node := range nodes {
		if historical[node.ID] {
			continue
		}
		inherit := func(sourceID string) {
			for _, category := range matches[sourceID] {
				for id := node.ID; id != "" && !historical[id] && !slices.Contains(labels[id], category); id = parents[id] {
					labels[id] = append(labels[id], category)
				}
			}
		}
		inherit(node.EntryID)
		for _, evidenceID := range node.EvidenceIDs {
			inherit(evidenceID)
		}
	}
	for _, ids := range labels {
		slices.Sort(ids)
	}
	return labels
}

func nodeParents(nodes []Node) map[string]string {
	parents := make(map[string]string, len(nodes))
	for _, node := range nodes {
		parents[node.ID] = node.ParentID
	}
	return parents
}

// A summary can express a finding in different words than its evidence.
// Retain most of the best constituent score when ranking a category match.
func inheritCategoryRelevance(nodes []Node, scores []float64, labels map[string][]string, parents map[string]string) map[string]float64 {
	const constituentDiscount = 0.95
	best := make(map[string]float64)
	for i, node := range nodes {
		if len(labels[node.ID]) == 0 {
			continue
		}
		score := scores[i] * constituentDiscount
		for id := node.ParentID; id != "" && score > best[id]; id = parents[id] {
			best[id] = score
		}
	}
	return best
}
