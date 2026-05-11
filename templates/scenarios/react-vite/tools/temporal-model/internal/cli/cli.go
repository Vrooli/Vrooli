package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"react-vite-temporal-model/internal/artifact"
	"react-vite-temporal-model/internal/codegen"
	"react-vite-temporal-model/internal/contract"
	"react-vite-temporal-model/internal/discovery"
	"react-vite-temporal-model/internal/filesystem"
	"react-vite-temporal-model/internal/model"
	"react-vite-temporal-model/internal/quint"
)

func Run(ctx context.Context, args []string, stdout io.Writer, _ io.Writer) error {
	return Service{
		Runner: quint.ExecRunner{},
		FS:     osFileSystem{},
		Stdout: stdout,
	}.Run(ctx, args)
}

type Service struct {
	Runner quint.Runner
	FS     FileSystem
	Stdout io.Writer
}

type FileSystem interface {
	MkdirAll(path string, perm os.FileMode) error
	WriteFile(path string, data []byte, perm os.FileMode) error
	ReadFile(path string) ([]byte, error)
}

type osFileSystem struct{}

func (osFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (osFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (osFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (s Service) Run(ctx context.Context, args []string) error {
	stdout := s.Stdout
	if stdout == nil {
		stdout = io.Discard
	}
	runner := s.Runner
	if runner == nil {
		runner = quint.ExecRunner{}
	}
	fs := s.FS
	if fs == nil {
		fs = osFileSystem{}
	}
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printHelp(stdout)
		return nil
	}
	command := args[0]
	flags, err := parseFlags(args[1:])
	if err != nil {
		return err
	}
	root, err := filepath.Abs(flags.root)
	if err != nil {
		return err
	}
	contracts, err := discovery.FindContracts(root)
	if err != nil {
		return err
	}
	selected := discovery.Filter(contracts, flags.flow)
	if flags.flow != "" && len(selected) == 0 {
		return fmt.Errorf("unknown flow id %s", flags.flow)
	}
	switch command {
	case "list":
		for _, c := range selected {
			fmt.Fprintln(stdout, c.FlowID)
		}
		return nil
	case "validate":
		for _, c := range selected {
			fmt.Fprintf(stdout, "valid %s\n", c.FlowID)
		}
		return nil
	case "generate":
		return s.generate(ctx, stdout, fs, runner, root, selected, false)
	case "check":
		for _, flow := range selected {
			if err := validateReplayFixture(flow, root); err != nil {
				return err
			}
		}
		return s.generate(ctx, stdout, fs, runner, root, selected, true)
	case "explain":
		if flags.flow == "" {
			return fmt.Errorf("explain requires --flow <flow-id>")
		}
		return explain(stdout, root, selected[0])
	default:
		return fmt.Errorf("unknown command %s", command)
	}
}

type flags struct {
	root string
	flow string
}

func parseFlags(args []string) (flags, error) {
	out := flags{root: "../.."}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--root":
			if i+1 >= len(args) || args[i+1] == "" {
				return out, fmt.Errorf("--root requires a path")
			}
			out.root = args[i+1]
			i++
		case "--flow":
			if i+1 >= len(args) || args[i+1] == "" {
				return out, fmt.Errorf("--flow requires a flow id")
			}
			out.flow = args[i+1]
			i++
		default:
			return out, fmt.Errorf("unknown argument %s", args[i])
		}
	}
	return out, nil
}

