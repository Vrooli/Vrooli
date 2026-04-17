// Package experiments provides CLI commands for experiment management.
//
// DOC: docs/reference/cli-commands.md#experiments
package experiments

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"prompt-manager/cli/internal/appctx"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// ExperimentResponse matches the API response for experiments.
type ExperimentResponse struct {
	ID              string              `json:"id"`
	SkillID         string              `json:"skillId"`
	Name            string              `json:"name"`
	Hypothesis      string              `json:"hypothesis,omitempty"`
	Status          string              `json:"status"`
	Arms            []ExperimentArmResp `json:"arms"`
	OutcomeCounts   map[string]int      `json:"outcomeCounts,omitempty"`
	StartedAt       *string             `json:"startedAt,omitempty"`
	ConcludedAt     *string             `json:"concludedAt,omitempty"`
	WinnerVariantID *string             `json:"winnerVariantId,omitempty"`
	Notes           string              `json:"notes,omitempty"`
	CreatedAt       string              `json:"createdAt"`
	UpdatedAt       string              `json:"updatedAt"`
	Revision        int                 `json:"revision"`
}

// ExperimentArmResp is the API representation of an experiment arm.
type ExperimentArmResp struct {
	VariantID   string  `json:"variantId"`
	VariantName string  `json:"variantName,omitempty"`
	Weight      float64 `json:"weight"`
}

// OutcomeResponse matches the API response for an outcome.
type OutcomeResponse struct {
	VariantID     string          `json:"variantId"`
	Source        string          `json:"source"`
	SchemaVersion int             `json:"schemaVersion"`
	RecordedAt    string          `json:"recordedAt"`
	Data          json.RawMessage `json:"data"`
}

// Commands returns the experiment command group.
func Commands(ctx appctx.Context) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Experiments",
		Commands: []cliapp.Command{
			{
				Name:        "experiment",
				Aliases:     []string{"experiments", "exp"},
				NeedsAPI:    true,
				Description: "Manage experiments (list|show|create|start|conclude|outcomes|delete)",
				Run: func(args []string) error {
					return route(ctx, args)
				},
			},
		},
	}
}

// route dispatches to the appropriate subcommand.
func route(ctx appctx.Context, args []string) error {
	if len(args) == 0 {
		return printUsage()
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "list", "ls":
		return cmdList(ctx, subArgs)
	case "show", "get":
		return cmdShow(ctx, subArgs)
	case "create", "add":
		return cmdCreate(ctx, subArgs)
	case "start":
		return cmdStart(ctx, subArgs)
	case "conclude":
		return cmdConclude(ctx, subArgs)
	case "outcomes":
		return cmdOutcomes(ctx, subArgs)
	case "delete", "rm":
		return cmdDelete(ctx, subArgs)
	default:
		return fmt.Errorf("unknown subcommand: %s\n\n%s", subcommand, usageText())
	}
}

func printUsage() error {
	fmt.Println(usageText())
	return nil
}

func usageText() string {
	return `Usage: prompt-manager experiment <subcommand> [args]

Subcommands:
  list, ls              List experiments [--skill SKILL_ID]
  show, get <eid>       Show experiment details
  create, add           Create a new experiment
  start <eid>           Start a draft experiment
  conclude <eid> <vid>  Conclude a running experiment with a winner
  outcomes <eid>        List recorded outcomes
  delete, rm <eid>      Delete an experiment`
}

func cmdList(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	skill := fs.String("skill", "", "Filter by skill ID")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	var experiments []ExperimentResponse
	if *skill != "" {
		if err := ctx.Get(fmt.Sprintf("/skills/%s/experiments", *skill), &experiments); err != nil {
			return fmt.Errorf("failed to list experiments: %w", err)
		}
	} else {
		if err := ctx.GetWithQuery("/experiments", url.Values{}, &experiments); err != nil {
			return fmt.Errorf("failed to list experiments: %w", err)
		}
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(experiments)
	}

	if len(experiments) == 0 {
		fmt.Println("No experiments found")
		return nil
	}

	fmt.Println("Experiments:")
	for _, e := range experiments {
		arms := fmt.Sprintf("%d arms", len(e.Arms))
		fmt.Printf("  %s  %-20s  %-10s  %s  skill=%s\n", e.ID, e.Name, e.Status, arms, e.SkillID)
	}
	return nil
}

