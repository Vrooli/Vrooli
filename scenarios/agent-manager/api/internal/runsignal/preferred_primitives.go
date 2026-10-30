package runsignal

import (
	_ "embed"
	"encoding/json"
	"strings"

	"agent-manager/internal/domain"
)

//go:embed preferred_primitives.json
var preferredPrimitiveJSON []byte

type PreferredPrimitive struct {
	IntentKey       string `json:"intent_key"`
	DiscouragedPath string `json:"discouraged_path"`
	PreferredPath   string `json:"preferred_path"`
	DetectShape     string `json:"detect_shape"`
	MinRepeat       int    `json:"min_repeat"`
	RationaleRef    string `json:"rationale_ref"`
	// RecallOnly limits the row to searches of prior work (docs, memory, plans,
	// earlier runs); searching a file or code in place is fine.
	RecallOnly bool `json:"recall_only,omitempty"`
}

func PreferredPrimitives() []PreferredPrimitive {
	var rows []PreferredPrimitive
	if json.Unmarshal(preferredPrimitiveJSON, &rows) != nil {
		return nil
	}
	return rows
}

// DetectWrongPrimitives is generic: all scenario-specific paths are supplied
// by the configured preferred-primitive table, never by detector logic.
func DetectWrongPrimitives(facts []InvocationFact, eventsByID map[string]*domain.RunEvent, events []*domain.RunEvent) []FrictionEpisode {
	var out []FrictionEpisode
	for _, row := range PreferredPrimitives() {
		if row.MinRepeat < 1 || row.DetectShape != "repeat-discouraged-without-preferred" {
			continue
		}
		count := 0
		preferredSeen := false
		var first, last InvocationFact
		var matched []InvocationFact
		for _, fact := range facts {
			path := strings.TrimSpace(fact.CommandPath)
			if path == row.PreferredPath || strings.HasPrefix(path, row.PreferredPath+" ") {
				preferredSeen = true
			}
			if path != row.DiscouragedPath && !strings.HasPrefix(path, row.DiscouragedPath+" ") {
				continue
			}
			if row.RecallOnly && !isRecallSearch(fact, eventsByID) {
				continue
			}
			count++
			matched = append(matched, fact)
			if first.CallEventID == "" {
				first = fact
			}
			last = fact
		}
		if count < row.MinRepeat || preferredSeen || first.CallEventID == "" {
			continue
		}
		episode := newEpisode("wrong-primitive", first, last, eventsByID, events)
		episode.price(matched, eventsByID)
		episode.Severity = "repeated"
		episode.RepeatedElement = row.PreferredPath
		episode.HonestyFlags = append(episode.HonestyFlags, "preferred-path-absent")
		if episode.SuspectedOwnerScenario == "" && first.Executable != "" {
			episode.SuspectedOwnerScenario = first.Executable
			episode.SuspectedOwnerCommand = first.CommandPath
			episode.OwnerConfidence = "manifest-derived"
		}
		out = append(out, episode)
	}
	return out
}

// recallRoots are where an agent's prior work lives: docs, memory, plans and
// earlier runs.
var recallRoots = []string{"docs", ".vrooli", ".claude/projects", "memory"}

// isRecallSearch is a search of prior work: a path under a recall root, or no
// path and a natural-language pattern of several words. Searching a file or
// code in place is not.
func isRecallSearch(fact InvocationFact, byID map[string]*domain.RunEvent) bool {
	var operands []string
	tokens := factTokens(fact, byID)
	isFind := len(tokens) > 0 && tokens[0] == "find"
	for _, token := range tokens[min(1, len(tokens)):] {
		if !isFlag(token) {
			operands = append(operands, token)
		}
	}
	paths := operands
	if !isFind && len(operands) > 0 {
		paths = operands[1:] // grep and rg: the first operand is the pattern
	}
	for _, path := range paths {
		for _, root := range recallRoots {
			if strings.HasPrefix(path, root) || strings.Contains(path, "/"+root) {
				return true
			}
		}
	}
	if len(paths) > 0 || len(operands) == 0 || isFind {
		return false
	}
	pattern := operands[0]
	return strings.Contains(strings.TrimSpace(pattern), " ") && strings.IndexAny(pattern, `\*[](){}^$|+?`) < 0
}
