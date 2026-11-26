package dochealth

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type assertionRecord struct {
	path       string
	line       int
	text       string
	normalized string
	base       string
	negative   bool
	section    string
}

var (
	assertionLanguage        = regexp.MustCompile(`(?i)\b(?:must|shall|should|may not|mustn't|shalln't|shouldn't|cannot|can not|do not|does not|is not|are not|never|always|is required|are required|is prohibited|are prohibited|forbidden)\b`)
	normChars                = regexp.MustCompile(`[^\p{L}\p{N}]+`)
	listPrefix               = regexp.MustCompile(`^(?:[-*+]\s+|\d+[.)]\s+)`)
	headingPrefix            = regexp.MustCompile(`^#{1,6}\s+`)
	boldLabel                = regexp.MustCompile(`^\*\*[^*]+:\*\*\s*$`)
	revisitLabel             = regexp.MustCompile(`(?i)^\*\*revisit if\.\s+never\.\*\*\s*$`)
	boilerplateHead          = regexp.MustCompile(`(?i)^#{1,6}\s+purpose of this document\s*#*\s*$`)
	retainedHistoricalNotice = regexp.MustCompile(`(?im)^>\s*\*\*SUPERSEDED\b.*historical provenance only`)
	negativeForms            = []struct{ from, to string }{
		{"must not", "must"}, {"mustn't", "must"},
		{"shall not", "shall"}, {"shalln't", "shall"},
		{"should not", "should"}, {"shouldn't", "should"},
		{"may not", "may"}, {"do not", "do"}, {"does not", "does"},
		{"is not", "is"}, {"are not", "are"}, {"can not", "can"}, {"cannot", "can"},
	}
)

// assertionFindings detects exact repeated normative statements, explicit
// opposite-polarity statements, and the known generic Purpose heading. Exact
// duplicates inside current documents and contradictions inside current
// documents are failures. Cross-document matches and repeats in retained
// evidence are warnings so their scope or historical provenance stays visible.
func assertionFindings(files []string, root string) []Finding {
	var records []assertionRecord
	findings := generatedBlockFindings(files, root)
	historical := make(map[string]bool)
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			findings = append(findings, Finding{Code: "file_read_error", Severity: SeverityFailure, Message: err.Error(), Path: file})
			continue
		}
		rel, err := filepath.Rel(root, file)
		if err != nil {
			rel = file
		}
		historical[filepath.ToSlash(rel)] = isHistoricalAssertionDoc(filepath.ToSlash(rel), content)
		lines := strings.Split(string(content), "\n")
		for index, raw := range lines {
			line := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))
			if boilerplateHead.MatchString(line) {
				severity := SeverityFailure
				if historical[filepath.ToSlash(rel)] {
					severity = SeverityWarning
				}
				findings = append(findings, Finding{
					Code: "boilerplate_purpose_heading", Severity: severity,
					Message: "Remove the generic heading while preserving any document-specific purpose guidance.",
					Path:    filepath.ToSlash(rel), Line: index + 1,
				})
			}
		}
		records = append(records, parseAssertionLines(filepath.ToSlash(rel), lines)...)
	}

	duplicates := make(map[string][]assertionRecord)
	polarities := make(map[string]map[bool][]assertionRecord)
	for _, record := range records {
		duplicates[record.normalized] = append(duplicates[record.normalized], record)
		byPolarity := polarities[record.base]
		if byPolarity == nil {
			byPolarity = make(map[bool][]assertionRecord)
			polarities[record.base] = byPolarity
		}
		byPolarity[record.negative] = append(byPolarity[record.negative], record)
	}

	keys := make([]string, 0, len(duplicates))
	for key := range duplicates {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		group := duplicates[key]
		if len(group) < 2 {
			continue
		}
		for index := 1; index < len(group); index++ {
			current := group[index]
			severity := SeverityWarning
			priorPath, priorLine := "", 0
			for priorIndex := index - 1; priorIndex >= 0; priorIndex-- {
				prior := group[priorIndex]
				if prior.path == current.path && prior.section == current.section {
					priorPath, priorLine = prior.path, prior.line
					if !historical[current.path] {
						severity = SeverityFailure
					}
					break
				}
				if priorPath == "" {
					priorPath, priorLine = prior.path, prior.line
				}
			}
			findings = append(findings, Finding{
				Code: "duplicate_assertion", Severity: severity,
				Message: "Exact normative statement repeats; related occurrence is " + priorPath + ":" + itoa(priorLine) + ". Compare section scope and authority.",
				Path:    current.path, Line: current.line, Target: current.text,
			})
		}
	}

	baseKeys := make([]string, 0, len(polarities))
	for key := range polarities {
		baseKeys = append(baseKeys, key)
	}
	sort.Strings(baseKeys)
	for _, key := range baseKeys {
		positive, negative := polarities[key][false], polarities[key][true]
		for _, left := range positive {
			for _, right := range negative {
				severity := SeverityWarning
				if left.path == right.path && left.section == right.section && !historical[right.path] {
					severity = SeverityFailure
				}
				findings = append(findings, Finding{
					Code: "contradictory_assertion", Severity: severity,
					Message: "Possible opposite-polarity assertions share the same wording; compare scope and authority. Related statement: " + left.path + ":" + itoa(left.line) + ".",
					Path:    right.path, Line: right.line, Target: right.text,
				})
			}
		}
	}

	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		if findings[i].Line != findings[j].Line {
			return findings[i].Line < findings[j].Line
		}
		return findings[i].Code < findings[j].Code
	})
	return findings
}

