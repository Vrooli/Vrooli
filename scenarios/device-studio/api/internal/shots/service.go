package shots

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"

	"device-studio/internal/composition"
	"device-studio/internal/projects"
	"github.com/google/uuid"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	"google.golang.org/protobuf/proto"
)

const (
	defaultPageSize = 50
	maximumPageSize = 100
)

type Service interface {
	Create(context.Context, string, string, *shotsv1.ShotSnapshot) (Shot, Revision, error)
	Save(context.Context, string, int64, string, *shotsv1.ShotSnapshot) (Shot, Revision, error)
	Get(context.Context, string) (Shot, Revision, error)
	GetRevision(context.Context, string, int64) (Revision, error)
	List(context.Context, string, int32, string) ([]Shot, string, error)
	ListRevisions(context.Context, string, int32, string) ([]Revision, string, error)
	Compile(context.Context, string, int64) (Revision, error)
	Validate(context.Context, string, *shotsv1.ShotSnapshot) (Revision, error)
	ResolveForRender(context.Context, string, int64) (ResolvedShot, error)
}

type service struct {
	repo     Repository
	projects projects.Service
	media    projects.SourceMediaRepository
	now      func() time.Time
	newID    func() string
}

func NewService(repo Repository, projectReader projects.Service, mediaReader projects.SourceMediaRepository, now func() time.Time, newID func() string) Service {
	if now == nil {
		now = time.Now
	}
	if newID == nil {
		newID = uuid.NewString
	}
	return &service{repo: repo, projects: projectReader, media: mediaReader, now: now, newID: newID}
}

func (s *service) Create(ctx context.Context, projectID, name string, snapshot *shotsv1.ShotSnapshot) (Shot, Revision, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return Shot{}, Revision{}, ErrInvalidShot{Field: "project_id", Reason: "required"}
	}
	name, err := validateName(name)
	if err != nil {
		return Shot{}, Revision{}, err
	}
	if _, err := s.projects.Get(ctx, projectID); err != nil {
		return Shot{}, Revision{}, err
	}
	now := s.now().UTC()
	shot := Shot{ID: s.newID(), ProjectID: projectID, Name: name, CreatedAt: now, UpdatedAt: now, HeadRevision: 1}
	revision, err := s.buildRevision(ctx, projectID, shot.ID, 1, name, snapshot, now)
	if err != nil {
		return Shot{}, Revision{}, err
	}
	if err := s.repo.Create(ctx, shot, revision); err != nil {
		return Shot{}, Revision{}, err
	}
	return shot, revision, nil
}

func (s *service) Save(ctx context.Context, id string, expected int64, name string, snapshot *shotsv1.ShotSnapshot) (Shot, Revision, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Shot{}, Revision{}, ErrInvalidShot{Field: "shot_id", Reason: "required"}
	}
	if expected <= 0 {
		return Shot{}, Revision{}, ErrInvalidShot{Field: "expected_revision", Reason: "must be positive"}
	}
	name, err := validateName(name)
	if err != nil {
		return Shot{}, Revision{}, err
	}
	shot, err := s.repo.Get(ctx, id)
	if err != nil {
		return Shot{}, Revision{}, err
	}
	if _, err := s.projects.Get(ctx, shot.ProjectID); err != nil {
		return Shot{}, Revision{}, err
	}
	now := s.now().UTC()
	revision, err := s.buildRevision(ctx, shot.ProjectID, id, expected+1, name, snapshot, now)
	if err != nil {
		return Shot{}, Revision{}, err
	}
	updated, err := s.repo.Save(ctx, id, expected, revision)
	if err != nil {
		return Shot{}, Revision{}, err
	}
	return updated, revision, nil
}

func (s *service) Get(ctx context.Context, id string) (Shot, Revision, error) {
	shot, err := s.repo.Get(ctx, strings.TrimSpace(id))
	if err != nil {
		return Shot{}, Revision{}, err
	}
	revision, err := s.repo.GetRevision(ctx, shot.ID, shot.HeadRevision)
	if err != nil {
		return Shot{}, Revision{}, err
	}
	return shot, revision, nil
}

func (s *service) GetRevision(ctx context.Context, id string, revision int64) (Revision, error) {
	id = strings.TrimSpace(id)
	if id == "" || revision <= 0 {
		return Revision{}, ErrInvalidShot{Field: "revision", Reason: "shot id and positive revision are required"}
	}
	return s.repo.GetRevision(ctx, id, revision)
}

