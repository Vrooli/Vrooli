// DOC: docs/reference/cli-commands.md
// DOC: docs/internal/SEAMS.md#4-cli--api-network-seam
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const (
	appName        = "brand-manager"
	appVersion     = "0.1.0"
	defaultAPIBase = ""
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

type App struct {
	core *cliapp.ScenarioApp
}

func NewApp() (*App, error) {
	env := cliapp.StandardScenarioEnv(appName, cliapp.ScenarioEnvOptions{
		ExtraAPIEnvVars: []string{"API_BASE_URL", "VITE_API_BASE_URL"},
	})
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name:              appName,
		Version:           appVersion,
		Description:       "Brand Manager \u2013 Full Branding Lifecycle for All Scenarios CLI",
		DefaultAPIBase:    defaultAPIBase,
		APIEnvVars:        env.APIEnvVars,
		APIPortEnvVars:    env.APIPortEnvVars,
		APIPortDetector:   cliutil.DetectPortFromVrooli(appName, "API_PORT"),
		ConfigDirEnvVars:  env.ConfigDirEnvVars,
		SourceRootEnvVars: env.SourceRootEnvVars,
		TokenEnvVars:      env.TokenEnvVars,
		BuildFingerprint:  buildFingerprint,
		BuildTimestamp:    buildTimestamp,
		BuildSourceRoot:   buildSourceRoot,
		AllowAnonymous:    true,
	})
	if err != nil {
		return nil, err
	}
	app := &App{core: core}
	app.core.SetCommands(app.registerCommands())
	return app, nil
}

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}

func (a *App) registerCommands() []cliapp.CommandGroup {
	health := cliapp.CommandGroup{
		Title: "Health",
		Commands: []cliapp.Command{
			{Name: "status", Aliases: []string{"health"}, NeedsAPI: true, Description: "Check API health and readiness", Run: a.cmdStatus},
		},
	}

	// [REQ:BM-REQ-CLI-CRUD] Brand CRUD + query commands
	brands := cliapp.CommandGroup{
		Title: "Brands",
		Commands: []cliapp.Command{
			{Name: "create", NeedsAPI: true, Description: "Create a new brand (--name required)", Run: a.cmdCreate},
			{Name: "list", NeedsAPI: true, Description: "List brands [--name FILTER] [--limit N] [--offset N]", Run: a.cmdList},
			{Name: "get", NeedsAPI: true, Description: "Get a brand by ID", Run: a.cmdGet},
			{Name: "update", NeedsAPI: true, Description: "Update a brand by ID", Run: a.cmdUpdate},
			{Name: "delete", NeedsAPI: true, Description: "Delete a brand by ID", Run: a.cmdDelete},
			{Name: "versions", NeedsAPI: true, Description: "List version history for a brand", Run: a.cmdVersions},
		},
	}

	// [REQ:BM-REQ-CLI-CRUD] Assignment + scenario commands
	assignments := cliapp.CommandGroup{
		Title: "Assignments",
		Commands: []cliapp.Command{
			{Name: "assign", NeedsAPI: true, Description: "Assign a brand to a scenario", Run: a.cmdAssign},
			{Name: "unassign", NeedsAPI: true, Description: "Remove a brand assignment by ID", Run: a.cmdUnassign},
			{Name: "scenario-status", NeedsAPI: true, Description: "Check branding status for a scenario", Run: a.cmdScenarioStatus},
		},
	}

	// [REQ:BM-REQ-CLI-DISCOVER] [REQ:BM-REQ-CLI-APPLY] [REQ:BM-REQ-CLI-STATUS] [REQ:BM-REQ-CLI-GEN]
	operations := cliapp.CommandGroup{
		Title: "Operations",
		Commands: []cliapp.Command{
			{Name: "generate", NeedsAPI: true, Description: "Generate brand elements via AI", Run: a.cmdGenerate},
			{Name: "discover", NeedsAPI: true, Description: "Discover existing branding in a scenario", Run: a.cmdDiscover},
			{Name: "apply", NeedsAPI: true, Description: "Apply a brand to a scenario", Run: a.cmdApply},
			{Name: "scan", NeedsAPI: true, Description: "Scan scenario for inline brand markers", Run: a.cmdScan},
		},
	}

	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
		},
	}

	return []cliapp.CommandGroup{health, brands, assignments, operations, config}
}

