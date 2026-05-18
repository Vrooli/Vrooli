package skill_catalog_test

import (
	"context"
	"testing"
	"time"

	skillCatalogH "development-toolchain-validator/handlers/skill_catalog"
	skillcatalog "development-toolchain-validator/internal/skill_catalog"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	skillcatalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/skill_catalog"
	skillcatalogconnect "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/skill_catalog/skill_catalog_v1connect"
)

// fakeService inlined per the same convention used in golden's handler
// tests: hand-rolled stub at the handler boundary keeps assertions local.
type fakeService struct {
	SyncOut skillcatalog.SyncResult
	SyncErr error

	ListOut []skillcatalog.Skill
	ListErr error

	GetID  string
	GetOut skillcatalog.Skill
	GetErr error
}

func (f *fakeService) Sync(context.Context) (skillcatalog.SyncResult, error) {
	return f.SyncOut, f.SyncErr
}

func (f *fakeService) List(context.Context) ([]skillcatalog.Skill, error) {
	return f.ListOut, f.ListErr
}

func (f *fakeService) Get(_ context.Context, id string) (skillcatalog.Skill, error) {
	f.GetID = id
	if f.GetErr != nil {
		return skillcatalog.Skill{}, f.GetErr
	}
	return f.GetOut, nil
}

var _ skillcatalog.Service = (*fakeService)(nil)

func newClient(t *testing.T, svc skillcatalog.Service) skillcatalogconnect.SkillCatalogServiceClient {
	t.Helper()
	logger, _ := connectxtest.NewLogger(t)
	path, handler := skillcatalogconnect.NewSkillCatalogServiceHandler(skillCatalogH.NewConnectHandler(skillCatalogH.Deps{
		Service: svc,
		Logger:  logger,
	}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: handler})
	return skillcatalogconnect.NewSkillCatalogServiceClient(server.Client(), server.URL)
}

func TestSync_ReturnsCatalogAndCounts(t *testing.T) {
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	client := newClient(t, &fakeService{SyncOut: skillcatalog.SyncResult{
		Skills: []skillcatalog.Skill{{ID: "plan-skill-discovery", Version: "v1", ContentHash: "h1", SyncedAt: now}},
		Added:  1, Updated: 0, Removed: 0,
	}})
	resp, err := client.Sync(context.Background(), connect.NewRequest(&skillcatalogv1.SyncRequest{}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Skills, 1)
	require.Equal(t, int32(1), resp.Msg.Added)
	require.Equal(t, "plan-skill-discovery", resp.Msg.Skills[0].Id)
}

func TestSync_NotReadyMapsToUnavailable(t *testing.T) {
	client := newClient(t, &fakeService{SyncErr: skillcatalog.ErrSyncFailed{
		Reason:   "prompt-manager is not running",
		NotReady: true,
	}})
	_, err := client.Sync(context.Background(), connect.NewRequest(&skillcatalogv1.SyncRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}

func TestList_PassesThrough(t *testing.T) {
	client := newClient(t, &fakeService{ListOut: []skillcatalog.Skill{
		{ID: "a", Version: "v1", ContentHash: "h1"},
		{ID: "b", Version: "v1", ContentHash: "h2"},
	}})
	resp, err := client.ListSkills(context.Background(), connect.NewRequest(&skillcatalogv1.ListSkillsRequest{}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Skills, 2)
}

func TestGet_NotFoundMapsToCodeNotFound(t *testing.T) {
	client := newClient(t, &fakeService{GetErr: skillcatalog.ErrSkillNotFound{ID: "ghost"}})
	_, err := client.GetSkill(context.Background(), connect.NewRequest(&skillcatalogv1.GetSkillRequest{Id: "ghost"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestGet_InvalidArgument(t *testing.T) {
	client := newClient(t, &fakeService{GetErr: skillcatalog.ErrInvalidSkill{Field: "id", Reason: "required"}})
	_, err := client.GetSkill(context.Background(), connect.NewRequest(&skillcatalogv1.GetSkillRequest{Id: ""}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