func (s Service) generate(ctx context.Context, stdout io.Writer, fs FileSystem, runner quint.Runner, root string, flows []model.Flow, check bool) error {
	version, err := runner.Run(ctx, quint.Command{Args: []string{"quint", "--version"}, Dir: root})
	if err != nil {
		return err
	}
	quintVersion := trim(version.Stdout)
	if quintVersion == "" {
		return fmt.Errorf("quint --version returned an empty version")
	}
	wrote := 0
	for _, flow := range flows {
		rendered := quint.Render(flow)
		modelPath := filesystem.Abs(root, flow.Outputs.ModelPath)
		if check {
			if err := assertFresh(fs, modelPath, []byte(rendered), flow.FlowID); err != nil {
				return err
			}
		} else {
			if err := fs.MkdirAll(filepath.Dir(modelPath), 0o755); err != nil {
				return err
			}
			if err := fs.WriteFile(modelPath, []byte(rendered), 0o644); err != nil {
				return err
			}
		}
		built, err := artifact.Build(ctx, flow, artifact.BuildOptions{
			Root:         root,
			Rendered:     rendered,
			QuintVersion: quintVersion,
			RunQuint:     true,
			Runner:       runner,
		})
		if err != nil {
			return err
		}
		data, err := artifact.CanonicalJSON(built)
		if err != nil {
			return err
		}
		renderedCode, err := codegen.Render(flow, built)
		if err != nil {
			return err
		}
		artifactPath := filesystem.Abs(root, flow.Outputs.ArtifactPath)
		if check {
			if err := assertFresh(fs, artifactPath, data, flow.FlowID); err != nil {
				return err
			}
			for _, file := range renderedCode.Files {
				if err := assertFresh(fs, filesystem.Abs(root, file.Path), file.Data, flow.FlowID); err != nil {
					return err
				}
			}
			fmt.Fprintf(stdout, "fresh %s\n", flow.FlowID)
			continue
		}
		if err := fs.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
			return err
		}
		if err := fs.WriteFile(artifactPath, data, 0o644); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "wrote %s\n", flow.Outputs.ModelPath)
		fmt.Fprintf(stdout, "wrote %s\n", flow.Outputs.ArtifactPath)
		for _, file := range renderedCode.Files {
			target := filesystem.Abs(root, file.Path)
			if err := fs.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := fs.WriteFile(target, file.Data, 0o644); err != nil {
				return err
			}
			fmt.Fprintf(stdout, "wrote %s\n", file.Path)
		}
		wrote++
	}
	if !check {
		fmt.Fprintf(stdout, "generated %d temporal flow(s)\n", wrote)
	}
	return nil
}

