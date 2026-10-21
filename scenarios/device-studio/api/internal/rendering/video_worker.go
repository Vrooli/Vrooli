package rendering

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"math"
	"math/big"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"device-studio/internal/composition"
	"device-studio/internal/presentations"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	VideoManifestName = "video-manifest.json"
	VideoReceiptName  = "video-receipt.json"
	VideoFailureName  = "video-worker-error.json"
	VideoProfileName  = "resolved-tool-profile.json"
	VideoSampleName   = "sample.png"
	VideoOutputName   = "output.mp4"
	// AcceptedInspectionManifestName stages the immutable manifest from the
	// original inspection attempt without colliding with this worker's own
	// inspection-manifest.json entrypoint name.
	AcceptedInspectionManifestName = "accepted-inspection-manifest.json"
	VideoProfileID                 = "device-studio-fixed-plate-video-v1"
	VideoScreenEntryProfileID      = "device-studio-screen-entry-video-v1"
	VideoCompositorID              = "go-fixed-plate-video-v1"
	VideoColorPolicy               = "sdr-unmarked-default709-to-srgb-v1"
	videoManifestVersion           = 1
	videoProtocolVersion           = 1
	videoMaxManifestBytes          = 128 << 10
	videoMaxReceiptBytes           = 128 << 10
	videoMaxOutputBytes            = 160 << 20
	videoMaxSeconds                = 10
	videoScreenEntryMaxSeconds     = 12
	videoMaxFrames                 = 360
	videoMaxPixels                 = 1920 * 1080
	videoMaxInputBytes             = 512 << 20
	videoMaxWorkerBytes            = 256 << 20
	videoMaxClockBytes             = inspectionClockHeader + inspectionMaxFrames*inspectionClockRecord
	videoMaxStderr                 = 64 << 10
	videoMaxStats                  = 4 << 20
	videoMaxWall                   = 300 * time.Second
)

var videoLetterboxMatte = color.NRGBA{R: 32, G: 34, B: 38, A: 255}

type VideoMode string

const (
	VideoModeSample VideoMode = "sample_png"
	VideoModeFinal  VideoMode = "final_mp4"
)

