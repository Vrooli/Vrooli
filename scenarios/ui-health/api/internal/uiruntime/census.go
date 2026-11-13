package uiruntime

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func pageCensus(layoutJSON, pageURL, viewport, screenshotRef string) PagePatternCensus {
	var layout struct {
		PatternCensus              map[string]int            `json:"patternCensus"`
		Tokens                     []string                  `json:"tokens"`
		CensusTokens               []string                  `json:"censusTokens"`
		TokenizedPatternProperties map[string]map[string]int `json:"tokenizedPatternProperties"`
	}
	if strings.TrimSpace(layoutJSON) == "" || json.Unmarshal([]byte(layoutJSON), &layout) != nil {
		return PagePatternCensus{}
	}
	parsed, _ := url.Parse(pageURL)
	route := "/"
	if parsed != nil && parsed.Path != "" {
		route = parsed.Path
	}
	tokens := layout.CensusTokens
	if len(tokens) == 0 {
		tokens = layout.Tokens
	}
	return PagePatternCensus{Route: route, Viewport: viewport, Patterns: layout.PatternCensus, Tokens: tokens, TokenizedPatternProperties: layout.TokenizedPatternProperties, ScreenshotRef: screenshotRef}
}

// PagePatternCensus is the computed-pattern inventory captured from one served
// route and viewport. Counts are retained so common repeated components weigh
// more than one-off decorative elements.
type PagePatternCensus struct {
	Scenario                   string                    `json:"scenario"`
	Route                      string                    `json:"route"`
	Viewport                   string                    `json:"viewport"`
	Patterns                   map[string]int            `json:"patterns"`
	Tokens                     []string                  `json:"tokens,omitempty"`
	TokenizedPatternProperties map[string]map[string]int `json:"tokenizedPatternProperties,omitempty"`
	ScreenshotRef              string                    `json:"screenshotRef,omitempty"`
}

// CensusOutlier reports page patterns that differ from the scenario-wide
// majority, or values that do not match any declared design token.
type CensusOutlier struct {
	Scenario        string  `json:"scenario"`
	Route           string  `json:"route"`
	Viewport        string  `json:"viewport"`
	Family          string  `json:"family,omitempty"`
	ElementFamily   string  `json:"elementFamily,omitempty"`
	Property        string  `json:"property,omitempty"`
	ElementPattern  string  `json:"elementPattern,omitempty"`
	ObservedCount   int     `json:"observedCount,omitempty"`
	Pattern         string  `json:"pattern"`
	DominantPattern string  `json:"dominantPattern,omitempty"`
	Reason          string  `json:"reason"`
	Score           float64 `json:"score"`
	ScreenshotRef   string  `json:"screenshotRef,omitempty"`
}

