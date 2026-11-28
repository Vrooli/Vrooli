package validation

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"path"
	"strings"

	"brand-manager/internal/profiles"
)

// placeholderHashes are known placeholder bytes that must never ship as a real
// icon: the react-vite template PNGs and the scenario-to-desktop 663-byte /
// 1817-byte placeholders.
var placeholderHashes = map[string]string{
	"80c4d8d2859afab47cd0a5b85f67d02d9fe0247e521cb980a9e462604d006e24": "react-vite template apple-icon-180.png",
	"d6fc31f168b5e32c1e7117faa208dc451e82a21d3b6fe2f30c45a47bffe8cb2a": "react-vite template favicon-196.png",
	"3ab1dfe4d938bc2a2c1f3d2b300828ed97c65d1a113128866b743b97aff2f654": "react-vite template manifest-icon-192.maskable.png",
	"7e41bbd6f20456270710e878b88cedae4175465dd28adad0905dc2d8c5f24796": "react-vite template manifest-icon-512.maskable.png",
	"75ccd7d14e470b80a07986ee693ba755aff663d9cb346aa7c492d04d3e8c8f78": "scenario-to-desktop 663-byte placeholder",
	"93d376406bbc46cab650351a81b16d8b4a36adc9979b299c27413d5527c0dc7d": "scenario-to-desktop 1817-byte placeholder",
	// scenario-to-desktop's exact-size grey placeholders (icon-<size>x<size>.png)
	// for the sizes its generator emits. The 256 and 512 values above are the
	// same bytes, retained with their historical origin labels.
	"8afc9cb0e46be8347433188dcacedbc95b8424a0f3e21a837d120dc47f2a4f59": "scenario-to-desktop 16x16 placeholder",
	"c8734c930135bfe23762c4f2c211d55c40de38ae7e7880cb041f969e8bf3aa6b": "scenario-to-desktop 32x32 placeholder",
	"05c38a2deadb0a6fd54a3dcfd747898aac4da476288aae1d2da06c4573537465": "scenario-to-desktop 48x48 placeholder",
	"ab53df3db0985319918519cb37e3ceea5a86ba969a08442cbfa4f0f6d6eca038": "scenario-to-desktop 128x128 placeholder",
	"d5980c80fb6855432d115ade65585585ece71982608d1b77351b11fbe749052b": "scenario-to-desktop 1024x1024 placeholder",
	// react-vite template placeholders under the /public/* layout (the template
	// now ships web-public-v1 names; the 4 entries above are the legacy
	// root-layout bytes still present in unmigrated fleet scenarios).
	"6ec56d91b27232305c9271800b7f36af478b08a3bdcccbb11b0fbc0ed85f1fde": "react-vite template apple-touch-icon.png",
	"182f80d2cc53eda7d4eb1afa6ca43da802a127793140656b59163aabca42f081": "react-vite template favicon-16.png",
	"bea3831ffad435516291003e907a6160e5834b75f7ac625eadd99f091b839e09": "react-vite template favicon-196.png",
	"4233184e0cad164596484b95c2d199ce3ce17e589d48a66a8b8a2d64b072305f": "react-vite template favicon-32.png",
	"8a251d975c3484cfc7d6eebb35b2e28360f6f4cdcf18502d2f642b73aad7b602": "react-vite template icon-192.png",
	"2970439bc43d56cd8ded8d2375dbe03c25d7d57d76fc0d4f5b72de9e3806fbaa": "react-vite template icon-512.png",
	"a3cdc03013621d05ec5e42aba24235ae452fd785f0adae153523386dee396255": "react-vite template og-image.png",
	"9e45b711f2317d1930df7595c9d49d437c81ac185aa9fff20738804eb9b8417b": "react-vite template logo.svg",
}

