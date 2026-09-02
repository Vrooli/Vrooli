package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"react-component-library/internal/themes"
)

var additions = map[string]string{
	"--badge-border":                    "var(--color-border)",
	"--border-focus":                    "2px",
	"--border-medium":                   "2px",
	"--border-thin":                     "1px",
	"--color-accent-subtle":             "color-mix(in srgb, var(--color-accent) 14%, transparent)",
	"--color-border-strong":             "color-mix(in srgb, var(--color-border) 72%, var(--color-foreground))",
	"--color-danger-border":             "color-mix(in srgb, var(--color-danger) 38%, var(--color-border))",
	"--color-danger-foreground":         "color-mix(in srgb, var(--color-danger) 78%, var(--color-foreground))",
	"--color-danger-foreground-inverse": "var(--color-primary-foreground)",
	"--color-danger-subtle":             "color-mix(in srgb, var(--color-danger) 12%, var(--color-surface))",
	"--color-field":                     "var(--color-surface)",
	"--color-focus-ring":                "var(--color-focus)",
	"--color-on-primary":                "var(--color-primary-foreground)",
	"--color-overlay":                   "var(--color-shell)",
	"--color-primary-hover":             "color-mix(in srgb, var(--color-primary) 88%, var(--color-foreground))",
	"--color-primary-strong":            "var(--color-primary)",
	"--color-scrim":                     "color-mix(in srgb, var(--color-shell) 52%, transparent)",
	"--color-success-foreground":        "color-mix(in srgb, var(--color-success) 76%, var(--color-foreground))",
	"--color-surface-sunken":            "color-mix(in srgb, var(--color-surface-muted) 72%, var(--color-background))",
	"--color-warning-foreground":        "color-mix(in srgb, var(--color-warning) 72%, var(--color-foreground))",
	"--color-warning-subtle":            "color-mix(in srgb, var(--color-warning) 16%, var(--color-surface))",
	"--content-min-height":              "12rem",
	"--control-size-xs":                 "32px",
	"--control-size-sm":                 "36px",
	"--control-size-md":                 "40px",
	"--control-size-lg":                 "44px",
	"--control-size-xl":                 "48px",
	"--control-size-icon":               "40px",
	"--dur-enter":                       "var(--dur-quick)",
	"--elev-subtle":                     "0 1px 2px rgba(9, 18, 22, .06)",
	"--focus-ring-color":                "var(--color-focus)",
	"--focus-ring-offset":               "2px",
	"--focus-ring-width":                "2px",
	"--font-size-lg":                    "18px",
	"--font-size-sm":                    "14px",
	"--icon-size-xs":                    "12px",
	"--layer-alert":                     "700",
	"--layer-menu":                      "610",
	"--overlay-dialog-lg":               "48rem",
	"--overlay-dialog-md":               "36rem",
	"--overlay-dialog-sm":               "24rem",
	"--overlay-drawer-top-gap":          "32px",
	"--overlay-grabber-block":           "4px",
	"--overlay-grabber-inline":          "36px",
	"--overlay-menu-align":              "0px",
	"--radius-overlay":                  "1rem",
	"--space-4xl":                       "80px",
	"--space-4xs":                       "4px",
	"--text-body-sm":                    "400 var(--text-body-sm-size) / var(--text-body-sm-line) var(--font-sans)",
	"--text-heading-lg":                 "600 20px / 26px var(--font-sans)",
	"--text-heading-sm":                 "600 16px / 22px var(--font-sans)",
	"--text-subtitle":                   "600 var(--text-subheading-size) / var(--text-subheading-line) var(--font-sans)",
	"--text-subtitle-tracking":          "0",
	"--text-title-tracking":             "-.01em",
	"--text-xs":                         "12px",
	"--tracking-caps":                   ".08em",
	"--tracking-tight":                  "-.02em",
	"--weight-medium":                   "500",
	// Provenance encodings and the wall figure were promoted from Command Center
	// on 2026-09-01 and hand-added to Tokens 1.0.2 without a canonical home; the
	// ramp is where every kit inherits them from.
	"--provenance-measured": "var(--color-success)",
	"--provenance-cached":   "var(--color-warning)",
	"--provenance-sample":   "#b7a6ff",
	"--provenance-absent":   "var(--color-muted-foreground)",
	"--glow-primary":        "rgba(51,214,255,.5)",
	"--text-wall":           "clamp(5rem, 16vw, 20rem)",
}

