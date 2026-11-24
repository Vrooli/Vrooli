package workflow

import (
	"encoding/json"

	sessionprofilepersistence "github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
)

type executionSessionProfile struct {
	storageState          json.RawMessage
	browserProfile        *sessionprofilepersistence.BrowserProfile
	sessionProfileVersion string
	openTabs              []sessionprofilepersistence.TabState
}

// resolveExecutionSessionProfile applies the same saved-session and browser
// profile rules to saved and ad-hoc workflow executions.
func (s *WorkflowService) resolveExecutionSessionProfile(
	flow *basworkflows.WorkflowDefinitionV2,
	sessionProfileID string,
	saveSessionProfileID string,
	restoreTabs bool,
	executionBrowserProfile *sessionprofilepersistence.BrowserProfile,
) (executionSessionProfile, error) {
	resolved := executionSessionProfile{}
	var profileBrowserSettings *sessionprofilepersistence.BrowserProfile
	if sessionProfileID != "" && s.sessionProfileService != nil {
		profile, err := s.sessionProfileService.GetProfile(sessionprofilepersistence.ProfileID(sessionProfileID))
		if err != nil {
			return executionSessionProfile{}, err
		}
		resolved.storageState = profile.StorageState
		profileBrowserSettings = profile.BrowserProfile
		resolved.sessionProfileVersion = profileVersionForReuse(profile)
		if restoreTabs && len(profile.OpenTabs) > 0 {
			resolved.openTabs = profile.OpenTabs
		}
		if _, err := s.sessionProfileService.Touch(sessionprofilepersistence.ProfileID(sessionProfileID)); err != nil && s.log != nil {
			s.log.WithError(err).WithField("session_profile_id", sessionProfileID).Warn("Failed to update session profile usage timestamp")
		}
	}
	if resolved.sessionProfileVersion == "" && saveSessionProfileID != "" {
		resolved.sessionProfileVersion = profileVersionForReuse(&sessionprofilepersistence.SessionProfile{ID: sessionprofilepersistence.ProfileID(saveSessionProfileID)})
	}

	var workflowBrowserProfile *sessionprofilepersistence.BrowserProfile
	if flow != nil && flow.Settings != nil && flow.Settings.BrowserProfile != nil {
		workflowBrowserProfile = sessionprofilepersistence.BrowserProfileFromProto(flow.Settings.BrowserProfile)
	}
	baseBrowserProfile := sessionprofilepersistence.MergeBrowserProfiles(workflowBrowserProfile, profileBrowserSettings)
	resolved.browserProfile = sessionprofilepersistence.MergeBrowserProfiles(baseBrowserProfile, executionBrowserProfile)
	return resolved, nil
}
