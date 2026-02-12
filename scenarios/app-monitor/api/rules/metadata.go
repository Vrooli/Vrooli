package rules

import (
	"regexp"
	"strings"
)

// fieldPatterns maps docstring field names to their keys in RuleDef.
var fieldPatterns = []struct {
	key     string
	pattern *regexp.Regexp
}{
	{"name", regexp.MustCompile(`(?m)^Rule:\s*(.+)$`)},
	{"id", regexp.MustCompile(`(?m)^ID:\s*(.+)$`)},
	{"description", regexp.MustCompile(`(?m)^Description:\s*(.+(?:\n\s{2,}.+)*)$`)},
	{"why", regexp.MustCompile(`(?m)^Why:\s*(.+(?:\n\s{2,}.+)*)$`)},
	{"category", regexp.MustCompile(`(?m)^Category:\s*(.+)$`)},
	{"severity", regexp.MustCompile(`(?m)^Severity:\s*(.+)$`)},
	{"slot", regexp.MustCompile(`(?m)^Slot:\s*(.+)$`)},
	{"slot_file", regexp.MustCompile(`(?m)^SlotFile:\s*(.+)$`)},
	{"tech_stack", regexp.MustCompile(`(?m)^TechStack:\s*(.+)$`)},
	{"recommendation", regexp.MustCompile(`(?m)^Recommendation:\s*(.+(?:\n\s{2,}.+)*)$`)},
	{"standard", regexp.MustCompile(`(?m)^Standard:\s*(.+)$`)},
}

var (
	goodExamplePattern = regexp.MustCompile(`(?m)^GoodExample:\n((?:[ \t]+.+\n?)+)`)
	badExamplePattern  = regexp.MustCompile(`(?m)^BadExample:\n((?:[ \t]+.+\n?)+)`)
)

// ParseRuleDoc extracts a RuleDef from a Go source file's block comment.
// Returns (def, true) if a valid rule docstring was found, (zero, false) otherwise.
func ParseRuleDoc(src string) (RuleDef, bool) {
	// Find the first block comment.
	start := strings.Index(src, "/*")
	if start < 0 {
		return RuleDef{}, false
	}
	end := strings.Index(src[start:], "*/")
	if end < 0 {
		return RuleDef{}, false
	}
	comment := src[start+2 : start+end]

	// Must have "Rule:" and "ID:" to be a rule docstring.
	if !strings.Contains(comment, "Rule:") || !strings.Contains(comment, "ID:") {
		return RuleDef{}, false
	}

	fields := map[string]string{}
	for _, fp := range fieldPatterns {
		m := fp.pattern.FindStringSubmatch(comment)
		if len(m) > 1 {
			fields[fp.key] = collapseMultiline(m[1])
		}
	}

	if fields["id"] == "" {
		return RuleDef{}, false
	}

	def := RuleDef{
		ID:             strings.TrimSpace(fields["id"]),
		Name:           strings.TrimSpace(fields["name"]),
		Description:    strings.TrimSpace(fields["description"]),
		Why:            strings.TrimSpace(fields["why"]),
		Category:       strings.TrimSpace(fields["category"]),
		Severity:       strings.TrimSpace(fields["severity"]),
		Slot:           strings.TrimSpace(fields["slot"]),
		SlotFile:       strings.TrimSpace(fields["slot_file"]),
		Recommendation: strings.TrimSpace(fields["recommendation"]),
		Standard:       strings.TrimSpace(fields["standard"]),
	}

	// Parse TechStack as comma-separated list.
	if ts := strings.TrimSpace(fields["tech_stack"]); ts != "" {
		parts := strings.Split(ts, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				def.TechStack = append(def.TechStack, p)
			}
		}
	}
	if def.TechStack == nil {
		def.TechStack = []string{}
	}

	// Parse examples (indented block after GoodExample:/BadExample:).
	if m := goodExamplePattern.FindStringSubmatch(comment); len(m) > 1 {
		def.GoodExample = dedentBlock(m[1])
	}
	if m := badExamplePattern.FindStringSubmatch(comment); len(m) > 1 {
		def.BadExample = dedentBlock(m[1])
	}

	return def, true
}

// collapseMultiline joins continuation lines (indented by 2+ spaces) into one string.
func collapseMultiline(s string) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	return strings.Join(lines, " ")
}

// dedentBlock removes common leading whitespace from a multi-line block.
func dedentBlock(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) == 0 {
		return ""
	}

	// Find minimum indent.
	minIndent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if minIndent < 0 || indent < minIndent {
			minIndent = indent
		}
	}
	if minIndent <= 0 {
		return strings.TrimSpace(s)
	}

	for i, line := range lines {
		if len(line) >= minIndent {
			lines[i] = line[minIndent:]
		}
	}
	return strings.Join(lines, "\n")
}
