package graph

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"

	"prompt-manager/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliutil"
)

type operatingGraphBlock struct {
	Metadata struct {
		ID    string `json:"id"`
		Scope string `json:"scope"`
		Team  string `json:"team"`
		Mode  string `json:"mode"`
	} `json:"metadata"`
	Graph struct {
		Nodes []any `json:"nodes"`
		Edges []any `json:"edges"`
	} `json:"graph"`
	Source struct {
		Path string `json:"path"`
		Line int    `json:"line"`
	} `json:"source"`
}

type operatingGraphFinding struct {
	Rule       string `json:"rule"`
	Severity   string `json:"severity"`
	SourcePath string `json:"source_path,omitempty"`
	Line       int    `json:"line,omitempty"`
	Detail     string `json:"detail"`
}

type operatingGraphValidation struct {
	Findings []operatingGraphFinding `json:"findings"`
	Errors   int                     `json:"errors"`
	Warnings int                     `json:"warnings"`
}

type operatingGraphListResponse struct {
	Graphs []operatingGraphBlock `json:"graphs"`
}

type operatingGraphValidationResponse struct {
	Graphs     []operatingGraphBlock    `json:"graphs"`
	Validation operatingGraphValidation `json:"validation"`
}

type operatingGraphDiff struct {
	Kind             string   `json:"kind"`
	Relationship     string   `json:"relationship"`
	Team             string   `json:"team"`
	Member           string   `json:"member,omitempty"`
	Topic            string   `json:"topic,omitempty"`
	Decision         string   `json:"decision,omitempty"`
	Path             string   `json:"path,omitempty"`
	External         string   `json:"external,omitempty"`
	TargetTeam       string   `json:"target_team,omitempty"`
	SourcePath       string   `json:"source_path,omitempty"`
	Line             int      `json:"line,omitempty"`
	RuntimePath      string   `json:"runtime_path,omitempty"`
	AcceptableFields []string `json:"acceptable_fields,omitempty"`
	Suggestions      []string `json:"suggestions,omitempty"`
	Detail           string   `json:"detail"`
}

type operatingGraphDiffResponse struct {
	Graphs []operatingGraphBlock `json:"graphs"`
	Diff   []operatingGraphDiff  `json:"diff"`
}

func cmdOperatingModel(ctx appctx.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: graph operating-model <list|validate|diff> [--team TEAM] [--id ID] [--json]")
	}
	switch args[0] {
	case "list":
		return cmdOperatingModelList(ctx, args[1:])
	case "validate":
		return cmdOperatingModelValidate(ctx, args[1:])
	case "diff":
		return cmdOperatingModelDiff(ctx, args[1:])
	default:
		return fmt.Errorf("unknown operating-model subcommand: %s", args[0])
	}
}

func cmdOperatingModelList(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("operating-model list", flag.ContinueOnError)
	team := fs.String("team", "", "Filter to one team")
	id := fs.String("id", "", "Filter to one graph id")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	var resp operatingGraphListResponse
	if err := ctx.GetWithQuery("/operating-graphs", operatingModelQuery(*team, *id), &resp); err != nil {
		return fmt.Errorf("failed to fetch operating graphs: %w", err)
	}
	if *jsonOut {
		return encodeJSON(resp)
	}
	fmt.Println("Status")
	fmt.Printf("Found %d operating graph(s).\n\n", len(resp.Graphs))
	fmt.Println("Triage")
	for _, g := range resp.Graphs {
		fmt.Printf("- %s team=%s mode=%s source=%s:%d nodes=%d edges=%d\n", g.Metadata.ID, g.Metadata.Team, g.Metadata.Mode, g.Source.Path, g.Source.Line, len(g.Graph.Nodes), len(g.Graph.Edges))
	}
	fmt.Println("\nNext Steps")
	fmt.Println("Run `prompt-manager graph operating-model validate --team <team>` to check contract drift.")
	return nil
}

func cmdOperatingModelValidate(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("operating-model validate", flag.ContinueOnError)
	team := fs.String("team", "", "Filter to one team")
	id := fs.String("id", "", "Filter to one graph id")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	var resp operatingGraphValidationResponse
	if err := ctx.GetWithQuery("/operating-graphs/validate", operatingModelQuery(*team, *id), &resp); err != nil {
		return fmt.Errorf("failed to validate operating graph: %w", err)
	}
	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(resp)
	} else {
		printOperatingModelValidation(resp)
	}
	if resp.Validation.Errors > 0 {
		return fmt.Errorf("operating-model validation failed: %d error(s)", resp.Validation.Errors)
	}
	return nil
}

