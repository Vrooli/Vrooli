package usage

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/proto"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	domainconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain/domainconnect"
	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
)

// frictionClientFor builds the Friction client the report and dismiss actions
// write through; tests replace it.
type frictionClientFor func(*cliapp.ScenarioApp) domainconnect.FrictionServiceClient

func frictionConnectClient(core *cliapp.ScenarioApp) domainconnect.FrictionServiceClient {
	h, base := cliapp.NewConnectHTTPClient(core)
	return domainconnect.NewFrictionServiceClient(h, base)
}

var fingerprintPositional = cliapp.Positional{Name: "fingerprint", Required: true, Description: "Finding fingerprint from `usage friction`"}

// findingSinceFlag names the window the finding was listed in.
var findingSinceFlag = cliapp.Flag{Name: "since", Description: "RFC 3339 start of the window the finding was listed in, as given to `usage friction` (default 7 days ago)"}

func frictionCommand(newClient clientFor) cliapp.Command {
	return usageCommand("friction", "What keeps going wrong for agents? Owner-level tool problems and classified operator interventions [--since RFC3339] [--status open|reported|dismissed] [--json]",
		"Lists owner-level tool problems from the tool friction profiles of the window (default the last 7 days): failures clustered by error signature, degradations a CLI reports, shape guessing, heavy output, polling and bypasses of a scenario's CLI. Each names its owner, cost in the unit of its harm (calls, time or context tokens) and share of the window's total, example commands and evidence runs. Operator interventions follow: operator messages the classifier put in a friction category that passed the accuracy gate, counted per category and owner with interventions per agent-hour, the agent work before them and example messages. While no category has passed the gate the section gives the classification coverage and says so. The coverage line says what the profiles could see: an empty list means nothing only beside it. `usage friction-report <fingerprint>` files a problem in the scenario-qa bug inbox and records the bug id; `usage friction-dismiss <fingerprint>` sets it aside.",
		[]cliapp.Flag{{Name: "since", Description: "RFC 3339 start of the friction window (default 7 days ago)"}, {Name: "status", Description: "open | reported | dismissed (default: all)"}, limitFlag},
		func(op cliapp.OperationContext) (proto.Message, error) {
			limit, err := parseLimit(op)
			if err != nil {
				return nil, err
			}
			response, err := newClient(op.Core()).UsageFriction(context.Background(), connect.NewRequest(&measurepb.UsageFrictionRequest{Since: strings.TrimSpace(op.Flag("since")), Status: strings.TrimSpace(op.Flag("status")), Limit: limit}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(response proto.Message) cliapp.ListReport {
			r := response.(*measurepb.UsageFrictionResponse)
			report := cliapp.ListReport{ResultsHeading: "Findings", Summary: []string{
				fmt.Sprintf("%d open · %d reported · %d dismissed as of %s; tool window since %s", r.GetOpen(), r.GetReported(), r.GetDismissed(), stamp(r.GetGeneratedAt()), stamp(r.GetWindowStart())),
				fmt.Sprintf("Listed %d; %d more under the materiality floor (score %.0f%%), %d more past --limit", len(r.GetToolFindings()), r.GetBelowFloor(), 100*r.GetMaterialityScore(), r.GetBeyondLimit()),
				"Coverage: " + r.GetCoverage().GetSentence(),
			}}
			if len(r.GetToolFindings()) == 0 {
				report.Summary = append(report.Summary, "No tool problem met its threshold in what the coverage line says was seen.")
			}
			for _, f := range r.GetToolFindings() {
				report.Results = append(report.Results,
					fmt.Sprintf("%s  [%s] %s", f.GetFingerprint(), f.GetKind(), f.GetDescription()),
					fmt.Sprintf("   owner %s · %d calls in %d runs · score %.1f%% · cost %s · %s", f.GetOwner(), f.GetCalls(), f.GetRuns(), 100*f.GetScore(), shareSentences(f.GetShares()), status(f.GetStatus(), f.GetBugId())))
				report.Results = append(report.Results, frictionDetailLines(f.GetRootCause(), f.GetSubs(), f.GetGuesses(), f.GetPrintedTokens(), f.GetReadTokens())...)
				for _, example := range f.GetExampleCommands() {
					report.Results = append(report.Results, "   e.g. "+example)
				}
			}
			report.Results = append(report.Results, interventionLines(r.GetInterventions())...)
			report.RetrievalHints = []string{"agent-manager usage friction-report <fingerprint>  # file it with scenario-qa", "agent-manager usage friction-dismiss <fingerprint>  # set it aside", "agent-manager measures get tool-profiles --param owner=<scenario>  # the 50 costliest command paths"}
			return report
		})
}

// interventionLines renders the operator interventions section. Without a
// shown category it states the coverage and why, never "no friction".
func interventionLines(section *measurepb.UsageInterventions) []string {
	if section == nil {
		return []string{"Operator interventions: unavailable"}
	}
	lines := []string{fmt.Sprintf("Operator interventions: %d operator messages, %.0f%% classified over %.1f agent-hours", section.GetOperatorMessages(), 100*section.GetCoverage(), section.GetAgentHours())}
	if len(section.GetCategories()) == 0 {
		return append(lines, "   none shown: "+section.GetUnshownReason()+" (this is not \"no friction\")")
	}
	for _, c := range section.GetCategories() {
		name, owner := c.GetCategory(), "owner "+c.GetOwner()
		if c.GetPooled() {
			name, owner = "operator friction (category unproven)", "owners by hint"
		}
		line := fmt.Sprintf("   %s · %d (%.2f per agent-hour) · %s · agent work before: %d requests, %d tool calls, %s tokens", name, c.GetInterventions(), c.GetPerAgentHour(), owner, c.GetWorkRequests(), c.GetWorkToolCalls(), compact(c.GetWorkTokens()))
		if c.GetRepeats() > 0 {
			line += fmt.Sprintf(" · %d repeat reports", c.GetRepeats())
		}
		lines = append(lines, line)
		for _, h := range c.GetHints() {
			lines = append(lines, fmt.Sprintf("      hint (unproven): %d look like %s · owner %s", h.GetCount(), h.GetCategory(), h.GetOwner()))
		}
	}
	for _, e := range section.GetExamples() {
		category := e.GetCategory()
		if e.GetPooled() {
			category += ", unproven"
		}
		line := fmt.Sprintf("   e.g. [%s] %q  run %s (agent-manager run report %s)", category, e.GetText(), e.GetRunId(), e.GetRunId())
		if e.GetRepeatOfEventId() != "" {
			line += " · repeats " + e.GetRepeatOfEventId()
		}
		lines = append(lines, line)
	}
	return lines
}

// frictionDetailLines states what a tool finding or action adds to its one-line
// description: the lines a root cause rolls up, the guessed commands with the
// alias the CLI suggested, and the tokens an
// output printed against those the model read.
func frictionDetailLines(root string, subs []*measurepb.UsageToolSub, guesses []*measurepb.UsageToolGuess, printed, read int64) []string {
	var lines []string
	if len(subs) > 1 {
		lines = append(lines, fmt.Sprintf("   root cause %q rolls up %d lines:", root, len(subs)))
		for _, sub := range subs {
			lines = append(lines, fmt.Sprintf("      %s in %d calls  %s", sub.GetKind(), sub.GetCalls(), sub.GetSignature()))
		}
	}
	for _, g := range guesses {
		if g.GetSuggestion() != "" {
			lines = append(lines, fmt.Sprintf("   guessed `%s` x%d: add alias `%s`→`%s`", g.GetToken(), g.GetCalls(), g.GetToken(), g.GetSuggestion()))
		} else {
			lines = append(lines, fmt.Sprintf("   guessed `%s` x%d: fix the help", g.GetToken(), g.GetCalls()))
		}
	}
	if printed > 0 && read > 0 {
		lines = append(lines, fmt.Sprintf("   largest call printed ~%s tokens; the model read %s", compact(printed), compact(read)))
	}
	return lines
}

// shareSentences states a finding's cost of each kind of harm over that
// kind's base as the API names it ("failure 101 calls = 12.4% of the window's
// 812 Vrooli CLI failed calls"): a roll-up of failures and notices has one
// share per kind, never one share over a mixed base.
func shareSentences(shares []*measurepb.UsageToolShare) string {
	parts := make([]string, 0, len(shares))
	for _, s := range shares {
		parts = append(parts, s.GetSentence())
	}
	if len(parts) == 0 {
		return "unknown"
	}
	return strings.Join(parts, "; ")
}

func status(value, bugID string) string {
	if bugID != "" {
		return value + " " + bugID
	}
	return value
}

func frictionReportCommand(newClient frictionClientFor) cliapp.Command {
	cmd := cliapp.Command{
		Name: "friction-report", NeedsAPI: true, Description: "File a friction finding in the scenario-qa bug inbox [--since RFC3339] [--json]",
		Usage:    "agent-manager usage friction-report <fingerprint> [--since RFC3339] [--json]",
		HelpText: "Files the finding with its description, owner, counts, cost and evidence runs through the report-bug path and records the bug id on it (status reported). The fingerprint is looked up in the window you listed it in: pass the same --since as `usage friction` (default the last 7 days). Reporting a finding that was already reported returns its bug id without filing another.",
		Args:     cliapp.ArgSchema{Positionals: []cliapp.Positional{fingerprintPositional}, Flags: []cliapp.Flag{findingSinceFlag}},
	}
	return cmd.WithPrimitive(cliapp.ProtoMutation(
		func(op cliapp.OperationContext) (*domainpb.ReportFrictionResponse, error) {
			response, err := newClient(op.Core()).ReportFriction(context.Background(), connect.NewRequest(&domainpb.ReportFrictionRequest{Fingerprint: strings.TrimSpace(op.Positional("fingerprint")), Since: strings.TrimSpace(op.Flag("since"))}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, r *domainpb.ReportFrictionResponse) cliapp.MutationReport {
			return cliapp.MutationReport{Result: []string{fmt.Sprintf("Reported as %s: %s", r.GetBugId(), r.GetDescription())}}
		}))
}

func frictionDismissCommand(newClient frictionClientFor) cliapp.Command {
	cmd := cliapp.Command{
		Name: "friction-dismiss", NeedsAPI: true, Description: "Set a friction finding aside [--since RFC3339] [--json]",
		Usage:    "agent-manager usage friction-dismiss <fingerprint> [--since RFC3339] [--json]",
		HelpText: "Marks the finding dismissed: it stays listed under --status dismissed and is no longer recommended. The fingerprint is looked up in the window you listed it in: pass the same --since as `usage friction` (default the last 7 days).",
		Args:     cliapp.ArgSchema{Positionals: []cliapp.Positional{fingerprintPositional}, Flags: []cliapp.Flag{findingSinceFlag}},
	}
	return cmd.WithPrimitive(cliapp.ProtoMutation(
		func(op cliapp.OperationContext) (*domainpb.DismissFrictionResponse, error) {
			response, err := newClient(op.Core()).DismissFriction(context.Background(), connect.NewRequest(&domainpb.DismissFrictionRequest{Fingerprint: strings.TrimSpace(op.Positional("fingerprint")), Since: strings.TrimSpace(op.Flag("since"))}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, r *domainpb.DismissFrictionResponse) cliapp.MutationReport {
			return cliapp.MutationReport{Result: []string{"Dismissed " + r.GetFingerprint()}}
		}))
}