// ruleDeclaredIconTargets checks every target the scenario declares in its
// branding block: existence, format magic, exact dimensions and no known
// placeholder bytes, plus the index.html marker block and relative manifest
// srcs. WARNING severity — it reports, it does not gate.
func ruleDeclaredIconTargets(c *scanContext) (Finding, bool) {
	serviceJSON, present := c.read(".vrooli/service.json")
	if !present {
		return Finding{}, false
	}
	brandSlug, targetIDs, err := profiles.Declared([]byte(serviceJSON))
	if err != nil {
		// No branding block is not a defect: most scenarios have not adopted the
		// pipeline yet, and the rule must not add fleet-wide noise for them.
		if errors.Is(err, profiles.ErrNoBranding) {
			return Finding{}, false
		}
		return Finding{}, false
	}

	var problems []string
	for _, id := range targetIDs {
		profile, ok := profiles.ByID(id)
		if !ok {
			continue
		}
		for _, target := range profile.Targets {
			if target.ID == "manifest" {
				continue
			}
			rel := path.Join(profile.Root, target.Path)
			data, ok := c.read(rel)
			if !ok {
				problems = append(problems, fmt.Sprintf("%s: missing", rel))
				continue
			}
			raw := []byte(data)
			if sum := sha256.Sum256(raw); placeholderHashes[hex.EncodeToString(sum[:])] != "" {
				problems = append(problems, fmt.Sprintf("%s: placeholder bytes", rel))
			}
			if target.Format == "png" {
				cfg, _, derr := image.DecodeConfig(bytes.NewReader(raw))
				if derr != nil {
					problems = append(problems, fmt.Sprintf("%s: not a valid PNG", rel))
					continue
				}
				if target.Width > 0 && (cfg.Width != target.Width || cfg.Height != target.Height) {
					problems = append(problems, fmt.Sprintf("%s: %dx%d, want %dx%d", rel, cfg.Width, cfg.Height, target.Width, target.Height))
				}
				if variantIsOpaque(target.Variant) {
					if !pngFullyOpaque(raw) {
						problems = append(problems, fmt.Sprintf("%s: %s variant must be fully opaque", rel, target.Variant))
					}
				}
				if target.Variant == profiles.VariantMaskable {
					if px, ok := maskableOutsideSafeZone(raw); ok && px > 0 {
						problems = append(problems, fmt.Sprintf("%s: %d pixel(s) of content fall outside the 0.40 safe zone", rel, px))
					}
				}
			}
			if target.Format == "svg" && !bytes.Contains(raw, []byte("<svg")) {
				problems = append(problems, fmt.Sprintf("%s: not an SVG", rel))
			}
			if target.Format == "ico" && !bytes.HasPrefix(raw, []byte{0x00, 0x00, 0x01, 0x00}) {
				problems = append(problems, fmt.Sprintf("%s: not an ICO", rel))
			}
			if target.Format == "icns" && !bytes.HasPrefix(raw, []byte("icns")) {
				problems = append(problems, fmt.Sprintf("%s: not an ICNS", rel))
			}
		}
		if profile.ID == profiles.WebPublicV1.ID {
			problems = append(problems, checkWebPublicWiring(c)...)
		}
	}
	if len(problems) == 0 {
		return Finding{}, false
	}
	return Finding{
		Severity:               SeverityWarning,
		Title:                  "Declared icon targets are incomplete or wrong",
		Description:            fmt.Sprintf("Scenario %q declares branding for brand %q but %d target problem(s) were found.", c.scenario, brandSlug, len(problems)),
		FilePath:               ".vrooli/service.json",
		WhyItMatters:           "Mis-sized, missing or placeholder icons ship an unprofessional identity and break install/desktop surfaces.",
		RecommendedRemediation: "Run `brand-manager apply run --brand-id <brand> --scenario <scenario> --elements icons`.",
		Evidence:               map[string]any{"problems": problems, "brand": brandSlug, "targets": targetIDs},
	}, true
}

// variantIsOpaque reports whether a variant must have no transparent pixels.
// Rounded and vector variants intentionally keep translucent corners.
func variantIsOpaque(v profiles.Variant) bool {
	return v == profiles.VariantFullBleed || v == profiles.VariantMaskable || v == profiles.VariantSocialCard
}

// pngFullyOpaque decodes PNG bytes and reports whether every pixel is opaque.
// A decode failure is reported as opaque so the caller's format/dimension check
// owns that error instead of double-reporting.
func pngFullyOpaque(raw []byte) bool {
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return true
	}
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a < 0xffff {
				return false
			}
		}
	}
	return true
}

// maskableOutsideSafeZone decodes a maskable PNG and counts pixels whose colour
// differs from the corner background by more than 16 levels but that lie
// outside the W3C maskable safe zone: a radius of 0.40 x edge around the centre,
// with a 1 px tolerance. It returns (count, true) when the image decoded, so a
// decode failure is not treated as a safe-zone violation.
func maskableOutsideSafeZone(raw []byte) (int, bool) {
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return 0, false
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		return 0, true
	}
	corners := []color.NRGBA{
		color.NRGBAModel.Convert(img.At(b.Min.X, b.Min.Y)).(color.NRGBA),
		color.NRGBAModel.Convert(img.At(b.Max.X-1, b.Min.Y)).(color.NRGBA),
		color.NRGBAModel.Convert(img.At(b.Min.X, b.Max.Y-1)).(color.NRGBA),
		color.NRGBAModel.Convert(img.At(b.Max.X-1, b.Max.Y-1)).(color.NRGBA),
	}
	var bg color.NRGBA
	for _, c := range corners {
		bg.R += c.R / 4
		bg.G += c.G / 4
		bg.B += c.B / 4
	}
	centerX := float64(b.Min.X) + float64(w)/2
	centerY := float64(b.Min.Y) + float64(h)/2
	radius := 0.40*float64(w) + 1
	var outside int
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			if maxChannelDiff(c, bg) <= 16 {
				continue
			}
			dx := float64(x) - centerX
			dy := float64(y) - centerY
			if math.Hypot(dx, dy) > radius {
				outside++
			}
		}
	}
	return outside, true
}

func maxChannelDiff(a, b color.NRGBA) int {
	diff := func(x, y uint8) int {
		d := int(x) - int(y)
		if d < 0 {
			return -d
		}
		return d
	}
	best := diff(a.R, b.R)
	if v := diff(a.G, b.G); v > best {
		best = v
	}
	if v := diff(a.B, b.B); v > best {
		best = v
	}
	return best
}

// checkWebPublicWiring checks the index.html marker block and the manifest srcs.
func checkWebPublicWiring(c *scanContext) []string {
	var problems []string
	html, ok := c.read(indexHTMLRel)
	if ok {
		if strings.Count(html, "brand-manager:icons:start") != 1 || strings.Count(html, "brand-manager:icons:end") != 1 {
			problems = append(problems, indexHTMLRel+": missing or duplicated brand-manager icon marker block")
		}
	}
	manifestRel := path.Join(profiles.WebPublicV1.Root, "site.webmanifest")
	if raw, ok := c.read(manifestRel); ok {
		var doc struct {
			Icons []struct {
				Src string `json:"src"`
			} `json:"icons"`
		}
		if jerr := json.Unmarshal([]byte(raw), &doc); jerr == nil {
			for _, ic := range doc.Icons {
				if strings.HasPrefix(ic.Src, "/") || ic.Src == "" {
					problems = append(problems, fmt.Sprintf("%s: icon src %q must be relative", manifestRel, ic.Src))
				}
			}
		}
	}
	return problems
}
