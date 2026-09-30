package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

// =============================================================================
// Storage: Agent Manager's own SQLite footprint
// =============================================================================

// storagePollInterval spaces health reads while waiting for a compaction.
var storagePollInterval = 5 * time.Second

func (a *App) cmdStorage(args []string) error {
	if len(args) == 0 {
		return nil
	}
	switch args[0] {
	case "status":
		return a.storageStatus(args[1:])
	case "reclaim":
		return a.storageReclaim(args[1:])
	case "compact":
		return a.storageCompact(args[1:])
	case "help", "-h", "--help":
		return nil
	default:
		return fmt.Errorf("unknown storage subcommand: %s\n\nRun 'agent-manager storage help' for usage", args[0])
	}
}

type storageStats struct {
	Path          string  `json:"path"`
	PageCount     int64   `json:"pageCount"`
	FreelistCount int64   `json:"freelistCount"`
	AutoVacuum    string  `json:"autoVacuum"`
	FileBytes     int64   `json:"fileBytes"`
	WALBytes      int64   `json:"walBytes"`
	LiveBytes     int64   `json:"liveBytes"`
	FreeBytes     int64   `json:"freeBytes"`
	BudgetBytes   int64   `json:"budgetBytes"`
	UsageRatio    float64 `json:"usageRatio"`
	Level         string  `json:"level"`
	Action        string  `json:"action"`
}

type storageCompaction struct {
	State   string `json:"state"`
	Actor   string `json:"actor"`
	Reason  string `json:"reason"`
	Error   string `json:"error"`
	Receipt *struct {
		Before          storageStats `json:"before"`
		After           storageStats `json:"after"`
		ReclaimedBytes  int64        `json:"reclaimedBytes"`
		AutoVacuumAfter string       `json:"autoVacuumAfter"`
		QuickCheck      string       `json:"quickCheck"`
		VacuumMillis    int64        `json:"vacuumMillis"`
		TotalMillis     int64        `json:"totalMillis"`
	} `json:"receipt"`
}

type storageHealthView struct {
	Storage    storageStats      `json:"storage"`
	Guidance   string            `json:"guidance"`
	Compaction storageCompaction `json:"compaction"`
}

func (a *App) storageHealth() ([]byte, storageHealthView, error) {
	var view storageHealthView
	body, err := a.services.Maintenance.api.Request("GET", "/api/v1/storage/health", nil, nil)
	if err != nil {
		return body, view, apiError(body, err)
	}
	if err := json.Unmarshal(body, &view); err != nil {
		return body, view, fmt.Errorf("decode storage health: %w", err)
	}
	return body, view, nil
}

func (a *App) storageStatus(args []string) error {
	fs := flag.NewFlagSet("storage status", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, view, err := a.storageHealth()
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	st := view.Storage
	fmt.Printf("File: %s (live %s, free %s, WAL %s) auto_vacuum=%s\n", humanBytes(st.FileBytes), humanBytes(st.LiveBytes), humanBytes(st.FreeBytes), humanBytes(st.WALBytes), st.AutoVacuum)
	if st.BudgetBytes > 0 {
		fmt.Printf("Budget: %s used %.1f%% level=%s action=%s\n", humanBytes(st.BudgetBytes), st.UsageRatio*100, st.Level, st.Action)
	} else {
		fmt.Printf("Budget: undeclared level=%s action=%s\n", st.Level, st.Action)
	}
	if view.Guidance != "" {
		fmt.Printf("Next: %s\n", view.Guidance)
	}
	fmt.Printf("Compaction: %s", view.Compaction.State)
	if view.Compaction.Error != "" {
		fmt.Printf(" (%s)", view.Compaction.Error)
	}
	fmt.Println()
	return nil
}

func (a *App) storageReclaim(args []string) error {
	fs := flag.NewFlagSet("storage reclaim", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	dryRun := fs.Bool("dry-run", false, "Preview without changing the file")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]bool{"dry_run": *dryRun})
	body, err := a.services.Maintenance.api.WithTimeout(3*time.Minute).Request("POST", "/api/v1/storage/reclaim", nil, payload)
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	var receipt struct {
		DryRun         bool         `json:"dryRun"`
		Action         string       `json:"action"`
		Before         storageStats `json:"before"`
		ReclaimedBytes int64        `json:"reclaimedBytes"`
		Complete       bool         `json:"complete"`
		DurationMillis int64        `json:"durationMillis"`
		Guidance       string       `json:"guidance"`
	}
	if err := json.Unmarshal(body, &receipt); err != nil {
		return fmt.Errorf("decode reclaim receipt: %w", err)
	}
	fmt.Printf("Action: %s free=%s dry_run=%t\n", receipt.Action, humanBytes(receipt.Before.FreeBytes), receipt.DryRun)
	if !receipt.DryRun {
		fmt.Printf("Reclaimed %s in %dms complete=%t\n", humanBytes(receipt.ReclaimedBytes), receipt.DurationMillis, receipt.Complete)
	}
	if receipt.Guidance != "" {
		fmt.Printf("Next: %s\n", receipt.Guidance)
	}
	return nil
}

func (a *App) storageCompact(args []string) error {
	fs := flag.NewFlagSet("storage compact", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	localOwner := fs.Bool("local-owner", false, "Explicit local operator authentication; unavailable inside an identified agent run")
	reason := fs.String("reason", "", "Reason for the compaction")
	noWait := fs.Bool("no-wait", false, "Return once the compaction starts")
	timeout := fs.Duration("timeout", 45*time.Minute, "How long to wait for the result")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*reason) == "" || len(*reason) > 512 {
		return fmt.Errorf("compact requires --reason with 1-512 bytes")
	}
	api := a.services.Maintenance.api
	if *localOwner {
		owner, err := localOwnerAPI(api)
		if err != nil {
			return err
		}
		api = owner
	}
	payload, _ := json.Marshal(map[string]string{"reason": *reason})
	body, err := api.Request("POST", "/api/v1/storage/compact", nil, payload)
	if err != nil {
		return apiError(body, err)
	}
	if *noWait {
		if *jsonOut {
			cliutil.PrintJSON(body)
		} else {
			fmt.Println("Compaction started; follow it with: agent-manager storage status")
		}
		return nil
	}
	deadline := time.Now().Add(*timeout)
	for {
		time.Sleep(storagePollInterval)
		raw, view, err := a.storageHealth()
		if err != nil {
			if time.Now().After(deadline) {
				return fmt.Errorf("compaction result unobserved before timeout: %w", err)
			}
			continue
		}
		switch view.Compaction.State {
		case "succeeded":
			if *jsonOut {
				cliutil.PrintJSON(raw)
				return nil
			}
			if r := view.Compaction.Receipt; r != nil {
				fmt.Printf("Compaction succeeded: %s -> %s (reclaimed %s) auto_vacuum=%s quick_check=%s vacuum=%s total=%s\n",
					humanBytes(r.Before.FileBytes+r.Before.WALBytes), humanBytes(r.After.FileBytes+r.After.WALBytes), humanBytes(r.ReclaimedBytes),
					r.AutoVacuumAfter, r.QuickCheck, time.Duration(r.VacuumMillis)*time.Millisecond, time.Duration(r.TotalMillis)*time.Millisecond)
			}
			return nil
		case "failed":
			if *jsonOut {
				cliutil.PrintJSON(raw)
			}
			return fmt.Errorf("compaction failed: %s", view.Compaction.Error)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("compaction still %s after %s; follow it with agent-manager storage status", view.Compaction.State, *timeout)
		}
	}
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
