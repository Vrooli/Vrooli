// Package pagescheme reads a Notification Hub decision page the way the Hub's
// sealed frame shows it: which colour schemes it is shown in, and which text the
// operator cannot read there. The Hub's packet check and the page tools
// (decisionpage) share it, so a preview and the check agree.
package pagescheme

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Page colour schemes. The Hub shows a page in its own light or dark theme,
// and the starter kit colours the page from tokens that follow that theme. A
// page that paints with literal colours (white cards, a light page background,
// dark text) and has no dark styles of its own only works in one scheme: shown
// in the other, the kit's text colour lands on the page's own background. The
// sealed frame therefore shows such a page in the scheme it was written for
// (the Hub's ui/src/features/packet/explainer/theme.ts makes the same call from
// the same rules), and the packet check reads the page in every scheme it will
// be shown in (Kit.Unreadable).
//
// The rules, in order:
//
//  1. A color-scheme the page declares (meta name="color-scheme", or the
//     color-scheme property on :root/html/body) decides: "light", "dark" or
//     both.
//  2. A page with styles of its own for dark (a prefers-color-scheme media
//     block, a [data-theme="dark"] selector, or a script that follows
//     hubthemechange) follows the Hub's theme.
//  3. A literal background or text colour on :root, html or body decides by its
//     lightness.
//  4. Otherwise the literal colours that are not paired (a background with no
//     text colour in the same rule, or a text colour with no background) vote:
//     a light background or dark text votes light, the reverse votes dark.
//  5. A page with neither (only the kit's var(--hub-*) tokens, or paired
//     colours, or none) follows the Hub's theme.
const (
	SchemeBoth  = "both"
	SchemeLight = "light"
	SchemeDark  = "dark"
)

// rgb is an opaque colour, channels 0..255.
type rgb struct{ r, g, b float64 }

func (c rgb) hex() string {
	clamp := func(v float64) int { return int(math.Max(0, math.Min(255, math.Round(v)))) }
	const digits = "0123456789abcdef"
	out := []byte{'#'}
	for _, v := range []int{clamp(c.r), clamp(c.g), clamp(c.b)} {
		out = append(out, digits[v>>4], digits[v&15])
	}
	return string(out)
}