var deadTokens = map[string]bool{
	"--radius-sm": true, "--radius-md": true, "--radius-lg": true,
	"--radius-xl": true, "--radius-none": true,
}

var (
	tokenDeclarationRE = regexp.MustCompile(`^(\s*)(--[A-Za-z0-9_-]+)\s*:\s*(.+);\s*$`)
	internalImportRE   = regexp.MustCompile(`@vrooli/react-component-library/([A-Za-z0-9_-]+)/([0-9]+\.[0-9]+\.[0-9]+)`)
)

func main() {
	rootFlag := flag.String("root", "", "repository root")
	check := flag.Bool("check", false, "fail when generated artifacts differ")
	bootstrapBase := flag.Bool("bootstrap-base", false, "initialize the shared base from the default kit's first :root block")
	baseStylesDraft := flag.String("base-styles-draft", "", "update the canonical-token managed region in a governed BaseStyles draft")
	tokensDraft := flag.String("tokens-draft", "", "write generated Tokens.tsx into a governed Tokens draft")
	normalizeDraft := flag.String("normalize-draft", "", "normalize token fallbacks in one governed component draft")
	flag.Parse()
	root, err := resolveRoot(*rootFlag)
	if err != nil {
		fail(err)
	}

	source := filepath.Join(root, "templates", "design", "_base", "tokens.css")
	if *bootstrapBase {
		source = filepath.Join(root, "templates", "design", "vrooli-default", "adapters", "react-vite-tailwind", "tokens.css")
	}
	rawSource, err := os.ReadFile(source)
	if err != nil {
		fail(err)
	}
	if *bootstrapBase {
		rawSource, err = firstRootBlock(rawSource)
		if err != nil {
			fail(err)
		}
	}
	tokens, err := themes.ParseTokenCSS(string(rawSource))
	if err != nil {
		fail(err)
	}
	byName := map[string]themes.DesignToken{}
	for _, token := range tokens {
		if !deadTokens[token.Name] {
			byName[token.Name] = token
		}
	}
	for name, value := range additions {
		if _, exists := byName[name]; !exists {
			byName[name] = themes.DesignToken{Name: name, Value: value}
		}
	}
	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)

	var css strings.Builder
	css.WriteString("/* Canonical shared vocabulary; run design-token-generate after editing. */\n:root {\n")
	for _, name := range names {
		token := byName[name]
		token.Tier = tierFor(name)
		fmt.Fprintf(&css, "  /* @tier %s */\n  %s: %s;\n", token.Tier, token.Name, token.Value)
		byName[name] = token
	}
	css.WriteString("}\n")

	var dictionary strings.Builder
	dictionary.WriteString("# Design Token Dictionary\n\nThis dictionary is generated from `templates/design/_base/tokens.css`. Kit files override values without changing token meaning or tier.\n\n| Property | Tier | Vrooli default | Meaning | Use when |\n|---|---|---|---|---|\n")
	for _, name := range names {
		token := byName[name]
		meaning, usage := describe(name)
		fmt.Fprintf(&dictionary, "| `%s` | %s | `%s` | %s | %s |\n", name, token.Tier, strings.ReplaceAll(token.Value, "|", "\\|"), meaning, usage)
	}

	outputs := map[string]string{
		filepath.Join(root, "templates", "design", "_base", "tokens.css"): css.String(),
		filepath.Join(root, "docs", "design", "TOKEN-DICTIONARY.md"):      dictionary.String(),
	}
	kits, err := listDesignKits(root)
	if err != nil {
		fail(err)
	}
	for _, kitID := range kits {
		tokensPath := filepath.Join(root, "templates", "design", kitID, "adapters", "react-vite-tailwind", "tokens.css")
		kitTokens, err := os.ReadFile(tokensPath)
		if err != nil {
			fail(err)
		}
		outputs[tokensPath] = normalizeKitTokens(string(kitTokens), byName)

		path := filepath.Join(root, "templates", "design", kitID, "DESIGN.md")
		current, err := os.ReadFile(path)
		if err != nil {
			fail(err)
		}
		normalized, err := normalizeDesignDocument(string(current))
		if err != nil {
			fail(fmt.Errorf("normalize %s: %w", path, err))
		}
		outputs[path] = normalized

		resolved, resolveErr := themes.ResolveKitTokens(root, kitID)
		if resolveErr != nil {
			fail(resolveErr)
		}
		theme, themeErr := generateTailwindTheme(resolved)
		if themeErr != nil {
			fail(themeErr)
		}
		outputs[filepath.Join(root, "templates", "design", kitID, "adapters", "react-vite-tailwind", "tailwind.theme.json")] = theme
	}
	generatedTokens := generateTokensSource(byName)
	if strings.TrimSpace(*tokensDraft) != "" {
		outputs[filepath.Join(strings.TrimSpace(*tokensDraft), "Tokens.tsx")] = generatedTokens
	} else if *check {
		latest, latestErr := latestTokensSourcePath(root)
		if latestErr != nil {
			fail(latestErr)
		}
		outputs[latest] = generatedTokens
	}
	if strings.TrimSpace(*baseStylesDraft) != "" {
		path, err := filepath.Abs(*baseStylesDraft)
		if err != nil {
			fail(err)
		}
		current, err := os.ReadFile(path)
		if err != nil {
			fail(err)
		}
		layer := "@layer rcl.tokens {\n" + indentCSSRoot(css.String()) + "}\n"
		updated, err := replaceManagedRegion(string(current), "/* rcl:canonical-tokens:begin */", "/* rcl:canonical-tokens:end */", layer)
		if err != nil {
			fail(err)
		}
		outputs[path] = updated
	}
	if strings.TrimSpace(*normalizeDraft) != "" {
		draftOutputs, err := normalizeComponentDraft(root, strings.TrimSpace(*normalizeDraft), byName)
		if err != nil {
			fail(err)
		}
		for path, content := range draftOutputs {
			outputs[path] = content
		}
	}
	changed := false
	for path, content := range outputs {
		current, _ := os.ReadFile(path)
		if generatedArtifactMatches(path, string(current), content) {
			continue
		}
		changed = true
		if *check {
			fmt.Fprintln(os.Stderr, "generated artifact is stale:", path)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			fail(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			fail(err)
		}
		fmt.Println("generated", path)
	}
	if *check && changed {
		os.Exit(1)
	}
}

