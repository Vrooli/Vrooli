package normalizer

import (
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// Markdown pass helpers (code blocks, tables, lists, paths)
// ---------------------------------------------------------------------------

// summarizeCodeBlock replaces a fenced code block with a short description.
func summarizeCodeBlock(match string) string {
	parts := reFencedCodeBlock.FindStringSubmatch(match)
	if len(parts) < 3 {
		return "Code block."
	}
	lang := parts[1]
	body := parts[2]
	lineCount := strings.Count(body, "\n") + 1

	if lang != "" {
		return fmt.Sprintf("Code block: %d lines of %s.", lineCount, lang)
	}
	return fmt.Sprintf("Code block: %d lines.", lineCount)
}

// describeTable converts a markdown table into a prose summary.
// Example output: "Table with 3 columns: Name, Age, City. 2 data rows."
func describeTable(match string) string {
	lines := strings.Split(strings.TrimRight(match, "\n"), "\n")
	if len(lines) < 2 {
		return match
	}

	// Parse header row.
	headers := parseTableRow(lines[0])
	if len(headers) == 0 {
		return match
	}

	// Count data rows (skip header and separator).
	dataRows := 0
	for i := 2; i < len(lines); i++ {
		row := strings.TrimSpace(lines[i])
		if row != "" && strings.Contains(row, "|") {
			dataRows++
		}
	}

	colList := strings.Join(headers, ", ")
	rowWord := "row"
	if dataRows != 1 {
		rowWord = "rows"
	}

	result := fmt.Sprintf("Table with %d columns: %s. %d data %s.",
		len(headers), colList, dataRows, rowWord)

	// Optionally include first few data rows as a preview.
	if dataRows > 0 && dataRows <= 5 {
		var previews []string
		for i := 2; i < len(lines) && len(previews) < 3; i++ {
			cells := parseTableRow(lines[i])
			if len(cells) > 0 {
				previews = append(previews, strings.Join(cells, ", "))
			}
		}
		if len(previews) > 0 {
			result += " Items include: " + strings.Join(previews, "; ") + "."
		}
	}

	return result + "\n"
}

// parseTableRow splits a markdown table row into trimmed cell values.
// Example: "| foo | bar |" → ["foo", "bar"].
func parseTableRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.Trim(line, "|")
	parts := strings.Split(line, "|")

	var cells []string
	for _, p := range parts {
		cell := strings.TrimSpace(p)
		// Skip separator cells (---, :--:, etc.)
		if cell != "" && !isSeparatorCell(cell) {
			cells = append(cells, cell)
		}
	}
	return cells
}

// isSeparatorCell returns true if a cell is a table separator like "---" or ":--:".
func isSeparatorCell(cell string) bool {
	for _, ch := range cell {
		if ch != '-' && ch != ':' && ch != ' ' {
			return false
		}
	}
	return true
}

// endsWithSentencePunctuation reports whether s ends with . ! ? or :
func endsWithSentencePunctuation(s string) bool {
	if s == "" {
		return false
	}
	switch s[len(s)-1] {
	case '.', '!', '?', ':':
		return true
	}
	return false
}

// stripBulletAddPeriod removes a bullet marker (-, *, +) and appends a period
// if the item doesn't already end with sentence punctuation.
// "- Install the package" → "Install the package."
func stripBulletAddPeriod(match string) string {
	item := strings.TrimSpace(reUnorderedList.FindStringSubmatch(match)[1])
	if item == "" {
		return ""
	}
	if !endsWithSentencePunctuation(item) {
		item += "."
	}
	return item
}

// keepNumberAddPeriod preserves the number prefix and appends a period if the
// item doesn't already end with sentence punctuation.
// "1. Install the package" → "1. Install the package."
func keepNumberAddPeriod(match string) string {
	sub := reOrderedList.FindStringSubmatch(match)
	item := strings.TrimSpace(sub[1])
	if item == "" {
		return ""
	}
	// Extract the number from the original match.
	numStr := strings.TrimSpace(reOrderedListNumber.FindString(match))
	if !endsWithSentencePunctuation(item) {
		item += "."
	}
	return numStr + " " + item
}

// extractBasename returns the last path segment of a file path.
// "/home/user/project/src/main.go" → "main.go"
func extractBasename(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	// Find last slash.
	idx := strings.LastIndex(path, "/")
	if idx < 0 || idx == len(path)-1 {
		return path
	}
	return path[idx+1:]
}