// luminance is the WCAG relative luminance.
func (c rgb) luminance() float64 {
	channel := func(v float64) float64 {
		v /= 255
		if v <= 0.03928 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*channel(c.r) + 0.7152*channel(c.g) + 0.0722*channel(c.b)
}

// isLight reports a colour a reader sees as light (a background for dark text).
func (c rgb) isLight() bool { return c.luminance() > 0.18 }

func contrast(a, b rgb) float64 {
	la, lb := a.luminance(), b.luminance()
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// over composites a colour with alpha over a base.
func over(c rgb, alpha float64, base rgb) rgb {
	return rgb{c.r*alpha + base.r*(1-alpha), c.g*alpha + base.g*(1-alpha), c.b*alpha + base.b*(1-alpha)}
}

// cssRule is one style rule of the page: its selector, its declarations
// (property -> value, lower-case property names), and the scheme it applies in
// ("" for both).
type cssRule struct {
	selector string
	decls    map[string]string
	scheme   string
	// inline marks a style attribute (selector `tag[style="#n"]`): it counts for
	// the scheme, and the cascade reads it from the element instead.
	inline bool
}

// pageStyles is what the theme rules read from a page.
type pageStyles struct {
	// kit are the starter kit's rules, which the frame applies under the page's.
	kit   []cssRule
	rules []cssRule
	// declared is the color-scheme the page declares ("" when none).
	declared string
	// darkStyles is true when the page styles dark itself.
	darkStyles bool
}

var (
	cssCommentPattern  = regexp.MustCompile(`(?s)/\*.*?\*/`)
	styleBlockPattern  = regexp.MustCompile(`(?is)<style\b[^>]*>(.*?)</style\s*>`)
	metaSchemePattern  = regexp.MustCompile(`(?is)<meta\b[^>]*\bname\s*=\s*["']?color-scheme\b[^>]*>`)
	themeScriptPattern = regexp.MustCompile(`hubthemechange|prefers-color-scheme`)
	darkSelector       = regexp.MustCompile(`(?i)\[data-theme\s*=\s*["']?dark["']?\]|\.dark\b|\.theme-dark\b`)
	lightSelector      = regexp.MustCompile(`(?i)\[data-theme\s*=\s*["']?light["']?\]`)
	rootSelector       = regexp.MustCompile(`(?i)^(?:html|:root|body)$`)
	notPattern         = regexp.MustCompile(`:not\([^()]*\)`)
	cssURLValuePattern = regexp.MustCompile(`url\([^)]*\)`)
)

// readPageStyles parses the page's style blocks and style attributes.
func (k *Kit) readPageStyles(page string) pageStyles {
	page = commentPattern.ReplaceAllString(page, "")
	styles := pageStyles{kit: k.rules}
	for _, block := range styleBlockPattern.FindAllStringSubmatch(page, -1) {
		styles.rules = append(styles.rules, parseCSS(cssCommentPattern.ReplaceAllString(block[1], ""), "")...)
	}
	markup := scriptBodyPattern.ReplaceAllString(styleBodyPattern.ReplaceAllString(page, "$1$2"), "$1$2")
	// Each style attribute is its own rule: two elements' attributes never pair up.
	inline := 0
	for _, tag := range tagPattern.FindAllStringSubmatch(markup, -1) {
		attrs := attributes(tag[2])
		if style, ok := attrs["style"]; ok && strings.TrimSpace(style) != "" {
			selector := fmt.Sprintf(`%s[style="#%d"]`, strings.ToLower(tag[1]), inline)
			inline++
			styles.rules = append(styles.rules, cssRule{selector: selector, decls: parseDeclarations(style), inline: true})
		}
	}
	if meta := metaSchemePattern.FindString(markup); meta != "" {
		styles.declared = schemeWords(attributes(meta)["content"])
	}
	for _, rule := range styles.rules {
		if rule.scheme == SchemeDark {
			styles.darkStyles = true
		}
		if value, ok := rule.decls["color-scheme"]; ok && rule.scheme == "" && styles.declared == "" && isRootRule(rule.selector) {
			styles.declared = schemeWords(value)
		}
	}
	for _, block := range scriptBlockPattern.FindAllStringSubmatch(page, -1) {
		if themeScriptPattern.MatchString(block[1]) {
			styles.darkStyles = true
		}
	}
	return styles
}

// schemeWords reads a color-scheme value: "light", "dark", SchemeBoth or "".
func schemeWords(value string) string {
	value = strings.ToLower(value)
	light, dark := strings.Contains(value, "light"), strings.Contains(value, "dark")
	switch {
	case light && dark:
		return SchemeBoth
	case light:
		return SchemeLight
	case dark:
		return SchemeDark
	}
	return ""
}

// parseCSS reads rules from a stylesheet. Media blocks that test
// prefers-color-scheme set the scheme of the rules inside; other media,
// @supports and @layer blocks are read as always applying; other at-rules
// (@keyframes, @font-face) are skipped.
func parseCSS(css, scheme string) []cssRule {
	var rules []cssRule
	for i := 0; i < len(css); {
		open := strings.IndexByte(css[i:], '{')
		if open < 0 {
			break
		}
		prelude := strings.TrimSpace(css[i : i+open])
		// An at-rule with no block (@import, @charset) ends at its semicolon.
		if semi := strings.LastIndexByte(prelude, ';'); semi >= 0 {
			prelude = strings.TrimSpace(prelude[semi+1:])
		}
		body, end := blockBody(css, i+open)
		i = end
		lower := strings.ToLower(prelude)
		switch {
		case strings.HasPrefix(lower, "@media"):
			inner := scheme
			if strings.Contains(lower, "prefers-color-scheme") {
				if strings.Contains(lower, "dark") {
					inner = SchemeDark
				} else {
					inner = SchemeLight
				}
			}
			rules = append(rules, parseCSS(body, inner)...)
		case strings.HasPrefix(lower, "@supports"), strings.HasPrefix(lower, "@layer"), strings.HasPrefix(lower, "@container"):
			rules = append(rules, parseCSS(body, scheme)...)
		case strings.HasPrefix(lower, "@"):
		default:
			for _, selector := range splitTop(prelude, ',') {
				selector = strings.Join(strings.Fields(selector), " ")
				if selector == "" {
					continue
				}
				ruleScheme := scheme
				if ruleScheme == "" {
					ruleScheme = selectorScheme(selector)
				}
				rules = append(rules, cssRule{selector: selector, decls: parseDeclarations(body), scheme: ruleScheme})
			}
		}
	}
	return rules
}

// blockBody returns the text inside the block that opens at css[open] and the
// index after its closing brace.
func blockBody(css string, open int) (string, int) {
	depth := 0
	for j := open; j < len(css); j++ {
		switch css[j] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return css[open+1 : j], j + 1
			}
		}
	}
	return css[open+1:], len(css)
}

