package pipeline

import (
	"encoding/json"
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// Register exposes the operator-facing delivery-ramp commands. Gate
// decisions remain API-owned; the CLI only transports requests and renders
// the typed JSON response.
func Register(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{{Name: "readiness", Description: "Inspect plugin publish readiness", NeedsAPI: true, Subcommands: []cliapp.Command{
		{Name: "list", Description: "List readiness and named blockers", Run: func(args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("usage: scenario-to-plugin readiness list")
			}
			raw, err := core.Get("/api/v1/readiness", nil)
			if err != nil {
				return err
			}
			fmt.Println(string(raw))
			return nil
		}},
	}}, {Name: "package", Description: "Compose and inspect packages", NeedsAPI: true, Subcommands: []cliapp.Command{
		{Name: "compose", Description: "Compose an Agent Plugin tree", Run: func(args []string) error {
			if len(args) < 1 || len(args) > 2 {
				return fmt.Errorf("usage: scenario-to-plugin package compose <scenario> [revision]")
			}
			revision := "working-tree"
			if len(args) == 2 {
				revision = args[1]
			}
			body, err := core.Request("POST", "/package/compose", nil, map[string]string{"scenario": args[0], "source_revision": revision})
			if err != nil {
				return err
			}
			fmt.Println(string(body))
			return nil
		}},
		{Name: "show", Description: "Show a package record", Run: func(args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("usage: scenario-to-plugin package show <package-id>")
			}
			raw, err := core.Get("/package/"+args[0], nil)
			if err != nil {
				return err
			}
			fmt.Println(string(raw))
			return nil
		}},
	}}, {Name: "publish", Description: "Governed publication operations", NeedsAPI: true, Subcommands: []cliapp.Command{
		{Name: "run", Description: "Publish only with a matching release decision", Run: func(args []string) error {
			if len(args) != 2 {
				return fmt.Errorf("usage: scenario-to-plugin publish run <package-id> <channel>")
			}
			body, err := core.Request("POST", "/publish", nil, map[string]string{"package_id": args[0], "channel": args[1]})
			if err != nil {
				return err
			}
			var response struct {
				Published bool   `json:"published"`
				Refusal   string `json:"refusal"`
			}
			if json.Unmarshal(body, &response) == nil && response.Refusal != "" {
				return fmt.Errorf("publication refused: %s", response.Refusal)
			}
			fmt.Println(string(body))
			return nil
		}},
	}}}
}
