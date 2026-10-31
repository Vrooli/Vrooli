// Package goal exposes the thin goal-loop harness. The goal-loop skill owns
// target resolution, routing, budgets, and stop rules; this package only
// loads that skill and carries the caller's sentence into its first phase.
package goal

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"prompt-manager/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliapp"
)

type readRequest struct {
	Identifiers []string `json:"identifiers"`
	Resolve     string   `json:"resolve,omitempty"`
	Output      string   `json:"output,omitempty"`
	Format      string   `json:"format,omitempty"`
}

type readResponse struct {
	Skills []struct {
		ID      string `json:"id"`
		Content string `json:"content"`
	} `json:"skills"`
}

// Commands returns the goal-loop harness command. --cadence is optional on
// purpose: the skill says the caller supplies cadence and the loop has no
// default.
func Commands(ctx appctx.Context) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Goals",
		Commands: []cliapp.Command{{
			Name:        "goal",
			Aliases:     []string{"goals"},
			NeedsAPI:    true,
			Description: "Begin a goal-loop cycle using the goal-loop skill",
			Usage:       `prompt-manager goal "<goal sentence>" [--cadence <interval>] [--json]`,
			HelpText:    "Loads goal-loop and passes the sentence to its Phase 0. Cadence is caller-owned and has no default.",
			Run: func(args []string) error {
				return run(ctx, args)
			},
		}},
	}
}

func run(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("goal", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	cadence := fs.String("cadence", "", "Caller-provided wake cadence; no default")
	jsonOutput := fs.Bool("json", false, "Emit the harness envelope as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 || strings.TrimSpace(fs.Arg(0)) == "" {
		return fmt.Errorf(`usage: prompt-manager goal "<goal sentence>" [--cadence <interval>] [--json]`)
	}

	var response readResponse
	if err := ctx.Post("/skills/read", readRequest{
		Identifiers: []string{"goal-loop"},
		Resolve:     "id",
		Output:      "skills",
		Format:      "markdown",
	}, &response); err != nil {
		return fmt.Errorf("failed to read goal-loop skill: %w", err)
	}
	if len(response.Skills) != 1 || strings.TrimSpace(response.Skills[0].Content) == "" {
		return fmt.Errorf("goal-loop skill was not returned")
	}

	envelope := struct {
		Goal    string `json:"goal"`
		SkillID string `json:"skill_id"`
		Cadence string `json:"cadence,omitempty"`
		Phase   string `json:"phase"`
		Skill   string `json:"skill"`
	}{
		Goal: fs.Arg(0), SkillID: "goal-loop", Cadence: strings.TrimSpace(*cadence),
		Phase: "resolve-and-gate", Skill: response.Skills[0].Content,
	}
	if *jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(envelope)
	}
	fmt.Printf("Goal: %s\nSkill: goal-loop\nPhase: %s\n", envelope.Goal, envelope.Phase)
	if envelope.Cadence == "" {
		fmt.Println("Cadence: caller-owned (none supplied)")
	} else {
		fmt.Printf("Cadence: %s\n", envelope.Cadence)
	}
	fmt.Println("\nPass this sentence to the goal-loop skill:")
	fmt.Println(envelope.Skill)
	return nil
}
