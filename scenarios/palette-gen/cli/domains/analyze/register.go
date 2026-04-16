package analyze

import (
	"fmt"
	"os"
	"strings"

	"palette-gen/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `analyze` subcommand group covering contrast/harmony/colorblind.
// Each subcommand is a thin wrapper over a single API endpoint.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "analyze",
		Description: "Analyze contrast, color harmony, and colorblind simulations",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "contrast", Description: "Check WCAG contrast between foreground and background colors", Run: func(args []string) error { return runContrast(core, args) }},
			{Name: "harmony", Description: "Analyze color harmony relationships", Run: func(args []string) error { return runHarmony(core, args) }},
			{Name: "colorblind", Description: "Simulate colorblindness (protanopia, deuteranopia, tritanopia)", Run: func(args []string) error { return runColorblind(core, args) }},
		},
	}
}

func runContrast(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("analyze contrast")
	bodyFile := fs.String("body-file", "", "Path to a JSON file containing the full request body (overrides positional arguments)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if fs.NArg() < 2 {
			return fmt.Errorf("usage: analyze contrast <foreground> <background>")
		}
		payload = map[string]interface{}{
			"foreground": strings.TrimSpace(fs.Arg(0)),
			"background": strings.TrimSpace(fs.Arg(1)),
		}
	}

	body, err := core.Request("POST", "/accessibility", nil, payload)
	if err != nil {
		return err
	}
	var resp support.AccessibilityResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{
		fmt.Sprintf("Contrast ratio: %.2f", resp.ContrastRatio),
	}
	if resp.Recommendation != "" {
		summary = append(summary, resp.Recommendation)
	}

	results := []string{
		fmt.Sprintf("WCAG AA (normal text): %t", resp.WCAGAA),
		fmt.Sprintf("WCAG AAA (normal text): %t", resp.WCAGAAA),
		fmt.Sprintf("WCAG AA (large text): %t", resp.LargeTextAA),
		fmt.Sprintf("WCAG AAA (large text): %t", resp.LargeTextAAA),
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Accessibility",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s analyze harmony <hex,hex,...>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runHarmony(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("analyze harmony")
	bodyFile := fs.String("body-file", "", "Path to a JSON file containing the full request body (overrides positional argument)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if fs.NArg() < 1 {
			return fmt.Errorf("usage: analyze harmony <hex,hex,...>")
		}
		colors := support.SplitColors(fs.Arg(0))
		if len(colors) == 0 {
			return fmt.Errorf("at least one color is required")
		}
		payload = map[string]interface{}{
			"colors": colors,
		}
	}

	body, err := core.Request("POST", "/harmony", nil, payload)
	if err != nil {
		return err
	}
	var resp support.HarmonyResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{
		fmt.Sprintf("Harmonious: %t", resp.IsHarmonious),
		fmt.Sprintf("Score: %.2f", resp.Score),
	}
	results := support.MapRows(resp.Analysis)

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Harmony analysis",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s analyze colorblind <hex,hex,...> <type>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runColorblind(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("analyze colorblind")
	bodyFile := fs.String("body-file", "", "Path to a JSON file containing the full request body (overrides positional arguments)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	var cbType string
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if fs.NArg() < 1 {
			return fmt.Errorf("usage: analyze colorblind <hex,hex,...> [type]")
		}
		colors := support.SplitColors(fs.Arg(0))
		if len(colors) == 0 {
			return fmt.Errorf("at least one color is required")
		}
		cbType = "protanopia"
		if fs.NArg() >= 2 {
			cbType = strings.TrimSpace(fs.Arg(1))
		}
		payload = map[string]interface{}{
			"colors": colors,
			"type":   cbType,
		}
	}

	body, err := core.Request("POST", "/colorblind", nil, payload)
	if err != nil {
		return err
	}
	var resp support.ColorblindResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{}
	if resp.Type != "" {
		summary = append(summary, fmt.Sprintf("Type: %s", resp.Type))
	} else if cbType != "" {
		summary = append(summary, fmt.Sprintf("Type: %s", cbType))
	}
	summary = append(summary, fmt.Sprintf("Simulated colors: %d", len(resp.Simulated)))

	results := make([]string, 0, len(resp.Simulated))
	for i, c := range resp.Simulated {
		results = append(results, fmt.Sprintf("%d. %s", i+1, c))
	}
	if len(results) == 0 {
		results = []string{"(no colors returned)"}
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Simulated palette",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s analyze colorblind <hex,hex,...> deuteranopia", support.CLIName),
			fmt.Sprintf("%s analyze colorblind <hex,hex,...> tritanopia", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
