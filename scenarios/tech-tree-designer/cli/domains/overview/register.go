package overview

import (
	"fmt"
	"os"

	"tech-tree-designer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Overview",
		Commands: []cliapp.Command{
			deps.Core.StandardStatusCommand(cliapp.StatusCommandOptions{
				Name:        "health",
				Description: "Check API health and runtime dependencies",
			}),
			{
				Name:        "overview",
				NeedsAPI:    true,
				Description: "Show strategic tech tree progress, milestones, and dependencies",
				Run: func(args []string) error {
					return runOverview(deps, args)
				},
			},
			{
				Name:        "analyze",
				NeedsAPI:    true,
				Description: "Run strategic analysis for the selected tech tree",
				Run: func(args []string) error {
					return runAnalyze(deps, args)
				},
			},
			{
				Name:        "recommend",
				NeedsAPI:    true,
				Description: "Get prioritized scenario recommendations",
				Run: func(args []string) error {
					return runRecommend(deps, args)
				},
			},
			{
				Name:        "visualize",
				Description: "Open the tech tree UI in a browser",
				Run: func(args []string) error {
					return runVisualize(deps, args)
				},
			},
		},
	}
}

type overviewReport struct {
	cliapp.OperationalReport `json:",inline"`
	Scope                    string `json:"scope"`
	SectorCount              int    `json:"sector_count"`
	AverageProgress          int    `json:"average_progress"`
}

func runOverview(deps support.Dependencies, args []string) error {
	fs := support.NewFlagSet("overview")
	verbose := fs.Bool("verbose", false, "Include milestones and dependencies")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	sectorsBody, err := deps.Get("/tech-tree/sectors", nil)
	if err != nil {
		return err
	}
	var sectorsResponse support.SectorListResponse
	if err := support.Decode(sectorsBody, &sectorsResponse); err != nil {
		return err
	}

	sectorCount := len(sectorsResponse.Sectors)
	avgProgress := 0
	if sectorCount > 0 {
		var total float64
		for _, sector := range sectorsResponse.Sectors {
			total += sector.ProgressPercentage
		}
		avgProgress = int(total / float64(sectorCount))
	}

	statusLines := []string{
		support.TreeScopeLine(deps.Selector),
		fmt.Sprintf("Sectors tracked: %d", sectorCount),
		fmt.Sprintf("Average progress: %d%%", avgProgress),
	}
	topSectors := sectorRows(sectorsResponse.Sectors, 5)
	triage := []cliapp.TriageGroup{
		{Heading: "Leading sectors", Items: topSectors},
	}

	if *verbose {
		milestonesBody, err := deps.Get("/milestones", nil)
		if err == nil {
			var milestones support.MilestonesResponse
			if support.Decode(milestonesBody, &milestones) == nil {
				triage = append(triage, cliapp.TriageGroup{
					Heading: "Milestones",
					Items:   milestoneRows(milestones.Milestones, 5),
				})
			}
		}

		dependenciesBody, err := deps.Get("/dependencies", nil)
		if err == nil {
			var dependencies support.DependenciesResponse
			if support.Decode(dependenciesBody, &dependencies) == nil {
				triage = append(triage, cliapp.TriageGroup{
					Heading: "Dependency pressure",
					Items:   dependencyRows(dependencies.Dependencies, true, 5),
				})
			}
		}
	}

	report := overviewReport{
		OperationalReport: cliapp.OperationalReport{
			Status: statusLines,
			Triage: triage,
			NextSteps: []string{
				"tech-tree-designer analyze --resources 8 --timeline 6",
				"tech-tree-designer trees list",
				"tech-tree-designer graph dependencies --bottlenecks",
			},
		},
		Scope:           support.TreeScopeLine(deps.Selector),
		SectorCount:     sectorCount,
		AverageProgress: avgProgress,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report.OperationalReport)
}

type analyzeReport struct {
	cliapp.OperationalReport `json:",inline"`
	Response                 support.AnalysisResponse `json:"response"`
}

func runAnalyze(deps support.Dependencies, args []string) error {
	fs := support.NewFlagSet("analyze")
	resources := fs.Int("resources", 5, "Available development resources (1-10 scale)")
	timeline := fs.Int("timeline", 12, "Planning horizon in months")
	priority := fs.String("priority", "", "Comma-separated priority sectors")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := deps.Request("POST", "/tech-tree/analyze", nil, map[string]interface{}{
		"current_resources": *resources,
		"time_horizon":      *timeline,
		"priority_sectors":  support.TrimmedCSV(*priority),
	})
	if err != nil {
		return err
	}
	var response support.AnalysisResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}

	report := analyzeReport{
		OperationalReport: cliapp.OperationalReport{
			Status: []string{
				support.TreeScopeLine(deps.Selector),
				fmt.Sprintf("Resources: %d", *resources),
				fmt.Sprintf("Timeline: %d months", *timeline),
				fmt.Sprintf("Recommendations returned: %d", len(response.Recommendations)),
			},
			Triage: []cliapp.TriageGroup{
				{Heading: "Top recommendations", Items: recommendationRows(response.Recommendations, 5)},
				{Heading: "Bottlenecks", Items: response.BottleneckAnalysis},
				{Heading: "Projected milestones", Items: timelineRows(response.ProjectedTimeline.Milestones, 5)},
			},
			NextSteps: []string{
				"tech-tree-designer recommend --resources 8",
				"tech-tree-designer progress list",
				"tech-tree-designer milestones list",
			},
		},
		Response: response,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report.OperationalReport)
}

