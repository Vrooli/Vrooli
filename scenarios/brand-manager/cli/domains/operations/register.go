package operations

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "brand-manager"

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Operations",
		Commands: []cliapp.Command{
			{Name: "generate", NeedsAPI: true, Description: "Generate brand elements via AI", Run: func(args []string) error { return runGenerate(core, args) }},
			{Name: "discover", NeedsAPI: true, Description: "Discover existing branding in a scenario", Run: func(args []string) error { return runDiscover(core, args) }},
			{Name: "apply", NeedsAPI: true, Description: "Apply a brand to a scenario", Run: func(args []string) error { return runApply(core, args) }},
			{Name: "scan", NeedsAPI: true, Description: "Scan scenario for inline brand markers", Run: func(args []string) error { return runScan(core, args) }},
		},
	}
}

func runDiscover(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("discover", flag.ContinueOnError)
	doImport := fs.Bool("import", false, "Import discovered state as a new brand")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: brand-manager discover <scenario> [--import] [--json]")
	}

	scenario := fs.Arg(0)
	path := "/discover/" + scenario
	method := "GET"
	if *doImport {
		path += "/import"
		method = "POST"
	}
	body, err := core.Request(method, path, nil, nil)
	if method == "GET" {
		body, err = core.Get(path, nil)
	}
	if err != nil {
		return err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	if *doImport {
		report := cliapp.MutationReport{
			Result:      []string{"Discovered branding imported", "Scenario: " + scenario},
			Changes:     []string{fmt.Sprintf("Confidence: %.0f%%", floatMetric(result["confidence"])*100)},
			NextCommand: []string{},
		}
		if brand, ok := result["brand"].(map[string]interface{}); ok {
			report.Result = append(report.Result, fmt.Sprintf("Brand ID: %v", brand["id"]))
			report.Changes = append(report.Changes, fmt.Sprintf("Brand name: %v", brand["name"]))
			report.NextCommand = append(report.NextCommand, cliName+" get "+fmt.Sprintf("%v", brand["id"]))
		}
		if *jsonOut {
			return cliapp.PrintReportJSON(os.Stdout, report)
		}
		return cliapp.RenderMutationReport(os.Stdout, report)
	}

	report := cliapp.ListReport{
		Summary:        []string{"Scenario: " + scenario, fmt.Sprintf("Confidence: %.0f%%", floatMetric(result["confidence"])*100)},
		Results:        discoverResults(result),
		RetrievalHints: []string{cliName + " discover " + scenario + " --import", cliName + " scan " + scenario},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runApply(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("apply", flag.ContinueOnError)
	scenario := fs.String("scenario", "", "Scenario to apply brand to (required)")
	elements := fs.String("elements", "", "Comma-separated elements to apply (e.g. colors,typography)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 || *scenario == "" {
		return fmt.Errorf("usage: brand-manager apply <brand-id> --scenario NAME [--elements colors,typography] [--json]")
	}

	brandID := fs.Arg(0)
	payload := map[string]interface{}{"scenario_name": *scenario}
	if *elements != "" {
		payload["elements"] = cliutil.ParseCSV(*elements)
	}
	body, err := core.Request("POST", "/brands/"+brandID+"/apply", nil, payload)
	if err != nil {
		return err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:  []string{"Brand applied", "Brand ID: " + brandID, "Scenario: " + *scenario},
		Changes: applyChanges(result),
		NextCommand: []string{
			cliName + " scenario-status " + *scenario,
		},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runGenerate(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	elements := fs.String("elements", "colors,typography,voice", "Comma-separated elements to generate")
	model := fs.String("model", "", "AI model override")
	imageType := fs.String("image", "", "Generate an image asset: 'logo' or 'favicon'")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: brand-manager generate <brand-id> [--elements colors,typography,voice] [--image logo|favicon] [--model MODEL] [--json]")
	}

	brandID := fs.Arg(0)
	path := "/brands/" + brandID + "/generate"
	payload := map[string]interface{}{"elements": cliutil.ParseCSV(*elements)}
	if *imageType != "" {
		path = "/brands/" + brandID + "/generate/image"
		payload = map[string]interface{}{"type": *imageType}
	}
	if *model != "" {
		payload["model"] = *model
	}
	body, err := core.Request("POST", path, nil, payload)
	if err != nil {
		return err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Generation complete", "Brand ID: " + brandID},
		Changes:     generationChanges(result, *imageType),
		NextCommand: []string{cliName + " get " + brandID},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runScan(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: brand-manager scan <scenario> [--json]")
	}

	scenario := fs.Arg(0)
	body, err := core.Get("/scan/"+scenario, nil)
	if err != nil {
		return err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{
			"Scenario: " + scenario,
			fmt.Sprintf("CSS markers: %v", result["css_markers"]),
			fmt.Sprintf("JSON keys: %v", result["json_keys"]),
			fmt.Sprintf("Total markers: %v", result["total"]),
		},
		Results:        scanResults(result),
		RetrievalHints: []string{cliName + " discover " + scenario, cliName + " scenario-status " + scenario},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func discoverResults(result map[string]interface{}) []string {
	lines := []string{}
	if sources, ok := result["sources"].([]interface{}); ok {
		for _, raw := range sources {
			if source, ok := raw.(map[string]interface{}); ok {
				lines = append(lines, fmt.Sprintf("%v | %v | %v field(s) | %.0f%% confidence", source["file"], source["type"], source["fields"], floatMetric(source["confidence"])*100))
			}
		}
	}
	if suggestions, ok := result["suggestions"].([]interface{}); ok {
		for _, suggestion := range suggestions {
			lines = append(lines, fmt.Sprintf("Suggestion: %v", suggestion))
		}
	}
	if len(lines) == 0 {
		return []string{"No branding state found"}
	}
	if result["draft_brand"] != nil {
		lines = append(lines, "Draft brand available for import")
	}
	return lines
}

func applyChanges(result map[string]interface{}) []string {
	lines := []string{}
	if applied, ok := result["applied"].([]interface{}); ok {
		for _, raw := range applied {
			if action, ok := raw.(map[string]interface{}); ok {
				lines = append(lines, fmt.Sprintf("Applied %v -> %v (%v)", action["element"], action["file"], action["type"]))
			}
		}
	}
	if skipped, ok := result["skipped"].([]interface{}); ok {
		for _, raw := range skipped {
			if skip, ok := raw.(map[string]interface{}); ok {
				lines = append(lines, fmt.Sprintf("Skipped %v: %v", skip["element"], skip["reason"]))
			}
		}
	}
	if len(lines) == 0 {
		lines = append(lines, "No changes were reported by the API")
	}
	return lines
}

func generationChanges(result map[string]interface{}, imageType string) []string {
	lines := []string{}
	if imageType != "" {
		lines = append(lines, fmt.Sprintf("Asset type: %v", result["type"]), fmt.Sprintf("Asset ID: %v", result["asset_id"]))
	} else {
		if applied, ok := result["applied"].([]interface{}); ok && len(applied) > 0 {
			values := make([]string, 0, len(applied))
			for _, raw := range applied {
				values = append(values, fmt.Sprintf("%v", raw))
			}
			lines = append(lines, "Generated elements: "+strings.Join(values, ", "))
		}
	}
	if provider, ok := result["provider"].(string); ok && provider != "" {
		lines = append(lines, fmt.Sprintf("Provider: %s (%v)", provider, result["model"]))
	}
	return lines
}

func scanResults(result map[string]interface{}) []string {
	results, ok := result["results"].([]interface{})
	if !ok || len(results) == 0 {
		return nil
	}
	lines := make([]string, 0, len(results))
	for _, raw := range results {
		if marker, ok := raw.(map[string]interface{}); ok {
			lines = append(lines, fmt.Sprintf("%v:%v [%v] %v", marker["file"], marker["line"], marker["type"], marker["marker"]))
		}
	}
	return lines
}

func floatMetric(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	default:
		return 0
	}
}
