package ops

import (
	"fmt"
	"image/color"
	"math"
	"sort"
	"strconv"
	"strings"
)

// Vectorize turns a flat raster mark into a clean SVG. It is the native-Go
// replacement for the ad-hoc OpenCV trace the 2026-09-15 Aquila session used:
// quantize to a small palette, build one mask per palette entry, drop the
// background layer, optionally clip to the largest figure, trace each mask's
// boundaries (outer contours and holes), simplify with Ramer–Douglas–Peucker and
// emit one evenodd path per layer.
//
// It reads in.Img (decoded by Execute) and ignores in.Bytes. Output Format is
// "svg"; the result is a vector, so enforcement of the "SVG is never a generic
// convert output" rule stays in the codec — only this op emits it.
func Vectorize(in RunInput) (RunResult, error) {
	b := in.Img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return RunResult{}, fmt.Errorf("%w: vectorize needs a non-empty image", ErrDecode)
	}
	px := make([]rgb, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.NRGBAModel.Convert(in.Img.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
			px[y*w+x] = rgb{c.R, c.G, c.B}
		}
	}

	p := in.Params
	tolerance := p.TolerancePx
	if tolerance <= 0 {
		tolerance = 0.8
	}
	minArea := p.MinAreaPx
	if minArea <= 0 {
		minArea = 40
	}

	palette, labels, err := quantize(px, p.Colors, p.KeepColors)
	if err != nil {
		return RunResult{}, err
	}
	if len(palette) == 0 {
		return RunResult{}, fmt.Errorf("vectorize: no layers survived quantization")
	}

	masks := make([][]bool, len(palette))
	for i := range masks {
		masks[i] = make([]bool, w*h)
	}
	for i, l := range labels {
		if l >= 0 {
			masks[l][i] = true
		}
	}

	// Drop the layer(s) that touch the image border: that is the enclosing tile
	// or page background, never part of the mark. When keep_colors pins the
	// palette the caller has already said which layers matter, so the border drop
	// is skipped — otherwise a white page and a white mark share one layer and
	// the whole layer would be lost.
	var dropped []int
	kept := make([]int, 0, len(masks))
	for i, m := range masks {
		if p.DropBackgroundLayers && len(p.KeepColors) == 0 && touchesBorder(m, w, h) {
			dropped = append(dropped, i)
			continue
		}
		kept = append(kept, i)
	}
	if len(kept) == 0 {
		return RunResult{}, fmt.Errorf("vectorize: every layer was dropped as background")
	}

	if p.ClipToLargestRoundedRegion {
		regionMask := clipRegionMask(labels, masks, kept, dropped, w, h)
		inset := p.InsetPx
		if inset <= 0 {
			inset = 0.02 * float64(minInt(w, h))
		}
		if region := regionFromHull(regionMask, w, h, inset); region != nil {
			for _, i := range kept {
				m := masks[i]
				for j := range m {
					if !region[j] {
						m[j] = false
					}
				}
			}
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d">`, w, h, w, h))
	// Recover the anti-aliased edge the hard ΔE threshold leaves as background:
	// grow each kept mask by one pixel so the traced stroke matches the source
	// width rather than the fully-saturated core.
	for _, i := range kept {
		masks[i] = dilate1(masks[i], w, h)
	}
	for _, i := range kept {
		contours := traceMask(masks[i], w, h)
		var d strings.Builder
		for _, c := range contours {
			simplified := simplifyClosed(c, tolerance)
			if polygonArea(simplified) < minArea {
				continue
			}
			writePath(&d, simplified)
		}
		if d.Len() == 0 {
			continue
		}
		// fill-rule is left at the default (nonzero) deliberately: the pure-Go
		// oksvg rasterizer ignores fill-rule="evenodd", so an evenodd path would
		// render its holes filled and the SVG and its rasters would drift. The
		// square-boundary tracer emits outer contours clockwise and holes
		// counter-clockwise, which nonzero resolves correctly in every renderer.
		sb.WriteString(fmt.Sprintf(`<path fill="%s" d="%s"/>`, hexRGB(palette[i]), d.String()))
	}
	sb.WriteString("</svg>")
	out := []byte(sb.String())
	return RunResult{Bytes: out, Format: FormatSVG, Mime: MIMEFor(FormatSVG), Width: w, Height: h}, nil
}

type rgb struct{ R, G, B uint8 }

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func hexRGB(c rgb) string {
	return "#" + hex2(c.R) + hex2(c.G) + hex2(c.B)
}

func hex2(v uint8) string {
	s := strconv.FormatUint(uint64(v), 16)
	if len(s) == 1 {
		return "0" + s
	}
	return s
}

// quantize resolves the palette and per-pixel labels. keep_colors, when
// non-empty, pins the palette and assigns a pixel only when it is within ΔE 12
// of one of the colours (otherwise it is background); otherwise k-means finds
// `colors` (default 4) clusters. The zero label is the first palette entry.
func quantize(px []rgb, colors int, keep []string) ([]rgb, []int, error) {
	if len(keep) > 0 {
		palette := make([]rgb, 0, len(keep))
		lab := make([]labColor, 0, len(keep))
		for _, h := range keep {
			c, err := parseHexColor(h)
			if err != nil {
				return nil, nil, err
			}
			r := rgb{c.R, c.G, c.B}
			palette = append(palette, r)
			lab = append(lab, toLab(r))
		}
		labels := make([]int, len(px))
		for i, c := range px {
			lc := toLab(c)
			best, bestD := -1, math.Inf(1)
			for j := range palette {
				if d := deltaE76(lc, lab[j]); d < bestD {
					best, bestD = j, d
				}
			}
			if best >= 0 && bestD <= 12 {
				labels[i] = best
			} else {
				labels[i] = -1
			}
		}
		return palette, labels, nil
	}

	if colors <= 0 {
		colors = 4
	}
	if colors > 16 {
		colors = 16
	}
	if colors > len(px) {
		colors = len(px)
	}
	palette, centroids := kmeans(px, colors)
	labels := make([]int, len(px))
	for i, c := range px {
		best, bestD := 0, math.Inf(1)
		for j, cc := range centroids {
			d := sqDist(c, cc)
			if d < bestD {
				best, bestD = j, d
			}
		}
		labels[i] = best
	}
	return palette, labels, nil
}

// kmeans is deterministic: pixels are pre-sorted, centroids seeded at evenly
// spaced positions, ties broken by lowest index, and the iteration order is
// stable. Two runs on the same image produce the same palette.
func kmeans(px []rgb, k int) ([]rgb, []rgb) {
	sorted := append([]rgb(nil), px...)
	sort.Slice(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		if a.R != b.R {
			return a.R < b.R
		}
		if a.G != b.G {
			return a.G < b.G
		}
		return a.B < b.B
	})
	centroids := make([]rgb, k)
	for i := 0; i < k; i++ {
		idx := 0
		if k > 1 {
			idx = i * (len(sorted) - 1) / (k - 1)
		}
		centroids[i] = sorted[idx]
	}
	for iter := 0; iter < 12; iter++ {
		sums := make([][3]float64, k)
		counts := make([]int, k)
		for _, c := range px {
			best, bestD := 0, math.Inf(1)
			for j, cc := range centroids {
				d := sqDist(c, cc)
				if d < bestD {
					best, bestD = j, d
				}
			}
			sums[best][0] += float64(c.R)
			sums[best][1] += float64(c.G)
			sums[best][2] += float64(c.B)
			counts[best]++
		}
		changed := false
		for j := range centroids {
			if counts[j] == 0 {
				continue
			}
			next := rgb{
				R: uint8(math.Round(sums[j][0] / float64(counts[j]))),
				G: uint8(math.Round(sums[j][1] / float64(counts[j]))),
				B: uint8(math.Round(sums[j][2] / float64(counts[j]))),
			}
			if next != centroids[j] {
				changed = true
			}
			centroids[j] = next
		}
		if !changed {
			break
		}
	}
	return append([]rgb(nil), centroids...), centroids
}

