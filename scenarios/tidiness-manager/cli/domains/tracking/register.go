package tracking

import (
	"fmt"
	"strings"
	"tidiness-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register() cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "tracking",
		Description: "Wrap visited-tracker workflows used during refactoring",
		Subcommands: []cliapp.Command{
			{Name: "visit", Description: "Record a refactor visit", Run: runVisit},
			{Name: "exclude", Description: "Exclude a file from future recommendations", Run: runExclude},
			{Name: "campaign-note", Description: "Add a campaign handoff note", Run: runCampaignNote},
		},
	}
}

func runVisit(args []string) error {
	fs := support.NewFlagSet("tracking visit")
	scenario := fs.String("scenario", "", "Scenario name")
	note := fs.String("note", "", "Optional note")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: tracking visit <file-path> --scenario <name> [--note ...]")
	}
	if strings.TrimSpace(*scenario) == "" {
		return fmt.Errorf("--scenario is required")
	}
	location, err := support.ScenarioPath(*scenario)
	if err != nil {
		return err
	}
	vtArgs := []string{"visit", fs.Arg(0), "--location", location, "--tag", "refactor"}
	if strings.TrimSpace(*note) != "" {
		vtArgs = append(vtArgs, "--note", strings.TrimSpace(*note))
	}
	return support.RunVisitedTracker(vtArgs...)
}

func runExclude(args []string) error {
	fs := support.NewFlagSet("tracking exclude")
	scenario := fs.String("scenario", "", "Scenario name")
	reason := fs.String("reason", "", "Reason for exclusion")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: tracking exclude <file-path> --scenario <name> [--reason ...]")
	}
	if strings.TrimSpace(*scenario) == "" {
		return fmt.Errorf("--scenario is required")
	}
	location, err := support.ScenarioPath(*scenario)
	if err != nil {
		return err
	}
	filePath := fs.Arg(0)
	visitNote := "Excluded"
	if strings.TrimSpace(*reason) != "" {
		visitNote = "Excluded: " + strings.TrimSpace(*reason)
	}
	if err := support.RunVisitedTracker("visit", filePath, "--location", location, "--tag", "refactor", "--note", visitNote); err != nil {
		return err
	}
	vtArgs := []string{"exclude", filePath, "--location", location, "--tag", "refactor"}
	if strings.TrimSpace(*reason) != "" {
		vtArgs = append(vtArgs, "--reason", strings.TrimSpace(*reason))
	}
	return support.RunVisitedTracker(vtArgs...)
}

func runCampaignNote(args []string) error {
	fs := support.NewFlagSet("tracking campaign-note")
	scenario := fs.String("scenario", "", "Scenario name")
	note := fs.String("note", "", "Campaign note")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*scenario) == "" {
		return fmt.Errorf("--scenario is required")
	}
	if strings.TrimSpace(*note) == "" {
		return fmt.Errorf("--note is required")
	}
	location, err := support.ScenarioPath(*scenario)
	if err != nil {
		return err
	}
	return support.RunVisitedTracker(
		"campaigns", "note",
		"--location", location,
		"--tag", "refactor",
		"--name", *scenario+" - Code Refactoring",
		"--note", strings.TrimSpace(*note),
	)
}
