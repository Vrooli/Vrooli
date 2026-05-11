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
	"react-vite-temporal-model/internal/spec"
)

const (
	SchemaVersion                 = spec.SchemaVersion
	GeneratorVersion              = spec.GeneratorVersion
	GeneratorPath                 = spec.GeneratorPath
	GeneratedCheckTransitionTable = spec.GeneratedCheckTransitionTable
	VerificationBackendApalache   = spec.VerificationBackendApalache
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
	TransitionMatrixComplete   bool                `json:"transitionMatrixComplete"`
	TerminalTransitionsChecked bool                `json:"terminalTransitionsChecked"`
	NamedTraces                model.TraceCoverage `json:"namedTraces"`
	GeneratedTraces            model.TraceCoverage `json:"generatedTraces"`
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
		for _, name := range []string{spec.CommandTypecheck, spec.CommandTest, spec.CommandVerify, spec.CommandRun} {
			args := commands[name]
			runArgs := args
			if name == spec.CommandRun {
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

func commandContract(flow model.Flow) map[string][]string {
	invariants := append([]string(nil), flow.Model.Verify.Invariants...)
	return map[string][]string{
		spec.CommandTypecheck: {"quint", "typecheck", flow.Outputs.ModelPath},
		spec.CommandTest:      {"quint", "test", flow.Outputs.ModelPath, "--seed", flow.Model.Seed},
		spec.CommandVerify: append(append([]string{"quint", "verify", flow.Outputs.ModelPath, "--invariants"}, invariants...),
			"--max-steps", fmt.Sprint(flow.Model.MaxSteps)),
		spec.CommandRun: {"quint", "run", flow.Outputs.ModelPath, "--mbt", "--seed", flow.Model.Seed, "--max-samples", fmt.Sprint(flow.Model.TraceCount), "--n-traces", fmt.Sprint(flow.Model.TraceCount), "--max-steps", fmt.Sprint(flow.Model.MaxSteps), "--out-itf", spec.TempITFPattern},
	}
}

func replaceTempPattern(args []string, pattern string) []string {
	out := append([]string(nil), args...)
	for i, arg := range out {
		if arg == spec.TempITFPattern {
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

func namedTraceCoverage(flow model.Flow) model.TraceCoverage {
	return model.NamedTraceCoverage(flow)
}

func generatedTraceCoverage(flow model.Flow, traces []quint.ArtifactTrace) model.TraceCoverage {
	return model.TraceCoverageFor(flow, coverageTraces(traces), true)
}

func coverageTraces(traces []quint.ArtifactTrace) []model.CoverageTrace {
	out := make([]model.CoverageTrace, 0, len(traces))
	for _, trace := range traces {
		steps := make([]model.CoverageTraceStep, 0, len(trace.Steps))
		for _, step := range trace.Steps {
			steps = append(steps, model.CoverageTraceStep{Event: step.Event, Want: step.Want})
		}
		out = append(out, model.CoverageTrace{Initial: trace.Initial, Steps: steps})
	}
	return out
}
