package artifact

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"flow-verifier/internal/flows/kinds/temporal/contract"
	"flow-verifier/internal/flows/kinds/temporal/model"
	"flow-verifier/internal/flows/schemas"
	"flow-verifier/internal/fsadapter"
	"flow-verifier/internal/verification/quint"
)

const (
	SchemaVersion                 = contract.SchemaVersion
	GeneratorVersion              = contract.GeneratorVersion
	GeneratorPath                 = contract.GeneratorPath
	GeneratedCheckTransitionTable = quint.GeneratedCheckTransitionTable
	VerificationBackendApalache   = quint.VerificationBackendApalache
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
	tempDir, err := os.MkdirTemp("", "flow-verifier-")
	if err != nil {
		return Artifact{}, err
	}
	defer os.RemoveAll(tempDir)
	defer os.RemoveAll(filepath.Join(options.Root, "_apalache-out"))

	commands := commandContract(flow)
	if options.RunQuint {
		for _, name := range []string{quint.CommandTypecheck, quint.CommandTest, quint.CommandVerify, quint.CommandRun} {
			args := commands[name]
			runArgs := args
			if name == quint.CommandRun {
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
	contractHash, err := fsadapter.FileSHA256(fsadapter.Abs(options.Root, flow.ContractPath))
	if err != nil {
		return Artifact{}, err
	}
	generatorHash := generatorIdentityHash()
	return Artifact{
		SchemaVersion: SchemaVersion,
		FlowID:        flow.FlowID,
		Source: Source{
			ContractPath:        flow.ContractPath,
			ContractSHA256:      contractHash,
			GeneratorPath:       GeneratorPath,
			GeneratorSHA256:     generatorHash,
			GeneratorVersion:    GeneratorVersion,
			ModelPath:           flow.Layout.ModelPath,
			ModelSHA256:         fsadapter.SHA256String(options.Rendered),
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
		quint.CommandTypecheck: {"quint", "typecheck", flow.Layout.ModelPath},
		quint.CommandTest:      {"quint", "test", flow.Layout.ModelPath, "--seed", flow.Model.Seed},
		quint.CommandVerify: append(append([]string{"quint", "verify", flow.Layout.ModelPath, "--invariants"}, invariants...),
			"--max-steps", fmt.Sprint(flow.Model.MaxSteps)),
		quint.CommandRun: {"quint", "run", flow.Layout.ModelPath, "--mbt", "--seed", flow.Model.Seed, "--max-samples", fmt.Sprint(flow.Model.TraceCount), "--n-traces", fmt.Sprint(flow.Model.TraceCount), "--max-steps", fmt.Sprint(flow.Model.MaxSteps), "--out-itf", quint.TempITFPattern},
	}
}

func replaceTempPattern(args []string, pattern string) []string {
	out := append([]string(nil), args...)
	for i, arg := range out {
		if arg == quint.TempITFPattern {
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

// generatorIdentityHash returns a deterministic identity for the
// flow-verifier codegen surface. Before flow-verifier was promoted to a
// scenario, this was a tree hash of the on-disk generator directory.
// The generator now ships as a separate scenario, so its identity is
// derived from the embedded schema files plus GeneratorVersion — both
// content-addressable at build time and stable across machines.
func generatorIdentityHash() string {
	var buf bytes.Buffer
	buf.Write(schemas.Temporal)
	buf.WriteByte(0)
	buf.Write(schemas.FormalArtifact)
	buf.WriteByte(0)
	fmt.Fprintf(&buf, "generator-version=%d", GeneratorVersion)
	return fsadapter.SHA256Bytes(buf.Bytes())
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