func sqDist(a, b rgb) float64 {
	dr := float64(a.R) - float64(b.R)
	dg := float64(a.G) - float64(b.G)
	db := float64(a.B) - float64(b.B)
	return dr*dr + dg*dg + db*db
}

// --- Lab / deltaE ---

type labColor struct{ L, A, B float64 }

func toLab(c rgb) labColor {
	r := linearize(float64(c.R) / 255)
	g := linearize(float64(c.G) / 255)
	b := linearize(float64(c.B) / 255)
	// sRGB D65 -> XYZ
	x := r*0.4124564 + g*0.3575761 + b*0.1804375
	y := r*0.2126729 + g*0.7151522 + b*0.0721750
	z := r*0.0193339 + g*0.1191920 + b*0.9503041
	// normalize for D65 white
	x /= 0.95047
	z /= 1.08883
	fx, fy, fz := labF(x), labF(y), labF(z)
	return labColor{116*fy - 16, 500 * (fx - fy), 200 * (fy - fz)}
}

func linearize(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func labF(t float64) float64 {
	if t > 0.008856 {
		return math.Cbrt(t)
	}
	return 7.787*t + 16.0/116.0
}

func deltaE76(a, b labColor) float64 {
	dl := a.L - b.L
	da := a.A - b.A
	db := a.B - b.B
	return math.Sqrt(dl*dl + da*da + db*db)
}

// --- masks ---

func touchesBorder(mask []bool, w, h int) bool {
	for x := 0; x < w; x++ {
		if mask[x] || mask[(h-1)*w+x] {
			return true
		}
	}
	for y := 0; y < h; y++ {
		if mask[y*w] || mask[y*w+w-1] {
			return true
		}
	}
	return false
}

func unionMasks(masks [][]bool, kept []int) []bool {
	if len(kept) == 0 {
		return nil
	}
	n := len(masks[kept[0]])
	out := make([]bool, n)
	for _, i := range kept {
		m := masks[i]
		for j := range m {
			if m[j] {
				out[j] = true
			}
		}
	}
	return out
}

// clipRegionMask selects the pixels that represent the enclosing tile: with
// keep_colors, the pixels assigned to no listed colour (label -1); with k-means
// and a border drop, the dropped layer(s); otherwise every pixel.
func clipRegionMask(labels []int, masks [][]bool, kept, dropped []int, w, h int) []bool {
	if len(dropped) > 0 {
		return unionMasks(masks, dropped)
	}
	if len(kept) < len(masks) || labelsContainBackground(labels) {
		region := make([]bool, w*h)
		for i, l := range labels {
			if l < 0 {
				region[i] = true
			}
		}
		if countTrue(region) > 0 {
			return region
		}
	}
	all := make([]int, len(masks))
	for i := range masks {
		all[i] = i
	}
	return unionMasks(masks, all)
}

func labelsContainBackground(labels []int) bool {
	for _, l := range labels {
		if l < 0 {
			return true
		}
	}
	return false
}

func countTrue(m []bool) int {
	n := 0
	for _, v := range m {
		if v {
			n++
		}
	}
	return n
}

// regionFromHull builds the clip mask as the convex hull of the largest
// connected component of regionMask, eroded by inset pixels. The hull follows
// the rounded tile's corners (which a plain bounding box cannot), and the
// erosion removes the page margin outside it.
func regionFromHull(regionMask []bool, w, h int, inset float64) []bool {
	if regionMask == nil {
		return nil
	}
	pts := largestComponentPoints(regionMask, w, h)
	if len(pts) < 3 {
		return nil
	}
	hull := convexHull(pts)
	if len(hull) < 3 {
		return nil
	}
	mask := rasterizeHull(hull, w, h)
	erodeMask(mask, w, h, int(math.Round(inset)))
	return mask
}

// largestComponentPoints returns the pixel centres of the largest 4-connected
// component of mask.
func largestComponentPoints(mask []bool, w, h int) []point {
	seen := make([]bool, len(mask))
	best := []point{}
	for start := 0; start < len(mask); start++ {
		if !mask[start] || seen[start] {
			continue
		}
		stack := []int{start}
		seen[start] = true
		comp := []point{}
		for len(stack) > 0 {
			i := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			x, y := i%w, i/w
			comp = append(comp, point{x, y})
			if n := i - 1; n >= 0 && x > 0 && !seen[n] && mask[n] {
				seen[n] = true
				stack = append(stack, n)
			}
			if n := i + 1; n < len(mask) && x < w-1 && !seen[n] && mask[n] {
				seen[n] = true
				stack = append(stack, n)
			}
			if n := i - w; n >= 0 && !seen[n] && mask[n] {
				seen[n] = true
				stack = append(stack, n)
			}
			if n := i + w; n < len(mask) && !seen[n] && mask[n] {
				seen[n] = true
				stack = append(stack, n)
			}
		}
		if len(comp) > len(best) {
			best = comp
		}
	}
	return best
}

// convexHull is Andrew's monotone chain, returning hull vertices in order.
func convexHull(pts []point) []point {
	if len(pts) < 3 {
		return pts
	}
	sorted := append([]point(nil), pts...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].X != sorted[j].X {
			return sorted[i].X < sorted[j].X
		}
		return sorted[i].Y < sorted[j].Y
	})
	cross := func(o, a, b point) int {
		return (a.X-o.X)*(b.Y-o.Y) - (a.Y-o.Y)*(b.X-o.X)
	}
	lower := make([]point, 0, len(sorted))
	for _, p := range sorted {
		for len(lower) >= 2 && cross(lower[len(lower)-2], lower[len(lower)-1], p) <= 0 {
			lower = lower[:len(lower)-1]
		}
		lower = append(lower, p)
	}
	upper := make([]point, 0, len(sorted))
	for i := len(sorted) - 1; i >= 0; i-- {
		p := sorted[i]
		for len(upper) >= 2 && cross(upper[len(upper)-2], upper[len(upper)-1], p) <= 0 {
			upper = upper[:len(upper)-1]
		}
		upper = append(upper, p)
	}
	return append(lower[:len(lower)-1], upper[:len(upper)-1]...)
}

