package approvals

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"deployment-manager/cli/cmdutil"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Commands provides CLI commands for deployment approval gating.
type Commands struct {
	api *cliutil.APIClient
}

// New creates a new approvals command set.
func New(api *cliutil.APIClient) *Commands {
	return &Commands{api: api}
}

// Run dispatches approval subcommands.
func (c *Commands) Run(args []string) error {
	if len(args) == 0 {
		return errors.New("subcommand required: list, get, create, decide, gate, platforms")
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		return c.list(rest)
	case "get":
		return c.get(rest)
	case "create":
		return c.create(rest)
	case "decide":
		return c.decide(rest)
	case "gate":
		return c.gate(rest)
	case "platforms":
		return c.platforms(rest)
	default:
		return fmt.Errorf("unknown approvals subcommand: %s", sub)
	}
}

func (c *Commands) list(args []string) error {
	fs := flag.NewFlagSet("approvals list", flag.ContinueOnError)
	commit := fs.String("commit", "", "Filter by git commit hash")
	format := fs.String("format", "", "Output format (json)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: approvals list <profile-id> [--commit <hash>]\n\n")
		fs.PrintDefaults()
	}
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("profile ID is required")
	}
	profileID := remaining[0]

	q := url.Values{}
	if *commit != "" {
		q.Set("commit", *commit)
	}

	body, err := c.api.Get("/api/v1/profiles/"+profileID+"/approvals", q)
	if err != nil {
		return err
	}

	formatVal := cmdutil.ResolveFormat(*format)
	if strings.ToLower(formatVal) == "json" {
		cliutil.PrintJSON(body)
		return nil
	}

	// Human-friendly table output
	var approvals []map[string]interface{}
	if err := json.Unmarshal(body, &approvals); err != nil {
		fmt.Println(string(body))
		return nil
	}

	if len(approvals) == 0 {
		fmt.Println("No approvals found.")
		return nil
	}

	headers := []string{"ID", "PLATFORM", "STATUS", "COMMIT", "REVIEWER", "UPDATED"}
	rows := make([][]string, 0, len(approvals))
	for _, a := range approvals {
		rows = append(rows, []string{
			str(a["id"]),
			str(a["platform"]),
			str(a["status"]),
			truncate(str(a["git_commit_hash"]), 12),
			str(a["approved_by"]),
			str(a["updated_at"]),
		})
	}
	cmdutil.PrintTable(headers, rows)
	return nil
}

func (c *Commands) get(args []string) error {
	fs := flag.NewFlagSet("approvals get", flag.ContinueOnError)
	format := fs.String("format", "", "Output format (json)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("approval ID is required")
	}

	body, err := c.api.Get("/api/v1/approvals/"+remaining[0], nil)
	if err != nil {
		return err
	}

	formatVal := cmdutil.ResolveFormat(*format)
	if strings.ToLower(formatVal) == "json" {
		cliutil.PrintJSON(body)
		return nil
	}

	// Human-friendly detail output
	var a map[string]interface{}
	if err := json.Unmarshal(body, &a); err != nil {
		fmt.Println(string(body))
		return nil
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Approval: %s", str(a["id"])),
			fmt.Sprintf("Profile: %s", str(a["profile_id"])),
			fmt.Sprintf("Platform: %s", str(a["platform"])),
			fmt.Sprintf("Status: %s", str(a["status"])),
		},
		ResultsHeading: "Details",
		Results: []string{
			fmt.Sprintf("Commit: %s", str(a["git_commit_hash"])),
			fmt.Sprintf("Reviewer: %s", fallbackValue(str(a["approved_by"]), "(none)")),
			fmt.Sprintf("Notes: %s", fallbackValue(str(a["notes"]), "(none)")),
			fmt.Sprintf("Validation: %s", fallbackValue(str(a["validation_id"]), "(none)")),
			fmt.Sprintf("Created: %s", str(a["created_at"])),
			fmt.Sprintf("Updated: %s", str(a["updated_at"])),
		},
		RetrievalHints: []string{
			fmt.Sprintf("deployment-manager approvals list %s", str(a["profile_id"])),
		},
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func (c *Commands) create(args []string) error {
	fs := flag.NewFlagSet("approvals create", flag.ContinueOnError)
	commit := fs.String("commit", "", "Git commit hash (required)")
	platform := fs.String("platform", "", "Target platform (required)")
	validationID := fs.String("validation-id", "", "Associated validation ID")
	format := fs.String("format", "", "Output format (json)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: approvals create <profile-id> --commit <hash> --platform <platform>\n\n")
		fs.PrintDefaults()
	}
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("profile ID is required")
	}
	if *commit == "" {
		return errors.New("--commit is required")
	}
	if *platform == "" {
		return errors.New("--platform is required")
	}

	payload := map[string]interface{}{
		"git_commit_hash": *commit,
		"platform":        *platform,
	}
	if *validationID != "" {
		payload["validation_id"] = *validationID
	}

	body, err := c.api.Request("POST", "/api/v1/profiles/"+remaining[0]+"/approvals", nil, payload)
	if err != nil {
		return err
	}

	formatVal := cmdutil.ResolveFormat(*format)
	if strings.ToLower(formatVal) == "json" {
		cliutil.PrintJSON(body)
		return nil
	}

	var a map[string]interface{}
	if err := json.Unmarshal(body, &a); err != nil {
		fmt.Println(string(body))
		return nil
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Approval created: %s", str(a["id"])),
		},
		Changes: []string{
			fmt.Sprintf("Profile: %s", remaining[0]),
			fmt.Sprintf("Platform: %s", *platform),
			fmt.Sprintf("Status: %s", str(a["status"])),
			fmt.Sprintf("Commit: %s", *commit),
		},
		NextCommand: []string{
			fmt.Sprintf("deployment-manager approvals get %s", str(a["id"])),
			fmt.Sprintf("deployment-manager approvals decide %s --decision approved --reviewer <name>", str(a["id"])),
		},
	})
}

