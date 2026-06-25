package spacedoc

import (
	"fmt"
	"strings"
)

// Parse reads a coverage-space markdown document and returns its normalized
// SpaceDefinition. projection is the caller's expected projection (the owner
// knows which one it owns); Parse cross-checks it against the doc's "This Space"
// table and fails on a mismatch so a misconfigured verb cannot silently emit the
// wrong projection.
func Parse(projection Projection, md []byte) (*SpaceDefinition, error) {
	if !projection.Valid() {
		return nil, fmt.Errorf("spacedoc: unknown projection %q", projection)
	}
	lines := strings.Split(strings.ReplaceAll(string(md), "\r\n", "\n"), "\n")

	meta := parseThisSpace(lines)
	if docProj := normalizeProjection(meta["projection"]); docProj != "" && docProj != projection {
		return nil, fmt.Errorf("spacedoc: doc projection %q does not match requested %q", docProj, projection)
	}

	def := &SpaceDefinition{
		SchemaVersion:         SchemaVersion,
		Projection:            projection,
		Owner:                 firstToken(meta["owner"]),
		DenominatorConfidence: normalizeConfidence(meta["denominator confidence"]),
		ConfidenceRationale:   strings.TrimSpace(meta["denominator confidence"]),
		Cells:                 parseCoverageGrid(lines, projection),
	}
	if def.DenominatorConfidence == "" {
		// Honest default: an unstated confidence is the least confident.
		def.DenominatorConfidence = ConfidenceSketch
	}
	if def.Owner == "" {
		return nil, fmt.Errorf("spacedoc: could not resolve owner from 'This Space' table")
	}
	if len(def.Cells) == 0 {
		return nil, fmt.Errorf("spacedoc: no cells parsed from 'Coverage Grid'")
	}
	return def, nil
}

// parseThisSpace extracts the key/value rows of the "## This Space" 2-column
// metadata table into a lower-cased-key map. Keys seen: "projection", "owner",
// "denominator confidence".
func parseThisSpace(lines []string) map[string]string {
	out := map[string]string{}
	in := false
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "## ") {
			in = strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(line, "## ")), "This Space")
			continue
		}
		if !in || !strings.HasPrefix(line, "|") {
			continue
		}
		cells := splitRow(line)
		if len(cells) < 2 || isSeparator(cells) {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(cells[0]))
		if key == "" {
			continue
		}
		out[key] = strings.TrimSpace(cells[1])
	}
	return out
}

// parseCoverageGrid walks every pipe table under "## Coverage Grid", tracking
// the current group from `### ` subheadings and bold category-only rows, and
// returns the normalized cells.
func parseCoverageGrid(lines []string, projection Projection) []Cell {
	var cells []Cell
	in := false
	group := ""
	var header []string
	colIdx := map[string]int{}
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(line, "## "):
			in = strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(line, "## ")), "Coverage Grid")
			header = nil
			continue
		case !in:
			continue
		case strings.HasPrefix(line, "### "):
			group = cleanGroup(strings.TrimPrefix(line, "### "))
			header = nil
			continue
		case !strings.HasPrefix(line, "|"):
			// Blank line or prose between tables ends the current table.
			if line == "" {
				header = nil
			}
			continue
		}

		row := splitRow(line)
		if isSeparator(row) {
			continue
		}
		if header == nil {
			header = row
			colIdx = mapColumns(header)
			continue
		}
		// A bold category-only row (first cell **X**, rest empty) sets the group.
		if label, ok := categoryRow(row); ok {
			group = label
			continue
		}
		cell, ok := buildCell(row, colIdx, group, projection)
		if ok {
			cells = append(cells, cell)
		}
	}
	return cells
}

// buildCell assembles a Cell from a data row given the column index map.
func buildCell(row []string, colIdx map[string]int, group string, projection Projection) (Cell, bool) {
	get := func(key string) string {
		if i, ok := colIdx[key]; ok && i < len(row) {
			return strings.TrimSpace(row[i])
		}
		return ""
	}
	id := strings.TrimSpace(strings.Trim(get("id"), "#"))
	question := cleanInline(get("question"))
	if id == "" || question == "" {
		return Cell{}, false
	}
	cell := Cell{
		ID:       id,
		Group:    group,
		Question: question,
		Owner:    cleanInline(get("owner")),
		Status:   normalizeStatus(get("status")),
	}
	if projection == ProjectionAnswer {
		cell.Basis = normalizeBasis(get("basis"))
	}
	if notes := cleanInline(get("notes")); notes != "" {
		cell.Notes = []string{notes}
	}
	return cell, true
}
