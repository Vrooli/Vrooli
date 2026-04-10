package task

import (
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// Run executes task subcommands.
func Run(client *Client, args []string) error {
	if len(args) == 0 {
		return printUsage()
	}

	switch args[0] {
	case "create":
		return runCreate(client, args[1:])
	case "list":
		return runList(client, args[1:])
	case "get":
		return runGet(client, args[1:])
	case "stop":
		return runStop(client, args[1:])
	case "agent-status":
		return runAgentStatus(client, args[1:])
	case "help", "-h", "--help":
		return printUsage()
	default:
		return fmt.Errorf("unknown subcommand: %s\n\nRun 'scenario-to-cloud task help' for usage", args[0])
	}
}

func printUsage() error {
	fmt.Println(`Usage: scenario-to-cloud task <command> [arguments]

Commands:
  create <deployment-id> <prompt>     Create a new AI task
  list <deployment-id>                List tasks for a deployment
  get <deployment-id> <task-id>       Get task details
  stop <deployment-id> <task-id>      Stop a running task
  agent-status                        Show AI agent status

Run 'scenario-to-cloud task <command> -h' for command-specific options.`)
	return nil
}

func runCreate(client *Client, args []string) error {
	var deploymentID, prompt, taskType string
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud task create <deployment-id> <prompt> [flags]

Creates a new AI task to investigate or diagnose issues.

Flags:
  --type <type>    Task type: investigate, diagnose, fix, analyze (default: investigate)
  --json           Output raw JSON

Examples:
  scenario-to-cloud task create abc123 "Why is the API returning 500 errors?"
  scenario-to-cloud task create abc123 "Check if postgres is healthy" --type diagnose`)
			return nil
		case "--type":
			if i+1 < len(args) {
				i++
				taskType = args[i]
			}
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") {
				if deploymentID == "" {
					deploymentID = args[i]
				} else if prompt == "" {
					prompt = args[i]
				} else {
					// Append to prompt if more args
					prompt = prompt + " " + args[i]
				}
			}
		}
	}

	if deploymentID == "" || prompt == "" {
		return fmt.Errorf("usage: scenario-to-cloud task create <deployment-id> <prompt>")
	}

	req := CreateRequest{
		Prompt: prompt,
		Type:   taskType,
	}

	body, resp, err := client.Create(deploymentID, req)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	t := resp.Task
	fmt.Printf("Created Task: %s\n", t.ID)
	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("Type:   %s\n", t.Type)
	fmt.Printf("Status: %s\n", t.Status)
	fmt.Printf("Prompt: %s\n", t.Prompt)
	fmt.Printf("\nUse 'task get %s %s' to check progress.\n", deploymentID, t.ID)

	return nil
}

func runList(client *Client, args []string) error {
	var deploymentID string
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud task list <deployment-id> [flags]

Flags:
  --json    Output raw JSON`)
			return nil
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") && deploymentID == "" {
				deploymentID = args[i]
			}
		}
	}

	if deploymentID == "" {
		return fmt.Errorf("usage: scenario-to-cloud task list <deployment-id>")
	}

	body, resp, err := client.List(deploymentID)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// Pretty print
	if len(resp.Tasks) == 0 {
		fmt.Println("No tasks found.")
		return nil
	}

	fmt.Printf("Tasks: %d\n", len(resp.Tasks))
	fmt.Println(strings.Repeat("-", 90))
	fmt.Printf("%-20s %-12s %-10s %-20s %s\n", "ID", "TYPE", "STATUS", "CREATED", "PROMPT")

	for _, t := range resp.Tasks {
		id := t.ID
		if len(id) > 16 {
			id = id[:16] + "..."
		}
		prompt := t.Prompt
		if len(prompt) > 30 {
			prompt = prompt[:27] + "..."
		}
		fmt.Printf("%-20s %-12s %-10s %-20s %s\n", id, t.Type, t.Status, t.CreatedAt, prompt)
	}

	return nil
}

func runGet(client *Client, args []string) error {
	var deploymentID, taskID string
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud task get <deployment-id> <task-id> [flags]

Flags:
  --json    Output raw JSON`)
			return nil
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") {
				if deploymentID == "" {
					deploymentID = args[i]
				} else if taskID == "" {
					taskID = args[i]
				}
			}
		}
	}

	if deploymentID == "" || taskID == "" {
		return fmt.Errorf("usage: scenario-to-cloud task get <deployment-id> <task-id>")
	}

	body, resp, err := client.Get(deploymentID, taskID)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// Pretty print
	t := resp.Task
	fmt.Printf("Task: %s\n", t.ID)
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("Type:       %s\n", t.Type)
	fmt.Printf("Status:     %s\n", t.Status)
	fmt.Printf("Prompt:     %s\n", t.Prompt)
	if t.Progress != "" {
		fmt.Printf("Progress:   %s\n", t.Progress)
	}
	fmt.Printf("Created:    %s\n", t.CreatedAt)
	if t.CompletedAt != "" {
		fmt.Printf("Completed:  %s\n", t.CompletedAt)
	}

	if t.Result != "" {
		fmt.Println("\nResult:")
		fmt.Println(t.Result)
	}

	if t.ErrorMessage != "" {
		fmt.Printf("\nError: %s\n", t.ErrorMessage)
	}

	return nil
}

func runStop(client *Client, args []string) error {
	var deploymentID, taskID string
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud task stop <deployment-id> <task-id> [flags]

Flags:
  --json    Output raw JSON`)
			return nil
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") {
				if deploymentID == "" {
					deploymentID = args[i]
				} else if taskID == "" {
					taskID = args[i]
				}
			}
		}
	}

	if deploymentID == "" || taskID == "" {
		return fmt.Errorf("usage: scenario-to-cloud task stop <deployment-id> <task-id>")
	}

	body, resp, err := client.Stop(deploymentID, taskID)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if resp.Success {
		fmt.Printf("Stopped task: %s\n", resp.TaskID)
	} else {
		fmt.Printf("Failed to stop task: %s\n", resp.Message)
	}

	return nil
}

func runAgentStatus(client *Client, args []string) error {
	jsonOutput := false

	for _, arg := range args {
		switch arg {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud task agent-status [flags]

Shows the status of the AI agent.

Flags:
  --json    Output raw JSON`)
			return nil
		case "--json":
			jsonOutput = true
		}
	}

	body, resp, err := client.AgentStatus()
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// Pretty print
	fmt.Println("AI Agent Status")
	fmt.Println(strings.Repeat("-", 40))

	status := "Unavailable"
	if resp.Available {
		status = "Available"
	}
	fmt.Printf("Status:       %s\n", status)

	if resp.Provider != "" {
		fmt.Printf("Provider:     %s\n", resp.Provider)
	}
	if resp.Model != "" {
		fmt.Printf("Model:        %s\n", resp.Model)
	}
	fmt.Printf("Active Tasks: %d\n", resp.ActiveTasks)
	fmt.Printf("Queued Tasks: %d\n", resp.QueuedTasks)

	if len(resp.Capabilities) > 0 {
		fmt.Printf("Capabilities: %s\n", strings.Join(resp.Capabilities, ", "))
	}

	return nil
}