// rasterizeHull marks every pixel centre inside the hull polygon.
func rasterizeHull(hull []point, w, h int) []bool {
	mask := make([]bool, w*h)
	fx := make([]float64, len(hull))
	fy := make([]float64, len(hull))
	for i, p := range hull {
		fx[i] = float64(p.X) + 0.5
		fy[i] = float64(p.Y) + 0.5
	}
	for y := 0; y < h; y++ {
		py := float64(y) + 0.5
		for x := 0; x < w; x++ {
			px := float64(x) + 0.5
			if pointInConvex(fx, fy, px, py) {
				mask[y*w+x] = true
			}
		}
	}
	return mask
}

func pointInConvex(fx, fy []float64, px, py float64) bool {
	n := len(fx)
	sign := 0
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		cr := (fx[j]-fx[i])*(py-fy[i]) - (fy[j]-fy[i])*(px-fx[i])
		if cr > 1e-9 {
			if sign == -1 {
				return false
			}
			sign = 1
		} else if cr < -1e-9 {
			if sign == 1 {
				return false
			}
			sign = -1
		}
	}
	return true
}

// erodeMask shrinks mask by a Chebyshev-radius box erosion.
func erodeMask(mask []bool, w, h, radius int) {
	if radius <= 0 {
		return
	}
	rowMin := make([]bool, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			ok := true
			for dx := -radius; dx <= radius && ok; dx++ {
				xx := x + dx
				if xx < 0 || xx >= w || !mask[y*w+xx] {
					ok = false
				}
			}
			rowMin[y*w+x] = ok
		}
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			ok := true
			for dy := -radius; dy <= radius && ok; dy++ {
				yy := y + dy
				if yy < 0 || yy >= h || !rowMin[yy*w+x] {
					ok = false
				}
			}
			mask[y*w+x] = ok
		}
	}
}

