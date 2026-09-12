// Package session_profiles hosts the BAS SessionProfilesService Connect-RPC handler.
//
// SessionProfilesService owns CRUD on persisted browser session profiles
// surfaced under Settings → Recording Sessions and the recording UI's profile
// picker. Storage/cookies/service-workers/history/tabs sub-resources live on
// the (forthcoming) RecordingsService.
package session_profiles

import (
	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/connectx"

	sessionprofilepersistence "github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
	sessionprofilesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/session_profiles/session_profilesconnect"
)

// Repo is the narrow seam over services/session-profile.Service that the
// SessionProfilesService handler needs.
type Repo interface {
	ListProfiles() ([]sessionprofilepersistence.SessionProfile, error)
	CreateProfile(name string) (*sessionprofilepersistence.SessionProfile, error)
	RenameProfile(id sessionprofilepersistence.ProfileID, name string) (*sessionprofilepersistence.SessionProfile, error)
	UpdateBrowserProfile(id sessionprofilepersistence.ProfileID, profile *sessionprofilepersistence.BrowserProfile) (*sessionprofilepersistence.SessionProfile, error)
	DeleteProfile(id sessionprofilepersistence.ProfileID) error
}

// Deps wires the session_profiles handler.
type Deps struct {
	Repo   Repo
	Logger *logrus.Logger
}

// Module builds the SessionProfilesService Connect handler.
func Module(d Deps) connectx.ServiceMount {
	if d.Logger == nil {
		panic("session_profiles.Module requires Deps.Logger")
	}
	if d.Repo == nil {
		panic("session_profiles.Module requires Deps.Repo")
	}
	path, handler := sessionprofilesconnect.NewSessionProfilesServiceHandler(&service{deps: d})
	return connectx.ServiceMount{Path: path, Handler: handler}
}

var _ sessionprofilesconnect.SessionProfilesServiceHandler = (*service)(nil)