func explain(stdout io.Writer, root string, flow model.Flow) error {
	fmt.Fprintf(stdout, "flow: %s\n", flow.FlowID)
	fmt.Fprintf(stdout, "contract: %s\n", flow.ContractPath)
	fmt.Fprintln(stdout, "source of truth: *.flow.json")
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Generated files:")
	fmt.Fprintf(stdout, "  model: %s\n", flow.Outputs.ModelPath)
	fmt.Fprintf(stdout, "  artifact: %s\n", flow.Outputs.ArtifactPath)
	fmt.Fprintf(stdout, "  declarations: %s\n", flow.Outputs.DeclarationsPath)
	if flow.Outputs.ReplayHelperPath != "" {
		fmt.Fprintf(stdout, "  replay helper: %s\n", flow.Outputs.ReplayHelperPath)
	}
	fmt.Fprintf(stdout, "  replay test: %s\n", flow.Outputs.ReplayTestPath)
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Runtime:")
	fmt.Fprintf(stdout, "  language: %s\n", runtimeLanguage(flow))
	fmt.Fprintf(stdout, "  status type: %s\n", runtimeStatusType(flow))
	fmt.Fprintf(stdout, "  event type: %s\n", runtimeEventType(flow))
	fmt.Fprintf(stdout, "  generated runtime unions: %s\n", yesNo(hasGeneratedRuntimeUnions(flow)))
	fmt.Fprintf(stdout, "  fixture contract: %s\n", yesNo(runtimeLanguage(flow) == "typescript"))
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Topology:")
	fmt.Fprintf(stdout, "  states: %d (initial %s; terminal %s)\n", len(flow.States), flow.Initial.ID, terminalSummary(flow))
	fmt.Fprintf(stdout, "  events: %d\n", len(flow.Events))
	fmt.Fprintf(stdout, "  expanded transitions: %d\n", flow.Matrix.Len())
	fmt.Fprintf(stdout, "  invalid transitions: %d\n", invalidTransitionCount(flow))
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Generated replay:")
	fmt.Fprintf(stdout, "  kind: %s\n", flow.Replay.Kind)
	fmt.Fprintf(stdout, "  test: %s\n", flow.Replay.TestPath)
	if flow.Replay.HelperPath != "" {
		fmt.Fprintf(stdout, "  helper: %s\n", flow.Replay.HelperPath)
	}
	if flow.Replay.FixtureModule != "" {
		fmt.Fprintf(stdout, "  fixture: %s (%s)\n", flow.Replay.FixtureModule, flow.Replay.FixtureExport)
	}
	fmt.Fprintf(stdout, "  transition: %s\n", flow.Replay.Transition.Function)
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Coverage requirements:")
	coveredStates, missingStates, coveredEvents, missingEvents := namedTraceCoverage(flow)
	fmt.Fprintf(stdout, "  named traces: %d\n", len(flow.Traces))
	fmt.Fprintf(stdout, "  named trace states: %s\n", coverageSummary(coveredStates, missingStates))
	fmt.Fprintf(stdout, "  named trace events: %s\n", coverageSummary(coveredEvents, missingEvents))
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Commands:")
	fmt.Fprintf(stdout, "  regenerate: cd %s && GOWORK=off go run . generate --root %s --flow %s\n", filepath.ToSlash(filepath.Join("tools", "temporal-model")), explainRoot(root), flow.FlowID)
	fmt.Fprintln(stdout, "  check: make temporal-models")
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Hand-authored follow-up files:")
	for _, path := range handAuthoredFollowUps(flow) {
		fmt.Fprintf(stdout, "  %s\n", path)
	}
	return nil
}

func runtimeLanguage(flow model.Flow) string {
	switch {
	case flow.Runtime.Go != nil:
		return "go"
	case flow.Runtime.TypeScript != nil:
		return "typescript"
	default:
		return "unknown"
	}
}

func runtimeStatusType(flow model.Flow) string {
	if flow.Runtime.Go != nil {
		return flow.Runtime.Go.StatusType
	}
	if flow.Runtime.TypeScript != nil {
		return flow.Runtime.TypeScript.StatusType
	}
	return ""
}

func runtimeEventType(flow model.Flow) string {
	if flow.Runtime.Go != nil {
		return flow.Runtime.Go.EventType
	}
	if flow.Runtime.TypeScript != nil {
		return flow.Runtime.TypeScript.EventType
	}
	return ""
}

func hasGeneratedRuntimeUnions(flow model.Flow) bool {
	return flow.Runtime.TypeScript != nil && flow.Runtime.TypeScript.StateUnionType != "" && flow.Runtime.TypeScript.EventUnionType != ""
}

func terminalSummary(flow model.Flow) string {
	var terminal []string
	for _, state := range flow.States {
		if state.Terminal {
			terminal = append(terminal, state.ID)
		}
	}
	if len(terminal) == 0 {
		return "none"
	}
	return strings.Join(terminal, ", ")
}

func invalidTransitionCount(flow model.Flow) int {
	total := 0
	for _, transition := range flow.Matrix.Rows() {
		if transition.WantError {
			total++
		}
	}
	return total
}

func namedTraceCoverage(flow model.Flow) ([]string, []string, []string, []string) {
	coveredStates := map[string]bool{}
	coveredEvents := map[string]bool{}
	for _, trace := range flow.Traces {
		coveredStates[trace.Initial] = true
		for _, step := range trace.Steps {
			coveredEvents[step.Event] = true
			coveredStates[step.Want] = true
		}
	}
	var states []string
	var missingStates []string
	for _, state := range flow.States {
		if coveredStates[state.ID] {
			states = append(states, state.ID)
		} else {
			missingStates = append(missingStates, state.ID)
		}
	}
	var events []string
	var missingEvents []string
	for _, event := range flow.Events {
		if coveredEvents[event.ID] {
			events = append(events, event.ID)
		} else {
			missingEvents = append(missingEvents, event.ID)
		}
	}
	return states, missingStates, events, missingEvents
}

