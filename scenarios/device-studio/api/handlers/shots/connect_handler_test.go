package shots

import (
	"context"
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"
	projects "device-studio/internal/projects"
	domain "device-studio/internal/shots"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	physicalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/physical"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
)

type videoSampleUnavailableSaveService struct{ domain.Service }

func (videoSampleUnavailableSaveService) Save(context.Context, string, int64, string, *shotsv1.ShotSnapshot) (domain.Shot, domain.Revision, error) {
	return domain.Shot{}, domain.Revision{}, domain.ErrVideoSampleUnavailable
}

func TestRevisionConflictIsTypedAndRedacted(t *testing.T) { // [REQ:DS-INTEGRATION-OPERATIONS]
	err := toConnectError(domain.ErrRevisionConflict{ShotID: "shot-a", Expected: 4, Current: 5}, nil)
	require.Equal(t, connect.CodeAborted, connect.CodeOf(err))
	var ce *connect.Error
	require.ErrorAs(t, err, &ce)
	require.NotContains(t, ce.Message(), "shot-a")
	var found bool
	for _, detail := range ce.Details() {
		message, detailErr := detail.Value()
		require.NoError(t, detailErr)
		conflict, ok := message.(*shotsv1.ShotConflict)
		if ok {
			found = true
			require.Equal(t, "shot-a", conflict.ShotId)
			require.EqualValues(t, 4, conflict.ExpectedRevision)
			require.EqualValues(t, 5, conflict.CurrentRevision)
		}
	}
	require.True(t, found)
}

func TestUnexpectedErrorsAreRedacted(t *testing.T) {
	err := toConnectError(errors.New("sqlite file /secret/path failed"), nil)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	require.NotContains(t, err.Error(), "/secret/path")
}

func TestMissingProjectFromDraftValidationMapsNotFound(t *testing.T) {
	err := toConnectError(projects.ErrProjectNotFound{ID: "project-secret"}, nil)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	require.NotContains(t, err.Error(), "project-secret")
}

func TestUnsupportedSavedPresentationVersionMapsFailedPrecondition(t *testing.T) {
	err := toConnectError(domain.ErrUnsupportedPresentationVersion{Version: "private-version"}, nil)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	require.Contains(t, err.Error(), "create a new revision")
	require.NotContains(t, err.Error(), "private-version")
}

func TestTamperedVideoSampleSaveMapsFailedPrecondition(t *testing.T) {
	h := NewConnectHandler(Deps{Service: videoSampleUnavailableSaveService{}})
	_, err := h.SaveShot(context.Background(), connect.NewRequest(&shotsv1.SaveShotRequest{ShotId: "shot", ExpectedRevision: 3}))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	require.Contains(t, err.Error(), "regenerate the sample")
	require.Contains(t, err.Error(), "accept it before saving")
	require.NotContains(t, err.Error(), "internal")
}

func TestPhysicalBindingReasonIsActionableAndPathSafe(t *testing.T) {
	staticReason := toConnectError(domain.ErrInvalidPhysicalBinding{Reason: "physical recording requires an accepted inspected video raster"}, nil)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(staticReason))
	require.Contains(t, staticReason.Error(), "accepted inspected video raster")

	privateReason := toConnectError(domain.ErrInvalidPhysicalBinding{Reason: "failed reading /private/runtime/source.mp4"}, nil)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(privateReason))
	require.NotContains(t, privateReason.Error(), "/private/runtime/source.mp4")
	require.Contains(t, privateReason.Error(), "check the physical device")

	longReason := toConnectError(domain.ErrInvalidPhysicalBinding{Reason: strings.Repeat("x", 241)}, nil)
	require.NotContains(t, longReason.Error(), strings.Repeat("x", 80))
	require.Contains(t, longReason.Error(), "check the physical device")

	unrelated := toConnectError(errors.New("private backend failure"), nil)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(unrelated))
	require.NotContains(t, unrelated.Error(), "private backend failure")
}

type physicalSetupValidatorStub struct{}

func (physicalSetupValidatorStub) ValidatePhysicalSetup(context.Context, *shotsv1.PhysicalShotSetup) error {
	return nil
}

func TestPhysicalPresetConnectSurfaceSavesAndListsImmutableCandidateSetup(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(domain.Schema)))
	repo, err := domain.NewSQLitePhysicalPresetRepository(db)
	require.NoError(t, err)
	service, err := domain.NewPhysicalPresetService(repo, physicalSetupValidatorStub{}, nil, nil)
	require.NoError(t, err)
	h := NewConnectHandler(Deps{Presets: service})
	setup := &shotsv1.PhysicalShotSetup{
		Device:    &shotsv1.ComponentRevisionRef{Id: "device", Revision: 1, Sha256: strings.Repeat("a", 64)},
		Scene:     &shotsv1.ComponentRevisionRef{Id: "scene", Revision: 1, Sha256: strings.Repeat("b", 64)},
		Motion:    &shotsv1.ComponentRevisionRef{Id: "motion", Revision: 1, Sha256: strings.Repeat("c", 64)},
		DisplayId: "display", DeviceSupportId: "device-support", SceneSupportId: "scene-support", CameraId: "camera",
		SceneSupportWorld: &physicalv1.TRS{TranslationM: &physicalv1.Vec3{Y: .75}, RotationXyzw: &physicalv1.QuatXYZW{W: 1}, Scale: &physicalv1.Vec3{X: 1, Y: 1, Z: 1}},
		SupportPivotM:     &physicalv1.Vec3{Y: .007},
		DeviceRootRest:    &physicalv1.TRS{TranslationM: &physicalv1.Vec3{}, RotationXyzw: &physicalv1.QuatXYZW{W: 1}, Scale: &physicalv1.Vec3{X: 1, Y: 1, Z: 1}},
		StillProfiles:     []shotsv1.PhysicalStillOutputProfile{shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_2560X1440_V1},
	}
	saved, err := h.SavePhysicalShotPreset(ctx, connect.NewRequest(&shotsv1.SavePhysicalShotPresetRequest{PresetId: "study-phone", ExpectedRevision: 0, Name: "Study phone", Setup: setup}))
	require.NoError(t, err)
	require.EqualValues(t, 1, saved.Msg.Revision.Revision)
	require.True(t, saved.Msg.Revision.CandidateOnly)
	got, err := h.GetPhysicalShotPreset(ctx, connect.NewRequest(&shotsv1.GetPhysicalShotPresetRequest{PresetId: "study-phone", Revision: 1}))
	require.NoError(t, err)
	require.Equal(t, saved.Msg.Revision.Sha256, got.Msg.Revision.Sha256)
	head, err := h.GetPhysicalShotPresetHead(ctx, connect.NewRequest(&shotsv1.GetPhysicalShotPresetHeadRequest{PresetId: "study-phone"}))
	require.NoError(t, err)
	require.EqualValues(t, 1, head.Msg.Revision.Revision)
	list, err := h.ListPhysicalShotPresets(ctx, connect.NewRequest(&shotsv1.ListPhysicalShotPresetsRequest{PageSize: 10}))
	require.NoError(t, err)
	require.Len(t, list.Msg.Presets, 1)
	require.True(t, list.Msg.Presets[0].CandidateOnly)
	revisions, err := h.ListPhysicalShotPresetRevisions(ctx, connect.NewRequest(&shotsv1.ListPhysicalShotPresetRevisionsRequest{PresetId: "study-phone", PageSize: 10}))
	require.NoError(t, err)
	require.Len(t, revisions.Msg.Revisions, 1)
}
