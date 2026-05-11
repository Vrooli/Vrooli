package artifact

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"react-vite-temporal-model/internal/filesystem"
	"react-vite-temporal-model/internal/model"
	"react-vite-temporal-model/internal/quint"
)

const (
	SchemaVersion                 = 4
	GeneratorVersion              = 5
	GeneratorPath                 = "tools/temporal-model"
	GeneratedCheckTransitionTable = model.GeneratedCheckTransitionTable
	VerificationBackendApalache   = "apalache"
)

type Artifact struct {
	SchemaVersion   int                   `json:"schemaVersion"`
	FlowID          string                `json:"flowId"`
	Source          Source                `json:"source"`
	Commands        map[string][]string   `json:"commands"`
	States          []string              `json:"states"`
	Events          []string              `json:"events"`
	Transitions     []model.Transition    `json:"transitions"`
	NamedTraces     []NamedTrace          `json:"namedTraces"`
	GeneratedTraces []quint.ArtifactTrace `json:"generatedTraces"`
	Invariants      []string              `json:"invariants"`
	GeneratedChecks []string              `json:"generatedChecks"`
	Coverage        Coverage              `json:"coverage"`
	Checks          Checks                `json:"checks"`
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

func Build(ctx context.Context, flow model.Flow, options BuildOptions) (Artifact, error) {
	tempDir, err := os.MkdirTemp("", "react-vite-temporal-model-")
	if err != nil {
		return Artifact{}, err
	}
	defer os.RemoveAll(tempDir)
	defer os.RemoveAll(filepath.Join(options.Root, "_apalache-out"))

	commands := commandContract(flow)
	if options.RunQuint {
		for _, name := range []string{"typecheck", "test", "verify", "run"} {
			args := commands[name]
			runArgs := args
			if name == "run" {
				runArgs = replaceTempPattern(args, filepath.Join(tempDir, flowFilePattern(flow)))
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

	generatedTraces, err := quint.NormalizeTraces(flow, tempDir)
	if err != nil {
		return Artifact{}, err
	}
	contractHash, err := filesystem.FileSHA256(filesystem.Abs(options.Root, flow.ContractPath))
	if err != nil {
		return Artifact{}, err
	}
	generatorHash, err := filesystem.TreeSHA256(filesystem.Abs(options.Root, GeneratorPath))
	if err != nil {
		return Artifact{}, err
	}
	return Artifact{
		SchemaVersion: SchemaVersion,
		FlowID:        flow.FlowID,
		Source: Source{
			ContractPath:        flow.ContractPath,
			ContractSHA256:      contractHash,
			GeneratorPath:       GeneratorPath,
			GeneratorSHA256:     generatorHash,
			GeneratorVersion:    GeneratorVersion,
			ModelPath:           flow.Outputs.ModelPath,
			ModelSHA256:         filesystem.SHA256String(options.Rendered),
			QuintVersion:        options.QuintVersion,
			VerificationBackend: VerificationBackendApalache,
		},
		Commands:        commands,
		States:          model.StateIDs(flow),
		Events:          model.EventIDs(flow),
		Transitions:     flow.Matrix.Rows(),
		NamedTraces:     namedTraces(flow),
		GeneratedTraces: generatedTraces,
		Invariants:      append([]string(nil), flow.Model.Verify.Invariants...),
		GeneratedChecks: []string{GeneratedCheckTransitionTable},
		Coverage:        coverage(flow, generatedTraces),
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

func commandContract(flow model.Flow) map[string][]string {
	invariants := append([]string(nil), flow.Model.Verify.Invariants...)
	return map[string][]string{
		"typecheck": {"quint", "typecheck", flow.Outputs.ModelPath},
		"test":      {"quint", "test", flow.Outputs.ModelPath, "--seed", flow.Model.Seed},
		"verify": append(append([]string{"quint", "verify", flow.Outputs.ModelPath, "--invariants"}, invariants...),
			"--max-steps", fmt.Sprint(flow.Model.MaxSteps)),
		"run": {"quint", "run", flow.Outputs.ModelPath, "--mbt", "--seed", flow.Model.Seed, "--max-samples", fmt.Sprint(flow.Model.TraceCount), "--n-traces", fmt.Sprint(flow.Model.TraceCount), "--max-steps", fmt.Sprint(flow.Model.MaxSteps), "--out-itf", "<temp-itf-pattern>"},
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

func flowFilePattern(flow model.Flow) string {
	name := ""
	for _, r := range flow.FlowID {
		if r == '.' {
			name += "-"
		} else {
			name += string(r)
		}
	}
	return name + "_{seq}.itf.json"
}

func namedTraces(flow model.Flow) []NamedTrace {
	out := make([]NamedTrace, 0, len(flow.Traces))
	for _, trace := range flow.Traces {
		steps := make([]quint.ArtifactTraceStep, 0, len(trace.Steps))
		for _, step := range trace.Steps {
			steps = append(steps, quint.ArtifactTraceStep{Event: step.Event, Want: step.Want, WantError: step.WantError})
		}
		out = append(out, NamedTrace{Name: trace.Name, Initial: trace.Initial, Steps: steps})
	}
	return out
}

func coverage(flow model.Flow, generatedTraces []quint.ArtifactTrace) Coverage {
	return Coverage{
		TransitionMatrixComplete:   flow.Matrix.Complete(),
		TerminalTransitionsChecked: flow.Matrix.TerminalTransitionsChecked(),
		NamedTraces:                namedTraceCoverage(flow),
		GeneratedTraces:            generatedTraceCoverage(flow, generatedTraces),
	}
}

func namedTraceCoverage(flow model.Flow) TraceCoverage {
	traces := make([]quint.ArtifactTrace, 0, len(flow.Traces))
	for _, trace := range flow.Traces {
		steps := make([]quint.ArtifactTraceStep, 0, len(trace.Steps))
		for _, step := range trace.Steps {
			steps = append(steps, quint.ArtifactTraceStep{Event: step.Event, Want: step.Want, WantError: step.WantError})
		}
		traces = append(traces, quint.ArtifactTrace{Name: trace.Name, Initial: trace.Initial, Steps: steps})
	}
	return traceCoverage(flow, traces, false)
}

func generatedTraceCoverage(flow model.Flow, traces []quint.ArtifactTrace) TraceCoverage {
	return traceCoverage(flow, traces, true)
}

func traceCoverage(flow model.Flow, traces []quint.ArtifactTrace, includePairs bool) TraceCoverage {
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
		AllStatesCovered: allStatesCovered(flow, states),
		AllEventsCovered: allEventsCovered(flow, events),
		CoveredStates:    coveredStates(flow, states),
		CoveredEvents:    coveredEvents(flow, events),
	}
	if includePairs {
		coverage.CoveredPairs = coveredPairs(flow, pairs)
		coverage.AllPairsCovered = boolPtr(allPairsCovered(flow, pairs))
	}
	return coverage
}

func boolPtr(value bool) *bool {
	return &value
}

func allStatesCovered(flow model.Flow, seen map[string]bool) bool {
	for _, state := range flow.States {
		if !seen[state.ID] {
			return false
		}
	}
	return true
}

func allEventsCovered(flow model.Flow, seen map[string]bool) bool {
	for _, event := range flow.Events {
		if !seen[event.ID] {
			return false
		}
	}
	return true
}

func allPairsCovered(flow model.Flow, seen map[string]bool) bool {
	for _, state := range flow.States {
		for _, event := range flow.Events {
			if !seen[state.ID+"/"+event.ID] {
				return false
			}
		}
	}
	return true
}

func coveredStates(flow model.Flow, seen map[string]bool) []string {
	out := []string{}
	for _, state := range flow.States {
		if seen[state.ID] {
			out = append(out, state.ID)
		}
	}
	return out
}

func coveredEvents(flow model.Flow, seen map[string]bool) []string {
	out := []string{}
	for _, event := range flow.Events {
		if seen[event.ID] {
			out = append(out, event.ID)
		}
	}
	return out
}

func coveredPairs(flow model.Flow, seen map[string]bool) []string {
	out := []string{}
	for _, state := range flow.States {
		for _, event := range flow.Events {
			key := state.ID + "/" + event.ID
			if seen[key] {
				out = append(out, key)
			}
		}
	}
	return out
}
