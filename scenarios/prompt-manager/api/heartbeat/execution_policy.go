package heartbeat

import (
	"errors"
	"fmt"
	"prompt-manager/store"
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
