// Package recordings hosts the BAS RecordingsService Connect-RPC handler.
//
// RecordingsService owns every sub-resource hanging off
// /recordings/sessions/{profile_id}/...:
//
//   - Storage state (cookies + localStorage)
//   - Service workers (live, via playwright-driver)
//   - Browser history (persisted, with playwright-driver re-navigation)
//   - Saved tabs (for session restoration)
//
// Session-profile CRUD itself lives on SessionProfilesService. The recording
// archive multipart-upload and WebSocket frame streams stay as documented REST
// exceptions; see docs/internal/REST_EXCEPTIONS.md.
package recordings

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/connectx"

	autodriver "github.com/vrooli/browser-automation-studio/automation/driver"
	sessionprofile "github.com/vrooli/browser-automation-studio/services/session-profile"
	sessionprofilepersistence "github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
	recordingsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/recordings/recordingsconnect"
)

// SessionProfileRepo is the narrow seam over services/session-profile.Service
// used by the RecordingsService handler. Tests inject a fake; production passes
// *sessionprofile.Service.
type SessionProfileRepo interface {
	GetProfile(id sessionprofilepersistence.ProfileID) (*sessionprofilepersistence.SessionProfile, error)
	SaveStorageState(id sessionprofilepersistence.ProfileID, storageState []byte) (*sessionprofilepersistence.SessionProfile, error)
	MaskStorageState(raw []byte) (*sessionprofile.MaskedStorageState, error)
	GetHistoryWithPruning(id sessionprofilepersistence.ProfileID) ([]sessionprofilepersistence.HistoryEntry, *sessionprofilepersistence.HistorySettings, error)
	ClearHistory(id sessionprofilepersistence.ProfileID) (*sessionprofilepersistence.SessionProfile, error)
	DeleteHistoryEntry(id sessionprofilepersistence.ProfileID, entryID string) (*sessionprofilepersistence.SessionProfile, error)
	UpdateHistorySettings(id sessionprofilepersistence.ProfileID, settings *sessionprofilepersistence.HistorySettings) (*sessionprofilepersistence.SessionProfile, error)
	GetSessionForProfile(profileID string) string
	SaveOpenTabs(id sessionprofilepersistence.ProfileID, tabs []sessionprofilepersistence.TabState) (*sessionprofilepersistence.SessionProfile, error)
}

// RecordModeService is the narrow seam over the live record-mode service used
// by the RecordingsService handler.
type RecordModeService interface {
	GetServiceWorkers(ctx context.Context, sessionID string) (*autodriver.GetServiceWorkersResponse, error)
	UnregisterAllServiceWorkers(ctx context.Context, sessionID string) (*autodriver.UnregisterServiceWorkersResponse, error)
	UnregisterServiceWorker(ctx context.Context, sessionID, scopeURL string) (*autodriver.UnregisterServiceWorkerResponse, error)
	DriverClient() autodriver.ClientInterface
}

// Deps wires the recordings handler. All four are required: a missing dep
// would silently disable a sub-area; the panic surfaces at boot.
type Deps struct {
	Repo       SessionProfileRepo
	RecordMode RecordModeService
	Logger     *logrus.Logger
}

// Module builds the RecordingsService Connect handler and returns it wrapped
// in a connectx.ServiceMount ready for connectx.RegisterChi.
func Module(d Deps) connectx.ServiceMount {
	if d.Logger == nil {
		panic("recordings.Module requires Deps.Logger")
	}
	if d.Repo == nil {
		panic("recordings.Module requires Deps.Repo")
	}
	if d.RecordMode == nil {
		panic("recordings.Module requires Deps.RecordMode")
	}
	path, handler := recordingsconnect.NewRecordingsServiceHandler(&service{deps: d})
	return connectx.ServiceMount{Path: path, Handler: handler}
}

var _ recordingsconnect.RecordingsServiceHandler = (*service)(nil)
