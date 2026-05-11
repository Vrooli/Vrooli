package cli

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"react-vite-temporal-model/internal/discovery"
	"react-vite-temporal-model/internal/layout"
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
	handAuthored := handAuthoredFiles(flow)
	writeSection(&b, "",
		pair("flow", flow.FlowID),
		pair("contract", flow.ContractPath),
		pair("layout", fmt.Sprintf("hand-authored: %d file(s) at %s; generated: %s/", len(handAuthored), flow.Layout.BaseDir, flow.Layout.BaseDir+"/generated/"+flow.Layout.FolderName)),
	)
	writeSection(&b, "Hand-authored (edit these)", handAuthored...)
	writeSection(&b, "Generated (regenerated; do not edit)",
		pair("model", flow.Layout.ModelPath),
		pair("artifact", flow.Layout.ArtifactPath),
		pair("runtime", flow.Layout.RuntimePath),
		pair("replay helper", flow.Layout.ReplayHelperPath),
	)
	writeSection(&b, "Runtime",
		pair("language", string(flow.Layout.Language)),
		pair("status type", runtimeStatusType(flow)),
		pair("event type", runtimeEventType(flow)),
		pair("generated runtime unions", yesNo(hasGeneratedRuntimeUnions(flow))),
		pair("fixture contract", yesNo(flow.Layout.Language == layout.LanguageTypeScript)),
		pair("subpackage import", layout.SubpackageImportPath(flow.Layout)),
	)
	writeSection(&b, "Topology",
		pair("states", fmt.Sprintf("%d (initial %s; terminal %s)", len(flow.States), flow.Initial.ID, terminalSummary(flow))),
		pair("events", fmt.Sprint(len(flow.Events))),
		pair("expanded transitions", fmt.Sprint(flow.Matrix.Len())),
		pair("invalid transitions", fmt.Sprint(invalidTransitionCount(flow))),
	)
	writeSection(&b, "Replay",
		pair("transition", flow.Replay.Transition.Function),
		pair("fixture", fixtureSummary(flow)),
	)
	writeSection(&b, "Coverage requirements",
		pair("named traces", fmt.Sprint(len(flow.Traces))),
		pair("named trace states", coverageSummary(coverage.CoveredStates, coverage.MissingStates)),
		pair("named trace events", coverageSummary(coverage.CoveredEvents, coverage.MissingEvents)),
	)
	writeSection(&b, "Commands",
		pair("regenerate", fmt.Sprintf("cd %s && GOWORK=off go run . generate --root %s --flow %s", filepath.ToSlash(filepath.Join("tools", "temporal-model")), explainRoot(root), flow.FlowID)),
		pair("check", "make temporal-models"),
	)
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

// handAuthoredFiles returns the paths the developer is expected to
// maintain by hand for this flow. The list is informational; the lint
// pass in temporal-model check enforces the contract on the test file.
func handAuthoredFiles(flow model.Flow) []string {
	out := []string{
		"contract: " + flow.ContractPath,
	}
	switch flow.Layout.Language {
	case layout.LanguageGo:
		out = append(out,
			"wrapper: "+flow.Layout.BaseDir+"/<wrapper>.go",
			"thin replay test (any *_test.go in "+flow.Layout.BaseDir+" that imports the generated subpackage and calls RunReplay)",
		)
	case layout.LanguageTypeScript:
		out = append(out,
			"wrapper: "+flow.Layout.BaseDir+"/"+strings.TrimPrefix(flow.Replay.Transition.Module, "./")+".ts",
			"fixtures: "+resolveTSImport(flow.ContractPath, flow.Replay.FixtureModule),
			"thin replay test (any *.test.ts in "+flow.Layout.BaseDir+" that imports replay.helper and calls runFormalReplay)",
		)
	}
	return out
}

func fixtureSummary(flow model.Flow) string {
	if flow.Replay.FixtureModule == "" {
		return "n/a (go-test replay)"
	}
	return fmt.Sprintf("%s (export %s)", flow.Replay.FixtureModule, flow.Replay.FixtureExport)
}

func resolveTSImport(fromPath string, module string) string {
	if module == "" {
		return ""
	}
	base := filepath.Dir(filepath.ToSlash(fromPath))
	return filepath.ToSlash(filepath.Clean(filepath.Join(base, filepath.FromSlash(module)))) + ".ts"
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