func generatedArtifactMatches(path, current, expected string) bool {
	if strings.HasSuffix(path, string(filepath.Separator)+"Tokens.tsx") {
		if marker := strings.Index(current, "/** @vrooliComponentSource foundations.tokens */"); marker >= 0 {
			current = current[marker:]
		}
	}
	return current == expected
}

type sourceFallback struct {
	property        string
	valueStart, end int
}

func listDesignKits(root string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(root, "templates", "design"))
	if err != nil {
		return nil, err
	}
	var kits []string
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), "_") {
			continue
		}
		raw, readErr := os.ReadFile(filepath.Join(root, "templates", "design", entry.Name(), "metadata.json"))
		if readErr != nil {
			return nil, readErr
		}
		var metadata struct {
			ID       string                     `json:"id"`
			Adapters map[string]json.RawMessage `json:"adapters"`
		}
		if decodeErr := json.Unmarshal(raw, &metadata); decodeErr != nil {
			return nil, decodeErr
		}
		if metadata.ID != entry.Name() {
			return nil, fmt.Errorf("design kit directory %s disagrees with metadata id %q", entry.Name(), metadata.ID)
		}
		if _, supported := metadata.Adapters["react-vite-tailwind"]; supported {
			kits = append(kits, metadata.ID)
		}
	}
	sort.Strings(kits)
	return kits, nil
}