type recommendReport struct {
	cliapp.ListReport `json:",inline"`
	Response          support.RecommendationsResponse `json:"response"`
}

func runRecommend(deps support.Dependencies, args []string) error {
	fs := support.NewFlagSet("recommend")
	resources := fs.Int("resources", 5, "Available development resources")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := deps.Get("/recommendations", map[string][]string{"resources": {fmt.Sprintf("%d", *resources)}})
	if err != nil {
		return err
	}
	var response support.RecommendationsResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}

	report := recommendReport{
		ListReport: cliapp.ListReport{
			Summary: []string{
				support.TreeScopeLine(deps.Selector),
				fmt.Sprintf("Resources: %d", *resources),
				fmt.Sprintf("Recommendations: %d", len(response.Recommendations)),
			},
			ResultsHeading: "Recommendations",
			Results:        recommendationRows(response.Recommendations, 10),
			RetrievalHints: []string{
				"tech-tree-designer analyze --resources 8 --timeline 6",
				"tech-tree-designer progress list --scenario <scenario>",
			},
		},
		Response: response,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report.ListReport)
}

type visualizeReport struct {
	cliapp.MutationReport `json:",inline"`
	URL                   string `json:"url"`
	Opened                bool   `json:"opened"`
}

func runVisualize(deps support.Dependencies, args []string) error {
	fs := support.NewFlagSet("visualize")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	url := support.DashboardURL()
	opened, err := support.OpenBrowser(url)
	if err != nil {
		return err
	}
	report := visualizeReport{
		MutationReport: cliapp.MutationReport{
			Result: []string{
				"Prepared the Tech Tree Designer dashboard.",
				"URL: " + url,
			},
			Changes: []string{
				support.TreeScopeLine(deps.Selector),
			},
			NextCommand: []string{
				"tech-tree-designer overview --verbose",
				"tech-tree-designer trees list",
			},
		},
		URL:    url,
		Opened: opened,
	}
	if !opened {
		report.Changes = append(report.Changes, "No compatible browser opener was found; open the URL manually.")
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report.MutationReport)
}

func sectorRows(sectors []support.Sector, limit int) []string {
	if len(sectors) == 0 {
		return []string{"No sectors found."}
	}
	if limit > len(sectors) {
		limit = len(sectors)
	}
	rows := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		sector := sectors[i]
		rows = append(rows, fmt.Sprintf("%s (%s) | progress %s | %d stages", sector.Name, sector.Category, support.FormatPercent(sector.ProgressPercentage), len(sector.Stages)))
	}
	return rows
}

func recommendationRows(recs []support.StrategicRecommendation, limit int) []string {
	if len(recs) == 0 {
		return []string{"No recommendations returned."}
	}
	if limit > len(recs) {
		limit = len(recs)
	}
	rows := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		rec := recs[i]
		rows = append(rows, fmt.Sprintf("%s | priority %.2f | impact %.2fx | %s", rec.Scenario, rec.PriorityScore, rec.ImpactMultiplier, rec.Reasoning))
	}
	return rows
}

func milestoneRows(items []support.StrategicMilestone, limit int) []string {
	if len(items) == 0 {
		return []string{"No milestones found."}
	}
	if limit > len(items) {
		limit = len(items)
	}
	rows := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		item := items[i]
		rows = append(rows, fmt.Sprintf("%s | %s complete | target %s", item.Name, support.FormatPercent(item.CompletionPercentage), support.FormatDate(item.EstimatedCompletionDate)))
	}
	return rows
}

func dependencyRows(items []support.DependencyEntry, bottlenecksOnly bool, limit int) []string {
	rows := make([]string, 0, len(items))
	for _, item := range items {
		if bottlenecksOnly && item.Dependency.DependencyStrength < 0.8 {
			continue
		}
		rows = append(rows, fmt.Sprintf("%s <- %s | %s | strength %.0f%%", item.DependentName, item.PrerequisiteName, item.Dependency.DependencyType, item.Dependency.DependencyStrength*100))
	}
	if len(rows) == 0 {
		return []string{"No dependencies matched the current filter."}
	}
	if limit > len(rows) {
		limit = len(rows)
	}
	return rows[:limit]
}

func timelineRows(items []support.MilestoneProjection, limit int) []string {
	if len(items) == 0 {
		return []string{"No projected milestones returned."}
	}
	if limit > len(items) {
		limit = len(items)
	}
	rows := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		item := items[i]
		rows = append(rows, fmt.Sprintf("%s | %s | confidence %.0f%%", item.Name, item.EstimatedCompletion.Format("2006-01-02"), item.Confidence*100))
	}
	return rows
}