// --- contour tracing ---

type point struct{ X, Y int }

// traceMask returns every boundary loop of a binary mask using square-boundary
// ("crack") following: each foreground pixel contributes the sides adjacent to
// background, and the directed unit edges are chained into closed loops. Outer
// boundaries and holes both come out as loops; fill-rule evenodd makes their
// direction irrelevant.
func traceMask(mask []bool, w, h int) [][]point {
	edges := map[point][]point{}
	add := func(from, to point) { edges[from] = append(edges[from], to) }
	at := func(x, y int) bool {
		if x < 0 || y < 0 || x >= w || y >= h {
			return false
		}
		return mask[y*w+x]
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if !at(x, y) {
				continue
			}
			if !at(x, y-1) {
				add(point{x, y}, point{x + 1, y})
			}
			if !at(x+1, y) {
				add(point{x + 1, y}, point{x + 1, y + 1})
			}
			if !at(x, y+1) {
				add(point{x + 1, y + 1}, point{x, y + 1})
			}
			if !at(x-1, y) {
				add(point{x, y + 1}, point{x, y})
			}
		}
	}
	used := map[[2]point]bool{}
	var loops [][]point
	// Deterministic start order.
	starts := make([]point, 0, len(edges))
	for p := range edges {
		starts = append(starts, p)
	}
	sort.Slice(starts, func(i, j int) bool {
		if starts[i].Y != starts[j].Y {
			return starts[i].Y < starts[j].Y
		}
		return starts[i].X < starts[j].X
	})
	for _, start := range starts {
		for {
			next := nextUnused(edges, used, start)
			if next == nil {
				break
			}
			cur := start
			loop := []point{cur}
			used[[2]point{cur, *next}] = true
			cur = *next
			for cur != start {
				loop = append(loop, cur)
				n := nextUnused(edges, used, cur)
				if n == nil {
					break
				}
				used[[2]point{cur, *n}] = true
				cur = *n
			}
			if len(loop) >= 3 {
				loops = append(loops, loop)
			}
		}
	}
	return loops
}

