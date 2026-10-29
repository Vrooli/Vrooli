package usage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/proto"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	domainconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain/domainconnect"
	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
)

// episodesClientFor builds the Episodes client the friction actions write
// through; tests replace it.
type episodesClientFor func(*cliapp.ScenarioApp) domainconnect.EpisodesServiceClient

func episodesConnectClient(core *cliapp.ScenarioApp) domainconnect.EpisodesServiceClient {
	h, base := cliapp.NewConnectHTTPClient(core)
	return domainconnect.NewEpisodesServiceClient(h, base)
}

var fingerprintPositional = cliapp.Positional{Name: "fingerprint", Required: true, Description: "Problem fingerprint from `usage friction`"}

func frictionCommand(newClient clientFor) cliapp.Command {
	return usageCommand("friction", "What keeps going wrong for agents? Recurring friction problems in plain words [--status open|reported|dismissed] [--json]",
		"Lists recurring friction problems (three or more episodes with one fingerprint in runs that used quota in the current windows), each as one sentence derived from the episodes' evidence with its owner, episode and run counts, elapsed time (typical and total; episodes can overlap), quota points, and durable status. `usage friction-report <fingerprint>` files a problem in the scenario-qa bug inbox and records the bug id; `usage friction-dismiss <fingerprint>` sets it aside.",
		[]cliapp.Flag{providerFlag, {Name: "status", Description: "open | reported | dismissed (default: all)"}, limitFlag},
		func(op cliapp.OperationContext) (proto.Message, error) {
			limit, err := parseLimit(op)
			if err != nil {
				return nil, err
			}
			response, err := newClient(op.Core()).UsageFriction(context.Background(), connect.NewRequest(&measurepb.UsageFrictionRequest{Provider: strings.TrimSpace(op.Flag("provider")), Status: strings.TrimSpace(op.Flag("status")), Limit: limit}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(response proto.Message) cliapp.ListReport {
			r := response.(*measurepb.UsageFrictionResponse)
			report := cliapp.ListReport{ResultsHeading: "Problems", Summary: []string{fmt.Sprintf("%d open · %d reported · %d dismissed as of %s; points are estimates", r.GetOpen(), r.GetReported(), r.GetDismissed(), stamp(r.GetGeneratedAt()))}}
			if r.GetRunsAnalysed() < r.GetRunsTotal() {
				report.Summary = append(report.Summary, fmt.Sprintf("Re-analysing %d of %d runs: counts and totals cover the analysed runs only.", r.GetRunsAnalysed(), r.GetRunsTotal()))
			}
			for _, p := range r.GetProblems() {
				status := p.GetStatus()
				if p.GetBugId() != "" {
					status += " " + p.GetBugId()
				}
				typical, total := formatMillis(p.GetTypicalWallClockMs()), formatMillis(p.GetWallClockMs())
				cost := fmt.Sprintf("typical %s wasted, total %s wasted", typical, total)
				switch p.GetTimeKind() {
				case "idle":
					cost = fmt.Sprintf("idle typically %s, %s in total (waiting, not wasted work)", typical, total)
				case "elapsed":
					cost = fmt.Sprintf("took typically %s, %s in total", typical, total)
				case "none":
					cost = fmt.Sprintf("%d tool calls and %s tokens of work after declaring done (not counted as lost time)", p.GetEpisodeTurns(), compact(p.GetTokens()))
				}
				report.Results = append(report.Results,
					fmt.Sprintf("%s  %s", p.GetFingerprint()[:min(12, len(p.GetFingerprint()))], p.GetDescription()),
					fmt.Sprintf("   owner %s · %d episodes in %d runs · %s · ≈%.1f pts of %s · %s", p.GetOwner(), p.GetEpisodes(), p.GetRuns(),
						cost, p.GetPoints(), windowTitle(windowScope{provider: p.GetProvider(), pool: p.GetPool(), window: p.GetWindow()}), status))
			}
			report.RetrievalHints = []string{"agent-manager usage friction-report <fingerprint>  # file it with scenario-qa", "agent-manager usage friction-dismiss <fingerprint>  # set it aside"}
			return report
		})
}

func frictionReportCommand(newClient episodesClientFor) cliapp.Command {
	cmd := cliapp.Command{
		Name: "friction-report", NeedsAPI: true, Description: "File a friction problem in the scenario-qa bug inbox [--json]",
		Usage:    "agent-manager usage friction-report <fingerprint> [--json]",
		HelpText: "Files the problem with its description, owner, counts, cost and evidence runs through the report-bug path and records the bug id on it (status reported). Reporting a problem that was already reported returns its bug id without filing another.",
		Args:     cliapp.ArgSchema{Positionals: []cliapp.Positional{fingerprintPositional}},
	}
	return cmd.WithPrimitive(cliapp.ProtoMutation(
		func(op cliapp.OperationContext) (*domainpb.ReportFrictionProblemResponse, error) {
			response, err := newClient(op.Core()).ReportFrictionProblem(context.Background(), connect.NewRequest(&domainpb.ReportFrictionProblemRequest{Fingerprint: strings.TrimSpace(op.Positional("fingerprint"))}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, r *domainpb.ReportFrictionProblemResponse) cliapp.MutationReport {
			return cliapp.MutationReport{Result: []string{fmt.Sprintf("Reported as %s: %s", r.GetBugId(), r.GetDescription())}}
		}))
}

func frictionDismissCommand(newClient episodesClientFor) cliapp.Command {
	cmd := cliapp.Command{
		Name: "friction-dismiss", NeedsAPI: true, Description: "Set a friction problem aside [--json]",
		Usage:    "agent-manager usage friction-dismiss <fingerprint> [--json]",
		HelpText: "Marks the problem dismissed: it stays listed under --status dismissed and is no longer recommended.",
		Args:     cliapp.ArgSchema{Positionals: []cliapp.Positional{fingerprintPositional}},
	}
	return cmd.WithPrimitive(cliapp.ProtoMutation(
		func(op cliapp.OperationContext) (*domainpb.DismissFrictionProblemResponse, error) {
			response, err := newClient(op.Core()).DismissFrictionProblem(context.Background(), connect.NewRequest(&domainpb.DismissFrictionProblemRequest{Fingerprint: strings.TrimSpace(op.Positional("fingerprint"))}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, r *domainpb.DismissFrictionProblemResponse) cliapp.MutationReport {
			return cliapp.MutationReport{Result: []string{"Dismissed " + r.GetFingerprint()}}
		}))
}

// formatMillis renders elapsed milliseconds the way an operator reads them.
func formatMillis(ms int64) string {
	d := time.Duration(ms) * time.Millisecond
	switch {
	case d < time.Second:
		return fmt.Sprintf("%d ms", ms)
	case d < time.Minute:
		return fmt.Sprintf("%.0f s", d.Seconds())
	case d < time.Hour:
		return fmt.Sprintf("%dm %02ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %02dm", int(d.Hours()), int(d.Minutes())%60)
}
