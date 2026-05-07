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
	Metadata operatingGraphMetadata `json:"metadata"`
	Graph    operatingGraph         `json:"graph"`
	Docs     operatingGraphDocs     `json:"docs,omitempty"`
	Source   operatingGraphSource   `json:"source"`
}

type operatingGraphMetadata struct {
	ID     string            `json:"id"`
	Scope  string            `json:"scope"`
	Team   string            `json:"team"`
	Mode   string            `json:"mode"`
	Status string            `json:"status,omitempty"`
	Extra  map[string]string `json:"extra,omitempty"`
}

type operatingGraph struct {
	ID        string               `json:"id"`
	Direction string               `json:"direction"`
	Nodes     []operatingGraphNode `json:"nodes"`
	Edges     []operatingGraphEdge `json:"edges"`
}

type operatingGraphSource struct {
	Path      string `json:"path"`
	Line      int    `json:"line"`
	FenceLine int    `json:"fence_line"`
}

type operatingGraphNode struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Value      string `json:"value"`
	Qualifier  string `json:"qualifier,omitempty"`
	Display    string `json:"display,omitempty"`
	RawLabel   string `json:"raw_label"`
	SourceLine int    `json:"source_line"`
	Implicit   bool   `json:"implicit,omitempty"`
}

type operatingGraphEdge struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Label      string `json:"label,omitempty"`
	SourceLine int    `json:"source_line"`
}

type operatingGraphDocs struct {
	TopicCatalog operatingTopicCatalogTable `json:"topic_catalog,omitempty"`
	Decisions    operatingDecisionTable     `json:"decisions,omitempty"`
}

type operatingTopicCatalogTable struct {
	HeaderLine int                        `json:"header_line,omitempty"`
	Rows       []operatingTopicCatalogRow `json:"rows,omitempty"`
	Present    bool                       `json:"present,omitempty"`
}

type operatingTopicCatalogRow struct {
	Topic      string                    `json:"topic"`
	Qualifier  string                    `json:"qualifier,omitempty"`
	Status     string                    `json:"status"`
	Writers    []operatingActorReference `json:"writers,omitempty"`
	Readers    []operatingActorReference `json:"readers,omitempty"`
	Purpose    string                    `json:"purpose"`
	SourceLine int                       `json:"source_line"`
	RawTopic   string                    `json:"raw_topic"`
}

type operatingDecisionTable struct {
	HeaderLine int                    `json:"header_line,omitempty"`
	Rows       []operatingDecisionRow `json:"rows,omitempty"`
	Present    bool                   `json:"present,omitempty"`
}

type operatingDecisionRow struct {
	Decision    string                    `json:"decision"`
	Owners      []operatingActorReference `json:"owners,omitempty"`
	Purpose     string                    `json:"purpose"`
	SourceLine  int                       `json:"source_line"`
	RawDecision string                    `json:"raw_decision"`
}

type operatingActorReference struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
	Raw   string `json:"raw"`
}

