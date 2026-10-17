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

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

var levelNumberRE = regexp.MustCompile(`\d+`)

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
	return BuildRunStandingView(ctx, resp, st.GetTarget(), st.GetStatus(), st.GetRunId(), nil, timedOut, st.GetRecommendedNextCheckSeconds(), scoreRunner)
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
		view.Scenario = terminal.GetTarget()
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
		state := ""
		if phase.GetPhasePresentation() == nil && phase.GetMaturityStanding() != nil {
			state = "legacy_maturity_standing"
		}
		phases = append(phases, Phase{
			Name:              phase.GetName(),
			Status:            phase.GetStatus(),
			DurationSeconds:   phase.GetDurationSeconds(),
			PhasePresentation: phase.GetPhasePresentation(),
			PresentationState: state,
			FindingsSummary:   FindingsSummaryFromProto(phase.GetFindingsSummary()),
		})
	}
	return phases
}

func phasesFromLiveStatus(st *runspb.RunLiveStatus) []Phase {
	presentations := st.GetTerminalPresentations()
	findings := st.GetTerminalFindingsSummaries()
	phases := make([]Phase, 0, len(presentations)+len(st.GetTerminalStandings()))
	for i, presentation := range presentations {
		if presentation == nil {
			continue
		}
		phase := Phase{
			Name:              firstNonEmpty(presentation.GetPhase(), st.GetActivePhase()),
			Status:            st.GetStatus(),
			PhasePresentation: presentation,
		}
		if i < len(findings) {
			phase.FindingsSummary = FindingsSummaryFromProto(findings[i])
		}
		phases = append(phases, phase)
	}
	for _, standing := range st.GetTerminalStandings() {
		if standing == nil {
			continue
		}
		phases = append(phases, Phase{Name: firstNonEmpty(standing.GetPhase(), st.GetActivePhase()), Status: st.GetStatus(), PresentationState: "legacy_maturity_standing"})
	}
	return phases
}

func summarizePhases(phases []Phase) execTypes.PhaseSummary {
	var summary execTypes.PhaseSummary
	for _, p := range phases {
		summary.Total++
		summary.DurationSeconds += int(p.DurationSeconds)
		summary.ObservationCount += len(p.Observations)
		switch strings.ToLower(p.Status) {
		case "passed":
			summary.Passed++
		case "failed", "aborted", "provider_unavailable":
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
		st := phase.PhasePresentation
		if st == nil || st.GetAtMaximum() || strings.TrimSpace(st.GetNextAction()) == "" {
			continue
		}
		candidates = append(candidates, priorityCandidate{
			phase:        phase.Name,
			standing:     st,
			level:        levelNumber(st.GetCurrentLevel()),
			priorityRank: capabilityPriorityRank(st),
			findingCount: len(st.GetBlockingFindingCodes()),
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
	if len(st.GetDocumentationTopics()) > 0 {
		topic = st.GetDocumentationTopics()[0]
	}
	phase := firstNonEmpty(st.GetPhase(), best.phase)
	return &execTypes.RunTopPriority{
		Phase:                   phase,
		Provider:                st.GetProvider(),
		CurrentLevel:            st.GetCurrentLevel(),
		CurrentLevelLabel:       st.GetCurrentLevelLabel(),
		NextLevel:               st.GetNextLevel(),
		NextMove:                st.GetNextAction(),
		NextMoveReason:          st.GetNextActionReason(),
		PriorityCapabilityID:    st.GetFocusCapabilityId(),
		PriorityCapabilityLabel: st.GetFocusCapabilityLabel(),
		DocSearchTopic:          topic,
		BlockingFindingCodes:    st.GetBlockingFindingCodes(),
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

func capabilityPriorityRank(st *commonv1.PhasePresentation) int {
	best := int(^uint(0) >> 1)
	for _, cap := range st.GetCapabilities() {
		if cap.GetPriorityRank() > 0 && int(cap.GetPriorityRank()) < best {
			best = int(cap.GetPriorityRank())
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
	standing     *commonv1.PhasePresentation
	level        int
	priorityRank int
	findingCount int
}
