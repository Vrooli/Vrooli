package shots_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"strings"
	"testing"
	"time"

	"device-studio/internal/presentations"
	"device-studio/internal/projects"
	"device-studio/internal/rendering"
	"device-studio/internal/shots"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	"google.golang.org/protobuf/proto"
)

type acceptingVideoSampleVerifier struct {
	wantProjectID string
	wantShotID    string
	wantBaseSHA   string
	called        bool
}

func (v *acceptingVideoSampleVerifier) VerifyVideoSample(_ context.Context, projectID, shotID string, _ int64, _ *shotsv1.ShotSnapshot, baseSHA string, _ *shotsv1.VideoSampleBinding) error {
	v.called = true
	if projectID != v.wantProjectID || shotID != v.wantShotID || baseSHA != v.wantBaseSHA {
		return errors.New("unexpected sample verification binding")
	}
	return nil
}

type mediaService struct {
	items map[string]projects.SourceMedia
	data  map[string][]byte
	opens int
}

type presentationVersionRepository struct {
	shots.Repository
	version string
}

func (r presentationVersionRepository) GetRevision(ctx context.Context, id string, number int64) (shots.Revision, error) {
	revision, err := r.Repository.GetRevision(ctx, id, number)
	if err == nil {
		revision.PresentationVersion = r.version
	}
	return revision, err
}

func (m *mediaService) Get(_ context.Context, projectID, id string) (projects.SourceMedia, error) {
	item, ok := m.items[id]
	if !ok || item.ProjectID != projectID {
		return projects.SourceMedia{}, projects.ErrSourceMediaNotFound{ID: id}
	}
	return item, nil
}

func (m *mediaService) Open(ctx context.Context, projectID, id string) (projects.SourceMedia, io.ReadCloser, error) {
	m.opens++
	item, err := m.Get(ctx, projectID, id)
	if err != nil {
		return projects.SourceMedia{}, nil, err
	}
	data := m.data[id]
	sum := sha256.Sum256(data)
	if int64(len(data)) != item.SizeBytes || hex.EncodeToString(sum[:]) != item.SHA256 {
		return projects.SourceMedia{}, nil, errors.New("test media bytes fail digest check")
	}
	return item, io.NopCloser(bytes.NewReader(data)), nil
}

func (*mediaService) Import(context.Context, string, string, int64, io.Reader) (projects.SourceMedia, error) {
	return projects.SourceMedia{}, errors.New("not implemented in shot test fake")
}

func (*mediaService) BeginImport(context.Context, string, string, int64, io.Reader) (projects.StagedSourceMediaImport, error) {
	return nil, errors.New("not implemented in shot test fake")
}

func (*mediaService) BindInspectionJob(context.Context, string, string, string, string) (projects.SourceMedia, error) {
	return projects.SourceMedia{}, errors.New("not implemented in shot test fake")
}

func (*mediaService) BindInspectionAttempt(context.Context, string, string, string, string, string, string) (projects.SourceMedia, error) {
	return projects.SourceMedia{}, errors.New("not implemented in shot test fake")
}

func (*mediaService) StoreInspectionArtifacts(context.Context, string, string, string, string, string, string, []byte, []byte) (string, string, error) {
	return "", "", errors.New("not implemented in shot test fake")
}

func (*mediaService) OpenInspectionArtifacts(context.Context, string, string, string, string, string, string) ([]byte, []byte, error) {
	return nil, nil, errors.New("not implemented in shot test fake")
}

func (*mediaService) ProjectInspection(context.Context, projects.InspectionProjection) error {
	return errors.New("not implemented in shot test fake")
}

func (*mediaService) List(context.Context, string, int32, string) ([]projects.SourceMedia, string, error) {
	return nil, "", errors.New("not implemented in shot test fake")
}

func setup(t *testing.T) (shots.Service, string, *mediaService) {
	service, projectID, media, _, _ := setupAll(t)
	return service, projectID, media
}

func setupAll(t *testing.T) (shots.Service, string, *mediaService, shots.Repository, projects.Service) {
	t.Helper()
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db,
		apidb.SchemaProviderFunc(projects.Schema), apidb.SchemaProviderFunc(shots.Schema)))
	projectRepo := projects.NewSQLiteRepository(db, nil)
	projectService := projects.NewService(projectRepo, nil)
	project, err := projectService.Create(ctx, projects.CreateInput{Name: "Test"})
	require.NoError(t, err)
	screenBytes := testPNG(t, 400, 800, 255, color.NRGBA{R: 40, G: 90, B: 150, A: 255})
	screenDigest := sha256.Sum256(screenBytes)
	media := &mediaService{items: map[string]projects.SourceMedia{
		"plate":  {ID: "plate", ProjectID: project.ID, SHA256: "plate-sha", SizeBytes: 500, MediaType: "image/png", Kind: projects.MediaKindImage, Readiness: projects.ReadinessReady, Width: 1200, Height: 800, RasterKnown: true, ColorModel: "rgba", ColorProfileStatus: "srgb"},
		"active": {ID: "active", ProjectID: project.ID, SHA256: "active-sha", SizeBytes: 200, MediaType: "image/png", Kind: projects.MediaKindImage, Readiness: projects.ReadinessReady, Width: 1200, Height: 800, RasterKnown: true, ColorModel: "grayscale", ColorProfileStatus: "srgb"},
		"screen": {ID: "screen", ProjectID: project.ID, SHA256: hex.EncodeToString(screenDigest[:]), SizeBytes: int64(len(screenBytes)), MediaType: "image/png", Kind: projects.MediaKindImage, Readiness: projects.ReadinessReady, Width: 400, Height: 800, RasterKnown: true, ColorModel: "rgba", ColorProfileStatus: "srgb"},
	}, data: map[string][]byte{"screen": screenBytes}}
	repo := shots.NewSQLiteRepository(db)
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	service := shots.NewService(repo, projectService, media, func() time.Time { return now }, nil)
	return service, project.ID, media, repo, projectService
}

func TestPresentationValidationNeverOpensPendingVideoOrOversizedImage(t *testing.T) {
	s, projectID, media := setup(t)
	media.items["pending-video"] = projects.SourceMedia{
		ID: "pending-video", ProjectID: projectID, SHA256: strings.Repeat("a", 64), SizeBytes: projects.MaxSourceVideoBytes,
		MediaType: "video/mp4", Kind: projects.MediaKindVideo, Readiness: projects.ReadinessPending,
	}
	media.items["oversized-image"] = projects.SourceMedia{
		ID: "oversized-image", ProjectID: projectID, SHA256: strings.Repeat("b", 64), SizeBytes: projects.MaxSourceImageBytes + 1,
		MediaType: "image/png", Kind: projects.MediaKindImage, Readiness: projects.ReadinessReady, Width: 400, Height: 800, RasterKnown: true,
	}
	for _, mediaID := range []string{"pending-video", "oversized-image"} {
		t.Run(mediaID, func(t *testing.T) {
			snapshot := validSnapshot()
			snapshot.Presentation.SourceMediaId = mediaID
			before := media.opens
			revision, err := s.Validate(context.Background(), projectID, snapshot)
			require.NoError(t, err)
			require.False(t, revision.RenderEligible)
			require.Equal(t, before, media.opens, "unsupported source must be rejected before Open")
			require.Contains(t, issueFields(revision), "presentation.source_media_id")
		})
	}
}

