package livedesktop

import (
	"fmt"
	"strings"

	"scenario-to-desktop-api/target"
)

// requestValidationError marks a failure caused by explicit validation input.
// Operational errors may contain the same words and must keep their server
// error status at the HTTP boundary.
type requestValidationError struct {
	message string
}

func (e *requestValidationError) Error() string { return e.message }

func validateElectronValidationRequest(opts target.ElectronLaunchOptions, renderer target.RendererExpectation) error {
	for _, field := range []struct {
		name  string
		value string
	}{
		{name: "context_id", value: opts.ContextID},
		{name: "artifact_digest", value: opts.ArtifactDigest},
		{name: "target_id", value: opts.TargetID},
		{name: "journey_id", value: opts.JourneyID},
		{name: "profile_id", value: opts.ProfileID},
		{name: "isolation_lease_id", value: opts.IsolationLeaseID},
	} {
		if strings.TrimSpace(field.value) == "" {
			return &requestValidationError{message: fmt.Sprintf("electron validation %s is required", field.name)}
		}
	}
	if strings.TrimSpace(renderer.ID) == "" && strings.TrimSpace(renderer.URLPrefix) == "" && strings.TrimSpace(renderer.Title) == "" {
		return &requestValidationError{message: "renderer identity is required; refusing to select an attachable page by order"}
	}
	return nil
}
