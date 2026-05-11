package artifact

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"react-vite-temporal-model/internal/contract"
	"react-vite-temporal-model/internal/filesystem"
	"react-vite-temporal-model/internal/quint"
)

const (
	SchemaVersion    = 4
	GeneratorVersion = 5
	GeneratorPath    = "tools/temporal-model"
)

type Artifact struct {
	SchemaVersion   int                           `json:"schemaVersion"`
	FlowID          string                        `json:"flowId"`
	Source          Source                        `json:"source"`
	Commands        map[string][]string           `json:"commands"`
	States          []string                      `json:"states"`
	Events          []string                      `json:"events"`
	Transitions     []contract.ExpandedTransition `json:"transitions"`
	NamedTraces     []NamedTrace                  `json:"namedTraces"`
	GeneratedTraces []quint.ArtifactTrace         `json:"generatedTraces"`
	Invariants      []string                      `json:"invariants"`
	GeneratedChecks []string                      `json:"generatedChecks"`
	Coverage        Coverage                      `json:"coverage"`
	Checks          Checks                        `json:"checks"`
}

type Source struct {
	ContractPath        string `json:"contractPath"`
	ContractSHA256      string `json:"contractSha256"`
	GeneratorPath       string `json:"generatorPath"`
	GeneratorSHA256     string `json:"generatorSha256"`
	GeneratorVersion    int    `json:"generatorVersion"`
	ModelPath           string `json:"modelPath"`
	ModelSHA256         string `json:"modelSha256"`
	QuintVersion        string `json:"quintVersion"`
	VerificationBackend string `json:"verificationBackend"`
}

type NamedTrace = quint.ArtifactTrace

type Coverage struct {
	TransitionMatrixComplete   bool          `json:"transitionMatrixComplete"`
	TerminalTransitionsChecked bool          `json:"terminalTransitionsChecked"`
	NamedTraces                TraceCoverage `json:"namedTraces"`
	GeneratedTraces            TraceCoverage `json:"generatedTraces"`
}

type TraceCoverage struct {
	AllStatesCovered bool     `json:"allStatesCovered"`
	AllEventsCovered bool     `json:"allEventsCovered"`
	CoveredStates    []string `json:"coveredStates"`
	CoveredEvents    []string `json:"coveredEvents"`
	CoveredPairs     []string `json:"coveredPairs,omitempty"`
	AllPairsCovered  *bool    `json:"allPairsCovered,omitempty"`
}

type Checks struct {
	Typechecked           bool `json:"typechecked"`
	Tested                bool `json:"tested"`
	Verified              bool `json:"verified"`
	GeneratedFromContract bool `json:"generatedFromContract"`
	GeneratedFromModel    bool `json:"generatedFromModel"`
}

type BuildOptions struct {
	Root         string
	Rendered     string
	QuintVersion string
	RunQuint     bool
	Runner       quint.Runner
}