func (a *App) apiPath(v1Path string) string {
	v1Path = strings.TrimSpace(v1Path)
	if v1Path == "" {
		return ""
	}
	if !strings.HasPrefix(v1Path, "/") {
		v1Path = "/" + v1Path
	}
	base := strings.TrimRight(strings.TrimSpace(a.core.HTTPClient.BaseURL()), "/")
	if strings.HasSuffix(base, "/api/v1") {
		return v1Path
	}
	return "/api/v1" + v1Path
}

// --- Health ---

type healthResponse struct {
	Status     string            `json:"status"`
	Service    string            `json:"service"`
	Version    string            `json:"version"`
	Readiness  bool              `json:"readiness"`
	Timestamp  string            `json:"timestamp"`
	Deps       map[string]string `json:"dependencies"`
	Error      string            `json:"error,omitempty"`
	Message    string            `json:"message,omitempty"`
	Operations map[string]any    `json:"operations,omitempty"`
}

func (a *App) cmdStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := a.core.APIClient.Get(a.apiPath("/health"), nil)
	if err != nil {
		return err
	}

	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}

	var parsed healthResponse
	if unmarshalErr := json.Unmarshal(body, &parsed); unmarshalErr == nil && parsed.Status != "" {
		fmt.Printf("Status: %s\n", parsed.Status)
		fmt.Printf("Ready: %v\n", parsed.Readiness)
		if parsed.Service != "" {
			fmt.Printf("Service: %s\n", parsed.Service)
		}
		if parsed.Version != "" {
			fmt.Printf("Version: %s\n", parsed.Version)
		}
		if len(parsed.Deps) > 0 {
			fmt.Println("Dependencies:")
			for key, value := range parsed.Deps {
				fmt.Printf("  %s: %s\n", key, value)
			}
		}
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}

// --- Brand CRUD ---

// cmdCreate handles `brand-manager create --name "My Brand"`. [REQ:BM-REQ-CLI-CRUD]
func (a *App) cmdCreate(args []string) error {
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

	body, err := a.core.APIClient.Request("POST", a.apiPath("/brands"), nil, payload)
	if err != nil {
		return err
	}

	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err == nil {
		fmt.Printf("Created brand: %s (id: %s)\n", result["name"], result["id"])
	} else {
		cliutil.PrintJSON(body)
	}
	return nil
}

// cmdList handles `brand-manager list`. [REQ:BM-REQ-CLI-CRUD]
func (a *App) cmdList(args []string) error {
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

	body, err := a.core.APIClient.Get(a.apiPath("/brands"), query)
	if err != nil {
		return err
	}

	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}

	var brands []map[string]interface{}
	if err := json.Unmarshal(body, &brands); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if len(brands) == 0 {
		fmt.Println("No brands found.")
		return nil
	}

	for _, b := range brands {
		fmt.Printf("  %s  %s  (v%v)\n", b["id"], b["name"], b["version"])
	}
	return nil
}

// cmdGet handles `brand-manager get <id>`. [REQ:BM-REQ-CLI-CRUD]
func (a *App) cmdGet(args []string) error {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() == 0 {
		return fmt.Errorf("usage: brand-manager get <brand-id> [--json]")
	}
	id := fs.Arg(0)

	body, err := a.core.APIClient.Get(a.apiPath("/brands/"+id), nil)
	if err != nil {
		return err
	}

	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}

	// Human-friendly output: show key fields
	var brand map[string]interface{}
	if err := json.Unmarshal(body, &brand); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Name: %s\n", brand["name"])
	fmt.Printf("ID: %s\n", brand["id"])
	fmt.Printf("Version: %v\n", brand["version"])
	if desc, ok := brand["description"].(string); ok && desc != "" {
		fmt.Printf("Description: %s\n", desc)
	}
	if colors, ok := brand["colors"].(map[string]interface{}); ok {
		fmt.Println("Colors:")
		for k, v := range colors {
			if vs, ok := v.(string); ok && vs != "" {
				fmt.Printf("  %s: %s\n", k, vs)
			}
		}
	}
	if typo, ok := brand["typography"].(map[string]interface{}); ok {
		fmt.Println("Typography:")
		for k, v := range typo {
			if vs, ok := v.(string); ok && vs != "" {
				fmt.Printf("  %s: %s\n", k, vs)
			}
		}
	}
	if voice, ok := brand["voice"].(map[string]interface{}); ok {
		fmt.Println("Voice:")
		for k, v := range voice {
			fmt.Printf("  %s: %v\n", k, v)
		}
	}
	return nil
}

