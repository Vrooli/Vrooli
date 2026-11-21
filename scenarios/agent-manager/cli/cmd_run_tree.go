// Responsibility: the nested-run commands — `run wait` (one run, or a parent's
// children) and `run tree`.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"agent-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
)

// `run wait` exit codes. 0 means the awaited condition held; 1 is any other
// error (bad arguments, unknown run, refusal).
const (
	// exitWaitTimedOut: --timeout elapsed and the run (or every child) is
	// still live. The work continues; wait again or inspect it.
	exitWaitTimedOut = 5
	// exitAgentManagerUnreachable: Agent Manager did not answer. Do not stall
	// on it: continue the work with the harness's native sub-agents.
	exitAgentManagerUnreachable = 6
)

// serverWaitBound is the longest single server-side wait the CLI requests
// (the API caps it at 15 minutes); longer waits are several calls.
const serverWaitBound = 10 * time.Minute

// unreachableGrace is how long a wait that has already reached Agent Manager
// keeps retrying after it stops answering, covering an ordinary restart.
// A first call that cannot reach it exits at once.
const unreachableGrace = 2 * time.Minute

// waitClock is replaced by tests.
var waitClock = struct {
	now   func() time.Time
	sleep func(time.Duration)
}{now: time.Now, sleep: time.Sleep}

// agentManagerUnreachable reports a transport failure or a gateway or
// availability status: the API did not serve the request.
func agentManagerUnreachable(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *cliutil.APIError
	if !errors.As(err, &apiErr) {
		return true
	}
	switch apiErr.StatusCode {
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}
	return false
}

// boundedWait repeats one server-side wait until it reports done or the
// overall timeout elapses (0 waits without an overall limit). It returns
// timedOut when the overall timeout elapsed first.
func boundedWait(timeout time.Duration, call func(bound time.Duration) (done bool, err error)) (timedOut bool, err error) {
	start := waitClock.now()
	reached := false
	var unreachableSince time.Time
	retry := 2 * time.Second
	for {
		bound := serverWaitBound
		if timeout > 0 {
			remaining := timeout - waitClock.now().Sub(start)
			if remaining < time.Second {
				return true, nil
			}
			bound = min(bound, remaining)
		}
		called := waitClock.now()
		done, err := call(bound.Round(time.Second))
		switch {
		case err == nil:
			reached, unreachableSince, retry = true, time.Time{}, 2*time.Second
			if done {
				return false, nil
			}
			// A server wait returns early only when it is done; never turn
			// an early "not yet" into a hot loop.
			if waitClock.now().Sub(called) < time.Second {
				waitClock.sleep(time.Second)
			}
		case !agentManagerUnreachable(err) || !reached:
			return false, err
		default:
			if unreachableSince.IsZero() {
				unreachableSince = waitClock.now()
			}
			if waitClock.now().Sub(unreachableSince) >= unreachableGrace {
				return false, err
			}
			waitClock.sleep(retry)
			retry = min(2*retry, 30*time.Second)
		}
	}
}

