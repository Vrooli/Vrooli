package session_profiles

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	sessionprofilepersistence "github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
	sessionprofilesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/session_profiles"
	sessionprofilesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/session_profiles/session_profilesconnect"
)

// fakeRepo implements Repo for handler tests.
type fakeRepo struct {
	mu       sync.Mutex
	items    map[sessionprofilepersistence.ProfileID]*sessionprofilepersistence.SessionProfile
	listErr  error
	createOv func(name string) (*sessionprofilepersistence.SessionProfile, error)
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{items: map[sessionprofilepersistence.ProfileID]*sessionprofilepersistence.SessionProfile{}}
}

func (f *fakeRepo) ListProfiles() ([]sessionprofilepersistence.SessionProfile, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]sessionprofilepersistence.SessionProfile, 0, len(f.items))
	for _, p := range f.items {
		out = append(out, *p)
	}
	return out, nil
}

func (f *fakeRepo) CreateProfile(name string) (*sessionprofilepersistence.SessionProfile, error) {
	if f.createOv != nil {
		return f.createOv(name)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	now := time.Now().UTC()
	id := sessionprofilepersistence.ProfileID(uuid.NewString())
	if name == "" {
		name = "Session 1"
	}
	p := &sessionprofilepersistence.SessionProfile{
		ID:         id,
		Name:       name,
		CreatedAt:  now,
		UpdatedAt:  now,
		LastUsedAt: now,
	}
	f.items[id] = p
	return p, nil
}

func (f *fakeRepo) RenameProfile(id sessionprofilepersistence.ProfileID, name string) (*sessionprofilepersistence.SessionProfile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.items[id]
	if !ok {
		return nil, errors.New("profile not found")
	}
	p.Name = name
	p.UpdatedAt = time.Now().UTC()
	return p, nil
}

func (f *fakeRepo) UpdateBrowserProfile(id sessionprofilepersistence.ProfileID, bp *sessionprofilepersistence.BrowserProfile) (*sessionprofilepersistence.SessionProfile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.items[id]
	if !ok {
		return nil, errors.New("profile not found")
	}
	p.BrowserProfile = bp
	p.UpdatedAt = time.Now().UTC()
	return p, nil
}

func (f *fakeRepo) DeleteProfile(id sessionprofilepersistence.ProfileID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.items[id]; !ok {
		return errors.New("profile not found")
	}
	delete(f.items, id)
	return nil
}

// newTestServer wires the handler into an httptest server and returns a Connect client.
func newTestServer(t *testing.T, repo Repo) (sessionprofilesconnect.SessionProfilesServiceClient, func()) {
	t.Helper()
	log := logrus.New()
	log.SetOutput(discardWriter{})
	mount := Module(Deps{Repo: repo, Logger: log})
	mux := http.NewServeMux()
	mux.Handle(mount.Path, mount.Handler)
	srv := httptest.NewServer(mux)
	client := sessionprofilesconnect.NewSessionProfilesServiceClient(srv.Client(), srv.URL)
	return client, srv.Close
}

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

// =============================================================================
// Module wiring
// =============================================================================

func TestModule_PanicsOnMissingDeps(t *testing.T) {
	require.PanicsWithValue(t, "session_profiles.Module requires Deps.Logger", func() {
		Module(Deps{})
	})
	require.PanicsWithValue(t, "session_profiles.Module requires Deps.Repo", func() {
		Module(Deps{Logger: logrus.New()})
	})
}

// =============================================================================
// List
// =============================================================================

func TestList_Empty(t *testing.T) {
	repo := newFakeRepo()
	client, cleanup := newTestServer(t, repo)
	defer cleanup()

	resp, err := client.List(context.Background(), connect.NewRequest(&sessionprofilesv1.ListSessionProfilesRequest{}))
	require.NoError(t, err)
	require.Empty(t, resp.Msg.GetProfiles())
}

func TestList_PopulatedAndStorageFlag(t *testing.T) {
	repo := newFakeRepo()
	p, _ := repo.CreateProfile("Alpha")
	p.StorageState = []byte(`{"cookies":[{"name":"x"}],"origins":[]}`)
	q, _ := repo.CreateProfile("Beta")
	q.StorageState = []byte(`{"cookies":[],"origins":[]}`)

	client, cleanup := newTestServer(t, repo)
	defer cleanup()

	resp, err := client.List(context.Background(), connect.NewRequest(&sessionprofilesv1.ListSessionProfilesRequest{}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetProfiles(), 2)

	byName := map[string]*sessionprofilesv1.SessionProfile{}
	for _, item := range resp.Msg.GetProfiles() {
		byName[item.GetName()] = item
	}
	require.True(t, byName["Alpha"].GetHasStorageState())
	require.False(t, byName["Beta"].GetHasStorageState())
}

func TestList_RepoError(t *testing.T) {
	repo := newFakeRepo()
	repo.listErr = errors.New("boom")
	client, cleanup := newTestServer(t, repo)
	defer cleanup()

	_, err := client.List(context.Background(), connect.NewRequest(&sessionprofilesv1.ListSessionProfilesRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

// =============================================================================
// Create
// =============================================================================

func TestCreate_DefaultName(t *testing.T) {
	repo := newFakeRepo()
	client, cleanup := newTestServer(t, repo)
	defer cleanup()

	resp, err := client.Create(context.Background(), connect.NewRequest(&sessionprofilesv1.CreateSessionProfileRequest{}))
	require.NoError(t, err)
	require.NotEmpty(t, resp.Msg.GetProfile().GetId())
	require.Equal(t, "Session 1", resp.Msg.GetProfile().GetName())
	require.NotNil(t, resp.Msg.GetProfile().GetCreatedAt())
}

func TestCreate_NamedAndPropagatesError(t *testing.T) {
	repo := newFakeRepo()
	client, cleanup := newTestServer(t, repo)
	defer cleanup()

	resp, err := client.Create(context.Background(), connect.NewRequest(&sessionprofilesv1.CreateSessionProfileRequest{Name: "Mine"}))
	require.NoError(t, err)
	require.Equal(t, "Mine", resp.Msg.GetProfile().GetName())

	repo.createOv = func(string) (*sessionprofilepersistence.SessionProfile, error) {
		return nil, errors.New("disk full")
	}
	_, err = client.Create(context.Background(), connect.NewRequest(&sessionprofilesv1.CreateSessionProfileRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

// =============================================================================
// Update
// =============================================================================

func TestUpdate_InvalidID(t *testing.T) {
	repo := newFakeRepo()
	client, cleanup := newTestServer(t, repo)
	defer cleanup()

	_, err := client.Update(context.Background(), connect.NewRequest(&sessionprofilesv1.UpdateSessionProfileRequest{
		Id: "not-a-uuid",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestUpdate_NothingProvided(t *testing.T) {
	repo := newFakeRepo()
	client, cleanup := newTestServer(t, repo)
	defer cleanup()

	id := uuid.NewString()
	_, err := client.Update(context.Background(), connect.NewRequest(&sessionprofilesv1.UpdateSessionProfileRequest{
		Id: id,
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestUpdate_RenameHappyPath(t *testing.T) {
	repo := newFakeRepo()
	p, _ := repo.CreateProfile("Old")
	client, cleanup := newTestServer(t, repo)
	defer cleanup()

	name := "New Name"
	resp, err := client.Update(context.Background(), connect.NewRequest(&sessionprofilesv1.UpdateSessionProfileRequest{
		Id:   string(p.ID),
		Name: &name,
	}))
	require.NoError(t, err)
	require.Equal(t, "New Name", resp.Msg.GetProfile().GetName())
}

func TestUpdate_BrowserProfileHappyPath(t *testing.T) {
	repo := newFakeRepo()
	p, _ := repo.CreateProfile("Alpha")
	client, cleanup := newTestServer(t, repo)
	defer cleanup()

	bp, err := structpb.NewStruct(map[string]any{"preset": "stealth"})
	require.NoError(t, err)
	resp, err := client.Update(context.Background(), connect.NewRequest(&sessionprofilesv1.UpdateSessionProfileRequest{
		Id:             string(p.ID),
		BrowserProfile: bp,
	}))
	require.NoError(t, err)
	require.NotNil(t, resp.Msg.GetProfile().GetBrowserProfile())
	require.Equal(t, "stealth", resp.Msg.GetProfile().GetBrowserProfile().Fields["preset"].GetStringValue())
}

func TestUpdate_NotFound(t *testing.T) {
	repo := newFakeRepo()
	client, cleanup := newTestServer(t, repo)
	defer cleanup()

	name := "x"
	_, err := client.Update(context.Background(), connect.NewRequest(&sessionprofilesv1.UpdateSessionProfileRequest{
		Id:   uuid.NewString(),
		Name: &name,
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

// =============================================================================
// Delete
// =============================================================================

func TestDelete_InvalidID(t *testing.T) {
	repo := newFakeRepo()
	client, cleanup := newTestServer(t, repo)
	defer cleanup()

	_, err := client.Delete(context.Background(), connect.NewRequest(&sessionprofilesv1.DeleteSessionProfileRequest{Id: "x"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestDelete_NotFound(t *testing.T) {
	repo := newFakeRepo()
	client, cleanup := newTestServer(t, repo)
	defer cleanup()

	_, err := client.Delete(context.Background(), connect.NewRequest(&sessionprofilesv1.DeleteSessionProfileRequest{Id: uuid.NewString()}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestDelete_HappyPath(t *testing.T) {
	repo := newFakeRepo()
	p, _ := repo.CreateProfile("Bye")
	client, cleanup := newTestServer(t, repo)
	defer cleanup()

	resp, err := client.Delete(context.Background(), connect.NewRequest(&sessionprofilesv1.DeleteSessionProfileRequest{Id: string(p.ID)}))
	require.NoError(t, err)
	require.Equal(t, string(p.ID), resp.Msg.GetId())
	require.Empty(t, repo.items)
}
