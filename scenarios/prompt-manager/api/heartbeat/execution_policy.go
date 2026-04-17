package heartbeat

import (
	"errors"
	"fmt"
	"prompt-manager/store"
	"prompt-manager/teamconfig"
)

// TeamDisabledError indicates a heartbeat execution was blocked because the team is off.
// This is a policy-level error surfaced to handlers so they can return a clear response.
type TeamDisabledError struct {
	TeamID string
}

func (e *TeamDisabledError) Error() string {
	return fmt.Sprintf("team %s is disabled", e.TeamID)
}

// IsTeamDisabled reports whether an error (possibly wrapped) indicates the team is disabled.
func IsTeamDisabled(err error) bool {
	var disabled *TeamDisabledError
	return errors.As(err, &disabled)
}

// validateTeamEnabled returns a TeamDisabledError when the team is not enabled.
func validateTeamEnabled(team *store.Team) error {
	if team == nil {
		return fmt.Errorf("team not found")
	}
	if !team.Enabled {
		return &TeamDisabledError{TeamID: team.ID}
	}
	return nil
}

// ProfileMismatchError indicates a profile's runner type is incompatible with
// the team's runtime mode.
type ProfileMismatchError struct {
	TeamID      string
	RuntimeMode string
	ProfileKey  string
	RunnerType  string
}

func (e *ProfileMismatchError) Error() string {
	return fmt.Sprintf(
		"profile %q uses runner %s which is incompatible with team %s runtime mode %q; single-process requires RUNNER_TYPE_CLAUDE_CODE",
		e.ProfileKey, e.RunnerType, e.TeamID, e.RuntimeMode,
	)
}

// IsProfileMismatch reports whether an error (possibly wrapped) is a ProfileMismatchError.
func IsProfileMismatch(err error) bool {
	var mismatch *ProfileMismatchError
	return errors.As(err, &mismatch)
}

// validateProfileCompatibility checks that a resolved profile's runner type
// is compatible with the team's runtime mode. Single-process teams require
// RUNNER_TYPE_CLAUDE_CODE. Multi-process teams allow any runner.
func validateProfileCompatibility(team *store.Team, profile *AgentProfile) error {
	if team.Runtime.Mode == teamconfig.RuntimeModeSingleProcess && profile.RunnerType != "RUNNER_TYPE_CLAUDE_CODE" {
		return &ProfileMismatchError{
			TeamID:      team.ID,
			RuntimeMode: team.Runtime.Mode,
			ProfileKey:  profile.ProfileKey,
			RunnerType:  profile.RunnerType,
		}
	}
	return nil
}