func generateTailwindTheme(tokens []themes.DesignToken) (string, error) {
	present := map[string]bool{}
	for _, token := range tokens {
		present[token.Name] = true
	}
	theme := map[string]any{}
	add := func(section, key, property string) {
		if !present[property] {
			return
		}
		values, ok := theme[section].(map[string]any)
		if !ok {
			values = map[string]any{}
			theme[section] = values
		}
		values[key] = "var(" + property + ")"
	}
	for key, property := range map[string]string{
		"app-background": "--color-background", "app-shell": "--color-shell", "app-surface": "--color-surface",
		"app-surface-muted": "--color-surface-muted", "app-surface-raised": "--color-surface-raised",
		"app-foreground": "--color-foreground", "app-muted-foreground": "--color-muted-foreground", "app-border": "--color-border",
		"app-primary": "--color-primary", "app-primary-foreground": "--color-primary-foreground", "app-accent": "--color-accent",
		"app-success": "--color-success", "app-danger": "--color-danger", "app-warning": "--color-warning", "app-info": "--color-info", "app-focus": "--color-focus",
	} {
		add("colors", key, property)
	}
	for _, name := range []string{"control", "panel", "sheet", "pill", "overlay"} {
		add("borderRadius", name, "--radius-"+name)
	}
	for _, name := range []string{"flat", "raised", "floating", "overlay", "modal", "subtle"} {
		add("boxShadow", name, "--elev-"+name)
	}
	for _, name := range []string{"sans", "mono"} {
		add("fontFamily", name, "--font-"+name)
	}
	for _, name := range []string{"4xs", "3xs", "2xs", "xs", "sm", "md", "lg", "xl", "2xl", "4xl"} {
		add("spacing", "space-"+name, "--space-"+name)
	}
	add("spacing", "touch", "--touch-target")
	add("spacing", "sidebar", "--sidebar-width")
	for _, name := range []string{"base", "raised", "sticky", "dropdown", "menu", "popover", "overlay", "modal", "toast", "tooltip", "alert"} {
		add("zIndex", name, "--layer-"+name)
	}
	for _, name := range []string{"hairline", "thin", "strong", "focus", "medium"} {
		add("borderWidth", name, "--border-"+name)
	}
	for _, name := range []string{"disabled", "muted", "scrim"} {
		add("opacity", name, "--opacity-"+name)
	}
	for _, name := range []string{"instant", "fast", "quick", "normal", "moderate", "slow", "deliberate", "enter"} {
		add("transitionDuration", name, "--dur-"+name)
	}
	for _, name := range []string{"standard", "enter", "exit"} {
		add("transitionTimingFunction", name, "--ease-"+name)
	}
	for _, name := range []string{"subtle", "expressive"} {
		add("transitionTimingFunction", "spring-"+name, "--spring-"+name)
	}
	add("minWidth", "touch", "--touch-target")
	add("minWidth", "sidebar", "--sidebar-min-width")
	add("maxWidth", "sidebar", "--sidebar-max-width")

	fontWeights := map[string]string{"display": "650", "title": "650", "heading": "600", "subheading": "600", "body": "400", "body-sm": "400", "label": "500", "caption": "600"}
	letterSpacing := map[string]string{"display": "-0.02em", "title": "-0.015em", "heading": "-0.01em", "label": "0.005em", "caption": "0.06em"}
	fontSizes := map[string]any{}
	for _, name := range []string{"display", "title", "heading", "subheading", "body", "body-sm", "label", "caption"} {
		if !present["--text-"+name+"-size"] || !present["--text-"+name+"-line"] {
			continue
		}
		options := map[string]string{"lineHeight": "var(--text-" + name + "-line)", "fontWeight": fontWeights[name]}
		if tracking := letterSpacing[name]; tracking != "" {
			options["letterSpacing"] = tracking
		}
		fontSizes[name] = []any{"var(--text-" + name + "-size)", options}
	}
	if len(fontSizes) > 0 {
		theme["fontSize"] = fontSizes
	}

	for _, name := range []string{"background", "background-deep", "surface", "surface-muted", "border", "foreground", "muted-foreground", "primary", "primary-foreground", "accent", "success", "warning", "danger", "gap", "stale"} {
		add("colors", "display-"+name, "--display-"+name)
	}
	for _, name := range []string{"panel", "tile", "pill"} {
		add("borderRadius", "display-"+name, "--display-radius-"+name)
	}
	add("boxShadow", "display-glow", "--display-glow")
	add("boxShadow", "display-panel", "--display-panel-shadow")
	add("fontFamily", "display", "--display-font")
	add("fontFamily", "display-numeric", "--display-font-numeric")
	add("spacing", "display", "--display-viewport-padding")
	add("spacing", "remote", "--display-remote-target")
	add("minHeight", "remote", "--display-remote-target")
	add("minWidth", "remote", "--display-remote-target")

	if present["--color-surface-primary"] || present["--section-spacing"] {
		for key, property := range map[string]string{"landing-background": "--color-background", "landing-surface-primary": "--color-surface-primary", "landing-surface-muted": "--color-surface-muted", "landing-surface-deep": "--color-surface-deep", "landing-surface-darker": "--color-surface-darker", "landing-surface-alt": "--color-surface-alt", "landing-text-primary": "--color-text-primary", "landing-text-secondary": "--color-text-secondary", "landing-text-muted": "--color-text-muted", "landing-accent-primary": "--color-accent-primary", "landing-accent-secondary": "--color-accent-secondary", "landing-accent-tertiary": "--color-accent-tertiary", "landing-success": "--color-success", "landing-warning": "--color-warning", "landing-danger": "--color-danger", "landing-border-subtle": "--color-border-subtle"} {
			add("colors", key, property)
		}
		add("borderRadius", "landing-control", "--radius-control")
		add("borderRadius", "landing-card", "--radius-card")
		add("borderRadius", "landing-panel", "--radius-panel")
		add("boxShadow", "landing-panel", "--shadow-panel")
		add("boxShadow", "landing-card", "--shadow-card")
		add("fontFamily", "landing-headline", "--font-headline")
		add("fontFamily", "landing-body", "--font-body")
		add("spacing", "landing-section", "--section-spacing")
		add("maxWidth", "landing-container", "--container-max-width")
	}

	raw, err := json.MarshalIndent(theme, "", "  ")
	if err != nil {
		return "", err
	}
	return string(raw) + "\n", nil
}