// splitTop splits text on sep outside parentheses, brackets and quotes.
func splitTop(text string, sep byte) []string {
	var parts []string
	depth, start := 0, 0
	var quote byte
	for j := 0; j < len(text); j++ {
		c := text[j]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '(' || c == '[':
			depth++
		case c == ')' || c == ']':
			depth--
		case c == sep && depth == 0:
			parts = append(parts, text[start:j])
			start = j + 1
		}
	}
	return append(parts, text[start:])
}

func parseDeclarations(body string) map[string]string {
	decls := map[string]string{}
	for _, decl := range splitTop(body, ';') {
		name, value, ok := strings.Cut(decl, ":")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		if !strings.HasPrefix(name, "--") {
			name = strings.ToLower(name)
		}
		value = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(value), "!important"))
		if name != "" && value != "" {
			decls[name] = strings.TrimSpace(value)
		}
	}
	return decls
}

// selectorScheme is the scheme a selector limits its rule to: dark for
// [data-theme="dark"] (or a .dark class), light for [data-theme="light"], ""
// otherwise. A test inside :not() limits nothing.
func selectorScheme(selector string) string {
	selector = notPattern.ReplaceAllString(selector, "")
	switch {
	case darkSelector.MatchString(selector):
		return SchemeDark
	case lightSelector.MatchString(selector):
		return SchemeLight
	}
	return ""
}

func isRootRule(selector string) bool {
	selector = notPattern.ReplaceAllString(selector, "")
	selector = darkSelector.ReplaceAllString(lightSelector.ReplaceAllString(selector, ""), "")
	return rootSelector.MatchString(strings.TrimSpace(selector))
}

// --- colours -----------------------------------------------------------------

var (
	hexColorPattern  = regexp.MustCompile(`(?i)#(?:[0-9a-f]{3,4}|[0-9a-f]{6}|[0-9a-f]{8})\b`)
	funcColorPattern = regexp.MustCompile(`(?i)\b(rgba?|hsla?)\(([^()]*)\)`)
	varPattern       = regexp.MustCompile(`var\(\s*(--[A-Za-z0-9_-]+)\s*(?:,\s*([^()]*(?:\([^()]*\))?[^()]*))?\)`)
	wordPattern      = regexp.MustCompile(`(?i)\b[a-z]+\b`)
)

// namedColors are the CSS colour names a page is likely to use.
var namedColors = map[string]string{
	"white": "#ffffff", "black": "#000000", "snow": "#fffafa", "ivory": "#fffff0", "whitesmoke": "#f5f5f5",
	"ghostwhite": "#f8f8ff", "aliceblue": "#f0f8ff", "azure": "#f0ffff", "mintcream": "#f5fffa", "honeydew": "#f0fff0",
	"seashell": "#fff5ee", "floralwhite": "#fffaf0", "oldlace": "#fdf5e6", "linen": "#faf0e6", "beige": "#f5f5dc",
	"lavender": "#e6e6fa", "lightyellow": "#ffffe0", "cornsilk": "#fff8dc", "papayawhip": "#ffefd5", "gainsboro": "#dcdcdc",
	"lightgray": "#d3d3d3", "lightgrey": "#d3d3d3", "silver": "#c0c0c0", "darkgray": "#a9a9a9", "darkgrey": "#a9a9a9",
	"gray": "#808080", "grey": "#808080", "dimgray": "#696969", "dimgrey": "#696969", "slategray": "#708090",
	"darkslategray": "#2f4f4f", "red": "#ff0000", "darkred": "#8b0000", "maroon": "#800000", "green": "#008000",
	"darkgreen": "#006400", "blue": "#0000ff", "navy": "#000080", "darkblue": "#00008b", "midnightblue": "#191970",
	"yellow": "#ffff00", "orange": "#ffa500", "purple": "#800080", "teal": "#008080", "olive": "#808000",
	"lime": "#00ff00", "aqua": "#00ffff", "cyan": "#00ffff", "fuchsia": "#ff00ff", "magenta": "#ff00ff",
	"lightblue": "#add8e6", "lightgreen": "#90ee90", "pink": "#ffc0cb", "lightpink": "#ffb6c1", "wheat": "#f5deb3",
}

