package shots_test

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"device-studio/internal/projects"
	"device-studio/internal/rendering"
	"device-studio/internal/shots"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
)

type mediaRepository struct {
	projects.SourceMediaRepository
	items map[string]projects.SourceMedia
}

func (m mediaRepository) GetSourceMedia(_ context.Context, projectID, id string) (projects.SourceMedia, error) {
	item, ok := m.items[id]
	if !ok || item.ProjectID != projectID {
		return projects.SourceMedia{}, projects.ErrSourceMediaNotFound{ID: id}
	}
	return item, nil
}

func setup(t *testing.T) (shots.Service, string, mediaRepository) {
	t.Helper()
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db,
		apidb.SchemaProviderFunc(projects.Schema), apidb.SchemaProviderFunc(shots.Schema)))
	projectRepo := projects.NewSQLiteRepository(db, nil)
	projectService := projects.NewService(projectRepo, nil)
	project, err := projectService.Create(ctx, projects.CreateInput{Name: "Test"})
	require.NoError(t, err)
	media := mediaRepository{items: map[string]projects.SourceMedia{
		"plate":  {ID: "plate", ProjectID: project.ID, SHA256: "plate-sha", SizeBytes: 500, MediaType: "image/png", Width: 1200, Height: 800, ColorModel: "rgba", ColorProfileStatus: "srgb"},
		"active": {ID: "active", ProjectID: project.ID, SHA256: "active-sha", SizeBytes: 200, MediaType: "image/png", Width: 1200, Height: 800, ColorModel: "grayscale", ColorProfileStatus: "srgb"},
		"screen": {ID: "screen", ProjectID: project.ID, SHA256: "screen-sha", SizeBytes: 350, MediaType: "image/png", Width: 400, Height: 800, ColorModel: "rgba", ColorProfileStatus: "srgb"},
	}}
	repo := shots.NewSQLiteRepository(db)
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	service := shots.NewService(repo, projectService, media, func() time.Time { return now }, nil)
	return service, project.ID, media
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

func TestCreateSaveConflictAndImmutableRevision(t *testing.T) {
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

func TestDraftUnsupportedAndMalformedGeometryPolicies(t *testing.T) {
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
	item.SHA256 = "changed-source-sha"
	media.items["screen"] = item
	second, err := s.Validate(context.Background(), projectID, snapshot)
	require.NoError(t, err)
	require.NotEqual(t, first.PresentationSHA256, second.PresentationSHA256)
}

func TestSemanticDigestsBindContentRatherThanMutableMediaIds(t *testing.T) {
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

func TestChromePayloadMustMatchPresentationAndCapture(t *testing.T) {
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
			p.PhoneSystemChrome = &shotsv1.PhoneSystemChrome{BatteryPercent: 101}
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

func TestPartialQuadRejectsNonFiniteButPersistsIncompleteGeometry(t *testing.T) {
	s, projectID, _ := setup(t)
	snapshot := &shotsv1.ShotSnapshot{Scene: &shotsv1.PhotographicScene{DisplayWidthPixels: 400, DisplayHeightPixels: 800,
		CoordinateConvention: shotsv1.CoordinateConvention_COORDINATE_CONVENTION_PLATE_RASTER_EDGE_PIXELS_V1,
		PlateQuad:            &shotsv1.PlateQuad{TopLeft: &shotsv1.ShotPoint{X: math.NaN(), Y: 5}}}}
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
