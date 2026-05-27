package claims

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Run dispatches the `test-genie claims` subcommand.
//
// Usage:
//
//	test-genie claims list
//	test-genie claims release <scenario> [--yes] [--actor <name>]
func Run(client *Client, args []string) error {
	if client == nil {
		return fmt.Errorf("client is required")
	}
	if len(args) == 0 {
		return fmt.Errorf("usage: claims <list|release> [scenario] [options]")
	}

	action := strings.ToLower(strings.TrimSpace(args[0]))
	rest := args[1:]

	switch action {
	case "list":
		return runList(client)
	case "release":
		return runRelease(client, rest)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

func runList(client *Client) error {
	claims, err := client.List()
	if err != nil {
		return err
	}
	if len(claims) == 0 {
		fmt.Println("No active playbooks claims.")
		return nil
	}
	now := time.Now().UTC()
	fmt.Printf("%-32s  %-36s  %-9s  %-16s  %-8s  %s\n",
		"SCENARIO", "RUN_ID", "MODE", "STARTED_BY", "ALIVE", "HEARTBEAT_AGE")
	for _, c := range claims {
		age := now.Sub(c.HeartbeatAt).Round(time.Second)
		alive := "yes"
		if !c.Alive {
			alive = "stale"
		}
		fmt.Printf("%-32s  %-36s  %-9s  %-16s  %-8s  %s\n",
			c.ScenarioName, c.RunID, c.Mode, c.StartedBy, alive, age)
	}
	return nil
}

func runRelease(client *Client, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: claims release <scenario> [--yes] [--actor <name>]")
	}
	scenario := strings.TrimSpace(args[0])
	if scenario == "" {
		return fmt.Errorf("scenario is required")
	}
	yes := false
	actor := strings.TrimSpace(os.Getenv("USER"))
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--yes", "-y":
			yes = true
		case "--actor":
			if i+1 >= len(args) {
				return fmt.Errorf("--actor requires a value")
			}
			actor = strings.TrimSpace(args[i+1])
			i++
		default:
			return fmt.Errorf("unknown option: %s", args[i])
		}
	}

	current, err := client.Get(scenario)
	if err != nil {
		return err
	}
	if current == nil {
		fmt.Printf("No active claim for scenario %q.\n", scenario)
		return nil
	}
	fmt.Printf("Active claim on %s:\n", scenario)
	fmt.Printf("  run_id:        %s\n", current.RunID)
	fmt.Printf("  mode:          %s\n", current.Mode)
	fmt.Printf("  started_by:    %s\n", current.StartedBy)
	fmt.Printf("  heartbeat_at:  %s\n", current.HeartbeatAt.Format(time.RFC3339))
	fmt.Printf("  alive:         %v\n", current.Alive)
	if !yes {
		fmt.Print("\nForce-release this claim? [y/N]: ")
		var resp string
		// A read error (EOF/empty input) leaves resp empty, which the check
		// below treats as "N" — the safe default for a destructive prompt.
		_, _ = fmt.Scanln(&resp)
		if !strings.EqualFold(strings.TrimSpace(resp), "y") {
			fmt.Println("Aborted.")
			return nil
		}
	}
	broken, err := client.Release(scenario, actor)
	if err != nil {
		return err
	}
	fmt.Printf("\nReleased claim %s on %s (actor=%s)\n", broken.RunID, scenario, actor)
	return nil
}
