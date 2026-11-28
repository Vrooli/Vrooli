package ai

import (
	"encoding/json"
	"image"
	"image/color"
	"testing"
)

// chartPage is a white page with figures drawn as outlined boxes (a torso with
// limbs reaching out is built from several strokes, so the figure is several
// separate-but-near blobs, like a real mannequin).
func chartPage(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := range img.Pix {
		img.Pix[i] = 255
	}
	return img
}

func stroke(img *image.NRGBA, r image.Rectangle) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			img.SetNRGBA(x, y, color.NRGBA{A: 255})
		}
	}
}

// figure draws a stick-like pose inside box: a torso, two arms that stop short of
// the torso (a hand's width of white between them) and two legs joined by a hip bar.
func figure(img *image.NRGBA, box image.Rectangle) {
	cx := (box.Min.X + box.Max.X) / 2
	stroke(img, image.Rect(cx-3, box.Min.Y, cx+3, box.Max.Y-box.Dy()/3))
	stroke(img, image.Rect(box.Min.X, box.Min.Y+box.Dy()/5, cx-12, box.Min.Y+box.Dy()/5+5))
	stroke(img, image.Rect(cx+12, box.Min.Y+box.Dy()/5, box.Max.X, box.Min.Y+box.Dy()/5+5))
	hip := box.Max.Y - box.Dy()/3
	stroke(img, image.Rect(cx-box.Dx()/4, hip, cx+box.Dx()/4+5, hip+5)) // a hip bar joins the legs to the torso
	stroke(img, image.Rect(cx-box.Dx()/4, box.Max.Y-box.Dy()/3, cx-box.Dx()/4+5, box.Max.Y))
	stroke(img, image.Rect(cx+box.Dx()/4, box.Max.Y-box.Dy()/3, cx+box.Dx()/4+5, box.Max.Y))
}

func split(t *testing.T, img image.Image, p PoseChartParams) PoseChart {
	t.Helper()
	raw, err := SplitPoseChart(img, p)
	if err != nil {
		t.Fatalf("SplitPoseChart: %v", err)
	}
	var chart PoseChart
	if err := json.Unmarshal(raw, &chart); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return chart
}

func defaultChartParams() PoseChartParams {
	return PoseChartParams{Gap: poseChartDefaultGap, Padding: poseChartDefaultPad}
}

func inside(cell PoseCell, truth image.Rectangle) bool {
	return cell.Box[0] <= truth.Min.X && cell.Box[1] <= truth.Min.Y && cell.Box[2] >= truth.Max.X && cell.Box[3] >= truth.Max.Y
}

// TestSplitFindsAnyArrangement is the contract: figures laid out in a grid of any
// shape, or scattered, come back one cell each, in reading order, with the whole
// drawing inside the cell and none of a neighbour's.
func TestSplitFindsAnyArrangement(t *testing.T) {
	cases := []struct {
		name  string
		truth []image.Rectangle // expected reading order
		rows  int
		cols  int
	}{
		{"three by two", []image.Rectangle{
			image.Rect(60, 40, 260, 400), image.Rect(420, 50, 640, 420), image.Rect(800, 40, 1000, 390),
			image.Rect(70, 600, 270, 980), image.Rect(430, 610, 650, 990), image.Rect(800, 600, 1000, 970)}, 2, 3},
		{"single pose", []image.Rectangle{image.Rect(300, 100, 700, 900)}, 1, 1},
		{"one row of four", []image.Rectangle{
			image.Rect(40, 100, 240, 500), image.Rect(330, 120, 520, 520), image.Rect(620, 90, 820, 480), image.Rect(910, 110, 1100, 510)}, 1, 4},
		{"staggered", []image.Rectangle{
			image.Rect(50, 50, 250, 450), image.Rect(450, 250, 650, 650), image.Rect(850, 60, 1050, 460),
			image.Rect(120, 760, 320, 1100)}, 3, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var w, h int
			for _, r := range tc.truth {
				w, h = max(w, r.Max.X+60), max(h, r.Max.Y+60)
			}
			page := chartPage(w, h)
			for _, r := range tc.truth {
				figure(page, r)
			}
			chart := split(t, page, defaultChartParams())
			if len(chart.Cells) != len(tc.truth) {
				t.Fatalf("found %d cells, want %d: %+v", len(chart.Cells), len(tc.truth), chart.Cells)
			}
			for i, cell := range chart.Cells {
				if cell.Number != i+1 {
					t.Errorf("cell %d numbered %d", i, cell.Number)
				}
				if !inside(cell, tc.truth[i]) {
					t.Errorf("cell %d box %v does not hold its figure %v", i+1, cell.Box, tc.truth[i])
				}
				for j, other := range tc.truth {
					if j != i && (cell.Box[0] < other.Max.X && cell.Box[2] > other.Min.X && cell.Box[1] < other.Max.Y && cell.Box[3] > other.Min.Y) {
						t.Errorf("cell %d box %v reaches figure %d at %v", i+1, cell.Box, j+1, other)
					}
				}
			}
			if chart.Method != "figures" || chart.Image.Width != w || chart.Image.Height != h {
				t.Errorf("document header wrong: %+v", chart)
			}
		})
	}
}

