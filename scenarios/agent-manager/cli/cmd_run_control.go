// Responsibility: retain cmd runs declarations within their original package.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"os"
	"strings"
	"time"
)

func (a *App) runStop(args []string) error {
	fs := flag.NewFlagSet("run stop", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	// Parse with positional ID first
	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if id == "" {
		return fmt.Errorf("usage: agent-manager run stop <id>")
	}

	body, resp, err := a.services.Runs.Stop(id)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	changes := []string{fmt.Sprintf("run_id=%s", id)}
	nextCommandID := id
	if resp != nil && resp.Run != nil {
		nextCommandID = resp.Run.Id
		changes = append(changes,
			fmt.Sprintf("status=%s", formatEnumValue(resp.Run.Status, "RUN_STATUS_", "_")),
			fmt.Sprintf("phase=%s", formatEnumValue(resp.Run.Phase, "RUN_PHASE_", "_")),
		)
	} else if resp != nil && resp.Status != "" {
		changes = append(changes, fmt.Sprintf("status=%s", resp.Status))
	}

	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result:      []string{"Run stop requested"},
		Changes:     changes,
		NextCommand: []string{fmt.Sprintf("agent-manager run get %s", nextCommandID)},
	})
}

func (a *App) runStopByTag(args []string) error {
	fs := flag.NewFlagSet("run stop-by-tag", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	// Parse with positional tag first
	var tag string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		tag = args[0]
		args = args[1:]
	}

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if tag == "" {
		return fmt.Errorf("usage: agent-manager run stop-by-tag <tag>")
	}

	body, resp, err := a.services.Runs.StopByTag(tag)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	changes := []string{fmt.Sprintf("tag=%s", tag)}
	nextCommand := fmt.Sprintf("agent-manager run get-by-tag %s", tag)
	if resp != nil && resp.Run != nil {
		changes = append(changes,
			fmt.Sprintf("run_id=%s", resp.Run.Id),
			fmt.Sprintf("status=%s", formatEnumValue(resp.Run.Status, "RUN_STATUS_", "_")),
			fmt.Sprintf("phase=%s", formatEnumValue(resp.Run.Phase, "RUN_PHASE_", "_")),
		)
		nextCommand = fmt.Sprintf("agent-manager run get %s", resp.Run.Id)
	} else if resp != nil && resp.Status != "" {
		changes = append(changes, fmt.Sprintf("status=%s", resp.Status))
	}

	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result:      []string{"Run stop requested"},
		Changes:     changes,
		NextCommand: []string{nextCommand},
	})
}

// =============================================================================
// Run Stop All
// =============================================================================

