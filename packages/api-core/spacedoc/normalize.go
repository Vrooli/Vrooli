package spacedoc

import "strings"

// splitRow splits a markdown table row "| a | b | c |" into its trimmed cells,
// dropping the empty leading/trailing fields produced by the border pipes.
func splitRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	out := make([]string, len(parts))
	for i, p := range parts {
		out[i] = strings.TrimSpace(p)
	}
	return out
}

// isSeparator reports whether a row is the markdown header separator (---|:--).
func isSeparator(cells []string) bool {
	any := false
	for _, c := range cells {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		any = true
		for _, r := range c {
			if r != '-' && r != ':' {
				return false
			}
		}
	}
	return any
}

// categoryRow reports whether a row is a bold category-only label (first cell
// "**Label**", remaining cells empty) and returns the cleaned label. The Guide
// space groups its rows this way instead of with ### subheadings.
func categoryRow(cells []string) (string, bool) {
	if len(cells) == 0 {
		return "", false
	}
	first := strings.TrimSpace(cells[0])
	if !strings.HasPrefix(first, "**") || !strings.HasSuffix(first, "**") {
		return "", false
	}
	for _, c := range cells[1:] {
		if strings.TrimSpace(c) != "" {
			return "", false
		}
	}
	return cleanGroup(first), true
}

// mapColumns maps logical column keys (id, question, owner, status, basis, notes)
// to their header indices by matching header-cell keywords.
func mapColumns(header []string) map[string]int {
	idx := map[string]int{}
	for i, h := range header {
		key := classifyHeader(h)
		if key == "" {
			continue
		}
		if _, exists := idx[key]; !exists {
			idx[key] = i
		}
	}
	return idx
}

func classifyHeader(h string) string {
	l := strings.ToLower(strings.TrimSpace(h))
	switch {
	case l == "#" || l == "id":
		return "id"
	case strings.Contains(l, "question"), strings.Contains(l, "concern"),
		strings.Contains(l, "swe task"), l == "task":
		return "question"
	case strings.Contains(l, "owner"), strings.Contains(l, "guiding skill"),
		strings.Contains(l, "skill"):
		return "owner"
	case strings.Contains(l, "status"):
		return "status"
	case strings.Contains(l, "basis"):
		return "basis"
	case strings.Contains(l, "note"), strings.Contains(l, "approach"):
		return "notes"
	default:
		return ""
	}
}

// normalizeProjection maps a "This Space" projection cell to a Projection.
func normalizeProjection(s string) Projection {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "answer":
		return ProjectionAnswer
	case "validate":
		return ProjectionValidate
	case "guide":
		return ProjectionGuide
	default:
		return ""
	}
}

// normalizeConfidence finds the first of AUTHORITATIVE/PARTIAL/SKETCH in the
// confidence cell, in document order (so validate's "Covered set:
// AUTHORITATIVE ... delta: SKETCH" reports AUTHORITATIVE as its headline).
func normalizeConfidence(s string) DenominatorConfidence {
	upper := strings.ToUpper(s)
	best := -1
	var found DenominatorConfidence
	for token, conf := range map[string]DenominatorConfidence{
		"AUTHORITATIVE": ConfidenceAuthoritative,
		"PARTIAL":       ConfidencePartial,
		"SKETCH":        ConfidenceSketch,
	} {
		if at := strings.Index(upper, token); at >= 0 && (best == -1 || at < best) {
			best = at
			found = conf
		}
	}
	return found
}

// normalizeStatus maps a status cell to a normalized CellStatus, taking the
// first recognized token (so "NOW (UI, CLI) / IN-REACH (API)" -> now and
// "IN-REACH (gap stub)" -> in_reach). COVERED -> now, PARTIAL -> in_reach.
func normalizeStatus(s string) CellStatus {
	upper := strings.ToUpper(s)
	type tok struct {
		needle string
		status CellStatus
	}
	// Order within the scan is positional; longest/most-specific needles must be
	// checked so "IN-REACH" is not shadowed by a bare substring.
	tokens := []tok{
		{"IN-REACH", StatusInReach},
		{"IN REACH", StatusInReach},
		{"COVERED", StatusNow},
		{"PARTIAL", StatusInReach},
		{"MISSING", StatusMissing},
		{"NOW", StatusNow},
	}
	best := -1
	out := CellStatus("")
	for _, t := range tokens {
		if at := strings.Index(upper, t.needle); at >= 0 && (best == -1 || at < best) {
			best = at
			out = t.status
		}
	}
	if out == "" {
		return StatusMissing
	}
	return out
}

// normalizeBasis maps a basis cell to a normalized Basis, taking the first
// recognized token. Heuristic-style tokens map to declared_unverified.
func normalizeBasis(s string) Basis {
	upper := strings.ToUpper(s)
	type tok struct {
		needle string
		basis  Basis
	}
	tokens := []tok{
		{"DECLARED_UNVERIFIED", BasisDeclaredUnverified},
		{"DECLARED", BasisDeclaredUnverified},
		{"CONTRADICTED", BasisContradicted},
		{"VALIDATED", BasisValidated},
		{"DERIVED", BasisDerived},
		{"ABSENT", BasisAbsent},
		{"HEURISTIC", BasisDeclaredUnverified},
		{"PARTIAL", BasisDeclaredUnverified},
	}
	best := -1
	out := Basis("")
	for _, t := range tokens {
		if at := strings.Index(upper, t.needle); at >= 0 && (best == -1 || at < best) {
			best = at
			out = t.basis
		}
	}
	return out
}

// firstToken returns the first whitespace-delimited token of s, used to reduce a
// "This Space" owner cell like "`search-hub` (holds the ...)" to "search-hub".
func firstToken(s string) string {
	s = cleanInline(s)
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// cleanGroup strips markdown emphasis and trailing decoration from a section or
// category label.
func cleanGroup(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "*")
	s = strings.ReplaceAll(s, "—", "-")
	return strings.TrimSpace(s)
}

// cleanInline strips inline markdown (backticks, bold/italic, the requester
// star) and collapses whitespace, leaving readable plain text.
func cleanInline(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "`", "")
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "⭐", "")
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(s)
}
