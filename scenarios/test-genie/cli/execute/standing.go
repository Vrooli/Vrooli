package execute

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"test-genie/cli/execute/report"
	execTypes "test-genie/cli/internal/execute"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

var levelNumberRE = regexp.MustCompile(`\d+`)

// StandingFromProto maps the server's per-phase maturity standing into the CLI
// projection. It is the single mapping both the human scorecard and the --json
// output derive from, so the two output modes can never diverge (parity). Returns
// nil when the phase carries no standing (native phases / providers with no
// ladder). Doc-search topics are attached separately (see docSearchTopics).
func StandingFromProto(st *runspb.PhaseMaturityStanding) *execTypes.MaturityStanding {
	if st == nil {
		return nil
	}
	out := &execTypes.MaturityStanding{
		Provider:                st.GetProvider(),
		Phase:                   st.GetPhase(),
		CurrentLevel:            st.GetCurrentLevel(),
		CurrentLevelLabel:       st.GetCurrentLevelLabel(),
		NextLevel:               st.GetNextLevel(),
		CeilingLevel:            st.GetCeilingLevel(),
		Clean:                   st.GetClean(),
		UnknownCount:            st.GetUnknownCount(),
		BlockingFindingCodes:    st.GetBlockingFindingCodes(),
		NextMove:                st.GetNextMove(),
		NextMoveReason:          st.GetNextMoveReason(),
		PriorityCapabilityID:    st.GetPriorityCapabilityId(),
		PriorityCapabilityLabel: st.GetPriorityCapabilityLabel(),
		NorthStar:               st.GetNorthStar(),
		AtMaximum:               st.GetAtMaximum(),
	}
	for _, c := range st.GetCapabilities() {
		if c == nil {
			continue
		}
		out.Capabilities = append(out.Capabilities, execTypes.CapabilityStanding{
			ID:                   c.GetId(),
			Label:                c.GetLabel(),
			CurrentLevel:         c.GetCurrentLevel(),
			CurrentLevelLabel:    c.GetCurrentLevelLabel(),
			NextLevel:            c.GetNextLevel(),
			CurrentSummary:       c.GetCurrentSummary(),
			NextUnlock:           c.GetNextUnlock(),
			Clean:                c.GetClean(),
			BlockingFindingCount: c.GetBlockingFindingCount(),
			BlockingFindingCodes: c.GetBlockingFindingCodes(),
			PriorityRank:         c.GetPriorityRank(),
			PriorityReason:       c.GetPriorityReason(),
		})
	}
	out.DocSearchTopics = docSearchTopics(out)
	return out
}

// docSearchTopics derives runnable search-hub query strings that resolve to the
// phase's structured remediation doc (Phase Capability Contract skeleton). The
// printer renders each as `search-hub query "<topic>" --type doc`. Suppressed at
// maximum maturity, where there is nothing to remediate.
func docSearchTopics(st *execTypes.MaturityStanding) []string {
	if st == nil || st.AtMaximum {
		return nil
	}
	phase := strings.TrimSpace(st.Phase)
	if phase == "" {
		return nil
	}
	// The general topic resolves to the "The rungs and their gates" / next-move
	// section; per-code topics resolve to "What each finding means" / "The
	// canonical fix" for the top blocking codes.
	topics := []string{phase + " maturity next move"}
	seen := map[string]bool{}
	for _, code := range st.BlockingFindingCodes {
		code = strings.TrimSpace(code)
		if code == "" || seen[code] {
			continue
		}
		seen[code] = true
		topics = append(topics, phase+" "+code+" canonical fix")
		if len(topics) >= 3 {
			break
		}
	}
	return topics
}

func FindingsSummaryFromProto(fs *runspb.PhaseFindingsSummary) *execTypes.FindingsSummary {
	if fs == nil {
		return nil
	}
	return &execTypes.FindingsSummary{
		Blockers: fs.GetBlockers(),
		Errors:   fs.GetErrors(),
		Warnings: fs.GetWarnings(),
		Infos:    fs.GetInfos(),
		Total:    fs.GetTotal(),
	}
}

