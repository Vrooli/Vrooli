package playbooksseed

import (
	"fmt"
	"strings"
)

// Run executes the playbooks seed lifecycle command.
// Usage:
//   test-genie playbooks-seed apply --scenario <name> [--retain]
//   test-genie playbooks-seed cleanup --scenario <name> --token <token>
func Run(client *Client, args []string) error {
	if client == nil {
		return fmt.Errorf("client is required")
	}
	if len(args) == 0 {
		return fmt.Errorf("usage: playbooks-seed <apply|cleanup> --scenario <name> [options]")
	}

	action := strings.ToLower(strings.TrimSpace(args[0]))
	args = args[1:]

	var scenario string
	var token string
	retain := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--scenario":
			if i+1 >= len(args) {
				return fmt.Errorf("--scenario requires a value")
			}
			scenario = strings.TrimSpace(args[i+1])
			i++
		case "--token":
			if i+1 >= len(args) {
				return fmt.Errorf("--token requires a value")
			}
			token = strings.TrimSpace(args[i+1])
			i++
		case "--retain":
			retain = true
		default:
			return fmt.Errorf("unknown option: %s", args[i])
		}
	}

	if scenario == "" {
		return fmt.Errorf("--scenario is required")
	}

	switch action {
	case "apply":
		resp, _, err := client.Apply(scenario, ApplyRequest{Retain: retain})
		if err != nil {
			return err
		}
		fmt.Printf("status: %s\n", resp.Status)
		fmt.Printf("scenario: %s\n", resp.Scenario)
		fmt.Printf("run_id: %s\n", resp.RunID)
		fmt.Printf("cleanup_token: %s\n", resp.CleanupToken)
		return nil
	case "cleanup":
		if token == "" {
			return fmt.Errorf("--token is required for cleanup")
		}
		resp, _, err := client.Cleanup(scenario, CleanupRequest{CleanupToken: token})
		if err != nil {
			return err
		}
		fmt.Printf("status: %s\n", resp.Status)
		fmt.Printf("scenario: %s\n", resp.Scenario)
		fmt.Printf("run_id: %s\n", resp.RunID)
		return nil
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}
