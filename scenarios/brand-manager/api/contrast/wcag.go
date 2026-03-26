// Package contrast implements WCAG 2.1 AA contrast ratio calculation and validation.
// [REQ:BM-REQ-WCAG-CALC] [REQ:BM-REQ-WCAG-VALIDATE] [REQ:BM-REQ-WCAG-REJECT]
//
// DOC: docs/reference/api-endpoints.md#contrast
// DOC: docs/reference/configuration.md#wcag-contrast
package contrast

import (
	"fmt"
	"math"
	"strings"

	"brand-manager/config"
)

// DefaultThresholds are the WCAG AA minimum contrast ratios (spec values).
const (
	DefaultAANormalText = 4.5
	DefaultAALargeText  = 3.0
)

// RGB represents a color in the sRGB color space with components in [0, 1].
type RGB struct {
	R, G, B float64
}

// PairResult holds the contrast check result for a single color pairing.
type PairResult struct {
	Foreground string  `json:"foreground"`
	Background string  `json:"background"`
	Ratio      float64 `json:"ratio"`
	AANormal   bool    `json:"aa_normal"`
	AALarge    bool    `json:"aa_large"`
}

// BrandCheckResult holds the full WCAG check result for a brand's color palette.
type BrandCheckResult struct {
	Pairs   []PairResult `json:"pairs"`
	PassAll bool         `json:"pass_all"`
}

// Checker holds tunable thresholds for WCAG contrast validation.
type Checker struct {
	AANormal  float64
	AALarge   float64
	Precision int
}

// NewChecker creates a Checker from Config values.
func NewChecker(cfg config.Config) *Checker {
	return &Checker{
		AANormal:  cfg.ContrastAANormal,
		AALarge:   cfg.ContrastAALarge,
		Precision: cfg.ContrastPrecision,
	}
}

// DefaultChecker returns a Checker with WCAG AA spec defaults.
func DefaultChecker() *Checker {
	return NewChecker(config.Default())
}

// ParseHex converts a hex color string (#RGB, #RRGGBB) to RGB.
// Returns an error for invalid formats.
func ParseHex(hex string) (RGB, error) {
	hex = strings.TrimPrefix(hex, "#")
	var r, g, b uint8
	switch len(hex) {
	case 3:
		_, err := fmt.Sscanf(hex, "%1x%1x%1x", &r, &g, &b)
		if err != nil {
			return RGB{}, fmt.Errorf("invalid hex color: #%s", hex)
		}
		r = r*16 + r
		g = g*16 + g
		b = b*16 + b
	case 6:
		_, err := fmt.Sscanf(hex, "%2x%2x%2x", &r, &g, &b)
		if err != nil {
			return RGB{}, fmt.Errorf("invalid hex color: #%s", hex)
		}
	default:
		return RGB{}, fmt.Errorf("invalid hex color length: #%s", hex)
	}
	return RGB{
		R: float64(r) / 255.0,
		G: float64(g) / 255.0,
		B: float64(b) / 255.0,
	}, nil
}