func (s *service) List(ctx context.Context, projectID string, pageSize int32, pageToken string) ([]Shot, string, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, "", ErrInvalidShot{Field: "project_id", Reason: "required"}
	}
	if _, err := s.projects.Get(ctx, projectID); err != nil {
		return nil, "", err
	}
	limit := normalizePageSize(pageSize)
	var after *Shot
	if pageToken != "" {
		cursor, err := decodeShotCursor(pageToken)
		if err != nil {
			return nil, "", ErrInvalidShot{Field: "page_token", Reason: "invalid"}
		}
		after = &Shot{ID: cursor.ID, CreatedAt: cursor.CreatedAt}
	}
	rows, err := s.repo.List(ctx, projectID, limit+1, after)
	if err != nil {
		return nil, "", err
	}
	next := ""
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[len(rows)-1]
		next = encodeShotCursor(shotCursor{ID: last.ID, CreatedAt: last.CreatedAt})
	}
	return rows, next, nil
}

func (s *service) ListRevisions(ctx context.Context, id string, pageSize int32, pageToken string) ([]Revision, string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, "", ErrInvalidShot{Field: "shot_id", Reason: "required"}
	}
	if _, err := s.repo.Get(ctx, id); err != nil {
		return nil, "", err
	}
	limit := normalizePageSize(pageSize)
	after := int64(0)
	if pageToken != "" {
		parsed, err := strconvParseRevisionToken(pageToken)
		if err != nil {
			return nil, "", ErrInvalidShot{Field: "page_token", Reason: "invalid"}
		}
		after = parsed
	}
	rows, err := s.repo.ListRevisions(ctx, id, limit+1, after)
	if err != nil {
		return nil, "", err
	}
	next := ""
	if len(rows) > limit {
		rows = rows[:limit]
		next = encodeRevisionToken(rows[len(rows)-1].Revision)
	}
	return rows, next, nil
}

func (s *service) Compile(ctx context.Context, id string, revision int64) (Revision, error) {
	shot, err := s.repo.Get(ctx, strings.TrimSpace(id))
	if err != nil {
		return Revision{}, err
	}
	stored, err := s.repo.GetRevision(ctx, shot.ID, revision)
	if err != nil {
		return Revision{}, err
	}
	current, err := s.buildRevision(ctx, shot.ProjectID, shot.ID, stored.Revision, stored.Name, stored.Snapshot, stored.CreatedAt)
	if err != nil {
		return Revision{}, err
	}
	if current.RecipeSHA256 != stored.RecipeSHA256 || current.SceneSHA256 != stored.SceneSHA256 || current.PresentationSHA256 != stored.PresentationSHA256 {
		return Revision{}, errors.New("stored shot revision failed canonical integrity verification")
	}
	return current, nil
}

func (s *service) Validate(ctx context.Context, projectID string, snapshot *shotsv1.ShotSnapshot) (Revision, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return Revision{}, ErrInvalidShot{Field: "project_id", Reason: "required"}
	}
	if _, err := s.projects.Get(ctx, projectID); err != nil {
		return Revision{}, err
	}
	return s.buildRevision(ctx, projectID, "", 0, "", snapshot, s.now().UTC())
}

func (s *service) ResolveForRender(ctx context.Context, id string, revision int64) (ResolvedShot, error) {
	shot, err := s.repo.Get(ctx, strings.TrimSpace(id))
	if err != nil {
		return ResolvedShot{}, err
	}
	compiled, err := s.Compile(ctx, shot.ID, revision)
	if err != nil {
		return ResolvedShot{}, err
	}
	if !compiled.RenderEligible {
		return ResolvedShot{}, ErrRenderBlocked{ShotID: shot.ID, Revision: revision, Issues: compiled.Issues}
	}
	return ResolvedShot{ProjectID: shot.ProjectID, Shot: shot, Revision: compiled}, nil
}

const (
	sceneSemanticVersion        = "device-studio-scene-v1"
	presentationSemanticVersion = "device-studio-presentation-v1"
	maxRasterSide               = composition.StillMaxDimension
	maxRasterPixels             = composition.StillMaxPixels
	maxWorkerImageBytes         = composition.StillMaxImageBytes
	maxWorkerInputBytes         = composition.StillMaxInputBytes
	maxDepictionRunes           = 120
	maxCarrierRunes             = 24
	maxClockRunes               = 8
	maxWindowTitleRunes         = 120
	maxAddressRunes             = 2048
)

