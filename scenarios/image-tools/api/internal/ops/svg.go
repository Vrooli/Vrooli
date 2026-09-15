package ops

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

// MaxSVGRasterDimension caps either side of an SVG rasterization so a tiny SVG
// declaring an enormous viewBox can't allocate an unbounded raster (the
// vector-format analogue of the decompression-bomb guard).
const MaxSVGRasterDimension = 8192

// Two rasterizers, and a rule that decides between them.
//
// `oksvg` is pure Go, fast, and pinned at a 2022 revision that understands
// paths, basic shapes, transforms and gradients. It does not understand
// `<filter>`, `<mask>`, `<pattern>` or CSS — and it does not say so. It draws
// what it recognises and drops the rest, so an SVG whose whole character comes
// from a turbulence filter rasterizes into a clean geometric shape that looks
// like a successful render and is not the picture the author described.
//
// Silent omission is the worst failure mode available to a rasterizer whose
// output a person judges by eye, because nothing downstream can detect it: the
// bytes are a valid PNG, the geometry is right, the gate passes, and the
// picture is wrong. So the choice is not "use the better one when we have it".
// The rule is:
//
//	an SVG that uses a feature oksvg silently drops is NEVER given to oksvg.
//	It is rendered by headless Chrome, or it fails by name.
//
// An SVG that uses none of them still takes the pure-Go path, which keeps the
// common case dependency-free and fast.

// highFidelityFeature is one SVG capability oksvg drops without reporting.
type highFidelityFeature struct {
	name    string
	pattern *regexp.Regexp
}

// The four features D4 names. Each is matched on its element or attribute form
// rather than on a bare word, so a path with "filter" in an id does not force
// the slow path.
var highFidelityFeatures = []highFidelityFeature{
	{"filter", regexp.MustCompile(`(?i)<\s*filter[\s>]|\bfilter\s*=\s*["']\s*url\(`)},
	{"mask", regexp.MustCompile(`(?i)<\s*mask[\s>]|\bmask\s*=\s*["']\s*url\(`)},
	{"pattern", regexp.MustCompile(`(?i)<\s*pattern[\s>]`)},
	{"css", regexp.MustCompile(`(?i)<\s*style[\s>]|\bstyle\s*=\s*["'][^"']*\bfilter\s*:`)},
}

// HighFidelitySVGFeatures reports which oksvg-unsupported features an SVG uses.
// It is exported because the decision it drives — pure-Go path or Chrome path —
// is one a caller may need to explain, and a test asserts it directly.
func HighFidelitySVGFeatures(data []byte) []string {
	var used []string
	for _, feature := range highFidelityFeatures {
		if feature.pattern.Match(data) {
			used = append(used, feature.name)
		}
	}
	return used
}

// ErrSVGFidelityUnavailable reports an SVG that needs the high-fidelity
// rasterizer on a host that has none. It names the features and the tool, so
// the message is a provisioning instruction rather than a puzzle.
type ErrSVGFidelityUnavailable struct {
	Features []string
}

func (e *ErrSVGFidelityUnavailable) Error() string {
	return fmt.Sprintf(
		"ops: this SVG uses %s, which the pure-Go rasterizer drops silently, and no headless Chrome binary was found on PATH. "+
			"Refusing rather than returning a plausible picture that is not the one described. "+
			"Install Google Chrome or Chromium, or set %s to its path.",
		strings.Join(e.Features, ", "), chromeBinaryEnv)
}

const (
	// chromeBinaryEnv overrides binary discovery, for a host that keeps Chrome
	// somewhere unusual or wants to pin a specific build for reproducibility.
	chromeBinaryEnv = "IMAGE_TOOLS_CHROME_BINARY"
	// chromeRenderTimeout bounds one rasterization. A static SVG renders in
	// well under a second; anything near this bound is a hung process.
	chromeRenderTimeout = 60 * time.Second
	// chromeVirtualTimeBudget lets Chrome settle layout and any CSS animation
	// to a fixed virtual instant before the screenshot, which is what makes the
	// output deterministic instead of a race against the compositor.
	chromeVirtualTimeBudget = 4000
)

var chromeCandidates = []string{
	"google-chrome-stable", "google-chrome", "chromium-browser", "chromium",
	"chrome", "headless-shell",
}

var (
	chromeOnce sync.Once
	chromePath string
)