func generateTokensSource(tokens map[string]themes.DesignToken) string {
	values := func(names ...string) string {
		var lines []string
		for _, name := range names {
			if _, ok := tokens[name]; ok {
				lines = append(lines, fmt.Sprintf("    \"var(%s)\",", name))
			}
		}
		return strings.Join(lines, "\n")
	}
	return `/** @vrooliComponentSource foundations.tokens */
// Code generated by design-token-generate; DO NOT EDIT.
export const TOKEN_RAMPS = {
  space: [
` + values("--space-4xs", "--space-3xs", "--space-2xs", "--space-xs", "--space-sm", "--space-md", "--space-lg", "--space-xl", "--space-2xl", "--space-4xl") + `
  ],
  text: [
` + values("--text-display", "--text-title", "--text-heading", "--text-body", "--text-label", "--text-caption", "--text-code", "--text-overline") + `
  ],
  radius: [
` + values("--radius-control", "--radius-panel", "--radius-sheet", "--radius-overlay", "--radius-pill") + `
  ],
  elevation: [
` + values("--elev-flat", "--elev-subtle", "--elev-raised", "--elev-floating", "--elev-overlay", "--elev-modal") + `
  ],
  layer: [
` + values("--layer-base", "--layer-raised", "--layer-sticky", "--layer-dropdown", "--layer-menu", "--layer-popover", "--layer-overlay", "--layer-modal", "--layer-toast", "--layer-tooltip", "--layer-alert") + `
  ],
  motion: [
` + values("--dur-instant", "--dur-fast", "--dur-quick", "--dur-normal", "--dur-moderate", "--dur-slow", "--dur-deliberate", "--dur-enter") + `
  ],
} as const;

export const SEMANTIC_TOKENS = {
  background: "var(--color-background)",
  foreground: "var(--color-foreground)",
  surface: "var(--color-surface)",
  surfaceMuted: "var(--color-surface-muted)",
  border: "var(--color-border)",
  muted: "var(--color-muted-foreground)",
  primary: "var(--color-primary)",
  primaryForeground: "var(--color-primary-foreground)",
  accent: "var(--color-accent)",
  success: "var(--color-success)",
  warning: "var(--color-warning)",
  danger: "var(--color-danger)",
  info: "var(--color-info)",
  focus: "var(--color-focus)",
} as const;

export const PROVENANCE_TOKENS = {
  measured: "var(--provenance-measured)",
  cached: "var(--provenance-cached)",
  sample: "var(--provenance-sample)",
  absent: "var(--provenance-absent)",
  glow: "var(--glow-primary)",
} as const;
export const COMPONENT_TOKENS = {
  controlSize: "var(--control-size-md)",
  controlRadius: "var(--control-radius)",
  controlPadding: "var(--control-padding)",
  panelRadius: "var(--panel-radius)",
  panelPadding: "var(--panel-padding)",
  focusRing: "var(--focus-ring)",
} as const;

export type TextStyle = keyof typeof TEXT_STYLES;
export const TEXT_STYLES = {
  display: "var(--text-display)",
  title: "var(--text-title)",
  heading: "var(--text-heading)",
  body: "var(--text-body)",
  label: "var(--text-label)",
  caption: "var(--text-caption)",
  code: "var(--text-code)",
  overline: "var(--text-overline)",
  wall: "var(--text-wall)",
} as const;

export const tokens = {
  ramps: TOKEN_RAMPS,
  semantic: SEMANTIC_TOKENS,
  component: COMPONENT_TOKENS,
  text: TEXT_STYLES,
} as const;
`
}

