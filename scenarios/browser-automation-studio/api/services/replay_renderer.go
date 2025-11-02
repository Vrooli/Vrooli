package services

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/browserless/runtime"
)

// RenderedMedia represents a generated media artifact ready for download.
type RenderedMedia struct {
	Path        string
	Filename    string
	ContentType string

	cleanup func()
}

// Cleanup removes any temporary artifacts associated with the rendered media.
func (m *RenderedMedia) Cleanup() {
	if m == nil || m.cleanup == nil {
		return
	}
	m.cleanup()
}

// ReplayRenderer renders replay export packages to video or gif assets using ffmpeg.
type ReplayRenderer struct {
	log            *logrus.Logger
	recordingsRoot string
	ffmpegPath     string
	httpClient     *http.Client
}

// RenderFormat enumerates supported render formats.
type RenderFormat string

const (
	// RenderFormatMP4 renders replay as MP4 video.
	RenderFormatMP4 RenderFormat = "mp4"
	// RenderFormatGIF renders replay as animated GIF.
	RenderFormatGIF RenderFormat = "gif"

	minCursorBoxSize       = 8
	defaultCursorBoxFactor = 0.02
	rendererDefaultAccent  = "#38BDF8"
)

// NewReplayRenderer constructs a replay renderer with sane defaults.
func NewReplayRenderer(log *logrus.Logger, recordingsRoot string) *ReplayRenderer {
	ffmpegPath := detectFFmpegBinary()
	client := &http.Client{Timeout: 45 * time.Second}
	return &ReplayRenderer{
		log:            log,
		recordingsRoot: recordingsRoot,
		ffmpegPath:     ffmpegPath,
		httpClient:     client,
	}
}

func detectFFmpegBinary() string {
	if custom := strings.TrimSpace(os.Getenv("FFMPEG_BIN")); custom != "" {
		return custom
	}
	if _, err := exec.LookPath("ffmpeg"); err == nil {
		return "ffmpeg"
	}
	defaultPath := filepath.Join(os.Getenv("HOME"), ".ffmpeg", "bin", "ffmpeg")
	return defaultPath
}

// Render creates a media artifact for the provided export package.
func (r *ReplayRenderer) Render(ctx context.Context, pkg *ReplayExportPackage, format RenderFormat, filename string) (*RenderedMedia, error) {
	if pkg == nil {
		return nil, errors.New("nil replay export package")
	}
	if len(pkg.Frames) == 0 {
		return nil, errors.New("export package missing frames")
	}
	if format != RenderFormatMP4 && format != RenderFormatGIF {
		return nil, fmt.Errorf("unsupported render format %q", format)
	}

	tempRoot, err := os.MkdirTemp("", "bas-render-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	cleanup := func() {
		_ = os.RemoveAll(tempRoot)
	}

	assetsDir := filepath.Join(tempRoot, "assets")
	framesDir := filepath.Join(tempRoot, "frames")
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		cleanup()
		return nil, fmt.Errorf("failed to create assets directory: %w", err)
	}
	if err := os.MkdirAll(framesDir, 0o755); err != nil {
		cleanup()
		return nil, fmt.Errorf("failed to create frames directory: %w", err)
	}

	assetPaths, err := r.materializeAssets(ctx, pkg, assetsDir)
	if err != nil {
		cleanup()
		return nil, err
	}

	segments, err := r.prepareFrameSegments(ctx, pkg, framesDir, assetPaths)
	if err != nil {
		cleanup()
		return nil, err
	}

	concatPath := filepath.Join(tempRoot, "frames.txt")
	if err := writeConcatManifest(concatPath, segments); err != nil {
		cleanup()
		return nil, err
	}

	baseVideoPath := filepath.Join(tempRoot, "replay.mp4")
	if err := r.assembleVideo(ctx, concatPath, baseVideoPath); err != nil {
		cleanup()
		return nil, err
	}

	finalPath := baseVideoPath
	contentType := "video/mp4"
	if format == RenderFormatGIF {
		gifPath := filepath.Join(tempRoot, "replay.gif")
		if err := r.convertToGIF(ctx, baseVideoPath, gifPath); err != nil {
			cleanup()
			return nil, err
		}
		finalPath = gifPath
		contentType = "image/gif"
	}

	if filename == "" {
		filename = defaultFilename(pkg, string(format))
	}
	filename = sanitizeFilename(filename)

	return &RenderedMedia{
		Path:        finalPath,
		Filename:    filename,
		ContentType: contentType,
		cleanup:     cleanup,
	}, nil
}

