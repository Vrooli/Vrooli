package ai

import (
	"encoding/json"
	"fmt"
	"image"
	"math"
	"sort"
	"strconv"
	"strings"

	"image-tools/internal/opparams"
)

// Pose charts are sheets of poses: a grid of figures on a page, or any other
// arrangement of panels. The split finds each figure from the page itself, so a
// 3x3 grid, a 2x5 strip and a scatter of poses all work without a layout being
// declared: ink that is closer to other ink than the gap parameter belongs to the
// same cell, everything else is white space between cells. A caller who knows the
// layout can pin it with `grid` ("3x3"), which only changes where the cuts are
// made, not how tightly each cell is trimmed to its drawing.

// PoseChartSchema names the document layout so a reader can reject a newer one.
const PoseChartSchema = "pose-chart-v1"

const (
	poseChartAnalysisEdge = 640    // longest side the page is analysed at
	poseChartInkDelta     = 40.0   // luminance distance from the page that counts as ink
	poseChartMinInk       = 0.0006 // smallest figure, as a share of the analysed page
	poseChartDefaultGap   = 0.02   // default gap between cells, as a share of the longest side
	poseChartDefaultPad   = 0.03   // default padding around each cell, as a share of its size
)

// PoseCell is one pose on the chart: its reading-order number (from 1), the row
// and column it sits in, and its box in the chart's own pixels.
type PoseCell struct {
	Number int    `json:"number"`
	Row    int    `json:"row"`
	Col    int    `json:"col"`
	Box    [4]int `json:"box"` // x0, y0, x1, y1
}

