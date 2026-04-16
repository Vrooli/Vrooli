package content

import (
	"fmt"
	"os"

	"campaign-content-studio/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `campaign-content-studio generate` as a flat command since
// content generation is a single mutating surface (`POST /generate`). Flags
// mirror the bash CLI: --include-images and --body-file for rich payloads.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Content",
		Commands: []cliapp.Command{
			{
				Name:        "generate",
				Description: "Generate content for a campaign (blog_post|social_media|marketing_copy|image)",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runGenerate(core, args) },
			},
		},
	}
}

func runGenerate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("generate")
	includeImages := fs.Bool("include-images", false, "Include image generation")
	bodyFile := fs.String("body-file", "", "Path to JSON file with full request body (overrides positional args)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if raw, err := support.ReadJSONFile(*bodyFile, false); err != nil {
		return err
	} else if raw != nil {
		payload = raw
	} else {
		if fs.NArg() < 3 {
			return fmt.Errorf("usage: generate <campaign-id> <content-type> <prompt> [--include-images] | --body-file <path>")
		}
		payload = map[string]interface{}{
			"campaign_id":    fs.Arg(0),
			"content_type":   fs.Arg(1),
			"prompt":         fs.Arg(2),
			"include_images": *includeImages,
		}
	}

	body, err := core.Request("POST", "/generate", nil, payload)
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := support.Decode(body, &result); err != nil {
		result = nil
	}

	report := cliapp.ListReport{
		Summary:        []string{"Generated content"},
		ResultsHeading: "Response",
		Results:        support.MapRows(result),
		RetrievalHints: []string{
			fmt.Sprintf("%s campaign list", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