func testPNG(t *testing.T, width, height int, alpha uint8, base color.NRGBA) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixel := base
			pixel.R += uint8(x % 16)
			pixel.G += uint8(y % 16)
			pixel.A = alpha
			img.SetNRGBA(x, y, pixel)
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func validSnapshot() *shotsv1.ShotSnapshot {
	return &shotsv1.ShotSnapshot{
		Scene: &shotsv1.PhotographicScene{
			PlateMediaId: "plate", ActiveMaskMediaId: "active", DisplayWidthPixels: 400, DisplayHeightPixels: 800,
			CoordinateConvention: shotsv1.CoordinateConvention_COORDINATE_CONVENTION_PLATE_RASTER_EDGE_PIXELS_V1,
			PlateQuad:            &shotsv1.PlateQuad{TopLeft: &shotsv1.ShotPoint{X: 100, Y: 50}, TopRight: &shotsv1.ShotPoint{X: 1100, Y: 50}, BottomRight: &shotsv1.ShotPoint{X: 1100, Y: 750}, BottomLeft: &shotsv1.ShotPoint{X: 100, Y: 750}},
		},
		Presentation: &shotsv1.ScreenPresentation{SourceMediaId: "screen", CaptureArea: shotsv1.CaptureArea_CAPTURE_AREA_APP_ONLY, Kind: shotsv1.PresentationKind_PRESENTATION_KIND_PLAIN},
		MotionMode:   shotsv1.MotionMode_MOTION_MODE_FIXED,
	}
}

func TestCreateSaveConflictAndImmutableRevision(t *testing.T) { // [REQ:DS-CONTRACT-SHOT]
	s, projectID, _ := setup(t)
	ctx := context.Background()
	shot, first, err := s.Create(ctx, projectID, "  First  ", validSnapshot())
	require.NoError(t, err)
	require.True(t, first.RenderEligible)
	require.Equal(t, int64(1), first.Revision)
	require.NotEmpty(t, first.RecipeSHA256)
	require.NotEmpty(t, first.SceneSHA256)
	require.NotEmpty(t, first.PresentationSHA256)
	require.Equal(t, "plate-sha", first.Plate.GetSha256())
	require.Equal(t, shotsv1.ScreenFitMode_SCREEN_FIT_MODE_CONTAIN, first.Snapshot.Presentation.GetFitMode())
	require.Equal(t, rendering.CompositorProfile, first.CompositorProfile)

	secondSnapshot := validSnapshot()
	secondSnapshot.Scene.DepictionLabel = "front view"
	updated, second, err := s.Save(ctx, shot.ID, 1, "Renamed", secondSnapshot)
	require.NoError(t, err)
	require.Equal(t, int64(2), updated.HeadRevision)
	require.Equal(t, "Renamed", updated.Name)
	require.NotEqual(t, first.RecipeSHA256, second.RecipeSHA256)

	old, err := s.GetRevision(ctx, shot.ID, 1)
	require.NoError(t, err)
	require.Equal(t, first.RecipeSHA256, old.RecipeSHA256)
	require.Equal(t, "", old.Snapshot.Scene.DepictionLabel)
	_, _, err = s.Save(ctx, shot.ID, 1, "Stale", validSnapshot())
	var conflict shots.ErrRevisionConflict
	require.ErrorAs(t, err, &conflict)
	require.EqualValues(t, 2, conflict.Current)
}

