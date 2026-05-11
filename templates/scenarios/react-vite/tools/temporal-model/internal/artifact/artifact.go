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
	SchemaVersion    = 2
	GeneratorVersion = 3
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
	AllStatesCovered      bool `json:"allStatesCovered"`
	AllEventsCovered      bool `json:"allEventsCovered"`
	AllPairsCovered       bool `json:"allPairsCovered"`
	TerminalStatesChecked bool `json:"terminalStatesChecked"`
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
		Coverage:        coverage(c),
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

func coverage(c contract.Contract) Coverage {
	pairs := map[string]bool{}
	for _, t := range c.ExpandedTransitions {
		pairs[t.From+"\x00"+t.Event] = true
	}
	traceStates := map[string]bool{}
	traceEvents := map[string]bool{}
	for _, trace := range c.Traces {
		traceStates[trace.Initial] = true
		for _, step := range trace.Steps {
			traceStates[step.Want] = true
			traceEvents[step.Event] = true
		}
	}
	allStates := true
	for _, state := range c.States {
		allStates = allStates && traceStates[state.ID]
	}
	allEvents := true
	for _, event := range c.Events {
		allEvents = allEvents && traceEvents[event.ID]
	}
	allPairs := true
	terminalChecked := true
	for _, state := range c.States {
		for _, event := range c.Events {
			if !pairs[state.ID+"\x00"+event.ID] {
				allPairs = false
				if state.Terminal {
					terminalChecked = false
				}
			}
		}
	}
	return Coverage{AllStatesCovered: allStates, AllEventsCovered: allEvents, AllPairsCovered: allPairs, TerminalStatesChecked: terminalChecked}
}
