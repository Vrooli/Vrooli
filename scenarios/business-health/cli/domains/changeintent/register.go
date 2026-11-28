package changeintent

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register adds a read-only owner command for bounded one-scenario intent
// traceability. It is intentionally separate from semantic SearchService hits.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{Title: "Change intent", Commands: []cliapp.Command{{
		Name: "change-intent", Description: "Relate one explicit change statement to checked-in requirements and source references",
		Usage: "business-health change-intent --scenario business-health --intent '...'", NeedsAPI: true,
		Run: func(args []string) error {
			fs := flag.NewFlagSet("change-intent", flag.ContinueOnError)
			scenario := fs.String("scenario", "", "Explicit scenario slug (currently business-health)")
			intent := fs.String("intent", "", "One change-intent statement (max 4096 bytes)")
			limit := fs.Int("limit", 10, "Maximum relation candidates (1..20)")
			jsonOutput := cliutil.JSONFlag(fs)
			if err := fs.Parse(args); err != nil {
				return err
			}
			if strings.TrimSpace(*scenario) != "business-health" || strings.TrimSpace(*intent) == "" {
				return fmt.Errorf("usage: business-health change-intent --scenario business-health --intent <statement> [--limit N] [--json]")
			}
			body, err := core.Request("POST", "/api/v1/change-intent/assess", nil, map[string]any{"scenario": *scenario, "intent": *intent, "limit": *limit})
			if err != nil {
				return err
			}
			if *jsonOutput {
				var data any
				if err := json.Unmarshal(body, &data); err != nil {
					return err
				}
				out, err := json.MarshalIndent(data, "", "  ")
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(os.Stdout, string(out))
				return err
			}
			var report struct {
				Scenario string `json:"scenario"`
				Results  []struct {
					Scenario      string `json:"scenario"`
					RequirementID string `json:"requirement_id"`
					Title         string `json:"title"`
					Relation      string `json:"relation"`
					RelationBasis string `json:"relation_basis"`
					Requirement   struct {
						Path   string `json:"path"`
						Line   int    `json:"line"`
						SHA256 string `json:"sha256"`
					} `json:"requirement_source"`
					Anchors []struct {
						Path        string `json:"path"`
						Line        int    `json:"line"`
						SHA256      string `json:"sha256"`
						Kind        string `json:"kind"`
						Attestation string `json:"attestation"`
					} `json:"implementation_anchors"`
				} `json:"results"`
				Limitations  []string `json:"limitations"`
				Degradations []string `json:"degradations"`
			}
			if err := json.Unmarshal(body, &report); err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "Assessment scenario: %s\n", report.Scenario)
			for _, result := range report.Results {
				fmt.Fprintf(os.Stdout, "%s · %s · %s · %s (%s)\n", result.Scenario, result.RequirementID, result.Relation, result.Title, result.RelationBasis)
				fmt.Fprintf(os.Stdout, "  declaration %s:%d · SHA-256 %s\n", result.Requirement.Path, result.Requirement.Line, result.Requirement.SHA256)
				for _, anchor := range result.Anchors {
					fmt.Fprintf(os.Stdout, "  %s:%d · %s · %s · SHA-256 %s\n", anchor.Path, anchor.Line, anchor.Kind, anchor.Attestation, anchor.SHA256)
				}
			}
			for _, limitation := range report.Limitations {
				fmt.Fprintf(os.Stdout, "Limit: %s\n", limitation)
			}
			for _, degradation := range report.Degradations {
				fmt.Fprintf(os.Stdout, "Partial scan: %s\n", degradation)
			}
			return nil
		},
	}}}
}