func nextUnused(edges map[point][]point, used map[[2]point]bool, from point) *point {
	for _, to := range edges[from] {
		if !used[[2]point{from, to}] {
			cp := to
			return &cp
		}
	}
	return nil
}

// --- simplification ---

// simplifyClosed applies Ramer–Douglas–Peucker to a closed polygon. The polygon
// is treated as a cycle anchored at its first vertex.
func simplifyClosed(pts []point, tolerance float64) []point {
	if len(pts) < 4 {
		return pts
	}
	closed := append(append([]point(nil), pts...), pts[0])
	keep := make([]bool, len(closed))
	keep[0] = true
	keep[len(closed)-1] = true
	rdp(closed, 0, len(closed)-1, tolerance, keep)
	out := make([]point, 0, len(pts))
	for i := 0; i < len(closed)-1; i++ {
		if keep[i] {
			out = append(out, closed[i])
		}
	}
	return out
}

func rdp(pts []point, start, end int, tolerance float64, keep []bool) {
	if end <= start+1 {
		return
	}
	maxD := 0.0
	maxIdx := -1
	for i := start + 1; i < end; i++ {
		d := perpendicularDistance(pts[i], pts[start], pts[end])
		if d > maxD {
			maxD = d
			maxIdx = i
		}
	}
	if maxD > tolerance && maxIdx > 0 {
		keep[maxIdx] = true
		rdp(pts, start, maxIdx, tolerance, keep)
		rdp(pts, maxIdx, end, tolerance, keep)
	}
}

func perpendicularDistance(p, a, b point) float64 {
	dx := float64(b.X - a.X)
	dy := float64(b.Y - a.Y)
	if dx == 0 && dy == 0 {
		return math.Hypot(float64(p.X-a.X), float64(p.Y-a.Y))
	}
	num := math.Abs(dy*float64(p.X-a.X) - dx*float64(p.Y-a.Y))
	return num / math.Hypot(dx, dy)
}

func polygonArea(pts []point) float64 {
	if len(pts) < 3 {
		return 0
	}
	var area float64
	for i := range pts {
		j := (i + 1) % len(pts)
		area += float64(pts[i].X)*float64(pts[j].Y) - float64(pts[j].X)*float64(pts[i].Y)
	}
	return math.Abs(area) / 2
}

func writePath(sb *strings.Builder, pts []point) {
	if len(pts) < 3 {
		return
	}
	for i, p := range pts {
		if i == 0 {
			fmt.Fprintf(sb, "M%d %d", p.X, p.Y)
		} else {
			fmt.Fprintf(sb, "L%d %d", p.X, p.Y)
		}
	}
	sb.WriteString("Z")
}

// dilate1 grows a binary mask by one pixel in the 4-connected neighbourhood.
func dilate1(mask []bool, w, h int) []bool {
	out := make([]bool, len(mask))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			if !mask[i] {
				continue
			}
			out[i] = true
			if x > 0 {
				out[i-1] = true
			}
			if x < w-1 {
				out[i+1] = true
			}
			if y > 0 {
				out[i-w] = true
			}
			if y < h-1 {
				out[i+w] = true
			}
		}
	}
	return out
}
