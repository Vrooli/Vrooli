// prompt-manager graph instruments — does each team declare the one scenario it
// reads for the state of its domain, or an honest, dated hole where that
// scenario is not yet built?
//
// The deadband is "declared or dated", never "present". A team with no
// instrument is in band as long as it says so with a date; a team that says
// nothing is not, because silence cannot be aged.
//
// DOC: docs/agent-system/TARGET_MODEL.md § The instrument: six invariants
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

type teamInstrument struct {
	Status          string   `json:"status"`
	Scenario        string   `json:"scenario,omitempty"`
	Archetype       string   `json:"archetype,omitempty"`
	CoversScenarios []string `json:"coversScenarios,omitempty"`
	GapMarker       string   `json:"gapMarker,omitempty"`
}

type instrumentReading struct {
	TeamID      string          `json:"teamId"`
	Declared    bool            `json:"declared"`
	Instrument  *teamInstrument `json:"instrument,omitempty"`
	Findings    []string        `json:"findings,omitempty"`
	GapOpenedOn string          `json:"gapOpenedOn,omitempty"`
}

type instrumentCoverageReport struct {
	Teams      []instrumentReading `json:"teams"`
	Live       int                 `json:"live"`
	Partial    int                 `json:"partial"`
	None       int                 `json:"none"`
	Undeclared int                 `json:"undeclared"`
	OutOfBand  int                 `json:"outOfBand"`
	Note       string              `json:"note"`
}

func cmdInstruments(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("instruments", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	var resp instrumentCoverageReport
	if err := ctx.GetWithQuery("/instruments", url.Values{}, &resp); err != nil {
		return err
	}
	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	fmt.Printf("%-20s %-10s %-24s %-18s %s\n", "TEAM", "STATUS", "SCENARIO", "ARCHETYPE", "GAP OPENED")
	for _, team := range resp.Teams {
		status, scenario, archetype, gap := "undeclared", "-", "-", "-"
		if team.Instrument != nil {
			if team.Instrument.Status != "" {
				status = team.Instrument.Status
			}
			if team.Instrument.Scenario != "" {
				scenario = team.Instrument.Scenario
			}
			if team.Instrument.Archetype != "" {
				archetype = team.Instrument.Archetype
			}
		}
		if team.GapOpenedOn != "" {
			gap = team.GapOpenedOn
		}
		fmt.Printf("%-20s %-10s %-24s %-18s %s\n", team.TeamID, status, scenario, archetype, gap)
	}

	for _, team := range resp.Teams {
		for _, finding := range team.Findings {
			fmt.Printf("\n%s: %s\n", team.TeamID, finding)
		}
	}

	fmt.Printf("\n%d live, %d partial, %d none, %d undeclared — %d out of band\n",
		resp.Live, resp.Partial, resp.None, resp.Undeclared, resp.OutOfBand)
	if resp.Note != "" {
		fmt.Printf("\n%s\n", strings.TrimSpace(resp.Note))
	}
	return nil
}
