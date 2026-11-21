package algorithm

import (
	"fmt"
	"os"
	"strings"

	"algorithm-library/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register wires the `algorithm` subcommand group, which covers the entire
// algorithm-centric API surface: search, retrieval, validation, benchmarking,
// comparison, execution tracing, AI suggestion, and problem cross-referencing.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "algorithm",
		Description: "Search, retrieve, validate, and benchmark algorithms",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "search", Description: "Search algorithms by query and filters", Run: func(args []string) error { return runSearch(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Get algorithm details and implementations", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "validate", Description: "Validate an implementation against test cases", Run: func(args []string) error { return runValidate(core, args) }},
			{Name: "benchmark", Description: "Benchmark an implementation across input sizes", Run: func(args []string) error { return runBenchmark(core, args) }},
			{Name: "execute", Description: "Execute an algorithm against custom input", Run: func(args []string) error { return runExecute(core, args) }},
			{Name: "validate-batch", Description: "Run a batch of validations from a JSON payload", Run: func(args []string) error { return runValidateBatch(core, args) }},
			{Name: "compare", Description: "Compare multiple algorithms on the same input", Run: func(args []string) error { return runCompare(core, args) }},
			{Name: "compare-visualize", Description: "Get visualization data for an algorithm comparison", Run: func(args []string) error { return runCompareVisualize(core, args) }},
			{Name: "trace", Description: "Generate an execution trace + visualization", Run: func(args []string) error { return runTrace(core, args) }},
			{Name: "suggest", Description: "Get AI-powered algorithm suggestions", Run: func(args []string) error { return runSuggest(core, args) }},
			{Name: "problems", Description: "List cross-platform problems mapped to an algorithm", Run: func(args []string) error { return runProblems(core, args) }},
		},
	}
}

