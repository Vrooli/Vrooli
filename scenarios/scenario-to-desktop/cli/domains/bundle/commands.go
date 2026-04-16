// Package bundle provides bundle-related CLI commands.
package bundle

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"scenario-to-desktop/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Commands provides bundle CLI commands.
type Commands struct {
	deps support.Dependencies
}

// New creates a new bundle Commands instance.
func New(deps support.Dependencies) *Commands {
	return &Commands{deps: deps}
}

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	cmds := New(deps)
	return cliapp.SubcommandGroup{
		Name:        "bundle",
		Description: "Bundle output utilities (run 'bundle help' for details)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "clean", Description: "Clean bundle output directory: clean <scenario> [--location-mode ...]", Run: cmds.Clean},
		},
	}
}

// Clean removes the bundle output directory for a scenario.
func (c *Commands) Clean(args []string) error {
	fs := flag.NewFlagSet("bundle-clean", flag.ContinueOnError)
	framework := fs.String("framework", "electron", "Desktop framework (default: electron)")
	locationMode := fs.String("location-mode", "proper", "Output location: proper (default), staging, temp")
	pipelineID := fs.String("pipeline-id", "", "Pipeline ID (required for staging/temp location-mode)")
	jsonOut := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("usage: bundle-clean <scenario> [--framework electron] [--location-mode proper|staging|temp] [--pipeline-id <id>] [--json]")
	}

	scenario := strings.TrimSpace(fs.Args()[0])
	if scenario == "" {
		return fmt.Errorf("scenario is required")
	}

	loc := strings.TrimSpace(*locationMode)
	if (loc == "staging" || loc == "temp") && strings.TrimSpace(*pipelineID) == "" {
		return fmt.Errorf("--pipeline-id is required when --location-mode is staging/temp")
	}

	body := map[string]interface{}{
		"framework":     strings.TrimSpace(*framework),
		"location_mode": loc,
		"pipeline_id":   strings.TrimSpace(*pipelineID),
	}

	respBytes, err := c.deps.Request("POST", fmt.Sprintf("/scenarios/%s/bundle/clean", scenario), nil, body)
	if err != nil {
		return err
	}

	if *jsonOut {
		fmt.Print(string(respBytes))
		if len(respBytes) == 0 || respBytes[len(respBytes)-1] != '\n' {
			fmt.Print("\n")
		}
		return nil
	}

	var resp struct {
		Path    string `json:"path"`
		Removed bool   `json:"removed"`
	}
	_ = json.Unmarshal(respBytes, &resp)
	if resp.Path != "" {
		if resp.Removed {
			fmt.Printf("Bundle cleaned: %s\n", resp.Path)
		} else {
			fmt.Printf("Bundle already clean: %s\n", resp.Path)
		}
		return nil
	}

	// Fallback to raw JSON if schema unexpectedly changes.
	cliutil.PrintJSON(json.RawMessage(respBytes))
	return nil
}