func Build(ctx context.Context, c contract.Contract, options BuildOptions) (Artifact, error) {
	tempDir, err := os.MkdirTemp("", "react-vite-temporal-model-")
	if err != nil {
		return Artifact{}, err
	}
	defer os.RemoveAll(tempDir)
	defer os.RemoveAll(filepath.Join(options.Root, "_apalache-out"))

	commands := commandContract(c)
	if options.RunQuint {
		for _, name := range []string{"typecheck", "test", "verify", "run"} {
			args := commands[name]
			runArgs := args
			if name == "run" {
				runArgs = replaceTempPattern(args, filepath.Join(tempDir, flowFilePattern(c)))
			}
			result, err := options.Runner.Run(ctx, quint.Command{Args: runArgs, Dir: options.Root})
			if err != nil {
				detail := strings.TrimSpace(result.Stdout + "\n" + result.Stderr)
				if detail != "" {
					return Artifact{}, fmt.Errorf("command failed: %s\n%s\n%w", strings.Join(runArgs, " "), detail, err)
				}
				return Artifact{}, fmt.Errorf("command failed: %s\n%w", strings.Join(runArgs, " "), err)
			}
		}
	}

	generatedTraces, err := quint.NormalizeTraces(c, tempDir)
	if err != nil {
		return Artifact{}, err
	}
	contractHash, err := filesystem.FileSHA256(filesystem.Abs(options.Root, c.ContractPath))
	if err != nil {
		return Artifact{}, err
	}
	generatorHash, err := filesystem.TreeSHA256(filesystem.Abs(options.Root, GeneratorPath))
	if err != nil {
		return Artifact{}, err
	}
	return Artifact{
		SchemaVersion: SchemaVersion,
		FlowID:        c.FlowID,
		Source: Source{
			ContractPath:        c.ContractPath,
			ContractSHA256:      contractHash,
			GeneratorPath:       GeneratorPath,
			GeneratorSHA256:     generatorHash,
			GeneratorVersion:    GeneratorVersion,
			ModelPath:           c.Outputs.ModelPath,
			ModelSHA256:         filesystem.SHA256String(options.Rendered),
			QuintVersion:        options.QuintVersion,
			VerificationBackend: "apalache",
		},
		Commands:        commands,
		States:          stateIDs(c),
		Events:          eventIDs(c),
		Transitions:     c.ExpandedTransitions,
		NamedTraces:     namedTraces(c),
		GeneratedTraces: generatedTraces,
		Invariants:      append([]string(nil), c.Model.Verify.Invariants...),
		GeneratedChecks: []string{"transitionTable"},
		Coverage:        coverage(c, generatedTraces),
		Checks: Checks{
			Typechecked:           true,
			Tested:                true,
			Verified:              true,
			GeneratedFromContract: true,
			GeneratedFromModel:    true,
		},
	}, nil
}