func runSearch(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("algorithm search")
	category := fs.String("category", "", "Filter by category (e.g. sorting, graph)")
	language := fs.String("language", "", "Filter by programming language")
	complexity := fs.String("complexity", "", "Filter by time or space complexity")
	difficulty := fs.String("difficulty", "", "Difficulty: easy|medium|hard")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: algorithm search <query> [--category N] [--language L] [--complexity C] [--difficulty D]")
	}
	query := strings.Join(fs.Args(), " ")

	params := support.BuildQuery(map[string]string{
		"query":      query,
		"category":   *category,
		"language":   *language,
		"complexity": *complexity,
		"difficulty": *difficulty,
	})
	body, err := core.Get("/algorithms/search", params)
	if err != nil {
		return err
	}

	var resp support.SearchResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	results := make([]string, 0, len(resp.Algorithms))
	for _, a := range resp.Algorithms {
		results = append(results, fmt.Sprintf(
			"%s (%s) | category=%s | time=%s space=%s | difficulty=%s | langs=%d tests=%d validated=%s",
			a.Name, a.DisplayName, a.Category, a.ComplexityTime, a.ComplexitySpace,
			a.Difficulty, a.LanguageCount, a.TestCaseCount, support.CheckMark(a.HasValidatedImpl),
		))
	}
	if len(results) == 0 {
		results = []string{"(no algorithms matched)"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Query %q matched %d of %d algorithms", query, len(resp.Algorithms), resp.Total)},
		ResultsHeading: "Algorithms",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s algorithm get <name>", support.CLIName),
			fmt.Sprintf("%s categories", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("algorithm get")
	language := fs.String("language", "", "Filter implementations by language")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: algorithm get <algorithm-id-or-name> [--language L]")
	}
	id := fs.Arg(0)

	params := support.BuildQuery(map[string]string{"language": *language})
	body, err := core.Get("/algorithms/"+id+"/implementations", params)
	if err != nil {
		return err
	}

	var resp support.ImplementationsResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{
		fmt.Sprintf("Algorithm: %s (%s)", resp.Algorithm.Name, resp.Algorithm.DisplayName),
		fmt.Sprintf("Category: %s | Difficulty: %s", resp.Algorithm.Category, resp.Algorithm.Difficulty),
		fmt.Sprintf("Complexity: time=%s space=%s", resp.Algorithm.ComplexityTime, resp.Algorithm.ComplexitySpace),
	}
	if resp.Algorithm.Description != "" {
		summary = append(summary, fmt.Sprintf("Description: %s", resp.Algorithm.Description))
	}

	results := make([]string, 0, len(resp.Implementations))
	for _, impl := range resp.Implementations {
		results = append(results, fmt.Sprintf(
			"[%s v%s] primary=%s validated=%s validations=%d score=%.2f",
			impl.Language, impl.Version, support.CheckMark(impl.IsPrimary),
			support.CheckMark(impl.Validated), impl.ValidationCount, impl.PerformanceScore,
		))
		if impl.Code != "" {
			results = append(results, indent(impl.Code))
		}
	}
	if len(results) == 0 {
		results = []string{"(no implementations available)"}
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Implementations",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s algorithm validate %s <file>", support.CLIName, id),
			fmt.Sprintf("%s algorithm problems %s", support.CLIName, id),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runValidate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("algorithm validate")
	language := fs.String("language", "", "Override language detection")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: algorithm validate <algorithm-id> <file> [--language L]")
	}
	id := fs.Arg(0)
	filePath := fs.Arg(1)

	lang := strings.TrimSpace(*language)
	if lang == "" {
		lang = support.DetectLanguage(filePath)
		if lang == "" {
			return fmt.Errorf("cannot detect language from %q; use --language", filePath)
		}
	}

	code, err := support.ReadFileText(filePath)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/algorithms/validate", nil, map[string]interface{}{
		"algorithm_id": id,
		"language":     lang,
		"code":         code,
	})
	if err != nil {
		return err
	}

	var resp support.ValidationResult
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	verdict := "FAILED"
	if resp.Valid {
		verdict = "PASSED"
	}
	summary := []string{
		fmt.Sprintf("Validation %s for %s (%s)", verdict, id, lang),
		fmt.Sprintf("Tests: %d", len(resp.TestResults)),
	}

	results := make([]string, 0, len(resp.TestResults)+len(resp.Performance))
	for _, tr := range resp.TestResults {
		line := fmt.Sprintf("%s | %s | %dms", tr.TestCaseID, passFlag(tr.Passed), tr.ExecutionTime)
		if tr.ErrorMessage != "" {
			line += " | error: " + tr.ErrorMessage
		}
		results = append(results, line)
	}
	if len(resp.Performance) > 0 {
		results = append(results, "")
		results = append(results, "Performance:")
		results = append(results, support.MapRows(resp.Performance)...)
	}
	if len(results) == 0 {
		results = []string{"(no test cases evaluated)"}
	}

	report := cliapp.MutationReport{
		Result:  summary,
		Changes: results,
		NextCommand: []string{
			fmt.Sprintf("%s algorithm benchmark %s %s", support.CLIName, id, filePath),
			fmt.Sprintf("%s algorithm get %s", support.CLIName, id),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runBenchmark(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("algorithm benchmark")
	language := fs.String("language", "", "Override language detection")
	sizes := fs.String("sizes", "10,100,1000", "Comma-separated input sizes")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: algorithm benchmark <algorithm-id> <file> [--language L] [--sizes N1,N2,...]")
	}
	id := fs.Arg(0)
	filePath := fs.Arg(1)

	lang := strings.TrimSpace(*language)
	if lang == "" {
		lang = support.DetectLanguage(filePath)
		if lang == "" {
			return fmt.Errorf("cannot detect language from %q; use --language", filePath)
		}
	}

	code, err := support.ReadFileText(filePath)
	if err != nil {
		return err
	}

	parsedSizes, err := parseSizes(*sizes)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/algorithms/benchmark", nil, map[string]interface{}{
		"algorithm_id": id,
		"language":     lang,
		"code":         code,
		"input_sizes":  parsedSizes,
	})
	if err != nil {
		return err
	}

	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Benchmarked %s (%s) across sizes %v", id, lang, parsedSizes)},
		Changes:     support.MapRows(data),
		NextCommand: []string{fmt.Sprintf("%s algorithm validate %s %s", support.CLIName, id, filePath)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runExecute(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("algorithm execute")
	bodyFile := fs.String("body-file", "", "Path to JSON request body (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.RequestRoot("POST", "/api/algorithm/execute", nil, payload)
	if err != nil {
		return err
	}
	return renderRawMutation(body, "Algorithm execute", *jsonOutput)
}

func runValidateBatch(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("algorithm validate-batch")
	bodyFile := fs.String("body-file", "", "Path to JSON batch request (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.RequestRoot("POST", "/api/algorithm/validate-batch", nil, payload)
	if err != nil {
		return err
	}
	return renderRawMutation(body, "Batch validation", *jsonOutput)
}

func runCompare(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("algorithm compare")
	bodyFile := fs.String("body-file", "", "Path to comparison JSON request (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/algorithms/compare", nil, payload)
	if err != nil {
		return err
	}
	return renderRawMutation(body, "Algorithm comparison", *jsonOutput)
}

func runCompareVisualize(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("algorithm compare-visualize")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: algorithm compare-visualize <algorithm-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/algorithms/"+id+"/compare", nil)
	if err != nil {
		return err
	}
	return renderRawList(body, fmt.Sprintf("Visualization payload for %s", id), "Fields", *jsonOutput)
}

func runTrace(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("algorithm trace")
	bodyFile := fs.String("body-file", "", "Path to trace request JSON (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/algorithms/trace", nil, payload)
	if err != nil {
		return err
	}
	return renderRawMutation(body, "Execution trace", *jsonOutput)
}

func runSuggest(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("algorithm suggest")
	bodyFile := fs.String("body-file", "", "Path to suggestion request JSON (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/algorithms/suggest", nil, payload)
	if err != nil {
		return err
	}
	return renderRawMutation(body, "Algorithm suggestion", *jsonOutput)
}

func runProblems(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("algorithm problems")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: algorithm problems <algorithm-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/algorithms/"+id+"/problems", nil)
	if err != nil {
		return err
	}
	return renderRawList(body, fmt.Sprintf("Problems mapped to %s", id), "Problems", *jsonOutput)
}

// renderRawMutation emits a MutationReport whose Changes are either a sorted
// object dump or the raw JSON for non-object shapes. Used for endpoints whose
// payloads vary and don't justify a typed struct in this thin wrapper.
func renderRawMutation(body []byte, title string, jsonOut bool) error {
	var generic interface{}
	if err := support.Decode(body, &generic); err != nil {
		return err
	}
	changes := renderGeneric(generic)
	report := cliapp.MutationReport{
		Result:  []string{title},
		Changes: changes,
	}
	if jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func renderRawList(body []byte, summary, heading string, jsonOut bool) error {
	var generic interface{}
	if err := support.Decode(body, &generic); err != nil {
		return err
	}
	report := cliapp.ListReport{
		Summary:        []string{summary},
		ResultsHeading: heading,
		Results:        renderGeneric(generic),
	}
	if jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func renderGeneric(value interface{}) []string {
	switch v := value.(type) {
	case map[string]interface{}:
		return support.MapRows(v)
	case []interface{}:
		if len(v) == 0 {
			return []string{"(empty list)"}
		}
		rows := make([]string, 0, len(v))
		for i, item := range v {
			rows = append(rows, fmt.Sprintf("%d: %s", i, support.RenderValue(item)))
		}
		return rows
	case nil:
		return []string{"(empty payload)"}
	default:
		return []string{support.RenderValue(v)}
	}
}

func parseSizes(raw string) ([]int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("--sizes cannot be empty")
	}
	parts := strings.Split(raw, ",")
	sizes := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		var n int
		if _, err := fmt.Sscanf(p, "%d", &n); err != nil {
			return nil, fmt.Errorf("invalid size %q: %w", p, err)
		}
		sizes = append(sizes, n)
	}
	if len(sizes) == 0 {
		return nil, fmt.Errorf("--sizes must include at least one integer")
	}
	return sizes, nil
}

func passFlag(passed bool) string {
	if passed {
		return "PASS"
	}
	return "FAIL"
}

func indent(code string) string {
	lines := strings.Split(code, "\n")
	for i, line := range lines {
		lines[i] = "    " + line
	}
	return strings.Join(lines, "\n")
}