func TestSplitReadingOrderRowsAndCols(t *testing.T) {
	page := chartPage(1100, 1100)
	truth := []image.Rectangle{
		image.Rect(60, 40, 260, 400), image.Rect(420, 50, 640, 420), image.Rect(800, 40, 1000, 390),
		image.Rect(70, 600, 270, 980), image.Rect(430, 610, 650, 990), image.Rect(800, 600, 1000, 970)}
	for _, r := range truth {
		figure(page, r)
	}
	chart := split(t, page, defaultChartParams())
	if chart.Rows != 2 || chart.Cols != 3 {
		t.Fatalf("layout %dx%d, want 2x3", chart.Rows, chart.Cols)
	}
	for i, cell := range chart.Cells {
		if cell.Row != i/3 || cell.Col != i%3 {
			t.Errorf("cell %d at row %d col %d, want %d,%d", i+1, cell.Row, cell.Col, i/3, i%3)
		}
	}
}

func TestSplitIgnoresSpecksAndFoldsContainedMarks(t *testing.T) {
	page := chartPage(1000, 700)
	figure(page, image.Rect(100, 50, 400, 600))
	stroke(page, image.Rect(700, 30, 704, 34))   // dust
	stroke(page, image.Rect(240, 300, 262, 322)) // a mark inside the figure's own box but out of reach of its strokes
	chart := split(t, page, defaultChartParams())
	if len(chart.Cells) != 1 {
		t.Fatalf("specks and contained marks made %d cells: %+v", len(chart.Cells), chart.Cells)
	}
}

func TestSplitPinnedGridTrimsEachBandToItsDrawing(t *testing.T) {
	page := chartPage(1000, 600)
	// Two figures touching in the middle would join on their own.
	figure(page, image.Rect(100, 50, 520, 550))
	figure(page, image.Rect(480, 60, 900, 540))
	free := split(t, page, defaultChartParams())
	if len(free.Cells) != 1 {
		t.Fatalf("touching figures should join without a grid, got %d cells", len(free.Cells))
	}
	pinned := split(t, page, PoseChartParams{Grid: "1x2", Gap: poseChartDefaultGap, Padding: 0})
	if pinned.Method != "grid" || len(pinned.Cells) != 2 {
		t.Fatalf("pinned grid gave %s with %d cells", pinned.Method, len(pinned.Cells))
	}
	if pinned.Cells[0].Box[2] > 520 || pinned.Cells[1].Box[0] < 470 {
		t.Errorf("bands overlap past the cut: %v %v", pinned.Cells[0].Box, pinned.Cells[1].Box)
	}
}

func TestSplitRefusesBlankPagesAndBadGrids(t *testing.T) {
	if _, err := SplitPoseChart(chartPage(400, 400), defaultChartParams()); err == nil {
		t.Error("a blank page should be refused")
	}
	page := chartPage(400, 400)
	figure(page, image.Rect(100, 50, 300, 350))
	for _, grid := range []string{"3", "0x3", "axb", "13x2", "2x2x2"} {
		if _, err := SplitPoseChart(page, PoseChartParams{Grid: grid, Gap: poseChartDefaultGap}); err == nil {
			t.Errorf("grid %q should be refused", grid)
		}
	}
}

func TestSplitWorksOnDarkPages(t *testing.T) {
	page := image.NewNRGBA(image.Rect(0, 0, 800, 600))
	for i := range page.Pix {
		page.Pix[i] = 20
		if i%4 == 3 {
			page.Pix[i] = 255
		}
	}
	for _, r := range []image.Rectangle{image.Rect(60, 50, 260, 520), image.Rect(450, 60, 700, 540)} {
		cx := (r.Min.X + r.Max.X) / 2
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := cx - 3; x < cx+3; x++ {
				page.SetNRGBA(x, y, color.NRGBA{R: 240, G: 240, B: 240, A: 255})
			}
		}
		for x := r.Min.X; x < r.Max.X; x++ {
			page.SetNRGBA(x, r.Min.Y+80, color.NRGBA{R: 240, G: 240, B: 240, A: 255})
		}
	}
	if chart := split(t, page, defaultChartParams()); len(chart.Cells) != 2 {
		t.Fatalf("dark page gave %d cells, want 2", len(chart.Cells))
	}
}
