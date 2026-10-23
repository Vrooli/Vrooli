package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-manager/internal/goalhome"

	"github.com/vrooli/cli-core/cliutil"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// effortPark is `agent-manager effort park <goal home> [--run <id>] [--timeout D]`:
// the orchestrator's park, with its length chosen by goal state
// (goalhome.DecidePark). It allows up to 1h while a direct child can still execute,
// up to 72h while ## Needs operator holds an item, and otherwise refuses with
// exit 4. It parks through the same server call as `run park` (producer
// children, keyed by the run). The server applies the same goalhome.DecidePark
// to a raw `run park` (report-only unless VROOLI_EFFORT_PARK_ENFORCE is on).
func (a *App) effortPark(args []string) error {
	fs := flag.NewFlagSet("effort park", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	runIDArg := fs.String("run", "", "The calling orchestrator's run ID (defaults to the verified identity token)")
	timeout := fs.Duration("timeout", 0, "Wake after this long; at most the rule's maximum (default: the maximum)")
	identityToken := fs.String("identity-token", "", "Owning run's identity token (defaults to $"+cliutil.EnvIdentityToken+")")
	dir, args := goalHomeArg(args)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if dir == "" || fs.NArg() != 0 {
		return fmt.Errorf("usage: agent-manager effort park <goal home> [--run <run-id>] [--timeout 1h] [--json]")
	}
	if *timeout < 0 {
		return fmt.Errorf("--timeout must be positive")
	}
	token := *identityToken
	if token == "" {
		token = os.Getenv(cliutil.EnvIdentityToken)
	}
	if token == "" {
		return fmt.Errorf("no identity token: set --identity-token or run inside an agent-manager run ($%s)", cliutil.EnvIdentityToken)
	}
	verified, err := (cliutil.IdentityEnv{Token: token}).VerifyIdentityWithAPIClient(a.services.Runs.api)
	if err != nil {
		return fmt.Errorf("verify park caller identity: %w", err)
	}
	if verified == nil || !verified.Valid || verified.Claims == nil || strings.TrimSpace(verified.Claims.RunID) == "" {
		reason := "identity token is not valid"
		if verified != nil && verified.Error != "" {
			reason = verified.Error
		}
		return fmt.Errorf("park caller identity: %s", reason)
	}
	callerRunID := strings.TrimSpace(verified.Claims.RunID)
	if suppliedRunID := strings.TrimSpace(*runIDArg); suppliedRunID != "" && suppliedRunID != callerRunID {
		return fmt.Errorf("--run %q does not match verified caller run %q", suppliedRunID, callerRunID)
	}
	runID := callerRunID
	_, targetRun, err := a.services.Runs.Get(runID)
	if err != nil {
		return fmt.Errorf("get park target run %s: %w", runID, err)
	}
	if targetRun == nil {
		return fmt.Errorf("get park target run %s returned no run", runID)
	}
	if targetRun.Id != runID {
		return fmt.Errorf("get park target run %s returned mismatched run ID %q", runID, targetRun.Id)
	}
	queue, err := os.ReadFile(filepath.Join(dir, goalhome.QueueFile))
	if err != nil {
		return fmt.Errorf("goal home %s: %w", dir, err)
	}
	_, children, err := a.services.Runs.ListChildren(runID, 1000)
	if err != nil {
		return fmt.Errorf("list direct children of %s: %w", runID, err)
	}
	var live []string
	for _, child := range children {
		if child != nil && isParkExecutionChildStatus(child.Status) {
			live = append(live, child.Id)
		}
	}
	decision := goalhome.DecidePark(live, goalhome.ParseQueue(queue), *timeout)
	result := struct {
		goalhome.ParkDecision
		Max     string          `json:"max,omitempty"`
		Timeout string          `json:"timeout,omitempty"`
		Park    json.RawMessage `json:"park,omitempty"`
	}{ParkDecision: decision}
	if decision.Max > 0 {
		result.Max = decision.Max.String()
	}
	if !decision.Allowed {
		if *jsonOut {
			return printParkResult(result, exitCodeError{code: exitRefused})
		}
		fmt.Printf("PARK_REFUSED %s: %s\n", decision.Rule, decision.Reason)
		return exitCodeError{code: exitRefused}
	}
	result.Timeout = decision.Timeout.String()
	if !*jsonOut {
		// Printed before parking: the park may end this turn.
		fmt.Printf("Parking %s for %s (%s): %s\n", runID, decision.Timeout, decision.Rule, decision.Reason)
	}
	body, resp, err := a.services.Runs.Park(runID, &domainpb.ParkRunRequest{
		RunId:         runID,
		Producer:      "children",
		Key:           runID,
		DeadlineUnix:  time.Now().Add(decision.Timeout).Unix(),
		IdentityToken: token,
	})
	if err != nil {
		return err
	}
	if *jsonOut || resp == nil {
		result.Park = body
		return printParkResult(result, nil)
	}
	fmt.Println(resp.Message)
	return nil
}

func printParkResult(result any, exit error) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(data)
	return exit
}

// isParkExecutionChildStatus identifies children whose executor may still hang.
// NEEDS_REVIEW remains nonterminal for approval and continuation, but its
// worker has exited and must not trigger the live-child park backstop.
func isParkExecutionChildStatus(status domainpb.RunStatus) bool {
	switch status {
	case domainpb.RunStatus_RUN_STATUS_PENDING,
		domainpb.RunStatus_RUN_STATUS_STARTING,
		domainpb.RunStatus_RUN_STATUS_RUNNING,
		domainpb.RunStatus_RUN_STATUS_PARKED:
		return true
	default:
		return false
	}
}
