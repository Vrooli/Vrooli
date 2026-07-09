package operatingmode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// exampleRunsDirName is the mode-owned subdirectory holding example-run
// fixtures under modes/<id>/.
const exampleRunsDirName = "example-runs"

// happyPathPresetID is the reserved id of a mode's straight-through preset.
// SimulateMode surfaces it first and falls back to it for unknown/empty preset
// requests, so every phase mode must own an example-run with this id.
const happyPathPresetID = "happy-path"

// ExampleRun is a mode-owned simulation fixture (schema kind
// operating-mode-example-run). It seeds each step's structured output and
// asserts the exact phase path the real generic guard evaluator produces.
// Example-runs never spawn agents, acquire locks, or persist state; they are
// how a mode is tested before use and the data behind the UI's simulation
// presets.
type ExampleRun struct {
	Kind        string `json:"kind"`
	ID          string `json:"id"`
	Mode        string `json:"mode"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
	// Branch and Scenario are operator-facing narrative surfaced in the UI Flow
	// tab's preset control: Branch is the one-line guarded edge this preset
	// exercises; Scenario is the prose framing of why the branch is taken.
	Branch   string `json:"branch,omitempty"`
	Scenario string `json:"scenario,omitempty"`
	// Initiative seeds the ephemeral sandbox initiative the simulated rounds run
	// against (items, criteria, framing). Nil uses generic defaults.
	Initiative   *ExampleRunInitiative `json:"initiative,omitempty"`
	Steps        []ExampleRunStep      `json:"steps,omitempty"`
	ExpectedPath []string              `json:"expected_path"`
}

// ExampleRunInitiative is the seeded sandbox initiative for an example-run —
// purely illustrative fixture data that never persists.
type ExampleRunInitiative struct {
	Title       string        `json:"title,omitempty"`
	Description string        `json:"description,omitempty"`
	Items       []ExampleItem `json:"items,omitempty"`
	Criteria    []string      `json:"criteria,omitempty"`
}

// ExampleItem is one seeded backlog item in an example-run's sandbox
// initiative. It mirrors RoundItem but is its own on-disk shape so the schema
// can constrain it independently.
type ExampleItem struct {
	Ref      string `json:"ref"`
	Title    string `json:"title,omitempty"`
	Status   string `json:"status,omitempty"`
	Priority int    `json:"priority,omitempty"`
	Effort   string `json:"effort,omitempty"`
}

// RoundItems maps the seeded example items onto the runtime RoundItem shape.
func (i *ExampleRunInitiative) RoundItems() []RoundItem {
	if i == nil {
		return nil
	}
	out := make([]RoundItem, 0, len(i.Items))
	for _, item := range i.Items {
		out = append(out, RoundItem(item))
	}
	return out
}

// ExampleRunStep is one seeded round: the phase it runs and the structured
// output fed to that phase's transition guards.
type ExampleRunStep struct {
	Phase  string         `json:"phase"`
	Output map[string]any `json:"output,omitempty"`
	Note   string         `json:"note,omitempty"`
}

// LoadExampleRun parses and validates a raw example-run document.
func LoadExampleRun(raw []byte) (ExampleRun, error) {
	if err := ValidateDocumentBytes(raw); err != nil {
		return ExampleRun{}, err
	}
	var run ExampleRun
	if err := json.Unmarshal(raw, &run); err != nil {
		return ExampleRun{}, fmt.Errorf("decode example-run document: %w", err)
	}
	if run.Kind != DocumentKindExampleRun {
		return ExampleRun{}, fmt.Errorf("expected document kind %q, got %q", DocumentKindExampleRun, run.Kind)
	}
	return run, nil
}

// WalkExampleRun replays an example-run against a mode's real generic guard
// graph and returns the walked phase path. It drives the walk from the mode's
// start phase, consuming each seeded step's output in order to evaluate the real
// guards, and continues through the final routed-to terminal phase. It verifies
// that no fabricated transitions occur and that the resulting path equals the
// fixture's declared expected_path. This is the mechanism that makes a mode
// testable before use.
func WalkExampleRun(def Definition, run ExampleRun) ([]Phase, error) {
	if run.Mode != string(def.Mode) {
		return nil, fmt.Errorf("example-run %q targets mode %q but was walked against %q", run.ID, run.Mode, def.Mode)
	}
	if len(run.Steps) == 0 {
		return nil, fmt.Errorf("example-run %q has no steps", run.ID)
	}
	if got := Phase(run.Steps[0].Phase); got != def.PhaseGraph.StartPhase {
		return nil, fmt.Errorf("example-run %q first step is phase %q but the mode starts at %q", run.ID, got, def.PhaseGraph.StartPhase)
	}

	terminal := terminalSet(def)
	maxHops := len(run.ExpectedPath) + 1
	walked := make([]Phase, 0, len(run.ExpectedPath))
	stepIdx := 0
	cur := def.PhaseGraph.StartPhase

	for hop := 0; ; hop++ {
		if hop > maxHops {
			return nil, fmt.Errorf("example-run %q did not terminate within %d hops (guards loop?)", run.ID, maxHops)
		}
		if _, ok := def.PhaseGraph.Phases[cur]; !ok {
			return nil, fmt.Errorf("example-run %q walked into unregistered phase %q", run.ID, cur)
		}
		walked = append(walked, cur)
		if _, isTerminal := terminal[cur]; isTerminal {
			break
		}

		var output map[string]any
		if stepIdx < len(run.Steps) && Phase(run.Steps[stepIdx].Phase) == cur {
			output = run.Steps[stepIdx].Output
			stepIdx++
		} else if stepIdx < len(run.Steps) {
			return nil, fmt.Errorf("example-run %q step %d is phase %q but the walk reached %q", run.ID, stepIdx, run.Steps[stepIdx].Phase, cur)
		}
		if err := validateExampleRunStepOutput(def, run, cur, output); err != nil {
			return nil, err
		}

		next, matched := selectNextPhases(def, cur, NewMapFieldLookup(output))
		if !matched {
			return nil, fmt.Errorf("example-run %q: no guard out of %q routed onward (seeded output=%v)", run.ID, cur, output)
		}
		if len(next) == 0 {
			// Guarded stop: a matched guard with no target (e.g. a blocked
			// progress decision) is a terminal outcome, not a dead end. The walk
			// ends here and the fixture's expected_path must terminate on this
			// phase.
			break
		}
		if len(next) > 1 {
			return nil, fmt.Errorf("example-run %q: phase %q offered multiple targets %v; example-runs must be deterministic", run.ID, cur, next)
		}
		cur = next[0]
	}

	if stepIdx != len(run.Steps) {
		return nil, fmt.Errorf("example-run %q left %d step(s) unconsumed; steps must match the walked path", run.ID, len(run.Steps)-stepIdx)
	}
	if err := assertPhasePath(run, walked); err != nil {
		return nil, err
	}
	return walked, nil
}

func validateExampleRunStepOutput(def Definition, run ExampleRun, phase Phase, output map[string]any) error {
	phaseDef, ok := def.PhaseGraph.Phases[phase]
	if !ok || phaseDef.DeclaredOutput == nil {
		return nil
	}
	missing, violations := validateDeclaredOutput(phaseDef.DeclaredOutput, output)
	if len(missing) == 0 && len(violations) == 0 {
		return nil
	}
	return fmt.Errorf("example-run %q phase %q output violates declared_output: missing=%v violations=%v", run.ID, phase, missing, violations)
}

// LoadExampleRunsForMode discovers and parses every example-run under
// modes/<id>/example-runs/*.json, sorted deterministically with the reserved
// happy-path preset first and the remainder alphabetically by id. A phase mode
// with no example-runs directory returns an empty slice (the simulator then
// synthesizes a generic happy-path walk); a malformed fixture fails loudly.
func LoadExampleRunsForMode(modeDir string) ([]ExampleRun, error) {
	dir := filepath.Join(modeDir, exampleRunsDirName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read example-runs dir %q: %w", dir, err)
	}
	runs := make([]ExampleRun, 0, len(entries))
	seen := map[string]struct{}{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %q: %w", path, err)
		}
		run, err := LoadExampleRun(raw)
		if err != nil {
			return nil, fmt.Errorf("load example-run %q: %w", path, err)
		}
		if _, dup := seen[run.ID]; dup {
			return nil, fmt.Errorf("duplicate example-run id %q under %q", run.ID, dir)
		}
		seen[run.ID] = struct{}{}
		runs = append(runs, run)
	}
	sortExampleRuns(runs)
	return runs, nil
}

// sortExampleRuns orders example-runs with the happy-path preset first (so the
// simulator's default selection and the UI's first tab are the straight-through
// case) and the rest alphabetically by id for stable presentation.
func sortExampleRuns(runs []ExampleRun) {
	sort.SliceStable(runs, func(i, j int) bool {
		li, lj := runs[i].ID == happyPathPresetID, runs[j].ID == happyPathPresetID
		if li != lj {
			return li
		}
		return runs[i].ID < runs[j].ID
	})
}

func terminalSet(def Definition) map[Phase]struct{} {
	set := make(map[Phase]struct{}, len(def.PhaseGraph.Terminal))
	for _, phase := range def.PhaseGraph.Terminal {
		set[phase] = struct{}{}
	}
	return set
}

// selectNextPhases evaluates a phase's guard graph in declared order against the
// given structured output and returns the target(s) of the first matching
// guard. matched is false when no guard matches (a terminal phase, or a genuine
// dead-end the fixture should not have reached). An empty To with matched=true
// is a guarded stop.
func selectNextPhases(def Definition, from Phase, lookup FieldLookup) (targets []Phase, matched bool) {
	for _, gt := range def.PhaseGraph.Guards[from] {
		if gt.When.Eval(lookup) {
			return gt.To, true
		}
	}
	return nil, false
}

func assertPhasePath(run ExampleRun, walked []Phase) error {
	if len(walked) != len(run.ExpectedPath) {
		return fmt.Errorf("example-run %q walked path %v does not match expected_path %v", run.ID, walked, run.ExpectedPath)
	}
	for i := range walked {
		if string(walked[i]) != run.ExpectedPath[i] {
			return fmt.Errorf("example-run %q walked path %v does not match expected_path %v", run.ID, walked, run.ExpectedPath)
		}
	}
	return nil
}