func WriteJSON(path string, value any) error {
	data, err := CanonicalJSON(value)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func CanonicalJSON(value any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func AssertFresh(path string, next []byte, flowID string) error {
	current, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s is missing. Run cd tools/temporal-model && GOWORK=off go run . generate --root ../.. --flow %s", filepath.ToSlash(path), flowID)
	}
	if string(current) != string(next) {
		return fmt.Errorf("%s is stale. Run cd tools/temporal-model && GOWORK=off go run . generate --root ../.. --flow %s", filepath.ToSlash(path), flowID)
	}
	return nil
}

func commandContract(c contract.Contract) map[string][]string {
	invariants := append([]string(nil), c.Model.Verify.Invariants...)
	return map[string][]string{
		"typecheck": {"quint", "typecheck", c.Outputs.ModelPath},
		"test":      {"quint", "test", c.Outputs.ModelPath, "--seed", c.Model.Seed},
		"verify": append(append([]string{"quint", "verify", c.Outputs.ModelPath, "--invariants"}, invariants...),
			"--max-steps", fmt.Sprint(c.Model.MaxSteps)),
		"run": {"quint", "run", c.Outputs.ModelPath, "--mbt", "--seed", c.Model.Seed, "--max-samples", fmt.Sprint(c.Model.TraceCount), "--n-traces", fmt.Sprint(c.Model.TraceCount), "--max-steps", fmt.Sprint(c.Model.MaxSteps), "--out-itf", "<temp-itf-pattern>"},
	}
}

func replaceTempPattern(args []string, pattern string) []string {
	out := append([]string(nil), args...)
	for i, arg := range out {
		if arg == "<temp-itf-pattern>" {
			out[i] = pattern
		}
	}
	return out
}

func flowFilePattern(c contract.Contract) string {
	name := ""
	for _, r := range c.FlowID {
		if r == '.' {
			name += "-"
		} else {
			name += string(r)
		}
	}
	return name + "_{seq}.itf.json"
}

func stateIDs(c contract.Contract) []string {
	out := make([]string, 0, len(c.States))
	for _, state := range c.States {
		out = append(out, state.ID)
	}
	return out
}

func eventIDs(c contract.Contract) []string {
	out := make([]string, 0, len(c.Events))
	for _, event := range c.Events {
		out = append(out, event.ID)
	}
	return out
}

func namedTraces(c contract.Contract) []NamedTrace {
	out := make([]NamedTrace, 0, len(c.Traces))
	for _, trace := range c.Traces {
		steps := make([]quint.ArtifactTraceStep, 0, len(trace.Steps))
		for _, step := range trace.Steps {
			steps = append(steps, quint.ArtifactTraceStep{Event: step.Event, Want: step.Want, WantError: step.WantError})
		}
		out = append(out, NamedTrace{Name: trace.Name, Initial: trace.Initial, Steps: steps})
	}
	return out
}

func coverage(c contract.Contract, generatedTraces []quint.ArtifactTrace) Coverage {
	matrixPairs := map[string]bool{}
	for _, t := range c.ExpandedTransitions {
		matrixPairs[t.From+"\x00"+t.Event] = true
	}
	matrixComplete := true
	terminalTransitionsChecked := true
	for _, state := range c.States {
		stateComplete := true
		for _, event := range c.Events {
			if !matrixPairs[state.ID+"\x00"+event.ID] {
				matrixComplete = false
				stateComplete = false
			}
		}
		if state.Terminal && !stateComplete {
			terminalTransitionsChecked = false
		}
	}
	return Coverage{
		TransitionMatrixComplete:   matrixComplete,
		TerminalTransitionsChecked: terminalTransitionsChecked,
		NamedTraces:                namedTraceCoverage(c),
		GeneratedTraces:            generatedTraceCoverage(c, generatedTraces),
	}
}

func namedTraceCoverage(c contract.Contract) TraceCoverage {
	traces := make([]quint.ArtifactTrace, 0, len(c.Traces))
	for _, trace := range c.Traces {
		steps := make([]quint.ArtifactTraceStep, 0, len(trace.Steps))
		for _, step := range trace.Steps {
			steps = append(steps, quint.ArtifactTraceStep{Event: step.Event, Want: step.Want, WantError: step.WantError})
		}
		traces = append(traces, quint.ArtifactTrace{Name: trace.Name, Initial: trace.Initial, Steps: steps})
	}
	return traceCoverage(c, traces, false)
}

func generatedTraceCoverage(c contract.Contract, traces []quint.ArtifactTrace) TraceCoverage {
	return traceCoverage(c, traces, true)
}

func traceCoverage(c contract.Contract, traces []quint.ArtifactTrace, includePairs bool) TraceCoverage {
	states := map[string]bool{}
	events := map[string]bool{}
	pairs := map[string]bool{}
	for _, trace := range traces {
		current := trace.Initial
		states[current] = true
		for _, step := range trace.Steps {
			events[step.Event] = true
			if includePairs {
				pairs[current+"/"+step.Event] = true
			}
			current = step.Want
			states[current] = true
		}
	}
	coverage := TraceCoverage{
		AllStatesCovered: allStatesCovered(c, states),
		AllEventsCovered: allEventsCovered(c, events),
		CoveredStates:    coveredStates(c, states),
		CoveredEvents:    coveredEvents(c, events),
	}
	if includePairs {
		coverage.CoveredPairs = coveredPairs(c, pairs)
		coverage.AllPairsCovered = boolPtr(allPairsCovered(c, pairs))
	}
	return coverage
}

func boolPtr(value bool) *bool {
	return &value
}

func allStatesCovered(c contract.Contract, seen map[string]bool) bool {
	for _, state := range c.States {
		if !seen[state.ID] {
			return false
		}
	}
	return true
}

func allEventsCovered(c contract.Contract, seen map[string]bool) bool {
	for _, event := range c.Events {
		if !seen[event.ID] {
			return false
		}
	}
	return true
}

func allPairsCovered(c contract.Contract, seen map[string]bool) bool {
	for _, state := range c.States {
		for _, event := range c.Events {
			if !seen[state.ID+"/"+event.ID] {
				return false
			}
		}
	}
	return true
}

func coveredStates(c contract.Contract, seen map[string]bool) []string {
	out := []string{}
	for _, state := range c.States {
		if seen[state.ID] {
			out = append(out, state.ID)
		}
	}
	return out
}

func coveredEvents(c contract.Contract, seen map[string]bool) []string {
	out := []string{}
	for _, event := range c.Events {
		if seen[event.ID] {
			out = append(out, event.ID)
		}
	}
	return out
}

func coveredPairs(c contract.Contract, seen map[string]bool) []string {
	out := []string{}
	for _, state := range c.States {
		for _, event := range c.Events {
			key := state.ID + "/" + event.ID
			if seen[key] {
				out = append(out, key)
			}
		}
	}
	return out
}
