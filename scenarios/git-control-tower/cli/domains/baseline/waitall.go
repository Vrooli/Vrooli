package baseline

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"

	baselinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines"
)

// repeatedStringFlag collects repeated --run values.
type repeatedStringFlag []string

func (r *repeatedStringFlag) String() string     { return strings.Join(*r, ",") }
func (r *repeatedStringFlag) Set(v string) error { *r = append(*r, v); return nil }

// diffRef is a parsed `--run scenario:name:run` handle for a started diff.
type diffRef struct {
	scenario string
	name     string
	run      string
}

// parseDiffRef parses "scenario:name:run". Scenario slugs, baseline names, and
// run ids never contain colons, so three colon-separated parts are unambiguous.
func parseDiffRef(s string) (diffRef, error) {
	parts := strings.SplitN(strings.TrimSpace(s), ":", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return diffRef{}, fmt.Errorf("invalid --run %q: want scenario:name:run", s)
	}
	return diffRef{scenario: parts[0], name: parts[1], run: parts[2]}, nil
}

// diffWaitResult pairs a diff handle with its resolved verdict (or in-flight/error state).
type diffWaitResult struct {
	ref        diffRef
	verdict    string
	inProgress bool
	err        error
}

// runDiffWaitAll resolves several started diffs concurrently — one server-side
// blocking GetDiffResult(Wait=true) per handle — so N parallel `baseline diff`
// runs resolve in ONE call instead of N sequential `diff status` polls. It exits
// with the worst diff code: regression(1) > not-ready(3) > not-comparable(2) >
// clean(0).
func runDiffWaitAll(core *cliapp.ScenarioApp, args []string) error {
	var runs repeatedStringFlag
	var jsonOut bool
	fs := newFlagSet("baseline diff wait-all")
	fs.Var(&runs, "run", "A started diff to resolve as scenario:name:run (repeatable)")
	fs.BoolVar(&jsonOut, "json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(runs) == 0 {
		return fmt.Errorf("usage: baseline diff wait-all --run <scenario>:<name>:<run> [--run ...] [--json]")
	}
	refs := make([]diffRef, 0, len(runs))
	for _, rf := range runs {
		ref, err := parseDiffRef(rf)
		if err != nil {
			return err
		}
		refs = append(refs, ref)
	}

	client := clientFactory(core)
	results := make([]diffWaitResult, len(refs))
	var wg sync.WaitGroup
	for i, ref := range refs {
		wg.Add(1)
		go func(i int, ref diffRef) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), baselineClientTimeout)
			defer cancel()
			resp, err := client.GetDiffResult(ctx, connect.NewRequest(&baselinesv1.GetDiffResultRequest{
				Scenario: ref.scenario, Name: ref.name, RunId: ref.run, Wait: true,
			}))
			r := diffWaitResult{ref: ref}
			switch {
			case err != nil:
				r.err = err
			case resp.Msg.GetStatus() == "in_progress":
				r.inProgress = true
			default:
				r.verdict = resp.Msg.GetDiff().GetVerdict()
			}
			results[i] = r
		}(i, ref)
	}
	wg.Wait()

	if jsonOut {
		printDiffWaitAllJSON(results)
	} else {
		printDiffWaitAll(results)
	}
	os.Exit(aggregateDiffCode(results))
	return nil
}

// aggregateDiffCode collapses per-diff outcomes to one code. A regression
// dominates; then any still-computing (not-ready); then any error/not-comparable;
// else clean.
func aggregateDiffCode(results []diffWaitResult) int {
	anyNotReady, anyNotComparable := false, false
	for _, r := range results {
		switch {
		case r.err != nil:
			anyNotComparable = true
		case r.inProgress:
			anyNotReady = true
		case r.verdict == "regression":
			return exitRegression
		case r.verdict == "not-comparable":
			anyNotComparable = true
		}
	}
	switch {
	case anyNotReady:
		return exitNotReady
	case anyNotComparable:
		return exitNotComparable
	default:
		return exitOK
	}
}

func printDiffWaitAll(results []diffWaitResult) {
	for _, r := range results {
		switch {
		case r.err != nil:
			fmt.Printf("?  %s/%s (run %s)  error: %v\n", r.ref.scenario, r.ref.name, r.ref.run, r.err)
		case r.inProgress:
			fmt.Printf("⏳ %s/%s (run %s)  still computing\n", r.ref.scenario, r.ref.name, r.ref.run)
		default:
			fmt.Printf("%s %s/%s (run %s)  %s\n", verdictMark(r.verdict), r.ref.scenario, r.ref.name, r.ref.run, r.verdict)
		}
	}
}

func printDiffWaitAllJSON(results []diffWaitResult) {
	out := make([]map[string]any, 0, len(results))
	for _, r := range results {
		m := map[string]any{"scenario": r.ref.scenario, "name": r.ref.name, "run": r.ref.run}
		switch {
		case r.err != nil:
			m["status"] = "error"
			m["error"] = r.err.Error()
		case r.inProgress:
			m["status"] = "in_progress"
		default:
			m["status"] = "ready"
			m["verdict"] = r.verdict
		}
		out = append(out, m)
	}
	_ = printJSON(out)
}