func coverageSummary(covered []string, missing []string) string {
	if len(missing) == 0 {
		return "all covered"
	}
	return fmt.Sprintf("covered %s; missing %s", strings.Join(covered, ", "), strings.Join(missing, ", "))
}

func handAuthoredFollowUps(flow model.Flow) []string {
	seen := map[string]bool{}
	var out []string
	add := func(path string) {
		if path == "" || seen[path] {
			return
		}
		seen[path] = true
		out = append(out, path)
	}
	add(inferRuntimeWrapper(flow.Outputs.DeclarationsPath))
	if model.ReplayKind(flow.Replay.Kind) == model.ReplayKindVitest {
		fixture, err := contract.ResolveTypeScriptImport(flow.Outputs.ReplayTestPath, flow.Replay.FixtureModule)
		if err == nil {
			add(fixture)
		}
	}
	return out
}

func validateReplayFixture(flow model.Flow, root string) error {
	if model.ReplayKind(flow.Replay.Kind) != model.ReplayKindVitest {
		return nil
	}
	path, err := contract.ResolveTypeScriptImport(flow.Outputs.ReplayTestPath, flow.Replay.FixtureModule)
	if err != nil {
		return fmt.Errorf("invalid replay fixture module for %s: %w", flow.FlowID, err)
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		return fmt.Errorf("invalid replay fixture for %s:\n  - replay fixture module %s for %s is missing or unreadable: %v", flow.FlowID, path, flow.FlowID, err)
	}
	if !hasTypeScriptExport(data, flow.Replay.FixtureExport) {
		return fmt.Errorf("invalid replay fixture for %s:\n  - replay fixture module %s for %s does not export %s", flow.FlowID, path, flow.FlowID, flow.Replay.FixtureExport)
	}
	return nil
}

func hasTypeScriptExport(data []byte, name string) bool {
	quoted := regexp.QuoteMeta(name)
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?m)^\s*export\s+(const|let|var|function|class)\s+` + quoted + `\b`),
		regexp.MustCompile(`(?m)^\s*export\s*\{[^}]*\b` + quoted + `\b[^}]*\}`),
	}
	for _, pattern := range patterns {
		if pattern.Match(data) {
			return true
		}
	}
	return false
}

func inferRuntimeWrapper(generated string) string {
	switch {
	case strings.HasSuffix(generated, ".generated.ts"):
		return strings.TrimSuffix(generated, ".generated.ts") + ".ts"
	default:
		return ""
	}
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func explainRoot(root string) string {
	if filepath.Base(root) == "react-vite" {
		return "../.."
	}
	return filepath.ToSlash(root)
}

func trim(value string) string {
	for len(value) > 0 && (value[len(value)-1] == '\n' || value[len(value)-1] == '\r' || value[len(value)-1] == ' ' || value[len(value)-1] == '\t') {
		value = value[:len(value)-1]
	}
	return value
}

func assertFresh(fs FileSystem, path string, next []byte, flowID string) error {
	current, err := fs.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s is missing. Run cd tools/temporal-model && GOWORK=off go run . generate --root ../.. --flow %s", filepath.ToSlash(path), flowID)
	}
	if string(current) != string(next) {
		return fmt.Errorf("%s is stale. Run cd tools/temporal-model && GOWORK=off go run . generate --root ../.. --flow %s", filepath.ToSlash(path), flowID)
	}
	return nil
}

func printHelp(stdout io.Writer) {
	fmt.Fprintln(stdout, "Usage: go run . <list|validate|generate|check|explain> [--root <path>] [--flow <flow-id>]")
}
