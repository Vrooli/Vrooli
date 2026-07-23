package runs

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"sync"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/vrooli/cli-core/cliutil"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	runs_v1connect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"
)

// runRef is a parsed `--run scenario:runID` handle.
type runRef struct {
	scenario string
	runID    string
}

// parseRunRef parses "scenario:runID". Run ids contain hyphens but never colons,
// so the FIRST colon separates the scenario from the id.
func parseRunRef(s string) (runRef, error) {
	s = strings.TrimSpace(s)
	i := strings.Index(s, ":")
	if i <= 0 || i >= len(s)-1 {
		return runRef{}, fmt.Errorf("invalid --run %q: want scenario:runID", s)
	}
	return runRef{scenario: s[:i], runID: s[i+1:]}, nil
}

// repeatedFlag collects repeated --run values.
type repeatedFlag []string

func (r *repeatedFlag) String() string     { return strings.Join(*r, ",") }
func (r *repeatedFlag) Set(v string) error { *r = append(*r, v); return nil }

// waitAllResult pairs a run handle with its resolved status (or error).
type waitAllResult struct {
	ref      runRef
	status   *runspb.RunLiveStatus
	timedOut bool
	// nonterminalWithoutTimeout is a malformed WaitRun outcome. It must be
	// treated like a recoverable detached wait, never as a terminal regression.
	nonterminalWithoutTimeout bool
	providerUnavailable       bool
	err                       error
}

// runWaitAll blocks until every named run is terminal (or --timeout elapses) via
// a client-side fan-out of concurrent unary WaitRun calls — NO new server RPC.
// It exits with one aggregate code so two parallel suites/diffs resolve in a
// single call instead of two backgrounded streams. Precedence (worst wins):
// failure(1) > still-in-flight-at-timeout(124) > not-comparable/error(2) >
// all-passed(0).
func runWaitAll(apiClient *cliutil.APIClient, args []string, w io.Writer) error {
	fs := flag.NewFlagSet("runs wait-all", flag.ContinueOnError)
	fs.SetOutput(w)
	var runs repeatedFlag
	fs.Var(&runs, "run", "A run to wait on as scenario:runID (repeatable)")
	timeout := fs.Int("timeout", 0, "Max seconds to block (0 = until all runs are terminal)")
	jsonOut := fs.Bool("json", false, "Emit a JSON array of run statuses")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(runs) == 0 {
		return fmt.Errorf("usage: test-genie runs wait-all --run <scenario>:<runID> [--run ...] [--timeout N] [--json]")
	}
	refs := make([]runRef, 0, len(runs))
	for _, rf := range runs {
		ref, err := parseRunRef(rf)
		if err != nil {
			return err
		}
		refs = append(refs, ref)
	}
	cl, err := client(apiClient)
	if err != nil {
		return err
	}

	results := waitAllFanOut(cl, refs, *timeout)
	if *jsonOut {
		if err := writeWaitAllJSON(w, results); err != nil {
			return err
		}
	} else {
		printWaitAll(w, results)
	}
	if code := aggregateExitCode(results); code != exitOK {
		return &exitErr{code: code, err: fmt.Errorf("wait-all: %d/%d run(s) not passing", countNotPassed(results), len(results))}
	}
	return nil
}

// waitAllFanOut runs one concurrent WaitRun per handle, preserving input order.
func waitAllFanOut(cl runs_v1connect.RunsServiceClient, refs []runRef, timeout int) []waitAllResult {
	results := make([]waitAllResult, len(refs))
	var wg sync.WaitGroup
	for i, ref := range refs {
		wg.Add(1)
		go func(i int, ref runRef) {
			defer wg.Done()
			resp, err := cl.WaitRun(context.Background(), connect.NewRequest(&runspb.WaitRunRequest{
				Scenario: ref.scenario, RunId: ref.runID, TimeoutSeconds: int32(timeout),
			}))
			r := waitAllResult{ref: ref}
			if err != nil {
				r.err = err
			} else {
				r.status = resp.Msg.GetStatus()
				r.timedOut = resp.Msg.GetTimedOut()
				r.nonterminalWithoutTimeout = !r.timedOut && !isTerminalWaitStatus(r.status.GetStatus())
				r.providerUnavailable = terminalHasProviderUnavailable(resp.Msg.GetTerminalRun())
			}
			results[i] = r
		}(i, ref)
	}
	wg.Wait()
	return results
}

// aggregateExitCode collapses the per-run outcomes to one code. A failure (or
// aborted) run dominates everything; then any still-in-flight-at-timeout; then
// any error/not-comparable; else all passed.
func aggregateExitCode(results []waitAllResult) int {
	anyTimeout, anyNotComparable := false, false
	for _, r := range results {
		switch {
		case r.err != nil:
			anyNotComparable = true
		case r.timedOut || r.nonterminalWithoutTimeout:
			anyTimeout = true
		case r.providerUnavailable:
			anyNotComparable = true
		case r.status.GetStatus() == "passed":
			// ok
		default:
			return exitRegression // failed/aborted dominates
		}
	}
	switch {
	case anyTimeout:
		return exitWaitTimeout
	case anyNotComparable:
		return exitNotComparable
	default:
		return exitOK
	}
}

func countNotPassed(results []waitAllResult) int {
	n := 0
	for _, r := range results {
		if r.err != nil || r.timedOut || r.nonterminalWithoutTimeout || r.providerUnavailable || r.status.GetStatus() != "passed" {
			n++
		}
	}
	return n
}

func printWaitAll(w io.Writer, results []waitAllResult) {
	for _, r := range results {
		switch {
		case r.err != nil:
			fmt.Fprintf(w, "?  %s %s  error: %v\n", r.ref.scenario, r.ref.runID, r.err)
		case r.timedOut || r.nonterminalWithoutTimeout:
			fmt.Fprintf(w, "⏳ %s %s  still running (%.0fs elapsed); reattach with: test-genie runs wait --json %s %s\n", r.ref.scenario, r.ref.runID, r.status.GetElapsedSeconds(), r.ref.scenario, r.ref.runID)
		default:
			mark := "✓"
			if r.status.GetStatus() != "passed" {
				mark = "✗"
			}
			fmt.Fprintf(w, "%s  %s %s  %s (%.0fs)\n", mark, r.ref.scenario, r.ref.runID, r.status.GetStatus(), r.status.GetElapsedSeconds())
		}
	}
}

// writeWaitAllJSON emits a JSON array of RunLiveStatus. An errored handle (the
// run was unreachable/unknown) is rendered as a synthetic status with
// status="error" so every requested handle appears in the array.
func writeWaitAllJSON(w io.Writer, results []waitAllResult) error {
	marshal := protojson.MarshalOptions{Multiline: true, Indent: "  "}
	parts := make([]string, 0, len(results))
	for _, r := range results {
		st := r.status
		if st == nil {
			st = &runspb.RunLiveStatus{Scenario: r.ref.scenario, RunId: r.ref.runID, Status: "error"}
			if r.err != nil {
				st.Error = r.err.Error()
			}
		}
		b, err := marshal.Marshal(st)
		if err != nil {
			return err
		}
		parts = append(parts, string(b))
	}
	fmt.Fprintf(w, "[%s]\n", strings.Join(parts, ", "))
	return nil
}
