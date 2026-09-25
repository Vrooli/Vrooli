package recordings

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	autodriver "github.com/vrooli/browser-automation-studio/automation/driver"
	autosession "github.com/vrooli/browser-automation-studio/automation/session"
	sessionprofile "github.com/vrooli/browser-automation-studio/services/session-profile"
	sessionprofilepersistence "github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
	recordingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/recordings"
	recordingsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/recordings/recordingsconnect"
)

func TestGetStorageStateRejectsPathID(t *testing.T) {
	root := t.TempDir()
	repo := sessionprofilepersistence.NewFileRepository(root, nil)
	client, cleanup := newTestServer(t, sessionprofile.NewService(repo, logrus.New()), nil)
	defer cleanup()
	_, err := client.GetStorageState(context.Background(), connect.NewRequest(&recordingsv1.GetStorageStateRequest{ProfileId: "../outside"}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	require.Empty(t, entries)
}

// =============================================================================
// Fakes
// =============================================================================

type fakeRepo struct {
	mu       sync.Mutex
	items    map[sessionprofilepersistence.ProfileID]*sessionprofilepersistence.SessionProfile
	sessions map[string]string // profileID -> sessionID
	storage  sessionprofile.MaskedStorageState
	maskErr  error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		items:    map[sessionprofilepersistence.ProfileID]*sessionprofilepersistence.SessionProfile{},
		sessions: map[string]string{},
	}
}

func (f *fakeRepo) put(p *sessionprofilepersistence.SessionProfile) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.items[p.ID] = p
}

func (f *fakeRepo) GetProfile(id sessionprofilepersistence.ProfileID) (*sessionprofilepersistence.SessionProfile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.items[id]
	if !ok {
		return nil, errors.New("session profile not found")
	}
	return p, nil
}

