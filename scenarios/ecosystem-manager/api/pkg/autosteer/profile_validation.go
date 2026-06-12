package autosteer

import (
	"fmt"
	"strings"

	"github.com/vrooli/maturity-go/dimensions"
	"github.com/vrooli/maturity-go/ladder"
)

// validMaxOpenSeverity is the accepted set of max-open-severity target values.
var validMaxOpenSeverity = map[string]struct{}{
	"": {}, "none": {},
	"info": {}, "low": {},
	"warning": {}, "warn": {}, "medium": {},
	"error": {}, "high": {},
	"blocker": {}, "critical": {},
}

// ValidateProfile validates an objective-function profile in place (normalizing
// local restriction masks). Catalog-aware mask reconciliation lives in
// ReconcileProfile.
func ValidateProfile(profile *AutoSteerProfile) error {
	if profile == nil {
		return fmt.Errorf("profile is required")
	}
	if strings.TrimSpace(profile.Name) == "" {
		return fmt.Errorf("profile name is required")
	}

	normalizedAllowed, err := normalizeSkillIDsStrict(profile.AllowedSkills, "allowed_skills")
	if err != nil {
		return err
	}
	profile.AllowedSkills = normalizedAllowed
	normalizedDenied, err := normalizeSkillIDsStrict(profile.DeniedSkills, "denied_skills")
	if err != nil {
		return err
	}
	profile.DeniedSkills = normalizedDenied

	// Dimension weights must reference the canonical vocabulary and be non-negative.
	for dim, weight := range profile.Objective.DimensionWeights {
		if !dimensions.IsValid(dimensions.Dimension(dim)) {
			return fmt.Errorf("objective.dimension_weights references unknown dimension %q", dim)
		}
		if weight < 0 {
			return fmt.Errorf("objective.dimension_weights[%q] must be non-negative", dim)
		}
	}

	// Targets.
	if _, ok := validMaxOpenSeverity[strings.ToLower(strings.TrimSpace(profile.Objective.Targets.MaxOpenSeverity))]; !ok {
		return fmt.Errorf("objective.targets.max_open_severity %q is not a recognized severity", profile.Objective.Targets.MaxOpenSeverity)
	}
	if pct := profile.Objective.Targets.OperationalTargetsPct; pct < 0 || pct > 100 {
		return fmt.Errorf("objective.targets.operational_targets_pct must be between 0 and 100")
	}

	// Budget.
	if profile.Budget.MaxIterations <= 0 {
		return fmt.Errorf("budget.max_iterations must be > 0")
	}
	if profile.Budget.DiminishingReturnsFloor < 0 {
		return fmt.Errorf("budget.diminishing_returns_floor must be non-negative")
	}
	if profile.Budget.ReauditCadence < 0 {
		return fmt.Errorf("budget.reaudit_cadence must be non-negative")
	}

	// Maturity ladder (optional). The rung definitions are canonical in the shared
	// maturity-go/ladder package; here we only validate the profile's tuning of them.
	if l := profile.Ladder; l != nil {
		if strings.TrimSpace(l.TopRung) != "" {
			if _, ok := ladder.ParseRung(l.TopRung); !ok {
				return fmt.Errorf("ladder.top_rung %q is not a recognized rung (R0–R4)", l.TopRung)
			}
		}
		if l.BoostFactor < 0 {
			return fmt.Errorf("ladder.boost_factor must be non-negative")
		}
		if l.StandardsMaxCount < 0 {
			return fmt.Errorf("ladder.standards_max_count must be non-negative")
		}
		if l.StructureMaxCount < 0 {
			return fmt.Errorf("ladder.structure_max_count must be non-negative")
		}
	}
	if len(profile.Objective.DimensionWeights) == 0 && !profile.ladderEnabled() {
		return fmt.Errorf("profile must declare at least one objective.dimension_weights entry or enable ladder")
	}

	// Baseline Modes promote block (optional). Normalizes the mode in place so an
	// empty value transparently means the end_of_engagement default downstream.
	if bp := profile.BaselinePromote; bp != nil {
		mode := strings.ToLower(strings.TrimSpace(bp.Mode))
		switch mode {
		case "", BaselinePromoteEndOfEngagement, BaselinePromoteCheckpointOnGreen:
		default:
			return fmt.Errorf("baseline_promote.mode %q is not recognized (end_of_engagement|checkpoint_on_green)", bp.Mode)
		}
		bp.Mode = mode
		if bp.CadenceIter < 0 {
			return fmt.Errorf("baseline_promote.cadence_iter must be non-negative")
		}
	}

	return nil
}