func (s *service) buildRevision(ctx context.Context, projectID, shotID string, revision int64, name string, snapshot *shotsv1.ShotSnapshot, createdAt time.Time) (Revision, error) {
	if snapshot == nil {
		snapshot = &shotsv1.ShotSnapshot{}
	}
	canonical := proto.Clone(snapshot).(*shotsv1.ShotSnapshot)
	result := Revision{ShotID: shotID, Revision: revision, CreatedAt: createdAt.UTC(), Name: name, Snapshot: canonical,
		SceneVersion: sceneSemanticVersion, PresentationVersion: presentationSemanticVersion, CompositorProfile: composition.StillProfile}
	addIssue := func(code shotsv1.ShotIssueCode, field, summary, repair string, blocks bool) {
		result.Issues = append(result.Issues, &shotsv1.ShotIssue{Code: code, FieldPath: field, Summary: summary, Repair: repair, BlocksRender: blocks})
	}
	malformed := func(field, reason string) (Revision, error) {
		return Revision{}, ErrInvalidShot{Field: field, Reason: reason}
	}
	resolve := func(field, id string, required bool) (*shotsv1.ResolvedSourceMedia, projects.SourceMedia, error) {
		if strings.TrimSpace(id) == "" {
			if required {
				addIssue(shotsv1.ShotIssueCode_SHOT_ISSUE_CODE_DRAFT_INCOMPLETE, field, "A source image is required.", "Choose source media from this project.", true)
			}
			return nil, projects.SourceMedia{}, nil
		}
		if s.media == nil {
			return nil, projects.SourceMedia{}, ErrSourceMediaUnavailable{}
		}
		media, err := s.media.GetSourceMedia(ctx, projectID, strings.TrimSpace(id))
		if err != nil {
			var missing projects.ErrSourceMediaNotFound
			if errors.As(err, &missing) {
				return nil, projects.SourceMedia{}, ErrInvalidShot{Field: field, Reason: "must reference source media in this project"}
			}
			return nil, projects.SourceMedia{}, err
		}
		resolved := &shotsv1.ResolvedSourceMedia{Id: media.ID, Sha256: media.SHA256, SizeBytes: media.SizeBytes, MediaType: media.MediaType, WidthPixels: media.Width, HeightPixels: media.Height}
		pixels := int64(media.Width) * int64(media.Height)
		if media.Width == 0 || media.Height == 0 || media.Width > maxRasterSide || media.Height > maxRasterSide || pixels <= 0 || pixels > maxRasterPixels || media.SizeBytes <= 0 || media.SizeBytes > maxWorkerImageBytes {
			addIssue(shotsv1.ShotIssueCode_SHOT_ISSUE_CODE_UNSUPPORTED_SOURCE_MEDIA, field, "This image exceeds the still renderer's bounded input profile.", "Choose an image within the supported dimensions and byte limits.", true)
		}
		if media.ColorProfileStatus != "srgb" && media.ColorProfileStatus != "unspecified" {
			addIssue(shotsv1.ShotIssueCode_SHOT_ISSUE_CODE_UNSUPPORTED_PROFILE, field, "The source color profile is not supported by this compositor.", "Use an unprofiled or sRGB source image.", true)
		}
		return resolved, media, nil
	}
	var plateMedia, activeMedia, hardwareMedia projects.SourceMedia
	if scene := canonical.GetScene(); scene == nil {
		addIssue(shotsv1.ShotIssueCode_SHOT_ISSUE_CODE_DRAFT_INCOMPLETE, "scene", "A photographic scene is required.", "Add a plate, display size, and screen geometry.", true)
	} else {
		var err error
		if scene.GetDepictionLabel() != "" && !validBoundedText(scene.GetDepictionLabel(), maxDepictionRunes) {
			return malformed("scene.depiction_label", "must be bounded text without control characters")
		}
		if scene.GetCoordinateConvention() < shotsv1.CoordinateConvention_COORDINATE_CONVENTION_UNSPECIFIED || scene.GetCoordinateConvention() > shotsv1.CoordinateConvention_COORDINATE_CONVENTION_PLATE_RASTER_EDGE_PIXELS_V1 {
			return malformed("scene.coordinate_convention", "unknown coordinate convention")
		}
		result.Plate, plateMedia, err = resolve("scene.plate_media_id", scene.GetPlateMediaId(), true)
		if err != nil {
			return Revision{}, err
		}
		result.ActiveMask, activeMedia, err = resolve("scene.active_mask_media_id", scene.GetActiveMaskMediaId(), true)
		if err != nil {
			return Revision{}, err
		}
		result.HardwareMask, hardwareMedia, err = resolve("scene.hardware_mask_media_id", scene.GetHardwareMaskMediaId(), false)
		if err != nil {
			return Revision{}, err
		}
		if scene.GetCoordinateConvention() != shotsv1.CoordinateConvention_COORDINATE_CONVENTION_PLATE_RASTER_EDGE_PIXELS_V1 && (scene.GetPlateQuad() != nil || scene.GetDisplayWidthPixels() != 0 || scene.GetDisplayHeightPixels() != 0) {
			return malformed("scene.coordinate_convention", "must use plate raster edge pixels v1")
		}
		if (scene.GetDisplayWidthPixels() == 0) != (scene.GetDisplayHeightPixels() == 0) {
			return malformed("scene.display_size", "width and height must both be supplied")
		}
		if scene.GetDisplayWidthPixels() > maxRasterSide || scene.GetDisplayHeightPixels() > maxRasterSide {
			return malformed("scene.display_size", "dimensions exceed supported limits")
		}
		if scene.GetDisplayWidthPixels() > 0 && int64(scene.GetDisplayWidthPixels())*int64(scene.GetDisplayHeightPixels()) > maxRasterPixels {
			return malformed("scene.display_size", "pixel area exceeds supported limits")
		}
		if result.ActiveMask != nil && activeMedia.ColorModel != "grayscale" && activeMedia.ColorModel != "grayscale-alpha" && activeMedia.ColorModel != "grayscale-alpha-16" {
			return malformed("scene.active_mask_media_id", "active mask must be grayscale or alpha coverage")
		}
		if result.HardwareMask != nil && hardwareMedia.ColorModel != "grayscale" && hardwareMedia.ColorModel != "grayscale-alpha" && hardwareMedia.ColorModel != "grayscale-alpha-16" {
			return malformed("scene.hardware_mask_media_id", "hardware mask must be grayscale or alpha coverage")
		}
		if result.Plate != nil && result.ActiveMask != nil && (plateMedia.Width != activeMedia.Width || plateMedia.Height != activeMedia.Height) {
			return malformed("scene.active_mask_media_id", "mask dimensions must match the plate")
		}
		if result.Plate != nil && result.HardwareMask != nil && (plateMedia.Width != hardwareMedia.Width || plateMedia.Height != hardwareMedia.Height) {
			return malformed("scene.hardware_mask_media_id", "mask dimensions must match the plate")
		}
		if q := scene.GetPlateQuad(); q != nil {
			corners, complete := quadPoints(q)
			provided := [4]*shotsv1.ShotPoint{q.GetTopLeft(), q.GetTopRight(), q.GetBottomRight(), q.GetBottomLeft()}
			for index, point := range corners {
				if math.IsNaN(point.X) || math.IsNaN(point.Y) || math.IsInf(point.X, 0) || math.IsInf(point.Y, 0) {
					return malformed("scene.plate_quad", "coordinates must be finite")
				}
				if provided[index] != nil && plateMedia.Width > 0 && plateMedia.Height > 0 && (point.X < 0 || point.Y < 0 || point.X > float64(plateMedia.Width) || point.Y > float64(plateMedia.Height)) {
					return malformed("scene.plate_quad", "provided corners must lie within the plate raster")
				}
			}
			if complete && scene.GetDisplayWidthPixels() > 0 && scene.GetDisplayHeightPixels() > 0 && result.Plate != nil {
				if plateMedia.Width <= maxRasterSide && plateMedia.Height <= maxRasterSide && int64(plateMedia.Width)*int64(plateMedia.Height) <= maxRasterPixels {
					if err := composition.ValidatePlateQuad(int(plateMedia.Width), int(plateMedia.Height), int(scene.GetDisplayWidthPixels()), int(scene.GetDisplayHeightPixels()), corners); err != nil {
						return malformed("scene.plate_quad", "geometry is invalid or unsafe")
					}
				}
			} else {
				addIssue(shotsv1.ShotIssueCode_SHOT_ISSUE_CODE_DRAFT_INCOMPLETE, "scene.plate_quad", "Screen geometry is incomplete.", "Choose a plate, set display pixels, and define all four corners.", true)
			}
		} else if result.Plate != nil || scene.GetDisplayWidthPixels() != 0 {
			addIssue(shotsv1.ShotIssueCode_SHOT_ISSUE_CODE_DRAFT_INCOMPLETE, "scene.plate_quad", "Screen geometry is incomplete.", "Set display pixels and all four plate-raster corners.", true)
		}
	}
	if presentation := canonical.GetPresentation(); presentation == nil {
		addIssue(shotsv1.ShotIssueCode_SHOT_ISSUE_CODE_DRAFT_INCOMPLETE, "presentation", "A screen presentation is required.", "Choose a source image and capture area.", true)
	} else {
		var err error
		if presentation.GetCaptureArea() < shotsv1.CaptureArea_CAPTURE_AREA_UNSPECIFIED || presentation.GetCaptureArea() > shotsv1.CaptureArea_CAPTURE_AREA_ALREADY_COMPOSED || presentation.GetKind() < shotsv1.PresentationKind_PRESENTATION_KIND_UNSPECIFIED || presentation.GetKind() > shotsv1.PresentationKind_PRESENTATION_KIND_BROWSER_DESKTOP || presentation.GetFitMode() < shotsv1.ScreenFitMode_SCREEN_FIT_MODE_UNSPECIFIED || presentation.GetFitMode() > shotsv1.ScreenFitMode_SCREEN_FIT_MODE_COVER {
			return malformed("presentation", "contains an unknown enum value")
		}
		result.Screen, _, err = resolve("presentation.source_media_id", presentation.GetSourceMediaId(), true)
		if err != nil {
			return Revision{}, err
		}
		if presentation.GetViewportWidthPixels() != 0 || presentation.GetViewportHeightPixels() != 0 {
			return malformed("presentation.viewport", "render viewport is server-derived; caller dimensions are not accepted")
		}
		if chrome := presentation.GetPhoneSystemChrome(); chrome != nil {
			if !validBoundedText(chrome.GetCarrierText(), maxCarrierRunes) || !validBoundedText(chrome.GetClockText(), maxClockRunes) || chrome.GetBatteryPercent() > 100 {
				return malformed("presentation.phone_system_chrome", "text must be bounded and battery percent must be between 0 and 100")
			}
			if presentation.GetKind() != shotsv1.PresentationKind_PRESENTATION_KIND_PHONE_SYSTEM || presentation.GetCaptureArea() == shotsv1.CaptureArea_CAPTURE_AREA_ALREADY_COMPOSED {
				return malformed("presentation.phone_system_chrome", "phone chrome is incompatible with the selected capture and presentation")
			}
		}
		if chrome := presentation.GetBrowserDesktopChrome(); chrome != nil {
			if !validBoundedText(chrome.GetWindowTitle(), maxWindowTitleRunes) || !validBoundedText(chrome.GetAddressText(), maxAddressRunes) {
				return malformed("presentation.browser_desktop_chrome", "text must be bounded and contain no control characters")
			}
			if presentation.GetKind() != shotsv1.PresentationKind_PRESENTATION_KIND_BROWSER_DESKTOP || presentation.GetCaptureArea() == shotsv1.CaptureArea_CAPTURE_AREA_ALREADY_COMPOSED {
				return malformed("presentation.browser_desktop_chrome", "browser chrome is incompatible with the selected capture and presentation")
			}
		}
		if presentation.GetCaptureArea() == shotsv1.CaptureArea_CAPTURE_AREA_UNSPECIFIED || presentation.GetKind() == shotsv1.PresentationKind_PRESENTATION_KIND_UNSPECIFIED {
			addIssue(shotsv1.ShotIssueCode_SHOT_ISSUE_CODE_DRAFT_INCOMPLETE, "presentation", "Capture area and presentation kind are required.", "Choose a supported capture area and presentation.", true)
		}
		if presentation.GetFitMode() != shotsv1.ScreenFitMode_SCREEN_FIT_MODE_UNSPECIFIED && presentation.GetFitMode() != shotsv1.ScreenFitMode_SCREEN_FIT_MODE_CONTAIN {
			addIssue(shotsv1.ShotIssueCode_SHOT_ISSUE_CODE_UNSUPPORTED_FIT_MODE, "presentation.fit_mode", "This fit mode is not supported by the current compositor.", "Choose contain fit.", true)
		}
		if presentation.GetFitMode() == shotsv1.ScreenFitMode_SCREEN_FIT_MODE_UNSPECIFIED {
			presentation.FitMode = shotsv1.ScreenFitMode_SCREEN_FIT_MODE_CONTAIN
		}
		if presentation.GetCaptureArea() == shotsv1.CaptureArea_CAPTURE_AREA_ALREADY_COMPOSED && presentation.GetKind() != shotsv1.PresentationKind_PRESENTATION_KIND_PLAIN {
			addIssue(shotsv1.ShotIssueCode_SHOT_ISSUE_CODE_UNSUPPORTED_PRESENTATION, "presentation.kind", "Already-composed captures cannot receive additional simulated chrome.", "Use plain presentation for an already-composed image.", true)
		}
		if presentation.GetCaptureArea() == shotsv1.CaptureArea_CAPTURE_AREA_ALREADY_COMPOSED && (presentation.GetPhoneSystemChrome() != nil || presentation.GetBrowserDesktopChrome() != nil) {
			return malformed("presentation.chrome", "already-composed captures cannot include simulated chrome payloads")
		}
		if presentation.GetKind() == shotsv1.PresentationKind_PRESENTATION_KIND_PHONE_SYSTEM && presentation.GetPhoneSystemChrome() != nil || presentation.GetKind() == shotsv1.PresentationKind_PRESENTATION_KIND_BROWSER_DESKTOP && presentation.GetBrowserDesktopChrome() != nil {
			addIssue(shotsv1.ShotIssueCode_SHOT_ISSUE_CODE_UNSUPPORTED_PRESENTATION, "presentation.kind", "Simulated chrome is not implemented by the still compositor.", "Use plain presentation until chrome rendering is available.", true)
		}
		if presentation.GetKind() != shotsv1.PresentationKind_PRESENTATION_KIND_PLAIN && presentation.GetKind() != shotsv1.PresentationKind_PRESENTATION_KIND_UNSPECIFIED {
			addIssue(shotsv1.ShotIssueCode_SHOT_ISSUE_CODE_UNSUPPORTED_PRESENTATION, "presentation.kind", "This presentation kind is not implemented by the still compositor.", "Use plain presentation.", true)
		}
	}
	if canonical.GetMotionMode() < shotsv1.MotionMode_MOTION_MODE_UNSPECIFIED || canonical.GetMotionMode() > shotsv1.MotionMode_MOTION_MODE_CROP_PUSH {
		return malformed("motion_mode", "unknown motion mode")
	} else if canonical.GetMotionMode() == shotsv1.MotionMode_MOTION_MODE_UNSPECIFIED {
		addIssue(shotsv1.ShotIssueCode_SHOT_ISSUE_CODE_DRAFT_INCOMPLETE, "motion_mode", "A motion mode is required.", "Choose fixed motion for a still composition.", true)
	} else if canonical.GetMotionMode() != shotsv1.MotionMode_MOTION_MODE_FIXED {
		addIssue(shotsv1.ShotIssueCode_SHOT_ISSUE_CODE_UNSUPPORTED_MOTION, "motion_mode", "This motion mode is not implemented by the still compositor.", "Choose fixed motion.", true)
	}
	totalInputBytes := int64(0)
	for _, source := range []*shotsv1.ResolvedSourceMedia{result.Plate, result.Screen, result.ActiveMask, result.HardwareMask} {
		if source != nil {
			totalInputBytes += source.GetSizeBytes()
		}
	}
	if totalInputBytes > maxWorkerInputBytes {
		addIssue(shotsv1.ShotIssueCode_SHOT_ISSUE_CODE_UNSUPPORTED_SOURCE_MEDIA, "sources", "Combined image bytes exceed the still renderer's bounded input profile.", "Choose a smaller set of source images.", true)
	}

	type mediaIdentity struct {
		SHA256        string
		Size          int64
		Width, Height uint32
	}
	identity := func(m *shotsv1.ResolvedSourceMedia) any {
		if m == nil {
			return nil
		}
		return mediaIdentity{m.Sha256, m.SizeBytes, m.WidthPixels, m.HeightPixels}
	}
	semanticHash := func(v any) (string, error) {
		data, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		sum := sha256.Sum256(data)
		return hex.EncodeToString(sum[:]), nil
	}
	var sceneSemantics *shotsv1.PhotographicScene
	if canonical.GetScene() != nil {
		sceneSemantics = proto.Clone(canonical.GetScene()).(*shotsv1.PhotographicScene)
		sceneSemantics.PlateMediaId = ""
		sceneSemantics.ActiveMaskMediaId = ""
		sceneSemantics.HardwareMaskMediaId = ""
	}
	var presentationSemantics *shotsv1.ScreenPresentation
	if canonical.GetPresentation() != nil {
		presentationSemantics = proto.Clone(canonical.GetPresentation()).(*shotsv1.ScreenPresentation)
		presentationSemantics.SourceMediaId = ""
	}
	sceneValue := struct {
		Version                 string
		Scene                   *shotsv1.PhotographicScene
		Plate, Active, Hardware any
	}{sceneSemanticVersion, sceneSemantics, identity(result.Plate), identity(result.ActiveMask), identity(result.HardwareMask)}
	presentationValue := struct {
		Version      string
		Presentation *shotsv1.ScreenPresentation
		Screen       any
	}{presentationSemanticVersion, presentationSemantics, identity(result.Screen)}
	var err error
	if result.SceneSHA256, err = semanticHash(sceneValue); err != nil {
		return Revision{}, err
	}
	if result.PresentationSHA256, err = semanticHash(presentationValue); err != nil {
		return Revision{}, err
	}
	result.RenderEligible = len(result.Issues) == 0
	recipe := struct {
		Version, Name, SceneSHA256, PresentationSHA256 string
		Snapshot                                       *shotsv1.ShotSnapshot
	}{"device-studio-shot-recipe-v1", name, result.SceneSHA256, result.PresentationSHA256, canonical}
	if result.RecipeSHA256, err = semanticHash(recipe); err != nil {
		return Revision{}, err
	}
	return result, nil
}

