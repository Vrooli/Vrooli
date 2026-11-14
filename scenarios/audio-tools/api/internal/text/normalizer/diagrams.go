package normalizer

import (
	"fmt"
	"strings"
	"unicode"
)

// ---------------------------------------------------------------------------
// Unfenced diagram / wireframe detection
// ---------------------------------------------------------------------------

// minDiagramRunLength is the minimum number of consecutive diagram-scored lines
// required to trigger replacement. Shorter runs are left alone to avoid
// false-positives on isolated decorative lines.
const minDiagramRunLength = 3

// diagramScoreThreshold is the minimum per-line score (0–1) to consider a line
// part of a diagram region.
const diagramScoreThreshold = 0.5

// replaceUnfencedDiagrams finds runs of diagram/wireframe lines not inside
// fenced code blocks and replaces each run with a speech-friendly summary.
func replaceUnfencedDiagrams(text string) string {
	lines := strings.Split(text, "\n")
	scores := make([]float64, len(lines))
	for i, line := range lines {
		scores[i] = diagramLineScore(line)
	}

	// Find runs of consecutive high-scoring lines, allowing a single
	// low-scoring "bridge" line between two high-scoring lines.
	type region struct{ start, end int } // [start, end)
	var regions []region

	i := 0
	for i < len(lines) {
		if scores[i] < diagramScoreThreshold {
			i++
			continue
		}
		// Start of a potential run.
		start := i
		i++
		for i < len(lines) {
			if scores[i] >= diagramScoreThreshold {
				i++
				continue
			}
			// Allow a 1-line bridge if the next line is high-scoring.
			if i+1 < len(lines) && scores[i+1] >= diagramScoreThreshold {
				i += 2
				continue
			}
			break
		}
		if i-start >= minDiagramRunLength {
			regions = append(regions, region{start, i})
		}
	}

	if len(regions) == 0 {
		return text
	}

	// Replace regions back-to-front to preserve line indices.
	for ri := len(regions) - 1; ri >= 0; ri-- {
		r := regions[ri]
		summary := classifyDiagramRegion(lines[r.start:r.end])
		newLines := make([]string, 0, len(lines)-(r.end-r.start)+1)
		newLines = append(newLines, lines[:r.start]...)
		newLines = append(newLines, summary)
		newLines = append(newLines, lines[r.end:]...)
		lines = newLines
	}

	return strings.Join(lines, "\n")
}

// diagramLineScore returns 0.0–1.0 indicating how "diagrammatic" a line is.
func diagramLineScore(line string) float64 {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return 0
	}

	score := 0.0

	// Box-drawing Unicode characters (high signal).
	boxCount := 0
	for _, r := range trimmed {
		if isBoxDrawingRune(r) {
			boxCount++
		}
	}
	if boxCount >= 2 {
		score = maxFloat(score, 0.9)
	}

	// Tree connectors: ├──, └──, │ (high signal).
	if reTreeConnector.MatchString(trimmed) {
		score = maxFloat(score, 0.85)
	}

	// Mermaid keyword anchor (high signal).
	if reMermaidKeyword.MatchString(trimmed) {
		score = maxFloat(score, 0.9)
	}

	// Mermaid body line: brackets/braces combined with an arrow (high signal).
	// e.g. "A[Start] --> B{Decision}"
	hasBrackets := strings.ContainsAny(trimmed, "[]{}")
	hasArrow := reDiagramArrow.MatchString(trimmed)
	if hasBrackets && hasArrow {
		score = maxFloat(score, 0.8)
	}

	// Arrow patterns: -->, ==>, ->, => (medium/high signal).
	arrowCount := len(reDiagramArrow.FindAllString(trimmed, -1))
	if arrowCount >= 2 {
		score = maxFloat(score, 0.8)
	} else if arrowCount == 1 {
		score = maxFloat(score, 0.5)
	}

	// Pipe-bordered line: starts and ends with | (common in wireframes).
	// Tables are already replaced before diagram detection runs.
	if len(trimmed) >= 3 && trimmed[0] == '|' && trimmed[len(trimmed)-1] == '|' {
		score = maxFloat(score, 0.7)
	}

	// Structural character ratio — lines dominated by +|-=<>/\^_ and similar.
	structural, total := 0, 0
	for _, r := range trimmed {
		if !unicode.IsSpace(r) {
			total++
			if isDiagramStructuralChar(r) {
				structural++
			}
		}
	}
	if total > 0 {
		ratio := float64(structural) / float64(total)
		if ratio > 0.6 {
			score = maxFloat(score, 0.8)
		} else if ratio > 0.4 {
			score = maxFloat(score, 0.5)
		}
	}

	// ASCII box junction pattern: +---+ or +===+ (high signal).
	if reASCIIBoxJunction.MatchString(trimmed) {
		score = maxFloat(score, 0.9)
	}

	return score
}

