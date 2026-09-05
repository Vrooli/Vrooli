package overview

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"deployment-manager/cli/cmdutil"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

type Commands struct {
	api *cliutil.APIClient
}

func New(api *cliutil.APIClient) *Commands {
	return &Commands{api: api}
}

func (c *Commands) Analyze(args []string) error {
	fs := flag.NewFlagSet("analyze", flag.ContinueOnError)
	format := fs.String("format", "", "output format (json|table)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("scenario is required")
	}
	scenario := remaining[0]
	body, err := c.api.Get("/api/v1/dependencies/analyze/"+scenario, nil)
	if err != nil {
		return err
	}
	if strings.ToLower(cmdutil.ResolveFormat(*format)) != "json" {
		return renderAnalyzeReport(scenario, body)
	}
	cmdutil.PrintByFormat(*format, body)
	return nil
}

func (c *Commands) Fitness(args []string) error {
	fs := flag.NewFlagSet("fitness", flag.ContinueOnError)
	tier := fs.String("tier", "2", "deployment tier")
	format := fs.String("format", "", "output format (json|table)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("scenario is required")
	}
	scenario := remaining[0]
	tierNum := cmdutil.TierToNumber(*tier)
	payload := map[string]interface{}{
		"scenario": scenario,
		"tiers":    []int{tierNum},
	}
	body, err := c.api.Request("POST", "/api/v1/fitness/score", nil, payload)
	if err != nil {
		return err
	}
	if strings.ToLower(cmdutil.ResolveFormat(*format)) != "json" {
		return renderFitnessReport(scenario, tierNum, body)
	}
	cmdutil.PrintByFormat(*format, body)
	return nil
}

func renderAnalyzeReport(scenario string, body []byte) error {
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		fmt.Println(string(body))
		return nil
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Scenario: %s", scenario),
		},
		ResultsHeading: "Dependency Analysis",
		RetrievalHints: []string{
			fmt.Sprintf("deployment-manager fitness %s --tier 2", scenario),
		},
	}
	for _, key := range sortedKeys(parsed) {
		report.Results = append(report.Results, fmt.Sprintf("%s: %v", key, parsed[key]))
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func renderFitnessReport(scenario string, tier int, body []byte) error {
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		fmt.Println(string(body))
		return nil
	}

	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Scenario: %s", scenario),
			fmt.Sprintf("Tier: %d", tier),
		},
	}
	if score, ok := parsed["score"]; ok {
		report.Status = append(report.Status, fmt.Sprintf("Score: %v", score))
	}
	triage := cliapp.TriageGroup{Heading: "Fitness Factors"}
	for _, key := range sortedKeys(parsed) {
		if key == "score" {
			continue
		}
		triage.Items = append(triage.Items, fmt.Sprintf("%s: %v", key, parsed[key]))
	}
	if len(triage.Items) > 0 {
		report.Triage = append(report.Triage, triage)
	}
	report.NextSteps = []string{
		fmt.Sprintf("deployment-manager analyze %s", scenario),
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func sortedKeys(values map[string]interface{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