func sanitizeFilename(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "browser-automation-replay"
	}
	trimmed = strings.ReplaceAll(trimmed, string(os.PathSeparator), "-")
	trimmed = strings.ReplaceAll(trimmed, "\x00", "")
	return trimmed
}

func defaultFilename(pkg *ReplayExportPackage, extension string) string {
	stem := "browser-automation-replay"
	if pkg != nil {
		execID := pkg.Execution.ExecutionID
		if execID != uuid.Nil {
			stem = fmt.Sprintf("browser-automation-replay-%s", execID.String()[:8])
		}
	}
	return fmt.Sprintf("%s.%s", stem, extension)
}

func (r *ReplayRenderer) materializeAssets(ctx context.Context, pkg *ReplayExportPackage, destDir string) (map[string]string, error) {
	paths := make(map[string]string, len(pkg.Assets))
	for _, asset := range pkg.Assets {
		if asset.ID == "" {
			continue
		}
		localPath, err := r.resolveAsset(ctx, pkg.Execution.ExecutionID, &asset, destDir)
		if err != nil {
			if r.log != nil {
				r.log.WithError(err).WithField("asset_id", asset.ID).Warn("Failed to materialize export asset")
			}
			continue
		}
		paths[asset.ID] = localPath
	}
	return paths, nil
}

func (r *ReplayRenderer) resolveAsset(ctx context.Context, executionID uuid.UUID, asset *ExportAsset, destDir string) (string, error) {
	if asset == nil || strings.TrimSpace(asset.Source) == "" {
		return "", errors.New("empty asset source")
	}

	// Attempt local resolution via recordings root first.
	if local, ok := r.mapRecordingAsset(asset.Source, executionID); ok {
		if fileExists(local) {
			return local, nil
		}
	}

	source := strings.TrimSpace(asset.Source)
	if strings.HasPrefix(source, "file://") {
		candidate := strings.TrimPrefix(source, "file://")
		if fileExists(candidate) {
			return candidate, nil
		}
	}
	if filepath.IsAbs(source) && fileExists(source) {
		return source, nil
	}

	// Remote fetch fallback.
	resolved, err := r.downloadAsset(ctx, source, destDir, asset.ID)
	if err != nil {
		return "", err
	}
	return resolved, nil
}

func (r *ReplayRenderer) mapRecordingAsset(source string, executionID uuid.UUID) (string, bool) {
	trimmed := source
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		parsed, err := url.Parse(trimmed)
		if err != nil {
			return "", false
		}
		trimmed = parsed.Path
	}
	trimmed = strings.TrimPrefix(trimmed, "/")
	if !strings.HasPrefix(trimmed, "api/v1/recordings/assets/") {
		return "", false
	}
	remainder := strings.TrimPrefix(trimmed, "api/v1/recordings/assets/")
	parts := strings.SplitN(remainder, "/", 2)
	if len(parts) != 2 {
		return "", false
	}
	if executionID != uuid.Nil && parts[0] != executionID.String() {
		return "", false
	}
	candidate := filepath.Join(r.recordingsRoot, parts[0], parts[1])
	return candidate, true
}