// ChromeBinary resolves the headless browser used for high-fidelity SVG, or ""
// when the host has none. The lookup is cached because it runs per rasterization
// and PATH does not change under a running process.
func ChromeBinary() string {
	chromeOnce.Do(func() {
		if explicit := strings.TrimSpace(os.Getenv(chromeBinaryEnv)); explicit != "" {
			if _, err := os.Stat(explicit); err == nil {
				chromePath = explicit
				return
			}
		}
		for _, candidate := range chromeCandidates {
			if resolved, err := exec.LookPath(candidate); err == nil {
				chromePath = resolved
				return
			}
		}
	})
	return chromePath
}

// rasterizeSVG renders SVG bytes to an image. width/height of 0 use the SVG's
// intrinsic viewBox size; a non-zero pair scales to that target. The result is
// bounded by MaxSVGRasterDimension on each side.
func rasterizeSVG(data []byte, width, height int) (image.Image, error) {
	w, h, err := svgTargetSize(data, width, height)
	if err != nil {
		return nil, err
	}
	features := HighFidelitySVGFeatures(data)
	if len(features) == 0 {
		return rasterizeSVGPureGo(data, w, h)
	}
	if ChromeBinary() == "" {
		return nil, &ErrSVGFidelityUnavailable{Features: features}
	}
	return rasterizeSVGChrome(data, w, h)
}

// svgTargetSize resolves the output geometry and enforces the bomb guard.
func svgTargetSize(data []byte, width, height int) (int, int, error) {
	w, h := width, height
	if w <= 0 || h <= 0 {
		intrinsicW, intrinsicH := intrinsicSVGSize(data)
		if w <= 0 {
			w = intrinsicW
		}
		if h <= 0 {
			h = intrinsicH
		}
	}
	if w <= 0 || h <= 0 {
		// No intrinsic size and no target: fall back to a sane default canvas.
		w, h = 512, 512
	}
	if w > MaxSVGRasterDimension || h > MaxSVGRasterDimension {
		return 0, 0, fmt.Errorf("%w: svg raster %dx%d exceeds %d px per side", ErrDecode, w, h, MaxSVGRasterDimension)
	}
	return w, h, nil
}

var (
	viewBoxPattern = regexp.MustCompile(`(?i)viewBox\s*=\s*["']\s*([-\d.eE]+)[,\s]+([-\d.eE]+)[,\s]+([-\d.eE]+)[,\s]+([-\d.eE]+)`)
	svgWidthAttr   = regexp.MustCompile(`(?i)<svg[^>]*?\bwidth\s*=\s*["']\s*([\d.]+)`)
	svgHeightAttr  = regexp.MustCompile(`(?i)<svg[^>]*?\bheight\s*=\s*["']\s*([\d.]+)`)
)

// intrinsicSVGSize reads the declared size without a full parse, because the
// Chrome path never parses the document itself and still has to know how large
// a window to open.
func intrinsicSVGSize(data []byte) (int, int) {
	if m := svgWidthAttr.FindSubmatch(data); m != nil {
		if n, err := strconv.ParseFloat(string(m[1]), 64); err == nil && n > 0 {
			if m2 := svgHeightAttr.FindSubmatch(data); m2 != nil {
				if n2, err2 := strconv.ParseFloat(string(m2[1]), 64); err2 == nil && n2 > 0 {
					return int(n), int(n2)
				}
			}
		}
	}
	if m := viewBoxPattern.FindSubmatch(data); m != nil {
		w, errW := strconv.ParseFloat(string(m[3]), 64)
		h, errH := strconv.ParseFloat(string(m[4]), 64)
		if errW == nil && errH == nil && w > 0 && h > 0 {
			return int(w), int(h)
		}
	}
	return 0, 0
}

func rasterizeSVGPureGo(data []byte, w, h int) (image.Image, error) {
	icon, err := oksvg.ReadIconStream(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: svg parse: %v", ErrDecode, err)
	}
	icon.SetTarget(0, 0, float64(w), float64(h))
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	scanner := rasterx.NewScannerGV(w, h, img, img.Bounds())
	raster := rasterx.NewDasher(w, h, scanner)
	icon.Draw(raster, 1.0)
	return img, nil
}

