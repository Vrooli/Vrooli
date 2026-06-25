package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdInitiativesContext(args []string) error {
	fs := flag.NewFlagSet("initiatives context", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	scenarioFlag := fs.String("scenario", "", "Scenario name — returns every initiative + orphan item targeting this scenario")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	name := strings.TrimSpace(*nameFlag)
	scenario := strings.TrimSpace(*scenarioFlag)

	switch {
	case name != "" && scenario != "":
		return fmt.Errorf("usage: initiatives context --name NAME | --scenario SCENARIO\n\n--name and --scenario are mutually exclusive")
	case scenario != "":
		return a.runScenarioContext(scenario, *jsonOut)
	case name == "":
		return fmt.Errorf("usage: initiatives context --name NAME | --scenario SCENARIO [--json]\n\none of --name or --scenario is required")
	}

	body, err := a.core.Get("/initiatives/"+name+"/context", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[InitiativeContextResponse](body)
	if err != nil {
		return err
	}

	printInitiativeContext(response)

	printCommandListSection("Next Steps", []string{
		cliCommand("initiatives", "get", "--name", response.Initiative.Name),
	})
	return nil
}

// printInitiativeContext renders the initiative detail, rollup, members, and
// upstream/downstream initiative sections.
func printInitiativeContext(response InitiativeContextResponse) {
	init := response.Initiative
	rollup := response.Rollup

	printSection("Initiative")
	fmt.Printf("  Name: %s\n", init.Name)
	fmt.Printf("  Title: %s\n", init.Title)
	if init.Description != "" {
		fmt.Printf("  Description: %s\n", init.Description)
	}
	fmt.Printf("  Status: %s\n", init.Status)
	if init.Mode != "" {
		fmt.Printf("  Mode: %s\n", init.Mode)
	}
	if init.Priority > 0 {
		fmt.Printf("  Priority: %d\n", init.Priority)
	}
	if len(init.DependsOn) > 0 {
		fmt.Printf("  Depends on: %s\n", strings.Join(init.DependsOn, ", "))
	}

	printSection("Rollup")
	fmt.Printf("  Total: %d | Completed: %d | In Progress: %d | Failed: %d | Pending: %d\n",
		rollup.Total, rollup.Completed, rollup.InProgress, rollup.Failed, rollup.Pending)

	printSection(fmt.Sprintf("Members (%d)", len(response.Items)))
	if len(response.Items) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, item := range response.Items {
			archived := ""
			if item.ArchivedAt != nil {
				archived = " [archived]"
			}
			fmt.Printf("  - %s/%s — %s (status=%s, priority=%d)%s\n",
				item.Kind, item.Name, item.Title, item.Status, item.Priority, archived)
			if len(item.DependsOn) > 0 {
				fmt.Printf("      depends on: %s\n", strings.Join(item.DependsOn, ", "))
			}
		}
	}

	printInitiativeRefs("Upstream initiatives", response.UpstreamInitiatives)
	printInitiativeRefs("Downstream initiatives", response.DownstreamInitiatives)
}

// printInitiativeRefs prints a titled section listing related initiatives.
func printInitiativeRefs(title string, refs []Initiative) {
	printSection(fmt.Sprintf("%s (%d)", title, len(refs)))
	if len(refs) == 0 {
		fmt.Println("  (none)")
		return
	}
	for _, ref := range refs {
		fmt.Printf("  - %s — %s (status=%s)\n", ref.Name, ref.Title, ref.Status)
	}
}

// runScenarioContext calls GET /scenarios/{name}/context and renders the
// coverage view: initiatives whose items target the scenario, orphan items
// targeting the scenario, and the combined rollup. Follows the Data
// Retrieval output contract (Summary -> Results -> Next Steps).
func (a *App) runScenarioContext(scenario string, jsonOut bool) error {
	body, err := a.core.Get("/scenarios/"+scenario+"/context", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[ScenarioContextResponse](body)
	if err != nil {
		return err
	}

	printSection("Summary")
	fmt.Printf("  Scenario: %s\n", response.ScenarioName)
	fmt.Printf("  Initiatives targeting: %d\n", len(response.Initiatives))
	fmt.Printf("  Orphan items (targeting but not assigned to an initiative): %d\n", len(response.OrphanItems))

	printSection("Rollup")
	r := response.Rollup
	fmt.Printf("  Total: %d | Completed: %d | In Progress: %d | Failed: %d | Pending: %d | Archived: %d\n",
		r.Total, r.Completed, r.InProgress, r.Failed, r.Pending, r.Archived)

	printSection(fmt.Sprintf("Initiatives (%d)", len(response.Initiatives)))
	if len(response.Initiatives) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, init := range response.Initiatives {
			i := init.Initiative
			rollup := init.Rollup
			fmt.Printf("  - %s (%s) — %s\n", i.Name, i.Status, i.Title)
			fmt.Printf("      Items: %d total, %d completed, %d in-progress, %d failed, %d pending\n",
				rollup.Total, rollup.Completed, rollup.InProgress, rollup.Failed, rollup.Pending)
		}
	}

	printSection(fmt.Sprintf("Orphan items (%d)", len(response.OrphanItems)))
	if len(response.OrphanItems) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, o := range response.OrphanItems {
			archived := ""
			if o.ArchivedAt != nil {
				archived = " [archived]"
			}
			fmt.Printf("  - %s/%s — %s (status=%s, priority=%d)%s\n",
				o.Kind, o.Name, o.Title, o.Status, o.Priority, archived)
		}
	}

	nextSteps := []string{
		cliCommand("backlog", "list", "--scenario", scenario),
		cliCommand("initiatives", "list", "--scenario", scenario),
	}
	if len(response.OrphanItems) > 0 {
		nextSteps = append(nextSteps, cliCommand("initiatives", "create",
			"--data", fmt.Sprintf("'{\"name\":\"%s-readiness\",\"title\":\"%s readiness\",\"items\":[…orphans to adopt…]}'",
				scenario, scenario)))
	}
	printCommandListSection("Next Steps", nextSteps)
	return nil
}