// cmdUpdate handles `brand-manager update <id> --name "New Name"`. [REQ:BM-REQ-CLI-CRUD]
func (a *App) cmdUpdate(args []string) error {
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

	body, err := a.core.APIClient.Request("PUT", a.apiPath("/brands/"+id), nil, payload)
	if err != nil {
		return err
	}

	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err == nil {
		fmt.Printf("Updated brand: %s (v%v)\n", result["name"], result["version"])
	} else {
		cliutil.PrintJSON(body)
	}
	return nil
}

// cmdDelete handles `brand-manager delete <id>`. [REQ:BM-REQ-CLI-CRUD]
func (a *App) cmdDelete(args []string) error {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() == 0 {
		return fmt.Errorf("usage: brand-manager delete <brand-id> [--json]")
	}
	id := fs.Arg(0)

	_, err := a.core.APIClient.Request("DELETE", a.apiPath("/brands/"+id), nil, nil)
	if err != nil {
		return err
	}

	if *jsonOut {
		fmt.Println(`{"success": true, "deleted": "` + id + `"}`)
		return nil
	}

	fmt.Printf("Deleted brand: %s\n", id)
	return nil
}

// --- Versions ---

// cmdVersions handles `brand-manager versions <brand-id>`. [REQ:BM-REQ-CLI-CRUD]
func (a *App) cmdVersions(args []string) error {
	fs := flag.NewFlagSet("versions", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() == 0 {
		return fmt.Errorf("usage: brand-manager versions <brand-id> [--json]")
	}
	brandID := fs.Arg(0)

	body, err := a.core.APIClient.Get(a.apiPath("/brands/"+brandID+"/versions"), nil)
	if err != nil {
		return err
	}

	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}

	var versions []map[string]interface{}
	if err := json.Unmarshal(body, &versions); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if len(versions) == 0 {
		fmt.Println("No versions found.")
		return nil
	}

	for _, v := range versions {
		ts := ""
		if created, ok := v["created_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339Nano, created); err == nil {
				ts = t.Format("2006-01-02 15:04")
			}
		}
		fmt.Printf("  v%v  %s  %s\n", v["version"], v["id"], ts)
	}
	return nil
}

// --- Assignments ---

