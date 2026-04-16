package brands

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "brand-manager"

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Brands",
		Commands: []cliapp.Command{
			{Name: "create", NeedsAPI: true, Description: "Create a new brand (--name required)", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "list", NeedsAPI: true, Description: "List brands [--name FILTER] [--limit N] [--offset N]", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", NeedsAPI: true, Description: "Get a brand by ID", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "update", NeedsAPI: true, Description: "Update a brand by ID", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "delete", NeedsAPI: true, Description: "Delete a brand by ID", Run: func(args []string) error { return runDelete(core, args) }},
			{Name: "versions", NeedsAPI: true, Description: "List version history for a brand", Run: func(args []string) error { return runVersions(core, args) }},
		},
	}
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	name := fs.String("name", "", "Brand name (required)")
	desc := fs.String("description", "", "Brand description")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *name == "" {
		return fmt.Errorf("--name is required\nUsage: brand-manager create --name NAME [--description DESC] [--json]")
	}

	payload := map[string]string{"name": *name}
	if *desc != "" {
		payload["description"] = *desc
	}
	body, err := core.Request("POST", "/brands", nil, payload)
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result: []string{"Brand created", fmt.Sprintf("Brand ID: %v", result["id"])},
		Changes: []string{
			fmt.Sprintf("Name: %v", result["name"]),
		},
		NextCommand: []string{cliName + " get " + fmt.Sprintf("%v", result["id"]), cliName + " generate " + fmt.Sprintf("%v", result["id"])},
	}
	if descValue, ok := result["description"].(string); ok && descValue != "" {
		report.Changes = append(report.Changes, "Description: "+descValue)
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	nameFilter := fs.String("name", "", "Filter brands by name (substring match)")
	limit := fs.Int("limit", 0, "Maximum number of results")
	offset := fs.Int("offset", 0, "Skip this many results")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	query := url.Values{}
	if *nameFilter != "" {
		query.Set("name", *nameFilter)
	}
	if *limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", *limit))
	}
	if *offset > 0 {
		query.Set("offset", fmt.Sprintf("%d", *offset))
	}

	body, err := core.Get("/brands", query)
	if err != nil {
		return err
	}
	var brands []map[string]interface{}
	if err := json.Unmarshal(body, &brands); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Brands found: %d", len(brands))},
		Results:        renderBrandRows(brands),
		RetrievalHints: []string{cliName + " get <brand-id>", cliName + " create --name \"New Brand\""},
	}
	if *nameFilter != "" {
		report.Summary = append(report.Summary, "Name filter: "+*nameFilter)
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: brand-manager get <brand-id> [--json]")
	}

	id := fs.Arg(0)
	body, err := core.Get("/brands/"+id, nil)
	if err != nil {
		return err
	}
	var brand map[string]interface{}
	if err := json.Unmarshal(body, &brand); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Brand: %v", brand["name"]), fmt.Sprintf("Brand ID: %v", brand["id"])},
		ResultsHeading: "Details",
		Results:        detailLines(brand),
		RetrievalHints: []string{cliName + " update " + id + " --name \"...\"", cliName + " versions " + id},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	name := fs.String("name", "", "New brand name")
	desc := fs.String("description", "", "New description")
	notes := fs.String("notes", "", "New notes")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: brand-manager update <brand-id> [--name NAME] [--description DESC] [--notes NOTES] [--json]")
	}
	id := fs.Arg(0)

	payload := make(map[string]string)
	if *name != "" {
		payload["name"] = *name
	}
	if *desc != "" {
		payload["description"] = *desc
	}
	if *notes != "" {
		payload["notes"] = *notes
	}
	if len(payload) == 0 {
		return fmt.Errorf("at least one field to update is required (--name, --description, --notes)")
	}

	body, err := core.Request("PUT", "/brands/"+id, nil, payload)
	if err != nil {
		return err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Brand updated", fmt.Sprintf("Brand ID: %v", result["id"])},
		Changes:     changedFields(payload),
		NextCommand: []string{cliName + " get " + id, cliName + " versions " + id},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: brand-manager delete <brand-id> [--json]")
	}
	id := fs.Arg(0)
	if _, err := core.Request("DELETE", "/brands/"+id, nil, nil); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Brand deleted", "Brand ID: " + id},
		Changes:     []string{"Removed brand definition from the catalog"},
		NextCommand: []string{cliName + " list"},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runVersions(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("versions", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: brand-manager versions <brand-id> [--json]")
	}

	brandID := fs.Arg(0)
	body, err := core.Get("/brands/"+brandID+"/versions", nil)
	if err != nil {
		return err
	}
	var versions []map[string]interface{}
	if err := json.Unmarshal(body, &versions); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"Brand ID: " + brandID, fmt.Sprintf("Versions found: %d", len(versions))},
		Results:        renderVersions(versions),
		RetrievalHints: []string{cliName + " get " + brandID},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func renderBrandRows(brands []map[string]interface{}) []string {
	if len(brands) == 0 {
		return nil
	}
	lines := make([]string, 0, len(brands))
	for _, brand := range brands {
		lines = append(lines, fmt.Sprintf("%v | %v | v%v", brand["id"], brand["name"], brand["version"]))
	}
	return lines
}

func detailLines(brand map[string]interface{}) []string {
	lines := []string{
		fmt.Sprintf("Version: %v", brand["version"]),
	}
	if desc, ok := brand["description"].(string); ok && desc != "" {
		lines = append(lines, "Description: "+desc)
	}
	appendNestedMap(&lines, "Colors", brand["colors"])
	appendNestedMap(&lines, "Typography", brand["typography"])
	appendNestedMap(&lines, "Voice", brand["voice"])
	return lines
}

func appendNestedMap(lines *[]string, heading string, value interface{}) {
	nested, ok := value.(map[string]interface{})
	if !ok || len(nested) == 0 {
		return
	}
	*lines = append(*lines, heading+":")
	keys := make([]string, 0, len(nested))
	for key := range nested {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		*lines = append(*lines, fmt.Sprintf("  %s: %v", key, nested[key]))
	}
}

func changedFields(payload map[string]string) []string {
	keys := make([]string, 0, len(payload))
	for key := range payload {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("%s: %s", strings.Title(key), payload[key]))
	}
	return lines
}

func renderVersions(versions []map[string]interface{}) []string {
	if len(versions) == 0 {
		return nil
	}
	lines := make([]string, 0, len(versions))
	for _, version := range versions {
		timestamp := ""
		if created, ok := version["created_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339Nano, created); err == nil {
				timestamp = t.Format("2006-01-02 15:04")
			}
		}
		lines = append(lines, fmt.Sprintf("v%v | %v | %s", version["version"], version["id"], timestamp))
	}
	return lines
}