func latestTokensSourcePath(root string) (string, error) {
	manifestPath := filepath.Join(root, "scenarios", "react-component-library", "library", "foundations", "Tokens", "component.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", err
	}
	var manifest struct {
		Latest string `json:"latest"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return "", err
	}
	if manifest.Latest == "" {
		return "", fmt.Errorf("Tokens manifest has no latest release")
	}
	return filepath.Join(filepath.Dir(manifestPath), "versions", manifest.Latest, "Tokens.tsx"), nil
}

func normalizeComponentDraft(root, asset string, canonical map[string]themes.DesignToken) (map[string]string, error) {
	manifestPath := filepath.Join(root, "scenarios", "react-component-library", "library", "components", asset, "component.json")
	rawManifest, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	var manifest struct {
		Draft string `json:"draft"`
	}
	if err := json.Unmarshal(rawManifest, &manifest); err != nil {
		return nil, err
	}
	if strings.TrimSpace(manifest.Draft) == "" {
		return nil, fmt.Errorf("component %s has no active draft; run components draft-begin first", asset)
	}
	dir := filepath.Join(filepath.Dir(manifestPath), "versions", manifest.Draft)
	var paths []string
	for _, extension := range []string{"*.ts", "*.tsx"} {
		matches, globErr := filepath.Glob(filepath.Join(dir, extension))
		if globErr != nil {
			return nil, globErr
		}
		paths = append(paths, matches...)
	}
	declared := map[string]bool{}
	activePins, err := loadActiveInternalPins(root)
	if err != nil {
		return nil, err
	}
	contents := map[string]string{}
	for _, path := range paths {
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, readErr
		}
		contents[path] = string(raw)
		for _, match := range tokenDeclarationRE.FindAllStringSubmatch(string(raw), -1) {
			declared[match[2]] = true
		}
	}
	outputs := map[string]string{}
	for path, content := range contents {
		content = strings.ReplaceAll(content, "--wc-kb-height", "--rcl-keyboard-inset")
		content = normalizeInactiveInternalPins(content, activePins)
		if asset == "Card" {
			content = strings.ReplaceAll(content, "boxShadow: `var(--elev-${elevation})`,", "boxShadow: { flat: \"var(--elev-flat)\", raised: \"var(--elev-raised)\", floating: \"var(--elev-floating)\", overlay: \"var(--elev-overlay)\" }[elevation],")
		}
		fallbacks := scanSourceFallbacks(content)
		for i := len(fallbacks) - 1; i >= 0; i-- {
			fallback := fallbacks[i]
			token, known := canonical[fallback.property]
			if !known || declared[fallback.property] || strings.HasPrefix(fallback.property, "--rcl-") {
				continue
			}
			replacement := token.Value
			lineStart := strings.LastIndex(content[:fallback.valueStart], "\n") + 1
			if strings.Count(content[lineStart:fallback.valueStart], `"`)%2 == 1 {
				replacement = strings.ReplaceAll(replacement, `"`, `\"`)
			}
			content = content[:fallback.valueStart] + replacement + content[fallback.end:]
		}
		outputs[path] = content
	}
	return outputs, nil
}

type activeInternalPin struct {
	latest     string
	deprecated map[string]bool
}

func loadActiveInternalPins(root string) (map[string]activeInternalPin, error) {
	paths, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", "*", "*", "component.json"))
	if err != nil {
		return nil, err
	}
	pins := make(map[string]activeInternalPin, len(paths))
	for _, path := range paths {
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, readErr
		}
		var manifest struct {
			LibraryID          string   `json:"libraryId"`
			Latest             string   `json:"latest"`
			DeprecatedVersions []string `json:"deprecatedVersions"`
		}
		if unmarshalErr := json.Unmarshal(raw, &manifest); unmarshalErr != nil {
			return nil, fmt.Errorf("parse %s: %w", path, unmarshalErr)
		}
		_, name, ok := strings.Cut(manifest.LibraryID, ":")
		if !ok || name == "" || manifest.Latest == "" {
			continue
		}
		deprecated := make(map[string]bool, len(manifest.DeprecatedVersions))
		for _, version := range manifest.DeprecatedVersions {
			deprecated[version] = true
		}
		pins[name] = activeInternalPin{latest: manifest.Latest, deprecated: deprecated}
	}
	return pins, nil
}

func normalizeInactiveInternalPins(content string, pins map[string]activeInternalPin) string {
	return internalImportRE.ReplaceAllStringFunc(content, func(specifier string) string {
		match := internalImportRE.FindStringSubmatch(specifier)
		if len(match) != 3 {
			return specifier
		}
		pin, known := pins[match[1]]
		if !known || !pin.deprecated[match[2]] {
			return specifier
		}
		return "@vrooli/react-component-library/" + match[1] + "/" + pin.latest
	})
}