func TestVideoSampleSaveUsesDistinctBaseAndFullRecipeHashes(t *testing.T) {
	s, projectID, media, _, _ := setupAll(t)
	streamIndex, startPTS, endPTS := int64(0), int64(0), int64(1000)
	videoSHA, clockSHA, receiptSHA := strings.Repeat("b", 64), strings.Repeat("c", 64), strings.Repeat("d", 64)
	media.items["video"] = projects.SourceMedia{
		ID: "video", ProjectID: projectID, SHA256: videoSHA, SizeBytes: 1234, MediaType: "video/mp4",
		Kind: projects.MediaKindVideo, Readiness: projects.ReadinessReady, InspectionJobID: "inspect-job", InspectionAttemptID: "inspect-attempt",
		InspectionReceipt: &projects.InspectionReceipt{
			Version: 1, ID: "inspect-receipt", SHA256: receiptSHA, SourceSHA256: videoSHA, SourceSizeBytes: 1234,
			JobID: "inspect-job", AttemptID: "inspect-attempt", ToolProfileID: "ffmpeg-video-v1", ToolProfileDigest: strings.Repeat("e", 64),
			InspectionProtocolVersion: 1, Classification: "accepted", VideoStreamIndex: &streamIndex,
			TimeBaseNumerator: 1, TimeBaseDenominator: 1000, StreamStartPTS: &startPTS, SourceEndPTS: &endPTS,
			LastFrameDurationTicks: 40, ClockSHA256: clockSHA, ClockArtifactRef: "opaque-clock", ClockArtifactSizeBytes: 24,
		},
	}
	base := validSnapshot()
	base.Presentation.SourceMediaId = "video"
	base.Presentation.Kind = shotsv1.PresentationKind_PRESENTATION_KIND_PHONE_SYSTEM
	base.Presentation.CaptureArea = shotsv1.CaptureArea_CAPTURE_AREA_APP_ONLY
	base.Presentation.PhoneSystemChrome = &shotsv1.PhoneSystemChrome{Appearance: shotsv1.PresentationAppearance_PRESENTATION_APPEARANCE_AUTO}
	base.VideoTiming = &shotsv1.VideoTiming{
		SourceMediaId: "video", SourceSha256: videoSHA, SourceSizeBytes: 1234,
		InspectionJobId: "inspect-job", InspectionAttemptId: "inspect-attempt", InspectionReceiptId: "inspect-receipt", InspectionReceiptSha256: receiptSHA,
		ClockArtifactId: "opaque-clock", ClockSha256: clockSHA, ClockSizeBytes: 24,
		SelectedVideoStream: 0, TimeBaseNum: 1, TimeBaseDen: 1000, StreamStartPts: &startPTS, SourceEndPts: endPTS,
		LastFrameDurationTicks: 40, TrimStartPts: 100, TrimEndPts: 500, PlaybackRateNum: 1, PlaybackRateDen: 1,
		OutputRateNum: 24, OutputRateDen: 1, EofPolicy: shotsv1.VideoEOFPolicy_VIDEO_EOF_POLICY_REJECT,
		VideoCompositionProfileId: "device-studio-fixed-plate-video-v1", VideoCompositionProfileVersion: 1,
	}
	shot, baseRevision, err := s.Create(context.Background(), projectID, "Moving phone", base)
	require.NoError(t, err)
	require.True(t, baseRevision.RenderEligible, "%+v", baseRevision.Issues)
	baseSHA, err := shots.VideoBaseRecipeSHA(baseRevision.Snapshot)
	require.NoError(t, err)
	verifier := &acceptingVideoSampleVerifier{wantProjectID: projectID, wantShotID: shot.ID, wantBaseSHA: baseSHA}
	shots.SetVideoSampleVerifier(s, verifier)

	updatedSnapshot := proto.Clone(baseRevision.Snapshot).(*shotsv1.ShotSnapshot)
	updatedSnapshot.VideoTiming.SampleBaseRecipeVersion = 1
	updatedSnapshot.VideoTiming.SampleBinding = &shotsv1.VideoSampleBinding{
		JobId: "sample-job", AttemptId: "sample-attempt", ReceiptId: "sample-job:sample-attempt", ReceiptSha256: strings.Repeat("f", 64),
		BaseShotRevision: 1, BaseRecipeVersion: "device-studio-video-base-recipe-v1", BaseRecipeSha256: baseSHA,
		ResolvedTopFillRed: 20, ResolvedTopFillGreen: 30, ResolvedTopFillBlue: 40,
		ResolvedBottomFillRed: 50, ResolvedBottomFillGreen: 60, ResolvedBottomFillBlue: 70,
		TopFillSourceResolved: true, BottomFillSourceResolved: true, StatusInkSourceResolved: true, HomeInkSourceResolved: true,
	}
	_, saved, err := s.Save(context.Background(), shot.ID, 1, shot.Name, updatedSnapshot)
	require.NoError(t, err)
	require.True(t, verifier.called, "save must consult the trusted sample verifier")
	require.Equal(t, baseSHA, saved.Snapshot.GetVideoTiming().GetSampleBinding().GetBaseRecipeSha256())
	require.NotEqual(t, baseSHA, saved.RecipeSHA256, "full saved recipe and sample base recipe must remain distinct hashes")
	require.Equal(t, baseSHA, func() string {
		got, hashErr := shots.VideoBaseRecipeSHA(saved.Snapshot)
		require.NoError(t, hashErr)
		return got
	}())
}

func TestManualAndBrowserVideoRevisionsDoNotRequireSampleBinding(t *testing.T) {
	s, projectID, media := setup(t)
	video := acceptedVideoSource(projectID)
	media.items[video.ID] = video
	for _, kind := range []shotsv1.PresentationKind{
		shotsv1.PresentationKind_PRESENTATION_KIND_PLAIN,
		shotsv1.PresentationKind_PRESENTATION_KIND_BROWSER_DESKTOP,
	} {
		t.Run(kind.String(), func(t *testing.T) {
			snapshot := validSnapshot()
			snapshot.Presentation.SourceMediaId = video.ID
			snapshot.Presentation.Kind = kind
			if kind == shotsv1.PresentationKind_PRESENTATION_KIND_BROWSER_DESKTOP {
				snapshot.Presentation.BrowserDesktopChrome = &shotsv1.BrowserDesktopChrome{WindowTitle: "Source preview", AddressText: "example.test"}
			}
			snapshot.VideoTiming = acceptedVideoTiming(video)
			shot, revision, err := s.Create(context.Background(), projectID, "Video preview", snapshot)
			require.NoError(t, err)
			require.True(t, revision.RenderEligible, "%+v", revision.Issues)
			require.Equal(t, presentations.SemanticVersionV2, revision.PresentationVersion, "video app rasters must use the worker-supported saved presentation version")
			_, configErr := presentations.ConfigFromSavedVersion(image.Pt(120, 120), revision.Snapshot.GetPresentation(), revision.PresentationVersion)
			require.NoError(t, configErr, "an eligible plain video revision must build the worker presentation")
			require.Nil(t, revision.Snapshot.GetVideoTiming().GetSampleBinding())
			compiled, err := s.Compile(context.Background(), shot.ID, revision.Revision)
			require.NoError(t, err)
			require.Equal(t, revision.RecipeSHA256, compiled.RecipeSHA256)
			require.Nil(t, compiled.Snapshot.GetVideoTiming().GetSampleBinding())
		})
	}
}

