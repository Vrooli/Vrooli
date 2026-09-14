package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/cli-core/cliutil"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
)

// =============================================================================
// Maintenance Command Dispatcher
// =============================================================================

func (a *App) cmdMaintenance(args []string) error {
	if len(args) == 0 {
		return nil
	}

	switch args[0] {
	case "status", "begin", "drain", "resume":
		return a.maintenanceAdmission(args[0], args[1:])
	case "purge":
		return a.maintenancePurge(args[1:])
	case "help", "-h", "--help":
		return nil
	default:
		return fmt.Errorf("unknown maintenance subcommand: %s\n\nRun 'agent-manager maintenance help' for usage", args[0])
	}
}

type maintenanceStanding struct {
	Closed    bool            `json:"closed"`
	Revision  int64           `json:"revision"`
	Owner     string          `json:"owner"`
	Reason    string          `json:"reason"`
	Admitting int             `json:"admitting"`
	Remaining *int            `json:"remaining"`
	Drained   bool            `json:"drained"`
	Inventory json.RawMessage `json:"inventory"`
}

func (a *App) maintenanceAdmission(operation string, args []string) error {
	fs := flag.NewFlagSet("maintenance "+operation, flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	localOwner := fs.Bool("local-owner", false, "Explicit local operator authentication; unavailable inside an identified agent run")
	reason := fs.String("reason", "", "Reason for planned maintenance")
	revision := fs.Int64("revision", -1, "Exact closed revision returned by begin or status")
	timeout := fs.Duration("timeout", time.Minute, "Drain attachment timeout, 1s-120s; timeout preserves work and the fence")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected maintenance arguments")
	}
	if operation == "begin" && (strings.TrimSpace(*reason) == "" || len(*reason) > 512) {
		return fmt.Errorf("begin requires --reason with 1-512 bytes")
	}
	if operation == "resume" && *revision < 0 {
		return fmt.Errorf("resume requires --revision from maintenance status")
	}
	if operation == "drain" && (*timeout < time.Second || *timeout > 120*time.Second || *timeout%time.Second != 0) {
		return fmt.Errorf("--timeout must be whole seconds between 1s and 120s")
	}
	api := a.services.Maintenance.api
	if *localOwner {
		if operation == "status" {
			return fmt.Errorf("status does not require --local-owner")
		}
		owner, err := localOwnerAPI(api)
		if err != nil {
			return err
		}
		api = owner
	}
	method, path := "GET", "/api/v1/maintenance/admission"
	var payload []byte
	if operation != "status" {
		method = "POST"
		input := map[string]any{}
		switch operation {
		case "begin":
			path += "/enter"
			input["reason"] = *reason
		case "resume":
			path += "/resume"
			input["revision"] = *revision
		case "drain":
			path += "/wait"
			input["timeoutSeconds"] = int(*timeout / time.Second)
			api = api.WithTimeout(*timeout + 10*time.Second)
		}
		payload, _ = json.Marshal(input)
	}
	body, requestErr := api.Request(method, path, nil, payload)
	// The canonical client retains HTTP error bodies in APIError, not in the
	// returned bytes. Preserve that bounded owner evidence before returning the
	// nonzero status; inventory failure does not erase the closed revision.
	if len(body) == 0 && requestErr != nil {
		var apiErr *cliutil.APIError
		if errors.As(requestErr, &apiErr) {
			body = apiErr.RawResponse
		}
	}
	if len(body) > 256*1024 {
		return fmt.Errorf("maintenance response exceeds the 256 KiB evidence limit; owner standing is unobserved")
	}
	if *jsonOut && len(body) > 0 {
		cliutil.PrintJSON(body)
	}
	var state maintenanceStanding
	rawState := body
	if requestErr != nil {
		var envelope struct {
			State json.RawMessage `json:"state"`
		}
		if json.Unmarshal(body, &envelope) != nil || len(envelope.State) == 0 || string(envelope.State) == "null" {
			// Lock contention deliberately does not read the gate mutex. An
			// unobserved state must not be displayed as an open zero revision.
			return apiError(body, requestErr)
		}
		rawState = envelope.State
	}
	var observed struct {
		Closed   *bool  `json:"closed"`
		Revision *int64 `json:"revision"`
	}
	if err := json.Unmarshal(rawState, &observed); err != nil || observed.Closed == nil || observed.Revision == nil {
		if requestErr != nil {
			return apiError(body, requestErr)
		}
		return fmt.Errorf("maintenance response does not contain an observed admission fence")
	}
	if err := json.Unmarshal(rawState, &state); err != nil {
		if requestErr != nil {
			return apiError(body, requestErr)
		}
		return fmt.Errorf("decode maintenance standing: %w", err)
	}
	if !*jsonOut {
		remaining := "unobserved"
		if state.Remaining != nil {
			remaining = fmt.Sprint(*state.Remaining)
		}
		fmt.Printf("Admission closed=%t revision=%d admitting=%d remaining=%s drained=%t\n", state.Closed, state.Revision, state.Admitting, remaining, state.Drained)
		if state.Reason != "" {
			fmt.Printf("Reason: %s\n", state.Reason)
		}
		if len(state.Inventory) > 0 && string(state.Inventory) != "null" {
			fmt.Printf("Inventory: %s\n", state.Inventory)
		}
		if state.Closed {
			fmt.Printf("Resume: agent-manager maintenance resume --revision %d --local-owner\n", state.Revision)
		}
	}
	if requestErr != nil {
		return fmt.Errorf("maintenance attachment ended; fence and admitted work remain unchanged: %w", apiError(body, requestErr))
	}
	if operation == "drain" && !state.Drained {
		return fmt.Errorf("maintenance drain is not complete")
	}
	return nil
}