func (c *Commands) decide(args []string) error {
	fs := flag.NewFlagSet("approvals decide", flag.ContinueOnError)
	decision := fs.String("decision", "", "Decision: approved or rejected (required)")
	reviewer := fs.String("reviewer", "", "Reviewer name (required)")
	notes := fs.String("notes", "", "Decision notes")
	format := fs.String("format", "", "Output format (json)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: approvals decide <approval-id> --decision <approved|rejected> --reviewer <name>\n\n")
		fs.PrintDefaults()
	}
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("approval ID is required")
	}
	if *decision != "approved" && *decision != "rejected" {
		return errors.New("--decision must be 'approved' or 'rejected'")
	}
	if *reviewer == "" {
		return errors.New("--reviewer is required")
	}

	payload := map[string]interface{}{
		"decision": *decision,
		"reviewer": *reviewer,
	}
	if *notes != "" {
		payload["notes"] = *notes
	}

	body, err := c.api.Request("POST", "/api/v1/approvals/"+remaining[0]+"/decide", nil, payload)
	if err != nil {
		return err
	}

	formatVal := cmdutil.ResolveFormat(*format)
	if strings.ToLower(formatVal) == "json" {
		cliutil.PrintJSON(body)
		return nil
	}

	var a map[string]interface{}
	if err := json.Unmarshal(body, &a); err != nil {
		fmt.Println(string(body))
		return nil
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Approval %s is now %s", str(a["id"]), str(a["status"])),
		},
		Changes: []string{
			fmt.Sprintf("Reviewer: %s", str(a["approved_by"])),
			fmt.Sprintf("Decision: %s", *decision),
			fmt.Sprintf("Notes: %s", fallbackValue(*notes, "(none)")),
		},
		NextCommand: []string{
			fmt.Sprintf("deployment-manager approvals get %s", str(a["id"])),
		},
	})
}

