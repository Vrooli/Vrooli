package overview

import (
	"errors"
	"flag"

	"deployment-manager/cli/cmdutil"

	"github.com/vrooli/cli-core/cliutil"
)

type Commands struct {
	api *cliutil.APIClient
}

func New(api *cliutil.APIClient) *Commands {
	return &Commands{api: api}
}

func (c *Commands) Status(args []string) error {
	body, err := c.api.Get("/health", nil)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func (c *Commands) Analyze(args []string) error {
	fs := flag.NewFlagSet("analyze", flag.ContinueOnError)
	format := fs.String("format", "", "output format (json|table)")
	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return errors.New("scenario is required")
	}
	scenario := remaining[0]
	body, err := c.api.Get("/api/v1/dependencies/analyze/"+scenario, nil)
	if err != nil {
		return err
	}
	cmdutil.PrintByFormat(*format, body)
	return nil
}

func (c *Commands) Fitness(args []string) error {
	fs := flag.NewFlagSet("fitness", flag.ContinueOnError)
	tier := fs.String("tier", "2", "deployment tier")
	format := fs.String("format", "", "output format (json|table)")
	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return errors.New("scenario is required")
	}
	scenario := remaining[0]
	tierNum := cmdutil.TierToNumber(*tier)
	payload := map[string]interface{}{
		"scenario": scenario,
		"tiers":    []int{tierNum},
	}
	body, err := c.api.Request("POST", "/api/v1/fitness/score", nil, payload)
	if err != nil {
		return err
	}
	cmdutil.PrintByFormat(*format, body)
	return nil
}