func (r *ReplayRenderer) downloadAsset(ctx context.Context, source, destDir, assetID string) (string, error) {
	parsed, err := url.Parse(source)
	if err != nil {
		return "", fmt.Errorf("invalid asset url %q: %w", source, err)
	}
	filename := sanitizeFilename(assetID)
	ext := path.Ext(parsed.Path)
	if ext == "" {
		ext = inferExtension(parsed.Path)
	}
	if ext == "" {
		ext = ".bin"
	}
	if filename == "" {
		filename = "asset"
	}
	finalName := fmt.Sprintf("%s%s", filename, ext)
	localPath := filepath.Join(destDir, finalName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request for asset download: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download asset %q: %w", parsed.String(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("asset download returned status %d", resp.StatusCode)
	}

	file, err := os.Create(localPath)
	if err != nil {
		return "", fmt.Errorf("failed to create asset file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return "", fmt.Errorf("failed to persist asset: %w", err)
	}

	return localPath, nil
}

func inferExtension(pathname string) string {
	switch strings.ToLower(filepath.Ext(pathname)) {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif":
		return filepath.Ext(pathname)
	}
	return ""
}

func fileExists(candidate string) bool {
	info, err := os.Stat(candidate)
	return err == nil && !info.IsDir()
}

type renderedSegment struct {
	Path       string
	DurationMs int
}

func (r *ReplayRenderer) prepareFrameSegments(ctx context.Context, pkg *ReplayExportPackage, framesDir string, assets map[string]string) ([]renderedSegment, error) {
	segments := make([]renderedSegment, 0, len(pkg.Frames))

	for frameIndex, frame := range pkg.Frames {
		screenshotPath := ""
		if frame.ScreenshotAssetID != "" {
			if candidate, ok := assets[frame.ScreenshotAssetID]; ok {
				screenshotPath = candidate
			}
		}
		if screenshotPath == "" {
			continue
		}

		viewport := frame.Viewport
		if viewport.Width <= 0 {
			viewport.Width = fallbackViewportWidth
		}
		if viewport.Height <= 0 {
			viewport.Height = fallbackViewportHeight
		}

		cursorSegments := buildCursorSegments(frame, viewport)
		if len(cursorSegments) == 0 {
			_, _, total := frameDurations(frame)
			cursorSegments = []cursorSegment{{DurationMs: total}}
		}

		frameBase := fmt.Sprintf("frame-%04d", frameIndex)
		for segIndex, seg := range cursorSegments {
			segmentName := frameBase
			if len(cursorSegments) > 1 {
				segmentName = fmt.Sprintf("%s-%03d", frameBase, segIndex)
			}
			segmentPath := filepath.Join(framesDir, segmentName+".png")

			filter := buildFrameFilter(frame, viewport, seg, pkg.Cursor)
			if err := r.renderSegment(ctx, screenshotPath, segmentPath, filter); err != nil {
				return nil, err
			}

			segments = append(segments, renderedSegment{Path: segmentPath, DurationMs: intMax(seg.DurationMs, 40)})
		}
	}

	if len(segments) == 0 {
		return nil, errors.New("no frames available for rendering")
	}

	return segments, nil
}

func (r *ReplayRenderer) renderSegment(ctx context.Context, inputPath, outputPath, filter string) error {
	args := []string{"-y", "-i", inputPath}
	if filter != "" {
		args = append(args, "-vf", filter)
	}
	args = append(args, "-frames:v", "1", outputPath)

	cmd := exec.CommandContext(ctx, r.ffmpegPath, args...)
	var stderr bytes.Buffer
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg segment render failed: %w (%s)", err, stderr.String())
	}
	return nil
}

func writeConcatManifest(manifestPath string, segments []renderedSegment) error {
	file, err := os.Create(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to create concat manifest: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, segment := range segments {
		if _, err := fmt.Fprintf(writer, "file '%s'\n", escapeFFmpegPath(segment.Path)); err != nil {
			return err
		}
		durationSeconds := math.Max(float64(segment.DurationMs)/1000.0, 0.1)
		if _, err := fmt.Fprintf(writer, "duration %.3f\n", durationSeconds); err != nil {
			return err
		}
	}
	// Repeat final frame to satisfy concat demuxer
	last := segments[len(segments)-1]
	if _, err := fmt.Fprintf(writer, "file '%s'\n", escapeFFmpegPath(last.Path)); err != nil {
		return err
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush concat manifest: %w", err)
	}
	return nil
}

func escapeFFmpegPath(value string) string {
	return strings.ReplaceAll(value, "'", "'\\''")
}

func (r *ReplayRenderer) assembleVideo(ctx context.Context, manifestPath, outputPath string) error {
	args := []string{
		"-y",
		"-f", "concat",
		"-safe", "0",
		"-i", manifestPath,
		"-vsync", "vfr",
		"-pix_fmt", "yuv420p",
		"-c:v", "libx264",
		"-profile:v", "high",
		"-level", "4.1",
		"-crf", "21",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, r.ffmpegPath, args...)
	var stderr bytes.Buffer
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg concat failed: %w (%s)", err, stderr.String())
	}
	return nil
}