func scanSourceFallbacks(source string) []sourceFallback {
	var result []sourceFallback
	for offset := 0; offset < len(source); {
		relative := strings.Index(source[offset:], "var(")
		if relative < 0 {
			break
		}
		start := offset + relative
		depth, comma, end := 0, -1, -1
		for i := start + len("var("); i < len(source); i++ {
			switch source[i] {
			case '(':
				depth++
			case ')':
				if depth == 0 {
					end = i
				} else {
					depth--
				}
			case ',':
				if depth == 0 && comma < 0 {
					comma = i
				}
			}
			if end >= 0 {
				break
			}
		}
		if end < 0 {
			break
		}
		if comma > 0 {
			property := strings.TrimSpace(source[start+len("var(") : comma])
			if strings.HasPrefix(property, "--") && !strings.Contains(property, "${") {
				valueStart := comma + 1
				for valueStart < end && (source[valueStart] == ' ' || source[valueStart] == '\t' || source[valueStart] == '\n') {
					valueStart++
				}
				result = append(result, sourceFallback{property: property, valueStart: valueStart, end: end})
			}
		}
		offset = end + 1
	}
	return result
}

func indentCSSRoot(css string) string {
	lines := strings.Split(css, "\n")
	if len(lines) > 0 && strings.HasPrefix(lines[0], "/* Canonical shared vocabulary") {
		lines = lines[1:]
	}
	for i, line := range lines {
		if line != "" {
			lines[i] = "  " + line
		}
	}
	return strings.Join(lines, "\n")
}

func replaceManagedRegion(content, begin, end, replacement string) (string, error) {
	start := strings.Index(content, begin)
	finish := strings.Index(content, end)
	if start < 0 || finish < 0 || finish < start {
		return "", fmt.Errorf("managed region %q .. %q is missing or malformed", begin, end)
	}
	start += len(begin)
	return content[:start] + "\n" + strings.TrimSuffix(replacement, "\n") + "\n" + content[finish:], nil
}

func normalizeKitTokens(content string, base map[string]themes.DesignToken) string {
	lines := strings.Split(content, "\n")
	result := make([]string, 0, len(lines))
	inRoot := false
	depth := 0
	for _, line := range lines {
		if !inRoot && strings.TrimSpace(line) == ":root {" {
			inRoot = true
			depth = 1
			result = append(result, line)
			continue
		}
		if !inRoot {
			result = append(result, line)
			continue
		}
		depth += strings.Count(line, "{") - strings.Count(line, "}")
		match := tokenDeclarationRE.FindStringSubmatch(line)
		if len(match) == 4 {
			name, value := match[2], strings.TrimSpace(match[3])
			if canonical, exists := base[name]; exists && canonical.Value == value {
				continue
			}
			if len(result) == 0 || !strings.Contains(result[len(result)-1], "@tier ") {
				result = append(result, match[1]+"/* @tier "+string(tierFor(name))+" */")
			}
		}
		result = append(result, line)
		if depth == 0 {
			inRoot = false
		}
	}
	return strings.Join(result, "\n")
}

func firstRootBlock(css []byte) ([]byte, error) {
	content := string(css)
	start := strings.Index(content, ":root")
	if start < 0 {
		return nil, fmt.Errorf("first :root block not found")
	}
	open := strings.Index(content[start:], "{")
	if open < 0 {
		return nil, fmt.Errorf("first :root block is malformed")
	}
	open += start
	depth := 0
	for i := open; i < len(content); i++ {
		switch content[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return []byte(content[start : i+1]), nil
			}
		}
	}
	return nil, fmt.Errorf("first :root block is unterminated")
}

// normalizeDesignDocument keeps narrative metadata and component-state
// guidance in DESIGN.md while making the CSS files the sole token authority.
func normalizeDesignDocument(content string) (string, error) {
	lines := strings.Split(content, "\n")
	if len(lines) < 3 || lines[0] != "---" {
		return "", fmt.Errorf("missing YAML front matter")
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return "", fmt.Errorf("unterminated YAML front matter")
	}
	remove := map[string]bool{"colors": true, "typography": true, "rounded": true, "spacing": true, "tokens": true}
	front := make([]string, 0, end)
	for i := 1; i < end; {
		line := lines[i]
		key := topLevelYAMLKey(line)
		if remove[key] {
			i = nextTopLevelYAML(lines, i+1, end)
			continue
		}
		if key == "components" {
			componentEnd := nextTopLevelYAML(lines, i+1, end)
			front = append(front, "components:")
			for _, component := range componentYAMLKeys(lines[i+1 : componentEnd]) {
				front = append(front, "  "+component+":", "    tokenSource: design-tokens.css")
			}
			i = componentEnd
			continue
		}
		front = append(front, line)
		i++
	}
	result := append([]string{"---"}, front...)
	result = append(result, lines[end:]...)
	return strings.Join(result, "\n"), nil
}

