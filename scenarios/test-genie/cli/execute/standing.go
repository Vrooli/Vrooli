package execute

import (
	"strings"

	execTypes "test-genie/cli/internal/execute"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

// standingFromProto maps the server's per-phase maturity standing into the CLI
// projection. It is the single mapping both the human scorecard and the --json
// output derive from, so the two output modes can never diverge (parity). Returns
// nil when the phase carries no standing (native phases / providers with no
// ladder). Doc-search topics are attached separately (see docSearchTopics).
func standingFromProto(st *runspb.PhaseMaturityStanding) *execTypes.MaturityStanding {
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

func findingsSummaryFromProto(fs *runspb.PhaseFindingsSummary) *execTypes.FindingsSummary {
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