func cmdOperatingModelDiff(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("operating-model diff", flag.ContinueOnError)
	team := fs.String("team", "", "Filter to one team")
	id := fs.String("id", "", "Filter to one graph id")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	var resp operatingGraphDiffResponse
	if err := ctx.GetWithQuery("/operating-graphs/diff", operatingModelQuery(*team, *id), &resp); err != nil {
		return fmt.Errorf("failed to diff operating graph: %w", err)
	}
	if *jsonOut {
		return encodeJSON(resp)
	}
	fmt.Println("Status")
	fmt.Printf("Found %d diff item(s).\n\n", len(resp.Diff))
	printOperatingModelDiffGroup("Graph Declares, Runtime Missing", resp.Diff, "graph_relationship_missing_in_runtime")
	fmt.Println()
	printOperatingModelDiffGroup("Runtime Declares, Graph Missing", resp.Diff, "runtime_relationship_missing_in_graph")
	fmt.Println("\nNext Steps")
	fmt.Println("Review whether each diff is a graph-doc omission or a runtime config change.")
	return nil
}

func operatingModelQuery(team, id string) url.Values {
	q := url.Values{}
	if strings.TrimSpace(team) != "" {
		q.Set("team", team)
	}
	if strings.TrimSpace(id) != "" {
		q.Set("id", id)
	}
	return q
}

func printOperatingModelValidation(resp operatingGraphValidationResponse) {
	fmt.Println("Status")
	fmt.Printf("Validated %d operating graph(s): %d error(s), %d warning(s).\n\n", len(resp.Graphs), resp.Validation.Errors, resp.Validation.Warnings)
	fmt.Println("Triage")
	if len(resp.Validation.Findings) == 0 {
		fmt.Println("- clean")
	} else {
		findings := append([]operatingGraphFinding(nil), resp.Validation.Findings...)
		sort.Slice(findings, func(i, j int) bool {
			if findings[i].Severity != findings[j].Severity {
				return findings[i].Severity < findings[j].Severity
			}
			return findings[i].Rule < findings[j].Rule
		})
		for _, f := range findings {
			loc := ""
			if f.SourcePath != "" {
				loc = fmt.Sprintf(" (%s", f.SourcePath)
				if f.Line > 0 {
					loc += fmt.Sprintf(":%d", f.Line)
				}
				loc += ")"
			}
			fmt.Printf("- [%s] %s: %s%s\n", strings.ToUpper(f.Severity), f.Rule, f.Detail, loc)
		}
	}
	fmt.Println("\nNext Steps")
	if resp.Validation.Errors > 0 {
		fmt.Println("Fix error findings before treating the graph as an enforceable contract.")
	} else if resp.Validation.Warnings > 0 {
		fmt.Println("Review warning findings and decide whether they are accepted target-state gaps.")
	} else {
		fmt.Println("No action required.")
	}
}

func printOperatingModelDiffGroup(title string, diffs []operatingGraphDiff, kind string) {
	fmt.Println(title)
	var printed bool
	for _, d := range diffs {
		if d.Kind != kind {
			continue
		}
		printed = true
		loc := d.SourcePath
		if loc != "" && d.Line > 0 {
			loc = fmt.Sprintf("%s:%d", loc, d.Line)
		}
		if loc == "" {
			loc = "unknown source"
		}
		fmt.Printf("- [%s] %s\n", d.Relationship, loc)
		fmt.Printf("  %s\n", d.Detail)
		if d.RuntimePath != "" {
			fmt.Printf("  Runtime file: %s\n", d.RuntimePath)
		}
		if len(d.AcceptableFields) > 0 {
			fmt.Printf("  Acceptable runtime fields: %s\n", strings.Join(d.AcceptableFields, ", "))
		}
		for _, suggestion := range d.Suggestions {
			fmt.Printf("  Suggested fix: %s\n", suggestion)
		}
	}
	if !printed {
		fmt.Println("- clean")
	}
}