func cmdShow(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: experiment show <eid>")
	}
	eid := fs.Arg(0)

	var exp ExperimentResponse
	if err := ctx.Get(fmt.Sprintf("/experiments/%s", eid), &exp); err != nil {
		return fmt.Errorf("failed to get experiment: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(exp)
	}

	fmt.Printf("Name:       %s\n", exp.Name)
	fmt.Printf("ID:         %s\n", exp.ID)
	fmt.Printf("Skill:      %s\n", exp.SkillID)
	fmt.Printf("Status:     %s\n", exp.Status)
	if exp.Hypothesis != "" {
		fmt.Printf("Hypothesis: %s\n", exp.Hypothesis)
	}
	fmt.Printf("Created:    %s\n", exp.CreatedAt)
	fmt.Printf("Updated:    %s\n", exp.UpdatedAt)
	if exp.StartedAt != nil {
		fmt.Printf("Started:    %s\n", *exp.StartedAt)
	}
	if exp.ConcludedAt != nil {
		fmt.Printf("Concluded:  %s\n", *exp.ConcludedAt)
	}
	if exp.WinnerVariantID != nil {
		fmt.Printf("Winner:     %s\n", *exp.WinnerVariantID)
	}
	if exp.Notes != "" {
		fmt.Printf("Notes:      %s\n", exp.Notes)
	}

	fmt.Println("\nArms:")
	for _, arm := range exp.Arms {
		name := arm.VariantName
		if name == "" {
			name = arm.VariantID
		}
		fmt.Printf("  %s  weight=%.2f  (%s)\n", arm.VariantID, arm.Weight, name)
	}

	if len(exp.OutcomeCounts) > 0 {
		fmt.Println("\nOutcome Counts:")
		for vid, count := range exp.OutcomeCounts {
			fmt.Printf("  %s: %d\n", vid, count)
		}
	}

	return nil
}

func cmdCreate(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	skill := fs.String("skill", "", "Skill ID (required)")
	name := fs.String("name", "", "Experiment name (required)")
	hypothesis := fs.String("hypothesis", "", "Experiment hypothesis")
	id := fs.String("id", "", "Experiment ID (auto-generated if omitted)")

	// Collect --arm flags manually since flag package doesn't support repeated flags
	var armStrs []string
	// We parse interspersed but need to extract --arm values first
	var filteredArgs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--arm" || arg == "-arm" {
			if i+1 < len(args) {
				armStrs = append(armStrs, args[i+1])
				i++
			}
		} else if strings.HasPrefix(arg, "--arm=") {
			armStrs = append(armStrs, strings.TrimPrefix(arg, "--arm="))
		} else if strings.HasPrefix(arg, "-arm=") {
			armStrs = append(armStrs, strings.TrimPrefix(arg, "-arm="))
		} else {
			filteredArgs = append(filteredArgs, arg)
		}
	}

	if err := cliutil.ParseInterspersed(fs, filteredArgs); err != nil {
		return err
	}

	if *skill == "" {
		return fmt.Errorf("--skill is required")
	}
	if *name == "" {
		return fmt.Errorf("--name is required")
	}
	if len(armStrs) < 2 {
		return fmt.Errorf("at least 2 --arm flags are required (format: VARIANT_ID:WEIGHT)")
	}

	type armInput struct {
		VariantID string  `json:"variantId"`
		Weight    float64 `json:"weight"`
	}

	arms := make([]armInput, 0, len(armStrs))
	for _, s := range armStrs {
		parts := strings.SplitN(s, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid --arm format %q, expected VARIANT_ID:WEIGHT", s)
		}
		var w float64
		if _, err := fmt.Sscanf(parts[1], "%f", &w); err != nil {
			return fmt.Errorf("invalid weight %q in --arm %q: %w", parts[1], s, err)
		}
		arms = append(arms, armInput{VariantID: parts[0], Weight: w})
	}

	expID := *id
	if expID == "" {
		expID = strings.ToLower(strings.ReplaceAll(*name, " ", "-"))
	}

	req := struct {
		ID         string     `json:"id"`
		SkillID    string     `json:"skillId"`
		Name       string     `json:"name"`
		Hypothesis string     `json:"hypothesis,omitempty"`
		Arms       []armInput `json:"arms"`
	}{
		ID:         expID,
		SkillID:    *skill,
		Name:       *name,
		Hypothesis: *hypothesis,
		Arms:       arms,
	}

	var exp ExperimentResponse
	if err := ctx.Post("/experiments", req, &exp); err != nil {
		return fmt.Errorf("failed to create experiment: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(exp)
	}

	fmt.Printf("Created experiment: %s [%s] for skill %s (status: %s)\n", exp.Name, exp.ID, exp.SkillID, exp.Status)
	return nil
}

