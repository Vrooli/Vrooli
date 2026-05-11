package cli

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"react-vite-temporal-model/internal/contract"
	"react-vite-temporal-model/internal/discovery"
	"react-vite-temporal-model/internal/model"
	"react-vite-temporal-model/internal/pipeline"
	"react-vite-temporal-model/internal/quint"
)

func Run(ctx context.Context, args []string, stdout io.Writer, _ io.Writer) error {
	return Service{
		Runner: quint.ExecRunner{},
		Stdout: stdout,
	}.Run(ctx, args)
}

type Service struct {
	Runner quint.Runner
	FS     pipeline.FileSystem
	Stdout io.Writer
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
		return pipeline.Run(ctx, pipeline.Options{Root: root, Flows: selected, Mode: pipeline.ModeGenerate, Runner: runner, FS: s.FS, Stdout: stdout})
	case "check":
		return pipeline.Run(ctx, pipeline.Options{Root: root, Flows: selected, Mode: pipeline.ModeCheck, Runner: runner, FS: s.FS, Stdout: stdout})
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

func explain(stdout io.Writer, root string, flow model.Flow) error {
	_, err := io.WriteString(stdout, buildExplainReport(root, flow))
	return err
}

func buildExplainReport(root string, flow model.Flow) string {
	coverage := model.NamedTraceCoverage(flow)
	var b strings.Builder
	writeSection(&b, "", pair("flow", flow.FlowID), pair("contract", flow.ContractPath), pair("source of truth", "*.flow.json"))
	writeSection(&b, "Generated files", generatedFileRows(flow)...)
	writeSection(&b, "Runtime",
		pair("language", runtimeLanguage(flow)),
		pair("status type", runtimeStatusType(flow)),
		pair("event type", runtimeEventType(flow)),
		pair("generated runtime unions", yesNo(hasGeneratedRuntimeUnions(flow))),
		pair("fixture contract", yesNo(runtimeLanguage(flow) == "typescript")),
	)
	writeSection(&b, "Topology",
		pair("states", fmt.Sprintf("%d (initial %s; terminal %s)", len(flow.States), flow.Initial.ID, terminalSummary(flow))),
		pair("events", fmt.Sprint(len(flow.Events))),
		pair("expanded transitions", fmt.Sprint(flow.Matrix.Len())),
		pair("invalid transitions", fmt.Sprint(invalidTransitionCount(flow))),
	)
	writeSection(&b, "Generated replay", replayRows(flow)...)
	writeSection(&b, "Coverage requirements",
		pair("named traces", fmt.Sprint(len(flow.Traces))),
		pair("named trace states", coverageSummary(coverage.CoveredStates, coverage.MissingStates)),
		pair("named trace events", coverageSummary(coverage.CoveredEvents, coverage.MissingEvents)),
	)
	writeSection(&b, "Commands",
		pair("regenerate", fmt.Sprintf("cd %s && GOWORK=off go run . generate --root %s --flow %s", filepath.ToSlash(filepath.Join("tools", "temporal-model")), explainRoot(root), flow.FlowID)),
		pair("check", "make temporal-models"),
	)
	writeSection(&b, "Hand-authored follow-up files", handAuthoredFollowUps(flow)...)
	return b.String()
}

func writeSection(b *strings.Builder, title string, rows ...string) {
	if b.Len() > 0 {
		b.WriteByte('\n')
	}
	if title != "" {
		fmt.Fprintf(b, "%s:\n", title)
	}
	for _, row := range rows {
		if title == "" {
			fmt.Fprintln(b, row)
		} else {
			fmt.Fprintf(b, "  %s\n", row)
		}
	}
}

func pair(label string, value string) string {
	return label + ": " + value
}

func generatedFileRows(flow model.Flow) []string {
	rows := []string{
		pair("model", flow.Outputs.ModelPath),
		pair("artifact", flow.Outputs.ArtifactPath),
		pair("declarations", flow.Outputs.DeclarationsPath),
	}
	if flow.Outputs.ReplayHelperPath != "" {
		rows = append(rows, pair("replay helper", flow.Outputs.ReplayHelperPath))
	}
	return append(rows, pair("replay test", flow.Outputs.ReplayTestPath))
}

func replayRows(flow model.Flow) []string {
	rows := []string{
		pair("kind", flow.Replay.Kind),
		pair("test", flow.Replay.TestPath),
	}
	if flow.Replay.HelperPath != "" {
		rows = append(rows, pair("helper", flow.Replay.HelperPath))
	}
	if flow.Replay.FixtureModule != "" {
		rows = append(rows, pair("fixture", fmt.Sprintf("%s (%s)", flow.Replay.FixtureModule, flow.Replay.FixtureExport)))
	}
	return append(rows, pair("transition", flow.Replay.Transition.Function))
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

func printHelp(stdout io.Writer) {
	fmt.Fprintln(stdout, "Usage: go run . <list|validate|generate|check|explain> [--root <path>] [--flow <flow-id>]")
}