type shotCursor struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

func encodeShotCursor(cursor shotCursor) string {
	data, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(data)
}

func decodeShotCursor(token string) (shotCursor, error) {
	if len(token) > 512 {
		return shotCursor{}, errors.New("invalid cursor")
	}
	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return shotCursor{}, err
	}
	var cursor shotCursor
	if err := json.Unmarshal(data, &cursor); err != nil || cursor.ID == "" || cursor.CreatedAt.IsZero() {
		return shotCursor{}, errors.New("invalid cursor")
	}
	return cursor, nil
}

func encodeRevisionToken(revision int64) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconvFormatRevision(revision)))
}

func strconvParseRevisionToken(token string) (int64, error) {
	if len(token) > 128 {
		return 0, ErrInvalidShot{Field: "page_token", Reason: "invalid"}
	}
	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return 0, err
	}
	revision, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil || revision <= 0 {
		return 0, ErrInvalidShot{Field: "page_token", Reason: "invalid"}
	}
	return revision, nil
}

func strconvFormatRevision(revision int64) string { return fmt.Sprintf("%d", revision) }

func normalizePageSize(size int32) int {
	if size <= 0 {
		return defaultPageSize
	}
	if size > maximumPageSize {
		return maximumPageSize
	}
	return int(size)
}

func validateName(value string) (string, error) {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) == 0 || len(runes) > 120 {
		return "", ErrInvalidShot{Field: "name", Reason: "must contain between 1 and 120 characters"}
	}
	for _, r := range runes {
		if unicode.IsControl(r) {
			return "", ErrInvalidShot{Field: "name", Reason: "control characters are not allowed"}
		}
	}
	return value, nil
}

func validBoundedText(value string, maxRunes int) bool {
	if len([]rune(value)) > maxRunes {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func quadPoints(q *shotsv1.PlateQuad) ([4]composition.Point, bool) {
	points := [4]*shotsv1.ShotPoint{q.GetTopLeft(), q.GetTopRight(), q.GetBottomRight(), q.GetBottomLeft()}
	var out [4]composition.Point
	complete := true
	for i, point := range points {
		if point == nil {
			complete = false
			continue
		}
		out[i] = composition.Point{X: point.GetX(), Y: point.GetY()}
	}
	return out, complete
}
