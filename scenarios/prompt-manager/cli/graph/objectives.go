// prompt-manager graph objectives — the objective join, read in both
// directions.
//
// This is the surface that answers "if I change an objective, what breaks?".
// Before it existed the answer was "nothing, until someone runs a document read
// by hand", which is why an unserved objective could stand indefinitely without
// any sensor moving.
//
// DOC: docs/director-swarm/strategy/OBJECTIVES.md § The coverage rule
package graph

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"prompt-manager/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliutil"
)

type objectiveTeamRef struct {
	TeamID   string `json:"teamId"`
	Role     string `json:"role,omitempty"`
	Coverage string `json:"coverage,omitempty"`
}

type objectiveRow struct {
	ID             string             `json:"id"`
	Title          string             `json:"title"`
	Class          string             `json:"class"`
	ServedBy       []objectiveTeamRef `json:"servedBy,omitempty"`
	DeclaredBy     []objectiveTeamRef `json:"declaredBy,omitempty"`
	GapMarker      string             `json:"gapMarker,omitempty"`
	EvidenceSource string             `json:"evidenceSource,omitempty"`
	HasEvidence    bool               `json:"hasEvidence"`
	Served         bool               `json:"served"`
}

type objectiveFinding struct {
	Rule       string `json:"rule"`
	Severity   string `json:"severity"`
	Team       string `json:"team,omitempty"`
	NodeID     string `json:"node_id,omitempty"`
	SourcePath string `json:"source_path,omitempty"`
	Detail     string `json:"detail"`
}

type objectiveResponse struct {
	SourcePath      string         `json:"sourcePath"`
	Rows            []objectiveRow `json:"rows"`
	UnattachedTeams []string       `json:"unattachedTeams,omitempty"`
	Unserved        int            `json:"unserved"`
	Undeclared      int            `json:"undeclaredHoles"`
	Validation      struct {
		Findings []objectiveFinding `json:"findings"`
		Errors   int                `json:"errors"`
		Warnings int                `json:"warnings"`
	} `json:"validation"`
}

func cmdObjectives(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("objectives", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	var resp objectiveResponse
	if err := ctx.GetWithQuery("/objectives", url.Values{}, &resp); err != nil {
		return err
	}
	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}
	printObjectives(resp)
	return nil
}

func printObjectives(resp objectiveResponse) {
	fmt.Printf("Objective coverage (%s)\n\n", resp.SourcePath)
	for _, row := range resp.Rows {
		status := "served"
		if !row.Served {
			status = "UNSERVED"
			if row.GapMarker != "" {
				status = "unserved (" + row.GapMarker + ")"
			}
		}
		fmt.Printf("  %-4s %-12s %-28s %s\n", row.ID, row.Class, truncateObjectiveTitle(row.Title, 28), status)
		if teams := formatObjectiveTeams(row.ServedBy); teams != "" {
			fmt.Printf("       table: %s\n", teams)
		}
		if teams := formatObjectiveTeams(row.DeclaredBy); teams != "" {
			fmt.Printf("       declared: %s\n", teams)
		}
		if !row.HasEvidence {
			fmt.Printf("       evidence: none — cannot be scored\n")
		}
	}

	if len(resp.UnattachedTeams) > 0 {
		fmt.Printf("\nTeams tracing to no objective: %s\n", strings.Join(resp.UnattachedTeams, ", "))
	}

	fmt.Printf("\n%d unserved (%d without a gap marker), %d error(s), %d warning(s)\n",
		resp.Unserved, resp.Undeclared, resp.Validation.Errors, resp.Validation.Warnings)

	if len(resp.Validation.Findings) > 0 {
		fmt.Println()
		for _, f := range resp.Validation.Findings {
			where := f.Team
			if f.NodeID != "" {
				where = strings.TrimSpace(where + " " + f.NodeID)
			}
			fmt.Printf("  [%s] %s (%s): %s\n", f.Severity, f.Rule, where, f.Detail)
		}
	}

	// An unserved objective is stated intent nobody is serving. Naming the
	// actuator here keeps the reader from treating the count as informational.
	if resp.Unserved > 0 {
		fmt.Println("\nActuator for an unserved objective: outcome-direction or capability work in director-swarm.")
	}
}

func formatObjectiveTeams(refs []objectiveTeamRef) string {
	if len(refs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(refs))
	for _, ref := range refs {
		part := ref.TeamID
		var qualifiers []string
		if ref.Role != "" {
			qualifiers = append(qualifiers, ref.Role)
		}
		if ref.Coverage == "partial" {
			qualifiers = append(qualifiers, "partial")
		}
		if len(qualifiers) > 0 {
			part += " (" + strings.Join(qualifiers, ", ") + ")"
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, ", ")
}

func truncateObjectiveTitle(title string, width int) string {
	runes := []rune(title)
	if len(runes) <= width {
		return title
	}
	return string(runes[:width-1]) + "…"
}