// RelativeLuminance calculates the relative luminance of an sRGB color
// per WCAG 2.1 definition (https://www.w3.org/TR/WCAG21/#dfn-relative-luminance).
func RelativeLuminance(c RGB) float64 {
	linearize := func(v float64) float64 {
		if v <= 0.04045 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*linearize(c.R) + 0.7152*linearize(c.G) + 0.0722*linearize(c.B)
}

// Ratio calculates the contrast ratio between two colors per WCAG 2.1.
// The result is always >= 1.0, with 21:1 being the maximum (black on white).
func Ratio(fg, bg RGB) float64 {
	l1 := RelativeLuminance(fg)
	l2 := RelativeLuminance(bg)
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}

// roundTo rounds a float to the specified number of decimal places.
func roundTo(v float64, decimals int) float64 {
	pow := math.Pow(10, float64(decimals))
	return math.Round(v*pow) / pow
}

// CheckPair evaluates contrast between two hex colors using default thresholds.
func CheckPair(fgHex, bgHex string) (PairResult, error) {
	return DefaultChecker().CheckPair(fgHex, bgHex)
}

// CheckPair evaluates contrast between two hex colors using the checker's thresholds.
func (ch *Checker) CheckPair(fgHex, bgHex string) (PairResult, error) {
	fg, err := ParseHex(fgHex)
	if err != nil {
		return PairResult{}, err
	}
	bg, err := ParseHex(bgHex)
	if err != nil {
		return PairResult{}, err
	}
	ratio := roundTo(Ratio(fg, bg), ch.Precision)
	return PairResult{
		Foreground: fgHex,
		Background: bgHex,
		Ratio:      ratio,
		AANormal:   ratio >= ch.AANormal,
		AALarge:    ratio >= ch.AALarge,
	}, nil
}

// CheckBrandColors validates the standard WCAG AA pairings for a brand color set
// using default thresholds.
func CheckBrandColors(primary, secondary, accent, background, surface, text string) (*BrandCheckResult, error) {
	return DefaultChecker().CheckBrandColors(primary, secondary, accent, background, surface, text)
}

// ColorRole identifies a named color in a brand palette, used to define
// which foreground/background pairings are checked for WCAG compliance.
type ColorRole string

const (
	RolePrimary    ColorRole = "primary"
	RoleSecondary  ColorRole = "secondary"
	RoleAccent     ColorRole = "accent"
	RoleBackground ColorRole = "background"
	RoleSurface    ColorRole = "surface"
	RoleText       ColorRole = "text"
)

// StandardPairing defines a foreground/background role combination that must
// meet WCAG AA contrast. Extend this slice to check additional combinations.
type StandardPairing struct {
	Foreground ColorRole
	Background ColorRole
}

// StandardPairings lists the WCAG AA pairings checked for every brand.
// These represent the most common text-on-surface combinations a user will see.
// To add a new pairing (e.g. secondary-on-surface), append here — no other
// code changes are needed.
var StandardPairings = []StandardPairing{
	{RoleText, RoleBackground},    // body text readability
	{RoleText, RoleSurface},       // card/panel text readability
	{RolePrimary, RoleBackground}, // primary buttons/links on page
	{RolePrimary, RoleSurface},    // primary buttons/links on cards
	{RoleAccent, RoleBackground},  // accent highlights on page
}

// resolveColor maps a ColorRole to the corresponding hex value from the palette.
func resolveColor(role ColorRole, primary, secondary, accent, background, surface, text string) string {
	switch role {
	case RolePrimary:
		return primary
	case RoleSecondary:
		return secondary
	case RoleAccent:
		return accent
	case RoleBackground:
		return background
	case RoleSurface:
		return surface
	case RoleText:
		return text
	default:
		return ""
	}
}

// CheckBrandColors validates the standard WCAG AA pairings for a brand color set.
// The pairings checked are defined by StandardPairings.
func (ch *Checker) CheckBrandColors(primary, secondary, accent, background, surface, text string) (*BrandCheckResult, error) {
	type pair struct{ fg, bg string }

	// Resolve role-based pairings to concrete hex values, skipping empty colors
	var activePairs []pair
	for _, sp := range StandardPairings {
		fg := resolveColor(sp.Foreground, primary, secondary, accent, background, surface, text)
		bg := resolveColor(sp.Background, primary, secondary, accent, background, surface, text)
		if fg != "" && bg != "" {
			activePairs = append(activePairs, pair{fg, bg})
		}
	}

	if len(activePairs) == 0 {
		return &BrandCheckResult{PassAll: true}, nil
	}

	result := &BrandCheckResult{PassAll: true}
	for _, p := range activePairs {
		pr, err := ch.CheckPair(p.fg, p.bg)
		if err != nil {
			return nil, err
		}
		result.Pairs = append(result.Pairs, pr)
		if !pr.AANormal {
			result.PassAll = false
		}
	}
	return result, nil
}