// printWaitResult prints a typed wait response as proto JSON or text. The
// children response keeps empty lists, so `ended` and `active` are always
// arrays for a reader, as in the park wake payload.
func printWaitResult(jsonOutput bool, message proto.Message, text func()) error {
	if !jsonOutput {
		text()
		return nil
	}
	options := cliProtoMarshalOptions
	if _, children := message.(*apipb.WaitRunChildrenResponse); children {
		options.EmitUnpopulated = true
	}
	body, err := options.Marshal(message)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

// reportUnreachable names the fallback; Agent Manager is never required.
func reportUnreachable(jsonOutput bool, err error) error {
	detail := strings.TrimSpace(err.Error())
	if jsonOutput {
		body, _ := json.Marshal(map[string]string{
			"outcome":  "agent_manager_unreachable",
			"error":    detail,
			"fallback": "continue with the harness's native sub-agents (Codex spawn_agent with fork_turns=none; Claude Code Agent tool) using the same brief",
		})
		cliutil.PrintJSON(body)
	}
	fmt.Fprintf(os.Stderr, "Agent Manager is unreachable (%s). Do not wait on it: continue with the harness's native sub-agents using the same brief.\n", detail)
	return exitCodeError{code: exitAgentManagerUnreachable}
}

func (a *App) runWait(args []string) error {
	fs := flag.NewFlagSet("run wait", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	children := fs.String("children", "", "Wait for a direct child of this parent run to settle instead of one run")
	timeout := fs.Duration("timeout", 30*time.Minute, "Overall wait bound; 0 waits until the condition holds")
	sinceFlag := fs.String("since", "", "With --children: report children that settled at or after this RFC3339 time (pass the previous observed_at)")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	id := fs.Arg(0)
	if fs.NArg() > 1 || (id == "") == (*children == "") {
		return fmt.Errorf("usage: agent-manager run wait <run-id> | --children <parent-run-id> [--since RFC3339] [--timeout 30m] [--json]")
	}
	if *timeout < 0 {
		return fmt.Errorf("--timeout must not be negative")
	}
	if *children == "" {
		if *sinceFlag != "" {
			return fmt.Errorf("--since applies only with --children")
		}
		if err := requireUUIDFlag("run id", id); err != nil {
			return err
		}
		return a.waitForRun(id, *timeout, *jsonOutput)
	}
	if err := requireUUIDFlag("--children", *children); err != nil {
		return err
	}
	since := waitClock.now()
	if *sinceFlag != "" {
		parsed, err := time.Parse(time.RFC3339Nano, *sinceFlag)
		if err != nil {
			return fmt.Errorf("--since %q is not an RFC3339 time: %w", *sinceFlag, err)
		}
		since = parsed
	}
	return a.waitForChildren(*children, since, *timeout, *jsonOutput)
}

func (a *App) waitForRun(id string, timeout time.Duration, jsonOutput bool) error {
	var last *apipb.WaitRunResponse
	timedOut, err := boundedWait(timeout, func(bound time.Duration) (bool, error) {
		response, err := a.services.Runs.Wait(id, bound)
		if err != nil {
			return false, err
		}
		last = response
		return response.GetOutcome() != "timed_out", nil
	})
	if err != nil {
		if agentManagerUnreachable(err) {
			return reportUnreachable(jsonOutput, err)
		}
		return apiError(nil, err)
	}
	if timedOut && last != nil {
		last.Outcome = "timed_out"
	}
	if err := printWaitResult(jsonOutput, last, func() {
		run := last.GetRun()
		fmt.Printf("Run %s: %s (outcome %s)\n", run.GetId(), formatEnumValue(run.GetStatus(), "RUN_STATUS_", "_"), last.GetOutcome())
		if run.GetStopReason() != "" {
			fmt.Printf("Stop reason: %s\n", run.GetStopReason())
		}
	}); err != nil {
		return err
	}
	if last.GetOutcome() == "timed_out" {
		return exitCodeError{code: exitWaitTimedOut}
	}
	return nil
}

func (a *App) waitForChildren(parentID string, since time.Time, timeout time.Duration, jsonOutput bool) error {
	var last *apipb.WaitRunChildrenResponse
	timedOut, err := boundedWait(timeout, func(bound time.Duration) (bool, error) {
		// Every call keeps the original instant: a child that settles
		// between calls is still reported.
		response, err := a.services.Runs.WaitChildren(parentID, since, bound)
		if err != nil {
			return false, err
		}
		last = response
		return response.GetWakeReason() != "timer", nil
	})
	if err != nil {
		if agentManagerUnreachable(err) {
			return reportUnreachable(jsonOutput, err)
		}
		return apiError(nil, err)
	}
	if timedOut && last != nil {
		last.WakeReason = "timer"
	}
	if err := printWaitResult(jsonOutput, last, func() {
		observed := last.GetObservedAt().AsTime().UTC().Format(time.RFC3339Nano)
		fmt.Printf("Children of %s: %s (observed %s)\n", parentID, last.GetWakeReason(), observed)
		for _, child := range last.GetEnded() {
			ended := ""
			if child.EndedAt != nil {
				ended = " at " + child.GetEndedAt().AsTime().UTC().Format(time.RFC3339)
			}
			fmt.Printf("  ended   %s  %s%s\n", child.GetRunId(), child.GetStatus(), ended)
		}
		for _, child := range last.GetActive() {
			fmt.Printf("  active  %s  %s\n", child.GetRunId(), child.GetStatus())
		}
		fmt.Printf("Next: agent-manager run wait --children %s --since %s\n", parentID, observed)
	}); err != nil {
		return err
	}
	if last.GetWakeReason() == "timer" {
		return exitCodeError{code: exitWaitTimedOut}
	}
	return nil
}

func (a *App) runTree(args []string) error {
	fs := flag.NewFlagSet("run tree", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id, args = args[0], args[1:]
	}
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if id == "" {
		return fmt.Errorf("usage: agent-manager run tree <run-id> [--json]")
	}
	if err := requireUUIDFlag("run id", id); err != nil {
		return err
	}
	body, tree, err := a.services.Runs.Tree(id)
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	printRunTreeNode(tree.GetRoot(), "")
	summary := fmt.Sprintf("%d run(s); max child depth %d", tree.GetNodeCount(), tree.GetMaxChildDepth())
	if tree.GetTruncated() {
		summary += "; truncated: more descendants exist than one read returns"
	}
	fmt.Println(summary)
	return nil
}

func printRunTreeNode(node *apipb.RunTreeNode, indent string) {
	if node == nil {
		return
	}
	weighted := "weighted=unknown"
	if node.GetWeightedTokensKnown() {
		weighted = fmt.Sprintf("weighted=%.0f", node.GetWeightedTokens())
	}
	runner := node.GetRunner()
	if model := node.GetModel(); model != "" {
		runner += "/" + model
	}
	if runner == "" {
		runner = "-"
	}
	goal := node.GetGoalDelivery()
	if goal == "" {
		goal = "-"
	}
	line := fmt.Sprintf("%s%s  d%d  %-11s %s  goal=%s  elapsed=%s  %s",
		indent, node.GetRunId(), node.GetDepth(), formatEnumValue(node.GetStatus(), "RUN_STATUS_", "_"), runner, goal,
		(time.Duration(node.GetElapsedSeconds()) * time.Second).String(), weighted)
	if node.OwnerExpiresAt != nil && isLiveRunStatus(node.GetStatus()) {
		left := time.Until(node.GetOwnerExpiresAt().AsTime()).Round(time.Minute)
		if left <= 0 {
			line += "  AUTHORITY EXPIRED (owner window ended: it cannot create children or wake its parent)"
		} else {
			line += "  authority=" + left.String()
		}
	}
	if node.GetOrphaned() {
		line += "  ORPHANED (parent ended; stop it explicitly if unwanted)"
	}
	fmt.Println(line)
	printStepBack(node.GetStepBack(), indent+"    ")
	for _, child := range node.GetChildren() {
		printRunTreeNode(child, indent+"  ")
	}
}

// printStepBack prints one line per transcript step-back trigger. They are
// observations for the supervisor, never a reason Agent Manager stops a run.
func printStepBack(triggers []*domainpb.StepBackTrigger, indent string) {
	for _, trigger := range triggers {
		fmt.Printf("%sSTEP-BACK %s (%s): %s; re-orient: %s\n", indent, trigger.GetId(), trigger.GetMechanism(), trigger.GetDetail(), trigger.GetRemedy())
	}
}
