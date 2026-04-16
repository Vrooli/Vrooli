package swaps

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"deployment-manager/cli/cmdutil"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

type Commands struct {
	api *cliutil.APIClient
}

func New(api *cliutil.APIClient) *Commands {
	return &Commands{api: api}
}

type SwapSuggestion struct {
	From   string  `json:"from"`
	To     string  `json:"to"`
	Reason string  `json:"reason"`
	Impact string  `json:"impact"`
	Score  float64 `json:"score"`
}

func (c *Commands) Run(args []string) error {
	if len(args) == 0 {
		return errors.New("swaps subcommand is required")
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		return c.list(rest)
	case "analyze":
		return c.analyze(rest)
	case "cascade":
		return c.cascade(rest)
	case "info":
		return c.info(rest)
	case "apply":
		return c.apply(rest)
	default:
		return errors.New("unknown swaps subcommand: " + sub)
	}
}

func (c *Commands) list(args []string) error {
	fs := flag.NewFlagSet("swaps list", flag.ContinueOnError)
	format := fs.String("format", "", "output format (json|table)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("scenario is required")
	}
	scenario := remaining[0]
	body, err := c.api.Get("/api/v1/swaps/suggest/"+scenario, nil)
	if err != nil {
		return err
	}
	formatVal := cmdutil.ResolveFormat(*format)
	if strings.ToLower(formatVal) == "table" {
		if err := renderSwapListReport(scenario, body); err == nil {
			return nil
		}
	}
	// Validate JSON shape even in default mode so agents get a clearer error.
	var suggestions []SwapSuggestion
	if err := json.Unmarshal(body, &suggestions); err != nil {
		return fmt.Errorf("parse swap suggestions: %w", err)
	}
	cmdutil.PrintByFormat(formatVal, body)
	return nil
}

func (c *Commands) analyze(args []string) error {
	fs := flag.NewFlagSet("swaps analyze", flag.ContinueOnError)
	format := fs.String("format", "", "output format (json)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) < 2 {
		return errors.New("usage: swaps analyze <from> <to>")
	}
	from := remaining[0]
	to := remaining[1]
	body, err := c.api.Get("/api/v1/swaps/analyze/"+from+"/"+to, nil)
	if err != nil {
		return err
	}
	cmdutil.PrintByFormat(*format, body)
	return nil
}

func (c *Commands) cascade(args []string) error {
	fs := flag.NewFlagSet("swaps cascade", flag.ContinueOnError)
	format := fs.String("format", "", "output format (json)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) < 2 {
		return errors.New("usage: swaps cascade <from> <to>")
	}
	from := remaining[0]
	to := remaining[1]
	body, err := c.api.Get("/api/v1/swaps/cascade/"+from+"/"+to, nil)
	if err != nil {
		return err
	}
	cmdutil.PrintByFormat(*format, body)
	return nil
}

func (c *Commands) info(args []string) error {
	fs := flag.NewFlagSet("swaps info", flag.ContinueOnError)
	format := fs.String("format", "", "output format (json)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("swap id is required")
	}
	id := remaining[0]
	body, err := c.api.Get("/api/v1/swaps/"+id, nil)
	if err != nil {
		return err
	}
	cmdutil.PrintByFormat(*format, body)
	return nil
}

func (c *Commands) apply(args []string) error {
	fs := flag.NewFlagSet("swaps apply", flag.ContinueOnError)
	showFitness := fs.Bool("show-fitness", false, "show fitness after apply")
	format := fs.String("format", "", "output format (json)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) < 3 {
		return errors.New("usage: swaps apply <profile_id> <from> <to>")
	}
	profileID := remaining[0]
	from := remaining[1]
	to := remaining[2]

	payload := map[string]string{"from": from, "to": to}
	_, err := c.api.Request("POST", "/api/v1/profiles/"+profileID+"/swaps", nil, payload)
	if err != nil {
		return err
	}
	if *showFitness {
		body, err := c.api.Get("/api/v1/profiles/"+profileID, nil)
		if err != nil {
			return err
		}
		cmdutil.PrintByFormat(*format, body)
		return nil
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result: []string{
			"Swap applied successfully",
		},
		Changes: []string{
			fmt.Sprintf("Profile: %s", profileID),
			fmt.Sprintf("From: %s", from),
			fmt.Sprintf("To: %s", to),
		},
		NextCommand: []string{
			fmt.Sprintf("deployment-manager profile show %s", profileID),
			fmt.Sprintf("deployment-manager swaps list %s", profileID),
		},
	})
}

func renderSwapListReport(scenario string, body []byte) error {
	var suggestions []SwapSuggestion
	if err := json.Unmarshal(body, &suggestions); err != nil {
		return err
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Scenario: %s", scenario),
			fmt.Sprintf("Suggestions: %d", len(suggestions)),
		},
		ResultsHeading: "Suggested Swaps",
		RetrievalHints: []string{
			fmt.Sprintf("deployment-manager swaps analyze <from> <to>"),
			fmt.Sprintf("deployment-manager swaps apply <profile> <from> <to>"),
		},
	}
	for _, s := range suggestions {
		line := fmt.Sprintf("%s -> %s impact=%s", s.From, s.To, s.Impact)
		if strings.TrimSpace(s.Reason) != "" {
			line += fmt.Sprintf(" reason=%s", s.Reason)
		}
		report.Results = append(report.Results, line)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
