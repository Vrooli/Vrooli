package validationprovider

import (
	"strings"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

// buildStanding maps the provider-returned MaturityAssessment into the compact
// per-phase standing carried in the run contract (Phase Capability Contract). It
// performs no recomputation — every field is a projection of what the provider
// already returned. Returns nil when the assessment declares no phase-level
// ladder (native phases, degraded providers), so the scorecard falls back to a
// plain status row rather than inventing a standing.
func buildStanding(provider Provider, a *commonv1.MaturityAssessment) *runspb.PhaseMaturityStanding {
	if a == nil {
		return nil
	}
	local := a.GetLocal()
	if local == nil {
		return nil
	}
	st := &runspb.PhaseMaturityStanding{
		Provider:             provider.ProviderScenario,
		Phase:                provider.Phase,
		CurrentLevel:         local.GetCurrentLevel(),
		NextLevel:            local.GetNextLevel(),
		Clean:                local.GetClean(),
		UnknownCount:         local.GetUnknownCount(),
		BlockingFindingCodes: append([]string(nil), local.GetBlockingFindingCodes()...),
	}
	levels := local.GetLevels()
	if len(levels) > 0 {
		top := levels[len(levels)-1]
		st.CeilingLevel = top.GetId()
		// The ladder's top rung carries the first-class North Star aspiration.
		st.NorthStar = firstNonEmpty(top.GetCapabilitySummary(), top.GetName())
	}
	if lvl := levelByID(levels, local.GetCurrentLevel()); lvl != nil {
		st.CurrentLevelLabel = firstNonEmpty(lvl.GetStatusLabel(), lvl.GetName())
	}
	// A phase is at maximum when it is clean at a rung with no further level to
	// unlock — the scorecard then suppresses the next-move/doc lines.
	st.AtMaximum = local.GetClean() && strings.TrimSpace(local.GetNextLevel()) == ""
	st.Capabilities = buildCapabilityStandings(a.GetCapabilities())

	// The single highest-unlock next move: the priority capability's next_unlock,
	// with the priority reason. Falls back to the phase-level current rung's
	// next_unlock when the provider declares no per-capability ladders.
	if focus := a.GetHighestPriorityCapability(); focus != nil && strings.TrimSpace(focus.GetCapabilityId()) != "" {
		st.PriorityCapabilityId = focus.GetCapabilityId()
		st.PriorityCapabilityLabel = focus.GetCapabilityLabel()
		st.NextMoveReason = focus.GetReason()
		if capab := findCapability(a.GetCapabilities(), focus.GetCapabilityId()); capab != nil {
			st.NextMove = capab.GetNextUnlock()
		}
	}
	if strings.TrimSpace(st.NextMove) == "" {
		if lvl := levelByID(levels, local.GetCurrentLevel()); lvl != nil {
			st.NextMove = lvl.GetNextUnlock()
		}
	}
	return st
}

func buildCapabilityStandings(capabilities []*commonv1.CapabilityMaturityAssessment) []*runspb.PhaseCapabilityStanding {
	if len(capabilities) == 0 {
		return nil
	}
	out := make([]*runspb.PhaseCapabilityStanding, 0, len(capabilities))
	for _, capab := range capabilities {
		if capab == nil {
			continue
		}
		cs := &runspb.PhaseCapabilityStanding{
			Id:                   capab.GetId(),
			Label:                capab.GetLabel(),
			CurrentLevel:         capab.GetCurrentLevel(),
			NextLevel:            capab.GetNextLevel(),
			CurrentSummary:       capab.GetCurrentSummary(),
			NextUnlock:           capab.GetNextUnlock(),
			Clean:                capab.GetClean(),
			BlockingFindingCount: int32(len(capab.GetBlockingFindingCodes())),
			BlockingFindingCodes: append([]string(nil), capab.GetBlockingFindingCodes()...),
			PriorityRank:         capab.GetPriorityRank(),
			PriorityReason:       capab.GetPriorityReason(),
		}
		if lvl := levelByID(capab.GetLevels(), capab.GetCurrentLevel()); lvl != nil {
			cs.CurrentLevelLabel = firstNonEmpty(lvl.GetStatusLabel(), lvl.GetName())
		}
		out = append(out, cs)
	}
	return out
}

// buildFindingsSummary tallies the provider findings by severity. Always returns
// a non-nil summary (all-zero for a clean phase) so consumers can distinguish
// "clean" from "no maturity ladder" (nil standing).
func buildFindingsSummary(s Summary) *runspb.PhaseFindingsSummary {
	return &runspb.PhaseFindingsSummary{
		Blockers: int32(s.Blockers),
		Errors:   int32(s.Errors),
		Warnings: int32(s.Warnings),
		Infos:    int32(s.Infos),
		Total:    int32(s.Blockers + s.Errors + s.Warnings + s.Infos),
	}
}

func levelByID(levels []*commonv1.LocalMaturityLevel, id string) *commonv1.LocalMaturityLevel {
	if strings.TrimSpace(id) == "" {
		return nil
	}
	for _, lvl := range levels {
		if lvl.GetId() == id {
			return lvl
		}
	}
	return nil
}

func findCapability(capabilities []*commonv1.CapabilityMaturityAssessment, id string) *commonv1.CapabilityMaturityAssessment {
	for _, capab := range capabilities {
		if capab.GetId() == id {
			return capab
		}
	}
	return nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