func TestPhotographicCameraMotionValidationAndRecipePinning(t *testing.T) {
	s, projectID, media := setup(t)
	video := acceptedVideoSource(projectID)
	media.items[video.ID] = video

	makeCameraSnapshot := func() *shotsv1.ShotSnapshot {
		snapshot := validSnapshot()
		snapshot.Presentation.SourceMediaId = video.ID
		snapshot.VideoTiming = acceptedVideoTiming(video)
		snapshot.VideoTiming.VideoCompositionProfileId = shots.VideoProfileScreenEntry
		snapshot.MotionMode = shotsv1.MotionMode_MOTION_MODE_CROP_PUSH
		snapshot.CameraMotion = &shotsv1.PhotographicCameraMotion{
			ProfileId: shots.CameraMotionProfileID, ProfileVersion: shots.CameraMotionProfileVersion,
			DurationOutputFrames: 10, MatteRgb: &shotsv1.PresentationRGB{},
		}
		return snapshot
	}

	shot, first, err := s.Create(context.Background(), projectID, "Screen entry", makeCameraSnapshot())
	require.NoError(t, err)
	require.True(t, first.RenderEligible, "%+v", first.Issues)
	require.Equal(t, shots.CameraMotionProfileID, first.Snapshot.GetCameraMotion().GetProfileId())
	baseRecipe, err := shots.VideoBaseRecipeSHA(first.Snapshot)
	require.NoError(t, err)

	changed := proto.Clone(first.Snapshot).(*shotsv1.ShotSnapshot)
	changed.CameraMotion.MatteRgb.Red = 12
	changed.CameraMotion.TargetRect = &shotsv1.PresentationRect{X: 4000, Y: 1000, Width: 5500, Height: 8000}
	changedBaseRecipe, err := shots.VideoBaseRecipeSHA(changed)
	require.NoError(t, err)
	require.NotEqual(t, baseRecipe, changedBaseRecipe, "camera config changes invalidate the accepted sample base recipe")
	_, second, err := s.Save(context.Background(), shot.ID, first.Revision, shot.Name, changed)
	require.NoError(t, err)
	require.NotEqual(t, first.RecipeSHA256, second.RecipeSHA256, "camera matte is part of the immutable recipe")
	require.Equal(t, uint32(4000), second.Snapshot.GetCameraMotion().GetTargetRect().GetX())
	old, err := s.GetRevision(context.Background(), shot.ID, first.Revision)
	require.NoError(t, err)
	require.Nil(t, old.Snapshot.GetCameraMotion().GetTargetRect())

	invalid := []struct {
		name   string
		mutate func(*shotsv1.ShotSnapshot)
	}{
		{"fixed forbids camera", func(s *shotsv1.ShotSnapshot) { s.MotionMode = shotsv1.MotionMode_MOTION_MODE_FIXED }},
		{"crop push requires camera", func(s *shotsv1.ShotSnapshot) { s.CameraMotion = nil }},
		{"target empty", func(s *shotsv1.ShotSnapshot) { s.CameraMotion.TargetRect = &shotsv1.PresentationRect{} }},
		{"target overflow", func(s *shotsv1.ShotSnapshot) {
			s.CameraMotion.TargetRect = &shotsv1.PresentationRect{X: 4000, Width: 7000, Height: 10000}
		}},
		{"target uint max", func(s *shotsv1.ShotSnapshot) {
			s.CameraMotion.TargetRect = &shotsv1.PresentationRect{X: 4294967295, Width: 1, Height: 10000}
		}},
		{"matte required", func(s *shotsv1.ShotSnapshot) { s.CameraMotion.MatteRgb = nil }},
		{"matte range", func(s *shotsv1.ShotSnapshot) { s.CameraMotion.MatteRgb.Red = 256 }},
		{"profile version", func(s *shotsv1.ShotSnapshot) { s.CameraMotion.ProfileVersion = 2 }},
		{"profile id", func(s *shotsv1.ShotSnapshot) { s.CameraMotion.ProfileId = "unknown" }},
		{"duration minimum", func(s *shotsv1.ShotSnapshot) { s.CameraMotion.DurationOutputFrames = 1 }},
		{"duration maximum", func(s *shotsv1.ShotSnapshot) { s.CameraMotion.DurationOutputFrames = 361 }},
		{"duration exceeds schedule", func(s *shotsv1.ShotSnapshot) { s.CameraMotion.DurationOutputFrames = 11 }},
	}
	for _, test := range invalid {
		t.Run(test.name, func(t *testing.T) {
			snapshot := makeCameraSnapshot()
			test.mutate(snapshot)
			_, _, err := s.Create(context.Background(), projectID, "Invalid camera", snapshot)
			require.Error(t, err)
		})
	}
	wrongTimingProfile := makeCameraSnapshot()
	wrongTimingProfile.VideoTiming.VideoCompositionProfileId = shots.VideoProfileFixed
	_, revision, err := s.Create(context.Background(), projectID, "Wrong timing profile", wrongTimingProfile)
	require.NoError(t, err)
	require.False(t, revision.RenderEligible)
	require.Contains(t, issueFields(revision), "video_timing")
}

func TestCropPushWithoutTimingRemainsExplicitlyIncomplete(t *testing.T) {
	s, projectID, _ := setup(t)
	snapshot := validSnapshot()
	snapshot.MotionMode = shotsv1.MotionMode_MOTION_MODE_CROP_PUSH
	snapshot.CameraMotion = &shotsv1.PhotographicCameraMotion{
		ProfileId: shots.CameraMotionProfileID, ProfileVersion: shots.CameraMotionProfileVersion,
		DurationOutputFrames: 10, MatteRgb: &shotsv1.PresentationRGB{},
	}
	_, revision, err := s.Create(context.Background(), projectID, "Draft camera", snapshot)
	require.NoError(t, err)
	require.False(t, revision.RenderEligible)
	require.Contains(t, issueFields(revision), "camera_motion")
}

func TestCameraProfileAcceptsEightToTwelveSecondSchedulesAtBothRates(t *testing.T) {
	for _, rate := range []int64{24, 30} {
		for _, seconds := range []int64{8, 12} {
			t.Run(fmt.Sprintf("%d-fps-%d-seconds", rate, seconds), func(t *testing.T) {
				s, projectID, media := setup(t)
				video := acceptedVideoSource(projectID)
				endPTS := int64(15_000)
				video.InspectionReceipt.SourceEndPTS = &endPTS
				media.items[video.ID] = video

				snapshot := validSnapshot()
				snapshot.Presentation.SourceMediaId = video.ID
				snapshot.VideoTiming = acceptedVideoTiming(video)
				snapshot.VideoTiming.VideoCompositionProfileId = shots.VideoProfileScreenEntry
				snapshot.VideoTiming.TrimStartPts = 0
				snapshot.VideoTiming.TrimEndPts = seconds * 1000
				snapshot.VideoTiming.OutputRateNum = uint32(rate)
				snapshot.MotionMode = shotsv1.MotionMode_MOTION_MODE_CROP_PUSH
				frames := uint32(seconds * rate)
				snapshot.CameraMotion = &shotsv1.PhotographicCameraMotion{
					ProfileId: shots.CameraMotionProfileID, ProfileVersion: shots.CameraMotionProfileVersion,
					DurationOutputFrames: frames, MatteRgb: &shotsv1.PresentationRGB{},
				}
				_, revision, err := s.Create(context.Background(), projectID, "Screen entry duration", snapshot)
				require.NoError(t, err)
				require.True(t, revision.RenderEligible, "%+v", revision.Issues)
				require.Equal(t, frames, revision.Snapshot.GetCameraMotion().GetDurationOutputFrames())
			})
		}
	}
}

func TestCameraProfileRejectsSchedulesLongerThanTwelveSeconds(t *testing.T) {
	s, projectID, media := setup(t)
	video := acceptedVideoSource(projectID)
	endPTS := int64(15_000)
	video.InspectionReceipt.SourceEndPTS = &endPTS
	media.items[video.ID] = video

	snapshot := validSnapshot()
	snapshot.Presentation.SourceMediaId = video.ID
	snapshot.VideoTiming = acceptedVideoTiming(video)
	snapshot.VideoTiming.VideoCompositionProfileId = shots.VideoProfileScreenEntry
	snapshot.VideoTiming.TrimStartPts = 0
	snapshot.VideoTiming.TrimEndPts = 12_001 // 12.001 seconds: 289 frames at 24 fps.
	snapshot.MotionMode = shotsv1.MotionMode_MOTION_MODE_CROP_PUSH
	snapshot.CameraMotion = &shotsv1.PhotographicCameraMotion{
		ProfileId: shots.CameraMotionProfileID, ProfileVersion: shots.CameraMotionProfileVersion,
		DurationOutputFrames: 120, MatteRgb: &shotsv1.PresentationRGB{},
	}

	_, revision, err := s.Create(context.Background(), projectID, "Overlong screen entry", snapshot)
	require.NoError(t, err)
	require.False(t, revision.RenderEligible)
	var durationIssue *shotsv1.ShotIssue
	for _, candidate := range revision.Issues {
		if candidate.GetFieldPath() == "video_timing" {
			durationIssue = candidate
			break
		}
	}
	require.NotNil(t, durationIssue)
	require.Contains(t, durationIssue.GetSummary(), "12 seconds")
	require.Contains(t, durationIssue.GetRepair(), "12 seconds")
}

