package quality

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"math"
	"reflect"
	"strings"

	"github.com/disintegration/imaging"
	"image-tools/internal/diff"
	"image-tools/internal/storage"
)

const maxPixels = 16_000_000

func decode(data []byte) (image.Image, error) {
	guard := storage.Guard{MaxBytes: 32 << 20, MaxMegapixels: 16, MaxDimension: 8192}
	if _, err := guard.Inspect(bytes.NewReader(data)); err != nil {
		return nil, err
	}
	im, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		return nil, err
	}
	if im.Bounds().Dx()*im.Bounds().Dy() > maxPixels {
		return nil, fmt.Errorf("quality: decoded image exceeds %d pixels", maxPixels)
	}
	return im, nil
}

func imageMetrics(kind string, reference, output []byte) (map[string]float64, []string, error) {
	a, err := decode(reference)
	if err != nil {
		return nil, nil, fmt.Errorf("reference: %w", err)
	}
	b, err := decode(output)
	if err != nil {
		return nil, nil, fmt.Errorf("output: %w", err)
	}
	match := a.Bounds().Size() == b.Bounds().Size()
	m := map[string]float64{"dimensions_match": 0}
	if match {
		m["dimensions_match"] = 1
	}
	if kind == "image" {
		d, err := diff.Compare(reference, output, diff.Params{Mode: diff.ModePerceptual})
		if err != nil {
			return nil, nil, err
		}
		m["reference.ssim"], m["reference.psnr"], m["reference.mae"], m["reference.phash_similarity"] = d.SSIM, d.PSNR, d.MAE/255, d.PhashSimilarity
		return m, d.Warnings, nil
	}
	// Resampling a segmentation would conceal a geometry regression. Reject it
	// instead of reporting a plausible IoU for incomparable pixel coordinates.
	if !match {
		return m, nil, fmt.Errorf("mask dimensions differ: reference %v, candidate %v", a.Bounds().Size(), b.Bounds().Size())
	}
	w, h := a.Bounds().Dx(), a.Bounds().Dy()
	ref, out := make([]bool, w*h), make([]bool, w*h)
	var intersection, union, positivesA, positivesB, absolute float64
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			av, bv := maskValue(a.At(a.Bounds().Min.X+x, a.Bounds().Min.Y+y), kind), maskValue(b.At(b.Bounds().Min.X+x, b.Bounds().Min.Y+y), kind)
			absolute += math.Abs(av - bv)
			pa, pb := av >= .5, bv >= .5
			ref[y*w+x], out[y*w+x] = pa, pb
			if pa {
				positivesA++
			}
			if pb {
				positivesB++
			}
			if pa && pb {
				intersection++
			}
			if pa || pb {
				union++
			}
		}
	}
	m["alpha.mae"] = absolute / float64(w*h)
	m["alpha.iou"] = ratio(intersection, union)
	m["alpha.precision"], m["alpha.recall"] = ratio(intersection, positivesB), ratio(intersection, positivesA)
	// An empty prediction against a nonempty reference has zero precision.
	if positivesB == 0 && positivesA > 0 {
		m["alpha.precision"] = 0
	}
	if positivesA == 0 && positivesB > 0 {
		m["alpha.recall"] = 0
	}
	ba, bb := boundary(ref, w, h), boundary(out, w, h)
	p, r := boundaryMatch(bb, ba, w, h), boundaryMatch(ba, bb, w, h)
	m["alpha.boundary_f1"] = 0
	if p+r > 0 {
		m["alpha.boundary_f1"] = 2 * p * r / (p + r)
	}
	return m, nil, nil
}

func maskValue(c color.Color, kind string) float64 {
	if kind == "matte" {
		_, _, _, a := c.RGBA()
		return float64(a) / 65535
	}
	n := color.NRGBAModel.Convert(c).(color.NRGBA)
	return (.2126*float64(n.R) + .7152*float64(n.G) + .0722*float64(n.B)) / 255
}
func ratio(n, d float64) float64 {
	if d == 0 {
		return 1
	}
	return n / d
}
func boundary(mask []bool, w, h int) []bool {
	out := make([]bool, len(mask))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if !mask[y*w+x] {
				continue
			}
			out[y*w+x] = x == 0 || y == 0 || x == w-1 || y == h-1 || !mask[y*w+x-1] || !mask[y*w+x+1] || !mask[(y-1)*w+x] || !mask[(y+1)*w+x]
		}
	}
	return out
}
func boundaryMatch(a, b []bool, w, h int) float64 {
	matched, total := 0., 0.
	for i, v := range a {
		if !v {
			continue
		}
		total++
		x, y := i%w, i/w
		found := false
		for dy := -1; dy <= 1 && !found; dy++ {
			for dx := -1; dx <= 1; dx++ {
				xx, yy := x+dx, y+dy
				if xx >= 0 && yy >= 0 && xx < w && yy < h && b[yy*w+xx] {
					found = true
					break
				}
			}
		}
		if found {
			matched++
		}
	}
	if total == 0 {
		for _, v := range b {
			if v {
				return 0
			}
		}
		return 1
	}
	return matched / total
}

func textMetrics(kind, reference, output string) (map[string]float64, error) {
	if kind == "json" {
		var a, b any
		if err := json.Unmarshal([]byte(reference), &a); err != nil {
			return nil, fmt.Errorf("reference JSON: %w", err)
		}
		if err := json.Unmarshal([]byte(output), &b); err != nil {
			return nil, fmt.Errorf("output JSON: %w", err)
		}
		am, bm := map[string]any{}, map[string]any{}
		flatten("", a, am)
		flatten("", b, bm)
		correct := 0.
		keys := map[string]bool{}
		for k, v := range am {
			keys[k] = true
			if bv, ok := bm[k]; ok && reflect.DeepEqual(v, bv) {
				correct++
			}
		}
		for k := range bm {
			keys[k] = true
		}
		exact := 0.
		if reflect.DeepEqual(a, b) {
			exact = 1
		}
		return map[string]float64{"json.exact": exact, "json.field_accuracy": ratio(correct, float64(len(keys)))}, nil
	}
	a, b := []rune(strings.Join(strings.Fields(reference), " ")), []rune(strings.Join(strings.Fields(output), " "))
	if len(a) > 10000 || len(b) > 10000 {
		return nil, fmt.Errorf("text metrics limit is 10000 characters")
	}
	exact := 0.
	if string(a) == string(b) {
		exact = 1
	}
	n := max(len(a), len(b))
	similarity := 1.
	if n > 0 {
		similarity = 1 - float64(distance(a, b))/float64(n)
	}
	return map[string]float64{"text.exact": exact, "text.similarity": similarity}, nil
}

func flatten(path string, v any, out map[string]any) {
	switch n := v.(type) {
	case map[string]any:
		if len(n) == 0 {
			out[path] = n
		}
		for k, x := range n {
			key := strings.ReplaceAll(strings.ReplaceAll(k, "~", "~0"), "/", "~1")
			flatten(path+"/"+key, x, out)
		}
	case []any:
		if len(n) == 0 {
			out[path] = n
		}
		for i, x := range n {
			flatten(fmt.Sprintf("%s/%d", path, i), x, out)
		}
	default:
		out[path] = v
	}
}
func distance(a, b []rune) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i, x := range a {
		row := make([]int, len(b)+1)
		row[0] = i + 1
		for j, y := range b {
			c := 0
			if x != y {
				c = 1
			}
			row[j+1] = min(row[j]+1, prev[j+1]+1, prev[j]+c)
		}
		prev = row
	}
	return prev[len(b)]
}
