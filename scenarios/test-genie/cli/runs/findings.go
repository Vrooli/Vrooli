package runs

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

// getRunFindingsCall maps `runs findings [<runID>|latest]` onto the
// RunsService.GetRunFindings RPC. run_id defaults to the latest run.
func getRunFindingsCall(apiClient *cliutil.APIClient) func(cliapp.OperationContext) (*runspb.GetRunFindingsResponse, error) {
	return func(ctx cliapp.OperationContext) (*runspb.GetRunFindingsResponse, error) {
		cl, err := client(apiClient)
		if err != nil {
			return nil, err
		}
		runID := strings.TrimSpace(ctx.Positional("run_id"))
		if runID == "" {
			runID = "latest"
		}
		resp, err := cl.GetRunFindings(context.Background(), connect.NewRequest(&runspb.GetRunFindingsRequest{
			Scenario: ctx.Flag("scenario"),
			RunId:    runID,
		}))
		if err != nil {
			return nil, err
		}
		return resp.Msg, nil
	}
}

// getRunFindingsReport renders the per-phase maturity standing as the human
// scorecard. cli-core owns the --json vs human selection (ProtoList), so both
// modes derive from the one GetRunFindingsResponse payload.
func getRunFindingsReport(_ cliapp.OperationContext, msg *runspb.GetRunFindingsResponse) cliapp.ListReport {
	summary := []string{fmt.Sprintf("Findings: %s run %s", msg.GetScenario(), msg.GetRunId())}
	if v := strings.TrimSpace(msg.GetVerdict()); v != "" {
		summary = append(summary, "Verdict: "+v)
	}
	var results []string
	var hints []string
	seenHint := map[string]bool{}
	standings := 0
	for _, p := range msg.GetPhases() {
		st := p.GetPhasePresentation()
		if st == nil {
			if historical := p.GetMaturityStanding(); historical != nil {
				standings++
				results = append(results, fmt.Sprintf("%-16s presentation: historical maturity standing (not canonical v1)", p.GetName()))
			}
			continue
		}
		standings++
		results = append(results, phaseStandingLines(p.GetName(), st)...)
		for _, topic := range findingsDocTopics(p.GetName(), st) {
			cmd := fmt.Sprintf("search-hub query %q --type doc", topic)
			if !seenHint[cmd] {
				seenHint[cmd] = true
				hints = append(hints, cmd)
			}
		}
	}
	if standings == 0 {
		results = append(results, "No phase declared a maturity standing for this run.")
	}
	return cliapp.ListReport{Summary: summary, ResultsHeading: "Standing", Results: results, RetrievalHints: hints}
}

// phaseStandingLines renders one phase's concise standing block. It suppresses
// the next-move line at maximum maturity.
func phaseStandingLines(phase string, st *commonv1.PhasePresentation) []string {
	lines := []string{fmt.Sprintf("%-16s %s", phase, rungLine(st))}
	if ns := strings.TrimSpace(st.GetNorthStar()); ns != "" {
		lines = append(lines, "  North Star: "+ns)
	}
	if st.GetAtMaximum() {
		lines = append(lines, "  Maximum maturity reached.")
		return lines
	}
	if move := strings.TrimSpace(st.GetNextAction()); move != "" {
		line := "  Next: " + move
		if reason := strings.TrimSpace(st.GetNextActionReason()); reason != "" {
			line += "  (" + reason + ")"
		}
		if capLabel := strings.TrimSpace(st.GetFocusCapabilityLabel()); capLabel != "" {
			line += "  [→ " + capLabel + "]"
		}
		lines = append(lines, line)
	}
	if codes := st.GetBlockingFindingCodes(); len(codes) > 0 {
		lines = append(lines, "  Blocking: "+strings.Join(codes, ", "))
	}
	return lines
}

func rungLine(st *commonv1.PhasePresentation) string {
	cur := firstNonEmptyStr(st.GetCurrentLevel(), "?")
	if st.GetAtMaximum() {
		return cur + " (ceiling)"
	}
	next := strings.TrimSpace(st.GetNextLevel())
	if next == "" {
		return cur
	}
	line := cur + " → " + next
	if ceil := strings.TrimSpace(st.GetCeilingLevel()); ceil != "" && ceil != next {
		line += fmt.Sprintf(" (ceiling %s)", ceil)
	}
	return line
}

// findingsDocTopics reads the provider-owned documentation topics verbatim.
func findingsDocTopics(phase string, st *commonv1.PhasePresentation) []string {
	if st.GetAtMaximum() || strings.TrimSpace(phase) == "" {
		return nil
	}
	return append([]string(nil), st.GetDocumentationTopics()...)
}

func firstNonEmptyStr(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
