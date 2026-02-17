// Package queue provides CLI commands for queue management.
package queue

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	"ecosystem-manager/cli/internal/appctx"
)

// QueueStatusResponse represents the queue status.
type QueueStatusResponse struct {
	IsActive           bool   `json:"is_active"`
	IsPaused           bool   `json:"is_paused"`
	IsRateLimitPaused  bool   `json:"is_rate_limit_paused"`
	RateLimitResumeAt  string `json:"rate_limit_resume_at,omitempty"`
	PendingCount       int    `json:"pending_count"`
	InProgressCount    int    `json:"in_progress_count"`
	RunningProcesses   int    `json:"running_processes"`
	AvailableSlots     int    `json:"available_slots"`
	MaxSlots           int    `json:"max_slots"`
	CooldownSeconds    int    `json:"cooldown_seconds"`
	TaskTimeoutMinutes int    `json:"task_timeout_minutes"`
}

// ActionResponse represents a generic action response.
type ActionResponse struct {
	Success bool   `json:"success"`
	DryRun  bool   `json:"dry_run,omitempty"`
	Message string `json:"message"`
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
  stop      Stop the queue processor`
}

func cmdStatus(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	var status QueueStatusResponse
	if err := ctx.Get("/queue/status", &status); err != nil {
		return fmt.Errorf("failed to get queue status: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(status)
	}

	active := "inactive"
	if status.IsActive {
		active = "active"
	}
	paused := ""
	if status.IsPaused {
		paused = " (paused)"
	}
	if status.IsRateLimitPaused {
		paused = fmt.Sprintf(" (rate-limited until %s)", status.RateLimitResumeAt)
	}

	fmt.Printf("Queue: %s%s\n", active, paused)
	fmt.Printf("Pending: %d\n", status.PendingCount)
	fmt.Printf("In Progress: %d\n", status.InProgressCount)
	fmt.Printf("Running Processes: %d / %d\n", status.RunningProcesses, status.MaxSlots)
	fmt.Printf("Available Slots: %d\n", status.AvailableSlots)
	fmt.Printf("Cooldown: %ds\n", status.CooldownSeconds)
	fmt.Printf("Task Timeout: %dm\n", status.TaskTimeoutMinutes)
	return nil
}

func cmdStart(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("start", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	var resp ActionResponse
	if err := ctx.Post("/queue/start", struct{}{}, &resp); err != nil {
		return fmt.Errorf("failed to start queue: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if resp.DryRun {
		fmt.Println("[DRY RUN] Queue processor would be started")
	} else if resp.Success {
		fmt.Println("Queue started successfully")
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
		return fmt.Errorf("failed to stop queue: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if resp.DryRun {
		fmt.Println("[DRY RUN] Queue processor would be stopped")
	} else if resp.Success {
		fmt.Println("Queue stopped successfully")
	} else {
		fmt.Printf("Failed to stop queue: %s\n", resp.Message)
	}
	return nil
}