// PoseChart is the split's result document.
type PoseChart struct {
	Schema string `json:"schema"`
	Image  struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"image"`
	Method string     `json:"method"` // "figures" (found from the page) or "grid" (pinned by the caller)
	Rows   int        `json:"rows"`
	Cols   int        `json:"cols"`
	Cells  []PoseCell `json:"cells"`
}

// PoseChartParams are the knobs of the split.
type PoseChartParams struct {
	Grid    string  // "RxC" to pin the layout; empty to find the figures
	Gap     float64 // share of the longest side that separates two cells
	Padding float64 // share of a cell's size added around its drawing
}

func poseChartParams(p opparams.PoseChartSplitParams) (PoseChartParams, error) {
	out := PoseChartParams{Gap: poseChartDefaultGap, Padding: poseChartDefaultPad}
	if p.Grid != nil {
		out.Grid = strings.TrimSpace(*p.Grid)
	}
	if p.Gap != nil {
		out.Gap = *p.Gap
	}
	if p.Padding != nil {
		out.Padding = *p.Padding
	}
	return out, nil
}

// SplitPoseChart finds the cells of a chart and returns the JSON document.
func SplitPoseChart(src image.Image, p PoseChartParams) ([]byte, error) {
	chart, err := splitPoseChart(src, p)
	if err != nil {
		return nil, err
	}
	return json.Marshal(chart)
}

func splitPoseChart(src image.Image, p PoseChartParams) (PoseChart, error) {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	chart := PoseChart{Schema: PoseChartSchema, Method: "figures"}
	chart.Image.Width, chart.Image.Height = w, h
	ink := newInkMap(src, poseChartAnalysisEdge)
	var boxes []image.Rectangle // analysis coordinates
	if p.Grid != "" {
		rows, cols, err := parsePoseGrid(p.Grid)
		if err != nil {
			return chart, err
		}
		chart.Method = "grid"
		boxes = ink.gridCells(rows, cols)
	} else {
		boxes = ink.figures(p.Gap)
	}
	if len(boxes) == 0 {
		return chart, fmt.Errorf("pose_chart_split: the page is blank, there is no pose to cut out")
	}
	rows := arrangeRows(boxes)
	cells := make([]PoseCell, 0, len(boxes))
	for r, row := range rows {
		for c, box := range row {
			cells = append(cells, PoseCell{Number: len(cells) + 1, Row: r, Col: c, Box: ink.toPage(box, w, h, p.Padding)})
		}
	}
	chart.Cells, chart.Rows = cells, len(rows)
	for _, row := range rows {
		chart.Cols = max(chart.Cols, len(row))
	}
	return chart, nil
}

func parsePoseGrid(spec string) (rows, cols int, err error) {
	parts := strings.Split(strings.ToLower(spec), "x")
	if len(parts) == 2 {
		rows, err = strconv.Atoi(strings.TrimSpace(parts[0]))
		if err == nil {
			cols, err = strconv.Atoi(strings.TrimSpace(parts[1]))
		}
	} else {
		err = fmt.Errorf("want ROWSxCOLS")
	}
	if err != nil || rows < 1 || cols < 1 || rows > 12 || cols > 12 {
		return 0, 0, fmt.Errorf("pose_chart_split: grid %q must be ROWSxCOLS between 1x1 and 12x12", spec)
	}
	return rows, cols, nil
}

// inkMap is the page reduced to a coarse "is there drawing here" grid. Each cell
// of the grid keeps the strongest difference from the page colour in the block of
// pixels it covers, so a one-pixel outline survives the reduction.
type inkMap struct {
	w, h  int
	scale float64 // page pixels per analysis pixel
	on    []bool
}

func newInkMap(src image.Image, edge int) inkMap {
	b := src.Bounds()
	pw, ph := b.Dx(), b.Dy()
	scale := math.Max(1, float64(max(pw, ph))/float64(edge))
	w, h := max(1, int(math.Ceil(float64(pw)/scale))), max(1, int(math.Ceil(float64(ph)/scale)))
	lum := func(x, y int) float64 {
		r, g, bl, _ := src.At(b.Min.X+x, b.Min.Y+y).RGBA()
		return (0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(bl>>8))
	}
	page := pageLuminance(pw, ph, lum)
	m := inkMap{w: w, h: h, scale: scale, on: make([]bool, w*h)}
	for ay := 0; ay < h; ay++ {
		for ax := 0; ax < w; ax++ {
			x0, y0 := int(float64(ax)*scale), int(float64(ay)*scale)
			x1, y1 := min(pw, int(float64(ax+1)*scale)), min(ph, int(float64(ay+1)*scale))
			for y := y0; y < y1 && !m.on[ay*w+ax]; y++ {
				for x := x0; x < x1; x++ {
					if math.Abs(lum(x, y)-page) > poseChartInkDelta {
						m.on[ay*w+ax] = true
						break
					}
				}
			}
		}
	}
	return m
}

// pageLuminance is the median luminance of the page border: the colour the
// figures are drawn on.
func pageLuminance(w, h int, lum func(x, y int) float64) float64 {
	var edge []float64
	for x := 0; x < w; x++ {
		edge = append(edge, lum(x, 0), lum(x, h-1))
	}
	for y := 0; y < h; y++ {
		edge = append(edge, lum(0, y), lum(w-1, y))
	}
	sort.Float64s(edge)
	return edge[len(edge)/2]
}

// figures returns one box per drawing: ink within `gap` of other ink is joined,
// each joined group is a cell, specks are dropped and a group lying wholly inside
// another (a hand beside a torso, a label in a frame) folds into it.
func (m inkMap) figures(gap float64) []image.Rectangle {
	radius := max(1, int(math.Round(gap*float64(max(m.w, m.h))/2)))
	joined := m.dilate(radius)
	labels := make([]int, m.w*m.h)
	var boxes []image.Rectangle
	var count []int
	for start := range joined {
		if !joined[start] || labels[start] != 0 {
			continue
		}
		id := len(boxes) + 1
		var box image.Rectangle
		n := 0
		stack := []int{start}
		labels[start] = id
		for len(stack) > 0 {
			i := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			x, y := i%m.w, i/m.w
			if m.on[i] {
				n++
				box = box.Union(image.Rect(x, y, x+1, y+1)) // an empty seed adopts the first pixel
			}
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := x+d[0], y+d[1]
				if nx < 0 || ny < 0 || nx >= m.w || ny >= m.h {
					continue
				}
				j := ny*m.w + nx
				if joined[j] && labels[j] == 0 {
					labels[j] = id
					stack = append(stack, j)
				}
			}
		}
		boxes = append(boxes, box)
		count = append(count, n)
	}
	floor := int(math.Ceil(poseChartMinInk * float64(m.w*m.h)))
	var kept []image.Rectangle
	for i, box := range boxes {
		if count[i] >= floor && !box.Empty() {
			kept = append(kept, box)
		}
	}
	return foldContained(kept)
}

// dilate grows the ink by `radius` analysis pixels (a square structuring element,
// done as two separable passes).
func (m inkMap) dilate(radius int) []bool {
	pass := func(in []bool, horizontal bool) []bool {
		out := make([]bool, len(in))
		for y := 0; y < m.h; y++ {
			for x := 0; x < m.w; x++ {
				if !in[y*m.w+x] {
					continue
				}
				for d := -radius; d <= radius; d++ {
					nx, ny := x, y
					if horizontal {
						nx += d
					} else {
						ny += d
					}
					if nx >= 0 && ny >= 0 && nx < m.w && ny < m.h {
						out[ny*m.w+nx] = true
					}
				}
			}
		}
		return out
	}
	return pass(pass(m.on, true), false)
}

func foldContained(boxes []image.Rectangle) []image.Rectangle {
	var out []image.Rectangle
	for i, a := range boxes {
		inside := false
		for j, b := range boxes {
			if i != j && a.In(b) && (a != b || i > j) {
				inside = true
				break
			}
		}
		if !inside {
			out = append(out, a)
		}
	}
	return out
}

// gridCells cuts the page into rows x cols equal bands and trims each to the
// drawing inside it; a band with no drawing is skipped.
func (m inkMap) gridCells(rows, cols int) []image.Rectangle {
	var out []image.Rectangle
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			band := image.Rect(c*m.w/cols, r*m.h/rows, (c+1)*m.w/cols, (r+1)*m.h/rows)
			var box image.Rectangle
			found := false
			for y := band.Min.Y; y < band.Max.Y; y++ {
				for x := band.Min.X; x < band.Max.X; x++ {
					if m.on[y*m.w+x] {
						box, found = box.Union(image.Rect(x, y, x+1, y+1)), true
					}
				}
			}
			if found {
				out = append(out, box)
			}
		}
	}
	return out
}

// arrangeRows orders cells the way a person reads a page. A cell starts a new row
// when its centre lies below the bottom of the shortest cell in the current row,
// so tall and short figures side by side stay in one row.
func arrangeRows(boxes []image.Rectangle) [][]image.Rectangle {
	order := append([]image.Rectangle(nil), boxes...)
	centre := func(r image.Rectangle) int { return (r.Min.Y + r.Max.Y) / 2 }
	sort.SliceStable(order, func(i, j int) bool { return centre(order[i]) < centre(order[j]) })
	var rows [][]image.Rectangle
	var current []image.Rectangle
	shortest := math.MaxInt
	for _, box := range order {
		if len(current) > 0 && centre(box) > shortest {
			rows, current, shortest = append(rows, current), nil, math.MaxInt
		}
		current = append(current, box)
		shortest = min(shortest, box.Max.Y)
	}
	if len(current) > 0 {
		rows = append(rows, current)
	}
	for _, row := range rows {
		sort.SliceStable(row, func(i, j int) bool { return row[i].Min.X < row[j].Min.X })
	}
	return rows
}

// toPage maps an analysis box to page pixels and pads it by a share of its size.
func (m inkMap) toPage(box image.Rectangle, pw, ph int, padding float64) [4]int {
	x0, y0 := float64(box.Min.X)*m.scale, float64(box.Min.Y)*m.scale
	x1, y1 := float64(box.Max.X)*m.scale, float64(box.Max.Y)*m.scale
	padX, padY := (x1-x0)*padding, (y1-y0)*padding
	return [4]int{
		max(0, int(math.Floor(x0-padX))), max(0, int(math.Floor(y0-padY))),
		min(pw, int(math.Ceil(x1+padX))), min(ph, int(math.Ceil(y1+padY))),
	}
}
