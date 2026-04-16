package connections

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "development-toolchain-validator"

type ConnectionResponse struct {
	ID               string `json:"id"`
	ReferenceID      string `json:"reference_id"`
	SkillID          string `json:"skill_id"`
	SkillVersion     string `json:"skill_version,omitempty"`
	SkillContentHash string `json:"skill_content_hash,omitempty"`
	ConnectedAt      string `json:"connected_at,omitempty"`
	UpdatedAt        string `json:"updated_at,omitempty"`
}

type ConnectionListResponse struct {
	Connections []ConnectionResponse `json:"connections"`
	Count       int                  `json:"count"`
}

type ConnectionConnectRequest struct {
	ReferenceID      string `json:"reference_id"`
	SkillID          string `json:"skill_id"`
	SkillVersion     string `json:"skill_version,omitempty"`
	SkillContentHash string `json:"skill_content_hash,omitempty"`
}

type DriftCheckRequest struct {
	CurrentVersion string `json:"current_version"`
	CurrentHash    string `json:"current_hash"`
}

type DriftStatusResponse struct {
	ConnectionID   string `json:"connection_id"`
	SkillID        string `json:"skill_id"`
	StoredVersion  string `json:"stored_version"`
	StoredHash     string `json:"stored_hash"`
	CurrentVersion string `json:"current_version"`
	CurrentHash    string `json:"current_hash"`
	HasDrifted     bool   `json:"has_drifted"`
	VersionChanged bool   `json:"version_changed"`
	ContentChanged bool   `json:"content_changed"`
}

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "connection",
		Description: "Manage prompt-manager skill connections",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, NeedsAPI: true, Description: "List skill connections", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, NeedsAPI: true, Description: "Get a connection by ID", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "connect", Aliases: []string{"add"}, NeedsAPI: true, Description: "Connect a skill to a reference", Run: func(args []string) error { return runConnect(core, args) }},
			{Name: "disconnect", Aliases: []string{"rm", "delete"}, NeedsAPI: true, Description: "Remove a connection", Run: func(args []string) error { return runDisconnect(core, args) }},
			{Name: "drift", Aliases: []string{"check"}, NeedsAPI: true, Description: "Check a connection for skill drift", Run: func(args []string) error { return runDrift(core, args) }},
		},
	}
}