func topLevelYAMLKey(line string) string {
	if strings.TrimSpace(line) == "" || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
		return ""
	}
	key, _, ok := strings.Cut(line, ":")
	if !ok {
		return ""
	}
	return strings.TrimSpace(key)
}

func nextTopLevelYAML(lines []string, start, end int) int {
	for i := start; i < end; i++ {
		if topLevelYAMLKey(lines[i]) != "" {
			return i
		}
	}
	return end
}

func componentYAMLKeys(lines []string) []string {
	keys := make([]string, 0)
	for _, line := range lines {
		if !strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "   ") {
			continue
		}
		key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if ok && strings.TrimSpace(value) == "" {
			keys = append(keys, strings.TrimSpace(key))
		}
	}
	return keys
}

func tierFor(name string) themes.TokenTier {
	if name == "--tap-target-min" || name == "--touch-target" || name == "--color-focus" || name == "--color-focus-ring" || name == "--border-focus" || name == "--opacity-disabled" || strings.HasPrefix(name, "--layer-") || strings.HasPrefix(name, "--focus-ring") {
		return themes.TokenTierContract
	}
	for _, prefix := range []string{"--space-", "--control-height", "--control-size", "--control-padding", "--panel-padding", "--sidebar-", "--overlay-", "--icon-size", "--border-hairline", "--border-strong", "--border-thin", "--border-medium", "--content-min-height"} {
		if strings.HasPrefix(name, prefix) {
			return themes.TokenTierRhythm
		}
	}
	return themes.TokenTierExpression
}

func describe(name string) (string, string) {
	label := strings.ReplaceAll(strings.TrimPrefix(name, "--"), "-", " ")
	switch {
	case strings.HasPrefix(name, "--color-"):
		return "Semantic color role for " + label + ".", "Styling the named semantic state or surface without embedding a literal color."
	case strings.HasPrefix(name, "--space-"):
		return "Spacing-ramp step " + strings.TrimPrefix(name, "--space-") + ".", "Choosing internal or inter-component spacing from the shared rhythm."
	case strings.HasPrefix(name, "--text-") || strings.HasPrefix(name, "--font-") || strings.HasPrefix(name, "--tracking-") || strings.HasPrefix(name, "--weight-"):
		return "Typography role for " + label + ".", "Applying the named text hierarchy without reconstructing font metrics."
	case strings.HasPrefix(name, "--layer-"):
		return "Stacking-order contract for " + label + ".", "Placing the named overlay class in the shared z-order."
	case strings.HasPrefix(name, "--radius-"):
		return "Corner treatment for " + label + ".", "Rounding the named surface or control role."
	case strings.HasPrefix(name, "--dur-") || strings.HasPrefix(name, "--ease-") || strings.HasPrefix(name, "--spring-") || strings.HasPrefix(name, "--motion-"):
		return "Motion timing role for " + label + ".", "Animating the named transition with the shared motion language."
	case strings.HasPrefix(name, "--provenance-"):
		return "Provenance encoding for a value that is " + strings.TrimPrefix(name, "--provenance-") + ".", "Marking where a displayed figure came from so measured, cached, sampled and absent values never share a colour."
	case strings.HasPrefix(name, "--glow-"):
		return "Ambient glow for " + label + ".", "Lifting a live or focused figure on dark display surfaces without a local shadow literal."
	case strings.HasPrefix(name, "--elev-") || strings.HasPrefix(name, "--shadow-"):
		return "Elevation treatment for " + label + ".", "Expressing the named depth level without a local shadow literal."
	default:
		return "Canonical design-system value for " + label + ".", "Implementing the named shared component contract."
	}
}

func resolveRoot(explicit string) (string, error) {
	if explicit != "" {
		return filepath.Abs(explicit)
	}
	current, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(current, "templates", "design", "vrooli-default", "metadata.json")); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("repository root not found")
		}
		current = parent
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "design-token-generate:", err)
	os.Exit(1)
}
