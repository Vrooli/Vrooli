package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"react-vite-temporal-model/internal/artifact"
	"react-vite-temporal-model/internal/codegen"
	"react-vite-temporal-model/internal/contract"
	"react-vite-temporal-model/internal/discovery"
	"react-vite-temporal-model/internal/filesystem"
	"react-vite-temporal-model/internal/quint"
)

func Run(ctx context.Context, args []string, stdout io.Writer, _ io.Writer) error {
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
	for _, c := range selected {
		if err := contract.ValidateReplayBindings(c, root); err != nil {
			return err
		}
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
		return generate(ctx, stdout, root, selected, false)
	case "check":
		return generate(ctx, stdout, root, selected, true)
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

func generate(ctx context.Context, stdout io.Writer, root string, contracts []contract.Contract, check bool) error {
	runner := quint.ExecRunner{}
	version, err := runner.Run(ctx, quint.Command{Args: []string{"quint", "--version"}, Dir: root})
	if err != nil {
		return err
	}
	quintVersion := trim(version.Stdout)
	if quintVersion == "" {
		return fmt.Errorf("quint --version returned an empty version")
	}
	wrote := 0
	for _, c := range contracts {
		rendered := quint.Render(c)
		modelPath := filesystem.Abs(root, c.Outputs.ModelPath)
		if check {
			if err := artifact.AssertFresh(modelPath, []byte(rendered), c.FlowID); err != nil {
				return err
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(modelPath), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(modelPath, []byte(rendered), 0o644); err != nil {
				return err
			}
		}
		built, err := artifact.Build(ctx, c, artifact.BuildOptions{
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
		declarations, err := codegen.Render(c, built)
		if err != nil {
			return err
		}
		artifactPath := filesystem.Abs(root, c.Outputs.ArtifactPath)
		if check {
			if err := artifact.AssertFresh(artifactPath, data, c.FlowID); err != nil {
				return err
			}
			if err := artifact.AssertFresh(filesystem.Abs(root, c.Outputs.DeclarationsPath), []byte(declarations), c.FlowID); err != nil {
				return err
			}
			fmt.Fprintf(stdout, "fresh %s\n", c.FlowID)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(artifactPath, data, 0o644); err != nil {
			return err
		}
		declarationsPath := filesystem.Abs(root, c.Outputs.DeclarationsPath)
		if err := os.MkdirAll(filepath.Dir(declarationsPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(declarationsPath, []byte(declarations), 0o644); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "wrote %s\n", c.Outputs.ModelPath)
		fmt.Fprintf(stdout, "wrote %s\n", c.Outputs.ArtifactPath)
		fmt.Fprintf(stdout, "wrote %s\n", c.Outputs.DeclarationsPath)
		wrote++
	}
	if !check {
		fmt.Fprintf(stdout, "generated %d temporal flow(s)\n", wrote)
	}
	return nil
}

func explain(stdout io.Writer, root string, flow contract.Contract) error {
	fmt.Fprintf(stdout, "flow: %s\n", flow.FlowID)
	fmt.Fprintf(stdout, "contract: %s\n", flow.ContractPath)
	fmt.Fprintln(stdout, "source of truth: *.flow.json")
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Generated files:")
	fmt.Fprintf(stdout, "  model: %s\n", flow.Outputs.ModelPath)
	fmt.Fprintf(stdout, "  artifact: %s\n", flow.Outputs.ArtifactPath)
	fmt.Fprintf(stdout, "  declarations: %s\n", flow.Outputs.DeclarationsPath)
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Runtime:")
	fmt.Fprintf(stdout, "  language: %s\n", runtimeLanguage(flow))
	fmt.Fprintf(stdout, "  status type: %s\n", runtimeStatusType(flow))
	fmt.Fprintf(stdout, "  event type: %s\n", runtimeEventType(flow))
	fmt.Fprintf(stdout, "  generated runtime unions: %s\n", yesNo(hasGeneratedRuntimeUnions(flow)))
	fmt.Fprintf(stdout, "  fixture contract: %s\n", yesNo(runtimeLanguage(flow) == "typescript"))
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Topology:")
	fmt.Fprintf(stdout, "  states: %d (initial %s; terminal %s)\n", len(flow.States), contract.Initial(flow).ID, terminalSummary(flow))
	fmt.Fprintf(stdout, "  events: %d\n", len(flow.Events))
	fmt.Fprintf(stdout, "  expanded transitions: %d\n", len(flow.ExpandedTransitions))
	fmt.Fprintf(stdout, "  invalid transitions: %d\n", invalidTransitionCount(flow))
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Replay bindings:")
	for _, binding := range flow.Replay.Bindings {
		fmt.Fprintf(stdout, "  ok %s\n", binding.Path)
		fmt.Fprintf(stdout, "    marker: %s\n", binding.Assertion)
		fmt.Fprintf(stdout, "    helpers: artifact freshness, transitions, traces\n")
	}
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

func runtimeLanguage(flow contract.Contract) string {
	switch {
	case flow.Runtime.Go != nil:
		return "go"
	case flow.Runtime.TypeScript != nil:
		return "typescript"
	default:
		return "unknown"
	}
}

func runtimeStatusType(flow contract.Contract) string {
	if flow.Runtime.Go != nil {
		return flow.Runtime.Go.StatusType
	}
	if flow.Runtime.TypeScript != nil {
		return flow.Runtime.TypeScript.StatusType
	}
	return ""
}

func runtimeEventType(flow contract.Contract) string {
	if flow.Runtime.Go != nil {
		return flow.Runtime.Go.EventType
	}
	if flow.Runtime.TypeScript != nil {
		return flow.Runtime.TypeScript.EventType
	}
	return ""
}

func hasGeneratedRuntimeUnions(flow contract.Contract) bool {
	return flow.Runtime.TypeScript != nil && flow.Runtime.TypeScript.StateUnionType != "" && flow.Runtime.TypeScript.EventUnionType != ""
}

func terminalSummary(flow contract.Contract) string {
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

func invalidTransitionCount(flow contract.Contract) int {
	total := 0
	for _, transition := range flow.ExpandedTransitions {
		if transition.WantError {
			total++
		}
	}
	return total
}

func namedTraceCoverage(flow contract.Contract) ([]string, []string, []string, []string) {
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

func handAuthoredFollowUps(flow contract.Contract) []string {
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
	for _, binding := range flow.Replay.Bindings {
		add(binding.Path)
	}
	return out
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

func printHelp(stdout io.Writer) {
	fmt.Fprintln(stdout, "Usage: go run . <list|validate|generate|check|explain> [--root <path>] [--flow <flow-id>]")
}