func acceptedVideoSource(projectID string) projects.SourceMedia {
	streamIndex, startPTS, endPTS := int64(0), int64(0), int64(1000)
	videoSHA, clockSHA, receiptSHA := strings.Repeat("b", 64), strings.Repeat("c", 64), strings.Repeat("d", 64)
	return projects.SourceMedia{
		ID: "video", ProjectID: projectID, SHA256: videoSHA, SizeBytes: 1234, MediaType: "video/mp4",
		Kind: projects.MediaKindVideo, Readiness: projects.ReadinessReady, InspectionJobID: "inspect-job", InspectionAttemptID: "inspect-attempt",
		InspectionReceipt: &projects.InspectionReceipt{
			Version: 1, ID: "inspect-receipt", SHA256: receiptSHA, SourceSHA256: videoSHA, SourceSizeBytes: 1234,
			JobID: "inspect-job", AttemptID: "inspect-attempt", ToolProfileID: "ffmpeg-video-v1", ToolProfileDigest: strings.Repeat("e", 64),
			InspectionProtocolVersion: 1, Classification: "accepted", VideoStreamIndex: &streamIndex,
			TimeBaseNumerator: 1, TimeBaseDenominator: 1000, StreamStartPTS: &startPTS, SourceEndPTS: &endPTS,
			LastFrameDurationTicks: 40, ClockSHA256: clockSHA, ClockArtifactRef: "opaque-clock", ClockArtifactSizeBytes: 24,
		},
	}
}

func acceptedVideoTiming(video projects.SourceMedia) *shotsv1.VideoTiming {
	r := video.InspectionReceipt
	return &shotsv1.VideoTiming{
		SourceMediaId: video.ID, SourceSha256: video.SHA256, SourceSizeBytes: video.SizeBytes,
		InspectionJobId: r.JobID, InspectionAttemptId: r.AttemptID, InspectionReceiptId: r.ID, InspectionReceiptSha256: r.SHA256,
		ClockArtifactId: r.ClockArtifactRef, ClockSha256: r.ClockSHA256, ClockSizeBytes: r.ClockArtifactSizeBytes,
		SelectedVideoStream: uint32(*r.VideoStreamIndex), TimeBaseNum: r.TimeBaseNumerator, TimeBaseDen: r.TimeBaseDenominator,
		StreamStartPts: r.StreamStartPTS, SourceEndPts: *r.SourceEndPTS, LastFrameDurationTicks: r.LastFrameDurationTicks,
		TrimStartPts: 100, TrimEndPts: 500, PlaybackRateNum: 1, PlaybackRateDen: 1,
		OutputRateNum: 24, OutputRateDen: 1, EofPolicy: shotsv1.VideoEOFPolicy_VIDEO_EOF_POLICY_REJECT,
		VideoCompositionProfileId: "device-studio-fixed-plate-video-v1", VideoCompositionProfileVersion: 1,
	}
}

func TestShotAndRevisionPagesRemainStableAfterSave(t *testing.T) {
	s, projectID, _ := setup(t)
	ctx := context.Background()
	created := make([]shots.Shot, 0, 3)
	for _, name := range []string{"One", "Two", "Three"} {
		shot, _, err := s.Create(ctx, projectID, name, &shotsv1.ShotSnapshot{})
		require.NoError(t, err)
		created = append(created, shot)
	}
	first, token, err := s.List(ctx, projectID, 2, "")
	require.NoError(t, err)
	require.Len(t, first, 2)
	require.NotEmpty(t, token)
	_, _, err = s.Save(ctx, first[0].ID, first[0].HeadRevision, "Renamed", &shotsv1.ShotSnapshot{})
	require.NoError(t, err)
	second, end, err := s.List(ctx, projectID, 2, token)
	require.NoError(t, err)
	require.Len(t, second, 1)
	require.Empty(t, end)
	require.NotEqual(t, first[0].ID, second[0].ID)
	require.NotEqual(t, first[1].ID, second[0].ID)

	pages, next, err := s.ListRevisions(ctx, first[0].ID, 1, "")
	require.NoError(t, err)
	require.Len(t, pages, 1)
	require.NotEmpty(t, next)
	older, end, err := s.ListRevisions(ctx, first[0].ID, 1, next)
	require.NoError(t, err)
	require.Len(t, older, 1)
	require.Empty(t, end)
	require.EqualValues(t, 1, older[0].Revision)
}

func TestDraftUnsupportedAndMalformedGeometryPolicies(t *testing.T) { // [REQ:DS-PHOTO-ELIGIBILITY]
	s, projectID, _ := setup(t)
	draft, err := s.Validate(context.Background(), projectID, &shotsv1.ShotSnapshot{})
	require.NoError(t, err)
	require.False(t, draft.RenderEligible)
	require.NotEmpty(t, draft.Issues)

	unsupported := validSnapshot()
	unsupported.MotionMode = shotsv1.MotionMode_MOTION_MODE_PARALLAX
	_, revision, err := s.Create(context.Background(), projectID, "Unsupported", unsupported)
	require.NoError(t, err)
	require.False(t, revision.RenderEligible)
	_, err = s.ResolveForRender(context.Background(), revision.ShotID, revision.Revision)
	var blocked shots.ErrRenderBlocked
	require.ErrorAs(t, err, &blocked)

	malformed := validSnapshot()
	malformed.Scene.PlateQuad.BottomLeft.X = math.NaN()
	_, _, err = s.Create(context.Background(), projectID, "Bad quad", malformed)
	var invalid shots.ErrInvalidShot
	require.ErrorAs(t, err, &invalid)
}