func (a *App) runStopAll(args []string) error {
	fs := flag.NewFlagSet("run stop-all", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	tagPrefix := fs.String("tag-prefix", "", "Only stop runs with this tag prefix")
	force := fs.Bool("force", false, "Force termination even if graceful stop fails")

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	req := &apipb.StopAllRunsRequest{Force: *force}
	if *tagPrefix != "" {
		req.TagPrefix = protoString(*tagPrefix)
	}
	body, result, err := a.services.Runs.StopAll(req)
	if err != nil {
		return err
	}

	if *jsonOutput || result == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if result != nil {
		fmt.Printf("Stopped:  %d\n", result.StoppedCount)
		if len(result.Failures) > 0 {
			fmt.Printf("Failed:   %d\n", len(result.Failures))
			failedIDs := make([]string, 0, len(result.Failures))
			for _, failure := range result.Failures {
				if failure == nil {
					continue
				}
				failedIDs = append(failedIDs, failure.RunId)
			}
			if len(failedIDs) > 0 {
				fmt.Printf("Failed IDs: %v\n", failedIDs)
			}
		}
	}
	return nil
}

// =============================================================================
// Run Quiesce (Baseline Modes promote drain)
// =============================================================================

func (a *App) runQuiesce(args []string) error {
	fs := flag.NewFlagSet("run quiesce", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	scenario := fs.String("scenario", "", "Target scenario slug to quiesce (required)")
	scopePrefix := fs.String("scope-prefix", "", "Override the working-tree scope (default scenarios/<scenario>)")
	tagPrefix := fs.String("tag-prefix", "", "Also enumerate in-flight runs by this tag prefix (whole-repo runs)")
	excludeRun := fs.String("exclude-run", "", "The promoting run's own ID, excluded from the drain set")
	timeout := fs.String("timeout", "", "Max wait for in-flight runs to terminate (e.g. 5m)")
	force := fs.Bool("force", false, "On timeout, cancel survivors instead of aborting")

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *scenario == "" {
		return fmt.Errorf("--scenario is required")
	}

	req := &apipb.QuiesceScenarioRequest{Scenario: *scenario, Force: *force}
	if *scopePrefix != "" {
		req.ScopePrefix = protoString(*scopePrefix)
	}
	if *tagPrefix != "" {
		req.TagPrefix = protoString(*tagPrefix)
	}
	if *excludeRun != "" {
		req.ExcludeRunId = protoString(*excludeRun)
	}
	if *timeout != "" {
		req.Timeout = protoString(*timeout)
	}

	body, result, err := a.services.Runs.Quiesce(req)
	if err != nil {
		return err
	}

	if *jsonOutput || result == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if result.Drained {
		fmt.Printf("✓ %s drained — safe to promote\n", result.Scenario)
	} else if result.Aborted {
		fmt.Printf("✗ %s NOT drained — %d run(s) still in-flight\n", result.Scenario, len(result.InFlight))
	}
	if len(result.Cancelled) > 0 {
		fmt.Printf("Cancelled: %d run(s)\n", len(result.Cancelled))
	}
	for _, ref := range result.InFlight {
		fmt.Printf("  in-flight: %s [%s] %s\n", ref.Id, ref.Status, ref.Tag)
	}
	if result.Reason != "" {
		fmt.Printf("→ %s\n", result.Reason)
	}
	return nil
}

// =============================================================================
// Run Approve
// =============================================================================

func (a *App) runApprove(args []string) error {
	fs := flag.NewFlagSet("run approve", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	actor := fs.String("actor", "", "Who is approving")
	commitMsg := fs.String("commit-msg", "", "Commit message for changes")
	force := fs.Bool("force", false, "Force approval despite conflicts")

	// Parse with positional ID first
	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if id == "" {
		return fmt.Errorf("usage: agent-manager run approve <id> [options]")
	}

	req := &apipb.ApproveRunRequest{
		RunId: id,
		Force: *force,
	}
	if trimmed := strings.TrimSpace(*actor); trimmed != "" {
		req.Actor = protoString(trimmed)
	}
	if *commitMsg != "" {
		req.CommitMsg = protoString(*commitMsg)
	}

	body, result, err := a.services.Runs.Approve(id, req)
	if err != nil {
		return err
	}

	if *jsonOutput || result == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if result != nil && result.Success {
		fmt.Printf("Approved run: %s\n", id)
		fmt.Printf("Applied: %d files\n", result.FilesApplied)
		if result.CommitHash != "" {
			fmt.Printf("Commit: %s\n", result.CommitHash)
		}
	} else {
		message := ""
		if result != nil {
			message = result.Message
		}
		fmt.Printf("Approval failed: %s\n", message)
	}
	return nil
}

// =============================================================================
// Run Reject
// =============================================================================

func (a *App) runReject(args []string) error {
	fs := flag.NewFlagSet("run reject", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	actor := fs.String("actor", "", "Who is rejecting")
	reason := fs.String("reason", "", "Reason for rejection")

	// Parse with positional ID first
	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if id == "" {
		return fmt.Errorf("usage: agent-manager run reject <id> [options]")
	}

	req := &apipb.RejectRunRequest{
		RunId:  id,
		Reason: *reason,
	}
	if trimmed := strings.TrimSpace(*actor); trimmed != "" {
		req.Actor = protoString(trimmed)
	}

	body, err := a.services.Runs.Reject(id, req)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Rejected run: %s\n", id)
	return nil
}

func (a *App) runDelete(args []string) error {
	fs := flag.NewFlagSet("run delete", flag.ContinueOnError)
	force := fs.Bool("force", false, "Skip confirmation")

	// Parse with positional ID first
	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if id == "" {
		return fmt.Errorf("usage: agent-manager run delete <id>")
	}

	if !*force {
		fmt.Printf("Delete run %s? [y/N]: ", id)
		var confirm string
		_, _ = fmt.Scanln(&confirm)
		if strings.ToLower(confirm) != "y" && strings.ToLower(confirm) != "yes" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	if err := a.services.Runs.Delete(id); err != nil {
		return err
	}

	fmt.Printf("Deleted run: %s\n", id)
	return nil
}

// =============================================================================
// Run Continue
// =============================================================================

func (a *App) runContinue(args []string) error {
	fs := flag.NewFlagSet("run continue", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	message := fs.String("message", "", "Follow-up message (required); a running interactive session receives it as its next user message")
	idempotencyKey := fs.String("idempotency-key", "", "Replay-safe continuation key; retain for retries of the same request")
	reinstallGoal := fs.Bool("reinstall-goal", false, "Reinstall the harness-native goal before the follow-up message")

	// Parse with positional ID first
	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if id == "" {
		return fmt.Errorf("usage: agent-manager run continue <id> --message <message> [--idempotency-key <key>] [--reinstall-goal]")
	}

	if *message == "" {
		return fmt.Errorf("--message is required")
	}

	req := &domainpb.ContinueRunRequest{
		RunId:          id,
		Message:        *message,
		IdempotencyKey: *idempotencyKey,
		ReinstallGoal:  *reinstallGoal,
	}

	body, run, err := a.services.Runs.Continue(id, req)
	if err != nil {
		return err
	}

	if *jsonOutput || run == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if run.Status == domainpb.RunStatus_RUN_STATUS_RUNNING {
		fmt.Printf("Transport accepted into running session: %s (provider consumption unknown; inspect later provider events)\n", run.Id)
		return nil
	}
	fmt.Printf("Continued run: %s (status: %s)\n", run.Id, formatEnumValue(run.Status, "RUN_STATUS_", "_"))
	return nil
}

// runPark parks a run on externally-owned async work. Invoked from inside an
// agent-manager-controlled run; it authenticates with the run's identity token
// (VROOLI_AGENT_IDENTITY_TOKEN by default) so the server can confirm the caller
// owns the run it is parking.
func (a *App) runPark(args []string) error {
	fs := flag.NewFlagSet("run park", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	producer := fs.String("producer", "", "Producer that resolves the await (e.g. test-genie, git-control-tower) (required)")
	key := fs.String("key", "", "Producer-scoped identifier of the awaited work (required)")
	deadlineUnix := fs.Int64("deadline-unix", 0, "Optional wait deadline as a Unix timestamp (seconds); 0 = default TTL")
	timeout := fs.Duration("timeout", 0, "Wake after this long even if the work is unresolved (timer wake), e.g. 15m; overrides --deadline-unix")
	identityToken := fs.String("identity-token", "", "Owning run's identity token (defaults to $"+cliutil.EnvIdentityToken+")")

	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if id == "" {
		return fmt.Errorf("usage: agent-manager run park <id> --producer <p> --key <k>")
	}
	if *producer == "" || *key == "" {
		return fmt.Errorf("--producer and --key are required")
	}
	if *timeout < 0 {
		return fmt.Errorf("--timeout must be positive")
	}
	if *timeout > 0 {
		*deadlineUnix = time.Now().Add(*timeout).Unix()
	}

	token := *identityToken
	if token == "" {
		token = os.Getenv(cliutil.EnvIdentityToken)
	}
	if token == "" {
		return fmt.Errorf("no identity token: set --identity-token or run inside an agent-manager run ($%s)", cliutil.EnvIdentityToken)
	}

	req := &domainpb.ParkRunRequest{
		RunId:         id,
		Producer:      *producer,
		Key:           *key,
		DeadlineUnix:  *deadlineUnix,
		IdentityToken: token,
	}

	body, resp, err := a.services.Runs.Park(id, req)
	if err != nil {
		return err
	}
	if *jsonOutput || resp == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	// The message is the clean turn-ending tool-result the agent should see.
	fmt.Println(resp.Message)
	return nil
}

// runWake wakes a parked run with a result injected as its next turn. This is an
// ops/manual-recovery verb; normal wake is driven by agent-manager's waiter.
func (a *App) runWake(args []string) error {
	fs := flag.NewFlagSet("run wake", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	result := fs.String("result", "", "Awaited result injected as the next turn")
	timedOut := fs.Bool("timed-out", false, "Frame the result as a park-deadline timeout")
	key := fs.String("key", "", "Wake every run parked on this await key instead of one run ID")
	producer := fs.String("producer", "children", "Await producer the key belongs to (with --key)")

	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if (id == "") == (*key == "") {
		return fmt.Errorf("usage: agent-manager run wake (<id> | --key <key> [--producer children]) [--result <r>] [--timed-out]")
	}
	if *key != "" {
		return a.runWakeByKey(strings.TrimSpace(*producer), strings.TrimSpace(*key), *result, *timedOut, *jsonOutput)
	}

	req := &domainpb.WakeRunRequest{
		RunId:    id,
		Result:   *result,
		TimedOut: *timedOut,
	}

	body, resp, err := a.services.Runs.Wake(id, req)
	if err != nil {
		return err
	}
	if *jsonOutput || resp == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if !resp.Success {
		fmt.Println("Run was not parked (no-op); wake is idempotent")
		return nil
	}
	if resp.Run != nil {
		fmt.Printf("Woke run: %s (status: %s)\n", resp.Run.Id, formatEnumValue(resp.Run.Status, "RUN_STATUS_", "_"))
	} else {
		fmt.Println("Wake requested")
	}
	return nil
}

// runWakeByKey wakes every run parked on producer/key. It is the friction
// wake: an epoch check or operator names the orchestrator's key without
// knowing which run currently holds it. The server matches the handle.
func (a *App) runWakeByKey(producer, key, result string, timedOut, jsonOutput bool) error {
	woken, err := a.services.Runs.WakeByKey(&domainpb.WakeParkedRunsRequest{Producer: producer, Key: key, Result: result, TimedOut: timedOut})
	if err != nil {
		return err
	}
	if jsonOutput {
		data, err := json.Marshal(map[string]any{"key": key, "woken": woken})
		if err != nil {
			return err
		}
		cliutil.PrintJSON(data)
		return nil
	}
	if len(woken) == 0 {
		fmt.Printf("No run is parked on key %s (no-op)\n", key)
		return nil
	}
	fmt.Printf("Woke %d run(s) parked on key %s: %s\n", len(woken), key, strings.Join(woken, ", "))
	return nil
}

// runAwaitResult re-fetches a run's most recently resolved await result without
// re-running the blocking producer. This is the deterministic retrieval path a
// woken agent uses if it did not receive — or wants to re-read — the result.
func (a *App) runAwaitResult(args []string) error {
	fs := flag.NewFlagSet("run await-result", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if id == "" {
		return fmt.Errorf("usage: agent-manager run await-result <id>")
	}

	body, resp, err := a.services.Runs.AwaitResult(id)
	if err != nil {
		return err
	}
	if *jsonOutput || resp == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if !resp.Found {
		fmt.Println("No awaited result recorded for this run.")
		return nil
	}
	if resp.Key != "" {
		fmt.Printf("Awaited: %s\n", resp.Key)
	}
	if resp.ResolvedAt != "" {
		fmt.Printf("Resolved at: %s\n", resp.ResolvedAt)
	}
	fmt.Printf("\nResult:\n%s\n", resp.Result)
	return nil
}

func (a *App) runRecover(args []string) error {
	fs := flag.NewFlagSet("run recover", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if id == "" {
		return fmt.Errorf("usage: agent-manager run recover <id>")
	}

	body, resp, err := a.services.Runs.Recover(id)
	if err != nil {
		return err
	}
	if *jsonOutput || resp == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Recovered:       %t\n", resp.Recovered)
	fmt.Printf("Idempotent:      %t\n", resp.Idempotent)
	if resp.Message != "" {
		fmt.Printf("Message:         %s\n", resp.Message)
	}
	if resp.Run != nil {
		fmt.Printf("Run ID:          %s\n", resp.Run.Id)
		fmt.Printf("Status:          %s\n", formatEnumValue(resp.Run.Status, "RUN_STATUS_", "_"))
	}
	return nil
}

// =============================================================================
// Run Investigate
// =============================================================================
