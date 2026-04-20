package overview

import (
	"fmt"
	"sort"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/initiatives"
)

// InitiativeEdgeSuggestion flags a probable drift between the explicit
// initiative dependency graph and the edges implied by the child backlog
// items' cross-initiative dependencies.
type InitiativeEdgeSuggestion struct {
	From              string   `json:"from"`
	To                string   `json:"to"`
	Direction         string   `json:"direction"`           // "missing_explicit" or "possibly_stale"
	InferredFromItems []string `json:"inferred_from_items"` // "kind/name" refs that triggered the inference
	Reason            string   `json:"reason"`
}

// ConsistencyReport groups drift signals the Portfolio Manager and humans
// should surface but never act on automatically.
type ConsistencyReport struct {
	InitiativeEdgeSuggestions []InitiativeEdgeSuggestion `json:"initiative_edge_suggestions"`
}

// maxSuggestions caps the payload to keep overview responses bounded even in
// large repos. Callers should treat the list as sampled when this cap is hit.
const maxSuggestions = 50

// computeInitiativeEdgeSuggestions inspects child backlog item dependencies
// to find cross-initiative edges that are not reflected in the parent
// initiatives' explicit depends_on lists ("missing_explicit"), and conversely
// explicit initiative edges with no supporting child dependency
// ("possibly_stale").
func computeInitiativeEdgeSuggestions(
	items []backlog.BacklogItem,
	inits []initiatives.InitiativeWithRollup,
) []InitiativeEdgeSuggestion {
	if len(inits) == 0 {
		return nil
	}

	// Build a lookup: "kind/name" -> initiative name that owns that item.
	owner := make(map[string]string, len(items))
	for _, init := range inits {
		for _, ref := range init.Initiative.Items {
			owner[ref] = init.Initiative.Name
		}
	}

	// Known initiatives so we can ignore deps referring to unknown ones.
	known := make(map[string]bool, len(inits))
	for _, init := range inits {
		known[init.Initiative.Name] = true
	}

	// Inferred edges from -> to with the item refs that caused the inference.
	type edgeKey struct{ From, To string }
	inferred := make(map[edgeKey][]string)

	for _, item := range items {
		fromInit := owner[itemKey(item)]
		if fromInit == "" {
			continue
		}
		for _, depRef := range item.DependsOn {
			toInit := owner[depRef]
			if toInit == "" || toInit == fromInit {
				continue
			}
			k := edgeKey{From: fromInit, To: toInit}
			inferred[k] = append(inferred[k], itemKey(item))
		}
	}

	// Explicit edges from the initiatives themselves.
	explicit := make(map[edgeKey]bool)
	for _, init := range inits {
		for _, dep := range init.Initiative.DependsOn {
			if !known[dep] {
				continue
			}
			explicit[edgeKey{From: init.Initiative.Name, To: dep}] = true
		}
	}

	var out []InitiativeEdgeSuggestion

	// 1. Inferred edges missing from explicit list.
	for k, evidence := range inferred {
		if explicit[k] {
			continue
		}
		sort.Strings(evidence)
		reason := fmt.Sprintf("%d child item(s) in %q depend on items owned by %q", len(evidence), k.From, k.To)
		out = append(out, InitiativeEdgeSuggestion{
			From:              k.From,
			To:                k.To,
			Direction:         "missing_explicit",
			InferredFromItems: evidence,
			Reason:            reason,
		})
	}

	// 2. Explicit edges with no supporting child evidence.
	for k := range explicit {
		if _, ok := inferred[k]; ok {
			continue
		}
		out = append(out, InitiativeEdgeSuggestion{
			From:              k.From,
			To:                k.To,
			Direction:         "possibly_stale",
			InferredFromItems: nil,
			Reason:            fmt.Sprintf("initiative %q declares a dependency on %q but no child items cross that boundary", k.From, k.To),
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].From != out[j].From {
			return out[i].From < out[j].From
		}
		if out[i].To != out[j].To {
			return out[i].To < out[j].To
		}
		return out[i].Direction < out[j].Direction
	})

	if len(out) > maxSuggestions {
		out = out[:maxSuggestions]
	}
	return out
}