func (c *Commands) gate(args []string) error {
	fs := flag.NewFlagSet("approvals gate", flag.ContinueOnError)
	commit := fs.String("commit", "", "Git commit hash (required)")
	format := fs.String("format", "", "Output format (json)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: approvals gate <profile-id> --commit <hash>\n\n")
		fs.PrintDefaults()
	}
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("profile ID is required")
	}
	if *commit == "" {
		return errors.New("--commit is required")
	}

	q := url.Values{}
	q.Set("commit", *commit)

	body, err := c.api.Get("/api/v1/profiles/"+remaining[0]+"/release-gate", q)
	if err != nil {
		return err
	}

	formatVal := cmdutil.ResolveFormat(*format)
	if strings.ToLower(formatVal) == "json" {
		cliutil.PrintJSON(body)
		return nil
	}

	var gate map[string]interface{}
	if err := json.Unmarshal(body, &gate); err != nil {
		fmt.Println(string(body))
		return nil
	}

	ready, _ := gate["ready"].(bool)
	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Profile: %s", remaining[0]),
			fmt.Sprintf("Commit: %s", *commit),
		},
	}
	if ready {
		report.Status = append([]string{"Release gate: READY"}, report.Status...)
	} else {
		report.Status = append([]string{"Release gate: BLOCKED"}, report.Status...)
	}

	if platforms, ok := gate["platforms"].([]interface{}); ok && len(platforms) > 0 {
		group := cliapp.TriageGroup{Heading: "Platform Breakdown"}
		for _, p := range platforms {
			pm, ok := p.(map[string]interface{})
			if !ok {
				continue
			}
			required := ""
			if req, ok := pm["required"].(bool); ok && req {
				required = " (required)"
			}
			group.Items = append(group.Items, fmt.Sprintf("%s %s%s", str(pm["platform"]), str(pm["status"]), required))
		}
		report.Triage = append(report.Triage, group)
	}

	if !ready {
		report.NextSteps = []string{
			"deployment-manager approvals create <profile-id> --commit <hash> --platform <platform>",
			"deployment-manager approvals decide <approval-id> --decision approved --reviewer <name>",
		}
	} else {
		report.NextSteps = []string{
			fmt.Sprintf("deployment-manager deploy %s", remaining[0]),
		}
	}

	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func (c *Commands) platforms(args []string) error {
	if len(args) == 0 {
		return errors.New("platforms subcommand required: set, get")
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "set":
		return c.platformsSet(rest)
	case "get":
		return c.platformsGet(rest)
	default:
		return fmt.Errorf("unknown platforms subcommand: %s", sub)
	}
}

func (c *Commands) platformsSet(args []string) error {
	fs := flag.NewFlagSet("approvals platforms set", flag.ContinueOnError)
	platforms := fs.String("platforms", "", "Comma-separated list of platforms (required)")
	format := fs.String("format", "", "Output format (json)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: approvals platforms set <profile-id> --platforms win,mac,linux\n\n")
		fs.PrintDefaults()
	}
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("profile ID is required")
	}
	if *platforms == "" {
		return errors.New("--platforms is required")
	}

	platList := strings.Split(*platforms, ",")
	for i := range platList {
		platList[i] = strings.TrimSpace(platList[i])
	}

	payload := map[string]interface{}{
		"platforms": platList,
	}

	body, err := c.api.Request("PUT", "/api/v1/profiles/"+remaining[0]+"/required-platforms", nil, payload)
	if err != nil {
		return err
	}

	formatVal := cmdutil.ResolveFormat(*format)
	if strings.ToLower(formatVal) == "json" {
		cliutil.PrintJSON(body)
		return nil
	}

	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Required platforms updated for %s", remaining[0]),
		},
		Changes: []string{
			fmt.Sprintf("Platforms: %s", strings.Join(platList, ", ")),
		},
		NextCommand: []string{
			fmt.Sprintf("deployment-manager approvals platforms get %s", remaining[0]),
		},
	})
}

func (c *Commands) platformsGet(args []string) error {
	fs := flag.NewFlagSet("approvals platforms get", flag.ContinueOnError)
	format := fs.String("format", "", "Output format (json)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("profile ID is required")
	}

	body, err := c.api.Get("/api/v1/profiles/"+remaining[0]+"/required-platforms", nil)
	if err != nil {
		return err
	}

	formatVal := cmdutil.ResolveFormat(*format)
	if strings.ToLower(formatVal) == "json" {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		fmt.Println(string(body))
		return nil
	}

	platforms, ok := resp["platforms"].([]interface{})
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Profile: %s", remaining[0]),
		},
		ResultsHeading: "Required Platforms",
		RetrievalHints: []string{
			fmt.Sprintf("deployment-manager approvals platforms set %s --platforms linux,macos,windows", remaining[0]),
		},
	}
	if !ok || len(platforms) == 0 {
		report.Results = []string{"(none configured)"}
		return cliapp.RenderListReport(os.Stdout, report)
	}
	for _, p := range platforms {
		report.Results = append(report.Results, str(p))
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

// str safely converts an interface{} to string.
func str(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// truncate shortens a string to max length, adding "..." if truncated.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func fallbackValue(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
