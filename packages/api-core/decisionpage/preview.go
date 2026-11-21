package decisionpage

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/vrooli/api-core/pagescheme"
)

// KitRelDir is where the Hub's page kit lives in the repository.
const KitRelDir = "scenarios/notification-hub/ui/src/features/packet/explainer/kit"

// SkillRelDir is where the notification-hub skill lives in the repository.
const SkillRelDir = "scenarios/notification-hub/skills/notification-hub"

// RepoRoot finds the Vrooli checkout: VROOLI_ROOT, else ~/Vrooli.
func RepoRoot() string {
	if root := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); root != "" {
		return root
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Vrooli")
}

// Shot is one screenshot to take.
type Shot struct {
	Width int
	Theme string // "light" or "dark"
}

// DefaultShots are the phone shots a page is judged on.
var DefaultShots = []Shot{{390, "light"}, {390, "dark"}}

// shownTheme is the theme the Hub shows a page in when its own theme is theme,
// and the scheme the page is locked to ("" when it follows the Hub). It is the
// Hub's own reading (pagescheme, which the packet check uses too) with the kit
// the preview injects: a page whose literal colours only work in one scheme is
// locked there with data-hub-scheme.
func shownTheme(html, kitCSS, theme string) (string, string) {
	switch scheme := pagescheme.NewKit(kitCSS).Scheme(html); scheme {
	case pagescheme.SchemeLight, pagescheme.SchemeDark:
		return scheme, scheme
	}
	return theme, ""
}

// Preview is the page as the Hub frames it, approximately: the kit's
// stylesheet and the base rule injected, the theme the Hub would show it in
// set on the root (theme, or the one scheme the page's own colours lock it
// to) and a stub bridge, so a page renders off-Hub as it will on the phone.
func Preview(html []byte, kitDir, theme string) (string, error) {
	css, err := os.ReadFile(filepath.Join(kitDir, "kit.css"))
	if err != nil {
		return "", fmt.Errorf("the Hub page kit: %w", err)
	}
	js, _ := os.ReadFile(filepath.Join(kitDir, "kit.js"))
	page := string(html)
	theme, locked := shownTheme(page, string(css), theme)
	kitOff := regexp.MustCompile(`(?i)<meta[^>]+name=["']hub-kit["'][^>]+content=["']off["']`).MatchString(page)
	inject := `<style data-hub="base">@layer hub-base { html, body { margin: 0; } }</style>`
	if !kitOff {
		inject += "<style data-hub=\"kit\">" + strings.ReplaceAll(string(css), "</style", "<\\/style") + "</style>"
	}
	inject += `<script data-hub="bridge">window.hub={theme:"` + theme + `",prefill:function(q,a){document.documentElement.setAttribute("data-prefilled",q+"="+a)},open:function(){}};</script>`
	if !kitOff && len(js) > 0 {
		inject += "<script data-hub=\"kit\">" + strings.ReplaceAll(string(js), "</script", "<\\/script") + "</script>"
	}
	head := regexp.MustCompile(`(?i)<head[^>]*>`)
	if loc := head.FindStringIndex(page); loc != nil {
		page = page[:loc[1]] + inject + page[loc[1]:]
	} else {
		page = inject + page
	}
	attrs := ` data-theme="` + theme + `"`
	if locked != "" {
		attrs += ` data-hub-scheme="` + locked + `"`
	}
	root := regexp.MustCompile(`(?i)<html([^>]*)>`)
	if root.MatchString(page) {
		page = root.ReplaceAllString(page, `<html$1`+attrs+`>`)
	} else {
		page = `<html` + attrs + `>` + page + `</html>`
	}
	return page, nil
}

// Browser finds a headless-capable Chrome.
func Browser() (string, error) {
	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser"} {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}
	return "", errors.New("no Chrome or Chromium on PATH")
}