type operatingGraphFinding struct {
	Rule       string `json:"rule"`
	Severity   string `json:"severity"`
	SourcePath string `json:"source_path,omitempty"`
	Line       int    `json:"line,omitempty"`
	GraphID    string `json:"graph_id,omitempty"`
	Team       string `json:"team,omitempty"`
	NodeID     string `json:"node_id,omitempty"`
	Edge       string `json:"edge,omitempty"`
	Member     string `json:"member,omitempty"`
	Topic      string `json:"topic,omitempty"`
	Decision   string `json:"decision,omitempty"`
	Path       string `json:"path,omitempty"`
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

type operatingGraphCoverageResponse struct {
	Graphs   []operatingGraphBlock    `json:"graphs"`
	Coverage []operatingGraphCoverage `json:"coverage"`
}

type operatingGraphCoverage struct {
	GraphID       string                          `json:"graph_id"`
	Team          string                          `json:"team"`
	Source        operatingGraphSource            `json:"source"`
	Relationships []operatingRelationshipCoverage `json:"relationships"`
	Prompts       operatingPromptCoverage         `json:"prompts"`
	Docs          operatingDocsCoverage           `json:"docs"`
	Exclusions    []operatingCoverageExclusion    `json:"exclusions"`
}

type operatingRelationshipCoverage struct {
	Relationship       string `json:"relationship"`
	RuntimeDeclared    int    `json:"runtime_declared"`
	GraphShown         int    `json:"graph_shown"`
	Matched            int    `json:"matched"`
	GraphOnly          int    `json:"graph_only"`
	RuntimeOnly        int    `json:"runtime_only"`
	ValidationRule     string `json:"validation_rule,omitempty"`
	ValidationSeverity string `json:"validation_severity,omitempty"`
	DiffRelationship   string `json:"diff_relationship,omitempty"`
}

type operatingPromptCoverage struct {
	GraphMembers               int    `json:"graph_members"`
	TopicContractPresent       int    `json:"topic_contract_present"`
	TopicContractSourceMatched int    `json:"topic_contract_source_matched"`
	TopicContractContentParity string `json:"topic_contract_content_parity"`
}

type operatingDocsCoverage struct {
	MermaidGraph          string `json:"mermaid_graph"`
	TopicCatalogTable     string `json:"topic_catalog_table"`
	TopicCatalogRows      int    `json:"topic_catalog_rows"`
	TopicCatalogMatched   int    `json:"topic_catalog_matched"`
	TopicCatalogGraphOnly int    `json:"topic_catalog_graph_only"`
	TopicCatalogDocsOnly  int    `json:"topic_catalog_docs_only"`
	TopicCatalogInvalid   int    `json:"topic_catalog_invalid"`
	DecisionsTable        string `json:"decisions_table"`
	DecisionsRows         int    `json:"decisions_rows"`
	DecisionsMatched      int    `json:"decisions_matched"`
	DecisionsGraphOnly    int    `json:"decisions_graph_only"`
	DecisionsDocsOnly     int    `json:"decisions_docs_only"`
	DecisionsInvalid      int    `json:"decisions_invalid"`
}

type operatingCoverageExclusion struct {
	Kind   string `json:"kind"`
	Count  int    `json:"count"`
	Detail string `json:"detail,omitempty"`
}

func cmdOperatingModel(ctx appctx.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: graph operating-model <list|validate|diff|coverage> [--team TEAM] [--id ID] [--json]")
	}
	switch args[0] {
	case "list":
		return cmdOperatingModelList(ctx, args[1:])
	case "validate":
		return cmdOperatingModelValidate(ctx, args[1:])
	case "diff":
		return cmdOperatingModelDiff(ctx, args[1:])
	case "coverage":
		return cmdOperatingModelCoverage(ctx, args[1:])
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
	if len(resp.Diff) == 0 {
		fmt.Println("No reconciliation required.")
	} else {
		fmt.Println("Review whether each diff is a graph-doc omission or a runtime config change.")
	}
	return nil
}

func cmdOperatingModelCoverage(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("operating-model coverage", flag.ContinueOnError)
	team := fs.String("team", "", "Filter to one team")
	id := fs.String("id", "", "Filter to one graph id")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	var resp operatingGraphCoverageResponse
	if err := ctx.GetWithQuery("/operating-graphs/coverage", operatingModelQuery(*team, *id), &resp); err != nil {
		return fmt.Errorf("failed to fetch operating graph coverage: %w", err)
	}
	if *jsonOut {
		return encodeJSON(resp)
	}
	printOperatingModelCoverage(resp)
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

func printOperatingModelCoverage(resp operatingGraphCoverageResponse) {
	fmt.Println("Status")
	fmt.Printf("Analyzed %d operating graph(s).\n\n", len(resp.Coverage))
	for _, cov := range resp.Coverage {
		fmt.Printf("Graph: %s", cov.GraphID)
		if cov.Team != "" {
			fmt.Printf(" team=%s", cov.Team)
		}
		if cov.Source.Path != "" {
			fmt.Printf(" source=%s:%d", cov.Source.Path, cov.Source.Line)
		}
		fmt.Println()

		fmt.Println("\nRelationship Coverage")
		if len(cov.Relationships) == 0 {
			fmt.Println("- none")
		} else {
			for _, rel := range cov.Relationships {
				fmt.Printf("- %s: runtime declared %d, graph shown %d, matched %d, graph-only %d, runtime-only %d",
					rel.Relationship, rel.RuntimeDeclared, rel.GraphShown, rel.Matched, rel.GraphOnly, rel.RuntimeOnly)
				if rel.ValidationSeverity != "" {
					fmt.Printf(" (%s)", rel.ValidationSeverity)
				}
				fmt.Println()
			}
		}

		fmt.Println("\nPrompt Coverage")
		fmt.Printf("- topic-contract section present: %d/%d graph members\n", cov.Prompts.TopicContractPresent, cov.Prompts.GraphMembers)
		fmt.Printf("- topic-contract source path: %d/%d graph members\n", cov.Prompts.TopicContractSourceMatched, cov.Prompts.GraphMembers)
		fmt.Printf("- content parity: %s\n", cov.Prompts.TopicContractContentParity)

		fmt.Println("\nDocs Coverage")
		fmt.Printf("- Mermaid graph: %s\n", cov.Docs.MermaidGraph)
		fmt.Printf("- Topic Catalog table: %s (rows %d, matched %d, graph-only %d, docs-only %d, invalid %d)\n",
			cov.Docs.TopicCatalogTable,
			cov.Docs.TopicCatalogRows,
			cov.Docs.TopicCatalogMatched,
			cov.Docs.TopicCatalogGraphOnly,
			cov.Docs.TopicCatalogDocsOnly,
			cov.Docs.TopicCatalogInvalid,
		)
		fmt.Printf("- Decisions table: %s (rows %d, matched %d, graph-only %d, docs-only %d, invalid %d)\n",
			cov.Docs.DecisionsTable,
			cov.Docs.DecisionsRows,
			cov.Docs.DecisionsMatched,
			cov.Docs.DecisionsGraphOnly,
			cov.Docs.DecisionsDocsOnly,
			cov.Docs.DecisionsInvalid,
		)

		fmt.Println("\nExcluded")
		if len(cov.Exclusions) == 0 {
			fmt.Println("- none")
		} else {
			for _, exclusion := range cov.Exclusions {
				fmt.Printf("- %s: %d", exclusion.Kind, exclusion.Count)
				if exclusion.Detail != "" {
					fmt.Printf(" (%s)", exclusion.Detail)
				}
				fmt.Println()
			}
		}
		fmt.Println()
	}
	if len(resp.Coverage) == 0 {
		fmt.Println("No checkable operating graph matched the filters.")
	}
}