// classifyDiagramRegion examines the lines in a diagram run and returns an
// appropriate speech-friendly summary string.
func classifyDiagramRegion(lines []string) string {
	block := strings.Join(lines, "\n")
	lineCount := len(lines)

	// Count signal types across the region.
	boxDrawingLines := 0
	treeLines := 0
	mermaidAnchor := ""
	arrowLines := 0
	asciiBoxLines := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Box drawing.
		boxCount := 0
		for _, r := range trimmed {
			if isBoxDrawingRune(r) {
				boxCount++
			}
		}
		if boxCount >= 2 {
			boxDrawingLines++
		}

		// ASCII box junctions.
		if reASCIIBoxJunction.MatchString(trimmed) {
			asciiBoxLines++
		}

		// Tree connectors.
		if reTreeConnector.MatchString(trimmed) {
			treeLines++
		}

		// Mermaid keyword.
		if mermaidAnchor == "" {
			if m := reMermaidKeyword.FindStringSubmatch(trimmed); len(m) > 1 {
				mermaidAnchor = m[1]
			}
		}

		// Arrows.
		if reDiagramArrow.MatchString(trimmed) {
			arrowLines++
		}
	}

	// Classify based on dominant signal.
	switch {
	case mermaidAnchor != "":
		return fmt.Sprintf("Diagram: %s with %d lines.", normalizeMermaidType(mermaidAnchor), lineCount)
	case treeLines > lineCount/2:
		entries := countTreeEntries(block)
		return fmt.Sprintf("File tree with %d entries.", entries)
	case boxDrawingLines > lineCount/3 || asciiBoxLines > lineCount/3:
		return "Wireframe diagram."
	case arrowLines > lineCount/3:
		return fmt.Sprintf("Flow diagram with %d lines.", lineCount)
	default:
		return "Diagram."
	}
}

// normalizeMermaidType maps mermaid keyword prefixes to human-readable names.
func normalizeMermaidType(keyword string) string {
	kw := strings.ToLower(strings.TrimSpace(keyword))
	switch {
	case strings.HasPrefix(kw, "graph"):
		return "graph"
	case strings.HasPrefix(kw, "flowchart"):
		return "flowchart"
	case strings.HasPrefix(kw, "sequencediagram"):
		return "sequence diagram"
	case strings.HasPrefix(kw, "classdiagram"):
		return "class diagram"
	case strings.HasPrefix(kw, "statediagram"):
		return "state diagram"
	case strings.HasPrefix(kw, "erdiagram"):
		return "entity relationship diagram"
	case strings.HasPrefix(kw, "gantt"):
		return "gantt chart"
	case strings.HasPrefix(kw, "pie"):
		return "pie chart"
	case strings.HasPrefix(kw, "gitgraph"):
		return "git graph"
	case strings.HasPrefix(kw, "journey"):
		return "user journey"
	default:
		return kw + " diagram"
	}
}

// countTreeEntries counts non-empty, non-blank lines in a tree diagram,
// which roughly corresponds to the number of files/directories listed.
func countTreeEntries(block string) int {
	count := 0
	for _, line := range strings.Split(block, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

// isBoxDrawingRune returns true for Unicode box-drawing characters
// (U+2500–U+257F) and related line-drawing symbols.
func isBoxDrawingRune(r rune) bool {
	return r >= 0x2500 && r <= 0x257F
}

// isDiagramStructuralChar returns true for characters commonly used to draw
// ASCII diagrams: box corners, lines, arrows, pipes.
func isDiagramStructuralChar(r rune) bool {
	switch r {
	case '+', '-', '|', '=', '<', '>', '/', '\\', '^', '_', '~',
		'[', ']', '{', '}', '(', ')',
		'→', '←', '↑', '↓', '⇒', '⇐':
		return true
	}
	return isBoxDrawingRune(r)
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
