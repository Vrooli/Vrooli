// prompt-manager graph orientation-cost — how much a reader must hold in their
// head to work inside each team.
//
// The reading is only meaningful against the previous audit record: the band is
// a trend, not a level. The output says so rather than leaving a reader to
// treat a single composite as a verdict.
//
// DOC: docs/agent-system/FRAMEWORK_HEALTH.md § Team orientation cost
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

type orientationComponents struct {
	Members int `json:"members"`
	// CanonLines is charged by consumer share: a plan-of-record document
	// declared for N consuming teams contributes lines/N.
	CanonLines int `json:"canonLines"`
	// SharedCanonLines is the full unsplit count of multi-consumer documents,
	// reported so the split cannot hide what a custodian actually carries.
	SharedCanonLines int `json:"sharedCanonLines,omitempty"`
	Topics           int `json:"topics"`
}

type orientationCost struct {
	TeamID           string                `json:"teamId"`
	Components       orientationComponents `json:"components"`
	Composite        int                   `json:"composite"`
	ScenarioCoverage int                   `json:"scenarioCoverage"`
	Scenarios        []string              `json:"scenarios,omitempty"`
	DomainAddresses  int                   `json:"domainAddresses"`
	Addresses        []string              `json:"addresses,omitempty"`
	ExternalActors   []string              `json:"externalActors,omitempty"`
	MissingCanon     []string              `json:"missingCanon,omitempty"`
}

type orientationCostReport struct {
	Teams []orientationCost `json:"teams"`
	Note  string            `json:"note"`
}

func cmdOrientationCost(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("orientation-cost", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	var resp orientationCostReport
	if err := ctx.GetWithQuery("/orientation-cost", url.Values{}, &resp); err != nil {
		return err
	}
	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	fmt.Printf("%-20s %9s %8s %6s %6s %9s %9s\n", "TEAM", "COMPOSITE", "MEMBERS", "CANON", "TOPICS", "COVERS", "ADDRESSES")
	for _, team := range resp.Teams {
		fmt.Printf("%-20s %9d %8d %6d %6d %9d %9d\n",
			team.TeamID, team.Composite, team.Components.Members,
			team.Components.CanonLines, team.Components.Topics,
			team.ScenarioCoverage, team.DomainAddresses)
	}
	for _, team := range resp.Teams {
		if team.DomainAddresses > 1 {
			fmt.Printf("\n%s names %d domain addresses: %s\n", team.TeamID, team.DomainAddresses, strings.Join(team.Addresses, ", "))
		}
	}
	for _, team := range resp.Teams {
		if len(team.MissingCanon) > 0 {
			fmt.Printf("\n%s declares canon that is not on disk: %s\n", team.TeamID, strings.Join(team.MissingCanon, ", "))
		}
	}
	if resp.Note != "" {
		fmt.Printf("\n%s\n", resp.Note)
	}
	return nil
}
