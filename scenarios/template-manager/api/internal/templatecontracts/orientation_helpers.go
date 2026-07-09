package templatecontracts

import (
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/clipolicy"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cliout"
	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
)

type InfoOutput struct {
	Scenario InfoScenarioData `json:"scenario"`
}

type InfoScenarioData struct {
	Generation      *scenariomodel.GenerationMetadata `json:"generation,omitempty"`
	TemplateDrifted bool                              `json:"template_drifted,omitempty"`
}

func ParseOrientationRequest(globalsJSON bool, args []string) (OrientationRequest, error) {
	parsed, err := commandtree.ParseArgs("template-manager orient", TemplateOrientHelpText(), orientArgSchema(), args)
	if err != nil {
		return OrientationRequest{}, err
	}
	return OrientationRequest{
		Name:     parsed.Positionals[0],
		JSON:     globalsJSON || parsed.HasFlag("--json"),
		Finalize: parsed.HasFlag("--finalize"),
	}, nil
}

func ParseDetemplateRequest(globalsJSON bool, args []string) (DetemplateRequest, error) {
	parsed, err := commandtree.ParseArgs("template-manager detemplate", TemplateDetemplateHelpText(), detemplateArgSchema(), args)
	if err != nil {
		return DetemplateRequest{}, err
	}
	return DetemplateRequest{
		Name:   parsed.Positionals[0],
		JSON:   globalsJSON || parsed.HasFlag("--json"),
		DryRun: parsed.HasFlag("--dry-run"),
	}, nil
}

func TemplateOrientHelpText() string {
	return commandtree.HelpText("", "template-manager orient", "Show or finalize generated scenario orientation progress.", commandtree.Help{}, orientArgSchema())
}

func TemplateDetemplateHelpText() string {
	return commandtree.HelpText("", "template-manager detemplate", "Remove the template example domain from a generated scenario.", commandtree.Help{}, detemplateArgSchema())
}

func orientArgSchema() commandtree.ArgSchema {
	return commandtree.ArgSchema{
		Positionals: []commandtree.PositionalArg{{Name: "scenario name", Required: true}},
		Options: []commandtree.OptionArg{
			commandtree.JSONOption(),
			{Name: "--finalize", Description: "Remove temporary orientation metadata after required checks pass"},
		},
	}
}

func detemplateArgSchema() commandtree.ArgSchema {
	return commandtree.ArgSchema{
		Positionals: []commandtree.PositionalArg{{Name: "scenario name", Required: true}},
		Options: []commandtree.OptionArg{
			commandtree.JSONOption(),
			{Name: "--dry-run", Description: "Preview removals without writing, deleting, or running finalizers"},
		},
	}
}

func RenderOrientationResponse(w io.Writer, format cliout.Format, report OrientationReport) error {
	if format == cliout.FormatJSON {
		return writeScenarioOrientationJSON(w, report)
	}
	if report.Finalized {
		_, _ = fmt.Fprintf(w, "Orientation finalized for %s\n", report.Scenario)
		if strings.TrimSpace(report.Message) != "" {
			_, _ = fmt.Fprintln(w, report.Message)
		}
		return nil
	}
	if strings.TrimSpace(report.Message) != "" && len(report.Steps) == 0 {
		_, _ = fmt.Fprintln(w, report.Message)
		return nil
	}
	_, _ = fmt.Fprintf(w, "Orientation for %s\n", report.Scenario)
	_, _ = fmt.Fprintf(w, "Progress: %d/%d required steps complete\n", report.Completed, report.Required)
	for _, step := range report.Steps {
		marker := "[ ]"
		if step.Complete {
			marker = "[x]"
		}
		_, _ = fmt.Fprintf(w, "  %s %s", marker, step.ID)
		if step.Title != "" {
			_, _ = fmt.Fprintf(w, " - %s", step.Title)
		}
		if !step.Required {
			_, _ = fmt.Fprint(w, " (optional)")
		}
		_, _ = fmt.Fprintln(w)
	}
	if report.FinalizeRequired {
		_, _ = fmt.Fprintln(w, "Run with --finalize after required steps pass.")
	}
	return nil
}

func HelpOnly(err error) error {
	return clipolicy.CommandHelpOnly(err.Error())
}