// svgHost is the page the SVG is rendered inside.
//
// Every rule here removes a source of non-determinism or of unwanted pixels:
// the margin reset stops the body's default 8px offsetting the drawing, the
// transparent background lets an SVG with no backdrop composite correctly, and
// the exact pixel sizing pins the drawing to the top-left corner at its target
// size.
//
// The size is written in `px` rather than `vw`/`vh` deliberately. Viewport
// units resolve against the layout viewport, which in headless Chrome is not
// the size `--window-size` asks for — the drawing came out correct in width and
// clipped 56px short in height, which is a picture that looks plausible and is
// missing its bottom edge. Pixels are the one unit whose meaning does not
// depend on how the browser decided to size its own window.
const svgHost = `<!doctype html><html><head><meta charset="utf-8"><style>
html,body{margin:0;padding:0;background:transparent;overflow:hidden;width:%dpx;height:%dpx}
svg{display:block;width:%dpx;height:%dpx}
</style></head><body>%s</body></html>`

// chromeViewportPad is how much taller than the target the window is opened.
//
// `--screenshot` captures the layout viewport, and headless Chrome's viewport
// is shorter than the window it was asked for by an amount that depends on the
// build. Rather than track that offset, the window is opened generously and the
// exact target rectangle is cropped from the top-left, which is where the page
// above pins the drawing. Over-requesting costs nothing; under-requesting
// silently truncates the picture.
const chromeViewportPad = 240

func rasterizeSVGChrome(data []byte, w, h int) (image.Image, error) {
	dir, err := os.MkdirTemp("", "image-tools-svg-")
	if err != nil {
		return nil, fmt.Errorf("%w: svg workspace: %v", ErrDecode, err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	page := filepath.Join(dir, "page.html")
	if err := os.WriteFile(page, []byte(fmt.Sprintf(svgHost, w, h, w, h, data)), 0o600); err != nil {
		return nil, fmt.Errorf("%w: svg page: %v", ErrDecode, err)
	}
	shot := filepath.Join(dir, "out.png")

	// --user-data-dir is per-render and inside the temp workspace: without it
	// concurrent rasterizations contend on one profile directory and Chrome
	// serialises or fails outright, which turns a parallel golden run into a
	// flake.
	args := []string{
		"--headless=new",
		"--disable-gpu",
		"--no-sandbox",
		"--disable-dev-shm-usage",
		"--hide-scrollbars",
		"--force-device-scale-factor=1",
		"--default-background-color=00000000",
		"--user-data-dir=" + filepath.Join(dir, "profile"),
		fmt.Sprintf("--virtual-time-budget=%d", chromeVirtualTimeBudget),
		fmt.Sprintf("--window-size=%d,%d", w, h+chromeViewportPad),
		"--screenshot=" + shot,
		"file://" + page,
	}
	cmd := exec.Command(ChromeBinary(), args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	done := make(chan error, 1)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("%w: start headless chrome: %v", ErrDecode, err)
	}
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			return nil, fmt.Errorf("%w: headless chrome: %v: %s", ErrDecode, err, strings.TrimSpace(stderr.String()))
		}
	case <-time.After(chromeRenderTimeout):
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("%w: headless chrome did not finish within %s", ErrDecode, chromeRenderTimeout)
	}

	encoded, err := os.ReadFile(shot)
	if err != nil {
		return nil, fmt.Errorf("%w: headless chrome wrote no screenshot: %v", ErrDecode, err)
	}
	img, err := png.Decode(bytes.NewReader(encoded))
	if err != nil {
		return nil, fmt.Errorf("%w: decode headless chrome screenshot: %v", ErrDecode, err)
	}
	// The screenshot is the padded viewport; the drawing is its top-left
	// corner. A short screenshot is a truncated picture, so it is an error
	// rather than something to pad back out.
	bounds := img.Bounds()
	if bounds.Dx() < w || bounds.Dy() < h {
		return nil, fmt.Errorf("%w: headless chrome rendered %dx%d for a %dx%d target; the viewport was smaller than the drawing",
			ErrDecode, bounds.Dx(), bounds.Dy(), w, h)
	}
	target := image.Rect(bounds.Min.X, bounds.Min.Y, bounds.Min.X+w, bounds.Min.Y+h)
	cropped := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(cropped, cropped.Bounds(), img, target.Min, draw.Src)
	return cropped, nil
}