func (r *ReplayRenderer) convertToGIF(ctx context.Context, inputPath, outputPath string) error {
	args := []string{
		"-y",
		"-i", inputPath,
		"-vf", "fps=12,scale=1280:-1:flags=lanczos",
		outputPath,
	}
	cmd := exec.CommandContext(ctx, r.ffmpegPath, args...)
	var stderr bytes.Buffer
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg gif conversion failed: %w (%s)", err, stderr.String())
	}
	return nil
}

type cursorSegment struct {
	CursorPoint *runtime.Point
	DurationMs  int
	DrawTrail   bool
}

func buildCursorSegments(frame ExportFrame, viewport ExportDimensions) []cursorSegment {
	trail := sanitizeTrail(frame.CursorTrail, viewport)
	motion, hold, total := frameDurations(frame)

	if len(trail) <= 1 {
		return []cursorSegment{{
			CursorPoint: finalPoint(trail, frame, viewport),
			DurationMs:  total,
			DrawTrail:   len(trail) > 0,
		}}
	}

	motionSegments := clampInt(len(trail)*6, 8, 45)
	perMotion := motion / motionSegments
	segments := make([]cursorSegment, 0, motionSegments+4)

	for i := 0; i < motionSegments; i++ {
		ratio := 1.0
		if motionSegments > 1 {
			ratio = float64(i) / float64(motionSegments-1)
		}
		point := interpolateTrail(trail, ratio)
		segments = append(segments, cursorSegment{
			CursorPoint: point,
			DurationMs:  intMax(perMotion, 40),
			DrawTrail:   true,
		})
	}

	if hold > 0 {
		holdSegments := clampInt(hold/200, 1, 8)
		perHold := hold / holdSegments
		final := trail[len(trail)-1]
		for i := 0; i < holdSegments; i++ {
			finalCopy := final
			segments = append(segments, cursorSegment{
				CursorPoint: &finalCopy,
				DurationMs:  intMax(perHold, 40),
				DrawTrail:   i == 0,
			})
		}
	}

	return segments
}

func frameDurations(frame ExportFrame) (int, int, int) {
	raw := frame.DurationMs
	if raw <= 0 {
		raw = frame.SummaryDuration()
	}
	if raw <= 0 {
		raw = 800
	}
	motion := intMax(raw, 400)
	hold := frame.HoldMs
	if hold <= 0 {
		hold = 650
	}
	return motion, hold, motion + hold
}

// SummaryDuration returns total frame duration if available.
func (f ExportFrame) SummaryDuration() int {
	return intMax(f.DurationMs, 0)
}

func sanitizeTrail(points []runtime.Point, viewport ExportDimensions) []runtime.Point {
	clamped := make([]runtime.Point, 0, len(points))
	for _, point := range points {
		if math.IsNaN(point.X) || math.IsInf(point.X, 0) || math.IsNaN(point.Y) || math.IsInf(point.Y, 0) {
			continue
		}
		clamped = append(clamped, runtime.Point{
			X: clampFloat(point.X, 0, float64(viewport.Width)),
			Y: clampFloat(point.Y, 0, float64(viewport.Height)),
		})
	}
	return clamped
}

func finalPoint(trail []runtime.Point, frame ExportFrame, viewport ExportDimensions) *runtime.Point {
	if len(trail) > 0 {
		last := trail[len(trail)-1]
		return &last
	}
	if frame.ClickPosition != nil {
		point := runtime.Point{
			X: clampFloat(frame.ClickPosition.X, 0, float64(viewport.Width)),
			Y: clampFloat(frame.ClickPosition.Y, 0, float64(viewport.Height)),
		}
		return &point
	}
	return nil
}

