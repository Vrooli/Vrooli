package graph

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"prompt-manager/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliutil"
)

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
	if *jsonOut {
		return printRawOperatingModelJSON(ctx, "/operating-models", operatingModelQuery(*team, *id))
	}
	var resp operatingModelListResponse
	if err := ctx.GetWithQuery("/operating-models", operatingModelQuery(*team, *id), &resp); err != nil {
		return fmt.Errorf("failed to fetch operating models: %w", err)
	}
	fmt.Println("Status")
	fmt.Printf("Found %d operating model(s).\n\n", len(resp.Models))
	fmt.Println("Triage")
	for _, model := range resp.Models {
		graph := operatingModelPrimaryGraph(model)
		fmt.Printf("- %s team=%s mode=%s source=%s:%d nodes=%d edges=%d\n", model.ID, model.Team, graph.Metadata.Mode, model.Source.Path, model.Source.Line, len(graph.Graph.Nodes), len(graph.Graph.Edges))
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
	var resp operatingModelValidationResponse
	if *jsonOut {
		raw, err := rawOperatingModelJSON(ctx, "/operating-models/validate", operatingModelQuery(*team, *id))
		if err != nil {
			return fmt.Errorf("failed to validate operating model: %w", err)
		}
		if err := json.Unmarshal(raw, &resp); err != nil {
			return fmt.Errorf("decode operating model validation response: %w", err)
		}
		raw = formatRawOperatingModelJSON(raw)
		os.Stdout.Write(raw)
		if len(raw) == 0 || raw[len(raw)-1] != '\n' {
			fmt.Println()
		}
	} else if err := ctx.GetWithQuery("/operating-models/validate", operatingModelQuery(*team, *id), &resp); err != nil {
		return fmt.Errorf("failed to validate operating model: %w", err)
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
	if *jsonOut {
		return printRawOperatingModelJSON(ctx, "/operating-models/diff", operatingModelQuery(*team, *id))
	}
	var resp operatingModelDiffResponse
	if err := ctx.GetWithQuery("/operating-models/diff", operatingModelQuery(*team, *id), &resp); err != nil {
		return fmt.Errorf("failed to diff operating model: %w", err)
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
	if *jsonOut {
		return printRawOperatingModelJSON(ctx, "/operating-models/coverage", operatingModelQuery(*team, *id))
	}
	var resp operatingModelCoverageResponse
	if err := ctx.GetWithQuery("/operating-models/coverage", operatingModelQuery(*team, *id), &resp); err != nil {
		return fmt.Errorf("failed to fetch operating model coverage: %w", err)
	}
	printOperatingModelCoverage(resp)
	return nil
}