// parsedColor is a colour found in a value: the colour and its alpha.
type parsedColor struct {
	c     rgb
	alpha float64
}

// firstColor finds the first literal colour in a value (a shorthand like
// "background: #fff url(x.png)" or a gradient's first stop counts).
func firstColor(value string) (parsedColor, bool) {
	if mix := strings.Index(strings.ToLower(value), "color-mix("); mix >= 0 {
		if color, ok := firstColor(value[:mix]); ok {
			return color, true
		}
		body, _ := parenBody(value, mix+len("color-mix"))
		return mixColor(body)
	}
	best, found, at := parsedColor{}, false, len(value)+1
	if loc := hexColorPattern.FindStringIndex(value); loc != nil && loc[0] < at {
		if c, ok := parseHex(value[loc[0]:loc[1]]); ok {
			best, found, at = c, true, loc[0]
		}
	}
	if loc := funcColorPattern.FindStringSubmatchIndex(value); loc != nil && loc[0] < at {
		if c, ok := parseFuncColor(strings.ToLower(value[loc[2]:loc[3]]), value[loc[4]:loc[5]]); ok {
			best, found, at = c, true, loc[0]
		}
	}
	stripped := funcColorPattern.ReplaceAllStringFunc(value, func(s string) string { return strings.Repeat(" ", len(s)) })
	stripped = cssURLValuePattern.ReplaceAllStringFunc(stripped, func(s string) string { return strings.Repeat(" ", len(s)) })
	for _, loc := range wordPattern.FindAllStringIndex(stripped, -1) {
		if loc[0] >= at {
			break
		}
		word := strings.ToLower(stripped[loc[0]:loc[1]])
		if word == "transparent" {
			return parsedColor{alpha: 0}, true
		}
		if hex, ok := namedColors[word]; ok {
			c, _ := parseHex(hex)
			return c, true
		}
	}
	return best, found
}

// mixColor evaluates the arguments of color-mix(in <space>, A p%, B q%) in
// sRGB: the two colours weighted by their percentages (50% each when neither
// is given), with alpha premultiplied, so mixing with transparent gives a
// translucent colour.
func mixColor(body string) (parsedColor, bool) {
	parts := splitTop(body, ',')
	if len(parts) != 3 {
		return parsedColor{}, false
	}
	read := func(part string) (parsedColor, float64, bool) {
		part = strings.TrimSpace(part)
		weight := -1.0
		if fields := strings.Fields(part); len(fields) > 1 && strings.HasSuffix(fields[len(fields)-1], "%") {
			if v, err := strconv.ParseFloat(strings.TrimSuffix(fields[len(fields)-1], "%"), 64); err == nil {
				weight = v / 100
				part = strings.TrimSpace(strings.TrimSuffix(part, fields[len(fields)-1]))
			}
		}
		color, ok := firstColor(part)
		return color, weight, ok
	}
	a, wa, okA := read(parts[1])
	b, wb, okB := read(parts[2])
	if !okA || !okB {
		return parsedColor{}, false
	}
	switch {
	case wa < 0 && wb < 0:
		wa, wb = 0.5, 0.5
	case wa < 0:
		wa = 1 - wb
	case wb < 0:
		wb = 1 - wa
	}
	if total := wa + wb; total > 0 {
		wa, wb = wa/total, wb/total
	}
	alpha := a.alpha*wa + b.alpha*wb
	if alpha <= 0 {
		return parsedColor{alpha: 0}, true
	}
	mix := func(x, y float64) float64 { return (x*a.alpha*wa + y*b.alpha*wb) / alpha }
	return parsedColor{c: rgb{mix(a.c.r, b.c.r), mix(a.c.g, b.c.g), mix(a.c.b, b.c.b)}, alpha: alpha}, true
}

