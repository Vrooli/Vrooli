package ai

import (
	"context"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"image-tools/internal/adapters"
	"image-tools/internal/models"
	"image-tools/internal/posemap"
)

// Operation parameters that say which pose a pose control reads from its source.
const (
	// ParamPoseCell is the 1-based number of the chart cell to read (see
	// pose_chart_split). Absent or 0 reads the whole source as one pose.
	ParamPoseCell = "pose_cell"
	// ParamPoseGrid pins the chart layout (ROWSxCOLS) the cell number refers to,
	// exactly as pose_chart_split's grid does.
	ParamPoseGrid = "pose_grid"
	// ParamPosePrepare is pose_extract's prepare mode.
	ParamPosePrepare = "pose_prepare"
)

// Generation canvas area for a pose control: the size the model family is trained
// for. SDXL-class controls draw at about a megapixel; everything older at 512².
const (
	poseCanvasAreaXL    = 1024 * 1024
	poseCanvasAreaSmall = 512 * 512
)

// derivePoseControl draws the OpenPose control image for the pose control adapter
// from its source file: the whole picture, or the chart cell the job's params name.
// It also settles the generation canvas: a job that gave no size is generated at
// the shape of the pose (a tall cell makes a tall image), so the control image and
// the picture it steers line up exactly.
func (e *Engine) derivePoseControl(ctx context.Context, tmpDir string, index int, adapter adapters.ResolvedAdapter, source string, params map[string]string) (string, error) {
	cell, err := poseCell(params)
	if err != nil {
		return "", err
	}
	var box *[4]float64
	var shape [2]float64 // width, height of what the control image shows
	if cell > 0 {
		b, err := poseChartCell(source, cell, params[ParamPoseGrid])
		if err != nil {
			return "", err
		}
		box, shape = &b, [2]float64{b[2] - b[0], b[3] - b[1]}
	} else {
		cfg, err := imageSize(source)
		if err != nil {
			return "", err
		}
		shape = [2]float64{float64(cfg.Width), float64(cfg.Height)}
	}
	if params == nil {
		return "", fmt.Errorf("pose control has no job parameters to record its canvas in")
	}
	width, height := givenSize(params)
	if width == 0 {
		area := poseCanvasAreaSmall
		if adapter.Architecture == models.ArchSDXL {
			area = poseCanvasAreaXL
		}
		width, height = posemap.Canvas(shape[0]/shape[1], area)
		params["width"], params["height"] = strconv.Itoa(width), strconv.Itoa(height)
	}
	output := filepath.Join(tmpDir, fmt.Sprintf("pose-map-%d.png", index))
	req := posemap.Request{Source: source, Output: output, Width: width, Height: height, Box: box, Prepare: params[ParamPosePrepare]}
	if err := e.deps.PosePreprocessor(ctx, req); err != nil {
		return "", err
	}
	if info, err := os.Stat(output); err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
		return "", fmt.Errorf("pose preprocessor produced no PNG")
	}
	return output, nil
}

func poseCell(params map[string]string) (int, error) {
	raw := strings.TrimSpace(params[ParamPoseCell])
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("%s %q must be a chart cell number (1, 2, ...)", ParamPoseCell, raw)
	}
	return n, nil
}

// poseChartCell is the box of cell `number` of the chart in the file at path.
func poseChartCell(path string, number int, grid string) ([4]float64, error) {
	f, err := os.Open(path)
	if err != nil {
		return [4]float64{}, err
	}
	defer func() { _ = f.Close() }()
	src, _, err := image.Decode(f)
	if err != nil {
		return [4]float64{}, fmt.Errorf("read pose chart: %w", err)
	}
	chart, err := splitPoseChart(src, PoseChartParams{Grid: strings.TrimSpace(grid), Gap: poseChartDefaultGap, Padding: poseChartDefaultPad})
	if err != nil {
		return [4]float64{}, err
	}
	for _, c := range chart.Cells {
		if c.Number == number {
			return [4]float64{float64(c.Box[0]), float64(c.Box[1]), float64(c.Box[2]), float64(c.Box[3])}, nil
		}
	}
	return [4]float64{}, fmt.Errorf("%s %d: the chart has %d cells", ParamPoseCell, number, len(chart.Cells))
}

func imageSize(path string) (image.Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return image.Config{}, err
	}
	defer func() { _ = f.Close() }()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return image.Config{}, fmt.Errorf("read pose source: %w", err)
	}
	return cfg, nil
}

// givenSize is the generation width and height the job asked for, or 0, 0 when it
// did not give both.
func givenSize(params map[string]string) (int, int) {
	w, errW := strconv.Atoi(params["width"])
	h, errH := strconv.Atoi(params["height"])
	if errW != nil || errH != nil || w < 1 || h < 1 || w > posemap.MaxCanvas || h > posemap.MaxCanvas {
		return 0, 0
	}
	return w, h
}