func TestCrossProjectMediaRejectedAndSemanticDigestTracksBytes(t *testing.T) {
	s, projectID, media := setup(t)
	snapshot := validSnapshot()
	snapshot.Presentation.SourceMediaId = "other-project"
	_, _, err := s.Create(context.Background(), projectID, "Wrong owner", snapshot)
	var invalid shots.ErrInvalidShot
	require.ErrorAs(t, err, &invalid)

	snapshot = validSnapshot()
	first, err := s.Validate(context.Background(), projectID, snapshot)
	require.NoError(t, err)
	item := media.items["screen"]
	changed := testPNG(t, 400, 800, 255, color.NRGBA{R: 160, G: 30, B: 20, A: 255})
	sum := sha256.Sum256(changed)
	item.SHA256 = hex.EncodeToString(sum[:])
	item.SizeBytes = int64(len(changed))
	media.items["screen"] = item
	media.data["screen"] = changed
	second, err := s.Validate(context.Background(), projectID, snapshot)
	require.NoError(t, err)
	require.NotEqual(t, first.PresentationSHA256, second.PresentationSHA256)
}

func TestSimulatedSourceBytesMustMatchResolvedDimensions(t *testing.T) {
	s, projectID, media := setup(t)
	changed := testPNG(t, 401, 800, 255, color.NRGBA{R: 20, G: 40, B: 60, A: 255})
	sum := sha256.Sum256(changed)
	item := media.items["screen"]
	item.SHA256 = hex.EncodeToString(sum[:])
	item.SizeBytes = int64(len(changed))
	media.items["screen"], media.data["screen"] = item, changed

	snapshot := validSnapshot()
	snapshot.Presentation.Kind = shotsv1.PresentationKind_PRESENTATION_KIND_PHONE_SYSTEM
	snapshot.Presentation.PhoneSystemChrome = &shotsv1.PhoneSystemChrome{}
	_, _, err := s.Create(context.Background(), projectID, "Mismatched screen metadata", snapshot)
	var invalid shots.ErrInvalidShot
	require.ErrorAs(t, err, &invalid)
}

func TestSemanticDigestsBindContentRatherThanMutableMediaIds(t *testing.T) { // [REQ:DS-CONTRACT-SHOT]
	s, projectID, media := setup(t)
	snapshot := validSnapshot()
	first, err := s.Validate(context.Background(), projectID, snapshot)
	require.NoError(t, err)
	media.items["plate-alias"] = media.items["plate"]
	snapshot.Scene.PlateMediaId = "plate-alias"
	second, err := s.Validate(context.Background(), projectID, snapshot)
	require.NoError(t, err)
	require.Equal(t, first.SceneSHA256, second.SceneSHA256)
	require.NotEqual(t, first.RecipeSHA256, second.RecipeSHA256, "the full canonical recipe retains the chosen opaque reference")
}

func TestUnknownEnumsAreRejected(t *testing.T) {
	s, projectID, _ := setup(t)
	snapshot := validSnapshot()
	snapshot.MotionMode = shotsv1.MotionMode(99)
	_, _, err := s.Create(context.Background(), projectID, "Unknown", snapshot)
	var invalid shots.ErrInvalidShot
	require.ErrorAs(t, err, &invalid)
}

func TestUnknownCoordinateEnumRejectedWithoutGeometry(t *testing.T) {
	s, projectID, _ := setup(t)
	snapshot := &shotsv1.ShotSnapshot{Scene: &shotsv1.PhotographicScene{CoordinateConvention: shotsv1.CoordinateConvention(91)}}
	_, _, err := s.Create(context.Background(), projectID, "Unknown coordinate", snapshot)
	var invalid shots.ErrInvalidShot
	require.ErrorAs(t, err, &invalid)
}

func TestChromePayloadMustMatchPresentationAndCapture(t *testing.T) { // [REQ:DS-SCREEN-CHROME]
	s, projectID, _ := setup(t)
	tests := []struct {
		name   string
		mutate func(*shotsv1.ScreenPresentation)
	}{
		{"phone payload on plain", func(p *shotsv1.ScreenPresentation) {
			p.PhoneSystemChrome = &shotsv1.PhoneSystemChrome{CarrierText: "Carrier"}
		}},
		{"browser payload on plain", func(p *shotsv1.ScreenPresentation) {
			p.BrowserDesktopChrome = &shotsv1.BrowserDesktopChrome{WindowTitle: "Window"}
		}},
		{"chrome on already composed", func(p *shotsv1.ScreenPresentation) {
			p.CaptureArea = shotsv1.CaptureArea_CAPTURE_AREA_ALREADY_COMPOSED
			p.PhoneSystemChrome = &shotsv1.PhoneSystemChrome{}
			p.Kind = shotsv1.PresentationKind_PRESENTATION_KIND_PLAIN
		}},
		{"battery out of range", func(p *shotsv1.ScreenPresentation) {
			p.Kind = shotsv1.PresentationKind_PRESENTATION_KIND_PHONE_SYSTEM
			battery := uint32(101)
			p.PhoneSystemChrome = &shotsv1.PhoneSystemChrome{BatteryPercent: &battery}
		}},
		{"overlong address", func(p *shotsv1.ScreenPresentation) {
			p.Kind = shotsv1.PresentationKind_PRESENTATION_KIND_BROWSER_DESKTOP
			p.BrowserDesktopChrome = &shotsv1.BrowserDesktopChrome{AddressText: strings.Repeat("a", 2049)}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			snapshot := validSnapshot()
			tc.mutate(snapshot.Presentation)
			_, _, err := s.Create(context.Background(), projectID, tc.name, snapshot)
			var invalid shots.ErrInvalidShot
			require.ErrorAs(t, err, &invalid)
		})
	}
}

func TestSavedPhoneAndBrowserPresentationsCompileToVersionedProfiles(t *testing.T) {
	s, projectID, _ := setup(t)
	for _, tc := range []struct {
		name    string
		kind    shotsv1.PresentationKind
		phone   *shotsv1.PhoneSystemChrome
		browser *shotsv1.BrowserDesktopChrome
		version string
		profile string
	}{
		{"phone", shotsv1.PresentationKind_PRESENTATION_KIND_PHONE_SYSTEM, &shotsv1.PhoneSystemChrome{}, nil, "device-studio-presentation-v3", "go-still-presentation-v2"},
		{"browser", shotsv1.PresentationKind_PRESENTATION_KIND_BROWSER_DESKTOP, nil, &shotsv1.BrowserDesktopChrome{}, "device-studio-presentation-v2", "go-still-presentation-v2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snapshot := validSnapshot()
			snapshot.Presentation.Kind = tc.kind
			snapshot.Presentation.CaptureArea = shotsv1.CaptureArea_CAPTURE_AREA_APP_ONLY
			snapshot.Presentation.PhoneSystemChrome = tc.phone
			snapshot.Presentation.BrowserDesktopChrome = tc.browser
			revision, err := s.Validate(context.Background(), projectID, snapshot)
			require.NoError(t, err)
			require.True(t, revision.RenderEligible, "%+v", revision.Issues)
			require.Equal(t, tc.version, revision.PresentationVersion)
			require.Equal(t, tc.profile, revision.CompositorProfile)
		})
	}
}