// localOwnerAPI exchanges explicit local human authority for an owner token.
// It is refused inside an identified agent run so an agent cannot elevate.
func localOwnerAPI(api *cliutil.APIClient) (*cliutil.APIClient, error) {
	if strings.TrimSpace(os.Getenv(cliutil.EnvIdentityToken)) != "" {
		return nil, fmt.Errorf("--local-owner is unavailable inside an identified agent run; use the granted credential")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	login, err := authn.ExchangeLocalMachinePrincipal(ctx)
	if err != nil {
		return nil, fmt.Errorf("local owner exchange unavailable: %w", err)
	}
	return api.WithToken(login.Tokens.AccessToken), nil
}

// =============================================================================
// Maintenance Purge
// =============================================================================

func (a *App) maintenancePurge(args []string) error {
	fs := flag.NewFlagSet("maintenance purge", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	pattern := fs.String("pattern", "", "Regex pattern to match (required)")
	targetsStr := fs.String("targets", "", "Comma-separated targets: profiles,tasks,runs (required)")
	dryRun := fs.Bool("dry-run", false, "Preview without deleting")
	force := fs.Bool("force", false, "Skip confirmation (for non-dry-run)")

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if *pattern == "" {
		return fmt.Errorf("--pattern is required")
	}

	if *targetsStr == "" {
		return fmt.Errorf("--targets is required (comma-separated: profiles,tasks,runs)")
	}

	// Parse targets
	targetStrs := strings.Split(*targetsStr, ",")
	targets := make([]apipb.PurgeTarget, 0, len(targetStrs))
	for _, t := range targetStrs {
		switch strings.TrimSpace(strings.ToLower(t)) {
		case "profiles":
			targets = append(targets, apipb.PurgeTarget_PURGE_TARGET_PROFILES)
		case "tasks":
			targets = append(targets, apipb.PurgeTarget_PURGE_TARGET_TASKS)
		case "runs":
			targets = append(targets, apipb.PurgeTarget_PURGE_TARGET_RUNS)
		default:
			return fmt.Errorf("invalid target: %s (valid: profiles, tasks, runs)", t)
		}
	}

	// Confirm if not dry-run and not forced
	if !*dryRun && !*force {
		fmt.Printf("Purge items matching pattern '%s' from %s? [y/N]: ", *pattern, *targetsStr)
		var confirm string
		_, _ = fmt.Scanln(&confirm)
		if strings.ToLower(confirm) != "y" && strings.ToLower(confirm) != "yes" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	req := &apipb.PurgeDataRequest{
		Pattern: *pattern,
		Targets: targets,
		DryRun:  *dryRun,
	}

	body, resp, err := a.services.Maintenance.Purge(req)
	if err != nil {
		return err
	}

	if *jsonOutput || resp == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	action := "Deleted"
	if resp.DryRun {
		action = "Would delete"
	}

	fmt.Printf("\nPurge Results (pattern: %s)\n", *pattern)
	fmt.Printf("%-10s  %s\n", "Target", "Matched → "+action)
	fmt.Printf("%-10s  %s\n", strings.Repeat("-", 10), strings.Repeat("-", 20))

	if resp.Matched != nil && resp.Deleted != nil {
		if resp.Matched.Profiles > 0 || resp.Deleted.Profiles > 0 {
			fmt.Printf("%-10s  %d → %d\n", "Profiles", resp.Matched.Profiles, resp.Deleted.Profiles)
		}
		if resp.Matched.Tasks > 0 || resp.Deleted.Tasks > 0 {
			fmt.Printf("%-10s  %d → %d\n", "Tasks", resp.Matched.Tasks, resp.Deleted.Tasks)
		}
		if resp.Matched.Runs > 0 || resp.Deleted.Runs > 0 {
			fmt.Printf("%-10s  %d → %d\n", "Runs", resp.Matched.Runs, resp.Deleted.Runs)
		}
	}

	if resp.DryRun {
		fmt.Println("\n(Dry run - no items were deleted)")
	}

	return nil
}
