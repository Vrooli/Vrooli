// Package queue provides CLI commands for queue management.
package queue

import (
	"ecosystem-manager/cli/internal/appctx"
	"ecosystem-manager/cli/internal/format"
	"flag"
	"fmt"
	"os"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// QueueStatusResponse represents the queue status.
type QueueStatusResponse struct {
	IsActive           bool     `json:"is_active"`
	IsPaused           bool     `json:"is_paused"`
	IsRateLimitPaused  bool     `json:"is_rate_limit_paused"`
	RateLimitResumeAt  string   `json:"rate_limit_resume_at,omitempty"`
	PendingCount       int      `json:"pending_count"`
	InProgressCount    int      `json:"in_progress_count"`
	RunningProcesses   int      `json:"running_processes"`
	AvailableSlots     int      `json:"available_slots"`
	MaxSlots           int      `json:"max_slots"`
	CooldownSeconds    int      `json:"cooldown_seconds"`
	TaskTimeoutMinutes int      `json:"task_timeout_minutes"`
	NextSteps          []string `json:"next_steps,omitempty"`
}

// ActionResponse represents a generic action response.
type ActionResponse struct {
	Success   bool     `json:"success"`
	DryRun    bool     `json:"dry_run,omitempty"`
	Message   string   `json:"message"`
	NextSteps []string `json:"next_steps,omitempty"`
}

// Commands returns the queue command group.
func Commands(ctx appctx.Context) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Queue",
		Commands: []cliapp.Command{
			{
				Name:        "queue",
				NeedsAPI:    true,
				Description: "Manage queue (status|start|stop)",
				Run: func(args []string) error {
					return route(ctx, args)
				},
			},
		},
	}
}

func route(ctx appctx.Context, args []string) error {
	if len(args) == 0 {
		return printUsage()
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "help":
		return printUsage()
	case "status":
		return cmdStatus(ctx, subArgs)
	case "start":
		return cmdStart(ctx, subArgs)
	case "stop":
		return cmdStop(ctx, subArgs)
	default:
		return fmt.Errorf("unknown subcommand: %s\n\n%s", subcommand, usageText())
	}
}

func printUsage() error {
	fmt.Println(usageText())
	return nil
}

func usageText() string {
	return `Usage: ecosystem-manager queue <subcommand>

Subcommands:
  status    Show queue status
  start     Start the queue processor
  stop      Stop the queue processor

Examples:
  ecosystem-manager queue status
  ecosystem-manager queue start
  ecosystem-manager queue status --json`
}

func cmdStatus(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	var status QueueStatusResponse
	if err := ctx.Get("/queue/status", &status); err != nil {
		return format.WrapAPIError("Failed to get queue status", err)
	}

	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, status)
	}

	active := "Queue processor is inactive"
	if status.IsActive {
		active = "Queue processor is active"
	}
	triage := []cliapp.TriageGroup{
		{
			Heading: "Capacity",
			Items: []string{
				fmt.Sprintf("Pending tasks: %d", status.PendingCount),
				fmt.Sprintf("In-progress tasks: %d", status.InProgressCount),
				fmt.Sprintf("Running processes: %d / %d", status.RunningProcesses, status.MaxSlots),
				fmt.Sprintf("Available slots: %d", status.AvailableSlots),
				fmt.Sprintf("Cooldown: %ds", status.CooldownSeconds),
				fmt.Sprintf("Task timeout: %dm", status.TaskTimeoutMinutes),
			},
		},
	}
	if status.IsPaused {
		active += " (paused)"
	}
	if status.IsRateLimitPaused {
		active += fmt.Sprintf(" (rate-limited until %s)", status.RateLimitResumeAt)
	}
	return cliapp.RenderOperationalReport(os.Stdout, cliapp.OperationalReport{
		Status: []string{active},
		Triage: triage,
		NextSteps: func() []string {
			if len(status.NextSteps) > 0 {
				return status.NextSteps
			}
			return []string{"ecosystem-manager task list"}
		}(),
	})
}

func cmdStart(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("start", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	var resp ActionResponse
	if err := ctx.Post("/queue/start", struct{}{}, &resp); err != nil {
		return format.WrapAPIError("Failed to start queue", err)
	}

	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}

	if resp.DryRun {
		fmt.Println("[DRY RUN] Queue processor would be started")
	} else if resp.Success {
		return format.MutationResult("Queue started successfully", "", resp.NextSteps)
	} else {
		fmt.Printf("Failed to start queue: %s\n", resp.Message)
	}
	return nil
}

func cmdStop(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("stop", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	var resp ActionResponse
	if err := ctx.Post("/queue/stop", struct{}{}, &resp); err != nil {
		return format.WrapAPIError("Failed to stop queue", err)
	}

	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}

	if resp.DryRun {
		fmt.Println("[DRY RUN] Queue processor would be stopped")
	} else if resp.Success {
		return format.MutationResult("Queue stopped successfully", "", resp.NextSteps)
	} else {
		fmt.Printf("Failed to stop queue: %s\n", resp.Message)
	}
	return nil
}