func cmdStart(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("start", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: experiment start <eid>")
	}
	eid := fs.Arg(0)

	var exp ExperimentResponse
	if err := ctx.Post(fmt.Sprintf("/experiments/%s/start", eid), struct{}{}, &exp); err != nil {
		return fmt.Errorf("failed to start experiment: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(exp)
	}

	fmt.Printf("Started experiment: %s [%s] (status: %s)\n", exp.Name, exp.ID, exp.Status)
	return nil
}

func cmdConclude(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("conclude", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	notes := fs.String("notes", "", "Conclusion notes")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 2 {
		return fmt.Errorf("usage: experiment conclude <eid> <winner-variant-id> [--notes TEXT]")
	}
	eid := fs.Arg(0)
	winnerVID := fs.Arg(1)

	req := struct {
		WinnerVariantID string `json:"winnerVariantId"`
		Notes           string `json:"notes,omitempty"`
	}{
		WinnerVariantID: winnerVID,
		Notes:           *notes,
	}

	var exp ExperimentResponse
	if err := ctx.Post(fmt.Sprintf("/experiments/%s/conclude", eid), req, &exp); err != nil {
		return fmt.Errorf("failed to conclude experiment: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(exp)
	}

	fmt.Printf("Concluded experiment: %s [%s] - winner: %s\n", exp.Name, exp.ID, winnerVID)
	return nil
}

func cmdOutcomes(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("outcomes", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: experiment outcomes <eid>")
	}
	eid := fs.Arg(0)

	var outcomes []OutcomeResponse
	if err := ctx.Get(fmt.Sprintf("/experiments/%s/outcomes", eid), &outcomes); err != nil {
		return fmt.Errorf("failed to get outcomes: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(outcomes)
	}

	if len(outcomes) == 0 {
		fmt.Println("No outcomes recorded")
		return nil
	}

	fmt.Printf("Outcomes (%d total):\n", len(outcomes))
	for _, o := range outcomes {
		fmt.Printf("  variant=%s  source=%s  schema=v%d  at=%s\n", o.VariantID, o.Source, o.SchemaVersion, o.RecordedAt)
	}
	return nil
}

func cmdDelete(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	force := fs.Bool("force", false, "Skip confirmation prompt")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: experiment delete <eid> [--force]")
	}
	eid := fs.Arg(0)

	if !*force {
		fmt.Printf("Delete experiment %q? [y/N]: ", eid)
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	if err := ctx.Delete(fmt.Sprintf("/experiments/%s", eid)); err != nil {
		return fmt.Errorf("failed to delete experiment: %w", err)
	}

	fmt.Printf("Deleted experiment: %s\n", eid)
	return nil
}