func parseHex(text string) (parsedColor, bool) {
	text = strings.TrimPrefix(text, "#")
	if len(text) == 3 || len(text) == 4 {
		expanded := make([]byte, 0, 8)
		for i := 0; i < len(text); i++ {
			expanded = append(expanded, text[i], text[i])
		}
		text = string(expanded)
	}
	if len(text) != 6 && len(text) != 8 {
		return parsedColor{}, false
	}
	value, err := strconv.ParseUint(text, 16, 32)
	if err != nil {
		return parsedColor{}, false
	}
	alpha := 1.0
	if len(text) == 8 {
		alpha = float64(value&0xff) / 255
		value >>= 8
	}
	return parsedColor{c: rgb{float64(value >> 16 & 0xff), float64(value >> 8 & 0xff), float64(value & 0xff)}, alpha: alpha}, true
}

func parseFuncColor(name, args string) (parsedColor, bool) {
	fields := strings.FieldsFunc(args, func(r rune) bool { return r == ',' || r == ' ' || r == '/' })
	if len(fields) < 3 {
		return parsedColor{}, false
	}
	number := func(field string, scale float64) (float64, bool) {
		if strings.HasSuffix(field, "%") {
			v, err := strconv.ParseFloat(strings.TrimSuffix(field, "%"), 64)
			return v / 100 * scale, err == nil
		}
		v, err := strconv.ParseFloat(strings.TrimSuffix(field, "deg"), 64)
		return v, err == nil
	}
	alpha := 1.0
	if len(fields) >= 4 {
		if a, ok := number(fields[3], 1); ok {
			alpha = a
		}
	}
	if strings.HasPrefix(name, "rgb") {
		r, ok1 := number(fields[0], 255)
		g, ok2 := number(fields[1], 255)
		b, ok3 := number(fields[2], 255)
		return parsedColor{c: rgb{r, g, b}, alpha: alpha}, ok1 && ok2 && ok3
	}
	h, ok1 := number(fields[0], 360)
	s, ok2 := number(fields[1], 1)
	l, ok3 := number(fields[2], 1)
	if !strings.HasSuffix(fields[1], "%") {
		s /= 100
	}
	if !strings.HasSuffix(fields[2], "%") {
		l /= 100
	}
	return parsedColor{c: hslToRGB(h, s, l), alpha: alpha}, ok1 && ok2 && ok3
}

func hslToRGB(h, s, l float64) rgb {
	h = math.Mod(math.Mod(h, 360)+360, 360) / 360
	if s == 0 {
		return rgb{l * 255, l * 255, l * 255}
	}
	q := l * (1 + s)
	if l >= 0.5 {
		q = l + s - l*s
	}
	p := 2*l - q
	hue := func(t float64) float64 {
		t = math.Mod(t+1, 1)
		switch {
		case t < 1.0/6:
			return p + (q-p)*6*t
		case t < 0.5:
			return q
		case t < 2.0/3:
			return p + (q-p)*(2.0/3-t)*6
		}
		return p
	}
	return rgb{hue(h+1.0/3) * 255, hue(h) * 255, hue(h-1.0/3) * 255}
}

// --- resolving a value in a scheme ------------------------------------------

// themeVars are the custom properties in force in one scheme: the kit's
// tokens, overridden by what the page defines on :root/html/body for that
// scheme.
func themeVars(styles pageStyles, scheme string) map[string]string {
	vars := map[string]string{}
	// The kit's rules, then the page's: for both schemes first, then the
	// scheme's own, so the later ones win as they do in the frame.
	for _, rules := range [][]cssRule{styles.kit, styles.rules} {
		for _, pass := range []string{"", scheme} {
			for _, rule := range rules {
				if rule.scheme != pass || !isRootRule(rule.selector) {
					continue
				}
				for name, value := range rule.decls {
					if strings.HasPrefix(name, "--") {
						vars[name] = value
					}
				}
			}
		}
	}
	return vars
}

// resolve substitutes custom properties in value (to a depth of four).
func resolve(value string, vars map[string]string) (string, bool) {
	for depth := 0; depth < 4 && strings.Contains(value, "var("); depth++ {
		ok := true
		value = varPattern.ReplaceAllStringFunc(value, func(match string) string {
			parts := varPattern.FindStringSubmatch(match)
			if v, found := vars[parts[1]]; found {
				return v
			}
			if parts[2] != "" {
				return parts[2]
			}
			ok = false
			return ""
		})
		if !ok {
			return value, false
		}
	}
	return value, !strings.Contains(value, "var(")
}

