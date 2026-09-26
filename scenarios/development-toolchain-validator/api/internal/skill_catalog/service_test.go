package skill_catalog_test

import (
	"context"
	"errors"
	"testing"
	"time"

	skillcatalog "development-toolchain-validator/internal/skill_catalog"
	scmocks "development-toolchain-validator/internal/skill_catalog/mocks"

	"github.com/vrooli/api-core/scheduletest"

	"github.com/stretchr/testify/require"
)

func newSvc(t *testing.T, source skillcatalog.SkillCatalogSource) (skillcatalog.Service, *scmocks.FakeRepository, *scheduletest.FakeClock) {
	t.Helper()
	repo := scmocks.NewFakeRepository()
	clk := scheduletest.New(time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC))
	return skillcatalog.NewService(repo, source, clk), repo, clk
}

func TestSync_AddsNewSkills(t *testing.T) {
	source := &scmocks.FakeSource{Skills: []skillcatalog.Skill{
		{ID: "implementation-plan-authoring", Version: "v1", ContentHash: "h1"},
		{ID: "test", Version: "v1", ContentHash: "h2"},
	}}
	svc, _, _ := newSvc(t, source)

	res, err := svc.Sync(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, res.Added)
	require.Equal(t, 0, res.Updated)
	require.Equal(t, 0, res.Removed)
	require.Len(t, res.Skills, 2)
}

func TestSync_UpdatesChangedSkills(t *testing.T) {
	source := &scmocks.FakeSource{Skills: []skillcatalog.Skill{
		{ID: "implementation-plan-authoring", Version: "v1", ContentHash: "h1"},
	}}
	svc, _, _ := newSvc(t, source)
	_, err := svc.Sync(context.Background())
	require.NoError(t, err)

	source.Skills[0].ContentHash = "h2-changed"
	res, err := svc.Sync(context.Background())
	require.NoError(t, err)
	require.Equal(t, 0, res.Added)
	require.Equal(t, 1, res.Updated)
	require.Equal(t, 0, res.Removed)
}

func TestSync_RemovesMissingSkills(t *testing.T) {
	source := &scmocks.FakeSource{Skills: []skillcatalog.Skill{
		{ID: "a", Version: "v1", ContentHash: "h1"},
		{ID: "b", Version: "v1", ContentHash: "h2"},
	}}
	svc, _, _ := newSvc(t, source)
	_, err := svc.Sync(context.Background())
	require.NoError(t, err)

	source.Skills = source.Skills[:1] // drop "b"
	res, err := svc.Sync(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, res.Removed)
	require.Len(t, res.Skills, 1)
	require.Equal(t, "a", res.Skills[0].ID)
}

func TestSync_StampsSyncedAtFromClock(t *testing.T) {
	source := &scmocks.FakeSource{Skills: []skillcatalog.Skill{
		{ID: "a", Version: "v1", ContentHash: "h1"},
	}}
	svc, _, clk := newSvc(t, source)
	clk.SetNow(time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC))

	res, err := svc.Sync(context.Background())
	require.NoError(t, err)
	require.Equal(t, clk.Now(), res.Skills[0].SyncedAt)
}

func TestSync_SkipsMalformedIDs(t *testing.T) {
	source := &scmocks.FakeSource{Skills: []skillcatalog.Skill{
		{ID: "good-skill", Version: "v1", ContentHash: "h1"},
		{ID: "", Version: "v1", ContentHash: "h2"},
		{ID: "  spaces only  ", Version: "v1", ContentHash: "h3"},
	}}
	svc, _, _ := newSvc(t, source)
	res, err := svc.Sync(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, res.Added)
	require.Len(t, res.Skills, 1)
}

func TestSync_FetchErrorPropagates(t *testing.T) {
	wantErr := skillcatalog.ErrSyncFailed{Reason: "discovery", NotReady: true}
	source := &scmocks.FakeSource{Err: wantErr}
	svc, _, _ := newSvc(t, source)
	_, err := svc.Sync(context.Background())
	require.Error(t, err)
	var sync skillcatalog.ErrSyncFailed
	require.True(t, errors.As(err, &sync))
	require.True(t, sync.NotReady)
}

func TestSync_NoSourceConfigured(t *testing.T) {
	svc, _, _ := newSvc(t, nil)
	_, err := svc.Sync(context.Background())
	require.Error(t, err)
	var sync skillcatalog.ErrSyncFailed
	require.True(t, errors.As(err, &sync))
}

func TestGet_EmptyIDRejected(t *testing.T) {
	svc, _, _ := newSvc(t, &scmocks.FakeSource{})
	_, err := svc.Get(context.Background(), "  ")
	require.Error(t, err)
	var invalid skillcatalog.ErrInvalidSkill
	require.True(t, errors.As(err, &invalid))
}

func TestList_PassesThrough(t *testing.T) {
	source := &scmocks.FakeSource{Skills: []skillcatalog.Skill{
		{ID: "a", Version: "v1", ContentHash: "h1"},
	}}
	svc, _, _ := newSvc(t, source)
	_, err := svc.Sync(context.Background())
	require.NoError(t, err)

	got, err := svc.List(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 1)
}