func Run(core *cliapp.ScenarioApp, args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	switch args[0] {
	case "help", "-h", "--help":
		printUsage()
		return nil
	case "list", "ls":
		return runList(core, args[1:])
	case "get", "show":
		return runGet(core, args[1:])
	case "connect", "add":
		return runConnect(core, args[1:])
	case "disconnect", "rm", "delete":
		return runDisconnect(core, args[1:])
	case "drift", "check":
		return runDrift(core, args[1:])
	default:
		return fmt.Errorf("unknown subcommand: %s\n\n%s", args[0], usageText())
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("connection list", flag.ContinueOnError)
	referenceID := fs.String("reference", "", "Filter by reference ID")
	skillID := fs.String("skill", "", "Filter by skill ID")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	query := url.Values{}
	if strings.TrimSpace(*referenceID) != "" {
		query.Set("reference_id", strings.TrimSpace(*referenceID))
	}
	if strings.TrimSpace(*skillID) != "" {
		query.Set("skill_id", strings.TrimSpace(*skillID))
	}

	var resp ConnectionListResponse
	if err := getWithQuery(core, "/connections", query, &resp); err != nil {
		return fmt.Errorf("failed to list connections: %w", err)
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Connections found: %d", resp.Count),
		},
		Results:        renderListResults(resp.Connections),
		RetrievalHints: []string{cliName + " connection get <connection-id>", cliName + " connection drift <connection-id> --version <version> --hash <hash>"},
	}
	if strings.TrimSpace(*referenceID) != "" {
		report.Summary = append(report.Summary, "Reference filter: "+strings.TrimSpace(*referenceID))
	}
	if strings.TrimSpace(*skillID) != "" {
		report.Summary = append(report.Summary, "Skill filter: "+strings.TrimSpace(*skillID))
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("connection get", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: connection get <id> [--json]")
	}

	connID := fs.Arg(0)
	var conn ConnectionResponse
	if err := get(core, fmt.Sprintf("/connections/%s", connID), &conn); err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}

	report := cliapp.ListReport{
		Summary:        []string{"Connection: " + conn.ID, "Skill: " + conn.SkillID},
		ResultsHeading: "Details",
		Results:        detailLines(conn),
		RetrievalHints: []string{cliName + " connection drift " + conn.ID + " --version <version> --hash <hash>", cliName + " connection disconnect " + conn.ID},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runConnect(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("connection connect", flag.ContinueOnError)
	referenceID := fs.String("reference", "", "Reference ID (required)")
	skillID := fs.String("skill", "", "Skill ID (required)")
	version := fs.String("version", "", "Skill version")
	hash := fs.String("hash", "", "Skill content hash")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *referenceID == "" || *skillID == "" {
		return fmt.Errorf("usage: connection connect --reference R --skill S [--version V] [--hash H] [--json]\n\nBoth --reference and --skill are required")
	}

	req := ConnectionConnectRequest{
		ReferenceID:      *referenceID,
		SkillID:          *skillID,
		SkillVersion:     *version,
		SkillContentHash: *hash,
	}

	var conn ConnectionResponse
	if err := request(core, "POST", "/connections", req, &conn); err != nil {
		return fmt.Errorf("failed to connect skill: %w", err)
	}

	report := cliapp.MutationReport{
		Result: []string{"Connection created", "Connection ID: " + conn.ID},
		Changes: []string{
			"Reference ID: " + conn.ReferenceID,
			"Skill ID: " + conn.SkillID,
		},
		NextCommand: []string{cliName + " connection get " + conn.ID, cliName + " connection drift " + conn.ID + " --version <version> --hash <hash>"},
	}
	if conn.SkillVersion != "" {
		report.Changes = append(report.Changes, "Skill version: "+conn.SkillVersion)
	}
	if conn.SkillContentHash != "" {
		report.Changes = append(report.Changes, "Skill content hash: "+conn.SkillContentHash)
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDisconnect(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("connection disconnect", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: connection disconnect <id> [--json]")
	}

	connID := fs.Arg(0)
	if _, err := core.Request("DELETE", fmt.Sprintf("/connections/%s", connID), nil, nil); err != nil {
		return fmt.Errorf("failed to disconnect skill: %w", err)
	}

	report := cliapp.MutationReport{
		Result:      []string{"Connection removed", "Connection ID: " + connID},
		Changes:     []string{"Detached skill metadata from the stored reference linkage"},
		NextCommand: []string{cliName + " connection list"},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDrift(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("connection drift", flag.ContinueOnError)
	version := fs.String("version", "", "Current skill version (required)")
	hash := fs.String("hash", "", "Current skill content hash (required)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: connection drift <id> --version V --hash H [--json]")
	}
	if *version == "" || *hash == "" {
		return fmt.Errorf("both --version and --hash are required for drift check")
	}

	connID := fs.Arg(0)
	req := DriftCheckRequest{
		CurrentVersion: *version,
		CurrentHash:    *hash,
	}

	var status DriftStatusResponse
	if err := request(core, "POST", fmt.Sprintf("/connections/%s/drift", connID), req, &status); err != nil {
		return fmt.Errorf("failed to check drift: %w", err)
	}

	report := cliapp.OperationalReport{
		Status: []string{
			"Connection ID: " + status.ConnectionID,
			"Skill: " + status.SkillID,
			fmt.Sprintf("Stored version: %s", status.StoredVersion),
			fmt.Sprintf("Current version: %s", status.CurrentVersion),
		},
		NextSteps: []string{
			cliName + " connection get " + status.ConnectionID,
		},
	}
	triage := cliapp.TriageGroup{Heading: "Drift"}
	if status.HasDrifted {
		triage.Items = append(triage.Items, "Drift detected")
		if status.VersionChanged {
			triage.Items = append(triage.Items, "Version changed")
		}
		if status.ContentChanged {
			triage.Items = append(triage.Items, "Content changed")
		}
		report.NextSteps = append(report.NextSteps, cliName+" connection disconnect "+status.ConnectionID, cliName+" connection connect --reference <reference-id> --skill "+status.SkillID)
	} else {
		triage.Items = append(triage.Items, "No drift detected")
		report.NextSteps = append(report.NextSteps, cliName+" connection list --skill "+status.SkillID)
	}
	report.Triage = []cliapp.TriageGroup{triage}
	report.Status = append(report.Status,
		fmt.Sprintf("Stored hash: %s", status.StoredHash),
		fmt.Sprintf("Current hash: %s", status.CurrentHash),
	)
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func get(core *cliapp.ScenarioApp, path string, result interface{}) error {
	return getWithQuery(core, path, nil, result)
}

func getWithQuery(core *cliapp.ScenarioApp, path string, query url.Values, result interface{}) error {
	body, err := core.Get(path, query)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, result)
}

func request(core *cliapp.ScenarioApp, method, path string, payload interface{}, result interface{}) error {
	body, err := core.Request(method, path, nil, payload)
	if err != nil {
		return err
	}
	if result == nil || len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, result)
}

func renderListResults(items []ConnectionResponse) []string {
	if len(items) == 0 {
		return nil
	}
	lines := make([]string, 0, len(items))
	for _, item := range items {
		version := item.SkillVersion
		if strings.TrimSpace(version) == "" {
			version = "(no version)"
		}
		lines = append(lines, fmt.Sprintf("%s | %s | %s | %s", item.ID, item.ReferenceID, item.SkillID, version))
	}
	return lines
}

func detailLines(conn ConnectionResponse) []string {
	lines := []string{
		"Reference ID: " + conn.ReferenceID,
		"Skill ID: " + conn.SkillID,
	}
	if conn.SkillVersion != "" {
		lines = append(lines, "Version: "+conn.SkillVersion)
	}
	if conn.SkillContentHash != "" {
		lines = append(lines, "Content hash: "+conn.SkillContentHash)
	}
	if conn.ConnectedAt != "" {
		lines = append(lines, "Connected: "+conn.ConnectedAt)
	}
	if conn.UpdatedAt != "" {
		lines = append(lines, "Updated: "+conn.UpdatedAt)
	}
	return lines
}

func printUsage() {
	fmt.Println(usageText())
}

func usageText() string {
	return `Usage: development-toolchain-validator connection <subcommand> [args]

Subcommands:
  list, ls                              List all skill connections
  get, show <id>                        Get connection by ID
  connect, add --reference R --skill S  Connect a skill to a reference
  disconnect, rm <id>                   Disconnect a skill
  drift, check <id> --version V --hash H  Check if skill has drifted

Flags:
  --json        Output as structured report JSON
  --reference   Filter by reference ID (list) or set reference ID (connect)
  --skill       Filter by skill ID (list) or set skill ID (connect)
  --version     Skill version (connect/drift)
  --hash        Skill content hash (connect/drift)

Examples:
  connection list --reference ref-123
  connection get conn-abc123
  connection connect --reference ref-123 --skill api-steer --version v1.0
  connection disconnect conn-abc123
  connection drift conn-abc123 --version v2.0 --hash newHash123`
}
