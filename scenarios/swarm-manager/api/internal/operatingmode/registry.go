// Package operatingmode defines Swarm Manager's operating mode registry.
//
// Operating modes describe the unit of work, phase graph, run strategy,
// artifact policy, prompt routing, profile policy, and audit posture for a
// methodology. The registry is intentionally static: mode behavior is explicit
// code, while AgentManager cost/capability details live in scenario-owned
// profile JSON files.
package operatingmode

import (
	"fmt"
	"sort"
	"strings"
)

var registry = map[Mode]Definition{
	ModeItemLevel:       itemLevelDefinition(),
	ModeHolisticLoop:    holisticLoopDefinition(),
	ModePhasedPlanDrain: phasedPlanDrainDefinition(),
}

func requiredArtifacts(artifacts []ArtifactDefinition) []ArtifactDefinition {
	required := make([]ArtifactDefinition, 0, len(artifacts))
	for _, artifact := range artifacts {
		if artifact.Required {
			required = append(required, artifact)
		}
	}
	return required
}

func DefaultMode() Mode {
	return ModeItemLevel
}

func NormalizeMode(raw string) Mode {
	mode := Mode(strings.ToLower(strings.TrimSpace(raw)))
	if mode == "" {
		return DefaultMode()
	}
	return mode
}

func ValidateMode(raw string) bool {
	_, ok := registry[NormalizeMode(raw)]
	return ok
}

func MustDefinition(mode Mode) Definition {
	def, err := DefinitionFor(mode)
	if err != nil {
		panic(err)
	}
	return def
}

func DefinitionFor(mode Mode) (Definition, error) {
	normalized := NormalizeMode(string(mode))
	def, ok := registry[normalized]
	if !ok {
		return Definition{}, fmt.Errorf("unknown operating mode %q", mode)
	}
	return def, nil
}

func Modes() []Mode {
	modes := make([]Mode, 0, len(registry))
	for mode := range registry {
		modes = append(modes, mode)
	}
	sort.Slice(modes, func(i, j int) bool { return modes[i] < modes[j] })
	return modes
}

func ModeList() string {
	modes := Modes()
	parts := make([]string, 0, len(modes))
	for _, mode := range modes {
		parts = append(parts, string(mode))
	}
	return strings.Join(parts, ", ")
}

// RequiredProfileKeys returns every AgentManager profile key referenced by the
// operating-mode registry. Profile JSON remains the source of profile defaults;
// the registry only declares which scenario-owned keys must exist before the
// API serves traffic.
func RequiredProfileKeys() ([]string, error) {
	if err := ValidateRegistry(); err != nil {
		return nil, err
	}
	keys := map[string]struct{}{}
	for mode, def := range registry {
		if err := collectProfileKey(keys, mode, def.Profile.DefaultProfileKey); err != nil {
			return nil, err
		}
		for phase, key := range def.Profile.PhaseProfiles {
			if err := collectProfileKey(keys, mode, key); err != nil {
				return nil, fmt.Errorf("phase %q: %w", phase, err)
			}
		}
		for phase, phaseDef := range def.PhaseGraph.Phases {
			if err := collectProfileKey(keys, mode, phaseDef.ProfileKey); err != nil {
				return nil, fmt.Errorf("phase %q: %w", phase, err)
			}
		}
	}

	out := make([]string, 0, len(keys))
	for key := range keys {
		out = append(out, key)
	}
	sort.Strings(out)
	return out, nil
}

func collectProfileKey(keys map[string]struct{}, mode Mode, key string) error {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return nil
	}
	if !strings.HasPrefix(trimmed, "swarm-manager/") {
		return fmt.Errorf("mode %q references non-scenario-owned AgentManager profile key %q", mode, trimmed)
	}
	keys[trimmed] = struct{}{}
	return nil
}

func (d Definition) PhaseDefinition(phase Phase) (PhaseDefinition, error) {
	p, ok := d.PhaseGraph.Phases[phase]
	if !ok {
		return PhaseDefinition{}, fmt.Errorf("mode %q does not define phase %q", d.Mode, phase)
	}
	return p, nil
}

func (p MetricsPolicy) CountsReplanSample(phase Phase) bool {
	return phaseInSet(phase, p.ReplanSamplePhases)
}

func (p MetricsPolicy) CountsAcceptanceSample(phase Phase) bool {
	return phaseInSet(phase, p.AcceptanceSamplePhases)
}

func (p MetricsPolicy) IsAcceptedVerdict(verdict string) bool {
	normalized := strings.ToLower(strings.TrimSpace(verdict))
	for _, accepted := range p.AcceptedVerdicts {
		if normalized == strings.ToLower(strings.TrimSpace(accepted)) {
			return true
		}
	}
	return false
}

func phaseInSet(phase Phase, phases []Phase) bool {
	for _, candidate := range phases {
		if phase == candidate {
			return true
		}
	}
	return false
}
