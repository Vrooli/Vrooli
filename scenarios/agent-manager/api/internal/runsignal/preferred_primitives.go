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
		for _, fact := range facts {
			path := strings.TrimSpace(fact.CommandPath)
			if path == row.PreferredPath || strings.HasPrefix(path, row.PreferredPath+" ") {
				preferredSeen = true
			}
			if path != row.DiscouragedPath && !strings.HasPrefix(path, row.DiscouragedPath+" ") {
				continue
			}
			count++
			if first.CallEventID == "" {
				first = fact
			}
			last = fact
		}
		if count < row.MinRepeat || preferredSeen || first.CallEventID == "" {
			continue
		}
		episode := newEpisode("wrong-primitive", first, last, eventsByID, events)
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
