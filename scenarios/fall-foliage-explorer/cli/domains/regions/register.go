package regions

import (
	"fmt"
	"os"

	"fall-foliage-explorer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `fall-foliage-explorer regions` as a flat command since
// regions is a single read-only surface (`GET /api/regions`).
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Regions",
		Commands: []cliapp.Command{
			{
				Name:        "regions",
				Description: "List foliage regions",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runList(core, args) },
			},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("regions")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/regions", nil)
	if err != nil {
		return err
	}
	var payload support.RegionsPayload
	if err := support.Decode(body, &payload); err != nil {
		return err
	}

	summary := []string{fmt.Sprintf("Regions: %d", len(payload.Regions))}
	if payload.Meta.Source != "" {
		summary = append(summary, fmt.Sprintf("Source: %s", payload.Meta.Source))
	}
	if payload.Meta.UsingFallback {
		summary = append(summary, "Using fallback dataset")
	}
	if payload.Meta.RetrievedAt != "" {
		summary = append(summary, fmt.Sprintf("Retrieved: %s", payload.Meta.RetrievedAt))
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Regions",
		Results:        regionRows(payload.Regions),
		RetrievalHints: []string{
			fmt.Sprintf("%s foliage status <region-id>", support.CLIName),
			fmt.Sprintf("%s foliage predict <region-id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func regionRows(regions []support.Region) []string {
	if len(regions) == 0 {
		return []string{"No regions available"}
	}
	rows := make([]string, 0, len(regions))
	for _, r := range regions {
		elev := "?"
		if r.ElevationMeters != nil {
			elev = fmt.Sprintf("%dm", *r.ElevationMeters)
		}
		week := "?"
		if r.TypicalPeakWeek != nil {
			week = fmt.Sprintf("w%d", *r.TypicalPeakWeek)
		}
		rows = append(rows, fmt.Sprintf("%d. %s, %s (%s) | %.4f, %.4f | elev=%s | typical peak=%s",
			r.ID, r.Name, r.State, r.Country, r.Latitude, r.Longitude, elev, week))
	}
	return rows
}