func interpolateTrail(trail []runtime.Point, ratio float64) *runtime.Point {
	if len(trail) == 0 {
		return nil
	}
	if len(trail) == 1 || ratio <= 0 {
		point := trail[0]
		return &point
	}
	if ratio >= 1 {
		point := trail[len(trail)-1]
		return &point
	}

	scaled := ratio * float64(len(trail)-1)
	index := int(math.Floor(scaled))
	segmentT := scaled - float64(index)
	start := trail[index]
	end := trail[index+1]

	lerp := func(a, b float64) float64 {
		if math.IsNaN(a) || math.IsInf(a, 0) {
			return b
		}
		if math.IsNaN(b) || math.IsInf(b, 0) {
			return a
		}
		return a + (b-a)*segmentT
	}

	point := runtime.Point{
		X: lerp(start.X, end.X),
		Y: lerp(start.Y, end.Y),
	}
	return &point
}


func buildFrameFilter(frame ExportFrame, viewport ExportDimensions, segment cursorSegment, cursorSpec ExportCursorSpec) string {
	filters := make([]string, 0, 16)

	accent := strings.TrimSpace(cursorSpec.AccentColor)
	if accent == "" {
		accent = rendererDefaultAccent
	}
	cursorScale := clampScale(cursorSpec.Scale)
	trailEnabled := cursorSpec.Trail.Enabled
	trailOpacity := cursorSpec.Trail.Opacity
	if trailOpacity <= 0 {
		trailOpacity = 0.55
	}
	trailWeight := cursorSpec.Trail.Weight
	if trailWeight <= 0 {
		trailWeight = 0.16
	}
	trailAlpha := opacityToAlpha(trailOpacity)
	clickEnabled := cursorSpec.ClickPulse.Enabled && !strings.EqualFold(cursorSpec.ClickAnim, "none")
	cursorVisible := !strings.EqualFold(cursorSpec.Style, "hidden") && !strings.EqualFold(cursorSpec.Style, "disabled")

	overlayBox := func(box *runtime.BoundingBox, color string, opacity float64, border *int) {
		if box == nil {
			return
		}
		x := clampFloat(box.X, 0, float64(viewport.Width))
		y := clampFloat(box.Y, 0, float64(viewport.Height))
		w := clampFloat(box.Width, 0, float64(viewport.Width))
		h := clampFloat(box.Height, 0, float64(viewport.Height))
		if w <= 0 || h <= 0 {
			return
		}
		alpha := opacityToAlpha(opacity)
		fillColor := hexToFFmpegColor(color, alpha)
		filters = append(filters, fmt.Sprintf("drawbox=x=%d:y=%d:w=%d:h=%d:color=%s:t=fill",
			int(math.Round(x)), int(math.Round(y)), int(math.Round(w)), int(math.Round(h)), fillColor,
		))
		if border != nil && *border > 0 {
			strokeColor := hexToFFmpegColor(color, "FF")
			filters = append(filters, fmt.Sprintf("drawbox=x=%d:y=%d:w=%d:h=%d:color=%s:t=%d",
				int(math.Round(x)), int(math.Round(y)), int(math.Round(w)), int(math.Round(h)), strokeColor, *border,
			))
		}
	}

	for _, region := range frame.MaskRegions {
		color := "#0F172A"
		opacity := region.Opacity
		if opacity <= 0 {
			opacity = 0.45
		}
		overlayBox(region.BoundingBox, color, opacity, nil)
	}

	for _, region := range frame.HighlightRegions {
		color := region.Color
		if strings.TrimSpace(color) == "" {
			color = "#38BDF8"
		}
		border := 3
		overlayBox(region.BoundingBox, color, 0.18, &border)
	}

	if frame.FocusedElement != nil && frame.FocusedElement.BoundingBox != nil {
		border := 4
		overlayBox(frame.FocusedElement.BoundingBox, "#38BDF8", 0, &border)
	}

	if clickEnabled && frame.ClickPosition != nil {
		size := cursorBoxSize(viewport, 0.02, cursorScale)
		x := clampFloat(frame.ClickPosition.X, 0, float64(viewport.Width)) - float64(size)/2
		y := clampFloat(frame.ClickPosition.Y, 0, float64(viewport.Height)) - float64(size)/2
		x = clampFloat(x, 0, float64(viewport.Width))
		y = clampFloat(y, 0, float64(viewport.Height))
		fill := hexToFFmpegColor("#FFFFFF", "AA")
		stroke := hexToFFmpegColor(accent, "FF")
		filters = append(filters,
			fmt.Sprintf("drawbox=x=%d:y=%d:w=%d:h=%d:color=%s:t=fill",
				int(math.Round(x)), int(math.Round(y)), size, size, fill),
		)
		filters = append(filters,
			fmt.Sprintf("drawbox=x=%d:y=%d:w=%d:h=%d:color=%s:t=2",
				int(math.Round(x)), int(math.Round(y)), size, size, stroke),
		)
	}

	if segment.DrawTrail && trailEnabled {
		trailColor := hexToFFmpegColor(accent, trailAlpha)
		trailSize := intMax(2, int(math.Round(4 * cursorScale * (0.8 + trailWeight*2))))
		for _, point := range frame.CursorTrail {
			x := clampFloat(point.X, 0, float64(viewport.Width))
			y := clampFloat(point.Y, 0, float64(viewport.Height))
			x -= float64(trailSize) / 2
			y -= float64(trailSize) / 2
			filters = append(filters, fmt.Sprintf("drawbox=x=%d:y=%d:w=%d:h=%d:color=%s:t=fill",
				int(math.Round(x)), int(math.Round(y)), trailSize, trailSize, trailColor,
			))
		}
	}

	if cursorVisible && segment.CursorPoint != nil {
		size := cursorBoxSize(viewport, defaultCursorBoxFactor, cursorScale)
		x := clampFloat(segment.CursorPoint.X, 0, float64(viewport.Width)) - float64(size)/2
		y := clampFloat(segment.CursorPoint.Y, 0, float64(viewport.Height)) - float64(size)/2
		x = clampFloat(x, 0, float64(viewport.Width))
		y = clampFloat(y, 0, float64(viewport.Height))
		fill := hexToFFmpegColor("#FFFFFF", "AA")
		stroke := hexToFFmpegColor(accent, "FF")
		filters = append(filters,
			fmt.Sprintf("drawbox=x=%d:y=%d:w=%d:h=%d:color=%s:t=fill",
				int(math.Round(x)), int(math.Round(y)), size, size, fill),
		)
		filters = append(filters,
			fmt.Sprintf("drawbox=x=%d:y=%d:w=%d:h=%d:color=%s:t=2",
				int(math.Round(x)), int(math.Round(y)), size, size, stroke),
		)
	}

	return strings.Join(filters, ",")
}