// cmdAssign handles `brand-manager assign --brand <id> --scenario <name>`. [REQ:BM-REQ-CLI-CRUD]
func (a *App) cmdAssign(args []string) error {
	fs := flag.NewFlagSet("assign", flag.ContinueOnError)
	brandID := fs.String("brand", "", "Brand ID to assign (required)")
	scenario := fs.String("scenario", "", "Scenario name to assign to (required)")
	elements := fs.String("elements", "", "Comma-separated elements to apply (e.g. colors,typography)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if *brandID == "" || *scenario == "" {
		return fmt.Errorf("--brand and --scenario are required\nUsage: brand-manager assign --brand ID --scenario NAME [--elements colors,typography] [--json]")
	}

	payload := map[string]interface{}{
		"brand_id":      *brandID,
		"scenario_name": *scenario,
	}
	if *elements != "" {
		payload["elements"] = cliutil.ParseCSV(*elements)
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/assignments"), nil, payload)
	if err != nil {
		return err
	}

	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err == nil {
		fmt.Printf("Assigned brand %s to scenario %s (assignment: %s)\n", *brandID, *scenario, result["id"])
	} else {
		cliutil.PrintJSON(body)
	}
	return nil
}

// cmdUnassign handles `brand-manager unassign <assignment-id>`. [REQ:BM-REQ-CLI-CRUD]
func (a *App) cmdUnassign(args []string) error {
	fs := flag.NewFlagSet("unassign", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() == 0 {
		return fmt.Errorf("usage: brand-manager unassign <assignment-id> [--json]")
	}
	id := fs.Arg(0)

	_, err := a.core.APIClient.Request("DELETE", a.apiPath("/assignments/"+id), nil, nil)
	if err != nil {
		return err
	}

	if *jsonOut {
		fmt.Println(`{"success": true, "deleted": "` + id + `"}`)
		return nil
	}

	fmt.Printf("Removed assignment: %s\n", id)
	return nil
}

// cmdScenarioStatus handles `brand-manager scenario-status <name>`. [REQ:BM-REQ-CLI-CRUD]
func (a *App) cmdScenarioStatus(args []string) error {
	fs := flag.NewFlagSet("scenario-status", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() == 0 {
		return fmt.Errorf("usage: brand-manager scenario-status <scenario-name> [--json]")
	}
	name := fs.Arg(0)

	body, err := a.core.APIClient.Get(a.apiPath("/scenarios/"+name+"/status"), nil)
	if err != nil {
		return err
	}

	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}

	var status map[string]interface{}
	if err := json.Unmarshal(body, &status); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Scenario: %s\n", status["scenario"])
	if hasBrand, ok := status["has_brand"].(bool); ok && hasBrand {
		fmt.Printf("Brand: %s (v%v)\n", status["brand_id"], status["brand_version"])
		if elems, ok := status["elements"].([]interface{}); ok && len(elems) > 0 {
			strs := make([]string, len(elems))
			for i, e := range elems {
				strs[i] = fmt.Sprintf("%v", e)
			}
			fmt.Printf("Elements: %s\n", strings.Join(strs, ", "))
		}
		if applied, ok := status["applied_at"].(string); ok {
			fmt.Printf("Applied: %s\n", applied)
		}
	} else {
		fmt.Println("Brand: (none assigned)")
		fmt.Println("\nNext: brand-manager assign --brand <id> --scenario " + name)
	}
	return nil
}

// --- Operations ---

// cmdDiscover handles `brand-manager discover <scenario>`. [REQ:BM-REQ-CLI-DISCOVER]
func (a *App) cmdDiscover(args []string) error {
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

	if *doImport {
		body, err := a.core.APIClient.Request("POST", a.apiPath("/discover/"+scenario+"/import"), nil, nil)
		if err != nil {
			return err
		}
		if *jsonOut {
			cliutil.PrintJSON(body)
			return nil
		}
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err == nil {
			if brand, ok := result["brand"].(map[string]interface{}); ok {
				fmt.Printf("Imported brand: %s (id: %s)\n", brand["name"], brand["id"])
			}
			if conf, ok := result["confidence"].(float64); ok {
				fmt.Printf("Confidence: %.0f%%\n", conf*100)
			}
		} else {
			cliutil.PrintJSON(body)
		}
		return nil
	}

	body, err := a.core.APIClient.Get(a.apiPath("/discover/"+scenario), nil)
	if err != nil {
		return err
	}

	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Scenario: %s\n", result["scenario"])
	if conf, ok := result["confidence"].(float64); ok {
		fmt.Printf("Confidence: %.0f%%\n", conf*100)
	}

	if sources, ok := result["sources"].([]interface{}); ok && len(sources) > 0 {
		fmt.Println("\nSources:")
		for _, s := range sources {
			src := s.(map[string]interface{})
			fmt.Printf("  %s (%s) — %v field(s), %.0f%% confidence\n",
				src["file"], src["type"], src["fields"], src["confidence"].(float64)*100)
		}
	} else {
		fmt.Println("\nNo branding state found.")
	}

	if suggestions, ok := result["suggestions"].([]interface{}); ok && len(suggestions) > 0 {
		fmt.Println("\nSuggestions:")
		for _, s := range suggestions {
			fmt.Printf("  • %s\n", s)
		}
	}

	if result["draft_brand"] != nil {
		fmt.Println("\nUse --import to create a brand from this discovery.")
	}

	return nil
}

// cmdApply handles `brand-manager apply <brand-id> --scenario <name>`. [REQ:BM-REQ-CLI-APPLY]
func (a *App) cmdApply(args []string) error {
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

	payload := map[string]interface{}{
		"scenario_name": *scenario,
	}
	if *elements != "" {
		payload["elements"] = cliutil.ParseCSV(*elements)
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/brands/"+brandID+"/apply"), nil, payload)
	if err != nil {
		return err
	}

	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Applied brand %s to %s\n", brandID, *scenario)

	if applied, ok := result["applied"].([]interface{}); ok {
		fmt.Printf("Actions: %d\n", len(applied))
		for _, a := range applied {
			action := a.(map[string]interface{})
			fmt.Printf("  ✓ %s → %s (%s)\n", action["element"], action["file"], action["type"])
		}
	}

	if skipped, ok := result["skipped"].([]interface{}); ok && len(skipped) > 0 {
		fmt.Println("Skipped:")
		for _, s := range skipped {
			skip := s.(map[string]interface{})
			fmt.Printf("  ✗ %s: %s\n", skip["element"], skip["reason"])
		}
	}

	return nil
}

// cmdGenerate handles `brand-manager generate <brand-id> --elements colors,typography`. [REQ:BM-REQ-CLI-GEN]
func (a *App) cmdGenerate(args []string) error {
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

	// Image generation mode
	if *imageType != "" {
		payload := map[string]string{"type": *imageType}
		if *model != "" {
			payload["model"] = *model
		}

		body, err := a.core.APIClient.Request("POST", a.apiPath("/brands/"+brandID+"/generate/image"), nil, payload)
		if err != nil {
			return err
		}

		if *jsonOut {
			cliutil.PrintJSON(body)
			return nil
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err == nil {
			fmt.Printf("Generated %s for brand %s\n", result["type"], brandID)
			fmt.Printf("Asset ID: %s\n", result["asset_id"])
			fmt.Printf("Provider: %s (%s)\n", result["provider"], result["model"])
		} else {
			cliutil.PrintJSON(body)
		}
		return nil
	}

	// Text generation mode
	payload := map[string]interface{}{
		"elements": cliutil.ParseCSV(*elements),
	}
	if *model != "" {
		payload["model"] = *model
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/brands/"+brandID+"/generate"), nil, payload)
	if err != nil {
		return err
	}

	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err == nil {
		fmt.Printf("Generated elements for brand %s\n", brandID)
		if applied, ok := result["applied"].([]interface{}); ok && len(applied) > 0 {
			strs := make([]string, len(applied))
			for i, e := range applied {
				strs[i] = fmt.Sprintf("%v", e)
			}
			fmt.Printf("Applied: %s\n", strings.Join(strs, ", "))
		}
		if p, ok := result["provider"].(string); ok && p != "" {
			fmt.Printf("Provider: %s (%s)\n", p, result["model"])
		}
	} else {
		cliutil.PrintJSON(body)
	}
	return nil
}

// cmdScan handles `brand-manager scan <scenario>`. [REQ:BM-REQ-CLI-STATUS]
func (a *App) cmdScan(args []string) error {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() == 0 {
		return fmt.Errorf("usage: brand-manager scan <scenario> [--json]")
	}
	scenario := fs.Arg(0)

	body, err := a.core.APIClient.Get(a.apiPath("/scan/"+scenario), nil)
	if err != nil {
		return err
	}

	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Scenario: %s\n", result["scenario"])
	fmt.Printf("CSS markers: %v\n", result["css_markers"])
	fmt.Printf("JSON keys: %v\n", result["json_keys"])
	fmt.Printf("Total: %v\n", result["total"])

	if results, ok := result["results"].([]interface{}); ok && len(results) > 0 {
		fmt.Println("\nMarkers:")
		for _, r := range results {
			m := r.(map[string]interface{})
			fmt.Printf("  %s:%v [%s] %s\n", m["file"], m["line"], m["type"], m["marker"])
		}
	}

	return nil
}
