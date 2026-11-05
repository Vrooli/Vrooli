package shots

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	physicalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/physical"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
)

type presetValidatorStub struct {
	calls int
	err   error
}

func (s *presetValidatorStub) ValidatePhysicalSetup(context.Context, *shotsv1.PhysicalShotSetup) error {
	s.calls++
	return s.err
}

func presetSetupFixture() *shotsv1.PhysicalShotSetup {
	ref := func(id, letter string) *shotsv1.ComponentRevisionRef {
		return &shotsv1.ComponentRevisionRef{Id: id, Revision: 1, Sha256: repeatedDigest(letter)}
	}
	return &shotsv1.PhysicalShotSetup{
		Device: ref("phone-device", "a"), Scene: ref("study-scene", "b"), Motion: ref("still-motion", "c"),
		DisplayId: "main-display", DeviceSupportId: "device-support", SceneSupportId: "scene-support", CameraId: "study-camera",
		SceneSupportWorld: testPresetTRS(0, .75, 0), DeviceRootRest: testPresetTRS(0, 0, 0),
		SupportPivotM: &physicalv1.Vec3{Y: .007},
		StillProfiles: []shotsv1.PhysicalStillOutputProfile{shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_2560X1440_V1},
	}
}

func repeatedDigest(letter string) string { return strings.Repeat(letter, 64) }

func testPresetTRS(x, y, z float64) *physicalv1.TRS {
	return &physicalv1.TRS{TranslationM: &physicalv1.Vec3{X: x, Y: y, Z: z}, RotationXyzw: &physicalv1.QuatXYZW{W: 1}, Scale: &physicalv1.Vec3{X: 1, Y: 1, Z: 1}}
}

func TestPhysicalPresetRepositoryCASReplayHistoryAndCorruption(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(Schema)))
	repo, err := NewSQLitePhysicalPresetRepository(db)
	require.NoError(t, err)
	validator := &presetValidatorStub{}
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	service, err := NewPhysicalPresetService(repo, validator, func() time.Time { return now }, func() string { return "phone-study-setup" })
	require.NoError(t, err)
	a := presetSetupFixture()
	a.DeviceRootRest.RotationXyzw = &physicalv1.QuatXYZW{X: .1, Y: .2, Z: .3, W: .4}
	first, err := service.Save(ctx, "phone-study-setup", 0, "Phone study", a)
	require.NoError(t, err)
	require.EqualValues(t, 1, first.Revision)
	require.NotEmpty(t, first.SHA256)
	replay, err := service.Save(ctx, "phone-study-setup", 0, "Phone study", a)
	require.NoError(t, err)
	require.Equal(t, first.SHA256, replay.SHA256)
	require.EqualValues(t, 1, replay.Revision)
	b := presetSetupFixture()
	b.DeviceRootRest.TranslationM.Z = .02
	second, err := service.Save(ctx, "phone-study-setup", 1, "Phone study", b)
	require.NoError(t, err)
	require.EqualValues(t, 2, second.Revision)
	_, err = service.Save(ctx, "phone-study-setup", 1, "Phone study", a)
	require.ErrorIs(t, err, ErrPhysicalPresetConflict)
	third, err := service.Save(ctx, "phone-study-setup", 2, "Phone study", a)
	require.NoError(t, err)
	require.EqualValues(t, 3, third.Revision)
	require.Equal(t, first.SHA256, third.SHA256)
	head, err := service.Head(ctx, "phone-study-setup")
	require.NoError(t, err)
	require.EqualValues(t, 3, head.Revision)
	rows, err := service.ListRevisions(ctx, "phone-study-setup", 0, 10)
	require.NoError(t, err)
	require.Len(t, rows, 3)
	heads, err := service.ListHeads(ctx, "", 10)
	require.NoError(t, err)
	require.Len(t, heads, 1)
	require.Equal(t, 5, validator.calls)
	_, err = db.Exec(`UPDATE physical_shot_preset_revisions SET candidate_json=candidate_json || ' ' WHERE preset_id=? AND revision=1`, "phone-study-setup")
	require.NoError(t, err)
	_, err = service.Get(ctx, "phone-study-setup", 1)
	require.Error(t, err)
}

func TestPhysicalPresetCanonicalProtoJSONRoundTripsCompactly(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(Schema)))
	repo, err := NewSQLitePhysicalPresetRepository(db)
	require.NoError(t, err)
	service, err := NewPhysicalPresetService(repo, &presetValidatorStub{}, nil, func() string { return "rotated-phone-setup" })
	require.NoError(t, err)
	setup := presetSetupFixture()
	setup.DeviceRootRest.RotationXyzw = &physicalv1.QuatXYZW{X: .1, Y: .2, Z: .3, W: .4}
	first, err := service.Save(ctx, "rotated-phone-setup", 0, "Rotated phone", setup)
	require.NoError(t, err)
	reopened, err := service.Get(ctx, first.PresetID, first.Revision)
	require.NoError(t, err)
	require.Equal(t, first.SHA256, reopened.SHA256)
	_, want, err := canonicalPhysicalSetup(reopened.Setup)
	require.NoError(t, err)
	var stored physicalPresetSnapshot
	require.NoError(t, json.Unmarshal(first.canonical, &stored))
	require.Equal(t, string(want), string(stored.Setup))
}

func TestPhysicalPresetRejectsUnsupportedProfilesAndValidatorFailures(t *testing.T) {
	db := databasetest.NewSQLite(t)
	ctx := context.Background()
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(Schema)))
	repo, err := NewSQLitePhysicalPresetRepository(db)
	require.NoError(t, err)
	validator := &presetValidatorStub{err: ErrInvalidPhysicalBinding{Reason: "selected scene support is incompatible"}}
	service, err := NewPhysicalPresetService(repo, validator, nil, func() string { return "new-setup" })
	require.NoError(t, err)
	bad := presetSetupFixture()
	bad.StillProfiles = []shotsv1.PhysicalStillOutputProfile{shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_UNSPECIFIED}
	_, err = service.Save(ctx, "", 0, "Setup", bad)
	require.Error(t, err)
	require.Equal(t, 0, validator.calls)
	_, err = service.Save(ctx, "", 0, "Setup", presetSetupFixture())
	require.Error(t, err)
	require.Contains(t, err.Error(), "incompatible")
	_, err = service.Head(ctx, "bad id")
	require.Error(t, err)
	_, err = service.ListHeads(ctx, "bad\nID", 10)
	require.Error(t, err)
	_, err = service.ListRevisions(ctx, "new-setup", 0, 101)
	require.Error(t, err)
}