func TestSimulatedPresentationDefaultsArePersistedAndExplicitZeroSurvives(t *testing.T) {
	s, projectID, _, _, _ := setupAll(t)
	snapshot := validSnapshot()
	snapshot.Presentation.Kind = shotsv1.PresentationKind_PRESENTATION_KIND_PHONE_SYSTEM
	snapshot.Presentation.PhoneSystemChrome = nil
	shot, revision, err := s.Create(context.Background(), projectID, "Default phone", snapshot)
	require.NoError(t, err)
	require.True(t, revision.RenderEligible, "%+v", revision.Issues)
	phone := revision.Snapshot.Presentation.PhoneSystemChrome
	require.Equal(t, "9:41", phone.GetClockText())
	require.Equal(t, "", phone.GetCarrierText())
	require.EqualValues(t, 84, phone.GetBatteryPercent())
	require.Equal(t, shotsv1.PresentationAppearance_PRESENTATION_APPEARANCE_AUTO, phone.GetAppearance())
	require.True(t, phone.GetShowStatus())
	require.True(t, phone.GetShowHomeIndicator())
	require.Equal(t, "device-studio-presentation-v3", revision.PresentationVersion)
	require.Equal(t, "go-still-presentation-v2", revision.CompositorProfile)

	zero := uint32(0)
	snapshot = validSnapshot()
	snapshot.Presentation.Kind = shotsv1.PresentationKind_PRESENTATION_KIND_PHONE_SYSTEM
	snapshot.Presentation.PhoneSystemChrome = &shotsv1.PhoneSystemChrome{BatteryPercent: &zero}
	_, revision, err = s.Save(context.Background(), shot.ID, 1, "Explicit zero", snapshot)
	require.NoError(t, err)
	require.EqualValues(t, 0, revision.Snapshot.Presentation.PhoneSystemChrome.GetBatteryPercent())
}

func TestStoredPresentationVersionControlsCompileDispatch(t *testing.T) {
	s, projectID, media, repo, projectService := setupAll(t)
	browser := validSnapshot()
	browser.Presentation.Kind = shotsv1.PresentationKind_PRESENTATION_KIND_BROWSER_DESKTOP
	browser.Presentation.BrowserDesktopChrome = &shotsv1.BrowserDesktopChrome{}
	browserShot, browserRevision, err := s.Create(context.Background(), projectID, "Browser v2", browser)
	require.NoError(t, err)
	require.Equal(t, "device-studio-presentation-v2", browserRevision.PresentationVersion)
	compiled, err := s.Compile(context.Background(), browserShot.ID, browserRevision.Revision)
	require.NoError(t, err)
	require.True(t, compiled.RenderEligible, "%+v", compiled.Issues)
	require.Equal(t, "device-studio-presentation-v2", compiled.PresentationVersion)
	unsupported := shots.NewService(presentationVersionRepository{Repository: repo, version: "device-studio-presentation-v3"}, projectService, media, nil, nil)
	_, err = unsupported.Compile(context.Background(), browserShot.ID, browserRevision.Revision)
	var unsupportedVersion shots.ErrUnsupportedPresentationVersion
	require.ErrorAs(t, err, &unsupportedVersion)
}

func TestSimulatedValidationUsesVerifiedSourceTransparencyGeometryAndGlyphs(t *testing.T) {
	s, projectID, media := setup(t)
	snapshot := validSnapshot()
	snapshot.Presentation.Kind = shotsv1.PresentationKind_PRESENTATION_KIND_PHONE_SYSTEM
	snapshot.Presentation.PhoneSystemChrome = &shotsv1.PhoneSystemChrome{}
	transparent := testPNG(t, 400, 800, 0, color.NRGBA{R: 10, G: 20, B: 30, A: 0})
	sum := sha256.Sum256(transparent)
	item := media.items["screen"]
	item.SHA256, item.SizeBytes = hex.EncodeToString(sum[:]), int64(len(transparent))
	media.items["screen"], media.data["screen"] = item, transparent
	_, revision, err := s.Create(context.Background(), projectID, "Matte needed", snapshot)
	require.NoError(t, err)
	require.False(t, revision.RenderEligible)
	require.Contains(t, issueFields(revision), "presentation.matte")

	snapshot.Presentation.Matte = &shotsv1.PresentationRGB{Red: 12, Green: 24, Blue: 36}
	_, revision, err = s.Create(context.Background(), projectID, "Matte provided", snapshot)
	require.NoError(t, err)
	require.True(t, revision.RenderEligible, "%+v", revision.Issues)

	snapshot.Presentation.PhoneSystemChrome.CarrierText = "Meridian漢"
	revision, err = s.Validate(context.Background(), projectID, snapshot)
	require.NoError(t, err)
	require.False(t, revision.RenderEligible)
	require.Contains(t, issueFields(revision), "presentation.chrome")
}

func TestPhoneV3CarrierFitFailureHasActionableFieldIssue(t *testing.T) {
	s, projectID, _ := setup(t)
	snapshot := validSnapshot()
	snapshot.Presentation.Kind = shotsv1.PresentationKind_PRESENTATION_KIND_PHONE_SYSTEM
	snapshot.Presentation.PhoneSystemChrome = &shotsv1.PhoneSystemChrome{CarrierText: strings.Repeat("W", 24)}
	revision, err := s.Validate(context.Background(), projectID, snapshot)
	require.NoError(t, err)
	require.False(t, revision.RenderEligible)
	require.Contains(t, issueFields(revision), "presentation.phone_system_chrome.carrier_text")
	for _, issue := range revision.Issues {
		if issue.GetFieldPath() == "presentation.phone_system_chrome.carrier_text" {
			require.Contains(t, strings.ToLower(issue.GetRepair()), "shorten")
			return
		}
	}
	t.Fatal("carrier issue had no actionable repair")
}

func TestPlainSemanticVersionPreservesStoredHashesAndUnknownVersionFailsClosed(t *testing.T) {
	s, projectID, media, repo, projectService := setupAll(t)
	shot, original, err := s.Create(context.Background(), projectID, "Legacy plain", validSnapshot())
	require.NoError(t, err)
	compiled, err := s.Compile(context.Background(), shot.ID, original.Revision)
	require.NoError(t, err)
	require.Equal(t, "device-studio-presentation-v1", compiled.PresentationVersion)
	require.Equal(t, original.PresentationSHA256, compiled.PresentationSHA256)
	require.Equal(t, original.RecipeSHA256, compiled.RecipeSHA256)
	require.Equal(t, "go-still-homography-v1", compiled.CompositorProfile)

	unknown := shots.NewService(presentationVersionRepository{Repository: repo, version: "device-studio-presentation-v99"}, projectService, media, nil, nil)
	_, err = unknown.Compile(context.Background(), shot.ID, original.Revision)
	var unsupported shots.ErrUnsupportedPresentationVersion
	require.ErrorAs(t, err, &unsupported)
	_, _, err = s.Create(context.Background(), projectID, "Plain matte rejected", func() *shotsv1.ShotSnapshot {
		bad := validSnapshot()
		bad.Presentation.Matte = &shotsv1.PresentationRGB{}
		return bad
	}())
	var invalid shots.ErrInvalidShot
	require.ErrorAs(t, err, &invalid)
}