func (f *fakeRepo) UpdateProfile(id sessionprofilepersistence.ProfileID, modify func(*sessionprofilepersistence.SessionProfile) error) (*sessionprofilepersistence.SessionProfile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.items[id]
	if !ok {
		return nil, errProfileNotFound
	}
	if err := modify(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (f *fakeRepo) UpdateStorageState(id sessionprofilepersistence.ProfileID, modify func(json.RawMessage) (json.RawMessage, error)) (*sessionprofilepersistence.SessionProfile, error) {
	return f.UpdateProfile(id, func(profile *sessionprofilepersistence.SessionProfile) error {
		state, err := modify(profile.StorageState)
		if err != nil {
			return err
		}
		profile.StorageState = state
		return nil
	})
}

func (f *fakeRepo) SaveStorageState(id sessionprofilepersistence.ProfileID, state []byte) (*sessionprofilepersistence.SessionProfile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.items[id]
	if !ok {
		return nil, errors.New("session profile not found")
	}
	p.StorageState = append([]byte(nil), state...)
	return p, nil
}

func (f *fakeRepo) MaskStorageState(_ []byte) (*sessionprofile.MaskedStorageState, error) {
	if f.maskErr != nil {
		return nil, f.maskErr
	}
	out := f.storage
	return &out, nil
}

func (f *fakeRepo) GetHistoryWithPruning(id sessionprofilepersistence.ProfileID) ([]sessionprofilepersistence.HistoryEntry, *sessionprofilepersistence.HistorySettings, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.items[id]
	if !ok {
		return nil, nil, errors.New("session profile not found")
	}
	return p.History, p.HistorySettings, nil
}

func (f *fakeRepo) ClearHistory(id sessionprofilepersistence.ProfileID) (*sessionprofilepersistence.SessionProfile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.items[id]
	if !ok {
		return nil, errors.New("session profile not found")
	}
	p.History = nil
	return p, nil
}

func (f *fakeRepo) DeleteHistoryEntry(id sessionprofilepersistence.ProfileID, entryID string) (*sessionprofilepersistence.SessionProfile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.items[id]
	if !ok {
		return nil, errors.New("session profile not found")
	}
	kept := make([]sessionprofilepersistence.HistoryEntry, 0, len(p.History))
	found := false
	for _, e := range p.History {
		if e.ID == entryID {
			found = true
			continue
		}
		kept = append(kept, e)
	}
	if !found {
		return nil, errors.New("entry not found")
	}
	p.History = kept
	return p, nil
}

func (f *fakeRepo) UpdateHistorySettings(id sessionprofilepersistence.ProfileID, settings *sessionprofilepersistence.HistorySettings) (*sessionprofilepersistence.SessionProfile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.items[id]
	if !ok {
		return nil, errors.New("session profile not found")
	}
	if settings.MaxEntries < 0 || settings.MaxEntries > 10000 {
		return nil, errors.New("maxEntries must be between 0 and 10000")
	}
	p.HistorySettings = settings
	return p, nil
}

func (f *fakeRepo) ResolveSessionForProfile(profileID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sessions[profileID], nil
}

func (f *fakeRepo) SaveOpenTabs(id sessionprofilepersistence.ProfileID, tabs []sessionprofilepersistence.TabState) (*sessionprofilepersistence.SessionProfile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.items[id]
	if !ok {
		return nil, errors.New("session profile not found")
	}
	p.OpenTabs = tabs
	return p, nil
}

type fakeRecordMode struct {
	session         *autosession.Session
	swResp          *autodriver.GetServiceWorkersResponse
	swErr           error
	clearResp       *autodriver.UnregisterServiceWorkersResponse
	deleteResp      *autodriver.UnregisterServiceWorkerResponse
	deleteErr       error
	workerCalls     int
	clearCalls      int
	deleteCalls     int
	getSessionCalls int
}

func (f *fakeRecordMode) GetServiceWorkers(_ context.Context, _ string) (*autodriver.GetServiceWorkersResponse, error) {
	f.workerCalls++
	if f.swErr != nil {
		return nil, f.swErr
	}
	return f.swResp, nil
}

func (f *fakeRecordMode) UnregisterAllServiceWorkers(_ context.Context, sessionID string) (*autodriver.UnregisterServiceWorkersResponse, error) {
	f.clearCalls++
	if f.clearResp == nil {
		return &autodriver.UnregisterServiceWorkersResponse{SessionID: sessionID, UnregisteredCount: 0}, nil
	}
	return f.clearResp, nil
}

func (f *fakeRecordMode) UnregisterServiceWorker(_ context.Context, sessionID, scopeURL string) (*autodriver.UnregisterServiceWorkerResponse, error) {
	f.deleteCalls++
	if f.deleteErr != nil {
		return nil, f.deleteErr
	}
	if f.deleteResp == nil {
		return &autodriver.UnregisterServiceWorkerResponse{SessionID: sessionID, Unregistered: scopeURL}, nil
	}
	return f.deleteResp, nil
}

func (f *fakeRecordMode) GetSession(string) (*autosession.Session, bool) {
	f.getSessionCalls++
	return f.session, f.session != nil
}

// =============================================================================
// Test harness
// =============================================================================

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

func newTestServer(t *testing.T, repo SessionProfileRepo, rm RecordModeService) (recordingsconnect.RecordingsServiceClient, func()) {
	t.Helper()
	log := logrus.New()
	log.SetOutput(discardWriter{})
	if rm == nil {
		rm = &fakeRecordMode{}
	}
	mount := Module(Deps{Repo: repo, RecordMode: rm, Logger: log})
	mux := http.NewServeMux()
	mux.Handle(mount.Path, mount.Handler)
	srv := httptest.NewServer(mux)
	client := recordingsconnect.NewRecordingsServiceClient(srv.Client(), srv.URL)
	return client, srv.Close
}

func newProfile(id string) *sessionprofilepersistence.SessionProfile {
	now := time.Now().UTC()
	return &sessionprofilepersistence.SessionProfile{
		ID:         sessionprofilepersistence.ProfileID(id),
		Name:       "Test",
		CreatedAt:  now,
		UpdatedAt:  now,
		LastUsedAt: now,
	}
}

// =============================================================================
// Module wiring
// =============================================================================

func TestModule_PanicsOnMissingDeps(t *testing.T) {
	require.PanicsWithValue(t, "recordings.Module requires Deps.Logger", func() { Module(Deps{}) })
	require.PanicsWithValue(t, "recordings.Module requires Deps.Repo", func() {
		Module(Deps{Logger: logrus.New()})
	})
	require.PanicsWithValue(t, "recordings.Module requires Deps.RecordMode", func() {
		Module(Deps{Logger: logrus.New(), Repo: newFakeRepo()})
	})
}

// =============================================================================
// Storage state
// =============================================================================

func TestGetStorageState_HappyPath(t *testing.T) {
	repo := newFakeRepo()
	p := newProfile("prof-1")
	repo.put(p)
	repo.storage = sessionprofile.MaskedStorageState{
		Cookies: []sessionprofile.MaskedCookie{{Name: "sid", Value: "***", ValueMasked: true, Domain: "x.io"}},
		Origins: []sessionprofile.MaskedOrigin{{Origin: "https://x.io", LocalStorage: []sessionprofile.MaskedLocalStorageItem{{Name: "k", Value: "v"}}}},
		Stats:   sessionprofile.MaskedStorageStats{CookieCount: 1, LocalStorageCount: 1, OriginCount: 1},
	}
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()

	resp, err := client.GetStorageState(context.Background(), connect.NewRequest(&recordingsv1.GetStorageStateRequest{ProfileId: "prof-1"}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetCookies(), 1)
	require.True(t, resp.Msg.GetCookies()[0].GetValueMasked())
	require.Equal(t, int32(1), resp.Msg.GetStats().GetCookieCount())
}

func TestGetStorageState_MissingProfileID(t *testing.T) {
	repo := newFakeRepo()
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()
	_, err := client.GetStorageState(context.Background(), connect.NewRequest(&recordingsv1.GetStorageStateRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestGetStorageState_NotFound(t *testing.T) {
	repo := newFakeRepo()
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()
	_, err := client.GetStorageState(context.Background(), connect.NewRequest(&recordingsv1.GetStorageStateRequest{ProfileId: "missing"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestClearAllStorage_HappyPath(t *testing.T) {
	repo := newFakeRepo()
	p := newProfile("prof-1")
	p.StorageState = []byte(`{"cookies":[{"name":"x","domain":"y.io"}],"origins":[]}`)
	repo.put(p)
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()
	resp, err := client.ClearAllStorage(context.Background(), connect.NewRequest(&recordingsv1.ClearAllStorageRequest{ProfileId: "prof-1"}))
	require.NoError(t, err)
	require.Equal(t, "cleared", resp.Msg.GetStatus())
	require.JSONEq(t, `{"cookies":[],"origins":[]}`, string(p.StorageState))
}

func TestClearAllStorage_NotFound(t *testing.T) {
	repo := newFakeRepo()
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()
	_, err := client.ClearAllStorage(context.Background(), connect.NewRequest(&recordingsv1.ClearAllStorageRequest{ProfileId: "ghost"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestClearAllStorage_MissingProfileID(t *testing.T) {
	repo := newFakeRepo()
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()
	_, err := client.ClearAllStorage(context.Background(), connect.NewRequest(&recordingsv1.ClearAllStorageRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestClearAllCookies_RemovesCookiesKeepsOrigins(t *testing.T) {
	repo := newFakeRepo()
	p := newProfile("p")
	p.StorageState = []byte(`{"cookies":[{"name":"a","domain":"x"}],"origins":[{"origin":"o","localStorage":[{"name":"k","value":"v"}]}]}`)
	repo.put(p)
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()
	_, err := client.ClearAllCookies(context.Background(), connect.NewRequest(&recordingsv1.ClearAllCookiesRequest{ProfileId: "p"}))
	require.NoError(t, err)
	var st playwrightStorageState
	require.NoError(t, json.Unmarshal(p.StorageState, &st))
	require.Empty(t, st.Cookies)
	require.Len(t, st.Origins, 1)
}

func TestDeleteCookiesByDomain_FiltersByDomain(t *testing.T) {
	repo := newFakeRepo()
	p := newProfile("p")
	p.StorageState = []byte(`{"cookies":[{"name":"a","domain":"x"},{"name":"b","domain":"y"}],"origins":[]}`)
	repo.put(p)
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()

	// missing domain → InvalidArgument
	_, err := client.DeleteCookiesByDomain(context.Background(), connect.NewRequest(&recordingsv1.DeleteCookiesByDomainRequest{ProfileId: "p"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	_, err = client.DeleteCookiesByDomain(context.Background(), connect.NewRequest(&recordingsv1.DeleteCookiesByDomainRequest{ProfileId: "p", Domain: "x"}))
	require.NoError(t, err)
	var st playwrightStorageState
	require.NoError(t, json.Unmarshal(p.StorageState, &st))
	require.Len(t, st.Cookies, 1)
	require.Equal(t, "y", st.Cookies[0].Domain)
}

func TestDeleteCookie_RemovesSpecific(t *testing.T) {
	repo := newFakeRepo()
	p := newProfile("p")
	p.StorageState = []byte(`{"cookies":[{"name":"a","domain":"x"},{"name":"b","domain":"x"}],"origins":[]}`)
	repo.put(p)
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()
	_, err := client.DeleteCookie(context.Background(), connect.NewRequest(&recordingsv1.DeleteCookieRequest{ProfileId: "p", Domain: "x", Name: "a"}))
	require.NoError(t, err)
	var st playwrightStorageState
	require.NoError(t, json.Unmarshal(p.StorageState, &st))
	require.Len(t, st.Cookies, 1)
	require.Equal(t, "b", st.Cookies[0].Name)
}

func TestClearAllLocalStorage_RemovesOriginsKeepsCookies(t *testing.T) {
	repo := newFakeRepo()
	p := newProfile("p")
	p.StorageState = []byte(`{"cookies":[{"name":"a","domain":"x"}],"origins":[{"origin":"o","localStorage":[]}]}`)
	repo.put(p)
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()
	_, err := client.ClearAllLocalStorage(context.Background(), connect.NewRequest(&recordingsv1.ClearAllLocalStorageRequest{ProfileId: "p"}))
	require.NoError(t, err)
	var st playwrightStorageState
	require.NoError(t, json.Unmarshal(p.StorageState, &st))
	require.Empty(t, st.Origins)
	require.Len(t, st.Cookies, 1)
}

func TestDeleteLocalStorageByOrigin(t *testing.T) {
	repo := newFakeRepo()
	p := newProfile("p")
	p.StorageState = []byte(`{"cookies":[],"origins":[{"origin":"a","localStorage":[]},{"origin":"b","localStorage":[]}]}`)
	repo.put(p)
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()
	// missing origin → InvalidArgument
	_, err := client.DeleteLocalStorageByOrigin(context.Background(), connect.NewRequest(&recordingsv1.DeleteLocalStorageByOriginRequest{ProfileId: "p"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	_, err = client.DeleteLocalStorageByOrigin(context.Background(), connect.NewRequest(&recordingsv1.DeleteLocalStorageByOriginRequest{ProfileId: "p", Origin: "a"}))
	require.NoError(t, err)
	var st playwrightStorageState
	require.NoError(t, json.Unmarshal(p.StorageState, &st))
	require.Len(t, st.Origins, 1)
	require.Equal(t, "b", st.Origins[0].Origin)
}

func TestDeleteLocalStorageItem_RemovesAndPrunesEmptyOrigin(t *testing.T) {
	repo := newFakeRepo()
	p := newProfile("p")
	p.StorageState = []byte(`{"cookies":[],"origins":[{"origin":"o","localStorage":[{"name":"a","value":"1"},{"name":"b","value":"2"}]}]}`)
	repo.put(p)
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()
	_, err := client.DeleteLocalStorageItem(context.Background(), connect.NewRequest(&recordingsv1.DeleteLocalStorageItemRequest{ProfileId: "p", Origin: "o", Name: "a"}))
	require.NoError(t, err)
	var st playwrightStorageState
	require.NoError(t, json.Unmarshal(p.StorageState, &st))
	require.Len(t, st.Origins, 1)
	require.Len(t, st.Origins[0].LocalStorage, 1)
	require.Equal(t, "b", st.Origins[0].LocalStorage[0].Name)

	// removing the last key prunes the origin
	_, err = client.DeleteLocalStorageItem(context.Background(), connect.NewRequest(&recordingsv1.DeleteLocalStorageItemRequest{ProfileId: "p", Origin: "o", Name: "b"}))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(p.StorageState, &st))
	require.Empty(t, st.Origins)
}

// =============================================================================
// Service workers
// =============================================================================

func TestGetServiceWorkers_NoActiveSession(t *testing.T) {
	repo := newFakeRepo()
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()
	resp, err := client.GetServiceWorkers(context.Background(), connect.NewRequest(&recordingsv1.GetServiceWorkersRequest{ProfileId: "p"}))
	require.NoError(t, err)
	require.Empty(t, resp.Msg.GetSessionId())
	require.Equal(t, "allow", resp.Msg.GetControl().GetMode())
	require.Contains(t, resp.Msg.GetMessage(), "No active session")
}

func TestGetServiceWorkers_HappyPath(t *testing.T) {
	repo := newFakeRepo()
	repo.sessions["p"] = "sess-1"
	rm := &fakeRecordMode{
		swResp: &autodriver.GetServiceWorkersResponse{
			SessionID: "sess-1",
			Workers:   []autodriver.ServiceWorkerInfo{{RegistrationID: "r1", ScopeURL: "https://x", Status: "running"}},
			Control:   autodriver.ServiceWorkerControl{Mode: "allow", DomainOverrides: []autodriver.ServiceWorkerDomainOverride{{Domain: "x", Mode: "block"}}},
		},
	}
	client, cleanup := newTestServer(t, repo, rm)
	defer cleanup()
	resp, err := client.GetServiceWorkers(context.Background(), connect.NewRequest(&recordingsv1.GetServiceWorkersRequest{ProfileId: "p"}))
	require.NoError(t, err)
	require.Equal(t, "sess-1", resp.Msg.GetSessionId())
	require.Len(t, resp.Msg.GetWorkers(), 1)
	require.Len(t, resp.Msg.GetControl().GetDomainOverrides(), 1)
}

func TestGetServiceWorkers_DriverError(t *testing.T) {
	repo := newFakeRepo()
	repo.sessions["p"] = "sess-1"
	rm := &fakeRecordMode{swErr: errors.New("driver down")}
	client, cleanup := newTestServer(t, repo, rm)
	defer cleanup()
	_, err := client.GetServiceWorkers(context.Background(), connect.NewRequest(&recordingsv1.GetServiceWorkersRequest{ProfileId: "p"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

func TestGetServiceWorkers_MissingProfileID(t *testing.T) {
	repo := newFakeRepo()
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()
	_, err := client.GetServiceWorkers(context.Background(), connect.NewRequest(&recordingsv1.GetServiceWorkersRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestClearAllServiceWorkers_NoActiveSession(t *testing.T) {
	repo := newFakeRepo()
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()
	resp, err := client.ClearAllServiceWorkers(context.Background(), connect.NewRequest(&recordingsv1.ClearAllServiceWorkersRequest{ProfileId: "p"}))
	require.NoError(t, err)
	require.Empty(t, resp.Msg.GetSessionId())
}

func TestDeleteServiceWorker_NoActiveSession(t *testing.T) {
	repo := newFakeRepo()
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()
	_, err := client.DeleteServiceWorker(context.Background(), connect.NewRequest(&recordingsv1.DeleteServiceWorkerRequest{ProfileId: "p", ScopeUrl: "https://x"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestDeleteServiceWorker_MissingScopeURL(t *testing.T) {
	repo := newFakeRepo()
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()
	_, err := client.DeleteServiceWorker(context.Background(), connect.NewRequest(&recordingsv1.DeleteServiceWorkerRequest{ProfileId: "p"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestDeleteServiceWorker_DriverError(t *testing.T) {
	repo := newFakeRepo()
	repo.sessions["p"] = "sess-1"
	rm := &fakeRecordMode{deleteResp: &autodriver.UnregisterServiceWorkerResponse{SessionID: "sess-1", Error: "not registered"}}
	client, cleanup := newTestServer(t, repo, rm)
	defer cleanup()
	_, err := client.DeleteServiceWorker(context.Background(), connect.NewRequest(&recordingsv1.DeleteServiceWorkerRequest{ProfileId: "p", ScopeUrl: "https://x"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestProfileScopedLiveOperationsRejectAmbiguousBindings(t *testing.T) {
	operations := []struct {
		name string
		call func(recordingsconnect.RecordingsServiceClient) error
	}{
		{
			name: "get service workers",
			call: func(client recordingsconnect.RecordingsServiceClient) error {
				_, err := client.GetServiceWorkers(context.Background(), connect.NewRequest(&recordingsv1.GetServiceWorkersRequest{ProfileId: "shared"}))
				return err
			},
		},
		{
			name: "clear service workers",
			call: func(client recordingsconnect.RecordingsServiceClient) error {
				_, err := client.ClearAllServiceWorkers(context.Background(), connect.NewRequest(&recordingsv1.ClearAllServiceWorkersRequest{ProfileId: "shared"}))
				return err
			},
		},
		{
			name: "delete service worker",
			call: func(client recordingsconnect.RecordingsServiceClient) error {
				_, err := client.DeleteServiceWorker(context.Background(), connect.NewRequest(&recordingsv1.DeleteServiceWorkerRequest{ProfileId: "shared", ScopeUrl: "https://fixture.invalid"}))
				return err
			},
		},
		{
			name: "navigate history",
			call: func(client recordingsconnect.RecordingsServiceClient) error {
				_, err := client.NavigateToHistoryURL(context.Background(), connect.NewRequest(&recordingsv1.NavigateToHistoryURLRequest{ProfileId: "shared", Url: "https://fixture.invalid"}))
				return err
			},
		},
	}
	for _, operation := range operations {
		t.Run(operation.name, func(t *testing.T) {
			repo := sessionprofile.NewService(sessionprofilepersistence.NewMockRepository(), nil)
			const profileID = "shared"
			repo.SetActiveSession("browser-a", profileID)
			repo.SetActiveSession("browser-b", profileID)
			rm := &fakeRecordMode{}
			client, cleanup := newTestServer(t, repo, rm)
			defer cleanup()

			err := operation.call(client)
			require.Error(t, err)
			require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
			require.Contains(t, err.Error(), "exactly one active browser session")
			require.Zero(t, rm.workerCalls)
			require.Zero(t, rm.clearCalls)
			require.Zero(t, rm.deleteCalls)
			require.Zero(t, rm.getSessionCalls)
			_, err = repo.ResolveSessionForProfile(profileID)
			require.ErrorIs(t, err, sessionprofile.ErrAmbiguousProfileSession)
			require.Equal(t, profileID, repo.GetActiveSession("browser-a"))
			require.Equal(t, profileID, repo.GetActiveSession("browser-b"))
		})
	}
}

// =============================================================================
// History
// =============================================================================

func TestGetHistory_HappyPath(t *testing.T) {
	repo := newFakeRepo()
	p := newProfile("p")
	p.History = []sessionprofilepersistence.HistoryEntry{
		{ID: "e2", URL: "https://b", Timestamp: "2025-01-02T00:00:00Z"},
		{ID: "e1", URL: "https://a", Timestamp: "2025-01-01T00:00:00Z"},
	}
	p.HistorySettings = sessionprofilepersistence.DefaultHistorySettings()
	repo.put(p)
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()
	resp, err := client.GetHistory(context.Background(), connect.NewRequest(&recordingsv1.GetHistoryRequest{ProfileId: "p"}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetEntries(), 2)
	require.Equal(t, int32(2), resp.Msg.GetStats().GetTotalEntries())
	require.Equal(t, "2025-01-02T00:00:00Z", resp.Msg.GetStats().GetNewestEntry())
	require.Equal(t, "2025-01-01T00:00:00Z", resp.Msg.GetStats().GetOldestEntry())
	require.NotNil(t, resp.Msg.GetSettings())
}

func TestGetHistory_NotFound(t *testing.T) {
	repo := newFakeRepo()
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()
	_, err := client.GetHistory(context.Background(), connect.NewRequest(&recordingsv1.GetHistoryRequest{ProfileId: "missing"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestClearHistory_HappyPath(t *testing.T) {
	repo := newFakeRepo()
	p := newProfile("p")
	p.History = []sessionprofilepersistence.HistoryEntry{{ID: "e1"}}
	repo.put(p)
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()
	resp, err := client.ClearHistory(context.Background(), connect.NewRequest(&recordingsv1.ClearHistoryRequest{ProfileId: "p"}))
	require.NoError(t, err)
	require.Equal(t, "cleared", resp.Msg.GetStatus())
	require.Empty(t, p.History)
}

func TestDeleteHistoryEntry_NotFound(t *testing.T) {
	repo := newFakeRepo()
	p := newProfile("p")
	repo.put(p)
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()
	_, err := client.DeleteHistoryEntry(context.Background(), connect.NewRequest(&recordingsv1.DeleteHistoryEntryRequest{ProfileId: "p", EntryId: "ghost"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestDeleteHistoryEntry_MissingEntryID(t *testing.T) {
	repo := newFakeRepo()
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()
	_, err := client.DeleteHistoryEntry(context.Background(), connect.NewRequest(&recordingsv1.DeleteHistoryEntryRequest{ProfileId: "p"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestUpdateHistorySettings_HappyPath(t *testing.T) {
	repo := newFakeRepo()
	p := newProfile("p")
	repo.put(p)
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()
	resp, err := client.UpdateHistorySettings(context.Background(), connect.NewRequest(&recordingsv1.UpdateHistorySettingsRequest{
		ProfileId: "p",
		Settings:  &recordingsv1.HistorySettings{MaxEntries: 50, RetentionDays: 7, CaptureThumbnails: true},
	}))
	require.NoError(t, err)
	require.Equal(t, int32(50), resp.Msg.GetSettings().GetMaxEntries())
}

func TestUpdateHistorySettings_OutOfRange(t *testing.T) {
	repo := newFakeRepo()
	p := newProfile("p")
	repo.put(p)
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()
	_, err := client.UpdateHistorySettings(context.Background(), connect.NewRequest(&recordingsv1.UpdateHistorySettingsRequest{
		ProfileId: "p",
		Settings:  &recordingsv1.HistorySettings{MaxEntries: 999999},
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestUpdateHistorySettings_MissingSettings(t *testing.T) {
	repo := newFakeRepo()
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()
	_, err := client.UpdateHistorySettings(context.Background(), connect.NewRequest(&recordingsv1.UpdateHistorySettingsRequest{ProfileId: "p"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestNavigateToHistoryURL_HappyPath(t *testing.T) {
	repo := newFakeRepo()
	repo.sessions["p"] = "sess-1"
	owner := uuid.New()
	requests := make(chan map[string]any, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/session/start" {
			_, _ = w.Write([]byte(`{"session_id":"sess-1","lease_id":"history-lease"}`))
			return
		}
		require.Equal(t, "/session/sess-1/record/navigate", r.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		requests <- body
		_, _ = w.Write([]byte(`{"driver_page_id":"initial-driver-page","url":"https://x","title":"X","can_go_back":true}`))
	}))
	defer server.Close()
	driverClient, err := autodriver.NewClientWithURL(server.URL, autodriver.WithoutCircuitBreaker())
	require.NoError(t, err)
	session, err := autosession.NewManagerWithClient(driverClient).Create(context.Background(), autosession.Spec{ExecutionID: owner, Mode: autosession.ModeRecording})
	require.NoError(t, err)
	client, cleanup := newTestServer(t, repo, &fakeRecordMode{session: session})
	defer cleanup()
	resp, err := client.NavigateToHistoryURL(context.Background(), connect.NewRequest(&recordingsv1.NavigateToHistoryURLRequest{
		ProfileId: "p", Url: "https://x", WaitUntil: "load", TimeoutMs: 1000,
	}))
	require.NoError(t, err)
	require.Equal(t, "X", resp.Msg.GetTitle())
	require.True(t, resp.Msg.GetCanGoBack())
	require.Len(t, requests, 1)
	require.Equal(t, map[string]any{"execution_id": owner.String(), "lease_id": "history-lease", "url": "https://x", "wait_until": "load", "timeout_ms": float64(1000)}, <-requests)
}

func TestNavigateToHistoryURL_NoSession(t *testing.T) {
	for _, id := range []string{"", "orphaned-profile-binding"} {
		t.Run(id, func(t *testing.T) {
			repo := newFakeRepo()
			repo.sessions["p"] = id
			client, cleanup := newTestServer(t, repo, nil)
			defer cleanup()
			_, err := client.NavigateToHistoryURL(context.Background(), connect.NewRequest(&recordingsv1.NavigateToHistoryURLRequest{ProfileId: "p", Url: "https://x"}))
			require.Error(t, err)
			require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
		})
	}
}

func TestNavigateToHistoryURL_MissingURL(t *testing.T) {
	repo := newFakeRepo()
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()
	_, err := client.NavigateToHistoryURL(context.Background(), connect.NewRequest(&recordingsv1.NavigateToHistoryURLRequest{ProfileId: "p"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

// =============================================================================
// Tabs
// =============================================================================

func TestGetSessionTabs_HappyPath(t *testing.T) {
	repo := newFakeRepo()
	p := newProfile("p")
	p.OpenTabs = []sessionprofilepersistence.TabState{
		{URL: "https://a", Title: "A", IsActive: true, Order: 0},
		{URL: "https://b", Title: "B", Order: 1},
	}
	repo.put(p)
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()
	resp, err := client.GetSessionTabs(context.Background(), connect.NewRequest(&recordingsv1.GetSessionTabsRequest{ProfileId: "p"}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetTabs(), 2)
	require.True(t, resp.Msg.GetTabs()[0].GetIsActive())
}

func TestGetSessionTabs_NotFound(t *testing.T) {
	repo := newFakeRepo()
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()
	_, err := client.GetSessionTabs(context.Background(), connect.NewRequest(&recordingsv1.GetSessionTabsRequest{ProfileId: "ghost"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestClearSessionTabs_HappyPath(t *testing.T) {
	repo := newFakeRepo()
	p := newProfile("p")
	p.OpenTabs = []sessionprofilepersistence.TabState{{URL: "https://a", Order: 0}}
	repo.put(p)
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()
	resp, err := client.ClearSessionTabs(context.Background(), connect.NewRequest(&recordingsv1.ClearSessionTabsRequest{ProfileId: "p"}))
	require.NoError(t, err)
	require.Equal(t, "cleared", resp.Msg.GetStatus())
	require.Empty(t, p.OpenTabs)
}

func TestDeleteSessionTab_HappyPath(t *testing.T) {
	repo := newFakeRepo()
	p := newProfile("p")
	p.OpenTabs = []sessionprofilepersistence.TabState{
		{URL: "https://a", Order: 0},
		{URL: "https://b", Order: 1},
		{URL: "https://c", Order: 2},
	}
	repo.put(p)
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()
	resp, err := client.DeleteSessionTab(context.Background(), connect.NewRequest(&recordingsv1.DeleteSessionTabRequest{ProfileId: "p", Order: 1}))
	require.NoError(t, err)
	require.Equal(t, "deleted", resp.Msg.GetStatus())
	require.Len(t, p.OpenTabs, 2)
	require.Equal(t, "https://a", p.OpenTabs[0].URL)
	require.Equal(t, 0, p.OpenTabs[0].Order)
	require.Equal(t, "https://c", p.OpenTabs[1].URL)
	require.Equal(t, 1, p.OpenTabs[1].Order) // renumbered
}

func TestDeleteSessionTab_NotFound(t *testing.T) {
	repo := newFakeRepo()
	p := newProfile("p")
	p.OpenTabs = []sessionprofilepersistence.TabState{{URL: "https://a", Order: 0}}
	repo.put(p)
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()
	_, err := client.DeleteSessionTab(context.Background(), connect.NewRequest(&recordingsv1.DeleteSessionTabRequest{ProfileId: "p", Order: 99}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	require.EqualError(t, err, "not_found: tab not found")
}

func TestDeleteSessionTab_MissingProfileID(t *testing.T) {
	repo := newFakeRepo()
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()
	_, err := client.DeleteSessionTab(context.Background(), connect.NewRequest(&recordingsv1.DeleteSessionTabRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

// A competing committed value is visible before transaction admission. Legacy
// Get/Save callers receive their earlier snapshot and must not overwrite it.
type concurrentProfileEdit struct {
	*fakeRepo
	latest *sessionprofilepersistence.SessionProfile
}

func (r *concurrentProfileEdit) GetProfile(id sessionprofilepersistence.ProfileID) (*sessionprofilepersistence.SessionProfile, error) {
	old, err := r.fakeRepo.GetProfile(id)
	if err != nil {
		return nil, err
	}
	r.put(r.latest)
	return old, nil
}

func (r *concurrentProfileEdit) UpdateProfile(id sessionprofilepersistence.ProfileID, modify func(*sessionprofilepersistence.SessionProfile) error) (*sessionprofilepersistence.SessionProfile, error) {
	r.put(r.latest)
	return r.fakeRepo.UpdateProfile(id, modify)
}

func (r *concurrentProfileEdit) UpdateStorageState(id sessionprofilepersistence.ProfileID, modify func(json.RawMessage) (json.RawMessage, error)) (*sessionprofilepersistence.SessionProfile, error) {
	r.put(r.latest)
	return r.fakeRepo.UpdateStorageState(id, modify)
}

// [REQ:BAS-RH-J06] Targeted edits preserve a separately acknowledged addition.
func TestRecordingProfileEditsPreserveConcurrentAddition(t *testing.T) {
	for _, target := range []string{"cookie", "tab"} {
		t.Run(target, func(t *testing.T) {
			old, latest := newProfile("identity"), newProfile("identity")
			old.StorageState = []byte(`{"cookies":[{"domain":"fixture.invalid","name":"remove","value":"old"}],"origins":[]}`)
			latest.StorageState = []byte(`{"cookies":[{"domain":"fixture.invalid","name":"remove","value":"old"},{"domain":"fixture.invalid","name":"keep","value":"new"}],"origins":[]}`)
			old.OpenTabs = []sessionprofilepersistence.TabState{{URL: "https://fixture.invalid/remove", Order: 0}}
			latest.OpenTabs = append(append([]sessionprofilepersistence.TabState(nil), old.OpenTabs...), sessionprofilepersistence.TabState{URL: "https://fixture.invalid/keep", Order: 1})
			repo := &concurrentProfileEdit{fakeRepo: newFakeRepo(), latest: latest}
			repo.put(old)
			client, closeServer := newTestServer(t, repo, nil)
			defer closeServer()
			if target == "cookie" {
				_, err := client.DeleteCookie(context.Background(), connect.NewRequest(&recordingsv1.DeleteCookieRequest{ProfileId: "identity", Domain: "fixture.invalid", Name: "remove"}))
				require.NoError(t, err)
				var state playwrightStorageState
				require.NoError(t, json.Unmarshal(latest.StorageState, &state))
				require.Len(t, state.Cookies, 1)
				require.Equal(t, "keep", state.Cookies[0].Name)
				require.Equal(t, "new", state.Cookies[0].Value)
			} else {
				_, err := client.DeleteSessionTab(context.Background(), connect.NewRequest(&recordingsv1.DeleteSessionTabRequest{ProfileId: "identity", Order: 0}))
				require.NoError(t, err)
				require.Equal(t, []sessionprofilepersistence.TabState{{URL: "https://fixture.invalid/keep", Order: 0}}, latest.OpenTabs)
			}
		})
	}
}

// Adversarial oracle: deleting a cookie cannot discard another authentication store.
func TestCookieEditPreservesOtherAuthenticationState(t *testing.T) {
	repo := newFakeRepo()
	profile := newProfile("identity")
	profile.StorageState = []byte(`{"cookies":[{"name":"remove","value":"old","domain":"fixture.invalid","path":"/","expires":-1,"httpOnly":false,"secure":true,"sameSite":"Lax"},{"name":"keep","value":"secret","domain":"fixture.invalid","path":"/","expires":-1,"httpOnly":false,"secure":true,"sameSite":"Lax","partitionKey":"https://fixture.invalid"}],"origins":[{"origin":"https://fixture.invalid","localStorage":[{"name":"identity","value":"local"}],"indexedDB":[{"name":"auth","version":1,"stores":[{"name":"tokens","records":[{"key":"session","value":"opaque-token"}]}]}]}]}`)
	repo.put(profile)
	client, cleanup := newTestServer(t, repo, nil)
	defer cleanup()
	_, err := client.DeleteCookie(context.Background(), connect.NewRequest(&recordingsv1.DeleteCookieRequest{ProfileId: "identity", Domain: "fixture.invalid", Name: "remove"}))
	require.NoError(t, err)
	expected := `{"cookies":[{"name":"keep","value":"secret","domain":"fixture.invalid","path":"/","expires":-1,"httpOnly":false,"secure":true,"sameSite":"Lax","partitionKey":"https://fixture.invalid"}],"origins":[{"origin":"https://fixture.invalid","localStorage":[{"name":"identity","value":"local"}],"indexedDB":[{"name":"auth","version":1,"stores":[{"name":"tokens","records":[{"key":"session","value":"opaque-token"}]}]}]}]}`
	require.JSONEq(t, expected, string(profile.StorageState), "a targeted cookie deletion must preserve other authentication state")
}

// Minimal test projection for assertions about known cookie/localStorage fields.
type playwrightStorageState struct {
	Cookies []playwrightCookie `json:"cookies"`
	Origins []playwrightOrigin `json:"origins"`
}

type playwrightCookie struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain"`
	Path     string  `json:"path"`
	Expires  float64 `json:"expires"`
	HttpOnly bool    `json:"httpOnly"`
	Secure   bool    `json:"secure"`
	SameSite string  `json:"sameSite"`
}

type playwrightOrigin struct {
	Origin       string                  `json:"origin"`
	LocalStorage []playwrightLocalStItem `json:"localStorage"`
}

type playwrightLocalStItem struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// [REQ:BAS-RH-J01] LocalStorage edits preserve other authentication and opaque state.
func TestLocalStorageEditsPreserveOpaqueState(t *testing.T) {
	fixture := `{"cookies":[{"name":"cookie","partitionKey":"partition"}],"futureRoot":{"counter":9007199254740993},"origins":[{"origin":"https://fixture.invalid","localStorage":[{"name":"remove","value":"old"},{"name":"keep","value":"later","futureField":42}],"indexedDB":[{"name":"auth","version":1,"token":"opaque"}],"futureOrigin":{"number":9007199254740993}},{"origin":"https://opaque.invalid","futureOnly":{"token":"retain"}},{"origin":"https://other.invalid","localStorage":[{"name":"other","value":"untouched"}]}]}`
	cases := []struct {
		name                 string
		edit                 func(recordingsconnect.RecordingsServiceClient) error
		expectedLocalStorage string
		originCount          int
	}{
		{"all", func(c recordingsconnect.RecordingsServiceClient) error {
			_, err := c.ClearAllLocalStorage(context.Background(), connect.NewRequest(&recordingsv1.ClearAllLocalStorageRequest{ProfileId: "identity"}))
			return err
		}, `[]`, 2},
		{"origin", func(c recordingsconnect.RecordingsServiceClient) error {
			_, err := c.DeleteLocalStorageByOrigin(context.Background(), connect.NewRequest(&recordingsv1.DeleteLocalStorageByOriginRequest{ProfileId: "identity", Origin: "https://fixture.invalid"}))
			return err
		}, `[]`, 3},
		{"item", func(c recordingsconnect.RecordingsServiceClient) error {
			_, err := c.DeleteLocalStorageItem(context.Background(), connect.NewRequest(&recordingsv1.DeleteLocalStorageItemRequest{ProfileId: "identity", Origin: "https://fixture.invalid", Name: "remove"}))
			return err
		}, `[{"name":"keep","value":"later","futureField":42}]`, 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeRepo()
			profile := newProfile("identity")
			profile.StorageState = []byte(fixture)
			repo.put(profile)
			client, cleanup := newTestServer(t, repo, nil)
			defer cleanup()
			require.NoError(t, tc.edit(client))
			var state map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(profile.StorageState, &state))
			require.JSONEq(t, `[{"name":"cookie","partitionKey":"partition"}]`, string(state["cookies"]))
			require.Contains(t, string(state["futureRoot"]), "9007199254740993", "opaque integers must not pass through float64")
			var origins []map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(state["origins"], &origins))
			require.Len(t, origins, tc.originCount)
			require.JSONEq(t, tc.expectedLocalStorage, string(origins[0]["localStorage"]))
			require.JSONEq(t, `[{"name":"auth","version":1,"token":"opaque"}]`, string(origins[0]["indexedDB"]))
			require.Contains(t, string(origins[0]["futureOrigin"]), "9007199254740993")
			require.JSONEq(t, `{"token":"retain"}`, string(origins[1]["futureOnly"]))
			require.NotContains(t, origins[1], "localStorage", "an origin without localStorage must be unchanged")
			if tc.originCount == 3 {
				require.JSONEq(t, `[{"name":"other","value":"untouched"}]`, string(origins[2]["localStorage"]))
			}
		})
	}
}

func TestMalformedTargetedStorageEditPreservesSnapshot(t *testing.T) {
	for _, fixture := range []string{`{"cookies":"malformed","origins":[]}`, `{"cookies":[42],"origins":[]}`, `{"cookies":[null],"origins":[]}`} {
		t.Run(fixture, func(t *testing.T) {
			repo := newFakeRepo()
			profile := newProfile("identity")
			profile.StorageState = []byte(fixture)
			repo.put(profile)
			client, cleanup := newTestServer(t, repo, nil)
			defer cleanup()
			_, err := client.DeleteCookie(context.Background(), connect.NewRequest(&recordingsv1.DeleteCookieRequest{ProfileId: "identity", Domain: "fixture.invalid", Name: "remove"}))
			require.Error(t, err)
			require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
			require.Equal(t, fixture, string(profile.StorageState))
		})
	}
	for _, fixture := range []string{`{"origins":"malformed"}`, `{"origins":[{"origin":"https://fixture.invalid","localStorage":"malformed"}]}`, `{"origins":[{"origin":"https://fixture.invalid","localStorage":[null]}]}`} {
		t.Run(fixture, func(t *testing.T) {
			repo := newFakeRepo()
			profile := newProfile("identity")
			profile.StorageState = []byte(fixture)
			repo.put(profile)
			client, cleanup := newTestServer(t, repo, nil)
			defer cleanup()
			_, err := client.DeleteLocalStorageItem(context.Background(), connect.NewRequest(&recordingsv1.DeleteLocalStorageItemRequest{ProfileId: "identity", Origin: "https://fixture.invalid", Name: "remove"}))
			require.Error(t, err)
			require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
			require.Equal(t, fixture, string(profile.StorageState))
		})
	}
}