// BuildRunStandingView creates the single curated terminal payload used by
// human and --json run renderers.
func BuildRunStandingView(ctx context.Context, resp Response, scenario, status, runID string, handle *RunHandle, timedOut bool, nextCheck int32, scoreRunner report.ScoreRunner) execTypes.RunStandingView {
	if runID == "" {
		runID = resp.ExecutionID
	}
	if status == "" {
		status = statusFromResponse(resp)
	}
	view := execTypes.RunStandingView{
		Success:                     resp.Success,
		Verdict:                     resp.Verdict,
		Status:                      status,
		Scenario:                    scenario,
		RunID:                       runID,
		ExecutionID:                 resp.ExecutionID,
		PhaseSummary:                resp.PhaseSummary,
		Phases:                      resp.Phases,
		Completeness:                report.LoadCompletenessSummary(ctx, scoreRunner, scenario),
		RunHandle:                   handle,
		RecommendedNextCheckSeconds: nextCheck,
		TimedOut:                    timedOut,
		Error:                       resp.Error,
	}
	view.TopPriority = TopPriorityFromPhases(view.Phases)
	return view
}

// BuildRunStandingViewFromLiveStatus projects a WaitRun/GetRunStatus snapshot
// onto the same curated terminal view as execute.
func BuildRunStandingViewFromLiveStatus(ctx context.Context, st *runspb.RunLiveStatus, timedOut bool, scoreRunner report.ScoreRunner) execTypes.RunStandingView {
	if st == nil {
		return execTypes.RunStandingView{TimedOut: timedOut}
	}
	phases := phasesFromLiveStatus(st)
	resp := Response{
		Success:      st.GetStatus() == "passed",
		Verdict:      st.GetVerdict(),
		ExecutionID:  st.GetRunId(),
		Phases:       phases,
		PhaseSummary: summarizePhases(phases),
		Error:        st.GetError(),
	}
	return BuildRunStandingView(ctx, resp, st.GetScenario(), st.GetStatus(), st.GetRunId(), nil, timedOut, st.GetRecommendedNextCheckSeconds(), scoreRunner)
}

// BuildRunStandingViewFromWaitResponse projects terminal phases from the
// canonical durable RunInfo carried by WaitRun. Live status remains the source
// for queued/in-progress timing, while terminal phase identity, status, and
// duration come from the exact same record as `runs show`.
func BuildRunStandingViewFromWaitResponse(ctx context.Context, response *runspb.WaitRunResponse, scoreRunner report.ScoreRunner) execTypes.RunStandingView {
	if response == nil {
		return execTypes.RunStandingView{}
	}
	view := BuildRunStandingViewFromLiveStatus(ctx, response.GetStatus(), response.GetTimedOut(), scoreRunner)
	if terminal := response.GetTerminalRun(); terminal != nil {
		view.RunID = terminal.GetRunId()
		view.ExecutionID = terminal.GetRunId()
		view.Scenario = terminal.GetScenario()
		view.Status = terminal.GetStatus()
		view.Success = terminal.GetStatus() == "passed"
		view.Phases = phasesFromTerminalRun(terminal)
		view.PhaseSummary = summarizePhases(view.Phases)
		view.TopPriority = TopPriorityFromPhases(view.Phases)
	}
	view.TerminalSnapshotSchemaVersion = response.GetTerminalSnapshotSchemaVersion()
	view.DegradedReasons = append([]string(nil), response.GetDegradedReasons()...)
	if len(view.DegradedReasons) == 0 && response.GetStatus() != nil {
		view.DegradedReasons = append([]string(nil), response.GetStatus().GetDegradedReasons()...)
	}
	return view
}

func phasesFromTerminalRun(run *runspb.RunInfo) []Phase {
	phases := make([]Phase, 0, len(run.GetPhases()))
	for _, phase := range run.GetPhases() {
		if phase == nil {
			continue
		}
		phases = append(phases, Phase{
			Name:             phase.GetName(),
			Status:           phase.GetStatus(),
			DurationSeconds:  phase.GetDurationSeconds(),
			MaturityStanding: StandingFromProto(phase.GetMaturityStanding()),
			FindingsSummary:  FindingsSummaryFromProto(phase.GetFindingsSummary()),
		})
	}
	return phases
}