func issueFields(revision shots.Revision) []string {
	fields := make([]string, 0, len(revision.Issues))
	for _, issue := range revision.Issues {
		fields = append(fields, issue.GetFieldPath())
	}
	return fields
}

func TestPartialQuadRejectsNonFiniteButPersistsIncompleteGeometry(t *testing.T) {
	s, projectID, _ := setup(t)
	snapshot := &shotsv1.ShotSnapshot{Scene: &shotsv1.PhotographicScene{
		DisplayWidthPixels: 400, DisplayHeightPixels: 800,
		CoordinateConvention: shotsv1.CoordinateConvention_COORDINATE_CONVENTION_PLATE_RASTER_EDGE_PIXELS_V1,
		PlateQuad:            &shotsv1.PlateQuad{TopLeft: &shotsv1.ShotPoint{X: math.NaN(), Y: 5}},
	}}
	_, _, err := s.Create(context.Background(), projectID, "Partial invalid", snapshot)
	var invalid shots.ErrInvalidShot
	require.ErrorAs(t, err, &invalid)

	snapshot.Scene.PlateQuad.TopLeft.X = 5
	_, revision, err := s.Create(context.Background(), projectID, "Partial draft", snapshot)
	require.NoError(t, err)
	require.False(t, revision.RenderEligible)
}

func TestWorkerImageAndAggregateLimitsBlockRender(t *testing.T) {
	s, projectID, media := setup(t)
	plate := media.items["plate"]
	plate.Width, plate.Height = 5000, 4000 // greater than worker pixel budget.
	media.items["plate"] = plate
	active := media.items["active"]
	active.Width, active.Height = plate.Width, plate.Height
	media.items["active"] = active
	revision, err := s.Validate(context.Background(), projectID, validSnapshot())
	require.NoError(t, err)
	require.False(t, revision.RenderEligible)
	foundUnsupported := false
	for _, issue := range revision.Issues {
		if issue.Code == shotsv1.ShotIssueCode_SHOT_ISSUE_CODE_UNSUPPORTED_SOURCE_MEDIA {
			foundUnsupported = true
		}
	}
	require.True(t, foundUnsupported)

	plate.Width, plate.Height, plate.SizeBytes = 1200, 800, 25<<20
	media.items["plate"] = plate
	active = media.items["active"]
	active.SizeBytes = 25 << 20
	active.Width, active.Height = 1200, 800
	media.items["active"] = active
	screen := media.items["screen"]
	screen.SizeBytes = 25 << 20
	media.items["screen"] = screen
	revision, err = s.Validate(context.Background(), projectID, validSnapshot())
	require.NoError(t, err)
	require.False(t, revision.RenderEligible)
	found := false
	for _, issue := range revision.Issues {
		if issue.Code == shotsv1.ShotIssueCode_SHOT_ISSUE_CODE_UNSUPPORTED_SOURCE_MEDIA && issue.FieldPath == "sources" {
			found = true
		}
	}
	require.True(t, found)
}

func TestVideoAudioPolicyIsExplicitAndBlocksUnsupportedRender(t *testing.T) { // [REQ:DS-SCREEN-AUDIO]
	s, projectID, media := setup(t)
	video := acceptedVideoSource(projectID)
	media.items[video.ID] = video
	var silentHashes []string
	for _, policy := range []shotsv1.VideoAudioPolicy{shotsv1.VideoAudioPolicy_VIDEO_AUDIO_POLICY_UNSPECIFIED, shotsv1.VideoAudioPolicy_VIDEO_AUDIO_POLICY_SILENT, shotsv1.VideoAudioPolicy_VIDEO_AUDIO_POLICY_PASSTHROUGH} {
		snapshot := validSnapshot()
		snapshot.Presentation.SourceMediaId = video.ID
		snapshot.VideoTiming = acceptedVideoTiming(video)
		snapshot.VideoTiming.AudioPolicy = policy
		shot, revision, err := s.Create(context.Background(), projectID, "Audio policy", snapshot)
		require.NoError(t, err)
		require.Equal(t, policy, revision.Snapshot.GetVideoTiming().GetAudioPolicy(), "save must not silently rewrite policy")
		compiled, err := s.Compile(context.Background(), shot.ID, revision.Revision)
		require.NoError(t, err)
		require.Equal(t, revision.RecipeSHA256, compiled.RecipeSHA256)
		require.Equal(t, policy, compiled.Snapshot.GetVideoTiming().GetAudioPolicy())
		_, resolveErr := s.ResolveForRender(context.Background(), shot.ID, revision.Revision)
		if policy == shotsv1.VideoAudioPolicy_VIDEO_AUDIO_POLICY_PASSTHROUGH {
			require.False(t, revision.RenderEligible)
			var blocked shots.ErrRenderBlocked
			require.ErrorAs(t, resolveErr, &blocked)
			require.Len(t, blocked.Issues, 1)
			require.Equal(t, shotsv1.ShotIssueCode_SHOT_ISSUE_CODE_UNSUPPORTED_AUDIO_POLICY, blocked.Issues[0].Code)
			require.Equal(t, "video_timing.audio_policy", blocked.Issues[0].FieldPath)
			require.Contains(t, blocked.Issues[0].Repair, "silent")
		} else {
			require.True(t, revision.RenderEligible, "%+v", revision.Issues)
			require.NoError(t, resolveErr)
			silentHashes = append(silentHashes, revision.RecipeSHA256)
		}
	}
	require.NotEqual(t, silentHashes[0], silentHashes[1], "explicit silent remains pinned, absent policy stays absent")
	for _, presentation := range []*shotsv1.ScreenPresentation{nil, validSnapshot().Presentation} {
		snapshot := &shotsv1.ShotSnapshot{Presentation: presentation, VideoTiming: &shotsv1.VideoTiming{AudioPolicy: 99}}
		_, _, err := s.Create(context.Background(), projectID, "Unknown audio", snapshot)
		var invalid shots.ErrInvalidShot
		require.ErrorAs(t, err, &invalid)
		require.Equal(t, "video_timing.audio_policy", invalid.Field)
	}
}