// usesThemeVar reports whether value depends on a custom property whose value
// differs between the schemes (a --hub-* token, or a page variable the page
// sets per scheme).
func usesThemeVar(value string, light, dark map[string]string) bool {
	for _, match := range varPattern.FindAllStringSubmatch(value, -1) {
		if light[match[1]] != dark[match[1]] {
			return true
		}
	}
	return false
}

// backgroundOf is the rule's background colour declaration, if any.
func backgroundOf(decls map[string]string) (string, bool) {
	if value, ok := decls["background-color"]; ok {
		return value, true
	}
	value, ok := decls["background"]
	return value, ok
}

// Kit is the starter kit stylesheet the sealed frame injects into every page.
// The theme rules read its tokens and rules, so a page is judged with the
// colours the operator sees.
type Kit struct{ rules []cssRule }

// NewKit parses the kit stylesheet (the Hub's
// ui/src/features/packet/explainer/kit/kit.css).
func NewKit(css string) *Kit {
	return &Kit{rules: parseCSS(cssCommentPattern.ReplaceAllString(css, ""), "")}
}

// Scheme says which schemes the Hub shows a page in: SchemeBoth (it follows the
// Hub's theme), SchemeLight or SchemeDark. See the rules above.
func (k *Kit) Scheme(page string) string {
	return k.readPageStyles(page).scheme()
}

func (styles pageStyles) scheme() string {
	if styles.declared != "" {
		return styles.declared
	}
	if styles.darkStyles {
		return SchemeBoth
	}
	light, dark := themeVars(styles, SchemeLight), themeVars(styles, SchemeDark)
	literal := func(value string) (parsedColor, bool) {
		if usesThemeVar(value, light, dark) {
			return parsedColor{}, false
		}
		resolved, ok := resolve(value, light)
		if !ok {
			return parsedColor{}, false
		}
		color, found := firstColor(resolved)
		return color, found && color.alpha >= 0.5
	}
	// 3. The root's own colours.
	for _, rule := range styles.rules {
		if rule.scheme != "" || !isRootRule(rule.selector) {
			continue
		}
		if value, ok := backgroundOf(rule.decls); ok {
			if color, found := literal(value); found {
				return lightOrDark(color.c.isLight())
			}
		}
	}
	for _, rule := range styles.rules {
		if rule.scheme != "" || !isRootRule(rule.selector) {
			continue
		}
		if value, ok := rule.decls["color"]; ok {
			if color, found := literal(value); found {
				return lightOrDark(!color.c.isLight())
			}
		}
	}
	// 4. Unpaired literal colours vote.
	votes := 0
	for _, rule := range mergeRules(styles.rules, "") {
		bg, hasBG := backgroundOf(rule.decls)
		fg, hasFG := rule.decls["color"]
		switch {
		case hasBG && !hasFG:
			if color, found := literal(bg); found {
				votes += vote(color.c.isLight())
			}
		case hasFG && !hasBG:
			if color, found := literal(fg); found {
				votes += vote(!color.c.isLight())
			}
		}
	}
	switch {
	case votes > 0:
		return SchemeLight
	case votes < 0:
		return SchemeDark
	}
	return SchemeBoth
}

func lightOrDark(light bool) string {
	if light {
		return SchemeLight
	}
	return SchemeDark
}

func vote(light bool) int {
	if light {
		return 1
	}
	return -1
}

// mergeRules joins the rules that apply in scheme ("" = rules for both only)
// by selector, later declarations winning, so a background and a text colour
// set in two rules for the same selector count as a pair.
func mergeRules(rules []cssRule, scheme string) []cssRule {
	index := map[string]int{}
	var merged []cssRule
	for _, rule := range rules {
		if rule.scheme != "" && rule.scheme != scheme {
			continue
		}
		key := rule.selector
		if rule.scheme != "" {
			key = darkSelector.ReplaceAllString(lightSelector.ReplaceAllString(key, ""), "")
			key = strings.Join(strings.Fields(key), " ")
			if key == "" {
				key = ":root"
			}
		}
		at, seen := index[key]
		if !seen {
			index[key] = len(merged)
			merged = append(merged, cssRule{selector: key, decls: map[string]string{}})
			at = len(merged) - 1
		}
		for name, value := range rule.decls {
			merged[at].decls[name] = value
		}
	}
	return merged
}