func parseAssertionLines(path string, lines []string) []assertionRecord {
	var out []assertionRecord
	inFence := false
	inTable := false
	assertionTable := false
	section := ""
	paragraphLine := 0
	var paragraph []string
	flush := func() {
		if len(paragraph) == 0 {
			return
		}
		text := strings.Join(paragraph, " ")
		for _, sentence := range splitAssertionSentences(text) {
			normalized, negative, ok := normalizeAssertion(sentence)
			if !ok {
				continue
			}
			lower := strings.ToLower(sentence)
			if strings.Contains(lower, "not only") {
				lower = strings.ReplaceAll(lower, "not only", "only")
			}
			for _, form := range negativeForms {
				lower = strings.ReplaceAll(lower, form.from, form.to)
			}
			base := normalizeWords(lower)
			out = append(out, assertionRecord{path: path, line: paragraphLine, text: strings.TrimSpace(sentence), normalized: normalized, base: base, negative: negative, section: section})
		}
		paragraph = paragraph[:0]
		paragraphLine = 0
	}
	for index, raw := range lines {
		trimmed := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			flush()
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if isMarkdownTableHeader(lines, index) {
			flush()
			inTable = true
			assertionTable = isAssertionTableHeader(lines[index])
			continue
		}
		if inTable && strings.HasPrefix(trimmed, "|") {
			flush()
			if assertionTable && !isMarkdownTableSeparator(trimmed) {
				paragraphLine = index + 1
				paragraph = append(paragraph, strings.Trim(trimmed, "| "))
				flush()
			}
			continue
		}
		inTable = false
		assertionTable = false
		if headingPrefix.MatchString(trimmed) {
			flush()
			section = trimmed
			continue
		}
		if trimmed == "" || strings.HasPrefix(trimmed, ">") || (strings.HasSuffix(trimmed, ":") && len(strings.Fields(trimmed)) <= 4 && assertionLanguage.MatchString(trimmed)) {
			flush()
			continue
		}
		if boldLabel.MatchString(trimmed) || revisitLabel.MatchString(trimmed) {
			flush()
			continue
		}
		if strings.HasPrefix(trimmed, "|") {
			flush()
			continue
		}
		if listPrefix.MatchString(trimmed) {
			flush()
			trimmed = listPrefix.ReplaceAllString(trimmed, "")
		}
		if paragraphLine == 0 {
			paragraphLine = index + 1
		}
		paragraph = append(paragraph, trimmed)
	}
	flush()
	return out
}

// isMarkdownTableHeader keeps column labels such as "Can Change | Cannot
// Change" out of assertion parsing. Data rows are parsed only when the table
// declares a statement or policy column; descriptive catalogs often contain
// words such as "must" that describe an error without asserting a document rule.
func isMarkdownTableHeader(lines []string, index int) bool {
	if index+1 >= len(lines) {
		return false
	}
	row := strings.TrimSpace(strings.TrimSuffix(lines[index], "\r"))
	separator := strings.TrimSpace(strings.TrimSuffix(lines[index+1], "\r"))
	if !strings.HasPrefix(row, "|") || !strings.HasSuffix(row, "|") || !strings.HasPrefix(separator, "|") || !strings.HasSuffix(separator, "|") {
		return false
	}
	headers := strings.Split(strings.Trim(row, "|"), "|")
	columns := strings.Split(strings.Trim(separator, "|"), "|")
	if len(headers) < 2 || len(headers) != len(columns) {
		return false
	}
	for _, column := range columns {
		marks := strings.Trim(strings.TrimSpace(column), ":")
		if len(marks) < 3 || strings.Trim(marks, "-") != "" {
			return false
		}
	}
	return true
}

