package session_profiles

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/vrooli/browser-automation-studio/internal/testutil"
	sessionprofilepersistence "github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
	sessionprofilesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/session_profiles"
	sessionprofilesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/session_profiles/session_profilesconnect"
)

// fakeRepo implements Repo for handler tests.
type fakeRepo struct {
	*sessionprofilepersistence.MockRepository
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{MockRepository: sessionprofilepersistence.NewMockRepository()}
}

func (f *fakeRepo) ListProfiles() ([]sessionprofilepersistence.SessionProfile, error) {
	return f.List()
}

func (f *fakeRepo) CreateProfile(name string) (*sessionprofilepersistence.SessionProfile, error) {
	if name == "" {
		name = "Session 1"
	}
	now := time.Now().UTC()
	p := &sessionprofilepersistence.SessionProfile{
		ID:         sessionprofilepersistence.ProfileID(uuid.NewString()),
		Name:       name,
		CreatedAt:  now,
		UpdatedAt:  now,
		LastUsedAt: now,
	}
	if err := f.Create(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (f *fakeRepo) RenameProfile(id sessionprofilepersistence.ProfileID, name string) (*sessionprofilepersistence.SessionProfile, error) {
	return f.Update(id, func(p *sessionprofilepersistence.SessionProfile) error {
		p.Name = name
		p.UpdatedAt = time.Now().UTC()
		return nil
	})
}

func (f *fakeRepo) UpdateBrowserProfile(id sessionprofilepersistence.ProfileID, bp *sessionprofilepersistence.BrowserProfile) (*sessionprofilepersistence.SessionProfile, error) {
	return f.Update(id, func(p *sessionprofilepersistence.SessionProfile) error {
		p.BrowserProfile = bp
		p.UpdatedAt = time.Now().UTC()
		return nil
	})
}

func (f *fakeRepo) DeleteProfile(id sessionprofilepersistence.ProfileID) error {
	return f.Delete(id)
}

// newTestServer wires the handler into an httptest server and returns a Connect client.
func newTestServer(t *testing.T, repo Repo) (sessionprofilesconnect.SessionProfilesServiceClient, func()) {
	t.Helper()
	log := logrus.New()
	log.SetOutput(discardWriter{})
	mount := Module(Deps{Repo: repo, Logger: log})
	mux := http.NewServeMux()
	mux.Handle(mount.Path, mount.Handler)
	srv := testutil.StartHTTPServer(t, mux)
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
	_, err := repo.Update(p.ID, func(profile *sessionprofilepersistence.SessionProfile) error {
		profile.StorageState = []byte(`{"cookies":[{"name":"x"}],"origins":[]}`)
		return nil
	})
	require.NoError(t, err)
	q, _ := repo.CreateProfile("Beta")
	_, err = repo.Update(q.ID, func(profile *sessionprofilepersistence.SessionProfile) error {
		profile.StorageState = []byte(`{"cookies":[],"origins":[]}`)
		return nil
	})
	require.NoError(t, err)

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
	repo.ListErr = errors.New("boom")
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

	repo.CreateErr = errors.New("disk full")
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
	require.Zero(t, repo.Count())
}
