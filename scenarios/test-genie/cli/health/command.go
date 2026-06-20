// Package health implements the `test-genie health` CLI verb: a thin client over
// the test-genie API's RunsService.GetSelfHealth Connect-RPC. It surfaces Test
// Genie's own self-observability — phase catalog, provider conformance, and the
// reliability ledger — for humans (default) and machines (--json). It closes the
// long-standing opaque-HTTP-500 capability gap (dec-1777068259096417622) by
// giving the meta-optimization loop a structured self-health signal.
package health

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/vrooli/cli-core/cliutil"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	"github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"
)

// newClient is a package var so tests can substitute a fake RunsServiceClient.
var newClient = func(apiClient *cliutil.APIClient) (runs_v1connect.RunsServiceClient, error) {
	baseURL := strings.TrimRight(apiClient.BaseURL(), "/")
	if baseURL == "" {
		return nil, errors.New("test-genie API base URL is not configured")
	}
	return runs_v1connect.NewRunsServiceClient(http.DefaultClient, baseURL), nil
}

// Run executes the `health` verb.
func Run(apiClient *cliutil.APIClient, args []string) error {
	return run(apiClient, args, os.Stdout)
}

func run(apiClient *cliutil.APIClient, args []string, w io.Writer) error {
	fs := flag.NewFlagSet("health", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	asJSON := fs.Bool("json", false, "Emit the full self-health payload as proto JSON")
	windowDays := fs.Int("window-days", 0, "Reliability-ledger look-back window (0 = server default)")
	skipConformance := fs.Bool("skip-conformance", false, "Skip the live provider conformance scan (faster)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	client, err := newClient(apiClient)
	if err != nil {
		return err
	}
	resp, err := client.GetSelfHealth(context.Background(), connect.NewRequest(&runspb.GetSelfHealthRequest{
		WindowDays:      int32(*windowDays),
		SkipConformance: *skipConformance,
	}))
	if err != nil {
		return fmt.Errorf("get self-health: %w", err)
	}
	sh := resp.Msg.GetSelfHealth()
	if sh == nil {
		return errors.New("empty self-health response")
	}

	if *asJSON {
		out, err := protojson.MarshalOptions{Multiline: true, Indent: "  ", EmitUnpopulated: true}.Marshal(resp.Msg)
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(out))
		return nil
	}

	printSummary(w, sh)
	return nil
}

func printSummary(w io.Writer, sh *runspb.SelfHealth) {
	cat := sh.GetCatalog()
	fmt.Fprintln(w, "Test Genie self-health")
	fmt.Fprintf(w, "  Catalog : %d phases (%d delegated, %d native)\n",
		cat.GetTotalPhases(), cat.GetDelegatedPhases(), cat.GetNativePhases())

	printConformance(w, sh)
	printLedger(w, sh.GetLedger())
}

func printConformance(w io.Writer, sh *runspb.SelfHealth) {
	freshness := sh.GetConformanceFreshness()
	conformance := sh.GetConformance()
	if freshness == "skipped" {
		fmt.Fprintln(w, "  Conformance: skipped")
		return
	}
	adopted, hardViolations := 0, 0
	for _, c := range conformance {
		if c.GetMetricsAdopted() {
			adopted++
		}
		if hasHardViolation(c) {
			hardViolations++
		}
	}
	fmt.Fprintf(w, "  Conformance (%s): %d providers, %d metrics-adopted, %d hard violation(s)\n",
		freshness, len(conformance), adopted, hardViolations)
	for _, c := range conformance {
		if hasHardViolation(c) {
			fmt.Fprintf(w, "      ! %s→%s adoption=%.0f%% %s\n",
				c.GetPhase(), c.GetProvider(), c.GetAdoptionScore()*100, strings.Join(c.GetViolations(), "; "))
		}
	}
}

func printLedger(w io.Writer, ledger *runspb.ReliabilityLedger) {
	if ledger == nil {
		return
	}
	fmt.Fprintf(w, "  Reliability (%dd window, %d runs): availability=%.1f%%\n",
		ledger.GetWindowDays(), ledger.GetRunCount(), ledger.GetAvailability()*100)
	if outcomes := ledger.GetRunOutcomes(); len(outcomes) > 0 {
		parts := make([]string, 0, len(outcomes))
		for _, o := range outcomes {
			parts = append(parts, fmt.Sprintf("%s=%d", o.GetOutcome(), o.GetCount()))
		}
		fmt.Fprintf(w, "      outcomes: %s\n", strings.Join(parts, " "))
	}

	// Rank phases by failure rate (then by failures) to surface the worst first.
	phases := append([]*runspb.PhaseReliability(nil), ledger.GetPhases()...)
	sort.Slice(phases, func(i, j int) bool {
		if phases[i].GetFailureRate() != phases[j].GetFailureRate() {
			return phases[i].GetFailureRate() > phases[j].GetFailureRate()
		}
		return phases[i].GetFailed() > phases[j].GetFailed()
	})
	for _, p := range phases {
		label := p.GetPhase()
		if p.GetProvider() != "" {
			label += " (" + p.GetProvider() + ")"
		}
		fmt.Fprintf(w, "      %-32s avail=%.0f%% fail=%.0f%% degraded=%d p50=%ds p95=%ds n=%d\n",
			label, p.GetAvailability()*100, p.GetFailureRate()*100, p.GetDegraded(),
			p.GetDuration().GetP50(), p.GetDuration().GetP95(), p.GetTotalObservations())
		for _, ws := range p.GetWorstScenarios() {
			fmt.Fprintf(w, "          worst: %s fail=%.0f%% (%d/%d)\n",
				ws.GetScenario(), ws.GetFailureRate()*100, ws.GetFailures(), ws.GetExecuted())
		}
	}
}

// hasHardViolation mirrors the API's conformance hard-violation rule.
func hasHardViolation(c *runspb.ProviderConformance) bool {
	if !c.GetSpecValid() {
		return true
	}
	if c.GetReachable() && (!c.GetContractValid() || !c.GetIdentityOk()) {
		return true
	}
	return false
}