func cursorBoxSize(viewport ExportDimensions, factor float64, scale float64) int {
	minDimension := math.Min(float64(viewport.Width), float64(viewport.Height))
	if minDimension <= 0 {
		return minCursorBoxSize
	}
	clampedScale := clampScale(scale)
	size := int(math.Round(minDimension * factor * clampedScale))
	if size < minCursorBoxSize {
		size = minCursorBoxSize
	}
	return size
}

func clampScale(value float64) float64 {
	if value <= 0 {
		return 1.0
	}
	if value < 0.4 {
		return 0.4
	}
	if value > 2.4 {
		return 2.4
	}
	return value
}

func opacityToAlpha(opacity float64) string {
	if opacity <= 0 {
		return "00"
	}
	if opacity >= 1 {
		return "FF"
	}
	value := int(math.Round(opacity * 255))
	if value < 0 {
		value = 0
	}
	if value > 255 {
		value = 255
	}
	return fmt.Sprintf("%02X", value)
}

func hexToFFmpegColor(hex, alpha string) string {
	clean := strings.TrimPrefix(strings.TrimSpace(hex), "#")
	switch len(clean) {
	case 3:
		return fmt.Sprintf("0x%c%c%c%c%c%c%s",
			rune(clean[0]), rune(clean[0]),
			rune(clean[1]), rune(clean[1]),
			rune(clean[2]), rune(clean[2]),
			alpha,
		)
	case 6:
		return fmt.Sprintf("0x%s%s", strings.ToUpper(clean), strings.ToUpper(alpha))
	case 8:
		return fmt.Sprintf("0x%s", strings.ToUpper(clean))
	default:
		return fmt.Sprintf("0xFFFFFF%s", strings.ToUpper(alpha))
	}
}

func clampFloat(value, min, max float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return min
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func intMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}
