// Package releases provides CLI commands for the deployment-manager release
// lifecycle (start a release, list, get, re-run verification). Mirrors the
// approvals CLI patterns: per-subcommand flag sets, --format json support,
// human-friendly default rendering via cliapp report helpers.
package releases

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

// Commands provides CLI commands for the release lifecycle.
type Commands struct {
	api *cliutil.APIClient
}

// New creates a new releases command set bound to the API client.
func New(api *cliutil.APIClient) *Commands {
	return &Commands{api: api}
}

// Run dispatches release subcommands.
func (c *Commands) Run(args []string) error {
	if len(args) == 0 {
		return errors.New("subcommand required: list, get, start, verify")
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		return c.list(rest)
	case "get":
		return c.get(rest)
	case "start":
		return c.start(rest)
	case "verify":
		return c.verify(rest)
	default:
		return fmt.Errorf("unknown releases subcommand: %s", sub)
	}
}

func (c *Commands) list(args []string) error {
	fs := flag.NewFlagSet("releases list", flag.ContinueOnError)
	limit := fs.Int("limit", 0, "Maximum number of releases to return (default 50)")
	format := fs.String("format", "", "Output format (json)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: releases list <profile-id> [--limit <n>]\n\n")
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
	if *limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", *limit))
	}

	body, err := c.api.Get("/api/v1/profiles/"+profileID+"/releases", q)
	if err != nil {
		return err
	}

	formatVal := cmdutil.ResolveFormat(*format)
	if strings.ToLower(formatVal) == "json" {
		cliutil.PrintJSON(body)
		return nil
	}

	var envelope struct {
		Releases []map[string]interface{} `json:"releases"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		fmt.Println(string(body))
		return nil
	}
	if len(envelope.Releases) == 0 {
		fmt.Println("No releases found.")
		return nil
	}
	headers := []string{"ID", "CHANNEL", "VERSION", "STATUS", "COMMIT", "CREATED"}
	rows := make([][]string, 0, len(envelope.Releases))
	for _, r := range envelope.Releases {
		rows = append(rows, []string{
			truncate(str(r["id"]), 12),
			str(r["channel"]),
			str(r["release_version"]),
			str(r["status"]),
			truncate(str(r["git_commit_hash"]), 12),
			str(r["created_at"]),
		})
	}
	cmdutil.PrintTable(headers, rows)
	return nil
}

func (c *Commands) get(args []string) error {
	fs := flag.NewFlagSet("releases get", flag.ContinueOnError)
	format := fs.String("format", "", "Output format (json)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: releases get <release-id>\n\n")
		fs.PrintDefaults()
	}
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("release ID is required")
	}

	body, err := c.api.Get("/api/v1/releases/"+remaining[0], nil)
	if err != nil {
		return err
	}
	formatVal := cmdutil.ResolveFormat(*format)
	if strings.ToLower(formatVal) == "json" {
		cliutil.PrintJSON(body)
		return nil
	}
	return renderReleaseDetail(body)
}

func (c *Commands) start(args []string) error {
	fs := flag.NewFlagSet("releases start", flag.ContinueOnError)
	channel := fs.String("channel", "", "Release channel (defaults to profile default; e.g. stable, beta)")
	commit := fs.String("commit", "", "Git commit hash for the release (required)")
	version := fs.String("version", "", "Release version string (required)")
	notes := fs.String("notes", "", "Release notes")
	releasedBy := fs.String("released-by", "", "Operator initiating the release")
	platformsCSV := fs.String("platforms", "", "Comma-separated target platforms (default: linux-x64,darwin-arm64,win-x64)")
	format := fs.String("format", "", "Output format (json)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: releases start <profile-id> --commit <hash> --version <version> [--channel <name>]\n\n")
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
	if *version == "" {
		return errors.New("--version is required")
	}

	payload := map[string]interface{}{
		"git_commit_hash": *commit,
		"release_version": *version,
	}
	if *channel != "" {
		payload["channel"] = *channel
	}
	if *notes != "" {
		payload["release_notes"] = *notes
	}
	if *releasedBy != "" {
		payload["released_by"] = *releasedBy
	}
	if *platformsCSV != "" {
		platList := strings.Split(*platformsCSV, ",")
		for i := range platList {
			platList[i] = strings.TrimSpace(platList[i])
		}
		payload["platforms"] = platList
	}

	body, err := c.api.Request("POST", "/api/v1/profiles/"+remaining[0]+"/releases/start", nil, payload)
	if err != nil {
		return err
	}
	formatVal := cmdutil.ResolveFormat(*format)
	if strings.ToLower(formatVal) == "json" {
		cliutil.PrintJSON(body)
		return nil
	}
	return renderStartResult(body)
}

func (c *Commands) verify(args []string) error {
	fs := flag.NewFlagSet("releases verify", flag.ContinueOnError)
	deep := fs.Bool("deep", false, "Run a deep verification (S3 reachability check)")
	format := fs.String("format", "", "Output format (json)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: releases verify <release-id> [--deep]\n\n")
		fs.PrintDefaults()
	}
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("release ID is required")
	}
	q := url.Values{}
	if *deep {
		q.Set("deep", "true")
	}
	body, err := c.api.Request("POST", "/api/v1/releases/"+remaining[0]+"/verify", q, nil)
	if err != nil {
		return err
	}
	formatVal := cmdutil.ResolveFormat(*format)
	if strings.ToLower(formatVal) == "json" {
		cliutil.PrintJSON(body)
		return nil
	}
	return renderReleaseDetail(body)
}

// renderReleaseDetail prints a release record (or {release: ...} envelope)
// as a human-readable report.
func renderReleaseDetail(body []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		fmt.Println(string(body))
		return nil
	}
	rel := raw
	if inner, ok := raw["release"].(map[string]interface{}); ok {
		rel = inner
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Release: %s", str(rel["id"])),
			fmt.Sprintf("Profile: %s", str(rel["profile_id"])),
			fmt.Sprintf("Channel: %s", str(rel["channel"])),
			fmt.Sprintf("Status: %s", str(rel["status"])),
			fmt.Sprintf("Version: %s", str(rel["release_version"])),
		},
		ResultsHeading: "Per-Platform State",
	}
	if platforms, ok := rel["platforms"].([]interface{}); ok && len(platforms) > 0 {
		for _, p := range platforms {
			pm, ok := p.(map[string]interface{})
			if !ok {
				continue
			}
			line := fmt.Sprintf("%s: %s", str(pm["platform"]), str(pm["status"]))
			if errMsg := str(pm["error"]); errMsg != "" {
				line += " (" + errMsg + ")"
			}
			report.Results = append(report.Results, line)
		}
	} else {
		report.Results = []string{"(no platform rows)"}
	}
	if evidence, ok := rel["verification_evidence"].([]interface{}); ok && len(evidence) > 0 {
		report.RetrievalHints = []string{
			fmt.Sprintf("Verified %d platform(s); use --format json for full evidence", len(evidence)),
		}
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

// renderStartResult prints the {release, steps} envelope returned by start.
func renderStartResult(body []byte) error {
	var env struct {
		Release map[string]interface{}   `json:"release"`
		Steps   []map[string]interface{} `json:"steps"`
		Status  string                   `json:"status"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		fmt.Println(string(body))
		return nil
	}
	if env.Release == nil {
		fmt.Println(string(body))
		return nil
	}
	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Release %s status: %s", str(env.Release["id"]), str(env.Release["status"])),
		},
		Changes: []string{
			fmt.Sprintf("Channel: %s", str(env.Release["channel"])),
			fmt.Sprintf("Commit: %s", truncate(str(env.Release["git_commit_hash"]), 12)),
			fmt.Sprintf("Version: %s", str(env.Release["release_version"])),
		},
		NextCommand: []string{
			fmt.Sprintf("deployment-manager releases get %s", str(env.Release["id"])),
			fmt.Sprintf("deployment-manager releases verify %s", str(env.Release["id"])),
		},
	}
	if len(env.Steps) > 0 {
		report.Changes = append(report.Changes, fmt.Sprintf("Steps: %d", len(env.Steps)))
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

// str converts an arbitrary value to a string for display.
func str(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// truncate clips a string to max length with "..." appended on overflow.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