func phasesFromLiveStatus(st *runspb.RunLiveStatus) []Phase {
	standings := st.GetTerminalStandings()
	findings := st.GetTerminalFindingsSummaries()
	phases := make([]Phase, 0, len(standings))
	for i, standing := range standings {
		if standing == nil {
			continue
		}
		phase := Phase{
			Name:             firstNonEmpty(standing.GetPhase(), st.GetActivePhase()),
			Status:           st.GetStatus(),
			MaturityStanding: StandingFromProto(standing),
		}
		if i < len(findings) {
			phase.FindingsSummary = FindingsSummaryFromProto(findings[i])
		}
		phases = append(phases, phase)
	}
	return phases
}

func summarizePhases(phases []Phase) execTypes.PhaseSummary {
	var summary execTypes.PhaseSummary
	for _, p := range phases {
		summary.Total++
		summary.DurationSeconds += int(p.DurationSeconds)
		switch p.Status {
		case "passed":
			summary.Passed++
		case "failed", "aborted":
			summary.Failed++
		case "skipped":
			summary.Skipped++
		}
	}
	return summary
}

func statusFromResponse(resp Response) string {
	if resp.Success {
		return "passed"
	}
	if strings.TrimSpace(resp.Verdict) != "" || strings.TrimSpace(resp.Error) != "" {
		return "failed"
	}
	return ""
}

// TopPriorityFromPhases selects the lowest-rung non-ceiling phase. Ties are
// resolved by priority rank, then finding count, then phase name for stability.
func TopPriorityFromPhases(phases []Phase) *execTypes.RunTopPriority {
	var candidates []priorityCandidate
	for _, phase := range phases {
		st := phase.MaturityStanding
		if st == nil || st.AtMaximum || strings.TrimSpace(st.NextMove) == "" {
			continue
		}
		candidates = append(candidates, priorityCandidate{
			phase:        phase.Name,
			standing:     st,
			level:        levelNumber(st.CurrentLevel),
			priorityRank: capabilityPriorityRank(st),
			findingCount: len(st.BlockingFindingCodes),
		})
	}
	if len(candidates) == 0 {
		return nil
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		if a.level != b.level {
			return a.level < b.level
		}
		if a.priorityRank != b.priorityRank {
			return a.priorityRank < b.priorityRank
		}
		if a.findingCount != b.findingCount {
			return a.findingCount > b.findingCount
		}
		return a.phase < b.phase
	})
	best := candidates[0]
	st := best.standing
	topic := ""
	if len(st.DocSearchTopics) > 0 {
		topic = st.DocSearchTopics[0]
	}
	phase := firstNonEmpty(st.Phase, best.phase)
	return &execTypes.RunTopPriority{
		Phase:                   phase,
		Provider:                st.Provider,
		CurrentLevel:            st.CurrentLevel,
		CurrentLevelLabel:       st.CurrentLevelLabel,
		NextLevel:               st.NextLevel,
		NextMove:                st.NextMove,
		NextMoveReason:          st.NextMoveReason,
		PriorityCapabilityID:    st.PriorityCapabilityID,
		PriorityCapabilityLabel: st.PriorityCapabilityLabel,
		DocSearchTopic:          topic,
		BlockingFindingCodes:    st.BlockingFindingCodes,
	}
}

func WriteRunStandingJSON(out io.Writer, view execTypes.RunStandingView) error {
	body, err := json.MarshalIndent(view, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal run standing view: %w", err)
	}
	_, err = fmt.Fprintf(out, "%s\n", body)
	return err
}

func levelNumber(level string) int {
	matches := levelNumberRE.FindAllString(level, -1)
	if len(matches) == 0 {
		return 99
	}
	n, err := strconv.Atoi(matches[len(matches)-1])
	if err != nil {
		return 99
	}
	return n
}

func capabilityPriorityRank(st *execTypes.MaturityStanding) int {
	best := int(^uint(0) >> 1)
	for _, cap := range st.Capabilities {
		if cap.PriorityRank > 0 && int(cap.PriorityRank) < best {
			best = int(cap.PriorityRank)
		}
	}
	return best
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

type priorityCandidate struct {
	phase        string
	standing     *execTypes.MaturityStanding
	level        int
	priorityRank int
	findingCount int
}