// CompareScenarioCensus finds patterns observed on a minority of the served
// pages for this scenario. A 0.5 threshold intentionally leaves two-way ties
// unreported; callers can raise or lower the threshold for review workflows.
func CompareScenarioCensus(pages []PagePatternCensus, minorityThreshold float64) []CensusOutlier {
	if len(pages) < 2 {
		return nil
	}
	byViewport := make(map[string][]PagePatternCensus)
	for _, page := range pages {
		byViewport[page.Viewport] = append(byViewport[page.Viewport], page)
	}
	if len(byViewport) > 1 {
		var out []CensusOutlier
		for _, viewportPages := range byViewport {
			out = append(out, CompareScenarioCensus(viewportPages, minorityThreshold)...)
		}
		sort.SliceStable(out, func(i, j int) bool {
			if out[i].Score == out[j].Score {
				return out[i].Route < out[j].Route
			}
			return out[i].Score > out[j].Score
		})
		return out
	}
	if minorityThreshold <= 0 || minorityThreshold >= 1 {
		minorityThreshold = 0.5
	}
	type patternStats struct {
		pages int
		count int
	}
	byFamily := make(map[string]map[string]*patternStats)
	familyPages := make(map[string]int)
	tokenSet := make(map[string]bool)
	for _, page := range pages {
		for _, token := range page.Tokens {
			if normalized, ok := normalizeDesignValue(token); ok {
				tokenSet[normalized] = true
			}
		}
		seen := make(map[string]bool)
		familiesOnPage := make(map[string]bool)
		for pattern, count := range page.Patterns {
			if count <= 0 {
				continue
			}
			family := patternFamily(pattern)
			if family == "" {
				continue
			}
			if byFamily[family] == nil {
				byFamily[family] = make(map[string]*patternStats)
			}
			if byFamily[family][pattern] == nil {
				byFamily[family][pattern] = &patternStats{}
			}
			byFamily[family][pattern].count += count
			familiesOnPage[family] = true
			if !seen[pattern] {
				byFamily[family][pattern].pages++
				seen[pattern] = true
			}
		}
		for family := range familiesOnPage {
			familyPages[family]++
		}
	}
	families := make([]string, 0, len(byFamily))
	for family := range byFamily {
		families = append(families, family)
	}
	sort.Strings(families)
	dominant := make(map[string]string)
	dominantPrevalence := make(map[string]float64)
	for _, family := range families {
		patterns := make([]string, 0, len(byFamily[family]))
		for pattern := range byFamily[family] {
			patterns = append(patterns, pattern)
		}
		sort.Strings(patterns)
		for _, pattern := range patterns {
			stats := byFamily[family][pattern]
			prevalence := float64(stats.pages) / float64(familyPages[family])
			if prevalence > 1-minorityThreshold {
				dominant[family] = pattern
				dominantPrevalence[family] = prevalence
				break
			}
		}
	}
	out := make([]CensusOutlier, 0)
	for _, page := range pages {
		for _, family := range families {
			dominantPattern := dominant[family]
			if dominantPattern == "" {
				continue
			}
			familyCount := 0
			observedPattern := ""
			observedCount := 0
			for _, candidate := range sortedKeys(page.Patterns) {
				count := page.Patterns[candidate]
				if count <= 0 || patternFamily(candidate) != family {
					continue
				}
				familyCount += count
				if candidate != dominantPattern && count > observedCount {
					observedPattern, observedCount = candidate, count
				}
			}
			if familyCount == 0 || page.Patterns[dominantPattern] > 0 {
				continue
			}
			if observedPattern != "" && hasUntokenizedDesignValue(observedPattern, observedCount, tokenSet, page.TokenizedPatternProperties) {
				out = append(out, CensusOutlier{
					Scenario: page.Scenario, Route: page.Route, Viewport: page.Viewport, Family: family, ElementFamily: family, ElementPattern: observedPattern, ObservedCount: observedCount, Pattern: observedPattern, DominantPattern: dominantPattern,
					Reason: "computed style differs from the dominant style on pages containing this element family", Score: dominantPrevalence[family], ScreenshotRef: page.ScreenshotRef,
				})
			}
		}
		if len(tokenSet) > 0 {
			if candidate := dominantUntokenizedDesignEvidence(page.Patterns, tokenSet, page.TokenizedPatternProperties); candidate.Value != "" {
				out = append(out, CensusOutlier{
					Scenario: page.Scenario, Route: page.Route, Viewport: page.Viewport,
					ElementFamily: candidate.Family, Property: candidate.Property, ElementPattern: candidate.Pattern, ObservedCount: candidate.Count, Pattern: candidate.Value,
					Reason: "computed design value is not present among the scenario's declared design tokens", Score: 0.75, ScreenshotRef: page.ScreenshotRef,
				})
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score == out[j].Score {
			return out[i].Route < out[j].Route
		}
		return out[i].Score > out[j].Score
	})
	return out
}

func dominantUntokenizedDesignValue(patterns map[string]int, tokenSet map[string]bool) string {
	return dominantUntokenizedDesignEvidence(patterns, tokenSet, nil).Value
}

type designValueEvidence struct {
	Value    string
	Family   string
	Property string
	Pattern  string
	Count    int
	Total    int
}

type designValueField struct {
	index    int
	property string
}

func dominantUntokenizedDesignEvidence(patterns map[string]int, tokenSet map[string]bool, tokenized map[string]map[string]int) designValueEvidence {
	counts := make(map[string]designValueEvidence)
	totals := make(map[string]int)
	for _, pattern := range sortedKeys(patterns) {
		count := patterns[pattern]
		if count <= 0 {
			continue
		}
		for _, candidate := range designValueEvidenceForPattern(pattern, count) {
			if candidate.Value == "" || candidate.Value == "0px" || candidate.Value == "transparent" {
				continue
			}
			normalized, ok := normalizeDesignValue(candidate.Value)
			if !ok || tokenSet[normalized] || tokenized[pattern][candidate.Property] >= count {
				continue
			}
			totals[candidate.Value] += candidate.Count
			prior, exists := counts[candidate.Value]
			if !exists || candidate.Count > prior.Count || (candidate.Count == prior.Count && (candidate.Pattern < prior.Pattern || (candidate.Pattern == prior.Pattern && candidate.Property < prior.Property))) {
				counts[candidate.Value] = candidate
			}
		}
	}
	best := designValueEvidence{}
	for value, candidate := range counts {
		candidate.Total = totals[value]
		if candidate.Total > best.Total || (candidate.Total == best.Total && (value < best.Value || (value == best.Value && candidate.Pattern < best.Pattern))) {
			best = candidate
		}
	}
	return best
}

func hasUntokenizedDesignValue(pattern string, count int, tokenSet map[string]bool, tokenized map[string]map[string]int) bool {
	for _, candidate := range designValueEvidenceForPattern(pattern, count) {
		normalized, ok := normalizeDesignValue(candidate.Value)
		if ok && !tokenSet[normalized] && tokenized[pattern][candidate.Property] < count {
			return true
		}
	}
	return false
}

func patternFamily(pattern string) string {
	if i := strings.IndexByte(pattern, ':'); i > 0 {
		family, fields := pattern[:i], strings.SplitN(pattern[i+1:], "|", 2)
		// Heading levels carry distinct semantic roles and should not be
		// compared as though h1 and h3 were interchangeable elements.
		if family == "heading" && len(fields) > 0 && fields[0] != "" {
			return family + ":" + fields[0]
		}
		return family
	}
	return ""
}

func designValueEvidenceForPattern(pattern string, count int) []designValueEvidence {
	parts := strings.SplitN(pattern, ":", 2)
	if len(parts) != 2 {
		return nil
	}
	family := parts[0]
	fields := strings.Split(parts[1], "|")
	indices := []designValueField{}
	switch family {
	case "button":
		indices = []designValueField{{0, "height"}, {2, "font-size"}, {4, "background-color"}, {5, "color"}}
	case "heading":
		// Computed line-height is a pixel result even when the source uses an
		// intentional unitless ratio (for example 1.3), so omit it.
		indices = []designValueField{{1, "font-size"}, {4, "color"}}
	case "spacing":
		indices = []designValueField{{0, "gap"}, {1, "padding"}, {2, "margin"}}
	case "footer":
		indices = []designValueField{{4, "background-color"}}
	}
	values := make([]designValueEvidence, 0, len(indices)+1)
	for _, item := range indices {
		if item.index < len(fields) {
			values = append(values, designValueEvidence{Value: strings.TrimSpace(fields[item.index]), Family: family, Property: item.property, Pattern: pattern, Count: count})
		}
	}
	if family == "button" && len(fields) > 1 {
		for _, radius := range strings.Fields(fields[1]) {
			values = append(values, designValueEvidence{Value: radius, Family: family, Property: "border-radius", Pattern: pattern, Count: count})
		}
	}
	return values
}

var (
	numericDesignValue = regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)?)(px|rem|em)?$`)
	rgbDesignValue     = regexp.MustCompile(`^rgba?\((.+)\)$`)
)

func normalizeDesignValue(value string) (string, bool) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" || value == "transparent" || value == "normal" || value == "auto" {
		return "", false
	}
	if strings.HasPrefix(value, "#") {
		if len(value) == 4 {
			return "#" + string([]byte{value[1], value[1], value[2], value[2], value[3], value[3]}), true
		}
		return value, true
	}
	if match := rgbDesignValue.FindStringSubmatch(value); len(match) == 2 {
		parts := strings.Fields(strings.NewReplacer(",", " ", "/", " ").Replace(match[1]))
		if len(parts) >= 3 {
			channels := make([]int, 3)
			valid := true
			for i := 0; i < 3; i++ {
				part := strings.TrimSuffix(parts[i], "%")
				channel, err := strconv.ParseFloat(part, 64)
				if err != nil {
					valid = false
					break
				}
				if strings.HasSuffix(parts[i], "%") {
					channel *= 2.55
				}
				channels[i] = int(channel + 0.5)
			}
			if valid {
				normalized := fmt.Sprintf("#%02x%02x%02x", channels[0], channels[1], channels[2])
				// Alpha changes the rendered color. A fully transparent color paints
				// nothing and must not be reported as an opaque token mismatch;
				// partially transparent colors remain distinct from opaque tokens.
				if len(parts) >= 4 {
					alphaText := strings.TrimSuffix(parts[3], "%")
					alpha, err := strconv.ParseFloat(alphaText, 64)
					if err == nil {
						if strings.HasSuffix(parts[3], "%") {
							alpha /= 100
						}
						if alpha <= 0 {
							return "", false
						}
						if alpha < 1 {
							return fmt.Sprintf("%s@%.3f", normalized, alpha), true
						}
					}
				}
				return normalized, true
			}
		}
	}
	match := numericDesignValue.FindStringSubmatch(value)
	if len(match) == 0 {
		return value, true
	}
	n, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return "", false
	}
	switch match[2] {
	case "rem", "em":
		n *= 16
	}
	return fmt.Sprintf("%.3fpx", n), true
}

func sortedKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