// VideoBinding names one immutable, regular file in the attempt workspace.
// IDs and storage keys stay in the caller's private closure, never in the worker.
type VideoBinding struct {
	Name      string `json:"name"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
}

type VideoScene struct {
	Plate         VideoBinding  `json:"plate"`
	ActiveMask    VideoBinding  `json:"active_mask"`
	HardwareMask  *VideoBinding `json:"hardware_mask,omitempty"`
	PlateWidth    int           `json:"plate_width,omitempty"`
	PlateHeight   int           `json:"plate_height,omitempty"`
	SurfaceWidth  int           `json:"surface_width"`
	SurfaceHeight int           `json:"surface_height"`
	Quad          [4]Corner     `json:"quad"`
	Samples       int           `json:"samples_per_axis"`
}

type VideoInspection struct {
	Source                 VideoBinding         `json:"source"`
	Manifest               VideoBinding         `json:"manifest"`
	Receipt                VideoBinding         `json:"inspection_receipt"`
	Clock                  VideoBinding         `json:"clock"`
	ProfileID              string               `json:"profile_id"`
	ProfileDigest          string               `json:"profile_digest"`
	ProtocolVersion        uint32               `json:"protocol_version"`
	SelectedStream         uint32               `json:"selected_stream"`
	TimeBase               composition.Rational `json:"time_base"`
	StreamStartPTS         int64                `json:"stream_start_pts"`
	SourceEndPTS           int64                `json:"source_end_pts"`
	LastFrameDurationTicks int64                `json:"last_frame_duration_ticks"`
	EOFPolicy              string               `json:"eof_policy"`
	FrameCount             uint32               `json:"frame_count"`
	Width                  int                  `json:"width"`
	Height                 int                  `json:"height"`
	ColorPrimaries         string               `json:"color_primaries"`
	ColorTransfer          string               `json:"color_transfer"`
	ColorMatrix            string               `json:"color_matrix"`
	ColorRange             string               `json:"color_range"`
	ColorPolicy            string               `json:"color_policy"`
}

type VideoSampleBinding struct {
	JobID                    string             `json:"job_id"`
	AttemptID                string             `json:"attempt_id"`
	ReceiptSHA256            string             `json:"receipt_sha256"`
	ManifestSHA256           string             `json:"manifest_sha256"`
	InspectionSHA256         string             `json:"inspection_sha256"`
	RecipeSHA256             string             `json:"recipe_sha256"`
	SourceSHA256             string             `json:"source_sha256"`
	ClockSHA256              string             `json:"clock_sha256"`
	TimingSHA256             string             `json:"timing_sha256"`
	ShotRevision             int64              `json:"shot_revision"`
	BaseShotRevision         int64              `json:"base_shot_revision"`
	BaseRecipeVersion        string             `json:"base_recipe_version"`
	BaseRecipeSHA256         string             `json:"base_recipe_sha256"`
	ResolvedTopFill          presentations.RGB  `json:"resolved_top_fill"`
	ResolvedBottomFill       presentations.RGB  `json:"resolved_bottom_fill"`
	ResolvedStatusInk        *presentations.RGB `json:"resolved_status_ink,omitempty"`
	ResolvedHomeInk          *presentations.RGB `json:"resolved_home_ink,omitempty"`
	TopFillSourceResolved    bool               `json:"top_fill_source_resolved"`
	BottomFillSourceResolved bool               `json:"bottom_fill_source_resolved"`
	StatusInkSourceResolved  bool               `json:"status_ink_source_resolved"`
	HomeInkSourceResolved    bool               `json:"home_ink_source_resolved"`
	FirstFrameRGBSHA256      string             `json:"first_frame_rgb_sha256"`
	PixelProfile             string             `json:"pixel_profile"`
	Width                    int                `json:"width"`
	Height                   int                `json:"height"`
	ReceiptArtifact          VideoBinding       `json:"receipt_artifact"`
}

// VideoManifest is the frozen, path-free protocol consumed by the private
// worker. Tool executable paths are supplied in the profile sidecar.
type VideoManifest struct {
	Version            int                   `json:"version"`
	ProtocolVersion    uint32                `json:"protocol_version"`
	JobID              string                `json:"job_id"`
	AttemptID          string                `json:"attempt_id"`
	InputFingerprint   string                `json:"input_fingerprint"`
	WorkerSHA256       string                `json:"worker_sha256"`
	Mode               VideoMode             `json:"mode"`
	VideoProfile       string                `json:"video_profile"`
	CompositorProfile  string                `json:"compositor_profile"`
	Camera             *Camera               `json:"camera,omitempty"`
	Inspection         VideoInspection       `json:"inspection"`
	TrimStartPTS       int64                 `json:"trim_start_pts"`
	TrimEndPTS         int64                 `json:"trim_end_pts"`
	PlaybackRate       composition.Rational  `json:"playback_rate"`
	OutputRate         composition.Rational  `json:"output_rate"`
	Width              int                   `json:"width"`
	Height             int                   `json:"height"`
	Scene              VideoScene            `json:"scene"`
	Presentation       *presentations.Config `json:"presentation,omitempty"`
	PresentationLayout *presentations.Layout `json:"presentation_layout,omitempty"`
	ShotRevision       int64                 `json:"shot_revision"`
	BaseShotRevision   int64                 `json:"base_shot_revision"`
	BaseRecipeVersion  string                `json:"base_recipe_version"`
	BaseRecipeSHA256   string                `json:"base_recipe_sha256"`
	RecipeSHA256       string                `json:"recipe_sha256"`
	Sample             *VideoSampleBinding   `json:"sample,omitempty"`
}

type VideoNativeFacts struct {
	Container          string               `json:"container"`
	Codec              string               `json:"codec"`
	PixelFormat        string               `json:"pixel_format"`
	Width              int                  `json:"width"`
	Height             int                  `json:"height"`
	FrameRate          composition.Rational `json:"frame_rate"`
	TimeBase           composition.Rational `json:"time_base"`
	FrameCount         uint32               `json:"frame_count"`
	FirstPTS           int64                `json:"first_pts"`
	LastPTS            int64                `json:"last_pts"`
	ColorPrimaries     string               `json:"color_primaries"`
	ColorTransfer      string               `json:"color_transfer"`
	ColorMatrix        string               `json:"color_matrix"`
	ColorRange         string               `json:"color_range"`
	FullDecodeVerified bool                 `json:"full_decode_verified"`
}

type VideoReceipt struct {
	Version                  int                   `json:"version"`
	ProtocolVersion          uint32                `json:"protocol_version"`
	JobID                    string                `json:"job_id"`
	AttemptID                string                `json:"attempt_id"`
	InputFingerprint         string                `json:"input_fingerprint"`
	ShotRevision             int64                 `json:"shot_revision"`
	BaseShotRevision         int64                 `json:"base_shot_revision"`
	BaseRecipeVersion        string                `json:"base_recipe_version"`
	BaseRecipeSHA256         string                `json:"base_recipe_sha256"`
	RecipeSHA256             string                `json:"recipe_sha256"`
	ManifestSHA256           string                `json:"manifest_sha256"`
	InspectionSHA256         string                `json:"inspection_sha256"`
	WorkerSHA256             string                `json:"worker_sha256"`
	Mode                     VideoMode             `json:"mode"`
	Camera                   *Camera               `json:"camera,omitempty"`
	OutputName               string                `json:"output_name"`
	OutputSHA256             string                `json:"output_sha256"`
	OutputSizeBytes          int64                 `json:"output_size_bytes"`
	OutputWidth              int                   `json:"output_width"`
	OutputHeight             int                   `json:"output_height"`
	FirstFrameRGBSHA256      string                `json:"first_frame_rgb_sha256"`
	PixelProfile             string                `json:"pixel_profile"`
	PresentationLayout       *presentations.Layout `json:"presentation_layout,omitempty"`
	ResolvedTopFill          presentations.RGB     `json:"resolved_top_fill"`
	ResolvedBottomFill       presentations.RGB     `json:"resolved_bottom_fill"`
	ResolvedStatusInk        *presentations.RGB    `json:"resolved_status_ink,omitempty"`
	ResolvedHomeInk          *presentations.RGB    `json:"resolved_home_ink,omitempty"`
	TopFillSourceResolved    bool                  `json:"top_fill_source_resolved"`
	BottomFillSourceResolved bool                  `json:"bottom_fill_source_resolved"`
	StatusInkSourceResolved  bool                  `json:"status_ink_source_resolved"`
	HomeInkSourceResolved    bool                  `json:"home_ink_source_resolved"`
	Native                   VideoNativeFacts      `json:"native"`
	CompletedAt              time.Time             `json:"completed_at"`
}

func IsVideoWorkerInvocation(args []string) bool {
	return len(args) > 0 && args[0] == "video-render-worker"
}

func RunVideoWorkerCommand(args []string) error {
	if len(args) != 3 || args[0] != "video-render-worker" || args[1] != "--manifest" {
		return errors.New("video-render-worker requires exactly --manifest <absolute-manifest-path>")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return RunVideoManifest(ctx, args[2])
}

func decodeCanonicalVideoManifest(data []byte) (VideoManifest, error) {
	var m VideoManifest
	if len(data) == 0 || len(data) > videoMaxManifestBytes {
		return m, errors.New("video manifest is empty or exceeds its size limit")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&m); err != nil {
		return m, fmt.Errorf("decode video manifest: %w", err)
	}
	if d.Decode(new(any)) != io.EOF {
		return m, errors.New("video manifest has trailing data")
	}
	b, err := json.Marshal(m)
	if err != nil {
		return m, err
	}
	if !bytes.Equal(data, b) && !bytes.Equal(data, append(b, '\n')) {
		return m, errors.New("video manifest is not canonical JSON")
	}
	return m, nil
}

// ValidateVideoManifest applies the worker's complete fixed-profile validation
// to a fully populated manifest before the coordinator persists or dispatches it.
func ValidateVideoManifest(m VideoManifest) error {
	return validateVideoManifest(m)
}

func validateVideoManifest(m VideoManifest) error {
	if m.Version != videoManifestVersion || m.ProtocolVersion != videoProtocolVersion || (m.Mode != VideoModeSample && m.Mode != VideoModeFinal) {
		return errors.New("video manifest version, protocol, or mode is unsupported")
	}
	if !identifierPattern.MatchString(m.JobID) || !identifierPattern.MatchString(m.AttemptID) || !digestPattern.MatchString(m.InputFingerprint) || !digestPattern.MatchString(m.WorkerSHA256) {
		return errors.New("video manifest identity is invalid")
	}
	if m.CompositorProfile != VideoCompositorID {
		return errors.New("video renderer profile is unsupported")
	}
	if m.Camera == nil {
		if m.VideoProfile != VideoProfileID {
			return errors.New("fixed video profile is unsupported")
		}
	} else if m.VideoProfile != VideoScreenEntryProfileID || m.Camera.ProfileID != ScreenEntryCameraProfileID || m.Camera.ProfileVersion != ScreenEntryCameraProfileVersion || m.Camera.MatteRGB == nil {
		return errors.New("screen-entry camera profile or matte is unsupported")
	}
	i := m.Inspection
	if err := validateVideoInspection(i); err != nil {
		return err
	}
	for _, b := range []VideoBinding{i.Source, i.Manifest, i.Receipt, i.Clock, m.Scene.Plate, m.Scene.ActiveMask} {
		if err := validateVideoBinding(b); err != nil {
			return err
		}
	}
	if m.Scene.HardwareMask != nil {
		if err := validateVideoBinding(*m.Scene.HardwareMask); err != nil {
			return err
		}
	}
	if m.TrimStartPTS < i.StreamStartPTS || m.TrimEndPTS <= m.TrimStartPTS || m.TrimEndPTS > i.SourceEndPTS || m.PlaybackRate.Num <= 0 || m.PlaybackRate.Den <= 0 || m.OutputRate.Den != 1 || (m.OutputRate.Num != 24 && m.OutputRate.Num != 30) {
		return errors.New("video trim or rational rate is unsupported")
	}
	if err := validateVideoDuration(m); err != nil {
		return err
	}
	if (m.Width != 1920 || m.Height != 1080) && (m.Width != 1080 || m.Height != 1920) {
		return errors.New("video output must be 1080p landscape or portrait")
	}
	if int64(m.Width)*int64(m.Height) > videoMaxPixels {
		return errors.New("video output raster exceeds profile limits")
	}
	if m.Scene.SurfaceWidth <= 0 || m.Scene.SurfaceHeight <= 0 || m.Scene.SurfaceWidth > composition.StillMaxDimension || m.Scene.SurfaceHeight > composition.StillMaxDimension {
		return errors.New("video scene dimensions are invalid")
	}
	if m.Scene.Samples < 0 || m.Scene.Samples > 4 {
		return errors.New("video composition sample count is invalid")
	}
	for n, p := range m.Scene.Quad {
		if math.IsNaN(p.X) || math.IsInf(p.X, 0) || math.IsNaN(p.Y) || math.IsInf(p.Y, 0) {
			return fmt.Errorf("video scene quad corner %d is not finite", n)
		}
	}
	if m.Presentation == nil || m.Presentation.Target != image.Pt(m.Scene.SurfaceWidth, m.Scene.SurfaceHeight) {
		return errors.New("video presentation is not pinned to the scene")
	}
	autoSample := m.Mode == VideoModeSample && m.Presentation.Phone != nil && m.Presentation.Phone.ProfileVersion == presentations.PhoneProfileVersionAuto
	resolvedAutoValues := autoSample && hasResolvedPhoneValues(videoResolvedFlags(m.Presentation))
	if (resolvedAutoValues && m.PresentationLayout != nil) || (!resolvedAutoValues && m.PresentationLayout == nil) {
		return errors.New("video presentation layout presence is inconsistent with sample mode")
	}
	if m.PresentationLayout != nil && (m.PresentationLayout.ProfileID == "" || m.PresentationLayout.DisplaySize != m.Presentation.Target || !digestPattern.MatchString(m.PresentationLayout.ConfigSHA256) || !digestPattern.MatchString(m.PresentationLayout.FontSHA256) || m.PresentationLayout.Rasterizer == "" || m.PresentationLayout.FillPolicy == "") {
		return errors.New("video presentation layout identity is incomplete")
	}
	if m.PresentationLayout != nil {
		bounds := image.Rect(0, 0, i.Width, i.Height)
		expected, err := presentations.BuildLayoutForVideoRaster(bounds, *m.Presentation)
		if err != nil {
			return fmt.Errorf("validate pinned video presentation: %w", err)
		}
		if !sameJSON(expected, *m.PresentationLayout) {
			return errors.New("video presentation layout differs from its pinned source raster and config")
		}
	}
	if m.Camera != nil {
		if m.Scene.PlateWidth <= 0 || m.Scene.PlateHeight <= 0 || m.Scene.PlateWidth > composition.StillMaxDimension || m.Scene.PlateHeight > composition.StillMaxDimension || int64(m.Scene.PlateWidth)*int64(m.Scene.PlateHeight) > composition.StillMaxPixels {
			return errors.New("camera plate dimensions are missing or exceed composition limits")
		}
		if _, err := newVideoCameraPath(m, image.Rect(0, 0, m.Scene.PlateWidth, m.Scene.PlateHeight), m.PresentationLayout); err != nil {
			return fmt.Errorf("validate screen-entry camera path: %w", err)
		}
	} else if m.Scene.PlateWidth != 0 || m.Scene.PlateHeight != 0 {
		return errors.New("fixed video manifest must omit camera-only plate dimensions")
	}
	if m.ShotRevision <= 0 || m.BaseShotRevision <= 0 || m.ShotRevision < m.BaseShotRevision || strings.TrimSpace(m.BaseRecipeVersion) == "" || len(m.BaseRecipeVersion) > 64 || !digestPattern.MatchString(m.BaseRecipeSHA256) || !digestPattern.MatchString(m.RecipeSHA256) {
		return errors.New("video shot recipe identity is invalid")
	}
	if m.Mode == VideoModeSample && m.ShotRevision != m.BaseShotRevision {
		return errors.New("sample must bind the exact base shot recipe")
	}
	if m.Mode == VideoModeFinal && m.Sample == nil && m.ShotRevision != m.BaseShotRevision {
		return errors.New("manual final output must bind one immutable shot revision")
	}
	if m.Mode == VideoModeSample && m.Sample != nil {
		return errors.New("sample job cannot depend on another sample")
	}
	if m.Mode == VideoModeFinal && m.Sample == nil && !videoPresentationHasManualFills(m.Presentation) {
		return errors.New("final video requires a verified AUTO sample or explicit presentation fills")
	}
	if m.Sample != nil {
		s := m.Sample
		if !identifierPattern.MatchString(s.JobID) || !identifierPattern.MatchString(s.AttemptID) || s.JobID == m.JobID || !digestPattern.MatchString(s.ReceiptSHA256) || !digestPattern.MatchString(s.ManifestSHA256) || !digestPattern.MatchString(s.InspectionSHA256) || s.InspectionSHA256 != VideoInspectionDigest(m.Inspection) || !digestPattern.MatchString(s.RecipeSHA256) || !digestPattern.MatchString(s.SourceSHA256) || !digestPattern.MatchString(s.ClockSHA256) || !digestPattern.MatchString(s.TimingSHA256) || !digestPattern.MatchString(s.FirstFrameRGBSHA256) || s.ShotRevision <= 0 || s.Width != m.Width || s.Height != m.Height || s.PixelProfile != "srgb-rgb24-v1" || s.BaseRecipeVersion != m.BaseRecipeVersion || s.BaseShotRevision != m.BaseShotRevision || s.BaseRecipeSHA256 != m.BaseRecipeSHA256 {
			return errors.New("video sample binding is invalid")
		}
		if m.Mode == VideoModeFinal {
			resolved := hasResolvedPhoneValues([4]bool{s.TopFillSourceResolved, s.BottomFillSourceResolved, s.StatusInkSourceResolved, s.HomeInkSourceResolved})
			if resolved && m.ShotRevision <= m.BaseShotRevision {
				return errors.New("AUTO sampled final shot must be a later immutable revision")
			}
			if !resolved && (m.ShotRevision != m.BaseShotRevision || m.RecipeSHA256 != s.RecipeSHA256) {
				return errors.New("manual or browser sample must remain bound to the same immutable revision")
			}
		}
		if err := validateVideoBinding(s.ReceiptArtifact); err != nil {
			return err
		}
		if s.ReceiptArtifact.SizeBytes > videoMaxReceiptBytes {
			return errors.New("sample receipt exceeds the fixed receipt limit")
		}
		if m.Mode == VideoModeFinal && m.Presentation.Phone != nil {
			if !sameRGB(m.Presentation.Phone.TopFill, &s.ResolvedTopFill) || !sameRGB(m.Presentation.Phone.BottomFill, &s.ResolvedBottomFill) || !sameRGB(m.Presentation.Phone.StatusInk, s.ResolvedStatusInk) || !sameRGB(m.Presentation.Phone.HomeInk, s.ResolvedHomeInk) {
				return errors.New("final phone presentation differs from accepted sample colors")
			}
		}
	}
	if m.Sample != nil && m.Sample.TimingSHA256 != videoTimingDigest(m) {
		return errors.New("video sample timing identity differs from final closure")
	}
	return nil
}

// validateVideoInspection checks the source facts shared by photographic and
// physical recording renders. Keep this deliberately independent of either
// manifest's naming: the same accepted immutable inspection closure is reused.
func validateVideoInspection(i VideoInspection) error {
	if i.ProfileID != inspectionProfileID || !digestPattern.MatchString(i.ProfileDigest) || i.ProtocolVersion != inspectionProtocol || i.EOFPolicy != "last_frame_duration" || i.ColorPolicy != inspectionColorPolicy {
		return errors.New("video inspection profile or policy is unsupported")
	}
	for _, b := range []VideoBinding{i.Source, i.Manifest, i.Receipt, i.Clock} {
		if err := validateVideoBinding(b); err != nil {
			return err
		}
	}
	if i.Manifest.Name != AcceptedInspectionManifestName || i.Receipt.Name != InspectionReceiptName || i.Clock.Name != InspectionClockName || i.Source.SizeBytes > videoMaxInputBytes || i.Clock.SizeBytes > videoMaxClockBytes {
		return errors.New("video inspection bindings are invalid")
	}
	if i.FrameCount == 0 || i.FrameCount > inspectionMaxFrames || i.Width <= 0 || i.Height <= 0 || i.Width > inspectionMaxDimension || i.Height > inspectionMaxDimension || i.TimeBase.Num <= 0 || i.TimeBase.Den <= 0 || i.SourceEndPTS <= i.StreamStartPTS || i.LastFrameDurationTicks <= 0 || i.SourceEndPTS > math.MaxInt64-i.LastFrameDurationTicks {
		return errors.New("video inspection clock facts are invalid")
	}
	return nil
}

func videoPresentationHasManualFills(c *presentations.Config) bool {
	if c == nil {
		return false
	}
	if c.Phone != nil {
		return c.Phone.TopFill != nil && c.Phone.BottomFill != nil
	}
	if c.Browser != nil {
		return c.Browser.DesktopColor != nil
	}
	return false
}

func sameRGB(a, b *presentations.RGB) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func videoTimingDigest(m VideoManifest) string {
	legacy := struct {
		Start, End       int64
		Playback, Output composition.Rational
		W, H             int
		Profile          string
	}{m.TrimStartPTS, m.TrimEndPTS, m.PlaybackRate, m.OutputRate, m.Width, m.Height, m.VideoProfile}
	var b []byte
	var err error
	if m.Camera == nil {
		b, err = json.Marshal(legacy)
	} else {
		b, err = json.Marshal(struct {
			Version string `json:"version"`
			Timing  any    `json:"timing"`
			Camera  Camera `json:"camera"`
		}{Version: "camera-timing-v1", Timing: legacy, Camera: *m.Camera})
	}
	if err != nil {
		return ""
	}
	return digest(b)
}

func VideoTimingDigest(m VideoManifest) string { return videoTimingDigest(m) }

// VideoInspectionDigest binds every source, receipt, clock, selected-stream,
// timebase, EOF, raster, and color-policy fact used by a video render.
func VideoInspectionDigest(i VideoInspection) string {
	data, err := json.Marshal(i)
	if err != nil {
		return ""
	}
	return digest(data)
}

// PlanVideoSample returns the exact source frame selected for output frame
// zero using the frozen inspection clock and the same schedule as the worker.
func PlanVideoSample(ctx context.Context, m VideoManifest, clockBytes []byte) (uint32, int64, error) {
	if ctx == nil || len(clockBytes) == 0 || len(clockBytes) > videoMaxClockBytes || m.Inspection.FrameCount == 0 || m.Inspection.FrameCount > inspectionMaxFrames {
		return 0, 0, errors.New("video sample planning inputs are outside their bounds")
	}
	if m.TrimStartPTS < m.Inspection.StreamStartPTS || m.TrimEndPTS > m.Inspection.SourceEndPTS {
		return 0, 0, errors.New("video sample trim lies outside the pinned source clock")
	}
	if err := validateVideoDuration(m); err != nil {
		return 0, 0, err
	}
	pts, err := decodeInspectionClock(clockBytes, m.Inspection.FrameCount)
	if err != nil {
		return 0, 0, err
	}
	plan, err := composition.PlanFrames(ctx, composition.FrameClock{
		SourcePTS: pts, SourceTimeBase: m.Inspection.TimeBase, SourceEndPTS: m.Inspection.SourceEndPTS,
		TrimStartPTS: m.TrimStartPTS, TrimEndPTS: m.TrimEndPTS, PlaybackRate: m.PlaybackRate,
		OutputRate: m.OutputRate, Underflow: composition.RangeReject, EOF: composition.RangeReject,
	})
	if err != nil {
		return 0, 0, err
	}
	if len(plan) == 0 || plan[0].SourceIndex < 0 || plan[0].SourceIndex >= len(pts) || plan[0].SourcePTS != pts[plan[0].SourceIndex] {
		return 0, 0, errors.New("video sample schedule did not resolve a source frame")
	}
	return uint32(plan[0].SourceIndex), pts[plan[0].SourceIndex], nil
}

func videoDurationAndFrameCount(m VideoManifest) (*big.Rat, int64, error) {
	if m.TrimEndPTS <= m.TrimStartPTS || m.Inspection.TimeBase.Num <= 0 || m.Inspection.TimeBase.Den <= 0 || m.PlaybackRate.Num <= 0 || m.PlaybackRate.Den <= 0 || m.OutputRate.Num <= 0 || m.OutputRate.Den <= 0 {
		return nil, 0, errors.New("video timing rationals must be positive and trim must be nonempty")
	}
	delta := new(big.Int).Sub(big.NewInt(m.TrimEndPTS), big.NewInt(m.TrimStartPTS))
	duration := new(big.Rat).SetInt(delta)
	duration.Mul(duration, new(big.Rat).SetFrac(big.NewInt(m.Inspection.TimeBase.Num), big.NewInt(m.Inspection.TimeBase.Den)))
	duration.Mul(duration, new(big.Rat).SetFrac(big.NewInt(m.PlaybackRate.Den), big.NewInt(m.PlaybackRate.Num)))
	frames := new(big.Rat).Mul(new(big.Rat).Set(duration), new(big.Rat).SetFrac(big.NewInt(m.OutputRate.Num), big.NewInt(m.OutputRate.Den)))
	quotient, remainder := new(big.Int).QuoRem(frames.Num(), frames.Denom(), new(big.Int))
	if remainder.Sign() > 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	frameCount := quotient
	if !frameCount.IsInt64() {
		return nil, 0, errors.New("video frame count overflows int64")
	}
	return duration, frameCount.Int64(), nil
}

func plannedVideoFrameCount(m VideoManifest) (int64, error) {
	_, count, err := videoDurationAndFrameCount(m)
	if err != nil {
		return 0, err
	}
	if count <= 0 || count > videoMaxFrames {
		return 0, errors.New("video frame schedule exceeds the fixed development profile")
	}
	return count, nil
}

func plannedOutputPTS(m VideoManifest, timeBase composition.Rational, count int) ([]int64, error) {
	if count <= 0 || count > videoMaxFrames || timeBase.Num <= 0 || timeBase.Den <= 0 || m.OutputRate.Num <= 0 || m.OutputRate.Den <= 0 {
		return nil, errors.New("encoded output timestamp rationals are invalid")
	}
	step := new(big.Rat).SetFrac(big.NewInt(m.OutputRate.Den), big.NewInt(m.OutputRate.Num))
	step.Quo(step, new(big.Rat).SetFrac(big.NewInt(timeBase.Num), big.NewInt(timeBase.Den)))
	if !step.IsInt() || !step.Num().IsInt64() || step.Num().Sign() <= 0 {
		return nil, errors.New("output frame cadence is not integral in encoded time base")
	}
	stepTicks := step.Num().Int64()
	pts := make([]int64, count)
	for i := range pts {
		if i > 0 && int64(i) > math.MaxInt64/stepTicks {
			return nil, errors.New("output PTS overflows int64")
		}
		pts[i] = int64(i) * stepTicks
	}
	return pts, nil
}

func validateVideoDuration(m VideoManifest) error {
	duration, count, err := videoDurationAndFrameCount(m)
	if err != nil {
		return err
	}
	maxSeconds := videoMaxSeconds
	if m.Camera != nil {
		maxSeconds = videoScreenEntryMaxSeconds
	}
	if duration.Cmp(big.NewRat(int64(maxSeconds), 1)) > 0 {
		if m.Camera != nil {
			return errors.New("screen-entry video trim exceeds the 12-second profile bound")
		}
		return errors.New("video trim exceeds the fixed ten-second development profile")
	}
	if count <= 0 || count > videoMaxFrames {
		return errors.New("video frame schedule exceeds the fixed development profile")
	}
	return nil
}

func validateVideoBinding(b VideoBinding) error {
	if b.Name == "" || filepath.IsAbs(b.Name) || filepath.Clean(b.Name) != b.Name || strings.ContainsAny(b.Name, "/\\:\x00") || b.SizeBytes <= 0 || b.SizeBytes > videoMaxInputBytes || !digestPattern.MatchString(b.SHA256) {
		return errors.New("video workspace binding is invalid")
	}
	if b.Name == VideoManifestName || b.Name == VideoReceiptName || b.Name == VideoFailureName || b.Name == VideoOutputName || b.Name == VideoSampleName || b.Name == VideoProfileName {
		return errors.New("video workspace binding uses a reserved name")
	}
	return nil
}

// VerifyVideoArtifacts performs only pure-Go verification. Native output
// probing and full MP4 decoding are receipt facts produced inside the owned worker.
func VerifyVideoArtifacts(ctx context.Context, manifestBytes, receiptBytes []byte, output io.Reader, outputSize int64) (VideoReceipt, error) {
	var receipt VideoReceipt
	if ctx == nil || output == nil {
		return receipt, errors.New("video artifact verification needs context and output reader")
	}
	m, err := decodeCanonicalVideoManifest(manifestBytes)
	if err != nil {
		return receipt, err
	}
	if err = validateVideoManifest(m); err != nil {
		return receipt, err
	}
	if len(receiptBytes) == 0 || len(receiptBytes) > videoMaxReceiptBytes {
		return receipt, errors.New("video receipt size is invalid")
	}
	d := json.NewDecoder(bytes.NewReader(receiptBytes))
	d.DisallowUnknownFields()
	if err = d.Decode(&receipt); err != nil {
		return receipt, err
	}
	if d.Decode(new(any)) != io.EOF {
		return receipt, errors.New("video receipt has trailing JSON")
	}
	canonical, e := json.Marshal(receipt)
	if e != nil || !bytes.Equal(canonical, receiptBytes) {
		return receipt, errors.New("video receipt is not canonical JSON")
	}
	if receipt.Version != videoManifestVersion || receipt.ProtocolVersion != videoProtocolVersion || receipt.JobID != m.JobID || receipt.AttemptID != m.AttemptID || receipt.InputFingerprint != m.InputFingerprint || receipt.ManifestSHA256 != digest(manifestBytes) || receipt.InspectionSHA256 != VideoInspectionDigest(m.Inspection) || receipt.WorkerSHA256 != m.WorkerSHA256 || receipt.Mode != m.Mode || !sameJSON(receipt.Camera, m.Camera) {
		return receipt, errors.New("video receipt identity differs from the immutable manifest")
	}
	if outputSize <= 0 || outputSize > videoMaxOutputBytes || receipt.OutputSizeBytes != outputSize || !digestPattern.MatchString(receipt.OutputSHA256) {
		return receipt, errors.New("video output size or digest is invalid")
	}
	var sampleBytes []byte
	var n int64
	var outputHash string
	if m.Mode == VideoModeSample {
		if outputSize > 64<<20 {
			return receipt, errors.New("sample PNG exceeds its strict byte limit")
		}
		sampleBytes, e = readContextBounded(ctx, output, outputSize+1)
		if e != nil {
			return receipt, e
		}
		n = int64(len(sampleBytes))
		outputHash = digest(sampleBytes)
	} else {
		h := sha256.New()
		n, e = hashVerifiedMP4(ctx, h, output, outputSize)
		if e != nil {
			return receipt, e
		}
		outputHash = hex.EncodeToString(h.Sum(nil))
	}
	if n != outputSize || n > videoMaxOutputBytes || outputHash != receipt.OutputSHA256 {
		return receipt, errors.New("video output bytes differ from receipt")
	}
	if m.Mode == VideoModeSample {
		if receipt.OutputName != VideoSampleName || receipt.OutputWidth != m.Width || receipt.OutputHeight != m.Height || receipt.Native.FullDecodeVerified || receipt.Native.FrameCount != 1 || receipt.PresentationLayout == nil {
			return receipt, errors.New("sample receipt facts are invalid")
		}
		wantFlags := videoResolvedFlags(m.Presentation)
		if receipt.TopFillSourceResolved != wantFlags[0] || receipt.BottomFillSourceResolved != wantFlags[1] || receipt.StatusInkSourceResolved != wantFlags[2] || receipt.HomeInkSourceResolved != wantFlags[3] {
			return receipt, errors.New("sample receipt color-source flags differ from the base presentation")
		}
		config, e := png.DecodeConfig(bytes.NewReader(sampleBytes))
		if e != nil || config.Width != m.Width || config.Height != m.Height || int64(config.Width)*int64(config.Height) > videoMaxPixels {
			return receipt, errors.New("sample PNG header dimensions are invalid")
		}
		decoded, e := png.Decode(bytes.NewReader(sampleBytes))
		if e != nil {
			return receipt, fmt.Errorf("decode sample PNG: %w", e)
		}
		if decoded.Bounds().Dx() != m.Width || decoded.Bounds().Dy() != m.Height {
			return receipt, errors.New("sample PNG dimensions differ from manifest")
		}
		rgb, e := imageRGB24(ctx, decoded)
		if e != nil {
			return receipt, e
		}
		if canonicalVideoPixelDigest(m.Width, m.Height, "srgb-rgb24-v1", rgb) != receipt.FirstFrameRGBSHA256 {
			return receipt, errors.New("sample PNG pixels differ from canonical first-frame digest")
		}
		if receipt.PresentationLayout.ResolvedTopFill != receipt.ResolvedTopFill || receipt.PresentationLayout.ResolvedBottomFill != receipt.ResolvedBottomFill || !sameRGB(receipt.PresentationLayout.ResolvedStatusInk, receipt.ResolvedStatusInk) || !sameRGB(receipt.PresentationLayout.ResolvedHomeInk, receipt.ResolvedHomeInk) {
			return receipt, errors.New("sample resolved presentation values differ from its layout")
		}
	} else {
		wantFrames, e := plannedVideoFrameCount(m)
		if e != nil {
			return receipt, e
		}
		wantPTS, ptsErr := plannedOutputPTS(m, receipt.Native.TimeBase, int(wantFrames))
		if ptsErr != nil || receipt.OutputName != VideoOutputName || receipt.OutputWidth != m.Width || receipt.OutputHeight != m.Height || receipt.Native.Width != m.Width || receipt.Native.Height != m.Height || !receipt.Native.FullDecodeVerified || receipt.Native.Codec != "h264" || receipt.Native.Container != "mov,mp4,m4a,3gp,3g2,mj2" || receipt.Native.PixelFormat != "yuv420p" || receipt.Native.FrameRate != m.OutputRate || receipt.Native.FrameCount != uint32(wantFrames) || len(wantPTS) == 0 || receipt.Native.FirstPTS != wantPTS[0] || receipt.Native.LastPTS != wantPTS[len(wantPTS)-1] || receipt.Native.ColorPrimaries != "bt709" || receipt.Native.ColorTransfer != "bt709" || receipt.Native.ColorMatrix != "bt709" || (receipt.Native.ColorRange != "tv" && receipt.Native.ColorRange != "mpeg") || !sameJSON(receipt.PresentationLayout, m.PresentationLayout) {
			return receipt, errors.New("final video receipt facts are invalid")
		}
		var wantFlags [4]bool
		if m.Sample != nil {
			wantFlags = [4]bool{m.Sample.TopFillSourceResolved, m.Sample.BottomFillSourceResolved, m.Sample.StatusInkSourceResolved, m.Sample.HomeInkSourceResolved}
		}
		if receipt.TopFillSourceResolved != wantFlags[0] || receipt.BottomFillSourceResolved != wantFlags[1] || receipt.StatusInkSourceResolved != wantFlags[2] || receipt.HomeInkSourceResolved != wantFlags[3] {
			return receipt, errors.New("final receipt color-source flags differ from its closure")
		}
	}
	if !digestPattern.MatchString(receipt.FirstFrameRGBSHA256) || receipt.PixelProfile != "srgb-rgb24-v1" || receipt.ShotRevision != m.ShotRevision || receipt.BaseShotRevision != m.BaseShotRevision || receipt.BaseRecipeVersion != m.BaseRecipeVersion || receipt.BaseRecipeSHA256 != m.BaseRecipeSHA256 || receipt.RecipeSHA256 != m.RecipeSHA256 {
		return receipt, errors.New("video first-frame or shot recipe identity is invalid")
	}
	if m.Mode == VideoModeFinal && m.Sample != nil && receipt.FirstFrameRGBSHA256 != m.Sample.FirstFrameRGBSHA256 {
		return receipt, errors.New("final first frame differs from accepted sample pixels")
	}
	if err := ctx.Err(); err != nil {
		return receipt, err
	}
	return receipt, nil
}

func hashVerifiedMP4(ctx context.Context, h hash.Hash, source io.Reader, outputSize int64) (int64, error) {
	if outputSize < 16 || outputSize > videoMaxOutputBytes {
		return 0, errors.New("MP4 output size is outside its bounded profile")
	}
	header := make([]byte, 8)
	if _, err := io.ReadFull(contextReader{ctx: ctx, src: source}, header); err != nil {
		return 0, fmt.Errorf("read MP4 file-type header: %w", err)
	}
	boxSize := int64(binary.BigEndian.Uint32(header[:4]))
	if string(header[4:8]) != "ftyp" || boxSize < 16 || boxSize > outputSize {
		return 0, errors.New("video output does not begin with a bounded MP4 ftyp box")
	}
	if _, err := h.Write(header); err != nil {
		return 0, err
	}
	n, err := io.Copy(h, io.LimitReader(contextReader{ctx: ctx, src: source}, videoMaxOutputBytes+1-int64(len(header))))
	return n + int64(len(header)), err
}

func videoResolvedFlags(config *presentations.Config) [4]bool {
	if config == nil || config.Phone == nil || config.Phone.ProfileVersion != presentations.PhoneProfileVersionAuto {
		return [4]bool{}
	}
	phone := config.Phone
	return [4]bool{phone.TopFill == nil, phone.BottomFill == nil, phone.StatusInk == nil, phone.HomeInk == nil}
}

func hasResolvedPhoneValues(flags [4]bool) bool {
	return flags[0] || flags[1] || flags[2] || flags[3]
}

type videoLoadedImage struct {
	binding VideoBinding
	data    []byte
	image   image.Image
}

func RunVideoManifest(ctx context.Context, manifestPath string) (retErr error) {
	if ctx == nil {
		return errors.New("video worker context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !filepath.IsAbs(manifestPath) || filepath.Clean(manifestPath) != manifestPath || filepath.Base(manifestPath) != VideoManifestName {
		return errors.New("video manifest must be the fixed filename in an absolute attempt workspace")
	}
	workspace := filepath.Dir(manifestPath)
	if err := validateWorkspacePath(workspace); err != nil {
		return err
	}
	root, err := os.OpenRoot(workspace)
	if err != nil {
		return fmt.Errorf("open video attempt workspace: %w", err)
	}
	defer root.Close()
	var manifest VideoManifest
	defer func() {
		if retErr != nil && manifest.JobID != "" && manifest.AttemptID != "" {
			if e := writeVideoFailure(root, manifest, retErr); e != nil {
				retErr = errors.Join(retErr, fmt.Errorf("write video failure diagnostic: %w", e))
			}
		}
	}()
	manifestBytes, err := readBoundedRegular(ctx, root, VideoManifestName, videoMaxManifestBytes, 0)
	if err != nil {
		return fmt.Errorf("read video manifest: %w", err)
	}
	manifest, err = decodeCanonicalVideoManifest(manifestBytes)
	if err != nil {
		return err
	}
	if err = validateVideoManifest(manifest); err != nil {
		return err
	}
	if err = verifyCurrentWorker(ctx, manifest.WorkerSHA256); err != nil {
		return fmt.Errorf("verify pinned video worker: %w", err)
	}
	for _, name := range []string{VideoReceiptName, VideoFailureName, VideoSampleName, VideoOutputName} {
		if _, e := root.Lstat(name); e == nil {
			return fmt.Errorf("video output %q already exists", name)
		} else if !errors.Is(e, os.ErrNotExist) {
			return fmt.Errorf("inspect reserved video output: %w", e)
		}
	}
	profileBytes, err := readBoundedRegular(ctx, root, VideoProfileName, inspectionMaxProfile, 0)
	if err != nil {
		return fmt.Errorf("read paired tool profile: %w", err)
	}
	var profile cliv1.RendererToolProfile
	if err = (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(profileBytes, &profile); err != nil {
		return errors.New("video tool profile is not valid renderer-profile JSON")
	}
	pair, err := LocalToolPairFromProfile(&profile)
	if err != nil {
		return err
	}
	if pair.ProfileID != manifest.Inspection.ProfileID || pair.ArtifactIdentity != manifest.Inspection.ProfileDigest || profileArtifactIdentity(&profile) != manifest.Inspection.ProfileDigest {
		return errors.New("video tool profile differs from accepted inspection profile")
	}
	ffmpeg, err := openVerifiedTool(ctx, pair.FFmpegPath, pair.FFmpegSize, pair.FFmpegSHA256)
	if err != nil {
		return fmt.Errorf("verify pinned ffmpeg: %w", err)
	}
	defer ffmpeg.Close()
	ffprobe, err := openVerifiedTool(ctx, pair.FFprobePath, pair.FFprobeSize, pair.FFprobeSHA256)
	if err != nil {
		return fmt.Errorf("verify pinned ffprobe: %w", err)
	}
	defer ffprobe.Close()
	if err := requireNoVideoInputsCollide(manifest); err != nil {
		return err
	}
	inspectionManifestBytes, err := readBoundedRegular(ctx, root, manifest.Inspection.Manifest.Name, inspectionMaxManifest, manifest.Inspection.Manifest.SizeBytes)
	if err != nil {
		return fmt.Errorf("read original source inspection manifest: %w", err)
	}
	inspectionReceiptBytes, err := readBoundedRegular(ctx, root, manifest.Inspection.Receipt.Name, maxReceiptBytes, manifest.Inspection.Receipt.SizeBytes)
	if err != nil {
		return fmt.Errorf("read accepted inspection receipt: %w", err)
	}
	clockBytes, err := readBoundedRegular(ctx, root, manifest.Inspection.Clock.Name, videoMaxClockBytes, manifest.Inspection.Clock.SizeBytes)
	if err != nil {
		return fmt.Errorf("read accepted source clock: %w", err)
	}
	var inspectionManifest InspectionManifest
	if err = decodeCanonicalInspectionManifest(inspectionManifestBytes, &inspectionManifest); err != nil {
		return fmt.Errorf("decode original inspection manifest: %w", err)
	}
	if inspectionManifest.Source.SHA256 != manifest.Inspection.Source.SHA256 || inspectionManifest.Source.SizeBytes != manifest.Inspection.Source.SizeBytes || inspectionManifest.ProfileID != manifest.Inspection.ProfileID || inspectionManifest.ProfileDigest != manifest.Inspection.ProfileDigest || inspectionManifest.ProtocolVersion != manifest.Inspection.ProtocolVersion {
		return errors.New("video source binding differs from original inspection manifest")
	}
	if digest(inspectionManifestBytes) != manifest.Inspection.Manifest.SHA256 || digest(inspectionReceiptBytes) != manifest.Inspection.Receipt.SHA256 || digest(clockBytes) != manifest.Inspection.Clock.SHA256 {
		return errors.New("video inspection artifact bytes differ from immutable bindings")
	}
	inspectionReceipt, err := VerifyInspectionArtifacts(inspectionManifestBytes, inspectionReceiptBytes, clockBytes)
	if err != nil {
		return fmt.Errorf("verify accepted source inspection: %w", err)
	}
	if err = matchVideoInspection(manifest.Inspection, inspectionReceipt); err != nil {
		return err
	}
	pts, err := decodeInspectionClock(clockBytes, inspectionReceipt.FrameCount)
	if err != nil {
		return err
	}
	var sampleReceipt VideoReceipt
	if manifest.Mode == VideoModeFinal && manifest.Sample != nil {
		sampleReceipt, err = readAndVerifySampleReceipt(ctx, root, manifest)
		if err != nil {
			return err
		}
	}
	loaded, err := loadVideoImages(ctx, root, manifest.Scene)
	if err != nil {
		return err
	}
	source, err := openInspectionSource(ctx, root, InspectionSource{Name: manifest.Inspection.Source.Name, SHA256: manifest.Inspection.Source.SHA256, SizeBytes: manifest.Inspection.Source.SizeBytes})
	if err != nil {
		return fmt.Errorf("open pinned source video: %w", err)
	}
	defer source.Close()
	deadline, cancel := context.WithTimeout(ctx, videoMaxWall)
	defer cancel()
	clock := composition.FrameClock{SourcePTS: pts, SourceTimeBase: manifest.Inspection.TimeBase, SourceEndPTS: manifest.Inspection.SourceEndPTS, TrimStartPTS: manifest.TrimStartPTS, TrimEndPTS: manifest.TrimEndPTS, PlaybackRate: manifest.PlaybackRate, OutputRate: manifest.OutputRate, Underflow: composition.RangeReject, EOF: composition.RangeReject}
	plan, err := composition.PlanFrames(deadline, clock)
	if err != nil {
		return fmt.Errorf("plan video output frames: %w", err)
	}
	if len(plan) > videoMaxFrames {
		return errors.New("video schedule exceeds the fixed 300-frame development profile")
	}
	if err := validatePlateForVideo(deadline, loaded[0].image, manifest); err != nil {
		return err
	}
	var cameraPath *videoCameraPath
	if manifest.Camera != nil {
		cameraPath, err = newVideoCameraPath(manifest, loaded[0].image.Bounds(), manifest.PresentationLayout)
		if err != nil {
			return fmt.Errorf("prepare screen-entry camera: %w", err)
		}
	}
	var firstDigest string
	var resolvedLayout presentations.Layout
	var outputFacts VideoNativeFacts
	var outputName string
	var outputSize int64
	var outputHash string
	var encoder *videoEncoder
	if manifest.Mode == VideoModeFinal {
		encoder, err = startVideoEncoder(deadline, root, ffmpeg, manifest, len(plan))
		if err != nil {
			return err
		}
		defer func() {
			if encoder != nil {
				encoder.abort()
			}
		}()
	}
	selected := 0
	outputsWanted := len(plan)
	if manifest.Mode == VideoModeSample {
		outputsWanted = 1
	}
	callback := func(index int, pts int64, rgb []byte) error {
		for selected < outputsWanted && plan[selected].SourceIndex == index {
			if err := deadline.Err(); err != nil {
				return err
			}
			app := image.NewNRGBA(image.Rect(0, 0, manifest.Inspection.Width, manifest.Inspection.Height))
			if err := expandRGB24(app.Pix, rgb); err != nil {
				return err
			}
			presented, e := presentations.Compose(deadline, app, *manifest.Presentation)
			if e != nil {
				return fmt.Errorf("compose frame presentation: %w", e)
			}
			if manifest.PresentationLayout != nil && !sameJSON(presented.Layout, *manifest.PresentationLayout) {
				return errors.New("frame presentation differs from frozen layout")
			}
			var frame image.Image
			if cameraPath != nil {
				if presented.Layout.AppViewport != cameraPath.viewport {
					return errors.New("frame active viewport differs from the prevalidated camera path")
				}
				viewport, viewportErr := presentations.ComposeActiveViewport(deadline, app, *manifest.Presentation, presented.Layout)
				if viewportErr != nil {
					return fmt.Errorf("compose clean active viewport for camera handoff: %w", viewportErr)
				}
				matte := color.NRGBA{R: cameraPath.matte.R, G: cameraPath.matte.G, B: cameraPath.matte.B, A: 255}
				if uint32(selected) >= cameraPath.duration-1 {
					frame, e = composition.RenderScreenEntryEndpoint(deadline, image.Pt(manifest.Width, manifest.Height), viewport, cameraPath.target, matte)
				} else {
					quad := [4]composition.Point{}
					for n, p := range manifest.Scene.Quad {
						quad[n] = composition.Point{X: p.X, Y: p.Y}
					}
					composed, composeErr := composition.Compose(deadline, composition.Inputs{Plate: loaded[0].image, Screen: presented.Raster, ActiveMask: loaded[1].image, HardwareMask: videoHardware(loaded), SurfaceSize: image.Pt(manifest.Scene.SurfaceWidth, manifest.Scene.SurfaceHeight), Quad: quad, Samples: manifest.Scene.Samples})
					if composeErr != nil {
						return fmt.Errorf("compose photographed video frame: %w", composeErr)
					}
					frame, e = resizeVideoFrameWithMatte(deadline, composed.Image, manifest.Width, manifest.Height, matte)
					if e == nil {
						to := cameraPath.quad(uint32(selected))
						blend, fade := cameraPath.blend(uint32(selected))
						frame, e = composition.RenderScreenEntry(deadline, frame, cameraPath.from, to, viewport, matte, blend, fade)
					}
				}
				if e != nil {
					return fmt.Errorf("render camera screen entry: %w", e)
				}
			}
			if frame == nil {
				quad := [4]composition.Point{}
				for n, p := range manifest.Scene.Quad {
					quad[n] = composition.Point{X: p.X, Y: p.Y}
				}
				composed, composeErr := composition.Compose(deadline, composition.Inputs{Plate: loaded[0].image, Screen: presented.Raster, ActiveMask: loaded[1].image, HardwareMask: videoHardware(loaded), SurfaceSize: image.Pt(manifest.Scene.SurfaceWidth, manifest.Scene.SurfaceHeight), Quad: quad, Samples: manifest.Scene.Samples})
				if composeErr != nil {
					return fmt.Errorf("compose photographed video frame: %w", composeErr)
				}
				frame, e = resizeVideoFrame(deadline, composed.Image, manifest.Width, manifest.Height)
				if e != nil {
					return e
				}
			}
			pixels, e := imageRGB24(deadline, frame)
			if e != nil {
				return e
			}
			pixelID := canonicalVideoPixelDigest(manifest.Width, manifest.Height, "srgb-rgb24-v1", pixels)
			if selected == 0 {
				firstDigest = pixelID
				resolvedLayout = presented.Layout
				if manifest.Mode == VideoModeFinal && manifest.Sample != nil && pixelID != sampleReceipt.FirstFrameRGBSHA256 {
					return errors.New("final frame zero differs from accepted sample pixels")
				}
			}
			if manifest.Mode == VideoModeSample {
				pngBytes, e := encodeSamplePNG(deadline, frame)
				if e != nil {
					return e
				}
				if e = publishExclusive(deadline, root, VideoSampleName, pngBytes, videoMaxOutputBytes); e != nil {
					return fmt.Errorf("publish sample PNG: %w", e)
				}
				outputName = VideoSampleName
				outputSize = int64(len(pngBytes))
				outputHash = digest(pngBytes)
			} else {
				if e = encoder.writeFrame(deadline, pixels); e != nil {
					return e
				}
			}
			selected++
		}
		return nil
	}
	decodeErr := decodeVideoRGB(deadline, ffmpeg, source, manifest, pts, callback)
	if decodeErr != nil {
		return fmt.Errorf("decode and compose source frames: %w", decodeErr)
	}
	if selected == 0 {
		return errors.New("video schedule did not select a decoded source frame")
	}
	if err = verifyInspectionSourceUnchanged(deadline, source, inspectionManifest.Source); err != nil {
		return fmt.Errorf("source changed during video render: %w", err)
	}
	if manifest.Mode == VideoModeFinal {
		if selected != len(plan) {
			return fmt.Errorf("composed %d output frames, expected %d", selected, len(plan))
		}
		outputSize, outputHash, err = encoder.finish(deadline)
		encoder = nil
		if err != nil {
			return fmt.Errorf("finish bounded MP4 encoder: %w", err)
		}
		outputName = VideoOutputName
		outputFacts, err = verifyEncodedVideo(deadline, ffprobe, ffmpeg, root, manifest, len(plan))
		if err != nil {
			return fmt.Errorf("verify encoded MP4: %w", err)
		}
	} else {
		outputFacts = VideoNativeFacts{FrameCount: 1}
	}
	if err = verifyVideoUnchanged(deadline, root, manifestBytes, profileBytes, inspectionManifestBytes, inspectionReceiptBytes, clockBytes, manifest, loaded); err != nil {
		return err
	}
	if err = verifyInspectionSourceUnchanged(deadline, source, inspectionManifest.Source); err != nil {
		return fmt.Errorf("source changed before receipt publication: %w", err)
	}
	if err = verifyOpenedVideoTool(deadline, ffmpeg, pair.FFmpegSize, pair.FFmpegSHA256); err != nil {
		return fmt.Errorf("ffmpeg changed during video render: %w", err)
	}
	if err = verifyOpenedVideoTool(deadline, ffprobe, pair.FFprobeSize, pair.FFprobeSHA256); err != nil {
		return fmt.Errorf("ffprobe changed during video render: %w", err)
	}
	if err = deadline.Err(); err != nil {
		return err
	}
	receipt := VideoReceipt{Version: videoManifestVersion, ProtocolVersion: videoProtocolVersion, JobID: manifest.JobID, AttemptID: manifest.AttemptID, InputFingerprint: manifest.InputFingerprint, ShotRevision: manifest.ShotRevision, BaseShotRevision: manifest.BaseShotRevision, BaseRecipeVersion: manifest.BaseRecipeVersion, BaseRecipeSHA256: manifest.BaseRecipeSHA256, RecipeSHA256: manifest.RecipeSHA256, ManifestSHA256: digest(manifestBytes), InspectionSHA256: VideoInspectionDigest(manifest.Inspection), WorkerSHA256: manifest.WorkerSHA256, Mode: manifest.Mode, Camera: manifest.Camera, OutputName: outputName, OutputSHA256: outputHash, OutputSizeBytes: outputSize, OutputWidth: manifest.Width, OutputHeight: manifest.Height, FirstFrameRGBSHA256: firstDigest, PixelProfile: "srgb-rgb24-v1", Native: outputFacts, CompletedAt: time.Now().UTC()}
	if manifest.Mode == VideoModeSample {
		receipt.PresentationLayout = &resolvedLayout
		receipt.ResolvedTopFill = resolvedLayout.ResolvedTopFill
		receipt.ResolvedBottomFill = resolvedLayout.ResolvedBottomFill
		receipt.ResolvedStatusInk = resolvedLayout.ResolvedStatusInk
		receipt.ResolvedHomeInk = resolvedLayout.ResolvedHomeInk
		flags := videoResolvedFlags(manifest.Presentation)
		receipt.TopFillSourceResolved = flags[0]
		receipt.BottomFillSourceResolved = flags[1]
		receipt.StatusInkSourceResolved = flags[2]
		receipt.HomeInkSourceResolved = flags[3]
	} else {
		receipt.PresentationLayout = manifest.PresentationLayout
		receipt.ResolvedTopFill = manifest.PresentationLayout.ResolvedTopFill
		receipt.ResolvedBottomFill = manifest.PresentationLayout.ResolvedBottomFill
		receipt.ResolvedStatusInk = manifest.PresentationLayout.ResolvedStatusInk
		receipt.ResolvedHomeInk = manifest.PresentationLayout.ResolvedHomeInk
		if manifest.Sample != nil {
			receipt.TopFillSourceResolved = manifest.Sample.TopFillSourceResolved
			receipt.BottomFillSourceResolved = manifest.Sample.BottomFillSourceResolved
			receipt.StatusInkSourceResolved = manifest.Sample.StatusInkSourceResolved
			receipt.HomeInkSourceResolved = manifest.Sample.HomeInkSourceResolved
		}
	}
	receiptBytes, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	if len(receiptBytes) > videoMaxReceiptBytes {
		return errors.New("video receipt exceeds fixed size")
	}
	if err = deadline.Err(); err != nil {
		return err
	}
	if err = publishExclusive(deadline, root, VideoReceiptName, receiptBytes, videoMaxReceiptBytes); err != nil {
		return fmt.Errorf("publish video receipt: %w", err)
	}
	return nil
}

func matchVideoInspection(v VideoInspection, r InspectionReceipt) error {
	if r.Classification != InspectionAccepted || r.VideoStreamIndex == nil || uint32(*r.VideoStreamIndex) != v.SelectedStream || r.TimeBaseNum != v.TimeBase.Num || r.TimeBaseDen != v.TimeBase.Den || r.StreamStartPTS == nil || *r.StreamStartPTS != v.StreamStartPTS || r.SourceEndPTS == nil || *r.SourceEndPTS != v.SourceEndPTS || r.LastFrameDurationTicks != v.LastFrameDurationTicks || r.EOFPolicy != v.EOFPolicy || r.FrameCount != v.FrameCount || int(r.DisplayWidth) != v.Width || int(r.DisplayHeight) != v.Height || r.ColorPrimaries != v.ColorPrimaries || r.ColorTransfer != v.ColorTransfer || r.ColorMatrix != v.ColorMatrix || r.ColorRange != v.ColorRange || r.ColorPolicy != v.ColorPolicy {
		return errors.New("video clock facts differ from accepted inspection receipt")
	}
	return nil
}

func requireNoVideoInputsCollide(m VideoManifest) error {
	seen := map[string]bool{VideoManifestName: true, VideoReceiptName: true, VideoFailureName: true, VideoProfileName: true, VideoSampleName: true, VideoOutputName: true}
	all := []VideoBinding{m.Inspection.Source, m.Inspection.Manifest, m.Inspection.Receipt, m.Inspection.Clock, m.Scene.Plate, m.Scene.ActiveMask}
	if m.Scene.HardwareMask != nil {
		all = append(all, *m.Scene.HardwareMask)
	}
	if m.Sample != nil {
		all = append(all, m.Sample.ReceiptArtifact)
	}
	for _, b := range all {
		if seen[b.Name] {
			return fmt.Errorf("video input %q collides with another binding", b.Name)
		}
		seen[b.Name] = true
	}
	return nil
}

func loadVideoImages(ctx context.Context, root *os.Root, s VideoScene) ([]videoLoadedImage, error) {
	bindings := []VideoBinding{s.Plate, s.ActiveMask}
	if s.HardwareMask != nil {
		bindings = append(bindings, *s.HardwareMask)
	}
	loaded := make([]videoLoadedImage, 0, len(bindings))
	var total int64
	for _, b := range bindings {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		data, err := readBoundedRegular(ctx, root, b.Name, composition.StillMaxImageBytes, b.SizeBytes)
		if err != nil {
			return nil, err
		}
		if digest(data) != b.SHA256 {
			return nil, fmt.Errorf("scene image %q digest mismatch", b.Name)
		}
		img, format, err := decodeImage(data)
		if err != nil {
			return nil, err
		}
		if format != "png" {
			return nil, errors.New("video plate and masks must be PNG")
		}
		total += int64(len(data))
		if total > 64<<20 {
			return nil, errors.New("video scene inputs exceed aggregate limit")
		}
		loaded = append(loaded, videoLoadedImage{binding: b, data: data, image: img})
	}
	return loaded, nil
}

func videoHardware(images []videoLoadedImage) image.Image {
	if len(images) > 2 {
		return images[2].image
	}
	return nil
}

func validatePlateForVideo(ctx context.Context, plate image.Image, m VideoManifest) error {
	b := plate.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return errors.New("video plate has empty dimensions")
	}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		if y&63 == 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
		}
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := plate.At(x, y).RGBA()
			if a != 65535 {
				return errors.New("video plate must be fully opaque")
			}
		}
	}
	return nil
}

func imageRGB24(ctx context.Context, img image.Image) ([]byte, error) {
	b := img.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 || int64(b.Dx())*int64(b.Dy()) > videoMaxPixels {
		return nil, errors.New("composed frame dimensions exceed video profile")
	}
	out := make([]byte, b.Dx()*b.Dy()*3)
	at := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		if y&31 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := img.At(x, y).RGBA()
			if a != 65535 {
				return nil, errors.New("composed video frame is not fully opaque")
			}
			out[at] = uint8(r >> 8)
			out[at+1] = uint8(g >> 8)
			out[at+2] = uint8(bl >> 8)
			at += 3
		}
	}
	return out, nil
}

func expandRGB24(dst, src []byte) error {
	if len(src) == 0 || len(src)%3 != 0 || len(dst) != len(src)/3*4 {
		return errors.New("decoded RGB frame has invalid packed size")
	}
	for in, out := 0, 0; in < len(src); in, out = in+3, out+4 {
		dst[out] = src[in]
		dst[out+1] = src[in+1]
		dst[out+2] = src[in+2]
		dst[out+3] = 255
	}
	return nil
}

func resizeVideoFrame(ctx context.Context, src image.Image, width, height int) (*image.RGBA64, error) {
	return resizeVideoFrameWithMatte(ctx, src, width, height, videoLetterboxMatte)
}

type videoFit struct {
	scale, offsetX, offsetY float64
}

func videoContainFit(source, target image.Point) (videoFit, error) {
	if source.X <= 0 || source.Y <= 0 || target.X <= 0 || target.Y <= 0 {
		return videoFit{}, errors.New("video fit dimensions must be positive")
	}
	fit := videoFit{scale: math.Min(float64(target.X)/float64(source.X), float64(target.Y)/float64(source.Y))}
	if math.IsNaN(fit.scale) || math.IsInf(fit.scale, 0) || fit.scale <= 0 {
		return videoFit{}, errors.New("video fit scale is invalid")
	}
	fit.offsetX = (float64(target.X) - float64(source.X)*fit.scale) / 2
	fit.offsetY = (float64(target.Y) - float64(source.Y)*fit.scale) / 2
	return fit, nil
}

func resizeVideoFrameWithMatte(ctx context.Context, src image.Image, width, height int, matte color.NRGBA) (*image.RGBA64, error) {
	if src == nil || width <= 0 || height <= 0 || int64(width)*int64(height) > videoMaxPixels {
		return nil, errors.New("video resample target exceeds profile")
	}
	b := src.Bounds()
	fit, err := videoContainFit(image.Pt(b.Dx(), b.Dy()), image.Pt(width, height))
	if err != nil {
		return nil, err
	}
	dst := image.NewRGBA64(image.Rect(0, 0, width, height))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: matte}, image.Point{}, draw.Src)
	for y := 0; y < height; y++ {
		if y&15 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		py := float64(y) + 0.5
		if py < fit.offsetY || py >= fit.offsetY+float64(b.Dy())*fit.scale {
			continue
		}
		fy := (py-fit.offsetY)/fit.scale - 0.5
		y0 := int(math.Floor(fy))
		wy := fy - float64(y0)
		if y0 < 0 {
			y0 = 0
			wy = 0
		}
		y1 := y0 + 1
		if y1 >= b.Dy() {
			y1 = b.Dy() - 1
		}
		for x := 0; x < width; x++ {
			px := float64(x) + 0.5
			if px < fit.offsetX || px >= fit.offsetX+float64(b.Dx())*fit.scale {
				continue
			}
			fx := (px-fit.offsetX)/fit.scale - 0.5
			x0 := int(math.Floor(fx))
			wx := fx - float64(x0)
			if x0 < 0 {
				x0 = 0
				wx = 0
			}
			x1 := x0 + 1
			if x1 >= b.Dx() {
				x1 = b.Dx() - 1
			}
			r, g, bl, a := bilinearRGBA64(src.At(b.Min.X+x0, b.Min.Y+y0), src.At(b.Min.X+x1, b.Min.Y+y0), src.At(b.Min.X+x0, b.Min.Y+y1), src.At(b.Min.X+x1, b.Min.Y+y1), wx, wy)
			if a != 65535 {
				return nil, errors.New("video resample encountered nonopaque scene pixel")
			}
			dst.SetRGBA64(x, y, color.RGBA64{R: r, G: g, B: bl, A: a})
		}
	}
	return dst, nil
}

func bilinearRGBA64(c00, c10, c01, c11 color.Color, wx, wy float64) (uint16, uint16, uint16, uint16) {
	colors := [4]color.Color{c00, c10, c01, c11}
	weights := [4]float64{(1 - wx) * (1 - wy), wx * (1 - wy), (1 - wx) * wy, wx * wy}
	var out [4]uint16
	for channel := 0; channel < 4; channel++ {
		sum := 0.0
		for i, c := range colors {
			r, g, b, a := c.RGBA()
			value := [4]uint32{r, g, b, a}
			sum += float64(value[channel]) * weights[i]
		}
		out[channel] = uint16(math.Round(sum))
	}
	return out[0], out[1], out[2], out[3]
}

func canonicalVideoPixelDigest(w, h int, profile string, rgb []byte) string {
	hasher := sha256.New()
	var header [12]byte
	binary.BigEndian.PutUint32(header[:4], uint32(w))
	binary.BigEndian.PutUint32(header[4:8], uint32(h))
	binary.BigEndian.PutUint32(header[8:], uint32(len(profile)))
	_, _ = hasher.Write(header[:])
	_, _ = io.WriteString(hasher, profile)
	_, _ = hasher.Write(rgb)
	return hex.EncodeToString(hasher.Sum(nil))
}

func encodeSamplePNG(ctx context.Context, img image.Image) ([]byte, error) {
	var output boundedVideoBuffer
	output.limit = 64 << 20
	if err := (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(contextWriter{ctx: ctx, dst: &output}, img); err != nil {
		return nil, err
	}
	if output.Len() == 0 {
		return nil, errors.New("sample PNG exceeds output limit")
	}
	return bytes.Clone(output.Bytes()), nil
}

type boundedVideoBuffer struct {
	buffer bytes.Buffer
	limit  int64
}

func (b *boundedVideoBuffer) Write(p []byte) (int, error) {
	if b.limit <= 0 || int64(len(p)) > b.limit-int64(b.buffer.Len()) {
		return 0, errors.New("sample PNG exceeds 64 MiB output limit")
	}
	return b.buffer.Write(p)
}

func (b *boundedVideoBuffer) Len() int      { return b.buffer.Len() }
func (b *boundedVideoBuffer) Bytes() []byte { return b.buffer.Bytes() }

func readAndVerifySampleReceipt(ctx context.Context, root *os.Root, m VideoManifest) (VideoReceipt, error) {
	var zero VideoReceipt
	if m.Sample == nil {
		return zero, errors.New("final video has no accepted sample binding")
	}
	b := m.Sample.ReceiptArtifact
	data, err := readBoundedRegular(ctx, root, b.Name, videoMaxReceiptBytes, b.SizeBytes)
	if err != nil {
		return zero, err
	}
	if digest(data) != b.SHA256 || b.SHA256 != m.Sample.ReceiptSHA256 {
		return zero, errors.New("accepted sample receipt digest mismatch")
	}
	var r VideoReceipt
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err = d.Decode(&r); err != nil {
		return zero, err
	}
	canonical, e := json.Marshal(r)
	if e != nil || !bytes.Equal(canonical, data) {
		return zero, errors.New("accepted sample receipt is not canonical")
	}
	if r.Mode != VideoModeSample || r.OutputName != VideoSampleName || r.OutputSizeBytes <= 0 || r.OutputSizeBytes > 64<<20 || !digestPattern.MatchString(r.OutputSHA256) || r.Native.FrameCount != 1 || r.Native.FullDecodeVerified || !digestPattern.MatchString(r.WorkerSHA256) || !digestPattern.MatchString(r.InputFingerprint) || !digestPattern.MatchString(r.ManifestSHA256) || !digestPattern.MatchString(r.InspectionSHA256) || r.JobID != m.Sample.JobID || r.AttemptID != m.Sample.AttemptID || r.FirstFrameRGBSHA256 != m.Sample.FirstFrameRGBSHA256 || r.OutputWidth != m.Width || r.OutputHeight != m.Height || r.PixelProfile != m.Sample.PixelProfile || r.ManifestSHA256 != m.Sample.ManifestSHA256 || r.InspectionSHA256 != m.Sample.InspectionSHA256 || r.RecipeSHA256 != m.Sample.RecipeSHA256 || r.ShotRevision != m.Sample.ShotRevision || r.BaseShotRevision != m.Sample.BaseShotRevision || r.BaseRecipeVersion != m.Sample.BaseRecipeVersion || r.BaseRecipeSHA256 != m.Sample.BaseRecipeSHA256 || r.TopFillSourceResolved != m.Sample.TopFillSourceResolved || r.BottomFillSourceResolved != m.Sample.BottomFillSourceResolved || r.StatusInkSourceResolved != m.Sample.StatusInkSourceResolved || r.HomeInkSourceResolved != m.Sample.HomeInkSourceResolved || r.ResolvedTopFill != m.Sample.ResolvedTopFill || r.ResolvedBottomFill != m.Sample.ResolvedBottomFill || !sameRGB(r.ResolvedStatusInk, m.Sample.ResolvedStatusInk) || !sameRGB(r.ResolvedHomeInk, m.Sample.ResolvedHomeInk) {
		return zero, errors.New("accepted sample receipt identity mismatch")
	}
	if m.Sample.SourceSHA256 != m.Inspection.Source.SHA256 || m.Sample.ClockSHA256 != m.Inspection.Clock.SHA256 || m.Sample.InspectionSHA256 != VideoInspectionDigest(m.Inspection) || m.Sample.BaseRecipeSHA256 != m.BaseRecipeSHA256 || m.Sample.ShotRevision != m.BaseShotRevision {
		return zero, errors.New("accepted sample recipe differs from final video closure")
	}
	if r.PresentationLayout == nil || m.PresentationLayout == nil || !sameSamplePresentationLayout(*r.PresentationLayout, *m.PresentationLayout) {
		return zero, errors.New("accepted sample presentation geometry differs from final video closure")
	}
	return r, nil
}

func sameSamplePresentationLayout(sample, final presentations.Layout) bool {
	sample.ConfigSHA256 = ""
	final.ConfigSHA256 = ""
	return sameJSON(sample, final)
}

func verifyVideoUnchanged(ctx context.Context, root *os.Root, manifestBytes, profileBytes, inspectionManifestBytes, receipt, clock []byte, m VideoManifest, loaded []videoLoadedImage) error {
	if data, e := readBoundedRegular(ctx, root, VideoManifestName, videoMaxManifestBytes, int64(len(manifestBytes))); e != nil || !bytes.Equal(data, manifestBytes) {
		return errors.New("video manifest changed during render")
	}
	for _, b := range []VideoBinding{m.Inspection.Manifest, m.Inspection.Receipt, m.Inspection.Clock, m.Scene.Plate, m.Scene.ActiveMask} {
		data, e := readBoundedRegular(ctx, root, b.Name, videoMaxInputBytes, b.SizeBytes)
		if e != nil || digest(data) != b.SHA256 {
			return fmt.Errorf("video input %q changed during render", b.Name)
		}
	}
	if data, e := readBoundedRegular(ctx, root, VideoProfileName, inspectionMaxProfile, int64(len(profileBytes))); e != nil || !bytes.Equal(data, profileBytes) {
		return errors.New("video tool profile changed during render")
	}
	if data, e := readBoundedRegular(ctx, root, m.Inspection.Manifest.Name, inspectionMaxManifest, m.Inspection.Manifest.SizeBytes); e != nil || !bytes.Equal(data, inspectionManifestBytes) {
		return errors.New("original inspection manifest changed during render")
	}
	if m.Scene.HardwareMask != nil {
		b := *m.Scene.HardwareMask
		data, e := readBoundedRegular(ctx, root, b.Name, videoMaxInputBytes, b.SizeBytes)
		if e != nil || digest(data) != b.SHA256 {
			return errors.New("video hardware mask changed during render")
		}
	}
	if data, e := readBoundedRegular(ctx, root, m.Inspection.Receipt.Name, maxReceiptBytes, m.Inspection.Receipt.SizeBytes); e != nil || !bytes.Equal(data, receipt) {
		return errors.New("accepted inspection receipt changed during render")
	}
	if data, e := readBoundedRegular(ctx, root, m.Inspection.Clock.Name, videoMaxClockBytes, m.Inspection.Clock.SizeBytes); e != nil || !bytes.Equal(data, clock) {
		return errors.New("accepted source clock changed during render")
	}
	if m.Sample != nil {
		b := m.Sample.ReceiptArtifact
		data, e := readBoundedRegular(ctx, root, b.Name, videoMaxReceiptBytes, b.SizeBytes)
		if e != nil || digest(data) != b.SHA256 || b.SHA256 != m.Sample.ReceiptSHA256 {
			return errors.New("accepted sample receipt changed during render")
		}
	}
	_ = loaded
	return nil
}

func verifyOpenedVideoTool(ctx context.Context, file *os.File, size int64, expected string) error {
	if ctx == nil || file == nil {
		return errors.New("tool identity check needs context and open file")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	h := sha256.New()
	n, err := io.Copy(h, contextReader{ctx: ctx, src: io.LimitReader(file, size+1)})
	if err != nil || n != size || hex.EncodeToString(h.Sum(nil)) != expected {
		return errors.New("open renderer executable bytes changed after use")
	}
	_, err = file.Seek(0, io.SeekStart)
	return err
}

func writeVideoFailure(root *os.Root, m VideoManifest, cause error) error {
	message := strings.ReplaceAll(cause.Error(), root.Name(), "<attempt-workspace>")
	message = truncateUTF8(message, 768)
	data, err := json.Marshal(InspectionFailure{Version: videoManifestVersion, Code: "video_render_failed", Message: message, JobID: m.JobID, AttemptID: m.AttemptID, FailedAt: time.Now().UTC().Format(time.RFC3339Nano)})
	if err != nil {
		return err
	}
	return publishExclusive(context.Background(), root, VideoFailureName, data, maxDiagnosticBytes)
}

type videoPTS struct {
	ordinal  int
	pts      int64
	timeBase string
}

func decodeVideoRGB(ctx context.Context, tool, source *os.File, m VideoManifest, expected []int64, onFrame func(int, int64, []byte) error) error {
	return decodeVideoRGBFrames(ctx, tool, source, m, expected, 0, onFrame)
}

// decodeVideoRGBPrefix decodes only the source prefix needed by a preplanned
// schedule. Every emitted RGB frame is still checked against its original PTS.
func decodeVideoRGBPrefix(ctx context.Context, tool, source *os.File, m VideoManifest, expected []int64, prefix uint32, onFrame func(int, int64, []byte) error) error {
	if prefix == 0 || prefix > uint32(len(expected)) {
		return errors.New("video decode prefix is outside the accepted source clock")
	}
	return decodeVideoRGBFrames(ctx, tool, source, m, expected[:prefix], prefix, onFrame)
}

func decodeVideoRGBFrames(ctx context.Context, tool, source *os.File, m VideoManifest, expected []int64, prefix uint32, onFrame func(int, int64, []byte) error) error {
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return err
	}
	frameSize64 := int64(m.Inspection.Width) * int64(m.Inspection.Height) * 3
	if frameSize64 <= 0 || frameSize64 > inspectionMaxPixels*3 {
		return errors.New("video decoded frame exceeds inspection raster limits")
	}
	statsR, statsW, err := os.Pipe()
	if err != nil {
		return err
	}
	defer statsR.Close()
	defer statsW.Close()
	args := []string{"-hide_banner", "-nostdin", "-v", "error", "-protocol_whitelist", "file,pipe", "-format_whitelist", "mov,matroska,webm", "-max_streams", "8", "-xerror", "-err_detect", "explode", "-threads", "2", "-filter_threads", "2", "-copyts", "-i", inspectionInputDescriptor, "-map", fmt.Sprintf("0:%d", m.Inspection.SelectedStream), "-an", "-sn", "-dn", "-fps_mode", "passthrough", "-enc_time_base:v", fmt.Sprintf("%d/%d", m.Inspection.TimeBase.Num, m.Inspection.TimeBase.Den), "-vf", videoInputFilter(m.Inspection), "-stats_enc_pre", "pipe:5", "-stats_enc_pre_fmt", "{n} {pts} {tb}"}
	if prefix > 0 {
		args = append(args, "-frames:v", strconv.FormatUint(uint64(prefix), 10))
	}
	args = append(args, "-c:v", "rawvideo", "-pix_fmt", "rgb24", "-f", "rawvideo", "pipe:1")
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := exec.CommandContext(runCtx, "/proc/self/fd/3", args...)
	cmd.Env = []string{"LANG=C", "LC_ALL=C"}
	cmd.ExtraFiles = []*os.File{tool, source, statsW}
	raw, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	var stderr boundedInspectionBuffer
	stderr.limit = videoMaxStderr
	cmd.Stderr = &stderr
	if err = cmd.Start(); err != nil {
		return err
	}
	_ = statsW.Close()
	stats := make(chan videoPTS, 1)
	statsDone := make(chan error, 1)
	go readVideoStats(runCtx, statsR, stats, statsDone, videoMaxStats)
	frame := make([]byte, int(frameSize64))
	count := 0
	commandWaited := false
	statsJoined := false
	cleanup := func() {
		cancel()
		if !commandWaited && cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			commandWaited = true
		}
		_ = statsR.Close()
		if !statsJoined {
			<-statsDone
			statsJoined = true
		}
	}
	defer cleanup()
	for {
		if err := ctx.Err(); err != nil {
			cleanup()
			return err
		}
		_, e := io.ReadFull(raw, frame)
		if errors.Is(e, io.EOF) {
			break
		}
		if e != nil {
			cleanup()
			return fmt.Errorf("raw RGB frame %d is truncated: %w", count, e)
		}
		if count >= len(expected) {
			cleanup()
			return errors.New("decoder emitted more frames than the accepted source clock")
		}
		var row videoPTS
		var ok bool
		select {
		case row, ok = <-stats:
			if !ok {
				cleanup()
				return errors.New("decoder timestamp stream ended before raw frames")
			}
		case <-runCtx.Done():
			cleanup()
			return runCtx.Err()
		}
		tb := fmt.Sprintf("%d/%d", m.Inspection.TimeBase.Num, m.Inspection.TimeBase.Den)
		if row.ordinal != count || row.pts != expected[count] || row.timeBase != tb {
			cleanup()
			return fmt.Errorf("decoded frame %d timestamp differs from accepted source clock", count)
		}
		if err = onFrame(count, row.pts, frame); err != nil {
			cleanup()
			return err
		}
		count++
	}
	// Drain timestamp EOF before waiting for the child. Otherwise an extra row
	// can fill the bounded channel and block the stats reader while the child
	// still has the stats pipe open.
	for {
		select {
		case _, ok := <-stats:
			if !ok {
				goto statsDrained
			}
			cleanup()
			return errors.New("decoder timestamp stream contains extra rows")
		case <-runCtx.Done():
			cleanup()
			return runCtx.Err()
		}
	}
statsDrained:
	statsErr := <-statsDone
	statsJoined = true
	if err = cmd.Wait(); err != nil {
		commandWaited = true
		if statsErr != nil {
			return fmt.Errorf("pinned raw RGB decoder failed: %w: %s (stats: %v)", err, truncateUTF8(stderr.data.String(), 512), statsErr)
		}
		return fmt.Errorf("pinned raw RGB decoder failed: %w: %s", err, truncateUTF8(stderr.data.String(), 512))
	}
	commandWaited = true
	if statsErr != nil {
		return fmt.Errorf("decoder timestamp stream failed: %w", statsErr)
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if count != len(expected) {
		return fmt.Errorf("decoder emitted %d RGB frames; accepted clock has %d", count, len(expected))
	}
	return nil
}

func readVideoStats(ctx context.Context, r io.Reader, out chan<- videoPTS, done chan<- error, maxBytes int64) {
	defer close(out)
	reader := bufio.NewReaderSize(r, 256)
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			done <- err
			return
		}
		lineBytes, err := reader.ReadSlice('\n')
		line := string(lineBytes)
		total += int64(len(line))
		if total > maxBytes || errors.Is(err, bufio.ErrBufferFull) {
			done <- errors.New("video decoder timestamp stream exceeds limit")
			return
		}
		if len(line) > 256 {
			done <- errors.New("video decoder timestamp row exceeds limit")
			return
		}
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) != 3 {
				done <- errors.New("video decoder timestamp row is malformed")
				return
			}
			n, e1 := strconv.Atoi(parts[0])
			p, e2 := strconv.ParseInt(parts[1], 10, 64)
			if e1 != nil || e2 != nil {
				done <- errors.New("video decoder timestamp row is invalid")
				return
			}
			select {
			case out <- videoPTS{n, p, parts[2]}:
			case <-ctx.Done():
				done <- ctx.Err()
				return
			}
		}
		if errors.Is(err, io.EOF) {
			if line != "" {
				done <- errors.New("video decoder timestamp final row lacks newline")
				return
			}
			done <- nil
			return
		}
		if err != nil {
			done <- err
			return
		}
	}
}

func videoInputFilter(i VideoInspection) string {
	space := i.ColorMatrix
	if space == "" {
		space = "bt709"
	}
	rng := i.ColorRange
	if rng == "tv" || rng == "mpeg" {
		rng = "tv"
	} else {
		rng = "pc"
	}
	prim := i.ColorPrimaries
	if prim == "" {
		prim = "bt709"
	}
	trc := i.ColorTransfer
	if trc == "" {
		trc = "bt709"
	}
	return fmt.Sprintf("colorspace=ispace=%s:irange=%s:iprimaries=%s:itrc=%s:space=gbr:range=pc:primaries=bt709:trc=iec61966-2-1,format=rgb24", space, rng, prim, trc)
}

type videoEncoder struct {
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	file       *os.File
	root       *os.Root
	temporary  string
	outputName string
	limit      *videoFileLimit
	stderr     *boundedInspectionBuffer
	finished   bool
	waited     bool
}
type videoFileLimit struct {
	file    *os.File
	written int64
	limit   int64
}

func (w *videoFileLimit) Write(p []byte) (int, error) {
	if int64(len(p)) > w.limit-w.written {
		return 0, errors.New("encoded MP4 exceeded 160 MiB output limit")
	}
	n, e := w.file.Write(p)
	w.written += int64(n)
	return n, e
}

func startVideoEncoder(ctx context.Context, root *os.Root, tool *os.File, m VideoManifest, frameCount int) (*videoEncoder, error) {
	return startVideoEncoderNamed(ctx, root, tool, m, frameCount, VideoOutputName)
}

func startVideoEncoderNamed(ctx context.Context, root *os.Root, tool *os.File, m VideoManifest, frameCount int, outputName string) (*videoEncoder, error) {
	if !isSafePhysicalName(outputName) || outputName == "" {
		return nil, errors.New("video encoder output name is invalid")
	}
	tmp := outputName + ".tmp"
	file, err := root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, err
	}
	args := []string{"-hide_banner", "-nostdin", "-v", "error", "-threads", "2", "-filter_threads", "2", "-protocol_whitelist", "file,pipe", "-f", "rawvideo", "-pix_fmt", "rgb24", "-video_size", fmt.Sprintf("%dx%d", m.Width, m.Height), "-framerate", fmt.Sprintf("%d/%d", m.OutputRate.Num, m.OutputRate.Den), "-i", "pipe:0", "-an", "-frames:v", fmt.Sprint(frameCount), "-vf", "zscale=primariesin=bt709:transferin=iec61966-2-1:matrixin=gbr:rangein=full:primaries=bt709:transfer=bt709:matrix=bt709:range=limited,format=yuv420p", "-c:v", "libx264", "-threads:v", "2", "-preset", "medium", "-crf", "18", "-bf", "0", "-pix_fmt", "yuv420p", "-color_range", "tv", "-colorspace", "bt709", "-color_primaries", "bt709", "-color_trc", "bt709", "-movflags", "+frag_keyframe+empty_moov+default_base_moof", "-f", "mp4", "pipe:1"}
	cmd := exec.CommandContext(ctx, "/proc/self/fd/3", args...)
	cmd.Env = []string{"LANG=C", "LC_ALL=C"}
	cmd.ExtraFiles = []*os.File{tool}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		_ = file.Close()
		_ = root.Remove(tmp)
		return nil, err
	}
	limit := &videoFileLimit{file: file, limit: videoMaxOutputBytes}
	var stderr boundedInspectionBuffer
	stderr.limit = videoMaxStderr
	cmd.Stdout = limit
	cmd.Stderr = &stderr
	if err = cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = file.Close()
		_ = root.Remove(tmp)
		return nil, err
	}
	return &videoEncoder{cmd: cmd, stdin: stdin, file: file, root: root, temporary: tmp, outputName: outputName, limit: limit, stderr: &stderr}, nil
}

func (e *videoEncoder) writeFrame(ctx context.Context, pixels []byte) error {
	if e == nil || e.finished {
		return errors.New("video encoder is not active")
	}
	for len(pixels) > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		chunk := pixels
		if len(chunk) > 64<<10 {
			chunk = chunk[:64<<10]
		}
		n, err := e.stdin.Write(chunk)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		pixels = pixels[n:]
	}
	return nil
}

func (e *videoEncoder) finish(ctx context.Context) (int64, string, error) {
	if e == nil || e.finished {
		return 0, "", errors.New("video encoder is not active")
	}
	e.finished = true
	if err := e.stdin.Close(); err != nil {
		_ = e.cmd.Process.Kill()
		_ = e.cmd.Wait()
		e.cleanup()
		return 0, "", err
	}
	err := e.cmd.Wait()
	e.waited = true
	if ctx.Err() != nil {
		e.cleanup()
		return 0, "", ctx.Err()
	}
	if err != nil {
		e.cleanup()
		return 0, "", fmt.Errorf("pinned MP4 encoder failed: %w: %s", err, truncateUTF8(e.stderr.data.String(), 512))
	}
	if e.limit.written <= 0 || e.limit.written > videoMaxOutputBytes {
		e.cleanup()
		return 0, "", errors.New("MP4 encoder produced invalid output size")
	}
	if err = e.file.Sync(); err != nil {
		e.cleanup()
		return 0, "", err
	}
	if err = e.file.Close(); err != nil {
		e.cleanup()
		return 0, "", err
	}
	f, err := e.root.Open(e.temporary)
	if err != nil {
		e.cleanup()
		return 0, "", err
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, videoMaxOutputBytes+1))
	closeErr := f.Close()
	if err != nil || closeErr != nil || n != e.limit.written {
		e.cleanup()
		return 0, "", errors.New("final MP4 file changed or has invalid length")
	}
	sum := hex.EncodeToString(h.Sum(nil))
	if err = e.root.Link(e.temporary, e.outputName); err != nil {
		e.cleanup()
		return 0, "", err
	}
	if err = e.root.Remove(e.temporary); err != nil {
		return 0, "", err
	}
	return n, sum, nil
}

func (e *videoEncoder) cleanup() {
	if e == nil {
		return
	}
	_ = e.stdin.Close()
	if e.cmd.Process != nil && !e.waited {
		_ = e.cmd.Process.Kill()
		_ = e.cmd.Wait()
		e.waited = true
	}
	_ = e.file.Close()
	_ = e.root.Remove(e.temporary)
	e.finished = true
}
func (e *videoEncoder) abort() { e.cleanup() }

type videoProbeDoc struct {
	Streams []struct {
		Index          int             `json:"index"`
		CodecName      string          `json:"codec_name"`
		Width          int             `json:"width"`
		Height         int             `json:"height"`
		PixelFormat    string          `json:"pix_fmt"`
		Rate           string          `json:"r_frame_rate"`
		Frames         json.RawMessage `json:"nb_read_frames"`
		TimeBase       string          `json:"time_base"`
		StartPTS       json.RawMessage `json:"start_pts"`
		ColorPrimaries string          `json:"color_primaries"`
		ColorTransfer  string          `json:"color_transfer"`
		ColorMatrix    string          `json:"color_space"`
		ColorRange     string          `json:"color_range"`
	} `json:"streams"`
	Format struct {
		Name string `json:"format_name"`
	} `json:"format"`
}

func verifyEncodedVideo(ctx context.Context, probe, ffmpeg *os.File, root *os.Root, m VideoManifest, expectedFrames int) (VideoNativeFacts, error) {
	return verifyEncodedVideoNamed(ctx, probe, ffmpeg, root, m, expectedFrames, VideoOutputName)
}

func verifyEncodedVideoNamed(ctx context.Context, probe, ffmpeg *os.File, root *os.Root, m VideoManifest, expectedFrames int, outputName string) (VideoNativeFacts, error) {
	var zero VideoNativeFacts
	if !isSafePhysicalName(outputName) || outputName == "" {
		return zero, errors.New("video output name is invalid")
	}
	out, err := root.Open(outputName)
	if err != nil {
		return zero, err
	}
	defer out.Close()
	runner := execInspectionRunner{}
	args := []string{"-v", "error", "-protocol_whitelist", "file,pipe", "-format_whitelist", "mov,matroska,webm", "-max_streams", "8", "-count_frames", "-show_entries", "stream=index,codec_name,width,height,pix_fmt,r_frame_rate,time_base,start_pts,nb_read_frames,color_primaries,color_transfer,color_space,color_range:format=format_name", "-show_streams", "-show_format", "-of", "json", inspectionInputDescriptor}
	data, err := runner.Probe(ctx, probe, args, out, inspectionMaxProbe)
	if err != nil {
		return zero, err
	}
	var doc videoProbeDoc
	if err = json.Unmarshal(data, &doc); err != nil {
		return zero, err
	}
	if len(doc.Streams) != 1 {
		return zero, errors.New("encoded video must contain exactly one stream")
	}
	s := doc.Streams[0]
	num, den, ok := parseRational(s.Rate)
	if !ok {
		return zero, errors.New("encoded video frame rate is unavailable")
	}
	timeNum, timeDen, ok := parseRational(s.TimeBase)
	startPTS, hasStart := parseRawInt(s.StartPTS)
	if !ok || !hasStart || startPTS != 0 {
		return zero, errors.New("encoded video time base or zero start PTS is invalid")
	}
	frames, hasFrames := parseRawInt(s.Frames)
	if !hasFrames || frames < 0 || frames != int64(expectedFrames) {
		return zero, errors.New("encoded video frame count differs from planned schedule")
	}
	if s.CodecName != "h264" || s.PixelFormat != "yuv420p" || s.Width != m.Width || s.Height != m.Height || doc.Format.Name != "mov,mp4,m4a,3gp,3g2,mj2" || s.ColorPrimaries != "bt709" || s.ColorTransfer != "bt709" || s.ColorMatrix != "bt709" || (s.ColorRange != "tv" && s.ColorRange != "mpeg") {
		return zero, errors.New("encoded video facts do not match pinned output profile")
	}
	if _, err = out.Seek(0, io.SeekStart); err != nil {
		return zero, err
	}
	decoded, err := runner.Decode(ctx, ffmpeg, []string{"-hide_banner", "-nostdin", "-v", "error", "-threads", "2", "-filter_threads", "2", "-protocol_whitelist", "file,pipe", "-format_whitelist", "mov,matroska,webm", "-max_streams", "8", "-xerror", "-err_detect", "explode", "-i", inspectionInputDescriptor, "-map", fmt.Sprintf("0:%d", s.Index), "-an", "-sn", "-dn", "-fps_mode", "passthrough", "-enc_time_base:v", fmt.Sprintf("%d/%d", timeNum, timeDen), "-stats_enc_pre", "pipe:1", "-stats_enc_pre_fmt", "{n} {pts} {tb}", "-f", "null", "-"}, out, videoMaxStats)
	if err != nil {
		return zero, fmt.Errorf("full output decode failed: %w", err)
	}
	want, err := plannedOutputPTS(m, composition.Rational{Num: timeNum, Den: timeDen}, expectedFrames)
	if err != nil {
		return zero, err
	}
	if err = compareDecodePTS(decoded, want, timeNum, timeDen); err != nil {
		return zero, fmt.Errorf("full output decode timestamps differ: %w", err)
	}
	return VideoNativeFacts{Container: doc.Format.Name, Codec: s.CodecName, PixelFormat: s.PixelFormat, Width: s.Width, Height: s.Height, FrameRate: composition.Rational{Num: num, Den: den}, TimeBase: composition.Rational{Num: timeNum, Den: timeDen}, FrameCount: uint32(frames), FirstPTS: want[0], LastPTS: want[len(want)-1], ColorPrimaries: s.ColorPrimaries, ColorTransfer: s.ColorTransfer, ColorMatrix: s.ColorMatrix, ColorRange: s.ColorRange, FullDecodeVerified: true}, nil
}