// Shoot photographs a page as the Hub frames it: one PNG per shot named
// <width>-<theme>.png in outDir, cropped to the page's length. <theme> is the
// Hub's theme; a page its own colours lock to one scheme is shown in that scheme.
func Shoot(ctx context.Context, pagePath, outDir, kitDir string, shots []Shot) ([]string, error) {
	html, err := os.ReadFile(pagePath)
	if err != nil {
		return nil, err
	}
	kitCSS, err := os.ReadFile(filepath.Join(kitDir, "kit.css"))
	if err != nil {
		return nil, fmt.Errorf("the Hub page kit: %w", err)
	}
	browser, err := Browser()
	if err != nil {
		return nil, err
	}
	if len(shots) == 0 {
		shots = DefaultShots
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	tmp, err := os.MkdirTemp(outDir, ".preview-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	var out []string
	for _, s := range shots {
		doc, err := Preview(html, kitDir, s.Theme)
		if err != nil {
			return out, err
		}
		src := filepath.Join(tmp, fmt.Sprintf("%d-%s.html", s.Width, s.Theme))
		if err := os.WriteFile(src, []byte(doc), 0o644); err != nil {
			return out, err
		}
		dst := filepath.Join(outDir, fmt.Sprintf("%d-%s.png", s.Width, s.Theme))
		cctx, cancel := context.WithTimeout(ctx, 60*time.Second)
		args := []string{
			"--headless=new", "--disable-gpu", "--no-sandbox", "--hide-scrollbars", "--no-first-run",
			"--force-device-scale-factor=1", "--user-data-dir=" + filepath.Join(tmp, "profile"),
			fmt.Sprintf("--window-size=%d,%d", s.Width, 6000), "--virtual-time-budget=2000",
			"--screenshot=" + dst, "file://" + src,
		}
		// The browser's own scheme follows what the Hub shows: a page locked to
		// light stays light in the dark shot, as the Hub's frame does.
		if shown, _ := shownTheme(string(html), string(kitCSS), s.Theme); shown == "dark" {
			args = append([]string{"--force-dark-mode", "--blink-settings=preferredColorScheme=0"}, args...)
		}
		cmd := exec.CommandContext(cctx, browser, args...)
		combined, err := cmd.CombinedOutput()
		cancel()
		if err != nil {
			return out, fmt.Errorf("chrome: %v: %s", err, clip(string(combined), 400))
		}
		if err := cropBottom(dst); err != nil {
			return out, err
		}
		out = append(out, dst)
	}
	return out, nil
}

// cropBottom trims the blank rows below the page's last content row. A row is
// blank when every pixel is within a small distance of its first pixel, so a
// dithered or slowly shading background (a dark-scheme gradient) is trimmed
// too, not only an exactly uniform one.
func cropBottom(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	img, err := png.Decode(f)
	f.Close()
	if err != nil {
		return err
	}
	b := img.Bounds()
	last := b.Min.Y
	for y := b.Max.Y - 1; y >= b.Min.Y; y-- {
		if !blankRow(img, y) {
			last = y
			break
		}
	}
	end := min(b.Max.Y, last+32)
	if end >= b.Max.Y {
		return nil
	}
	sub, ok := img.(interface {
		SubImage(image.Rectangle) image.Image
	})
	if !ok {
		return nil
	}
	cropped := sub.SubImage(image.Rect(b.Min.X, b.Min.Y, b.Max.X, end))
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	return png.Encode(out, cropped)
}

// blankRow reports whether row y is one colour, within blankTolerance per
// channel (8-bit), as a page background is and content never is.
func blankRow(img image.Image, y int) bool {
	const blankTolerance = 6
	b := img.Bounds()
	r0, g0, b0, _ := img.At(b.Min.X, y).RGBA()
	for x := b.Min.X + 1; x < b.Max.X; x++ {
		r, g, bl, _ := img.At(x, y).RGBA()
		if far(r, r0, blankTolerance) || far(g, g0, blankTolerance) || far(bl, b0, blankTolerance) {
			return false
		}
	}
	return true
}

func far(a, b uint32, tolerance uint32) bool {
	a, b = a>>8, b>>8
	if a > b {
		return a-b > tolerance
	}
	return b-a > tolerance
}
