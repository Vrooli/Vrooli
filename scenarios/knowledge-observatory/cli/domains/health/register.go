package health

import (
	"flag"
	"fmt"
	"time"

	"knowledge-observatory/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Health",
		Commands: []cliapp.Command{
			{Name: "health", NeedsAPI: true, Description: "Get knowledge health metrics", Run: func(args []string) error { return run(deps, args) }},
			{Name: "metrics", NeedsAPI: true, Description: "Alias for health", Run: func(args []string) error { return run(deps, args) }},
		},
	}
}

func run(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("health", flag.ContinueOnError)
	watch := fs.Bool("watch", false, "Poll health metrics continuously")
	intervalRaw := fs.String("interval", "5s", "Polling interval when --watch is set")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if !*watch {
		body, err := deps.ScenarioApp().Request("GET", "/knowledge/health", nil, nil)
		if err != nil {
			return err
		}
		cliutil.PrintJSON(body)
		return nil
	}

	interval, err := support.ParseDuration(*intervalRaw)
	if err != nil {
		return err
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		body, err := deps.ScenarioApp().Request("GET", "/knowledge/health", nil, nil)
		if err != nil {
			return err
		}
		fmt.Printf("== %s ==\n", time.Now().UTC().Format(time.RFC3339))
		cliutil.PrintJSON(body)
		<-ticker.C
	}
}
