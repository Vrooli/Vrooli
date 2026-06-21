// Package fleet implements the `test-genie fleet` CLI verb group — a thin client
// over the test-genie API's RunsService.GetFleetHealth Connect-RPC. It surfaces
// fleet-wide health aggregated over STORED runs (never a live fleet run): the
// most-errored scenarios, finding-source clustering, and explicit
// never-tested-in-window coverage gaps, every datum as-of stamped. It is the
// read side of the Stage 3 fleet backbone, consumed by humans (default) and the
// meta-optimization loop (--json).
package fleet

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

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

// Run dispatches the `fleet` subcommands.
func Run(apiClient *cliutil.APIClient, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: test-genie fleet status [--json] [--window-days N] [--roster]")
	}
	switch args[0] {
	case "status":
		return runStatus(apiClient, args[1:], os.Stdout)
	default:
		return fmt.Errorf("unknown fleet subcommand %q (expected: status)", args[0])
	}
}

func runStatus(apiClient *cliutil.APIClient, args []string, w io.Writer) error {
	fs := flag.NewFlagSet("fleet status", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	asJSON := fs.Bool("json", false, "Emit the full fleet-health payload as proto JSON")
	windowDays := fs.Int("window-days", 0, "Fleet aggregation look-back window in days (0 = server default)")
	roster := fs.Bool("roster", false, "Cross-reference the on-disk fleet roster to list never-tested-in-window scenarios")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	client, err := newClient(apiClient)
	if err != nil {
		return err
	}
	resp, err := client.GetFleetHealth(context.Background(), connect.NewRequest(&runspb.GetFleetHealthRequest{
		WindowDays:    int32(*windowDays),
		IncludeRoster: *roster,
	}))
	if err != nil {
		return fmt.Errorf("get fleet-health: %w", err)
	}
	fh := resp.Msg.GetFleetHealth()
	if fh == nil {
		return errors.New("empty fleet-health response")
	}

	if *asJSON {
		out, err := protojson.MarshalOptions{Multiline: true, Indent: "  ", EmitUnpopulated: true}.Marshal(resp.Msg)
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(out))
		return nil
	}

	printFleet(w, fh)
	return nil
}

func printFleet(w io.Writer, fh *runspb.FleetHealth) {
	asOf := fh.GetCapturedAt()
	if t, err := time.Parse(time.RFC3339Nano, asOf); err == nil {
		asOf = t.Format("2006-01-02 15:04 MST")
	}
	fmt.Fprintf(w, "Fleet health (%dd window, as of %s)\n", fh.GetWindowDays(), asOf)
	fmt.Fprintf(w, "  Scenarios: %d tested / %d total · %d runs · %d issues\n",
		fh.GetScenariosTested(), fh.GetScenariosTotal(), fh.GetTotalRuns(), fh.GetTotalIssues())

	if sources := fh.GetTopFindingSources(); len(sources) > 0 {
		parts := make([]string, 0, len(sources))
		for _, s := range sources {
			parts = append(parts, fmt.Sprintf("%s=%d", s.GetSource(), s.GetIssues()))
		}
		fmt.Fprintf(w, "  Top finding sources: %s\n", strings.Join(parts, " "))
	}

	// Scenarios arrive most-errored first; show the head of the ranking.
	scenarios := fh.GetScenarios()
	fmt.Fprintln(w, "  Most-errored scenarios:")
	shown := 0
	for _, sc := range scenarios {
		if sc.GetFailedRuns() == 0 && sc.GetIssues() == 0 {
			break // ranking is most-errored first; the rest are clean
		}
		fmt.Fprintf(w, "      %-32s fail=%.0f%% (%d/%d runs) issues=%d last=%s %s\n",
			sc.GetScenario(), sc.GetFailureRate()*100, sc.GetFailedRuns(), sc.GetRuns(),
			sc.GetIssues(), sc.GetLastOutcome(), ageLabel(sc.GetAgeDays()))
		shown++
		if shown >= 10 {
			break
		}
	}
	if shown == 0 {
		fmt.Fprintln(w, "      (none — every tested scenario is passing in the window)")
	}

	if never := fh.GetNeverTestedInWindow(); len(never) > 0 {
		fmt.Fprintf(w, "  Never tested in window (%d): %s\n", len(never), strings.Join(never, ", "))
	}
}

// ageLabel renders the staleness of a scenario's last run for humans.
func ageLabel(ageDays float64) string {
	if ageDays <= 0 {
		return ""
	}
	if ageDays < 1 {
		return fmt.Sprintf("(%.0fh ago)", ageDays*24)
	}
	return fmt.Sprintf("(%.0fd ago)", ageDays)
}