func isAssertionTableHeader(row string) bool {
	for _, cell := range strings.Split(strings.Trim(row, "|"), "|") {
		column := strings.ToLower(strings.TrimSpace(cell))
		column = strings.Trim(normChars.ReplaceAllString(column, " "), " ")
		switch column {
		case "rule", "rules", "statement", "statements", "requirement", "requirements", "invariant", "invariants", "obligation", "obligations", "policy", "policies", "constraint", "constraints", "contract", "contracts", "expected behavior", "prohibited behavior":
			return true
		}
	}
	return false
}

func isMarkdownTableSeparator(row string) bool {
	if !strings.HasPrefix(row, "|") || !strings.HasSuffix(row, "|") {
		return false
	}
	for _, cell := range strings.Split(strings.Trim(row, "|"), "|") {
		marks := strings.Trim(strings.TrimSpace(cell), ":")
		if len(marks) < 3 || strings.Trim(marks, "-") != "" {
			return false
		}
	}
	return true
}

// isHistoricalAssertionDoc distinguishes durable evidence from current
// instructions without suppressing it. Findings remain visible as warnings
// for explicit superseded records, evidence trees, and byte snapshots; these
// records do not block an operator-facing docs phase as if they were current
// policy.
func isHistoricalAssertionDoc(path string, content []byte) bool {
	if retainedHistoricalNotice.Match(content) {
		return true
	}
	segments := strings.Split(filepath.ToSlash(path), "/")
	docsInternal := -1
	for index := 0; index+1 < len(segments); index++ {
		if segments[index] == "docs" && segments[index+1] == "internal" {
			docsInternal = index
			break
		}
	}
	if docsInternal < 0 {
		return false
	}
	for _, segment := range segments[docsInternal+2:] {
		if segment == "evidence" || segment == "sources" || segment == "bytes" {
			return true
		}
	}
	return false
}

func splitAssertionSentences(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	var out []string
	start := 0
	for index := 0; index+1 < len(text); index++ {
		if (text[index] == '.' || text[index] == '!' || text[index] == '?') && text[index+1] == ' ' {
			end := index + 2
			for end < len(text) && text[end] == ' ' {
				end++
			}
			if end < len(text) && text[end] >= 'A' && text[end] <= 'Z' {
				if sentence := strings.TrimSpace(text[start : index+1]); sentence != "" {
					out = append(out, sentence)
				}
				start = end
				index = end - 1
			}
		}
	}
	if sentence := strings.TrimSpace(text[start:]); sentence != "" {
		out = append(out, sentence)
	}
	return out
}

func normalizeAssertion(text string) (string, bool, bool) {
	lower := strings.ToLower(strings.TrimSpace(text))
	if !assertionLanguage.MatchString(lower) {
		return "", false, false
	}
	negative := false
	if strings.Contains(lower, "not only") {
		lower = strings.ReplaceAll(lower, "not only", "only")
	}
	for _, form := range negativeForms {
		if strings.Contains(lower, form.from) {
			negative = true
		}
	}
	if strings.Contains(lower, "never") || strings.Contains(lower, "forbidden") || strings.Contains(lower, "prohibited") {
		negative = true
	}
	return normalizeWords(lower), negative, true
}

func normalizeWords(lower string) string {
	for _, marker := range []string{"must", "shall", "should", "may", "always", "never", "required", "prohibited", "forbidden"} {
		lower = strings.ReplaceAll(lower, marker, " "+marker+" ")
	}
	normalized := strings.Join(strings.Fields(normChars.ReplaceAllString(lower, " ")), " ")
	return normalized
}

// itoa avoids importing fmt in this small scanner.
func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	var digits [20]byte
	i := len(digits)
	for value > 0 {
		i--
		digits[i] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[i:])
}
