// Package operatingmode defines Swarm Manager's operating mode registry.
//
// Operating modes describe the unit of work, phase graph, run strategy,
// artifact policy, prompt routing, profile policy, and audit posture for a
// methodology. The registry is data-driven: each mode is a data folder
// (scenarios/swarm-manager/modes/<id>/mode.json) validated by
// .vrooli/schemas/operating-mode.schema.json and loaded into the typed
// Definition by the loader. There is no hardcoded Go mode definition and no
// static registry map; adding or changing a mode is a data edit plus a
// restart, never a code change. AgentManager cost/capability details live in
// scenario-owned profile JSON files.
package operatingmode

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"swarm-manager/internal/pathutil"
)

// modesDirName is the mode-data subdirectory under the scenario root.
const modesDirName = "modes"

var (
	registryMu   sync.RWMutex
	registry     map[Mode]Definition
	registryErr  error
	registryOnce sync.Once
)

// LoadRegistry loads every mode from modesDir (validating the full set) and
// installs it as the process registry, replacing any previously loaded modes.
// The server calls it once at startup with <scenarioRoot>/modes so the exact
// resolved root is used; a failure is fatal to startup. Loading is also the
// validation step — LoadModesFromDir runs ValidateLoadedModes internally.
func LoadRegistry(modesDir string) error {
	defs, err := LoadModesFromDir(modesDir)
	if err != nil {
		return err
	}
	registryMu.Lock()
	registry = defs
	registryErr = nil
	registryMu.Unlock()
	return nil
}

// ensureRegistry returns the loaded registry, lazily loading it from the
// resolved scenario modes dir the first time it is needed. This lets consumers
// (and tests in dependent packages) that never call LoadRegistry explicitly
// still observe a populated, validated registry. Once LoadRegistry has run
// successfully the lazy path is skipped.
func ensureRegistry() (map[Mode]Definition, error) {
	registryMu.RLock()
	if registry != nil {
		defs := registry
		registryMu.RUnlock()
		return defs, nil
	}
	registryMu.RUnlock()

	registryOnce.Do(func() {
		dir := filepath.Join(pathutil.ResolveScenarioRoot("swarm-manager"), modesDirName)
		defs, err := LoadModesFromDir(dir)
		registryMu.Lock()
		if registry == nil {
			registry = defs
			registryErr = err
		}
		registryMu.Unlock()
	})

	registryMu.RLock()
	defer registryMu.RUnlock()
	if registry == nil {
		if registryErr != nil {
			return nil, registryErr
		}
		return nil, fmt.Errorf("operating-mode registry is not loaded")
	}
	return registry, nil
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

// DefaultMode is the mode value blank initiative metadata normalizes to: the
// member-item-strategy sentinel. It is NOT a registered operating mode — see
// ModeItemLevel and IsMemberItemStrategySentinel.
func DefaultMode() Mode {
	return ModeItemLevel
}

// NormalizeMode is the SINGLE server-side seam for the persisted wire-value
// policy: an initiative's mode may be persisted as blank OR as "item-level",
// and both mean the member-item workflow strategy. Persisted initiative.json
// files are never rewritten to collapse the two forms; every reader normalizes
// through here (directly or via initiatives.NormalizeMode) instead of
// hand-rolling a blank→"item-level" ternary. The UI-side twin of this policy
// is ui/src/lib/member-item-strategy.ts (normalizeModeWireValue).
func NormalizeMode(raw string) Mode {
	mode := Mode(strings.ToLower(strings.TrimSpace(raw)))
	if mode == "" {
		return DefaultMode()
	}
	return mode
}

// IsMemberItemStrategySentinel reports whether raw is the persisted
// member-item workflow strategy wire value ("item-level", or blank which
// normalizes to it). The sentinel is domain strategy configuration on the
// initiative, not a registered operating mode: it has no Definition, and
// DefinitionFor rejects it.
func IsMemberItemStrategySentinel(raw string) bool {
	return NormalizeMode(raw) == ModeItemLevel
}

// ValidateMode reports whether raw names a REGISTERED operating mode. The
// member-item-strategy sentinel is deliberately not valid here; initiative
// mode-field validation accepts it separately (initiatives.ValidateMode).
func ValidateMode(raw string) bool {
	defs, err := ensureRegistry()
	if err != nil {
		return false
	}
	_, ok := defs[NormalizeMode(raw)]
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
	if normalized == ModeItemLevel {
		return Definition{}, fmt.Errorf("%q is the member-item workflow strategy, not an operating mode; it has no mode definition", ModeItemLevel)
	}
	defs, err := ensureRegistry()
	if err != nil {
		return Definition{}, err
	}
	def, ok := defs[normalized]
	if !ok {
		return Definition{}, fmt.Errorf("unknown operating mode %q", mode)
	}
	return def, nil
}

func Modes() []Mode {
	defs, err := ensureRegistry()
	if err != nil {
		return nil
	}
	return SortedModes(defs)
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
	defs, err := ensureRegistry()
	if err != nil {
		return nil, err
	}
	keys := map[string]struct{}{}
	for mode, def := range defs {
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
