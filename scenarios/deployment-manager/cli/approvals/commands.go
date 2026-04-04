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

	fmt.Printf("Approval: %s\n", str(a["id"]))
	fmt.Printf("  Profile:    %s\n", str(a["profile_id"]))
	fmt.Printf("  Platform:   %s\n", str(a["platform"]))
	fmt.Printf("  Status:     %s\n", str(a["status"]))
	fmt.Printf("  Commit:     %s\n", str(a["git_commit_hash"]))
	if v := str(a["approved_by"]); v != "" {
		fmt.Printf("  Reviewer:   %s\n", v)
	}
	if v := str(a["notes"]); v != "" {
		fmt.Printf("  Notes:      %s\n", v)
	}
	if v := str(a["validation_id"]); v != "" {
		fmt.Printf("  Validation: %s\n", v)
	}
	fmt.Printf("  Created:    %s\n", str(a["created_at"]))
	fmt.Printf("  Updated:    %s\n", str(a["updated_at"]))
	return nil
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
	fmt.Printf("Approval created: %s (status: %s)\n", str(a["id"]), str(a["status"]))
	return nil
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
	fmt.Printf("Approval %s: %s by %s\n", str(a["id"]), str(a["status"]), str(a["approved_by"]))
	return nil
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

	// Operational output: Status -> Triage -> Next Steps
	var gate map[string]interface{}
	if err := json.Unmarshal(body, &gate); err != nil {
		fmt.Println(string(body))
		return nil
	}

	ready, _ := gate["ready"].(bool)

	// Status
	if ready {
		fmt.Println("Status: READY - All required platforms approved")
	} else {
		fmt.Println("Status: BLOCKED - Not all required platforms approved")
	}

	// Triage: per-platform breakdown
	if platforms, ok := gate["platforms"].([]interface{}); ok && len(platforms) > 0 {
		fmt.Println("\nPlatform Breakdown:")
		for _, p := range platforms {
			pm, ok := p.(map[string]interface{})
			if !ok {
				continue
			}
			required := ""
			if req, ok := pm["required"].(bool); ok && req {
				required = " (required)"
			}
			fmt.Printf("  %-12s %s%s\n", str(pm["platform"]), str(pm["status"]), required)
		}
	}

	// Next Steps
	if !ready {
		fmt.Println("\nNext Steps:")
		fmt.Println("  1. Create approvals for missing platforms: approvals create <profile-id> --commit <hash> --platform <plat>")
		fmt.Println("  2. Approve pending items: approvals decide <id> --decision approved --reviewer <name>")
	}

	return nil
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

	fmt.Printf("Required platforms set: %s\n", strings.Join(platList, ", "))
	return nil
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
	if !ok || len(platforms) == 0 {
		fmt.Println("No required platforms configured.")
		return nil
	}

	fmt.Println("Required platforms:")
	for _, p := range platforms {
		fmt.Printf("  - %s\n", str(p))
	}
	return nil
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
